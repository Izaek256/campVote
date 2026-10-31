"use client";

import Link from "next/link";
import { use, useEffect, useMemo, useRef, useState } from "react";
import { ConfirmDialog } from "@/components/confirm-dialog";
import { Countdown } from "@/components/countdown";
import { Icon } from "@/components/icon";
import { RequireAuth } from "@/components/require-auth";
import { SiteHeader } from "@/components/site-header";
import { VoteRecordedNotice } from "@/components/vote-recorded-notice";
import { ApiError, elections } from "@/lib/api";
import { formatEAT } from "@/lib/format";
import {
  ABSTAIN,
  blankAllowed,
  buildSelections,
  isReferendum,
  limits,
  validate,
} from "@/lib/ballot";
import type { Ballot, BallotPosition, SubmitResult } from "@/lib/types";

/** Selections live only in this component's state, never in storage or the URL. */
function BallotForm({ id }: { id: string }) {
  const [ballot, setBallot] = useState<Ballot | null>(null);
  const [selections, setSelections] = useState<Record<string, string[]>>({});
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [loadError, setLoadError] = useState("");
  const [loadStatus, setLoadStatus] = useState(0);
  const [submitError, setSubmitError] = useState("");
  const [alreadyVoted, setAlreadyVoted] = useState(false);
  const [stale, setStale] = useState(false);
  const [unknownOutcome, setUnknownOutcome] = useState(false);
  const [reviewing, setReviewing] = useState(false);
  const [confirmed, setConfirmed] = useState(false);
  const [confirming, setConfirming] = useState(false);
  const [busy, setBusy] = useState(false);
  const [receipt, setReceipt] = useState<SubmitResult | null>(null);
  // One key per attempt set. A manual retry reuses it; nothing rotates it on
  // its own, so a lost response can never become a second ballot.
  const attemptKey = useRef<string>(
    typeof crypto !== "undefined" && "randomUUID" in crypto
      ? crypto.randomUUID()
      : `attempt-${Date.now()}-${Math.random().toString(36).slice(2)}`
  );

  useEffect(() => {
    let active = true;
    elections
      .ballot(id)
      .then((data) => {
        if (active) setBallot(data);
      })
      .catch((err) => {
        if (!active) return;
        setLoadStatus(err instanceof ApiError ? err.status : 0);
        setLoadError(err instanceof ApiError ? err.message : "The ballot could not be loaded.");
      });
    elections
      .startBallot(id)
      .then(() => {
        if (active) setAlreadyVoted(false);
      })
      .catch((err) => {
        if (active && err instanceof ApiError && err.status === 409) setAlreadyVoted(true);
      });
    return () => {
      active = false;
    };
  }, [id]);

  const answered = useMemo(
    () => Object.values(selections).filter((chosen) => chosen.length > 0).length,
    [selections]
  );

  function choose(position: BallotPosition, value: string) {
    if (alreadyVoted || stale) return;
    const { max } = limits(position);
    setErrors((current) => {
      const next = { ...current };
      delete next[position.id];
      return next;
    });
    setSelections((current) => {
      const chosen = current[position.id] ?? [];
      if (position.question_type === "MULTI_CHOICE") {
        if (chosen.includes(value)) {
          return { ...current, [position.id]: chosen.filter((item) => item !== value) };
        }
        if (chosen.includes(ABSTAIN)) {
          return { ...current, [position.id]: [value] };
        }
        if (chosen.length >= max) return current;
        return { ...current, [position.id]: [...chosen, value] };
      }
      return { ...current, [position.id]: [value] };
    });
  }

  function requestReview() {
    if (!ballot) return;
    const found = validate(ballot, selections);
    setErrors(found);
    if (Object.keys(found).length > 0) {
      const first = document.querySelector<HTMLElement>("[data-invalid='true']");
      first?.scrollIntoView({ block: "center" });
      first?.focus();
      return;
    }
    setSubmitError("");
    setConfirmed(false);
    setReviewing(true);
    window.scrollTo({ top: 0 });
  }

  function requestSubmit() {
    if (!confirmed) return;
    setConfirming(true);
  }

  async function submit() {
    if (!ballot) return;
    setBusy(true);
    setSubmitError("");
    setUnknownOutcome(false);
    try {
      const result = await elections.submitBallot(
        ballot.election_id,
        buildSelections(ballot, selections),
        { ballotVersionId: ballot.ballot_version_id, idempotencyKey: attemptKey.current }
      );
      setReceipt(result);
      setSelections({});
      setConfirming(false);
      setReviewing(false);
    } catch (err) {
      setConfirming(false);
      if (err instanceof ApiError) {
        if (err.status === 409 && /changed/i.test(err.message)) {
          setStale(true);
        } else if (err.status === 409) {
          setAlreadyVoted(true);
        }
        setSubmitError(err.message);
      } else {
        // The request may have committed before the response was lost.
        setUnknownOutcome(true);
        setSubmitError("We could not confirm your vote. Check your status before trying again.");
      }
    } finally {
      setBusy(false);
    }
  }

  function choiceLabel(position: BallotPosition, value: string): string {
    if (value === ABSTAIN) return "Leave this position blank";
    if (value === "YES") return "YES";
    if (value === "NO") return "NO";
    return position.candidates.find((candidate) => candidate.id === value)?.full_name ?? "—";
  }

  if (receipt) {
    return (
      <main className="app-wrap narrow">
        <VoteRecordedNotice electionTitle={ballot?.title} />
        <div className="panel stack" style={{ marginTop: 20 }}>
          <div>
            <span className="lbl">Participation reference</span>
            <p className="code-block" style={{ marginTop: 8 }}>
              {receipt.confirmation_token}
            </p>
            <p className="hint" style={{ marginTop: 8 }}>
              Keep this private. It proves that you voted, not how you voted.
            </p>
          </div>
          <div className="row">
            <Link className="btn" href={`/elections/${id}/status`}>
              <Icon name="shield" size="s" />
              Check vote status
            </Link>
            <Link className="btn ghost" href="/elections">
              Back to my elections
            </Link>
          </div>
        </div>
      </main>
    );
  }

  const blocked = Boolean(loadError) || alreadyVoted || stale;

  return (
    <main className="app-wrap">
      <nav className="crumbs" aria-label="Breadcrumb" style={{ marginBottom: 16 }}>
        <Link href="/elections">Elections</Link>
        <span>/</span>
        <Link href={`/elections/${id}`}>{ballot?.title ?? "Election"}</Link>
        <span>/</span>
        <span>{reviewing ? "Review" : "Ballot"}</span>
      </nav>

      <div className="page-head">
        <div>
          <span className="lbl">Secret ballot</span>
          <h1>{ballot?.title ?? "Loading ballot…"}</h1>
          {ballot ? (
            <p>
              Closes {formatEAT(ballot.ends_at)}. You may leave an optional position blank. You
              cannot change your ballot after you submit it.
              {ballot.ballot_version ? ` Ballot version ${ballot.ballot_version}.` : ""}
            </p>
          ) : null}
        </div>
        {ballot ? <Countdown endsAt={ballot.ends_at} /> : null}
      </div>

      {loadError ? (
        <div
          className={`banner ${loadStatus === 403 || loadStatus === 409 ? "hatch" : "bad"}`}
          role="alert"
        >
          <Icon name={loadStatus === 401 ? "lock" : "alert"} />
          <div>
            <b>
              {loadStatus === 401
                ? "Your session expired"
                : loadStatus === 403
                  ? "The ballot is not available to you"
                  : "Ballot unavailable"}
            </b>
            {loadError}
            {loadStatus === 401 ? (
              <>
                {" "}
                <Link href="/login">Sign in again</Link> to continue. Your selections are not saved.
              </>
            ) : null}
          </div>
        </div>
      ) : null}

      {alreadyVoted ? (
        <div className="banner ok" style={{ marginBottom: 18 }}>
          <Icon name="check" />
          <div>
            <b>You have already voted</b>A ballot is final once accepted, so this page will not
            submit another one. <Link href={`/elections/${id}/status`}>Check your status</Link>.
          </div>
        </div>
      ) : null}

      {stale ? (
        <div className="banner hatch" role="alert" style={{ marginBottom: 18 }}>
          <Icon name="alert" />
          <div>
            <b>The ballot changed while you were filling it in</b>
            Nothing has been recorded. Reload the ballot and choose again.
            <button
              type="button"
              className="btn sm"
              style={{ marginLeft: 12 }}
              onClick={() => window.location.reload()}
            >
              Reload ballot
            </button>
          </div>
        </div>
      ) : null}

      {unknownOutcome ? (
        <div className="banner bad" role="alert" style={{ marginBottom: 18 }}>
          <Icon name="alert" />
          <div>
            <b>We could not confirm your vote</b>
            {submitError} Do not submit again until you have checked your status.
            <div style={{ marginTop: 8 }}>
              <Link className="btn sm" href={`/elections/${id}/status`}>
                Check status
              </Link>
            </div>
          </div>
        </div>
      ) : null}

      {!ballot && !loadError ? (
        <div className="stack" role="status" aria-busy="true" aria-label="Loading ballot">
          <div className="ballot-position">
            <span className="skel" style={{ height: 18, width: "35%" }} />
            <span className="skel" style={{ display: "block", height: 60, marginTop: 12 }} />
            <span className="skel" style={{ display: "block", height: 60, marginTop: 8 }} />
          </div>
        </div>
      ) : null}

      {ballot && reviewing ? (
        <section className="panel stack" aria-labelledby="review-heading">
          <div className="section-h" style={{ marginTop: 0 }}>
            <h2 id="review-heading">Review your selections</h2>
            <span className="mono">
              {answered} of {ballot.positions.length} answered
            </span>
          </div>
          <p className="hint">
            Check each position. This is your last chance to change a choice. Nothing is recorded
            until you press Submit.
          </p>
          <dl className="kv">
            {ballot.positions.map((position) => {
              const chosen = selections[position.id] ?? [];
              return [
                <dt key={`${position.id}-t`}>{position.title}</dt>,
                <dd key={`${position.id}-d`}>
                  {chosen.length === 0 ? (
                    <span className="hint">Left blank</span>
                  ) : (
                    chosen.map((value) => <div key={value}>{choiceLabel(position, value)}</div>)
                  )}
                  <button
                    type="button"
                    className="btn ghost sm"
                    style={{ marginTop: 6 }}
                    onClick={() => {
                      setReviewing(false);
                      document
                        .getElementById(`pos-${position.id}`)
                        ?.scrollIntoView({ block: "center" });
                    }}
                  >
                    Edit
                  </button>
                </dd>,
              ];
            })}
          </dl>

          <div className="banner">
            <Icon name="lock" />
            <div>
              <b>Your choices stay secret</b>
              The record proves that you voted, not how you voted. Officials and candidates cannot
              see your choices.
            </div>
          </div>

          <label className="field" style={{ gap: 8 }}>
            <span className="row" style={{ alignItems: "flex-start" }}>
              <input
                type="checkbox"
                checked={confirmed}
                onChange={(event) => setConfirmed(event.target.checked)}
                aria-describedby="confirm-help"
              />
              <span>
                I understand this ballot is final and cannot be changed or submitted again.
              </span>
            </span>
            <span className="hint" id="confirm-help">
              If the network drops after you press Submit, check your status before trying again.
            </span>
          </label>

          {submitError && !unknownOutcome ? (
            <div className="banner bad" role="alert">
              <Icon name="alert" />
              <div>
                <b>Your ballot was not accepted</b>
                {submitError}
              </div>
            </div>
          ) : null}

          <div className="row" style={{ justifyContent: "space-between" }}>
            <button type="button" className="btn ghost" onClick={() => setReviewing(false)}>
              Back to ballot
            </button>
            <button
              type="button"
              className="btn primary lg"
              onClick={requestSubmit}
              disabled={!confirmed || busy || alreadyVoted || stale}
            >
              {busy ? "Submitting…" : "Submit ballot"}
            </button>
          </div>
        </section>
      ) : null}

      {ballot && !reviewing ? (
        <>
          <div className="banner" style={{ marginBottom: 18 }}>
            <Icon name="lock" />
            <div>
              <b>Your vote is secret</b>
              This page is not stored, and no official or candidate can see your choices.
            </div>
          </div>

          {ballot.positions.map((position) => {
            const chosen = selections[position.id] ?? [];
            const { min, max } = limits(position);
            const error = errors[position.id];
            const multi = position.question_type === "MULTI_CHOICE";
            return (
              <section
                className="ballot-position"
                key={position.id}
                data-invalid={error ? "true" : undefined}
                tabIndex={error ? -1 : undefined}
                aria-labelledby={`pos-${position.id}`}
              >
                <div className="row-between">
                  <div>
                    <span className="lbl">
                      Position {position.display_order || ""}{" "}
                      {min > 0 ? "· required" : "· optional"}
                    </span>
                    <h2 id={`pos-${position.id}`} style={{ fontSize: 24, marginTop: 4 }}>
                      {position.title}
                    </h2>
                    {position.description ? <p className="hint">{position.description}</p> : null}
                  </div>
                  {multi ? (
                    <span
                      className={`counter${chosen.length === max ? " full" : ""}${error ? " bad" : ""}`}
                      aria-live="polite"
                    >
                      {chosen.length} of {max} selected
                    </span>
                  ) : null}
                </div>

                {/* biome-ignore lint/a11y/useAriaPropsSupportedByRole: role is group or radiogroup; both support aria-labelledby. */}
                <div
                  className="ballot-list"
                  role={multi ? "group" : "radiogroup"}
                  aria-labelledby={`pos-${position.id}`}
                >
                  {isReferendum(position) ? (
                    <>
                      <button
                        type="button"
                        className={`opt${error ? " bad" : ""}`}
                        role="radio"
                        aria-checked={chosen.includes("YES")}
                        disabled={blocked}
                        onClick={() => choose(position, "YES")}
                      >
                        <span className="box rd" />
                        <span className="nm">YES</span>
                      </button>
                      <button
                        type="button"
                        className={`opt${error ? " bad" : ""}`}
                        role="radio"
                        aria-checked={chosen.includes("NO")}
                        disabled={blocked}
                        onClick={() => choose(position, "NO")}
                      >
                        <span className="box rd" />
                        <span className="nm">NO</span>
                      </button>
                    </>
                  ) : (
                    position.candidates.map((candidate) => (
                      // biome-ignore lint/a11y/useAriaPropsSupportedByRole: role is a literal radio or checkbox; both support aria-checked.
                      <button
                        type="button"
                        className={`opt${error ? " bad" : ""}`}
                        key={candidate.id}
                        role={multi ? "checkbox" : "radio"}
                        aria-checked={chosen.includes(candidate.id)}
                        disabled={blocked}
                        onClick={() => choose(position, candidate.id)}
                      >
                        <span className={`box${multi ? "" : " rd"}`}>
                          {multi ? <Icon name="check" size="s" /> : null}
                        </span>
                        <span
                          className="ph r11"
                          style={{ width: 48, flex: "none" }}
                          aria-hidden="true"
                        >
                          <span style={{ padding: "0 4px" }}>{candidate.ballot_number}</span>
                        </span>
                        <span style={{ minWidth: 0 }}>
                          <span className="nm">{candidate.full_name}</span>
                          {candidate.manifesto ? (
                            <span className="candidate-meta">{candidate.manifesto}</span>
                          ) : null}
                        </span>
                      </button>
                    ))
                  )}

                  {blankAllowed(position) ? (
                    // biome-ignore lint/a11y/useAriaPropsSupportedByRole: role is a literal radio or checkbox; both support aria-checked.
                    <button
                      type="button"
                      className="opt"
                      role={multi ? "checkbox" : "radio"}
                      aria-checked={chosen.includes(ABSTAIN)}
                      disabled={blocked}
                      onClick={() => choose(position, ABSTAIN)}
                    >
                      <span className={`box${multi ? "" : " rd"}`}>
                        {multi ? <Icon name="check" size="s" /> : null}
                      </span>
                      <span className="nm">Leave this position blank</span>
                    </button>
                  ) : null}
                </div>

                {error ? (
                  <span
                    className="err"
                    style={{ display: "flex", gap: 6, marginTop: 10 }}
                    role="alert"
                  >
                    <Icon name="alert" size="s" />
                    {error}
                  </span>
                ) : null}
              </section>
            );
          })}

          {submitError && !unknownOutcome ? (
            <div className="banner bad" role="alert" style={{ marginTop: 18 }}>
              <Icon name="alert" />
              <div>
                <b>Your ballot was not accepted</b>
                {submitError}
              </div>
            </div>
          ) : null}

          <div className="ballot-actions">
            <span className="hint">
              {answered} of {ballot.positions.length} positions answered
            </span>
            <div className="row">
              <Link className="btn ghost" href={`/elections/${id}`}>
                Back
              </Link>
              <button
                type="button"
                className="btn primary lg"
                onClick={requestReview}
                disabled={blocked || busy}
              >
                Review selections
              </button>
            </div>
          </div>
        </>
      ) : null}

      <ConfirmDialog
        open={confirming}
        title="Submit this ballot?"
        confirmLabel="Submit ballot"
        busy={busy}
        onClose={() => setConfirming(false)}
        onConfirm={submit}
      >
        <p>
          You answered <b>{answered}</b> of {ballot?.positions.length ?? 0} positions. A ballot is
          final once accepted: it cannot be changed, withdrawn or submitted again.
        </p>
        <div className="banner">
          <Icon name="lock" />
          <div>
            <b>Your choices stay secret</b>
            The receipt proves you voted, not how.
          </div>
        </div>
      </ConfirmDialog>
    </main>
  );
}

export default function BallotPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  return (
    <RequireAuth>
      <SiteHeader />
      <BallotForm id={id} />
    </RequireAuth>
  );
}
