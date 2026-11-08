# CI Pipeline

How continuous integration for campVote is wired, what each check proves, and how
to reproduce it locally. The workflow itself is
[`.github/workflows/ci.yml`](../../.github/workflows/ci.yml).

## 1. What runs

Three independent jobs, all on `ubuntu-latest`:

| Job | Name shown in GitHub | Proves |
|---|---|---|
| `backend` | Backend (Go) | gofmt, golangci-lint, `go vet`, `go test -race` against PostgreSQL |
| `frontend` | Frontend (Web) | `tsc --noEmit`, Vitest, `next build` |
| `format` | Format & Lint (Biome) | Repository-wide Biome format + lint |

Each job runs from a clean checkout. Jobs do not share state.

### Triggers

```yaml
on:
  push:        { branches: [develop, main] }
  pull_request:{ branches: [develop, main] }
  workflow_dispatch:
```

This mirrors the branch policy in [`CONTRIBUTING.md`](../../CONTRIBUTING.md):
work branches reach `develop` through a pull request, so the PR trigger already
covers them. There is deliberately **no** `push` trigger for feature/fix branches
— that would run every commit twice. `push` fires only for the integration
branches themselves (post-merge validation). `workflow_dispatch` allows a manual
run.

### Guardrails on the workflow

- `concurrency: ci-${{ github.ref }}` with `cancel-in-progress: true` — a new
  commit supersedes the run in flight, so stale runs do not occupy runners.
- `permissions: contents: read` — no job writes to the repository, so a PR from a
  fork never gets a writable token.
- `timeout-minutes` on every job (25 / 20 / 10) — a hung step fails instead of
  burning the six-hour default.

## 2. Tool versions

| Tool | Version in CI | Source of truth |
|---|---|---|
| Go | `1.26` (newest 1.26.x patch) | `apps/api/go.mod` header asks for ≥ 1.26.10 |
| golangci-lint | `v2.13` | `golangci/golangci-lint-action@v9` input |
| Node.js | `20` | `apps/web` package engines / workflow |
| pnpm | from `packageManager` field | `package.json` (`pnpm@10.33.4`) |
| PostgreSQL | `postgres:16-alpine` | `docker-compose.yaml` (kept in step) |
| Biome | from `package.json` devDependencies | `biome.jsonc` |

`golangci-lint` is run by the action, not by a separate shell step. The action
installs the requested version **and** runs it (`install-only` defaults to
`false`), so adding a second `golangci-lint run` step would repeat the same check.
The action's `verify` option (default `true`) validates the config file against
the golangci-lint JSON Schema; it is left at the default on purpose.

Go is pinned as `'1.26'` and not via `go-version-file`. The `go` directive in
`go.mod` is still `1.26.4`, which carries standard-library CVEs fixed in 1.26.10;
`go-version-file` would pin CI to that vulnerable patch. Raise the directive, then
switch. See [`../testing.md`](../testing.md) ("Known limitations").

## 3. Disposable PostgreSQL

`internal/testkit` and `internal/testutil` run integration tests against a real
PostgreSQL when the environment points at one. CI starts a throwaway service and
sets:

```
GRIT_TEST_DATABASE_URL=postgres://campvote:campvote@localhost:5432/campvote_test_ci?sslmode=disable
TEST_DATABASE_URL=postgres://campvote:campvote@localhost:5432/campvote_test_ci?sslmode=disable
CAMPVOTE_TEST_DB=1
```

The credentials are CI-only throwaways declared inline in the workflow — they are
not secrets and never point at any deployed database. `testutil.RequireTestDB`
additionally refuses a URL whose host is not `localhost`, `127.0.0.1` or
`postgres`, or whose database name does not start with `campvote_test_` unless
`CAMPVOTE_TEST_DB=1`. See [`../testing.md`](../testing.md).

Because these variables are set, the same `go test ./...` that skips integration
tests on an unconfigured workstation runs them in CI.

## 4. Reproducing CI locally

Run from the repository root unless noted. On Windows in PowerShell, prefix Go /
Node commands with an explicit `$env:Path` if the tools are not already on it.

```sh
# Format & lint (root job)
pnpm install --frozen-lockfile
pnpm format:check                 # biome check .

# Frontend job
cd apps/web
pnpm type-check                   # tsc --noEmit
pnpm test                         # vitest run
pnpm build                        # next build

# Backend job
cd apps/api
gofmt -l .                        # must print nothing
go mod download
golangci-lint run                 # same linter set as CI
go vet ./...
go test -race -coverprofile=coverage.out ./...
```

To reproduce the PostgreSQL-backed backend job locally, start a disposable
database and point the variables at it — never at a deployed database:

```sh
createdb campvote_test_ci
export GRIT_TEST_DATABASE_URL='postgres://localhost:5432/campvote_test_ci?sslmode=disable'
export TEST_DATABASE_URL="$GRIT_TEST_DATABASE_URL"
export CAMPVOTE_TEST_DB=1
cd apps/api && go test ./...
```

`apps/web/.next` is gitignored, so a clean checkout is unaffected; but a stale
local `.next` can break `pnpm type-check` on `**/*.ts` includes. Delete
`apps/web/.next` if type-check reports phantom modules from removed routes.

## 5. Diagnosing a red run

| Symptom | Likely cause | First thing to check |
|---|---|---|
| `Format & Lint (Biome)` fails | Unformatted file or lint finding | Run `pnpm format:check`; the log names file and rule |
| `Frontend` fails at type-check | Type error, or stale `.next` | Run `pnpm type-check`; delete `apps/web/.next` if modules are phantom |
| `Frontend` fails at build | Next.js build error | Run `pnpm build` locally |
| `Backend` fails at gofmt | File not gofmt-formatted | `gofmt -l .` lists offenders; `gofmt -w .` to fix |
| `Backend` fails at golangci-lint | Linter finding or invalid config | Run `golangci-lint run`; a config error points at `.golangci.yml` |
| `Backend` fails at test | Real failure, or DB not reachable | Confirm the `postgres` service is healthy; run the package alone with `-run` |
| Backend test build error `requires cgo` | cgo-only package on a no-gcc host | Local-only; CI has gcc. See [`../testing.md`](../testing.md) |

The workflow deliberately contains no `continue-on-error` on any quality step, so
a failing check cannot be silently absorbed into a green run. `codecov-action` is
the single exception: `fail_ci_if_error: false` means a coverage-upload outage
does not fail an otherwise-green test run (coverage is reported, not enforced).

## 6. Required checks for branch protection

So that a PR into `develop` or `main` cannot merge on a red build, mark these as
required status checks in the repository settings:

- `Backend (Go)`
- `Frontend (Web)`
- `Format & Lint (Biome)`

Branch protection itself is a repository setting, not something the workflow can
enforce; it must be configured (and kept) in GitHub.

## 7. Known limitations and deferred work

- **golangci-lint is not vendored.** It is not a repository dependency, so the
  backend lint check reproduces locally only if the tool is installed:
  `go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.13.0`.
  Without it (and without access to the Go module proxy) the lint step can only
  be observed in CI. This is a local-environment gap, not a reason to weaken the
  CI check.
- **One workflow, three jobs.** Lint and test share a job where the tooling does.
  Splitting golangci-lint into its own job is possible but adds a cold Go checkout
  and build-cache restore for no isolation benefit at this scale.
- **No `go-version-file`.** See §2; gated on the `go.mod` directive.
- **Actions are pinned to major tags** (`@v4`, `@v5`, `@v9`), not commit SHAs. SHA
  pinning is stricter supply-chain hygiene and is deferred rather than done
  silently.
- **No coverage threshold.** Coverage is uploaded for visibility only.
- **Local ≠ CI for cgo packages.** The four cgo SQLite test files cannot build on
  a gcc-less Windows host; they run in CI. See [`../testing.md`](../testing.md).
- **A single green run is not proof of a durable fix.** This pipeline was
  stabilised by removing duplicate checks and defects that blocked legitimate
  checks; each change still needs a passing run on `develop` to be considered
  confirmed.
