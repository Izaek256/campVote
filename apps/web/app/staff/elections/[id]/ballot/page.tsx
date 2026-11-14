"use client";

import Link from "next/link";
import { use, useCallback, useEffect, useState } from "react";
import { BallotView } from "@/components/ballot-view";
import { ConfirmDialog } from "@/components/confirm-dialog";
import { EmptyState } from "@/components/empty-state";
import { Icon } from "@/components/icon";
import { LoadError } from "@/components/load-error";
import { RequireAuth } from "@/components/require-auth";
import { StaffShell } from "@/components/staff-shell";
import { useToast } from "@/components/toast";
import { ApiError, elections, staff } from "@/lib/api";
import { formatEAT } from "@/lib/format";
import type { BallotContent, Election } from "@/lib/types";

type Version = {
  id: string;
  version: number;
  status: string;
  published_by: string;
  published_at: string;
};

function Builder({ id }: { id: string }) {
  const toast = useToast();
  const [versions, setVersions] = useState<Version[] | null>(null);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [selected, setSelected] = useState<Version | null>(null);
  const [content, setContent] = useState<BallotContent | null>(null);
  const [election, setElection] = useState<Election | null>(null);
  const [busy, setBusy] = useState(false);
  const [confirming, setConfirming] = useState(false);

  const load = useCallback(async () => {
    try {
      const [list, current] = await Promise.all([staff.versions(id), elections.get(id)]);
      setVersions(list.versions ?? []);
      setElection(current);
      const latest = list.versions?.[0];
      setSelected((previous) => previous ?? latest ?? null);
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

  useEffect(() => {
    if (!selected) {
      setContent(null);
      return;
    }
    let active = true;
    staff
      .version(id, selected.id)
      .then((data) => {
        if (active) setContent(data.content);
      })
      .catch((err) => {
        if (active)
          toast.error(
            err instanceof ApiError ? err.message : "The ballot preview could not be loaded."
          );
      });
    return () => {
      active = false;
    };
  }, [id, selected, toast]);

  async function publish() {
    setBusy(true);
    try {
      const published = await staff.publish(id);
      toast.success(`Ballot version ${published.version} published.`);
      setConfirming(false);
      setSelected(null);
      await load();
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "The ballot could not be published.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <StaffShell title="Ballot">
      <div className="page-head">
        <div>
          <span className="lbl">Ballot</span>
          <h1>Ballot builder</h1>
          <p>
            Publishing freezes the ballot content and creates a new version. Voters always see the
            latest published version.
          </p>
        </div>
        <div className="row">
          <Link className="btn" href={`/staff/elections/${id}/ballot/versions`}>
            <Icon name="hist" size="s" />
            Versions
          </Link>
          <Link className="btn" href={`/staff/elections/${id}/ballot/preview`}>
            <Icon name="eye" size="s" />
            Preview
          </Link>
          <button
            type="button"
            className="btn primary"
            onClick={() => setConfirming(true)}
            disabled={busy || !election}
          >
            {busy ? "Publishing…" : "Publish ballot"}
          </button>
        </div>
      </div>

      {loadError ? (
        <LoadError
          title="We could not load the ballot versions"
          message={loadError}
          backHref={`/staff/elections/${id}`}
          backLabel="Back to the election"
        />
      ) : !versions ? (
        <div className="panel stack" aria-busy="true">
          <span className="skel" style={{ height: 18, width: "30%" }} />
          <span className="skel" style={{ height: 14 }} />
        </div>
      ) : null}

      {versions && versions.length === 0 ? (
        <EmptyState
          title="No ballot published yet"
          action={
            <button type="button" className="btn primary" onClick={() => setConfirming(true)}>
              Publish the first version
            </button>
          }
        >
          Add positions and approved candidates first, then publish. Publishing is audited.
        </EmptyState>
      ) : null}

      {versions && versions.length > 0 ? (
        <div className="split">
          <div className="panel stack">
            <div className="section-h" style={{ marginTop: 0 }}>
              <h2>Preview</h2>
              {selected ? <span className="mono">v{selected.version}</span> : null}
            </div>
            {content ? (
              <BallotView content={content} mode="preview" />
            ) : (
              <p className="hint">Choose a version to see its ballot content.</p>
            )}
          </div>

          <aside className="stack">
            <div className="panel stack">
              <span className="lbl">Versions</span>
              <div className="stack" role="radiogroup" aria-label="Ballot versions">
                {versions.map((version) => (
                  <button
                    key={version.id}
                    type="button"
                    className="opt"
                    role="radio"
                    aria-checked={selected?.id === version.id}
                    onClick={() => setSelected(version)}
                  >
                    <span className="box rd" />
                    <span>
                      <span className="nm">Version {version.version}</span>
                      <span className="candidate-meta">
                        {version.status} · {formatEAT(version.published_at)}
                      </span>
                    </span>
                  </button>
                ))}
              </div>
            </div>
            {selected ? (
              <div className="panel stack">
                <span className="hint">Published by {selected.published_by}</span>
              </div>
            ) : null}
          </aside>
        </div>
      ) : null}

      <ConfirmDialog
        open={confirming}
        title="Publish this ballot?"
        confirmLabel="Publish"
        busy={busy}
        onClose={() => setConfirming(false)}
        onConfirm={publish}
      >
        <p>
          A new immutable version will be created from the current positions and approved
          candidates. Voters will see it immediately.
        </p>
      </ConfirmDialog>
    </StaffShell>
  );
}

export default function BallotBuilderPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  return (
    <RequireAuth staffOnly>
      <Builder id={id} />
    </RequireAuth>
  );
}
