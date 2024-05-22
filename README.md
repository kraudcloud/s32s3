# rclone s3 to s3 with crypt

This repository is me messing around with using rclone for encrypted s3 to s3 backup. The destination should never be able to read the objects nor own the decryption key.

```sh
export KUBECONFIG=$(pwd)/kubeconfig.yaml
kind create cluster --config kind.yaml
```

```sh
helmfile apply
```

Now add objects/buckets to `bitnami-minio.minio:9001` (port forward to the UI or something)

Create a user on both minios and set the creds in [rclone.yaml](./rclone.yaml) the secret.

```sh
kubectl apply -f rclone.yaml
```

The backup is now running in the background
