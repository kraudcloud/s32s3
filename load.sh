#!/usr/bin/env sh

docker build ./image -t rclone
kind load docker-image --name rclone-s3-backup rclone
