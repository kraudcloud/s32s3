package main

import (
	"bytes"
	"context"
	"fmt"
)

type Target interface {
	ListBuckets(ctx context.Context) ([]string, error)
	BackupMeta(ctx context.Context) ([]byte, error)
	RestoreMeta(ctx context.Context, meta []byte) error
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
