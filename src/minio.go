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

const (
	// SourceMetadata is the name of the file that contains the additional metadata for an instance, such as IAM configuration
	SourceMetadata = "metadata.tar.gz"

	// fileIAM is the name of the file that contains the IAM configuration
	fileIAM = "iam.zip"

	// fileBuckets is the name of the file that contains the bucket metadata
	fileBuckets = "buckets.zip"

	// fileConfig is the name of the file that contains the Minio configuration
	fileConfig = "config.txt"
)

// SourceMetadata returns the additional metadata for an instance
//
// The exported data includes:
// - IAM configuration (fileIAM)
// - Bucket metadata (fileBuckets)
// - Minio configuration (fileConfig)
//
// The exported data is stored in a tar.gz file in the temporary directory, and the path to the file is returned.
func (m *Minio) SourceMetadata(ctx context.Context) (string, error) {
	dir, err := os.MkdirTemp("", "s32s3-*")
	if err != nil {
		return "", err
	}

	f, err := os.Create(filepath.Join(dir, SourceMetadata))
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
		Name: fileIAM,
		Mode: 0o644,
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
		Name: fileBuckets,
		Mode: 0o644,
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
		Name: fileConfig,
		Mode: 0o644,
		Size: int64(len(oidc)),
	})
	if err != nil {
		return "", err
	}

	archive.Write(oidc)
	return f.Name(), nil
}

// RestoreMeta restores the additional metadata for an instance, such as IAM configuration, bucket metadata, and OIDC configuration, from a tar.gz archive.
// The archive is expected to contain the following files:
// - fileIAM: IAM configuration
// - fileBuckets: Bucket metadata
// - fileConfig: OIDC configuration
func (m *Minio) RestoreMeta(ctx context.Context, meta string) error {
	f, err := os.Open(meta)
	if err != nil {
		return err
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gzr.Close()

	archive := tar.NewReader(gzr)

	for {
		header, err := archive.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		switch header.Name {
		case fileIAM:
			if err := m.adminClient.ImportIAM(ctx, io.NopCloser(archive)); err != nil {
				return err
			}
		case fileBuckets:
			resp, err := m.adminClient.ImportBucketMetadata(ctx, "", io.NopCloser(archive))
			if err != nil {
				return err
			}

			for name, value := range resp.Buckets {
				if value.Err != "" {
					m.log.Error("failed to import bucket", "bucket", name, "err", value.Err)
					continue
				}

				m.log.Info("imported bucket", "bucket", name, "value", value)
			}

		case fileConfig:
			if err := m.adminClient.SetConfig(ctx, io.NopCloser(archive)); err != nil {
				return err
			}
		}
	}

	return nil
}

type BackupBucketOptions struct {
	ExpirationDays int
	Bucket         string
}

// AssertOrCreateBucket ensures that the specified bucket exists, and if not, creates it with versioning and a lifecycle policy to expire old versions.
// The bucket will be created with the specified expiration days for old versions. If no expiration days are provided, a default of 7 days will be used.
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
