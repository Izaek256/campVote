"use client";

import Link from "next/link";
import { useCallback, useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { AppShell } from "@/components/app-shell";
import { ConfirmDialog } from "@/components/confirm-dialog";
import { EmptyState } from "@/components/empty-state";
import { FilePicker } from "@/components/file-picker";
import { Icon } from "@/components/icon";
import { LoadError } from "@/components/load-error";
import { RequireAuth } from "@/components/require-auth";
import { StatusChip } from "@/components/status-chip";
import { useAuth } from "@/components/auth-provider";
import { useToast } from "@/components/toast";
import { ApiError, candidates, fileUrl, upload } from "@/lib/api";
import {
  candidateStatusClass,
  candidateStatusLabel,
  formatEAT,
  formatNumber,
  formatPercent,
} from "@/lib/format";
import type { MyCandidate } from "@/lib/types";

/**
 * The votes a candidate received, read from the published count. It shows only
 * their own figure: a ballot never identifies a voter, so nothing here can
 * point back at one.
 */
function VotesDialog({ entry, onClose }: { entry: MyCandidate | null; onClose: () => void }) {
  const ref = useRef<HTMLDialogElement>(null);

  useEffect(() => {
    const dialog = ref.current;
    if (!dialog) return;
    if (entry && !dialog.open) dialog.showModal();
    if (!entry && dialog.open) dialog.close();
  }, [entry]);

  const result = entry?.result;
  const certified = result?.snapshot_status === "CERTIFIED";

  return (
    <dialog ref={ref} className="modal-dialog modal" onClose={onClose}>
      <div className="modal-h">
        <h3>Your votes</h3>
      </div>
      <div className="modal-b">
        {entry ? (
          <div className="stack">
            <p className="hint" style={{ margin: 0 }}>
              {entry.election_title} · {entry.position_title}
            </p>
            {result ? (
              <>
                <div className="row" style={{ gap: 24 }}>
                  <div className="field">
                    <span className="lbl">Votes received</span>
                    <b style={{ fontSize: 28 }}>{formatNumber(result.votes)}</b>
                  </div>
                  <div className="field">
                    <span className="lbl">Share of valid votes</span>
                    <b style={{ fontSize: 28 }}>{formatPercent(result.share_percent)}</b>
                  </div>
                </div>
                <div className="row">
                  {result.is_tie ? (
                    <span className="chip review">Tied</span>
                  ) : result.is_winner ? (
                    <span className="chip final">{certified ? "Elected" : "Leading"}</span>
                  ) : (
                    <span className="chip draft">Not elected</span>
                  )}
                  <span className="hint">
                    {certified ? "Certified count" : "Provisional count"}
                  </span>
                </div>
                <p className="hint" style={{ margin: 0 }}>
                  Your share is of the valid votes for this position. Only your own figure is shown;
                  ballots never identify a voter.
                </p>
              </>
            ) : (
              <p className="hint" style={{ margin: 0 }}>
                The count for this position has not been published yet.
              </p>
            )}
          </div>
        ) : null}
      </div>
      <div className="modal-f">
        <button type="button" className="btn ghost" onClick={onClose}>
          Close
        </button>
      </div>
    </dialog>
  );
}

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
  const [withdrawing, setWithdrawing] = useState<Pick<
    MyCandidate,
    "candidate" | "election_title" | "position_title"
  > | null>(null);
  const [votesFor, setVotesFor] = useState<MyCandidate | null>(null);
  const [reason, setReason] = useState("");
  const [photoBusy, setPhotoBusy] = useState("");
  const [manifestoBusy, setManifestoBusy] = useState("");
  const [removingFor, setRemovingFor] = useState<MyCandidate["candidate"] | null>(null);
  const [manifestoDraft, setManifestoDraft] = useState<
    Record<string, { text: string; url: string }>
  >({});

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

  /** The manifesto form value for a candidate, defaulting to what they saved. */
  function draftFor(candidate: MyCandidate["candidate"]) {
    return (
      manifestoDraft[candidate.id] ?? {
        text: candidate.manifesto ?? "",
        url: candidate.manifesto_url ?? "",
      }
    );
  }

  function updateDraft(candidateId: string, patch: Partial<{ text: string; url: string }>) {
    setManifestoDraft((prev) => {
      const current = prev[candidateId] ?? { text: "", url: "" };
      return { ...prev, [candidateId]: { ...current, ...patch } };
    });
  }

  async function saveManifesto(candidate: MyCandidate["candidate"]) {
    const draft = draftFor(candidate);
    if (!draft.text.trim() && !draft.url.trim()) {
      toast.error("Write your manifesto or attach a document.");
      return;
    }
    setManifestoBusy(candidate.id);
    try {
      await candidates.setManifesto(candidate.id, {
        manifesto: draft.text.trim(),
        manifesto_url: draft.url.trim(),
      });
      toast.success("Manifesto saved. Voters can read it before voting opens.");
      await load();
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "The manifesto could not be saved.");
    } finally {
      setManifestoBusy("");
    }
  }

  /**
   * Remove a published manifesto (text and document) so the candidate can start
   * again. The server refuses this once voting has opened, like any edit.
   */
  async function removeManifesto(candidate: MyCandidate["candidate"]) {
    setManifestoBusy(candidate.id);
    try {
      await candidates.setManifesto(candidate.id, { clear: true });
      setManifestoDraft((prev) => ({ ...prev, [candidate.id]: { text: "", url: "" } }));
      toast.success("Manifesto removed.");
      setRemovingFor(null);
      await load();
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "The manifesto could not be removed.");
    } finally {
      setManifestoBusy("");
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
          {items.map((item) => {
            const { candidate, election_title, position_title } = item;
            // Once voting has opened the manifesto and ballot photo are frozen,
            // and once the election has closed the candidacy is over: the card
            // is read-only and shows the votes that name received.
            const closed = item.election_status === "CLOSED";
            const votingOpen = closed || new Date(item.election_starts_at).getTime() <= Date.now();
            const canWithdraw =
              !votingOpen &&
              candidate.approval_status !== "WITHDRAWN" &&
              candidate.approval_status !== "REJECTED" &&
              candidate.approval_status !== "APPROVED";
            const approved = candidate.approval_status === "APPROVED";
            const canEdit = approved && !votingOpen;
            return (
              <article
                className="panel"
                key={candidate.id}
                style={closed ? { opacity: 0.72 } : undefined}
              >
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
                          filter: closed ? "grayscale(1)" : undefined,
                        }}
                      />
                    ) : null}
                    <div>
                      {closed ? (
                        <StatusChip status={item.election_status} />
                      ) : (
                        <span className={`chip ${candidateStatusClass(candidate.approval_status)}`}>
                          {candidateStatusLabel(candidate.approval_status)}
                        </span>
                      )}
                      <h2 style={{ marginTop: 8 }}>{election_title || "Election"}</h2>
                      <p className="hint">
                        {position_title || "Position"} · applied {formatEAT(candidate.created_at)}
                      </p>
                      {closed ? (
                        <p className="hint">
                          This election has closed. You stood as a candidate and the count is final.
                        </p>
                      ) : null}
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
                    {item.result ? (
                      <button
                        type="button"
                        className="btn sm primary"
                        onClick={() => setVotesFor(item)}
                      >
                        <Icon name="chart" size="s" />
                        View votes
                      </button>
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
                {approved && votingOpen && !closed ? (
                  <div className="banner" style={{ marginTop: 14 }}>
                    <Icon name="lock" size="s" />
                    <div>
                      <b>Editing is locked</b>
                      Voting has started, so your ballot photo and manifesto can no longer be
                      changed.
                    </div>
                  </div>
                ) : null}
                {canEdit ? (
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
                {canEdit ? (
                  <div
                    className="panel stack"
                    style={{ marginTop: 14, background: "var(--sunken)" }}
                  >
                    <span className="lbl">Manifesto</span>
                    <p className="hint" style={{ margin: 0 }}>
                      Voters who qualify for this election can read your manifesto before voting
                      opens. You can edit it until voting starts.
                    </p>
                    <div className="field">
                      <label htmlFor={`manifesto-text-${candidate.id}`}>Your manifesto</label>
                      <textarea
                        id={`manifesto-text-${candidate.id}`}
                        className="ta"
                        rows={6}
                        value={draftFor(candidate).text}
                        onChange={(event) =>
                          updateDraft(candidate.id, { text: event.target.value })
                        }
                        placeholder="What will you do if elected? Explain your priorities."
                      />
                    </div>
                    <FilePicker
                      accepts="pdf"
                      maxSize={5 * 1024 * 1024}
                      label="Manifesto document (optional)"
                      hint="A PDF voters can download in addition to the text above."
                      value={draftFor(candidate).url}
                      onChange={(url) => updateDraft(candidate.id, { url })}
                      disabled={manifestoBusy === candidate.id}
                    />
                    <div className="row">
                      <button
                        type="button"
                        className="btn sm primary"
                        disabled={manifestoBusy === candidate.id}
                        onClick={() => saveManifesto(candidate)}
                      >
                        {manifestoBusy === candidate.id ? "Saving…" : "Save manifesto"}
                      </button>
                      {candidate.manifesto || candidate.manifesto_url ? (
                        <button
                          type="button"
                          className="btn sm danger-o"
                          disabled={manifestoBusy === candidate.id}
                          onClick={() => setRemovingFor(candidate)}
                        >
                          <Icon name="trash" size="s" />
                          Remove manifesto
                        </button>
                      ) : null}
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

      <ConfirmDialog
        open={removingFor !== null}
        title="Remove your manifesto?"
        confirmLabel="Remove manifesto"
        destructive
        busy={manifestoBusy !== "" && removingFor !== null && manifestoBusy === removingFor.id}
        onClose={() => setRemovingFor(null)}
        onConfirm={() => removingFor && removeManifesto(removingFor)}
      >
        <p>
          This removes the manifesto text and any document you attached for{" "}
          <b>{removingFor?.full_name || "your candidacy"}</b>. Voters will no longer read it, and
          you can publish a new one until voting opens.
        </p>
      </ConfirmDialog>

      <VotesDialog entry={votesFor} onClose={() => setVotesFor(null)} />
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
