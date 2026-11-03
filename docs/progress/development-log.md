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

---

## [2026-11-02] Phase 2: Positions-First Workflow, Position Eligibility and Max Candidates

**Branch:** `feature/mvp-phases-3-7`, not yet committed or merged.
**Goal:** make positions the durable, first-class object. An administrator creates a position, collects applications, reviews them against a max-candidate cap, links the position to an election, and only then is the election scheduled and opened. Roles stay at exactly two: `ADMIN` and `STUDENT_VOTER`.

### Activities Completed
1. **Schema (`apps/api/internal/models/`):**
   - `Position.ElectionID` changed from `string` to `*string` (nullable) so a position can exist unlinked; `Status` (`OPEN`/`CLOSED`), `MaxCandidates`, and the applicant eligibility rule (`AllowedDomain`, `AllowedYears`, `AllowedCourses`) added. Helpers `Ptr[T]`, `SetElection`, `ElectionRef`, `AcceptingApplications`.
   - This repository has **no `migrations/` directory**; the schema is model-driven. `AutoMigrate` applies the change and `internal/migrate/history.go` records the before/after diff in `grit_migrations` / `grit_migration_changes`. The requested versioned migration is satisfied in this idiom (documented in the feature doc).
2. **Shared eligibility rule (`apps/api/internal/eligibility/identity.go`):** one `Rule` type now decides both an election's voter eligibility and a position's candidate eligibility (`RuleForElection`, `RuleForPosition`, `Allows`), derived from the verified registration-number email. The applicant never supplies their identity.
3. **Handlers/routes (`apps/api/internal/handlers/position_workflow.go`):**
   - Admin (staff group): `GET/POST /positions`, `PUT/DELETE /positions/:position_id`, `GET /positions/:position_id/applications`, approve/reject, `POST /positions/:position_id/link`.
   - Applicant (authenticated): `GET /positions/eligible`, `POST /positions/:position_id/apply`.
   - Server-enforced rules: eligibility re-checked on apply (`403`), one live application per applicant per position (`409`), max-candidate cap enforced on approval (`409`), linking refuses `FINALIZED`/`ARCHIVED`/`CANCELLED` and moves candidates into the election in one transaction.
   - Route policy entries added to `internal/policy/routes.go` (`TestEveryRouteHasAPolicy` passes).
4. **Web (`apps/web`):**
   - `app/staff/positions/page.tsx` — admin positions page: list/create/edit/delete, eligibility rule + max candidates, application review with approve/reject reason, link panel, `?new=1` deep link.
   - `app/positions/page.tsx` — student "Run for a position" page: only eligible positions, apply form, "awaiting review" state.
   - `components/app-shell.tsx` — role-aware sidebar: ADMIN gets **New election** and **New position** quick actions plus a standalone **Positions** item; STUDENT_VOTER gets Elections / Run for a position / My candidacy / Account.
   - `lib/types.ts`, `lib/api.ts` extended with the position workflow types and calls.
5. **Tests (`apps/api/internal/handlers/position_workflow_test.go`):** eligibility scoping of the eligible list, ineligible apply rejected, duplicate apply rejected, max-candidate cap enforced server-side, linking adopts applications, standalone creation admin-only.

### Checks Run
- `go build ./...` — PASS.
- `go vet ./...` — PASS.
- `go test ./internal/handlers/... ./internal/routes/... ./internal/policy/... ./internal/eligibility/... ./internal/models/... -count=1` — PASS (handlers 24.273s); new workflow tests 5/5 PASS; `TestEveryRouteHasAPolicy` PASS.
- `npm run type-check` — PASS.
- `npx biome check app/staff/positions/page.tsx app/positions/page.tsx components/app-shell.tsx lib/types.ts` — PASS.
- `npm run test` — 12/12 PASS.
- `npm run build` — PASS (routes `/positions` and `/staff/positions` emitted).

### Documentation
- Created `docs/features/positions-first-workflow.md`.
- Updated `docs/README.md` (Section 4 UI index).

### Deferred (Phase 3)
- Automatic ballot creation for approved candidates is not yet wired to the ballot compiler.
- Live admin results dashboard and broadcasting results to students on close.
- Server-side filtering of the student elections list by eligibility and hiding an election after the student votes (only the positions views filter by eligibility today).

### Notes
- The pre-existing Biome `format` failure on `apps/web/lib/api.ts` is CRLF noise (293 CRLF lines at `HEAD`), not caused by this work; left untouched to avoid a whole-file diff.
- An ADMIN holds the `*` grant, so it satisfies `ballots.edit` and `candidates.review` with no extra role wiring.

---

## [2026-11-02] Phase 3: Student Results Tab and Post-Vote Election Handling

**Branch:** `feature/mvp-phases-3-7`, not yet committed or merged.
**Goal:** finish the student half of the confirmed workflow. After a student votes, that election leaves the Elections page and moves to a Results tab where it stays pending until the count is published. No backend change was required — the server already filters the elections list by eligibility and already tallies and broadcasts on close.

### What was already in place (verified, not rebuilt)
- **Eligibility filtering of the student list** — `ElectionLifecycleHandler.List` skips elections whose `eligibility.RuleAllows` rule the caller fails, and always shows elections the caller holds a role on.
- **Automatic ballot generation + admin confirmation** — `handlers.BallotContentFor` compiles the ballot from approved positions and candidates; `POST /elections/:id/ballot/publish` freezes it into an immutable, hashed `BallotVersion` and is driven from the admin **Ballot builder** page. `ApproveApplication` assigns `BallotNumber` automatically.
- **Live admin dashboard** — the admin **Monitor** page shows turnout and participation while open; the **Results workspace** reads provisional figures for holders of `results.view_live` (an ADMIN holds `*`).
- **Broadcast on close** — `lifecycle.OnClosed = results.OnClose` records the first snapshot when the election closes and `results.NotifySnapshot` publishes on `public-election-results.<election_id>`, which the voter results page already subscribes to.

### Changes made (frontend only)
1. **New page `apps/web/app/results/page.tsx`** — the student Results tab. Loads `elections.list()` (already eligibility-filtered), keeps only elections where `my_vote_status.voted`, and splits them into **Awaiting results** (not yet closed: "vote recorded" + closing time) and **Published results** (closed: a link to `/elections/:id/results`, which applies the election's publication policy).
2. **`apps/web/components/app-shell.tsx`** — added **Results** (`/results`) to the student sidebar between Elections and Run for a position.
3. **`apps/web/app/elections/page.tsx`** — elections the voter has already voted in are filtered out of the list and the vote-status button branch was removed (that flow now lives in Results). An "you have voted" empty state with a link to Results appears when every visible election is already voted, and the header gained a **My results** link.

### Checks Run
- `npm run type-check` — PASS.
- `npx biome check app/results/page.tsx app/elections/page.tsx components/app-shell.tsx` — PASS.
- `npm run test` — 12/12 PASS.
- `npm run build` — PASS; the route list now includes `/results`.
- No backend file changed in this phase, so no backend suite was re-run.

### Documentation
- Updated `docs/ui-views.md` (My Elections row; new My Results row).

### Deferred / Blocked
- The admin Monitor page deliberately hides candidate totals while voting is open; provisional figures are available to an ADMIN on the Results workspace (`results.view_live`). Showing live shares on the Monitor page itself would be a privacy-visible change and was left as-is pending an explicit decision.
- No automated UI test covers the new Results tab; the existing Vitest suite is limited to ballot and format helpers.

---

## [2026-11-02] Toast Notifications Replace Inline Banners

**Branch:** `feature/mvp-phases-3-7`, not yet committed or merged.
**Goal:** make confirmations and errors more visible. Requested by the maintainer as "errors and confirmation banners should be changed to Toasts for more visibility", with the scope explicitly widened (on being asked) to every banner in the app and an 8-second auto-dismiss for failures.

### What was wrong before
`app/globals.css` already carried `.toasts` / `.toast` / `.toast.bad` / `.toast.ok` / `.toast.out` and the `tin` / `tout` keyframes, but **no component ever rendered them** — dead template CSS. There was no `useToast`, no provider, and no `className="toasts"` anywhere.

### Changes made (frontend only)
1. **New `apps/web/components/toast.tsx`** — a `ToastProvider` plus a `useToast()` hook returning `success`, `error` and `warn`. Toasts auto-dismiss (4s for confirmations, 8s for failures), can be dismissed by hand, animate out over 250ms, and use `role="alert"` for failures and `role="status"` otherwise. All timers are cleared on unmount.
2. **`apps/web/app/layout.tsx`** — `ToastProvider` mounted inside `AuthProvider`, so `useToast()` works in any client component without prop drilling.
3. **`apps/web/app/globals.css`** — added the missing `.toast.warn` variant beside the existing rules.
4. **`apps/web/components/load-error.tsx`** (new) — see the regression below.
5. **~60 banner sites across the app converted to toasts.** Deliberately left inline, because they are not notifications: static instructional copy (privacy/ballot-secrecy notices), render-derived "why this is unavailable" panels (`banner hatch`, e.g. "Positions are locked", "Applications are closed", "Turnout is hidden to protect privacy"), and receipts the user must keep on screen (`VoteRecordedNotice`, the register page's development verification code).

### Regression found and fixed in the same change
Converting the banners also deleted the `if (error) { return <banner bad …> }` early returns. Because several pages guard their loading skeleton on the data itself (`if (!items)`, `{!results ? …}`), a **failed initial load left the loading skeleton on screen for ever** — the toast is gone after 8 seconds, so the page looked hung. An audit found **13 further pages** with this defect beyond the 4 fixed by hand.

Decision recorded with the maintainer: a failed page load gets a **persistent inline panel in addition to the toast**, not a toast alone. `components/load-error.tsx` implements that panel (message, `Try again`, and a back link or a caller-supplied action such as `Sign in again`), and it is now used by every page that loads its own data. The loading skeleton can no longer outlive a failed request.

Also fixed while converting: `app/staff/elections/[id]/results/page.tsx` had turned a *restricted results* message into a green success toast; the 403 case is now `toast.warn` and a genuine load failure is `toast.error`. On the maintainer's decision the recount summary stays toast-only (the per-position differences table below it is unchanged).

### Checks Run
- `npm run type-check` — PASS (exit 0).
- `npx biome lint` — 1 error, the pre-existing `noRedundantAlt` on `alt="Passport photo"` in `app/staff/elections/[id]/candidates/page.tsx` (untouched; verified as pre-existing with `git diff`). No new findings from this work.
- `npm run test` — 12/12 PASS.
- `npm run build` — PASS, all 28 routes emitted.
- No backend file changed, so no backend suite was re-run.

### Notes / Deferred
- `apps/web/app/login/page.tsx` and `apps/web/app/register/page.tsx` keep their `error` state because it also drives the invalid-input styling; each message is therefore produced twice (one `setError`, one `toast.error`). Behaviour is correct but the expression is duplicated — a tidy-up, not a defect.
- CRLF: many files in this repository are `i/lf w/crlf`, so Biome `format` reports whole-file noise on them. Only the three genuinely mis-formatted files were formatted; the rest were left alone rather than rewriting line endings.
- No automated test covers toasts; the Vitest suite is limited to ballot and format helpers. Verification is the type check, lint, build and the manual pass above.

---

## [2026-11-03] Domain-Default Bug Fix, Simplified Election Form, and App-Wide Theme Switcher

**Branch:** `feature/mvp-phases-3-7`, not yet committed or merged.
**Goal:** three maintainer-reported items in one pass — (1) a position linked to an election never appeared to a logged-in student, (2) the election creation form had "a lot of ambiguous steps" and should capture only the important details, (3) dark/light mode was missing from parts of the app.

### 1. Bug — a student saw no positions
**Symptom:** an administrator created a position and an election, linked them, but the student's **Run for a position** page showed an empty list.

**Root cause (three independent layers, all writing the same wrong domain):**
| Layer | Detail |
|---|---|
| Position create form | pre-filled `allowed_domain` with `std.mast.ac.ug`, so every new position was restricted |
| `ElectionLifecycleHandler.Create` | force-defaulted a blank `AllowedDomain` to `std.mast.ac.ug` |
| `models.Election.AllowedDomain` | carried the GORM tag `default:'std.mast.ac.ug'`, a **database** default applied on insert even when the Go value was `""` — the decisive cause |
The student's registration email is `…@std.must.ac.ug`, so `eligibility.Rule.Allows` rejected the `std.mast.ac.ug` mismatch and `GET /positions/eligible` filtered the position out. A blank domain means "any student with a parseable registration email".

**Fix:**
- `apps/web/app/staff/positions/page.tsx` — `emptyForm.allowed_domain` starts `""`; the eligibility section reads *"Leave everything blank to admit every student"* and the input placeholder is *"Leave blank for any student"*.
- `apps/api/internal/handlers/election_lifecycle.go` — removed the `if election.AllowedDomain == "" { … = "std.mast.ac.ug" }` force-default; `lifecycle.validDomains` already accepts an empty domain.
- `apps/api/internal/models/election.go` — removed the GORM `default:'std.mast.ac.ug'` tag from `AllowedDomain` (the position model never had one).
- Live data (local dev DB, via `psql`): blanked `allowed_domain` on the affected position/election rows and dropped the leftover Postgres column default (`ALTER TABLE elections ALTER COLUMN allowed_domain DROP DEFAULT`).

**Regression test:** `TestElectionCreateLeavesDomainOpenToAnyStudent` (`apps/api/internal/handlers/election_lifecycle_test.go`) asserts the stored election has an empty `allowed_domain` and that a voter on a different domain sees it. It fails against the previous model tag and passes after the fix.

### 2. Election creation simplified ("keep it simple")
The create form was reduced to **Election title**, **Voting opens (EAT)** and **Voting closes (EAT)** — the minimum needed to start. Removed from create (still editable later on the election page): description, category, organising unit, and results policy. Those now fall back to a safe server default, and new elections start as a private `DRAFT`.

### 3. Dark/light mode switcher, app-wide
The theme engine (`components/theme-toggle.tsx`, the CSS variables in `globals.css`, the `localStorage` key `"theme"`) already existed and the public/student header already rendered the toggle. The gaps closed here:
- `components/app-shell.tsx` — added `<ThemeToggle />` to the admin/student top bar (previously no theme control there).
- `app/login/page.tsx`, `app/register/page.tsx` — added the toggle to the auth card header.
- `app/layout.tsx` — a small pre-paint script applies the saved theme before first paint so pages no longer flash the default mode on load.

### Checks Run
- `go build ./...` — PASS.
- `go vet ./internal/handlers/` — PASS.
- `go test ./internal/handlers/... ./internal/models/... ./internal/eligibility/... ./internal/lifecycle/... ./internal/routes/... ./internal/policy/... -count=1` — PASS. Includes the new regression test.
- `npm run type-check` — PASS (exit 0).
- `go test ./...` — **environmental failures only**, in `internal/services` (passkey/recovery/saga) with `Binary was compiled with 'CGO_ENABLED=0', go-sqlite3 requires cgo to work. This is a stub`. These are not caused by this work and are unrelated to the changed packages.

### Documentation
- Created `docs/debugging/positions-hidden-by-domain-default.md` (symptom, reproduction, root-cause table, fix, verification, limitations).

### Deferred / Known limitations
- `SIGNUP_ALLOWED_DOMAINS` (`auth.go`) and the `NEXT_PUBLIC_*` defaults still say `std.mast.ac.ug` / `mast.ac.ug`, which look like a typo against `must.ac.ug` in `SRS.md` / `STACK.md` / `ADR.md` / `AGENTS.md`. Left unchanged here (out of scope for this pass); flagged in the debugging doc.
- Browser (Playwright) evidence for the theme switcher and the simplified form: no Playwright suite exists locally.

---

## [2026-11-03] Three-Stage Election Lifecycle, Rich Candidate Applications, Admin Review Modal, and Election Delete

**Branch:** `feature/mvp-phases-3-7` (continuing work on the same task branch).
**Goal:** Reduce the election lifecycle to exactly three stages, let a student submit a full candidacy application (personal details, passport photo, partial transcript), let an admin open that application in a modal and approve/reject it, and let an admin delete an election that has not yet collected a ballot.

### 1. Lifecycle reduced to three stages (back-end)
The lifecycle state machine in `apps/api/internal/lifecycle/lifecycle.go` now has a 3×3 `transitionMatrix`:

| From \ To | DRAFT | OPEN | CLOSED |
|---|---|---|---|
| **DRAFT** | — | auto at start time | allowed (close early) |
| **OPEN** | — | — | auto at end time / close early |
| **CLOSED** | — | — | terminal |

- Removed `SCHEDULED`, `RESULTS_UNDER_REVIEW`, `FINALIZED`, `ARCHIVED`, `CANCELLED`, `PENDING_APPROVAL`, `REGISTRATION`.
- `Ensure` auto-opens a `DRAFT` at its start time and auto-closes an `OPEN` at its end time; `SyncAll` sweeps due elections once a minute.
- A client-initiated `POST /elections/:id/transition {to: "OPEN"}` is **refused** — opening is a clock event, not a manual action.
- Certification no longer moves the election on: it relabels the current tally snapshot `PROVISIONAL` → `CERTIFIED` and is only accepted while the election is `CLOSED`.
- `ValidateConfiguration` / `ReadyToOpen` unchanged in spirit; readiness is advisory (it no longer gates a manual open).

### 2. Ballot-safe election delete (back-end)
`DELETE /elections/:id` (`apps/api/internal/handlers/election_lifecycle.go`):
1. `lifecycle.Ensure` to reconcile status with the clock first.
2. Count `AnonymousBallot` rows for the election; if **any** exist → `409 Conflict` and the election must be closed instead (its result stays auditable).
3. Otherwise, in one transaction: insert the `ELECTION_DELETED` audit row, then delete candidates, positions and the election.

New audit action `AuditActionElectionDeleted = "ELECTION_DELETED"` in `apps/api/internal/models/election_audit.go`. The route is listed in `internal/policy/routes.go` so `TestEveryRouteHasAPolicy` stays green.

### 3. Richer student application (back-end + front-end)
`POST /positions/:id/apply` (`apps/api/internal/handlers/position_workflow.go`) now **requires**: date of birth, sex, passport photo URL and partial-transcript URL. Identity fields (registration number, programme, study year) are derived from the authenticated student's email rather than typed in. New `Candidate` columns carry `sex`, `religion` and `partial_transcript_url`.
The apply form (`apps/web/app/positions/page.tsx`) uploads via the existing `POST /uploads` endpoint (`upload()` helper in `lib/api.ts`) and shows a passport/transcript preview.

### 4. Admin application review modal (front-end)
New `apps/web/components/application-review-dialog.tsx` — a native `<dialog>` (focus-contained, Escape-to-close, focus return) that shows the passport thumbnail, status chip and a `<dl className="kv">` of names / campus id / programme / year / DOB / age / sex / religion / residence / contact, document links (passport, partial transcript, optional student ID), slogan and bio, plus a reviewer note and a reason textarea. It offers **Previous / Next** navigation and **Approve / Reject** actions (reject requires a reason). `apps/web/app/staff/positions/page.tsx` opens it on row click / a **Review** button.

### 5. Delete-election UI (front-end)
`apps/web/app/staff/elections/[id]/page.tsx` gained a **Delete election** button in the Audit panel that opens a destructive `ConfirmDialog` and, on success, calls `elections.remove(id)` and redirects to `/staff`. The copy states that an election which has already collected a ballot cannot be deleted and should be closed instead.

### 6. Front-end lifecycle sweep
Every remaining reference to a removed status was cleared from `apps/web` (a repo-wide grep for `RESULTS_UNDER_REVIEW|FINALIZED|ARCHIVED|CANCELLED|SCHEDULED|PENDING_APPROVAL|REGISTRATION` in `*.tsx`/`*.ts` returns no matches): `app/results/page.tsx` and `app/staff/elections/[id]/results/page.tsx` now treat only `CLOSED` as countable; `app/staff/elections/[id]/page.tsx` filters manual transitions down to *close early*; `lib/types.ts`, `lib/format.ts` and their tests are reduced to `DRAFT | OPEN | CLOSED`.

### Checks Run
- `npm run type-check` — PASS (exit 0).
- `npm test` — PASS (12/12, 2 files).
- `npm run build` — PASS (Next.js 16.4.3; 28 routes emitted, incl. `/positions`, `/staff/positions`, `/results`, `/staff/elections/[id]`).
- `go test ./internal/handlers/... ./internal/lifecycle/... ./internal/policy/... ./internal/results/...` — PASS (all ok, cached).
- No Playwright/E2E suite exists locally, so the modal and delete flow have no browser-automation evidence.

### Documentation
- Rewrote `docs/election-lifecycle.md` for the three-stage model (transition matrix, auto open/close, lapsed-draft rule, client-open refusal, certification relabel, freeze/extension/early-close, and a new §6 on `DELETE /elections/:id`).
- Updated `docs/features/positions-first-workflow.md` (Link refuses a `CLOSED` election; Application-details bullet; "Admin application review" subsection; UI and Checks tables).
- Updated `docs/rbac.md` (results-visibility table reduced to `DRAFT`/`OPEN`/`CLOSED`; separation-of-duties bullet for CLOSED-only certification).

### Deferred / Known limitations
- Pre-existing Biome **lint** findings remain untouched: `next.config.ts` (`useNodejsImportProtocol`), `app/layout.tsx` (`noDangerouslySetInnerHtml` on the pre-paint theme script) and `app/staff/elections/[id]/candidates/page.tsx` (`noRedundantAlt`).
- `npm run lint` (= `biome check .`) also reports **formatter** diffs across many untouched files because the working tree is `i/lf w/crlf` on Windows; this is environmental and no code change was made to mask it.
- No browser evidence for the review modal or the delete confirm dialog (no local Playwright suite).

---

## [2026-11-03] Single-Choice Ballot, Complete Application Intake, Ballot Photos, Short Elections, and Post-Vote Confetti

**Branch:** `feature/mvp-phases-3-7` (continuing work on the same task branch).
**Goal:** a batch of maintainer-reported fixes — (1) a ballot should be a single pick per position with no review step, (2) the admin must actually receive every uploaded file and detail from a candidacy application, (3) the candidate's passport photo should appear on the ballot, an approved candidate may not withdraw but may upload a replacement ballot image, (4) after voting a confetti + tick modal replaces the ballot then returns home, (5) "Vote now" becomes an end-time-gated "View results", (6) positions lock once voting begins, (7) elections may be any length, e.g. 2 minutes or 20 hours.

### 1. Single-choice ballot, cast without a review screen (front-end)
`apps/web/app/elections/[id]/ballot/page.tsx`: the `reviewing` / `confirmed` states were removed. `choose()` now replaces the current pick for a non-multi position (and tapping the same option clears it); a `MULTI_CHOICE` position keeps its toggle behaviour. "Cast vote" validates and opens the existing `ConfirmDialog`, then submits directly. Copy now reads *"Choose one candidate per position."* The server rule (max 1 for a non-multi position) was already enforced by `validateSelectionsAgainst`.

### 2. Every application detail reaches the admin (back-end + front-end)
The student apply form (`apps/web/app/elections/[id]/candidate/apply/page.tsx`) now collects **sex**, **religion** and a **partial-transcript PDF** in addition to date of birth, passport photo and student ID, and posts `sex`, `religion` and `partial_transcript_url`. `POST /elections/:id/candidates/apply` (`apps/api/internal/handlers/candidate.go`) validates and persists all of them.
The admin review page (`apps/web/app/staff/elections/[id]/candidates/page.tsx`) now shows **sex** and **religion**, a **Partial transcript (PDF)** link and, when present, a **Ballot photo** link, alongside the existing passport/ID thumbnails (each of which is already a link).

### 3. Passport photo on the ballot; approved candidates cannot withdraw but may replace the ballot image
- `apps/api/internal/models/candidate.go` gained `BallotPhotoURL`; `apps/api/internal/handlers/ballot_version.go` renders `photo_url` with precedence **BallotPhotoURL → PassportPhotoURL → PhotoURL**, so the passport photo is the default ballot image.
- New route `POST /candidates/my/:candidate_id/ballot-photo` (`SetBallotPhoto`, `apps/api/internal/routes/routes.go`): approved-only, refused once voting has opened, audited.
- `WithdrawMine` now also refuses `APPROVED` (`"An approved application can no longer be withdrawn."`).
- The candidacy page (`apps/web/app/candidate/page.tsx`) hides **Withdraw** for `APPROVED` and shows a **Replace image** upload for approved candidates (uses `candidates.setBallotPhoto`). `ballot_photo_url` was added to the shared `Candidate` type and `setBallotPhoto` to `lib/api.ts`.
- The ballot page renders `candidate.photo_url` as `<img className="ballot-photo">` (falls back to the ballot-number placeholder).

### 4. Post-vote confetti + tick modal (front-end)
New `apps/web/components/vote-success.tsx` (72 seeded confetti pieces, an SVG tick, `.stamp`/`.badge` styling, auto-continues after ~3.2s, plus a "Go home now" button). Once `submit()` succeeds, the ballot page renders this overlay in place of the ballot and calls `router.replace("/")` when done. New `.vote-success` / `.vote-confetti` styles and a `prefers-reduced-motion` guard were added to `globals.css`.

### 5. "Vote now" → end-time-gated "View results"
`apps/web/app/elections/page.tsx` and `.../elections/[id]/page.tsx` now show **View results** only once the election has ended (`status === "CLOSED"` or `ends_at <= now`) and **Vote now** only while `accepting_votes`; before that the card links to the election page. The results link is hidden before the close time.

### 6. Positions lock when voting begins
Already enforced server-side: `GET /positions/eligible` (`position_workflow.go`) drops positions whose linked election has started or is closed, and `POST /positions/:id/apply` refuses once the election has begun. Confirmed unchanged; the student positions page consumes `positions.eligible()`.

### 7. Elections of any length (back-end + front-end)
`apps/api/internal/lifecycle/lifecycle.go`: `MinimumDuration` lowered from `time.Hour` to `2 * time.Minute`, so a two-minute poll is accepted; the corresponding `lifecycle_test.go` case was changed from 30 minutes to 1 minute (previously below the old floor). The staff create form (`app/staff/page.tsx`) and election edit form (`app/staff/elections/[id]/page.tsx`) gained **quick-length** buttons (2 minutes, 1 hour, 6 hours, 20 hours, 1 day, 1 week) that compute `ends_at` from `starts_at`; arbitrary custom lengths remain possible.

### Checks Run
- `go build ./...` — PASS.
- `go test ./internal/handlers/... ./internal/lifecycle/... ./internal/models/...` — PASS.
- `npm run type-check` — PASS (exit 0).
- `npm run test` — PASS (12/12, 2 files).
- Formatted the touched front-end files with `biome check --write`; no new lint findings (the pre-existing `noRedundantAlt` and the environmental CRLF formatter noise remain, as above).

### Documentation
- Appended this entry to `docs/progress/development-log.md`. The single-choice rule, ballot-photo precedence and the 2-minute minimum are behaviour changes other docs may eventually restate; only the progress log was updated in this pass.

### Deferred / Known limitations
- `BallotCandidate` (the ballot payload) still types the image as `photo_url` only; the precedence is resolved on the server, so no separate `ballot_photo_url` field is exposed to the voter payload.
- No browser/Playwright evidence for the confetti modal, the ballot-photo upload or the duration presets (no local Playwright suite).
- The apply form keeps **Ballot photo** as an optional extra on top of the required **Passport photo**; the two remain distinct columns.

