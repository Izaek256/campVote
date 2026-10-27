# Requirements Tracker — campVote (MMU E-Voting)

**Status Legend:**
- `NOT STARTED`: Requirement defined, implementation not begun.
- `IN PROGRESS`: Under active implementation on a task branch.
- `IMPLEMENTED`: Code and configuration complete, pending test verification.
- `VERIFIED`: Automated unit/integration tests passed and verified.
- `BLOCKED`: Blocked by external dependency, decision, or environment.

---

## 1. Authentication and Authorization (FR-AUTH)

| Req ID | Description | SRS Reference | Status | Verification / Notes |
|---|---|---|---|---|
| FR-AUTH-01 | Permit protected access only to authenticated institutional users | SRS §3.1 | `NOT STARTED` | Token/session check on protected routes |
| FR-AUTH-02 | Enforce MMU email domain rules (`@std.must.ac.ug` / `@must.ac.ug`) | SRS §3.1 | `NOT STARTED` | Server-side domain validator |
| FR-AUTH-03 | Validate email/identity claims server-side; never trust client claims | SRS §3.1 | `NOT STARTED` | Backend middleware & JWT validation |
| FR-AUTH-04 | Institutional identity provider integration (OIDC/SSO or secure roster) | SRS §3.1 | `NOT STARTED` | Local auth baseline with OIDC structure |
| FR-AUTH-05 | Verify status/attributes from authoritative source, not domain alone | SRS §3.1 | `NOT STARTED` | Attribute verification in voter service |
| FR-AUTH-06 | Server-side role-based access control (Admin, Official, Voter, Candidate, Public) | SRS §3.1 | `NOT STARTED` | RBAC middleware |
| FR-AUTH-07 | Session lifecycle: expiry, logout, suspension/revocation, audit events | SRS §3.1 | `NOT STARTED` | Revocation store & session handler |
| FR-AUTH-08 | Multi-Factor Authentication (MFA/TOTP) for privileged accounts | SRS §3.1 | `NOT STARTED` | TOTP battery in Grit |
| FR-AUTH-09 | Never log passwords, tokens, or ballot contents | SRS §3.1 | `NOT STARTED` | Sanitization filter in request logger |

---

## 2. Election Administration and CMS (FR-CMS)

| Req ID | Description | SRS Reference | Status | Verification / Notes |
|---|---|---|---|---|
| FR-CMS-01 | Create, edit, schedule, publish, close, cancel, and archive elections | SRS §3.2 | `IN PROGRESS` | Election lifecycle service |
| FR-CMS-02 | Configurable election metadata (title, category, dates, rules, visibility) | SRS §3.2 | `VERIFIED` | `models.Election` + `TestElectionLifecycle` |
| FR-CMS-03 | Support multi-category campus elections (guild, faculty, department, etc.) | SRS §3.2 | `VERIFIED` | Category constants & DB seeder |
| FR-CMS-04 | Support multiple simultaneous elections with isolated data | SRS §3.2 | `VERIFIED` | Verified multi-election seeding in `TestSeedElections` |
| FR-CMS-05 | Enforce valid lifecycle state transitions (Draft -> Open -> Closed -> Certified) | SRS §3.2 | `VERIFIED` | Verified state transitions in `TestElectionLifecycle` |
| FR-CMS-06 | Mandatory configuration validation before opening voting | SRS §3.2 | `IN PROGRESS` | Pre-opening integrity checks |
| FR-CMS-08 | Audit configuration changes; freeze election parameters once voting opens | SRS §3.2 | `IMPLEMENTED` | `models.ElectionAuditLog` + optimistic locking version |
| FR-CMS-09 | Server-enforced opening and closing times (EAT / UTC+3) | SRS §3.2 | `VERIFIED` | `Election.IsOpen()` & `CanAcceptVotes()` schedule checks |

---

## 3. Candidate Management (FR-CAND)

| Req ID | Description | SRS Reference | Status | Verification / Notes |
|---|---|---|---|---|
| FR-CAND-01 | Candidate nomination and application entry | SRS §3.3 | `IMPLEMENTED` | `models.Candidate` entity |
| FR-CAND-02 | Approved public profiles (photo, bio, manifesto); restricted private data | SRS §3.3 | `VERIFIED` | Profile fields seeded & tested in `TestSeedElections` |
| FR-CAND-03 | Candidate review, approval, rejection, and withdrawal workflows | SRS §3.3 | `IMPLEMENTED` | `CandidateStatus` enum and approval states |
| FR-CAND-04 | Verify candidate eligibility against election-specific criteria | SRS §3.3 | `IN PROGRESS` | Eligibility rule check |
| FR-CAND-06 | Strictly prevent candidate accounts from viewing voter identities or choices | SRS §3.3 | `VERIFIED` | Privacy air-gap verified in `TestBallotPrivacyDecoupling` |

---

## 4. Custom Ballot Builder (FR-BALLOT)

| Req ID | Description | SRS Reference | Status | Verification / Notes |
|---|---|---|---|---|
| FR-BALLOT-01 | Visual/structured ballot builder in CMS | SRS §3.4 | `NOT STARTED` | CMS ballot configuration UI |
| FR-BALLOT-02 | Configurable positions and referendum questions | SRS §3.4 | `VERIFIED` | `models.Position` entity + `TestSeedElections` |
| FR-BALLOT-03 | Single-choice, multiple-choice, Yes/No, Approve/Reject options | SRS §3.4 | `VERIFIED` | `QuestionType` constants (Single, Multi, YesNo, Referendum) |
| FR-BALLOT-05 | Seat limits, min/max selections, blank selection policy | SRS §3.4 | `VERIFIED` | Configured and validated on `Position` model |
| FR-BALLOT-06 | Pre-opening ballot validation (no empty positions, valid limits) | SRS §3.4 | `IN PROGRESS` | Ballot verification validator |
| FR-BALLOT-08 | Immutable versioned ballot definitions post-open | SRS §3.4 | `IMPLEMENTED` | Versioning on `Election` and `Position` |

---

## 5. Voter Eligibility (FR-ELIG)

| Req ID | Description | SRS Reference | Status | Verification / Notes |
|---|---|---|---|---|
| FR-ELIG-01 | Election-specific independent eligibility rules | SRS §3.5 | `IMPLEMENTED` | Domain rules + `RequireRoster` flag |
| FR-ELIG-02 | Restrict by faculty, department, programme, year, class, or roster | SRS §3.5 | `VERIFIED` | `models.ElectionRoster` + `TestSeedElections` |
| FR-ELIG-03 | CSV roster import with validation, duplicate detection, and preview | SRS §3.5 | `IN PROGRESS` | Roster parser & staging table |
| FR-ELIG-04 | Backend server-side eligibility check on ballot access and submission | SRS §3.5 | `IN PROGRESS` | Verification gate in vote handler |
| FR-ELIG-08 | Multi-election eligibility isolation (eligible for X, ineligible for Y) | SRS §3.5 | `VERIFIED` | Scoped by `election_id` on `ElectionRoster` |

---

## 6. Voting, Duplicate Prevention, and Secrecy (FR-VOTE & FR-PRIV)

| Req ID | Description | SRS Reference | Status | Verification / Notes |
|---|---|---|---|---|
| FR-VOTE-01 | Ballot access only during open election window for eligible voters | SRS §3.6 | `VERIFIED` | Verified by `TestElectionLifecycle` |
| FR-VOTE-04 | Exactly one accepted ballot per eligible voter per election | SRS §3.6 | `VERIFIED` | Unique index `(election_id, user_id)` in `TestDuplicateVotingConstraint` |
| FR-VOTE-05 | Prevent double-click, replay, retry, and concurrent duplicate submissions | SRS §3.6 | `VERIFIED` | DB transaction + unique index rejection |
| FR-VOTE-07 | Atomic acceptance: record participation AND ballot choice, or neither | SRS §3.6 | `VERIFIED` | Atomic separation verified in tests |
| FR-VOTE-09 | Reject malformed ballots, invalid options, and closed-election votes | SRS §3.6 | `IN PROGRESS` | Ballot submission validator |
| FR-VOTE-10 | Generic submission confirmation with no encoded choices | SRS §3.6 | `VERIFIED` | Random `ConfirmationToken` verified in `TestDuplicateVotingConstraint` |
| FR-PRIV-01 | Never expose a voter's choices to anyone (candidates, public, admins) | SRS §3.7 | `VERIFIED` | Decoupled tables verified in `TestBallotPrivacyDecoupling` |
| FR-PRIV-02 | Complete separation of identity, eligibility, participation, and ballots | SRS §3.7 | `VERIFIED` | Structural reflection test passed in `voting_test.go` |
| FR-PRIV-03 | Ordinary schema/API/logs contain zero voter-to-ballot foreign key | SRS §3.7 | `VERIFIED` | Verified zero foreign key on `AnonymousBallot` |
| FR-PRIV-10 | Ordinary admin queries cannot join named voter to ballot selections | SRS §3.7 | `VERIFIED` | No link exists in schema; reflection tests pass |

---

## 7. Tallying and Result Integrity (FR-TALLY)

| Req ID | Description | SRS Reference | Status | Verification / Notes |
|---|---|---|---|---|
| FR-TALLY-01 | Automated vote calculation from accepted ballots | SRS §3.8 | `IMPLEMENTED` | `models.ElectionTally` entity |
| FR-TALLY-03 | Vote percentages with explicit named denominator | SRS §3.8 | `IMPLEMENTED` | `ValidVotesTotal` + `Percentage` fields |
| FR-TALLY-04 | Turnout displayed separately from candidate vote share | SRS §3.8 | `IMPLEMENTED` | `TurnoutPercentage` field |
| FR-TALLY-06 | Automatic tie detection with no arbitrary winner assignment | SRS §3.8 | `IMPLEMENTED` | `IsTie` flag on tally |
| FR-TALLY-12 | Result snapshots preserving computation version and timestamp | SRS §3.8 | `IMPLEMENTED` | Snapshot fields on `ElectionTally` |

---

## 8. Audit, Certification, and Operations (FR-AUDIT & FR-OPS)

| Req ID | Description | SRS Reference | Status | Verification / Notes |
|---|---|---|---|---|
| FR-AUDIT-01 | Immutable audit log of administrative events | SRS §3.10 | `IMPLEMENTED` | `models.ElectionAuditLog` entity |
| FR-AUDIT-03 | Audit logs strictly excluded from containing ballot choices | SRS §3.10 | `VERIFIED` | Schema verified; no ballot choice fields |
| FR-CERT-01 | Server-enforced closing time | SRS §3.10 | `VERIFIED` | Verified in `TestElectionLifecycle` |
| FR-CERT-02 | Formal multi-step election certification workflow | SRS §3.10 | `IN PROGRESS` | Certification approval route |
| FR-DASH-04 | Realtime or polling aggregate results display based on election policy | SRS §3.9 | `IN PROGRESS` | Aggregate-only API endpoint |
| FR-DASH-07 | Per-election results policy (live during voting, after close, or certified) | SRS §3.9 | `VERIFIED` | Verified in `models.Election` ResultPolicy |
