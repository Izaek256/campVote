/**
 * Dashboard shell layout.
 *
 * Wraps every page inside the (dashboard) route group. Provides a minimal
 * two-column shell (sidebar + main) so admin pages have a consistent chrome.
 * Authentication enforcement will be added here once the auth layer is wired;
 * for now the layout simply renders children.
 */
export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex min-h-screen bg-bg">
      {/* ── Sidebar ─────────────────────────────────────────────────── */}
      <aside className="hidden w-56 shrink-0 flex-col border-r border-border bg-surface md:flex">
        <div className="flex h-14 items-center gap-2 border-b border-border px-4">
          <span className="h-6 w-6 rounded bg-foreground" aria-hidden="true" />
          <span className="font-bold text-foreground">
            camp<span className="font-normal">Vote</span>
          </span>
          <span className="ml-1 rounded border border-border px-1.5 py-0.5 font-mono text-xs text-text-muted">
            Admin
          </span>
        </div>
        <nav className="flex flex-col gap-0.5 p-3" aria-label="Dashboard navigation">
          <NavLink href="/account">Account</NavLink>
          <NavLink href="/resources/users">Users</NavLink>
          <div className="my-1 border-t border-border" />
          <NavLink href="/system/deleted-accounts">Deleted accounts</NavLink>
          <NavLink href="/system/trash">Trash</NavLink>
        </nav>
      </aside>

      {/* ── Main content ─────────────────────────────────────────────── */}
      <main className="min-w-0 flex-1 p-6 md:p-8">{children}</main>
    </div>
  );
}

function NavLink({ href, children }: { href: string; children: React.ReactNode }) {
  return (
    <a
      href={href}
      className="flex items-center rounded-lg px-3 py-2 text-sm font-medium text-text-secondary transition-colors hover:bg-bg-secondary hover:text-foreground"
    >
      {children}
    </a>
  );
}
