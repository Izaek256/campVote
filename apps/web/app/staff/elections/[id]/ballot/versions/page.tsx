"use client";

import { useState, useEffect } from "react";
import { useParams, useSearchParams } from "next/navigation";
import { BallotVersion, BallotContent } from "@campvote/shared/types/ballot_version";

export default function BallotVersionsPage() {
  const params = useParams();
  const searchParams = useSearchParams();
  const electionId = params.id;
  const [versions, setVersions] = useState<BallotVersion[]>([]);
  const [selectedV1, setSelectedV1] = useState<string>(searchParams.get("v1") || "");
  const [selectedV2, setSelectedV2] = useState<string>(searchParams.get("v2") || "");
  const [comparison, setComparison] = useState<any>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchVersions();
  }, [electionId]);

  const fetchVersions = async () => {
    try {
      const response = await fetch(`/api/v1/staff/elections/${electionId}/ballot/versions`);
      if (!response.ok) throw new Error("Failed to fetch versions");
      const data = await response.json();
      setVersions(data.versions || []);
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  const handleCompare = async () => {
    if (!selectedV1 || !selectedV2) {
      alert("Select two versions to compare");
      return;
    }

    try {
      const response = await fetch(
        `/api/v1/staff/elections/${electionId}/ballot/versions/compare?v1=${selectedV1}&v2=${selectedV2}`
      );
      if (!response.ok) throw new Error("Failed to compare versions");
      const data = await response.json();
      setComparison(data);
    } catch (err) {
      console.error(err);
      alert("Failed to compare versions");
    }
  };

  if (loading) return <div className="p-8">Loading...</div>;

  return (
    <div className="p-8">
      <h1 className="text-2xl font-bold mb-6">Ballot Versions</h1>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-6">
        <div>
          <label className="block text-sm font-medium mb-2">Version 1</label>
          <select
            value={selectedV1}
            onChange={(e) => setSelectedV1(e.target.value)}
            className="w-full border rounded px-3 py-2"
          >
            <option value="">Select version</option>
            {versions.map((v) => (
              <option key={v.id} value={v.id}>
                Version {v.version} - {new Date(v.published_at).toLocaleDateString()}
              </option>
            ))}
          </select>
        </div>
        <div>
          <label className="block text-sm font-medium mb-2">Version 2</label>
          <select
            value={selectedV2}
            onChange={(e) => setSelectedV2(e.target.value)}
            className="w-full border rounded px-3 py-2"
          >
            <option value="">Select version</option>
            {versions.map((v) => (
              <option key={v.id} value={v.id}>
                Version {v.version} - {new Date(v.published_at).toLocaleDateString()}
              </option>
            ))}
          </select>
        </div>
      </div>

      <button
        onClick={handleCompare}
        className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 mb-6"
      >
        Compare Versions
      </button>

      {comparison && (
        <div className="border rounded-lg p-6">
          <h2 className="text-lg font-semibold mb-4">Comparison Result</h2>
          <div className={`p-4 rounded mb-4 ${comparison.same_hash ? "bg-green-50 text-green-800" : "bg-yellow-50 text-yellow-800"}`}>
            {comparison.same_hash ? "Versions are identical" : "Versions differ"}
          </div>
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            <div>
              <h3 className="font-medium mb-2">Version 1</h3>
              <div className="text-sm text-gray-600">
                Version {comparison.v1.version} • {comparison.v1.status}
              </div>
              <div className="text-xs text-gray-500 font-mono mt-1">
                Hash: {comparison.v1.content_hash}
              </div>
            </div>
            <div>
              <h3 className="font-medium mb-2">Version 2</h3>
              <div className="text-sm text-gray-600">
                Version {comparison.v2.version} • {comparison.v2.status}
              </div>
              <div className="text-xs text-gray-500 font-mono mt-1">
                Hash: {comparison.v2.content_hash}
              </div>
            </div>
          </div>
        </div>
      )}

      <div className="mt-8">
        <h2 className="text-lg font-semibold mb-4">All Versions</h2>
        <div className="space-y-2">
          {versions.map((version) => (
            <div key={version.id} className="border rounded p-4">
              <div className="flex justify-between items-center">
                <div>
                  <span className="font-medium">Version {version.version}</span>
                  {version.status === "CURRENT" && (
                    <span className="ml-2 text-xs bg-green-100 text-green-800 px-2 py-1 rounded">
                      Current
                    </span>
                  )}
                  {version.status === "SUPERSEDED" && (
                    <span className="ml-2 text-xs bg-gray-100 text-gray-800 px-2 py-1 rounded">
                      Superseded
                    </span>
                  )}
                </div>
                <div className="text-sm text-gray-500">
                  {new Date(version.published_at).toLocaleString()}
                </div>
              </div>
              <div className="mt-2 text-xs text-gray-600 font-mono">
                Hash: {version.content_hash}
              </div>
              <div className="mt-2 text-sm text-gray-600">
                Published by: {version.published_by}
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
