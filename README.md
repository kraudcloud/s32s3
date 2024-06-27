# S32S3 Backup

s32s3 is a tool for backing up data from one S3-compatible storage to another, with encryption. It uses rclone's S3 and crypt backends to securely transfer and store data.

## Features

- Backup from one S3-compatible storage to another
- Encryption of backed-up data using rclone's crypt backend

## Configuration

Use the [helm chart](./Chart.yaml) to provide relevant configuration options.

## Parameters

### Image Configuration

| Name               | Description                         | Value                 |
| ------------------ | ----------------------------------- | --------------------- |
| `image.repository` | Repository of the image             | `ctr.0x.pt/ops/s32s3` |
| `image.pullPolicy` | Pull policy for the image           | `IfNotPresent`        |
| `image.tag`        | Tag of the image                    | `latest`              |
| `imagePullSecrets` | Image pull secrets                  | `[]`                  |
| `nameOverride`     | Override the name of the chart      | `""`                  |
| `fullnameOverride` | Override the full name of the chart | `""`                  |

### Restore

| Name              | Description         | Value   |
| ----------------- | ------------------- | ------- |
| `restore.enabled` | Enable restore mode | `false` |

### Configuration

| Name                                   | Description                                                                                                   | Value       |
| -------------------------------------- | ------------------------------------------------------------------------------------------------------------- | ----------- |
| `config.schedule`                      | Cron schedule for backups                                                                                     | `* * * * *` |
| `config.backupBucket`                  | Name of the backup bucket                                                                                     | `backups`   |
| `config.expirationDays`                | Number of days until deleted versions are removed from backups                                                | `7`         |
| `config.extraEnv`                      | Extra environment variables                                                                                   | `{}`        |
| `config.destination.access_key_id`     | Destination access key ID. Using valueFrom referencing the minio secret is recommended for easy restores.     | `{}`        |
| `config.destination.secret_access_key` | Destination secret access key. Using valueFrom referencing the minio secret is recommended for easy restores. | `{}`        |
| `config.destination.endpoint`          | Destination endpoint                                                                                          | `{}`        |
| `config.destination.region`            | Destination region                                                                                            | `{}`        |
| `config.destination.provider`          | Destination provider                                                                                          | `{}`        |
| `config.source.access_key_id`          | Source access key ID                                                                                          | `{}`        |
| `config.source.secret_access_key`      | Source secret access key                                                                                      | `{}`        |
| `config.source.endpoint`               | Source endpoint                                                                                               | `{}`        |
| `config.source.region`                 | Source region                                                                                                 | `{}`        |
| `config.source.provider`               | Source provider                                                                                               | `{}`        |
| `config.crypt.password`                | Encryption password                                                                                           | `{}`        |
| `config.crypt.password2`               | Secondary encryption password                                                                                 | `{}`        |
| `podAnnotations`                       | Annotations for pods                                                                                          | `{}`        |
| `podLabels`                            | Labels for pods                                                                                               | `{}`        |
| `podSecurityContext`                   | Security context for pods                                                                                     | `{}`        |
| `resources`                            | Resource requests and limits                                                                                  | `{}`        |
| `nodeSelector`                         | Node selector for pods                                                                                        | `{}`        |
| `tolerations`                          | Tolerations for pods                                                                                          | `[]`        |
| `affinity`                             | Affinity for pods                                                                                             | `{}`        |
