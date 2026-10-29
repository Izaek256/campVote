"use client";

import { useState, useEffect } from "react";
import { Candidate } from "@campvote/shared/types/candidate";

export default function MyCandidacyPage() {
  const [candidates, setCandidates] = useState<Candidate[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchMyCandidacies();
  }, []);

  const fetchMyCandidacies = async () => {
    try {
      const response = await fetch("/api/v1/protected/candidates/my");
      if (!response.ok) throw new Error("Failed to fetch candidacies");
      const data = await response.json();
      setCandidates(data.candidates || []);
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  const handleWithdraw = async (candidateId: string, electionId: string) => {
    const reason = prompt("Reason for withdrawal:");
    if (!reason) return;

    try {
      const response = await fetch(
        `/api/v1/staff/elections/${electionId}/candidates/${candidateId}/withdraw`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ reason }),
        }
      );
      if (response.ok) fetchMyCandidacies();
    } catch (err) {
      console.error(err);
    }
  };

  if (loading) return <div className="p-8">Loading...</div>;

  return (
    <div className="p-8 max-w-4xl mx-auto">
      <h1 className="text-2xl font-bold mb-6">My Candidacies</h1>

      {candidates.length === 0 ? (
        <div className="text-center py-12 text-gray-500">
          You have no active candidacies.
        </div>
      ) : (
        <div className="space-y-4">
          {candidates.map((candidate) => (
            <div key={candidate.id} className="border rounded-lg p-6">
              <div className="flex justify-between items-start">
                <div className="flex gap-4">
                  {candidate.photo_url && (
                    <img
                      src={candidate.photo_url}
                      alt={candidate.full_name}
                      className="w-20 h-20 rounded-full object-cover"
                    />
                  )}
                  <div>
                    <h3 className="font-semibold text-lg">{candidate.full_name}</h3>
                    <p className="text-sm text-gray-500 mt-1">
                      Status: <span className={`inline-block px-2 py-1 rounded text-xs ${
                        candidate.approval_status === "APPROVED" ? "bg-green-100 text-green-800" :
                        candidate.approval_status === "REJECTED" ? "bg-red-100 text-red-800" :
                        candidate.approval_status === "WITHDRAWN" ? "bg-gray-100 text-gray-800" :
                        "bg-yellow-100 text-yellow-800"
                      }`}>{candidate.approval_status}</span>
                    </p>
                    {candidate.reviewed_at && (
                      <p className="text-sm text-gray-500">
                        Reviewed: {new Date(candidate.reviewed_at).toLocaleDateString()}
                      </p>
                    )}
                    {candidate.review_notes && (
                      <p className="text-sm text-gray-600 mt-2 italic">"{candidate.review_notes}"</p>
                    )}
                  </div>
                </div>
                {candidate.approval_status === "APPROVED" && (
                  <button
                    onClick={() => handleWithdraw(candidate.id, candidate.election_id)}
                    className="px-4 py-2 text-sm bg-red-600 text-white rounded hover:bg-red-700"
                  >
                    Withdraw
                  </button>
                )}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
