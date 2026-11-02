"use client";

import Link from "next/link";
import { useEffect, useMemo, useState } from "react";
import { useAuth } from "@/components/auth-provider";
import { AppShell } from "@/components/app-shell";
import { Icon } from "@/components/icon";
import { LoadError } from "@/components/load-error";
import { RequireAuth } from "@/components/require-auth";
import { StatusChip } from "@/components/status-chip";
import { useToast } from "@/components/toast";
import { ApiError, elections } from "@/lib/api";
import { categoryLabel, formatShortEAT, statusLabel, timezoneClock } from "@/lib/format";
import type { Election } from "@/lib/types";

type Filter = "ALL" | "OPEN" | "CLOSED";

function ElectionsView() {
  const { user } = useAuth();
  const toast = useToast();
  const [items, setItems] = useState<Election[] | null>(null);
  const [filter, setFilter] = useState<Filter>("ALL");
  const [loadError, setLoadError] = useState<string | null>(null);

  useEffect(() => {
    const controller = new AbortController();
    elections
      .list()
      .then(setItems)
      .catch((err) => {
        if (!controller.signal.aborted) {
          const message = err instanceof ApiError ? err.message : "We could not load elections.";
          setLoadError(message);
          toast.error(message);
        }
      });
    return () => controller.abort();
  }, [toast]);

  // An election the voter has already voted in is finished with them: it moves
  // to the Results tab and is no longer offered here.
  const visible = useMemo(() => (items ?? []).filter((e) => !e.my_vote_status?.voted), [items]);

  const filtered = useMemo(() => {
    if (filter === "ALL") return visible;
    return visible.filter((e) => e.status === filter);
  }, [visible, filter]);

  const open = visible.filter((e) => e.accepting_votes).length;

  return (
    <AppShell title="Elections">
      <div className="page-head">
        <div>
          <span className="lbl">My elections</span>
          <h1>Guild elections</h1>
          <p>
            {user ? `Signed in as ${user.first_name} ${user.last_name}. ` : ""}
            Eligibility is checked on the server for every ballot.
          </p>
        </div>
        {items && items.length > 0 ? (
          <div className="row">
            <span className="chip open">{open} open now</span>
            <Link className="btn sm" href="/results">
              <Icon name="chart" size="s" />
              My results
            </Link>
            <span className="mono hint">
              server {items[0] ? timezoneClock(items[0].server_time) : "—"} EAT
            </span>
          </div>
        ) : null}
      </div>

      <div className="row" style={{ marginBottom: 20 }}>
        {(["ALL", "OPEN", "CLOSED"] as Filter[]).map((value) => (
          <button
            key={value}
            type="button"
            className={`btn sm${filter === value ? " primary" : " ghost"}`}
            aria-pressed={filter === value}
            onClick={() => setFilter(value)}
          >
            {value === "ALL" ? "All" : statusLabel(value as Election["status"])}
          </button>
        ))}
      </div>

      {loadError ? (
        <LoadError title="We could not load your elections" message={loadError} />
      ) : !items ? (
        <div className="grid3" role="status" aria-busy="true" aria-label="Loading elections">
          {[0, 1, 2].map((key) => (
            <div className="card" key={key}>
              <div className="cb stack">
                <span className="skel" style={{ width: "35%", height: 12 }} />
                <span className="skel" style={{ height: 18 }} />
                <span className="skel" style={{ height: 12, width: "75%" }} />
                <span className="skel" style={{ height: 36 }} />
              </div>
            </div>
          ))}
        </div>
      ) : null}

      {items && filtered.length === 0 ? (
        <div className="empty">
          <div className="ic" aria-hidden="true">
            <Icon name="ballot" size="l" />
          </div>
          <b>{visible.length === 0 && items.length > 0 ? "You have voted" : "No elections here"}</b>
          <span className="hint">
            {items.length === 0
              ? "No election has been published yet. Check back when the Electoral Commission opens one."
              : visible.length === 0
                ? "You have voted in every election you can see. Your ballots and results are in the Results tab."
                : "No election matches this filter."}
          </span>
          {visible.length === 0 && items.length > 0 ? (
            <Link className="btn primary" href="/results">
              <Icon name="chart" size="s" />
              Go to my results
            </Link>
          ) : null}
        </div>
      ) : null}

      {filtered.length > 0 ? (
        <div className="grid3">
          {filtered.map((election) => (
            <article className="card election-card" key={election.id}>
              <div className="cb">
                <div className="row-between">
                  <StatusChip status={election.status} />
                  <span className="tag">{categoryLabel(election.category)}</span>
                </div>
                <b style={{ fontSize: 18, lineHeight: 1.3 }}>{election.title}</b>
                <span className="hint">
                  {election.accepting_votes
                    ? `Closes ${formatShortEAT(election.ends_at)}`
                    : election.status === "CLOSED"
                      ? `Voting closed ${formatShortEAT(election.ends_at)}`
                      : `Voting opens ${formatShortEAT(election.starts_at)}`}
                </span>
                <span className="hint">{election.organizing_unit}</span>
                <div style={{ marginTop: "auto", paddingTop: 8 }}>
                  <Link
                    className={`btn block${election.accepting_votes ? " primary" : ""}`}
                    href={`/elections/${election.id}`}
                  >
                    {election.accepting_votes ? "Vote now" : "View election"}
                  </Link>
                </div>
              </div>
            </article>
          ))}
        </div>
      ) : null}
    </AppShell>
  );
}

export default function ElectionsPage() {
  return (
    <RequireAuth>
      <ElectionsView />
    </RequireAuth>
  );
}
