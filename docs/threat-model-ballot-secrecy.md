# Threat Model: Ballot Secrecy — ISSUE-07

## Executive Summary

This threat model analyzes the risks to ballot secrecy in the campVote system. It identifies potential attack vectors, assesses their impact, and documents the mitigations in place. The system is designed for academic campus elections and is not certified for high-stakes official elections.

## Threat Actors

### 1. Database Administrator
**Capabilities:**
- Full database access (SELECT, INSERT, UPDATE, DELETE on all tables)
- Can read logs and backups
- Can modify database configuration

**Attack Vectors:**
- Direct SQL queries to join participation and ballot tables
- Export of complete database
- Modification of audit logs to hide activity
- Correlation of timestamps between tables

**Mitigations:**
- Schema separation: `VoterParticipation` has no FK to `AnonymousBallot`
- No shared columns (user_id, email, etc.) between the tables
- Ballot IDs are random UUIDs, not sequential
- Timestamps are second-precision, not microsecond
- Audit logging of all privileged access
- Role separation where practical (different DB roles for participation vs ballot)
- Documented policy: DB admin access is logged and reviewed

**Residual Risk:** Medium - A determined DB admin with full privileges could potentially correlate data through timing analysis, but this would leave audit trails and requires out-of-band effort.

### 2. Application Developer
**Capabilities:**
- Access to application code
- Debug logging access
- Can deploy new versions
- Can add instrumentation or logging

**Attack Vectors:**
- Add logging that records choices alongside user identity
- Modify code to add correlation columns
- Introduce bugs that leak data
- Access debug logs containing sensitive data

**Mitigations:**
- Code review process (AGENTS.md rules)
- Privacy rules in AGENTS.md explicitly forbid logging choices with identity
- Automated log scanning for forbidden patterns
- No secrets or PII in code
- Review of all logging statements in ballot-related code
- Test coverage for privacy violations

**Residual Risk:** Low - Developer changes go through review and CI; malicious code would be detected in review or audit.

### 3. System Administrator / DevOps
**Capabilities:**
- Server access
- Log file access
- Network monitoring
- Can install monitoring or instrumentation

**Attack Vectors:**
- Packet capture to correlate voting traffic with users
- Access to application logs
- Memory dumps of running processes
- Install custom monitoring that leaks data

**Mitigations:**
- HTTPS/TLS for all traffic (in transit encryption)
- Application logs do not contain ballot choices
- Process memory is transient; not persisted by default
- Monitoring is through standard tools (not custom logging of request bodies)
- Session-based authentication; no persistent tokens with choice data

**Residual Risk:** Low-Medium - Network-level correlation is possible with additional context (ISP-level cooperation), but requires capabilities beyond typical system admin.

### 4. Insider (Election Officer)
**Capabilities:**
- Access to admin dashboard
- Can view turnout counts
- Can access eligibility rosters
- Can see audit logs

**Attack Vectors:**
- Cross-reference eligibility roster with turnout to identify non-voters
- Use knowledge of who is eligible to infer choices in small elections
- Access to timing data in audit logs

**Mitigations:**
- Admin dashboard shows only counts, not per-voter lists
- Audit logs do not contain ballot choices
- Small-group suppression on turnout display
- Officer actions are themselves audited
- Policy restrictions on data access

**Residual Risk:** Medium - In very small elections (e.g., <10 voters), an officer with the eligibility roster could potentially correlate who voted with eligibility to infer some information, but not specific choices.

### 5. External Attacker (Network)
**Capabilities:**
- Can intercept network traffic
- Can attempt injection attacks
- Can attempt to exploit vulnerabilities

**Attack Vectors:**
- Man-in-the-middle to read ballot submission
- SQL injection to exfiltrate data
- XSS to steal session tokens
- CSRF to force unwanted votes

**Mitigations:**
- HTTPS/TLS for all API endpoints
- Parameterized queries (GORM) prevent SQL injection
- CSRF protection on state-changing operations
- Input validation and sanitization
- Session-based authentication with secure cookies
- Rate limiting to prevent brute force

**Residual Risk:** Low - Standard web security practices are in place; TLS protects network traffic.

### 6. Browser Extension / Malware
**Capabilities:**
- Can read browser memory
- Can access localStorage/sessionStorage
- Can intercept network requests
- Can inject scripts

**Attack Vectors:**
- Read in-memory ballot selections before submission
- Access stored tokens or session data
- Capture network requests with ballot data

**Mitigations:**
- Ballot selections are never persisted to localStorage, sessionStorage, IndexedDB, or cookies
- Selections live only in React component state (transient)
- No choice data in URLs or query strings
- Submission over HTTPS protects network requests
- CSP headers restrict script injection

**Residual Risk:** Low-Medium - Malicious extension could read in-memory state during voting, but cannot correlate it with identity without additional context (they don't have the server-side user mapping).

## Attack Scenarios

### Scenario 1: Timestamp Correlation
**Description:** Attacker correlates the timestamp in `VoterParticipation.VotedAt` with `AnonymousBallot.CastAt` to link a user to their ballot.

**Feasibility:** Possible but difficult
- Timestamps are second-precision, not microsecond
- No sequential ordering of ballots
- Multiple votes may cast in the same second
- Requires knowledge of which users were voting at which times

**Mitigation:** 
- Coarse timestamps (second precision)
- Random ballot IDs
- No sequential ballot numbers
- If needed: add small random delay to ballot insertion to decorrelate timing

**Residual Risk:** Low - Would require precise timing knowledge and additional context.

### Scenario 2: Sequential Ballot IDs
**Description:** Attacker uses sequential ballot IDs to infer voting order and correlate with other data.

**Feasibility:** Not possible
- Ballot IDs are random UUIDs (generated by `ids.New()`)
- No sequential numbering or timestamp-based IDs

**Mitigation:** Already implemented via UUID generation

**Residual Risk:** None

### Scenario 3: Log Leakage
**Description:** Application logs accidentally record ballot choices alongside user identity or IP address.

**Feasibility:** Possible if code change introduces this
- Debug logging could expose request bodies
- Error handlers might log sensitive data

**Mitigation:**
- Code review explicitly checks for privacy violations
- Automated test scans logs for forbidden patterns
- AGENTS.md rule #9 explicitly forbids this
- Test case: inject synthetic votes and check logs

**Residual Risk:** Low - Review process and automated checks prevent this.

### Scenario 4: Backup/Export Correlation
**Description:** Attacker obtains a database backup and correlates data across tables.

**Feasibility:** Difficult but not impossible with full DB access
- Full backup contains both tables
- Attacker can run arbitrary queries offline
- No technical barrier to correlation queries

**Mitigation:**
- Schema makes correlation non-trivial (no FKs, no shared identifying columns)
- Timestamp coarsening
- Random IDs
- Access to backups is restricted and audited
- Backup access is logged
- If needed: encrypt backups with keys not available to DB admin

**Residual Risk:** Medium - Requires DB backup access and effort to correlate; audit trail exists.

### Scenario 5: Browser Storage Persistence
**Description:** Ballot selections are saved to browser storage, allowing later extraction.

**Feasibility:** Possible if implementation stores selections
- localStorage is readable by any script on the domain
- SessionStorage persists for the session
- Cookies can be read

**Mitigation:**
- Ballot selections are only in React component state (in-memory)
- Never saved to localStorage, sessionStorage, IndexedDB, or cookies
- No choice data in URLs or history
- Browser test verifies no storage after voting

**Residual Risk:** None - Implementation avoids persistence.

### Scenario 6: Receipt with Choices
**Description:** Confirmation page shows choices, creating a receipt that proves how the voter voted.

**Feasibility:** Possible if UI shows choices after submission
- User could screenshot or print
- Browser could save the page

**Mitigation:**
- Confirmation shows only that a vote was recorded and when
- No choice echo on confirmation page
- No "download my ballot" or "print my vote" feature
- AGENTS.md rule #9 explicitly forbids this

**Residual Risk:** None - UI does not provide choice receipt.

## Small Electorate Problem

### Problem
In elections with very few voters (e.g., 5-10 people), turnout counts alone may reveal who voted:
- If 5 people are eligible and 3 voted, the 2 who didn't vote are identifiable
- If turnout is shown per faculty/department and a department has 3 eligible voters, revealing 2 voted identifies the non-voter

### Mitigation
- Configurable small-group suppression threshold
- When group size < threshold, display "Hidden to protect privacy" instead of counts
- This is a policy decision; threshold should be set based on institution requirements

### Residual Risk
Medium - This is a fundamental limitation of any system that records participation. The suppression threshold reduces but does not eliminate the risk in very small elections.

## Recommendations

### Immediate (Implemented)
- ✅ Schema separation (no FKs between participation and ballot)
- ✅ No identifying columns in ballot table
- ✅ Random ballot IDs
- ✅ Coarse timestamps
- ✅ No choice persistence in browser
- ✅ No choice receipt in UI
- ✅ Audit logging of admin actions
- ✅ Automated schema regression test

### Short-term (Recommended)
- 🔄 Configure small-group suppression threshold (e.g., groups < 10)
- 🔄 Add log scanning for forbidden patterns in CI
- 🔄 Document backup access policy and retention
- 🔄 Add random delay to ballot insertion to further decorrelate timing
- 🔄 Implement database role separation (participation role vs ballot role)

### Long-term (If Required for Higher Stakes)
- 📋 End-to-end verifiable cryptography (voter can verify without revealing)
- 📋 Multi-party computation for ballot decryption
- 📋 Voter-verified paper audit trail (VVPAT)
- 📋 Independent security audit and certification
- 📋 Formal threat modeling by external security firm

## Comparison to Standards

### ISO/IEC 27001 (Information Security)
- Partial compliance: Access control, logging, and separation of duties are implemented
- Gap: No formal ISMS, no external certification

### Common Criteria (EAL)
- Not compliant: No formal security target or evaluation

### NIST Voluntary Voting System Guidelines (VVSG)
- Partial compliance: Many principles are followed (separation, audit, logging)
- Gap: No end-to-end verifiability, no paper trail, no independent testing

## Conclusion

The campVote system implements reasonable privacy protections for an academic campus election prototype:

**Strengths:**
- Clean schema separation prevents trivial correlation
- No identifying data in ballot table
- No choice persistence in browser
- Audit logging of privileged access
- Automated regression testing

**Limitations:**
- Not certified for high-stakes elections
- Small electorate risk remains (mitigated by suppression threshold)
- Database superuser could theoretically correlate with effort
- No end-to-end cryptographic verifiability

**Residual Risk Assessment:** Medium-Low
- For typical campus elections (hundreds to thousands of voters): Low risk
- For very small elections (<20 voters): Medium risk (due to turnout counts)

The system is appropriate for its stated purpose (academic prototype) but should not be used for official, legally binding elections without additional security measures and independent certification.
