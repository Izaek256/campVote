"use client";

import { use, useCallback, useEffect, useState } from "react";
import { Icon } from "@/components/icon";
import { RequireAuth } from "@/components/require-auth";
import { StaffShell } from "@/components/staff-shell";
import { StatusChip } from "@/components/status-chip";
import { useToast } from "@/components/toast";
import { useChannel } from "@/hooks/use-realtime";
import { ApiError, elections, staff } from "@/lib/api";
import { formatEAT, formatNumber, formatPercent } from "@/lib/format";
import type { Election, Results } from "@/lib/types";

type Turnout = Awaited<ReturnType<typeof staff.turnout>>;

function Monitor({ id }: { id: string }) {
  const toast = useToast();
  const [election, setElection] = useState<Election | null>(null);
  const [turnout, setTurnout] = useState<Turnout | null>(null);
  const [results, setResults] = useState<Results | null>(null);
  const [loading, setLoading] = useState(true);
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null);
  const [autoRefresh, setAutoRefresh] = useState(true);

  const load = useCallback(async () => {
    try {
      const [current, currentTurnout, currentResults] = await Promise.all([
        elections.get(id),
        staff.turnout(id),
        elections.results(id).catch(() => null),
      ]);
      setElection(current);
      setTurnout(currentTurnout);
      setResults(currentResults);
      setLastUpdated(new Date());
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "The monitor could not be loaded.");
    } finally {
      setLoading(false);
    }
  }, [id, toast.error]);

  useEffect(() => {
    load();
  }, [load]);

  // The timer is the fallback; a results snapshot (including the automatic one
  // at close) also wakes the page immediately.
  useEffect(() => {
    if (!autoRefresh) return;
    const timer = setInterval(load, 15000);
    return () => clearInterval(timer);
  }, [autoRefresh, load]);

  useChannel(`public-election-results.${id}`, {
    "results.updated": () => {
      void load();
    },
  });

  const isOpen = election?.status === "OPEN";
  const snapshotPositions = results?.snapshot?.positions ?? [];
  const participation = turnout?.positions ?? [];

  return (
    <StaffShell title="Monitor">
      <div className="page-head">
        <div>
          <span className="lbl">Live participation</span>
          <h1>{election?.title ?? "Election monitor"}</h1>
          <p>
            Turnout only while voting is open. Candidate counts appear after the election closes and
            are never shown on this page early.
          </p>
        </div>
        <div className="row">
          {election ? <StatusChip status={election.status} /> : null}
          <button
            type="button"
            className={`btn sm${autoRefresh ? " primary" : ""}`}
            onClick={() => setAutoRefresh((value) => !value)}
          >
            {autoRefresh ? "Live updates on" : "Live updates off"}
          </button>
          <button type="button" className="btn sm" onClick={load}>
            Refresh now
          </button>
        </div>
      </div>

      {loading ? (
        <div className="panel stack" aria-busy="true">
          <span className="skel" style={{ height: 18, width: "30%" }} />
          <span className="skel" style={{ height: 14 }} />
        </div>
      ) : null}

      {election ? (
        <div className="kpis" style={{ marginBottom: 20 }}>
          <div className="kpi">
            <span className="l">Eligible voters</span>
            <span className="v">{turnout ? formatNumber(turnout.eligible) : "—"}</span>
          </div>
          <div className="kpi">
            <span className="l">Ballots cast</span>
            <span className="v">{turnout ? formatNumber(turnout.voted) : "—"}</span>
          </div>
          <div className="kpi">
            <span className="l">Turnout</span>
            <span className="v">{turnout ? formatPercent(turnout.percent) : "—"}</span>
          </div>
          <div className="kpi">
            <span className="l">Window</span>
            <span className="v" style={{ fontSize: 14 }}>
              {formatEAT(election.starts_at)} – {formatEAT(election.ends_at)}
            </span>
          </div>
        </div>
      ) : null}

      {turnout?.suppressed ? (
        <div className="banner hatch" role="status" style={{ marginBottom: 16 }}>
          <Icon name="lock" />
          <div>
            <b>Turnout is hidden for this group</b>
            The electorate is below the configured privacy threshold.
          </div>
        </div>
      ) : null}

      {participation.length > 0 ? (
        <section className="panel" style={{ marginBottom: 16 }}>
          <div className="section-h" style={{ marginTop: 0 }}>
            <h2>Participation by position</h2>
            <span className="mono">no candidate totals while open</span>
          </div>
          <div className="stack">
            {participation.map((position) => (
              <div className="row-between" key={position.position_id}>
                <span>
                  <b>{position.title}</b>
                  <span className="hint" style={{ display: "block" }}>
                    {formatNumber(position.ballots_answered)} ballots answered ·{" "}
                    {formatNumber(position.blank_count)} blank ·{" "}
                    {formatNumber(position.abstain_count)} abstained
                  </span>
                </span>
              </div>
            ))}
          </div>
        </section>
      ) : null}

      {!isOpen && snapshotPositions.length > 0 ? (
        <section className="panel">
          <div className="section-h" style={{ marginTop: 0 }}>
            <h2>Results by position</h2>
            <span className="mono">published after close</span>
          </div>
          <div className="stack">
            {snapshotPositions.map((position) => (
              <div key={position.position_id} className="stack">
                <b>{position.title}</b>
                <div className="tally">
                  {position.rows.map((row) => (
                    <div className="trow" key={row.candidate_id}>
                      <span>
                        {row.name}
                        {row.is_winner ? <span className="lead-tag">Winner</span> : null}
                      </span>
                      <div className={`bar${row.is_winner ? " lead" : ""}`}>
                        <i
                          style={{
                            width: `${Math.min(100, Math.max(0, row.share_percent))}%`,
                          }}
                        />
                      </div>
                      <span className="vl">
                        <b>{formatNumber(row.votes)}</b> · {formatPercent(row.share_percent)}
                      </span>
                    </div>
                  ))}
                </div>
              </div>
            ))}
          </div>
        </section>
      ) : null}

      {!loading && participation.length === 0 ? (
        <div className="panel stack">
          <b>{isOpen ? "No ballots have been cast yet" : "No results yet"}</b>
          <span className="hint">
            {isOpen
              ? "This page updates as ballots arrive. Candidate counts stay hidden until the election closes."
              : "Results are tallied automatically when voting closes."}
          </span>
        </div>
      ) : null}

      {lastUpdated ? (
        <p className="hint" style={{ marginTop: 16 }}>
          Last updated {lastUpdated.toLocaleTimeString()}
        </p>
      ) : null}
    </StaffShell>
  );
}

export default function ElectionMonitorPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  return (
    <RequireAuth adminOnly>
      <Monitor id={id} />
    </RequireAuth>
  );
}
