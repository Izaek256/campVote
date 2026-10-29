"use client";

import { useState, useEffect } from "react";
import { useParams } from "next/navigation";

interface TurnoutData {
  eligible: number;
  voted: number;
  percent: number;
  suppressed?: boolean;
  last_refresh: string;
}

export default function TurnoutPage() {
  const params = useParams();
  const electionId = params.id;
  const [turnout, setTurnout] = useState<TurnoutData | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchTurnout();
  }, [electionId]);

  const fetchTurnout = async () => {
    try {
      const response = await fetch(`/api/v1/staff/elections/${electionId}/turnout`);
      if (!response.ok) throw new Error("Failed to fetch turnout");
      const data = await response.json();
      setTurnout(data);
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  if (loading) return <div className="p-8">Loading...</div>;

  if (!turnout) return <div className="p-8 text-red-600">Failed to load turnout data</div>;

  return (
    <div className="p-8">
      <h1 className="text-2xl font-bold mb-6">Participation & Turnout</h1>

      {turnout.suppressed ? (
        <div className="bg-yellow-50 border border-yellow-200 text-yellow-800 p-6 rounded-lg">
          <h2 className="text-lg font-semibold mb-2">Hidden to Protect Privacy</h2>
          <p className="text-sm">
            Turnout counts for this election are not displayed because the number of eligible voters
            is below the configured privacy threshold. This prevents the identification of individual
            voters through turnout data.
          </p>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-6">
          <div className="bg-white border rounded-lg p-6">
            <p className="text-sm text-gray-500 mb-1">Eligible Voters</p>
            <p className="text-3xl font-bold">{turnout.eligible}</p>
          </div>
          <div className="bg-white border rounded-lg p-6">
            <p className="text-sm text-gray-500 mb-1">Votes Cast</p>
            <p className="text-3xl font-bold">{turnout.voted}</p>
          </div>
          <div className="bg-white border rounded-lg p-6">
            <p className="text-sm text-gray-500 mb-1">Turnout</p>
            <p className="text-3xl font-bold">{turnout.percent.toFixed(1)}%</p>
          </div>
        </div>
      )}

      <div className="text-sm text-gray-500">
        Last refreshed: {new Date(turnout.last_refresh).toLocaleString()}
      </div>

      <div className="mt-6 bg-blue-50 border border-blue-200 text-blue-900 p-4 rounded">
        <p className="font-medium text-sm">Privacy Note</p>
        <p className="text-xs mt-1">
          This panel shows counts only. It does not show a list of who voted or any ballot choices.
          Individual voter participation records are stored separately from ballot selections to protect privacy.
        </p>
      </div>
    </div>
  );
}
