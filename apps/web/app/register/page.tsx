"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect } from "react";
import { useAuth } from "@/components/auth-provider";
import { GoogleSignInButton } from "@/components/google-signin";
import { Icon } from "@/components/icon";
import { ThemeToggle } from "@/components/theme-toggle";
import { STUDENT_EMAIL_DOMAIN } from "@/lib/config";

/**
 * Student registration with Google.
 *
 * Google Identity Services returns an ID token to this page; the backend
 * verifies it and creates or signs in the student voter account. No email
 * verification code is sent or accepted here.
 *
 * SRS FR-AUTH-01, FR-AUTH-02, FR-AUTH-03, FR-AUTH-04.
 */
export default function RegisterPage() {
  const { user, loading, signInWithGoogle } = useAuth();
  const router = useRouter();

  useEffect(() => {
    if (!loading && user) router.replace("/elections");
  }, [loading, user, router]);

  return (
    <main className="auth-wrap">
      <div className="auth-content">
        <section className="auth-card" aria-labelledby="register-title">
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

          <h1 id="register-title">Create your account</h1>
          <p className="lead">
            Continue with your university Google account to register as a student voter.
          </p>

          <GoogleSignInButton
            text="signup_with"
            onCredential={async (idToken) => {
              await signInWithGoogle(idToken);
              router.replace("/elections");
            }}
          />

          <p className="hint" style={{ marginTop: 16 }}>
            Use your <b>@{STUDENT_EMAIL_DOMAIN}</b> student address. Staff and official accounts
            are created by an election administrator.
          </p>

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

          <div className="banner" style={{ marginTop: 20 }}>
            <Icon name="shield" />
            <div>
              <b>Your ballot is always secret</b>
              Google sign-in confirms who you are. Your actual vote choices are never linked to your
              account.
            </div>
          </div>
        </section>
      </div>
    </main>
  );
}
