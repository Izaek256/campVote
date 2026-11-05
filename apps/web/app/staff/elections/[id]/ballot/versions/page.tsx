"use client";

import Link from "next/link";
import { use, useCallback, useEffect, useState } from "react";
import { EmptyState } from "@/components/empty-state";
import { Icon } from "@/components/icon";
import { LoadError } from "@/components/load-error";
import { RequireAuth } from "@/components/require-auth";
import { StaffShell } from "@/components/staff-shell";
import { useToast } from "@/components/toast";
import { ApiError, staff } from "@/lib/api";
import { formatEAT } from "@/lib/format";

type Version = {
  id: string;
  version: number;
  content_hash: string;
  status: string;
  published_by: string;
  published_at: string;
};

type Comparison = {
  same_hash: boolean;
  v1: Version;
  v2: Version;
};

function Versions({ id }: { id: string }) {
  const toast = useToast();
  const [versions, setVersions] = useState<Version[] | null>(null);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [v1, setV1] = useState("");
  const [v2, setV2] = useState("");
  const [comparison, setComparison] = useState<Comparison | null>(null);
  const [busy, setBusy] = useState(false);

  const load = useCallback(async () => {
    try {
      const data = await staff.versions(id);
      setVersions(data.versions ?? []);
    } catch (err) {
      const message =
        err instanceof ApiError ? err.message : "Ballot versions could not be loaded.";
      setLoadError(message);
      toast.error(message);
    }
  }, [id, toast]);

  useEffect(() => {
    load();
  }, [load]);

  async function compare() {
    if (!v1 || !v2) {
      toast.error("Choose two versions to compare.");
      return;
    }
    if (v1 === v2) {
      toast.error("Choose two different versions.");
      return;
    }
    setBusy(true);
    try {
      setComparison(await staff.compare(id, v1, v2));
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "The versions could not be compared.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <StaffShell title="Ballot versions">
      <div className="page-head">
        <div>
          <span className="lbl">Ballot history</span>
          <h1>Ballot versions</h1>
          <p>Every publish creates an immutable version with a content hash.</p>
        </div>
        <Link className="btn" href={`/staff/elections/${id}/ballot`}>
          Back to ballot
        </Link>
      </div>

      {loadError ? (
        <LoadError
          title="We could not load the ballot versions"
          message={loadError}
          backHref={`/staff/elections/${id}/ballot`}
          backLabel="Back to the ballot"
        />
      ) : !versions ? (
        <div className="panel stack" aria-busy="true">
          <span className="skel" style={{ height: 18, width: "30%" }} />
          <span className="skel" style={{ height: 14 }} />
        </div>
      ) : null}

      {versions && versions.length === 0 ? (
        <EmptyState title="No versions yet">
          Publish the ballot to create the first version.
        </EmptyState>
      ) : null}

      {versions && versions.length > 0 ? (
        <>
          <section className="panel stack" style={{ marginBottom: 20 }}>
            <div className="section-h" style={{ marginTop: 0 }}>
              <h2>Compare two versions</h2>
            </div>
            <div className="form-grid">
              <div className="field">
                <label htmlFor="v1">Version 1</label>
                <select
                  id="v1"
                  className="sel"
                  value={v1}
                  onChange={(event) => setV1(event.target.value)}
                >
                  <option value="">Select a version</option>
                  {versions.map((version) => (
                    <option key={version.id} value={version.id}>
                      Version {version.version} · {formatEAT(version.published_at)}
                    </option>
                  ))}
                </select>
              </div>
              <div className="field">
                <label htmlFor="v2">Version 2</label>
                <select
                  id="v2"
                  className="sel"
                  value={v2}
                  onChange={(event) => setV2(event.target.value)}
                >
                  <option value="">Select a version</option>
                  {versions.map((version) => (
                    <option key={version.id} value={version.id}>
                      Version {version.version} · {formatEAT(version.published_at)}
                    </option>
                  ))}
                </select>
              </div>
            </div>
            <div>
              <button type="button" className="btn primary" onClick={compare} disabled={busy}>
                {busy ? "Comparing…" : "Compare versions"}
              </button>
            </div>
          </section>

          {comparison ? (
            <div
              className={`banner ${comparison.same_hash ? "ok" : "hatch"}`}
              style={{ marginBottom: 20 }}
            >
              <Icon name={comparison.same_hash ? "check" : "alert"} />
              <div>
                <b>{comparison.same_hash ? "Identical content" : "Content differs"}</b>
                Version {comparison.v1.version} and version {comparison.v2.version}{" "}
                {comparison.same_hash
                  ? "have the same content hash."
                  : "have different content hashes."}
                <div className="mono" style={{ marginTop: 8 }}>
                  v{comparison.v1.version}: {comparison.v1.content_hash}
                </div>
                <div className="mono">
                  v{comparison.v2.version}: {comparison.v2.content_hash}
                </div>
              </div>
            </div>
          ) : null}

          <div className="panel pad0">
            <table className="tbl">
              <thead>
                <tr>
                  <th>Version</th>
                  <th>Status</th>
                  <th>Published</th>
                  <th>Hash</th>
                </tr>
              </thead>
              <tbody>
                {versions.map((version) => (
                  <tr key={version.id}>
                    <td>
                      <b>v{version.version}</b>
                    </td>
                    <td>{version.status.toLowerCase()}</td>
                    <td>
                      {formatEAT(version.published_at)}
                      <span className="candidate-meta">{version.published_by}</span>
                    </td>
                    <td className="mono">{version.content_hash.slice(0, 16)}…</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </>
      ) : null}
    </StaffShell>
  );
}

export default function BallotVersionsPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  return (
    <RequireAuth adminOnly>
      <Versions id={id} />
    </RequireAuth>
  );
}
