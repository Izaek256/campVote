# Google Sign-In (Google Identity Services)

**Branch:** `feature/google-signin-authentication`
**Status:** implemented; backend tests written, frontend tests passing; no live Google authentication has been run (see [Verification](#verification)).
**Scope:** student identity verification with a Google ID token, replacing the email-OTP registration form as the primary student sign-in.

## Why

Email OTP proves control of a mailbox, not that the mailbox belongs to the person
who typed it, and it depends on working mail delivery. Google Identity Services
(GIS) returns a signed ID token that the backend verifies with Google's own
library. The verified claims are a stronger identity signal and remove the mail
step from student registration.

This is still **not** proof of university enrolment. The student email domain is
a policy gate; eligibility for a particular election remains the roster's
decision (see `docs/eligibility.md`).

## What the browser does

`apps/web/components/google-signin.tsx`:

1. Loads `https://accounts.google.com/gsi/client` once per page, after hydration.
2. Calls `google.accounts.id.initialize` with `NEXT_PUBLIC_GOOGLE_CLIENT_ID`,
   `ux_mode: "popup"`, `auto_select: false`.
3. Renders a **first-party** `<button>` styled with the CampVote UI kit
   (`.btn block lg`) and the official four-colour Google "G" mark. Clicking it
   calls `google.accounts.id.prompt()`, which shows Google's One Tap / popup.
4. A `MomentNotification` listener turns a suppressed prompt into an actionable
   alert (`unregistered_origin`, `invalid_client`, etc.) instead of failing
   silently.
5. On success, passes **only the raw `credential`** to the caller. The component
   never decodes it, never stores it, and never treats its claims as trusted.
6. The caller (`AuthProvider.signInWithGoogle`) sends `{ id_token }` to
   `POST /api/v1/auth/google`.

If `NEXT_PUBLIC_GOOGLE_CLIENT_ID` is empty the component renders a visible
"Google sign-in is unavailable" banner and never loads the script.

Both `/login` and `/register` are **Google-only**: they show brand copy and the
single Google button, with no email/password form and no OTP form. Staff and
administrator accounts are created with `go run ./cmd/bootstrap` (or by an
administrator) and then sign in with the same button — see the backend rules
below.

## What the backend does

`apps/api/internal/services/google.go`:

- `IDTokenVerifier` wraps `google.golang.org/api/idtoken`. `idtoken.Validate`
  checks the signature against Google's published keys, the issuer, the
  audience (`GOOGLE_OAUTH_CLIENT_ID`) and the expiry **before** any claim is
  returned.
- The verifier maps `sub`, `email`, `email_verified`, `hd`, `given_name`,
  `family_name`, `name`, `picture`. It refuses a token with no subject.
- `ErrGoogleNotConfigured` is returned when no client ID is configured. There is
  no fallback to another sign-in method.

`apps/api/internal/handlers/auth_google.go` (`POST /api/v1/auth/google`):

| Step | Check | Failure |
|---|---|---|
| 1 | `GOOGLE_OAUTH_CLIENT_ID` and verifier present | `503 GOOGLE_NOT_CONFIGURED` |
| 2 | Body is `{ "id_token": "…" }` | `400 VALIDATION_ERROR` |
| 3 | Google signature / issuer / audience / expiry | `401 INVALID_CREDENTIALS` (detail logged, token never logged) |
| 4 | `email_verified` | `401 INVALID_CREDENTIALS` |
| 5 | `email` present | `401 INVALID_CREDENTIALS` |
| 6 | Email is on `STUDENT_EMAIL_DOMAIN` (self-register) or `ADMIN_EMAIL_DOMAIN` (link-only) | `403 FORBIDDEN` |
| 7 | If `GOOGLE_OAUTH_HOSTED_DOMAIN` is set, `hd` matches | `403 FORBIDDEN` |
| 8 | Account lookup / creation / linking | `401`, `403`, `409` or `500` |
| 9 | Issue the same session, cookies, CSRF and audit events as password login | `500` |

Account resolution (`resolveGoogleUser`):

1. Match the immutable `google_id` (subject) first. A disabled account is
   refused.
2. Otherwise match `LOWER(email)`. A disabled account is refused. An account
   already linked to a *different* subject is a conflict (`409`) — never an
   automatic merge.
3. Otherwise **create** a passwordless `STUDENT_VOTER` account (student domain
   only), active, provider `google`, email verified now. No admin role is ever
   granted.
4. A unique-index race on create re-resolves once without creating.

**Student vs staff.** A verified address on the student domain may self-register
(step 3). A verified address on the staff/admin domain can only **link into an
account that already exists**; if none exists the endpoint returns
`403 FORBIDDEN` ("No CampVote account exists for that address…"). Because a
Google token therefore can never mint an account on the staff domain, it can
never grant the `ADMIN` role either. Staff and administrator accounts are created
by an administrator (`go run ./cmd/bootstrap` creates the first one).

Linking an existing account (`linkGoogleIdentity`) never changes its role, name,
student number or password. If the account was **unverified**, the old password
is cleared and its sessions are revoked before the link is written, because
somebody may have registered that address first. This mirrors the redirect-OAuth
policy.

New accounts are written through `services.CreateUser`, so the existing email
uniqueness check and model hooks still apply.

## Configuration

| Variable | Where | Required | Meaning |
|---|---|---|---|
| `GOOGLE_OAUTH_CLIENT_ID` | API | yes, for Google sign-in | OAuth 2.0 Web Application client ID. Empty disables the endpoint with `503`. |
| `GOOGLE_OAUTH_HOSTED_DOMAIN` | API | optional | Workspace `hd` value every token must carry, e.g. `must.ac.ug`. Empty means the student-domain check is the only gate. |
| `NEXT_PUBLIC_GOOGLE_CLIENT_ID` | Web | yes, for Google sign-in | The same client ID. Inlined at build time; changing it needs a rebuild. |

`GOOGLE_OAUTH_CLIENT_ID` is deliberately separate from `GOOGLE_CLIENT_ID`, which
belongs to the older redirect-based Goth provider. They may name the same client,
but only `GOOGLE_OAUTH_CLIENT_ID` is required for student sign-in. Never put the
client **secret** in the web app or in the browser.

## Google Cloud Console setup

1. Create (or reuse) a project in <https://console.cloud.google.com/>.
2. **APIs & Services → OAuth consent screen**: configure it, add the student
   domain, and publish when ready. External + testing users must be added while
   the app is unverified.
3. **APIs & Services → Credentials → Create credentials → OAuth client ID →
   Web application**.
4. **Authorized JavaScript origins** (not redirect URIs — GIS popup mode needs
   origins):
   - `http://localhost:3000`
   - `https://camp-vote-jade.vercel.app`
   - any other origin the web app is actually served from.
5. Copy the client ID into `GOOGLE_OAUTH_CLIENT_ID` (API) and
   `NEXT_PUBLIC_GOOGLE_CLIENT_ID` (web), then rebuild/restart both.
6. Optionally set `GOOGLE_OAUTH_HOSTED_DOMAIN` to the Workspace primary domain.

## Deployment

- Set both variables in the deployment environment; do not commit them.
- Rebuild the web app after changing `NEXT_PUBLIC_GOOGLE_CLIENT_ID` — Next
  inlines it.
- Restart the API after changing either API variable.
- Confirm the API log no longer prints the `GOOGLE_OAUTH_CLIENT_ID is not set`
  warning.

## Rollback

1. Unset `GOOGLE_OAUTH_CLIENT_ID` and restart the API. `/auth/google` then
   answers `503 GOOGLE_NOT_CONFIGURED`; the old OTP endpoints remain mounted.
2. Unset `NEXT_PUBLIC_GOOGLE_CLIENT_ID` and rebuild the web app: `/login` and
   `/register` then show the "Google sign-in is unavailable" banner. There is no
   password or OTP form to fall back to on those pages, so restoring sign-in
   means restoring a client ID (or reintroducing a form in code).
3. The migration only adds a partial unique index and clears duplicate
   `google_id` values on all but the oldest row. Accounts are not deleted. If the
   index must be removed:
   `DROP INDEX IF EXISTS idx_users_google_id;`
   Cleared `google_id` values relink on the owner's next Google sign-in; there is
   no automatic backfill.

## Troubleshooting

| Symptom | Likely cause |
|---|---|
| Banner "Google sign-in is unavailable" | `NEXT_PUBLIC_GOOGLE_CLIENT_ID` empty or the web app was not rebuilt. |
| Clicking the button shows "this site is not registered with the Google client" | The page origin is missing from Authorized JavaScript origins (`unregistered_origin`). |
| Clicking the button shows "the Google client ID is not accepted" | API and web are using different client IDs, or the client is not a Web client (`invalid_client`). |
| Popup: `origin_mismatch` | The page origin is missing from Authorized JavaScript origins. |
| API `503 GOOGLE_NOT_CONFIGURED` | `GOOGLE_OAUTH_CLIENT_ID` empty or API not restarted. |
| API `401` with `audience` in the log | API and web are using different client IDs. |
| API `403` "not part of the university's Google Workspace" | `GOOGLE_OAUTH_HOSTED_DOMAIN` is set and the token's `hd` differs (personal Gmail, or a secondary domain). |
| API `409` | The email and subject belong to different CampVote accounts; resolve manually. |
| Script blocked | A content-security policy or network blocker is stopping `accounts.google.com`. |

## Verification

Commands run on this branch (Windows, Go 1.26, Node 20 / pnpm 10, PostgreSQL not
required for the targeted suites):

| Check | Command (in `apps/api` unless noted) | Result |
|---|---|---|
| Build | `go build ./...` | pass |
| Vet | `go vet ./...` | pass |
| Google handler | `go test ./internal/handlers/... -run TestGoogleSignIn` | pass (8 tests) |
| Google verifier | `go test ./internal/services/... -run 'Google|IDTokenVerifier'` | pass (4 tests) |
| Migration | `go test ./internal/migrations/...` | pass |
| Touched packages | `go test ./internal/config/... ./internal/handlers/... ./internal/migrations/... ./internal/models/... ./internal/policy/... ./internal/routes/...` | pass |
| Web type check | `pnpm type-check` (in `apps/web`) | pass |
| Web tests | `pnpm test` (in `apps/web`) | pass (35 tests; 9 for Google sign-in) |
| Web build | `pnpm build` (in `apps/web`) | pass |

Environment limits on this machine, recorded rather than worked around:

- `go test ./...` cannot pass locally end to end: the `mattn/go-sqlite3` suites
  (`internal/saga`, and the passkey/recovery/saga tests in `internal/services`)
  need cgo and there is no C compiler here. Every such failure is the
  `CGO_ENABLED=0` stub message, and the Google tests in that package use the
  pure-Go `glebarez/sqlite` driver, so they run. CI runs the full suite with
  `-race` and a C compiler.
- `golangci-lint` is not installed locally; CI runs `golangci-lint run`.
- `pnpm lint` and `pnpm format:check` fail on `origin/main` independently of
  this branch: `apps/web/biome.json` and the root `biome.jsonc` are both root
  configurations, so Biome reports "Found a nested root configuration" from the
  repository root and, run from `apps/web`, applies default (tab) formatting to
  a space-indented codebase. Left for the maintainer; not caused by this change.

Coverage:

- Backend: handler tests inject a fake `GoogleVerifier` and cover configured /
  not configured, missing / malformed token, verification failure, unverified
  email, missing email, wrong student domain, hosted-domain mismatch, new
  account creation, existing verified link, existing unverified link (password
  cleared, sessions revoked), duplicate subject/email race, disabled account,
  staff address that links into an existing administrator without changing role
  or password, staff address with no account (rejected, nothing created), session
  creation and audit events. Service tests cover claim mapping and
  configuration errors. Migration tests cover deduplication and the partial
  unique index on SQLite.
- Frontend: Vitest + Testing Library cover rendering our own button, missing
  configuration, clicking to open One Tap, an actionable message when One Tap is
  suppressed, success, API rejection, a prompt response without a credential,
  network failure, duplicate submission and the loading state.

**Not verified:** a real Google popup and a real Google-issued token. That needs
a configured client ID and a browser session against Google, which this
environment does not have.

## Known limitations and assumptions

- **No GIS nonce.** The flow relies on the popup being bound to the page's
  origin and audience, plus the existing CSRF and rate-limit protections. A
  nonce would harden against token replay from a compromised page; it is not
  implemented. Documented as a deliberate scope decision, not an oversight.
- `hd` is optional. Without it, any Google account whose address is on the
  student domain is accepted. Google-hosted student addresses on the configured
  domain are the assumed deployment.
- Google proves control of a Google account, not enrolment, faculty or
  eligibility. Roster checks remain separate.
- Staff/admin accounts are never created by this endpoint; a staff address can
  only link into an account an administrator already created, so the endpoint
  cannot escalate anyone to `ADMIN`.
- The sign-in button is first-party and opens Google's One Tap / popup, so the
  served origin must be listed under Authorized JavaScript origins. If One Tap is
  suppressed (for example the origin is not registered, or the browser blocks
  third-party storage), the page shows the reason instead of a silently dead
  button.
- The OTP endpoints remain mounted for backward compatibility, but `/login` and
  `/register` are now Google-only and do not use them.
- No live end-to-end test against Google. No production environment variables
  were changed.
