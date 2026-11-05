# Development progress log

Add a dated section when a meaningful task, investigation, or handover occurs. Keep detailed discussions in GitHub issues/PRs and link them here.

## 2026-10-29 — ISSUE-06 and ISSUE-07 Implementation

- **Issue / PR:** ISSUE-06 (Ballot builder, candidate workflow, versioned ballot definitions) and ISSUE-07 (Secret-ballot privacy boundary)
- **Branch / commit:** 
  - feat/issue-06-ballot-candidate-workflows (5a606ef)
  - feat/issue-07-secret-ballot-boundary (094bd8e)
- **Changed:**
  - ISSUE-06: Added BallotVersion model with SHA-256 content hash and immutability, candidate nomination/approval workflow, position management, ballot builder with validation, and 8 UI views
  - ISSUE-07: Added privacy documentation, threat model, schema regression test, turnout API with small-group suppression, and 4 privacy UI components
- **Verification run:** Database migration successful (added ballot_versions table and candidate workflow fields)
- **Result:** Both issues implemented according to AGENTS.md rules. Backend models, API handlers, UI components, and documentation completed.
- **Known limitations / blockers:**
  - File upload sanitization for candidate photos not yet implemented (deferred)
  - DB role separation for participation/ballot tables documented but not implemented (deferred)
  - Backend tests and browser tests not yet written (deferred for validation phase)
- **Next step:** Create pull requests for both branches to develop.

## 2026-11-05 — Docker Compose containerises the API only

- **Issue / PR:** none yet — maintainer request in-session: "docker should only containerise the API". Needs an issue before the PR per CONTRIBUTING.md.
- **Branch / commit:** `feature/domains-admins-uploads-email` (uncommitted at the time of this entry)
- **Changed:**
  - `docker-compose.yaml`: dropped `web`, `redis` and `mailpit`; only `api` (built from `apps/api/Dockerfile`) and `postgres` remain. `REDIS_URL` now interpolates to empty on purpose, mail defaults to `MAIL_MAILER=log`, and `APP_URL`/`STORAGE_*` stay literals so a host-shaped `.env` cannot break the container.
  - `apps/api/Dockerfile`: creates and chowns `/app/uploads` + `/app/storage/mail` before `USER app`, otherwise the first upload or mail write fails as the unprivileged user.
  - `apps/api/entrypoint.sh`: `exec "$@"` when arguments are given. Previously `docker compose run api ./bootstrap` ignored the argument and started a second API.
  - `.env.example`, `README.md`: rewritten Docker/Option 1 setup for the new file; removed the stale `--profile wsl`/`--profile container` instructions, the deleted `compose.dev.yaml` references, `SESSION_SECRET`, and the deleted `apps/admin`.
  - `docs/operations/docker-compose.md` (new), indexed in `docs/README.md`; `docs/MVP_NOTES.md` assumptions 10–12 corrected.
- **Verification run:** `docker compose up -d` → both containers healthy; migrations built the schema from empty; `/api/health` `status ok` with `redis: off`, `jobs: off`, `email: log`, `storage: local`, 67 tables; signup printed its verification code to `docker compose logs api` and wrote the HTML copy; `run api ./bootstrap` created an admin; login `HTTP 200`; `exec api ./seed` still skips demo elections without `SEED_DEMO_DATA=true`. Details in `docs/operations/docker-compose.md`.
- **Known limitations / blockers:**
  - CI (`.github/workflows/ci.yml`) builds no Docker image, so none of the above is covered by the pipeline.
  - On this machine the default bridge network has no egress during builds, so the image was built with `docker build --network=host`; documented in README troubleshooting rather than baked into compose.
  - `POSTGRES_PASSWORD` must be URL-safe because one value feeds both the container's account and the API's DSN.
- **Next step:** open an issue for this change, commit on a `docs`+`fix` branch as CONTRIBUTING.md requires, and run the PR CI checks.
- **Status:** superseded the same day by the entry below. The maintainer's intent was an API-only file with an external database, so the `postgres` service added here was removed again, and the `POSTGRES_PASSWORD` URL-safety limitation it introduced went with it.

## 2026-11-05 — Docker Compose: the API is the only service, database external

- **Issue / PR:** none yet — maintainer correction in-session: "only containerize the api, db is external, .env all in this file". Needs an issue before the PR per CONTRIBUTING.md.
- **Branch / commit:** `feature/domains-admins-uploads-email` (uncommitted at the time of this entry)
- **Changed:**
  - `docker-compose.yaml`: one service, `api`. The `postgres` service from the entry above is removed, along with its volume, the `networks:` block and both `ports:` mappings. Configuration now comes from `.env` alone via `env_file: .env` — no `${VAR:-default}` interpolation and no container-side overrides — and `network_mode: host` makes the `.env` addresses (`localhost:5432` and friends) mean the machine, as they do for a host-run API.
  - `apps/api/Dockerfile`: the writable directory is now `/app/storage` (covering the `storage/app` uploads root and `storage/mail` log mail), matching the config defaults, so one volume at `/app/storage` needs no environment overrides.
  - `docker-compose.override.yaml` (the local port-5433 workaround for the removed `postgres` service) deleted.
  - `README.md`: Option 1 rewritten for a single container with an external database; Option 2 retitled to the `docker run` route with the bridge-network `localhost` caveat; Option 3 now starts PostgreSQL with `docker run`, since compose no longer has one; the environment-variable table corrected (`POSTGRES_PASSWORD` read by no application code; `APP_URL`, `CORS_ORIGINS`, `STORAGE_DRIVER` added).
  - `.env.example`: compose now passes the file through untouched, so its notes describe the container reading the same values as the host, the `REDIS_URL=` empty-vs-unset distinction, and percent-encoding inside `DATABASE_URL` only.
  - `docs/operations/docker-compose.md` rewritten for this state; `docs/MVP_NOTES.md` assumptions 10–12 updated again.
- **Verification run:** rebuilt the image and ran `docker compose up -d` against this machine's own PostgreSQL using the operator's real `.env`, unmodified. `docker compose config` shows one service; the container reported `Database connected successfully` and migrated the existing schema, then `The schema was already up to date` on a second start; `/api/health` from the host returned `status: ok`, 67 tables, `redis: off`, `jobs: off`, `email: failover(smtp,log)`, `storage: local`; `touch` as `uid=100(app)` succeeded in both storage directories; `docker compose run --rm api echo …` proved the entrypoint argument pass-through. Signup and bootstrap were deliberately not exercised — the database is a real development database, and test rows there would survive `down -v`. Details in the operations doc.
- **Known limitations / blockers:**
  - CI builds no Docker image, so nothing above is covered by the pipeline.
  - `network_mode: host` means the container cannot start while a host-run API holds `8080`, and it can reach every port on the machine. The bridge + `host.docker.internal` alternative and its cost (a second DSN) are recorded in the operations doc and README.
  - This `.env` sets `APP_URL=http://localhost:3000`, so the API logs a port-mismatch warning at boot and stores file URLs on `:3000`; `apps/web/lib/api.ts` rewrites that origin to `NEXT_PUBLIC_API_URL`, so files display. The compose file no longer forces `APP_URL`, by design.
  - `STORAGE_DRIVER` is unset in this `.env`, so the API probes for MinIO and falls back to local disk with a log line; setting `STORAGE_DRIVER=local` in `.env` removes the probe.
- **Next step:** open an issue capturing "compose contains exactly one service; database, cache, mail and web are external and configured through `.env`", then commit these changes on a task branch and let PR CI run.
