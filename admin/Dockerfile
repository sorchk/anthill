FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY tcp-admin /app/tcp-admin

EXPOSE 8080

ENV TZ=Asia/Shanghai
ENV GIN_MODE=release

CMD ["/app/tcp-admin"]