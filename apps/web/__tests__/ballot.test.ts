import { describe, expect, it } from "vitest";
import { ABSTAIN, buildSelections, limits, validate } from "@/lib/ballot";
import type { Ballot, BallotPosition } from "@/lib/types";

function position(overrides: Partial<BallotPosition>): BallotPosition {
  return {
    id: "p1",
    title: "President",
    description: "",
    question_type: "SINGLE_CHOICE",
    min_selections: 1,
    max_selections: 1,
    allow_blank: false,
    display_order: 1,
    candidates: [],
    ...overrides,
  };
}

function ballot(...positions: BallotPosition[]): Ballot {
  return {
    election_id: "e1",
    title: "Test election",
    starts_at: "2026-10-28T08:00:00Z",
    ends_at: "2026-10-29T08:00:00Z",
    positions,
  };
}

describe("ballot validation", () => {
  it("requires a choice on a required single-choice position", () => {
    const errors = validate(ballot(position({})), {});
    expect(errors.p1).toMatch(/choose an option/i);
  });

  it("accepts an allowed blank and rejects a blank that is not allowed", () => {
    expect(validate(ballot(position({ allow_blank: true })), { p1: [ABSTAIN] })).toEqual({});
    expect(validate(ballot(position({})), { p1: [ABSTAIN] }).p1).toMatch(/cannot be left blank/i);
    const forced = validate(ballot(position({ min_selections: 1 })), {});
    expect(Object.keys(forced)).toEqual(["p1"]);
  });

  it("caps a multi-choice position at its maximum", () => {
    const multi = position({
      id: "p2",
      question_type: "MULTI_CHOICE",
      min_selections: 1,
      max_selections: 2,
    });
    expect(limits(multi)).toEqual({ min: 1, max: 2 });
    expect(validate(ballot(multi), { p2: ["a", "b"] })).toEqual({});
    expect(validate(ballot(multi), { p2: ["a", "b", "c"] }).p2).toMatch(/at most 2/i);
  });

  it("never lets an abstention be combined with a candidate", () => {
    const multi = position({
      id: "p3",
      question_type: "MULTI_CHOICE",
      min_selections: 0,
      max_selections: 2,
      allow_blank: true,
    });
    expect(validate(ballot(multi), { p3: [ABSTAIN, "candidate"] }).p3).toMatch(/only choice/i);
  });
});

describe("ballot payload", () => {
  it("sends candidate ids and explicit choices in the server's shape", () => {
    const referendum = position({ id: "p4", question_type: "REFERENDUM", allow_blank: true });
    const rows = buildSelections(ballot(position({}), referendum), {
      p1: ["candidate-1"],
      p4: ["YES"],
    });
    expect(rows).toEqual([
      { position_id: "p1", candidate_id: "candidate-1" },
      { position_id: "p4", choice: "YES" },
    ]);
  });

  it("omits positions the voter left out entirely", () => {
    expect(buildSelections(ballot(position({ allow_blank: true })), {})).toEqual([]);
  });
});
