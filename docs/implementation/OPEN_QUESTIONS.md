# Open questions requiring confirmation

Do not silently guess these. Create/link issues and record decisions in `ADR.md` or an additional ADR.

- [ ] **Institutional identity:** Does MMU provide approved SSO/OIDC or another identity verification service?
- [ ] **Eligibility data:** Who is authoritative for current student/staff status, faculty, department, programme, class, year, and role? How often is it updated?
- [ ] **Eligibility identity:** ISSUE-04 uses verified institutional email as the identity key and `student_number` as the per-election secondary key. Confirm this is acceptable, or require SSO/OIDC to assert student number and status.
- [x] **Revocation after a vote (decided):** a ballot already accepted stands when the voter is later revoked; the revocation only blocks further ballots. See `docs/eligibility.md` §8.
- [ ] **Eligibility freeze:** should the roster freeze automatically when polls open, or remain a deliberate officer action (current behaviour)?
- [ ] **Roster import approval:** who may import a roll, and is a second approver required before it becomes authoritative?
- [ ] **Email domains:** Confirm the proposed `@std.must.ac.ug` and `@must.ac.ug` domains with university IT; domain possession alone does not prove eligibility.
- [ ] **Roster import fallback:** Who can upload rosters, what validation/approval is required, and how are disputes and corrections handled?
- [ ] **Election rules:** Who approves nomination rules, ballot limits, tie handling, election extensions, recounts, and certification?
- [ ] **Results visibility:** For each election, are results live provisional, released after closing, or published after certification?
- [ ] **Ballot privacy model:** What privacy guarantee is required for the assignment, and what correlation risks remain through timestamps, access logs, tokens, and backups?
- [ ] **Deployment:** Where will the academic demo run, who owns the credentials, and what data-retention/backup policy applies?
- [ ] **Notifications:** Is email needed for the MVP, and which approved provider or local preview service will be used?
- [ ] **Scope boundary:** Which features are mandatory for the assessment and which are optional stretch goals?
