# SRS v1.1 → Implementation Traceability Matrix
## campVote — MMU Campus E-Voting System
**Audit date:** 2026-10-28  
**Audited by:** ISSUE-00 implementation audit (chore/issue-00-implementation-audit)  
**Authoritative source:** `develop` branch, commit `22c6255` (Grit Triple monorepo scaffold + campus voting domain)

> **ISSUE-02 update (branch `feat/issue-02-auth-session-hardening`):** the FR-AUTH and NFR-SEC rows marked below were re-verified against the hardening work and its new HTTP-level tests. See [`auth.md`](auth.md) for thresholds, cookie flags, bootstrap steps and exact results. These rows describe the task branch until it is merged into `develop`.
>
> **ISSUE-03 update (branch `feat/issue-03-server-side-rbac`):** route policies, SRS roles, per-election scoping, separation of duties, results visibility and certification are implemented. See [`rbac.md`](rbac.md) for the role matrix, 401/403/404 semantics, the 181-route inventory and exact test results.
>
> **ISSUE-05 update (branch `feat/issue-05-election-lifecycle`):** election creation/editing, the full state machine, schedule validation, server-clock opening/closing, scheduler recovery, configuration freeze and audited exceptions are implemented. See [`election-lifecycle.md`](election-lifecycle.md) for the matrix, boundary semantics and exact test results.

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
| FR-AUTH-01 | Permit protected access only to authenticated institutional identities | ✅ Implemented | `middleware/auth.go` (token + live session + active user); `routes/routes.go` (`protected` group with `middleware.Auth`) | `handlers/auth_security_test.go::TestLogout_RevokesCookieSession`, `::TestDeactivation_RevokesSessionsAndBlocksNextRequest` | Access tokens require a live server-side session and an active user; revoked/logged-out sessions get 401. Institutional domain check remains FR-AUTH-02. | ISSUE-02 |
| FR-AUTH-02 | Enforce `@std.must.ac.ug` / `@must.ac.ug` domain policy, server-side | ❌ Missing | `models/election.go` L59 (AllowedDomains field on Election, not enforced at auth) | — | Domain stored per-election but no registration/login domain validator exists | ISSUE-01 |
| FR-AUTH-03 | Validate email/identity claims server-side | 🔶 Partial | `middleware/auth.go`; `services/auth.go` | `handlers/auth_test.go`; `handlers/auth_security_test.go` | Token type, signature, expiry, session liveness and active status validated; institutional domain not enforced | ISSUE-02 |
| FR-AUTH-04 | Prefer university SSO/OIDC | 🔶 Partial | `handlers/auth_oauth.go`; `handlers/sso.go`; `services/sso.go` | `handlers/sso_test.go` | Generic OIDC/SAML scaffolded; MMU-specific IdP not configured | ISSUE-01 |
| FR-AUTH-05 | Verify current institutional status from authoritative directory | ❌ Missing | — | — | No directory/roster integration at auth layer | ISSUE-05 |
| FR-AUTH-06 | RBAC with least privilege on every protected backend operation | ✅ Implemented | `internal/policy/` (route table + object policy); `internal/authz/`; `middleware/auth.go`; `models/role.go`, `models/election_role.go` | `routes/policy_inventory_test.go::TestEveryRouteHasAPolicy`; `policy/election_test.go`; `handlers/election_rbac_test.go` | All 181 non-library routes have an explicit policy and unlisted routes are denied; SRS roles, per-election scoping and separation of duties enforced. New election endpoints add policies; the inventory test fails otherwise | ISSUE-03 |
| FR-AUTH-07 | Session expiry, logout, suspension, security-event logging | ✅ Implemented | `services/session.go` (7-day idle, 30-day absolute, rotation/replay); `handlers/auth.go::Logout`; `handlers/user.go`; `services/activity.go` | `services/session_test.go`; `handlers/auth_security_test.go::TestPasswordReset_RevokesSessionsAndCannotBeReplayed`, `::TestLogout_RevokesBearerSession`, `::TestDeactivation_RevokesSessionsAndBlocksNextRequest` | Logout, password reset and deactivation revoke sessions; login/lockout/reset/deactivation audited | ISSUE-02 |
| FR-AUTH-08 | MFA/TOTP for privileged accounts | ✅ Implemented | `handlers/totp.go`; `services/two_factor.go` | `services/two_factor_test.go` | TOTP + backup codes + passkeys scaffolded and tested | ISSUE-01 |
| FR-AUTH-09 | Never log passwords, tokens, ballot contents | 🔶 Partial | `middleware/security_log.go`; `middleware/logger.go`; `models/user.go` (`json:"-"`); `services/activity.go` | `handlers/auth_security_test.go::TestRegister_StoresHashAndDoesNotLeakPassword` | Passwords are hash-only and absent from API responses/audit rows; explicit ballot-content logging still untested because no voting route exists | ISSUE-02 |

---

## 3.2 Election CMS & Lifecycle (FR-CMS)

| Req ID | Requirement | Status | Code Location | Test Location | Evidence | Owning Issue |
|---|---|---|---|---|---|---|
| FR-CMS-01 | Create/edit/schedule/publish/close/cancel/archive elections | ✅ Implemented | `handlers/election_lifecycle.go`; `internal/lifecycle/lifecycle.go` | `handlers/election_lifecycle_test.go::TestElectionLifecycleAPIValidatesSchedulesFreezesAndAudits` | Create/edit, schedule, open/close by server clock, cancel and archive; scoped permissions; audited | ISSUE-05 |
| FR-CMS-02 | Full election configuration (title, dates, category, eligibility, ballot, visibility) | ✅ Implemented | `models/election.go` L44–71 | `models/voting_test.go::TestElectionLifecycle` | All config fields on Election model; lifecycle tested | ISSUE-02 |
| FR-CMS-03 | Multi-category elections (Guild, Faculty, Department, Programme, Class, Club, etc.) | ✅ Implemented | `models/election.go` L11–21 | `database/elections_seeder_test.go::TestSeedElections` | 9 category constants; Guild + Faculty seeded and tested | ISSUE-02 |
| FR-CMS-04 | Multiple simultaneous elections, isolated data | ✅ Implemented | `models/election.go`; `models/participation.go` (election-scoped unique index) | `database/elections_seeder_test.go` | Multi-election seeder creates 2 isolated elections; participation scoped by `election_id` | ISSUE-02 |
| FR-CMS-05 | Lifecycle states: Draft, Registration, Pending, Scheduled, Open, Closed, Review, Finalized, Archived, Cancelled | ✅ Implemented | `models/election.go` L24–35; `internal/lifecycle/lifecycle.go` | `internal/lifecycle/lifecycle_test.go::TestTransitionMatrix` | All 10 states and every allowed/forbidden pair are enforced by the backend | ISSUE-05 |
| FR-CMS-06 | Validate configuration before opening | ✅ Implemented | `internal/lifecycle/lifecycle.go::ValidateConfiguration`; `handlers/election_lifecycle.go` | `internal/lifecycle/lifecycle_test.go::TestValidateConfiguration`, `::TestValidateConfigurationRequiresPositionsAndCandidates`, `::TestEnsureDoesNotOpenInvalidConfiguration` | Schedule, duration, policy, domains, positions, selection limits and approved candidates are checked before scheduling/opening; invalid config fails closed | ISSUE-05 |
| FR-CMS-07 | Reusable election templates | ❌ Missing | — | — | Not yet modelled | ISSUE-02 |
| FR-CMS-08 | Audit config changes; freeze ballot after opening | ✅ Implemented | `handlers/election_lifecycle.go`; `internal/lifecycle/lifecycle.go::IsFrozen`; `models/election_audit.go` | `handlers/election_lifecycle_test.go::TestElectionLifecycleAPIValidatesSchedulesFreezesAndAudits`, `::TestOfficialCanEditButCannotCancelOrExtend` | Edits before open audited as `ELECTION_UPDATED`; after open blocked except a reasoned, publisher-authorised `ends_at` extension audited as `ELECTION_EXTENDED`; optimistic version check | ISSUE-05 |
| FR-CMS-09 | Server-enforced schedule (open/close by time) | ✅ Implemented | `models/election.go::IsOpen()`; `internal/lifecycle/lifecycle.go::Ensure`, `SyncAll`; `internal/jobs/workers.go` | `internal/lifecycle/lifecycle_test.go::TestOpenAndCloseBoundaries`, `::TestMissedSchedulerTickCannotExtendVoting`, `::TestSchedulerRestartReconcilesMissedTransitions`; `handlers/election_lifecycle_test.go::TestSpoofedDateHeaderCannotExtendVoting` | Server clock only; lazy check on every ballot path plus a one-minute scheduler; end-exclusive boundary; spoofed `Date` header has no effect | ISSUE-05 |
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
| FR-ELIG-01 | Independent eligibility rules per election | ✅ Implemented | `models/election.go` (AllowedDomains; deprecated RequireRoster); `internal/eligibility/service.go::CheckElection` | `internal/eligibility/eligibility_test.go::TestEligibilityIsPerElection` | Domain list and roster are read per election; RequireRoster no longer grants access | ISSUE-04 |
| FR-ELIG-02 | Restrict by faculty, department, programme, year, class, or roster | ✅ Implemented | `models/roster.go` (Faculty, Department, Programme, StudyYear); `internal/eligibility/service.go` | `internal/eligibility/eligibility_test.go::TestClockAndStatusGate` | Roster is the only evidence; statuses refuse access | ISSUE-04 |
| FR-ELIG-03 | CSV roster import with validation, duplicate detection, preview | ✅ Implemented | `internal/eligibility/import.go`; `internal/handlers/eligibility.go` | `internal/eligibility/eligibility_test.go::TestImportPreviewRejectsBadAndDuplicateRowsWithoutWriting`; `TestImportCommitRecordsProvenance`; `internal/handlers/eligibility_test.go::TestRosterImportPreviewCommitFreezeAndReport` | Preview/commit, per-row reasons, duplicate identity, content hash, atomic refusal | ISSUE-04 |
| FR-ELIG-04 | Backend eligibility check at ballot access and submission | ✅ Implemented | `internal/handlers/ballot.go` (View/Start/Submit); `internal/eligibility/service.go::CheckElection` | `internal/handlers/eligibility_test.go::TestEligibilityGateOnBallotViewStartAndSubmit` | Same gate on view, start and submit; rechecked inside the submit transaction | ISSUE-04 |
| FR-ELIG-05 | Restrict access to confidential eligibility lists | ✅ Implemented | `internal/authz/permissions.go` (`rosters.view`/`rosters.edit`/`rosters.import`); `internal/policy/election.go::AuthorityFor`; `internal/handlers/eligibility.go` | `internal/handlers/eligibility_test.go::TestVoterCannotChangeOwnEligibility`; `policy/election_test.go::TestAuthorityScopesOfficialToAssignedElection` | Every roster endpoint resolves election authority; voters get 403, cross-election gets 404 | ISSUE-04 |
| FR-ELIG-06 | Log eligibility changes, especially post-open | ✅ Implemented | `internal/eligibility/import.go`; `internal/eligibility/service.go` (`ROSTER_IMPORTED`, `ROSTER_REVOKED`, `ELIGIBILITY_OVERRIDE`, `ELIGIBILITY_FROZEN`) | `internal/eligibility/eligibility_test.go::TestFreezeRefusesNewEntriesButAllowsRevocation`; `TestImportCommitRecordsProvenance` | Imports, revocations, overrides and freezes are audited with roster IDs and reasons only | ISSUE-04 |
| FR-ELIG-07 | Handle suspended/graduated/ineligible accounts | ✅ Implemented | `models/roster.go` statuses; `internal/eligibility/service.go::CheckElection` | `internal/eligibility/eligibility_test.go::TestFreezeRefusesNewEntriesButAllowsRevocation`; `internal/handlers/eligibility_test.go::TestRosterImportPreviewCommitFreezeAndReport` | Non-ELIGIBLE statuses refuse ballot access immediately | ISSUE-04 |
| FR-ELIG-08 | Multi-election eligibility isolation | ✅ Implemented | `models/roster.go` (unique indexes scoped by `election_id`); `internal/eligibility/service.go` | `internal/eligibility/eligibility_test.go::TestEligibilityIsPerElection`; `internal/handlers/eligibility_test.go::TestEligibilityGateOnBallotViewStartAndSubmit` | Eligibility in one election does not carry to another | ISSUE-04 |

---

## 3.6 Voting & Duplicate Prevention (FR-VOTE)

| Req ID | Requirement | Status | Code Location | Test Location | Evidence | Owning Issue |
|---|---|---|---|---|---|---|
| FR-VOTE-01 | Ballot access only when election is open and voter eligible | ✅ Implemented | `models/election.go::IsOpen()`; `internal/handlers/ballot.go::gate` | `internal/handlers/eligibility_test.go::TestEligibilityGateOnBallotViewStartAndSubmit` | View, start and submit all refuse a closed election or ineligible account | ISSUE-04 |
| FR-VOTE-02 | Present ballot version approved for that election | ❌ Missing | — | — | No ballot versioning yet; the minimal view lists approved positions/candidates | ISSUE-06 |
| FR-VOTE-03 | Voter reviews selections before submission | ❌ Missing | — | — | No frontend ballot review step | ISSUE-06 |
| FR-VOTE-04 | Exactly one accepted ballot per eligible voter per election | ✅ Implemented | `models/participation.go` L21–25 (unique index `idx_participation_election_user`) | `models/voting_test.go::TestDuplicateVotingConstraint` | DB-level unique constraint tested with SQLite; duplicate rejected | ISSUE-06 |
| FR-VOTE-05 | Prevent double-click/retry/concurrent/replay duplicates | ✅ Implemented | `models/participation.go` (unique constraint rejects race); `middleware/idempotency.go` | `models/voting_test.go::TestDuplicateVotingConstraint` | Second create fails with UNIQUE constraint error | ISSUE-06 |
| FR-VOTE-06 | Server-side transactions and DB constraints (not frontend alone) | ✅ Implemented | `models/participation.go` (DB unique index); `internal/handlers/ballot.go::Submit` | `models/voting_test.go::TestDuplicateVotingConstraint`; `internal/handlers/eligibility_test.go::TestEligibilityGateOnBallotViewStartAndSubmit` | Constraint enforced at DB level; submit checks inside the transaction | ISSUE-04 |
| FR-VOTE-07 | Atomic: accept ballot + record participation, or neither | ✅ Implemented | `internal/handlers/ballot.go::Submit` (single `DB.Transaction`) | `internal/handlers/eligibility_test.go::TestEligibilityGateOnBallotViewStartAndSubmit` | Participation and anonymous ballot are written together or rolled back | ISSUE-04 |
| FR-VOTE-08 | Recheck election status, eligibility, ballot validity at submission | ✅ Implemented | `internal/handlers/ballot.go::Submit` (reloads election and rechecks inside the transaction) | `internal/handlers/eligibility_test.go::TestEligibilityGateOnBallotViewStartAndSubmit` | Cross-election submit and revoked eligibility are refused | ISSUE-04 |
| FR-VOTE-09 | Reject malformed ballots, closed elections, invalid selections | 🔶 Partial | `internal/handlers/ballot.go::validateSelections` | `internal/handlers/eligibility_test.go::TestMultiChoiceBallotAcceptsDistinctChoicesAndRejectsDuplicates` | Positions, candidates, min/max, abstention and duplicate choices validated; ballot-version rules pending ISSUE-06 | ISSUE-04 |
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
| FR-PRIV-06 | Limit access to identity/eligibility records | 🔶 Partial | `internal/policy/election.go::AuthorityFor`; `internal/authz/permissions.go` (`rosters.*`) | `policy/election_test.go`; `handlers/election_rbac_test.go::TestRoutePolicyMatrixForElectionEndpoints` | Election-scoped authority returns 404 outside the actor's elections; roster/participation endpoints still pending | ISSUE-03 |
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
| FR-DASH-06 | Label results as Live/Provisional/Closed/Certified | 🔶 Partial | `handlers/election_rbac.go::Results`; `models/election.go` (`Status` + `ResultPolicy`) | `policy/election_test.go::TestResultsVisibilityFollowsElectionPolicy` | API labels `FINALIZED`/`ARCHIVED` as `CERTIFIED` and everything else `PROVISIONAL`; no Live/Closed UI label split yet | ISSUE-03 |
| FR-DASH-07 | Per-election results policy (live, after-close, after-cert) | ✅ Implemented | `models/election.go` L38–42 (`ResultPolicy` constants) | — | Three constants defined and seeded | ISSUE-07 |
| FR-DASH-08 | Show denominator for each percentage; distinguish turnout from vote share | 🔶 Partial | `models/tally.go` (fields defined) | — | Fields defined; no results API | ISSUE-07 |
| FR-DASH-09 | Never label provisional leader as winner | 🔶 Partial | `handlers/election_rbac.go::Results` | `handlers/election_rbac_test.go::TestRoutePolicyMatrixForElectionEndpoints` | Response carries `PROVISIONAL`/`CERTIFIED`, never "winner"; no leader field exists yet | ISSUE-03 |
| FR-DASH-10 | Expose only aggregate results; no voter identities or ballots | ✅ Implemented | `handlers/election_rbac.go::Results`; `models/tally.go` (no identity columns) | `handlers/election_rbac_test.go::TestRoutePolicyMatrixForElectionEndpoints`; `models/voting_test.go::TestBallotPrivacyDecoupling` | Results endpoint returns `ElectionTally` aggregates only; the table has no voter linkage | ISSUE-03 |
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
| FR-AUDIT-04 | Restrict detailed audit to authorized reviewers | 🔶 Partial | `internal/authz/permissions.go` (`audit.view`); `models/role.go` (`CHIEF_ELECTION_ADMIN`, `ELECTION_AUDITOR`) | `authz/grants_test.go::TestSeedRoles` | `audit.view` is granted only to chiefs/admins/auditors, but the election audit read endpoint is not built yet | ISSUE-03 |
| FR-CERT-01 | Stop accepting votes at server-enforced closing time | ✅ Implemented | `models/election.go::CanAcceptVotes()`; `internal/lifecycle/lifecycle.go::Ensure` | `internal/lifecycle/lifecycle_test.go::TestOpenAndCloseBoundaries`, `::TestMissedSchedulerTickCannotExtendVoting` | End-exclusive gate; a missed scheduler tick cannot extend voting | ISSUE-05 |
| FR-CERT-02 | Formal certification only by authorized roles after review | ✅ Implemented | `handlers/election_rbac.go::Certify`; `internal/policy/election.go::CertifyDecision` | `policy/election_test.go::TestCertifyRequiresTwoOfficialsAndADifferentActor`; `handlers/election_rbac_test.go` | Requires `elections.certify`, `CLOSED`/`REVIEW`, two assigned officials, and an actor other than the creator; writes `FINALIZED` + audit row in one transaction | ISSUE-03 |
| FR-CERT-03 | Certified results protected against silent changes | ❌ Missing | — | — | No certification service | ISSUE-08 |
| FR-CERT-04 | Reopening/post-opening changes require authorization + audit | ✅ Implemented | `handlers/election_lifecycle.go::updateFrozen`; `internal/lifecycle/lifecycle.go` | `handlers/election_lifecycle_test.go::TestElectionLifecycleAPIValidatesSchedulesFreezesAndAudits`, `::TestOfficialCanEditButCannotCancelOrExtend` | Post-open changes are blocked; the only exception is a publisher-authorised, reasoned end-time extension, audited with old/new times. Reopening a closed election is refused | ISSUE-05 |

---

## 4. Non-Functional Requirements

| Req ID | Requirement | Status | Code Location | Test Location | Evidence | Owning Issue |
|---|---|---|---|---|---|---|
| NFR-SEC-01 | HTTPS for all production traffic | N/A | `compose.dev.yaml` (no TLS); ADR-012 | — | TLS is deployment concern; reverse proxy to be configured with MMU IT | ISSUE-12 |
| NFR-SEC-02 | Server-side authorization on every protected operation | ✅ Implemented | `internal/policy/policy.go` (global default-deny + guard); `internal/policy/routes.go`; `internal/policy/election.go`; `middleware/auth.go` | `routes/policy_inventory_test.go`; `policy/election_test.go`; `handlers/election_rbac_test.go` | Every registered route has an explicit policy; unlisted routes denied; per-election object scoping tested; admin routes refuse voter tokens | ISSUE-03 |
| NFR-SEC-03 | Secure cookies/sessions, rate limiting, MFA | 🔶 Partial | `services/auth.go::SetAuthCookies`; `middleware/csrf.go`; `sentinel/`; `handlers/totp.go`, `handlers/auth_lockout.go` | `middleware/csrf_test.go`; `handlers/auth_security_test.go`; `services/two_factor_test.go` | HttpOnly/Secure/SameSite cookies, CSRF double-submit, per-account lockout and TOTP tested; election-specific MFA policy not configured | ISSUE-02 |
| NFR-SEC-04 | Injection, XSS, CSRF, brute-force, IDOR protection | 🔶 Partial | `internal/policy/election.go` (IDOR); `middleware/csrf.go`; `sanitize/html.go`; Sentinel WAF; `handlers/auth_lockout.go` | `handlers/election_rbac_test.go::TestRoutePolicyMatrixForElectionEndpoints`; `sanitize/html_test.go`; `middleware/csrf_test.go`; `handlers/auth_security_test.go` | Cross-election IDOR returns 404 and is audited; WAF, CSRF, sanitiser and brute-force lockout tested; SQL-injection coverage still comes from GORM parameterisation only | ISSUE-03 |
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
| NFR-MAINT-04 | Automated tests for auth, eligibility, concurrency, secrecy, tallying | 🔶 Partial | `models/voting_test.go` (privacy + duplicate) | See test inventory; `handlers/auth_security_test.go`; `middleware/csrf_test.go`; `database/bootstrap_test.go` | Auth flows, token replay/expiry, lockout, cookies, CSRF and bootstrap now covered over HTTP; concurrency/tally tests absent | ISSUE-06 |
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

