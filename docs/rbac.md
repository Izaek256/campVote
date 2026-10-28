# Role-Based Access Control for Campus Elections

**Status:** implemented and tested on `feat/issue-03-server-side-rbac` (ISSUE-03), 2026-10-28.
**Scope:** SRS §2.1 roles, the route policy table, per-election scoping, separation of duties, results visibility, certification and denial audit.
**Out of scope:** election-specific eligibility (ISSUE-04), voting endpoints (ISSUE-06), and any admin/voter UI.

This document records what the code does today. The authoritative role names are in [SRS §2.1](../SRS.md); the executable copy is `apps/api/internal/models/user.go` and `models/role.go`.

## Design in one paragraph

Two independent layers decide access. **Route policy** (`internal/policy/routes.go`) answers "may this role call this endpoint at all" and is a complete, explicit table of method + full path; a route with no row is refused and the inventory test fails. **Object policy** (`internal/policy/election.go`) answers "may this person touch this election" and runs in the handler, because the URL alone cannot say which election a user is assigned to. The second layer turns a cross-election request into `404`, not `403`, so a guessed id does not confirm that another election exists.

## SRS roles

| SRS role | Constant | Authority |
|---|---|---|
| System Administrator | `ADMIN` | Global. `*` grant; every route and every election. |
| Chief Election Administrator | `CHIEF_ELECTION_ADMIN` | Global election administration, including certification. |
| Election Official | `ELECTION_OFFICIAL` | Per-election. Works only on assigned elections. |
| Student Voter | `STUDENT_VOTER` | Self-service only; no privileged grant. |
| Staff Voter | `STAFF_VOTER` | Self-service only; no privileged grant. |
| Candidate | `CANDIDATE` | Per-election. Reads published results; never voter data or choices. |
| Election Observer-Auditor | `ELECTION_AUDITOR` | Per-election. Reads authorised evidence and audit records; never ballot choices. |

The three Grit scaffold roles (`ADMIN`, `EDITOR`, `USER`) remain because existing data and routes use them. `EDITOR` and `USER` receive no campus-election authority beyond what their legacy grants already allow.

### Global grants (seeded)

| Role | Grants |
|---|---|
| `ADMIN` | `*` |
| `EDITOR` | `uploads.create`, `uploads.view`, `uploads.delete`, `users.view` |
| `USER` | none |
| `CHIEF_ELECTION_ADMIN` | `elections.*`, `candidates.*`, `rosters.*`, `ballots.*`, `results.*`, `audit.view` |
| `ELECTION_OFFICIAL` | `elections.view`, `elections.edit`, `candidates.*`, `rosters.view`, `ballots.view`, `results.view`, `results.view_live` |
| `STUDENT_VOTER` | none |
| `STAFF_VOTER` | none |
| `CANDIDATE` | `elections.view`, `candidates.view`, `results.view` |
| `ELECTION_AUDITOR` | `elections.view`, `candidates.view`, `results.view`, `audit.view` |

`SeedRoles` is idempotent and never overwrites an existing row, so an operator's edits survive a restart.

### Per-election assignments

`models.ElectionRole` stores `(election_id, user_id, role)` with a unique index. Only `CHIEF_ELECTION_ADMIN`, `ELECTION_OFFICIAL`, `CANDIDATE` and `ELECTION_AUDITOR` may be assigned per election. For officials, candidates and auditors the assignment **narrows** authority to that election; it does not widen their global role. A `CHIEF_ELECTION_ADMIN` assignment acts as a scoped chief for that election.

An actor is treated as global when they are `ADMIN`, are `CHIEF_ELECTION_ADMIN`, or hold a global grant for `elections.view` through a non-scoped role. Holding `ELECTION_OFFICIAL` / `CANDIDATE` / `ELECTION_AUDITOR` never makes an actor global, even if the matching global role row also carries `elections.view`.

## Role × operation matrix

`E` = allowed for the assigned election. `G` = allowed for every election. `–` = refused.

| Operation | ADMIN | CHIEF | OFFICIAL (E) | AUDITOR (E) | CANDIDATE (E) | VOTER |
|---|---|---|---|---|---|---|
| View election metadata | G | G | E | E | E | own elections only* |
| Edit election configuration | G | G | E | – | – | – |
| Manage candidates | G | G | E | view | view | – |
| View eligibility roster | G | G | E | – | – | – |
| Edit/import eligibility roster | G | G | – | – | – | – |
| View ballot configuration | G | G | E | – | – | – |
| Edit ballot configuration | G | G | – | – | – | – |
| List/assign/remove election roles | G | G | – | – | – | – |
| Read provisional (live) results | G | G | E | – | – | – |
| Read published results | G | G | E | E | E | – |
| Certify results | G | G | second official only | – | – | – |
| Read election audit records | G | G | – | E | – | – |

\* Election-list and metadata endpoints are not implemented yet; when they are, they must add a route policy and scope the rows to the caller. The inventory test fails otherwise.

`ELECTION_OFFICIAL` is deliberately **not** allowed to edit rosters, ballots or role assignments. Those are chief-administrator operations in this matrix.

## Route policy enforcement

- `policy.Declared()` is mounted globally and refuses any matched route without a table row (`403`). It runs before authentication, so a route mounted in the wrong group is still denied.
- `policy.Guard(db)` is mounted after auth on the protected, staff and admin groups and enforces the declared requirement (`401` when no session, `403` when the grant/role is missing).
- Library mounts (`/docs`, `/pulse`, `/sentinel`, `/studio`, `/files/`) are excluded by prefix; they carry their own credentials.
- The table currently covers **181 routes**, 30 of them public. `TestEveryRouteHasAPolicy` walks `engine.Routes()` and fails when a registered route has no policy, which is the guard against default-allow drift.

Requirement kinds in the table:

| Kind | Meaning |
|---|---|
| `Public` | No session: sign-in, registration, password reset, public pages, webhook signatures, WebSocket/SSE handshakes. |
| `Authenticated` | Any live session; the handler still scopes rows. |
| `Permission` | One catalog key, e.g. `elections.roles`. |
| `Role` | One legacy role, used for `ADMIN`-only routes. |
| `RoleOrPermission` | `ADMIN` or a grant; mirrors the scaffold's existing `RequireRole(ADMIN, "perm:…")`. |

Inline `middleware.RequireRole` calls are left in place as defence-in-depth. The route table is the central declaration and must not be weaker than the inline guard.

## Object-level checks (IDOR)

Every new election endpoint resolves the election through `policy.AuthorityFor` before reading or writing:

1. Load the election by id. Missing election → `ErrNotVisible` → `404`.
2. Load the actor's per-election assignments, merge their role grants.
3. If the actor is not global and has no assignment → `ErrNotVisible` → `404`.
4. Check the object permission; failure → `403` and an `authz.denied` audit row.

Consequences:

| Situation | Response |
|---|---|
| No session / invalid session | `401` |
| Authenticated, route grant missing (e.g. candidate calling role management) | `403` |
| Authenticated, in-scope, object permission missing | `403` |
| Nonexistent election id | `404` |
| Existing election the caller has no relationship with | `404` (same body as above) |
| Role assignment does not exist on remove | `404` |

## Results visibility

`ResultPolicy` is read from the election, then combined with the actor's grant. A tally is **provisional** while the election status is `OPEN`; published otherwise.

| Policy | Status | Requirement |
|---|---|---|
| `LIVE` | `OPEN` | `results.view_live` |
| `LIVE` | closed/review/finalized | `results.view` |
| `AFTER_CLOSE` | `OPEN`, `DRAFT`, `SCHEDULED`, `REGISTRATION`, `PENDING_APPROVAL` | refused: "Results are published after voting closes" |
| `AFTER_CLOSE` | `CLOSED`, `REVIEW`, `FINALIZED`, `ARCHIVED` | `results.view` |
| `CERTIFIED_ONLY` | before `FINALIZED` | refused: "published only after certification" |
| `CERTIFIED_ONLY` | `FINALIZED`, `ARCHIVED` | `results.view` |
| unrecognised policy | any | refused (fail closed) |

The response labels `FINALIZED`/`ARCHIVED` as `CERTIFIED` and everything else as `PROVISIONAL`; it never uses "winner" (FR-DASH-06, FR-DASH-09). Tallies come from `ElectionTally`, which has no voter identity columns, so a results response cannot join to a voter.

Candidates and auditors hold `results.view`, so they can read a published tally. Only officials, chiefs and `ADMIN` hold `results.view_live`.

## Separation of duties

- A candidate cannot be assigned `ELECTION_OFFICIAL`, `CHIEF_ELECTION_ADMIN` or `ELECTION_AUDITOR` in the same election; assigning any of those to an existing candidate is refused the same way.
- An official or auditor cannot be made a candidate in an election they administer or observe.
- Certification requires **at least two distinct users** assigned as official or chief on that election, and the certifying actor must be one of them unless they are global.
- The election creator cannot certify its own results.
- Certification is only possible from `CLOSED` or `REVIEW`, writes `FINALIZED` and an `ElectionAuditLog` row (`results_certified`) in one transaction.

## Live role changes

Roles are not baked into the session token. `AuthorityFor` reads `election_roles` on every request, and the auth middleware resolves global grants per request (with the existing cache, invalidated on assignment changes). A user who signs in before being assigned sees the new authority on their next request; removing the role takes effect on the next request with the same token. `TestRoleChangeAppliesToExistingSessionAndCrossElection` proves both directions.

## Audit events

| Action | Where | Contents |
|---|---|---|
| `authz.denied` | `user_activity` (`warn`) | actor, method, route, reason, resource id |
| `election.role_assigned` | `user_activity` (`info`) | actor, target email, role, election id |
| `election.role_removed` | `user_activity` (`info`) | actor, role, election id |
| `results_certified` | `election_audit_logs` | actor, election id, IP, timestamp |

Denial and role events never include request bodies, ballot choices, voter participation or ballot ids. The results and role endpoints return aggregate/assignment data only.

## Tests and verification

Executed on Windows, 2026-10-28, from `apps/api`:

```bash
go build ./...
go test ./internal/policy ./internal/routes ./internal/handlers ./internal/authz ./internal/models -count=1
```

All five packages pass. `go test ./...` still fails only in `internal/saga` and `internal/services` with the pre-existing `CGO_ENABLED=0` SQLite-driver stub; no failure there is related to this issue.

Relevant tests:

- `routes/policy_inventory_test.go::TestEveryRouteHasAPolicy` — every registered route has a policy.
- `policy/election_test.go`:
  - `TestAuthorityScopesOfficialToAssignedElection`
  - `TestCandidateCannotAdministerOwnElection`
  - `TestRoleAssignmentConflict`
  - `TestResultsVisibilityFollowsElectionPolicy`
  - `TestRoleChangeAppliesToNextRequest`
  - `TestCertifyRequiresTwoOfficialsAndADifferentActor`
- `handlers/election_rbac_test.go`:
  - `TestRoutePolicyMatrixForElectionEndpoints` — 401/403/404 matrix, voter token refused on an admin route, candidate refused provisional results, candidate allowed published results, cross-election 404, denial audited.
  - `TestRoleChangeAppliesToExistingSessionAndCrossElection` — live grant/revoke with an existing token, cross-election isolation, candidate/official separation of duties.
- `authz/grants_test.go::TestSeedRoles` — seeds nine roles and does not clobber operator edits.

## Limitations and follow-ups

- Eligibility (who may vote in which election) is ISSUE-04. This issue authorises operations; it does not decide voter eligibility.
- Only five election endpoints exist so far (roles list/assign/remove, results, certify). Remaining election, candidate, roster and ballot endpoints must be added with a policy row; the inventory test enforces that.
- Roster and ballot edit permissions exist in the catalog but no routes use them yet; the officer role is intentionally read-only for both.
- A per-election `CHIEF_ELECTION_ADMIN` assignment behaves as a scoped chief. If the university wants chief authority to be global-only, remove it from `policy.ValidElectionRole` and from the assignment UI.
- Custom global roles that hold `elections.view` are treated as global for election metadata. This is intentional for operator-defined roles but documented so it is not mistaken for scoping.
- No UI consumes these endpoints yet; server-side enforcement is what this issue delivers.
- This is an academic prototype, not a certified election system.
