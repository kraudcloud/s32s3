FROM golang:1.23.2-alpine3.20 AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go build -o /bin/s32s3

FROM rclone/rclone:1.68.1

COPY --from=builder /bin/s32s3 /bin/s32s3

ENTRYPOINT [ "/bin/s32s3" ]