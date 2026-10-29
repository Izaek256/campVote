# UI Views Registry

This document tracks all UI views in the campVote application, their routes, roles, API endpoints, and owning issues.

## ISSUE-06 Views

| View / Route | Roles | Data / API | Key Elements | States | Notes | Owning Issue |
|---|---|---|---|---|---|---|
| Positions Manager | `/staff/elections/:id/positions` | Officer | `GET/POST/PUT/DELETE /api/v1/staff/elections/:id/positions` | Table of positions; add/edit form with title, description, question type, seats, selection rules | Locked-after-publish banner; validation errors per field | Controls disabled with explanation once config is frozen; server still enforces | ISSUE-06 |
| Candidate Nominations Queue | `/staff/elections/:id/candidates` | Officer | `GET /api/v1/staff/elections/:id/candidates` (with status filter) | Status filter tabs (Nominated, Approved, Rejected, Withdrawn); row actions; detail drawer | Empty queue; eligibility-failed indicator from server | Approve/reject use confirm dialog with reason; show who acted and when | ISSUE-06 |
| Candidate Editor | `/staff/elections/:id/candidates/:candidate_id` | Officer; owning Candidate | `PUT /api/v1/staff/elections/:id/candidates/:candidate_id` | Name, position, manifesto, photo upload with type/size hints and preview | Upload rejected (type/size/content); unsaved-changes guard | Render manifesto with output encoding; no raw HTML injection | ISSUE-06 |
| My Candidacy | `/candidate` | Candidate | `GET /api/v1/protected/candidates/my` | Status timeline; withdraw button with consequence text | Not-nominated empty state; withdrawal-after-publish warning per documented policy | Never shows other candidates' private review notes | ISSUE-06 |
| Ballot Builder | `/staff/elections/:id/ballot` | Officer | `POST /api/v1/staff/elections/:id/ballot/publish` | Position sections with candidate list and ordering rule; validation panel listing every rule violation | Cannot publish while validation panel is non-empty | Only approved candidates selectable | ISSUE-06 |
| Publish Ballot Dialog | (modal) | Officer | `POST /api/v1/staff/elections/:id/ballot/publish` | Summary of positions/candidates, resulting version number and content hash, irreversible-action warning, reason field | Publish failure keeps draft intact | Shows hash after publish for later comparison | ISSUE-06 |
| Ballot Versions | `/staff/elections/:id/ballot/versions` | Officer, Auditor | `GET /api/v1/staff/elections/:id/ballot/versions` | Version list (number, hash, publisher, timestamp, status); side-by-side diff | Old versions remain viewable and read-only | Edits after publish route the user to "create new version" flow, never in-place edit | ISSUE-06 |
| Ballot Preview | `/staff/elections/:id/ballot/preview` | Officer | `GET /api/v1/staff/elections/:id/ballot/versions/:version_id` | Shared BallotView component in preview mode; persistent banner "Preview — no vote or participation is recorded"; submit disabled | n/a | Must not call any vote or participation endpoint | ISSUE-06 |

## ISSUE-07 Views

| View / Route | Roles | Data / API | Key Elements | States | Notes | Owning Issue |
|---|---|---|---|---|---|---|
| Ballot Secrecy Statement | `/privacy/ballot-secrecy` and inline modal | Any authenticated user | Static content from `docs/privacy-boundary.md` | Plain-language sections: What is protected, What is not guaranteed, Who can see participation, Who cannot see choices | n/a | Copy is reviewed by stakeholders; no words like "100% anonymous" or "untraceable" | ISSUE-07 |
| Pre-vote Notice | (component) | Eligible voter | None | Short summary + link to full statement; acknowledgement is optional unless SRS requires it | Dismissal remembered server-side or not at all (no choice data in browser storage) | Used by ISSUE-08 | ISSUE-07 |
| Participation/Turnout Panel | `/staff/elections/:id/turnout` | Officer, Auditor | `GET /api/v1/staff/elections/:id/turnout` | Counts only: eligible, voted, percent; optional per-faculty aggregates only if group size ≥ configured minimum | Counters, refresh timestamp (server time) | Small-group suppression message ("hidden to protect privacy") | Never a per-voter list of who voted unless SRS explicitly requires; never any choice data | ISSUE-07 |
| Voter Confirmation Wording | (component `VoteRecordedNotice`) | Voter | None | States that a vote was recorded and when (coarse time), not what it was | n/a | No "download/print/share my ballot" actions; no choice echo | ISSUE-07 |

## Pre-ISSUE-06 Views

| View / Route | Roles | Data / API | Key Elements | States | Notes | Owning Issue |
|---|---|---|---|---|---|---|
| Elections List | `/elections` | Voter | `GET /api/v1/protected/elections` | Cards/rows: title, phase badge, opens/closes in explicit timezone, server-time banner, action button | Empty ("no elections for you yet"); not-eligible shown with reason from server | Phase comes from the server response only | ISSUE-04 |
| Election Detail | `/elections/:id` | Voter | `GET /api/v1/protected/elections/:id` | Election info, eligibility status, action button | Not eligible; already voted; not yet open; closed | Server-enforced eligibility | ISSUE-04 |
| Ballot View | `/elections/:id/ballot` | Eligible voter | `GET /api/v1/protected/elections/:id/ballot` | BallotView in vote mode with selections | Not eligible; already voted; not yet open; closed | Selections only in memory (A3) | ISSUE-04 |
| Results | `/elections/:id/results` | Voter / public per policy | `GET /api/v1/protected/elections/:id/results` | Results display per visibility policy | Not yet available; no data if policy not met | Visibility timing enforced by server | ISSUE-04 |
| Staff Elections | `/staff/elections` | Staff | `GET /api/v1/staff/elections` | List of elections with staff actions | Empty state | Server role enforcement | ISSUE-04 |
| Staff Election Detail | `/staff/elections/:id` | Staff | `GET /api/v1/staff/elections/:id` | Election configuration, roster management, eligibility actions | Not authorized | RBAC enforced | ISSUE-04 |
