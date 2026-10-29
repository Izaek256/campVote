#!/bin/bash
set -euo pipefail

# up-wsl.sh - Workaround for Docker Compose BuildKit host network permission issues in WSL
# Tries standard compose build first, falls back to docker buildx bake with --allow=network.host

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

cd "$PROJECT_DIR"

echo "=== campVote WSL Startup Script ==="
echo "Project directory: $PROJECT_DIR"
echo ""

# Try standard compose build first (may work if entitlement already granted)
echo "Attempting standard Docker Compose build..."
if COMPOSE_BAKE=false docker compose --profile wsl up -d --build "$@"; then
    echo "✓ Build succeeded with standard Docker Compose"
    echo "Used: COMPOSE_BAKE=false docker compose --profile wsl up -d --build"
    exit 0
fi

BUILD_EXIT_CODE=$?
echo "✗ Standard build failed (exit code: $BUILD_EXIT_CODE)"
echo ""

# Fallback: use docker buildx bake with --allow=network.host
echo "Falling back to docker buildx bake with --allow=network.host..."
if COMPOSE_PROFILES=wsl docker buildx bake --allow=network.host -f docker-compose.yaml; then
    echo "✓ Build succeeded with docker buildx bake"
    echo "Starting containers..."
    if docker compose --profile wsl up -d "$@"; then
        echo "✓ Containers started successfully"
        echo "Used: COMPOSE_PROFILES=wsl docker buildx bake --allow=network.host -f docker-compose.yaml"
        echo "      then: docker compose --profile wsl up -d"
        exit 0
    else
        echo "✗ Failed to start containers"
        exit 1
    fi
else
    echo "✗ docker buildx bake also failed"
    exit 1
fi
