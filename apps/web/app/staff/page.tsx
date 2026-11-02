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
import { ApiError, elections } from "@/lib/api";
import { ELECTION_AUTHORITY } from "@/lib/config";
import { categoryLabel, formatShortEAT } from "@/lib/format";
import type { Election } from "@/lib/types";

const emptyForm = {
  title: "",
  description: "",
  category: "GUILD",
  organizing_unit: "",
  starts_at: "",
  ends_at: "",
  result_policy: "AFTER_CLOSE",
};

/** datetime-local is rendered as EAT; the API stores UTC. */
function toISO(value: string): string {
  return value ? `${value}:00+03:00` : "";
}

function StaffHome() {
  const router = useRouter();
  const toast = useToast();
  const [items, setItems] = useState<Election[] | null>(null);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [showForm, setShowForm] = useState(false);
  const [form, setForm] = useState(emptyForm);
  const [fields, setFields] = useState<Record<string, string>>({});
  const [busy, setBusy] = useState(false);

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

  return (
    <StaffShell>
      <div className="page-head" style={{ marginBottom: 20 }}>
        <div>
          <span className="lbl">Officials</span>
          <h1>Elections</h1>
          <p>
            Create an election quickly with minimal info, then add details later. Drafts stay
            private to you.
          </p>
        </div>
        <div className="row">
          <button type="button" className="btn" onClick={refresh}>
            <Icon name="hist" size="s" />
            Refresh
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
            <div className="field full">
              <label htmlFor="description">Description (optional)</label>
              <textarea
                id="description"
                className="ta"
                value={form.description}
                onChange={(event) => setForm({ ...form, description: event.target.value })}
                placeholder="What voters are deciding."
              />
            </div>
            <div className="field">
              <label htmlFor="category">Category</label>
              <select
                id="category"
                className="sel"
                value={form.category}
                onChange={(event) => setForm({ ...form, category: event.target.value })}
              >
                {[
                  "GUILD",
                  "FACULTY",
                  "DEPARTMENT",
                  "PROGRAMME",
                  "CLASS",
                  "CLUB",
                  "ASSOCIATION",
                  "COMMITTEE",
                  "REFERENDUM",
                ].map((value) => (
                  <option key={value} value={value}>
                    {categoryLabel(value)}
                  </option>
                ))}
              </select>
            </div>
            <div className="field">
              <label htmlFor="unit">Organising unit (optional)</label>
              <input
                id="unit"
                className="in"
                value={form.organizing_unit}
                onChange={(event) => setForm({ ...form, organizing_unit: event.target.value })}
                placeholder={ELECTION_AUTHORITY}
              />
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
            <div className="field">
              <label htmlFor="policy">Results policy</label>
              <select
                id="policy"
                className="sel"
                value={form.result_policy}
                onChange={(event) => setForm({ ...form, result_policy: event.target.value })}
              >
                <option value="AFTER_CLOSE">After close</option>
                <option value="LIVE">Live provisional</option>
                <option value="CERTIFIED_ONLY">Certified only</option>
              </select>
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
