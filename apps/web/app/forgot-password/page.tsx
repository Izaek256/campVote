"use client";

import Link from "next/link";
import { useState } from "react";
import { Icon } from "@/components/icon";
import { ThemeToggle } from "@/components/theme-toggle";
import { useToast } from "@/components/toast";
import { ApiError, auth } from "@/lib/api";
import { CAMPUS_NAME, STUDENT_EMAIL_DOMAIN, STAFF_EMAIL_DOMAIN } from "@/lib/config";

export default function ForgotPasswordPage() {
  const toast = useToast();
  const [email, setEmail] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const [sent, setSent] = useState(false);

  async function handleSubmit(event: React.FormEvent) {
    event.preventDefault();
    setError("");
    setBusy(true);
    try {
      await auth.forgotPassword(email.trim());
      setSent(true);
      toast.success("If that address exists, a password reset link has been sent to it.");
    } catch (err) {
      setError(
        err instanceof ApiError ? err.message : "We could not send a reset link. Try again."
      );
      toast.error(
        err instanceof ApiError ? err.message : "We could not send a reset link. Try again."
      );
    } finally {
      setBusy(false);
    }
  }

  return (
    <main className="auth-wrap">
      <div className="auth-card">
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
        <h1>Reset your password</h1>
        <p className="lead">
          Enter your {CAMPUS_NAME} email address and we&apos;ll send you a link to reset your
          password.
        </p>

        {sent ? (
          <div className="panel stack" style={{ marginBottom: 20 }}>
            <div style={{ color: "var(--success)" }}>
              <Icon name="mail" size="l" />
            </div>
            <h2 style={{ marginTop: 12 }}>Check your email</h2>
            <p className="hint">
              We&apos;ve sent a password reset link to <b>{email}</b>. The link expires in 1 hour.
            </p>
            <p className="hint" style={{ marginTop: 8 }}>
              Didn&apos;t receive it? Check your spam folder or try again.
            </p>
            <button
              className="btn"
              type="button"
              onClick={() => setSent(false)}
              style={{ marginTop: 16 }}
            >
              Try another email
            </button>
          </div>
        ) : (
          <form className="form-stack" onSubmit={handleSubmit} noValidate>
            <div className="field">
              <label htmlFor="email">Email</label>
              <input
                id="email"
                className={`in${error ? " bad" : ""}`}
                type="email"
                autoComplete="username"
                value={email}
                onChange={(event) => setEmail(event.target.value)}
                placeholder={`you@${STUDENT_EMAIL_DOMAIN}`}
                required
              />
              <span className="hint">
                Use your verified @{STUDENT_EMAIL_DOMAIN} or @{STAFF_EMAIL_DOMAIN} address.
              </span>
            </div>
            <button className="btn primary block lg" type="submit" disabled={busy}>
              {busy ? "Sending…" : "Send reset link"}
            </button>
          </form>
        )}

        <p
          style={{
            marginTop: 16,
            fontSize: 14,
            textAlign: "center",
            color: "var(--ink-3)",
          }}
        >
          Remember your password?{" "}
          <Link href="/login" style={{ color: "var(--primary)" }}>
            Sign in
          </Link>
        </p>
      </div>
    </main>
  );
}
