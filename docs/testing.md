# Testing

## Isolation strategy

Each test run uses a fresh, disposable database. The connection URL is read from
`TEST_DATABASE_URL` (never `DATABASE_URL`). A guard refuses to run if the URL does
not look disposable: the database name must start with `campvote_test_` or the
`CAMPVOTE_TEST_DB=1` env flag must be set, and the host must be in the allowlist
(`localhost`, `127.0.0.1`, `postgres`).

## Running

```sh
pnpm test          # all
pnpm test:api      # Go API
```

Migrations run from an empty DB as the first step of every integration run.

## Factories

Synthetic factories live in `apps/api/internal/testutil/factories` (test tree only).
All emails use `@test.invalid`. No seed/demo data is inserted by migrations or app code.

## Clock

Server time is injectable via the `Clock` interface; tests use a controllable fake clock.
No test uses `sleep` to wait for election windows.

## Known limitations

These are environmental facts, not defects. They explain why a local run can
differ from CI, and none of them is worked around by weakening a check.

### Go patch version

The `go` directive in `apps/api/go.mod` is `1.26.4`, but the comment directly
above it requires **Go 1.26.10** (standard-library fixes, including GO-2026-6617
in the HTTP/2 HPACK encoder). CI therefore installs `go-version: '1.26'`, which
resolves to the newest 1.26.x patch on the runner, rather than reading the
directive with `go-version-file` (that would pin CI to the vulnerable 1.26.4).
Raise the `go` directive to match before switching CI to `go-version-file`.

### cgo-only test packages

Four test files import the cgo SQLite driver (`gorm.io/driver/sqlite`):
`internal/saga/saga_test.go` and the `internal/services` passkey/recovery/saga
tests. They compile only with `CGO_ENABLED=1` and a C toolchain. GitHub's
`ubuntu-latest` runner ships gcc, so they run in CI. On a Windows workstation
without gcc the Go toolchain defaults to `CGO_ENABLED=0` and those packages fail
to build ("go-sqlite3 requires cgo to work. This is a stub"). Every other test
package uses the pure-Go driver (`github.com/glebarez/sqlite`) or PostgreSQL and
is unaffected. To run them locally, install a C toolchain and build with cgo
enabled. The race detector (`go test -race`, used in CI) also needs cgo on
Windows, so it cannot run on a gcc-less workstation at all.

### Integration tests skip when unconfigured

Tests that need a real PostgreSQL call `testutil.RequireTestDB`. Without
`TEST_DATABASE_URL` (or with a URL that fails the disposable-database guard) they
skip rather than run. A local `go test ./...` can therefore report success while
those tests never executed. CI sets `GRIT_TEST_DATABASE_URL`, `TEST_DATABASE_URL`
and `CAMPVOTE_TEST_DB`, so the same command there runs them against a disposable
`postgres:16-alpine` service. See [`testing/ci-pipeline.md`](testing/ci-pipeline.md).
" content">
D:\campVote\docs\testing.md