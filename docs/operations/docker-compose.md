# Docker Compose — API Container Only

**Date:** 2026-11-05
**Branch:** `feature/domains-admins-uploads-email`
**Files:** `docker-compose.yaml`, `apps/api/Dockerfile`, `apps/api/entrypoint.sh`,
`scripts/docker-api.sh`, `package.json`, `.env.example`, `README.md`,
`docs/MVP_NOTES.md`, `docs/README.md`

**Related:** [`docker-full-stack.md`](docker-full-stack.md) covers the opt-in
WSL2/Linux full stack (`docker-compose.wsl.yaml`) and the Windows Docker Desktop
PostgreSQL setup (`docker-compose.windows.yaml`). This document remains about
the API-only default.

## Goal

`docker-compose.yaml` containerises **the Go API and nothing else**. The database
is external: a PostgreSQL the operator already runs, named by `DATABASE_URL` in
`.env`. The previous version also built a web image and ran Redis, Mailpit and —
in the first cut of this change — its own `postgres` container, so a frontend edit
cost an image rebuild and `up` started services nobody had asked for.

## What the file contains now

| Service | Image | Notes |
|---|---|---|
| `api` | built from `apps/api/Dockerfile`, tagged `campvote-api:latest` | The only service. `network_mode: host`, `env_file: .env`, one volume at `/app/storage`. |

Removed since the original five-service file: `web`, `redis`, `mailpit`, and the
`postgres` container that this change briefly added back. With them went the web
image's `NEXT_PUBLIC_API_URL` build argument, the `depends_on` health gate, the
top-level `networks:` block, and both `ports:` mappings.

## Configuration: `.env` is the file, unchanged

`env_file: .env` hands the container exactly what a host-run `pnpm dev:api` gets.
Nothing is interpolated with `${VAR:-default}`, reassembled, or overridden, so
there is one place to look when the API misbehaves and no divergence between the
two ways of running it. The cost is that `.env` is now a prerequisite for
`docker compose up`, and that a missing `DATABASE_URL` or a database that is not
listening fails at the entrypoint rather than being papered over by a default.

`POSTGRES_PASSWORD` is in `.env` in some setups but is read by no application
code; it exists as a note and as the `-e` value for starting a database with
`docker run`. Compose no longer uses it at all, which also retires the constraint
described under "Deferred" below.

## Why `network_mode: host`

An untouched `.env` and a bridge network cannot both be true. `.env` says
`DATABASE_URL=…@localhost:5432/…`, and on the default bridge network `localhost`
is the container, so the API would look for the database in its own empty network
namespace. Sharing the host namespace makes every `localhost` in `.env` mean the
machine — which is what it already means when the API runs on the host.

Consequences, all documented in the README:

- `ports:` would be ignored, so it is not in the file. The API binds the host's
  `8080` directly, and nothing else may hold it, `pnpm dev:api` included.
- No published-port conflict with a host PostgreSQL on `5432`, which was the
  single biggest obstacle when the compose file ran its own database here.
- The container can reach every port on the machine. The alternative — bridge plus
  `extra_hosts: ["host.docker.internal:host-gateway"]` — keeps that isolation but
  needs `host.docker.internal` in `DATABASE_URL`, and that name does not resolve
  outside Docker, so the host run would need a second DSN. Chosen: fewer moving
  parts and one configuration, on a local development machine.

## Entrypoint: arguments now replace the server

`apps/api/entrypoint.sh` ran `./migrate` then `exec ./server` and ignored `$@`, so
the documented `docker compose run ... api ./bootstrap` started a second API
instead of creating an admin — the image's `ENTRYPOINT` form swallows the argument.
It now runs the migrations, then `exec "$@"` when given a command. `./seed`,
`./bootstrap` and `sh` are reachable through `docker compose run` (and were always
reachable through `docker compose exec`, which bypasses the entrypoint).

## What is not containerised, and what that costs

- **Redis.** Whatever `REDIS_URL` in `.env` says. Left out entirely, the API falls
  back to `redis://localhost:6380` and dials it; with host networking that at
  least reaches a Redis on the machine, and if none runs, `/api/health` reports
  `redis: off` with the reason. Setting `REDIS_URL=` (empty, present) is the
  deliberate "no Redis" and skips the cache client, the asynq worker and cron in
  `cmd/server/main.go`, while `internal/routes` leaves the realtime hub
  in-process. Elections then open and close **lazily**, when a request reaches
  `lifecycle.Ensure`, rather than on a schedule.
- **Mailpit / SMTP.** `MAIL_MAILER` picks the driver. Unset in development means
  "try SMTP on `localhost:1025`, then fall back to the log"
  (`internal/mail/from_config.go`); with the log driver nothing is sent, each
  message is printed to the container log and an HTML copy is written under
  `MAIL_LOG_PATH`. Verification codes are read with `docker compose logs api`
  instead of at `http://localhost:8025`. The log transport is refused when
  `APP_ENV=production` unless `MAIL_ALLOW_LOG_IN_PRODUCTION=true`, because reset
  links would sit in a log.
- **Web app.** `pnpm install && pnpm dev:web` → `http://localhost:3000`, with
  `CORS_ORIGINS` in `.env` listing that origin. `apps/web/Dockerfile` is untouched
  and still builds a standalone image for any deployment that wants one; it is
  simply no longer part of the dev loop.
- **PostgreSQL.** Started and backed up by the operator. `/api/health` measures it
  like any other dependency, so the table count and latency in the response are
  the external database's.

## Image change

`apps/api/Dockerfile` creates `/app/storage/app` and `/app/storage/mail` and chowns
`/app/storage` to the unprivileged `app` user before `USER app`. Both are written
to at runtime: the `STORAGE_DRIVER=local` root (`STORAGE_LOCAL_ROOT=storage/app`)
and the log mailer's directory (`MAIL_LOG_PATH=storage/mail`), each resolved
against `WORKDIR /app`. `/app` is root-owned, so without this the API boots
happily and the first upload or email fails with a permission error — a 500 at
runtime instead of anything visible at startup.

Because Docker seeds a named volume's ownership from the image on first creation,
a `campvote_storage` volume created **before** this change keeps root ownership;
`docker volume rm campvote_campvote_storage` recreates it app-owned.

Mounting one volume over `/app/storage` rather than one per directory means a
single `down -v` clears both uploads and mail, and the config defaults need no
container-specific overrides at all.

## Build-time network: Docker 29's entitlement gate

The image build runs `apk add` and `go mod download`, which need DNS and
outbound HTTPS. On this machine the default Docker bridge has neither (a
container on it cannot resolve `dl-cdn.alpinelinux.org`, while the same container
with `--network host` can), so a plain build fails at the first `apk add` with
`DNS: transient error`.

The obvious fix is a host-network build, but Docker 29 / BuildKit refuses one
unless the CLI grants the entitlement:

```text
additional privileges requested: pass "--allow=network.host" to grant requested privileges
```

`build.entitlements: [network.host]` in `docker-compose.yaml` only *requests* the
entitlement; it does not grant it, and Compose v5 has no `--allow` flag for
`docker compose build`. Adding it to the compose file therefore made
`docker compose build` fail on every host, including those whose bridge network
works. It was removed, and `scripts/docker-api.sh` (exposed as
`pnpm docker:build` / `pnpm docker:up`) builds directly:

```bash
docker build --network=host --allow=network.host -t campvote-api:latest ./apps/api
docker compose up -d --no-build
```

The tag matches what Compose expects, so the started container is the freshly
built image. The runtime `network_mode: host` remains, so a `localhost` in
`DATABASE_URL` reaches the external PostgreSQL.

## Commands

```bash
pnpm docker:up                        # build with host networking, then start; migrations run on start
# On a host whose default bridge has DNS/egress, this plain form is enough:
docker compose up --build -d          # build and start the API; migrations run on start
docker compose ps                     # api healthy
docker compose logs -f api            # server log, including mailed codes
docker compose exec api ./seed        # optional demo data, against the external database
docker compose run --rm -e BOOTSTRAP_ADMIN_EMAIL=... -e BOOTSTRAP_ADMIN_PASSWORD=... api ./bootstrap
docker compose down                   # removes the container; the external database is untouched
docker compose down -v                # also deletes the uploads/mail volume
```

## Verification run

Ran on this machine against its own PostgreSQL (`127.0.0.1:5432`, database
`campvote_dev`) using the operator's real `.env`, unmodified (Docker 29.9.2,
Compose v5.6.1).

| Check | Result |
|---|---|
| Bridge vs host DNS | `alpine:3.20` on the default bridge could not resolve `dl-cdn.alpinelinux.org` (timed out); the same container with `--network host` resolved it and fetched `APKINDEX.tar.gz` over HTTPS. The Go proxy (`proxy.golang.org`) answered from both host and a host-network container. |
| `docker compose build api` | refused with `additional privileges requested: pass "--allow=network.host" to grant requested privileges` — the reason the compose `entitlements` key was removed and the build moved to the script |
| `pnpm docker:build` | exported `campvote-api:latest` successfully; `apk add` and the cached `go mod download` layer both resolved under host networking |
| `pnpm docker:up` | built the image, started `campvote-api-1`, and printed `Up (health: starting)` before the health check settled |
| `docker compose config` | one service; the environment it prints is the `.env` file's own values, with nothing added |
| `docker compose ps` | `campvote-api-1` `Up (healthy)`; no other container started, no ports published |
| `GET http://localhost:8080/api/health` from the host | `status: ok`; `database.ok` with 67 tables over the external database; `redis.state: off` ("Redis was not reachable when the API started… set `REDIS_URL=` in .env to mean it"); `jobs.state: off`; `email.driver: failover(smtp,log)` ok; `storage.driver: local` ok |
| Boot log | `Database connected successfully` against the host's PostgreSQL, proving host networking carries `localhost:5432` out of the container |
| Second start | `Migrations completed successfully. The schema was already up to date.` — idempotent against a real, already-migrated database |
| Writability as `app` | `uid=100(app)`; `touch` succeeded in `/app/storage/app` and `/app/storage/mail`, both owned `app:app` |
| `docker compose run --rm api echo …` | logged `Running echo … instead of the server` and printed the argument — the pass-through fix above |
| `APP_URL` warning | `WARNING: APP_URL is http://localhost:3000 but this server listens on port 8080` appeared because this `.env` points `APP_URL` at the frontend; stored-file URLs are rewritten to `NEXT_PUBLIC_API_URL` by `apps/web/lib/api.ts`, so files still open. Documented rather than overridden. |

A full `--no-cache` rebuild was also attempted to prove a cold `go mod download`
under host networking. It completed the `apk add` stage but was still inside
`go mod download` after 12+ minutes on this link and was stopped; it had not
errored. The normal build reused the validated module layer, so the Dockerfile
itself was exercised end to end without a code change. Treat a cold build on a
slow connection as slow, not hung, unless it prints a network error.

Not run: CI. `.github/workflows/ci.yml` builds no Docker image, so nothing here is
covered by the pipeline; the checks above are the manual ones listed in this
section. No Go or TypeScript source changed, so the Go/frontend suites were not
re-run. Signup/bootstrap were deliberately **not** exercised this time: the
database is a real development database with existing accounts, and inserting a
test administrator or voter there would leave data behind that `down -v` cannot
remove.

The container was left running for the operator; `docker compose down` stops and
removes it and leaves the external database alone.

## Out of scope / deferred

- No `scripts/up-wsl.sh`, no `--profile wsl` / `--profile container`. The README
  sections describing those were stale (they referenced a deleted script and
  profiles removed when the compose file was rewritten for the MVP) and were
  rewritten against the actual file.
- README Option 3 (apps on the host, database in a container) now uses a plain
  `docker run` for PostgreSQL, since the compose file has no database to start.
- Passwords with characters that need percent-encoding (`@`, `%`, `/`, `#`) still
  have to be written percent-encoded **inside `DATABASE_URL`**, because pgx
  decodes it. That is now purely a `.env`-spelling matter: the value is no longer
  duplicated into a database container's account password, so the "cannot work in
  both places at once" trap is gone.
- `docker compose config` prints the resolved environment, secrets included. Fine
  locally, wrong for a bug report; the README troubleshooting section says so.
- `docs/traceability.md` still cites `compose.dev.yaml` (deleted) as evidence for
  NFR-SEC-01 and NFR-SEC-06. Historical record, left as written.
