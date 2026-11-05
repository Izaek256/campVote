#!/bin/sh
# Build the API image, and optionally start it with Compose.
#
# Why this exists instead of `docker compose build`:
#   * Docker 29 refuses a host-network build unless the CLI grants the
#     entitlement ("pass --allow=network.host"). compose.yaml cannot grant it;
#     its build.entitlements key only requests it.
#   * On some hosts, including the development machine this was written on, the
#     default Docker bridge has no DNS or outbound egress, so `apk add` in the
#     Dockerfile fails with "DNS: transient error" and `go mod download` would
#     fail too. The host network reaches the package mirrors and the Go proxy.
#
# The image is tagged campvote-api:latest, which is exactly what
# docker-compose.yaml expects. The runtime container still uses
# network_mode: host, so DATABASE_URL's localhost reaches the external Postgres.
#
#   ./scripts/docker-api.sh build   # image only
#   ./scripts/docker-api.sh up      # build, then start without rebuilding
set -eu

cd "$(dirname "$0")/.."

IMAGE=campvote-api:latest

build() {
    docker build \
        --network=host \
        --allow=network.host \
        -t "$IMAGE" \
        ./apps/api
}

case "${1:-up}" in
    build)
        build
        ;;
    up)
        build
        docker compose up -d --no-build
        docker compose ps
        ;;
    *)
        echo "usage: $0 [build|up]" >&2
        exit 2
        ;;
esac
