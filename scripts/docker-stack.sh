#!/bin/sh
# Build and control the full WSL/Linux development stack.
#
# Docker 29 refuses a host-network build unless the CLI grants the entitlement
# ("pass --allow=network.host"), and compose cannot grant it. On hosts whose
# default bridge has no DNS or egress, building directly with both flags is the
# only reliable path; the images are tagged exactly as docker-compose.wsl.yaml
# expects, and `up --no-build` then uses them.
#
#   ./scripts/docker-stack.sh build   # API and web images
#   ./scripts/docker-stack.sh up      # build, then start db + api + web
#   ./scripts/docker-stack.sh down    # stop the stack; volumes are kept
set -eu

cd "$(dirname "$0")/.."

COMPOSE_FILE=docker-compose.wsl.yaml
API_IMAGE=campvote-api:latest
WEB_IMAGE=campvote-web:latest
WEB_API_URL=${NEXT_PUBLIC_API_URL:-http://localhost:8080}

build() {
    docker build \
        --network=host \
        --allow=network.host \
        -t "$API_IMAGE" \
        ./apps/api

    docker build \
        --network=host \
        --allow=network.host \
        --build-arg "NEXT_PUBLIC_API_URL=$WEB_API_URL" \
        -t "$WEB_IMAGE" \
        -f apps/web/Dockerfile \
        .
}

case "${1:-up}" in
    build)
        build
        ;;
    up)
        build
        docker compose -f "$COMPOSE_FILE" up -d --no-build
        docker compose -f "$COMPOSE_FILE" ps
        ;;
    down)
        docker compose -f "$COMPOSE_FILE" down
        ;;
    *)
        echo "usage: $0 [build|up|down]" >&2
        exit 2
        ;;
esac
