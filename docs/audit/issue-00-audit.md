# ISSUE-00 — Implementation Audit & SRS Traceability
## campVote — MMU Campus E-Voting System

| Field | Value |
|---|---|
| Date | 2026-10-28 |
| Branch | `chore/issue-00-implementation-audit` |
| Auditor | AI agent (AGENTS.md compliant) |
| Base commit | `22c6255` on `develop` |
| Authoritative source | `https://github.com/Izaek256/campVote.git` |

---

## 1. Source of Truth

### 1.1 Repository status

The canonical repository (`https://github.com/Izaek256/campVote.git`) is **not empty**.  
Three commits on `develop`/`main` represent the authoritative source:

| Commit | Message | Branch |
|---|---|---|
| `22c6255` | Merge pull request #1 from Izaek256/feature/01-grit-triple-monorepo | develop / main |
| `f23a666` | feat(domain): scaffold Grit Triple monorepo and implement campus voting domain with privacy air-gap | merged via PR #1 |
| `608453b` | docs(kickstart): establish project specifications, rules, and documentation baseline | initial |

**Branches present:**

| Branch | Remote | Notes |
|---|---|---|
| `main` | `origin/main` | ✅ exists |
| `develop` | `origin/develop` | ✅ exists |
| `feature/01-grit-triple-monorepo` | `origin/feature/01-grit-triple-monorepo` | merged; remote HEAD points here (likely leftover default) |

> [!WARNING]
> `remotes/origin/HEAD -> origin/feature/01-grit-triple-monorepo` — the remote's default branch pointer still points at the feature branch that was already merged. This should be corrected to `main` or `develop` in the GitHub repository settings.

### 1.2 Branch protection

Branch protection was **not verified** from this audit (requires GitHub API or Settings UI access). The `CONTRIBUTING.md` and `AGENTS.md` require protection on `main` and `develop`. This must be confirmed manually.

---

## 2. Tech Stack & Versions

### 2.1 Actual stack (verified from scaffold)

| Layer | Actual | STACK.md Spec | Status |
|---|---|---|---|
| Full-stack framework | Grit Framework `v3.411.17` | Grit Framework | ✅ Match |
| Architecture | Triple (Web + Admin + Go API) | Triple | ✅ Match |
| Backend language | Go `1.26.4` | Go | ✅ Match |
| Backend HTTP | Gin `v1.11.0` (via Grit) | Gin (through Grit) | ✅ Match |
| ORM | GORM `v1.31.1` (via Grit) | GORM (through Grit) | ✅ Match |
| Public frontend | Next.js (TypeScript) — `apps/web` | Next.js (preferred) | ✅ Match |
| Admin frontend | Next.js/Grit admin — `apps/admin` | Grit generated admin | ✅ Match |
| Frontend language | TypeScript | TypeScript | ✅ Match |
| Shared types | `packages/shared` (Zod + TS, 51 models synced via `grit sync`) | `grit sync` | ✅ Match |
| Production database | PostgreSQL 16 (`compose.dev.yaml`) | PostgreSQL | ✅ Match |
| Dev/test DB | SQLite via `glebarez/sqlite` (pure-Go, in-memory) | Not in STACK.md | **Addition** — documented in `SETUP_LOG.md` |
| Local infra | Docker Compose (`compose.dev.yaml`) | Docker + Compose | ✅ Match |
| Cache/queue | Redis (optional, via `REDIS_URL`) | Redis if required | ✅ Match |
| Realtime | WebSocket hub + SSE fallback (`internal/realtime/`) | Grit realtime | ✅ Match |
| Email | Mailer with multiple drivers (SMTP, Resend, SendGrid, Mailgun, SES) | Grit mail | ✅ Match |
| Storage | S3-compatible + local (`internal/storage/`) | S3/MinIO | ✅ Match |
| Security suite | Sentinel v2.7.0 (WAF, rate limiting, AuthShield) | Not in STACK.md | **Addition** — significant |
| Observability | Pulse v1.5.1 (request tracing, DB monitoring) | Not in STACK.md | **Addition** |
| Monorepo tooling | Turborepo `v2.0.0` + pnpm `10.33.4` | Not in STACK.md | **Addition** |
| Code quality | Biome `2.5.14` (TS) + golangci-lint | Not in STACK.md | **Addition** |
| Load testing | k6 scripts (`tests/k6/`) | Mentioned generically | ✅ Match |
| CI/CD | No active workflow yet (`.github/workflows/README.md` explains why) | Repository CI pipeline | **Deviation — see §5** |

### 2.2 Deviations from STACK.md / ADR.md

| # | Deviation | Risk | Recommendation |
|---|---|---|---|
| D-001 | No active CI workflow — workflows directory contains only a README | HIGH: no automated checks on PRs | Create `.github/workflows/ci.yml` after confirming real workspace paths (ISSUE-14) |
| D-002 | SQLite used for all unit/seeder tests (not PostgreSQL) | MEDIUM: unique constraint behaviour may differ | Add PostgreSQL integration test job alongside SQLite unit tests (ISSUE-14) |
| D-003 | `compose.dev.yaml` provides PostgreSQL but `SETUP_LOG.md` notes Docker not installed in host PATH | MEDIUM: developer may not have Docker | Document fallback / add Docker install instructions to README |
| D-004 | Remote HEAD → `feature/01-grit-triple-monorepo` instead of `main`/`develop` | LOW: cosmetic but confusing | Change default branch in GitHub Settings |
| D-005 | `apps/admin` and `apps/web` are minimal scaffolds — no election-specific pages built | EXPECTED: Phase 0 work — flagged for tracking | Implement in ISSUE-02 through ISSUE-08 |
| D-006 | Sentinel v2 and Pulse added (not in STACK.md) | NEUTRAL: beneficial additions | Add to `STACK.md` |
| D-007 | No OIDC/SSO integration with MMU IdP | EXPECTED: pending university decision (OPEN_QUESTIONS.md) | Track in ISSUE-01 |

---

## 3. Database Tables & Migrations

### 3.1 Table inventory

All tables are managed through GORM `AutoMigrate` — there are no SQL migration files. Migration is applied by `cmd/migrate/main.go` calling `db.AutoMigrate(models.Models()...)`.

**Campus-voting domain tables (new in this branch):**

| Table | Go Model | Purpose | Key Constraints |
|---|---|---|---|
| `elections` | `models.Election` | Voting events with schedule and policy | `status` index; `starts_at`/`ends_at` indexes; `created_by_id` index |
| `positions` | `models.Position` | Ballot positions/questions per election | FK `election_id`; cascade on delete |
| `candidates` | `models.Candidate` | Candidates per position | FK `election_id`, `position_id`; cascade |
| `anonymous_ballots` | `models.AnonymousBallot` | Accepted ballot selections — **NO voter FK** | `election_id` index; `status` index |
| `ballot_selections` | `models.BallotSelection` | Per-position choices within a ballot | FK `ballot_id`, `position_id`; `candidate_id` (nullable) |
| `voter_participations` | `models.VoterParticipation` | Who voted — **NO ballot FK** | **UNIQUE `(election_id, user_id)`** — duplicate-vote prevention |
| `election_rosters` | `models.ElectionRoster` | Approved eligible voters per election | **UNIQUE `(election_id, email)`**, **UNIQUE `(election_id, student_number)`** |
| `election_tally` | `models.ElectionTally` | Result snapshots per position/candidate | `election_id`, `position_id` indexes |
| `election_audit_logs` | `models.ElectionAuditLog` | Administrative events — **no ballot choices** | `election_id`, `actor_id`, `action` indexes |

**Grit framework tables (inherited from scaffold):**

`users`, `sessions`, `roles`, `user_roles`, `api_keys`, `email_verification_tokens`, `password_reset_tokens`, `magic_link_tokens`, `totp_*`, `passkeys`, `trusted_devices`, `uploads`, `blogs`, `notifications`, `activity_logs`, `outbox_messages`, `webhooks`, `feature_flags`, `settings`, `backup_schedules`, `backups`, `tickets`, `ticket_replies`, `form_shares`, `form_submissions`, `import_jobs`, `saga_runs`, `saga_steps`, `sso_connections`, `saml_keypairs`, `deletion_journal`, `access_reviews`, `access_review_items`, `recovery_contacts`, `recovery_contact_tokens`, `user_activities`, `dashboard_layouts`

### 3.2 Migration process

- **Mechanism:** GORM `AutoMigrate` — not file-based SQL migrations.  
- **Command:** `go run ./cmd/migrate/main.go` from `apps/api/`  
- **From empty DB:** Runs `db.AutoMigrate(models.Models()...)` — creates all tables on first run.  
- **Dev/test shortcut:** Tests use `db.AutoMigrate(models.Models()...)` against in-memory SQLite (zero-dependency).

> [!IMPORTANT]
> AutoMigrate does not drop columns or roll back schema changes. For production use, consider adopting a versioned migration tool (e.g., `golang-migrate`, `atlas`, or `dbmate`) and adding a `migrations/` directory. This is tracked as ISSUE-15.

### 3.3 Migration verification status

- ✅ Verified: `go test ./internal/models/...` and `go test ./internal/database/...` run AutoMigrate against SQLite in-memory — pass.
- ❌ Not verified: clean PostgreSQL start from `compose.dev.yaml` was not run during this audit (Docker not available in host PATH per `SETUP_LOG.md`).

---

## 4. Seed / Demo / Default Data Inventory

### 4.1 Users seeder (`apps/api/internal/database/users_seeder.go`)

| Account | Email | Role | Password | Condition |
|---|---|---|---|---|
| Default admin | `admin@example.com` | ADMIN | `admin123` (dev default) or `SEED_ADMIN_PASSWORD` env | Only when `APP_ENV=development` OR `SEED_ADMIN_PASSWORD` set |
| Demo user | `jane@example.com` | EDITOR | `admin123` | `APP_ENV=development` only |
| Demo user | `robert@example.com` | USER | `admin123` | `APP_ENV=development` only |
| Demo user | `emily@example.com` | USER | `admin123` | `APP_ENV=development` only |
| Demo user | `michael@example.com` | USER (inactive) | `admin123` | `APP_ENV=development` only |

> [!CAUTION]
> `admin@example.com` / `admin123` is **not** an MMU institutional account. This is a generic Grit scaffold default and must be replaced with an MMU-domain admin account before any demo or staging deployment. The seeder correctly refuses `admin123` outside `APP_ENV=development` unless `SEED_ADMIN_PASSWORD` is set.

### 4.2 Elections seeder (`apps/api/internal/database/elections_seeder.go`)

| Item | Detail | Condition |
|---|---|---|
| MMU demo admin | `commissioner@must.ac.ug` / `admin123` | Seeded if no elections exist |
| Demo voter 1 | `ivan.mugisha@std.must.ac.ug` / `admin123` | Seeded with election data |
| Demo voter 2 | `doreen.kemigisa@std.must.ac.ug` / `admin123` | Seeded with election data |
| Guild election | "MMU Guild Representative Council Elections 2026/2027" — status `OPEN` | Created if no elections exist |
| Faculty election | "Faculty of Science & Technology Student Representative" — status `SCHEDULED` | Created with guild election |
| Guild positions | President, Vice President, Referendum — 3 positions | Under guild election |
| Guild candidates | Akiki Brenda, Mugisha Davis, Tusiime Faith (President); Katusabe Emmanuel, Natukunda Peace (VP); YES, NO (Referendum) — 7 total | Under guild election |
| Roster entries | Ivan Mugisha (22/U/1001/PS, CS Y3), Doreen Kemigisa (22/U/1002/PS, BBA Y2) | Under guild election |

> [!NOTE]
> All seed data is **synthetic demo data only**. No real student records, real votes, or real election outcomes are present. Seed passwords (`admin123`) are guarded by `APP_ENV=development` checks.

### 4.3 Secret scan

**Searched patterns:** `admin123`, `changeme`, `password=`, `INSERT INTO`, hardcoded emails, JWT secrets, private keys.

**Findings:**

| File | Line | Finding | Risk |
|---|---|---|---|
| `apps/api/internal/database/users_seeder.go` | 42, 63, 80–83, 97 | `admin123` — dev default credential | LOW — guarded by `APP_ENV=development` check |
| `apps/api/internal/database/elections_seeder.go` | 33, 41, 49 | `admin123` — demo account credentials | LOW — idempotent seeder; synthetic accounts only |
| `apps/api/internal/password/password.go` | 221 | `admin123` in common-passwords blocklist | NONE — this is a blocklist, not a credential |
| `.env.example` | 9 | `change-me-local-only` placeholder password | NONE — placeholder only, untracked |
| No `.env` files committed | — | ✅ Confirmed clean | — |
| No JWT secrets, private keys, or tokens | — | ✅ Confirmed clean | — |

**Full Git history scanned** (3 commits): No secrets found in any commit.

---

## 5. API Route Inventory

All routes are served under `/api/v1/` (versioned). Unversioned `/api/...` requests are rewritten to `/api/v1/...` by the legacy alias.

### 5.1 Public (no authentication required)

| Method | Path | Auth | Role | Purpose |
|---|---|---|---|---|
| GET | `/api/health` | None | None | Health check |
| GET | `/api/v1/health` | None | None | Health check (versioned) |
| GET | `/.well-known/jwks.json` | None | None | JWT public keys |
| GET | `/api/ws` | Cookie or `?token=JWT` | Any authenticated | WebSocket (realtime) |
| GET | `/api/events` | Cookie or `?token=JWT` | Any authenticated | SSE (realtime fallback) |
| POST | `/webhooks/:provider` | Provider signature | None | Webhook receiver |
| GET | `/api/v1/blogs` | None | None | List published blogs |
| GET | `/api/v1/blogs/:slug` | None | None | Get blog by slug |
| GET | `/api/v1/public/*` | API key | None | Public resource endpoints |
| GET | `/llms.txt` | None | None | LLMs route index (dev only) |
| GET | `/llms-full.txt` | None | None | LLMs route details (dev only) |

### 5.2 Auth (public — no session yet)

| Method | Path | Auth | Role |
|---|---|---|---|
| POST | `/api/v1/auth/register` | None | None |
| POST | `/api/v1/auth/login` | None | None |
| POST | `/api/v1/auth/refresh` | Refresh cookie | None |
| POST | `/api/v1/auth/forgot-password` | None | None |
| POST | `/api/v1/auth/reset-password` | Reset token | None |
| POST | `/api/v1/auth/verify-email` | Email token | None |
| POST | `/api/v1/auth/magic-link` | None | None |
| POST | `/api/v1/auth/magic-link/consume` | Magic link token | None |
| GET | `/api/v1/auth/oauth/:provider` | None | None |
| GET | `/api/v1/auth/oauth/:provider/callback` | OAuth code | None |
| POST | `/api/v1/auth/sso/discover` | None | None |
| GET | `/api/v1/auth/sso/:slug` | None | None |
| GET | `/api/v1/auth/sso/:slug/callback` | OIDC token | None |
| GET | `/api/v1/auth/saml/:slug/metadata` | None | None |
| GET | `/api/v1/auth/saml/:slug` | None | None |
| POST | `/api/v1/auth/saml/:slug/acs` | SAML assertion | None |
| POST | `/api/v1/auth/passkeys/login/begin` | None | None |
| POST | `/api/v1/auth/passkeys/login/finish` | Challenge | None |
| POST | `/api/v1/auth/totp/verify` | Pending token | None |
| POST | `/api/v1/auth/totp/backup-codes/verify` | Pending token | None |

### 5.3 Protected (JWT or API key required — any authenticated user)

| Method | Path | Auth | Role | Purpose |
|---|---|---|---|---|
| GET | `/api/v1/auth/me` | JWT | Any | Current user profile |
| GET | `/api/v1/auth/permissions` | JWT | Any | Caller's permissions |
| GET | `/api/v1/system/modules` | JWT | Any | Enabled modules |
| POST | `/api/v1/auth/logout` | JWT | Any | Logout |
| GET | `/api/v1/auth/sessions` | JWT | Any | List active sessions |
| DELETE | `/api/v1/auth/sessions/:id` | JWT | Any | Revoke session |
| POST | `/api/v1/auth/sessions/revoke-all` | JWT | Any | Revoke all sessions |
| POST/GET/DELETE | `/api/v1/auth/totp/*` | JWT | Any | TOTP 2FA management |
| GET/POST/PATCH/DELETE | `/api/v1/auth/passkeys/*` | JWT | Any | Passkey management |
| GET/POST/DELETE | `/api/v1/auth/recovery/*` | JWT | Any | Recovery contacts |
| GET/POST/DELETE | `/api/v1/api-keys` | JWT | Any | API key management |
| POST | `/api/v1/uploads` | JWT | Any | File upload |
| GET | `/api/v1/uploads/*` | JWT | Any | Upload management |
| POST/GET/DELETE | `/api/v1/profile` | JWT | Any | User profile |
| GET/POST | `/api/v1/notifications/*` | JWT | Any | Notifications |
| GET/PUT | `/api/v1/dashboard-layout` | JWT | Any | Dashboard layout |
| POST/GET/PATCH | `/api/v1/tickets/*` | JWT | Any | Support tickets |
| GET/POST/PUT | `/api/v1/sync/*` | JWT | Any | Offline sync |
| GET | `/api/v1/settings` | JWT | Any | Read settings |
| POST/GET | `/api/v1/ai/*` | JWT | Any (MODULE_AI=true) | AI completions |
| GET | `/api/v1/imports/:id` | JWT | Any | Import job status |
| GET | `/api/v1/users/:id/gdpr-export` | JWT | Own or ADMIN | GDPR data export |

### 5.4 Staff (JWT required + `perm:*` or ADMIN)

| Method | Path | Auth | Role |
|---|---|---|---|
| GET/POST/PUT/DELETE | `/api/v1/users/*` | JWT | ADMIN or `perm:users.*` |
| POST | `/api/v1/users/:id/gdpr-erase` | JWT | ADMIN or `perm:users.delete` |
| GET | `/api/v1/gdpr/journal` | JWT | ADMIN or `perm:audit.view` |
| GET | `/api/v1/admin/activity*` | JWT | ADMIN or `perm:audit.view` |
| GET | `/api/v1/user-activity*` | JWT | ADMIN or `perm:audit.view` |
| GET | `/api/v1/audit/ocsf` | JWT | ADMIN or `perm:audit.view` |
| GET/POST/PUT/DELETE | `/api/v1/roles/*` | JWT | ADMIN or `perm:roles.*` |
| GET/POST | `/api/v1/access-reviews/*` | JWT | ADMIN or `perm:roles.*` |
| GET | `/api/v1/permissions` | JWT | ADMIN or `perm:roles.view` |
| GET | `/api/v1/scale` | JWT | ADMIN or `perm:system.view` |
| GET/POST/DELETE | `/api/v1/admin/jobs/*` | JWT | ADMIN or `perm:jobs.*` |
| GET/POST | `/api/v1/admin/sagas/*` | JWT | ADMIN or `perm:jobs.*` |
| GET | `/api/v1/admin/routes` | JWT | ADMIN or `perm:settings.view` |
| GET | `/api/v1/admin/cron/tasks` | JWT | ADMIN or `perm:jobs.view` |
| GET | `/api/v1/admin/mail/*` | JWT | ADMIN or `perm:system.view` |
| GET | `/api/v1/admin/security/*` | JWT | ADMIN or `perm:system.view` |
| GET | `/api/v1/admin/observability/*` | JWT | ADMIN or `perm:system.view` |
| GET/POST/DELETE | `/api/v1/admin/trash/*` | JWT | ADMIN or `perm:system.view` |
| GET/POST/DELETE | `/api/v1/admin/deleted-accounts/*` | JWT | ADMIN |
| GET | `/api/v1/admin/webhooks` | JWT | ADMIN or `perm:system.view` |
| GET | `/api/v1/admin/flags/*` | JWT | ADMIN or `perm:system.view` |
| GET/POST/PUT/DELETE | `/api/v1/admin/blogs/*` | JWT | ADMIN or `perm:blogs.*` |
| GET | `/api/v1/admin/dashboard/*` | JWT | ADMIN or `perm:*.view` |
| GET/POST/PUT | `/api/v1/backups/*` | JWT | ADMIN or `perm:backups.*` |

### 5.5 Admin (JWT required + ADMIN role)

| Method | Path | Auth | Role |
|---|---|---|---|
| POST | `/api/v1/admin/activity/reseal` | JWT | ADMIN |
| POST | `/api/v1/admin/webhooks/:id/replay` | JWT | ADMIN |
| POST/PUT/DELETE | `/api/v1/admin/flags/*` | JWT | ADMIN |
| GET/POST/PUT/DELETE | `/api/v1/sso/connections/*` | JWT | ADMIN |
| GET/POST/PATCH/DELETE | `/api/v1/admin/form-shares/*` | JWT | ADMIN |
| GET | `/api/v1/admin/form-submissions` | JWT | ADMIN |
| PUT/DELETE | `/api/v1/settings/*` | JWT | ADMIN |

### 5.6 Election-specific routes

> [!IMPORTANT]
> **Zero election-specific API routes exist today.** The following are planned but not yet implemented. See `docs/handover/current-state.md` §6.

| Method | Planned Path | Auth | Role | Status |
|---|---|---|---|---|
| GET | `/api/v1/elections` | JWT | Any eligible voter | ❌ Not built |
| POST | `/api/v1/elections` | JWT | Admin/Official | ❌ Not built |
| GET | `/api/v1/elections/:id` | JWT | Any | ❌ Not built |
| GET | `/api/v1/elections/:id/ballot` | JWT | Eligible voter | ❌ Not built |
| POST | `/api/v1/elections/:id/vote` | JWT | Eligible voter | ❌ Not built |
| GET | `/api/v1/elections/:id/results` | JWT/None (policy) | Per election policy | ❌ Not built |
| GET | `/api/v1/elections/:id/tally` | JWT | Admin/Official | ❌ Not built |
| GET | `/api/v1/elections/:id/roster` | JWT | Admin/Official | ❌ Not built |
| POST | `/api/v1/elections/:id/roster/import` | JWT | Admin/Official | ❌ Not built |
| GET/POST | `/api/v1/elections/:id/candidates` | JWT | Admin/Official | ❌ Not built |
| POST | `/api/v1/elections/:id/certify` | JWT | Chief Admin | ❌ Not built |

---

## 6. Test Inventory

### 6.1 Go tests (unit + integration)

| Test File | Tests | Coverage Area | State |
|---|---|---|---|
| `models/voting_test.go` | `TestBallotPrivacyDecoupling`, `TestElectionLifecycle`, `TestDuplicateVotingConstraint` | Privacy air-gap, election schedule, duplicate-vote prevention | ✅ PASS (verified in handover doc) |
| `database/elections_seeder_test.go` | `TestSeedElections` | Seeder idempotency, guild election data integrity | ✅ PASS |
| `appendonly/appendonly_test.go` | Append-only log chain | Audit log tamper-evidence | ✅ PASS (Grit framework) |
| `authz/*_test.go` | RBAC actor, gate, ceiling, grants, permissions | Authorization framework | ✅ PASS (Grit framework) |
| `handlers/auth_test.go` | Auth handler | Registration, login, JWT | ✅ PASS (Grit framework) |
| `services/session_test.go` | Session revocation | Session management | ✅ PASS (Grit framework) |
| `services/two_factor_test.go` | TOTP setup, enable, verify | 2FA | ✅ PASS (Grit framework) |
| `handlers/role_test.go` | Role assignment | RBAC | ✅ PASS (Grit framework) |
| `realtime/*_test.go` | Hub, channels, backplane | Realtime | ✅ PASS (Grit framework) |
| `concurrency/concurrency_test.go` | Concurrent access | Concurrency primitives | ✅ PASS (Grit framework) |
| *(90+ other Grit framework tests)* | Infrastructure | Auth, sessions, jobs, mail, storage, etc. | ✅ PASS (per handover doc) |

**Election-specific tests absent:**

| Missing Test | What it should cover | Issue |
|---|---|---|
| Ballot submission handler | Eligibility gate, atomic acceptance, error cases | ISSUE-06 |
| Tally service | Known-answer vectors, tie detection, rounding | ISSUE-08 |
| Eligibility service | Domain check, roster lookup, suspended/graduated | ISSUE-05 |
| Concurrency / race condition | Two simultaneous votes by same user | ISSUE-06 |
| Election API routes | CRUD, lifecycle transitions, auth on each route | ISSUE-02 |
| Results publication | Policy enforcement (live/after-close/certified) | ISSUE-07 |
| Roster import | CSV parsing, duplicate detection, validation | ISSUE-05 |
| Privacy threat model | Timing, backup correlation | ISSUE-09 |

### 6.2 How to run Go tests

```powershell
# From apps/api/
go test -v ./...
# Or specifically:
go test -v ./internal/models/...
go test -v ./internal/database/...
```

All tests use in-memory SQLite (`glebarez/sqlite`) — no database connection required.

### 6.3 Frontend tests

| App | Test Directory | Framework | Status |
|---|---|---|---|
| `apps/web` | `apps/web/__tests__` | Unknown (Grit scaffold) | ❌ No test files observed; scaffold minimal |
| `apps/admin` | `apps/admin/__tests__` | Unknown (Grit scaffold) | ❌ No test files observed |
| Root | `pnpm test:e2e` / Playwright | Playwright 1.48.0 | ❌ No e2e test files yet |
| Load tests | `tests/k6/` | k6 | ❌ Scripts scaffolded; not run against live stack |

---

## 7. README Run Instructions Assessment

**Current README state:** Accurate for the current pre-implementation stage.  
**Gaps for a clean-clone follow-through:**

1. README does not include actual `go run ./cmd/migrate/main.go` or `go run ./cmd/server/main.go` commands — it defers to "generated scaffold determines commands."
2. No step-by-step "start the API locally" walkthrough with concrete commands.
3. Docker not available in the developer environment; README mentions Docker but `SETUP_LOG.md` notes Docker not in PATH.
4. Frontend `pnpm install` / `pnpm dev` steps not yet confirmed to work (minimal scaffold, no election pages).

**Recommended README additions** (deferred to relevant issues):

```
# Quick start (backend only — no Docker)
cp .env.example .env  # Fill in DATABASE_URL for your local PostgreSQL
cd apps/api
go run ./cmd/migrate/main.go
APP_ENV=development go run ./cmd/seed/main.go
go run ./cmd/server/main.go
```

---

## 8. Re-scoped Issue Plan

| Issue | Original Scope | Revised Status | Reasoning |
|---|---|---|---|
| ISSUE-01: Authentication | Institutional auth + RBAC | **Full** | Auth infra from Grit (JWT, TOTP, passkeys, SSO) is wired; MMU domain enforcement and election-specific roles missing |
| ISSUE-02: Election CMS | Election lifecycle + admin UI | **Full** | Models and constants implemented; all HTTP handlers, admin UI, lifecycle transitions missing |
| ISSUE-03: Candidate management | Candidate CRUD + approval | **Full** | Model exists; handlers, eligibility check, approval workflow missing |
| ISSUE-04: Ballot builder | Visual ballot builder + validation | **Full** | Position/Question model complete; no ballot builder UI, no pre-open validator, no ballot fetch endpoint |
| ISSUE-05: Eligibility | Roster import + eligibility gate | **Full** | Roster model complete; no CSV import, no eligibility check service, no RBAC on roster endpoints |
| ISSUE-06: Voting + privacy | Atomic submission + concurrency | **Full** | Privacy architecture verified; no submission handler, no election ballot endpoint, no concurrency test |
| ISSUE-07: Results dashboard | Public results API + realtime | **Full** | ResultPolicy constants present; no results endpoint, no election public page, no realtime channel |
| ISSUE-08: Tally + certification | Tally service + certification | **Full** | Snapshot model present; no tally computation, no known-answer tests, no certification handler |
| ISSUE-09: Audit + privacy review | Threat model + audit hooks | **Full** | ElectionAuditLog model present; no service emitting events, no threat model document |
| ISSUE-10: Reports | PDF + CSV election reports | **Full** | Generic export infrastructure from Grit; no election-specific report generator |
| ISSUE-11: Notifications | Election lifecycle notifications | **Reduced** | Generic notification + mail infrastructure from Grit; wire election events to existing mailer |
| ISSUE-12: Security + operations | Security review + deployment | **Full** | Sentinel + WAF present; no election-specific IDOR tests, no CI workflow, no TLS config |
| ISSUE-13: Load testing | k6 + concurrency | **Reduced** | k6 scripts scaffolded; run against live stack once API is deployed |
| ISSUE-14: CI/CD | CI workflows | **Full** | No active CI workflow; `.github/workflows/README.md` explains; create after stack verified |
| ISSUE-15: Migration tooling | Versioned migration files | **New** | AutoMigrate lacks rollback and SQL history; recommend versioned migration tool |

---

## 9. Findings Summary

### Critical gaps
1. **No election API endpoints** — zero HTTP routes for elections, candidates, voting, results, or roster operations.
2. **No CI pipeline** — PRs merge without automated checks.
3. **MMU domain enforcement absent** — `AllowedDomains` field exists on Election but is not enforced at registration/login.
4. **Tally service absent** — ElectionTally model is a stub; no computation, no known-answer tests.
5. **Privacy threat model not documented** — ADR-006 requires this before production.

### Significant positives
1. **Privacy air-gap is correct by design** — `AnonymousBallot` has no voter FK; `VoterParticipation` has no ballot FK; confirmed by structural reflection test.
2. **Duplicate-vote prevention is correct** — DB-level unique constraint `(election_id, user_id)` enforced and tested.
3. **Election models are comprehensive** — all lifecycle states, categories, question types, roster statuses, candidate statuses, and tally fields are defined.
4. **Grit scaffold quality is high** — Sentinel WAF, TOTP, passkeys, SSO, realtime, audit log, GDPR, access reviews all wired.
5. **Secret hygiene is clean** — no real credentials in any commit; `admin123` guarded by `APP_ENV=development`.

### Risks
1. **AutoMigrate vs versioned migrations** — schema history cannot be replayed or rolled back. Must address before production elections.
2. **SQLite vs PostgreSQL tests** — the duplicate-vote constraint test runs on SQLite; behaviour confirmed but PostgreSQL integration test still needed.
3. **No concurrent-submission test** — the uniqueness constraint should prevent races but this has not been load-tested.
4. **Remote HEAD points at merged feature branch** — minor repo hygiene issue.

---

## 10. Open Questions

Carried forward from `docs/implementation/OPEN_QUESTIONS.md` — no answers received yet:

1. Does MMU provide approved SSO/OIDC?
2. Who is authoritative for current student/staff status?
3. Confirm `@std.must.ac.ug` and `@must.ac.ug` domains with university IT.
4. Who can upload rosters, and what approval is required?
5. Who approves election rules, tie handling, and certification?
6. What is the required ballot privacy guarantee (table separation vs. cryptographic)?
7. Where will the academic demo run and who owns credentials?
8. Is email needed for MVP?
9. Which features are mandatory for the assessment vs. stretch goals?

---

*Document generated: 2026-10-28. Branch: `chore/issue-00-implementation-audit`. See `docs/traceability.md` for the full per-requirement matrix.*

