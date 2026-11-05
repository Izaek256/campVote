# campVote — MMU Campus E-Voting System

Repository: <https://github.com/Izaek256/campVote>

campVote is a proposed web-based voting platform for guild elections and other authorized campus elections at Mountains of the Moon University (MMU). This repository is intended for an academic project and prototype. It is **not certified for real or legally binding elections**.

## Start here

1. Read [`AGENTS.md`](AGENTS.md) and [`CONTRIBUTING.md`](CONTRIBUTING.md). These govern AI-assisted and human contributions.
2. Read [`SRS.md`](SRS.md) for requirements, [`STACK.md`](STACK.md) for the proposed stack, and [`ADR.md`](ADR.md) for recorded architecture decisions.
3. Review [`docs/implementation/ROADMAP.md`](docs/implementation/ROADMAP.md) and the open questions in [`docs/implementation/OPEN_QUESTIONS.md`](docs/implementation/OPEN_QUESTIONS.md).
4. Check out `develop` and create a short-lived feature branch from it. Do not work directly on `main` or `develop`.
5. Verify the installed Grit CLI and its current Triple scaffold options before generating code. See `STACK.md`; do not assume commands or generated paths from an unpinned release.
6. Generate the Grit project in a temporary directory first, inspect the scaffold, and then merge these project documents into the repository without overwriting generated configuration blindly.
7. Record actual tool versions, setup commands, and generated folder paths in `docs/implementation/SETUP_LOG.md`.

## Project documentation

- [`SRS.md`](SRS.md) — functional and non-functional requirements.
- [`STACK.md`](STACK.md) — proposed stack and implementation guidance.
- [`ADR.md`](ADR.md) — architecture decisions and verification steps.
- [`docs/implementation/ROADMAP.md`](docs/implementation/ROADMAP.md) — suggested delivery order and acceptance gates.
- [`docs/implementation/OPEN_QUESTIONS.md`](docs/implementation/OPEN_QUESTIONS.md) — questions requiring confirmation, especially institutional identity and eligibility.
- [`docs/implementation/SETUP_LOG.md`](docs/implementation/SETUP_LOG.md) — actual scaffold/setup record.
- [`docs/README.md`](docs/README.md) — documentation index and logging conventions.

## Branch and pull-request policy

- `main`: stable/release branch.
- `develop`: integration/staging branch.
- `feature/<issue>-short-name`: feature branches created from `develop`.
- `fix/<issue>-short-name`: ordinary fixes created from `develop`.
- `hotfix/<issue>-short-name`: urgent production fixes created from `main`; after merging, sync the fix back to `develop`.
- Open PRs for all changes. Feature/fix PRs target `develop`; release PRs target `main` from `develop`.
- Link every PR to an issue and include tests, docs, screenshots where relevant, and risks.

## Technology and prerequisites

The proposed stack is Grit Triple (public web + admin + Go API), Go/Gin/GORM, TypeScript/Next.js, PostgreSQL, and Docker Compose. Confirm the exact generated stack and version requirements against the pinned Grit CLI before implementing. Redis, object storage, email preview, and realtime services are optional until justified by requirements.

Install current supported versions of Git, Docker Engine/Desktop with Compose, Go, Node.js and the package manager required by the generated scaffold, plus the Grit CLI. Record the actual versions in `docs/implementation/SETUP_LOG.md`.

## Setup options

You can run campVote in several ways:

1. **Docker Compose (API container only)** - the Go API is the only container; its database is external, from `.env`, and the Next.js app runs on your machine
2. **API container with `docker run`** - the same single container without Compose, for a managed database or a host that has no Compose
3. **Local development** - run the Go API and Next.js apps locally, against PostgreSQL in Docker or installed locally

Choose the setup that fits your workflow.

---

## Option 1: Docker Compose (API Container Only)

This is the Docker setup the repository ships: **the Go API is the only container in `docker-compose.yaml`**. Nothing else is in the file — the database is external, a PostgreSQL that `.env`'s `DATABASE_URL` already points at, whether it runs on this machine or somewhere else.

The Next.js app is deliberately not containerised either — run it on the host and it talks to the API at `http://localhost:8080`. `apps/web/Dockerfile` stays in the repository for a deployment that wants a built web image; it is simply not what day-to-day development uses, because the frontend rebuilds on every edit and hot reload is worth more than container parity.

### Prerequisites

- Docker Engine with Docker Compose
- A running PostgreSQL the API can connect to, with its database already created
- Node.js 20+ and pnpm (for the web app on the host)
- Git

### Step-by-step setup

```bash
# 1. Clone the repository
git clone https://github.com/Izaek256/campVote.git
cd campVote

# 2. Set up environment variables
cp .env.example .env
# Edit .env. This file is the container's whole configuration: compose reads it
# with `env_file`, so every value the API sees is exactly what is written here,
# interpolated, rebuilt or overridden nowhere.
#   DATABASE_URL   required, and it must point at a database that is listening.
#                  This is the same string the host-run API uses, unchanged.
#   JWT_SECRET     at least 32 characters: openssl rand -hex 32
#   REDIS_URL      set it empty to switch the cache, job queue and cron off on
#                  purpose. Left out entirely, the API falls back to
#                  redis://localhost:6380 and dials that whether or not Redis runs.
#   STORAGE_DRIVER set to local to choose the disk directly; unset, the default is
#                  minio, which probes for MinIO and then falls back to the disk.
#   MAIL_MAILER    log prints each message instead of sending it; unset in
#                  development means "try SMTP on localhost:1025, then the log".
#   SEED_DEMO_DATA true seeds synthetic demo accounts (development only)

# 3. Build and start the API container
docker compose up --build -d      # or: docker compose up -d, if the image is built

# 4. Check service status — api should report healthy
docker compose ps

# 5. Run the frontend on the host
pnpm install
pnpm dev:web   # http://localhost:3000
```

The image's entrypoint (`apps/api/entrypoint.sh`) runs the migrations against `DATABASE_URL` before starting the server, so a first `up` needs no separate migrate step.

### Why the API container uses `network_mode: host`

`.env` says `DATABASE_URL=…@localhost:5432/…`, and inside a container on the default bridge network `localhost` is the container itself, so that address would point at an empty network namespace and never at the database. Sharing the host's network namespace makes every `localhost` in `.env` mean this machine — the same thing it means when the API runs on the host — so no configuration has to be written twice.

Three consequences:

- `ports:` has no effect; the API binds the host's `8080` directly.
- Nothing else may hold `8080` while the container runs, `pnpm dev:api` included.
- The container can reach every port on the machine. If that isolation matters, remove `network_mode`, add `extra_hosts: ["host.docker.internal:host-gateway"]`, and put `host.docker.internal` in `DATABASE_URL` — then a second DSN is needed for the host run, since that name does not resolve outside Docker.

### What is not containerised, and what that costs

| Left out | How the API reaches it |
|---|---|
| PostgreSQL | External, from `.env`'s `DATABASE_URL`. Start it yourself; the entrypoint retries until it answers. |
| Redis | Whatever `REDIS_URL` says. Empty means no cache, job queue or cron: elections open and close lazily when a request reaches them instead of on a schedule, and `/api/health` reports redis as `off`, not down. |
| Mailpit / SMTP | Whatever `MAIL_MAILER` and the `SMTP_*`/provider variables say. With the log driver nothing is sent and port 8025 does not exist: each message — verification codes and reset links included — is printed to `docker compose logs api` and saved as HTML under `MAIL_LOG_PATH` (`storage/mail` by default, inside the volume). |
| Web app | Run `pnpm dev:web`; `CORS_ORIGINS` in `.env` must list `http://localhost:3000`. |

Uploads and those HTML mail files are the container's only writable state, on the `campvote_storage` volume at `/app/storage`. `docker compose down -v` deletes it.

### One-time setup (first run only)

Migrations already ran when the container started. Two things are still yours to do, and both run inside the API container, which already carries `DATABASE_URL`:

```bash
# (Optional) Seed scaffold data. Demo elections are skipped unless
# SEED_DEMO_DATA=true is in .env and the container was recreated with it.
docker compose exec api ./seed

# Create the first administrator account
docker compose run --rm -e BOOTSTRAP_ADMIN_EMAIL=admin@must.ac.ug \
  -e BOOTSTRAP_ADMIN_PASSWORD='your-secure-password-here' \
  -e BOOTSTRAP_ADMIN_FIRST_NAME=Admin \
  -e BOOTSTRAP_ADMIN_LAST_NAME=User \
  api ./bootstrap
```

`./seed` and `./bootstrap` are prebuilt binaries in the image (`cmd/seed`, `cmd/bootstrap`), and the entrypoint runs any command it is given instead of the server, which is what makes `docker compose run api ./bootstrap` reach the tool rather than booting another API. The demo seed also runs automatically at startup when both `APP_ENV=development` and `SEED_DEMO_DATA=true`.

**Important:** `bootstrap` only works once. After an admin exists it refuses to run. Create further staff accounts from the staff pages at `/staff`.

### Access the applications

- **Voter portal & staff pages** (host, `pnpm dev:web`): http://localhost:3000
  - Voter pages: `/`, `/elections`, `/elections/[id]/ballot`, `/elections/[id]/results`
  - Admin pages: `/staff`, `/staff/elections/[id]`
- **API**: http://localhost:8080, health at `/api/health`
- **Verification codes**: `docker compose logs api` — there is no Mailpit UI on :8025 any more

### Accounts

No demo accounts or demo elections are seeded. Create the first administrator
with `cmd/bootstrap`, then register voters at `/register`:

- **Staff admin**: the email and password you set in `BOOTSTRAP_ADMIN_EMAIL` / `BOOTSTRAP_ADMIN_PASSWORD`
- **Student voter**: register a campus email such as `2024bse088@std.mast.ac.ug`

Demo seeding is opt-in only (both `APP_ENV=development` **and** `SEED_DEMO_DATA=true`).
Never use demo data in production.

### Stopping Docker Compose

```bash
# Stop and remove the API container. The external database is not part of
# compose, so it keeps running and keeps its data.
docker compose down

# Also delete the volume holding uploads and log-driver mail (destructive)
docker compose down -v

# View logs
docker compose logs -f api
```

### Troubleshooting Docker setup

```bash
# Is the container up, and is the API healthy?
docker compose ps
docker compose logs --tail=50 api

# What environment the container will get, taken from .env. This prints the
# secrets themselves, so use it locally and never paste its output into an issue.
docker compose config

# Restart the API
docker compose restart api

# Rebuild the API image after backend changes
docker compose up -d --build api

# Health, including whether redis/jobs are off on purpose
curl -s http://localhost:8080/api/health

# Complete reset: container, image cache and the storage volume
docker compose down -v
docker system prune -f
```

These failures are specific to this split setup:

- **The container restarts in a loop, `dial tcp …:5432: connect: connection refused`.** The external database is not running, or `DATABASE_URL` names a host or database that does not exist. The entrypoint runs migrations first and exits when they cannot connect; `docker compose logs api` shows the failure. Compose cannot start the database for you any more.
- **`address already in use` on port 8080.** With `network_mode: host` the container binds the machine's own port, so a host-run API (`pnpm dev:api`, or air) holding `8080` blocks it. Stop one of the two.
- **`localhost:5432` refused, although the database is running on the host.** That only happens if `network_mode: host` was removed: on a bridge network `localhost` is the container. Either keep host networking or give `DATABASE_URL` a `host.docker.internal` host together with `extra_hosts: ["host.docker.internal:host-gateway"]`.
- **`password authentication failed` against the external database.** The driver percent-decodes the password inside `DATABASE_URL`, so a real password containing `@`, `%`, `/` or `#` must be written in the URL percent-encoded (`@` as `%40`), while the same string pasted into a GUI client must be written raw. Compose no longer builds any DSN, so this is purely about how `.env` spells it — the password is never copied to a container as an account secret.
- **Uploads fail with a permission error on an older install.** `/app/storage` is a named volume, and Docker takes its ownership from the image the first time it is created. A volume created before the image owned that directory stays root-owned, so `docker volume rm campvote_campvote_storage` and restart to get an app-owned one.
- **The web app on the host cannot sign in.** It must reach the API at `http://localhost:8080`, an origin `CORS_ORIGINS` must list. `NEXT_PUBLIC_API_URL` is baked at build time, so change it and restart `pnpm dev:web` rather than only the container.
- **Stored images 404, or the API logs `WARNING: APP_URL is http://localhost:3000 but this server listens on port 8080`.** `APP_URL` is where the API thinks it is reachable, and it builds file URLs from it; the web app rewrites that origin to `NEXT_PUBLIC_API_URL` before loading a file, so images still open. Set `APP_URL=http://localhost:8080` in `.env` to silence the warning and store the honest URL — the same advice applies to the host run.
- **The build cannot reach the internet** (`temporary error` from `apk`, or `go mod download` timing out) on a machine whose default bridge network has no egress. Build with host networking: `docker build --network=host -t campvote-api ./apps/api`, then `docker compose up -d`. This is a property of that Docker installation, not of the image; nothing in the repository needs changing.

---

## Option 2: API Container With `docker run` (No Compose)

Use this instead of Option 1 when Compose is not available, or when the database is a managed service elsewhere (Neon, Supabase, RDS, a Postgres server you run on another box) and the container only has to reach out. Only the Go API runs in a container; it connects to the external database through `DATABASE_URL`.

**Note:** `docker-compose.yaml` already treats the database as external, so `docker compose up -d` does everything below. One difference matters here: `docker run` puts the container on the default bridge network, where `localhost` is the container, so a `DATABASE_URL` pointing at `localhost:5432` will not find a database on the machine. Either add `--network=host` (Linux, the same trick the compose file uses) or name the database's real host in the URL.

### Prerequisites

- Docker Engine installed
- An external PostgreSQL database reachable from the container, with a database and user already created
- Its connection string, for example `postgres://USER:PASSWORD@DB_HOST:5432/DB_NAME?sslmode=require`

### Step-by-step setup

```bash
# 1. Clone the repository
git clone https://github.com/Izaek256/campVote.git
cd campVote

# 2. Build the API image (build context is apps/api)
docker build -t campvote-api ./apps/api
# `docker compose build api` produces the same image under the same tag,
# campvote-api, because Compose names it after the project directory and service.
# An image that is only ever built this way does not exist on any registry: a
# `docker run campvote-api` on a machine that has never built it fails with
# "pull access denied", which is Docker Hub saying it has no such repository.

# 3. Run the container against your external database.
#    JWT_SECRET is required and must be at least 32 characters (openssl rand -hex 32).
docker run -d --name campvote-api \
  -p 8080:8080 \
  -e APP_ENV=development \
  -e APP_URL=http://localhost:8080 \
  -e DATABASE_URL='postgres://USER:PASSWORD@DB_HOST:5432/DB_NAME?sslmode=require' \
  -e JWT_SECRET='replace-with-a-long-random-secret-32plus-chars' \
  -e CORS_ORIGINS='http://localhost:3000' \
  -e REDIS_URL='' \
  -e STORAGE_DRIVER=local \
  -v campvote_storage:/app/storage \
  campvote-api

# 3b. Or, on your VPS, copy your .env file across and pass it with --env-file
#     instead of repeating every -e flag above.
docker run -d --name campvote-api \
  -p 8080:8080 \
  --env-file .env \
  -v campvote_storage:/app/storage \
  campvote-api
```

The storage volume goes at `/app/storage` rather than at one uploads folder because that is where both writable defaults live: `STORAGE_LOCAL_ROOT=storage/app` and `MAIL_LOG_PATH=storage/mail`, resolved against the image's `/app` working directory.

When using `--env-file .env`, make sure that file already contains the values this container needs: `DATABASE_URL` (pointing at a database this container can reach — `localhost` means the container unless you also pass `--network=host`) and a `JWT_SECRET` of at least 32 characters. Set `REDIS_URL` to a reachable address or leave it empty, and set `APP_URL` and `CORS_ORIGINS` to your frontend's address. Docker ignores blank lines and lines beginning with `#`, so the comments in `.env.example` are fine to keep.

The image's entrypoint runs the database migrations and then starts the API, so no separate migrate step is required. The API is then available at http://localhost:8080, with the health check at `/api/health`.

### Create the first administrator

The container already carries `DATABASE_URL`, so bootstrap reuses it:

```bash
docker exec -e BOOTSTRAP_ADMIN_EMAIL=admin@must.ac.ug \
  -e BOOTSTRAP_ADMIN_PASSWORD='your-secure-password-12+chars' \
  -e BOOTSTRAP_ADMIN_FIRST_NAME=Admin \
  -e BOOTSTRAP_ADMIN_LAST_NAME=User \
  campvote-api ./bootstrap
```

### Notes

- `REDIS_URL` is left empty here to disable background jobs (elections then open/close lazily on request). Set it to a reachable Redis URL to enable the election scheduler and job queue.
- If you run with `APP_ENV=production` and `STORAGE_DRIVER=local`, you must also set `ALLOW_LOCAL_STORAGE_IN_PRODUCTION=true` (with the uploads volume backed up), or switch to an S3-compatible driver.
- Mail: set `MAIL_MAILER` and the relevant `SMTP_*`/provider variables to a reachable server. For a browser-free local loop, `MAIL_MAILER=log` writes each message to stdout and to `MAIL_LOG_PATH` instead of sending it — useful here, because docker-compose.yaml no longer bundles Mailpit either.
- Set `APP_URL` and `CORS_ORIGINS` to the addresses of your deployed web frontend.

### Stopping

```bash
# Stop and remove the container (uploaded files stay on the volume)
docker stop campvote-api && docker rm campvote-api

# Also remove the volume holding uploads and log-driver mail (destructive)
docker volume rm campvote_storage
```

---

## Option 3: Local Development (PostgreSQL in Docker, Apps Local)

This runs the Go API and the Next.js app locally on your machine, against a PostgreSQL in a container. `docker-compose.yaml` holds only the API now, so the database below is started on its own with `docker run`; nothing in the repository manages it.

### Prerequisites

- Docker Engine (only for the PostgreSQL container)
- Go 1.23+
- Node.js 20+
- pnpm package manager
- Git

### Step-by-step setup

```bash
# 1. Clone the repository
git clone https://github.com/Izaek256/campVote.git
cd campVote

# 2. Install pnpm (if not already installed)
npm install -g pnpm@10.33.4

# 3. Install all workspace dependencies
pnpm install

# 4. Start a PostgreSQL for the API to use (the compose file has none)
docker run -d --name campvote-postgres \
  -e POSTGRES_DB=campvote_dev \
  -e POSTGRES_USER=campvote_dev \
  -e POSTGRES_PASSWORD=your-local-password \
  -p 127.0.0.1:5432:5432 \
  -v campvote_postgres_data:/var/lib/postgresql/data \
  postgres:16-alpine
# Skip this if a PostgreSQL already runs on the machine, and use its credentials
# in step 5 instead. Stop the container with docker stop campvote-postgres.

# 5. Set up environment variables
cp .env.example .env
# Edit .env to set secure passwords
# Minimum required changes:
# DATABASE_URL=postgres://campvote_dev:your-local-password@localhost:5432/campvote_dev?sslmode=disable
#                  (the API reads this and nothing else to find the database;
#                   percent-encode @, %, / and # inside the password here)
# JWT_SECRET=dev-jwt-secret-change-in-production

# 6. Run database migrations
cd apps/api
go run ./cmd/migrate
cd ../..

# 7. (Optional) Seed demo data
cd apps/api
go run ./cmd/seed
cd ../..

# 8. Create the first administrator
cd apps/api
BOOTSTRAP_ADMIN_EMAIL=admin@must.ac.ug \
BOOTSTRAP_ADMIN_PASSWORD='your-secure-password-here' \
BOOTSTRAP_ADMIN_FIRST_NAME=Admin \
BOOTSTRAP_ADMIN_LAST_NAME=User \
go run ./cmd/bootstrap
cd ../..

# 9. Start all applications in parallel
pnpm dev
```

### Access the applications

- **Voter portal & Admin dashboard**: http://localhost:3000
  - Voter pages: `/`, `/elections`, `/elections/[id]/ballot`, `/elections/[id]/results`
  - Admin pages: `/staff`, `/staff/elections/[id]`
- **API**: http://localhost:8080

### Running individual services

```bash
# Start only the API (air hot reload, see apps/api/.air.toml)
pnpm dev:api

# Start only the voter portal
pnpm dev:web
```

### Stopping local development

```bash
# Stop the running apps (Ctrl+C in the terminal where pnpm dev is running)

# Stop the database container (preserves its volume)
docker stop campvote-postgres

# Stop and delete it, then its data (destructive)
docker rm -v campvote-postgres
docker volume rm campvote_postgres_data
```

### Accounts

No demo accounts are seeded. Continue with the manual setup in this section:

- **Staff admin**: the email and password you set with `cmd/bootstrap`
- **Student voter**: register a campus email such as `2024bse088@std.mast.ac.ug`

---

## Option 4: Local Development (PostgreSQL Local on Windows, Apps Local)

This runs PostgreSQL locally on Windows (not in Docker) and runs the Go API and Next.js apps locally. Use this if you have PostgreSQL installed directly on Windows.

### Prerequisites

- PostgreSQL 16+ installed locally on Windows
- Go 1.23+
- Node.js 20+
- pnpm package manager
- Git

### Step-by-step setup

```bash
# 1. Clone the repository
git clone https://github.com/Izaek256/campVote.git
cd campVote

# 2. Install pnpm (if not already installed)
npm install -g pnpm@10.33.4

# 3. Install all workspace dependencies
pnpm install

# 4. Set up environment variables
cp .env.example .env
# Edit .env to set secure passwords
# Minimum required changes:
# POSTGRES_PASSWORD=your-windows-postgres-password
# DATABASE_URL=postgres://campvote_dev:your-password@localhost:5432/campvote_dev?sslmode=disable&client_encoding=UTF8
# JWT_SECRET=dev-jwt-secret-change-in-production

# 5. Create PostgreSQL user and database (one-time setup)
# Open PowerShell or Command Prompt as administrator:
psql -U postgres -c "CREATE USER campvote_dev WITH PASSWORD 'your-password';"
psql -U postgres -c "CREATE DATABASE campvote_dev OWNER campvote_dev;"
psql -U postgres -d campvote_dev -c "GRANT ALL ON SCHEMA public TO campvote_dev;"
psql -U postgres -d campvote_dev -c "ALTER SCHEMA public OWNER TO campvote_dev;"

# 6. Run database migrations
cd apps/api
go run ./cmd/migrate
cd ../..

# 7. (Optional) Seed demo data
cd apps/api
go run ./cmd/seed
cd ../..

# 8. Create the first administrator
cd apps/api
BOOTSTRAP_ADMIN_EMAIL=admin@must.ac.ug \
BOOTSTRAP_ADMIN_PASSWORD='your-secure-password-12+chars' \
BOOTSTRAP_ADMIN_FIRST_NAME=Admin \
BOOTSTRAP_ADMIN_LAST_NAME=User \
go run ./cmd/bootstrap
cd ../..

# 9. Remove bootstrap credentials from .env after successful creation

# 10. Start all applications in parallel
pnpm dev
```

### Access the applications

- **Voter portal & Admin dashboard**: http://localhost:3000
  - Voter pages: `/`, `/elections`, `/elections/[id]/ballot`, `/elections/[id]/results`
  - Admin pages: `/staff`, `/staff/elections/[id]`
- **API**: http://localhost:8080
- **API Documentation**: http://localhost:8080/docs
- **GORM Studio**: http://localhost:8080/studio

### Running individual services

```bash
# Start only the API
cd apps/api && go run ./cmd/server

# Start only the voter portal (staff pages live in the same app under /staff)
cd apps/web && pnpm dev
```

### Stopping local development

```bash
# Stop the running apps (Ctrl+C in the terminal)
# PostgreSQL stays running as a Windows service
```

### Accounts

No demo accounts are seeded.

- **Staff admin**: the email and password you set with `cmd/bootstrap`
- **Student voter**: register a campus email such as `2024bse088@std.mast.ac.ug`

---

## Common tasks (both setups)

### Frontend-specific notes

The voter portal and the staff pages are one Next.js application, `apps/web`:

- Node.js 20+ (check `package.json` for exact version)
- pnpm package manager (configured in `pnpm-workspace.yaml`)

To build for production:

```bash
# Build all packages
pnpm build

# Build the web app on its own
cd apps/web && pnpm build
```

### Authentication and first administrator

No administrator is seeded by default. The bootstrap command:

- Only works on a fresh database
- Requires a 12+ character password
- Refuses to run once any administrator exists
- Should be run once, then the `BOOTSTRAP_ADMIN_*` variables removed

After bootstrap, create further staff accounts from the staff pages at `/staff`. See [`docs/auth.md`](docs/auth.md) for details on cookie flags, lockout thresholds, and authentication hardening.

### Environment variables

Key variables in `.env`:

| Variable | Required | Purpose |
|----------|----------|---------|
| `APP_ENV` | Yes | Set to `development` for demo data |
| `APP_URL` | Yes | Where the API is reachable in a browser; file URLs are built from it, so on localhost it should name the API's own port, `http://localhost:8080` |
| `CORS_ORIGINS` | Yes for the browser | Comma-separated origins allowed to call the API, e.g. `http://localhost:3000` |
| `DATABASE_URL` | Yes | PostgreSQL connection string — the only thing the API uses to find the database, containerised or not |
| `JWT_SECRET` | Yes | Signing secret for tokens and session cookies, 32+ characters |
| `POSTGRES_PASSWORD` | No | Only a note to yourself and for `docker run` of a database container; no application code reads it |
| `REDIS_URL` | No | Empty means no cache, jobs or cron, on purpose; unset falls back to `redis://localhost:6380`. See Option 1 |
| `STORAGE_DRIVER` | No | `local` keeps uploads on disk at `STORAGE_LOCAL_ROOT` (`storage/app`); unset means `minio`, which falls back to the disk when no credentials are set |
| `MAIL_MAILER` | No | `log` prints each message to stdout and to `MAIL_LOG_PATH`; unset in development tries SMTP on `localhost:1025` then falls back to the log |
| `REQUIRE_EMAIL_VERIFICATION` | No | Default `false` for local dev |
| `LOGIN_MAX_ATTEMPTS` | No | Default `10` (set `0` to disable) |
| `PASSWORD_BREACH_CHECK` | No | Default `false` |

See `.env.example` for all available variables and explanations.

### Common commands

```bash
pnpm build          # build all packages
pnpm lint           # lint all packages
pnpm type-check     # type check all packages
pnpm test           # run all tests
pnpm test:api       # Go tests
pnpm format         # format with biome
```

## Testing and delivery

The generated scaffold determines exact commands. Before merging, run the applicable formatters, linters, type checks, unit tests, database migration/integration tests, frontend build, and critical end-to-end tests. Add CI workflows only after confirming real workspace paths and commands; do not add a green-but-empty pipeline or pretend checks ran when they did not.

## Security and privacy baseline

- Enforce eligibility and one accepted ballot per voter/election on the server and in database constraints/transactions.
- Keep voter identity/participation data separate from ballot selections. Do not expose identity-to-choice mappings to candidates or ordinary administrators.
- Do not log ballot selections alongside identifying information.
- Treat email-domain checks as identity hints, not proof of current enrollment or election-specific eligibility.
- Use synthetic data only; document that this academic prototype is not independently certified for real elections.
