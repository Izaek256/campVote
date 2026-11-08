"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { Icon } from "@/components/icon";
import { LoadError } from "@/components/load-error";
import { RequireAuth } from "@/components/require-auth";
import { StaffShell } from "@/components/staff-shell";
import { StatusChip } from "@/components/status-chip";
import { useToast } from "@/components/toast";
import { ApiError, elections, auth } from "@/lib/api";
import { categoryLabel, formatShortEAT } from "@/lib/format";
import type { Election } from "@/lib/types";

// Keep creation minimal: a title and the voting window are all that is needed
// to start. Everything else (category, description, organising unit, results
// policy, eligibility) falls back to a safe server default and can be set
// later on the election page.
const emptyForm = {
  title: "",
  starts_at: "",
  ends_at: "",
};

/** datetime-local is rendered as EAT; the API stores UTC. */
function toISO(value: string): string {
  return value ? `${value}:00+03:00` : "";
}

const pad = (value: number) => String(value).padStart(2, "0");

/** Add a length to a datetime-local value (both stay in the same local wall clock). */
function addMinutes(value: string, minutes: number): string {
  if (!value) return "";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "";
  date.setMinutes(date.getMinutes() + minutes);
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}T${pad(
    date.getHours()
  )}:${pad(date.getMinutes())}`;
}

// Officials may run anything from a two-minute demonstration poll to a
// multi-day students' guild election.
const durationPresets = [
  { label: "2 minutes", minutes: 2 },
  { label: "1 hour", minutes: 60 },
  { label: "6 hours", minutes: 6 * 60 },
  { label: "20 hours", minutes: 20 * 60 },
  { label: "1 day", minutes: 24 * 60 },
  { label: "1 week", minutes: 7 * 24 * 60 },
];

function StaffHome() {
  const router = useRouter();
  const toast = useToast();
  const [items, setItems] = useState<Election[] | null>(null);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [showForm, setShowForm] = useState(false);
  const [form, setForm] = useState(emptyForm);
  const [fields, setFields] = useState<Record<string, string>>({});
  const [busy, setBusy] = useState(false);

  // Admin invite form state
  const [showInviteForm, setShowInviteForm] = useState(false);
  const [inviteForm, setInviteForm] = useState({
    first_name: "",
    last_name: "",
    email: "",
  });
  const [inviteFields, setInviteFields] = useState<Record<string, string>>({});
  const [inviteBusy, setInviteBusy] = useState(false);

  function load() {
    elections
      .list()
      .then(setItems)
      .catch((err) => {
        const message = err instanceof ApiError ? err.message : "Elections could not be loaded.";
        setLoadError(message);
        toast.error(message);
      });
  }

  useEffect(load, [toast.error]);

  // Quick action from the sidebar: /staff?new=1 opens the create form.
  useEffect(() => {
    if (new URLSearchParams(window.location.search).get("new") === "1") setShowForm(true);
  }, []);

  async function create(event: React.FormEvent) {
    event.preventDefault();
    setBusy(true);
    setFields({});
    try {
      const created = await elections.create({
        ...form,
        starts_at: toISO(form.starts_at),
        ends_at: toISO(form.ends_at),
      });
      router.push(`/staff/elections/${created.id}`);
    } catch (err) {
      if (err instanceof ApiError) {
        toast.error(err.message);
        setFields(err.fields ?? {});
      } else {
        toast.error("The election could not be created.");
      }
    } finally {
      setBusy(false);
    }
  }

  async function refresh() {
    load();
    toast.success("Reloaded from the server.");
  }

  async function inviteAdmin(event: React.FormEvent) {
    event.preventDefault();
    setInviteBusy(true);
    setInviteFields({});
    try {
      await auth.inviteAdmin(inviteForm);
      toast.success(
        "Invitation sent. The administrator will choose a password from the emailed link."
      );
      setShowInviteForm(false);
      setInviteForm({ first_name: "", last_name: "", email: "" });
    } catch (err) {
      if (err instanceof ApiError) {
        toast.error(err.message);
        setInviteFields(err.fields ?? {});
      } else {
        toast.error("The invitation could not be sent.");
      }
    } finally {
      setInviteBusy(false);
    }
  }

  return (
    <StaffShell>
      <div className="page-head" style={{ marginBottom: 20 }}>
        <div>
          <span className="lbl">Officials</span>
          <h1>Elections</h1>
          <p>
            Create an election with a title and its voting window, then add positions, candidates
            and eligibility. Drafts stay private to you.
          </p>
        </div>
        <div className="row">
          <button type="button" className="btn" onClick={refresh}>
            <Icon name="hist" size="s" />
            Refresh
          </button>
          <button type="button" className="btn" onClick={() => setShowInviteForm((open) => !open)}>
            <Icon name="user" size="s" />
            Invite admin
          </button>
          <button
            type="button"
            className="btn primary"
            onClick={() => setShowForm((open) => !open)}
          >
            <Icon name="plus" />
            New election
          </button>
        </div>
      </div>

      {showForm ? (
        <form className="panel stack" onSubmit={create} style={{ marginBottom: 24 }}>
          <div className="section-h" style={{ marginTop: 0 }}>
            <h2>Create a draft election</h2>
            <span className="mono">starts as DRAFT</span>
          </div>
          <div className="form-grid">
            <div className="field full">
              <label htmlFor="title">Election title *</label>
              <input
                id="title"
                className={`in${fields.title ? " bad" : ""}`}
                value={form.title}
                onChange={(event) => setForm({ ...form, title: event.target.value })}
                placeholder="Guild Elections 2026/27"
                required
              />
              {fields.title ? <span className="err">{fields.title}</span> : null}
            </div>
            <div className="field">
              <label htmlFor="starts">Voting opens (EAT)</label>
              <input
                id="starts"
                className={`in${fields.starts_at ? " bad" : ""}`}
                type="datetime-local"
                value={form.starts_at}
                onChange={(event) => setForm({ ...form, starts_at: event.target.value })}
              />
              {fields.starts_at ? <span className="err">{fields.starts_at}</span> : null}
            </div>
            <div className="field">
              <label htmlFor="ends">Voting closes (EAT)</label>
              <input
                id="ends"
                className={`in${fields.ends_at ? " bad" : ""}`}
                type="datetime-local"
                value={form.ends_at}
                onChange={(event) => setForm({ ...form, ends_at: event.target.value })}
              />
              {fields.ends_at ? <span className="err">{fields.ends_at}</span> : null}
            </div>
            <div className="field full">
              <span className="lbl">Quick length from the opening time</span>
              <div className="row" style={{ flexWrap: "wrap", gap: 8 }}>
                {durationPresets.map((preset) => (
                  <button
                    key={preset.label}
                    type="button"
                    className="btn sm"
                    disabled={!form.starts_at}
                    onClick={() =>
                      setForm({
                        ...form,
                        ends_at: addMinutes(form.starts_at, preset.minutes),
                      })
                    }
                  >
                    {preset.label}
                  </button>
                ))}
              </div>
              <span className="hint">
                Set the opening time first, then tap a length. Any custom length is allowed.
              </span>
            </div>
          </div>
          <div className="row">
            <button className="btn primary" type="submit" disabled={busy}>
              {busy ? "Creating…" : "Create draft"}
            </button>
            <button className="btn ghost" type="button" onClick={() => setShowForm(false)}>
              Cancel
            </button>
          </div>
        </form>
      ) : null}

      {showInviteForm ? (
        <form className="panel stack" onSubmit={inviteAdmin} style={{ marginBottom: 24 }}>
          <div className="section-h" style={{ marginTop: 0 }}>
            <h2>Invite an administrator</h2>
            <span className="mono">must use @must.ac.ug domain</span>
          </div>
          <div className="form-grid">
            <div className="field">
              <label htmlFor="invite-first-name">First name *</label>
              <input
                id="invite-first-name"
                className={`in${inviteFields.first_name ? " bad" : ""}`}
                value={inviteForm.first_name}
                onChange={(event) =>
                  setInviteForm({
                    ...inviteForm,
                    first_name: event.target.value,
                  })
                }
                placeholder="Jane"
                required
                minLength={2}
              />
              {inviteFields.first_name ? (
                <span className="err">{inviteFields.first_name}</span>
              ) : null}
            </div>
            <div className="field">
              <label htmlFor="invite-last-name">Last name *</label>
              <input
                id="invite-last-name"
                className={`in${inviteFields.last_name ? " bad" : ""}`}
                value={inviteForm.last_name}
                onChange={(event) =>
                  setInviteForm({
                    ...inviteForm,
                    last_name: event.target.value,
                  })
                }
                placeholder="Doe"
                required
                minLength={2}
              />
              {inviteFields.last_name ? (
                <span className="err">{inviteFields.last_name}</span>
              ) : null}
            </div>
            <div className="field full">
              <label htmlFor="invite-email">Email (must be @must.ac.ug) *</label>
              <input
                id="invite-email"
                className={`in${inviteFields.email ? " bad" : ""}`}
                type="email"
                value={inviteForm.email}
                onChange={(event) => setInviteForm({ ...inviteForm, email: event.target.value })}
                placeholder="jane.doe@must.ac.ug"
                required
              />
              {inviteFields.email ? <span className="err">{inviteFields.email}</span> : null}
              <span className="hint">
                The invited person will receive an email with a link to set their password.
              </span>
            </div>
          </div>
          <div className="row">
            <button className="btn primary" type="submit" disabled={inviteBusy}>
              {inviteBusy ? "Sending…" : "Send invitation"}
            </button>
            <button className="btn ghost" type="button" onClick={() => setShowInviteForm(false)}>
              Cancel
            </button>
          </div>
        </form>
      ) : null}

      {loadError ? (
        <LoadError title="We could not load the elections" message={loadError} />
      ) : !items ? (
        <div className="panel stack" aria-busy="true">
          <span className="skel" style={{ height: 14 }} />
          <span className="skel" style={{ height: 14, width: "80%" }} />
        </div>
      ) : null}

      {items ? (
        <div className="tbl-wrap panel pad0">
          <table className="tbl">
            <thead>
              <tr>
                <th>Election</th>
                <th>Category</th>
                <th>Status</th>
                <th>Schedule</th>
                <th>Version</th>
                <th aria-label="Actions" />
              </tr>
            </thead>
            <tbody>
              {items.length === 0 ? (
                <tr>
                  <td colSpan={6}>No elections yet. Create the first draft.</td>
                </tr>
              ) : (
                items.map((election) => (
                  <tr key={election.id}>
                    <td>
                      <b>{election.title}</b>
                      <span className="hint" style={{ display: "block" }}>
                        {election.organizing_unit}
                      </span>
                    </td>
                    <td>
                      <span className="tag">{categoryLabel(election.category)}</span>
                    </td>
                    <td>
                      <StatusChip status={election.status} />
                    </td>
                    <td className="mono">
                      {formatShortEAT(election.starts_at)}
                      <span className="hint" style={{ display: "block" }}>
                        to {formatShortEAT(election.ends_at)}
                      </span>
                    </td>
                    <td className="r mono">v{election.version}</td>
                    <td>
                      <div className="table-actions">
                        <Link className="btn sm" href={`/staff/elections/${election.id}`}>
                          Manage
                        </Link>
                      </div>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      ) : null}
    </StaffShell>
  );
}

export default function StaffPage() {
  return (
    <RequireAuth adminOnly>
      <StaffHome />
    </RequireAuth>
  );
}
