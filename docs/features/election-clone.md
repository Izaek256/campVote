# Copying an election (issue #37)

**Status:** implemented on `feature/37-planned-features`.
**Scope:** re-run a recurring poll without rebuilding it by hand.

## What a copy contains

`POST /api/v1/elections/:id/clone` creates a new `DRAFT` election. It copies:

- title (unless a new one is sent), description, category, organising unit,
  result policy and the eligibility rule (domain, years, courses,
  `require_roster`);
- every position of the source election, in display order;
- every candidate that is not `WITHDRAWN` or `REJECTED`, including application
  details and uploaded document links, remapped to the copied positions;
- the whole `election_rosters` list, with statuses kept.

It deliberately copies **none** of the history:

- no ballots, selections or participation records;
- no result snapshots or recounts;
- no ballot versions;
- no `ElectionRole` assignments — the caller becomes the copy's creator.

The new schedule is supplied by the caller (`starts_at`, `ends_at`, RFC 3339);
the copy starts as a draft whatever state the source is in.

## Permissions

The route requires `elections.create` on the source election. A global
administrator, the source creator, and a scoped `CHIEF_ELECTION_ADMIN` pass;
`ELECTION_OFFICIAL` does not, because a copy is a new election rather than work
on the existing one. The copy is audited as `ELECTION_CLONED` with the source
id, the new schedule and the number of positions and candidates copied.

## UI

On `/staff/elections/:id` a **Re-run this election** panel takes a title
(optional), an opening time and a closing time. A successful copy redirects to
the new election's page.

## Tests

`apps/api/internal/handlers/election_lifecycle_test.go` —
`TestCloneCopiesTheBallotAndListButNoHistory` checks the copied config, the
remapped candidate, the roster, and the absence of ballots, ballot versions and
role assignments; it also checks that an unrelated student is refused.

## Known limits

- A copied roster keeps revoked statuses. An officer can restore a row in the
  new election through the roster endpoint.
- The copied candidates keep their approval status and ballot number, because
  the same documents were already reviewed. Re-screening them is not required,
  but nothing prevents an officer from reviewing them again.
