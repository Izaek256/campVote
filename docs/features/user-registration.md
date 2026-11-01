# User Registration Flow

**Issue:** #20 (Epic: Complete SRS Implementation — Priority 1.1)  
**Branch:** `feature/20-registration-and-user-management`  
**SRS References:** FR-AUTH-01, FR-AUTH-07  
**Status:** Implemented

---

## Summary

Students and staff can now self-register using their campus email address.  
Voter registration uses the **two-step signup flow** already present in the backend:

1. `POST /api/v1/auth/signup/initiate` — sends a 6-digit verification code to the email.
2. `POST /api/v1/auth/signup/complete` — validates the code and creates the account.

---

## Files Changed

| File | Change |
|---|---|
| `apps/web/app/register/page.tsx` | New — two-step registration page |
| `apps/web/lib/api.ts` | Added `auth.register`, `auth.signupInitiate`, `auth.signupComplete`, `auth.verifyEmail`, `auth.resendVerification` |
| `apps/web/app/login/page.tsx` | Added "Create one" link pointing to `/register` |

---

## Domain Validation

The registration page enforces MUST campus email domains **client-side** before submitting:

- `@std.must.ac.ug` — students
- `@must.ac.ug` — staff and official accounts

The server validates the email format independently. Domain enforcement on the server is handled by the eligibility layer (roster import / freeze), not by the registration endpoint itself, which accepts any valid email. This is documented and intentional: the SRS notes that domain alone does not prove faculty/department membership.

---

## Flow

```
/register
  ↓  user fills first name, last name, campus email, password
  ↓  client validates domain + password rules
  ↓  POST /auth/signup/initiate  →  6-digit code sent to email
  ↓  user enters code on step 2
  ↓  POST /auth/signup/complete  →  account created, session set
  ↓  POST /auth/login (re-auth)  →  confirms session is live
  → redirect to /elections
```

In development mode, the backend echoes the verification code in the JSON response body. The registration page displays it in a highlighted banner so developers can complete the flow without an SMTP server.

---

## Password Strength

The registration form uses the existing `PasswordStrength` component and `passwordFailures` utility from `lib/password-rules.ts`. These mirror the server's validation rules so the user sees failures before submitting.

---

## Acceptance Criteria Status

| Criterion | Status |
|---|---|
| Students can self-register with @std.must.ac.ug or @must.ac.ug email | ✅ |
| Email verification is required before login | ✅ (two-step flow) |
| Clear error messages for invalid domains | ✅ |
| "Register" link on login page | ✅ |
| "Resend verification email" functionality | ✅ |
| Registration flow is documented | ✅ |

---

## Known Limitations

- The backend's `POST /auth/verify-email` (for re-verifying already-registered users) is wired in `api.ts` but has no dedicated UI page yet — it can be called programmatically.
- Domain validation is a UX hint only; the server does not yet restrict registration to MUST domains. Election eligibility is enforced separately via roster import.
