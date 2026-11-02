"use client";

import { useEffect, useRef } from "react";
import { Icon } from "@/components/icon";
import { candidateStatusClass, candidateStatusLabel } from "@/lib/format";
import type { Candidate } from "@/lib/types";

/** Age in whole years from an ISO date of birth, or null when it is unreadable. */
function ageFrom(dateOfBirth?: string | null): number | null {
  if (!dateOfBirth) return null;
  const born = new Date(dateOfBirth);
  if (Number.isNaN(born.getTime())) return null;
  const now = new Date();
  let age = now.getUTCFullYear() - born.getUTCFullYear();
  const monthDiff = now.getUTCMonth() - born.getUTCMonth();
  if (monthDiff < 0 || (monthDiff === 0 && now.getUTCDate() < born.getUTCDate())) age -= 1;
  return age >= 0 && age < 130 ? age : null;
}

function Row({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div className="field">
      <span className="lbl">{label}</span>
      <span>{value || "—"}</span>
    </div>
  );
}

/** A document the applicant uploaded, shown as a link that opens in a new tab. */
function DocumentLink({ url, label }: { url?: string; label: string }) {
  if (!url) return <span className="candidate-meta">{label}: not provided</span>;
  return (
    <a className="btn sm" href={url} target="_blank" rel="noreferrer">
      <Icon name="file" size="s" />
      {label}
    </a>
  );
}

/**
 * The screening dialog an administrator opens from an application row. It shows
 * every detail the applicant provided and records the approve/reject decision.
 * Native <dialog> gives focus containment, Escape to close and focus return.
 */
export function ApplicationReviewDialog({
  candidate,
  index,
  total,
  reason,
  onReasonChange,
  busy,
  onPrev,
  onNext,
  onApprove,
  onReject,
  onClose,
}: {
  candidate: Candidate | null;
  index: number;
  total: number;
  reason: string;
  onReasonChange: (value: string) => void;
  busy?: boolean;
  onPrev: () => void;
  onNext: () => void;
  onApprove: () => void;
  onReject: () => void;
  onClose: () => void;
}) {
  const ref = useRef<HTMLDialogElement>(null);
  const open = candidate !== null;

  useEffect(() => {
    const dialog = ref.current;
    if (!dialog) return;
    if (open && !dialog.open) dialog.showModal();
    if (!open && dialog.open) dialog.close();
  }, [open]);

  const reviewable =
    candidate?.approval_status === "NOMINATED" || candidate?.approval_status === "PENDING";
  const age = ageFrom(candidate?.date_of_birth);

  return (
    <dialog
      ref={ref}
      className="modal-dialog modal"
      style={{ maxWidth: 680 }}
      onCancel={(event) => {
        event.preventDefault();
        if (!busy) onClose();
      }}
      onClose={onClose}
    >
      {candidate ? (
        <>
          <div className="modal-h">
            <div>
              <span className="lbl">
                Application {index + 1} of {total}
              </span>
              <h3>{candidate.full_name}</h3>
            </div>
            <button
              type="button"
              className="btn sm ghost"
              aria-label="Close"
              onClick={onClose}
              disabled={busy}
            >
              <Icon name="x" size="s" />
            </button>
          </div>

          <div className="modal-b">
            <div className="row-between" style={{ alignItems: "flex-start", gap: 16 }}>
              <div className="row" style={{ alignItems: "flex-start", gap: 12 }}>
                {candidate.passport_photo_url ? (
                  <a href={candidate.passport_photo_url} target="_blank" rel="noreferrer">
                    {/* eslint-disable-next-line @next/next/no-img-element */}
                    <img
                      src={candidate.passport_photo_url}
                      alt={`Passport of ${candidate.full_name}`}
                      width={96}
                      height={96}
                      style={{ objectFit: "cover", border: "1px solid var(--line)" }}
                    />
                  </a>
                ) : null}
                <div>
                  <span className={`chip ${candidateStatusClass(candidate.approval_status)}`}>
                    {candidateStatusLabel(candidate.approval_status)}
                  </span>
                  <span className="candidate-meta">Registration {candidate.registration_number || "—"}</span>
                  <span className="candidate-meta">
                    {candidate.programme || "—"}
                    {candidate.study_year ? ` · Year ${candidate.study_year}` : ""}
                  </span>
                </div>
              </div>
            </div>

            <dl className="kv">
              <dt>Names</dt>
              <dd>{candidate.full_name}</dd>
              <dt>Campus ID</dt>
              <dd>{candidate.registration_number || "—"}</dd>
              <dt>Programme</dt>
              <dd>{candidate.programme || "—"}</dd>
              <dt>Year of study</dt>
              <dd>{candidate.study_year || "—"}</dd>
              <dt>Date of birth</dt>
              <dd>{candidate.date_of_birth?.slice(0, 10) || "—"}</dd>
              <dt>Age</dt>
              <dd>{age ?? "—"}</dd>
              <dt>Sex</dt>
              <dd>{candidate.sex || "—"}</dd>
              <dt>Religion</dt>
              <dd>{candidate.religion || "—"}</dd>
              <dt>Residence</dt>
              <dd>{candidate.campus_residence || "—"}</dd>
              <dt>Contact</dt>
              <dd>
                {candidate.contact_email || "—"}
                {candidate.contact_phone ? ` · ${candidate.contact_phone}` : ""}
              </dd>
            </dl>

            <div className="row" style={{ flexWrap: "wrap", gap: 8 }}>
              <DocumentLink url={candidate.passport_photo_url} label="Passport photo" />
              <DocumentLink url={candidate.partial_transcript_url} label="Partial transcript" />
              {candidate.id_document_url ? (
                <DocumentLink url={candidate.id_document_url} label="Student ID" />
              ) : null}
            </div>

            {candidate.slogan ? (
              <Row label="Slogan" value={candidate.slogan} />
            ) : null}
            {candidate.bio ? (
              <div>
                <span className="lbl">Why they are standing</span>
                <p style={{ whiteSpace: "pre-wrap" }}>{candidate.bio}</p>
              </div>
            ) : null}

            {candidate.review_notes ? (
              <div>
                <span className="lbl">Reviewer note</span>
                <p className="hint">{candidate.review_notes}</p>
              </div>
            ) : null}

            {reviewable ? (
              <div className="field">
                <label htmlFor="application-reason">Note or reason (optional to approve)</label>
                <textarea
                  id="application-reason"
                  className="ta"
                  rows={3}
                  value={reason}
                  onChange={(event) => onReasonChange(event.target.value)}
                  placeholder="Recorded with the decision and shown to the applicant if rejected."
                  disabled={busy}
                />
              </div>
            ) : null}

            <div className="row" style={{ justifyContent: "space-between" }}>
              <span className="row">
                <button
                  type="button"
                  className="btn sm"
                  onClick={onPrev}
                  disabled={busy || index === 0}
                >
                  Previous
                </button>
                <button
                  type="button"
                  className="btn sm"
                  onClick={onNext}
                  disabled={busy || index >= total - 1}
                >
                  Next
                </button>
              </span>
            </div>
          </div>

          <div className="modal-f">
            <button type="button" className="btn ghost" onClick={onClose} disabled={busy}>
              Close
            </button>
            {reviewable ? (
              <>
                <button
                  type="button"
                  className="btn danger-o"
                  onClick={onReject}
                  disabled={busy}
                >
                  Reject
                </button>
                <button type="button" className="btn primary" onClick={onApprove} disabled={busy}>
                  {busy ? "Working…" : "Approve"}
                </button>
              </>
            ) : null}
          </div>
        </>
      ) : null}
    </dialog>
  );
}
