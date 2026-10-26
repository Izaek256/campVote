# Software Requirements Specification (SRS)
## MMU Campus E-Voting System

| Document field | Value |
|---|---|
| Project | MMU Campus E-Voting System |
| Institution | Mountains of the Moon University (MMU) |
| Version | 1.1 |
| Status | Proposed baseline for review |
| Date | 24 October 2026 |
| Scope | Guild elections and other authorized campus elections |

## 1. Introduction

### 1.1 Purpose
This Software Requirements Specification defines the functional, security, privacy, performance, operational, and acceptance requirements for a web-based electronic voting platform for Mountains of the Moon University (MMU). It is intended for university stakeholders, developers, testers, election officials, system administrators, and maintainers.

### 1.2 Scope
The system will primarily support university guild elections and also smaller elections conducted by faculties, departments, programmes, classes, clubs, student associations, committees, and other authorized campus bodies. It will provide an integrated election CMS and dashboard, candidate management, custom ballots, institutional authentication, eligibility rules, secret ballots, automated tallying, live or delayed results, reports, and auditable administration.

The platform is a university/campus system, not a national election platform. It will be browser-based and responsive on desktop, tablet, and mobile devices.

### 1.3 Objectives
- Digitize and simplify guild and campus elections.
- Authenticate approved institutional users and verify their actual eligibility.
- Allow election-specific restrictions based on verified student/staff attributes.
- Enforce one accepted ballot per eligible voter per election by default.
- Keep ballot selections secret from candidates and ordinary administrators.
- Count votes automatically using documented, tested, reproducible rules.
- Publish appropriately transparent turnout and results information.
- Allow authorized administrators to configure elections without changing application code.
- Preserve election records, audit evidence, and approved final results.

### 1.4 Definitions
- **Election:** A defined voting event with a schedule, eligibility policy, ballot, candidates/options, and tallying rules.
- **Voter:** A verified person eligible to participate in a particular election.
- **Ballot:** The set of selections submitted for an election.
- **Participation record:** A record that an eligible person has submitted a ballot, without storing their selections.
- **Secret ballot:** A ballot designed so a voter's identity cannot be linked to their selections through ordinary application records or interfaces.
- **Turnout:** Number of participating eligible voters divided by the eligible population, using the election's approved denominator.
- **Provisional result:** A result published before formal certification.
- **Certified result:** A result reviewed and formally approved under university procedures.

## 2. Overall Description

### 2.1 User classes
1. **System Administrator:** Manages technical configuration, accounts, security, backups, and operations. This role does not automatically confer access to ballot choices.
2. **Chief Election Administrator:** Configures and oversees elections, candidates, eligibility, schedules, publication, and certification.
3. **Election Official:** Performs explicitly delegated tasks for specified elections.
4. **Student Voter:** Votes when authenticated and eligible.
5. **Staff/Official Voter:** Participates only in elections that permit their verified account class.
6. **Candidate:** Maintains permitted profile information and views public statistics/results; cannot access voter identities or choices.
7. **Public Viewer:** Views public election information and only the aggregate results permitted by the election's publication policy.
8. **Election Observer/Auditor (optional role):** Reviews authorized aggregate evidence and audit records without receiving access to secret ballots.

### 2.2 Operating environment
- Current versions of Chrome, Edge, Firefox, and Safari.
- Responsive desktop, laptop, tablet, and mobile web interfaces.
- HTTPS-enabled application deployment.
- PostgreSQL relational database for production.
- Docker-based local development and reproducible deployment.
- University-approved hosting, email, identity, and directory integrations where available.

### 2.3 Constraints and assumptions
- Initial email policy: students use `@std.must.ac.ug`; staff and officials use `@must.ac.ug`. The university must confirm these exact domains and any exceptions.
- A domain match is not proof of current enrollment, employment, faculty, department, class, or voting eligibility.
- The institution must identify an authoritative student/staff directory, roster, or approved eligibility-import process.
- All times are stored unambiguously and displayed in East Africa Time (EAT, UTC+3) by default.
- Election rules, tie resolution, disputes, retention, and certification must be approved by the university.
- Live results are configurable because publicizing leaders during voting can influence voters who have not voted.

## 3. Functional Requirements

### 3.1 Authentication and authorization
- **FR-AUTH-01:** Permit protected access only to identities authenticated through an approved institutional identity flow.
- **FR-AUTH-02:** Apply the approved student domain policy (`@std.must.ac.ug`) and staff/official domain policy (`@must.ac.ug`), configurable by authorized administrators.
- **FR-AUTH-03:** Validate the verified email/identity claims server-side; never trust a client-supplied email or role.
- **FR-AUTH-04:** Prefer university SSO/OIDC or another trusted identity integration where available.
- **FR-AUTH-05:** Verify current institutional status and relevant membership attributes using a trusted directory or an approved, controlled roster-import process.
- **FR-AUTH-06:** Enforce role-based access control and least privilege on every protected backend operation.
- **FR-AUTH-07:** Support account suspension/revocation, session expiry, logout, and security-event logging.
- **FR-AUTH-08:** Protect privileged accounts with MFA where supported by the identity provider.
- **FR-AUTH-09:** Do not log passwords, access tokens, secret ballot contents, or other authentication secrets.

### 3.2 Election CMS and lifecycle
- **FR-CMS-01:** Authorized administrators can create, edit, schedule, publish, close, cancel, and archive elections.
- **FR-CMS-02:** Election configuration includes title, description, category, organizing unit, dates, time zone, eligibility rules, ballot structure, tally method, result visibility, instructions, and status.
- **FR-CMS-03:** Support guild, faculty, department, programme, class, club, association, committee, referendum, and configurable election categories.
- **FR-CMS-04:** Support multiple simultaneous elections with isolated candidates, eligible populations, ballots, results, and permissions.
- **FR-CMS-05:** Provide lifecycle states: Draft, Registration, Pending Approval, Scheduled, Open, Closed, Results Under Review, Finalized, Archived, and Cancelled.
- **FR-CMS-06:** Validate election configuration before opening.
- **FR-CMS-07:** Allow reusable templates for common election types.
- **FR-CMS-08:** Preserve configuration versions and audit changes. After voting opens, changes affecting eligibility, ballot interpretation, candidates, or tallying are prohibited by default; any exception requires formal authorization and an audit record.
- **FR-CMS-09:** Open and close elections according to server-enforced schedules.
- **FR-CMS-10:** Publish notices, schedules, instructions, and approved updates.

### 3.3 Candidate management
- **FR-CAND-01:** Support official entry of candidates and/or candidate applications.
- **FR-CAND-02:** Store approved public profile fields: name, photograph, biography, manifesto, election, position, and application status. Keep private identifiers and email addresses restricted.
- **FR-CAND-03:** Support review, approval, rejection, correction requests, and withdrawal under election rules.
- **FR-CAND-04:** Verify candidate eligibility using election-specific rules and trusted attributes.
- **FR-CAND-05:** Display approved candidate information to eligible voters and public viewers as permitted.
- **FR-CAND-06:** Prevent candidates from accessing voter lists, individual participation histories, or ballot choices.

### 3.4 Custom ballot builder
- **FR-BALLOT-01:** Provide a visual ballot builder inside the CMS.
- **FR-BALLOT-02:** Support configurable positions and questions, not only hardcoded guild roles.
- **FR-BALLOT-03:** Support single-choice, multiple-choice, Yes/No, Approve/Reject, referendum, and configurable answer-option questions.
- **FR-BALLOT-04:** Support ranked-choice voting only when its counting method and tie rules are explicitly selected, documented, and tested.
- **FR-BALLOT-05:** Configure minimum/maximum selections, number of seats, instructions, and whether a question may be left blank.
- **FR-BALLOT-06:** Validate missing options, invalid selection limits, missing required positions, and other ballot errors before an election opens.
- **FR-BALLOT-07:** Preview ballots in desktop and mobile layouts.
- **FR-BALLOT-08:** Version the approved ballot definition. Do not silently reinterpret submitted ballots after voting opens.

### 3.5 Eligibility and election restrictions
- **FR-ELIG-01:** Maintain independent eligibility rules for every election.
- **FR-ELIG-02:** Restrict participation by verified user class, faculty, department, programme, year, class, membership, or approved roster.
- **FR-ELIG-03:** Allow approved CSV imports of eligible voters, with schema validation, duplicate detection, error reporting, and preview before commit.
- **FR-ELIG-04:** Check eligibility on the backend at ballot access and submission; hiding a ballot in the UI is not sufficient.
- **FR-ELIG-05:** Restrict access to confidential eligibility lists to authorized roles.
- **FR-ELIG-06:** Log eligibility data changes, especially after an election is scheduled or opened.
- **FR-ELIG-07:** Handle suspended, graduated, withdrawn, or otherwise ineligible accounts according to approved institutional data.
- **FR-ELIG-08:** Make it possible for a user to be eligible for one election and ineligible for another.

### 3.6 Voting and duplicate prevention
- **FR-VOTE-01:** Allow ballot access only if the election is open and the user is authenticated and eligible.
- **FR-VOTE-02:** Present the ballot version approved for that election.
- **FR-VOTE-03:** Allow voters to review selections before submission.
- **FR-VOTE-04:** Enforce one accepted ballot per eligible voter per election by default. Multi-position elections normally submit one ballot containing selections for all positions.
- **FR-VOTE-05:** Prevent duplicate acceptance from double-clicks, retries, page refreshes, concurrent requests, or replayed requests.
- **FR-VOTE-06:** Enforce duplicate prevention using server-side transactions and database constraints, not frontend checks alone.
- **FR-VOTE-07:** Atomically accept the ballot and record participation, or accept neither. The implementation must not leave a voter marked as voted without a valid ballot or count a ballot twice.
- **FR-VOTE-08:** Recheck election status, eligibility, ballot validity, and participation state at submission time.
- **FR-VOTE-09:** Reject malformed ballots, invalid selections, submissions for closed elections, and attempts to edit accepted ballots through ordinary application functions.
- **FR-VOTE-10:** Provide a generic acceptance confirmation that does not reveal or encode selections.
- **FR-VOTE-11:** If a network interruption leaves submission status uncertain, allow the voter to check whether a ballot was accepted without submitting a duplicate.
- **FR-VOTE-12:** Keep election data isolated so votes from one election cannot enter another election's tally.

### 3.7 Ballot secrecy and privacy
- **FR-PRIV-01:** Do not expose a voter's choices to candidates, public viewers, or ordinary election administrators.
- **FR-PRIV-02:** Separate identity/authentication data, eligibility data, participation records, and ballot contents.
- **FR-PRIV-03:** The ordinary schema, API, logs, analytics, and admin screens must not expose a direct voter-to-ballot mapping.
- **FR-PRIV-04:** Do not store candidate selections in user profiles, participation records, request logs, analytics events, or notifications.
- **FR-PRIV-05:** Use generic, non-transferable submission confirmation; do not issue receipts that prove a particular choice.
- **FR-PRIV-06:** Limit access to identity and eligibility records and protect data in transit and at rest as appropriate.
- **FR-PRIV-07:** Conduct a privacy threat assessment covering database access, privileged operators, backups, timing correlation, application logs, and infrastructure access.
- **FR-PRIV-08:** Document the actual anonymity guarantee and residual risks. Hiding names in the UI is not sufficient.
- **FR-PRIV-09:** Evaluate anonymous credentials, unlinkable voting tokens, or cryptographically verifiable secret ballots if stronger anonymity is required.
- **FR-PRIV-10:** Ensure the system cannot provide ordinary administrators with a convenient query that joins a named voter to ballot choices.

**Required design note:** A conventional database that records a user ID alongside ballot selections does not meet this SRS. Separating tables alone may also be insufficient if a common operator can correlate timestamps, logs, tokens, or backups. The architecture and threat model must explicitly address this.

### 3.8 Tallying and result integrity
- **FR-TALLY-01:** Calculate results automatically from accepted ballots using documented rules.
- **FR-TALLY-02:** Tally each position/question independently according to its configured method.
- **FR-TALLY-03:** Calculate vote percentages using an explicitly named denominator. For a single-choice position: candidate valid votes / total valid votes for that position × 100.
- **FR-TALLY-04:** Display turnout separately from candidate vote share.
- **FR-TALLY-05:** Treat blank selections, abstentions, invalid ballots, and multi-select questions according to documented election rules.
- **FR-TALLY-06:** Detect ties and apply only the approved tie procedure; never invent a winner.
- **FR-TALLY-07:** Produce identical results from the same accepted ballots, ballot configuration, and tally rules.
- **FR-TALLY-08:** Validate ballot structure and reject duplicate, malformed, or cross-election ballot records.
- **FR-TALLY-09:** Provide reconciliation statistics, including eligible population, participation count, accepted ballots, and valid selections where applicable.
- **FR-TALLY-10:** Support a controlled recount without editing or deleting original ballots.
- **FR-TALLY-11:** Record the reason, actor, time, rule version, and output of recounts and corrections.
- **FR-TALLY-12:** Preserve a final result snapshot and the configuration/rule version used to calculate it.
- **FR-TALLY-13:** Use independent test vectors with known expected results for every supported tally method.
- **FR-TALLY-14:** Use automated property, boundary, concurrency, and regression tests for tallying and ballot validation.
- **FR-TALLY-15:** Provide a reproducible tally process and a reconciliation report. Automatic counting alone is not proof of correctness.

### 3.9 Public dashboard and results
- **FR-DASH-01:** Provide public or university-accessible election pages according to each election's access policy.
- **FR-DASH-02:** Display election title, category, status, schedule, positions, approved candidates, and public manifestos.
- **FR-DASH-03:** Display permitted eligible-voter totals, participation counts, and turnout.
- **FR-DASH-04:** When live results are enabled, display current candidate totals, vote percentages, and the current leader for each applicable position.
- **FR-DASH-05:** Refresh results automatically without requiring a manual page reload.
- **FR-DASH-06:** Clearly label results as Live/Provisional, Polls Closed—Pending Verification, Under Review, or Certified Final.
- **FR-DASH-07:** Allow each election to select a results policy: live during voting, after polls close, or after certification.
- **FR-DASH-08:** Display the denominator for each percentage and distinguish turnout from vote share.
- **FR-DASH-09:** Never label a provisional leader as the winner before the election's rules permit it.
- **FR-DASH-10:** Expose only aggregate results and approved public information; do not expose voter identities, private participation histories, or ballots.
- **FR-DASH-11:** Provide permitted historical results and certified reports.

### 3.10 Notifications, reports, audit, certification
- **FR-OPS-01:** Support in-app and institutional email notifications for schedules, opening/closing, and official results where enabled.
- **FR-OPS-02:** Notifications must not disclose ballot choices or imply a particular choice.
- **FR-REPORT-01:** Generate election reports containing election metadata, candidates, participation, turnout, vote totals, percentages, status, and certification details.
- **FR-REPORT-02:** Export authorized reports to PDF and CSV.
- **FR-REPORT-03:** Ensure exported totals match the relevant result snapshot.
- **FR-AUDIT-01:** Record significant administrative events: election configuration, candidate decisions, roster imports, opening/closing, role changes, publication, certification, recounts, and exceptional interventions.
- **FR-AUDIT-02:** Protect audit records against unauthorized alteration and deletion.
- **FR-AUDIT-03:** Audit logs must not contain ballot selections or direct identity-to-choice links.
- **FR-AUDIT-04:** Restrict detailed audit access to authorized reviewers.
- **FR-CERT-01:** Stop accepting votes at the server-enforced closing time.
- **FR-CERT-02:** Allow final results to be certified only by authorized roles after the required review.
- **FR-CERT-03:** Preserve certified results against silent changes.
- **FR-CERT-04:** Reopening, cancellation, or post-opening changes require authorization, a documented reason, and an audit event.

## 4. External Interface Requirements

### 4.1 User interface
Provide separate, role-appropriate experiences for public viewers, voters, candidates, election officials, and system administrators. Use responsive layouts, clear voting instructions, keyboard navigation, accessible form controls, and explicit submission states.

### 4.2 Identity interface
Prefer a university-controlled OIDC/SSO integration. If unavailable, use an institution-approved verification mechanism and trusted roster imports. Do not treat domain matching alone as identity or eligibility proof.

### 4.3 Integration interfaces
Potential integrations include university SSO, student information systems, staff directories, institutional email, and university websites. Implement only with university authorization and supported interfaces.

## 5. Non-Functional Requirements

### 5.1 Security
- **NFR-SEC-01:** HTTPS for all production traffic.
- **NFR-SEC-02:** Server-side authorization on every protected operation.
- **NFR-SEC-03:** Use secure framework authentication features, safe cookie/session configuration, rate limiting, and MFA for privileged users where possible.
- **NFR-SEC-04:** Protect against injection, XSS, CSRF where applicable, broken access control, brute force, replay, and insecure direct object references.
- **NFR-SEC-05:** Store secrets outside source control; use separate development and production credentials.
- **NFR-SEC-06:** Apply least privilege to database and deployment accounts.
- **NFR-SEC-07:** Conduct dependency, configuration, and security testing before an official election.
- **NFR-SEC-08:** No single administrator should be able to silently change ballots, votes, tally rules, and certified results without detection.

### 5.2 Reliability and availability
- **NFR-REL-01:** Persist accepted ballots durably.
- **NFR-REL-02:** Maintain transactional consistency across ballot acceptance and participation recording.
- **NFR-REL-03:** Document and test backup restoration and disaster recovery.
- **NFR-REL-04:** Recovery must not duplicate ballots or silently lose accepted ballots.
- **NFR-REL-05:** Proposed availability target: at least 99.5% during scheduled election periods, excluding approved maintenance. Review this target against the university's risk tolerance.

### 5.3 Performance and scale
- **NFR-PERF-01:** Under an agreed normal load, 95% of ordinary page/API requests should complete within 2 seconds, excluding external identity-provider delays.
- **NFR-PERF-02:** Under an agreed normal load, 95% of ballot submissions should receive a response within 3 seconds.
- **NFR-PERF-03:** Where live publication is enabled, aggregate results should normally update within 5 seconds of acceptance.
- **NFR-PERF-04:** Establish capacity through load tests based on expected concurrent guild-election voters; do not claim capacity without measurement.
- **NFR-PERF-05:** Reporting/tally work must not unnecessarily block ballot submissions.

### 5.4 Usability and accessibility
- **NFR-UX-01:** A first-time voter should be able to complete a standard ballot without specialized training.
- **NFR-UX-02:** Target WCAG 2.2 AA accessibility.
- **NFR-UX-03:** Support current mainstream browsers and common smartphone screen sizes.
- **NFR-UX-04:** Make errors, selection limits, deadlines, and submission status clear.

### 5.5 Maintainability and testing
- **NFR-MAINT-01:** Use modular backend services and clear frontend feature boundaries.
- **NFR-MAINT-02:** Keep election rules configurable and documented.
- **NFR-MAINT-03:** Manage schema changes through version-controlled migrations.
- **NFR-MAINT-04:** Automate tests for authentication, eligibility, concurrency, duplicate prevention, ballot secrecy controls, tallying, publication, permissions, and recovery.
- **NFR-MAINT-05:** Pin and document framework/tool versions for reproducible builds.

### 5.6 Data protection
- **NFR-DATA-01:** Collect only information needed for approved functions.
- **NFR-DATA-02:** Define retention periods for identity, eligibility, participation, ballot, audit, and report records.
- **NFR-DATA-03:** Follow applicable Ugandan data-protection requirements and university policy.
- **NFR-DATA-04:** Have a documented incident and breach-response process.

## 6. Proposed Logical Architecture

1. **Web frontend:** voter pages, candidate profiles, public dashboard, and election information.
2. **Administrative frontend/CMS:** election configuration, candidate review, ballot builder, eligibility management, reports, and certification.
3. **Backend API:** enforces business rules, identity claims, permissions, schedules, ballot validation, and result access.
4. **Identity and eligibility service:** maps verified institutional identity and trusted attributes to election-specific eligibility.
5. **Election management module:** schedules, lifecycle, candidates, positions, ballot versions, and result policy.
6. **Voting subsystem:** validates and accepts ballots, prevents duplicate participation, and protects ballot secrecy.
7. **Tallying subsystem:** deterministic calculation, reconciliation, and result snapshots.
8. **Publication/realtime service:** exposes approved aggregate data only.
9. **Audit/reporting module:** protected administrative history and permitted reports.
10. **PostgreSQL:** durable production data with constraints, transactions, backups, and restricted credentials.

The logical separation is mandatory; the exact process/deployment split can be decided during implementation. The identity and ballot systems must be designed to reduce correlation risk, not merely separated in the UI.

## 7. Core Data Entities

- **User/Identity:** internal ID, verified institutional identity, account state, roles.
- **Institutional Profile:** approved student/staff classification and membership attributes.
- **Election:** metadata, schedule, lifecycle, visibility policy.
- **Eligibility Rule/Roster:** election-specific criteria and approved roster membership.
- **Candidate:** public profile and restricted application details.
- **Position/Question:** seat count, options, selection rules, tally method.
- **Ballot Definition Version:** approved structure and version.
- **Participation Record:** evidence of participation, separate from ballot choices.
- **Anonymous Ballot:** selections without a direct voter identity reference.
- **Result Snapshot:** totals, percentages, tally-rule version, state, certification.
- **Audit Event:** actor, action, target, timestamp, outcome, reason where applicable.
- **Notification:** message type, recipient, and delivery status.

### 7.1 Data constraints
- At most one accepted participation record per voter per election under the default policy.
- Each ballot belongs to one election and conforms to its approved ballot version.
- Each candidate belongs to the appropriate election/position.
- Ballots and participation updates are transactionally consistent.
- Finalized results reference the rules and ballot version used.
- Ordinary data access must not provide a direct join from voter identity to selections.

## 8. Key Use Cases

### UC-01 Create an election
1. Authorized administrator opens the CMS and creates an election.
2. Enters category, title, description, schedule, result policy, and eligibility.
3. Creates positions and configures ballots.
4. Adds candidates or enables applications.
5. System validates the configuration.
6. Administrator saves draft or submits for approval.
7. System records changes in the audit trail.

### UC-02 Approve a candidate
1. Candidate or official submits profile/application information.
2. System validates required fields and eligibility.
3. Authorized official approves, rejects, or requests corrections.
4. System records the decision and displays the permitted status.

### UC-03 Cast a secret ballot
1. Voter signs in using an approved institutional identity.
2. System checks eligibility for the selected election.
3. Voter reviews the ballot and selects answers.
4. Voter submits.
5. Backend rechecks election status, ballot validity, eligibility, and prior participation.
6. System atomically accepts the ballot and records participation separately.
7. Voter receives a generic confirmation.
8. Retry/status-check flows cannot create duplicate ballots.

### UC-04 Monitor results
1. Viewer opens a permitted election dashboard.
2. System shows election status, candidates, and approved aggregate statistics.
3. If enabled, live totals and percentages refresh automatically.
4. UI labels provisional and certified results correctly.
5. No private voter or ballot information is exposed.

### UC-05 Close and certify
1. Server stops accepting ballots at the closing time.
2. Tallying engine calculates provisional totals.
3. System runs integrity and reconciliation checks.
4. Authorized officials review results and approve certification.
5. System preserves the certified snapshot and audit event.
6. Results are published according to the configured policy.

## 9. Business Rules
- **BR-01:** Institutional domain approval is necessary but not sufficient for current membership or election eligibility.
- **BR-02:** Eligibility is election-specific.
- **BR-03:** One accepted ballot per voter per election is the default.
- **BR-04:** Candidate identities and voter choices remain confidential.
- **BR-05:** No election opens before mandatory validation succeeds.
- **BR-06:** Backend time and rules govern opening/closing, not the browser clock.
- **BR-07:** Percentages state their denominator and distinguish vote share from turnout.
- **BR-08:** A live leader is not automatically a winner.
- **BR-09:** Certified results cannot be silently overwritten.
- **BR-10:** Recounts and exceptional actions are authorized, documented, and auditable.

## 10. Acceptance Criteria

### Identity and eligibility
- Approved student and staff identities can authenticate under the university's confirmed policy.
- A non-approved domain is rejected.
- A valid domain with an inactive or ineligible institutional status cannot vote.
- Eligibility for one election does not grant eligibility for another.

### Vote integrity
- A valid voter can submit one ballot.
- Duplicate, concurrent, retried, malformed, and late submissions cannot create extra accepted ballots.
- An uncertain network response can be checked safely.
- Ballot acceptance and participation status cannot become inconsistent.

### Secrecy
- Candidates and public viewers cannot inspect individual choices.
- Ordinary admin screens and APIs do not expose identity-to-ballot mappings.
- Logs, metrics, and audit records do not disclose choices.
- A documented threat review addresses timing, privileged database access, backups, and correlation.

### Tally and results
- Known test ballots produce exact expected totals.
- Percentage denominators are correct and clearly labelled.
- Multi-position and referendum tallies follow their rules.
- Ties are detected and handled according to approved policy.
- Recounts are deterministic and match the accepted ballot set.
- Certified reports match the certified result snapshot.

### Operations
- Unauthorized election changes are rejected.
- Administrative actions are audited.
- Results are clearly labelled provisional or certified.
- Load, backup restoration, and incident-recovery tests pass before a real election.

## 11. Risks and Mitigations

| Risk | Mitigation |
|---|---|
| Email domain mistaken for proof of eligibility | Trusted identity/directory integration or controlled roster process |
| Duplicate votes | Transactional backend checks and unique database constraints |
| Identity/ballot correlation | Separate data flows, minimize metadata, restrict privileged access, assess timing and backups |
| Incorrect tally logic | Independent test vectors, property tests, reconciliation, deterministic recount |
| Administrative interference | Least privilege, separation of duties, protected audit trail, certification process |
| Live results influence voters | Per-election visibility policy |
| Election-day traffic overload | Load tests, capacity planning, monitoring, recovery runbook |
| Network uncertainty | Idempotent submission and status-check flow |
| Data leak | Minimal collection, access control, retention limits, incident response |
| Disputed results | Documented recount, certification, and dispute procedures |

## 12. Delivery Phases

### Phase 1 — Core platform
Institutional authentication, roles, election CMS, candidate management, custom ballot builder, eligibility, secret voting controls, duplicate prevention, automatic tallying, results dashboard, reports, and audit events.

### Phase 2 — Production readiness
Trusted directory/SSO integration, privacy threat assessment, independent security review, concurrency/load tests, recovery drills, recount/certification workflows, and operational runbooks.

### Phase 3 — Advanced verifiability
Cryptographic verification, independent observer workflows, advanced voting methods, expanded institutional integrations, and additional analytics.

## 13. Go-Live Gate
The system must not be used for an official guild election until:
1. The university confirms identity domains and authoritative eligibility data.
2. Election rules, tie procedures, disputes, and certification are approved.
3. Ballot secrecy has been threat-modelled and tested.
4. Duplicate prevention and tallying pass automated and independent test cases.
5. Load, backup restoration, and incident-response drills pass.
6. A responsible election authority signs off on the system and procedures.

## 14. References
- Grit Framework documentation: https://gritframework.dev/docs
- Grit quick start: https://gritframework.dev/docs/getting-started/quick-start
- Grit project creation: https://gritframework.dev/docs/getting-started/create-a-project
- The final implementation should also document applicable university policies and current Ugandan data-protection requirements during project governance review.
