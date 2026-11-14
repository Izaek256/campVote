# Restricted elections (issue #37)

**Status:** implemented on `feature/37-planned-features`.
**Scope:** an election whose electorate is an explicit list rather than an
eligibility rule.

## What changes

`Election.RequireRoster` turns the existing `election_rosters` table into the
whole electorate:

- **Seeing.** `electionVisibleTo` returns true only for the election's creator,
  an account with a role on the election, or an account whose email is on the
  roster. This applies before the usual "closed elections are public" shortcut,
  so closing a restricted election does not publish its existence.
- **Voting.** `eligibility.CheckRoster` denies a caller with no roster row when
  `RequireRoster` is set. A row with a non-`ELIGIBLE` status is still refused.
  The profile gate in `CheckElection` is unchanged.
- **Other elections.** Being on one restricted list does not affect any other
  election; those still use their own rule or list.

## Managing the list

The roster is the same table the eligibility importer writes, so nothing new is
stored.

- `POST /api/v1/elections/:id/roster/preview` and `.../import` accept the CSV
  (`student_number,email,full_name` required). Re-importing updates rows in
  place.
- `GET /api/v1/elections/:id/roster` lists the rows for staff (`rosters.view`).
- `POST /api/v1/elections/:id/roster/:rosterId/status` revokes or restores one
  row with a mandatory reason, written to the election audit log.

For an unrestricted election the existing freeze still applies: after voting
opens, new entries are refused. For a restricted election the freeze is skipped,
because the list *is* the electorate and an officer may still need to add or
remove a name while polls are open. Both kinds refuse changes once the election
is closed or its end time has passed (`ErrElectionClosed`, HTTP 409).

## UI

On `/staff/elections/:id` the eligibility form carries an **Only voters on the
imported list may see or vote** checkbox. When it is on, a **Voter list** panel
appears with a CSV import and a row per voter, plus a **Remove** action that
revokes with a typed reason. The panel is hidden when the toggle is off.

## Tests

- `apps/api/internal/eligibility/eligibility_test.go` —
  `TestRestrictedElectionNeedsARosterEntry` (no row refused, row accepted) and
  `TestRestrictedRosterCanChangeUntilClose` (import works after freeze, refused
  after the end time).
- `apps/api/internal/handlers/election_lifecycle_test.go` —
  `TestRestrictedElectionIsHiddenFromUnlistedStudents` (hidden before and after
  closing, visible once listed).

## Known limits

- There is no pagination or search on the roster panel; it shows the first 2000
  rows in email order.
- The roster stores whatever the CSV says. Nothing verifies that the email
  belongs to the student number beyond the import's own uniqueness checks.
  belongs to the student number beyond the import's own uniqueness checks.
