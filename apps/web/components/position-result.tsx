import { formatNumber, formatPercent } from "@/lib/format";
import type { PositionResult as PositionResultType, ResultOutcome } from "@/lib/types";

/**
 * PositionResult — ISSUE-09.
 *
 * One contest. The outcome word comes from the server: a candidate with the
 * most votes is not a winner unless the server said WINNER, and a tie is
 * spelled out as unresolved rather than settled by ordering. The bars are
 * decoration; the table underneath is the accessible equivalent and the thing
 * a screen reader or a printout gets.
 */
const OUTCOME_TEXT: Record<ResultOutcome, string> = {
  WINNER: "Result declared",
  TIE_UNRESOLVED: "Tie — unresolved",
  QUORUM_NOT_MET: "Quorum not met",
  NO_VOTES: "No votes recorded",
};

const OUTCOME_CLASS: Record<ResultOutcome, string> = {
  WINNER: "outcome ok",
  TIE_UNRESOLVED: "outcome warn",
  QUORUM_NOT_MET: "outcome warn",
  NO_VOTES: "outcome",
};

function outcomeNote(position: PositionResultType): string {
  switch (position.outcome) {
    case "TIE_UNRESOLVED":
      return "Candidates are tied on votes. This position has no winner until the tie is resolved by the rules set before the election.";
    case "QUORUM_NOT_MET":
      return `Turnout did not reach the ${formatPercent(position.quorum_percent, 0)} quorum set for this position, so no result is declared.`;
    case "NO_VOTES":
      return "No valid vote was cast for this position.";
    default:
      return position.seats > 1
        ? `${position.seats} seats filled from this count.`
        : "One seat filled from this count.";
  }
}

export function PositionResult({ position }: { position: PositionResultType }) {
  const winnerNames = new Set(position.winners);
  const tied = new Set(position.tied_candidates ?? []);

  return (
    <section className="panel position-result" aria-labelledby={`pos-${position.position_id}`}>
      <div className="pos-head">
        <h3 id={`pos-${position.position_id}`}>{position.title}</h3>
        <span className={OUTCOME_CLASS[position.outcome]}>{OUTCOME_TEXT[position.outcome]}</span>
      </div>
      <p className="hint" style={{ marginTop: 0 }}>
        {outcomeNote(position)}
      </p>

      {position.outcome === "NO_VOTES" ? null : (
        // Decorative bars. The table below carries the same numbers and is
        // what assistive technology reads, so the chart is never the only
        // representation.
        <div className="tally" aria-hidden="true">
          {position.rows.map((row) => (
            <div className="trow" key={row.candidate_id}>
              <span>
                <b>{row.name}</b>
                {winnerNames.has(row.candidate_id) ? (
                  <span className="lead-tag">Elected</span>
                ) : null}
                {tied.has(row.candidate_id) ? <span className="lead-tag">Tied</span> : null}
              </span>
              <div className={`bar${winnerNames.has(row.candidate_id) ? " lead" : ""}`}>
                <i style={{ width: `${Math.min(100, Math.max(0, row.share_percent))}%` }} />
              </div>
              <span className="vl">
                <b>{formatNumber(row.votes)}</b> · {formatPercent(row.share_percent)}
              </span>
            </div>
          ))}
        </div>
      )}

      <div className="tbl-wrap" style={{ marginTop: 14 }}>
        <table className="tbl">
          <caption className="sr">
            {position.title}: votes and share of valid votes per candidate. Share is out of{" "}
            {formatNumber(position.valid_votes)} valid votes.
          </caption>
          <thead>
            <tr>
              <th scope="col">Candidate</th>
              <th scope="col" className="r">
                Votes
              </th>
              <th scope="col" className="r">
                Share of valid votes
              </th>
              <th scope="col">Outcome</th>
            </tr>
          </thead>
          <tbody>
            {position.rows.map((row) => (
              <tr key={row.candidate_id}>
                <th scope="row">{row.name}</th>
                <td className="r">{formatNumber(row.votes)}</td>
                <td className="r">{formatPercent(row.share_percent)}</td>
                <td>
                  {winnerNames.has(row.candidate_id)
                    ? "Elected"
                    : tied.has(row.candidate_id)
                      ? "Tied — unresolved"
                      : "Not elected"}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <dl className="kv pos-foot">
        <dt>Valid votes for this position</dt>
        <dd>{formatNumber(position.valid_votes)}</dd>
        <dt>Blank</dt>
        <dd>{formatNumber(position.blank_count)}</dd>
        <dt>Abstain</dt>
        <dd>{formatNumber(position.abstain_count)}</dd>
      </dl>
    </section>
  );
}
