"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect } from "react";
import { useAuth } from "@/components/auth-provider";
import { GoogleSignInButton } from "@/components/google-signin";
import { Icon } from "@/components/icon";
import { ThemeToggle } from "@/components/theme-toggle";
import { CAMPUS_NAME } from "@/lib/config";

/**
 * Student sign-in with Google only.
 *
 * Google Identity Services returns an ID token to this page; the backend
 * verifies it and signs the account in. Administrator password sign-in lives
 * on its own page and is linked from this page's footer.
 */
export default function LoginPage() {
  const { user, loading, signInWithGoogle } = useAuth();
  const router = useRouter();

  useEffect(() => {
    if (!loading && user) router.replace(user.role === "ADMIN" ? "/staff" : "/elections");
  }, [loading, user, router]);

  return (
    <main className="auth-wrap">
      <div className="auth-content">
        <section className="auth-card" aria-labelledby="login-title">
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
          <h1 id="login-title">Campus elections</h1>
          <p className="lead">
            Students, sign in with your {CAMPUS_NAME} Google account to see the elections you are
            eligible for.
          </p>

          <GoogleSignInButton
            onCredential={async (idToken) => {
              await signInWithGoogle(idToken);
              router.replace("/elections");
            }}
          />

          <p
            style={{
              marginTop: 20,
              fontSize: 14,
              textAlign: "center",
              color: "var(--ink-3)",
            }}
          >
            Don&apos;t have an account?{" "}
            <Link href="/register" style={{ color: "var(--primary)" }}>
              Create one
            </Link>
          </p>

          <div className="banner" style={{ marginTop: 20 }}>
            <Icon name="lock" />
            <div>
              <b>Your ballot is secret</b>
              Officials and candidates cannot see your choices. Sign-in protects your eligibility,
              not your vote.
            </div>
          </div>
        </section>
        <footer className="auth-footer">
          <span>University election administrators</span>
          <Link href="/admin/login">Administrator sign-in</Link>
        </footer>
      </div>
    </main>
  );
}
