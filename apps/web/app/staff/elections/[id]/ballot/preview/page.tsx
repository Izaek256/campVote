"use client";

import { useState, useEffect } from "react";
import { useParams } from "next/navigation";
import { BallotView } from "@campvote/web/components/ballot-view";
import { BallotContent } from "@campvote/shared/types/ballot_version";

export default function BallotPreviewPage() {
  const params = useParams();
  const electionId = params.id;
  const [content, setContent] = useState<BallotContent | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchLatestVersion();
  }, [electionId]);

  const fetchLatestVersion = async () => {
    try {
      const response = await fetch(`/api/v1/staff/elections/${electionId}/ballot/versions`);
      if (!response.ok) throw new Error("Failed to fetch versions");
      const data = await response.json();
      if (data.versions && data.versions.length > 0) {
        const latest = data.versions[0];
        const versionResponse = await fetch(`/api/v1/staff/elections/${electionId}/ballot/versions/${latest.id}`);
        if (versionResponse.ok) {
          const versionData = await versionResponse.json();
          setContent(versionData.content);
        }
      }
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  if (loading) return <div className="p-8">Loading...</div>;

  return (
    <div className="p-8">
      <div className="bg-yellow-50 border border-yellow-200 text-yellow-800 px-4 py-3 rounded mb-6">
        <p className="font-medium">Preview Mode</p>
        <p className="text-sm">No vote or participation is recorded. This is a ballot preview only.</p>
      </div>

      {content ? (
        <BallotView content={content} mode="preview" />
      ) : (
        <div className="text-center py-12 text-gray-500">
          No ballot version published yet. Go to the ballot builder to create and publish a version.
        </div>
      )}
    </div>
  );
}
