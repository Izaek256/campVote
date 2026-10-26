# Implementation roadmap

This is a proposed order, not a claim that any phase is implemented. Track work as GitHub issues and update this document when priorities change.

## Phase 0 — Verify the foundation

- [ ] Confirm the current repository state, branch protections, and required CI checks.
- [ ] Pin Grit CLI and tool versions; inspect `grit new --help` and official docs for the installed version.
- [ ] Generate the Triple scaffold in a temporary directory and inspect generated apps, auth, RBAC, database, migrations, and Compose services.
- [ ] Merge baseline docs carefully; document real setup commands and workspace paths.
- [ ] Confirm local boot, clean database migration, and basic health checks.

**Gate:** another developer can clone the repo and follow the recorded setup without guessing.

## Phase 1 — Domain model and access control

- [ ] Define users, institutional identity references, roles, elections, positions, candidates, ballot definitions/versions, participation records, and audit events.
- [ ] Write migration and authorization tests before exposing admin CRUD.
- [ ] Implement role and election-scoped permissions on the server.
- [ ] Define synthetic seed data and repeatable test fixtures.

**Gate:** permission tests prove unauthorized users cannot administer elections or access restricted records.

## Phase 2 — Election CMS

- [ ] Create/edit/draft/schedule/open/close election lifecycle.
- [ ] Manage positions, candidate applications/approval, and ballot choices.
- [ ] Configure eligibility attributes and result visibility policy.
- [ ] Validate lifecycle transitions and record administrative actions.

**Gate:** invalid state transitions and unauthorized actions are rejected and tested.

## Phase 3 — Eligibility and voting

- [ ] Confirm the approved institutional identity source or documented roster-import fallback.
- [ ] Implement election-specific eligibility checks using verified attributes.
- [ ] Validate ballot version, selection limits, election state, and candidate/option validity server-side.
- [ ] Enforce one accepted ballot per eligible voter/election with transactional logic and database constraints.
- [ ] Separate participation records from ballot selections and review timestamp/token/log correlation risks.
- [ ] Test duplicate and concurrent submissions, including rollback behavior.

**Gate:** privacy and duplicate-vote tests pass against a clean PostgreSQL database.

## Phase 4 — Tally and results

- [ ] Implement deterministic tallying with documented tie/rounding rules.
- [ ] Verify tally outputs using independent known-answer datasets.
- [ ] Publish only aggregate information permitted by the election policy.
- [ ] Version and audit provisional/final result snapshots; test recount reproducibility.

**Gate:** tally and publication policy tests pass; no identifying ballot information is exposed.

## Phase 5 — UX and notifications

- [ ] Build responsive voter flows with accessible selection/review/confirmation states.
- [ ] Build admin workflows and clear validation/error states.
- [ ] Add email or realtime functionality only when needed and verified against the scaffold.
- [ ] Add browser end-to-end tests for critical flows.

## Phase 6 — Security, operations, and handover

- [ ] Threat-model authentication, eligibility, ballot secrecy, insider access, backups, logs, and recovery.
- [ ] Add dependency/security checks and appropriate rate limits.
- [ ] Test backups/restores and operational health checks.
- [ ] Run realistic load/concurrency tests and document limitations.
- [ ] Prepare demo data, user guide, known limitations, deployment checklist, and final handover.

**Release constraint:** this project is an academic prototype unless separately reviewed, approved, and independently assessed for real-election use.
