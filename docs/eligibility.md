# Eligibility Verification (ISSUE-04)

Status: **implemented and tested on branch `feat/issue-04-eligibility-verification`**.
Awaiting review and merge into `develop`. Ballot versioning, tallying and the
voter/admin UI are separate issues (ISSUE-06/08).

## 1. What this guarantees

- Eligibility is decided **per election**, on the server, from an authoritative
  roster entry. A valid institutional email domain alone never grants a vote.
- The ballot view, start and submit endpoints all run the same
  `eligibility.CheckElection` gate. Hiding a button is never the control.
- An account's email is verified before it can use a roster entry, so changing
  the address on the profile cannot borrow an unregistered student's vote.
- Import is preview-then-commit, atomic, provenance-recorded and audited.
- Users cannot edit their own eligibility: the profile endpoint ignores roster
  fields, and every roster/import/freeze/report route requires an
  election-scoped officer permission.

## 2. Evidence model and identity key

| Concept | Value |
|---|---|
| Authoritative evidence | `election_rosters` row for the election |
| Identity key | normalized institutional email (`LOWER(TRIM(email))`) |
| Secondary key | `student_number`, unique per election |
| Domain prerequisite | `elections.allowed_domains` (comma-separated; empty means "roster only") |
| Account proof | `users.email_verified_at IS NOT NULL` |

`Election.RequireRoster` is **deprecated**. It is kept so old rows migrate, but
it no longer grants or denies access by itself; a roster entry is always
required.

The email is the join between an account and a roster row, which is why the
account must have verified that address. Registration, OAuth/SSO and first-admin
bootstrap set `email_verified_at`. Changing the email through `PUT /api/profile`
clears it until the new address is confirmed.

## 3. Server-side gate

`internal/eligibility/service.go::CheckElection` evaluates, in order:

1. election exists and `IsOpen(now)` (status + schedule);
2. account is present and its email is verified;
3. email matches `allowed_domains` (when configured);
4. a roster row exists for `election_id` + email;
5. the row's status is `ELIGIBLE`;
6. the row was not created after the eligibility freeze.

Every refusal returns a voter-safe reason. It never names another person or
exposes roster contents. Denials on the ballot gate are logged with
`policy.LogDenial`.

`Submit` rechecks eligibility and open status **inside** the transaction that
writes `voter_participations` and the anonymous `anonymous_ballots` row, so a
revocation that lands between the HTTP gate and the write still refuses the
ballot.

## 4. Roster CSV import

Endpoints:

| Method | Path | Permission |
|---|---|---|
| `POST` | `/api/v1/elections/:id/roster/preview` | `rosters.import` |
| `POST` | `/api/v1/elections/:id/roster/import` | `rosters.import` |
| `POST` | `/api/v1/elections/:id/roster/:rosterId/status` | `rosters.edit` |
| `POST` | `/api/v1/elections/:id/eligibility/freeze` | `rosters.edit` |
| `GET` | `/api/v1/elections/:id/eligibility/report` | `rosters.view` |

The upload may be `multipart/form-data` with a `file` field, or a raw
`text/csv` body (scripts and tests). Maximum size is 8 MiB.

Required columns: `student_number`, `email`, `full_name`.
Optional columns: `reg_number`, `faculty`, `department`, `programme`,
`study_year`, `status`.

Header matching is case-insensitive, ignores a UTF-8 BOM, and treats spaces as
underscores. Each row is validated:

- `student_number`, `email` and `full_name` are required;
- the email must be a bare valid address;
- `study_year` must be an integer 1–10 (default 1);
- `status` must be `ELIGIBLE`, `INELIGIBLE`, `GRADUATED` or `SUSPENDED`
  (default `ELIGIBLE`);
- duplicate student numbers within the file are rejected;
- duplicate emails within the file are rejected;
- an email already belonging to a different student number in the same election
  is rejected as a duplicate identity.

Preview writes nothing. It returns counts (`creates`, `updates`, `unchanged`,
`revokes`), per-line errors and a SHA-256 content hash. Import validates again
inside one transaction: **any invalid row refuses the whole file**, so preview
and commit cannot disagree.

Commit records an `eligibility_import_batches` row (election, officer, source,
file name, content hash, counts) and stamps each roster row with `source`,
`import_batch_id` and `imported_by_id`. Real registrar rolls are never committed
to the repository or used in tests; tests use synthetic rows only.

## 5. Freeze, revocation and overrides

- `Freeze` is explicit and only allowed while the election is `OPEN` and inside
  its schedule. It snapshots `eligibility_frozen_at` and the eligible count and
  writes `ELIGIBILITY_FROZEN`.
- After the freeze, imports may **revoke** (`status != ELIGIBLE`) but may not
  add rows, change identity fields, or restore eligibility. Restoring is an
  explicit officer override with its own audit action.
- `SetStatus` requires a reason, refuses terminal election statuses and refuses
  once `ends_at` has passed, and writes `ROSTER_REVOKED` or
  `ELIGIBILITY_OVERRIDE`.
- Revoking eligibility **after a ballot was accepted does not delete or
  invalidate that ballot or its participation record**. It stops future ballot
  access only. This is a policy choice that needs stakeholder sign-off (see
  §8).

## 6. Audit

Election audit actions added for this feature:

| Action | Written when |
|---|---|
| `ROSTER_IMPORTED` | a CSV import commits (details carry batch, hash and counts) |
| `ROSTER_REVOKED` | one row is revoked by import or status change |
| `ELIGIBILITY_OVERRIDE` | an officer restores eligibility explicitly |
| `ELIGIBILITY_FROZEN` | the electorate snapshot is taken |

Audit rows contain roster IDs, statuses, reasons and counts. They never contain
ballot selections or a voter-to-choice mapping.

## 7. Report

`GET /api/v1/elections/:id/eligibility/report` returns counts only:
total, eligible, counts by status, participation, turnout, freeze snapshot and
the last import batch. It deliberately contains no email, student number, name
or user ID, and tests assert those strings are absent.

## 8. Open policy questions for stakeholder sign-off

1. **Revocation after a vote:** should an accepted ballot be discarded when the
   voter is later found ineligible, or should it stand? Current behaviour:
   it stands.
2. **Identity proof:** is verified institutional email acceptable, or must the
   university provide SSO/OIDC that asserts student number and status?
3. **Freeze timing:** should the snapshot happen automatically when the election
   opens, or stay a deliberate officer action (current behaviour)?
4. **Duplicate identities:** is email the agreed identity key, or should student
   number be authoritative when the two disagree?
5. **Who may import:** which officers may upload a roll, and is a second
   approver required before a roll becomes authoritative?
6. **Disputes:** who adjudicates an eligibility dispute during voting, and what
   evidence is required?

## 9. Test evidence

Run from `apps/api`:

```text
go build ./...                                                       PASS
go test ./internal/eligibility ./internal/handlers -run 'TestEligibility|TestDomain|TestClock|TestFreeze|TestImport|TestReport|TestVoterCannot|TestRosterImport|TestUnverified|TestMultiChoice' -count=1   PASS
go test ./internal/routes ./internal/policy ./internal/models ./internal/database ./internal/authz -count=1   PASS
```

Covered paths:

- domain-only email refused; listed eligible accepted; case-insensitive email;
- per-election isolation;
- before-open and after-close clock refusals;
- unverified email refused even when it matches a roster row;
- freeze refuses new rows, allows revocation, refuses import restore, records
  the override;
- freeze/status refused after close;
- import preview rejects malformed, duplicate and out-of-range rows without
  writing; a refused import writes nothing;
- import provenance and audit;
- duplicate identity across student numbers;
- report contains counts only;
- voter cannot change roster fields or reach officer routes;
- ballot view/start/submit gate and cross-election tamper;
- multi-choice accepts distinct choices and rejects the same choice twice.

## 10. Known limitations

- The ballot endpoints here are the minimum needed to enforce eligibility.
  Ballot versioning, confirmation receipts, tallying and result publication are
  ISSUE-06/08.
- `Election.RequireRoster` remains on the model as a deprecated column.
- Tests run on SQLite; the concurrency tests for duplicate submissions still
  need a PostgreSQL run (`GRIT_TEST_DATABASE_URL`) before release.
- No UI yet: the import/preview and voter eligibility screens are frontend work.
