"use client";

import Link from "next/link";
import { use, useCallback, useEffect, useState } from "react";
import { ConfirmDialog } from "@/components/confirm-dialog";
import { ElectionTabs } from "@/components/election-tabs";
import { EmptyState } from "@/components/empty-state";
import { Icon } from "@/components/icon";
import { RequireAuth } from "@/components/require-auth";
import { StaffShell } from "@/components/staff-shell";
import { ApiError, elections, positions } from "@/lib/api";
import type { Election, Position } from "@/lib/types";

type Form = {
  title: string;
  description: string;
  requirements: string;
  question_type: string;
  seats: number;
  min_selections: number;
  max_selections: number;
  allow_blank: boolean;
  display_order: number;
};

const emptyForm: Form = {
  title: "",
  description: "",
  requirements: "",
  question_type: "SINGLE_CHOICE",
  seats: 1,
  min_selections: 0,
  max_selections: 1,
  allow_blank: true,
  display_order: 0,
};

function Manage({ id }: { id: string }) {
  const [items, setItems] = useState<Position[] | null>(null);
  const [election, setElection] = useState<Election | null>(null);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [busy, setBusy] = useState(false);
  const [editing, setEditing] = useState<Position | "new" | null>(null);
  const [form, setForm] = useState<Form>(emptyForm);
  const [deleting, setDeleting] = useState<Position | null>(null);

  const load = useCallback(async () => {
    try {
      const [list, current] = await Promise.all([positions.list(id), elections.get(id)]);
      setItems(list.positions ?? []);
      setElection(current);
      setError("");
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Positions could not be loaded.");
    }
  }, [id]);

  useEffect(() => {
    load();
  }, [load]);

  const editable = election?.status === "DRAFT";

  function startCreate() {
    setEditing("new");
    setForm({ ...emptyForm, display_order: items?.length ?? 0 });
    setError("");
    setNotice("");
  }

  function startEdit(position: Position) {
    setEditing(position);
    setForm({
      title: position.title,
      description: position.description,
      requirements: position.requirements ?? "",
      question_type: position.question_type,
      seats: position.seats,
      min_selections: position.min_selections,
      max_selections: position.max_selections,
      allow_blank: position.allow_blank,
      display_order: position.display_order,
    });
    setError("");
    setNotice("");
  }

  async function save(event: React.FormEvent) {
    event.preventDefault();
    if (!form.title.trim()) {
      setError("Give the position a title.");
      return;
    }
    setBusy(true);
    setError("");
    try {
      if (editing === "new") {
        await positions.create(id, form);
        setNotice("Position created.");
      } else if (editing) {
        await positions.update(id, editing.id, form);
        setNotice("Position saved.");
      }
      setEditing(null);
      await load();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "The position could not be saved.");
    } finally {
      setBusy(false);
    }
  }

  async function remove() {
    if (!deleting) return;
    setBusy(true);
    setError("");
    try {
      await positions.remove(id, deleting.id);
      setNotice("Position deleted.");
      setDeleting(null);
      await load();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "The position could not be deleted.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <StaffShell title="Positions">
      <div className="page-head">
        <div>
          <span className="lbl">Ballot setup</span>
          <h1>Positions</h1>
          <p>Every position becomes one contest on the ballot.</p>
        </div>
        <div className="row">
          <Link className="btn" href={`/staff/elections/${id}`}>
            Back to election
          </Link>
          <button type="button" className="btn primary" onClick={startCreate} disabled={!editable}>
            <Icon name="plus" size="s" />
            Add position
          </button>
        </div>
      </div>

      <ElectionTabs id={id} />

      {!editable && election ? (
        <div className="banner hatch" style={{ marginBottom: 16 }}>
          <Icon name="lock" />
          <div>
            <b>Positions are locked</b>
            Positions can only be edited while the election is a draft. This election is{" "}
            {election.status.replaceAll("_", " ").toLowerCase()}.
          </div>
        </div>
      ) : null}

      {error ? (
        <div className="banner bad" role="alert" style={{ marginBottom: 16 }}>
          <Icon name="alert" />
          <div>
            <b>Something needs attention</b>
            {error}
          </div>
        </div>
      ) : null}
      {notice ? (
        <div className="banner ok" role="status" style={{ marginBottom: 16 }}>
          <Icon name="check" />
          <div>{notice}</div>
        </div>
      ) : null}

      {editing ? (
        <form className="panel stack" onSubmit={save} style={{ marginBottom: 20 }}>
          <div className="section-h" style={{ marginTop: 0 }}>
            <h2>{editing === "new" ? "New position" : `Edit ${editing.title}`}</h2>
          </div>
          <div className="form-grid">
            <div className="field full">
              <label htmlFor="position-title">Title *</label>
              <input
                id="position-title"
                className="in"
                value={form.title}
                onChange={(event) => setForm({ ...form, title: event.target.value })}
                placeholder="e.g. Guild President"
                required
              />
            </div>
            <div className="field full">
              <label htmlFor="position-description">Description</label>
              <textarea
                id="position-description"
                className="ta"
                rows={3}
                value={form.description}
                onChange={(event) => setForm({ ...form, description: event.target.value })}
              />
            </div>
            <div className="field full">
              <label htmlFor="position-requirements">Requirements</label>
              <textarea
                id="position-requirements"
                className="ta"
                rows={4}
                value={form.requirements}
                onChange={(event) => setForm({ ...form, requirements: event.target.value })}
                placeholder="Eligibility, experience or documents applicants must have. Shown before they apply."
              />
            </div>
            <div className="field">
              <label htmlFor="position-type">Question type</label>
              <select
                id="position-type"
                className="sel"
                value={form.question_type}
                onChange={(event) => setForm({ ...form, question_type: event.target.value })}
              >
                <option value="SINGLE_CHOICE">Single choice</option>
                <option value="MULTI_CHOICE">Multi choice</option>
                <option value="YES_NO">Yes / No</option>
                <option value="REFERENDUM">Referendum</option>
              </select>
            </div>
            <div className="field">
              <label htmlFor="position-order">Display order</label>
              <input
                id="position-order"
                className="in"
                type="number"
                min={0}
                value={form.display_order}
                onChange={(event) =>
                  setForm({ ...form, display_order: Number(event.target.value) })
                }
              />
            </div>
            <div className="field">
              <label htmlFor="position-seats">Seats</label>
              <input
                id="position-seats"
                className="in"
                type="number"
                min={1}
                value={form.seats}
                onChange={(event) => setForm({ ...form, seats: Number(event.target.value) })}
              />
            </div>
            <div className="field">
              <label htmlFor="position-min">Minimum selections</label>
              <input
                id="position-min"
                className="in"
                type="number"
                min={0}
                value={form.min_selections}
                onChange={(event) =>
                  setForm({ ...form, min_selections: Number(event.target.value) })
                }
              />
            </div>
            <div className="field">
              <label htmlFor="position-max">Maximum selections</label>
              <input
                id="position-max"
                className="in"
                type="number"
                min={1}
                value={form.max_selections}
                onChange={(event) =>
                  setForm({ ...form, max_selections: Number(event.target.value) })
                }
              />
            </div>
            <label className="chk" style={{ alignSelf: "end" }}>
              <input
                type="checkbox"
                checked={form.allow_blank}
                onChange={(event) => setForm({ ...form, allow_blank: event.target.checked })}
              />
              <span>Allow leaving this position blank</span>
            </label>
          </div>
          <div className="row">
            <button className="btn primary" type="submit" disabled={busy}>
              {busy ? "Saving…" : "Save position"}
            </button>
            <button type="button" className="btn ghost" onClick={() => setEditing(null)}>
              Cancel
            </button>
          </div>
        </form>
      ) : null}

      {!items && !error ? (
        <div className="panel stack" aria-busy="true">
          <span className="skel" style={{ height: 18, width: "30%" }} />
          <span className="skel" style={{ height: 14 }} />
        </div>
      ) : null}

      {items && items.length === 0 ? (
        <EmptyState
          title="No positions yet"
          action={
            editable ? (
              <button type="button" className="btn primary" onClick={startCreate}>
                Add the first position
              </button>
            ) : undefined
          }
        >
          Add the positions voters will choose between, such as Guild President or Faculty
          Representative.
        </EmptyState>
      ) : null}

      {items && items.length > 0 ? (
        <div className="panel pad0">
          <table className="tbl">
            <thead>
              <tr>
                <th>Position</th>
                <th>Type</th>
                <th className="r">Seats</th>
                <th className="r">Selections</th>
                <th />
              </tr>
            </thead>
            <tbody>
              {items.map((position) => (
                <tr key={position.id}>
                  <td>
                    <b>{position.title}</b>
                    {position.description ? (
                      <span className="candidate-meta">{position.description}</span>
                    ) : null}
                    {position.requirements ? (
                      <span className="candidate-meta">
                        <b>Requirements:</b> {position.requirements}
                      </span>
                    ) : null}
                  </td>
                  <td>{position.question_type.replaceAll("_", " ").toLowerCase()}</td>
                  <td className="r">{position.seats}</td>
                  <td className="r">
                    {position.min_selections}–{position.max_selections}
                  </td>
                  <td className="r">
                    <span className="row" style={{ justifyContent: "flex-end" }}>
                      <button
                        type="button"
                        className="btn sm"
                        onClick={() => startEdit(position)}
                        disabled={!editable}
                      >
                        Edit
                      </button>
                      <button
                        type="button"
                        className="btn sm danger-o"
                        onClick={() => setDeleting(position)}
                        disabled={!editable}
                      >
                        Delete
                      </button>
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : null}

      <ConfirmDialog
        open={deleting !== null}
        title="Delete this position?"
        confirmLabel="Delete"
        destructive
        busy={busy}
        onClose={() => setDeleting(null)}
        onConfirm={remove}
      >
        <p>
          <b>{deleting?.title}</b> will be removed from the ballot. This cannot be undone.
        </p>
      </ConfirmDialog>
    </StaffShell>
  );
}

export default function PositionsPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  return (
    <RequireAuth>
      <Manage id={id} />
    </RequireAuth>
  );
}
