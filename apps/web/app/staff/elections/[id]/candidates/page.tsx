"use client";

import Link from "next/link";
import { use, useCallback, useEffect, useState } from "react";
import { ConfirmDialog } from "@/components/confirm-dialog";
import { ElectionTabs } from "@/components/election-tabs";
import { EmptyState } from "@/components/empty-state";
import { Icon } from "@/components/icon";
import { RequireAuth } from "@/components/require-auth";
import { StaffShell } from "@/components/staff-shell";
import { ApiError, candidates, positions } from "@/lib/api";
import { candidateStatusClass, candidateStatusLabel, formatEAT } from "@/lib/format";
import type { Candidate, Position } from "@/lib/types";

const filters = ["ALL", "NOMINATED", "APPROVED", "REJECTED", "WITHDRAWN"] as const;

function Review({ id }: { id: string }) {
  const [items, setItems] = useState<Candidate[] | null>(null);
  const [configuredPositions, setConfiguredPositions] = useState<Position[]>([]);
  const [filter, setFilter] = useState<(typeof filters)[number]>("ALL");
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [busy, setBusy] = useState(false);
  const [reviewing, setReviewing] = useState<Candidate | null>(null);
  const [decision, setDecision] = useState<"approve" | "reject">("approve");
  const [reason, setReason] = useState("");

  const load = useCallback(async () => {
    try {
      const [list, configured] = await Promise.all([
        candidates.list(id, filter === "ALL" ? undefined : filter),
        positions.list(id),
      ]);
      setItems(list.candidates ?? []);
      setConfiguredPositions(configured.positions ?? []);
      setError("");
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Candidates could not be loaded.");
    }
  }, [id, filter, positions.list]);

  useEffect(() => {
    load();
  }, [load]);

  function open(candidate: Candidate, next: "approve" | "reject") {
    setReviewing(candidate);
    setDecision(next);
    setReason("");
    setError("");
    setNotice("");
  }

  async function submit() {
    if (!reviewing) return;
    if (!reason.trim()) {
      setError("Add a reason before recording the decision.");
      return;
    }
    setBusy(true);
    setError("");
    try {
      if (decision === "approve") {
        await candidates.approve(id, reviewing.id, reason.trim());
        setNotice(`${reviewing.full_name} was approved.`);
      } else {
        await candidates.reject(id, reviewing.id, reason.trim());
        setNotice(`${reviewing.full_name} was rejected.`);
      }
      setReviewing(null);
      await load();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "The decision was refused.");
    } finally {
      setBusy(false);
    }
  }

  const positionTitle = (positionId: string) =>
    configuredPositions.find((position) => position.id === positionId)?.title ?? "Position";

  return (
    <StaffShell title="Candidates">
      <div className="page-head">
        <div>
          <span className="lbl">Nominations</span>
          <h1>Candidates</h1>
          <p>Review applications, check documents and record an approval or rejection.</p>
        </div>
        <Link className="btn" href={`/staff/elections/${id}`}>
          Back to election
        </Link>
      </div>

      <ElectionTabs id={id} />

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

      <div className="row" style={{ marginBottom: 16 }}>
        {filters.map((value) => (
          <button
            key={value}
            type="button"
            className={`btn sm${filter === value ? " primary" : ""}`}
            aria-pressed={filter === value}
            onClick={() => setFilter(value)}
          >
            {value === "ALL" ? "All" : candidateStatusLabel(value)}
          </button>
        ))}
      </div>

      {!items && !error ? (
        <div className="panel stack" aria-busy="true">
          <span className="skel" style={{ height: 18, width: "30%" }} />
          <span className="skel" style={{ height: 14 }} />
        </div>
      ) : null}

      {items && items.length === 0 ? (
        <EmptyState title="No applications in this view">
          Applications appear here as soon as a student applies or an official nominates somebody.
        </EmptyState>
      ) : null}

      {items && items.length > 0 ? (
        <div className="stack">
          {items.map((candidate) => (
            <article className="panel" key={candidate.id}>
              <div className="row-between" style={{ alignItems: "flex-start", gap: 16 }}>
                <div className="row" style={{ alignItems: "flex-start", gap: 16 }}>
                  {candidate.photo_url ? (
                    // eslint-disable-next-line @next/next/no-img-element
                    <img
                      src={candidate.photo_url}
                      alt=""
                      width={56}
                      height={56}
                      style={{ objectFit: "cover", border: "1px solid var(--line)" }}
                    />
                  ) : null}
                  <div>
                    <span className={`chip ${candidateStatusClass(candidate.approval_status)}`}>
                      {candidateStatusLabel(candidate.approval_status)}
                    </span>
                    <h2 style={{ marginTop: 8 }}>{candidate.full_name}</h2>
                    <p className="hint">
                      {positionTitle(candidate.position_id)} · applied{" "}
                      {formatEAT(candidate.created_at)}
                      {candidate.ballot_number ? ` · ballot #${candidate.ballot_number}` : ""}
                    </p>
                    {candidate.slogan ? <p style={{ marginTop: 6 }}>{candidate.slogan}</p> : null}
                    {candidate.review_notes ? (
                      <p className="hint">Reviewer note: {candidate.review_notes}</p>
                    ) : null}
                  </div>
                </div>
                <div className="row">
                  {candidate.manifesto_url ? (
                    <a
                      className="btn sm"
                      href={candidate.manifesto_url}
                      target="_blank"
                      rel="noreferrer"
                    >
                      <Icon name="file" size="s" />
                      Manifesto
                    </a>
                  ) : null}
                  {candidate.approval_status === "NOMINATED" ||
                  candidate.approval_status === "PENDING" ? (
                    <>
                      <button
                        type="button"
                        className="btn sm primary"
                        onClick={() => open(candidate, "approve")}
                      >
                        Approve
                      </button>
                      <button
                        type="button"
                        className="btn sm danger-o"
                        onClick={() => open(candidate, "reject")}
                      >
                        Reject
                      </button>
                    </>
                  ) : null}
                </div>
              </div>
              {candidate.bio || candidate.manifesto ? (
                <details style={{ marginTop: 12 }}>
                  <summary className="hint">Read the application</summary>
                  {candidate.bio ? <p style={{ marginTop: 8 }}>{candidate.bio}</p> : null}
                  {candidate.manifesto ? <p className="hint">{candidate.manifesto}</p> : null}
                  {candidate.contact_email ? (
                    <p className="hint">Contact: {candidate.contact_email}</p>
                  ) : null}
                  {candidate.contact_phone ? (
                    <p className="hint">Phone: {candidate.contact_phone}</p>
                  ) : null}
                </details>
              ) : null}
            </article>
          ))}
        </div>
      ) : null}

      <ConfirmDialog
        open={reviewing !== null}
        title={decision === "approve" ? "Approve this candidate?" : "Reject this candidate?"}
        confirmLabel={decision === "approve" ? "Approve" : "Reject"}
        destructive={decision === "reject"}
        busy={busy}
        onClose={() => setReviewing(null)}
        onConfirm={submit}
      >
        <p>
          {decision === "approve" ? "Approving" : "Rejecting"} <b>{reviewing?.full_name}</b> for{" "}
          <b>{reviewing ? positionTitle(reviewing.position_id) : ""}</b>. The decision is audited
          and the applicant sees your reason.
        </p>
        <div className="field">
          <label htmlFor="review-reason">Reason (required)</label>
          <textarea
            id="review-reason"
            className="ta"
            value={reason}
            onChange={(event) => setReason(event.target.value)}
            placeholder={
              decision === "approve"
                ? "Documents checked and the application meets the rules."
                : "Explain what is missing or why the application cannot be accepted."
            }
          ></textarea>
        </div>
      </ConfirmDialog>
    </StaffShell>
  );
}

export default function CandidatesPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  return (
    <RequireAuth>
      <Review id={id} />
    </RequireAuth>
  );
}
