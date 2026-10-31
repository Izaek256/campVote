"use client";

import { useCallback, useEffect, useState } from "react";
import { useParams } from "next/navigation";
import type { BallotVersion } from "@campvote/shared/types/ballot_version";

export default function BallotBuilderPage() {
  const params = useParams();
  const electionId = params.id;
  const [versions, setVersions] = useState<BallotVersion[]>([]);
  const [currentVersion, setCurrentVersion] = useState<BallotVersion | null>(null);
  // Placeholder until the publish-validation endpoint is wired up.
  const [validationErrors] = useState<string[]>([]);
  const [loading, setLoading] = useState(true);
  const [publishing, setPublishing] = useState(false);

  const fetchVersions = useCallback(async () => {
    try {
      const response = await fetch(`/api/v1/staff/elections/${electionId}/ballot/versions`);
      if (!response.ok) throw new Error("Failed to fetch versions");
      const data = await response.json();
      setVersions(data.versions || []);
      if (data.versions.length > 0) {
        setCurrentVersion(data.versions[0]);
      }
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  }, [electionId]);

  useEffect(() => {
    fetchVersions();
  }, [fetchVersions]);

  const handlePublish = async () => {
    if (validationErrors.length > 0) {
      alert("Fix validation errors before publishing");
      return;
    }

    if (
      !confirm("Publish this ballot? This action is irreversible and will create a new version.")
    ) {
      return;
    }

    setPublishing(true);
    try {
      const response = await fetch(`/api/v1/staff/elections/${electionId}/ballot/publish`, {
        method: "POST",
      });
      if (!response.ok) throw new Error("Failed to publish");
      const data = await response.json();
      alert(`Ballot published!\nVersion: ${data.version}\nHash: ${data.content_hash}`);
      fetchVersions();
    } catch (err) {
      console.error(err);
      alert("Failed to publish ballot");
    } finally {
      setPublishing(false);
    }
  };

  if (loading) return <div className="p-8">Loading...</div>;

  return (
    <div className="p-8">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold">Ballot Builder</h1>
        <button
          onClick={handlePublish}
          disabled={publishing || validationErrors.length > 0}
          className={`px-4 py-2 rounded ${
            publishing || validationErrors.length > 0
              ? "bg-gray-300 cursor-not-allowed"
              : "bg-green-600 text-white hover:bg-green-700"
          }`}
        >
          {publishing ? "Publishing..." : "Publish Ballot"}
        </button>
      </div>

      {validationErrors.length > 0 && (
        <div className="bg-red-50 border border-red-200 text-red-800 p-4 rounded mb-6">
          <h3 className="font-semibold mb-2">Validation Errors</h3>
          <ul className="list-disc list-inside space-y-1">
            {validationErrors.map((error, idx) => (
              <li key={idx}>{error}</li>
            ))}
          </ul>
        </div>
      )}

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div>
          <h2 className="text-lg font-semibold mb-4">Ballot Versions</h2>
          {versions.length === 0 ? (
            <div className="text-center py-8 text-gray-500 border rounded">
              No versions published yet
            </div>
          ) : (
            <div className="space-y-2">
              {versions.map((version) => (
                <div
                  key={version.id}
                  className={`p-4 border rounded cursor-pointer ${
                    currentVersion?.id === version.id ? "border-green-600 bg-green-50" : ""
                  }`}
                  onClick={() => setCurrentVersion(version)}
                >
                  <div className="flex justify-between items-center">
                    <div>
                      <span className="font-medium">Version {version.version}</span>
                      {version.status === "CURRENT" && (
                        <span className="ml-2 text-xs bg-green-100 text-green-800 px-2 py-1 rounded">
                          Current
                        </span>
                      )}
                    </div>
                    <span className="text-sm text-gray-500">
                      {new Date(version.published_at).toLocaleDateString()}
                    </span>
                  </div>
                  <div className="mt-2 text-xs text-gray-600 font-mono">
                    Hash: {version.content_hash.slice(0, 16)}...
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        <div>
          <h2 className="text-lg font-semibold mb-4">Preview</h2>
          {currentVersion ? (
            <div className="border rounded p-4">
              <div className="text-sm text-gray-600 mb-4">
                Version {currentVersion.version} • Hash: {currentVersion.content_hash}
              </div>
              <div className="bg-yellow-50 border border-yellow-200 text-yellow-800 px-4 py-3 rounded mb-4">
                <p className="font-medium">Preview Mode</p>
                <p className="text-sm">No vote or participation is recorded.</p>
              </div>
              <div className="text-sm text-gray-500">
                Ballot content would render here with the BallotView component.
              </div>
            </div>
          ) : (
            <div className="text-center py-8 text-gray-500 border rounded">
              Select a version to preview
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
