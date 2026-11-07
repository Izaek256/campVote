# Google sign-in: One Tap prompt suppressed by third-party-cookie restrictions

**Branch:** `feature/google-signin-authentication`
**Reported:** maintainer — clicking "Continue with Google" produced
*"Google sign-in could not open. Allow third-party cookies for this site, then try
again."*, and the console showed the Google Identity Services warning
*"GSI_LOGGER: Your client application uses one of the Google One Tap prompt UI
status methods that may stop functioning when FedCM becomes mandatory."*
**Status:** Fixed; the button now runs Google's Sign in with Google button flow.
Regression tests added.

---

## Symptom

1. The sign-in button did nothing and showed the third-party-cookie message.
2. The GSI_LOGGER FedCM deprecation warning was logged on the sign-in pages.

## Reproduction

1. Open `/login` (or `/register`) in Chrome with third-party cookies restricted
   (the Chrome default direction since the Privacy Sandbox rollout) or after the
   user has opted out of One Tap.
2. Click "Continue with Google".
3. The alert about third-party cookies appears; no Google prompt opens.

## Root cause

The button depended on **Google One Tap** (`google.accounts.id.prompt()`) to
obtain a credential, and translated a suppressed prompt into an instruction to
enable third-party cookies.

`apps/web/components/google-signin.tsx`:

- `initialize()` was called without opting into FedCM.
- `startGoogleSignIn()` called `google.accounts.id.prompt()` as the only way to
  get a credential from a click.
- `oneTapMessage()` read `isNotDisplayed()`, `getNotDisplayedReason()` and
  `getSkippedReason()`, and its `default` branch returned the cookie sentence.

One Tap is an automatic sign-in prompt, not a click-driven sign-in flow. Its
display depends on third-party cookies and is suppressed when they are blocked
or when the user has dismissed/opted out of One Tap. The suppression reason did
not match any of the three reasons the switch handled, so `oneTapMessage()` fell
through to the default branch — CampVote's own "allow third-party cookies"
sentence. That text is application copy, not a Google message.

The GSI_LOGGER warning is the same defect reported by Google: the display-moment
methods (`isDisplayMoment`, `isDisplayed`, `isNotDisplayed`,
`getNotDisplayedReason`) and `getSkippedReason` are deprecated under FedCM and
the `prompt` callback no longer returns display-moment notifications.
References: [Migrate to FedCM](https://developers.google.com/identity/gsi/web/guides/fedcm-migration),
[Sign in with Google JS reference](https://developers.google.com/identity/gsi/web/reference/js-reference).

No backend, session-cookie, CORS or domain-enforcement change was needed: the Go
side verifies the ID token (`idtoken.Validate`: signature, issuer, audience,
expiry), checks `email_verified`, and enforces `@std.must.ac.ug` /
`@must.ac.ug` with an exact final-domain match
(`apps/api/internal/config/domains.go`).

## Fix

Switch the button to Google's own **Sign in with Google button** flow:

1. `apps/web/components/google-signin.tsx` — `initialize()` now sets
   `use_fedcm_for_button: true`, and the component renders Google's button with
   `google.accounts.id.renderButton()` instead of calling `prompt()`. The
   deprecated One Tap status methods are removed entirely, along with the cookie
   message. The credential `callback` and the `onCredential → /auth/google`
   handoff are unchanged.
2. Google's button cannot be restyled, so its theme follows the page
   (`data-theme` / `prefers-color-scheme`); a `MutationObserver` re-renders it on
   a theme switch so light/dark both look correct.
3. `apps/web/__tests__/google-signin.test.tsx` — updated the mock to
   `renderButton`, and added two regression tests: Google's button is rendered
   (CampVote renders no button of its own), and the deprecated `prompt` entry
   point is never called.

## Verification

| Check | Command (in `apps/web`) | Result |
|---|---|---|
| Type check | `pnpm type-check` | pass |
| Unit tests | `pnpm test` | pass (35 tests; 9 for Google sign-in) |
| Production build | `pnpm build` | pass |

The regression tests fail against the previous `prompt`-based component and pass
after the fix.

**Not verified:** a live Google popup and a real Google-issued token. That needs
a configured client ID and a browser session against Google, which this
environment does not have.

## Deployment

No API, database or configuration change is required — the fix is web-only.

1. Deploy the updated web app. `NEXT_PUBLIC_GOOGLE_CLIENT_ID` is inlined at
   build time, so a **rebuild** (not just a restart) is required.
2. Confirm the deployed origin is listed under the OAuth client's Authorized
   JavaScript origins (see `docs/implementation/GOOGLE_SIGNIN.md`).
3. Hard-refresh `/login` and confirm Google's own button appears and no longer
   shows the "allow third-party cookies" text.
4. No environment variables change; no production data is touched.

## Remaining limitations

- The page now shows Google's official button rather than a bespoke CampVote
  button, because Google's brand rules require its own button for the button
  flow.
- If the deployment serves the web app on a site different from the API **and**
  does not proxy `/api` (so `NEXT_PUBLIC_API_URL` points straight at the API
  origin), the `SameSite=Lax` session cookies are dropped cross-site and the
  user is signed in and immediately 401s. The Vercel build proxies `/api` to the
  API, which keeps the cookies first-party — see `docs/operations/vercel-proxy.md`.
- No GIS nonce (unchanged from the original implementation).
