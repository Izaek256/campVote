# Students could not see the position (and election) an administrator had created

**Branch:** `feature/mvp-phases-3-7`
**Reported:** maintainer — "I made a position and an election and linked them, but when I log in as student I don't see any positions."
**Status:** Fixed; regression test added.

---

## Symptom

An administrator created a position, created an election and linked the two.
Signing in as the student, `/positions` ("Run for a position") showed the empty
state and the election was likewise missing from the student's Elections list.

## Reproduction

1. As an administrator, create a position and an election and link them (the
   linked position was `Guild President 26/27`, `OPEN`, linked to the
   `REGISTRATION` election of the same name).
2. Sign in as the student account `2024bse088@std.must.ac.ug`.
3. `/positions` returns `{"positions":[]}`.

## Root cause

Two layers wrote the same hard-coded campus domain into every new row.

`GET /positions/eligible` returns a position only when
`eligibility.RuleForPosition(...).Allows(identity)` is true, and
`Rule.Allows` requires `identity.Domain == rule.AllowedDomain` whenever
`AllowedDomain` is non-empty. The student's domain is `std.must.ac.ug`; the
stored rule said `std.mast.ac.ug`, so the position was filtered out on the
server, not merely hidden in the UI.

That `std.mast.ac.ug` came from two independent defaults:

| Layer | Location | Behaviour |
|---|---|---|
| Position form | `apps/web/app/staff/positions/page.tsx` — `emptyForm.allowed_domain` | Pre-filled the field, so every position saved that domain. |
| Election model | `apps/api/internal/models/election.go` — `AllowedDomain string \`gorm:"...;default:'std.mast.ac.ug'"\`` | The GORM default was applied on **every insert**, so even a blank value was stored as `std.mast.ac.ug`. |
| Election handler | `apps/api/internal/handlers/election_lifecycle.go` — `Create` | Force-defaulted a blank `AllowedDomain` to `std.mast.ac.ug` before validating. |

`models.Position.AllowedDomain` carries **no** database default, so positions
were only affected by the form; elections were affected by all three.

## Fix

Blank now means "any student" everywhere, matching the confirmed decision.

1. `apps/web/app/staff/positions/page.tsx` — `emptyForm.allowed_domain` is `""`
   (the field is labelled *optional* and says "Leave blank for any student").
2. `apps/api/internal/models/election.go` — removed the
   `default:'std.mast.ac.ug'` GORM tag from `Election.AllowedDomain`.
3. `apps/api/internal/handlers/election_lifecycle.go` — removed the
   `if election.AllowedDomain == "" { ... = "std.mast.ac.ug" }` force-default.
4. Existing rows were corrected in the development database (may 2026-11-03):
   `positions.allowed_domain` and `elections.allowed_domain` were blanked for
   the linked position/election, and
   `ALTER TABLE elections ALTER COLUMN allowed_domain DROP DEFAULT;` removed the
   column default left over from the old model tag.

`lifecycle.validDomains` already treats an empty value as valid
("roster-only eligibility"), so no validation change was needed.

## Verification

- New regression test
  `TestElectionCreateLeavesDomainOpenToAnyStudent`
  (`apps/api/internal/handlers/election_lifecycle_test.go`): creates an election
  with no domain, asserts the stored `allowed_domain` is empty, publishes it and
  asserts a voter at `std.must.ac.ug` can see it. It fails against the previous
  model tag and passes after the fix.
- `go build ./...` — PASS. `go test ./internal/handlers/...` — PASS.
- `npm run type-check` — PASS.

## Remaining limitations

- The application still ships `std.mast.ac.ug` / `mast.ac.ug` as the default
  signup and staff domains (`SIGNUP_ALLOWED_DOMAINS` in
  `internal/handlers/auth.go`, `NEXT_PUBLIC_*` defaults in `apps/web/lib/config.ts`).
  `SRS.md`, `STACK.md`, `ADR.md` and `AGENTS.md` all state the real domains are
  `std.must.ac.ug` / `must.ac.ug`, so these defaults look like the same typo.
  They were left untouched here to keep the change scoped; a deployment should
  set `SIGNUP_ALLOWED_DOMAINS` and the `NEXT_PUBLIC_*_EMAIL_DOMAIN` variables.
- `docs/MVP_NOTES.md` and several `README.md` examples still show
  `std.mast.ac.ug`.
- A student account created before the two-role change can still carry the
  legacy `USER` role. Both `USER` and `STUDENT_VOTER` have empty grant lists, so
  this does not block voting; it is a data tidy-up, not a defect.
