"use client";

import Link from "next/link";
import { use, useCallback, useEffect, useState } from "react";
import { ConfirmDialog } from "@/components/confirm-dialog";
import { Icon } from "@/components/icon";
import { PositionResult } from "@/components/position-result";
import { RequireAuth } from "@/components/require-auth";
import { ResultStatusBadge } from "@/components/result-status-badge";
import { StaffShell } from "@/components/staff-shell";
import { useToast } from "@/components/toast";
import { TurnoutSummary } from "@/components/turnout-summary";
import { ApiError, documents, elections } from "@/lib/api";
import { formatEAT, formatNumber } from "@/lib/format";
import type { ResultSnapshot, Results, ResultsHistory } from "@/lib/types";

/**
 * Results workspace — ISSUE-09.
 *
 * Officers run a tally, read the snapshot it produced, recount against it and
 * certify it. Everything on this page is a server figure: the hash, the shares,
 * the outcome per position. Nothing here edits a snapshot; a correction is a
 * new tally that supersedes the certified one and keeps it visible.
 */
function Workspace({ id }: { id: string }) {
  const toast = useToast();
  const [election, setElection] = useState<{
    title: string;
    status: string;
  } | null>(null);
  const [results, setResults] = useState<Results | null>(null);
  const [history, setHistory] = useState<ResultsHistory | null>(null);
  const [busy, setBusy] = useState(false);
  const [downloading, setDownloading] = useState("");

  // Confirm dialogs carry the reason the server requires on every one of these.
  const [tallyDialog, setTallyDialog] = useState(false);
  const [tallyReason, setTallyReason] = useState("");
  const [recountFor, setRecountFor] = useState<ResultSnapshot | null>(null);
  const [recountReason, setRecountReason] = useState("");
  const [certifyFor, setCertifyFor] = useState<ResultSnapshot | null>(null);
  const [certifyReason, setCertifyReason] = useState("");
  const [recountResult, setRecountResult] = useState<{
    identical: boolean;
    position_diffs?: Array<{
      title: string;
      outcome_before: string;
      outcome_after: string;
      candidates?: Array<{
        name: string;
        before: number;
        after: number;
        delta: number;
      }>;
    }>;
  } | null>(null);

  const load = useCallback(async () => {
    const restricted: string[] = [];
    const current = await elections.get(id).catch((err) => {
      toast.error(err instanceof ApiError ? err.message : "Election could not be loaded.");
      return null;
    });
    if (current) setElection({ title: current.title, status: current.status });

    const resultsResult = await elections.results(id).catch((err) => {
      if (err instanceof ApiError && err.status === 403) {
        restricted.push(err.message);
      } else {
        toast.error(err instanceof ApiError ? err.message : "Results could not be loaded.");
      }
      return null;
    });
    setResults(resultsResult);

    const historyResult = await elections.resultsHistory(id).catch(() => null);
    setHistory(historyResult);

    if (restricted.length) toast.warn(restricted.join(" "));
  }, [id, toast]);

  useEffect(() => {
    load();
  }, [load]);

  async function act(action: () => Promise<unknown>, message: string) {
    setBusy(true);
    try {
      await action();
      toast.success(message);
      await load();
      return true;
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "That did not complete.");
      return false;
    } finally {
      setBusy(false);
    }
  }

  async function runTally() {
    const ok = await act(
      () => elections.runTally(id, tallyReason.trim() || undefined),
      "Tally recorded."
    );
    if (ok) {
      setTallyDialog(false);
      setTallyReason("");
    }
  }

  async function runRecount() {
    if (!recountFor) return;
    const snapshotID = recountFor.id;
    setBusy(true);
    try {
      const result = await elections.recount(id, snapshotID, recountReason.trim());
      setRecountResult(result.differences);
      toast.success(
        result.identical
          ? "Recount reproduced the stored result exactly. Ballots were not changed."
          : "Recount found differences. Ballots were not changed."
      );
      await load();
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "The recount did not complete.");
    } finally {
      setBusy(false);
      setRecountFor(null);
      setRecountReason("");
    }
  }

  async function runCertify() {
    if (!certifyFor) return;
    const snapshotID = certifyFor.id;
    const ok = await act(
      () => elections.certify(id, snapshotID, certifyReason.trim()),
      "Results certified."
    );
    if (ok) {
      setCertifyFor(null);
      setCertifyReason("");
    }
  }

  const snapshot = results?.snapshot;
  // In the three-stage lifecycle only CLOSED produces a tally.
  const closedOrLater = election?.status === "CLOSED";

  // The ledger is exactly what this workspace publishes; the information sheet
  // is the public notice. Both are generated by the API.
  async function download(kind: "results" | "info") {
    setDownloading(kind);
    try {
      if (kind === "results") await documents.resultsPdf(id);
      else await documents.infoPdf(id);
    } catch (err) {
      toast.error(
        err instanceof ApiError ? err.message : "The document could not be downloaded."
      );
    } finally {
      setDownloading("");
    }
  }

  return (
    <StaffShell title="Results workspace">
      <nav className="crumbs" aria-label="Breadcrumb" style={{ marginBottom: 16 }}>
        <Link href="/staff">Staff</Link>
        <span>/</span>
        <Link href={`/staff/elections/${id}`}>Election</Link>
        <span>/</span>
        <span>Results</span>
      </nav>

      <div className="page-head">
        <div>
          <span className="lbl">Results</span>
          <h1>{election?.title ?? "Results"}</h1>
          <p>
            A tally is reproducible from stored ballots. Certification pins one snapshot; a
            correction is a new record that supersedes it.
          </p>
        </div>
        <div className="row" style={{ gap: 10, flexWrap: "wrap" }}>
          {snapshot ? (
            <ResultStatusBadge status={snapshot.status} version={snapshot.version} />
          ) : null}
          <button
            type="button"
            className="btn"
            disabled={downloading !== "" || !snapshot}
            onClick={() => download("results")}
          >
            <Icon name="dl" size="s" />
            {downloading === "results" ? "Preparing…" : "Download PDF"}
          </button>
          <button
            type="button"
            className="btn"
            disabled={downloading !== ""}
            onClick={() => download("info")}
          >
            <Icon name="file" size="s" />
            {downloading === "info" ? "Preparing…" : "Information sheet"}
          </button>
        </div>
      </div>

      <div className="stack">
        <section className="panel">
          <div className="section-h" style={{ marginTop: 0 }}>
            <h2>Tally</h2>
            <span className="mono">{election?.status ?? "—"}</span>
          </div>
          {!closedOrLater ? (
            <p className="hint" style={{ marginTop: 0 }}>
              A tally can only be run once voting has closed. This election is currently{" "}
              <b>{election?.status ?? "unknown"}</b>.
            </p>
          ) : (
            <>
              <p className="hint" style={{ marginTop: 0 }}>
                Counting stored ballots against the ballot definition. Nothing here changes a
                ballot.
              </p>
              {snapshot ? (
                <p>
                  Current snapshot <b>v{snapshot.version}</b>, computed{" "}
                  {formatEAT(snapshot.computed_at)}, hash{" "}
                  <code className="mono">{snapshot.content_hash.slice(0, 16)}…</code>
                </p>
              ) : (
                <p className="hint">No tally has been run for this election yet.</p>
              )}
              <div className="row" style={{ gap: 10, flexWrap: "wrap" }}>
                <button
                  type="button"
                  className="btn primary"
                  disabled={busy}
                  onClick={() => setTallyDialog(true)}
                >
                  {snapshot?.status === "CERTIFIED" ? "Run correction tally" : "Run tally"}
                </button>
                <Link className="btn" href={`/staff/elections/${id}/results/history`}>
                  Result history
                </Link>
              </div>
            </>
          )}
        </section>

        {snapshot ? (
          <>
            <TurnoutSummary turnout={snapshot.turnout} spoiled={snapshot.ballots_spoiled} />
            {snapshot.positions.map((position) => (
              <PositionResult key={position.position_id} position={position} />
            ))}

            <section className="panel">
              <div className="section-h" style={{ marginTop: 0 }}>
                <h2>Certification</h2>
                {snapshot.status === "CERTIFIED" ? (
                  <ResultStatusBadge status="CERTIFIED" version={snapshot.version} />
                ) : null}
              </div>
              {snapshot.status === "CERTIFIED" ? (
                <dl className="kv">
                  <dt>Certified</dt>
                  <dd>{formatEAT(snapshot.certified_at ?? "")}</dd>
                  <dt>Reason</dt>
                  <dd>{snapshot.certify_reason || "—"}</dd>
                  <dt>Hash</dt>
                  <dd>
                    <code className="mono">{snapshot.content_hash}</code>
                  </dd>
                </dl>
              ) : (
                <>
                  <p className="hint" style={{ marginTop: 0 }}>
                    Certification pins this snapshot&apos;s hash and is irreversible. The election
                    creator cannot certify; the change is written to the audit trail. A reason is
                    required.
                  </p>
                  <button
                    type="button"
                    className="btn primary"
                    disabled={busy || !closedOrLater}
                    onClick={() => setCertifyFor(snapshot)}
                  >
                    <Icon name="shield" size="s" />
                    Certify snapshot v{snapshot.version}
                  </button>
                </>
              )}
            </section>

            <section className="panel">
              <div className="section-h" style={{ marginTop: 0 }}>
                <h2>Recount</h2>
              </div>
              <p className="hint" style={{ marginTop: 0 }}>
                Re-runs the count from stored ballots and compares it to a snapshot. Ballots are
                never altered by a recount.
              </p>
              <button
                type="button"
                className="btn"
                disabled={busy || !closedOrLater}
                onClick={() => setRecountFor(snapshot)}
              >
                Recount against v{snapshot.version}
              </button>

              {recountResult ? (
                <div style={{ marginTop: 16 }}>
                  {recountResult.position_diffs?.length ? (
                    <div className="tbl-wrap" style={{ marginTop: 12 }}>
                      <table className="tbl">
                        <caption className="sr">
                          Differences between the recount and the snapshot
                        </caption>
                        <thead>
                          <tr>
                            <th scope="col">Position</th>
                            <th scope="col">Outcome before</th>
                            <th scope="col">Outcome after</th>
                            <th scope="col">Candidate movement</th>
                          </tr>
                        </thead>
                        <tbody>
                          {recountResult.position_diffs.map((diff) => (
                            <tr key={diff.title}>
                              <th scope="row">{diff.title}</th>
                              <td>{diff.outcome_before.replaceAll("_", " ").toLowerCase()}</td>
                              <td>{diff.outcome_after.replaceAll("_", " ").toLowerCase()}</td>
                              <td>
                                {diff.candidates?.length
                                  ? diff.candidates
                                      .map(
                                        (c) =>
                                          `${c.name} ${c.before} → ${c.after} (${c.delta > 0 ? "+" : ""}${c.delta})`
                                      )
                                      .join("; ")
                                  : "—"}
                              </td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>
                  ) : null}
                </div>
              ) : null}
            </section>
          </>
        ) : null}

        {history && history.snapshots.length > 1 ? (
          <section className="panel">
            <div className="section-h" style={{ marginTop: 0 }}>
              <h2>Snapshots</h2>
              <span className="mono">{formatNumber(history.snapshots.length)}</span>
            </div>
            <div>
              {history.snapshots.map((item) => (
                <div className="snapshot-row" key={item.id}>
                  <div className="meta">
                    <b>v{item.version}</b>
                    <ResultStatusBadge status={item.status} />
                    <span className="hint">{formatEAT(item.certified_at ?? item.computed_at)}</span>
                  </div>
                  <code className="hash">{item.content_hash}</code>
                </div>
              ))}
            </div>
          </section>
        ) : null}
      </div>

      <ConfirmDialog
        open={tallyDialog}
        title={snapshot?.status === "CERTIFIED" ? "Run a correction tally?" : "Run a tally?"}
        confirmLabel="Run tally"
        busy={busy}
        onClose={() => setTallyDialog(false)}
        onConfirm={runTally}
      >
        <p>
          {snapshot?.status === "CERTIFIED"
            ? "A certified result already exists. A correction is a new snapshot that supersedes it; the original stays visible in the history. A reason is required."
            : "This counts stored ballots against the ballot definition and records a new provisional snapshot."}
        </p>
        <div className="field">
          <label htmlFor="tally-reason">
            Reason {snapshot?.status === "CERTIFIED" ? "" : "(optional)"}
          </label>
          <textarea
            id="tally-reason"
            className="ta"
            value={tallyReason}
            onChange={(event) => setTallyReason(event.target.value)}
          />
        </div>
      </ConfirmDialog>

      <ConfirmDialog
        open={recountFor !== null}
        title="Run a recount?"
        confirmLabel="Run recount"
        busy={busy}
        onClose={() => setRecountFor(null)}
        onConfirm={runRecount}
      >
        <p>
          Re-runs the count from stored ballots and compares it to snapshot v{recountFor?.version}.
          Ballots are never changed by a recount.
        </p>
        <div className="field">
          <label htmlFor="recount-reason">Reason</label>
          <textarea
            id="recount-reason"
            className="ta"
            value={recountReason}
            onChange={(event) => setRecountReason(event.target.value)}
          />
        </div>
      </ConfirmDialog>

      <ConfirmDialog
        open={certifyFor !== null}
        title="Certify these results?"
        confirmLabel="Certify"
        busy={busy}
        onClose={() => setCertifyFor(null)}
        onConfirm={runCertify}
      >
        <p>
          Certification is irreversible. Snapshot v{certifyFor?.version} with hash{" "}
          <code className="mono">{certifyFor?.content_hash.slice(0, 24)}…</code> will be pinned to
          this election. The election creator cannot certify, and a reason is required.
        </p>
        <div className="field">
          <label htmlFor="certify-reason">Reason</label>
          <textarea
            id="certify-reason"
            className="ta"
            value={certifyReason}
            onChange={(event) => setCertifyReason(event.target.value)}
          />
        </div>
      </ConfirmDialog>
    </StaffShell>
  );
}

export default function ResultsWorkspacePage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  return (
    <RequireAuth adminOnly>
      <Workspace id={id} />
    </RequireAuth>
  );
}
