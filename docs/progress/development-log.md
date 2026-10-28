# Development Progress Log — campVote

Chronological record of completed work, verified test results, architecture decisions, and next actions.

---

## [2026-10-27] Phase 1: Initial Repository Audit, Toolchain Verification, and Foundation

### Activities Completed
1. **Repository & Workspace Inspection:**
   - Evaluated git state: repo initialized on `main`, zero commits, remote `origin` at `https://github.com/Izaek256/campVote.git`.
   - Inspected untracked kickstart files: `SRS.md`, `STACK.md`, `ADR.md`, `AGENTS.md`, `CONTRIBUTING.md`, `README.md`, `START_HERE.md`, `compose.dev.yaml`, `.env.example`, `.gitignore`, `.github/`, `docs/`.
   - Verified kickstart archive in `C:\Users\Izaek\Downloads\campVote_development_kickstart\campVote_kickstart`; confirmed complete parity with repository contents.

2. **Environment & Toolchain Audit:**
   - Windows 11 AMD64 host.
   - Go: `go1.26.4 windows/amd64`
   - Node.js: `v24.11.0`
   - npm: `11.6.4`
   - Installed official **Grit Framework CLI** `v3.411.17` via official Windows installer to `C:\Users\Izaek\.grit\bin\grit.exe`.
   - Installed **pnpm** `12.15.0` globally via npm to `C:\Users\Izaek\AppData\Roaming\npm\pnpm.cmd`.
   - Verified that Docker is currently not in the local PATH; confirmed that Grit's native zero-dependency SQLite and in-memory engine (`DB_PROVIDER=sqlite` / `DB_PROVIDER=memory`) provides reproducible local development and CI testing capabilities, with PostgreSQL configured for Docker Compose and production environments.

3. **Safe Cleanup & Hygiene:**
   - Audited all root files. No files identified as obsolete or disposable.
   - Updated `.gitignore` to exclude `.kilo/` agent harness files and worktrees.
   - Documented cleanup details in `docs/maintenance/repository-cleanup.md`.

4. **Scaffold Exploration in Temporary Sandbox:**
   - Generated a test Grit Triple project (`grit new scaffold-test --triple --next --db=postgres`) in `$env:TEMP\scaffold-test`.
   - Validated monorepo structure: `apps/api` (Go Gin/GORM), `apps/web` (Next.js voter portal), `apps/admin` (Next.js CMS), `packages/shared`, `packages/upload`.
   - Inspected Go backend internal packages: `auth`, `rbac`, `db`, `models`, `settings`, `testkit`.

5. **Requirements Tracking & Documentation Baseline:**
   - Created `docs/progress/requirements-tracker.md` mapping SRS functional areas to implementation status.
   - Updated setup documentation with verified versions and commands.

### Next Steps
1. Commit the initial baseline documentation and repository configuration to `main`.
2. Branch `develop` from `main`.
3. Create task branch `feature/01-grit-triple-monorepo` from `develop`.
4. Integrate the verified Grit Triple monorepo architecture (`apps/api`, `apps/web`, `apps/admin`, `packages/shared`) preserving our custom kickstart documentation and configuration.
5. Configure database provider with dual-mode support (SQLite for instant local testing, PostgreSQL for Compose/production).

---

## [2026-10-27] Phase 2: Monorepo Scaffold, Toolchain Pinning, and Clean Compilation

### Activities Completed
1. **Git Workflow Execution:**
   - Committed initial kickstart baseline to `main`: commit `608453b`.
   - Created `develop` branch from `main`.
   - Created task branch `feature/01-grit-triple-monorepo` from `develop`.
2. **Grit Monorepo Integration:**
   - Merged standard Grit triple-app structure (`apps/api`, `apps/web`, `apps/admin`, `packages/shared`, `packages/upload`, `tests/`, `package.json`, `pnpm-workspace.yaml`, `turbo.json`, `biome.jsonc`).
   - Renamed Go module across all Go source files to `campvote/apps/api`.
   - Pinned `go 1.26.4` in `apps/api/go.mod` matching the local Windows host compiler.
   - Removed optional CGo vips files (`internal/media/image_vips.go`, `internal/media/transform_vips.go`) to allow pure-Go compilation without external headers.
   - Resolved `undefined: table` in `apps/api/internal/access/` by creating `registry.go`.
   - Verified that `apps/api/cmd/server` and `apps/api/cmd/migrate` compile with zero errors.

---

## [2026-10-27] Phase 3: Campus Voting Domain Models, Seeder, and Automated Tests

### Activities Completed
1. **Implemented Campus Voting Domain Models (`apps/api/internal/models/`):**
   - `Election`: Multi-category support (Guild, Faculty, Department, Referendum), server-enforced schedule (UTC stored, displayed in EAT), lifecycle states, and result visibility policy.
   - `Position`: Single/multi-choice, referendum, seats, blank choice rules.
   - `Candidate`: Manifestos, bios, approval states.
   - `ElectionRoster`: Verified institutional eligibility list with composite unique constraints on `(election_id, email)` and `(election_id, student_number)`.
   - `VoterParticipation`: Isolated record of participation with random `ConfirmationToken` receipt; unique constraint `(election_id, user_id)` guarantees strictly one vote per voter.
   - `AnonymousBallot` & `BallotSelection`: Strictly decoupled ballot container with zero foreign keys or user references.
   - `ElectionTally`: Results snapshot with named denominators (`ValidVotesTotal`, `TurnoutPercentage`).
   - `ElectionAuditLog`: Tamper-evident administrative audit log.
   - Registered all 9 models in `user.go` `Models()` registry.
2. **Automated Test Battery (`apps/api/internal/models/voting_test.go`):**
   - `TestBallotPrivacyDecoupling`: Enforces FR-PRIV-01/02/03 via Go reflection; asserts that `AnonymousBallot` and `BallotSelection` have zero voter identification fields, and `VoterParticipation` has zero choice fields.
   - `TestElectionLifecycle`: Tests schedule checks and state transitions.
   - `TestDuplicateVotingConstraint`: Tests that database-level unique constraints reject duplicate participation attempts.
   - Ran `go test -v ./internal/models` — ALL 7 TESTS PASSED.
3. **Database Seeder (`apps/api/internal/database/elections_seeder.go`):**
   - Seeded MMU sample accounts: `commissioner@must.ac.ug`, `ivan.mugisha@std.must.ac.ug`, `doreen.kemigisa@std.must.ac.ug`.
   - Seeded Guild Representative Council election (OPEN) with Guild President (3 candidates with full manifestos), Vice Guild President (2 candidates), and Constitutional Amendment Referendum (YES/NO).
   - Seeded Faculty of Science & Technology election (SCHEDULED).
   - Seeded sample student roster records.
   - Added `elections_seeder_test.go` verifying idempotent execution — PASSED.
4. **TypeScript & Zod Synchronization (`grit sync`):**
   - Synced 51 models to TypeScript interfaces and Zod schemas in `packages/shared/types/` and `packages/shared/schemas/`.
5. **Documentation:**
   - Created `docs/features/campus-voting-domain.md`.
   - Updated `docs/progress/requirements-tracker.md`.

---

## [2026-10-28] ISSUE-03: Server-Side RBAC for Campus Elections

**Branch:** `feat/issue-03-server-side-rbac` (from `origin/develop` at `5f0680f`), not yet committed or merged.
**Goal:** every API route has an explicit policy; SRS roles, per-election scoping, separation of duties, results visibility and certification are enforced server-side.

### Activities Completed
1. **Route policy table (`apps/api/internal/policy/`):**
   - `policy.go` — global `Declared()` default-deny and `Guard(db)` enforcement for public/authenticated/permission/role requirements.
   - `routes.go` — explicit method + full-path table for all 181 non-library routes (30 public).
   - `election.go` — per-election authority, results visibility, separation of duties and certification rules.
2. **Roles and permissions:**
   - Added SRS constants (`CHIEF_ELECTION_ADMIN`, `ELECTION_OFFICIAL`, `STUDENT_VOTER`, `STAFF_VOTER`, `CANDIDATE`, `ELECTION_AUDITOR`) and `models.ElectionRole` with a unique `(election_id, user_id, role)` index.
   - Expanded `DefaultRoles` from 3 to 9 seeded roles; added `elections.*`, `candidates.*`, `rosters.*`, `ballots.*`, `results.view`, `results.view_live` to the permission catalog.
3. **Election endpoints (`handlers/election_rbac.go`):**
   - `GET /api/v1/elections/:id/roles`
   - `PUT /api/v1/elections/:id/roles/:userId`
   - `DELETE /api/v1/elections/:id/roles/:userId/:role`
   - `GET /api/v1/elections/:id/results`
   - `POST /api/v1/elections/:id/certify`
   - Each resolves the election through `policy.AuthorityFor`; cross-election and unknown ids both return 404.
4. **Audit:** route and object denials write `authz.denied`; role changes write `election.role_assigned` / `election.role_removed`; certification writes `RESULTS_CERTIFIED` in the same transaction.

### Checks Run
- `go build ./...` — PASS.
- `go test ./internal/policy ./internal/routes ./internal/handlers ./internal/authz ./internal/models -count=1` — PASS (5 packages).
- `go test ./...` — only `internal/saga` and `internal/services` fail, with the pre-existing `CGO_ENABLED=0` SQLite stub. No ISSUE-03 test fails.

### Documentation
- Created `docs/rbac.md` (role matrix, route inventory, 401/403/404 semantics, results policy, separation of duties, audit, limitations).
- Updated `docs/traceability.md` (FR-AUTH-06, FR-ELIG-05, FR-PRIV-06, FR-DASH-06/09/10, FR-AUDIT-04, FR-CERT-02, NFR-SEC-02/04).

### Next Steps
1. Commit the branch and open a PR into `develop` (do not merge without owner approval).
2. ISSUE-04 eligibility: enforce rosters per election; the role grants and scoping already exist.
3. Remaining election/candidate/roster/ballot endpoints must add a route policy; the inventory test enforces it.



---

## [2026-10-28] ISSUE-04: Per-Election Eligibility Verification

**Branch:** `feat/issue-04-eligibility-verification` (from `feat/issue-03-server-side-rbac`, because PR #7 is not yet merged into `develop`), not yet committed or merged.
**Goal:** eligibility decided server-side from authoritative roster evidence; CSV import with provenance; freeze/revocation; audit; no domain-only access; no self-service eligibility.

### Activities Completed
1. **Eligibility service (`apps/api/internal/eligibility/`):**
   - `service.go` — single `CheckElection` gate (open schedule, verified email, allowed domain, roster row, status, post-freeze evidence); `Freeze`, `SetStatus`, `BuildReport`.
   - `import.go` — CSV parse/validate/preview, duplicate-identity detection, atomic commit, import-batch provenance, freeze rules, audit.
2. **Models:** roster provenance/status fields, `EligibilityImportBatch`, election freeze fields, new audit action constants; all registered in `models.Models()`.
3. **Handlers/routes:**
   - `handlers/eligibility.go` — own decision, roster preview/import/status, freeze, officer report.
   - `handlers/ballot.go` — minimal voter view/start/submit with the eligibility gate on all three and transactional participation + anonymous ballot.
   - Routes and policies wired; `PermRosterImport` added.
4. **Identity protection:** eligibility now requires `users.email_verified_at`, so changing the profile email (which clears verification) cannot borrow an unregistered student's roster entry. Demo seeder and `testkit.User` mark their fixture accounts verified.
5. **Ballot validation fix:** multi-choice positions accept several distinct choices and reject the same choice twice; only multi-choice may take more than one answer; an abstention must be the only choice for its position.
6. **Seeder:** guild election keeps `RequireRoster: true`; the field is deprecated and no longer grants access by itself.

### Checks Run
- `go build ./...` — PASS.
- `go test ./internal/eligibility ./internal/handlers -run 'TestEligibility|TestDomain|TestClock|TestFreeze|TestImport|TestReport|TestVoterCannot|TestRosterImport|TestUnverified|TestMultiChoice' -count=1` — PASS.
- `go test ./internal/routes ./internal/policy ./internal/models ./internal/database ./internal/authz ./internal/handlers ./internal/eligibility -count=1` — PASS (7 packages).
- `go test ./... -count=1` — FAIL only in `internal/saga` and `internal/services`, all with the pre-existing `CGO_ENABLED=0` go-sqlite3 stub. No ISSUE-04 failure.

### Documentation
- Created `docs/eligibility.md` (evidence model, verified-email identity, CSV schema, provenance, freeze/revocation, audit, report, policy questions, limitations).
- Updated `docs/README.md`, `docs/traceability.md` (FR-ELIG-01..08, FR-VOTE-01/06/07/08/09) and `docs/handover/current-state.md`.

### Next Steps
1. Commit the branch and open a PR into `develop` (do not merge without owner approval).
2. Run the suite against PostgreSQL (`GRIT_TEST_DATABASE_URL`) before release, especially concurrent duplicate submissions.
3. ISSUE-06/08: ballot versioning, confirmation receipts, tallying and results publication; frontend eligibility/import screens.

---

## [2026-10-28] ISSUE-04 follow-up: six-digit codes, standing ballots, admin role control

**Branch:** `feat/issue-04-eligibility-verification`.
**Goal:** apply the maintainer's confirmed decisions to the ISSUE-04 work.

### Changes
1. **Email verification is now a six-digit code:** `GenerateVerificationCode` uses `crypto/rand`; the code is stored as an HMAC-SHA256 hash keyed with `JWT_SECRET` (a plain hash of a one-million-code space is reversible from a dump); TTL is 10 minutes; `POST /api/auth/verify-email` takes `{"code":"123456"}`; the mail template and preview show the code and expiry; development still returns `verify_code`, guarded on `APP_ENV=development`.
2. **Ballots stand after post-vote revocation:** confirmed as policy. `TestRevocationAfterVotingLeavesTheBallotStanding` proves revocation blocks further ballots but leaves the anonymous ballot, participation record and tally untouched.
3. **The system administrator can give and remove any role:** confirmed with `TestRoleAPI_AdminCanGrantAndRevokeAnyRole`, including granting `ADMIN` and clearing all roles with an empty `role_ids`.

### Checks Run
- `go build ./...` — PASS.
- `go test ./internal/routes ./internal/policy ./internal/models ./internal/database ./internal/authz ./internal/handlers ./internal/eligibility ./internal/mail -count=1` — PASS.
- `go test ./... -count=1` — FAIL only in `internal/saga` and `internal/services`, all with the pre-existing `CGO_ENABLED=0` go-sqlite3 stub. No failure in the changed packages.

### Documentation
- Updated `docs/auth.md`, `docs/rbac.md`, `docs/eligibility.md` and `docs/implementation/OPEN_QUESTIONS.md`.
