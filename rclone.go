package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// EncodeConfig writes the backup configuration to the provided io.Writer in INI format.
func EncodeConfig(w io.Writer, c BackupConfig) error {
	fmt.Fprintf(w, "# backup_bucket = %s\n", c.BackupBucket)
	fmt.Fprintf(w, "# expiration_days = %d\n", c.ExpirationDays)
	if err := c.Source.EncodeIni(w); err != nil {
		return fmt.Errorf("source: encode ini: %w", err)
	}

	if err := c.Dest.EncodeIni(w); err != nil {
		return fmt.Errorf("dest: encode ini: %w", err)
	}

	if err := c.Crypt.EncodeIni(w); err != nil {
		return fmt.Errorf("crypt: encode ini: %w", err)
	}

	return nil
}

type SyncBucketOptions struct {
	Bucket string
	Source string
	Dest   string
	At     *string
	log    *slog.Logger
}

// RcloneSyncBucket syncs the specified source bucket to the specified destination bucket using the rclone command.
func RcloneSyncBucket(ctx context.Context, config BackupConfig, opts SyncBucketOptions) error {
	f, err := os.CreateTemp("", "rclone.conf")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	defer os.Remove(f.Name())
	defer f.Close()

	if opts.At != nil {
		config.Dest.Value.VersionAt.Set(*opts.At)
	}
	err = EncodeConfig(f, config)
	if err != nil {
		return fmt.Errorf("build rclone config: %w", err)
	}

	args := []string{
		"sync",
		"--config", f.Name(),
		fmt.Sprintf("%s:%s", opts.Source, opts.Bucket),
		fmt.Sprintf("%s:%s", opts.Dest, opts.Bucket),
	}

	opts.log.Info("running rclone", "args", args)

	cmd := exec.CommandContext(ctx, "rclone", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("rclone sync: %w", err)
	}

	opts.log.Info("rclone sync complete")
	return nil
}

type SyncFileOptions struct {
	File string
	Dest string
	At   *string
	log  *slog.Logger
}

// RcloneSyncFile syncs a local file to the specified destination using the rclone command.
func RcloneSyncFile(ctx context.Context, config BackupConfig, opts SyncFileOptions) error {
	f, err := os.CreateTemp("", "rclone.conf")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	defer os.Remove(f.Name())
	defer f.Close()

	if opts.At != nil {
		config.Dest.Value.VersionAt.Set(*opts.At)
	}
	err = EncodeConfig(f, config)
	if err != nil {
		return fmt.Errorf("build rclone config: %w", err)
	}

	args := []string{
		"sync",
		"--config", f.Name(),
		opts.File,
		fmt.Sprintf("%s:", opts.Dest),
	}

	opts.log.Info("running rclone", "args", args)
	cmd := exec.CommandContext(ctx, "rclone", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("rclone sync: %w", err)
	}

	opts.log.Info("rclone sync complete")
	return nil
}

type DownloadFileOptions struct {
	File   string
	Source string
	At     *string
	log    *slog.Logger
}

// RcloneDownloadFile downloads a file from the specified source location to a temporary directory.
func RcloneDownloadFile(ctx context.Context, config BackupConfig, opts DownloadFileOptions) (string, error) {
	f, err := os.CreateTemp("", "rclone.conf")
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	defer os.Remove(f.Name())
	defer f.Close()

	if opts.At != nil {
		config.Dest.Value.VersionAt.Set(*opts.At)
	}
	err = EncodeConfig(f, config)
	if err != nil {
		return "", fmt.Errorf("build rclone config: %w", err)
	}

	dir, err := os.MkdirTemp("", "s32s3-*")
	if err != nil {
		return "", err
	}

	args := []string{
		"copy",
		"--config", f.Name(),
		fmt.Sprintf("%s:%s", opts.Source, opts.File),
		dir,
	}

	opts.log.Info("running rclone", "args", args)
	cmd := exec.CommandContext(ctx, "rclone", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		return "", fmt.Errorf("rclone copy: %w", err)
	}

	opts.log.Info("rclone copy complete")
	return filepath.Join(dir, opts.File), nil
}

type ListBucketsOptions struct {
	Remote string
	At     *string
	log    *slog.Logger
}

// RcloneListBucketsRemote lists the buckets in the specified remote location using the provided BackupConfig and ListBucketsOptions.
func RcloneListBucketsRemote(ctx context.Context, config BackupConfig, opts ListBucketsOptions) ([]string, error) {
	f, err := os.CreateTemp("", "rclone.conf")
	if err != nil {
		return nil, fmt.Errorf("create temp file: %w", err)
	}
	defer os.Remove(f.Name())
	defer f.Close()

	if opts.At != nil {
		config.Dest.Value.VersionAt.Set(*opts.At)
	}
	err = EncodeConfig(f, config)
	if err != nil {
		return nil, fmt.Errorf("build rclone config: %w", err)
	}

	args := []string{
		"lsjson",
		"--config", f.Name(),
		fmt.Sprintf("%s:", opts.Remote),
	}

	opts.log.Info("running rclone", "args", args)

	cmd := exec.CommandContext(ctx, "rclone", args...)
	cmd.Stderr = os.Stderr
	data, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("rclone lsjson: %w", err)
	}

	var files []FileInfo
	err = json.Unmarshal(data, &files)
	if err != nil {
		return nil, fmt.Errorf("unmarshal json: %w", err)
	}

	buckets := make([]string, 0, len(files))
	for _, f := range files {
		if f.IsBucket {
			buckets = append(buckets, f.Name)
		}
	}

	opts.log.Info("rclone lsjson complete", "buckets", buckets)
	return buckets, nil
}

type FileInfo struct {
	Hashes        Hashes    `json:"Hashes"`
	ID            string    `json:"ID"`
	OrigID        string    `json:"OrigID"`
	IsBucket      bool      `json:"IsBucket"`
	IsDir         bool      `json:"IsDir"`
	MimeType      string    `json:"MimeType"`
	ModTime       time.Time `json:"ModTime"`
	Name          string    `json:"Name"`
	Encrypted     string    `json:"Encrypted"`
	EncryptedPath string    `json:"EncryptedPath"`
	Path          string    `json:"Path"`
	Size          int64     `json:"Size"`
	Tier          string    `json:"Tier"`
}

type Hashes struct {
	SHA1        string `json:"SHA-1"`
	MD5         string `json:"MD5"`
	DropboxHash string `json:"DropboxHash"`
}
