package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/sourcegraph/conc/iter"
	"github.com/urfave/cli/v3"
)

func main() {
	commands := []*cli.Command{
		{
			Name:  "backup",
			Usage: "backup all buckets and instance metadata",
			Action: func(ctx context.Context, c *cli.Command) error {
				Backup(ctx)
				return nil
			},
		},
		{
			Name:  "restore",
			Usage: "restore all buckets and instance metadata",
			Flags: []cli.Flag{
				&cli.TimestampFlag{
					Name:  "at",
					Usage: "restore at specific time",
					Config: cli.TimestampConfig{
						Layout:   "2006-01-02T15:04:05",
						Timezone: time.Local,
					},
				},
			},
			Action: func(ctx context.Context, c *cli.Command) error {
				at := c.Timestamp("at")
				if at.IsZero() {
					Restore(ctx)
					return nil
				}

				return fmt.Errorf("restore at is not implemented (at: %s)", at.String())
			},
		},
		{
			Name:  "rclone-config",
			Usage: "show rclone config",
			Action: func(ctx context.Context, c *cli.Command) error {
				RcloneConfig(ctx)
				return nil
			},
		},
		{
			Name:  "minio-config",
			Usage: "dump minio instance config",
			Action: func(ctx context.Context, c *cli.Command) error {
				MinioConfig(ctx)
				return nil
			},
		},
	}

	app := &cli.Command{
		Name:     "s32s3",
		Usage:    "Backup and restore S3 buckets",
		Commands: commands,
	}

	err := app.Run(context.Background(), os.Args)
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}

func Restore(ctx context.Context) {
	config, err := Config()
	if err != nil {
		panic(err)
	}

	l := slog.New(slog.NewTextHandler(os.Stdout, nil))
	// first restore meta
	file, err := RcloneDownloadFile(ctx, config, DownloadFileOptions{
		File:   SourceMetadata,
		Source: config.Crypt.Name,
		log:    l.With("target", config.Crypt.Name),
	})
	if err != nil {
		panic(err)
	}

	m, err := NewMinio(l, config.Source.Value)
	if err != nil {
		panic(err)
	}
	err = m.RestoreMeta(ctx, file)
	if err != nil {
		panic(err)
	}

	buckets, err := RcloneListBucketsRemote(ctx, config, ListBucketsOptions{
		Remote: config.Crypt.Name,
		log:    l.With("target", config.Crypt.Name),
	})
	if err != nil {
		panic(err)
	}

	iter.ForEach(buckets, func(bucket *string) {
		l := l.With("bucket", *bucket).With("source", config.Crypt.Name).With("dest", config.Source.Name)
		l.Info("restoring bucket")
		err := RcloneSyncBucket(ctx, config, SyncBucketOptions{
			Bucket: *bucket,
			Source: config.Crypt.Name,
			Dest:   config.Source.Name,
			log:    l,
		})
		if err != nil {
			l.Error("failed to restore bucket", "err", err)
			return
		}
	})
}

func RcloneConfig(ctx context.Context) {
	config, err := Config()
	if err != nil {
		panic(err)
	}

	EncodeConfig(os.Stdout, config)
}

func MinioConfig(ctx context.Context) {
	config, err := Config()
	if err != nil {
		panic(err)
	}

	l := slog.New(slog.NewTextHandler(os.Stdout, nil))
	src, err := NewMinio(l.With("target", config.Source.Name), config.Source.Value)
	if err != nil {
		panic(err)
	}

	path, err := src.SourceMetadata(ctx)
	if err != nil {
		panic(err)
	}

	fmt.Println(path)
}

func Backup(ctx context.Context) {
	config, err := Config()
	if err != nil {
		panic(err)
	}

	l := slog.New(slog.NewTextHandler(os.Stdout, nil))
	src, err := NewMinio(l.With("target", config.Source.Name), config.Source.Value)
	if err != nil {
		panic(err)
	}

	dest, err := NewMinio(l.With("target", config.Dest.Name), config.Dest.Value)
	if err != nil {
		panic(err)
	}

	err = dest.AssertOrCreateBucket(ctx, BackupBucketOptions{
		Bucket:         config.BackupBucket,
		ExpirationDays: config.ExpirationDays,
	})
	if err != nil {
		panic(err)
	}

	metapath, err := src.SourceMetadata(ctx)
	if err != nil {
		panic(err)
	}

	RcloneSyncFile(ctx, config, SyncFileOptions{
		File: metapath,
		Dest: config.Crypt.Name,
		log:  l,
	})

	buckets, err := src.ListBuckets(ctx)
	if err != nil {
		panic(err)
	}

	iter.ForEach(buckets, func(bucket *string) {
		l := l.With("bucket", *bucket).With("source", config.Source.Name).With("dest", config.Crypt.Name)
		l.Info("backing up bucket")
		err := RcloneSyncBucket(ctx, config, SyncBucketOptions{
			Bucket: *bucket,
			Source: config.Source.Name,
			Dest:   config.Crypt.Name,
			log:    l,
		})
		if err != nil {
			l.Error("failed to backup bucket", "err", err)
			return
		}
	})
}
