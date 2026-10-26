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
| FR-CMS-01 | Create, edit, schedule, publish, close, cancel, and archive elections | SRS §3.2 | `NOT STARTED` | Election lifecycle service |
| FR-CMS-02 | Configurable election metadata (title, category, dates, rules, visibility) | SRS §3.2 | `NOT STARTED` | Election model & validation |
| FR-CMS-03 | Support multi-category campus elections (guild, faculty, department, etc.) | SRS §3.2 | `NOT STARTED` | Category enum & filtering |
| FR-CMS-04 | Support multiple simultaneous elections with isolated data | SRS §3.2 | `NOT STARTED` | Multi-tenant election scoping |
| FR-CMS-05 | Enforce valid lifecycle state transitions (Draft -> Open -> Closed -> Certified) | SRS §3.2 | `NOT STARTED` | State machine validation |
| FR-CMS-06 | Mandatory configuration validation before opening voting | SRS §3.2 | `NOT STARTED` | Pre-opening integrity checks |
| FR-CMS-08 | Audit configuration changes; freeze election parameters once voting opens | SRS §3.2 | `NOT STARTED` | Immutability constraint post-open |
| FR-CMS-09 | Server-enforced opening and closing times (EAT / UTC+3) | SRS §3.2 | `NOT STARTED` | Backend time enforcement |

---

## 3. Candidate Management (FR-CAND)

| Req ID | Description | SRS Reference | Status | Verification / Notes |
|---|---|---|---|---|
| FR-CAND-01 | Candidate nomination and application entry | SRS §3.3 | `NOT STARTED` | Candidate submission endpoint |
| FR-CAND-02 | Approved public profiles (photo, bio, manifesto); restricted private data | SRS §3.3 | `NOT STARTED` | Public vs private DTO separation |
| FR-CAND-03 | Candidate review, approval, rejection, and withdrawal workflows | SRS §3.3 | `NOT STARTED` | Admin approval handler |
| FR-CAND-04 | Verify candidate eligibility against election-specific criteria | SRS §3.3 | `NOT STARTED` | Eligibility rule check |
| FR-CAND-06 | Strictly prevent candidate accounts from viewing voter identities or choices | SRS §3.3 | `NOT STARTED` | RBAC barrier |

---

## 4. Custom Ballot Builder (FR-BALLOT)

| Req ID | Description | SRS Reference | Status | Verification / Notes |
|---|---|---|---|---|
| FR-BALLOT-01 | Visual/structured ballot builder in CMS | SRS §3.4 | `NOT STARTED` | CMS ballot configuration UI |
| FR-BALLOT-02 | Configurable positions and referendum questions | SRS §3.4 | `NOT STARTED` | Position & question entities |
| FR-BALLOT-03 | Single-choice, multiple-choice, Yes/No, Approve/Reject options | SRS §3.4 | `NOT STARTED` | Question type enum |
| FR-BALLOT-05 | Seat limits, min/max selections, blank selection policy | SRS §3.4 | `NOT STARTED` | Validation rules per question |
| FR-BALLOT-06 | Pre-opening ballot validation (no empty positions, valid limits) | SRS §3.4 | `NOT STARTED` | Ballot verification validator |
| FR-BALLOT-08 | Immutable versioned ballot definitions post-open | SRS §3.4 | `NOT STARTED` | Ballot versioning |

---

## 5. Voter Eligibility (FR-ELIG)

| Req ID | Description | SRS Reference | Status | Verification / Notes |
|---|---|---|---|---|
| FR-ELIG-01 | Election-specific independent eligibility rules | SRS §3.5 | `NOT STARTED` | Rule engine |
| FR-ELIG-02 | Restrict by faculty, department, programme, year, class, or roster | SRS §3.5 | `NOT STARTED` | Attribute matching |
| FR-ELIG-03 | CSV roster import with validation, duplicate detection, and preview | SRS §3.5 | `NOT STARTED` | Roster parser & staging table |
| FR-ELIG-04 | Backend server-side eligibility check on ballot access and submission | SRS §3.5 | `NOT STARTED` | Verification gate in vote handler |
| FR-ELIG-08 | Multi-election eligibility isolation (eligible for X, ineligible for Y) | SRS §3.5 | `NOT STARTED` | Scoped eligibility query |

---

## 6. Voting, Duplicate Prevention, and Secrecy (FR-VOTE & FR-PRIV)

| Req ID | Description | SRS Reference | Status | Verification / Notes |
|---|---|---|---|---|
| FR-VOTE-01 | Ballot access only during open election window for eligible voters | SRS §3.6 | `NOT STARTED` | Access control gate |
| FR-VOTE-04 | Exactly one accepted ballot per eligible voter per election | SRS §3.6 | `NOT STARTED` | Unique participation constraint |
| FR-VOTE-05 | Prevent double-click, replay, retry, and concurrent duplicate submissions | SRS §3.6 | `NOT STARTED` | DB transaction + unique index |
| FR-VOTE-07 | Atomic acceptance: record participation AND ballot choice, or neither | SRS §3.6 | `NOT STARTED` | Atomic DB transaction rollback |
| FR-VOTE-09 | Reject malformed ballots, invalid options, and closed-election votes | SRS §3.6 | `NOT STARTED` | Ballot submission validator |
| FR-VOTE-10 | Generic submission confirmation with no encoded choices | SRS §3.6 | `NOT STARTED` | Blind confirmation receipt |
| FR-PRIV-01 | Never expose a voter's choices to anyone (candidates, public, admins) | SRS §3.7 | `NOT STARTED` | Decoupled tables & queries |
| FR-PRIV-02 | Structural schema separation of voter identity, participation, and ballot | SRS §3.7 | `NOT STARTED` | `voter_participations` vs `anonymous_ballots` |
| FR-PRIV-03 | Ordinary schema, API, logs, and admin views have no join path voter-to-ballot | SRS §3.7 | `NOT STARTED` | Architecture review & threat model |

---

## 7. Tallying, Results, and Audit (FR-TALLY & FR-DASH & FR-AUDIT)

| Req ID | Description | SRS Reference | Status | Verification / Notes |
|---|---|---|---|---|
| FR-TALLY-01 | Deterministic automatic tallying using documented rules | SRS §3.8 | `NOT STARTED` | Tally calculation service |
| FR-TALLY-03 | Explicit percentage denominators (vote share vs turnout) | SRS §3.8 | `NOT STARTED` | Mathematical formula tests |
| FR-TALLY-06 | Tie detection and adherence to approved tie procedure | SRS §3.8 | `NOT STARTED` | Tie detection flag |
| FR-TALLY-10 | Controlled recount reproducibility without altering original ballots | SRS §3.8 | `NOT STARTED` | Recount audit trail |
| FR-TALLY-12 | Versioned result snapshots (provisional vs certified) | SRS §3.8 | `NOT STARTED` | Result snapshot table |
| FR-DASH-04 | Realtime or polling aggregate results display based on election policy | SRS §3.9 | `NOT STARTED` | Aggregate-only API endpoint |
| FR-DASH-07 | Per-election results policy (live during voting, after close, or certified) | SRS §3.9 | `NOT STARTED` | Policy enforcement gate |
| FR-AUDIT-01 | Immutable audit log of administrative actions | SRS §3.10 | `NOT STARTED` | Audit event table |
| FR-CERT-02 | Formal multi-step election certification workflow | SRS §3.10 | `NOT STARTED` | Certification approval route |
