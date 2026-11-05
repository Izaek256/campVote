# Ballot Secrecy and Privacy Boundary — ISSUE-07

## Overview

This document describes the privacy guarantees and limitations of the campVote ballot secrecy implementation. It is intended for stakeholders, auditors, and users.

## What Is Protected

1. **Ballot Choices**: Your selections (which candidates you voted for) are stored anonymously in the `AnonymousBallot` table. This table contains no identifying information such as:
   - User ID
   - Email address
   - Student number
   - IP address
   - Session ID
   - Browser user agent

2. **Separation of Concerns**: The system keeps two separate tables:
   - `VoterParticipation`: Records ONLY who voted and when (for eligibility enforcement and turnout counts)
   - `AnonymousBallot`: Records ONLY what was voted (for tallying)
   
   These tables have no foreign key relationship. It is not possible to join them through the database schema.

3. **No Receipt**: The confirmation after voting states only that a vote was recorded and when. It does not show or encode your choices in any way that could be used as a receipt.

## What Is Not Guaranteed

1. **100% Anonymity**: This is an academic prototype, not a certified election system. While the architecture follows best practices for ballot secrecy, it has not undergone independent security certification or formal cryptographic audit.

2. **Small Electorates**: In very small elections (e.g., fewer than 10 voters), turnout counts alone may reveal who voted. The system applies small-group suppression thresholds where configured, but this is a policy decision, not a technical guarantee.

3. **Database Superuser**: A database administrator with full privileges could potentially:
   - Correlate timestamps between participation and ballot tables
   - Modify data to re-link records
   - Access logs that might contain correlation data
   
   This risk is mitigated by:
   - Timestamp coarsening (second precision, not microsecond)
   - No sequential or predictable ballot IDs
   - Audit logging of privileged access
   - Separation of database roles where practical

4. **Timing Correlation**: If an attacker can observe both:
   - The exact time a specific user accessed the voting interface
   - The exact time a ballot was cast
   
   They might infer a correlation. This risk is reduced by:
   - Random ballot IDs
   - Coarse timestamps
   - No sequential ordering of ballots

5. **Network/Traffic Analysis**: Network-level observers (ISP, network administrators) could potentially correlate voting traffic with individual users if they have additional context.

## Who Can See Participation (That You Voted)

The following roles can see that you voted in an election, but NOT what you voted for:

- **Election Officers**: Can view turnout counts and eligibility lists
- **Auditors**: Can access participation records for audit purposes
- **System Administrators**: Have database access but are bound by policy and audit logging

These roles see:
- Count of eligible voters
- Count of voters who have participated
- Turnout percentage
- Per-faculty or per-department aggregates (if group size exceeds suppression threshold)

They do NOT see:
- A list of which specific individuals voted (unless explicitly required by policy for audit purposes)
- Any ballot choices or selections

## Who Cannot See Choices

The following roles NEVER see ballot choices tied to identity:

- **Candidates**: Never see who voted for them or any choice data
- **Other Voters**: Never see any other voter's choices
- **Public**: Only see aggregate results (counts and percentages) when results are published
- **Network/Access Logs**: Do not contain ballot choices (enforced by code review and logging policy)

## Technical Safeguards

### Database Schema

- `AnonymousBallot` table has no user_id, email, student_number, IP, session_id, or user_agent columns
- `VoterParticipation` table has no foreign key to `AnonymousBallot` or `BallotSelection`
- Ballot IDs are random UUIDs, not sequential or timestamp-based
- Timestamps are stored with second precision, not microsecond precision
- Automated regression test (TestPrivacySchemaRegression) runs in CI to detect forbidden columns or foreign keys

### Application Code

- Server-side validation enforces one ballot per eligible voter
- Logs are reviewed to ensure no ballot choices are logged alongside identity
- Error messages never leak choice data or user correlation
- Ballot selections live only in in-memory component state during voting; they are never stored in browser storage, URLs, or cookies

### Audit Trail

- All administrative actions are logged to `ElectionAuditLog`
- Audit entries never contain ballot choices
- Access to participation records is itself audited
- Privileged database access is logged where possible

## Residual Risks

1. **Small Electorates**: Turnout counts in elections with very few voters may be identifying. Mitigation: configurable small-group suppression threshold.

2. **Database Superuser**: A compromised or malicious database administrator could theoretically correlate data across tables. Mitigation: role separation, audit logging, restricted access.

3. **Log Correlation**: If logs accidentally include timing data for both participation and ballot insertion, correlation might be possible. Mitigation: code review, automated log scanning for forbidden patterns.

4. **Memory/Process Dump**: If the application crashes and a memory dump is captured, it might contain in-memory ballot selections alongside session data. Mitigation: process memory is transient; dumps are not retained by default.

5. **Browser Storage**: Malicious browser extensions could potentially read in-memory state. Mitigation: selections are never persisted to localStorage, sessionStorage, IndexedDB, or cookies.

## Comparison to Certified Systems

Certified high-stakes election systems typically include additional security measures not implemented here:

- **End-to-End Verifiable Cryptography**: Voters can verify their vote was counted without revealing it
- **Coercion Resistance**: Protocols to prevent vote-buying or forced voting
- **Independent Audit**: Formal cryptographic audit by external security firms
- **Multi-Party Computation**: Threshold cryptography where no single party can decrypt votes
- **Paper Trail**: Voter-verified paper audit trails (VVPAT)

These are out of scope for this academic prototype unless explicitly required by the SRS.

## References

- SRS.md (Software Requirements Specification)
- ADR.md (Architecture Decision Records)
- AGENTS.md (Development rules and privacy constraints)
- models/ballot.go (AnonymousBallot implementation)
- models/participation.go (VoterParticipation implementation)
- models/privacy_test.go (Automated regression test)

## Statement for UI

The following plain-language statement is approved for use in the ballot secrecy page and privacy notice:

---

**Your Ballot Privacy**

campVote is designed to protect your ballot choices. Here's what you should know:

**What is protected:**
- The specific candidates or options you select are stored anonymously
- Election officers can see turnout counts but cannot see individual choices
- Candidates never see who voted for them
- Your confirmation receipt only shows that you voted, not what you voted for

**What is not guaranteed:**
- This is an academic prototype, not a certified election system
- In very small elections, turnout counts alone might reveal who participated
- Database administrators with full access have technical capability to access data, though this is restricted by policy and audit

**What we do to protect you:**
- Ballot choices are stored separately from who voted
- No identifying information (email, student number, IP address) is stored with your choices
- Your selections are never saved in your browser or on your device
- All administrative access is logged for audit

If you have concerns about privacy, contact the election administrator or the MMU Electoral Commission.

---

This statement must be used verbatim in the UI. Do not add words like "100% anonymous" or "completely untraceable" without the risk qualifiers documented above.
