package main

import (
	"context"
	"log/slog"
	"net/url"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/minio/minio-go/v7/pkg/lifecycle"
	"github.com/rclone/rclone/backend/s3"
)

type Minio struct {
	config s3.Options
	client *minio.Client
}

var _ Target = (*Minio)(nil)

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

// BackupMeta returns the additional metadata for an instance, such as IAM configuration
// OIDC configuration, etc.
// Only `Minio` is supposed to support it
func (m *Minio) BackupMeta(ctx context.Context) ([]byte, error) {
	return nil, nil
}

// RestoreMeta restores the metadata for an instance.
func (m *Minio) RestoreMeta(ctx context.Context, meta []byte) error {
	return nil
}

func (m *Minio) AssertOrCreateBucket(ctx context.Context, name string) error {
	log := slog.With("bucket", name)
	log.Info("checking if bucket exists", "bucket", name)
	exists, err := m.client.BucketExists(ctx, name)
	if err != nil {
		return err
	}

	if exists {
		return nil
	}

	log.Info("creating bucket")
	err = m.client.MakeBucket(ctx, name, minio.MakeBucketOptions{})
	if err != nil {
		return err
	}

	log.Info("enabling versioning")
	err = m.client.EnableVersioning(ctx, name)
	if err != nil {
		return err
	}

	log.Info("setting lifecycle")
	err = m.client.SetBucketLifecycle(ctx, name, &lifecycle.Configuration{
		Rules: []lifecycle.Rule{
			{
				NoncurrentVersionExpiration: lifecycle.NoncurrentVersionExpiration{
					NoncurrentDays: 30,
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

func NewMinio(config s3.Options) (*Minio, error) {
	u, err := url.Parse(config.Endpoint)
	if err != nil {
		return &Minio{}, err
	}

	c, err := minio.New(u.Host, &minio.Options{
		Creds: credentials.New(&credentials.Static{
			Value: credentials.Value{
				AccessKeyID:     config.AccessKeyID,
				SecretAccessKey: config.SecretAccessKey,
			},
		}),
		Region: config.Region,
	})
	if err != nil {
		return &Minio{}, err
	}

	return &Minio{config: config, client: c}, nil
}
