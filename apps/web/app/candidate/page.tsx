"use client";

import Link from "next/link";
import { useCallback, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { AppShell } from "@/components/app-shell";
import { ConfirmDialog } from "@/components/confirm-dialog";
import { EmptyState } from "@/components/empty-state";
import { Icon } from "@/components/icon";
import { LoadError } from "@/components/load-error";
import { RequireAuth } from "@/components/require-auth";
import { useAuth } from "@/components/auth-provider";
import { useToast } from "@/components/toast";
import { ApiError, candidates, fileUrl, upload } from "@/lib/api";
import { candidateStatusClass, candidateStatusLabel, formatEAT } from "@/lib/format";
import type { MyCandidate } from "@/lib/types";

function MyCandidacy() {
  const { user } = useAuth();
  const router = useRouter();

  // Administrators do not have candidacies; redirect to staff dashboard
  useEffect(() => {
    if (user?.role === "ADMIN") {
      router.replace("/staff");
    }
  }, [user, router]);
  const [items, setItems] = useState<MyCandidate[] | null>(null);
  const [busy, setBusy] = useState(false);
  const [loadError, setLoadError] = useState<string | null>(null);

  const toast = useToast();
  const [withdrawing, setWithdrawing] = useState<MyCandidate | null>(null);
  const [reason, setReason] = useState("");
  const [photoBusy, setPhotoBusy] = useState("");

  const load = useCallback(async () => {
    try {
      const data = await candidates.mine();
      setItems(data.candidates ?? []);
    } catch (err) {
      const message =
        err instanceof ApiError ? err.message : "Your applications could not be loaded.";
      setLoadError(message);
      toast.error(message);
    }
  }, [toast.error]);

  useEffect(() => {
    load();
  }, [load]);

  async function withdraw() {
    if (!withdrawing) return;
    if (!reason.trim()) {
      toast.error("Add a reason before withdrawing.");
      return;
    }
    setBusy(true);
    try {
      await candidates.withdraw(withdrawing.candidate.id, reason.trim());
      toast.success("Your application has been withdrawn.");
      setWithdrawing(null);
      setReason("");
      await load();
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "The withdrawal was refused.");
    } finally {
      setBusy(false);
    }
  }

  async function changeBallotPhoto(candidateId: string, file: File | undefined) {
    if (!file) return;
    setPhotoBusy(candidateId);
    try {
      const ref = await upload(file, {
        accepts: "image",
        maxSize: 2 * 1024 * 1024,
      });
      await candidates.setBallotPhoto(candidateId, ref.url);
      toast.success("Ballot photo updated.");
      await load();
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "The ballot photo could not be updated.");
    } finally {
      setPhotoBusy("");
    }
  }

  return (
    <AppShell title="My candidacy">
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

      {loadError ? (
        <LoadError
          title="We could not load your applications"
          message={loadError}
          backHref="/elections"
          backLabel="Browse elections"
        />
      ) : !items ? (
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
          Open an election that has not started voting, then choose &quot;Become a candidate&quot;.
        </EmptyState>
      ) : null}

      {items && items.length > 0 ? (
        <div className="stack">
          {items.map(({ candidate, election_title, position_title }) => {
            const canWithdraw =
              candidate.approval_status !== "WITHDRAWN" &&
              candidate.approval_status !== "REJECTED" &&
              candidate.approval_status !== "APPROVED";
            const approved = candidate.approval_status === "APPROVED";
            return (
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
                      <h2 style={{ marginTop: 8 }}>{election_title || "Election"}</h2>
                      <p className="hint">
                        {position_title || "Position"} · applied {formatEAT(candidate.created_at)}
                      </p>
                      {candidate.slogan ? <p style={{ marginTop: 6 }}>{candidate.slogan}</p> : null}
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
                    {canWithdraw ? (
                      <button
                        type="button"
                        className="btn sm danger-o"
                        onClick={() => {
                          setWithdrawing({
                            candidate,
                            election_title,
                            position_title,
                          });
                          setReason("");
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
                {approved ? (
                  <div
                    className="panel stack"
                    style={{ marginTop: 14, background: "var(--sunken)" }}
                  >
                    <span className="lbl">Ballot photo</span>
                    <p className="hint" style={{ margin: 0 }}>
                      Approved candidates cannot withdraw. You may replace the image shown on the
                      ballot until voting opens — otherwise your passport photo is used.
                    </p>
                    <div className="row" style={{ alignItems: "center", gap: 12 }}>
                      {candidate.ballot_photo_url ? (
                        // eslint-disable-next-line @next/next/no-img-element
                        <img
                          src={fileUrl(candidate.ballot_photo_url)}
                          alt=""
                          width={56}
                          height={56}
                          style={{
                            objectFit: "cover",
                            border: "1px solid var(--line)",
                          }}
                        />
                      ) : null}
                      <label className="btn sm" style={{ cursor: "pointer" }}>
                        <Icon name="up" size="s" />
                        {photoBusy === candidate.id ? "Uploading…" : "Replace image"}
                        <input
                          type="file"
                          accept="image/jpeg,image/png"
                          hidden
                          disabled={photoBusy !== ""}
                          onChange={(event) =>
                            changeBallotPhoto(candidate.id, event.target.files?.[0])
                          }
                        />
                      </label>
                    </div>
                  </div>
                ) : null}
              </article>
            );
          })}
        </div>
      ) : null}

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
    </AppShell>
  );
}

export default function MyCandidacyPage() {
  return (
    <RequireAuth>
      <MyCandidacy />
    </RequireAuth>
  );
}
