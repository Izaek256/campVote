# Technology Stack — MMU Campus E-Voting System

**Version:** 1.0  
**Date:** 24 October 2026  
**Status:** Proposed implementation baseline  
**Primary framework:** Grit Framework — https://gritframework.dev/

## 1. Stack Summary

| Layer | Selection | Purpose / notes |
|---|---|---|
| Full-stack framework | **Grit Framework** | Go-based meta-framework with Gin/GORM, React frontends, admin capabilities, code generation, and project tooling. Verify capabilities against the pinned CLI release. |
| Architecture | **Grit Triple: Web + Admin + API** | Separate public/voter web app, administrative CMS, and Go API in a monorepo. |
| Backend language | **Go** | API, election domain services, authorization, voting, tallying, reporting. |
| Backend HTTP framework | **Gin (through Grit)** | Routing and HTTP handling as provided by the framework. |
| ORM | **GORM (through Grit)** | Database access; use explicit transactions and reviewed constraints for vote acceptance. |
| Main public frontend | **Next.js (preferred Grit frontend option)** | Public election pages, voter flows, candidate profiles, results dashboard. Confirm against team preference and generated project. |
| Admin frontend | **Grit generated admin panel** | CMS, candidate review, eligibility configuration, ballot builder, reports, and certification workflows. Customize for election-specific permissions. |
| Frontend language | **TypeScript** | Frontend application and shared types. |
| Shared type/schema generation | **Grit `grit sync` / generated types and schemas** | Keep API models and frontend contracts aligned; review generated schemas for security-sensitive fields. |
| Production database | **PostgreSQL** | Authoritative relational data store. |
| Local infrastructure | **Docker + Docker Compose** | Reproducible development dependencies and containerized deployment packaging. |
| Cache/queue/realtime support | **Redis, if required by the selected Grit modules** | Use for features such as queueing/cache/realtime only when needed; PostgreSQL remains authoritative for votes. |
| Candidate image/file storage | **S3-compatible storage / MinIO locally, if needed** | Candidate photos and manifestos; do not store ballot secrecy material in public object storage. |
| Email | **Grit-supported transactional mail integration** | Election notices and official result notifications; Mailhog or equivalent for local testing. |
| Realtime results | **Grit-supported realtime/WebSocket facility if verified; otherwise polling/SSE fallback** | Publish aggregate-only data according to each election's result policy. |
| Containers | **Docker images + Compose** | Local setup and repeatable packaging. Production orchestrator/host selected with MMU IT. |
| Reverse proxy / TLS | **Caddy or university-approved reverse proxy** | HTTPS and routing; confirm deployment requirements. |
| Testing | Go tests, API/integration tests, frontend tests, browser E2E, load tests | Exact test libraries to be selected after scaffold inspection. |
| Source control | Git + hosted repository approved by project team | Versioning, code review, protected main branch, release tags. |
| CI/CD | Repository CI pipeline | Format, lint, tests, dependency checks, build, container scan, deployment approval. |
| Observability | Grit-supported monitoring plus structured logs/metrics | Do not log ballot choices or identity-to-choice links. Verify generated tools and access controls. |

## 2. Grit Project Mode

Use Grit's **Triple** architecture because the system requires both a public/voter experience and an administrative CMS, backed by a central API. The exact scaffold command should be verified with the installed version's `grit new --help` and current documentation. The general Grit docs describe Triple as a web app + admin panel + Go API.

Suggested project name: `mmu-campus-voting` (lowercase, hyphens only).

Do not assume generated folder paths, flags, auth behavior, or service names without inspecting the scaffold from the pinned release.

Official references:
- Grit docs: https://gritframework.dev/docs
- Quick start: https://gritframework.dev/docs/getting-started/quick-start
- Create a project: https://gritframework.dev/docs/getting-started/create-a-project

## 3. Core Services and Responsibilities

### 3.1 Public web app
- Public election listings and details.
- Voter sign-in entry point.
- Candidate profiles and manifestos.
- Eligibility-aware election access.
- Ballot completion, review, and submission.
- Generic submission confirmation.
- Results and turnout dashboard.
- Historical certified results.

The browser is never authoritative for eligibility, election timing, vote validity, or tallying.

### 3.2 Admin/CMS
- Election lifecycle and schedules.
- Candidate application and approval.
- Ballot builder and preview.
- Eligibility rules and roster imports.
- Results visibility settings.
- Reports, audit review, recount initiation, and certification.

Use the admin framework for normal resource management, but implement narrow custom workflows for sensitive operations. Disable generic CRUD operations for accepted ballots, participation records, tally internals, and certified results.

### 3.3 Go API/domain services
Recommended logical modules:
- `identity`: verified institutional identity mapping.
- `authorization`: role and permission checks.
- `eligibility`: election-specific voter and candidate checks.
- `elections`: lifecycle, schedules, publication policy.
- `candidates`: applications and public profiles.
- `ballots`: ballot definition/version validation.
- `voting`: submission and duplicate prevention.
- `privacy`: ballot/participation boundary and data minimization.
- `tally`: deterministic counting and reconciliation.
- `results`: provisional/certified snapshots and aggregate publication.
- `audit`: administrative events.
- `reports`: PDF/CSV generation.
- `notifications`: email/in-app delivery.
- `operations`: health checks, metrics, and operational status.

These are conceptual boundaries; use the actual structure and naming conventions produced by Grit.

## 4. Database and Data Rules

PostgreSQL is the production system of record. Apply:
- Version-controlled schema migrations.
- Foreign keys, unique constraints, and check constraints.
- Transactional vote acceptance.
- A unique participation constraint per voter/election where the chosen anonymity model supports that identifier.
- Explicit isolation/concurrency testing.
- Least-privilege database accounts.
- Encrypted, access-restricted backups.
- No generic CRUD mutation of accepted ballots or certified result snapshots.

### Ballot secrecy
Identity, eligibility, participation, and ballot selections are separate logical data domains. Do not store a voter ID directly on a ballot-choice record. A simple table split is not a complete anonymity guarantee if timestamps, tokens, logs, or backups allow correlation. Complete a threat model before production; adopt anonymous credentials or a cryptographic voting design if the required guarantee demands it.

### Tallying
Tallying must be deterministic and tested using independent known-answer datasets. Store the rule/ballot version used for each result snapshot. Recounts recompute results without editing original ballots. Publish only aggregates permitted by the election's result-visibility policy.

## 5. Docker and Supporting Services

Use Docker Compose for local development. Grit documentation indicates the scaffold may include PostgreSQL, Redis, MinIO, and Mailhog; inspect the actual generated file and remove services not needed.

| Service | Local use | Production guidance |
|---|---|---|
| Go API | Application backend | Container or Grit-supported deployment |
| Public web app | Voter/public interface | Container/static output according to chosen Grit architecture |
| Admin app | Election CMS | Restrict access by identity and role |
| PostgreSQL | Main database | Private network, persistent storage, backup/restore drills |
| Redis | Optional cache/queue/realtime support | Private network; not authoritative vote storage |
| MinIO | Optional local S3-compatible storage | Use approved private object storage in production if needed |
| Mailhog | Local email preview | Do not use as production mail service |
| Reverse proxy | Optional local proxy | TLS termination and security headers in production |

Rules:
- Do not expose database or Redis ports publicly by default.
- Use non-default credentials.
- Keep secrets out of Git.
- Pin production image versions.
- Use health checks and persistent volumes where appropriate.
- Test backup restoration, not just backup creation.
- Keep production and development data/credentials separate.

## 6. Authentication and Institutional Directory

Preferred integration: MMU-approved SSO/OIDC or trusted institutional identity service. Confirm with university IT:
- Exact student domain (`@std.must.ac.ug`) and staff/official domain (`@must.ac.ug`).
- Whether SSO/OIDC is available.
- How current enrollment/employment is checked.
- Which attributes are available (faculty, department, programme, year, class, role).
- How status changes are synchronized.
- Who owns roster data and resolves eligibility disputes.

Fallback: controlled CSV roster imports with validation, duplicate detection, preview, audit records, access restrictions, and a defined update procedure.

Domain validation alone is not sufficient to prove current enrollment or eligibility.

## 7. Realtime and Results Publication

Prefer Grit's supported realtime service only after confirming its behavior in the pinned version. Otherwise, start with periodic polling or Server-Sent Events; a simple reliable mechanism is better than an unverified realtime integration.

Realtime events should contain only aggregate values, for example:
- Election ID or public slug.
- Position ID.
- Aggregate candidate totals and percentages, if the policy allows them.
- Aggregate turnout.
- Result status/version and update time.

Do not publish voter IDs, individual ballot IDs linked to people, exact individual vote events, or ballot selections when those could expose identity or choices.

Each election chooses one policy:
1. Live provisional results.
2. Results after closing.
3. Results after certification.

The UI must label provisional leaders clearly and must not declare a winner before the election's rules permit it.

## 8. Quality and Testing Stack

Minimum required testing layers:
1. **Go unit tests:** eligibility rules, ballot validation, tally methods, tie handling, percentages.
2. **Database integration tests:** migrations, constraints, transaction rollback, duplicate submissions.
3. **API tests:** authentication, role permissions, election states, invalid requests, result visibility.
4. **Frontend tests:** ballot usability, selection limits, errors, submission state, accessible controls.
5. **End-to-end tests:** voter sign-in through ballot submission; admin election setup; closure and certification.
6. **Concurrency tests:** simultaneous submissions for the same voter/election.
7. **Property/boundary tests:** tallying, malformed ballots, empty/large inputs, ties, rounding.
8. **Privacy/security tests:** API/schema/log inspection for identity-to-ballot links; privilege escalation; object-level authorization.
9. **Load tests:** expected concurrent guild-election traffic and realtime dashboard traffic.
10. **Recovery tests:** restore backups and reconcile ballots/results.

Acceptance gates must be automated where practical and run in CI. A green build alone does not establish election readiness.

## 9. CI/CD and Repository Policy

Recommended pipeline:
1. Format and lint Go/TypeScript.
2. Run unit tests.
3. Start PostgreSQL and any required dependencies.
4. Apply migrations to a clean database.
5. Run integration and API tests.
6. Run frontend/build checks.
7. Run security/dependency checks.
8. Build production artifacts/containers.
9. Require review/approval before production deployment.
10. Run post-deployment health checks.

Repository policy:
- Protect the main branch.
- Require pull requests and review for sensitive voting changes.
- Use short-lived feature branches.
- Tag approved releases.
- Never commit `.env` files, production database dumps, private voter rosters, secrets, or live election data.
- Review Grit-generated diffs after every framework upgrade.

## 10. Environment Configuration

Use separate configuration for local, test, staging, and production environments. Exact variable names must follow the generated Grit version and should be documented in `.env.example` without real secrets.

Expected configuration categories:
- Database connection.
- Session/JWT/cookie settings.
- OIDC/SSO credentials and callback URLs.
- Email provider credentials.
- Object storage settings, if used.
- Redis settings, if used.
- Public web/admin/API URLs.
- CORS and trusted origins.
- TLS/reverse proxy settings.
- Logging and monitoring.
- Election timezone and operational policies.

Never put production secrets in source control or frontend environment variables that are bundled into client assets.

## 11. Version Policy

- Pin the Grit CLI/framework release used to scaffold the project.
- Record Go, Node.js, package manager, PostgreSQL, Docker image, and frontend versions.
- Commit package lockfiles and Go module files.
- Upgrade dependencies intentionally, with tests and review.
- Check Grit's current official docs and changelog before changing versions.
- Do not rely on unpinned `latest` dependencies in production.

Grit's documentation changes over time; the scaffold and the docs matching the installed CLI version are the source of truth for exact commands and generated paths.

## 12. Suggested Initial Setup Workflow

1. Install the prerequisites listed by the current Grit documentation.
2. Install and verify the Grit CLI.
3. Run `grit new --help` and confirm Triple/Next.js options for the installed version.
4. Scaffold `mmu-campus-voting` in Triple mode.
5. Review generated authentication, RBAC, database, migration, Docker Compose, and admin features.
6. Configure PostgreSQL and run the framework's documented migration workflow.
7. Remove or disable unused infrastructure services.
8. Add domain models and tests in small vertical slices.
9. Implement election lifecycle and candidate management first.
10. Implement eligibility and ballot schema validation.
11. Complete the privacy threat model before implementing production ballot storage.
12. Implement atomic submission and duplicate prevention.
13. Implement deterministic tallying with known-answer tests.
14. Add aggregate-only dashboard updates and result policy.
15. Run security, load, and recovery tests before a pilot election.

## 13. Official Documentation
- Grit Framework: https://gritframework.dev/
- Grit documentation: https://gritframework.dev/docs
- Quick start: https://gritframework.dev/docs/getting-started/quick-start
- Create a project: https://gritframework.dev/docs/getting-started/create-a-project
- Grit source repository referenced by its docs: https://github.com/MUKE-coder/grit
