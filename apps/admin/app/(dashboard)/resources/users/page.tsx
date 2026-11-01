"use client";

import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { PageHeader } from "@/components/chrome/PageHeader";
import { apiClient } from "@/lib/api-client";
import { getApiErrorMessage } from "@/lib/api-core";
import { ConfirmModal } from "@/components/ui/confirm-modal";
import { AlertTriangle, Pencil, Plus, Search, Trash2, UserCheck, UserX, X } from "@/lib/icons";

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

type Role =
  | "ADMIN"
  | "EDITOR"
  | "USER"
  | "CHIEF_ELECTION_ADMIN"
  | "ELECTION_OFFICIAL"
  | "STUDENT_VOTER"
  | "STAFF_VOTER"
  | "CANDIDATE"
  | "ELECTION_AUDITOR";

interface User {
  id: string;
  first_name?: string;
  last_name?: string;
  email: string;
  role: Role | string;
  active: boolean;
  email_verified_at?: string | null;
  created_at: string;
}

interface UsersListResponse {
  data: User[];
  meta?: { total?: number; page?: number; per_page?: number };
}

interface UserFormValues {
  first_name: string;
  last_name: string;
  email: string;
  role: string;
  password?: string;
  active: boolean;
}

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const ROLES: { value: string; label: string }[] = [
  { value: "USER", label: "User" },
  { value: "STUDENT_VOTER", label: "Student Voter" },
  { value: "STAFF_VOTER", label: "Staff Voter" },
  { value: "CANDIDATE", label: "Candidate" },
  { value: "ELECTION_OFFICIAL", label: "Election Official" },
  { value: "CHIEF_ELECTION_ADMIN", label: "Chief Election Admin" },
  { value: "ELECTION_AUDITOR", label: "Election Auditor" },
  { value: "EDITOR", label: "Editor" },
  { value: "ADMIN", label: "Admin" },
];

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function fullName(user: User): string {
  return [user.first_name, user.last_name].filter(Boolean).join(" ") || "—";
}

function when(iso?: string): string {
  if (!iso) return "—";
  const d = new Date(iso);
  return Number.isNaN(d.getTime())
    ? "—"
    : d.toLocaleDateString(undefined, { day: "numeric", month: "short", year: "numeric" });
}

const inputClass =
  "w-full rounded-lg border border-border bg-bg-secondary px-3 py-2.5 text-sm text-foreground placeholder:text-text-muted focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent";

// ---------------------------------------------------------------------------
// User form modal
// ---------------------------------------------------------------------------

interface UserFormModalProps {
  /** null = create, User = edit */
  user: User | null;
  onClose: () => void;
  onSaved: () => void;
}

function UserFormModal({ user, onClose, onSaved }: UserFormModalProps) {
  const isEdit = user !== null;
  const [values, setValues] = useState<UserFormValues>({
    first_name: user?.first_name ?? "",
    last_name: user?.last_name ?? "",
    email: user?.email ?? "",
    role: user?.role ?? "USER",
    password: "",
    active: user?.active ?? true,
  });
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  function set<K extends keyof UserFormValues>(key: K, val: UserFormValues[K]) {
    setValues((prev) => ({ ...prev, [key]: val }));
  }

  async function handleSubmit(event: React.FormEvent) {
    event.preventDefault();
    setError(null);
    setBusy(true);

    try {
      const body: Record<string, unknown> = {
        first_name: values.first_name.trim(),
        last_name: values.last_name.trim(),
        email: values.email.trim(),
        role: values.role,
        active: values.active,
      };
      // Only include password when provided — on edit, leaving it blank means "no change".
      if (values.password && values.password.trim()) {
        body.password = values.password;
      }

      if (isEdit) {
        await apiClient.put(`/api/users/${user.id}`, body);
      } else {
        if (!values.password?.trim()) {
          setError("Password is required when creating a new account.");
          setBusy(false);
          return;
        }
        await apiClient.post("/api/users", body);
      }
      onSaved();
      onClose();
    } catch (err) {
      setError(
        getApiErrorMessage(err, isEdit ? "Could not update user." : "Could not create user.")
      );
    } finally {
      setBusy(false);
    }
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-scrim p-4"
      role="dialog"
      aria-modal="true"
      aria-labelledby="user-form-title"
    >
      <div className="w-full max-w-lg rounded-xl border border-border bg-bg-elevated shadow-xl">
        {/* Header */}
        <div className="flex items-center justify-between border-b border-border px-6 py-4">
          <h2 id="user-form-title" className="text-lg font-semibold text-foreground">
            {isEdit ? "Edit user" : "Create user"}
          </h2>
          <button
            type="button"
            onClick={onClose}
            className="rounded-lg p-1 text-text-muted transition-colors hover:bg-bg-hover hover:text-foreground"
            aria-label="Close dialog"
          >
            <X className="h-5 w-5" aria-hidden="true" />
          </button>
        </div>

        {/* Form */}
        <form onSubmit={handleSubmit} noValidate>
          <div className="space-y-4 px-6 py-5">
            {error && (
              <div
                role="alert"
                className="flex items-start gap-2 rounded-lg border border-danger/30 bg-danger/5 px-4 py-3 text-sm text-danger"
              >
                <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" aria-hidden="true" />
                <span>{error}</span>
              </div>
            )}

            <div className="grid grid-cols-2 gap-3">
              <div className="flex flex-col gap-1.5">
                <label htmlFor="uf-first-name" className="text-sm font-medium text-foreground">
                  First name
                </label>
                <input
                  id="uf-first-name"
                  type="text"
                  className={inputClass}
                  value={values.first_name}
                  onChange={(e) => set("first_name", e.target.value)}
                  autoComplete="given-name"
                  required
                />
              </div>
              <div className="flex flex-col gap-1.5">
                <label htmlFor="uf-last-name" className="text-sm font-medium text-foreground">
                  Last name
                </label>
                <input
                  id="uf-last-name"
                  type="text"
                  className={inputClass}
                  value={values.last_name}
                  onChange={(e) => set("last_name", e.target.value)}
                  autoComplete="family-name"
                  required
                />
              </div>
            </div>

            <div className="flex flex-col gap-1.5">
              <label htmlFor="uf-email" className="text-sm font-medium text-foreground">
                Email
              </label>
              <input
                id="uf-email"
                type="email"
                className={inputClass}
                value={values.email}
                onChange={(e) => set("email", e.target.value)}
                autoComplete="email"
                required
              />
            </div>

            <div className="flex flex-col gap-1.5">
              <label htmlFor="uf-role" className="text-sm font-medium text-foreground">
                Role
              </label>
              <select
                id="uf-role"
                className={inputClass}
                value={values.role}
                onChange={(e) => set("role", e.target.value)}
              >
                {ROLES.map((r) => (
                  <option key={r.value} value={r.value}>
                    {r.label}
                  </option>
                ))}
              </select>
            </div>

            <div className="flex flex-col gap-1.5">
              <label htmlFor="uf-password" className="text-sm font-medium text-foreground">
                Password{isEdit ? " (leave blank to keep current)" : ""}
              </label>
              <input
                id="uf-password"
                type="password"
                className={inputClass}
                value={values.password ?? ""}
                onChange={(e) => set("password", e.target.value)}
                autoComplete="new-password"
                required={!isEdit}
                minLength={8}
                placeholder={isEdit ? "Leave blank to keep current" : "Min. 8 characters"}
              />
            </div>

            <div className="flex items-center gap-3">
              <input
                id="uf-active"
                type="checkbox"
                className="h-4 w-4 rounded border-border accent-accent"
                checked={values.active}
                onChange={(e) => set("active", e.target.checked)}
              />
              <label htmlFor="uf-active" className="text-sm text-foreground">
                Account is active (user can sign in)
              </label>
            </div>
          </div>

          {/* Footer */}
          <div className="flex justify-end gap-2 border-t border-border bg-bg-secondary px-6 py-4">
            <button
              type="button"
              onClick={onClose}
              className="rounded-lg border border-border px-4 py-2 text-sm text-text-secondary transition-colors hover:bg-bg-hover hover:text-foreground"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={busy}
              className="rounded-lg bg-accent px-4 py-2 text-sm font-medium text-white transition-opacity hover:opacity-90 disabled:opacity-50"
            >
              {busy ? "Saving…" : isEdit ? "Save changes" : "Create user"}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Main page
// ---------------------------------------------------------------------------

export default function UsersPage() {
  const queryClient = useQueryClient();

  // Filters
  const [search, setSearch] = useState("");
  const [roleFilter, setRoleFilter] = useState<string>("all");
  const [statusFilter, setStatusFilter] = useState<"all" | "active" | "inactive">("all");

  // Bulk selection
  const [selected, setSelected] = useState<Set<string>>(new Set());

  // Modals
  const [editTarget, setEditTarget] = useState<User | null | "new">(null);
  const [deleteTarget, setDeleteTarget] = useState<User | null>(null);
  const [bulkAction, setBulkAction] = useState<"suspend" | "unsuspend" | null>(null);

  const [error, setError] = useState<string | null>(null);

  // ── Data fetching ─────────────────────────────────────────────────────────

  const { data, isPending } = useQuery<UsersListResponse>({
    queryKey: ["users"],
    queryFn: async () => (await apiClient.get<UsersListResponse>("/api/users")).data,
  });

  function refresh() {
    void queryClient.invalidateQueries({ queryKey: ["users"] });
    setSelected(new Set());
  }

  // ── Mutations ─────────────────────────────────────────────────────────────

  const deleteUser = useMutation({
    mutationFn: (user: User) => apiClient.delete(`/api/users/${user.id}`),
    onSuccess: refresh,
    onError: (e) => setError(getApiErrorMessage(e, "Could not delete that user.")),
  });

  const bulkUpdate = useMutation({
    mutationFn: async ({ ids, active }: { ids: string[]; active: boolean }) => {
      await Promise.all(ids.map((id) => apiClient.put(`/api/users/${id}`, { active })));
    },
    onSuccess: refresh,
    onError: (e) => setError(getApiErrorMessage(e, "Could not update those users.")),
  });

  // ── Filtering ─────────────────────────────────────────────────────────────

  const allUsers = data?.data ?? [];

  const filtered = allUsers.filter((u) => {
    const q = search.toLowerCase();
    const matchesSearch =
      !q || u.email.toLowerCase().includes(q) || fullName(u).toLowerCase().includes(q);

    const matchesRole = roleFilter === "all" || u.role === roleFilter;
    const matchesStatus =
      statusFilter === "all" ||
      (statusFilter === "active" && u.active) ||
      (statusFilter === "inactive" && !u.active);

    return matchesSearch && matchesRole && matchesStatus;
  });

  // ── Bulk selection helpers ────────────────────────────────────────────────

  function toggleSelect(id: string) {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }

  function toggleSelectAll() {
    if (selected.size === filtered.length) {
      setSelected(new Set());
    } else {
      setSelected(new Set(filtered.map((u) => u.id)));
    }
  }

  function handleBulkConfirm() {
    if (!bulkAction) return;
    bulkUpdate.mutate({ ids: Array.from(selected), active: bulkAction === "unsuspend" });
    setBulkAction(null);
  }

  // ── Render ────────────────────────────────────────────────────────────────

  return (
    <div>
      <PageHeader
        title="Users"
        subtitle={`${allUsers.length} account${allUsers.length === 1 ? "" : "s"} · manage roles, status, and access.`}
        action={
          <button
            type="button"
            onClick={() => setEditTarget("new")}
            className="inline-flex items-center gap-2 rounded-lg bg-accent px-4 py-2 text-sm font-medium text-white transition-opacity hover:opacity-90"
          >
            <Plus className="h-4 w-4" aria-hidden="true" />
            Create user
          </button>
        }
      />

      {/* Error banner */}
      {error && (
        <div
          role="alert"
          className="mb-4 flex items-start gap-2 rounded-lg border border-danger/30 bg-danger/5 px-4 py-3 text-sm text-danger"
        >
          <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" aria-hidden="true" />
          <span>{error}</span>
          <button
            type="button"
            onClick={() => setError(null)}
            className="ml-auto shrink-0 text-danger/70 hover:text-danger"
            aria-label="Dismiss error"
          >
            <X className="h-4 w-4" aria-hidden="true" />
          </button>
        </div>
      )}

      {/* Filters bar */}
      <div className="mb-4 flex flex-wrap items-center gap-3">
        {/* Search */}
        <div className="relative flex-1 min-w-[200px] max-w-sm">
          <Search
            className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-text-muted"
            aria-hidden="true"
          />
          <input
            type="search"
            placeholder="Search name or email…"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="w-full rounded-lg border border-border bg-bg-secondary py-2 pl-9 pr-3 text-sm text-foreground placeholder:text-text-muted focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent"
            aria-label="Search users"
          />
        </div>

        {/* Role filter */}
        <select
          value={roleFilter}
          onChange={(e) => setRoleFilter(e.target.value)}
          className="rounded-lg border border-border bg-bg-secondary px-3 py-2 text-sm text-foreground focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent"
          aria-label="Filter by role"
        >
          <option value="all">All roles</option>
          {ROLES.map((r) => (
            <option key={r.value} value={r.value}>
              {r.label}
            </option>
          ))}
        </select>

        {/* Status filter */}
        <select
          value={statusFilter}
          onChange={(e) => setStatusFilter(e.target.value as typeof statusFilter)}
          className="rounded-lg border border-border bg-bg-secondary px-3 py-2 text-sm text-foreground focus:border-accent focus:outline-none focus:ring-1 focus:ring-accent"
          aria-label="Filter by status"
        >
          <option value="all">All statuses</option>
          <option value="active">Active</option>
          <option value="inactive">Suspended</option>
        </select>
      </div>

      {/* Bulk actions bar */}
      {selected.size > 0 && (
        <div className="mb-4 flex items-center gap-3 rounded-lg border border-border bg-bg-secondary px-4 py-2.5 text-sm">
          <span className="text-text-secondary">
            <strong className="text-foreground">{selected.size}</strong> selected
          </span>
          <div className="flex gap-2">
            <button
              type="button"
              onClick={() => setBulkAction("unsuspend")}
              disabled={bulkUpdate.isPending}
              className="inline-flex items-center gap-1.5 rounded-lg border border-border px-2.5 py-1 text-xs transition-colors hover:bg-bg-hover"
            >
              <UserCheck className="h-3.5 w-3.5" aria-hidden="true" />
              Activate
            </button>
            <button
              type="button"
              onClick={() => setBulkAction("suspend")}
              disabled={bulkUpdate.isPending}
              className="inline-flex items-center gap-1.5 rounded-lg border border-border px-2.5 py-1 text-xs text-danger transition-colors hover:bg-danger/10"
            >
              <UserX className="h-3.5 w-3.5" aria-hidden="true" />
              Suspend
            </button>
          </div>
          <button
            type="button"
            onClick={() => setSelected(new Set())}
            className="ml-auto text-text-muted hover:text-foreground"
            aria-label="Clear selection"
          >
            <X className="h-4 w-4" aria-hidden="true" />
          </button>
        </div>
      )}

      {/* Table */}
      <div className="rounded-xl border border-border bg-bg-elevated">
        {isPending ? (
          <p className="py-16 text-center text-sm text-text-muted">Loading…</p>
        ) : filtered.length === 0 ? (
          <p className="py-16 text-center text-sm text-text-muted">
            {allUsers.length === 0
              ? "No users found. Create the first one."
              : "No users match the current filters."}
          </p>
        ) : (
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border text-left text-xs text-text-muted">
                <th className="w-10 px-4 py-3">
                  <input
                    type="checkbox"
                    checked={selected.size === filtered.length && filtered.length > 0}
                    ref={(el) => {
                      if (el)
                        el.indeterminate = selected.size > 0 && selected.size < filtered.length;
                    }}
                    onChange={toggleSelectAll}
                    className="h-4 w-4 rounded border-border accent-accent"
                    aria-label="Select all"
                  />
                </th>
                <th className="px-4 py-3 font-medium">User</th>
                <th className="px-4 py-3 font-medium">Role</th>
                <th className="px-4 py-3 font-medium">Status</th>
                <th className="px-4 py-3 font-medium">Email verified</th>
                <th className="px-4 py-3 font-medium">Created</th>
                <th className="px-4 py-3 text-right font-medium">Actions</th>
              </tr>
            </thead>
            <tbody>
              {filtered.map((user) => (
                <tr key={user.id} className="border-b border-border/60 last:border-0">
                  <td className="px-4 py-3">
                    <input
                      type="checkbox"
                      checked={selected.has(user.id)}
                      onChange={() => toggleSelect(user.id)}
                      className="h-4 w-4 rounded border-border accent-accent"
                      aria-label={`Select ${fullName(user)}`}
                    />
                  </td>
                  <td className="px-4 py-3">
                    <span className="font-medium text-foreground">{fullName(user)}</span>
                    <span className="ml-2 font-mono text-xs text-text-muted">{user.email}</span>
                  </td>
                  <td className="px-4 py-3">
                    <RoleBadge role={user.role} />
                  </td>
                  <td className="px-4 py-3">
                    <StatusBadge active={user.active} />
                  </td>
                  <td className="px-4 py-3 text-text-secondary">
                    {user.email_verified_at ? (
                      <span className="text-green-600 dark:text-green-400">Verified</span>
                    ) : (
                      <span className="text-text-muted">Unverified</span>
                    )}
                  </td>
                  <td className="px-4 py-3 text-text-secondary">{when(user.created_at)}</td>
                  <td className="px-4 py-3">
                    <div className="flex items-center justify-end gap-1">
                      <button
                        type="button"
                        onClick={() => setEditTarget(user)}
                        className="inline-flex items-center gap-1.5 rounded-lg border border-border px-2.5 py-1 text-xs text-text-secondary transition-colors hover:bg-bg-hover hover:text-foreground"
                        aria-label={`Edit ${fullName(user)}`}
                      >
                        <Pencil className="h-3 w-3" aria-hidden="true" />
                        Edit
                      </button>
                      <button
                        type="button"
                        onClick={() => setDeleteTarget(user)}
                        className="inline-flex items-center gap-1.5 rounded-lg border border-transparent px-2.5 py-1 text-xs text-danger transition-colors hover:bg-danger/10"
                        aria-label={`Delete ${fullName(user)}`}
                      >
                        <Trash2 className="h-3 w-3" aria-hidden="true" />
                        Delete
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      <p className="mt-3 text-xs text-text-muted">
        Showing {filtered.length} of {allUsers.length} users.
      </p>

      {/* Modals */}
      {(editTarget === "new" || editTarget !== null) && editTarget !== null && (
        <UserFormModal
          user={editTarget === "new" ? null : editTarget}
          onClose={() => setEditTarget(null)}
          onSaved={refresh}
        />
      )}

      <ConfirmModal
        open={deleteTarget !== null}
        title="Delete this user?"
        description={
          deleteTarget
            ? `${fullName(deleteTarget)} (${deleteTarget.email}) will be soft-deleted. You can restore them from Deleted Accounts.`
            : ""
        }
        confirmLabel="Delete"
        variant="danger"
        onConfirm={() => {
          if (deleteTarget) deleteUser.mutate(deleteTarget);
          setDeleteTarget(null);
        }}
        onCancel={() => setDeleteTarget(null)}
      />

      <ConfirmModal
        open={bulkAction !== null}
        title={bulkAction === "suspend" ? "Suspend selected users?" : "Activate selected users?"}
        description={
          bulkAction === "suspend"
            ? `${selected.size} users will be suspended and cannot sign in until reactivated.`
            : `${selected.size} users will be reactivated and can sign in again.`
        }
        confirmLabel={bulkAction === "suspend" ? "Suspend" : "Activate"}
        variant={bulkAction === "suspend" ? "danger" : "primary"}
        onConfirm={handleBulkConfirm}
        onCancel={() => setBulkAction(null)}
      />
    </div>
  );
}

// ---------------------------------------------------------------------------
// Small presentational components
// ---------------------------------------------------------------------------

function RoleBadge({ role }: { role: string }) {
  const label = ROLES.find((r) => r.value === role)?.label ?? role;
  const isAdmin = role === "ADMIN" || role === "CHIEF_ELECTION_ADMIN";
  return (
    <span
      className={`inline-flex rounded-full px-2 py-0.5 text-xs font-medium ${
        isAdmin ? "bg-accent/10 text-accent" : "bg-bg-secondary text-text-secondary"
      }`}
    >
      {label}
    </span>
  );
}

function StatusBadge({ active }: { active: boolean }) {
  return (
    <span
      className={`inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-xs font-medium ${
        active
          ? "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400"
          : "bg-bg-secondary text-text-muted"
      }`}
    >
      <span
        className={`h-1.5 w-1.5 rounded-full ${active ? "bg-green-500" : "bg-text-muted"}`}
        aria-hidden="true"
      />
      {active ? "Active" : "Suspended"}
    </span>
  );
}
