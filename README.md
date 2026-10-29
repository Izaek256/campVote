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

You can run campVote in two ways:

1. **Docker Compose (recommended)** - All services run in containers
2. **Local development** - PostgreSQL in Docker, apps run locally

Choose the setup that fits your workflow.

---

## Option 1: Docker Compose (All Services in Containers)

This runs PostgreSQL, API, voter portal, and admin dashboard entirely in Docker containers. Two profiles are available:

- **WSL setup** (`--profile wsl`): Uses your local PostgreSQL on Windows (for WSL2 Docker)
- **Containerized setup** (`--profile container`): Runs PostgreSQL in a Docker container (for non-WSL or fully containerized)

### Running on WSL (Windows Subsystem for Linux)

**Important:** Docker's default bridge network in WSL2 often has no outbound internet access, causing build failures (npm, apk, go mod downloads timeout). A workaround is in place:

- Build blocks use `network: host` to access the internet during builds
- Use the provided script `./scripts/up-wsl.sh` which handles BuildKit permission requirements
- Once Docker networking (MTU/iptables) is fixed, the `network: host` lines and script can be removed

```bash
# In WSL at /mnt/d/campVote
./scripts/up-wsl.sh
```

The script tries:
1. Standard `docker compose` build (may work if entitlement already granted)
2. Falls back to `docker buildx bake --allow=network.host` if standard build fails
3. Prints which path was used

**Note:** The script does not modify your system (daemon.json, iptables, wsl.conf). The `network: host` is only for build time; running containers use the normal `campvote` bridge network.

### Prerequisites

- Docker Engine and Docker Compose installed
- Git
- For WSL setup: PostgreSQL running on Windows at localhost:5432

### Step-by-step setup (choose your profile)

#### For WSL (local PostgreSQL on Windows):

```bash
# 1. Clone the repository
git clone https://github.com/Izaek256/campVote.git
cd campVote

# 2. Set up environment variables
cp .env.example .env
# Edit .env to set secure passwords
# POSTGRES_PASSWORD must match your Windows PostgreSQL password
# Example:
# POSTGRES_PASSWORD=your-windows-pg-password
# SESSION_SECRET=dev-session-secret-change-in-production
# JWT_SECRET=dev-jwt-secret-change-in-production

# 3. Build and start services with WSL profile
docker compose --profile wsl up -d --build

# 4. Check service status
docker compose ps
# All services should show "running"
```

#### For containerized (PostgreSQL in Docker):

```bash
# 1. Clone the repository
git clone https://github.com/Izaek256/campVote.git
cd campVote

# 2. Set up environment variables
cp .env.example .env
# Edit .env to set secure passwords
# Example:
# POSTGRES_PASSWORD=campvote_dev_password
# SESSION_SECRET=dev-session-secret-change-in-production
# JWT_SECRET=dev-jwt-secret-change-in-production

# 3. Build and start services with container profile
docker compose --profile container up -d --build

# 4. Check service status
docker compose ps
# All services should show "running" (postgres should be "healthy")
```

### One-time database setup (first run only)

Run these commands from inside the API container:

**For WSL setup:**
```bash
# Run database migrations
docker compose --profile wsl exec api go run ./cmd/migrate

# (Optional) Seed demo data for development
docker compose --profile wsl exec api go run ./cmd/seed

# Create the first administrator account
docker compose --profile wsl exec -e BOOTSTRAP_ADMIN_EMAIL=admin@must.ac.ug \
  -e BOOTSTRAP_ADMIN_PASSWORD='your-secure-password-here' \
  -e BOOTSTRAP_ADMIN_FIRST_NAME=Admin \
  -e BOOTSTRAP_ADMIN_LAST_NAME=User \
  api go run ./cmd/bootstrap
```

**For containerized setup:**
```bash
# Run database migrations
docker compose --profile container exec api-container go run ./cmd/migrate

# (Optional) Seed demo data for development
docker compose --profile container exec api-container go run ./cmd/seed

# Create the first administrator account
docker compose --profile container exec -e BOOTSTRAP_ADMIN_EMAIL=admin@must.ac.ug \
  -e BOOTSTRAP_ADMIN_PASSWORD='your-secure-password-here' \
  -e BOOTSTRAP_ADMIN_FIRST_NAME=Admin \
  -e BOOTSTRAP_ADMIN_LAST_NAME=User \
  api-container go run ./cmd/bootstrap
```

**Important:** The bootstrap command only works once. After an admin exists, you cannot run it again. Use the admin dashboard to create additional staff accounts.

### Access the applications

- **Voter portal & Admin dashboard**: http://localhost:3000
  - Voter pages: `/`, `/elections`, `/elections/[id]/ballot`, `/elections/[id]/results`
  - Admin pages: `/staff`, `/staff/elections/[id]`
- **API**: http://localhost:8080

### Demo accounts

When `APP_ENV=development`, the following demo accounts are created:

- **Student voter**: `ivan.mugisha@std.must.ac.ug` / `admin123`
- **Staff admin**: `admin@must.ac.ug` (use the password you set during bootstrap)

**Warning:** Demo accounts and data are for local development only. Never use them in production.

### Stopping Docker Compose

```bash
# Stop WSL setup (preserves local database data)
docker compose --profile wsl down

# Stop containerized setup (preserves database data)
docker compose --profile container down

# Stop containerized setup and remove database data (destructive - all data lost)
docker compose --profile container down -v

# View logs
docker compose --profile wsl logs -f [service-name]
# Example: docker compose --profile wsl logs -f api
```

### Troubleshooting Docker setup

```bash
# Check if containers are running
docker compose ps

# Check which profile is active
docker compose config

# Restart a specific service (WSL)
docker compose --profile wsl restart api

# Rebuild a service after code changes (WSL)
docker compose --profile wsl up -d --build api

# View real-time logs for a service (containerized)
docker compose --profile container logs -f web

# Remove all containers and volumes (complete reset)
docker compose --profile container down -v
docker system prune -f
```

---

## Option 2: Local Development (PostgreSQL in Docker, Apps Local)

This runs PostgreSQL in Docker but runs the Go API and Next.js apps locally on your machine. This is the standard setup for active development.

### Prerequisites

- Docker Engine and Docker Compose (for PostgreSQL only)
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

# 4. Start PostgreSQL in Docker
docker compose -f compose.dev.yaml up -d

# 5. Set up environment variables
cp .env.example .env
# Edit .env to set secure passwords
# Minimum required changes:
# POSTGRES_PASSWORD=your-local-password
# DATABASE_URL=postgres://campvote_dev:your-local-password@localhost:5432/campvote_dev?sslmode=disable
# SESSION_SECRET=dev-session-secret-change-in-production
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
# Start only the API
pnpm dev:api

# Start only the voter portal
pnpm dev:web

# Start only the admin dashboard
pnpm dev:admin
```

### Stopping local development

```bash
# Stop the running apps (Ctrl+C in the terminal where pnpm dev is running)

# Stop PostgreSQL (preserves data)
docker compose -f compose.dev.yaml down

# Stop PostgreSQL and remove data (destructive)
docker compose -f compose.dev.yaml down -v
```

### Demo accounts

Same as Docker setup:
- **Student voter**: `ivan.mugisha@std.must.ac.ug` / `admin123`
- **Staff admin**: `admin@must.ac.ug` (use your bootstrap password)

---

## Option 3: Local Development (PostgreSQL Local on Windows, Apps Local)

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
# SESSION_SECRET=dev-session-secret-change-in-production
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

# Start only the voter portal
cd apps/web && pnpm dev

# Start only the admin dashboard
cd apps/admin && pnpm dev
```

### Stopping local development

```bash
# Stop the running apps (Ctrl+C in the terminal)
# PostgreSQL stays running as a Windows service
```

### Demo accounts

- **Student voter**: `ivan.mugisha@std.must.ac.ug` / `admin123`
- **Staff admin**: Your bootstrap email and password

---

## Common tasks (both setups)

### Frontend-specific notes

The voter portal (`apps/web`) and admin dashboard (`apps/admin`) are Next.js applications:

- Node.js 20+ (check `package.json` for exact version)
- pnpm package manager (configured in `pnpm-workspace.yaml`)

To build for production:

```bash
# Build all packages
pnpm build

# Build specific package
cd apps/web && pnpm build
cd apps/admin && pnpm build
```

### Authentication and first administrator

No administrator is seeded by default. The bootstrap command:

- Only works on a fresh database
- Requires a 12+ character password
- Refuses to run once any administrator exists
- Should be run once, then the `BOOTSTRAP_ADMIN_*` variables removed

After bootstrap, use the admin dashboard to create additional staff accounts. See [`docs/auth.md`](docs/auth.md) for details on cookie flags, lockout thresholds, and authentication hardening.

### Environment variables

Key variables in `.env`:

| Variable | Required | Purpose |
|----------|----------|---------|
| `APP_ENV` | Yes | Set to `development` for demo data |
| `DATABASE_URL` | Yes | PostgreSQL connection string |
| `SESSION_SECRET` | Yes | Cookie signing secret |
| `JWT_SECRET` | Yes | JWT token signing secret |
| `POSTGRES_PASSWORD` | Yes | Database password (Docker setup) |
| `REQUIRE_EMAIL_VERIFICATION` | No | Default `false` for local dev |
| `LOGIN_MAX_ATTEMPTS` | No | Default `10` (set `0` to disable) |
| `PASSWORD_BREACH_CHECK` | No | Default `false` |

See `.env.example` for all available variables and explanations.

### Common commands

`ash
pnpm build          # build all packages
pnpm lint           # lint all packages
pnpm type-check     # type check all packages
pnpm test           # run all tests
pnpm test:api       # Go tests
pnpm format         # format with biome
`
## Testing and delivery

The generated scaffold determines exact commands. Before merging, run the applicable formatters, linters, type checks, unit tests, database migration/integration tests, frontend build, and critical end-to-end tests. Add CI workflows only after confirming real workspace paths and commands; do not add a green-but-empty pipeline or pretend checks ran when they did not.

## Security and privacy baseline

- Enforce eligibility and one accepted ballot per voter/election on the server and in database constraints/transactions.
- Keep voter identity/participation data separate from ballot selections. Do not expose identity-to-choice mappings to candidates or ordinary administrators.
- Do not log ballot selections alongside identifying information.
- Treat email-domain checks as identity hints, not proof of current enrollment or election-specific eligibility.
- Use synthetic data only; document that this academic prototype is not independently certified for real elections.
