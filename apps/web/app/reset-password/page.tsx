"use client";

import Link from "next/link";
import { Suspense, useState } from "react";
import { useSearchParams } from "next/navigation";
import { Icon } from "@/components/icon";
import { ThemeToggle } from "@/components/theme-toggle";
import { useToast } from "@/components/toast";
import { ApiError, auth } from "@/lib/api";

function ResetPasswordForm() {
  const searchParams = useSearchParams();
  const toast = useToast();

  const token = searchParams.get("token") || "";
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const [done, setDone] = useState(false);

  async function handleSubmit(event: React.FormEvent) {
    event.preventDefault();
    setError("");

    if (password.length < 8) {
      setError("Password must be at least 8 characters");
      return;
    }

    if (password !== confirmPassword) {
      setError("Passwords do not match");
      return;
    }

    if (!token) {
      setError("Invalid or expired reset link");
      return;
    }

    setBusy(true);
    try {
      await auth.resetPassword({ token, password });
      setDone(true);
      toast.success("Your password has been reset. Please sign in with your new password.");
    } catch (err) {
      setError(
        err instanceof ApiError ? err.message : "We could not reset your password. Try again."
      );
      toast.error(
        err instanceof ApiError ? err.message : "We could not reset your password. Try again."
      );
    } finally {
      setBusy(false);
    }
  }

  if (!token) {
    return (
      <main className="auth-wrap">
        <div className="auth-card">
          <div className="row" style={{ justifyContent: "flex-end", marginBottom: 4 }}>
            <ThemeToggle />
          </div>
          <div style={{ color: "var(--danger)" }}>
            <Icon name="alert" size="l" />
          </div>
          <h1 style={{ marginTop: 16 }}>Invalid reset link</h1>
          <p className="lead">
            This password reset link is invalid or has expired. Please request a new one.
          </p>
          <Link className="btn primary block lg" href="/forgot-password" style={{ marginTop: 20 }}>
            Request new reset link
          </Link>
        </div>
      </main>
    );
  }

  if (done) {
    return (
      <main className="auth-wrap">
        <div className="auth-card">
          <div className="row" style={{ justifyContent: "flex-end", marginBottom: 4 }}>
            <ThemeToggle />
          </div>
          <div style={{ color: "var(--success)" }}>
            <Icon name="check" size="l" />
          </div>
          <h1 style={{ marginTop: 16 }}>Password reset</h1>
          <p className="lead">
            Your password has been successfully reset. You can now sign in with your new password.
          </p>
          <Link className="btn primary block lg" href="/login" style={{ marginTop: 20 }}>
            Sign in
          </Link>
        </div>
      </main>
    );
  }

  return (
    <main className="auth-wrap">
      <div className="auth-card">
        <div className="row" style={{ justifyContent: "flex-end", marginBottom: 4 }}>
          <ThemeToggle />
        </div>
        <Link className="brand row" href="/login" style={{ marginBottom: 24 }}>
          <span className="mark">
            <Icon name="moon" size="l" />
          </span>
          <span style={{ fontWeight: 700, fontSize: 18 }}>
            camp<b style={{ fontWeight: 400 }}>Vote</b>
          </span>
        </Link>
        <h1>Set new password</h1>
        <p className="lead">
          Enter your new password below. Make sure it's at least 8 characters long.
        </p>

        <form className="form-stack" onSubmit={handleSubmit} noValidate>
          <div className="field">
            <label htmlFor="password">New password</label>
            <input
              id="password"
              className={`in${error ? " bad" : ""}`}
              type="password"
              autoComplete="new-password"
              value={password}
              onChange={(event) => setPassword(event.target.value)}
              placeholder="At least 8 characters"
              required
              minLength={8}
            />
          </div>
          <div className="field">
            <label htmlFor="confirm-password">Confirm password</label>
            <input
              id="confirm-password"
              className={`in${error ? " bad" : ""}`}
              type="password"
              autoComplete="new-password"
              value={confirmPassword}
              onChange={(event) => setConfirmPassword(event.target.value)}
              placeholder="Same as above"
              required
              minLength={8}
            />
          </div>
          {error && (
            <div className="banner bad" style={{ marginTop: 12 }}>
              <Icon name="alert" size="s" />
              <span>{error}</span>
            </div>
          )}
          <button className="btn primary block lg" type="submit" disabled={busy}>
            {busy ? "Resetting…" : "Reset password"}
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
          Remember your password?{" "}
          <Link href="/login" style={{ color: "var(--primary)" }}>
            Sign in
          </Link>
        </p>
      </div>
    </main>
  );
}

export default function ResetPasswordPage() {
  return (
    <Suspense fallback={<div className="auth-wrap"><div className="auth-card">Loading...</div></div>}>
      <ResetPasswordForm />
    </Suspense>
  );
}
