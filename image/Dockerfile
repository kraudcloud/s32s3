FROM rclone/rclone

RUN apk add --no-cache bash jq

COPY ./entrypoint.sh /entrypoint.sh

ENTRYPOINT [ "/entrypoint.sh" ]