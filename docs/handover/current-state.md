# Current System State & Handover — campVote

## 1. Project Context
- **Project Name:** campVote (Mountains of the Moon University - MMU Campus E-Voting System)
- **Repository:** `https://github.com/Izaek256/campVote.git`
- **Current Branch:** `feat/issue-03-server-side-rbac` (from `origin/develop` at `5f0680f`), PR [#7](https://github.com/Izaek256/campVote/pull/7) into `develop`.
- **Current Status:** Monorepo established; backend compiles; campus voting domain models and privacy air-gapping in place; ISSUE-02 authentication hardening is merged into `develop`; ISSUE-03 server-side RBAC is implemented, tested and awaiting review. TypeScript types are synced.
- **Not yet built:** election/candidate/roster/ballot CRUD endpoints, voting endpoints, eligibility enforcement (ISSUE-04), and the voter/admin UI flows.

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
- Progress Log: [`docs/progress/development-log.md`](../progress/development-log.md)

## 5. Verified Checks & Results
1. `go build ./...` — **PASS** (ISSUE-03).
2. `go test ./internal/policy ./internal/routes ./internal/handlers ./internal/authz ./internal/models -count=1` — **PASS** (5 packages, ISSUE-03).
3. `go test -v ./internal/models` — **PASS** (`TestBallotPrivacyDecoupling`, `TestElectionLifecycle`, `TestDuplicateVotingConstraint`, flag persistence).
4. `go test -v ./internal/database` — **PASS** (`TestSeedElections`, demo MMU data and idempotency).
5. `go test ./...` — **FAIL only in `internal/saga` and `internal/services`**, with the pre-existing `CGO_ENABLED=0` SQLite stub. No failure is related to ISSUE-02 or ISSUE-03.
6. `grit sync` — **PASS** (51 models synchronized before ISSUE-03; re-run after model changes).

## 6. Immediate Next Steps
1. Review and merge PR [#7](https://github.com/Izaek256/campVote/pull/7) into `develop` (owner approval required; do not self-merge).
2. ISSUE-04 eligibility: enforce roster membership per election at ballot access and submission. Route permissions (`rosters.*`) and per-election scoping already exist.
3. Build the remaining election/candidate/roster/ballot endpoints. Each new route must add a row to `apps/api/internal/policy/routes.go`; `TestEveryRouteHasAPolicy` fails otherwise.
4. Build voting endpoints with transactional duplicate prevention and blind confirmation tokens, then wire results/turnout to the existing visibility policy.
5. Run `grit sync` after model changes and add the voter/admin UI flows.
