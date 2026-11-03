"use client";

import { AppShell } from "./app-shell";

/**
 * Admin pages use the shared role-aware shell. The election setup steps are
 * derived from the URL, so pages only pass an optional title.
 */
export function StaffShell({ children, title }: { children: React.ReactNode; title?: string }) {
  return <AppShell title={title}>{children}</AppShell>;
}
