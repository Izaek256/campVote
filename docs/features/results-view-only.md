# Results are view-only for voters (issue #37)

**Status:** implemented on `feature/37-planned-features`.
**Scope:** voters read results on screen; downloadable ledgers stay with staff.

## What changed

- The **Download PDF** action is gone from the voter pages
  `/results` and `/elections/:id/results`.
- The only remaining result documents are the officer reports
  (`GET /api/v1/elections/:id/reports/summary.pdf` and
  `.../verification.pdf`), which sit behind the staff routes and the
  `audit.view` permission. A voter cannot reach them.
- Staff keep the download on `/staff/elections/:id/results`, including the
  election information sheet.

The results screen itself is unchanged: it still shows the published snapshot
with its shares, turnout and certification label, and still obeys the
election's result policy. What a voter can do with the data is now read-only.

## Why

The downloadable ledger was a client-side rendering of the same published
snapshot, so it exposed nothing a voter could not already read. Removing it
keeps the voter surface view-only and leaves document distribution to the
officers who can account for it.

## Tests

No server behaviour changed. The access boundary is the existing policy
`audit.view` on the two report routes, covered by
`apps/api/internal/policy/election_test.go` and the route policy table. The web
pages were checked with `npx biome check .` and `npx tsc --noEmit`.
