"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { useAuth } from "@/components/auth-provider";
import { Icon } from "@/components/icon";
import { PasswordStrength } from "@/components/password-strength";
import { ThemeToggle } from "@/components/theme-toggle";
import { useToast } from "@/components/toast";
import { ApiError, auth } from "@/lib/api";
import { STAFF_EMAIL_DOMAIN, STUDENT_EMAIL_DOMAIN } from "@/lib/config";
import { passwordFailures } from "@/lib/password-rules";

// The backend enforces SIGNUP_ALLOWED_DOMAINS; this client check gives instant
// feedback without a round trip. Domain names come from config so the UI stays
// neutral.
function isAllowedDomain(email: string): boolean {
  const lower = email.toLowerCase().trim();
  return lower.endsWith(`@${STUDENT_EMAIL_DOMAIN}`) || lower.endsWith(`@${STAFF_EMAIL_DOMAIN}`);
}

/**
 * The self-registration page.
 *
 * Uses the two-step signup flow: POST /auth/signup/initiate sends a 6-digit
 * verification code to the address; POST /auth/signup/complete exchanges it
 * for a live session.  In the development environment the backend includes the
 * code in the initiate response so you do not need an SMTP server.
 *
 * SRS FR-AUTH-01, FR-AUTH-07.
 */
export default function RegisterPage() {
  const { user, loading, signIn } = useAuth();
  const router = useRouter();

  // Step: "details" → user fills in the form; "verify" → they type the code.
  const [step, setStep] = useState<"details" | "verify">("details");

  // Form fields
  const [firstName, setFirstName] = useState("");
  const [lastName, setLastName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [code, setCode] = useState("");

  // Dev-mode hint: backend echoes the code when not in production.
  const [devCode, setDevCode] = useState<string | null>(null);

  const [error, setError] = useState("");
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const [busy, setBusy] = useState(false);

  const toast = useToast();

  // Already signed in — bounce straight to elections.
  useEffect(() => {
    if (!loading && user) router.replace("/elections");
  }, [loading, user, router]);

  // ── Step 1: validate locally and send the verification email ──────────────

  function validateDetails(): boolean {
    const errs: Record<string, string> = {};

    if (!firstName.trim() || firstName.trim().length < 2) {
      errs.firstName = "Please enter your first name (at least 2 characters).";
    }
    if (!lastName.trim() || lastName.trim().length < 2) {
      errs.lastName = "Please enter your last name (at least 2 characters).";
    }
    if (!email.trim()) {
      errs.email = "Email is required.";
    } else if (!isAllowedDomain(email)) {
      errs.email = `Please use your @${STUDENT_EMAIL_DOMAIN} (student) or @${STAFF_EMAIL_DOMAIN} (staff/official) address.`;
    }
    const pwFails = passwordFailures(password, [email, firstName, lastName]);
    if (pwFails.length > 0) {
      errs.password = "Please fix the password issues shown below.";
    }

    setFieldErrors(errs);
    return Object.keys(errs).length === 0;
  }

  async function handleDetailsSubmit(event: React.FormEvent) {
    event.preventDefault();
    setError("");
    if (!validateDetails()) return;

    setBusy(true);
    try {
      const result = await auth.signupInitiate(email.trim());
      // Development: backend echoes the code as `verification_code`.
      if (result.verification_code) setDevCode(result.verification_code);
      setStep("verify");
    } catch (err) {
      if (err instanceof ApiError) {
        if (err.status === 409) {
          setFieldErrors({
            email: "An account with this email already exists.",
          });
        } else {
          setError(err.message);
          toast.error(err.message);
        }
      } else {
        setError("Something went wrong. Please try again.");
        toast.error("Something went wrong. Please try again.");
      }
    } finally {
      setBusy(false);
    }
  }

  // ── Step 2: complete signup with the verification code ────────────────────

  async function handleVerifySubmit(event: React.FormEvent) {
    event.preventDefault();
    setError("");

    if (!code.trim() || code.trim().length !== 6) {
      setError("Please enter the 6-digit code from your email.");
      toast.error("Please enter the 6-digit code from your email.");
      return;
    }

    setBusy(true);
    try {
      await auth.signupComplete({
        email: email.trim(),
        verification_code: code.trim(),
        first_name: firstName.trim(),
        last_name: lastName.trim(),
        password,
      });
      // signupComplete sets auth cookies; restore session via signIn or just
      // reload the me() check by redirecting — the AuthProvider will pick it up.
      // Re-login is the cleanest path: it confirms the session is live.
      await signIn(email.trim(), password);
      router.replace("/elections");
    } catch (err) {
      setError(
        err instanceof ApiError
          ? err.message
          : "We could not complete your registration. Please try again."
      );
      toast.error(
        err instanceof ApiError
          ? err.message
          : "We could not complete your registration. Please try again."
      );
    } finally {
      setBusy(false);
    }
  }

  async function handleResend() {
    setError("");
    setDevCode(null);
    setBusy(true);
    try {
      const result = await auth.signupInitiate(email.trim());
      if (result.verification_code) setDevCode(result.verification_code);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Could not resend. Please try again.");
      toast.error(err instanceof ApiError ? err.message : "Could not resend. Please try again.");
    } finally {
      setBusy(false);
    }
  }

  // ── Render ────────────────────────────────────────────────────────────────

  return (
    <main className="auth-wrap">
      <div className="auth-card">
        {/* Brand */}
        <div className="row" style={{ justifyContent: "flex-end", marginBottom: 4 }}>
          <ThemeToggle />
        </div>
        <Link className="brand row" href="/login" style={{ marginBottom: 24 }}>
          <span className="mark">
            <Icon name="ballot" size="l" />
          </span>
          <span style={{ fontWeight: 700, fontSize: 18 }}>
            camp<b style={{ fontWeight: 400 }}>Vote</b>
          </span>
        </Link>

        {step === "details" ? (
          <>
            <h1>Create your account</h1>
            <p className="lead">
              Register with your campus email to participate in campus elections.
            </p>

            <form className="form-stack" onSubmit={handleDetailsSubmit} noValidate>
              {/* Name row */}
              <div
                style={{
                  display: "grid",
                  gridTemplateColumns: "1fr 1fr",
                  gap: 12,
                }}
              >
                <div className="field">
                  <label htmlFor="first-name">First name</label>
                  <input
                    id="first-name"
                    className={`in${fieldErrors.firstName ? " bad" : ""}`}
                    type="text"
                    autoComplete="given-name"
                    value={firstName}
                    onChange={(e) => setFirstName(e.target.value)}
                    placeholder="Ada"
                    required
                    aria-describedby={fieldErrors.firstName ? "err-first-name" : undefined}
                  />
                  {fieldErrors.firstName ? (
                    <span id="err-first-name" className="err" role="alert">
                      <Icon name="alert" size="s" />
                      {fieldErrors.firstName}
                    </span>
                  ) : null}
                </div>
                <div className="field">
                  <label htmlFor="last-name">Last name</label>
                  <input
                    id="last-name"
                    className={`in${fieldErrors.lastName ? " bad" : ""}`}
                    type="text"
                    autoComplete="family-name"
                    value={lastName}
                    onChange={(e) => setLastName(e.target.value)}
                    placeholder="Lovelace"
                    required
                    aria-describedby={fieldErrors.lastName ? "err-last-name" : undefined}
                  />
                  {fieldErrors.lastName ? (
                    <span id="err-last-name" className="err" role="alert">
                      <Icon name="alert" size="s" />
                      {fieldErrors.lastName}
                    </span>
                  ) : null}
                </div>
              </div>

              {/* Email */}
              <div className="field">
                <label htmlFor="email">Campus email</label>
                <input
                  id="email"
                  className={`in${fieldErrors.email ? " bad" : ""}`}
                  type="email"
                  autoComplete="email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  placeholder={`you@${STUDENT_EMAIL_DOMAIN}`}
                  required
                  aria-describedby={fieldErrors.email ? "err-email" : "hint-email"}
                />
                {fieldErrors.email ? (
                  <span id="err-email" className="err" role="alert">
                    <Icon name="alert" size="s" />
                    {fieldErrors.email}
                  </span>
                ) : (
                  <span id="hint-email" className="hint">
                    Use your @{STUDENT_EMAIL_DOMAIN} or @{STAFF_EMAIL_DOMAIN} address.
                  </span>
                )}
              </div>

              {/* Password */}
              <div className="field">
                <label htmlFor="password">Password</label>
                <input
                  id="password"
                  className={`in${fieldErrors.password ? " bad" : ""}`}
                  type="password"
                  autoComplete="new-password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  required
                  aria-describedby="password-strength"
                />
                {fieldErrors.password ? (
                  <span className="err" role="alert">
                    <Icon name="alert" size="s" />
                    {fieldErrors.password}
                  </span>
                ) : null}
                <PasswordStrength
                  value={password}
                  about={[email, firstName, lastName]}
                  id="password-strength"
                />
              </div>

              <button className="btn primary block lg" type="submit" disabled={busy}>
                {busy ? "Sending verification…" : "Continue"}
              </button>
            </form>

            <p
              style={{
                marginTop: 20,
                fontSize: 14,
                textAlign: "center",
                color: "var(--ink-3)",
              }}
            >
              Already have an account?{" "}
              <Link href="/login" style={{ color: "var(--primary)" }}>
                Sign in
              </Link>
            </p>
          </>
        ) : (
          <>
            <h1>Check your email</h1>
            <p className="lead">
              We sent a 6-digit code to <strong>{email}</strong>. Enter it below to finish creating
              your account.
            </p>

            <form className="form-stack" onSubmit={handleVerifySubmit} noValidate>
              <div className="field">
                <label htmlFor="code">Verification code</label>
                <input
                  id="code"
                  className={`in${error ? " bad" : ""}`}
                  type="text"
                  inputMode="numeric"
                  autoComplete="one-time-code"
                  maxLength={6}
                  value={code}
                  onChange={(e) => setCode(e.target.value.replace(/\D/g, "").slice(0, 6))}
                  placeholder="123456"
                  required
                />
              </div>

              {/* Dev hint */}
              {devCode ? (
                <div className="banner ok">
                  <Icon name="info" size="s" />
                  <div>
                    <b>Development mode</b>
                    Your code is:{" "}
                    <span className="mono" style={{ letterSpacing: 4, fontWeight: 700 }}>
                      {devCode}
                    </span>
                  </div>
                </div>
              ) : null}

              <button className="btn primary block lg" type="submit" disabled={busy}>
                {busy ? "Verifying…" : "Create account"}
              </button>
            </form>

            <p
              style={{
                marginTop: 16,
                fontSize: 14,
                textAlign: "center",
                color: "var(--ink-3)",
              }}
            >
              Didn&apos;t receive it?{" "}
              <button
                className="btn ghost sm"
                type="button"
                onClick={handleResend}
                disabled={busy}
                style={{
                  display: "inline-flex",
                  fontSize: 14,
                  minHeight: "auto",
                  padding: "0 4px",
                }}
              >
                Resend code
              </button>
              {" · "}
              <button
                className="btn ghost sm"
                type="button"
                onClick={() => {
                  setStep("details");
                  setError("");
                  setCode("");
                  setDevCode(null);
                }}
                style={{
                  display: "inline-flex",
                  fontSize: 14,
                  minHeight: "auto",
                  padding: "0 4px",
                }}
              >
                Change email
              </button>
            </p>
          </>
        )}

        <div className="banner" style={{ marginTop: 20 }}>
          <Icon name="shield" />
          <div>
            <b>Your ballot is always secret</b>
            Registration identifies your eligibility. Your actual vote choices are never linked to
            your account.
          </div>
        </div>
      </div>
    </main>
  );
}
