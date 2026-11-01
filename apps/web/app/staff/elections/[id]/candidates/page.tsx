"use client";

import { useCallback, useEffect, useState } from "react";
import { useParams } from "next/navigation";
import type { Candidate } from "@campvote/shared/types/candidate";

type StatusFilter = "ALL" | "NOMINATED" | "APPROVED" | "REJECTED" | "WITHDRAWN";

export default function CandidatesPage() {
  const params = useParams();
  const electionId = params.id;
  const [candidates, setCandidates] = useState<Candidate[]>([]);
  const [loading, setLoading] = useState(true);
  const [filter, setFilter] = useState<StatusFilter>("ALL");
  const [selectedCandidate, setSelectedCandidate] = useState<Candidate | null>(null);
  const [showModal, setShowModal] = useState(false);

  const fetchCandidates = useCallback(async () => {
    try {
      const url =
        filter === "ALL"
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
  }, [electionId, filter]);

  useEffect(() => {
    fetchCandidates();
  }, [fetchCandidates]);

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

  const handleViewDetails = (candidate: Candidate) => {
    setSelectedCandidate(candidate);
    setShowModal(true);
  };

  if (loading) return <div className="p-8">Loading...</div>;

  return (
    <div className="p-4 md:p-8">
      <h1 className="text-xl md:text-2xl font-bold mb-6">Candidate Nominations</h1>

      <div className="flex flex-wrap gap-2 mb-6">
        {(["ALL", "NOMINATED", "APPROVED", "REJECTED", "WITHDRAWN"] as StatusFilter[]).map(
          (status) => (
            <button
              key={status}
              onClick={() => setFilter(status)}
              className={`px-3 py-2 rounded text-sm md:text-base ${
                filter === status ? "bg-green-600 text-white" : "bg-gray-100 hover:bg-gray-200"
              }`}
            >
              {status}
            </button>
          )
        )}
      </div>

      {candidates.length === 0 ? (
        <div className="text-center py-12 text-gray-500">No candidates found for this filter.</div>
      ) : (
        <div className="space-y-4">
          {candidates.map((candidate) => (
            <div key={candidate.id} className="border rounded-lg p-4">
              <div className="flex flex-col md:flex-row justify-between items-start gap-4">
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
                    {candidate.registration_number && (
                      <p className="text-sm text-gray-500">Reg No: {candidate.registration_number}</p>
                    )}
                    <p className="text-sm text-gray-500 mt-1">
                      Status:{" "}
                      <span
                        className={`inline-block px-2 py-1 rounded text-xs ${
                          candidate.approval_status === "APPROVED"
                            ? "bg-green-100 text-green-800"
                            : candidate.approval_status === "REJECTED"
                              ? "bg-red-100 text-red-800"
                              : candidate.approval_status === "WITHDRAWN"
                                ? "bg-gray-100 text-gray-800"
                                : "bg-yellow-100 text-yellow-800"
                        }`}
                      >
                        {candidate.approval_status}
                      </span>
                    </p>
                    {candidate.manifesto && (
                      <p className="text-sm text-gray-600 mt-1 line-clamp-2">
                        {candidate.manifesto}
                      </p>
                    )}
                    {candidate.review_notes && (
                      <p className="text-sm text-gray-500 mt-1 italic">
                        Note: {candidate.review_notes}
                      </p>
                    )}
                  </div>
                </div>
                <div className="flex gap-2 flex-wrap">
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
                  <button
                    onClick={() => handleViewDetails(candidate)}
                    className="px-3 py-1 text-sm border rounded hover:bg-gray-50"
                  >
                    View Details
                  </button>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}

      {showModal && selectedCandidate && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50">
          <div className="bg-white rounded-lg max-w-2xl w-full max-h-[90vh] overflow-y-auto p-6">
            <div className="flex justify-between items-center mb-4">
              <h2 className="text-xl font-bold">Candidate Details</h2>
              <button
                onClick={() => setShowModal(false)}
                className="text-gray-500 hover:text-gray-700 text-2xl"
              >
                &times;
              </button>
            </div>

            <div className="space-y-4">
              {selectedCandidate.photo_url && (
                <img
                  src={selectedCandidate.photo_url}
                  alt={selectedCandidate.full_name}
                  className="w-32 h-32 rounded-full object-cover mx-auto"
                />
              )}

              <div>
                <h3 className="font-semibold text-lg">{selectedCandidate.full_name}</h3>
                <p className="text-sm text-gray-500">
                  Status:{" "}
                  <span
                    className={`inline-block px-2 py-1 rounded text-xs ${
                      selectedCandidate.approval_status === "APPROVED"
                        ? "bg-green-100 text-green-800"
                        : selectedCandidate.approval_status === "REJECTED"
                          ? "bg-red-100 text-red-800"
                          : "bg-yellow-100 text-yellow-800"
                    }`}
                  >
                    {selectedCandidate.approval_status}
                  </span>
                </p>
              </div>

              {selectedCandidate.date_of_birth && (
                <div>
                  <p className="text-sm font-medium">Date of Birth:</p>
                  <p className="text-sm text-gray-600">
                    {new Date(selectedCandidate.date_of_birth).toLocaleDateString()}
                  </p>
                </div>
              )}

              {selectedCandidate.registration_number && (
                <div>
                  <p className="text-sm font-medium">Registration Number:</p>
                  <p className="text-sm text-gray-600">{selectedCandidate.registration_number}</p>
                </div>
              )}

              {selectedCandidate.campus_residence && (
                <div>
                  <p className="text-sm font-medium">Campus Residence:</p>
                  <p className="text-sm text-gray-600">{selectedCandidate.campus_residence}</p>
                </div>
              )}

              {selectedCandidate.manifesto && (
                <div>
                  <p className="text-sm font-medium">Manifesto:</p>
                  <p className="text-sm text-gray-600">{selectedCandidate.manifesto}</p>
                </div>
              )}

              {selectedCandidate.bio && (
                <div>
                  <p className="text-sm font-medium">Bio:</p>
                  <p className="text-sm text-gray-600">{selectedCandidate.bio}</p>
                </div>
              )}

              {selectedCandidate.id_document_url && (
                <div>
                  <p className="text-sm font-medium">ID Document:</p>
                  <a
                    href={selectedCandidate.id_document_url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-blue-600 hover:underline text-sm"
                  >
                    View Document
                  </a>
                </div>
              )}

              {selectedCandidate.passport_photo_url && (
                <div>
                  <p className="text-sm font-medium">Passport Photo:</p>
                  <a
                    href={selectedCandidate.passport_photo_url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-blue-600 hover:underline text-sm"
                  >
                    View Photo
                  </a>
                </div>
              )}

              {selectedCandidate.review_notes && (
                <div>
                  <p className="text-sm font-medium">Review Notes:</p>
                  <p className="text-sm text-gray-600 italic">{selectedCandidate.review_notes}</p>
                </div>
              )}

              {selectedCandidate.reviewed_at && (
                <div>
                  <p className="text-sm font-medium">Reviewed At:</p>
                  <p className="text-sm text-gray-600">
                    {new Date(selectedCandidate.reviewed_at).toLocaleString()}
                  </p>
                </div>
              )}

              {selectedCandidate.approval_status === "NOMINATED" && (
                <div className="flex gap-2 pt-4 border-t">
                  <button
                    onClick={() => {
                      handleApprove(selectedCandidate.id);
                      setShowModal(false);
                    }}
                    className="px-4 py-2 bg-green-600 text-white rounded hover:bg-green-700"
                  >
                    Approve
                  </button>
                  <button
                    onClick={() => {
                      handleReject(selectedCandidate.id);
                      setShowModal(false);
                    }}
                    className="px-4 py-2 bg-red-600 text-white rounded hover:bg-red-700"
                  >
                    Reject
                  </button>
                </div>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
