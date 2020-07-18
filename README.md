# OpenNMS Kafka Producer Receiver [![Go Report Card](https://goreportcard.com/badge/github.com/agalue/producer-receiver)](https://goreportcard.com/report/github.com/agalue/producer-receiver)

A sample Kafka Consumer application to display the producer payload in JSON format into standard output for troubleshooting purposes.

This solution requires using the OpenNMS Kafka Producer. This feature can export events, alarms, metrics, nodes and edges from the OpenNMS database to Kafka. All the payloads are stored using Google Protobuf.

It exposes Prometheus compatible metrics through port 8181, using the `/metrics` endpoint.

This repository also contains a Dockerfile to compile and build an image with the tool, which can be fully customized through environment variables.

The `protobuf` directory contains the GPB definitions extracted from OpenNMS source code contains. If any of those files change, make sure to re-generate the protobuf code by using the [build.sh](protobuf/build.sh) command, which expects to have `protoc` installed on your system.

## Requirements

* `BOOTSTRAP_SERVERS` environment variable with Kafka Bootstrap Server (i.e. `kafka01:9092`)
* `SOURCE_TOPIC` environment variable with the source Kafka Topic with GPB Payload
* `GROUP_ID` environment variable with the Consumer Group ID (defaults to `opennms`)
* `MESSAGE_KIND` environment variable with the payload type. Valid values are: alarm, event, node, metric, edge (defaults to `alarm`).
* To pass consumer settings, add an environment variable with the prefix `KAFKA_`, for example: `KAFKA_AUTO_OFFSET_RESET`.

For consumer settings, the character underscore will be replaced with a dot and converted to lowercase. For example, `KAFKA_AUTO_OFFSET_RESET` will be configured as `auto.offset.reset`.

## Build

In order to build the application:

```bash
docker build -t agalue/producer-receiver-go:latest .
docker push agalue/producer-receiver-go:latest
```

> *NOTE*: Please use your own Docker Hub account or use the image provided on my account.

To build the controller locally for testing:

```bash
export GO111MODULE="on"

go build
./producer-receiver
```
