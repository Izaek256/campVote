# Documentation Index — campVote

## 1. Baseline Specifications

- [`../SRS.md`](../SRS.md) — System Requirements Specification (functional, security, privacy).
- [`../STACK.md`](../STACK.md) — Technology stack specification (Grit Triple, Go, Next.js, PostgreSQL).
- [`../ADR.md`](../ADR.md) — Architecture Decision Records (ADR-001 through ADR-014).
- [`../AGENTS.md`](../AGENTS.md) — Non-negotiable AI agent development rules.
- [`../CONTRIBUTING.md`](../CONTRIBUTING.md) — Git workflow, branch policy, commit conventions.

## 2. Requirements & Progress Tracking

- [`progress/requirements-tracker.md`](progress/requirements-tracker.md) — Requirement-to-implementation status matrix.
- [`progress/development-log.md`](progress/development-log.md) — Chronological log of changes, checks, and next tasks.
- [`handover/current-state.md`](handover/current-state.md) — Current state of the system for developers and agents.

## 3. Implementation Records

- [`auth.md`](auth.md) — ISSUE-02 authentication, session, lockout, CSRF and first-admin bootstrap status with test evidence.
- [`rbac.md`](rbac.md) — ISSUE-03 server-side RBAC: SRS role matrix, route policies, per-election scoping, results visibility, certification and test evidence.
- [`implementation/ROADMAP.md`](implementation/ROADMAP.md) — Phased milestone delivery order.
- [`implementation/OPEN_QUESTIONS.md`](implementation/OPEN_QUESTIONS.md) — Unresolved domain and institutional questions.
- [`implementation/SETUP_LOG.md`](implementation/SETUP_LOG.md) — Verified toolchain versions and scaffold facts.
- [`maintenance/repository-cleanup.md`](maintenance/repository-cleanup.md) — Repository audit and file hygiene log.

## 4. Audit & Traceability

- [`traceability.md`](traceability.md) — SRS v1.1 → implementation traceability matrix (per-requirement status, code & test locations).
- [`audit/issue-00-audit.md`](audit/issue-00-audit.md) — ISSUE-00 full implementation audit: source of truth, stack, tables, seeds, routes, tests, re-scoped plan.

## 5. Templates & Quality

- [`debugging/BUG_REPORT_TEMPLATE.md`](debugging/BUG_REPORT_TEMPLATE.md) — Systematic defect investigation template.
- [`reviews/REVIEW_TEMPLATE.md`](reviews/REVIEW_TEMPLATE.md) — Structured code review checklist.
