# rclone s3 to s3 with crypt

This repository is me messing around with using rclone for encrypted s3 to s3 backup. The destination should never be able to read the objects nor own the decryption key.

## Chart

There's example usage for the chart in [the helmfile](./helmfile.yaml).

## TODO

- Don't use seperate jobs to build the secret, just build it on the fly before running restore/backup.
- Build a secret for each sub-section (crypt, source, destionation) to pull the values from. This allows the user to edit fields of those directly instead of editing the helm chart. Allows for quick password rotation.
