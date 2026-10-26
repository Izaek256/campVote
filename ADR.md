# Architecture Decision Records (ADR)
## MMU Campus E-Voting System

| Field | Value |
|---|---|
| Project | MMU Campus E-Voting System |
| Version | 1.0 |
| Date | 24 October 2026 |
| Status | Proposed decisions for implementation baseline |
| Decision owners | Project team and authorized MMU stakeholders |

This document records architectural decisions and their rationale. Decisions marked **Accepted for project planning** establish the intended direction, but framework-specific assumptions must be verified against the actual Grit version scaffolded for the repository. University policy decisions remain subject to stakeholder approval.

## ADR-001 — Use Grit Framework as the Application Foundation

**Status:** Accepted for project planning

### Context
The system needs a backend, public web experience, integrated admin/CMS, authentication and role controls, shared types, database integration, and a maintainable structure. Building all infrastructure manually would consume time and introduce avoidable inconsistencies.

### Decision
Use [Grit Framework](https://gritframework.dev/) as the full-stack application foundation. Grit describes itself as a Go-based framework using Gin and GORM, with React frontend options, code generation, an admin panel, authentication/RBAC, background jobs, and Docker-oriented project scaffolding.

Use Grit's generated code as a starting point, not as proof that the election-specific security model is complete. Election logic, secret-ballot design, eligibility, transactional duplicate prevention, tallying, and certification must be deliberately implemented and tested.

### Rationale
- Integrated Go API, frontend, admin, and shared type-generation workflow.
- Resource generation can reduce repetitive CRUD work for candidates, elections, positions, and reports.
- Go's concurrency model is suitable for handling concurrent API traffic.
- PostgreSQL is supported for production according to Grit's documentation.
- Self-hosting and Docker deployment are compatible with university-controlled operations.

### Consequences
- Developers must learn the framework's conventions and Go/GORM patterns.
- Generated CRUD endpoints must be reviewed and restricted; generic CRUD must never expose direct mutation of ballots, tallies, or certified results.
- Framework upgrades must be tested against custom security-sensitive code.
- Exact commands, generated folders, and available features must be checked against the installed CLI version.

### Alternatives considered
- Build a custom Go/Gin application without Grit: more control but more initial infrastructure work.
- Django/DRF: mature but not the chosen stack.
- Next.js-only backend: not selected because the project has chosen Grit and its Go API.

### Verification
Scaffold a minimal project, inspect authentication, authorization, database migrations, API error handling, admin permissions, and tests before implementing election-specific resources.

---

## ADR-002 — Select the Triple Architecture: Web + Admin + Go API

**Status:** Accepted for project planning

### Context
The application has two distinct interaction areas: public/voter-facing election pages and an administrative CMS. Election operations should be governed by a backend API rather than frontend-only rules.

### Decision
Use Grit's **Triple** architecture: separate web frontend, admin frontend, and Go API in one repository/monorepo. Prefer Grit's Next.js frontend option for the public web application unless the team explicitly chooses TanStack Router/Vite after a prototype. Keep the backend authoritative for identity, permissions, eligibility, schedules, vote acceptance, and results.

Expected conceptual components:
- Public/voter web application.
- Grit admin panel, customized for election workflows.
- Go API.
- Shared generated TypeScript types/schemas.
- PostgreSQL and supporting services through Docker Compose.

### Rationale
- Clear separation between public and administrative UI.
- Admin access can use a narrower, role-specific interface.
- Public results can be served through safe aggregate endpoints.
- The API can enforce identical election rules regardless of client.

### Consequences
- More deployment components than a single-binary architecture.
- Requires coordinated frontend/API configuration and CORS/cookie policy.
- Must prevent admin-only data from being bundled into public clients.
- A separate frontend is not a security boundary; backend authorization remains mandatory.

### Alternatives considered
- Single architecture: simpler deployment but less separation between public app and admin UI.
- Double architecture: suitable if admin is built entirely inside the main web application; not preferred because the project explicitly requires a full CMS/dashboard.
- API-only: no built-in end-user web experience.

### Verification
Confirm the selected Grit CLI version supports the chosen flags/options and inspect the generated repo before writing custom features. Avoid guessing command flags; use `grit new --help` and the version-matched docs.

---

## ADR-003 — Use PostgreSQL as the Production Database

**Status:** Accepted

### Context
Election records require transactional consistency, uniqueness constraints, durable storage, reliable backups, and support for concurrent submissions and aggregate queries.

### Decision
Use PostgreSQL as the authoritative production database. Use the database features supported by the selected Grit/GORM version. Local developer infrastructure will run PostgreSQL through Docker Compose.

Do not use SQLite as the production system of record for the hosted multi-user election service. Grit may use SQLite for certain development, test, or desktop modes; this project should explicitly configure PostgreSQL for its main application environment.

### Rationale
- Strong transaction and constraint support.
- Unique indexes can help enforce one participation record per user/election.
- Good tooling for backup, restore, monitoring, and concurrency.
- Widely deployable in university-controlled infrastructure.

### Consequences
- Requires database provisioning, backups, migrations, monitoring, and credential management.
- Production credentials and data must never be copied into development.
- Transactions and isolation behavior must be tested under concurrent ballot submissions.

### Required database safeguards
- Unique constraint on the applicable participation key, e.g. `(election_id, voter_subject_id)` in the participation subsystem.
- Foreign keys and check constraints where appropriate.
- Transactions around ballot acceptance and participation recording.
- Least-privilege database roles.
- No generic admin CRUD for ballot contents or certified results.
- Backups encrypted and access-restricted.
- Migration files reviewed and version-controlled.

**Privacy caveat:** A unique participation key prevents duplicate voting but creates a record of who participated. It must not be joined to ballot choices. If the threat model requires resistance even to privileged operators, use a stronger anonymous-token or cryptographic design rather than relying on table separation alone.

### Alternatives considered
- SQLite: not selected as production database for a central concurrent service.
- MySQL: capable, but PostgreSQL is the selected project database.
- Managed PostgreSQL: acceptable later if university policy permits; it does not change the logical architecture.

### Verification
Test simultaneous submissions, transaction rollback, uniqueness conflicts, migration from an empty database, backup restoration, and tally reproducibility.

---

## ADR-004 — Use Docker and Docker Compose for Local Infrastructure and Deployment Packaging

**Status:** Accepted

### Context
Developers need repeatable environments, and deployment should reduce differences between local testing and production.

### Decision
Use Docker Compose for local development dependencies and a documented containerized deployment path. The Grit scaffold may include PostgreSQL, Redis, MinIO, and Mailhog; retain only services the application actually uses and confirm the generated Compose file for the chosen version.

Expected local services:
- PostgreSQL.
- Redis if required by the selected Grit features for cache, queues, rate limiting, or realtime.
- MinIO only if S3-compatible object storage is needed for candidate images/manifests.
- Mailhog or equivalent only for development email testing.
- Application containers if the team elects to run the full stack in Compose.

### Rationale
- Reproducible local setup.
- Easier onboarding and integration testing.
- Clear separation of services and configuration.
- Supports deployment on a university server or compatible VPS.

### Consequences
- Team members need Docker Engine/Desktop and sufficient resources.
- Compose is not by itself a complete production security, backup, or orchestration solution.
- Production volumes, networks, secrets, TLS, monitoring, and restore procedures must be configured separately.

### Security rules
- Never publish PostgreSQL/Redis ports publicly without a deliberate access requirement.
- Use non-default production credentials.
- Use health checks and persistent volumes where appropriate.
- Pin image versions; avoid unreviewed `latest` tags in production.
- Keep secrets out of committed Compose files.
- Do not assume local Mailhog or MinIO settings are production-ready.

### Alternatives considered
- Manual host installation: more environment drift.
- Kubernetes: unnecessary complexity for the initial campus deployment unless institutional operations require it.

### Verification
A new developer should be able to follow the README and start a clean environment, run migrations, seed development data, run tests, and stop the services without touching production.

---

## ADR-005 — Institutional Identity and Eligibility Must Be Separate

**Status:** Accepted; university integration details pending

### Context
The requested email domains are `@std.must.ac.ug` for students and `@must.ac.ug` for staff/officials. Domain checks alone do not establish current enrollment or class/faculty membership.

### Decision
Prefer university-controlled SSO/OIDC or a trusted identity source. Use verified identity claims for sign-in and an authoritative student/staff directory or controlled roster for election eligibility. If no integration exists, use an administrator-approved CSV roster process with validation, audit, and documented refresh rules.

### Rationale
- Prevents inactive or incorrectly classified accounts from voting.
- Supports faculty, department, class, year, and staff-only elections.
- Keeps authentication distinct from authorization to vote in a particular election.

### Consequences
- Requires agreement with the university's IT/registry stakeholders.
- Eligibility data must have owners, refresh schedules, and a dispute/correction procedure.
- Domain policy must be configurable, with exact domains confirmed before go-live.

### Required behavior
1. Verify identity with a trusted mechanism.
2. Determine the institutional class and relevant attributes.
3. Evaluate eligibility for the selected election.
4. Recheck eligibility at submission.
5. Audit eligibility changes without exposing ballot choices.

### Alternatives considered
- Domain suffix alone: rejected as insufficient.
- Manual voter list only: acceptable fallback but more operational work and higher risk of stale data.

### Verification
Test approved/inactive accounts, wrong domains, mismatched faculties, stale rosters, duplicates, and changes during an election.

---

## ADR-006 — Secret Ballot Design Is a Core Domain Boundary

**Status:** Accepted; exact cryptographic design requires threat-model review

### Context
The key requirement is that a candidate cannot know who voted for them. The system must also reduce the risk of administrators or infrastructure operators linking identities to selections.

### Decision
Separate identity/authentication, eligibility, participation, and ballot contents. Do not store voter identity directly on ballot records. Minimize metadata and avoid logging ballot selections. Ordinary admin screens and APIs must not provide a voter-to-choice lookup.

Before production, complete a privacy threat model that considers privileged database access, application logs, timestamps, backups, unique tokens, and correlation across services. If the required threat model includes malicious privileged operators, adopt an anonymous credential/token or cryptographically verifiable voting design reviewed by an independent expert.

### Rationale
- Ballot secrecy must be built into data flow and permissions from the start.
- UI hiding alone does not protect against database or log correlation.
- Election confidence requires a clear, honest statement of the actual privacy guarantee.

### Consequences
- More design and testing than ordinary form submission.
- Participation control and ballot acceptance must be coordinated without exposing a direct identity-to-choice link.
- Some stronger anonymity approaches require cryptographic expertise and independent review.
- Do not claim “cryptographically anonymous” or “end-to-end verifiable” unless that property is demonstrated by the implemented design.

### Design constraints
- No `voter_id` foreign key on a ballot-choice record.
- Participation records contain no selections.
- Ballot receipt must not prove a choice to a coercer or candidate.
- Avoid detailed per-vote timestamps in interfaces and general-purpose logs.
- Restrict access to backups and database snapshots.
- Define data retention and deletion policies before go-live.
- Test the privacy boundary with an adversarial review.

### Alternatives considered
- Store voter ID with selections and hide it in the UI: rejected.
- Separate tables but leave joinable timestamps/tokens/logs: insufficient without threat analysis.
- Anonymous credentials/cryptographic ballot system: recommended if higher assurance is required; implementation choice is pending expert review.

### Verification
Document the threat model and data flows; perform code, schema, log, permission, and backup inspection; verify that standard administrative access cannot identify individual choices.

---

## ADR-007 — Server-Side Atomic Vote Acceptance and Idempotency

**Status:** Accepted

### Context
Users may double-click, retry after a timeout, or submit concurrently from multiple tabs/devices. Client-side disabling is not sufficient.

### Decision
Make the backend the authority for vote acceptance. Validate election state, eligibility, ballot schema, and participation state on submission. Use a database transaction and uniqueness/consistency constraints so the ballot is accepted once or not at all. Provide a safe submission-status check for uncertain network responses.

### Rationale
- Prevents duplicate acceptance and partial state.
- Handles race conditions.
- Makes retries safer.
- Enables auditable and repeatable testing.

### Consequences
- Conflict responses must be handled clearly in the UI.
- The exact transaction pattern must be validated against GORM/PostgreSQL behavior.
- Do not assume a generic CRUD generator implements these invariants automatically.

### Verification
Run concurrent duplicate requests and fault-injection tests. Assert exactly one accepted ballot and consistent participation state. Verify rollback on every failure point.

---

## ADR-008 — Deterministic Tallying, Reconciliation, and Certification

**Status:** Accepted

### Context
Automatic arithmetic is not enough to guarantee a correct election. Tallying bugs, incorrect denominators, configuration changes, and hidden data corruption can produce incorrect results.

### Decision
Implement tallying as a deterministic domain service that consumes accepted ballots and a versioned ballot/rule definition. Maintain independent test vectors with expected results, reconciliation reports, reproducible recounts, and an explicit certification workflow.

### Rationale
- Same inputs and rules should produce the same outputs.
- Independent tests reduce the risk of a shared implementation error.
- Certification distinguishes provisional calculations from official outcomes.
- Recounts should not modify original ballots.

### Consequences
- Each supported ballot type needs formal counting rules and tests.
- Ranked-choice voting requires additional documented elimination and tie rules.
- Reports must identify whether results are provisional, under review, or certified.

### Verification
Test normal, empty, malformed, boundary, tied, multi-position, abstention, and high-volume cases. Compare result snapshots and exports. Recount the same ballots and verify identical output.

---

## ADR-009 — Live Results Are Configurable per Election

**Status:** Accepted

### Context
The requested dashboard should show who is leading and by what percentage in real time. Publishing leaders during an active election may influence voters who have not yet voted.

### Decision
Provide an election-level result publication policy:
1. Live provisional results during voting.
2. Results after polls close.
3. Results after certification.

The selected policy must be displayed in the election configuration and enforced by the API. Live data must be aggregate-only.

### Rationale
- Meets transparency requirements while allowing election officials to avoid strategic influence where necessary.
- Makes policy explicit rather than embedding a single behavior in code.
- Keeps provisional leader status distinct from winner/certified status.

### Consequences
- Realtime channels must authorize access and publish only permitted aggregates.
- The public interface must label provisional data and display the percentage denominator.
- A current leader is not necessarily the final winner.

### Verification
Test each visibility mode, unauthorized access, realtime updates, closure, certification, and absence of personal/ballot data in published events.

---

## ADR-010 — Use Grit's Generated Resources Carefully; Domain Logic Owns Election Invariants

**Status:** Accepted

### Context
Grit resource generation can create models, API handlers, shared schemas, hooks, and admin screens quickly. Generic CRUD is unsafe for integrity-sensitive election records.

### Decision
Use generated resources for low-risk administrative entities where suitable, such as election metadata, candidate profiles, announcements, and report configuration. Implement custom domain services and narrowly scoped endpoints for eligibility decisions, ballot acceptance, tallying, recounts, and certification. Disable generic create/update/delete endpoints for accepted ballots and certified result records.

### Rationale
- Preserves Grit's productivity benefits without treating generated CRUD as election logic.
- Centralizes invariants in tested backend services.
- Reduces the chance that a future admin form bypasses voting rules.

### Consequences
- Some generated files/resources will need deliberate customization.
- Framework upgrades and code generation must be reviewed for permission regressions.
- Domain-level tests are required even when CRUD endpoints are generated.

### Verification
Review route registration and resource permissions. Confirm no generic endpoint can edit accepted ballots, participation records, tally output, or certified snapshots.

---

## ADR-011 — Realtime Updates Use Aggregate-Only Events

**Status:** Proposed; confirm the Grit realtime facilities in the pinned version

### Context
The dashboard needs updates without manual reloads, but public realtime events must not leak identities or ballot contents.

### Decision
Use Grit's supported realtime/WebSocket mechanism if available and suitable in the selected version; otherwise use polling or server-sent events as a simpler fallback. Publish aggregate result/turnout events only, subject to the election's result-visibility policy. Do not publish a per-vote event containing candidate selection and precise voter-related metadata.

### Rationale
- Enables timely dashboard refreshes.
- Minimizes sensitive data in event streams.
- Allows a safe fallback if realtime integration complicates initial delivery.

### Consequences
- Realtime access needs authentication/authorization and origin controls where relevant.
- Events should be aggregated or batched where appropriate.
- Reconnect behavior must not expose data before the viewer is authorized.

### Verification
Inspect payloads, test authorization and reconnection, and confirm events contain no voter identifiers, ballot IDs linked to identities, or individual choices when prohibited.

---

## ADR-012 — Deployment, Secrets, Backups, and Operations

**Status:** Accepted

### Context
An election platform needs dependable operations during scheduled, high-attention events.

### Decision
Use environment-specific configuration, Docker-based deployment packaging, TLS termination, restricted service networking, managed secrets, monitoring, and scheduled encrypted backups. Select the actual hosting target with university IT; initial deployment should be self-hostable on an approved VPS or university server.

### Rationale
- Repeatable deployments.
- Avoids dependence on a single frontend hosting vendor.
- Supports university control over election records.
- Makes recovery procedures testable.

### Consequences
- Operations ownership must be assigned.
- Production database access and backup access must be restricted.
- Backups and restores require scheduled drills.
- Docker alone does not provide full production orchestration or security.

### Verification
Test clean deployment, TLS, least-privilege accounts, backup restoration, monitoring alerts, and documented incident recovery.

---

## Decision Review Triggers
Revisit these decisions if:
- MMU mandates a specific identity provider or hosting environment.
- The selected Grit release changes architecture or security behavior.
- A privacy assessment requires cryptographic anonymity beyond data separation.
- Load tests show the selected deployment cannot handle expected voting traffic.
- University policy changes result publication, retention, certification, or dispute requirements.
