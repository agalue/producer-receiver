FROM golang:alpine AS builder
WORKDIR /app
ADD ./ /app/
RUN echo "@edgecommunity http://dl-cdn.alpinelinux.org/alpine/edge/community" >> /etc/apk/repositories && \
    apk update && \
    apk add --no-cache alpine-sdk git librdkafka-dev@edgecommunity && \
    CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -tags static_all,netgo,musl -o producer-receiver .

FROM alpine
COPY --from=builder /app/producer-receiver /bin/producer-receiver
COPY ./docker-entrypoint.sh /
RUN apk update && \
    apk add --no-cache bash && \
    rm -rf /var/cache/apk/* && \
    addgroup -S onms && \
    adduser -S -G onms onms && \
    chmod +x /bin/producer-receiver
USER onms
LABEL maintainer="Alejandro Galue <agalue@opennms.org>" \
      name="OpenNMS Kafka Producer Receiver"
ENTRYPOINT [ "/docker-entrypoint.sh" ]
