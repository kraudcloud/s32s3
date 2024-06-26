package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/sourcegraph/conc/iter"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: s32s3 <command>")
		fmt.Println("Commands:")
		fmt.Println("  backup - backup all buckets")
		fmt.Println("  restore - restore all buckets")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "backup":
		Backup()
	case "restore":
		Restore()
	default:
		fmt.Println("Unknown command", os.Args[1])
		os.Exit(1)
	}
}

func Restore() {
	config, err := Config()
	if err != nil {
		panic(err)
	}

	l := slog.New(slog.NewTextHandler(os.Stdout, nil))
	dst, err := TargetFactory(config.Dest, l.With("target", config.Dest.Name))
	if err != nil {
		panic(err)
	}

	meta, err := dst.BackupMeta(context.Background())
	if err != nil {
		panic(err)
	}

	src, err := TargetFactory(config.Source, l.With("target", config.Source.Name))
	if err != nil {
		panic(err)
	}

	// TODO
	err = src.RestoreMeta(context.Background(), meta)
	if err != nil {
		panic(err)
	}

	buckets, err := RcloneListBucketsRemote(context.Background(), l.With("target", config.Crypt.Name), config, config.Crypt.Name)
	if err != nil {
		panic(err)
	}

	iter.ForEach(buckets, func(bucket *string) {
		l := l.With("bucket", *bucket).With("source", config.Crypt.Name).With("dest", config.Source.Name)
		l.Info("restoring bucket")
		err := RcloneSyncBucket(context.Background(), l, config, SyncBucketOptions{
			Bucket: *bucket,
			Source: config.Crypt.Name,
			Dest:   config.Source.Name,
		})
		if err != nil {
			l.Error("failed to restore bucket", "err", err)
			return
		}
	})
}

func Backup() {
	config, err := Config()
	if err != nil {
		panic(err)
	}

	l := slog.New(slog.NewTextHandler(os.Stdout, nil))
	src, err := TargetFactory(config.Source, l.With("target", config.Source.Name))
	if err != nil {
		panic(err)
	}

	dest, err := TargetFactory(config.Dest, l.With("target", config.Dest.Name))
	if err != nil {
		panic(err)
	}

	err = dest.AssertOrCreateBucket(context.Background(), config.BackupBucket)
	if err != nil {
		panic(err)
	}

	buckets, err := src.ListBuckets(context.Background())
	if err != nil {
		panic(err)
	}

	iter.ForEach(buckets, func(bucket *string) {
		l := l.With("bucket", *bucket).With("source", config.Source.Name).With("dest", config.Crypt.Name)
		l.Info("backing up bucket")
		err := RcloneSyncBucket(context.Background(), l, config, SyncBucketOptions{
			Bucket: *bucket,
			Source: config.Source.Name,
			Dest:   config.Crypt.Name,
		})
		if err != nil {
			l.Error("failed to backup bucket", "err", err)
			return
		}
	})
}
