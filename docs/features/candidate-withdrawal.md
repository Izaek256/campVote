# Candidate withdrawal after approval (issue #37)

**Status:** implemented on `feature/37-planned-features`.
**Scope:** an approved candidate can step down until the poll opens.

## Rule

A candidate may withdraw while the election is still `DRAFT`. Once the election
opens the ballot is fixed: withdrawing would change the contest after voters
started voting, so both the self-service and the officer route answer `409`.

The lifecycle is reconciled with `lifecycle.Ensure` before the check, so an
election whose start time has passed is treated as open even if no request had
updated its status yet.

## Evidence

A withdrawal must carry at least one of:

- a typed `reason` (up to 2000 characters), or
- an uploaded `document_url` — a signed letter uploaded through the normal
  upload endpoint.

Either is accepted; both may be sent. The columns are
`withdrawal_reason` and `withdrawal_document_url` (added to `candidates` by
AutoMigrate).

## Endpoints

- `POST /api/v1/candidates/my/:candidate_id/withdraw` — the candidate's own
  record (`nominated_by` must be the caller). Body: `{ "reason"?, "document_url"? }`.
- `POST /api/v1/elections/:id/candidates/:candidate_id/withdraw` — an officer
  recording the same decision. Requires `candidates.edit` on the election and is
  gated by the same draft-only window.

Both write `CANDIDATE_WITHDRAWN` to the election audit log, naming the actor and
the reason (or the letter link). Withdrawn candidates are filtered out of every
ballot version, which selects `APPROVED` candidates only.

## UI

- `/candidate`: approved candidates get a **Withdraw** action before voting
  opens. The dialog takes a reason and an optional signed PDF.
- `/staff/elections/:id/candidates`: approved candidates in a `DRAFT` election
  get a **Withdraw** action with a required reason.

## Tests

`apps/api/internal/handlers/candidate_workflow_test.go` —
`TestApprovedCandidateCanWithdrawOnlyBeforeTheElectionOpens` (reason or letter
required, audit written, refused once open) and
`TestAdministratorConfirmsAWithdrawalBeforeTheElectionOpens` (officer route and
the same window).

## Known limits

- There is no ballot-version handling for a withdrawal: the window closes before
  any version is published, so there is nothing to re-version.
- A candidate cannot undo a withdrawal. Officials would nominate them again.
