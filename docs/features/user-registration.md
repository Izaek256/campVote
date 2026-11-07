# User Registration Flow

**Issue:** #20 (Epic: Complete SRS Implementation — Priority 1.1)  
**Branch:** `feature/20-registration-and-user-management`  
**SRS References:** FR-AUTH-01, FR-AUTH-07  
**Status:** Implemented

---

## Summary

Students self-register with their campus email address; staff and administrator
accounts are created by an administrator.

**Current state (branch `feature/google-signin-authentication`):** `/register`
**and** `/login` are Google-only and use **Google Sign-In (Google Identity
Services)**. Neither page has an email/password or OTP form. The old two-step
email-OTP signup endpoints still exist in the backend for backward compatibility,
but the pages no longer call them. See
[`../implementation/GOOGLE_SIGNIN.md`](../implementation/GOOGLE_SIGNIN.md).

Google flow:

1. The page renders Google's own **Sign in with Google** button
   (`google.accounts.id.renderButton()`, via `NEXT_PUBLIC_GOOGLE_CLIENT_ID`). It
   uses the button flow with `use_fedcm_for_button: true`, not One Tap.
2. Google returns a signed ID token to the browser.
3. `POST /api/v1/auth/google` verifies it and creates or links the account. A
   student address may create a new account; a staff address only links into an
   account that already exists.

Legacy two-step flow (still mounted):

1. `POST /api/v1/auth/signup/initiate` — sends a 6-digit verification code to the email.
2. `POST /api/v1/auth/signup/complete` — validates the code and creates the account.

---

## Files Changed

| File | Change |
|---|---|
| `apps/web/app/register/page.tsx` | Google-only registration page |
| `apps/web/components/google-signin.tsx` | Google's own Sign in with Google button (`renderButton()`, FedCM button flow) |
| `apps/web/lib/api.ts` | Added `auth.googleSignIn`; existing signup/verify helpers retained |
| `apps/web/app/login/page.tsx` | Google-only sign-in page; email/password form removed |

---

## Domain Validation

There is no registration form, so nothing is validated client-side any more. The
Google endpoint enforces the domain **on the server** from the verified ID token:

- `@std.must.ac.ug` — students (may create an account)
- `@must.ac.ug` — staff and official accounts (sign-in only; the account must
  already exist)

The Google endpoint accepts only these domains; any other verified address is
rejected with `403`. Domain membership is still not proof of faculty/department
membership — eligibility is enforced separately by the eligibility layer (roster
import / freeze).

---

## Flow

```
/register  (or /login)
  ↓  user clicks Google's own "Sign in with Google" button → Google popup
  ↓  Google returns a signed ID token to the page
  ↓  POST /auth/google  →  API verifies signature/issuer/audience/expiry,
  ↓                        email_verified, student or staff domain, hosted domain
  ↓  student account created (first time) or an existing account linked by
  ↓  verified email; a staff address must already have an account
  → redirect to /elections
```

The legacy OTP flow is still available through `/auth/signup/initiate` and
`/auth/signup/complete` for clients that call it directly.

In development mode, the backend echoes OTP verification codes in the JSON
response body when mail delivery fails. That fallback belongs to the legacy OTP
flow only; Google sign-in does not use it.

---

## Password Strength

Google sign-in has no password, and neither `/login` nor `/register` renders a
password form. The `PasswordStrength` component and `passwordFailures` utility
in `lib/password-rules.ts` are retained for the legacy OTP flow (which still
creates password accounts) and for administrator accounts created by
`cmd/bootstrap`.

---

## Acceptance Criteria Status

| Criterion | Status |
|---|---|
| Students can self-register with @std.must.ac.ug or @must.ac.ug email | ✅ (students self-register; staff/admin accounts are created by an administrator and then sign in) |
| Identity is verified before an account is created | ✅ (Google ID token; legacy OTP flow retained) |
| Clear error messages for invalid domains | ✅ |
| "Register" link on login page | ✅ |
| "Resend verification email" functionality | ✅ (legacy OTP flow) |
| Registration flow is documented | ✅ |

---

## Known Limitations

- Google proves control of a Google account, not university enrolment. The
  student-domain check is policy, not proof of faculty/department membership;
  election eligibility is enforced separately via roster import.
- The registration page requires a configured Google client. Without it the page
  shows a visible "Google sign-in is unavailable" banner.
- The backend's `POST /auth/verify-email` (for re-verifying already-registered users) is wired in `api.ts` but has no dedicated UI page yet — it can be called programmatically.
