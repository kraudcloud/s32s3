package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"

	"github.com/minio/madmin-go/v3"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/minio/minio-go/v7/pkg/lifecycle"
	"github.com/rclone/rclone/backend/s3"
)

type Minio struct {
	log         *slog.Logger
	config      s3.Options
	client      *minio.Client
	adminClient *madmin.AdminClient
}

// ListBuckets returns a list of buckets in the Minio instance.
func (m *Minio) ListBuckets(ctx context.Context) ([]string, error) {
	var buckets []string

	bks, err := m.client.ListBuckets(ctx)
	if err != nil {
		return nil, err
	}

	for _, bucket := range bks {
		buckets = append(buckets, bucket.Name)
	}

	return buckets, nil
}

// SourceMetadata returns the additional metadata for an instance, such as IAM configuration
// OIDC configuration, etc.
func (m *Minio) SourceMetadata(ctx context.Context) (string, error) {
	dir, err := os.MkdirTemp("", "s32s3-*")
	if err != nil {
		return "", err
	}

	f, err := os.Create(filepath.Join(dir, "metadata.tar.gz"))
	if err != nil {
		return "", err
	}
	defer f.Close()

	gzw := gzip.NewWriter(f)
	defer gzw.Close()

	archive := tar.NewWriter(gzw)
	defer archive.Close()

	// IAM
	iam, err := m.adminClient.ExportIAM(ctx)
	if err != nil {
		return "", err
	}

	buf := bytes.NewBuffer(nil)
	io.Copy(buf, iam)
	iam.Close()

	err = archive.WriteHeader(&tar.Header{
		Name: "iam.zip",
		Mode: 0644,
		Size: int64(buf.Len()),
	})
	if err != nil {
		return "", err
	}
	io.Copy(archive, buf)

	// Buckets
	buf.Reset()
	buckets, err := m.adminClient.ExportBucketMetadata(ctx, "")
	if err != nil {
		return "", err
	}
	io.Copy(buf, buckets)
	buckets.Close()

	err = archive.WriteHeader(&tar.Header{
		Name: "buckets.zip",
		Mode: 0644,
		Size: int64(buf.Len()),
	})
	if err != nil {
		return "", err
	}
	io.Copy(archive, buf)

	// OIDC
	buf.Reset()
	oidc, err := m.adminClient.GetConfig(ctx)
	if err != nil {
		return "", err
	}

	err = archive.WriteHeader(&tar.Header{
		Name: "config.txt",
		Mode: 0644,
		Size: int64(len(oidc)),
	})
	if err != nil {
		return "", err
	}

	archive.Write(oidc)
	return f.Name(), nil
}

// SourceRestoreMeta restores the additional metadata for an instance, such as IAM configuration
func (m *Minio) RestoreMeta(ctx context.Context, meta string) error {
	return nil
}

type BackupBucketOptions struct {
	ExpirationDays int
	Bucket         string
}

func (m *Minio) AssertOrCreateBucket(ctx context.Context, opt BackupBucketOptions) error {
	log := m.log.With("bucket", opt.Bucket)
	log.Info("checking if bucket exists")
	exists, err := m.client.BucketExists(ctx, opt.Bucket)
	if err != nil {
		return err
	}

	if exists {
		log.Info("found bucket")
		return nil
	}

	log.Info("creating bucket")
	err = m.client.MakeBucket(ctx, opt.Bucket, minio.MakeBucketOptions{})
	if err != nil {
		return err
	}

	if opt.ExpirationDays < 0 {
		return nil
	}
	if opt.ExpirationDays == 0 {
		opt.ExpirationDays = 7
	}

	log.Info("enabling versioning")
	err = m.client.EnableVersioning(ctx, opt.Bucket)
	if err != nil {
		return err
	}

	log.Info("setting lifecycle")
	err = m.client.SetBucketLifecycle(ctx, opt.Bucket, &lifecycle.Configuration{
		Rules: []lifecycle.Rule{
			{
				NoncurrentVersionExpiration: lifecycle.NoncurrentVersionExpiration{
					NoncurrentDays: lifecycle.ExpirationDays(opt.ExpirationDays),
				},
				Status: "Enabled",
			},
		},
	})
	if err != nil {
		return err
	}

	return nil
}

func NewMinio(logger *slog.Logger, config s3.Options) (*Minio, error) {
	u, err := url.Parse(config.Endpoint)
	if err != nil {
		return &Minio{}, err
	}

	creds := credentials.New(&credentials.Static{
		Value: credentials.Value{
			AccessKeyID:     config.AccessKeyID,
			SecretAccessKey: config.SecretAccessKey,
		},
	})

	c, err := minio.New(u.Host, &minio.Options{
		Creds:  creds,
		Region: config.Region,
	})
	if err != nil {
		return &Minio{}, err
	}

	ac, err := madmin.NewWithOptions(u.Host, &madmin.Options{
		Creds: creds,
	})
	if err != nil {
		return &Minio{}, err
	}

	return &Minio{log: logger, config: config, client: c, adminClient: ac}, nil
}
