#!/usr/bin/env bash

validate_bucket_name() {
    local bucket_name=$1
    local regex='^[a-z0-9][a-z0-9.-]*$'

    if ! [[ $bucket_name =~ $regex ]]; then
        echo "Invalid bucket name: '$bucket_name'. Must consist of lowercase letters, numbers, dots, and hyphens."
        exit 1
    fi

    return 0
}

config_file=${RCLONE_CONFIG_PATH-/etc/rclone.conf}
source_name=${RCLONE_SOURCE_NAME-source:}
dest_name=${RCLONE_DESTINATION_NAME-dest:}
bucket_prefix=${RCLONE_BUCKET_PREFIX?"RCLONE_BUCKET_PREFIX is required"}
rclone_args=${RCLONE_ARGS-"--fast-list --checksum --update --use-server-modtime"}

validate_bucket_name "$bucket_prefix"

echo config file: "$config_file", source: "$source_name", dest: "$dest_name"

# list buckets as json with rclone
buckets=$(rclone --config "$config_file" lsjson "$source_name")

echo "found $(echo "$buckets" | jq length) buckets"

echo "$buckets" | jq -r '.[].Name' | while read bucket; do
    echo "syncing bucket $source_name$bucket to $dest_name$bucket_prefix$bucket"
    rclone --config "$config_file" sync $rclone_args "$source_name$bucket" "$dest_name$bucket_prefix$bucket"
done
