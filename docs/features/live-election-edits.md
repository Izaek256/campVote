# Editing a live election (issue #37)

**Status:** implemented on `feature/37-planned-features`.
**Scope:** the small set of changes an `OPEN` election still accepts.

## Allowed while open

- **Title** — corrected through `PUT /api/v1/elections/:id`.
- **Description** — same call.
- **A later end time** — `ends_at` may be moved forward only.

Everything else is refused with `409`: start time, category, organising unit,
result policy, eligibility rule, `require_roster` and public listing. Moving the
end time earlier is refused too; the way to stop voting early is the close
transition.

The start-time input is disabled in the UI while the election is open, so the
form cannot submit it by accident.

## Audit

Each accepted live edit writes `ELECTION_UPDATED` with `"live_edit": true` in
the details. A post-opening extension still needs:

- a stated `reason`;
- the `elections.publish` permission;
- a valid schedule (after the start, at least one hour long, still in the
  future).

It is recorded as `ELECTION_EXTENDED` with the old and new end times, the
reason, and `"live_edit": true`.

## Tests

`apps/api/internal/handlers/election_lifecycle_test.go` —
`TestElectionLifecycleAPIValidatesSchedulesFreezesAndAudits` checks that a title
and description edit succeeds after opening, that an eligibility or category
change is refused, and that the audit row is marked `live_edit`.
`TestElectionOfficialCanEditButNotPublishOrClose` checks the permission split
for extensions and closes.

## Known limits

- The rule is enforced by comparing changed fields; a request that changes
  nothing is treated as a no-op by the normal update path.
- A typo in the schedule cannot be corrected while open; close and copy the
  election instead.
