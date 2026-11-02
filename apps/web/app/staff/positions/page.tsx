"use client";

import Link from "next/link";
import { useCallback, useEffect, useState } from "react";
import { ConfirmDialog } from "@/components/confirm-dialog";
import { EmptyState } from "@/components/empty-state";
import { Icon } from "@/components/icon";
import { LoadError } from "@/components/load-error";
import { RequireAuth } from "@/components/require-auth";
import { StaffShell } from "@/components/staff-shell";
import { useToast } from "@/components/toast";
import { ApiError, elections, positions } from "@/lib/api";
import type { Candidate, Election, PositionApplications, PositionWithMeta } from "@/lib/types";

type Form = {
  title: string;
  description: string;
  requirements: string;
  question_type: string;
  seats: number;
  min_selections: number;
  max_selections: number;
  max_candidates: number;
  allowed_domain: string;
  allowed_years: string;
  allowed_courses: string;
  status: string;
  allow_blank: boolean;
  display_order: number;
  election_id: string;
};

// One position is one contest. The eligibility fields decide who may stand for
// it, kept separate from the election's own voter eligibility.
const emptyForm: Form = {
  title: "",
  description: "",
  requirements: "",
  question_type: "SINGLE_CHOICE",
  seats: 1,
  min_selections: 0,
  max_selections: 1,
  max_candidates: 0,
  allowed_domain: "std.mast.ac.ug",
  allowed_years: "",
  allowed_courses: "",
  status: "OPEN",
  allow_blank: true,
  display_order: 0,
  election_id: "",
};

function eligibilitySummary(position: PositionWithMeta): string {
  const parts: string[] = [];
  if (position.allowed_domain) parts.push(`@${position.allowed_domain}`);
  if (position.allowed_years) parts.push(`years ${position.allowed_years}`);
  if (position.allowed_courses) parts.push(`courses ${position.allowed_courses}`);
  return parts.length ? `Only ${parts.join(" · ")}` : "Open to any student";
}

function ManagePositions() {
  const [items, setItems] = useState<PositionWithMeta[] | null>(null);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [electionList, setElectionList] = useState<Election[]>([]);
  const [filter, setFilter] = useState<"all" | "unlinked">("all");
  const [busy, setBusy] = useState(false);

  const toast = useToast();

  const [editing, setEditing] = useState<PositionWithMeta | "new" | null>(null);
  const [form, setForm] = useState<Form>(emptyForm);
  const [deleting, setDeleting] = useState<PositionWithMeta | null>(null);

  const [reviewing, setReviewing] = useState<PositionApplications | null>(null);
  const [rejecting, setRejecting] = useState<{ candidate: Candidate; reason: string } | null>(null);
  const [linking, setLinking] = useState<PositionWithMeta | null>(null);
  const [linkTarget, setLinkTarget] = useState("");

  const load = useCallback(async () => {
    try {
      const [list, allElections] = await Promise.all([
        positions.listAll({ unlinked: filter === "unlinked" }),
        elections.list(),
      ]);
      setItems(list.positions ?? []);
      setElectionList(allElections ?? []);
    } catch (err) {
      const message = err instanceof ApiError ? err.message : "Positions could not be loaded.";
      setLoadError(message);
      toast.error(message);
    }
  }, [filter, toast.error]);

  useEffect(() => {
    load();
  }, [load]);

  useEffect(() => {
    if (new URLSearchParams(window.location.search).get("new") === "1") {
      setEditing("new");
    }
  }, []);

  function startCreate() {
    setEditing("new");
    setForm({ ...emptyForm, display_order: items?.length ?? 0 });
  }

  function startEdit(position: PositionWithMeta) {
    setEditing(position);
    setForm({
      title: position.title,
      description: position.description,
      requirements: position.requirements ?? "",
      question_type: position.question_type,
      seats: position.seats,
      min_selections: position.min_selections,
      max_selections: position.max_selections,
      max_candidates: position.max_candidates,
      allowed_domain: position.allowed_domain ?? "",
      allowed_years: position.allowed_years ?? "",
      allowed_courses: position.allowed_courses ?? "",
      status: position.status,
      allow_blank: position.allow_blank,
      display_order: position.display_order,
      election_id: position.election_id ?? "",
    });
  }

  async function save(event: React.FormEvent) {
    event.preventDefault();
    if (!form.title.trim()) {
      toast.error("Give the position a title.");
      return;
    }
    setBusy(true);
    try {
      if (editing === "new") {
        const created = await positions.createStandalone(form);
        if (form.election_id) await positions.link(created.id, form.election_id);
        toast.success("Position created.");
      } else if (editing) {
        await positions.updateStandalone(editing.id, form);
        // The election link is separate so moving a position keeps its
        // applications attached to it.
        if ((editing.election_id ?? "") !== form.election_id) {
          await positions.link(editing.id, form.election_id);
        }
        toast.success("Position saved.");
      }
      setEditing(null);
      await load();
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "The position could not be saved.");
    } finally {
      setBusy(false);
    }
  }

  async function remove() {
    if (!deleting) return;
    setBusy(true);
    try {
      await positions.removeStandalone(deleting.id);
      toast.success("Position deleted.");
      setDeleting(null);
      await load();
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "The position could not be deleted.");
    } finally {
      setBusy(false);
    }
  }

  async function openReview(position: { id: string }) {
    try {
      const data = await positions.applications(position.id);
      setReviewing(data);
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Applications could not be loaded.");
    }
  }

  async function approve(candidate: Candidate) {
    if (!reviewing) return;
    setBusy(true);
    try {
      await positions.approve(reviewing.position.id, candidate.id, "Approved by an administrator");
      toast.success(`${candidate.full_name} approved as a candidate.`);
      await openReview(reviewing.position);
      await load();
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "The application could not be approved.");
    } finally {
      setBusy(false);
    }
  }

  async function confirmReject() {
    if (!reviewing || !rejecting) return;
    setBusy(true);
    try {
      await positions.reject(reviewing.position.id, rejecting.candidate.id, rejecting.reason);
      toast.success("Application rejected.");
      setRejecting(null);
      await openReview(reviewing.position);
      await load();
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "The application could not be rejected.");
    } finally {
      setBusy(false);
    }
  }

  async function saveLink() {
    if (!linking) return;
    setBusy(true);
    try {
      await positions.link(linking.id, linkTarget);
      toast.success(linkTarget ? "Position linked to the election." : "Position unlinked.");
      setLinking(null);
      await load();
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "The position could not be linked.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <StaffShell title="Positions">
      <div className="page-head">
        <div>
          <span className="lbl">Step 1 — Before elections</span>
          <h1>Positions</h1>
          <p>
            Create a position, set who may stand for it, and collect applications. Link it to an
            election once the candidate list is settled.
          </p>
        </div>
        <div className="row">
          <button type="button" className="btn primary" onClick={startCreate}>
            <Icon name="plus" size="s" />
            New position
          </button>
        </div>
      </div>

      {editing ? (
        <form className="panel stack" onSubmit={save} style={{ marginBottom: 20 }}>
          <div className="section-h" style={{ marginTop: 0 }}>
            <h2>{editing === "new" ? "New position" : `Edit ${editing.title}`}</h2>
          </div>
          <div className="form-grid">
            <div className="field full">
              <label htmlFor="p-title">Title *</label>
              <input
                id="p-title"
                className="in"
                value={form.title}
                onChange={(event) => setForm({ ...form, title: event.target.value })}
                placeholder="e.g. Guild President"
                required
              />
            </div>
            <div className="field full">
              <label htmlFor="p-description">Description</label>
              <textarea
                id="p-description"
                className="ta"
                rows={3}
                value={form.description}
                onChange={(event) => setForm({ ...form, description: event.target.value })}
              />
            </div>
            <div className="field full">
              <label htmlFor="p-requirements">Requirements</label>
              <textarea
                id="p-requirements"
                className="ta"
                rows={3}
                value={form.requirements}
                onChange={(event) => setForm({ ...form, requirements: event.target.value })}
                placeholder="Anything applicants must have. Shown before they apply."
              />
            </div>
            <div className="field">
              <label htmlFor="p-type">Question type</label>
              <select
                id="p-type"
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
              <label htmlFor="p-seats">Seats</label>
              <input
                id="p-seats"
                className="in"
                type="number"
                min={1}
                value={form.seats}
                onChange={(event) => setForm({ ...form, seats: Number(event.target.value) })}
              />
            </div>
            <div className="field">
              <label htmlFor="p-min">Minimum selections</label>
              <input
                id="p-min"
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
              <label htmlFor="p-max">Maximum selections</label>
              <input
                id="p-max"
                className="in"
                type="number"
                min={1}
                value={form.max_selections}
                onChange={(event) =>
                  setForm({ ...form, max_selections: Number(event.target.value) })
                }
              />
            </div>
            <div className="field">
              <label htmlFor="p-candidates">Maximum candidates</label>
              <input
                id="p-candidates"
                className="in"
                type="number"
                min={0}
                value={form.max_candidates}
                onChange={(event) =>
                  setForm({ ...form, max_candidates: Number(event.target.value) })
                }
              />
              <span className="hint">How many applications can be approved. 0 means no limit.</span>
            </div>
          </div>

          <div className="section-h">
            <h2>Who may stand for this position</h2>
            <p>Leave years and courses blank to admit every student on the domain.</p>
          </div>
          <div className="form-grid">
            <div className="field">
              <label htmlFor="p-domain">Student domain</label>
              <input
                id="p-domain"
                className="in"
                value={form.allowed_domain}
                onChange={(event) => setForm({ ...form, allowed_domain: event.target.value })}
                placeholder="std.mast.ac.ug"
              />
            </div>
            <div className="field">
              <label htmlFor="p-years">Allowed years</label>
              <input
                id="p-years"
                className="in"
                value={form.allowed_years}
                onChange={(event) => setForm({ ...form, allowed_years: event.target.value })}
                placeholder="e.g. 2022,2023"
              />
            </div>
            <div className="field">
              <label htmlFor="p-courses">Allowed courses</label>
              <input
                id="p-courses"
                className="in"
                value={form.allowed_courses}
                onChange={(event) => setForm({ ...form, allowed_courses: event.target.value })}
                placeholder="e.g. bse,bcs"
              />
            </div>
            <div className="field">
              <label htmlFor="p-status">Applications</label>
              <select
                id="p-status"
                className="sel"
                value={form.status}
                onChange={(event) => setForm({ ...form, status: event.target.value })}
              >
                <option value="OPEN">Open — accepting applications</option>
                <option value="CLOSED">Closed — no new applications</option>
              </select>
            </div>
            <div className="field">
              <label htmlFor="p-election">Election (optional)</label>
              <select
                id="p-election"
                className="sel"
                value={form.election_id}
                onChange={(event) => setForm({ ...form, election_id: event.target.value })}
              >
                <option value="">Not linked yet</option>
                {electionList.map((election) => (
                  <option key={election.id} value={election.id}>
                    {election.title}
                  </option>
                ))}
              </select>
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

      {linking ? (
        <div className="panel stack" style={{ marginBottom: 20 }}>
          <div className="section-h" style={{ marginTop: 0 }}>
            <h2>Link {linking.title} to an election</h2>
            <p>Applications already received follow the position into the election.</p>
          </div>
          <div className="field">
            <label htmlFor="link-election">Election</label>
            <select
              id="link-election"
              className="sel"
              value={linkTarget}
              onChange={(event) => setLinkTarget(event.target.value)}
            >
              <option value="">Not linked</option>
              {electionList.map((election) => (
                <option key={election.id} value={election.id}>
                  {election.title}
                </option>
              ))}
            </select>
          </div>
          <div className="row">
            <button type="button" className="btn primary" onClick={saveLink} disabled={busy}>
              {busy ? "Saving…" : "Save link"}
            </button>
            <button type="button" className="btn ghost" onClick={() => setLinking(null)}>
              Cancel
            </button>
          </div>
        </div>
      ) : null}

      {reviewing ? (
        <div className="panel stack" style={{ marginBottom: 20 }}>
          <div className="section-h" style={{ marginTop: 0 }}>
            <h2>Applications for {reviewing.position.title}</h2>
            <p>
              {reviewing.approved_count} approved
              {reviewing.position.max_candidates > 0
                ? ` of ${reviewing.position.max_candidates} allowed`
                : ""}
              .
            </p>
          </div>
          {reviewing.applications.length === 0 ? (
            <EmptyState title="No applications yet">
              Students who apply will appear here for review.
            </EmptyState>
          ) : (
            <table className="tbl">
              <thead>
                <tr>
                  <th>Applicant</th>
                  <th>Status</th>
                  <th />
                </tr>
              </thead>
              <tbody>
                {reviewing.applications.map((candidate) => (
                  <tr key={candidate.id}>
                    <td>
                      <b>{candidate.full_name}</b>
                      {candidate.programme || candidate.study_year ? (
                        <span className="candidate-meta">
                          {candidate.programme}{" "}
                          {candidate.study_year ? `· year ${candidate.study_year}` : ""}
                        </span>
                      ) : null}
                    </td>
                    <td>
                      <span
                        className={`chip ${candidate.approval_status === "APPROVED" ? "open" : candidate.approval_status === "REJECTED" ? "cancel" : "draft"}`}
                      >
                        {candidate.approval_status.replaceAll("_", " ").toLowerCase()}
                      </span>
                    </td>
                    <td className="r">
                      {candidate.approval_status === "NOMINATED" ||
                      candidate.approval_status === "PENDING" ? (
                        <span className="row" style={{ justifyContent: "flex-end" }}>
                          <button
                            type="button"
                            className="btn sm"
                            onClick={() => approve(candidate)}
                            disabled={busy}
                          >
                            Approve
                          </button>
                          <button
                            type="button"
                            className="btn sm danger-o"
                            onClick={() => setRejecting({ candidate, reason: "" })}
                            disabled={busy}
                          >
                            Reject
                          </button>
                        </span>
                      ) : null}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
          <div className="row">
            <button type="button" className="btn ghost" onClick={() => setReviewing(null)}>
              Close
            </button>
          </div>
        </div>
      ) : null}

      <div className="row" style={{ marginBottom: 12 }}>
        <button
          type="button"
          className={`btn sm ${filter === "all" ? "primary" : "ghost"}`}
          onClick={() => setFilter("all")}
        >
          All positions
        </button>
        <button
          type="button"
          className={`btn sm ${filter === "unlinked" ? "primary" : "ghost"}`}
          onClick={() => setFilter("unlinked")}
        >
          Not linked to an election
        </button>
      </div>

      {loadError ? (
        <LoadError
          title="We could not load the positions"
          message={loadError}
          backHref="/staff"
          backLabel="Back to the dashboard"
        />
      ) : !items ? (
        <div className="panel stack" aria-busy="true">
          <span className="skel" style={{ height: 18, width: "30%" }} />
          <span className="skel" style={{ height: 14 }} />
        </div>
      ) : null}

      {items && items.length === 0 ? (
        <EmptyState
          title="No positions yet"
          action={
            <button type="button" className="btn primary" onClick={startCreate}>
              Create the first position
            </button>
          }
        >
          Start with a position such as Guild President, then collect applications before scheduling
          the election.
        </EmptyState>
      ) : null}

      {items && items.length > 0 ? (
        <div className="panel pad0">
          <table className="tbl">
            <thead>
              <tr>
                <th>Position</th>
                <th className="r">Applications</th>
                <th>Election</th>
                <th>Applications</th>
                <th />
              </tr>
            </thead>
            <tbody>
              {items.map((position) => (
                <tr key={position.id}>
                  <td>
                    <b>{position.title}</b>
                    <span className="candidate-meta">{eligibilitySummary(position)}</span>
                    {position.max_candidates > 0 ? (
                      <span className="candidate-meta">
                        Up to {position.max_candidates} candidates
                      </span>
                    ) : null}
                  </td>
                  <td className="r">
                    {position.approved_count} / {position.application_count}
                  </td>
                  <td>
                    {position.election_id ? (
                      <Link href={`/staff/elections/${position.election_id}/positions`}>
                        {position.election_title || "View election"}
                      </Link>
                    ) : (
                      <span className="candidate-meta">Not linked</span>
                    )}
                  </td>
                  <td>
                    <span className={`chip ${position.status === "OPEN" ? "open" : "closed"}`}>
                      {position.status === "OPEN" ? "open" : "closed"}
                    </span>
                  </td>
                  <td className="r">
                    <span className="row" style={{ justifyContent: "flex-end" }}>
                      <button type="button" className="btn sm" onClick={() => openReview(position)}>
                        Applications
                      </button>
                      <button
                        type="button"
                        className="btn sm"
                        onClick={() => {
                          setLinking(position);
                          setLinkTarget(position.election_id ?? "");
                        }}
                      >
                        Link
                      </button>
                      <button type="button" className="btn sm" onClick={() => startEdit(position)}>
                        Edit
                      </button>
                      <button
                        type="button"
                        className="btn sm danger-o"
                        onClick={() => setDeleting(position)}
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
          <b>{deleting?.title}</b> and its applications will be removed. This cannot be undone.
        </p>
      </ConfirmDialog>

      <ConfirmDialog
        open={rejecting !== null}
        title="Reject this application?"
        confirmLabel="Reject"
        destructive
        busy={busy}
        onClose={() => setRejecting(null)}
        onConfirm={confirmReject}
      >
        <p>
          Give <b>{rejecting?.candidate.full_name}</b> a reason.
        </p>
        <div className="field">
          <label htmlFor="reject-reason">Reason</label>
          <textarea
            id="reject-reason"
            className="ta"
            rows={3}
            value={rejecting?.reason ?? ""}
            onChange={(event) =>
              setRejecting((current) =>
                current ? { ...current, reason: event.target.value } : current
              )
            }
          />
        </div>
      </ConfirmDialog>
    </StaffShell>
  );
}

export default function StaffPositionsPage() {
  return (
    <RequireAuth adminOnly>
      <ManagePositions />
    </RequireAuth>
  );
}
