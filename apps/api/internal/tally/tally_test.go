package tally_test

import (
	"testing"

	"campvote/apps/api/internal/tally"
)

// fixture is a synthetic election: three candidates for one seat, a second
// multi-seat position, and a referendum-style yes/no.
func fixture() tally.Input {
	return tally.Input{
		ElectionID:     "e-tally-fixture",
		EligibleVoters: 100,
		Positions: []tally.PositionRule{
			{
				ID: "p-president", Title: "President", QuestionType: "SINGLE_CHOICE",
				Seats: 1, DisplayOrder: 1,
				Candidates: []tally.CandidateRef{
					{ID: "c-ada", Name: "Ada"},
					{ID: "c-ben", Name: "Ben"},
					{ID: "c-cy", Name: "Cy"},
				},
			},
			{
				ID: "p-reps", Title: "Representatives", QuestionType: "MULTI_CHOICE",
				Seats: 2, DisplayOrder: 2,
				Candidates: []tally.CandidateRef{
					{ID: "c-dee", Name: "Dee"},
					{ID: "c-eli", Name: "Eli"},
					{ID: "c-fay", Name: "Fay"},
				},
			},
		},
	}
}

func ballot(id string, selections ...tally.Selection) tally.Ballot {
	return tally.Ballot{ID: id, Status: "VALID", Selections: selections}
}

func pick(position, candidate string) tally.Selection {
	return tally.Selection{PositionID: position, CandidateID: candidate, Choice: candidate}
}

func TestSingleSeatWinnerAndShare(t *testing.T) {
	in := fixture()
	in.Ballots = []tally.Ballot{
		ballot("b1", pick("p-president", "c-ada"), pick("p-reps", "c-dee")),
		ballot("b2", pick("p-president", "c-ada"), pick("p-reps", "c-dee")),
		ballot("b3", pick("p-president", "c-ben"), pick("p-reps", "c-eli")),
	}

	result := tally.Compute(in)
	president := result.Positions[0]

	if president.Outcome != tally.OutcomeWinner {
		t.Fatalf("outcome = %q, want %q", president.Outcome, tally.OutcomeWinner)
	}
	if len(president.Winners) != 1 || president.Winners[0] != "c-ada" {
		t.Fatalf("winners = %v, want [c-ada]", president.Winners)
	}
	if president.Rows[0].Votes != 2 || president.Rows[1].Votes != 1 {
		t.Fatalf("rows = %+v, want ada=2 ben=1", president.Rows)
	}
	// 2 of 3 valid votes.
	if president.Rows[0].SharePercent != 66.67 {
		t.Fatalf("ada share = %v, want 66.67", president.Rows[0].SharePercent)
	}
	if result.Turnout.Percent != 3 || result.Turnout.BallotsCast != 3 {
		t.Fatalf("turnout = %+v, want 3 cast / 3%%", result.Turnout)
	}
}

func TestTieIsUnresolvedAndNeverArbitrary(t *testing.T) {
	in := fixture()
	in.Ballots = []tally.Ballot{
		ballot("b1", pick("p-president", "c-ada")),
		ballot("b2", pick("p-president", "c-ben")),
	}

	result := tally.Compute(in)
	president := result.Positions[0]

	if president.Outcome != tally.OutcomeTieUnresolved {
		t.Fatalf("outcome = %q, want %q", president.Outcome, tally.OutcomeTieUnresolved)
	}
	if len(president.Winners) != 0 {
		t.Fatalf("winners = %v, want none declared on a tie", president.Winners)
	}
	if len(president.TiedCandidates) != 2 {
		t.Fatalf("tied = %v, want ada and ben", president.TiedCandidates)
	}
	for _, row := range president.Rows {
		if row.Votes == 1 && !row.IsTie {
			t.Fatalf("row %+v tied on votes but IsTie is false", row)
		}
		if row.IsWinner {
			t.Fatalf("row %+v is marked winner during a tie", row)
		}
	}
}

func TestMultiSeatTieAtCutLineLeavesSeatUnresolved(t *testing.T) {
	in := fixture()
	// Dee 3, Eli 2, Fay 2: two seats, cut line at 2, three tied at the line
	// for one remaining seat.
	in.Ballots = []tally.Ballot{
		ballot("b1", pick("p-reps", "c-dee"), pick("p-reps", "c-eli")),
		ballot("b2", pick("p-reps", "c-dee"), pick("p-reps", "c-eli")),
		ballot("b3", pick("p-reps", "c-dee"), pick("p-reps", "c-fay")),
		ballot("b4", pick("p-reps", "c-fay")),
	}

	reps := tally.Compute(in).Positions[1]
	if reps.Outcome != tally.OutcomeTieUnresolved {
		t.Fatalf("outcome = %q, want %q", reps.Outcome, tally.OutcomeTieUnresolved)
	}
	if len(reps.Winners) != 1 || reps.Winners[0] != "c-dee" {
		t.Fatalf("winners = %v, want only the candidate strictly above the cut line", reps.Winners)
	}
	if len(reps.TiedCandidates) != 2 {
		t.Fatalf("tied = %v, want eli and fay", reps.TiedCandidates)
	}
}

func TestMultiSeatClearCutFillsEverySeat(t *testing.T) {
	in := fixture()
	in.Ballots = []tally.Ballot{
		ballot("b1", pick("p-reps", "c-dee"), pick("p-reps", "c-eli")),
		ballot("b2", pick("p-reps", "c-dee"), pick("p-reps", "c-eli")),
		ballot("b3", pick("p-reps", "c-fay")),
	}

	reps := tally.Compute(in).Positions[1]
	if reps.Outcome != tally.OutcomeWinner {
		t.Fatalf("outcome = %q, want %q", reps.Outcome, tally.OutcomeWinner)
	}
	if len(reps.Winners) != 2 {
		t.Fatalf("winners = %v, want two seats filled", reps.Winners)
	}
}

func TestBlankAndAbstainAreReportedSeparatelyFromValidVotes(t *testing.T) {
	in := fixture()
	in.Ballots = []tally.Ballot{
		ballot("b1", pick("p-president", "c-ada")),
		ballot("b2"), // no answer for this position at all: blank
		{ID: "b3", Status: "VALID", Selections: []tally.Selection{
			{PositionID: "p-president", Choice: "ABSTAIN"},
		}},
		{ID: "b4", Status: "SPOILED", Selections: nil},
	}

	result := tally.Compute(in)
	president := result.Positions[0]

	if president.BlankCount != 1 {
		t.Fatalf("blank = %d, want 1", president.BlankCount)
	}
	if president.AbstainCount != 1 {
		t.Fatalf("abstain = %d, want 1", president.AbstainCount)
	}
	// Spoiled ballots are not valid votes and are counted separately.
	if result.BallotsSpoiled != 1 {
		t.Fatalf("spoiled = %d, want 1", result.BallotsSpoiled)
	}
	if president.ValidVotes != 1 {
		t.Fatalf("valid votes = %d, want 1 (only ada)", president.ValidVotes)
	}
	// Turnout counts cast ballots only, so spoiled must not inflate it.
	if result.Turnout.BallotsCast != 3 {
		t.Fatalf("cast = %d, want 3 (spoiled excluded)", result.Turnout.BallotsCast)
	}
	if president.Rows[0].SharePercent != 100 {
		t.Fatalf("ada share = %v, want 100 (1 of 1 valid)", president.Rows[0].SharePercent)
	}
}

func TestZeroVotesAndQuorumFailureAreExplicit(t *testing.T) {
	in := fixture()
	// No ballots at all.
	empty := tally.Compute(in)
	president := empty.Positions[0]
	if president.Outcome != tally.OutcomeNoVotes {
		t.Fatalf("outcome = %q, want %q", president.Outcome, tally.OutcomeNoVotes)
	}
	if president.Rows[0].SharePercent != 0 || president.Rows[0].IsWinner {
		t.Fatalf("row = %+v, want zero share and no winner", president.Rows[0])
	}

	// Quorum: position demands 50% turnout, only 30% cast.
	quorum := fixture()
	quorum.Positions[0].QuorumPercent = 50
	quorum.Ballots = []tally.Ballot{
		ballot("b1", pick("p-president", "c-ada")),
		ballot("b2", pick("p-president", "c-ben")),
		ballot("b3", pick("p-president", "c-cy")),
	}
	result := tally.Compute(quorum)
	if result.Turnout.Percent != 3 {
		t.Fatalf("turnout = %v, want 3", result.Turnout.Percent)
	}
	if result.Positions[0].Outcome != tally.OutcomeQuorumNotMet {
		t.Fatalf("outcome = %q, want %q", result.Positions[0].Outcome, tally.OutcomeQuorumNotMet)
	}
	if len(result.Positions[0].Winners) != 0 {
		t.Fatalf("winners = %v, want none when quorum fails", result.Positions[0].Winners)
	}
}

func TestRecomputeIsDeterministicAndHashesIdentically(t *testing.T) {
	in := fixture()
	in.Ballots = []tally.Ballot{
		ballot("b1", pick("p-president", "c-cy"), pick("p-reps", "c-fay")),
		ballot("b2", pick("p-president", "c-ada"), pick("p-reps", "c-dee")),
		ballot("b3", pick("p-president", "c-cy"), pick("p-reps", "c-eli")),
		ballot("b4", pick("p-president", "c-ben"), pick("p-reps", "c-dee")),
	}

	first := tally.Compute(in)
	second := tally.Compute(in)

	if tally.Hash(first) != tally.Hash(second) {
		t.Fatalf("hashes differ:\n%v\n%v", tally.Hash(first), tally.Hash(second))
	}
	diff := tally.Compare(first, second)
	if !diff.Identical {
		t.Fatalf("recount of identical ballots reported differences: %+v", diff)
	}
}

func TestBallotOrderDoesNotChangeTheResult(t *testing.T) {
	in := fixture()
	in.Ballots = []tally.Ballot{
		ballot("b1", pick("p-president", "c-ada")),
		ballot("b2", pick("p-president", "c-ben")),
		ballot("b3", pick("p-president", "c-cy")),
	}
	shuffled := fixture()
	shuffled.Ballots = []tally.Ballot{
		in.Ballots[2], in.Ballots[0], in.Ballots[1],
	}

	if tally.Hash(tally.Compute(in)) != tally.Hash(tally.Compute(shuffled)) {
		t.Fatal("ballot order changed the tally hash; the tally is not deterministic")
	}
}

func TestCompareReportsPerCandidateDifferences(t *testing.T) {
	in := fixture()
	in.Ballots = []tally.Ballot{
		ballot("b1", pick("p-president", "c-ada")),
		ballot("b2", pick("p-president", "c-ada")),
		ballot("b3", pick("p-president", "c-ben")),
	}
	before := tally.Compute(in)

	// A recount over a different set of stored ballots (the injected
	// discrepancy case): ben gains one.
	after := tally.Compute(in)
	for i := range after.Positions[0].Rows {
		if after.Positions[0].Rows[i].CandidateID == "c-ben" {
			after.Positions[0].Rows[i].Votes++
			after.Positions[0].Rows[i].SharePercent = 50
		}
	}

	diff := tally.Compare(before, after)
	if diff.Identical {
		t.Fatal("compare reported identical for differing tallies")
	}
	if len(diff.PositionDiffs) != 1 || len(diff.PositionDiffs[0].Candidates) != 1 {
		t.Fatalf("diffs = %+v, want one position with one candidate moved", diff.PositionDiffs)
	}
	moved := diff.PositionDiffs[0].Candidates[0]
	if moved.CandidateID != "c-ben" || moved.Before != 1 || moved.After != 2 || moved.Delta != 1 {
		t.Fatalf("moved = %+v, want ben 1 -> 2 (+1)", moved)
	}
}
