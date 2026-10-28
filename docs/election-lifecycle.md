# Election Lifecycle, Schedule and Configuration (ISSUE-05)

Status: **implemented and tested on `feat/issue-05-election-lifecycle`**, 2026-10-28.
The branch starts from the ISSUE-04 head, which is already merged into
`develop` via PR #8. It needs a PR into `develop`.

Scope: election states, legal transitions, scheduling, pre-open validation,
configuration freeze, audited exceptional changes, and the server clock.
Out of scope: ballot content (ISSUE-06), vote storage and tallying (ISSUE-07/08),
certification semantics beyond the existing endpoint (ISSUE-09), and any UI.

## 1. What this guarantees

- The backend owns the clock. Request headers, browser time and UI state never
  decide whether voting is open.
- An election accepts votes only when its stored status is `OPEN` **and** server
  time is inside `[starts_at, ends_at)`.
- Missing a scheduler tick cannot extend voting: every ballot request runs the
  same lazy reconciliation first, and `IsOpen` is false after `ends_at` even if
  the status is stale.
- An election cannot open with an invalid schedule, missing positions, duplicate
  position titles or a choice position without an approved candidate.
- After voting opens, configuration is frozen. The only exception is an
  end-time extension while the election is `OPEN`, with a reason, an authorised
  publisher and an audit row.
- Every transition and configuration edit writes an `ElectionAuditLog` row with
  actor, action, reason/details, IP for people, and server time.

## 2. States and the transition matrix

The ten states are the SRS FR-CMS-05 states already defined in
`apps/api/internal/models/election.go`. `SUSPENDED` is **not** in the SRS and is
not implemented.

| From | Allowed to |
|---|---|
| `DRAFT` | `REGISTRATION`, `PENDING_APPROVAL`, `SCHEDULED`, `CANCELLED` |
| `REGISTRATION` | `PENDING_APPROVAL`, `SCHEDULED`, `CANCELLED` |
| `PENDING_APPROVAL` | `REGISTRATION`, `SCHEDULED`, `CANCELLED` |
| `SCHEDULED` | `OPEN`, `CLOSED`, `CANCELLED` |
| `OPEN` | `CLOSED`, `CANCELLED` |
| `CLOSED` | `RESULTS_UNDER_REVIEW`, `FINALIZED`, `ARCHIVED` |
| `RESULTS_UNDER_REVIEW` | `FINALIZED`, `ARCHIVED` |
| `FINALIZED` | `ARCHIVED` |
| `ARCHIVED` | terminal |
| `CANCELLED` | terminal |

The executable matrix is `internal/lifecycle/transitionMatrix`. Two arrows are
not reachable through the API:

- `SCHEDULED -> OPEN` is done automatically by the server clock.
- `CLOSED`/`RESULTS_UNDER_REVIEW -> FINALIZED` is done by
  `POST /elections/:id/certify`, which keeps the existing separation-of-duties
  checks.

`POST /elections/:id/transitions` refuses those two targets with a clear 409.

## 3. Boundary semantics

Decided once and tested:

```text
voting window = [starts_at, ends_at)
```

- At exactly `starts_at`, voting is open.
- At exactly `ends_at`, voting is closed.
- One nanosecond before `ends_at`, voting is open.
- `SCHEDULED` with `now >= ends_at` closes without ever opening.

`Election.IsOpen` and the lifecycle tests use the same rule.

## 4. Server time and scheduler downtime

Two mechanisms use the same `internal/lifecycle` code:

1. **Lazy reconciliation** (`lifecycle.Ensure`) runs on every ballot view,
   start and submit, on eligibility changes, and on lifecycle reads. A request
   after `starts_at` opens a due `SCHEDULED` election; a request after `ends_at`
   closes an `OPEN` election.
2. **Scheduler** (`elections:lifecycle`, every minute) calls
   `lifecycle.SyncAll`. It makes the transition timestamp timely when Redis and
   the worker are healthy, but it is not the correctness guarantee.

A missed tick or an API restart therefore repairs itself on the next request or
scheduler run. An ended election never accepts a ballot while waiting for the
repair. A scheduler task failure is retried by the queue, and failure to open
because configuration is invalid is fail-closed: the election stays
`SCHEDULED`, logs the field errors, and does not write `ELECTION_OPENED`.

## 5. Configuration validation

`ValidateBasics` checks the election's own fields on create/update. At
`SCHEDULED` and at opening, `ValidateConfiguration` additionally checks the
ballot:

- title is present; category is one of the nine SRS categories;
- result policy is `LIVE`, `AFTER_CLOSE` or `CERTIFIED_ONLY`;
- `starts_at < ends_at`, and the window is at least **1 hour**;
- scheduling requires `ends_at` to still be in the future;
- allowed domains are comma-separated bare domains (empty means roster-only);
- at least one position exists;
- position titles are unique within the election;
- question types and selection limits are valid, and only `MULTI_CHOICE` may
  take more than one selection;
- every `SINGLE_CHOICE` or `MULTI_CHOICE` position has at least one approved
  candidate. `YES_NO`/`REFERENDUM` positions do not need candidate rows.

Failure returns `422 VALIDATION_ERROR` with a per-field `details` map, so the
admin form can put each reason under its input.

### Timezones

- The API accepts RFC 3339 timestamps **with an explicit offset** or `Z`, for
  example `2026-11-01T09:00:00+03:00`.
- The handler converts to UTC before storing; the database stores UTC.
- A timestamp without an offset is rejected as ambiguous (`422`).
- Responses expose `starts_at`/`ends_at`/`server_time` in UTC and
  `display_timezone: "Africa/Kampala"` for the UI to format. That value is for
  display only.
- Tests cover a US spring-forward DST window where the wall clock jumps, to
  prove the stored absolute window is correct.

## 6. Freeze and exceptional changes

Configuration freezes when voting opens (`OPEN` and every later state).

Before `OPEN`, configuration is editable by `elections.edit` and audited as
`ELECTION_UPDATED`. Once open:

- title, description, category, organizing unit, start time, result policy,
  allowed domains and eligibility changes are **blocked** (`409`);
- the only accepted exception is moving `ends_at` **later** while the election
  is `OPEN`;
- that extension requires `elections.publish`, a non-empty reason, a valid new
  window and is audited as `ELECTION_EXTENDED` with old/new end times and the
  reason;
- extending an election that has already ended is refused: the lazy
  reconciliation closes it first, and closed elections cannot be extended;
- `CANCELLED` and `ARCHIVED` are terminal and cannot be edited.

Positions/candidates/roster routes are separate issues. When they are added they
must call the same freeze rule; today the election endpoint is the only mutable
configuration surface.

Early close, cancellation and archiving require `elections.publish` and a reason
(early close and cancellation; archive is audited as well). The same applies to
closing a `SCHEDULED` election before its window.

## 7. API

| Method | Path | Access | Purpose |
|---|---|---|---|
| `POST` | `/api/v1/elections` | `elections.create` | Create a `DRAFT`; schedule is required. |
| `GET` | `/api/v1/elections/:id` | `elections.view`, election-scoped | Metadata, phase, server time, allowed transitions. |
| `PUT` | `/api/v1/elections/:id` | `elections.edit`, election-scoped | Edit before open; audited end extension after open. |
| `POST` | `/api/v1/elections/:id/transitions` | `elections.edit`, election-scoped | Explicit state change; extra rules in the handler. |
| `GET` | `/api/v1/elections/:id/lifecycle` | any signed-in caller | Display-only phase and server time; no roster or ballot data. |

`POST /transitions` body:

```json
{ "to": "SCHEDULED", "reason": "nominations closed" }
```

`PUT /elections/:id` accepts any of `title`, `description`, `category`,
`organizing_unit`, `starts_at`, `ends_at`, `result_policy`,
`allowed_domains`, plus `reason`. After open, only `ends_at` (later) and
`reason` are accepted.

Permission checks happen twice, as in ISSUE-03: the route policy checks the
grant, and `policy.AuthorityFor` checks the caller's relationship to this
election. Unknown and out-of-scope ids both return `404`.

Cross-election isolation is preserved: a scoped official assigned to election A
cannot read or mutate election B.

## 8. Audit actions

Every row is an `election_audit_logs` row. Ballot choices never appear.

| Action | Written when |
|---|---|
| `ELECTION_CREATED` | a draft is created |
| `ELECTION_UPDATED` | configuration is edited before opening |
| `ELECTION_EXTENDED` | post-opening end-time exception, with reason |
| `ELECTION_SCHEDULED` | transition to `SCHEDULED` |
| `ELECTION_OPENED` | server clock or scheduler opens |
| `ELECTION_CLOSED` | server clock, scheduler or authorised early close |
| `ELECTION_CANCELLED` | authorised cancellation with reason |
| `ELECTION_REVIEW_STARTED` | transition to `RESULTS_UNDER_REVIEW` |
| `ELECTION_ARCHIVED` | transition to `ARCHIVED` |
| `RESULTS_CERTIFIED` | existing certification endpoint |

Automatic rows use `actor_id = "system"`; people get their user id and, where a
request exists, the client IP. A transition row carries `{from,to,reason}`.
A configuration row carries the changed field names, reason and old/new
schedule.

## 9. Tests and evidence

From `apps/api`:

```text
go build ./...                                                                  PASS
go test ./internal/lifecycle -count=1                                          PASS
go test ./internal/handlers -run 'TestElectionLifecycleAPI|TestOfficialCanEdit|TestSpoofedDate|TestTimezoneOffsets' -count=1  PASS
go test ./internal/eligibility ./internal/policy ./internal/routes ./internal/models ./internal/cron ./internal/jobs -count=1 PASS
go test ./internal/handlers -count=1                                            PASS
```

Covered:

- `TestTransitionMatrix` — every ordered pair of the ten states, including all
  forbidden pairs, against an independently written expected matrix.
- `TestOpenAndCloseBoundaries` — just before start, at start, just before end,
  at end, after end.
- `TestMissedSchedulerTickCannotExtendVoting` — `OPEN` past its end without a
  scheduler run is already closed to votes and is audited when reconciled.
- `TestEnsureDoesNotOpenInvalidConfiguration` — no positions means no open and
  no `ELECTION_OPENED` row.
- `TestSchedulerRestartReconcilesMissedTransitions` — a restart opens a due
  election and closes an ended one.
- `TestValidateConfiguration`, `TestValidateConfigurationRequiresPositionsAndCandidates`
  — schedule, duration, category, policy, domain, duplicate/invalid positions
  and missing approved candidates.
- `TestExceptionalCloseAndCancelRequireAReason`, `TestIllegalTransitionHasAClearError`.
- `TestElectionLifecycleAPIValidatesSchedulesFreezesAndAudits` — HTTP create,
  refused schedule without a ballot, schedule, freeze, extension with/without
  reason, manual-open refusal, early close and audit contents.
- `TestOfficialCanEditButCannotCancelOrExtend` — `elections.edit` versus
  `elections.publish`.
- `TestSpoofedDateHeaderCannotExtendVoting` — a `Date` header naming a time
  inside the window does not revive an ended election.
- `TestTimezoneOffsetsAreNormalizedAndNaiveTimesRejected` — DST offset handling,
  UTC storage, naive timestamp refusal.

The suite was run on Windows with SQLite (`github.com/glebarez/sqlite`). A
PostgreSQL run (`GRIT_TEST_DATABASE_URL`) is still required before release,
especially for the conditional status update and optimistic version update.

## 10. Known limitations

- No candidate/position/roster edit endpoints exist yet, so their freeze
  enforcement is a documented rule for ISSUE-06/08, not executable code here.
- No UI consumes these endpoints yet. The phase response exists for the future
  voter/admin screens.
- `SUSPENDED` is not implemented because it is not in the SRS. If the university
  adds it, extend `transitionMatrix` and the API tests together.
- Extension is the only post-open exception. Reopening a closed election and
  changing eligibility after the freeze require separate, explicitly approved
  procedures.
- The scheduler needs Redis; without it the lazy path is still correct but the
  audit timestamp waits for the next request.
