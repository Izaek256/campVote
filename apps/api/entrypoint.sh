#!/bin/sh
# entrypoint.sh — run migrations then start the server.
# Any argument replaces the server, so `docker compose run api ./bootstrap`
# (or ./seed, or sh) stays a way to reach the image's other commands: the
# ENTRYPOINT form would otherwise swallow the argument and boot the API.
# Keep it minimal: no bashisms, no set -e trap games.
set -e

echo "[entrypoint] Running database migrations..."
./migrate

if [ "$#" -gt 0 ]; then
    echo "[entrypoint] Running $* instead of the server..."
    exec "$@"
fi

echo "[entrypoint] Starting server..."
exec ./server
