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
- `POST /api/v1/elections/:id/roster/entries` adds pasted addresses. The server
  splits on commas, semicolons, spaces and newlines, trims and lower-cases each
  one, drops repeats, validates the address, and derives the student number and
  starting name from the local part. One bad address refuses the whole paste.
- `PUT /api/v1/elections/:id/roster/:rosterId` corrects an address, student
  number or name. Uniqueness per election is still enforced.
- `GET /api/v1/elections/:id/roster` lists the rows for staff (`rosters.view`).
- `POST /api/v1/elections/:id/roster/:rosterId/status` revokes or restores one
  row with a mandatory reason, written to the election audit log.

For an unrestricted election the existing freeze still applies: it is taken
automatically when voting opens, and from then on new entries are refused. For a
restricted election the freeze is skipped, because the list *is* the electorate
and an officer may still need to add or remove a name while polls are open. Both
kinds refuse changes once the election is closed or its end time has passed
(`ErrElectionClosed`, HTTP 409).

A revoked row no longer grants visibility: the audience check counts only
`ELIGIBLE` rows, so removing a voter takes them off both the list and the ballot.

## UI

On `/staff/elections/:id` the eligibility form carries an **Only voters on the
imported list may see or vote** checkbox. The **Voter list** panel is always
shown: with the checkbox off it explains how to enable the list, and with it on
it offers two clearly labelled ways to add names — a textarea for pasting
addresses and a CSV import with a downloadable template — plus a row per voter
with **Edit**, **Remove** (revoke) and **Restore** actions. The residence rule
shows the chosen areas as removable pills, picked from the active residence
areas; saved names that no longer exist stay visible so editing the form cannot
drop them silently.

## Tests

- `apps/api/internal/eligibility/eligibility_test.go` —
  `TestRestrictedElectionNeedsARosterEntry` (no row refused, row accepted),
  `TestRestrictedRosterCanChangeUntilClose` (import works after freeze, refused
  after the end time), `TestManualRosterAddAndEdit` and
  `TestRestrictedVoterAddedAfterFreezeCanVote` (a late addition can vote).
- `apps/api/internal/handlers/eligibility_test.go` —
  `TestManualRosterAddEditAndRestore` (permissions, server-side cleaning, edit,
  revoke/restore).
- `apps/api/internal/handlers/election_lifecycle_test.go` —
  `TestRestrictedElectionIsHiddenFromUnlistedStudents` (hidden before and after
  closing, visible once listed, hidden again after revocation).

## Known limits

- There is no pagination or search on the roster panel; it shows the first 2000
  rows in email order.
- The roster stores whatever the CSV says. Nothing verifies that the email
  belongs to the student number beyond the import's own uniqueness checks. A
  manually added address is trusted the same way.
