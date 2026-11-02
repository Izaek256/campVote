"use client";

import Link from "next/link";
import { use, useCallback, useEffect, useState } from "react";
import { BallotView } from "@/components/ballot-view";
import { EmptyState } from "@/components/empty-state";
import { Icon } from "@/components/icon";
import { RequireAuth } from "@/components/require-auth";
import { StaffShell } from "@/components/staff-shell";
import { useToast } from "@/components/toast";
import { ApiError, staff } from "@/lib/api";
import type { BallotContent } from "@/lib/types";

function Preview({ id }: { id: string }) {
  const toast = useToast();
  const [content, setContent] = useState<BallotContent | null>(null);
  const [version, setVersion] = useState<number | null>(null);
  const [loaded, setLoaded] = useState(false);

  const load = useCallback(async () => {
    try {
      const list = await staff.versions(id);
      const latest = list.versions?.[0];
      if (latest) {
        const data = await staff.version(id, latest.id);
        setContent(data.content);
        setVersion(latest.version);
      }
    } catch (err) {
      toast.error(
        err instanceof ApiError ? err.message : "The ballot preview could not be loaded."
      );
    } finally {
      setLoaded(true);
    }
  }, [id, toast]);

  useEffect(() => {
    load();
  }, [load]);

  return (
    <StaffShell title="Ballot preview">
      <div className="page-head">
        <div>
          <span className="lbl">Preview</span>
          <h1>Ballot preview</h1>
          <p>This is exactly what voters see. Nothing is recorded.</p>
        </div>
        <Link className="btn" href={`/staff/elections/${id}/ballot`}>
          Back to ballot
        </Link>
      </div>

      {!loaded ? (
        <div className="panel stack" aria-busy="true">
          <span className="skel" style={{ height: 18, width: "30%" }} />
          <span className="skel" style={{ height: 14 }} />
        </div>
      ) : null}

      {loaded && !content ? (
        <EmptyState
          title="No ballot published yet"
          action={
            <Link className="btn primary" href={`/staff/elections/${id}/ballot`}>
              Open ballot builder
            </Link>
          }
        >
          Publish a ballot version, then come back to preview it.
        </EmptyState>
      ) : null}

      {content ? (
        <>
          <div className="banner" style={{ marginBottom: 18 }}>
            <Icon name="eye" />
            <div>
              <b>Preview mode</b>
              {version ? `Version ${version}. ` : ""}No vote or participation is recorded.
            </div>
          </div>
          <BallotView content={content} mode="preview" />
        </>
      ) : null}
    </StaffShell>
  );
}

export default function BallotPreviewPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  return (
    <RequireAuth adminOnly>
      <Preview id={id} />
    </RequireAuth>
  );
}
