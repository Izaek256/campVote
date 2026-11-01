"use client";

import { useCallback, useEffect, useState } from "react";
import type { Candidate } from "@campvote/shared/types/candidate";

export default function MyCandidacyPage() {
  const [candidates, setCandidates] = useState<Candidate[]>([]);
  const [loading, setLoading] = useState(true);
  const [showApplicationForm, setShowApplicationForm] = useState(false);
  const [elections, setElections] = useState<any[]>([]);
  const [formData, setFormData] = useState({
    election_id: "",
    position_id: "",
    full_name: "",
    date_of_birth: "",
    registration_number: "",
    campus_residence: "",
    manifesto: "",
    bio: "",
    photo_url: "",
    id_document_url: "",
    passport_photo_url: "",
  });
  const [uploading, setUploading] = useState(false);

  const fetchMyCandidacies = useCallback(async () => {
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
  }, []);

  const fetchElections = useCallback(async () => {
    try {
      const response = await fetch("/api/v1/protected/elections");
      if (!response.ok) throw new Error("Failed to fetch elections");
      const data = await response.json();
      setElections(data.elections || []);
    } catch (err) {
      console.error(err);
    }
  }, []);

  useEffect(() => {
    fetchMyCandidacies();
    fetchElections();
  }, [fetchMyCandidacies, fetchElections]);

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

  const handleFileUpload = async (file: File, type: "photo" | "id" | "passport") => {
    const formData = new FormData();
    formData.append("file", file);

    try {
      setUploading(true);
      const response = await fetch("/api/v1/uploads?accepts=image&max_size=5242880", {
        method: "POST",
        body: formData,
      });
      if (!response.ok) throw new Error("Upload failed");
      const data = await response.json();
      return data.data.url;
    } catch (err) {
      console.error(err);
      alert("Upload failed. Please try again.");
      return null;
    } finally {
      setUploading(false);
    }
  };

  const handleSubmitApplication = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      const response = await fetch(`/api/v1/staff/elections/${formData.election_id}/candidates`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(formData),
      });
      if (!response.ok) throw new Error("Failed to submit application");
      setShowApplicationForm(false);
      setFormData({
        election_id: "",
        position_id: "",
        full_name: "",
        date_of_birth: "",
        registration_number: "",
        campus_residence: "",
        manifesto: "",
        bio: "",
        photo_url: "",
        id_document_url: "",
        passport_photo_url: "",
      });
      fetchMyCandidacies();
      alert("Application submitted successfully!");
    } catch (err) {
      console.error(err);
      alert("Failed to submit application. Please try again.");
    }
  };

  if (loading) return <div className="p-8">Loading...</div>;

  return (
    <div className="p-4 md:p-8 max-w-4xl mx-auto">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-xl md:text-2xl font-bold">My Candidacies</h1>
        <button
          onClick={() => setShowApplicationForm(!showApplicationForm)}
          className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 text-sm md:text-base"
        >
          {showApplicationForm ? "Cancel" : "New Application"}
        </button>
      </div>

      {showApplicationForm && (
        <div className="bg-gray-50 rounded-lg p-6 mb-6">
          <h2 className="text-lg font-semibold mb-4">Submit Candidacy Application</h2>
          <form onSubmit={handleSubmitApplication} className="space-y-4">
            <div>
              <label className="block text-sm font-medium mb-1">Election *</label>
              <select
                required
                value={formData.election_id}
                onChange={(e) => setFormData({ ...formData, election_id: e.target.value })}
                className="w-full p-2 border rounded"
              >
                <option value="">Select an election</option>
                {elections.map((election) => (
                  <option key={election.id} value={election.id}>
                    {election.title}
                  </option>
                ))}
              </select>
            </div>

            <div>
              <label className="block text-sm font-medium mb-1">Position *</label>
              <select
                required
                value={formData.position_id}
                onChange={(e) => setFormData({ ...formData, position_id: e.target.value })}
                className="w-full p-2 border rounded"
                disabled={!formData.election_id}
              >
                <option value="">Select a position</option>
                {elections
                  .find((e) => e.id === formData.election_id)
                  ?.positions?.map((position: any) => (
                    <option key={position.id} value={position.id}>
                      {position.title}
                    </option>
                  ))}
              </select>
            </div>

            <div>
              <label className="block text-sm font-medium mb-1">Full Name *</label>
              <input
                type="text"
                required
                value={formData.full_name}
                onChange={(e) => setFormData({ ...formData, full_name: e.target.value })}
                className="w-full p-2 border rounded"
              />
            </div>

            <div>
              <label className="block text-sm font-medium mb-1">Date of Birth *</label>
              <input
                type="date"
                required
                value={formData.date_of_birth}
                onChange={(e) => setFormData({ ...formData, date_of_birth: e.target.value })}
                className="w-full p-2 border rounded"
              />
            </div>

            <div>
              <label className="block text-sm font-medium mb-1">Registration Number *</label>
              <input
                type="text"
                required
                value={formData.registration_number}
                onChange={(e) => setFormData({ ...formData, registration_number: e.target.value })}
                className="w-full p-2 border rounded"
                placeholder="e.g., 22/U/1001/PS"
              />
            </div>

            <div>
              <label className="block text-sm font-medium mb-1">Campus Residence *</label>
              <input
                type="text"
                required
                value={formData.campus_residence}
                onChange={(e) => setFormData({ ...formData, campus_residence: e.target.value })}
                className="w-full p-2 border rounded"
                placeholder="e.g., Block A, Room 102"
              />
            </div>

            <div>
              <label className="block text-sm font-medium mb-1">Passport Photo *</label>
              <input
                type="file"
                required
                accept="image/*"
                onChange={async (e) => {
                  const file = e.target.files?.[0];
                  if (file) {
                    const url = await handleFileUpload(file, "photo");
                    if (url) setFormData({ ...formData, photo_url: url });
                  }
                }}
                className="w-full p-2 border rounded"
              />
              {uploading && <p className="text-sm text-gray-500 mt-1">Uploading...</p>}
            </div>

            <div>
              <label className="block text-sm font-medium mb-1">ID Document *</label>
              <input
                type="file"
                required
                accept="image/*,.pdf"
                onChange={async (e) => {
                  const file = e.target.files?.[0];
                  if (file) {
                    const url = await handleFileUpload(file, "id");
                    if (url) setFormData({ ...formData, id_document_url: url });
                  }
                }}
                className="w-full p-2 border rounded"
              />
            </div>

            <div>
              <label className="block text-sm font-medium mb-1">Manifesto</label>
              <textarea
                value={formData.manifesto}
                onChange={(e) => setFormData({ ...formData, manifesto: e.target.value })}
                className="w-full p-2 border rounded"
                rows={4}
                placeholder="Tell voters about your platform..."
              />
            </div>

            <div>
              <label className="block text-sm font-medium mb-1">Bio</label>
              <textarea
                value={formData.bio}
                onChange={(e) => setFormData({ ...formData, bio: e.target.value })}
                className="w-full p-2 border rounded"
                rows={3}
                placeholder="Brief introduction about yourself..."
              />
            </div>

            <button
              type="submit"
              disabled={uploading}
              className="w-full px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 disabled:bg-gray-400"
            >
              {uploading ? "Submitting..." : "Submit Application"}
            </button>
          </form>
        </div>
      )}

      {candidates.length === 0 ? (
        <div className="text-center py-12 text-gray-500">You have no active candidacies.</div>
      ) : (
        <div className="space-y-4">
          {candidates.map((candidate) => (
            <div key={candidate.id} className="border rounded-lg p-4 md:p-6">
              <div className="flex flex-col md:flex-row justify-between items-start gap-4">
                <div className="flex gap-4">
                  {candidate.photo_url && (
                    <img
                      src={candidate.photo_url}
                      alt={candidate.full_name}
                      className="w-16 h-16 md:w-20 md:h-20 rounded-full object-cover"
                    />
                  )}
                  <div>
                    <h3 className="font-semibold text-lg">{candidate.full_name}</h3>
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
                    {candidate.registration_number && (
                      <p className="text-sm text-gray-500">Reg No: {candidate.registration_number}</p>
                    )}
                    {candidate.reviewed_at && (
                      <p className="text-sm text-gray-500">
                        Reviewed: {new Date(candidate.reviewed_at).toLocaleDateString()}
                      </p>
                    )}
                    {candidate.review_notes && (
                      <p className="text-sm text-gray-600 mt-2 italic">
                        "{candidate.review_notes}"
                      </p>
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
