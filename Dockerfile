FROM golang:1.22-alpine3.20 as builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY src/ .

RUN go build -o /bin/s32s3

FROM rclone/rclone:1.63.1

COPY --from=builder /bin/s32s3 /bin/s32s3

ENTRYPOINT [ "/bin/s32s3" ]