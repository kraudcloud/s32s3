FROM rclone/rclone

RUN apk add --no-cache bash jq curl

COPY ./entrypoint.sh /entrypoint.sh

ENTRYPOINT [ "/entrypoint.sh" ]