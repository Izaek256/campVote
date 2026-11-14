"use client";

import Link from "next/link";
import { useEffect, useMemo, useState } from "react";
import { AppShell } from "@/components/app-shell";
import { EmptyState } from "@/components/empty-state";
import { Icon } from "@/components/icon";
import { LoadError } from "@/components/load-error";
import { RequireAuth } from "@/components/require-auth";
import { StatusChip } from "@/components/status-chip";
import { useToast } from "@/components/toast";
import { ApiError, elections } from "@/lib/api";
import { formatShortEAT } from "@/lib/format";
import type { Election } from "@/lib/types";

// In the three-stage lifecycle a closed election is one the count can already
// run for. The results policy still decides whether the figures are published
// to a voter.
function isClosed(status: Election["status"]): boolean {
  return status === "CLOSED";
}

function ResultsView() {
  const toast = useToast();
  const [items, setItems] = useState<Election[] | null>(null);
  const [loadError, setLoadError] = useState<string | null>(null);

  useEffect(() => {
    const controller = new AbortController();
    elections
      .list()
      .then((data) => {
        if (!controller.signal.aborted) setItems(data);
      })
      .catch((err) => {
        if (!controller.signal.aborted) {
          const message = err instanceof ApiError ? err.message : "We could not load your results.";
          setLoadError(message);
          toast.error(message);
        }
      });
    return () => controller.abort();
  }, [toast]);

  // A student waits on the elections they voted in that have not closed yet.
  const pending = useMemo(
    () =>
      (items ?? []).filter(
        (election) => election.my_vote_status?.voted && !isClosed(election.status)
      ),
    [items]
  );
  // Results are public record: every closed election appears here so the whole
  // campus can read the outcome, whether or not a student was eligible to vote
  // in it. The API only returns elections this caller may see.
  const published = useMemo(
    () => (items ?? []).filter((election) => isClosed(election.status)),
    [items]
  );

  return (
    <AppShell title="Results">
      <div className="page-head">
        <div>
          <span className="lbl">Results</span>
          <h1>My results</h1>
          <p>
            Elections you voted in stay pending until they close. Every closed election is listed
            under published results, so you can see what came out of the elections held on campus.
          </p>
        </div>
        <Link className="btn" href="/elections">
          <Icon name="ballot" size="s" />
          Find an election
        </Link>
      </div>

      {loadError ? (
        <LoadError
          title="We could not load the results"
          message={loadError}
          backHref="/elections"
          backLabel="Back to elections"
        />
      ) : !items ? (
        <div className="panel stack" role="status" aria-busy="true" aria-label="Loading results">
          <span className="skel" style={{ height: 18, width: "30%" }} />
          <span className="skel" style={{ height: 14 }} />
        </div>
      ) : null}

      {items && published.length === 0 && pending.length === 0 ? (
        <EmptyState
          title="Nothing to show yet"
          action={
            <Link className="btn primary" href="/elections">
              Browse elections
            </Link>
          }
        >
          No election has closed yet. Once a count is published it will appear here for the whole
          campus to read.
        </EmptyState>
      ) : null}

      {pending.length > 0 ? (
        <section style={{ marginBottom: 24 }}>
          <div className="section-h">
            <h2>Awaiting results</h2>
            <span className="mono">{pending.length}</span>
          </div>
          <div className="grid3">
            {pending.map((election) => (
              <article className="card" key={election.id}>
                <div className="cb">
                  <div className="row-between">
                    <StatusChip status={election.status} />
                    <span className="chip draft">
                      <Icon name="clock" size="s" /> Vote recorded
                    </span>
                  </div>
                  <b style={{ fontSize: 18, lineHeight: 1.3 }}>{election.title}</b>
                  <span className="hint">
                    Closes {formatShortEAT(election.ends_at)} — the result is published after the
                    count.
                  </span>
                </div>
              </article>
            ))}
          </div>
        </section>
      ) : null}

      {published.length > 0 ? (
        <section>
          <div className="section-h">
            <h2>Published results</h2>
            <span className="mono">{published.length}</span>
          </div>
          <div className="grid3">
            {published.map((election) => (
              <article className="card" key={election.id}>
                <div className="cb">
                  <div className="row-between">
                    <StatusChip status={election.status} />
                    {election.my_vote_status?.voted ? (
                      <span className="chip draft">
                        <Icon name="check" size="s" /> You voted
                      </span>
                    ) : null}
                  </div>
                  <b style={{ fontSize: 18, lineHeight: 1.3 }}>{election.title}</b>
                  <span className="hint">Closed {formatShortEAT(election.ends_at)}</span>
                  <div className="stack" style={{ marginTop: "auto", paddingTop: 8 }}>
                    <Link className="btn block primary" href={`/elections/${election.id}/results`}>
                      View results
                    </Link>
                  </div>
                </div>
              </article>
            ))}
          </div>
        </section>
      ) : null}
    </AppShell>
  );
}

export default function ResultsPage() {
  return (
    <RequireAuth>
      <ResultsView />
    </RequireAuth>
  );
}
