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

---

## [2026-10-28] ISSUE-05: Election Lifecycle, Schedule and Configuration Rules

**Branch:** `feat/issue-05-election-lifecycle` (from the ISSUE-04 head, which is merged into `develop` via PR #8), not yet merged.
**Goal:** one server-owned state machine for elections; voting open only on server time; no open with incomplete configuration; frozen configuration after opening with one audited exception; `[starts_at, ends_at)` boundaries.

### Activities Completed
1. **Lifecycle package (`apps/api/internal/lifecycle/`):**
   - `lifecycle.go` — the full ten-state SRS transition matrix, `ValidateBasics`/`ValidateConfiguration`/`ReadyToOpen`, `Ensure` (lazy reconcile), `SyncAll` (scheduler), `Apply` (validated transition plus audit in one transaction), and the display-only `Phase`.
   - `lifecycle_test.go` — every ordered state pair, boundary instants, missed scheduler tick, restart reconcile, invalid configuration fail-closed, validation fields, reason requirements.
2. **Handlers and routes (`apps/api/internal/handlers/election_lifecycle.go`):**
   - `POST /api/v1/elections`, `GET/PUT /api/v1/elections/:id`, `POST /api/v1/elections/:id/transitions`, `GET /api/v1/elections/:id/lifecycle`.
   - After `OPEN`, all configuration edits are rejected except a later `ends_at` while `OPEN`, which needs `elections.publish`, a reason, a valid future window, and an `ELECTION_EXTENDED` audit row.
   - `SCHEDULED -> OPEN` and `-> FINALIZED` are refused by the transition endpoint; they are reached by the server clock and the existing certification endpoint.
3. **Server clock integration:** `lifecycle.Ensure` runs on ballot view/start/submit and eligibility gate/freeze/status, so a missed scheduler tick cannot extend voting or let an ended election accept a ballot. A spoofed `Date` header is ignored.
4. **Scheduler:** `elections:lifecycle` every minute calls `SyncAll`; failure to open on invalid configuration is logged and leaves the election `SCHEDULED` (fail-closed).
5. **Policy and audit:** added `elections.publish`; new audit actions `ELECTION_REVIEW_STARTED`, `ELECTION_ARCHIVED`, `ELECTION_EXTENDED`; automatic transitions record `actor_id = "system"`.
6. **Timezones:** RFC 3339 with an explicit offset required; timestamps stored UTC; responses carry `display_timezone: "Africa/Kampala"`; a US DST window is tested.

### Checks Run
- `gofmt -l` on all changed Go files — clean.
- `go build ./...` — PASS.
- `go test ./internal/lifecycle ./internal/eligibility ./internal/policy ./internal/routes ./internal/models ./internal/cron ./internal/jobs -count=1` — PASS.
- `go test ./internal/handlers -count=1` — PASS.

### Documentation
- Created `docs/election-lifecycle.md` (states, boundary semantics, validation, freeze/exception, API, audit, tests, limitations).
- Updated `docs/README.md`, `docs/traceability.md`, `docs/progress/requirements-tracker.md` and `docs/handover/current-state.md`.

### Next Steps
1. Commit this branch and open a PR into `develop` (do not merge without owner approval).
2. Run the suite against PostgreSQL (`GRIT_TEST_DATABASE_URL`), especially the conditional status update and the optimistic version update.
3. ISSUE-06/08: position, candidate and roster edit endpoints must call the same freeze rule; ballot versioning, tallies and results publication remain.
4. No UI consumes the lifecycle endpoints yet.

---

## [2026-10-30] ISSUE-08: Atomic, Idempotent Vote Submission and the Voter Flow

**Branch:** `feat/issue-08-atomic-vote-submission` (from `develop` at `2b26360`), not yet merged.
**Goal:** a voter votes exactly once per election; the submission is one transaction bound to the ballot version the voter saw; a lost response can be recovered from without risking a second ballot.

### Activities Completed
1. **Models (`apps/api/internal/models/`):**
   - `VoterParticipation.IdempotencyKey` plus two unique indexes: `(election_id, user_id)` and `(election_id, idempotency_key)`; random key generated in `BeforeCreate` when the client sends none.
   - `AnonymousBallot.BallotVersionID` so a ballot is always interpretable against one published definition.
   - `ElectionAuditLog` action `VOTE_CAST`.
2. **Submission (`apps/api/internal/handlers/ballot.go`):**
   - `Submit` validates the whole payload against the current `BallotVersion` before the transaction, then re-checks eligibility, open status and the single-participation rule inside it, and writes participation + anonymous ballot + audit row together or not at all.
   - Idempotent replay: the same `idempotency_key` returns the original outcome and confirmation token with `200`; a new key after a ballot exists returns `409`; a stale `ballot_version_id` returns `409` and records nothing.
   - Closing-time race rule: acceptance is decided at commit time inside the transaction by the server clock; a request that started before `ends_at` but commits after it is refused.
   - Centralised `validateSelectionsAgainst` shared by the published-version path and the legacy live-definition path.
   - `VoteStatus` returns `{voted, recorded_at (15-minute bucket), phase, server_time}` — participation only.
3. **Routes:** `GET /api/v1/elections/:id/my-vote-status`; policy entry updated.
4. **Election list:** `my_vote_status` per election so `/elections` shows `Vote` vs `View status` without N+1 requests.
5. **Web:**
   - `lib/api.ts` `submitBallot(id, selections, {ballotVersionId, idempotencyKey})` and `voteStatus(id)`; `lib/types.ts` `VoteStatus`, ballot-version fields.
   - `/elections/[id]/ballot` rewritten: inline review step, confirm checkbox, in-memory idempotency key reused on retry, unknown-outcome banner with **Check status**, stale-version reload banner, already-voted banner, session-expired banner.
   - New `/elections/[id]/status`: voted / no vote / closed / session-expired, no re-vote path.
   - `/elections` cards and `/elections/[id]` use `my_vote_status`.
   - `VoteRecordedNotice` rewritten in the app's CSS vocabulary (it was using Tailwind classes that do not exist in this project).
6. **Design reference committed:** `docs/ui-kit/campVote-UI-Kit.html` (A0).
7. **Baseline fixes carried on this branch (not ISSUE-08 scope, but required to build/lint):**
   - `candidate.go` `Update()` declared an unused `userID` — the API did not compile on `develop`.
   - `apps/web/tsconfig.json` mapped `@campvote/shared/*` and `@campvote/web/*`; ISSSE-06/07 files imported them and the type check failed on `develop`.
   - `packages/shared/types/ballot_version.ts` rewritten as plain interfaces (the file imported `zod`, which is not a dependency of any workspace package).
   - Repo formatted with Biome and the ISSUE-06/07 staff pages converted to `useCallback` fetchers so lint passes; the golangci-lint CI step pinned to v2 (the workflow used a v1 binary against a v2 config, so the backend job failed on `develop`).

### Checks Run
- `go test ./internal/handlers/ -run 'Vote|Ballot|Submission' -count=1` — PASS (idempotency, parallel submit, rejected ballot writes nothing, closing-time race, vote status).
- `pnpm --filter @campvote/web type-check`, `lint`, `test`, `build` — PASS (build emits the new `/elections/[id]/status` route).
- `pnpm exec biome check .` — PASS.

### Documentation
- Created `docs/vote-submission.md`.
- Added the ISSUE-08 view table to `docs/ui-views.md`.
- Updated `docs/README.md` (index + new UI section).

### Deferred / Blocked
- Browser (Playwright) evidence: no Playwright suite exists and no Docker/browser environment was available locally. The API tests above are the evidence for the same rules; ISSUE-11 owns the browser pass.
- No fault-injection test forcing a mid-transaction failure; the transaction and the DB constraints are the guarantee, and the duplicate path is covered by the constraint test.
