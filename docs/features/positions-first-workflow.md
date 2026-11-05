# Positions-First Workflow (Phase 2)

**Branch:** `feature/mvp-phases-3-7`
**Status:** Implemented (backend + admin/student UI), tests passing
**Related docs:** [`eligibility.md`](../eligibility.md), [`rbac.md`](../rbac.md), [`implementation/ROADMAP.md`](../implementation/ROADMAP.md)

---

## Summary

The redesign establishes a **positions-first** order of operations. A position is
the durable thing (e.g. *Guild President*); an election is scheduled around the
positions that will be contested in it.

```
create position  →  collect applications  →  review & approve (bounded by max candidates)
                 →  link position to an election  →  election is scheduled and opened
```

A position therefore exists **before** any election does. `Position.ElectionID`
is nullable; an administrator links the position to its election once the
candidate list is settled, and the position's applications move with it.

There are exactly two roles: **ADMIN** and **STUDENT_VOTER**. An ADMIN can grant
ADMIN to any other account. Everything an administrator needs day-to-day is a
sidebar item, not something buried inside one election.

---

## Schema change

This repository **has no `migrations/` directory and no numbered SQL files**.
The schema is model-driven: GORM `AutoMigrate` builds and alters tables at
startup, and `internal/migrate/history.go` snapshots the schema before and
after, diffs it, and writes the run into `grit_migrations` /
`grit_migration_changes` so a rollback is computable. The requested "versioned
migration" is therefore satisfied in this repository's idiom: **the change lands
in the model plus the recorded grit migration diff**, not in a hand-written SQL
file.

`models.Position` changed:

| Field | Change |
|---|---|
| `ElectionID` | `string` → `*string` (nullable). A position can exist unlinked. |
| `Status` | New — `OPEN` (default) or `CLOSED`; controls whether applications are accepted. |
| `MaxCandidates` | New — cap on approved candidates; `0` means no limit. |
| `AllowedDomain`, `AllowedYears`, `AllowedCourses` | New — the position's own applicant eligibility rule. |

Helpers added: `models.Ptr[T]` (for optional pointer columns),
`Position.SetElection(id)` / `Position.ElectionRef()` / `Position.AcceptingApplications()`.

---

## Eligibility: one rule, two uses

`internal/eligibility` now exposes a shared `Rule` so an **election's voter
eligibility** and a **position's candidate eligibility** are evaluated by
exactly the same code:

```go
type Rule struct{ AllowedDomain, AllowedYears, AllowedCourses string }
func RuleForElection(*models.Election) Rule
func RuleForPosition(*models.Position) Rule
func (Rule) Allows(Identity) bool   // domain required; years/courses optional
```

The identity comes from the verified registration-number email via
`ParseEmail` (`2024bse088@std.mast.ac.ug` → year `2024`, course `bse`,
number `088`). It is derived on the server; the applicant never supplies it.

---

## API

### Admin (staff group; requires the named permission)

| Method | Path | Permission | Handler |
|---|---|---|---|
| GET | `/api/v1/positions` | `ballots.view` | `PositionHandler.ListAll` |
| POST | `/api/v1/positions` | `ballots.edit` | `PositionHandler.CreateStandalone` |
| PUT | `/api/v1/positions/:position_id` | `ballots.edit` | `PositionHandler.UpdateStandalone` |
| DELETE | `/api/v1/positions/:position_id` | `ballots.edit` | `PositionHandler.DeleteStandalone` |
| GET | `/api/v1/positions/:position_id/applications` | `ballots.view` | `PositionHandler.Applications` |
| POST | `/api/v1/positions/:position_id/applications/:candidate_id/approve` | `candidates.review` | `PositionHandler.ApproveApplication` |
| POST | `/api/v1/positions/:position_id/applications/:candidate_id/reject` | `candidates.review` | `PositionHandler.RejectApplication` |
| POST | `/api/v1/positions/:position_id/link` | `ballots.edit` | `PositionHandler.Link` |

`GET /positions` accepts `?election_id=`, `?unlinked=1` and `?status=` and
returns each position with `application_count`, `approved_count` and
`election_title`.

### Applicant (authenticated)

| Method | Path | Handler |
|---|---|---|
| GET | `/api/v1/positions/eligible` | `PositionHandler.Eligible` |
| POST | `/api/v1/positions/:position_id/apply` | `PositionHandler.Apply` |

`/positions/eligible` returns only positions that are **OPEN** and that the
caller's parsed identity satisfies, each flagged `applied`.

### Rules enforced on the server

- **Eligibility** — `Apply` re-checks the position's rule; an ineligible caller
  gets `403`. A staff/external address that does not parse gets an empty
  eligible list.
- **One live application per applicant per position** — a duplicate returns `409`.
- **Maximum candidates** — `ApproveApplication` counts already-approved siblings
  and returns `409` once the cap is reached. The cap is not a UI concern.
- **Linking** — `Link` refuses a `CLOSED` election (the three-stage lifecycle
  has no other blocking state), and in one transaction sets the position's
  election and moves its candidates' `election_id`, so tallying sees them under
  the election they contest.
- **Application details** — `Apply` requires a date of birth (valid `YYYY-MM-DD`),
  a sex of `MALE`/`FEMALE`, an uploaded passport photo and an uploaded partial
  transcript, and accepts religion, campus residence, contact details, slogan,
  bio and manifesto. `RegistrationNumber`, `Programme` and `StudyYear` are
  derived from the verified registration email, never from the request body.
- **Audit** — every position/application change writes `ElectionAuditLog` when
  the position is linked, and always `services.LogActivity`. Ballot choices are
  never recorded here.

### Admin application review

Applications are reviewed from a modal rather than a bare approve button.
`components/application-review-dialog.tsx` is a native `<dialog>` (focus
containment, Escape to close, focus return) that shows the applicant's passport
thumbnail, status chip, registration number, programme and year, and a
definition list of names, campus id, programme, year, date of birth, computed
age, sex, religion, residence and contact, plus links to the passport and
transcript documents, the slogan and the bio. The reviewer browses applications
with **Previous**/**Next** without closing the dialog and approves or rejects
from the footer; a rejection reason is required and approve takes an optional
note. The list rows in `/staff/positions` are clickable and carry a **Review**
button that opens the same dialog.

---

## UI

Admin navigation is role-aware (`components/app-shell.tsx`). ADMIN receives
sidebar quick actions **New election** and **New position**, and a
**Positions** item independent of any election. STUDENT_VOTER receives
**Elections**, **Run for a position**, **My candidacy** and **Account**.

| Route | Role | Purpose |
|---|---|---|
| `/staff/positions` | ADMIN | List/create/edit/delete positions, set the eligibility rule and max candidates, review applications in the review dialog, approve/reject, link to an election. Supports `?new=1` from the sidebar quick action. |
| `/positions` | STUDENT_VOTER | Browse only the positions the student is eligible for and apply with the full candidate profile (names, DOB and computed age, sex, religion, residence, slogan, bio, passport photo and partial transcript uploads); an applied position shows as *awaiting review*. |

---

## Checks run

| Check | Command | Result |
|---|---|---|
| Backend tests | `go test ./internal/handlers/... ./internal/lifecycle/... ./internal/policy/... ./internal/results/...` | pass |
| Frontend type-check | `npm run type-check` | pass |
| Frontend tests | `npm run test` | 12/12 pass |
| Frontend build | `npm run build` | pass (routes `/positions` and `/staff/positions` present) |

New backend test coverage: eligibility scoping of the eligible list, ineligible
apply rejected, duplicate apply rejected, max-candidate cap enforced
server-side, linking adopts applications, standalone creation being admin-only
with an invalid question type rejected, and the rich application payload
(date of birth, sex, passport photo and transcript) being required.

Biome lint also reports three pre-existing findings that this work does not
introduce and does not fix: `lint/style/useNodejsImportProtocol` in
`next.config.ts`, `lint/security/noDangerouslySetInnerHtml` in `app/layout.tsx`
(the pre-paint theme script) and `lint/a11y/noRedundantAlt` in
`app/staff/elections/[id]/candidates/page.tsx`. None block `type-check`, tests
or the build. In addition, Biome `format` reports CRLF noise on files checked
out on Windows.

---

## Known limitations / deferred to Phase 3

> Phase 3 followed this. The student-side items below are now done — see
> `docs/progress/development-log.md` (Phase 3 entry): after voting, an election
> leaves the Elections page and appears in the new `/results` tab. Ballot
> composition from approved candidates, the live admin dashboard and the
> on-close tally broadcast were already implemented before either phase.

- **Automatic ballot creation** for approved candidates is not yet wired to the
  ballot compiler. Approved candidates carry a `BallotNumber`, but the election's
  ballot is still composed separately.
- **Live results dashboard** and **broadcasting results to students on close**
  are not part of this phase.
- **Server-side filtering of the student elections list by eligibility** and
  **hiding an election after the student votes** remain as they were; only the
  positions views filter by eligibility today.
- `Position.ElectionID` is nullable, so existing election-scoped position routes
  (`/elections/:id/positions`) still work unchanged.

---

## Notes for the maintainer

- The pre-existing Biome `format` failure on `apps/web/lib/api.ts` is CRLF noise
  (the file had 293 CRLF lines at `HEAD`); it is not caused by this work and was
  left untouched to avoid a whole-file diff.
- An ADMIN holds the `*` grant, so it satisfies `ballots.edit` and
  `candidates.review` without any extra role wiring.
