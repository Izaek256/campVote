# campVote MVP Notes

> Created: 2026-11-02 | Branch: feature/1-mvp-phase1-infra

This file records assumptions, decisions, and known gaps as work progresses through the MVP phases.

---

## Assumptions

1. **Email format**: Student emails match `^(\d{4})([a-z][a-z-]*)(\d{1,5})@<domain>$` — e.g. `2024BSE088@std.mast.ac.ug`. The domain per the prompt is `std.mast.ac.ug` (note: **mast** not must). All eligibility rules use this format.
2. **Academic year / study year**: Study year computed as `currentYear - enrolmentYear + 1`; academic year starts in August.
3. **Session secret**: The existing codebase derives the gorilla/sessions cookie key from `JWT_SECRET` via SHA-256. The old compose `SESSION_SECRET` variable was unused; compose now documents `JWT_SECRET` only.
4. **Storage in Docker**: `STORAGE_DRIVER=local`, `STORAGE_LOCAL_ROOT=/app/uploads`, named volume. Uploads persist across restarts but not across `docker compose down -v`.
5. **No real student data**: All demo accounts use `@std.mast.ac.ug` with synthetic names.
6. **`apps/admin` deletion**: The `apps/admin` folder is removed because its Dockerfile referenced a non-existent `apps/admin/public` directory, it was not in compose, and all staff functionality lives in `/staff` routes within `apps/web`.
7. **Automatic tally**: On close transition, `tally.Compute` is called and a snapshot persisted. Tally failure is logged but does not fail the transition; retried on next `Read`.
8. **`SIGNUP_ALLOWED_DOMAINS`**: Default is `std.mast.ac.ug`. Staff/admin accounts are created via bootstrap or role assignment.
9. **Election branding**: Replaced all MUST/MMU strings with neutral `Campus Electoral Commission` / `std.mast.ac.ug`.
10. **Redis**: Optional, and not started by compose. `docker-compose.yaml` contains one service — the API — so `REDIS_URL` comes from `.env` as written. Set it empty to switch cache, jobs and cron off deliberately; left out, the API falls back to `redis://localhost:6380`. Without Redis, the lazy `lifecycle.Ensure` path still opens/closes elections on request.
11. **Mail in Docker**: from `.env`. No Mailpit service, so no :8025 UI exists; `MAIL_MAILER=log` prints each message to `docker compose logs api` and saves an HTML copy under `MAIL_LOG_PATH` (`storage/mail`, on the `/app/storage` volume). Unset, development tries SMTP on `localhost:1025` and then falls back to the log.
12. **Web app and database not containerised**: run `pnpm dev:web` on the host against `http://localhost:8080`, and point `.env`'s `DATABASE_URL` at a PostgreSQL you already run — the API container shares the host's network, so `localhost` in that URL means the machine. `apps/web/Dockerfile` remains for deployments that want an image. See `docs/operations/docker-compose.md`.

---

## Phase 1: Infra + Cleanup

### Files Changed
- `apps/api/Dockerfile` — alpine runtime, CA certs, non-root user, entrypoint, all 4 binaries
- `apps/web/Dockerfile` — standalone output, pnpm, non-root
- `apps/web/next.config.ts` — added `output: "standalone"`
- `docker-compose.yaml` — rewritten: no profiles, adds redis + mailpit
- `.dockerignore` — added
- `.env.example` — updated to match actual env var names
- `compose.dev.yaml` — deleted (superseded)
- `apps/web/app/staff/elections/[id]/monitor/page.tsx` — fixed `params.id` type, `results.positions` -> `results.snapshot?.positions`
- `apps/admin/` — deleted
- `package.json` — removed admin scripts
- `turbo.json` — removed admin build task

### Commands Run
_(filled after each phase verification)_

---

## Phase 2: Eligibility by registration-number email

### Files Changed
- `apps/api/internal/eligibility/identity.go` — new: `ParseEmail`, `RuleAllows`
- `apps/api/internal/models/election.go` — added `AllowedDomain`, `AllowedYears`, `AllowedCourses`
- `apps/api/internal/eligibility/service.go` — rule-based check; no roster row = allowed
- `apps/api/internal/config/domains.go` — hardcoded MUST lists removed
- `apps/api/internal/handlers/election_lifecycle.go` — `List` filters by eligibility for students
- `apps/api/internal/handlers/auth.go` — `SignupInitiate` checks `SIGNUP_ALLOWED_DOMAINS`
- `apps/api/internal/database/elections_seeder.go` — uses new fields
- `apps/api/internal/eligibility/*_test.go` — rewritten

---

## Phase 3: Positions, Candidates, Vetting

### Files Changed
- `apps/api/internal/models/position.go` — added `Requirements string`
- `apps/api/internal/models/candidate.go` — added `Programme string`, `StudyYear int`
- `apps/api/internal/handlers/candidate.go` — auto-fills programme/year from email
- `apps/web/app/elections/[id]/candidate/apply/page.tsx` — DOB, passport photo, ID photo
- `apps/web/app/staff/elections/[id]/candidates/page.tsx` — photos, DOB, programme, year; reason required for reject

---

## Phase 4: Election Setup, Schedule, Temporary Admins

_(to be filled)_

---

## Phase 5: Voting, Live Participation, Automatic Results

_(to be filled)_

---

## Phase 6: UI Kit, Branding, Polish

_(to be filled)_

---

## Phase 7: Demo Data, Smoke Test, E2E Verification

_(to be filled)_

---

## Known Gaps / Out of Scope for MVP

- **Cryptographic E2E verifiability**: Not implemented; ballot secrecy enforced by DB constraint separation.
- **Formal certification audit trail**: `RESULTS_UNDER_REVIEW` / `FINALIZED` statuses exist but full workflow deferred.
- **SSO/SAML**: Grit scaffold included; not wired to campus systems.
- **Playwright E2E tests**: `scripts/smoke.sh` covers critical flows via curl; Playwright deferred.
- **Multi-winner highlight**: `seats > 1` positions work at data layer; winner UI limited to single-choice for MVP.
- **Real domain verification**: String suffix match against `SIGNUP_ALLOWED_DOMAINS`; no LDAP/HR query.
