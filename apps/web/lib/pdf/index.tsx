"use client";

/**
 * Client-side generation of the two downloadable election documents.
 *
 * They are built in the browser with the same UI kit tokens as the app, so
 * they can be laid out to the design and paginated to fit their content. The
 * server still decides what may be published: the results ledger is refused
 * until the election's policy allows it, exactly as the results screen is.
 */
import { pdf } from "@react-pdf/renderer";
import { ApiError, candidates, elections } from "../api";
import type { BallotContent, Election } from "../types";
import { ElectionInfoSheetDocument } from "./election-info-sheet";
import { ResultsLedgerDocument } from "./results-ledger";

function save(blob: Blob, filename: string): void {
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = filename;
  document.body.appendChild(link);
  link.click();
  link.remove();
  URL.revokeObjectURL(url);
}

/**
 * The election as this reader may see it. Voters reach an election through the
 * list rather than the officer-only detail route, so both documents read it
 * the same way the screens do.
 */
async function loadElection(electionId: string): Promise<Election> {
  const list = await elections.list();
  const found = list.find((item) => item.id === electionId);
  if (!found) throw new ApiError("This election is not available to you.", 404);
  return found;
}

/** The certified ledger: exactly what the results screen publishes. */
export async function downloadResultsLedger(electionId: string): Promise<void> {
  const [election, results] = await Promise.all([
    loadElection(electionId),
    elections.results(electionId),
  ]);
  if (!results.available || !results.snapshot) {
    throw new ApiError(
      results.reason ?? "The results for this election are not published yet.",
      409
    );
  }
  const blob = await pdf(
    <ResultsLedgerDocument
      election={election}
      snapshot={results.snapshot}
      faculties={results.turnout_by_faculty ?? []}
    />
  ).toBlob();
  save(blob, `results-${electionId}.pdf`);
}

/** The public notice: key facts, the approved candidates and how to vote. */
export async function downloadInfoSheet(electionId: string): Promise<void> {
  const election = await loadElection(electionId);
  let positions: BallotContent["positions"] = [];
  try {
    const view = await candidates.forVoters(electionId);
    positions = view.positions;
  } catch {
    // The approved candidates are not readable until the election is open to
    // this reader. The sheet still prints its key facts without them.
  }
  const blob = await pdf(
    <ElectionInfoSheetDocument election={election} positions={positions} />
  ).toBlob();
  save(blob, `election-info-${electionId}.pdf`);
}
