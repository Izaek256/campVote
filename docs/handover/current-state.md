# Current System State & Handover — campVote

## 1. Project Context
- **Project Name:** campVote (Mountains of the Moon University - MMU Campus E-Voting System)
- **Repository:** `https://github.com/Izaek256/campVote.git`
- **Current Branch:** `feat/issue-05-election-lifecycle` (from the ISSUE-04 head, which is merged into `develop` via PR [#8](https://github.com/Izaek256/campVote/pull/8)); not yet merged.
- **Current Status:** Monorepo established; backend compiles; campus voting domain models and privacy air-gapping in place; ISSUE-02 authentication hardening, ISSUE-03 server-side RBAC and ISSUE-04 per-election eligibility are merged into `develop`; ISSUE-05 election lifecycle, schedule validation, server-clock opening/closing, scheduler recovery and configuration freeze are implemented and tested on the task branch. TypeScript types are synced for ISSUE-03; re-run `grit sync` after the model changes.
- **Not yet built:** candidate/position/ballot CRUD endpoints, ballot versioning and tallying (ISSUE-06/08), results publication UI, and the voter/admin UI flows.

## 2. Architecture & Tech Stack
- **Framework:** Grit Framework v3.411.17 (`grit`)
- **Backend:** Go 1.26.4 + Gin + GORM (`apps/api`)
- **Frontends:** Next.js (TypeScript) with separate `apps/web` (voter portal) and `apps/admin` (election CMS)
- **Shared Types:** TypeScript interfaces and Zod schemas (`packages/shared`) synchronized from Go structs via `grit sync`
- **Monorepo:** Turborepo + pnpm workspaces (`apps/*`, `packages/*`)
- **Database:** PostgreSQL (production / Docker Compose) + SQLite pure-Go (`github.com/glebarez/sqlite`) for instant zero-dependency testing

## 3. Environment & Tooling Verified
- OS: Windows 11 AMD64
- Go: `go1.26.4 windows/amd64`
- Node.js: `v24.11.0`
- pnpm: `10.33.4` (in project) / `12.15.0`
- Grit CLI: `v3.411.17` (`C:\Users\Izaek\.grit\bin\grit.exe`)
- Local `CGO_ENABLED=0` and no C compiler, so tests using the cgo SQLite driver cannot run here (see §5).

## 4. Key Documentation Links
- Requirements: [`SRS.md`](../../SRS.md)
- Technology: [`STACK.md`](../../STACK.md)
- Architecture Decisions: [`ADR.md`](../../ADR.md)
- Governance: [`AGENTS.md`](../../AGENTS.md) & [`CONTRIBUTING.md`](../../CONTRIBUTING.md)
- Requirements Tracker: [`docs/progress/requirements-tracker.md`](../progress/requirements-tracker.md)
- Feature Specification: [`docs/features/campus-voting-domain.md`](../features/campus-voting-domain.md)
- Authentication record: [`docs/auth.md`](../auth.md)
- RBAC record: [`docs/rbac.md`](../rbac.md)
- Eligibility record: [`docs/eligibility.md`](../eligibility.md)
- Lifecycle record: [`docs/election-lifecycle.md`](../election-lifecycle.md)
- Progress Log: [`docs/progress/development-log.md`](../progress/development-log.md)

## 5. Verified Checks & Results
1. `go build ./...` — **PASS** (ISSUE-03, ISSUE-04 and ISSUE-05).
2. `go test ./internal/routes ./internal/policy ./internal/models ./internal/database ./internal/authz ./internal/handlers ./internal/eligibility ./internal/lifecycle ./internal/jobs -count=1` — **PASS** (9 packages, ISSUE-05).
3. `go test ./internal/handlers -count=1` — **PASS** (ISSUE-05; includes ISSUE-03/04 handler tests).
4. `go test ./internal/lifecycle -count=1` — **PASS** (transition matrix, boundary clock, scheduler recovery, validation, DST/timezone preparation).
5. `go test -v ./internal/models` — **PASS** (`TestBallotPrivacyDecoupling`, `TestElectionLifecycle`, `TestDuplicateVotingConstraint`, flag persistence).
6. `go test -v ./internal/database` — **PASS** (`TestSeedElections`, demo MMU data and idempotency).
7. `go test ./...` — **FAIL only in `internal/saga` and `internal/services`**, all with the pre-existing `CGO_ENABLED=0` go-sqlite3 stub. No failure is related to ISSUE-02, ISSUE-03, ISSUE-04 or ISSUE-05.
8. `grit sync` — **PASS** before ISSUE-03 (51 models); re-run after the ISSUE-04/05 model changes.

## 6. Immediate Next Steps
1. Commit ISSUE-05 and open a PR into `develop` (owner approval required; do not self-merge).
2. Run the suite against PostgreSQL (`GRIT_TEST_DATABASE_URL`) before release, especially concurrent duplicate submissions, freeze/import races and lifecycle conditional updates.
3. Build the remaining candidate/position endpoints. Each new route must add a row to `apps/api/internal/policy/routes.go`; `TestEveryRouteHasAPolicy` fails otherwise. Candidate/position writes must call the lifecycle freeze rule.
4. ISSUE-06/08: ballot versioning, confirmation receipts, tallying, results publication, then the voter/admin UI flows.
5. Run `grit sync` after the model changes.
