"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { useAuth } from "@/components/auth-provider";
import { Icon } from "@/components/icon";
import { ApiError } from "@/lib/api";
import {
  CAMPUS_NAME,
  DEMO_STUDENT_EMAIL,
  DEMO_STUDENT_PASSWORD,
  STAFF_EMAIL_DOMAIN,
  STUDENT_EMAIL_DOMAIN,
} from "@/lib/config";

export default function LoginPage() {
  const { user, loading, signIn } = useAuth();
  const router = useRouter();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (!loading && user) router.replace("/elections");
  }, [loading, user, router]);

  async function handleSubmit(event: React.FormEvent) {
    event.preventDefault();
    setError("");
    setBusy(true);
    try {
      await signIn(email.trim(), password);
      router.replace("/elections");
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "We could not sign you in. Try again.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <main className="auth-wrap">
      <div className="auth-card">
        <Link className="brand row" href="/login" style={{ marginBottom: 24 }}>
          <span className="mark">
            <Icon name="moon" size="l" />
          </span>
          <span style={{ fontWeight: 700, fontSize: 18 }}>
            camp<b style={{ fontWeight: 400 }}>Vote</b>
          </span>
        </Link>
        <h1>Campus elections</h1>
        <p className="lead">
          Sign in with your {CAMPUS_NAME} account to see the elections you are eligible for.
        </p>

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
          <div className="field">
            <label htmlFor="password">Password</label>
            <input
              id="password"
              className={`in${error ? " bad" : ""}`}
              type="password"
              autoComplete="current-password"
              value={password}
              onChange={(event) => setPassword(event.target.value)}
              required
            />
          </div>
          {error ? (
            <span className="field err" role="alert">
              <Icon name="alert" size="s" />
              {error}
            </span>
          ) : null}
          <button className="btn primary block lg" type="submit" disabled={busy}>
            {busy ? "Signing in…" : "Sign in"}
          </button>
        </form>

        <p style={{ marginTop: 16, fontSize: 14, textAlign: "center", color: "var(--ink-3)" }}>
          Don&apos;t have an account?{" "}
          <Link href="/register" style={{ color: "var(--primary)" }}>
            Create one
          </Link>
        </p>

        <div className="banner" style={{ marginTop: 20 }}>
          <Icon name="lock" />
          <div>
            <b>Your ballot is secret</b>
            Officials and candidates cannot see your choices. Sign-in protects your eligibility, not
            your vote.
          </div>
        </div>

        {process.env.NODE_ENV !== "production" ? (
          <p className="hint" style={{ marginTop: 16 }}>
            Local demo voter: <span className="mono">{DEMO_STUDENT_EMAIL}</span> /{" "}
            <span className="mono">{DEMO_STUDENT_PASSWORD}</span>. Development data only.
          </p>
        ) : null}
      </div>
    </main>
  );
}
