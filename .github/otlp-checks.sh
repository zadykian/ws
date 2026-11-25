#!/bin/bash
# The server and host jobs' check that Docker's containers reach the collector, run as root after
# setup, with the cases' image as $1. A container sends an empty OTLP/gRPC export to the bridge
# address, which the collector takes through otelcol-docker.socket and answers with grpc-status 0.
# The control sends to OTLP/HTTP's port, where no socket listens, and is refused.
# ubuntu/roles/otelcol/README.md#docker's-containers
set -euo pipefail

image=$1
bridge=$(docker network inspect -f '{{(index .IPAM.Config 0).Gateway}}' bridge)

# send posts the request, a frame of no bytes, and succeeds where the reply's trailers say 0.
send() {
    docker run --rm "$image" sh -ec '
        apk add -q --no-cache curl
        printf "\0\0\0\0\0" | curl --silent --show-error --max-time 10 --http2-prior-knowledge \
            --header "content-type: application/grpc" --header "te: trailers" --data-binary @- \
            --dump-header - --output /dev/null \
            "http://$1:$2/opentelemetry.proto.collector.logs.v1.LogsService/Export" |
            grep -qiE "^grpc-status: 0[[:space:]]*$"' sh "$bridge" "$1"
}

send 4317 || {
    echo "::error::a container's OTLP/gRPC export to $bridge:4317 was not taken"
    exit 1
}
echo "ok: a container's OTLP/gRPC export reaches the collector"
if send 4318; then
    echo "::error::a container reached $bridge:4318, where no socket listens"
    exit 1
fi
echo "ok: the bridge address has no OTLP/HTTP port"
