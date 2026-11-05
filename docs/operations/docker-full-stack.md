# Full-Stack Docker (WSL2/Linux) and Windows PostgreSQL

**Date:** 2026-11-05
**Branch:** `feature/domains-admins-uploads-email`
**Files:** `docker-compose.wsl.yaml`, `docker-compose.windows.yaml`,
`scripts/docker-stack.sh`, `package.json`, `.env.example`, `apps/web/public/.gitkeep`,
`README.md`

## Goal

The original `docker-compose.yaml` intentionally containerises **only the Go
API**. It uses `network_mode: host` so an untouched `.env` can name a PostgreSQL
on the host, and it leaves the Next.js app running with hot reload. That remains
the default for day-to-day API work.

This change adds two opt-in setups without disturbing it:

| File | What runs | Use it when |
|---|---|---|
| `docker-compose.wsl.yaml` | PostgreSQL + Go API + Next.js web, on a private bridge network | You want the whole product running behind one command in WSL2 or Linux |
| `docker-compose.windows.yaml` | PostgreSQL only, published on `127.0.0.1:5432` | Docker Desktop on Windows runs the database; API and web run on the host or in WSL |

## Networking decisions

### Full stack uses a bridge, not host networking

The full stack needs service-name DNS, so all three services join a dedicated
bridge network named `campvote`:

- `db` is reached by the API at `db:5432`, not `localhost`.
- `api` is reached by the web build through the browser-facing
  `http://localhost:8080`; `NEXT_PUBLIC_API_URL` is baked into the bundle, so it
  must be the address the browser uses, never the internal service name.
- `web` is published on `3000`, `api` on `8080`.
- PostgreSQL is **not** published in the full stack. Use
  `docker compose exec db psql` to reach it, or add a port mapping deliberately.

`CORS_ORIGINS`, `APP_URL` and `API_URL` are overridden to the published
addresses so the API's own host-network defaults from `.env` cannot leak into
the containers.

### The database is local to the full stack

`docker-compose.wsl.yaml` uses fixed development credentials
(`campvote_dev` / `campvote_dev_password`) and overrides `DATABASE_URL` inside the
API container. It never reads or writes the external PostgreSQL named by `.env`.
That keeps the stack reproducible and avoids URL-encoding a password into a
Compose-generated DSN.

These credentials are for local development only and are not a production
secret. `down -v` deletes the database with the other volumes.

### Windows database keeps the app's own `.env`

`docker-compose.windows.yaml` containerises only PostgreSQL. It reads
`POSTGRES_DB`, `POSTGRES_USER` and `POSTGRES_PASSWORD` from `.env` with local
defaults, publishes the port on loopback, and leaves `DATABASE_URL` alone. The
operator points `DATABASE_URL` at `localhost:5432` and percent-encodes any
`@`, `%`, `/` or `#` in the password, as the driver requires.

The port is bound to `127.0.0.1`, so the database is not exposed on the campus
network.

## Build-time network workaround

`scripts/docker-stack.sh` builds both images directly:

```bash
docker build --network=host --allow=network.host -t campvote-api:latest ./apps/api
docker build --network=host --allow=network.host \
  --build-arg NEXT_PUBLIC_API_URL=http://localhost:8080 \
  -t campvote-web:latest -f apps/web/Dockerfile .
```

Docker 29 requires `--allow=network.host` for a host-network build, and Compose
cannot grant that entitlement. It is also the only path on this machine, whose
default bridge has no DNS or egress: the web image's `pnpm install` and the API's
`apk add`/`go mod download` both need the network. The script then runs
`docker compose -f docker-compose.wsl.yaml up -d --no-build`, so Compose starts
the exact images just built.

`pnpm docker:stack` wraps `up`; `pnpm docker:stack:build` builds only;
`pnpm docker:stack:down` stops the stack.

## Commands

```bash
# Full stack
pnpm docker:stack
docker compose -f docker-compose.wsl.yaml ps
docker compose -f docker-compose.wsl.yaml logs -f api web
docker compose -f docker-compose.wsl.yaml exec api ./bootstrap ...  # first admin
docker compose -f docker-compose.wsl.yaml exec api ./seed           # optional demo data
docker compose -f docker-compose.wsl.yaml exec db psql -U campvote_dev -d campvote_dev
docker compose -f docker-compose.wsl.yaml down     # keeps volumes
docker compose -f docker-compose.wsl.yaml down -v  # destructive

# Windows PostgreSQL only
docker compose -f docker-compose.windows.yaml up -d
docker compose -f docker-compose.windows.yaml ps
docker compose -f docker-compose.windows.yaml down
docker compose -f docker-compose.windows.yaml down -v  # destructive
```

## Verification run

Checks run on this Linux machine (Docker 29.9.2, Compose v5.6.1) with the
repository's `.env`. The full-stack compose file overrides `DATABASE_URL`, so
the operator's external PostgreSQL was not touched.

| Check | Result |
|---|---|
| `docker compose -f docker-compose.wsl.yaml config` | three services; `DATABASE_URL` points at `db:5432`; API and web published on 8080/3000 |
| `docker compose -f docker-compose.windows.yaml config` | one `postgres` service; loopback port 5432; health check reads container env. Not started here: the host's own PostgreSQL already holds 5432. |
| `docker build ... -t campvote-api:latest ./apps/api` | exported successfully |
| `docker build ... -t campvote-web:latest -f apps/web/Dockerfile .` | Next.js production build completed and the image exported. The first attempt failed because `apps/web/public` did not exist; `apps/web/public/.gitkeep` now keeps the Dockerfile's `COPY` valid. |
| `docker compose -f docker-compose.wsl.yaml up -d --no-build` | `db`, `api` and `web` started; API and database reached `healthy`, then web started |
| `GET http://localhost:8080/api/health` | `status: ok`; `database.ok` with 67 tables on the compose `db` container |
| `GET http://localhost:3000/` | HTTP 307 to `/elections`, which returns 200; `/login` returns 200 |
| Web health check | initially failed because Alpine resolved `localhost` to `::1` while Next listened on IPv4 only; changed the probe to `127.0.0.1:3000` and the container reached `healthy` |
| Logs | API connected to the compose `db`; PostgreSQL was ready on `5432` inside the network |

Supporting checks for the same working tree:

| Check | Result |
|---|---|
| `cd apps/api && go build ./...` | PASS |
| `go test ./internal/config ./internal/settings ./internal/mail -count=1` | PASS |
| `pnpm --filter @campvote/web type-check` | PASS after `pnpm install --frozen-lockfile` |
| `pnpm --filter @campvote/web test` | PASS, 25/25 tests in 4 files |
| `go test ./internal/handlers` | Pre-existing failure in `TestAuthHandler_Login_Success` (401 after register). The same test fails on `HEAD` in a detached worktree, so it is not caused by this change. |

## Out of scope / deferred

- No Redis, MinIO, Mailpit or reverse proxy in the full stack. `REDIS_URL` is
  empty, so caching, jobs and scheduled election transitions are off; elections
  open and close lazily on request.
- No production image hardening, TLS, secrets manager or backup policy. The
  full-stack file pins local passwords on purpose and is not a deployment file.
- The Windows compose file does not run the API or web app. Doing that on Windows
  without WSL is a separate deployment decision, not a local-development one.
- `go test ./internal/handlers` still has the pre-existing login test failure
  described above; fixing that test harness is separate work.
