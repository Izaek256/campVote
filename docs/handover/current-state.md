# Current System State & Handover — campVote

## 1. Project Context
- **Project Name:** campVote (Mountains of the Moon University - MMU Campus E-Voting System)
- **Repository:** `https://github.com/Izaek256/campVote.git`
- **Current Branch:** `feat/issue-04-eligibility-verification` (from `feat/issue-03-server-side-rbac`), not yet committed or merged; ISSUE-03 PR [#7](https://github.com/Izaek256/campVote/pull/7) into `develop` is still awaiting review.
- **Current Status:** Monorepo established; backend compiles; campus voting domain models and privacy air-gapping in place; ISSUE-02 authentication hardening is merged into `develop`; ISSUE-03 server-side RBAC is implemented, tested and awaiting review; ISSUE-04 per-election eligibility, roster import, freeze/revocation, audit and the minimal voter ballot gate are implemented and tested. TypeScript types are synced for ISSUE-03; re-run `grit sync` after the ISSUE-04 model changes.
- **Not yet built:** election/candidate CRUD endpoints, ballot versioning and tallying (ISSUE-06/08), results publication UI, and the voter/admin UI flows.

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
- Progress Log: [`docs/progress/development-log.md`](../progress/development-log.md)

## 5. Verified Checks & Results
1. `go build ./...` — **PASS** (ISSUE-03 and ISSUE-04).
2. `go test ./internal/routes ./internal/policy ./internal/models ./internal/database ./internal/authz ./internal/handlers ./internal/eligibility -count=1` — **PASS** (7 packages, ISSUE-04).
3. `go test ./internal/eligibility ./internal/handlers -run 'TestEligibility|TestDomain|TestClock|TestFreeze|TestImport|TestReport|TestVoterCannot|TestRosterImport|TestUnverified|TestMultiChoice' -count=1` — **PASS** (ISSUE-04).
4. `go test -v ./internal/models` — **PASS** (`TestBallotPrivacyDecoupling`, `TestElectionLifecycle`, `TestDuplicateVotingConstraint`, flag persistence).
5. `go test -v ./internal/database` — **PASS** (`TestSeedElections`, demo MMU data and idempotency).
6. `go test ./...` — **FAIL only in `internal/saga` and `internal/services`**, all with the pre-existing `CGO_ENABLED=0` go-sqlite3 stub. No failure is related to ISSUE-02, ISSUE-03 or ISSUE-04.
7. `grit sync` — **PASS** before ISSUE-03 (51 models); re-run after the ISSUE-04 model changes.

## 6. Immediate Next Steps
1. Review and merge PR [#7](https://github.com/Izaek256/campVote/pull/7) into `develop` (owner approval required; do not self-merge). ISSUE-04 branches from ISSUE-03, so merge ISSUE-03 first.
2. Commit ISSUE-04 and open a PR into `develop`; do not self-merge.
3. Run the suite against PostgreSQL (`GRIT_TEST_DATABASE_URL`) before release, especially concurrent duplicate submissions and freeze/import races.
4. Build the remaining election/candidate endpoints. Each new route must add a row to `apps/api/internal/policy/routes.go`; `TestEveryRouteHasAPolicy` fails otherwise.
5. ISSUE-06/08: ballot versioning, confirmation receipts, tallying, results publication, then the voter/admin UI flows.
6. Run `grit sync` after the ISSUE-04 model changes.
