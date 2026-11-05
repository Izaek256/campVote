# Atomic Vote Submission — ISSUE-08

## Overview

ISSUE-08 makes a ballot submission atomic, idempotent and concurrency-safe, and
builds the voter-facing views that drive it. A voter votes exactly once per
election, the rules of the loaded ballot definition are enforced server-side,
and a lost HTTP response can be recovered from without risking a second ballot.

Status: implemented on `feat/issue-08-atomic-vote-submission` (see
`docs/progress/development-log.md`).

## Design

### Records and privacy

Two records are written, matching the ISSUE-07 privacy boundary:

- `VoterParticipation` — proof the voter voted (per election and user).
- `AnonymousBallot` — the selections, carrying `ElectionID`,
  `BallotVersionID` and the selection list.

The ballot row holds no `UserID`, no email, no token. The participation row
holds no choice. Nothing written by this issue links a voter to a choice.

### One transaction

`POST /api/v1/elections/:id/ballot/submit` (`handlers.BallotHandler.Submit`)
runs one database transaction that:

1. reloads the election,
2. re-checks eligibility with the server clock (`eligibility.CheckElection`,
   which also reconciles the lifecycle),
3. validates the selections against the published `BallotVersion` the voter
   loaded,
4. inserts `VoterParticipation`,
5. inserts `AnonymousBallot`,
6. writes a `VOTE_CAST` audit row referencing participation only.

A failure at any step rolls back all of them. There is no partial state.

### Single-ballot guarantee

- The `VoterParticipation` model has a unique index on
  `(election_id, user_id)`. The database refuses a second ballot even if the
  application logic were bypassed.
- A second unique index on `(election_id, idempotency_key)` makes the idempotency
  replay a database fact, not a lookup table. The key is random-generated in
  `BeforeCreate` when a client does not send one.

### Idempotency

The client generates one idempotency key per attempt set and reuses it for
manual retries (see `apps/web/app/elections/[id]/ballot/page.tsx`). The rule:

- Same key, same voter, same election → the first outcome is replayed with
  `200 OK` and the original confirmation token. No second ballot is written.
- New key after a successful ballot → `409 Conflict` ("you have already voted").
- Stale `ballot_version_id` (the ballot was re-published mid-flow) →
  `409 Conflict` with the stale-ballot message. Nothing is recorded.

### Closing-time race rule

Acceptance is decided **at commit time, inside the transaction, by the server
clock**, not when the request started. A request that arrives before `ends_at`
but whose transaction runs after it is refused with `409` and writes nothing.
The `Submit` path calls `lifecycle.Ensure` first, so a missed scheduler tick
cannot keep an election open.

Covered by `TestSubmissionRefusedWhenServerClockCrossesCloseDuringRequest`,
which moves the injectable clock between the eligibility gate and the
transaction.

### Ballot version binding

- `View` and `Start` return the current `BallotVersion` id. The client echoes
  it back on submit.
- `Submit` compares it to the current published version and rejects a mismatch
  (`409`). An unknown position, unknown candidate, duplicate choice,
  under-selection or over-selection is a `400` with a message that names no
  other voter.
- Legacy path: if an election has never published a version, submission
  validates against the live positions (`BallotContentFor`) so elections
  opened before ISSUE-06 still work.

### Vote status (recovery)

`GET /api/v1/elections/:id/my-vote-status` returns
`{voted, recorded_at?, phase, server_time}` for the authenticated voter only.

- `recorded_at` is truncated to 15-minute buckets: enough for the voter to
  recognise their session, too coarse to correlate with ballot storage order.
- The response carries no choice field, and the test
  `TestVoteStatusBeforeVotingIsFalseAndCarriesNoChoiceField` asserts that no
  choice-like key is present.

### No partial writes under fault

Any error inside the transaction (constraint, validation, audit write) returns
upwards and GORM rolls the transaction back.
`TestRejectedBallotWritesNothing` asserts participation and ballot counts stay
at zero for a refused submission.

## API

| Method | Path | Auth | Notes |
|---|---|---|---|
| GET | `/api/v1/elections/:id/ballot` | authenticated, eligible | Ballot content + `ballot_version_id` |
| POST | `/api/v1/elections/:id/ballot/start` | authenticated, eligible | `409` if already voted; returns `ballot_version_id` |
| POST | `/api/v1/elections/:id/ballot/submit` | authenticated, eligible | Atomic, idempotent; `201` new, `200` replay, `409` closed/already-voted/stale, `400` bad selections |
| GET | `/api/v1/elections/:id/my-vote-status` | authenticated | Participation only |

The election list (`GET /api/v1/elections`) also includes a coarse
`my_vote_status` per election so the list page can show `Vote` vs `View status`
without one request per card.

## Voter views

| Route | State |
|---|---|
| `/elections` | Cards show server phase plus the voter's own status; voted elections link to status, open ones to the ballot |
| `/elections/:id` | Vote CTA hidden once the server says the voter has voted; replaced by "View vote status" |
| `/elections/:id/ballot` | Fill in → review step → confirm checkbox → submit. Breadcrumbs, per-position min/max counters, blank (abstain) option where allowed, radio/checkbox groups |
| `/elections/:id/status` | Voted / not voted / election closed / session expired. No re-vote path is offered for a recorded vote |

Selections live only in component state. Nothing is written to storage, the URL
or history. A failed submit clears nothing until the voter leaves, and the
in-memory idempotency key is reused for a manual retry.

### Unknown outcome

If the submit request fails without a server answer (network drop), the ballot
page renders the "We could not confirm your vote" banner with a **Check status**
link and warns against re-submitting. The status page is the authority: it says
whether a ballot exists. It never offers a second submission when one is
recorded.

### Stale ballot

If the server answers `409` with the stale-ballot message, the page shows "The
ballot changed while you were filling it in", records nothing, and offers a
reload. The voter re-selects against the new version.

## Tests

`apps/api/internal/handlers/vote_submission_test.go`:

- `TestVoteSubmissionIsIdempotentAndRecordsOneBallot` — stale version refused,
  same key replays the same token, new key refused, one participation and one
  ballot at the end, audit row carries no choice, status shape.
- `TestParallelSubmissionsByOneVoterRecordExactlyOneBallot` — 8 parallel
  submissions with distinct keys, exactly one `201`, counts reconcile to 1/1.
- `TestRejectedBallotWritesNothing` — invalid selections write nothing.
- `TestSubmissionRefusedWhenServerClockCrossesCloseDuringRequest` — the
  closing-time race rule.
- `TestVoteStatusBeforeVotingIsFalseAndCarriesNoChoiceField` — privacy shape.

Run locally with:

```bash
cd apps/api
go test ./internal/handlers/ -run 'Vote|Ballot|Submission' -count=1
```

CI runs the whole suite with CGO and Postgres-compatible drivers.

## Known limitations

- No fault-injection harness (a failing statement between the participation and
  ballot inserts) — the guarantee rests on GORM's transaction and the database,
  and the constraint test covers the duplicate path rather than a mid-write
  crash. Worth adding a Postgres-side test with a forced constraint violation.
- Multi-tab double voting is prevented server-side, but the second tab only
  learns about it on its next request; there is no cross-tab push.
- The client reuses one idempotency key per page load. A page reload after an
  unknown outcome generates a new key; recovery is via the status page, not the
  same key.
- Browser/E2E evidence (video, screenshots, storage inspection) is deferred:
  no Playwright suite exists in this repository yet, and no local Docker or
  browser-run environment was available for this change. The API-level tests
  above are the evidence for the same rules.
