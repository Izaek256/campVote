# Rejected and withdrawn candidates appeared on the results page

**Branch:** `hotfix/candidate-results-visibility`
**Reported:** maintainer — a candidate rejected or withdrawn before the election opened did not appear on the ballot, but appeared on the results page after close with 0 votes.
**Status:** Fixed; regression test added.

---

## Symptom

A candidate whose application was rejected, or who withdrew, was correctly
absent from the ballot. After the election closed the results page listed the
same candidate with `0` votes and `0%` share, although they had never stood.

## Root cause

The ballot and the tally read the candidate table differently.

- `BallotContentFor` (`apps/api/internal/handlers/ballot_version.go`) loads only
  candidates with `approval_status = APPROVED`, so rejected, withdrawn, pending
  and nominated candidates never reach the ballot.
- `results.Compute` (`apps/api/internal/results/service.go`) preloaded **every**
  candidate of the position. The tally therefore emitted a result row for each
  non-standing candidate, which the results page rendered with zero votes.

## Fix

`results.Compute` now preloads only approved candidates, matching the ballot:

```go
Preload("Candidates", "approval_status = ?", models.CandidateStatusApproved)
```

Both the close hook and a manual tally/recount use this one path, so the
provisional snapshot written at close no longer contains non-standing
candidates.

## Verification

- `gofmt -l .` — clean; `go vet ./internal/results/ ./internal/handlers/` — PASS.
- `go test ./internal/results/ ./internal/tally/ ./internal/handlers/ -count=1` — PASS.
- Regression coverage: `closedElection` in
  `apps/api/internal/results/service_test.go` now seeds one approved candidate
  plus one rejected and one withdrawn candidate, and asserts the tally reports
  the single standing candidate only.
- Results handler fixtures now mark their candidates `APPROVED`, as a real
  ballot would.

## Remaining limitations

- Snapshots already stored before this fix keep their content, because a
  snapshot is an immutable, hashed record. Affected elections need a new tally
  (a correction where one was already certified) to drop the non-standing rows.
- Withdrawals are only accepted while the election is `DRAFT`, so filtering by
  `APPROVED` cannot hide a candidate who had already received votes.
