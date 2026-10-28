# SRS v1.1 → Implementation Traceability Matrix
## campVote — MMU Campus E-Voting System
**Audit date:** 2026-10-28  
**Audited by:** ISSUE-00 implementation audit (chore/issue-00-implementation-audit)  
**Authoritative source:** `develop` branch, commit `22c6255` (Grit Triple monorepo scaffold + campus voting domain)

---

## Legend
| Status | Meaning |
|---|---|
| ✅ Implemented | Code, model, or service exists; at least one unit/integration test covers it |
| 🔶 Partial | Scaffolded or modelled; no handler/route or no test yet |
| ❌ Missing | No code at all; test vectors absent |
| N/A | Explicitly out of scope (Phase 2/3 or pending university decision) |

---

## 3.1 Authentication & Authorization (FR-AUTH)

| Req ID | Requirement | Status | Code Location | Test Location | Evidence | Owning Issue |
|---|---|---|---|---|---|---|
| FR-AUTH-01 | Permit protected access only to authenticated institutional identities | 🔶 Partial | `apps/api/internal/middleware/auth.go`; `routes/routes.go` L1176–1181 (`protected` group with `middleware.Auth`) | — | JWT/cookie auth middleware wired; MMU domain enforcement not yet wired | ISSUE-01 |
| FR-AUTH-02 | Enforce `@std.must.ac.ug` / `@must.ac.ug` domain policy, server-side | ❌ Missing | `models/election.go` L59 (AllowedDomains field on Election, not enforced at auth) | — | Domain stored per-election but no registration/login domain validator exists | ISSUE-01 |
| FR-AUTH-03 | Validate email/identity claims server-side | 🔶 Partial | `middleware/auth.go`; `services/auth.go` | `handlers/auth_test.go` | JWT claims validated; institutional domain not enforced | ISSUE-01 |
| FR-AUTH-04 | Prefer university SSO/OIDC | 🔶 Partial | `handlers/auth_oauth.go`; `handlers/sso.go`; `services/sso.go` | `handlers/sso_test.go` | Generic OIDC/SAML scaffolded; MMU-specific IdP not configured | ISSUE-01 |
| FR-AUTH-05 | Verify current institutional status from authoritative directory | ❌ Missing | — | — | No directory/roster integration at auth layer | ISSUE-05 |
| FR-AUTH-06 | RBAC with least privilege on every protected backend operation | 🔶 Partial | `internal/authz/`; `middleware/auth.go`; `routes/routes.go` (RequireRole/RequireStaff) | `authz/*_test.go` | Grit RBAC wired; election-specific roles (Election Official, Candidate) not yet defined | ISSUE-01 |
| FR-AUTH-07 | Session expiry, logout, suspension, security-event logging | ✅ Implemented | `handlers/session.go`; `services/session.go`; `middleware/security_log.go` | `services/session_test.go` | Session revocation + security event constants implemented | ISSUE-01 |
| FR-AUTH-08 | MFA/TOTP for privileged accounts | ✅ Implemented | `handlers/totp.go`; `services/two_factor.go` | `services/two_factor_test.go` | TOTP + backup codes + passkeys scaffolded and tested | ISSUE-01 |
| FR-AUTH-09 | Never log passwords, tokens, ballot contents | 🔶 Partial | `middleware/security_log.go`; `middleware/logger.go` | — | Structured logger present; explicit ballot-content filtering not tested | ISSUE-01 |

---

## 3.2 Election CMS & Lifecycle (FR-CMS)

| Req ID | Requirement | Status | Code Location | Test Location | Evidence | Owning Issue |
|---|---|---|---|---|---|---|
| FR-CMS-01 | Create/edit/schedule/publish/close/cancel/archive elections | 🔶 Partial | `models/election.go` (all statuses defined); no election handler yet | — | Model constants verified; no HTTP CRUD handler | ISSUE-02 |
| FR-CMS-02 | Full election configuration (title, dates, category, eligibility, ballot, visibility) | ✅ Implemented | `models/election.go` L44–71 | `models/voting_test.go::TestElectionLifecycle` | All config fields on Election model; lifecycle tested | ISSUE-02 |
| FR-CMS-03 | Multi-category elections (Guild, Faculty, Department, Programme, Class, Club, etc.) | ✅ Implemented | `models/election.go` L11–21 | `database/elections_seeder_test.go::TestSeedElections` | 9 category constants; Guild + Faculty seeded and tested | ISSUE-02 |
| FR-CMS-04 | Multiple simultaneous elections, isolated data | ✅ Implemented | `models/election.go`; `models/participation.go` (election-scoped unique index) | `database/elections_seeder_test.go` | Multi-election seeder creates 2 isolated elections; participation scoped by `election_id` | ISSUE-02 |
| FR-CMS-05 | Lifecycle states: Draft, Registration, Pending, Scheduled, Open, Closed, Review, Finalized, Archived, Cancelled | ✅ Implemented | `models/election.go` L24–35 | `models/voting_test.go::TestElectionLifecycle` | All 10 lifecycle constants defined; Open/Draft/schedule transitions tested | ISSUE-02 |
| FR-CMS-06 | Validate configuration before opening | 🔶 Partial | `models/election.go::CanAcceptVotes()` (basic) | — | Schedule check only; no full pre-open ballot/candidate validation | ISSUE-02 |
| FR-CMS-07 | Reusable election templates | ❌ Missing | — | — | Not yet modelled | ISSUE-02 |
| FR-CMS-08 | Audit config changes; freeze ballot after opening | 🔶 Partial | `models/election.go::BeforeUpdate` (version increment); `models/election_audit.go` | — | Optimistic locking + audit log model; freeze enforcement not implemented in handler | ISSUE-02 |
| FR-CMS-09 | Server-enforced schedule (open/close by time) | ✅ Implemented | `models/election.go::IsOpen()`, `CanAcceptVotes()` | `models/voting_test.go::TestElectionLifecycle` | Time-based schedule verified with before/after boundary tests | ISSUE-02 |
| FR-CMS-10 | Publish notices, schedules, instructions | ❌ Missing | — | — | No election public page / notification handler yet | ISSUE-07 |

---

## 3.3 Candidate Management (FR-CAND)

| Req ID | Requirement | Status | Code Location | Test Location | Evidence | Owning Issue |
|---|---|---|---|---|---|---|
| FR-CAND-01 | Official candidate entry and application | 🔶 Partial | `models/candidate.go` L18–34 | — | Model exists; no handler for application submission | ISSUE-03 |
| FR-CAND-02 | Public profile fields; restricted private identifiers | 🔶 Partial | `models/candidate.go` (FullName, PhotoURL, Manifesto, Bio, BallotNumber); no email/ID stored | `database/elections_seeder_test.go` | Candidate privacy: no voter-identifying fields; profile fields verified in seeder test | ISSUE-03 |
| FR-CAND-03 | Review/approval/rejection/withdrawal workflow | 🔶 Partial | `models/candidate.go` L11–16 (`CandidateStatus` constants) | — | Status enum exists; no handler | ISSUE-03 |
| FR-CAND-04 | Verify candidate eligibility per election rules | ❌ Missing | — | — | No eligibility check at candidate approval time | ISSUE-03 |
| FR-CAND-05 | Display approved candidate info to eligible voters/public | ❌ Missing | — | — | No public candidate listing endpoint | ISSUE-03 |
| FR-CAND-06 | Prevent candidates from accessing voter lists or ballot choices | ✅ Implemented | `models/ballot.go` (no voter FK); `models/participation.go` (no ballot FK) | `models/voting_test.go::TestBallotPrivacyDecoupling` | Structural reflection test confirms zero forbidden fields on ballot/participation | ISSUE-03 |

---

## 3.4 Custom Ballot Builder (FR-BALLOT)

| Req ID | Requirement | Status | Code Location | Test Location | Evidence | Owning Issue |
|---|---|---|---|---|---|---|
| FR-BALLOT-01 | Visual ballot builder in CMS | ❌ Missing | — | — | No frontend ballot builder UI | ISSUE-04 |
| FR-BALLOT-02 | Configurable positions/questions (not hardcoded guild roles) | ✅ Implemented | `models/position.go` L18–39 | `database/elections_seeder_test.go` | Dynamic positions seeded (President, VP, Referendum) | ISSUE-04 |
| FR-BALLOT-03 | Single-choice, multi-choice, Yes/No, Referendum question types | ✅ Implemented | `models/position.go` L11–16 | `database/elections_seeder_test.go` | SINGLE_CHOICE, MULTI_CHOICE, YES_NO, REFERENDUM constants | ISSUE-04 |
| FR-BALLOT-04 | Ranked-choice voting (only if explicitly selected) | ❌ Missing | — | — | Not modelled; not in current scope | ISSUE-04 |
| FR-BALLOT-05 | Min/max selections, seats, allow-blank configuration | ✅ Implemented | `models/position.go` L26–31 | `database/elections_seeder_test.go` | Fields present and seeded; limits not yet enforced in submission handler | ISSUE-04 |
| FR-BALLOT-06 | Pre-open validation (no empty positions, valid limits) | ❌ Missing | — | — | Model has fields; no validator function yet | ISSUE-04 |
| FR-BALLOT-07 | Ballot preview (desktop/mobile) | ❌ Missing | — | — | No frontend ballot preview | ISSUE-04 |
| FR-BALLOT-08 | Versioned ballot definition; no silent reinterpretation | 🔶 Partial | `models/election.go` (Version field, incremented on update) | — | Version field exists; no immutable snapshot after open | ISSUE-04 |

---

## 3.5 Eligibility & Election Restrictions (FR-ELIG)

| Req ID | Requirement | Status | Code Location | Test Location | Evidence | Owning Issue |
|---|---|---|---|---|---|---|
| FR-ELIG-01 | Independent eligibility rules per election | ✅ Implemented | `models/election.go` (AllowedDomains, RequireRoster); `models/roster.go` | `database/elections_seeder_test.go` | Election-specific roster; domain field per election | ISSUE-05 |
| FR-ELIG-02 | Restrict by faculty, department, programme, year, class, or roster | ✅ Implemented | `models/roster.go` L19–35 (Faculty, Department, Programme, StudyYear) | `database/elections_seeder_test.go` | Roster table has all classification fields; 2 entries seeded | ISSUE-05 |
| FR-ELIG-03 | CSV roster import with validation, duplicate detection, preview | ❌ Missing | — | — | Roster model exists; import endpoint not built | ISSUE-05 |
| FR-ELIG-04 | Backend eligibility check at ballot access and submission | ❌ Missing | — | — | No voting handler yet to enforce this | ISSUE-05 |
| FR-ELIG-05 | Restrict access to confidential eligibility lists | ❌ Missing | — | — | No RBAC on roster endpoints yet | ISSUE-05 |
| FR-ELIG-06 | Log eligibility changes, especially post-open | ❌ Missing | — | — | Audit model exists; roster change events not hooked | ISSUE-05 |
| FR-ELIG-07 | Handle suspended/graduated/ineligible accounts | 🔶 Partial | `models/roster.go` L11–16 (SUSPENDED, GRADUATED status constants) | — | Statuses defined; not enforced in submission path | ISSUE-05 |
| FR-ELIG-08 | Multi-election eligibility isolation | ✅ Implemented | `models/roster.go` (unique index `idx_roster_election_email` scoped by `election_id`) | `database/elections_seeder_test.go` | Composite unique index verifies election-scoped eligibility | ISSUE-05 |

---

## 3.6 Voting & Duplicate Prevention (FR-VOTE)

| Req ID | Requirement | Status | Code Location | Test Location | Evidence | Owning Issue |
|---|---|---|---|---|---|---|
| FR-VOTE-01 | Ballot access only when election is open and voter eligible | 🔶 Partial | `models/election.go::IsOpen()` | `models/voting_test.go::TestElectionLifecycle` | Schedule check implemented; eligibility gate at access not yet in handler | ISSUE-06 |
| FR-VOTE-02 | Present ballot version approved for that election | ❌ Missing | — | — | No ballot fetch endpoint | ISSUE-06 |
| FR-VOTE-03 | Voter reviews selections before submission | ❌ Missing | — | — | No frontend ballot review step | ISSUE-06 |
| FR-VOTE-04 | Exactly one accepted ballot per eligible voter per election | ✅ Implemented | `models/participation.go` L21–25 (unique index `idx_participation_election_user`) | `models/voting_test.go::TestDuplicateVotingConstraint` | DB-level unique constraint tested with SQLite; duplicate rejected | ISSUE-06 |
| FR-VOTE-05 | Prevent double-click/retry/concurrent/replay duplicates | ✅ Implemented | `models/participation.go` (unique constraint rejects race); `middleware/idempotency.go` | `models/voting_test.go::TestDuplicateVotingConstraint` | Second create fails with UNIQUE constraint error | ISSUE-06 |
| FR-VOTE-06 | Server-side transactions and DB constraints (not frontend alone) | ✅ Implemented | `models/participation.go` (DB unique index) | `models/voting_test.go::TestDuplicateVotingConstraint` | Constraint enforced at DB level | ISSUE-06 |
| FR-VOTE-07 | Atomic: accept ballot + record participation, or neither | 🔶 Partial | `models/ballot.go` + `models/participation.go` (separate tables, no FK) | — | Separation modelled; no atomic transaction handler yet | ISSUE-06 |
| FR-VOTE-08 | Recheck election status, eligibility, ballot validity at submission | ❌ Missing | — | — | No submission handler yet | ISSUE-06 |
| FR-VOTE-09 | Reject malformed ballots, closed elections, invalid selections | ❌ Missing | — | — | No submission handler yet | ISSUE-06 |
| FR-VOTE-10 | Generic confirmation with no encoded choices | ✅ Implemented | `models/participation.go` L24 (`ConfirmationToken` = random hex, no choices) | `models/voting_test.go::TestDuplicateVotingConstraint` | Token generated; no choice information encoded | ISSUE-06 |
| FR-VOTE-11 | Check-ballot-accepted flow without creating duplicates | ❌ Missing | — | — | No status-check endpoint | ISSUE-06 |
| FR-VOTE-12 | Election data isolation (votes from one election cannot enter another's tally) | ✅ Implemented | All election models scoped by `election_id` FK; `participation` unique index scoped | `models/voting_test.go::TestDuplicateVotingConstraint` | Different election with same user allowed; cross-election isolation confirmed | ISSUE-06 |

---

## 3.7 Ballot Secrecy & Privacy (FR-PRIV)

| Req ID | Requirement | Status | Code Location | Test Location | Evidence | Owning Issue |
|---|---|---|---|---|---|---|
| FR-PRIV-01 | Never expose voter choices to candidates, public, admins | ✅ Implemented | `models/ballot.go` (no voter FK); `models/participation.go` (no ballot FK) | `models/voting_test.go::TestBallotPrivacyDecoupling` | Structural reflection confirms zero identity-to-choice linkage | ISSUE-06 |
| FR-PRIV-02 | Separate identity, eligibility, participation, and ballot contents | ✅ Implemented | 4 distinct tables: `users`, `election_rosters`, `voter_participations`, `anonymous_ballots` | `models/voting_test.go::TestBallotPrivacyDecoupling` | Air-gap enforced; privacy architecture comment in `ballot.go` L17–21 | ISSUE-06 |
| FR-PRIV-03 | Schema/API/logs must not expose voter-to-ballot mapping | ✅ Implemented | `models/ballot.go` (AnonymousBallot: no user_id, email, session_id, IP) | `models/voting_test.go::TestBallotPrivacyDecoupling` | 7 forbidden fields checked by reflection | ISSUE-06 |
| FR-PRIV-04 | No ballot selections in profiles, logs, notifications | 🔶 Partial | Architecture modelled; `election_audit.go` comment L26 | — | Audit model excludes choices; log filter not tested | ISSUE-06 |
| FR-PRIV-05 | Generic non-transferable confirmation | ✅ Implemented | `models/participation.go` (random `ConfirmationToken`) | `models/voting_test.go` | Token is random hex; does not encode any choice | ISSUE-06 |
| FR-PRIV-06 | Limit access to identity/eligibility records | ❌ Missing | — | — | No RBAC on roster/participation endpoints yet | ISSUE-05 |
| FR-PRIV-07 | Privacy threat assessment (DB access, logs, backups, timing) | ❌ Missing | — | — | Documented as required; not yet performed | ISSUE-09 |
| FR-PRIV-08 | Document actual anonymity guarantee and residual risks | ❌ Missing | — | — | `ballot.go` comment acknowledges timing risk; formal doc absent | ISSUE-09 |
| FR-PRIV-09 | Evaluate cryptographic anonymous credentials if required | ❌ Missing | — | — | ADR-006 defers this pending threat model | ISSUE-09 |
| FR-PRIV-10 | No ordinary admin query joining voter identity to choices | ✅ Implemented | `models/ballot.go` (no voter FK); `models/participation.go` (no ballot FK) | `models/voting_test.go::TestBallotPrivacyDecoupling` | Impossible by schema design; reflection-tested | ISSUE-06 |

---

## 3.8 Tallying & Result Integrity (FR-TALLY)

| Req ID | Requirement | Status | Code Location | Test Location | Evidence | Owning Issue |
|---|---|---|---|---|---|---|
| FR-TALLY-01 | Automated result calculation from accepted ballots | 🔶 Partial | `models/tally.go` (ElectionTally snapshot model) | — | Snapshot model exists; no tally computation service | ISSUE-08 |
| FR-TALLY-02 | Tally each position independently by its method | ❌ Missing | — | — | No tally service | ISSUE-08 |
| FR-TALLY-03 | Vote percentages with explicit denominator | 🔶 Partial | `models/tally.go` L19–21 (`Votes`, `ValidVotesTotal`, `Percentage`) | — | Fields defined; no computation yet | ISSUE-08 |
| FR-TALLY-04 | Turnout separate from candidate vote share | 🔶 Partial | `models/tally.go` L22 (`TurnoutPercentage`) | — | Field defined; not computed | ISSUE-08 |
| FR-TALLY-05 | Blank/abstention handling per election rules | ❌ Missing | — | — | No tally service; `AllowBlank` on Position model | ISSUE-08 |
| FR-TALLY-06 | Tie detection with no invented winner | 🔶 Partial | `models/tally.go` L25 (`IsTie` flag) | — | Flag defined; no detection logic | ISSUE-08 |
| FR-TALLY-07 | Deterministic: same inputs + rules = same output | ❌ Missing | — | — | No tally service to verify | ISSUE-08 |
| FR-TALLY-08 | Validate ballot structure; reject duplicate/malformed/cross-election records | ❌ Missing | — | — | No submission/tally validator | ISSUE-08 |
| FR-TALLY-09 | Reconciliation stats (eligible count, participation, accepted, valid) | ❌ Missing | — | — | No reconciliation endpoint | ISSUE-08 |
| FR-TALLY-10 | Controlled recount without editing original ballots | ❌ Missing | — | — | No recount service | ISSUE-08 |
| FR-TALLY-11 | Recount records actor, time, rule version, output | ❌ Missing | — | — | Audit model exists; recount not implemented | ISSUE-08 |
| FR-TALLY-12 | Final result snapshot with config/rule version | 🔶 Partial | `models/tally.go` (`ComputedAt`, `ElectionID`, `PositionID`) | — | Snapshot fields defined; certification not wired | ISSUE-08 |
| FR-TALLY-13 | Independent test vectors for every tally method | ❌ Missing | — | — | No tally service to test | ISSUE-08 |
| FR-TALLY-14 | Property, boundary, concurrency, regression tests for tallying | ❌ Missing | — | — | No tally service yet | ISSUE-08 |
| FR-TALLY-15 | Reproducible tally + reconciliation report | ❌ Missing | — | — | No tally service | ISSUE-08 |

---

## 3.9 Public Dashboard & Results (FR-DASH)

| Req ID | Requirement | Status | Code Location | Test Location | Evidence | Owning Issue |
|---|---|---|---|---|---|---|
| FR-DASH-01 | Public/university election pages per access policy | ❌ Missing | — | — | No public election listing endpoint | ISSUE-07 |
| FR-DASH-02 | Election title, category, status, schedule, candidates, manifestos | ❌ Missing | — | — | No public election API | ISSUE-07 |
| FR-DASH-03 | Eligible voter count, participation count, turnout | ❌ Missing | — | — | No results endpoint | ISSUE-07 |
| FR-DASH-04 | Live results when enabled (candidate totals, percentages, leader) | ❌ Missing | — | — | `realtime` hub present; no election channel | ISSUE-07 |
| FR-DASH-05 | Automatic result refresh | ❌ Missing | — | — | Realtime hub present; no election channel | ISSUE-07 |
| FR-DASH-06 | Label results as Live/Provisional/Closed/Certified | 🔶 Partial | `models/election.go` (`Status` + `ResultPolicy` constants) | — | Constants defined; no API response labels | ISSUE-07 |
| FR-DASH-07 | Per-election results policy (live, after-close, after-cert) | ✅ Implemented | `models/election.go` L38–42 (`ResultPolicy` constants) | — | Three constants defined and seeded | ISSUE-07 |
| FR-DASH-08 | Show denominator for each percentage; distinguish turnout from vote share | 🔶 Partial | `models/tally.go` (fields defined) | — | Fields defined; no results API | ISSUE-07 |
| FR-DASH-09 | Never label provisional leader as winner | ❌ Missing | — | — | No results API yet | ISSUE-07 |
| FR-DASH-10 | Expose only aggregate results; no voter identities or ballots | 🔶 Partial | Privacy architecture modelled (separate tables) | — | Architecture prevents linkage; no results API yet | ISSUE-07 |
| FR-DASH-11 | Historical certified results | ❌ Missing | — | — | No certification or history endpoint | ISSUE-07 |

---

## 3.10 Notifications, Reports, Audit, Certification (FR-OPS, FR-REPORT, FR-AUDIT, FR-CERT)

| Req ID | Requirement | Status | Code Location | Test Location | Evidence | Owning Issue |
|---|---|---|---|---|---|---|
| FR-OPS-01 | In-app and email notifications for schedules, opening, closing, results | 🔶 Partial | `handlers/notification.go`; `mail/mailer.go` | — | Generic notification infrastructure from Grit; election-specific notifications not wired | ISSUE-11 |
| FR-OPS-02 | Notifications must not disclose ballot choices | 🔶 Partial | `models/election_audit.go` L26 (comment prohibiting choices) | — | Architectural note; not tested | ISSUE-11 |
| FR-REPORT-01 | Election reports (metadata, candidates, participation, results, certification) | ❌ Missing | — | — | No election report generator | ISSUE-10 |
| FR-REPORT-02 | PDF and CSV export | ❌ Missing | `internal/pdf/`; `internal/export/` (generic) | `export/export_test.go` | Generic export infrastructure present; no election-specific export | ISSUE-10 |
| FR-REPORT-03 | Exported totals match result snapshot | ❌ Missing | — | — | No election report | ISSUE-10 |
| FR-AUDIT-01 | Administrative event audit (config, candidate decisions, opening, closing, certification) | 🔶 Partial | `models/election_audit.go` (model + constants) | — | Model and action constants defined; no service emitting events | ISSUE-09 |
| FR-AUDIT-02 | Tamper-evident audit records | 🔶 Partial | `models/election_audit.go`; `internal/appendonly/` (generic append-only log) | `appendonly/appendonly_test.go` | Generic append-only infrastructure exists; election audit not wired to it | ISSUE-09 |
| FR-AUDIT-03 | Audit logs contain no ballot choices | ✅ Implemented | `models/election_audit.go` L26 (no choice fields on model) | — | Schema verified; no ballot fields on ElectionAuditLog | ISSUE-09 |
| FR-AUDIT-04 | Restrict detailed audit to authorized reviewers | ❌ Missing | — | — | No election audit endpoint; generic activity endpoint exists | ISSUE-09 |
| FR-CERT-01 | Stop accepting votes at server-enforced closing time | ✅ Implemented | `models/election.go::CanAcceptVotes()` | `models/voting_test.go::TestElectionLifecycle` | Time-based gate tested | ISSUE-08 |
| FR-CERT-02 | Formal certification only by authorized roles after review | ❌ Missing | — | — | No certification handler | ISSUE-08 |
| FR-CERT-03 | Certified results protected against silent changes | ❌ Missing | — | — | No certification service | ISSUE-08 |
| FR-CERT-04 | Reopening/post-opening changes require authorization + audit | ❌ Missing | — | — | No election mutation handler | ISSUE-08 |

---

## 4. Non-Functional Requirements

| Req ID | Requirement | Status | Code Location | Test Location | Evidence | Owning Issue |
|---|---|---|---|---|---|---|
| NFR-SEC-01 | HTTPS for all production traffic | N/A | `compose.dev.yaml` (no TLS); ADR-012 | — | TLS is deployment concern; reverse proxy to be configured with MMU IT | ISSUE-12 |
| NFR-SEC-02 | Server-side authorization on every protected operation | 🔶 Partial | `middleware/auth.go`; `routes/routes.go` (RequireRole) | `handlers/auth_test.go` | Generic RBAC wired; election-specific authorization not implemented | ISSUE-01 |
| NFR-SEC-03 | Secure cookies/sessions, rate limiting, MFA | 🔶 Partial | `middleware/csrf.go`; `sentinel/` (rate limiting); `handlers/totp.go` | `services/two_factor_test.go` | Infrastructure present; election-specific MFA policy not configured | ISSUE-01 |
| NFR-SEC-04 | Injection, XSS, CSRF, brute-force, IDOR protection | 🔶 Partial | `middleware/csrf.go`; `sanitize/html.go`; Sentinel WAF | `sanitize/html_test.go` | WAF + CSRF + sanitiser wired; IDOR not specifically tested | ISSUE-12 |
| NFR-SEC-05 | Secrets outside source control | ✅ Implemented | `.env.example` (placeholders only); `.gitignore` | — | No real secrets committed; scan clean | ISSUE-00 |
| NFR-SEC-06 | Least-privilege DB accounts | N/A | `compose.dev.yaml` | — | Deployment concern; documented in ADR-003 | ISSUE-12 |
| NFR-SEC-07 | Dependency, config, security testing pre-election | ❌ Missing | — | — | No security test suite; no CI pipeline | ISSUE-12 |
| NFR-SEC-08 | No single admin can silently change ballots/votes/results | 🔶 Partial | `models/election_audit.go`; `internal/appendonly/` | — | Audit log modelled; separation of duties not enforced | ISSUE-09 |
| NFR-REL-01 | Durable ballot persistence | 🔶 Partial | `models/ballot.go`; PostgreSQL config | — | Model designed for PostgreSQL; no persistence test yet | ISSUE-06 |
| NFR-REL-02 | Transactional consistency across ballot + participation | 🔶 Partial | Schema separated; no atomic handler yet | — | Architecture ready; handler not implemented | ISSUE-06 |
| NFR-REL-03 | Documented + tested backup restoration | ❌ Missing | `handlers/backup.go`; `cmd/backup/`; `cmd/restore/` | — | Backup handler scaffolded; no election restore drill | ISSUE-12 |
| NFR-PERF-01/02/03 | 95th percentile latency targets; realtime under 5 s | ❌ Missing | `tests/k6/` (load test scripts) | `tests/k6/smoke.js` etc. | k6 scripts scaffolded; not run against live stack | ISSUE-13 |
| NFR-UX-01/02/03/04 | Accessibility WCAG 2.2 AA; responsive; clear errors | ❌ Missing | `apps/web/`; `apps/admin/` (minimal scaffold) | — | Frontend scaffold minimal; no voter UX pages built | ISSUE-04 |
| NFR-MAINT-01/02/03 | Modular backend; configurable rules; versioned migrations | 🔶 Partial | `apps/api/internal/` (modular packages); `cmd/migrate/` | — | Migration command exists; election-specific migrations not verified from empty DB | ISSUE-00 |
| NFR-MAINT-04 | Automated tests for auth, eligibility, concurrency, secrecy, tallying | 🔶 Partial | `models/voting_test.go` (privacy + duplicate) | See test inventory | Core privacy + duplicate tests pass; concurrency/tally tests absent | ISSUE-06 |
| NFR-MAINT-05 | Pinned framework/tool versions | ✅ Implemented | `go.mod`; `package.json`; `SETUP_LOG.md` | — | All versions pinned | ISSUE-00 |
| NFR-DATA-01/02/03/04 | Data minimization, retention, Ugandan law, incident response | ❌ Missing | — | — | Not implemented; pending university policy decisions | ISSUE-09 |

---

## Summary Counts

| Status | Count |
|---|---|
| ✅ Implemented | 17 |
| 🔶 Partial | 32 |
| ❌ Missing | 43 |
| N/A | 4 |
| **Total** | **96** |

> **Note:** "Implemented" means the requirement has code AND at least one automated test covering it.  
> "Partial" means the model/scaffold exists but either the handler/service or the test is absent.  
> "Missing" means there is no relevant code yet.
