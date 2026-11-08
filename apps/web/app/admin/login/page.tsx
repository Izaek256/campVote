"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { type FormEvent, useEffect, useState } from "react";
import { useAuth } from "@/components/auth-provider";
import { Icon } from "@/components/icon";
import { ThemeToggle } from "@/components/theme-toggle";
import { ApiError } from "@/lib/api";
import { STAFF_EMAIL_DOMAIN } from "@/lib/config";

function SignInError({ error }: { error: unknown }) {
  const message =
    error instanceof ApiError || error instanceof Error
      ? error.message
      : "Administrator sign-in failed. Please try again.";
  const requestId = error instanceof ApiError ? error.requestId : "";
  const fields = error instanceof ApiError ? error.fields : undefined;

  return (
    <div className="auth-error" role="alert">
      <p>{message}</p>
      {fields
        ? Object.entries(fields).map(([field, detail]) => (
            <p key={field}>
              {field.replaceAll("_", " ")}: {detail}
            </p>
          ))
        : null}
      {requestId ? <p className="auth-request-id">Request ID: {requestId}</p> : null}
    </div>
  );
}

export default function AdminLoginPage() {
  const { user, loading, signInAdmin, verifyAdminSecondFactor } = useAuth();
  const router = useRouter();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [code, setCode] = useState("");
  const [challenge, setChallenge] = useState<{
    pendingToken: string;
    method: string;
  } | null>(null);
  const [useBackupCode, setUseBackupCode] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<unknown>(null);

  useEffect(() => {
    if (!loading && user) router.replace(user.role === "ADMIN" ? "/staff" : "/elections");
  }, [loading, user, router]);

  async function submitPassword(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError(null);

    const normalizedEmail = email.trim().toLowerCase();
    const parts = normalizedEmail.split("@");
    if (parts.length !== 2 || parts[1] !== STAFF_EMAIL_DOMAIN.toLowerCase()) {
      setError(new Error(`Use your administrator email address ending in @${STAFF_EMAIL_DOMAIN}.`));
      return;
    }

    setSubmitting(true);
    try {
      const result = await signInAdmin(normalizedEmail, password);
      if (result.requiresSecondFactor) {
        setChallenge({ pendingToken: result.pendingToken, method: result.method });
        setPassword("");
      } else {
        router.replace("/staff");
      }
    } catch (cause) {
      setError(cause);
    } finally {
      setSubmitting(false);
    }
  }

  async function submitSecondFactor(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!challenge) return;
    setError(null);
    setSubmitting(true);
    try {
      await verifyAdminSecondFactor(challenge.pendingToken, code.trim(), useBackupCode);
      router.replace("/staff");
    } catch (cause) {
      setError(cause);
    } finally {
      setSubmitting(false);
    }
  }

  function restartSignIn() {
    setChallenge(null);
    setCode("");
    setUseBackupCode(false);
    setError(null);
  }

  return (
    <main className="auth-wrap">
      <div className="auth-content">
        <section className="auth-card" aria-labelledby="admin-login-title">
          <div className="row" style={{ justifyContent: "flex-end", marginBottom: 4 }}>
            <ThemeToggle />
          </div>
          <Link className="brand row" href="/" style={{ marginBottom: 24 }}>
            <span className="mark">
              <Icon name="ballot" size="l" />
            </span>
            <span style={{ fontWeight: 700, fontSize: 18 }}>
              camp<b style={{ fontWeight: 400 }}>Vote</b>
            </span>
          </Link>

          {loading ? (
            <p className="hint" role="status">
              Checking your session…
            </p>
          ) : challenge ? (
            <>
              <h1 id="admin-login-title">Verify it&apos;s you</h1>
              <p className="lead">
                {challenge.method === "email"
                  ? "Enter the sign-in code sent to your administrator email."
                  : "Enter the current code from your authenticator app."}
              </p>
              <form className="form-stack" onSubmit={submitSecondFactor}>
                <div className="field">
                  <label htmlFor="admin-two-factor-code">
                    {useBackupCode ? "Backup code" : "Verification code"}
                  </label>
                  <input
                    id="admin-two-factor-code"
                    className="in"
                    type="text"
                    inputMode={useBackupCode ? "text" : "numeric"}
                    autoComplete="one-time-code"
                    value={code}
                    onChange={(event) => setCode(event.target.value)}
                    required
                  />
                </div>
                {error ? <SignInError error={error} /> : null}
                <button className="btn primary block" type="submit" disabled={submitting}>
                  {submitting ? "Verifying…" : "Verify and sign in"}
                </button>
              </form>
              <div className="auth-secondary-actions">
                <button
                  className="btn ghost block"
                  type="button"
                  disabled={submitting}
                  onClick={() => {
                    setUseBackupCode((current) => !current);
                    setCode("");
                    setError(null);
                  }}
                >
                  {useBackupCode ? "Use an authenticator code" : "Use a backup code"}
                </button>
                <button
                  className="auth-text-button"
                  type="button"
                  disabled={submitting}
                  onClick={restartSignIn}
                >
                  Use another account
                </button>
              </div>
            </>
          ) : (
            <>
              <h1 id="admin-login-title">Administrator sign-in</h1>
              <p className="lead">
                Sign in with the administrator account created by your election system
                administrator.
              </p>
              <form className="form-stack" onSubmit={submitPassword}>
                <div className="field">
                  <label htmlFor="admin-email">University email</label>
                  <input
                    id="admin-email"
                    className="in"
                    type="email"
                    autoComplete="username"
                    autoCapitalize="none"
                    value={email}
                    onChange={(event) => setEmail(event.target.value)}
                    required
                  />
                  <span className="hint">Use an address ending in @{STAFF_EMAIL_DOMAIN}.</span>
                </div>
                <div className="field">
                  <label htmlFor="admin-password">Password</label>
                  <input
                    id="admin-password"
                    className="in"
                    type="password"
                    autoComplete="current-password"
                    value={password}
                    onChange={(event) => setPassword(event.target.value)}
                    required
                  />
                </div>
                {error ? <SignInError error={error} /> : null}
                <button
                  className="btn primary block"
                  type="submit"
                  disabled={submitting || loading}
                >
                  {submitting ? "Signing in…" : "Sign in"}
                </button>
              </form>
              <div className="auth-secondary-actions">
                <Link href="/forgot-password">Forgot your password?</Link>
              </div>
            </>
          )}
        </section>

        <footer className="auth-footer">
          <span>Student voter?</span>
          <Link href="/login">Continue with Google</Link>
        </footer>
      </div>
    </main>
  );
}
