#!/bin/sh
# entrypoint.sh — run migrations then start the server.
# Keep it minimal: no bashisms, no set -e trap games.
set -e

echo "[entrypoint] Running database migrations..."
./migrate

echo "[entrypoint] Starting server..."
exec ./server
