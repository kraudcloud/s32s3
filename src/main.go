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

func Restore() error {
	panic("TODO: unimplemented")
}

func Backup() {
	config, err := Config()
	if err != nil {
		panic(err)
	}

	src, err := NewMinio(config.Source.Value)
	if err != nil {
		panic(err)
	}

	dest, err := NewMinio(config.Dest.Value)
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

	l := slog.New(slog.NewTextHandler(os.Stdout, nil))
	iter.ForEach(buckets, func(bucket *string) {
		l := l.With("bucket", *bucket)
		l.Info("syncing bucket")
		err := RcloneSyncBucket(context.Background(), l, config, *bucket)
		if err != nil {
			panic(err)
		}
	})
}
