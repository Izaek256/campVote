"use client";

import Link from "next/link";
import { useEffect, useMemo, useState } from "react";
import { useAuth } from "@/components/auth-provider";
import { Icon } from "@/components/icon";
import { RequireAuth } from "@/components/require-auth";
import { SiteHeader } from "@/components/site-header";
import { StatusChip } from "@/components/status-chip";
import { ApiError, elections } from "@/lib/api";
import { categoryLabel, formatShortEAT, statusLabel, timezoneClock } from "@/lib/format";
import type { Election } from "@/lib/types";

type Filter = "ALL" | "OPEN" | "SCHEDULED" | "CLOSED";

function ElectionsView() {
  const { user } = useAuth();
  const [items, setItems] = useState<Election[] | null>(null);
  const [error, setError] = useState("");
  const [filter, setFilter] = useState<Filter>("ALL");

  useEffect(() => {
    const controller = new AbortController();
    elections
      .list()
      .then(setItems)
      .catch((err) => {
        if (!controller.signal.aborted) {
          setError(err instanceof ApiError ? err.message : "We could not load elections.");
        }
      });
    return () => controller.abort();
  }, []);

  const filtered = useMemo(() => {
    if (!items) return [];
    if (filter === "ALL") return items;
    if (filter === "CLOSED") {
      return items.filter((e) =>
        ["CLOSED", "RESULTS_UNDER_REVIEW", "FINALIZED", "ARCHIVED"].includes(e.status)
      );
    }
    return items.filter((e) => e.status === filter);
  }, [items, filter]);

  const open = items?.filter((e) => e.accepting_votes).length ?? 0;

  return (
    <>
      <SiteHeader />
      <main className="app-wrap">
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
              <span className="mono hint">
                server {items[0] ? timezoneClock(items[0].server_time) : "—"} EAT
              </span>
            </div>
          ) : null}
        </div>

        {error ? (
          <div className="banner bad" role="alert">
            <Icon name="alert" />
            <div>
              <b>We could not load elections</b>
              {error}
            </div>
          </div>
        ) : null}

        <div className="row" style={{ marginBottom: 20 }}>
          {(["ALL", "OPEN", "SCHEDULED", "CLOSED"] as Filter[]).map((value) => (
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

        {!items && !error ? (
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
            <b>No elections here</b>
            <span className="hint">
              {items.length === 0
                ? "No election has been published yet. Check back when the Electoral Commission opens one."
                : "No election matches this filter."}
            </span>
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
                      : `Voting ${election.status === "SCHEDULED" ? "opens" : "opened"} ${formatShortEAT(election.starts_at)}`}
                  </span>
                  <span className="hint">{election.organizing_unit}</span>
                  <div style={{ marginTop: "auto", paddingTop: 8 }}>
                    <Link className="btn primary block" href={`/elections/${election.id}`}>
                      {election.accepting_votes ? "Vote now" : "View election"}
                    </Link>
                  </div>
                </div>
              </article>
            ))}
          </div>
        ) : null}
      </main>
    </>
  );
}

export default function ElectionsPage() {
  return (
    <RequireAuth>
      <ElectionsView />
    </RequireAuth>
  );
}
