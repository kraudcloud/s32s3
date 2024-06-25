# rclone s3 to s3 with crypt

This repository is me messing around with using rclone for encrypted s3 to s3 backup. The destination should never be able to read the objects nor own the decryption key.

## Chart

There's example usage for the chart in [the helmfile](./helmfile.yaml).
