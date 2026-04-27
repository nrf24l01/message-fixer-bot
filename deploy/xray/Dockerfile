FROM teddysun/xray:1.8.24

RUN apk add --no-cache python3

COPY entrypoint.sh /entrypoint.sh

ENTRYPOINT ["/bin/sh", "/entrypoint.sh"]
