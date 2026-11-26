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

/**
 * A candidate an administrator selected. They accept first, then submit the
 * same profile details plus a mandatory PDF, MP3 or MP4 manifesto. The same
 * form edits those details until voting opens.
 */
function SelectedCandidacy({ item, onSaved }: { item: MyCandidate; onSaved: () => void }) {
  const toast = useToast();
  const candidate = item.candidate;
  const [busy, setBusy] = useState(false);
  const [fullName, setFullName] = useState(
    candidate.full_name === candidate.invited_email ? "" : candidate.full_name
  );
  const [dateOfBirth, setDateOfBirth] = useState(candidate.date_of_birth?.slice(0, 10) ?? "");
  const [sex, setSex] = useState(candidate.sex ?? "");
  const [slogan, setSlogan] = useState(candidate.slogan ?? "");
  const [bio, setBio] = useState(candidate.bio ?? "");
  const [manifesto, setManifesto] = useState(candidate.manifesto ?? "");
  const [manifestoUrl, setManifestoUrl] = useState(candidate.manifesto_url ?? "");
  const [passportPhotoUrl, setPassportPhotoUrl] = useState(candidate.passport_photo_url ?? "");
  const [contactEmail, setContactEmail] = useState(candidate.contact_email ?? "");
  const [contactPhone, setContactPhone] = useState(candidate.contact_phone ?? "");

  async function submit(event: React.FormEvent) {
    event.preventDefault();
    if (!dateOfBirth || (sex !== "MALE" && sex !== "FEMALE")) {
      toast.error("Enter your date of birth and sex.");
      return;
    }
    if (!passportPhotoUrl) {
      toast.error("Upload your passport photo.");
      return;
    }
    if (!manifestoUrl) {
      toast.error("Upload your manifesto (PDF, MP3 or MP4).");
      return;
    }
    setBusy(true);
    try {
      await candidates.submit(candidate.id, {
        full_name: fullName.trim(),
        date_of_birth: dateOfBirth,
        sex,
        slogan: slogan.trim(),
        bio: bio.trim(),
        manifesto: manifesto.trim(),
        manifesto_url: manifestoUrl,
        passport_photo_url: passportPhotoUrl,
        contact_email: contactEmail.trim(),
        contact_phone: contactPhone.trim(),
      });
      toast.success("Your candidacy is published.");
      onSaved();
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Your candidacy could not be saved.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <form className="panel stack" style={{ marginTop: 14 }} onSubmit={submit}>
      <span className="lbl">Your details</span>
      <p className="hint" style={{ margin: 0 }}>
        You were selected for this position. Fill in your details and publish a manifesto; no
        transcript or approval is needed. You can edit this until voting opens.
      </p>
      <div className="form-grid">
        <div className="field">
          <label htmlFor={`sel-name-${candidate.id}`}>Name on the ballot *</label>
          <input
            id={`sel-name-${candidate.id}`}
            className="in"
            value={fullName}
            onChange={(event) => setFullName(event.target.value)}
            maxLength={255}
            required
          />
        </div>
        <div className="field">
          <label htmlFor={`sel-dob-${candidate.id}`}>Date of birth *</label>
          <input
            id={`sel-dob-${candidate.id}`}
            className="in"
            type="date"
            value={dateOfBirth}
            onChange={(event) => setDateOfBirth(event.target.value)}
            required
          />
        </div>
        <div className="field">
          <label htmlFor={`sel-sex-${candidate.id}`}>Sex *</label>
          <select
            id={`sel-sex-${candidate.id}`}
            className="sel"
            value={sex}
            onChange={(event) => setSex(event.target.value)}
            required
          >
            <option value="">Select…</option>
            <option value="MALE">Male</option>
            <option value="FEMALE">Female</option>
          </select>
        </div>
        <div className="field">
          <label htmlFor={`sel-slogan-${candidate.id}`}>Slogan (optional)</label>
          <input
            id={`sel-slogan-${candidate.id}`}
            className="in"
            value={slogan}
            onChange={(event) => setSlogan(event.target.value)}
            maxLength={200}
          />
        </div>
      </div>
      <div className="field">
        <label htmlFor={`sel-bio-${candidate.id}`}>Biography</label>
        <textarea
          id={`sel-bio-${candidate.id}`}
          className="ta"
          rows={4}
          value={bio}
          onChange={(event) => setBio(event.target.value)}
          maxLength={5000}
        />
      </div>
      <div className="field">
        <label htmlFor={`sel-manifesto-${candidate.id}`}>Manifesto summary</label>
        <textarea
          id={`sel-manifesto-${candidate.id}`}
          className="ta"
          rows={4}
          value={manifesto}
          onChange={(event) => setManifesto(event.target.value)}
          maxLength={20000}
        />
      </div>
      <div className="form-grid">
        <FilePicker
          accepts="image"
          maxSize={2 * 1024 * 1024}
          value={passportPhotoUrl}
          onChange={setPassportPhotoUrl}
          label="Passport photo *"
          hint="JPEG, PNG, GIF, WebP · max 2 MB"
        />
        <FilePicker
          accepts="manifesto"
          maxSize={100 * 1024 * 1024}
          value={manifestoUrl}
          onChange={setManifestoUrl}
          label="Manifesto file *"
          hint="PDF, MP3 or MP4 · max 100 MB"
        />
      </div>
      <div className="form-grid">
        <div className="field">
          <label htmlFor={`sel-email-${candidate.id}`}>Contact email</label>
          <input
            id={`sel-email-${candidate.id}`}
            className="in"
            type="email"
            value={contactEmail}
            onChange={(event) => setContactEmail(event.target.value)}
          />
        </div>
        <div className="field">
          <label htmlFor={`sel-phone-${candidate.id}`}>Phone (optional)</label>
          <input
            id={`sel-phone-${candidate.id}`}
            className="in"
            value={contactPhone}
            onChange={(event) => setContactPhone(event.target.value)}
          />
        </div>
      </div>
      <div className="row">
        <button className="btn primary" type="submit" disabled={busy}>
          {busy ? "Saving…" : candidate.approval_status === "APPROVED" ? "Save changes" : "Publish candidacy"}
        </button>
      </div>
    </form>
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
  const [withdrawDoc, setWithdrawDoc] = useState("");
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

  async function acceptInvitation(candidateId: string) {
    setBusy(true);
    try {
      await candidates.accept(candidateId);
      toast.success("Invitation accepted. Add your details and manifesto.");
      await load();
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "The invitation could not be accepted.");
    } finally {
      setBusy(false);
    }
  }

  async function withdraw() {
    if (!withdrawing) return;
    if (!reason.trim() && !withdrawDoc) {
      toast.error("Type a reason or attach a signed letter before withdrawing.");
      return;
    }
    setBusy(true);
    try {
      await candidates.withdraw(withdrawing.candidate.id, {
        reason: reason.trim(),
        documentUrl: withdrawDoc || undefined,
      });
      toast.success("Your application has been withdrawn.");
      setWithdrawing(null);
      setReason("");
      setWithdrawDoc("");
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
              candidate.approval_status !== "REJECTED";
            const approved = candidate.approval_status === "APPROVED";
            const invited = Boolean(candidate.invited_email);
            const awaitingAcceptance = candidate.approval_status === "SELECTED";
            const selectedEditable =
              invited &&
              !votingOpen &&
              (candidate.approval_status === "ACCEPTED" || approved);
            // An invited candidate edits through the selection form, which keeps
            // the manifesto mandatory; the general manifesto editor would allow
            // removing it.
            const canEdit = approved && !votingOpen && !invited;
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
                    {awaitingAcceptance ? (
                      <button
                        type="button"
                        className="btn sm primary"
                        disabled={busy}
                        onClick={() => acceptInvitation(candidate.id)}
                      >
                        <Icon name="check" size="s" />
                        Accept invitation
                      </button>
                    ) : null}
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
                {awaitingAcceptance ? (
                  <div className="banner" style={{ marginTop: 14 }}>
                    <Icon name="info" size="s" />
                    <div>
                      <b>You have been selected</b>
                      An administrator selected you for this position. Accept the invitation, then
                      add your details and a manifesto. No transcript or approval step is needed.
                    </div>
                  </div>
                ) : null}
                {selectedEditable ? <SelectedCandidacy item={item} onSaved={load} /> : null}
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
                      You may replace the image shown on the ballot until voting opens — otherwise
                      your passport photo is used. You can also withdraw before voting opens.
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
                      accepts="manifesto"
                      maxSize={100 * 1024 * 1024}
                      label="Manifesto document (optional)"
                      hint="A PDF, MP3 or MP4 voters can read or play in addition to the text above."
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
        <FilePicker
          accepts="pdf"
          maxSize={5 * 1024 * 1024}
          value={withdrawDoc}
          onChange={setWithdrawDoc}
          label="Signed withdrawal letter (optional)"
          hint="Use this instead of typing a reason, or alongside it."
        />
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
