# Workflows

## [`ci.yml`](ci.yml) — CI

Validates the Go API and the Next.js web app on pull requests into `develop` /
`main` and on pushes to those branches. Three jobs:

| Job | Check |
|---|---|
| `Backend (Go)` | gofmt, golangci-lint, `go vet`, `go test -race` (disposable PostgreSQL service) |
| `Frontend (Web)` | `tsc --noEmit`, Vitest, `next build` |
| `Format & Lint (Biome)` | repository-wide Biome check |

The workflow assumes a clean checkout and needs **no repository secrets** — the
PostgreSQL credentials are CI-only throwaways for the service container, never a
deployed database.

For the design rationale, tool-version matrix, local reproduction commands,
diagnosis table and known limitations, see
[`docs/testing/ci-pipeline.md`](../../docs/testing/ci-pipeline.md).

### Required branch protection

To keep a red build unmergeable, mark these as required status checks for
`develop` and `main` in the repository settings:

- `Backend (Go)`
- `Frontend (Web)`
- `Format & Lint (Biome)`
