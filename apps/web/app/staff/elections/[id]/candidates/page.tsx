"use client";

import Link from "next/link";
import { use, useCallback, useEffect, useState } from "react";
import { ConfirmDialog } from "@/components/confirm-dialog";
import { EmptyState } from "@/components/empty-state";
import { Icon } from "@/components/icon";
import { LoadError } from "@/components/load-error";
import { RequireAuth } from "@/components/require-auth";
import { StaffShell } from "@/components/staff-shell";
import { useToast } from "@/components/toast";
import { ApiError, candidates, fileUrl, positions } from "@/lib/api";
import { candidateStatusClass, candidateStatusLabel, formatEAT } from "@/lib/format";
import type { Candidate, Position } from "@/lib/types";

const filters = ["ALL", "NOMINATED", "APPROVED", "REJECTED", "WITHDRAWN"] as const;

function Review({ id }: { id: string }) {
  const toast = useToast();
  const [loadError, setLoadError] = useState<string | null>(null);
  const [items, setItems] = useState<Candidate[] | null>(null);
  const [configuredPositions, setConfiguredPositions] = useState<Position[]>([]);
  const [filter, setFilter] = useState<(typeof filters)[number]>("ALL");
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
    } catch (err) {
      const message = err instanceof ApiError ? err.message : "Candidates could not be loaded.";
      setLoadError(message);
      toast.error(message);
    }
  }, [id, filter, toast]);

  useEffect(() => {
    load();
  }, [load]);

  function open(candidate: Candidate, next: "approve" | "reject") {
    setReviewing(candidate);
    setDecision(next);
    setReason("");
  }

  async function submit() {
    if (!reviewing) return;
    if (!reason.trim()) {
      toast.error("Add a reason before recording the decision.");
      return;
    }
    setBusy(true);
    try {
      if (decision === "approve") {
        await candidates.approve(id, reviewing.id, reason.trim());
        toast.success(`${reviewing.full_name} was approved.`);
      } else {
        await candidates.reject(id, reviewing.id, reason.trim());
        toast.success(`${reviewing.full_name} was rejected.`);
      }
      setReviewing(null);
      await load();
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "The decision was refused.");
    } finally {
      setBusy(false);
    }
  }

  const positionFor = (positionId: string) =>
    configuredPositions.find((position) => position.id === positionId);
  const positionTitle = (positionId: string) => positionFor(positionId)?.title ?? "Position";

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

      {loadError ? (
        <LoadError
          title="We could not load the candidates"
          message={loadError}
          backHref={`/staff/elections/${id}`}
          backLabel="Back to the election"
        />
      ) : !items ? (
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
                      src={fileUrl(candidate.photo_url)}
                      alt=""
                      width={56}
                      height={56}
                      style={{
                        objectFit: "cover",
                        border: "1px solid var(--line)",
                      }}
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
                      href={fileUrl(candidate.manifesto_url)}
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
              <div className="form-grid" style={{ marginTop: 16 }}>
                <div className="field">
                  <span className="lbl">Registration number</span>
                  <span>{candidate.registration_number || "—"}</span>
                </div>
                <div className="field">
                  <span className="lbl">Programme / year</span>
                  <span>
                    {candidate.programme || "—"}
                    {candidate.study_year ? ` · Year ${candidate.study_year}` : ""}
                  </span>
                </div>
                <div className="field">
                  <span className="lbl">Date of birth</span>
                  <span>{candidate.date_of_birth?.slice(0, 10) || "—"}</span>
                </div>
                <div className="field">
                  <span className="lbl">Sex</span>
                  <span>{candidate.sex || "—"}</span>
                </div>
                <div className="field">
                  <span className="lbl">Religion</span>
                  <span>{candidate.religion || "—"}</span>
                </div>
                <div className="field">
                  <span className="lbl">Position requirements</span>
                  <span style={{ whiteSpace: "pre-wrap" }}>
                    {positionFor(candidate.position_id)?.requirements || "—"}
                  </span>
                </div>
              </div>

              <div className="row" style={{ marginTop: 12, flexWrap: "wrap", gap: 8 }}>
                {candidate.partial_transcript_url ? (
                  <a
                    className="btn sm"
                    href={fileUrl(candidate.partial_transcript_url)}
                    target="_blank"
                    rel="noreferrer"
                  >
                    <Icon name="file" size="s" />
                    Partial transcript (PDF)
                  </a>
                ) : null}
                {candidate.ballot_photo_url ? (
                  <a
                    className="btn sm"
                    href={fileUrl(candidate.ballot_photo_url)}
                    target="_blank"
                    rel="noreferrer"
                  >
                    <Icon name="eye" size="s" />
                    Ballot photo
                  </a>
                ) : null}
              </div>

              {candidate.passport_photo_url || candidate.id_document_url ? (
                <div className="row" style={{ marginTop: 12, alignItems: "flex-start", gap: 16 }}>
                  {candidate.passport_photo_url ? (
                    <a
                      href={fileUrl(candidate.passport_photo_url)}
                      target="_blank"
                      rel="noreferrer"
                      className="stack"
                      style={{ gap: 4 }}
                    >
                      {/* eslint-disable-next-line @next/next/no-img-element */}
                      <img
                        src={fileUrl(candidate.passport_photo_url)}
                        alt={`${candidate.full_name}'s passport`}
                        width={112}
                        height={112}
                        style={{
                          objectFit: "cover",
                          border: "1px solid var(--line)",
                        }}
                      />
                      <span className="hint">Passport photo</span>
                    </a>
                  ) : null}
                  {candidate.id_document_url ? (
                    <a
                      href={fileUrl(candidate.id_document_url)}
                      target="_blank"
                      rel="noreferrer"
                      className="stack"
                      style={{ gap: 4 }}
                    >
                      {/* eslint-disable-next-line @next/next/no-img-element */}
                      <img
                        src={fileUrl(candidate.id_document_url)}
                        alt="Student ID"
                        width={112}
                        height={112}
                        style={{
                          objectFit: "cover",
                          border: "1px solid var(--line)",
                        }}
                      />
                      <span className="hint">Student ID</span>
                    </a>
                  ) : null}
                </div>
              ) : null}

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
    <RequireAuth adminOnly>
      <Review id={id} />
    </RequireAuth>
  );
}
