#!/usr/bin/env bash

set -euo pipefail

config_file=${RCLONE_CONFIG_PATH-/config/rclone/rclone.conf}
source_name=${RCLONE_SOURCE_NAME-source:}
dest_name=${RCLONE_DESTINATION_NAME-dest:}
bucket_prefix=${RCLONE_BUCKET_PREFIX?"RCLONE_BUCKET_PREFIX is required"}
rclone_args=${RCLONE_ARGS-"--fast-list --checksum --update --use-server-modtime"}

validate_bucket_name() {
  local bucket_name=$1
  local regex='^[a-z0-9][a-z0-9.-]*$'

  if ! [[ $bucket_name =~ $regex ]]; then
    echo "Invalid bucket name: '$bucket_name'. Must consist of lowercase letters, numbers, dots, and hyphens."
    exit 1
  fi

  return 0
}

backup() {
  echo backup args: config file: "$config_file", source: "$source_name", dest: "$dest_name", bucket prefix: "$bucket_prefix"
  validate_bucket_name "$bucket_prefix"

  local buckets=$(rclone --config "$config_file" lsjson "$source_name")
  echo "found $(jq length -n $buckets) buckets"

  jq -r '.[].Name' -n $buckets | while read bucket; do
    echo "syncing bucket $source_name$bucket to $dest_name$bucket_prefix$bucket"
    rclone --config "$config_file" sync $rclone_args "$source_name$bucket" "$dest_name$bucket_prefix$bucket"
  done
}

restore() {
  echo restore args: config file: "$config_file", source: "$source_name", dest: "$dest_name", bucket prefix: "$bucket_prefix"
  validate_bucket_name "$bucket_prefix"

  local buckets=$(rclone --config "$config_file" lsjson "$dest_name")
  echo "found $(echo "$buckets" | jq length) buckets"

  echo "$buckets" | jq -r '.[].Name' | while read bucket; do
    # filter the buckets that have the prefix
    if ! [[ $bucket =~ ^$bucket_prefix ]]; then
      echo "skipping bucket $bucket"
      continue
    fi

    # remove the prefix from the bucket name
    local src_bucket=${bucket#$bucket_prefix}

    echo "syncing bucket $dest_name$bucket to $source_name$bucket"
    rclone --config "$config_file" sync $rclone_args "$dest_name$bucket" "$source_name$src_bucket"
  done
}

# switch on the first argument
case "$1" in
backup)
  backup
  ;;
restore)
  restore
  ;;
*)
  echo "Usage: $0 {backup|restore}"
  exit 1
  ;;
esac
