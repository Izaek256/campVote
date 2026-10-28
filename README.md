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

## Local database option

`compose.dev.yaml` starts only a local PostgreSQL service for early development. It is optional: if the generated Grit scaffold already provides a database service, use one setup path rather than starting two PostgreSQL instances. Change the sample credentials before use and never reuse them in production.

```bash
docker compose -f compose.dev.yaml up -d
# To stop the database without deleting its data:
docker compose -f compose.dev.yaml down
# To remove the local database volume too (destructive):
docker compose -f compose.dev.yaml down -v
```

Copy `.env.example` to `.env` only after checking which variables the pinned scaffold actually expects. `.env` must remain untracked. Never use real voter rosters, live election data, or production secrets for local development.

## Quick start (local development)

`ash
# 1. Install dependencies
pnpm install

# 2. Start local PostgreSQL
docker compose -f compose.dev.yaml up -d

# 3. Copy env template and fill local values
cp .env.example .env

# 4. Run migrations (from apps/api)
cd apps/api && go run ./cmd/migrate 2>/dev/null; cd ../..

# 5. Start all apps
pnpm dev
`

- Web: http://localhost:3000
- API: http://localhost:8080
- Admin: http://localhost:3001

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
