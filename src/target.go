package main

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"

	"github.com/rclone/rclone/backend/s3"
)

type Target interface {
	ListBuckets(ctx context.Context) ([]string, error)
	BackupMeta(ctx context.Context) ([]byte, error)
	RestoreMeta(ctx context.Context, meta []byte) error
	AssertOrCreateBucket(ctx context.Context, name string) error
}

func TargetFactory(config Wrapped[s3.Options], logger *slog.Logger) (Target, error) {
	switch config.Value.Provider {
	case "Minio":
		return NewMinio(logger, config.Value, config.Name == destName)
	case "mock":
		return &MockTarget{}, nil
	default:
		return nil, fmt.Errorf("unknown target type: %s for %s", config.Type, config.Name)
	}
}

type MockTarget struct {
	Buckets []string
	Meta    []byte
}

func (m MockTarget) ListBuckets(ctx context.Context) ([]string, error) {
	return m.Buckets, nil
}

func (m MockTarget) BackupMeta(ctx context.Context) ([]byte, error) {
	return m.Meta, nil
}

func (m *MockTarget) RestoreMeta(ctx context.Context, meta []byte) error {
	if len(m.Meta) != 0 {
		if !bytes.Equal(m.Meta, meta) {
			return fmt.Errorf("metadata mismatch")
		}
	} else {
		m.Meta = meta
	}

	return nil
}

func (m *MockTarget) AssertOrCreateBucket(ctx context.Context, name string) error {
	for _, bucket := range m.Buckets {
		if bucket == name {
			return nil // Bucket already exists
		}
	}

	// Bucket doesn't exist, create it
	m.Buckets = append(m.Buckets, name)
	return nil
}
