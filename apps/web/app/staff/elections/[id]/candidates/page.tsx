"use client";

import { useState, useEffect } from "react";
import { useParams } from "next/navigation";
import { Candidate } from "@campvote/shared/types/candidate";

type StatusFilter = "ALL" | "NOMINATED" | "APPROVED" | "REJECTED" | "WITHDRAWN";

export default function CandidatesPage() {
  const params = useParams();
  const electionId = params.id;
  const [candidates, setCandidates] = useState<Candidate[]>([]);
  const [loading, setLoading] = useState(true);
  const [filter, setFilter] = useState<StatusFilter>("ALL");

  useEffect(() => {
    fetchCandidates();
  }, [electionId, filter]);

  const fetchCandidates = async () => {
    try {
      const url = filter === "ALL"
        ? `/api/v1/staff/elections/${electionId}/candidates`
        : `/api/v1/staff/elections/${electionId}/candidates?status=${filter}`;
      const response = await fetch(url);
      if (!response.ok) throw new Error("Failed to fetch candidates");
      const data = await response.json();
      setCandidates(data.candidates || []);
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  const handleApprove = async (candidateId: string) => {
    const reason = prompt("Reason for approval:");
    if (!reason) return;

    try {
      const response = await fetch(
        `/api/v1/staff/elections/${electionId}/candidates/${candidateId}/approve`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ reason }),
        }
      );
      if (response.ok) fetchCandidates();
    } catch (err) {
      console.error(err);
    }
  };

  const handleReject = async (candidateId: string) => {
    const reason = prompt("Reason for rejection:");
    if (!reason) return;

    try {
      const response = await fetch(
        `/api/v1/staff/elections/${electionId}/candidates/${candidateId}/reject`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ reason }),
        }
      );
      if (response.ok) fetchCandidates();
    } catch (err) {
      console.error(err);
    }
  };

  if (loading) return <div className="p-8">Loading...</div>;

  return (
    <div className="p-8">
      <h1 className="text-2xl font-bold mb-6">Candidate Nominations</h1>

      <div className="flex gap-2 mb-6">
        {(["ALL", "NOMINATED", "APPROVED", "REJECTED", "WITHDRAWN"] as StatusFilter[]).map((status) => (
          <button
            key={status}
            onClick={() => setFilter(status)}
            className={`px-4 py-2 rounded ${
              filter === status
                ? "bg-green-600 text-white"
                : "bg-gray-100 hover:bg-gray-200"
            }`}
          >
            {status}
          </button>
        ))}
      </div>

      {candidates.length === 0 ? (
        <div className="text-center py-12 text-gray-500">
          No candidates found for this filter.
        </div>
      ) : (
        <div className="space-y-4">
          {candidates.map((candidate) => (
            <div key={candidate.id} className="border rounded-lg p-4">
              <div className="flex justify-between items-start">
                <div className="flex gap-4">
                  {candidate.photo_url && (
                    <img
                      src={candidate.photo_url}
                      alt={candidate.full_name}
                      className="w-16 h-16 rounded-full object-cover"
                    />
                  )}
                  <div>
                    <h3 className="font-semibold">{candidate.full_name}</h3>
                    <p className="text-sm text-gray-500">
                      Status: <span className={`inline-block px-2 py-1 rounded text-xs ${
                        candidate.approval_status === "APPROVED" ? "bg-green-100 text-green-800" :
                        candidate.approval_status === "REJECTED" ? "bg-red-100 text-red-800" :
                        candidate.approval_status === "WITHDRAWN" ? "bg-gray-100 text-gray-800" :
                        "bg-yellow-100 text-yellow-800"
                      }`}>{candidate.approval_status}</span>
                    </p>
                    {candidate.manifesto && (
                      <p className="text-sm text-gray-600 mt-1 line-clamp-2">{candidate.manifesto}</p>
                    )}
                    {candidate.review_notes && (
                      <p className="text-sm text-gray-500 mt-1 italic">Note: {candidate.review_notes}</p>
                    )}
                  </div>
                </div>
                <div className="flex gap-2">
                  {candidate.approval_status === "NOMINATED" && (
                    <>
                      <button
                        onClick={() => handleApprove(candidate.id)}
                        className="px-3 py-1 text-sm bg-green-600 text-white rounded hover:bg-green-700"
                      >
                        Approve
                      </button>
                      <button
                        onClick={() => handleReject(candidate.id)}
                        className="px-3 py-1 text-sm bg-red-600 text-white rounded hover:bg-red-700"
                      >
                        Reject
                      </button>
                    </>
                  )}
                  <button className="px-3 py-1 text-sm border rounded hover:bg-gray-50">
                    View Details
                  </button>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
