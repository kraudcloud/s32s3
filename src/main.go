package main

import (
	"context"
	"log/slog"
	"os"
	"sync"
)

func main() {
	config, err := Config()
	if err != nil {
		panic(err)
	}

	src, err := NewMinio(config.Source.S3Config)
	if err != nil {
		panic(err)
	}

	buckets, err := src.ListBuckets(context.Background())
	if err != nil {
		panic(err)
	}

	l := slog.New(slog.NewTextHandler(os.Stdout, nil))

	wg := sync.WaitGroup{}
	for _, bucket := range buckets {
		wg.Add(1)
		go func(bucket string) {
			defer wg.Done()
			l := l.With("bucket", bucket)
			l.Info("syncing bucket")
			err := RcloneSyncBucket(context.Background(), l, config, bucket)
			if err != nil {
				panic(err)
			}
		}(bucket)
	}

	wg.Wait()
}
