"use client";

import Link from "next/link";
import { use, useCallback, useEffect, useState } from "react";
import { Icon } from "@/components/icon";
import { LoadError } from "@/components/load-error";
import { RequireAuth } from "@/components/require-auth";
import { StaffShell } from "@/components/staff-shell";
import { useToast } from "@/components/toast";
import { ApiError, staff } from "@/lib/api";
import { formatEAT, formatNumber, formatPercent } from "@/lib/format";

type Turnout = {
  eligible: number;
  voted: number;
  percent: number;
  suppressed: boolean;
  last_refresh: string;
};

function Panel({ id }: { id: string }) {
  const toast = useToast();
  const [turnout, setTurnout] = useState<Turnout | null>(null);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const load = useCallback(async () => {
    try {
      setTurnout(await staff.turnout(id));
    } catch (err) {
      const message = err instanceof ApiError ? err.message : "Turnout could not be loaded.";
      setLoadError(message);
      toast.error(message);
    }
  }, [id, toast.error]);

  useEffect(() => {
    load();
  }, [load]);

  async function refresh() {
    setBusy(true);
    await load();
    setBusy(false);
  }

  return (
    <StaffShell title="Turnout">
      <div className="page-head">
        <div>
          <span className="lbl">Participation</span>
          <h1>Turnout</h1>
          <p>Counts only. This page never shows who voted or how anybody voted.</p>
        </div>
        <div className="row">
          <Link className="btn" href={`/staff/elections/${id}`}>
            Back to election
          </Link>
          <button type="button" className="btn" onClick={refresh} disabled={busy}>
            {busy ? "Refreshing…" : "Refresh"}
          </button>
        </div>
      </div>

      {loadError ? (
        <LoadError
          title="We could not load turnout"
          message={loadError}
          backHref={`/staff/elections/${id}`}
          backLabel="Back to the election"
        />
      ) : !turnout ? (
        <div className="panel stack" aria-busy="true">
          <span className="skel" style={{ height: 18, width: "30%" }} />
          <span className="skel" style={{ height: 14 }} />
        </div>
      ) : null}

      {turnout?.suppressed ? (
        <div className="banner hatch" role="status">
          <Icon name="lock" />
          <div>
            <b>Turnout is hidden to protect privacy</b>
            The number of eligible voters is below the configured privacy threshold, so turnout
            counts are withheld. This prevents identifying individual voters.
          </div>
        </div>
      ) : null}

      {turnout && !turnout.suppressed ? (
        <div className="kpis" style={{ marginBottom: 20 }}>
          <div className="kpi">
            <span className="l">Eligible voters</span>
            <span className="v">{formatNumber(turnout.eligible)}</span>
          </div>
          <div className="kpi">
            <span className="l">Ballots cast</span>
            <span className="v">{formatNumber(turnout.voted)}</span>
          </div>
          <div className="kpi">
            <span className="l">Turnout</span>
            <span className="v">{formatPercent(turnout.percent)}</span>
          </div>
        </div>
      ) : null}

      {turnout ? <p className="hint">Last refreshed {formatEAT(turnout.last_refresh)}</p> : null}

      <div className="banner" style={{ marginTop: 20 }}>
        <Icon name="shield" />
        <div>
          <b>Participation is separate from the ballot</b>
          The record that somebody voted is stored separately from their choices, so turnout cannot
          be linked to a ballot.
        </div>
      </div>
    </StaffShell>
  );
}

export default function TurnoutPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  return (
    <RequireAuth adminOnly>
      <Panel id={id} />
    </RequireAuth>
  );
}
