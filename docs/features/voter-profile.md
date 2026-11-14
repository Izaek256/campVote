# Voter profile and residence areas (issue #37)

**Status:** implemented on `feature/37-planned-features`.
**Scope:** the campus profile a student completes before voting, the
administrator-maintained residence list, and the server-side gate.

## Profile

A voter supplies three fields once, on first sign-in:

| Field | Rule | Who may change it later |
|---|---|---|
| `age` | whole number 16–100 | nobody |
| `sex` | `MALE` or `FEMALE` | nobody |
| `residence_area_id` | an **active** row from `residence_areas` | an administrator |

The rules exist in two places. `PUT /api/v1/profile` validates them and returns
`400` with a readable message, and the `users` table carries `CHECK` constraints
for age bounds and the two sex values, so a write that bypasses the handler is
also refused.

The profile is part of `users`, not of a ballot or a participation row. It is
never joined to a choice.

### Server-side block

`eligibility.CheckElection` — the gate used by the ballot view, start and submit
paths — refuses an account whose profile is incomplete. `CheckRoster`, used by
candidate applications before polls open, deliberately does **not** require it.
A hidden submit button is therefore never the only control.

### The pop-up

`apps/web/components/profile-gate.tsx` is mounted once in the root layout. It
opens a native, blocking modal for a signed-in student with missing fields and
asks only for what is missing. Two behaviours worth knowing:

- The modal cannot be dismissed with Escape or a close button while it can be
  completed, because the ballot is refused without it.
- If the residence list is empty there is nothing the student can complete, so it
  offers **Later** instead of trapping them. The ballot stays refused until an
  administrator adds areas and the student picks one.

A same-page reload is not needed: saving calls `refresh()` on the auth provider,
which re-reads `/auth/me`.

## Residence areas

`models.ResidenceArea` is a small administrator-managed table. It is **not
seeded**: the demo seed leaves it empty on purpose, because inventing halls and
hostels in a voting system is worse than an honest empty state.

- `GET /api/v1/residence-areas` — signed-in users; active rows only, name order.
- `POST /api/v1/residence-areas` — `users.edit`; names are 2–120 characters and
  unique.
- `DELETE /api/v1/residence-areas/:id` — `users.edit`; deactivates the area and
  clears `users.residence_area_id` for every profile pointing at it **in one
  transaction**, so no profile keeps a dangling reference. The row stays for
  history and the action is written to the activity log.
- `PUT /api/v1/residence-assignments` — `users.edit`; names a student by email
  and sets or clears their residence. This is the only profile field an
  administrator edits; age and sex remain as the student supplied them.

The UI is `/staff/areas` (global administrators only), which lists areas, adds
one, deactivates one behind a confirmation, and corrects a student's residence.

## Tests

- `apps/api/internal/eligibility/eligibility_test.go` —
  `TestVotingNeedsACompleteProfile` (gate refuses, `CheckRoster` does not).
- `apps/api/internal/handlers/residence_test.go` — one-time profile rules,
  out-of-range and unknown values, deactivation clearing profiles, and the
  administrator correction touching only the residence.
- `apps/api/internal/models/voting_test.go` and the migration tests still pass;
  the new constraints arrive through `AutoMigrate`.

## Known limits

- There is no per-faculty or per-hall reporting on the profile fields; only the
  voting gate uses them.
- A student cannot self-correct a wrong residence by design (issue #37 says
  administrators correct residence only). If that proves too strict in use, the
  change is a rule in `campusProfileUpdates`, not a schema change.
