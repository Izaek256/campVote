import { formatNumber, formatPercent } from "@/lib/format";
import type { Turnout } from "@/lib/types";

/**
 * TurnoutSummary — ISSUE-09.
 *
 * Turnout answers "how many people voted" and vote share answers "who did they
 * vote for". They are different questions with different denominators, so this
 * is a separate block and never a row inside a candidate list. Nothing here is
 * calculated in the browser: the server sends the percentages.
 */
export function TurnoutSummary({ turnout, spoiled }: { turnout: Turnout; spoiled?: number }) {
  return (
    <section className="panel turnout-block" aria-labelledby="turnout-heading">
      <h2 id="turnout-heading" className="turnout-h">
        Turnout
      </h2>
      <p className="hint" style={{ marginTop: 0 }}>
        Ballots cast out of eligible voters. This is not vote share — candidate percentages below
        are counted only among valid votes for that position.
      </p>
      <dl className="kv">
        <dt>Eligible voters</dt>
        <dd>{turnout.known ? formatNumber(turnout.eligible_voters) : "Not recorded"}</dd>
        <dt>Ballots cast</dt>
        <dd>{formatNumber(turnout.ballots_cast)}</dd>
        <dt>Turnout</dt>
        <dd>
          <b>{turnout.known ? formatPercent(turnout.percent) : "—"}</b>
        </dd>
        {typeof spoiled === "number" && spoiled > 0 ? (
          <>
            <dt>Spoiled ballots</dt>
            <dd>{formatNumber(spoiled)} (not counted as valid votes anywhere)</dd>
          </>
        ) : null}
      </dl>
    </section>
  );
}
