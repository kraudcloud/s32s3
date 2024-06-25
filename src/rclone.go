package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"time"
)

func EncodeConfig(w io.Writer, c BackupConfig) error {
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

func RcloneSyncBucket(ctx context.Context, log *slog.Logger, config BackupConfig, bucket string) error {
	f, err := os.CreateTemp("", "rclone.conf")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	defer os.Remove(f.Name())
	defer f.Close()

	err = EncodeConfig(f, config)
	if err != nil {
		return fmt.Errorf("build rclone config: %w", err)
	}

	args := []string{
		"sync",
		"--config", f.Name(),
		fmt.Sprintf("%s:%s", config.Source.Name, bucket),
		fmt.Sprintf("%s:%s", config.Crypt.Name, bucket),
	}

	log.Info("running rclone", "args", args)

	cmd := exec.CommandContext(ctx, "rclone", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("rclone sync: %w", err)
	}

	log.Info("rclone sync complete")
	return nil
}

func RcloneListBucketsRemote(ctx context.Context, log *slog.Logger, config BackupConfig, remote string) ([]string, error) {
	f, err := os.CreateTemp("", "rclone.conf")
	if err != nil {
		return nil, fmt.Errorf("create temp file: %w", err)
	}
	defer os.Remove(f.Name())
	defer f.Close()

	err = EncodeConfig(f, config)
	if err != nil {
		return nil, fmt.Errorf("build rclone config: %w", err)
	}

	args := []string{
		"lsjson",
		"--config", f.Name(),
		fmt.Sprintf("%s:", remote),
	}
	log.Info("running rclone", "args", args)
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

	log.Info("rclone lsjson complete", "buckets", buckets)
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
