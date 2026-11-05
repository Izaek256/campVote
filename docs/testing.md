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
" content">
D:\campVote\docs\testing.md