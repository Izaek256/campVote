# Certification (issue #37)

**Status:** implemented on `feature/37-planned-features`.
**Scope:** who may certify a result, and what every results view then reads.

## Who may certify

`POST /api/v1/elections/:id/results/certify` requires:

- the `elections.certify` permission on the election (a global administrator,
  the creator, or an assigned `CHIEF_ELECTION_ADMIN`);
- the election to be `CLOSED`;
- the actor to be the creator, an assigned official/chief, or a global
  administrator.

The previous rule refused the creator and required a second assigned official,
so a small class or club election with a single administrator could never be
certified. The creator now certifies its own result and a single official is
enough. The safe minimum stays: an account with no relationship to the election
still cannot certify.

## What is stored

Certifying relabels the current snapshot from `PROVISIONAL` to `CERTIFIED` in a
transaction that also writes `RESULTS_CERTIFIED` to the election audit log. The
snapshot keeps `certified_by_id`, `certified_at` and `certify_reason`.

A certified snapshot is immutable. A later tally is a new version that
supersedes it; the original stays on the record.

## What every view reads

- The officer results page and history read the stored snapshot and show
  `CERTIFIED`, the certifier and the timestamp.
- The voter results page reads the certified snapshot from
  `GET /elections/:id/results`; the API prefers a certified snapshot for
  non-staff callers.
- The candidate view reads the same snapshot status.
- The downloadable reports render the same snapshot.

## Tests

- `apps/api/internal/policy/election_test.go` —
  `TestCertifyIsOpenToTheCreatorAndAnyAssignedOfficial`.
- `apps/api/internal/handlers/results_test.go` —
  `TestElectionCreatorCertifiesAndEveryViewReadsTheCertifiedSnapshot` checks the
  creator's certification, the stored certifier/timestamp, and that the officer
  and voter views both read `CERTIFIED`.

## Known limits

- There is no second-person approval step. The protection is that only the
  election's own officials, its creator or a system administrator can certify.
- Certification does not move the election on; the three-stage lifecycle has no
  `FINALIZED` state.
