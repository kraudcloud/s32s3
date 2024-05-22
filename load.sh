#!/usr/bin/env sh

docker build ./image -t ctr.0x.pt/ops/s32s3:latest
kind load docker-image --name rclone-s3-backup ctr.0x.pt/ops/s32s3:latest
