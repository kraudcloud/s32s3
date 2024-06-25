package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"

	"github.com/rclone/rclone/fs/config/obscure"
)

func writeConfig(w io.Writer, c BackupConfig) error {
	pass1, err := obscure.Obscure(c.Crypt.Password)
	if err != nil {
		return fmt.Errorf("obscure password: %w", err)
	}

	pass2, err := obscure.Obscure(c.Crypt.Password2)
	if err != nil {
		return fmt.Errorf("obscure password2: %w", err)
	}

	err = EncodeIni(w, map[string]map[string]any{
		c.Source.Name: {
			"type":              "s3",
			"env_auth":          "false",
			"provider":          c.Source.Provider,
			"region":            c.Source.Region,
			"endpoint":          c.Source.Endpoint,
			"access_key_id":     c.Source.AccessKeyID,
			"secret_access_key": c.Source.SecretAccessKey,
		},
		c.Dest.Name: {
			"type":              "s3",
			"env_auth":          "false",
			"provider":          c.Dest.Provider,
			"region":            c.Dest.Region,
			"endpoint":          c.Dest.Endpoint,
			"access_key_id":     c.Dest.AccessKeyID,
			"secret_access_key": c.Dest.SecretAccessKey,
		},
		c.Crypt.Name: {
			"type":      "crypt",
			"password":  pass1,
			"password2": pass2,
			// https://rclone.org/crypt/#configuration
			"remote": fmt.Sprintf("%s:", c.Dest.Name),
		},
	})
	if err != nil {
		return fmt.Errorf("encode ini: %w", err)
	}

	return nil
}

func EncodeIni(w io.Writer, c map[string]map[string]any) error {
	for section, values := range c {
		_, err := fmt.Fprintf(w, "[%s]\n", section)
		if err != nil {
			return err
		}

		for key, value := range values {
			_, err := fmt.Fprintf(w, "%s = %v\n", key, value)
			if err != nil {
				return err
			}
		}

		_, err = fmt.Fprintln(w)
		if err != nil {
			return err
		}
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

	err = writeConfig(io.MultiWriter(f, os.Stderr), config)
	if err != nil {
		return fmt.Errorf("build rclone config: %w", err)
	}

	f.Sync()

	args := []string{
		"sync",
		"--config", f.Name(),
		fmt.Sprintf("%s:%s", config.Source.Name, bucket),
		fmt.Sprintf("%s:%s%s", config.Crypt.Name, config.Prefix, bucket),
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
