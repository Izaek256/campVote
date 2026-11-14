import { renderToBuffer } from "@react-pdf/renderer";
import { createElement } from "react";
import { describe, expect, it } from "vitest";
import { ElectionInfoSheetDocument } from "@/lib/pdf/election-info-sheet";
import { ResultsLedgerDocument } from "@/lib/pdf/results-ledger";
import type {
  BallotContent,
  Election,
  FacultyTurnout,
  PositionResult,
  ResultSnapshot,
} from "@/lib/types";

/**
 * The downloadable documents are laid out in the browser from the same data the
 * screens show. These tests render them in Node so a broken layout or a
 * document that cannot paginate is caught before it reaches a voter.
 */

type PdfDocument = Parameters<typeof renderToBuffer>[0];

/** `renderToBuffer` is typed against the document element, not the component. */
function render(element: React.ReactElement): Promise<Buffer> {
  return renderToBuffer(element as PdfDocument);
}

function election(overrides: Partial<Election> = {}): Election {
  return {
    id: "EL-2026-007",
    title: "Guild Elections 2026/27",
    description: "The annual guild elections.",
    category: "GUILD",
    organizing_unit: "Guild Students' Association",
    status: "CLOSED",
    starts_at: "2026-10-26T05:00:00Z",
    ends_at: "2026-10-28T14:00:00Z",
    result_policy: "AFTER_CLOSE",
    allowed_domain: "",
    allowed_years: "",
    eligibility_frozen_count: 5860,
    created_by_id: "user-1",
    version: 1,
    created_at: "2026-10-01T09:00:00Z",
    updated_at: "2026-10-29T07:52:00Z",
    phase: "CLOSED",
    server_time: "2026-10-29T07:52:00Z",
    accepting_votes: false,
    display_timezone: "Africa/Kampala",
    allowed_transitions: null,
    public_listing: false,
    ...overrides,
  };
}

function position(index: number): PositionResult {
  const base = index * 100;
  return {
    position_id: `pos-${index}`,
    title: `Position ${index}`,
    question_type: "SINGLE_CHOICE",
    seats: 1,
    rows: [
      {
        candidate_id: `cand-${index}-1`,
        name: `Candidate ${index} Alpha`,
        votes: base + 1200,
        share_percent: 52.4,
        is_winner: true,
        is_tie: false,
      },
      {
        candidate_id: `cand-${index}-2`,
        name: `Candidate ${index} Beta`,
        votes: base + 830,
        share_percent: 36.2,
        is_winner: false,
        is_tie: false,
      },
      {
        candidate_id: `cand-${index}-3`,
        name: `Candidate ${index} Gamma`,
        votes: base + 260,
        share_percent: 11.4,
        is_winner: false,
        is_tie: false,
      },
    ],
    valid_votes: base + 2290,
    blank_count: 206,
    abstain_count: 0,
    ballots_answered: base + 2290,
    quorum_percent: 0,
    quorum_met: true,
    outcome: "WINNER",
    winners: [`cand-${index}-1`],
  };
}

function snapshot(
  positionCount: number,
  status: ResultSnapshot["status"] = "CERTIFIED"
): ResultSnapshot {
  return {
    id: "snap-1",
    version: 1,
    status,
    computed_at: "2026-10-29T07:52:00Z",
    computed_by_id: "user-1",
    eligible_voters: 5860,
    ballots_cast: 2417,
    turnout_percent: 41.2,
    turnout: { eligible_voters: 5860, ballots_cast: 2417, percent: 41.2, known: true },
    positions: Array.from({ length: positionCount }, (_, i) => position(i + 1)),
    ballots_spoiled: 12,
  };
}

function faculties(count: number): FacultyTurnout[] {
  return Array.from({ length: count }, (_, i) => ({
    faculty: `Faculty ${i + 1}`,
    eligible: 1520 - i * 10,
    ballots: 702 - i * 5,
    percent: 46.2,
  }));
}

function voterPositions(count: number): BallotContent["positions"] {
  return Array.from({ length: count }, (_, i) => ({
    id: `pos-${i + 1}`,
    title: `Position ${i + 1}`,
    description: "",
    question_type: "SINGLE_CHOICE",
    min_selections: 1,
    max_selections: 1,
    seats: 1,
    allow_blank: false,
    display_order: i + 1,
    candidates: [
      {
        id: `cand-${i + 1}-1`,
        full_name: `Candidate ${i + 1} Alpha`,
        manifesto: "A manifesto.",
        ballot_number: 1,
      },
      {
        id: `cand-${i + 1}-2`,
        full_name: `Candidate ${i + 1} Beta`,
        manifesto: "Another manifesto.",
        ballot_number: 2,
      },
    ],
  }));
}

/** A rendered PDF is a valid PDF, and its pages are countable in the raw bytes. */
function pages(buffer: Buffer): number {
  return (buffer.toString("latin1").match(/\/Type\s*\/Page\b/g) ?? []).length;
}

describe("results ledger", () => {
  it("renders a valid PDF for a certified election", async () => {
    const buffer = await render(
      createElement(ResultsLedgerDocument, { election: election(), snapshot: snapshot(3) })
    );
    expect(buffer.subarray(0, 5).toString()).toBe("%PDF-");
    expect(pages(buffer)).toBeGreaterThanOrEqual(1);
  });

  it("renders a provisional count as well", async () => {
    const buffer = await render(
      createElement(ResultsLedgerDocument, {
        election: election({ status: "OPEN" }),
        snapshot: snapshot(2, "PROVISIONAL"),
      })
    );
    expect(buffer.subarray(0, 5).toString()).toBe("%PDF-");
  });

  it("adds pages only as content needs them", async () => {
    const short = await render(
      createElement(ResultsLedgerDocument, { election: election(), snapshot: snapshot(1) })
    );
    const long = await render(
      createElement(ResultsLedgerDocument, { election: election(), snapshot: snapshot(24) })
    );
    expect(pages(long)).toBeGreaterThan(pages(short));
  });

  it("prints the participation table only when the server reports faculties", async () => {
    const withoutTable = await render(
      createElement(ResultsLedgerDocument, { election: election(), snapshot: snapshot(1) })
    );
    const withTable = await render(
      createElement(ResultsLedgerDocument, {
        election: election(),
        snapshot: snapshot(1),
        faculties: faculties(2),
      })
    );
    expect(withTable.subarray(0, 5).toString()).toBe("%PDF-");
    // Two rows fit on the first page, so the table adds figures, not a page.
    expect(pages(withTable)).toBe(pages(withoutTable));
  });

  it("grows the ledger as faculties are added", async () => {
    const few = await render(
      createElement(ResultsLedgerDocument, {
        election: election(),
        snapshot: snapshot(1),
        faculties: faculties(2),
      })
    );
    const many = await render(
      createElement(ResultsLedgerDocument, {
        election: election(),
        snapshot: snapshot(1),
        faculties: faculties(60),
      })
    );
    expect(pages(many)).toBeGreaterThan(pages(few));
  });

  it("renders a withheld group rather than dropping it", async () => {
    const buffer = await render(
      createElement(ResultsLedgerDocument, {
        election: election(),
        snapshot: snapshot(1),
        faculties: [
          { faculty: "Science", eligible: 1520, ballots: 702, percent: 46.2 },
          { faculty: "Education", eligible: 0, ballots: 0, percent: 0, suppressed: true },
        ],
      })
    );
    expect(buffer.subarray(0, 5).toString()).toBe("%PDF-");
  });
});

describe("election information sheet", () => {
  it("renders without any approved candidates yet", async () => {
    const buffer = await render(
      createElement(ElectionInfoSheetDocument, {
        election: election({ status: "DRAFT", eligibility_frozen_count: 0 }),
        positions: [],
      })
    );
    expect(buffer.subarray(0, 5).toString()).toBe("%PDF-");
    expect(pages(buffer)).toBe(1);
  });

  it("renders a short sheet for a handful of candidates", async () => {
    const buffer = await render(
      createElement(ElectionInfoSheetDocument, {
        election: election({ status: "OPEN" }),
        positions: voterPositions(3),
      })
    );
    expect(buffer.subarray(0, 5).toString()).toBe("%PDF-");
    expect(pages(buffer)).toBeLessThanOrEqual(2);
  });
});
