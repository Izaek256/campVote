# Authentication, Sessions & Account Lifecycle

**Status:** implemented and tested on `feat/issue-02-auth-session-hardening` (ISSUE-02), 2026-10-28.
**Scope:** password storage, registration, login, email verification, password reset, sessions, cookies, CSRF, lockout, deactivation, audit, first-administrator bootstrap.

This document records what the code actually does today and which test proves it. Voting routes do not exist yet, so "an unverified account cannot vote" is currently enforced at sign-in rather than at a ballot endpoint (see [Limitations](#limitations)).

## Criterion-by-criterion status

| ISSUE-02 criterion | Status | Where |
|---|---|---|
| Passwords stored only as strong salted hashes; never logged or returned | Implemented | `models.User.BeforeCreate` (bcrypt, default cost); `Password` has `json:"-"` |
| Unverified accounts cannot perform voter actions | Partial — enforced at login when `REQUIRE_EMAIL_VERIFICATION=true`; no voting route exists yet | `handlers/auth.go::loginRefusal` |
| Verification/reset tokens single-use, expiring, unguessable | Implemented | `services/email_verification.go`, `services/password_reset.go` (256-bit random, stored as SHA-256 hash, conditional UPDATE) |
| Reset/login responses do not reveal whether an account exists | Implemented | `handlers/auth.go::invalidCredentials`, `spendPasswordCheck`; `ForgotPassword` generic response |
| Rate-limit / lock out brute force, with tests | Implemented | Sentinel per-route limits; per-account lockout in `handlers/auth_lockout.go` |
| Sessions: idle + absolute expiry, rotate on login, revoke on logout / password change / deactivation | Implemented | `services/session.go` (7-day idle, 30-day absolute); `SetAuthCookies`; `RevokeAllUserSessions` |
| Cookies HttpOnly / Secure / SameSite; CSRF enforced for cookie-auth mutations | Implemented | `services/auth.go::SetAuthCookies`; `middleware/csrf.go::AutoCSRF` |
| Deactivated user's existing session rejected on next request | Implemented | `handlers/user.go::Update` revokes all sessions; `middleware/auth.go` also refuses inactive users |
| No default or seeded credentials; one-time first-admin bootstrap | Implemented (demo users remain development-only) | `database/bootstrap.go`, `cmd/bootstrap`; seeders gated on `APP_ENV=development` |
| Audit events for login success/failure, lockout, reset, deactivation | Implemented | `services/activity.go` + handlers (`auth.account_locked`, `auth.password_reset`, `user.deactivated`, …) |
| HTTP-level tests: register → verify → login → logout → reset, replay, expiry, lockout, cookies | Implemented | `handlers/auth_security_test.go`, `middleware/csrf_test.go` |

## Passwords

- Hashed with bcrypt at `bcrypt.DefaultCost` on create and on password change/reset. The plaintext is never persisted.
- `json:"-"` on `models.User.Password`, so no API response can carry the hash either.
- Policy (`internal/password`): minimum 8 characters, maximum 72 bytes, letters plus another class, rejects common passwords and personal fragments (name/email). Optional breach-list check via `PASSWORD_BREACH_CHECK`.
- The reset and change paths run the same checks as registration. A reset consumes the token before validating the new password, so a failed reset cannot leave a live token behind.

## Registration and email verification

- `POST /api/auth/verify-email` consumes a six-digit code in one transaction: marks the code used, then sets `email_verified_at` only if the address still matches the address the code was issued for.
- Codes: six digits from `crypto/rand`, stored as an HMAC-SHA256 hash keyed with the application `JWT_SECRET`, 10-minute TTL. The keyed hash matters because the six-digit space is small enough to brute-force from a plain hash. Issuing a new code marks every earlier unused code for that user as used.
- Replay, expiry, unknown and email-changed all return `400 INVALID_LINK` with one message.
- `REQUIRE_EMAIL_VERIFICATION` (default `false`) gates sign-in. It is opt-in so existing local accounts are not locked out; **production deployments should set it to `true`**. Social/SSO sign-ins are unaffected because the IdP already proved the address.
- `POST /api/auth/send-verification` is authenticated (prevents using it as a spam relay) and rate-limited in the router. In development the response also carries `verify_code`, guarded on `APP_ENV=development`.

## Login, rate limiting and lockout

- Unknown email, wrong password, no-password social account and a locked account all return `401 INVALID_CREDENTIALS` with the same body. A dummy bcrypt comparison equalises the response time for unknown addresses.
- Only wrong passwords on real accounts are counted (`failed_login_count`), incremented in SQL so concurrent attempts cannot overwrite each other.
- Threshold and window: `LOGIN_MAX_ATTEMPTS` (default 10) and `LOGIN_LOCKOUT_MINUTES` (default 15). `LOGIN_MAX_ATTEMPTS<=0` disables the per-account lock.
- Reaching the threshold sets `locked_until`, clears the failure counter, deletes pending TOTP tokens, writes `auth.account_locked` (`warn`), and emails the owner a reset link (the lock is keyed on an email anyone can guess, so the mailbox is the only non-leaking notice channel). The HTTP response never mentions the lock.
- A locked account is not password-checked, so guessing the right password cannot confirm it through the lock.
- A completed sign-in clears `failed_login_count` and `locked_until`. A successful password reset also clears the lock.
- Admins can unlock early through `PUT /api/users/:id/unlock` (audited as `user.unlock`).

## Password reset

- `POST /api/auth/forgot-password` always answers `200` with the same message, whether or not the address exists. Token creation, storage, audit and email delivery run off the request path so response time does not leak existence.
- Tokens: 256-bit random hex, stored as SHA-256 hash, 1-hour TTL, single-use. Issuing a new token burns earlier unused ones.
- Consumption is one conditional `UPDATE ... WHERE used_at IS NULL AND expires_at > now`, so two concurrent redemptions cannot both succeed.
- On success: password re-hashed, **all** sessions revoked (`RevokeAllUserSessions`), lockout cleared, `auth.password_reset` (`warn`) audited.
- A reset link is logged only outside production when no mailer is configured. In production a missing mailer is logged as a misconfiguration and the token is not printed.

## Sessions and cookies

- Every sign-in creates a `sessions` row: SHA-256 of the refresh token, user, IP, user agent, `last_seen_at`, `expires_at`. The raw refresh token is never stored.
- Lifetimes: 7-day idle timeout, 30-day absolute timeout (package vars in `services/session.go`).
- Rotation on every refresh. Presenting a previous (already rotated) refresh token is treated as theft: the session is revoked and the refresh gets `401 SESSION_REVOKED`.
- Logout revokes the session before clearing cookies. Browsers are matched by the `grit_refresh` cookie; native bearer clients by the `session_id` claim set by the auth middleware.
- Cookies set by `SetAuthCookies`:

  | Cookie | HttpOnly | Secure | SameSite | Path |
  |---|---|---|---|---|
  | `grit_access` | yes | on HTTPS / `X-Forwarded-Proto: https` | Lax | `/` |
  | `grit_refresh` | yes | same | Lax | `/api/v1/auth` |
  | `grit_signed_in` | yes (no secret) | same | Lax | `/` |

- `CSRF` / `AutoCSRF`: safe methods issue a readable `grit_csrf` cookie (SameSite=Lax, Secure behind TLS). Cookie-authenticated `POST/PUT/PATCH/DELETE` must echo it in `X-CSRF-Token`; bearer requests are exempt; login/register/refresh/forgot/reset/TOTP-verify and SAML ACS are exempt because they bootstrap the session.

## Account deactivation and deletion

- An administrator setting `active=false` (or soft-deleting a user) revokes all of that user's sessions and writes an audit row (`user.deactivated`, `user.deleted`; self-close writes `user.self_deleted`).
- The auth middleware refuses inactive users and every request with a revoked session. The next request after deactivation therefore fails even if it still carries an unexpired access token.

## First administrator (no default credentials)

There is **no seeded administrator** and no default password. Demo student accounts (`ivan.mugisha@…`, `doreen.kemigisa@…`) and demo elections are created only when `APP_ENV=development`.

Create the first administrator once, with environment variables that are removed afterwards:

```bash
cd apps/api
BOOTSTRAP_ADMIN_EMAIL=admin@must.ac.ug \
BOOTSTRAP_ADMIN_PASSWORD='choose-a-long-unique-password' \
BOOTSTRAP_ADMIN_FIRST_NAME=First \
BOOTSTRAP_ADMIN_LAST_NAME=Admin \
go run ./cmd/bootstrap
```

Rules enforced by `database.CreateFirstAdmin`:

- Email must parse and password must be at least 12 characters and pass the password policy.
- The account is created as `ADMIN`, active, with `email_verified_at` set (the operator controls the deployment; requiring mail first could lock them out of the only account that can configure mail).
- It refuses with `ErrAdministratorExists` if **any** administrator exists, including a soft-deleted one. There is no force flag.
- Migrations must have run first (`go run ./cmd/migrate`); the command checks for the users table.

## Audit events

| Action | Severity | Written by |
|---|---|---|
| `auth.register` | info | register |
| `auth.login` | info | login |
| `auth.login_failed` | warn | unknown email / wrong password |
| `auth.login_locked` | warn | attempt against a locked account |
| `auth.login_blocked` | warn | disabled or unverified account after a correct password |
| `auth.account_locked` | warn | lockout threshold reached |
| `auth.logout` | info | logout |
| `auth.password_reset_requested` | info | forgot-password |
| `auth.password_reset` | warn | successful reset |
| `user.deactivated` / `user.deleted` / `user.self_deleted` | warn | account lifecycle |
| `user.unlock` | warn | administrator unlock |

Audit rows never include a password, reset/verification token, or ballot content.

## Verification run

Executed on Windows, 2026-10-28, from `apps/api`:

```bash
go build ./...
go test ./internal/handlers ./internal/middleware ./internal/database ./internal/password -count=1
```

Result: build clean; all four packages `ok` (handlers ≈15 s, middleware ≈3.5 s, database ≈4 s, password ≈1.4 s).

Tests added or extended for this issue:

- `handlers/auth_security_test.go`
  - `TestRegister_StoresHashAndDoesNotLeakPassword`
  - `TestAuthCookies_AreHardened`
  - `TestLogin_GenericErrorsForUnknownEmailAndWrongPassword`
  - `TestLogin_LocksAfterRepeatedFailuresAndRecoversAfterWindow` (time-travel past the window, no sleeping)
  - `TestLogin_RequiresVerifiedEmailAndVerifyLinkIsSingleUse`
  - `TestVerifyEmail_ExpiredTokenIsRefused`
  - `TestPasswordReset_RevokesSessionsAndCannotBeReplayed`
  - `TestLogout_RevokesCookieSession`
  - `TestLogout_RevokesBearerSession`
  - `TestDeactivation_RevokesSessionsAndBlocksNextRequest`
- `middleware/csrf_test.go`: cookie mutation needs a matching header, bearer bypass, login bootstrap exemption, readable `grit_csrf`, Secure behind `X-Forwarded-Proto: https`.
- `database/elections_seeder_test.go`: demo seeding skips outside `APP_ENV=development` and seeds no administrator.
- `database/bootstrap_test.go`: verified administrator, hash-only storage, one-time refusal, weak credential rejection, soft-deleted admin still counts.

## Limitations

- `go test ./internal/services` as a whole is blocked in this environment by pre-existing `CGO_ENABLED=0` SQLite-driver failures in unrelated passkey/recovery/saga tests. The auth/session service logic is exercised through the handler tests above; targeted service tests pass.
- Email verification is opt-in. Until `REQUIRE_EMAIL_VERIFICATION=true` is set in production, an unverified account can still sign in (but still cannot vote once voting routes enforce the same gate).
- Email domain (`@std.must.ac.ug` / `@must.ac.ug`) is **not** proof of enrollment, faculty, or eligibility. Domain policy and roster checks remain separate work.
- Rate limiting at the IP/route layer comes from Sentinel and is not exhaustively load-tested here; the per-account lockout is the tested brute-force defence.
- Demo users share a known password (`admin123`) and exist only under `APP_ENV=development`. Never run the seeders with production data or credentials.
- This is an academic prototype, not a certified election system.
