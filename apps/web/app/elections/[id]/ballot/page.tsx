"use client";

import Link from "next/link";
import { use, useEffect, useMemo, useState } from "react";
import { ConfirmDialog } from "@/components/confirm-dialog";
import { Countdown } from "@/components/countdown";
import { Icon } from "@/components/icon";
import { RequireAuth } from "@/components/require-auth";
import { SiteHeader } from "@/components/site-header";
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

function BallotForm({ id }: { id: string }) {
  const [ballot, setBallot] = useState<Ballot | null>(null);
  const [selections, setSelections] = useState<Record<string, string[]>>({});
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [loadError, setLoadError] = useState("");
  const [submitError, setSubmitError] = useState("");
  const [alreadyVoted, setAlreadyVoted] = useState(false);
  const [confirming, setConfirming] = useState(false);
  const [busy, setBusy] = useState(false);
  const [receipt, setReceipt] = useState<SubmitResult | null>(null);

  useEffect(() => {
    let active = true;
    elections
      .ballot(id)
      .then((data) => {
        if (active) setBallot(data);
      })
      .catch((err) => {
        if (active)
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
    if (alreadyVoted) return;
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

  function requestSubmit() {
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
    setConfirming(true);
  }

  async function submit() {
    if (!ballot) return;
    setBusy(true);
    setSubmitError("");
    try {
      const result = await elections.submitBallot(
        ballot.election_id,
        buildSelections(ballot, selections)
      );
      setReceipt(result);
      setConfirming(false);
    } catch (err) {
      setConfirming(false);
      if (err instanceof ApiError && err.status === 409) {
        setAlreadyVoted(true);
      }
      setSubmitError(
        err instanceof ApiError ? err.message : "Your ballot was not accepted. Try again."
      );
    } finally {
      setBusy(false);
    }
  }

  if (receipt) {
    return (
      <main className="app-wrap narrow">
        <div className="stamp-wrap">
          <div className="stamp-box">
            <span className="stamp go" style={{ display: "inline-block" }}>
              <span className="badge" aria-hidden="true">
                <svg className="tick" viewBox="0 0 24 24" fill="none" stroke="currentColor">
                  <path d="M20 6 9 17l-5-5" />
                </svg>
              </span>
            </span>
            <h2>Your ballot has been accepted</h2>
            <p className="hint">
              It cannot be changed or submitted again. Officials and candidates cannot see your
              choices.
            </p>
            <div style={{ marginTop: 20 }}>
              <span className="lbl">Confirmation receipt</span>
              <p className="code-block" style={{ marginTop: 8 }}>
                {receipt.confirmation_token}
              </p>
              <p className="hint" style={{ marginTop: 8 }}>
                Keep this private. It proves participation, not how you voted.
              </p>
            </div>
            <p style={{ marginTop: 24 }}>
              <Link className="btn primary" href="/elections">
                Back to my elections
              </Link>
            </p>
          </div>
        </div>
      </main>
    );
  }

  return (
    <main className="app-wrap">
      <nav className="crumbs" aria-label="Breadcrumb" style={{ marginBottom: 16 }}>
        <Link href="/elections">Elections</Link>
        <span>/</span>
        <Link href={`/elections/${id}`}>{ballot?.title ?? "Election"}</Link>
        <span>/</span>
        <span>Ballot</span>
      </nav>

      <div className="page-head">
        <div>
          <span className="lbl">Secret ballot</span>
          <h1>{ballot?.title ?? "Loading ballot…"}</h1>
          {ballot ? (
            <p>
              Closes {formatEAT(ballot.ends_at)}. You may leave an optional position blank. You
              cannot change your ballot after you submit it.
            </p>
          ) : null}
        </div>
        {ballot ? <Countdown endsAt={ballot.ends_at} /> : null}
      </div>

      {loadError ? (
        <div className="banner bad" role="alert">
          <Icon name="alert" />
          <div>
            <b>Ballot unavailable</b>
            {loadError}
          </div>
        </div>
      ) : null}

      {alreadyVoted ? (
        <div className="banner ok" style={{ marginBottom: 18 }}>
          <Icon name="check" />
          <div>
            <b>You have already voted</b>A ballot is final once accepted, so this page will not
            submit another one.
          </div>
        </div>
      ) : null}

      <div className="banner" style={{ marginBottom: 18 }}>
        <Icon name="lock" />
        <div>
          <b>Your vote is secret</b>
          This page is not stored, and no official or candidate can see your choices.
        </div>
      </div>

      {!ballot && !loadError ? (
        <div className="stack" role="status" aria-busy="true" aria-label="Loading ballot">
          <div className="ballot-position">
            <span className="skel" style={{ height: 18, width: "35%" }} />
            <span className="skel" style={{ display: "block", height: 60, marginTop: 12 }} />
            <span className="skel" style={{ display: "block", height: 60, marginTop: 8 }} />
          </div>
        </div>
      ) : null}

      {ballot ? (
        <>
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
                        disabled={alreadyVoted}
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
                        disabled={alreadyVoted}
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
                        disabled={alreadyVoted}
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
                      disabled={alreadyVoted}
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

          {submitError ? (
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
                onClick={requestSubmit}
                disabled={alreadyVoted || busy}
              >
                Submit ballot
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
