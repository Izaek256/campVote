# Election Lifecycle, Schedule and Configuration (ISSUE-05)

Status: **implemented and tested**, three-stage model recorded 2026-11-03 on
`feature/mvp-phases-3-7`. The branch needs a PR into `develop`.

Scope: election states, legal transitions, scheduling, pre-open validation,
configuration freeze, audited exceptional changes, and the server clock.
Out of scope: ballot content (ISSUE-06), vote storage and tallying (ISSUE-07/08),
and the voter-facing tally policy (ISSUE-09).

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
- An election that has not yet collected a ballot can be deleted by an
  administrator; once a ballot exists, deletion is refused so the evidence of the
  contest is never destroyed.
- Every transition, configuration edit and deletion writes an `ElectionAuditLog`
  row with actor, action, reason/details, IP for people, and server time.

## 2. States and the transition matrix

The lifecycle has **three stages**. The earlier ten SRS states
(`REGISTRATION`, `PENDING_APPROVAL`, `SCHEDULED`, `RESULTS_UNDER_REVIEW`,
`FINALIZED`, `ARCHIVED`, `CANCELLED`, `SUSPENDED`) were reduced to the minimum
that the product needs. `models.ElectionStatusDraft/Open/Closed` are the only
status constants.

| From | Allowed to |
|---|---|
| `DRAFT` | `OPEN`, `CLOSED` |
| `OPEN` | `CLOSED` |
| `CLOSED` | terminal |

The executable matrix is `internal/lifecycle/transitionMatrix`. Two arrows are
not reachable through `POST /elections/:id/transitions`:

- `DRAFT -> OPEN` is done automatically by the server clock at `starts_at`,
  once `ReadyToOpen` passes.
- `DRAFT -> CLOSED` is the *lapsed draft*: the clock closes a draft whose window
  ended before it ever opened, so no election is left unresolved forever.

`POST /transitions` refuses a client-supplied `to: OPEN` with a `409`
("Elections open automatically at their start time"). Closing before `ends_at`
is the only manual transition and it is an *early close* (see §6).

Certification no longer moves the election on: there is no `FINALIZED` stage.
`POST /elections/:id/certify` relabels the current result snapshot from
`PROVISIONAL` to `CERTIFIED` and is only valid while the election is `CLOSED`.

## 3. Boundary semantics

Decided once and tested:

```text
voting window = [starts_at, ends_at)
```

- At exactly `starts_at`, voting is open.
- At exactly `ends_at`, voting is closed.
- One nanosecond before `ends_at`, voting is open.
- A `DRAFT` with `now >= ends_at` closes without ever opening.

`Election.IsOpen` and the lifecycle tests use the same rule.

## 4. Server time and scheduler downtime

Two mechanisms use the same `internal/lifecycle` code:

1. **Lazy reconciliation** (`lifecycle.Ensure`) runs on every ballot view,
   start and submit, on eligibility changes, and on lifecycle reads. A request
   after `starts_at` opens a due `DRAFT` election; a request after `ends_at`
   closes an `OPEN` election and closes a lapsed draft.
2. **Scheduler** (`elections:lifecycle`, every minute) calls
   `lifecycle.SyncAll`. It makes the transition timestamp timely when Redis and
   the worker are healthy, but it is not the correctness guarantee.

A missed tick or an API restart therefore repairs itself on the next request or
scheduler run. An ended election never accepts a ballot while waiting for the
repair. A scheduler task failure is retried by the queue, and failure to open
because configuration is invalid is fail-closed: the election stays `DRAFT`,
logs the field errors, and does not write `ELECTION_OPENED`.

## 5. Configuration validation

`ValidateBasics` checks the election's own fields on create/update.
`ValidateConfiguration` additionally checks the ballot, and is used by the
readiness endpoint and at opening:

- title is present; category is a known SRS category;
- result policy is `LIVE`, `AFTER_CLOSE` or `CERTIFIED_ONLY`;
- `starts_at < ends_at`, and the window is at least **1 hour**;
- readiness additionally requires `ends_at` to still be in the future;
- allowed domains are comma-separated bare domains (empty means roster-only);
- at least one position exists;
- position titles are unique within the election;
- question types and selection limits are valid, and only `MULTI_CHOICE` may
  take more than one selection;
- every `SINGLE_CHOICE` or `MULTI_CHOICE` position has at least one approved
  candidate. `YES_NO`/`REFERENDUM` positions do not need candidate rows.

`ReadyToOpen` is `ValidateConfiguration` plus the window check (`starts_at <=
now < ends_at`). Failure returns `422 VALIDATION_ERROR` with a per-field
`details` map, so the admin form can put each reason under its input.

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

## 6. Freeze, exceptional changes and deletion

Configuration freezes when voting opens (`OPEN` and `CLOSED`). Before `OPEN`,
configuration is editable by `elections.edit` and audited as `ELECTION_UPDATED`.
Once open:

- title, description, category, organizing unit, start time, result policy,
  allowed domains and eligibility changes are **blocked** (`409`);
- the only accepted exception is moving `ends_at` **later** while the election
  is `OPEN`;
- that extension requires `elections.publish`, a non-empty reason, a valid new
  window and is audited as `ELECTION_EXTENDED` with old/new end times and the
  reason;
- extending an election that has already ended is refused: the lazy
  reconciliation closes it first, and closed elections cannot be extended.

An **early close** (`OPEN -> CLOSED` before `ends_at`) requires
`elections.publish` and a non-empty reason, and is audited as
`ELECTION_CLOSED`. A close at or after `ends_at` is performed by the clock and
needs nothing from a caller.

`DELETE /elections/:id` requires `elections.edit` (election-scoped). It runs
`Ensure` first, then counts stored `AnonymousBallot` rows. If any ballot exists
the request is refused with `409` ("close it instead"). Otherwise the deletion,
its `ELECTION_DELETED` audit row, the election's candidates and its positions
are removed in one transaction. Ballot choices are never involved.

## 7. API

| Method | Path | Access | Purpose |
|---|---|---|---|
| `POST` | `/api/v1/elections` | `elections.create` | Create a `DRAFT`; schedule is required. |
| `GET` | `/api/v1/elections/:id` | `elections.view`, election-scoped | Metadata, phase, server time, allowed transitions. |
| `PUT` | `/api/v1/elections/:id` | `elections.edit`, election-scoped | Edit before open; audited end extension after open. |
| `DELETE` | `/api/v1/elections/:id` | `elections.edit`, election-scoped | Delete an election that has no recorded ballots. |
| `POST` | `/api/v1/elections/:id/transitions` | `elections.edit`, election-scoped | Explicit state change; extra rules in the handler. |
| `GET` | `/api/v1/elections/:id/lifecycle` | any signed-in caller | Display-only phase and server time; no roster or ballot data. |

`POST /transitions` body:

```json
{ "to": "CLOSED", "reason": "counting under way" }
```

`PUT /elections/:id` accepts any of `title`, `description`, `category`,
`organizing_unit`, `starts_at`, `ends_at`, `result_policy`, `allowed_domain`,
`allowed_years`, `allowed_courses`, plus `reason`. After open, only `ends_at`
(later) and `reason` are accepted.

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
| `ELECTION_OPENED` | server clock or scheduler opens |
| `ELECTION_CLOSED` | server clock, scheduler or authorised early close |
| `ELECTION_DELETED` | an election with no ballots is deleted |
| `RESULTS_CERTIFIED` | certification endpoint |

Automatic rows use `actor_id = "system"`; people get their user id and, where a
request exists, the client IP. A transition row carries `{from,to,reason}`.
A configuration row carries the changed field names, reason and old/new
schedule.

## 9. Tests and evidence

From `apps/api`:

```text
go test ./internal/handlers/... ./internal/lifecycle/... ./internal/policy/... ./internal/results/...  PASS
```

Covered:

- `TestTransitionMatrix` — every ordered pair of the three states, including all
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
- `TestExceptionalCloseRequiresAReason`, `TestIllegalTransitionHasAClearError`.
- `TestAdminCanDeleteDraftButNotAnElectionWithBallots` — a draft deletes and
  leaves no rows; an election with an `AnonymousBallot` returns `409` and
  survives.
- `TestElectionLifecycleAPIValidatesSchedulesFreezesAndAudits` — HTTP create,
  freeze, extension with/without reason, manual-open refusal, early close and
  audit contents.
- `TestOfficialCanEditButCannotCancelOrExtend`, plus the results RBAC matrix
  which now reads a `CERTIFIED_ONLY` election at `CLOSED`.

The suite was run on Windows with SQLite (`github.com/glebarez/sqlite`). A
PostgreSQL run (`GRIT_TEST_DATABASE_URL`) is still required before release,
especially for the conditional status update and optimistic version update.

## 10. Known limitations

- No candidate/position/roster edit endpoints exist yet, so their freeze
  enforcement is a documented rule, not executable code here.
- `SUSPENDED` is not implemented because it is not needed by the three-stage
  model. If the university adds a suspension state, extend `transitionMatrix`
  and the API tests together.
- Extension is the only post-open configuration exception. Reopening a closed
  election and changing eligibility after the freeze require separate,
  explicitly approved procedures.
- The scheduler needs Redis; without it the lazy path is still correct but the
  audit timestamp waits for the next request.
