"use client";

import { useCallback, useEffect, useState } from "react";
import { useParams } from "next/navigation";
import { Icon } from "@/components/icon";
import { RequireAuth } from "@/components/require-auth";
import { StaffShell } from "@/components/staff-shell";
import { StatusChip } from "@/components/status-chip";
import { ApiError, elections } from "@/lib/api";
import { formatEAT, formatNumber, formatPercent } from "@/lib/format";
import type { Election, PositionResult, Results } from "@/lib/types";

export default function ElectionMonitorPage() {
  const params = useParams();
  // params.id may be string | string[] | undefined in Next.js 13+
  const electionId = String(params.id ?? "");
  const [election, setElection] = useState<Election | null>(null);
  const [results, setResults] = useState<Results | null>(null);
  const [loading, setLoading] = useState(true);
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null);
  const [autoRefresh, setAutoRefresh] = useState(true);

  const loadData = useCallback(async () => {
    try {
      const [currentElection, currentResults] = await Promise.all([
        elections.get(electionId),
        elections.results(electionId).catch(() => null),
      ]);
      setElection(currentElection);
      setResults(currentResults);
      setLastUpdated(new Date());
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  }, [electionId]);

  useEffect(() => {
    loadData();
  }, [loadData]);

  useEffect(() => {
    if (!autoRefresh) return;

    const interval = setInterval(() => {
      loadData();
    }, 30000); // Refresh every 30 seconds

    return () => clearInterval(interval);
  }, [autoRefresh, loadData]);

  if (loading) return <div className="p-8">Loading...</div>;

  // Positions come from the snapshot, not directly from results
  const snapshotPositions: PositionResult[] = results?.snapshot?.positions ?? [];
  const isOpen = election?.status === "OPEN";

  return (
    <StaffShell>
      <div className="p-4 md:p-8">
        <div className="flex justify-between items-center mb-6">
          <div>
            <h1 className="text-xl md:text-2xl font-bold">Election Monitor</h1>
            {election && <p className="text-gray-600">{election.title}</p>}
          </div>
          <div className="flex items-center gap-4">
            <div className="flex items-center gap-2">
              <span className="text-sm text-gray-500">Auto-refresh:</span>
              <button
                onClick={() => setAutoRefresh(!autoRefresh)}
                className={`px-3 py-1 rounded text-sm ${
                  autoRefresh ? "bg-green-100 text-green-800" : "bg-gray-100"
                }`}
              >
                {autoRefresh ? "On" : "Off"}
              </button>
            </div>
            <button
              onClick={loadData}
              className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 text-sm"
            >
              Refresh Now
            </button>
          </div>
        </div>

        {lastUpdated && (
          <p className="text-sm text-gray-500 mb-6">
            Last updated: {lastUpdated.toLocaleTimeString()}
          </p>
        )}

        {election && (
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-8">
            <div className="bg-white border rounded-lg p-4">
              <h3 className="text-sm font-medium text-gray-500 mb-2">Status</h3>
              <StatusChip status={election.status} />
            </div>
            <div className="bg-white border rounded-lg p-4">
              <h3 className="text-sm font-medium text-gray-500 mb-2">Schedule</h3>
              <p className="text-sm">
                {formatEAT(election.starts_at)} - {formatEAT(election.ends_at)}
              </p>
            </div>
            <div className="bg-white border rounded-lg p-4">
              <h3 className="text-sm font-medium text-gray-500 mb-2">Turnout</h3>
              <p className="text-sm font-semibold">
                {results?.snapshot
                  ? `${formatNumber(results.snapshot.ballots_cast)} / ${formatNumber(results.snapshot.eligible_voters)} (${formatPercent(results.snapshot.turnout_percent)})`
                  : "—"}
              </p>
            </div>
          </div>
        )}

        {/* Participation per position — never show per-candidate counts while OPEN */}
        {snapshotPositions.length > 0 ? (
          <div className="space-y-6">
            <h2 className="text-lg font-semibold">
              {isOpen ? "Participation by Position" : "Results by Position"}
            </h2>
            {snapshotPositions.map((position: PositionResult) => (
              <div key={position.position_id} className="bg-white border rounded-lg p-4">
                <h3 className="font-semibold mb-4">{position.title}</h3>
                <p className="text-sm text-gray-600 mb-2">
                  Ballots answered: {formatNumber(position.ballots_answered)}
                  {" · "}Blank: {formatNumber(position.blank_count)}
                </p>
                {/* Only show per-candidate breakdown when election is NOT open */}
                {!isOpen && position.rows.map((row) => (
                  <div key={row.candidate_id} className="space-y-2 mt-2">
                    <div className="flex justify-between items-center">
                      <div className="flex items-center gap-2">
                        {row.is_winner && (
                          <span className="text-xs bg-green-100 text-green-800 px-2 py-0.5 rounded">
                            Winner
                          </span>
                        )}
                        <p className="font-medium">{row.name}</p>
                      </div>
                      <div className="text-right">
                        <p className="font-semibold">{formatPercent(row.share_percent)}</p>
                        <p className="text-sm text-gray-500">{formatNumber(row.votes)} votes</p>
                      </div>
                    </div>
                    <div className="w-full bg-gray-200 rounded-full h-2">
                      <div
                        className="bg-green-600 h-2 rounded-full transition-all duration-500"
                        style={{ width: `${row.share_percent}%` }}
                      />
                    </div>
                  </div>
                ))}
                {isOpen && (
                  <p className="text-sm text-amber-600 mt-2">
                    ⚠️ Candidate vote counts are hidden while the election is open.
                  </p>
                )}
              </div>
            ))}
          </div>
        ) : (
          <div className="bg-white border rounded-lg p-8 text-center text-gray-500">
            <p>No results available yet.</p>
            <p className="text-sm mt-2">
              {isOpen
                ? "Results will be published automatically when the election closes."
                : "Results will appear here once ballots are cast."}
            </p>
          </div>
        )}
      </div>
    </StaffShell>
  );
}
