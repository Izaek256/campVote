"use client";

import { useRouter } from "next/navigation";
import { useAuth } from "@/components/auth-provider";
import { Icon } from "@/components/icon";
import { RequireAuth } from "@/components/require-auth";
import { SiteHeader } from "@/components/site-header";

function AccountView() {
  const { user, signOut } = useAuth();
  const router = useRouter();
  if (!user) return null;

  async function handleSignOut() {
    await signOut();
    router.push("/login");
  }

  return (
    <main className="app-wrap narrow">
      <div className="page-head">
        <div>
          <span className="lbl">Account</span>
          <h1>
            {user.first_name} {user.last_name}
          </h1>
          <p>{user.email}</p>
        </div>
      </div>

      <div className="panel stack">
        <dl className="kv">
          <dt>Role</dt>
          <dd>{user.role.replaceAll("_", " ").toLowerCase()}</dd>
          <dt>Email verified</dt>
          <dd>{user.email_verified_at ? "Yes" : "No"}</dd>
          <dt>Account created</dt>
          <dd>{new Date(user.created_at).toLocaleDateString("en-GB")}</dd>
        </dl>
        <div className="banner">
          <Icon name="lock" />
          <div>
            <b>Your ballot is secret</b>
            Your account proves you are eligible. It is never stored beside your choices.
          </div>
        </div>
        <div>
          <button type="button" className="btn danger-o" onClick={handleSignOut}>
            <Icon name="out" />
            Sign out
          </button>
        </div>
      </div>
    </main>
  );
}

export default function AccountPage() {
  return (
    <RequireAuth>
      <SiteHeader />
      <AccountView />
    </RequireAuth>
  );
}
