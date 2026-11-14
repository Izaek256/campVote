# Documentation Index — CampVote

## 1. Baseline Specifications

- [`../SRS.md`](../SRS.md) — System Requirements Specification (functional, security, privacy).
- [`../STACK.md`](../STACK.md) — Technology stack specification (Grit Triple, Go, Next.js, PostgreSQL).
- [`../ADR.md`](../ADR.md) — Architecture Decision Records (ADR-001 through ADR-013).
- [`../AGENTS.md`](../AGENTS.md) — Non-negotiable AI agent development rules.
- [`../CONTRIBUTING.md`](../CONTRIBUTING.md) — Git workflow, branch policy, commit conventions.

## 2. Requirements & Progress Tracking

- [`progress/requirements-tracker.md`](progress/requirements-tracker.md) — Requirement-to-implementation status matrix.
- [`progress/development-log.md`](progress/development-log.md) — Chronological log of changes, checks, and next tasks.
- [`handover/current-state.md`](handover/current-state.md) — Current state of the system for developers and agents.

## 3. Implementation Records

- [`auth.md`](auth.md) — ISSUE-02 authentication, session, lockout, CSRF and first-admin bootstrap status with test evidence.
- [`features/admin-login-mobile.md`](features/admin-login-mobile.md) — ISSUE-30 student/admin sign-in separation, administrator 2FA, server role enforcement and mobile layout checks.
- [`implementation/GOOGLE_SIGNIN.md`](implementation/GOOGLE_SIGNIN.md) — Google Identity Services student sign-in: token verification, account linking, configuration, Google Cloud setup, rollback and troubleshooting.
- [`rbac.md`](rbac.md) — ISSUE-03 server-side RBAC: SRS role matrix, route policies, per-election scoping, results visibility, certification and test evidence.
- [`eligibility.md`](eligibility.md) — ISSUE-04 per-election eligibility: roster evidence, verified-email identity, CSV import, freeze/revocation, audit and open policy questions.
- [`election-lifecycle.md`](election-lifecycle.md) — ISSUE-05 election states, transition matrix, schedule boundaries, server clock, configuration validation/freeze, audited exceptions and test evidence.
- [`vote-submission.md`](vote-submission.md) — ISSUE-08 atomic, idempotent, concurrency-safe ballot submission, ballot-version binding, vote-status endpoint, voter views and test evidence.
- [`implementation/ROADMAP.md`](implementation/ROADMAP.md) — Phased milestone delivery order.
- [`implementation/OPEN_QUESTIONS.md`](implementation/OPEN_QUESTIONS.md) — Unresolved domain and institutional questions.
- [`implementation/SETUP_LOG.md`](implementation/SETUP_LOG.md) — Verified toolchain versions and scaffold facts.
- [`maintenance/repository-cleanup.md`](maintenance/repository-cleanup.md) — Repository audit and file hygiene log.

## 4. Operations

- [`operations/docker-compose.md`](operations/docker-compose.md) — Why compose containerises only the API (database, cache, mail and web external, configured through `.env`), what that costs, and the verified setup commands.
- [`operations/docker-full-stack.md`](operations/docker-full-stack.md) — WSL2/Linux full stack (PostgreSQL + API + web) and the Windows Docker Desktop PostgreSQL setup, with commands, networking and verification.
- [`operations/vercel-proxy.md`](operations/vercel-proxy.md) — Hosting the web app on Vercel with the API elsewhere: mixed-content and cross-site cookie reasons for proxying `/api` and `/files`, the environment variables, and the limits of the arrangement.

## 5. UI

- [`ui-views.md`](ui-views.md) — Registry of every UI view: route, roles, API, states, owning issue.
- [`ui-kit/campVote-UI-Kit.html`](ui-kit/campVote-UI-Kit.html) — Design reference (A0): components, tokens and layout patterns every view follows.
- [`features/user-registration.md`](features/user-registration.md) — ISSUE-20 voter self-registration flow (FR-AUTH-01, FR-AUTH-07).
- [`features/admin-user-management.md`](features/admin-user-management.md) — ISSUE-20 admin user management UI + missing admin infrastructure (FR-AUTH-06, FR-AUTH-07).
- [`features/positions-first-workflow.md`](features/positions-first-workflow.md) — Phase 2 positions-first workflow: standalone positions, position eligibility, max candidates, admin/student views.
- [`features/responsive-ui-usability.md`](features/responsive-ui-usability.md) — Mobile header/layout improvements, clearer ballot position settings, scroll behavior, and viewport audit status.
- [`features/landing-page.md`](features/landing-page.md) — ISSUE-31 public landing page with admin CMS, draft/publish separation, upcoming elections, SEO and social previews.
- [`features/voter-profile.md`](features/voter-profile.md) — issue #37 voter profile (age, sex, residence), administrator-managed residence areas, and the server-side ballot gate.
- [`features/restricted-elections.md`](features/restricted-elections.md) — issue #37 restricted elections: an imported roster as the whole electorate, visibility, and roster changes until close.
- [`features/election-clone.md`](features/election-clone.md) — issue #37 copying an election: what is copied, what is left behind, and who may do it.
- [`features/candidate-withdrawal.md`](features/candidate-withdrawal.md) — issue #37 withdrawal after approval, before the poll opens, with typed or uploaded evidence.
- [`features/live-election-edits.md`](features/live-election-edits.md) — issue #37 the title, description and end-time edits an open election accepts, and their audit.
- [`features/results-view-only.md`](features/results-view-only.md) — issue #37 voters read results on screen; downloadable ledgers stay with staff.
- [`features/certification.md`](features/certification.md) — issue #37 the creator and assigned admins certify, the stored certifier/timestamp, and the CERTIFIED snapshot every view reads.

## 6. Audit & Traceability

- [`traceability.md`](traceability.md) — SRS v1.1 → implementation traceability matrix (per-requirement status, code & test locations).
- [`audit/issue-00-audit.md`](audit/issue-00-audit.md) — ISSUE-00 full implementation audit: source of truth, stack, tables, seeds, routes, tests, re-scoped plan.

## 7. Templates & Quality

- [`debugging/BUG_REPORT_TEMPLATE.md`](debugging/BUG_REPORT_TEMPLATE.md) — Systematic defect investigation template.
- [`reviews/REVIEW_TEMPLATE.md`](reviews/REVIEW_TEMPLATE.md) — Structured code review checklist.

## 7. Testing & CI

- [`testing.md`](testing.md) — Test isolation, running, factories, clock, and known limitations.
- [`testing/ci-pipeline.md`](testing/ci-pipeline.md) — CI jobs, tool versions, local reproduction, diagnosis table, required checks.
