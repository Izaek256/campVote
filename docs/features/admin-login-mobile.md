# Administrator Sign-In and Mobile Layout

**Issue:** [#30](https://github.com/Izaek256/campVote/issues/30)  
**Branch:** `feature/30-admin-login-mobile`  
**Status:** implemented and verified; native mobile keyboard behavior remains unverified.

## Sign-In Paths

- Students use Google from `/login` and `/register`.
- Administrators use email and password at `/admin/login`. The entry link appears
  in the student login footer.
- Password sign-in is accepted only when the account has the `ADMIN` role and
  its exact email domain matches `ADMIN_EMAIL_DOMAIN` (default `must.ac.ug`).
  A client-side domain check helps prevent mistakes but is not the security
  boundary.
- Google sign-in is restricted to the configured student email domain and
  `STUDENT_VOTER` accounts. Staff and administrator accounts cannot use it.
- Administrator two-factor authentication supports authenticator, email and
  backup codes through the existing API challenge flow.
- Authenticated requests load the current active user from the database. Staff
  routes use the server-side route policy, role and permission guards; hiding a
  route in the web app does not grant or deny API access.

Create the first administrator with the one-time `go run ./cmd/bootstrap`
command documented in the root README. Use that account's email and password at
`/admin/login`. Later administrator accounts are invited by an existing
administrator and set their password through the emailed reset link.

## Mobile Layout

The shared auth shell keeps its content column centered and capped at 420px.
At widths up to 640px it uses safe-area-aware gutters, a single-column card,
readable input text and touch targets of at least 44px. The Google Identity
Services button measures its available container width and remains capped at
384px.

Elections and staff data pages use the shared responsive app shell. Tables stay
inside `.tbl-wrap` horizontal scrollers rather than widening the viewport.

## Acceptance Checks

- [x] Admin password sign-in rejects student domains, wrong domains and
  non-admin database roles without issuing a session.
- [x] Students continue to sign in with Google; administrators are refused by
  the Google endpoint.
- [x] Admin TOTP and backup-code challenges complete before staff navigation.
- [x] `/login`, `/register`, `/admin/login`, `/elections`, and the staff
  dashboard table are checked at 360px and 390px widths with no page-level
  horizontal overflow. A 360px by 340px short viewport keeps the focused
  password field visible after scrolling.
- [x] Relevant API and frontend tests, type checks, lint and production build
  pass.

## Verification

Verified 2026-11-08 on `feature/30-admin-login-mobile` using the local Next.js
app and Playwright Chromium 156 mobile emulation (`isMobile` and touch enabled).
API responses were stubbed with synthetic student/admin sessions and a
synthetic election so the authenticated election and staff table layouts could
be rendered without using real accounts or data.

- At 360px, the auth card measured 312px wide and was centered with 24px side
  gutters; at 390px it measured 342px with the same gutters. `document` width
  matched the viewport at both sizes on every checked route.
- Auth actions and navigation links measured at least 44px high. The Google
  sign-in container has a 48px minimum height.
- The staff table remained inside `.tbl-wrap` with `overflow-x: auto`; its
  content was wider than its 316px (360px viewport) and 346px (390px viewport)
  container without widening the document.
- At 360px by 340px, focusing the administrator password field scrolled it
  into view; it remained within the viewport and the document had no horizontal
  overflow. Headless Chromium does not open a native mobile keyboard, so that
  OS-level behavior was not verified.

Checks run:

- `go test ./internal/handlers -count=1` — passed.
- `pnpm exec vitest run --maxWorkers=1` from `apps/web` — 6 files, 40 tests
  passed.
- `pnpm --dir apps/web type-check` — passed.
- `pnpm --dir apps/web build` — passed; `/admin/login` was included in the
  production routes.
- `pnpm exec biome lint ...` from `apps/web` on the changed TypeScript files —
  passed.

The initial parallel Vitest invocation hit worker-start timeouts; the
single-worker rerun passed. The repo-wide lint command was not treated as
passing because it includes generated `.next` output; changed TypeScript files
were linted directly.
