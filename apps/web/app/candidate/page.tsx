"use client";

import Link from "next/link";
import { useCallback, useEffect, useState } from "react";
import { ConfirmDialog } from "@/components/confirm-dialog";
import { EmptyState } from "@/components/empty-state";
import { Icon } from "@/components/icon";
import { RequireAuth } from "@/components/require-auth";
import { SiteHeader } from "@/components/site-header";
import { ApiError, candidates } from "@/lib/api";
import { candidateStatusClass, candidateStatusLabel, formatEAT } from "@/lib/format";
import type { MyCandidate } from "@/lib/types";

function MyCandidacy() {
  const [items, setItems] = useState<MyCandidate[] | null>(null);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [busy, setBusy] = useState(false);
  const [withdrawing, setWithdrawing] = useState<MyCandidate | null>(null);
  const [reason, setReason] = useState("");

  const load = useCallback(async () => {
    try {
      const data = await candidates.mine();
      setItems(data.candidates ?? []);
      setError("");
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Your applications could not be loaded.");
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  async function withdraw() {
    if (!withdrawing) return;
    if (!reason.trim()) {
      setError("Add a reason before withdrawing.");
      return;
    }
    setBusy(true);
    setError("");
    try {
      await candidates.withdraw(withdrawing.candidate.id, reason.trim());
      setNotice("Your application has been withdrawn.");
      setWithdrawing(null);
      setReason("");
      await load();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "The withdrawal was refused.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <>
      <SiteHeader />
      <main className="app-wrap">
        <div className="page-head">
          <div>
            <span className="lbl">Candidate</span>
            <h1>My candidacy</h1>
            <p>Track every application and read the reviewer&apos;s decision.</p>
          </div>
          <Link className="btn" href="/elections">
            <Icon name="ballot" size="s" />
            Find an election
          </Link>
        </div>

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

        {!items && !error ? (
          <div className="panel stack" aria-busy="true">
            <span className="skel" style={{ height: 18, width: "35%" }} />
            <span className="skel" style={{ height: 14 }} />
          </div>
        ) : null}

        {items && items.length === 0 ? (
          <EmptyState
            title="You have not applied yet"
            action={
              <Link className="btn primary" href="/elections">
                Browse elections
              </Link>
            }
          >
            Open an election that has not started voting, then choose &quot;Become a
            candidate&quot;.
          </EmptyState>
        ) : null}

        {items && items.length > 0 ? (
          <div className="stack">
            {items.map(({ candidate, election_title, position_title }) => {
              const canWithdraw =
                candidate.approval_status !== "WITHDRAWN" &&
                candidate.approval_status !== "REJECTED";
              return (
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
                        <h2 style={{ marginTop: 8 }}>{election_title || "Election"}</h2>
                        <p className="hint">
                          {position_title || "Position"} · applied {formatEAT(candidate.created_at)}
                        </p>
                        {candidate.slogan ? (
                          <p style={{ marginTop: 6 }}>{candidate.slogan}</p>
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
                      {canWithdraw ? (
                        <button
                          type="button"
                          className="btn sm danger-o"
                          onClick={() => {
                            setWithdrawing({ candidate, election_title, position_title });
                            setReason("");
                            setError("");
                          }}
                        >
                          Withdraw
                        </button>
                      ) : null}
                    </div>
                  </div>
                  {candidate.review_notes ? (
                    <div className="banner" style={{ marginTop: 14 }}>
                      <Icon name="info" size="s" />
                      <div>
                        <b>Reviewer note</b>
                        {candidate.review_notes}
                      </div>
                    </div>
                  ) : null}
                </article>
              );
            })}
          </div>
        ) : null}
      </main>

      <ConfirmDialog
        open={withdrawing !== null}
        title="Withdraw this application?"
        confirmLabel="Withdraw"
        destructive
        busy={busy}
        onClose={() => setWithdrawing(null)}
        onConfirm={withdraw}
      >
        <p>
          You are withdrawing your application for{" "}
          <b>{withdrawing?.position_title || "this position"}</b>. Officials will see the reason.
        </p>
        <div className="field">
          <label htmlFor="withdraw-reason">Reason</label>
          <textarea
            id="withdraw-reason"
            className="ta"
            value={reason}
            onChange={(event) => setReason(event.target.value)}
            placeholder="Why are you withdrawing?"
          />
        </div>
      </ConfirmDialog>
    </>
  );
}

export default function MyCandidacyPage() {
  return (
    <RequireAuth>
      <MyCandidacy />
    </RequireAuth>
  );
}
