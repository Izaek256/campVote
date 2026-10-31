// Package tally counts stored ballots into an aggregate result.
//
// It is deliberately free of database, HTTP and clock concerns: the same input
// always produces the same output, which is what makes a recount meaningful and
// a certification snapshot verifiable. Every share and percentage is computed
// here and never re-derived by a client.
package tally

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math"
	"sort"
	"strings"
)

// Outcome labels. These are the words the UI shows; a leading candidate is not
// a winner unless the server says so.
const (
	OutcomeWinner        = "WINNER"
	OutcomeTieUnresolved = "TIE_UNRESOLVED"
	OutcomeQuorumNotMet  = "QUORUM_NOT_MET"
	OutcomeNoVotes       = "NO_VOTES"
)

// PositionRule is the counting rule for one contest, taken from the position
// the ballot version carried.
type PositionRule struct {
	ID            string         `json:"id"`
	Title         string         `json:"title"`
	QuestionType  string         `json:"question_type"`
	Seats         int            `json:"seats"`
	QuorumPercent float64        `json:"quorum_percent"`
	DisplayOrder  int            `json:"display_order"`
	Candidates    []CandidateRef `json:"candidates"`
}

// CandidateRef is display metadata for a candidate standing in a position.
type CandidateRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Selection is one choice on one ballot. CandidateID is empty for an abstain
// or blank answer.
type Selection struct {
	PositionID  string `json:"position_id"`
	CandidateID string `json:"candidate_id,omitempty"`
	Choice      string `json:"choice"`
}

// Ballot is one accepted ballot's choices. It carries no voter data at all.
type Ballot struct {
	ID         string      `json:"id"`
	Status     string      `json:"status"`
	Selections []Selection `json:"selections"`
}

// Input is everything Compute needs.
type Input struct {
	ElectionID     string
	EligibleVoters int
	Positions      []PositionRule
	Ballots        []Ballot
}

// ResultRow is one candidate's tally inside a position.
type ResultRow struct {
	CandidateID  string  `json:"candidate_id"`
	Name         string  `json:"name"`
	Votes        int     `json:"votes"`
	SharePercent float64 `json:"share_percent"`
	IsWinner     bool    `json:"is_winner"`
	IsTie        bool    `json:"is_tie"`
}

// PositionResult is the counted outcome of one contest.
type PositionResult struct {
	PositionID      string      `json:"position_id"`
	Title           string      `json:"title"`
	QuestionType    string      `json:"question_type"`
	Seats           int         `json:"seats"`
	Rows            []ResultRow `json:"rows"`
	ValidVotes      int         `json:"valid_votes"`
	BlankCount      int         `json:"blank_count"`
	AbstainCount    int         `json:"abstain_count"`
	BallotsAnswered int         `json:"ballots_answered"`
	QuorumPercent   float64     `json:"quorum_percent"`
	QuorumMet       bool        `json:"quorum_met"`
	Outcome         string      `json:"outcome"`
	Winners         []string    `json:"winners"`
	TiedCandidates  []string    `json:"tied_candidates,omitempty"`
}

// Turnout is reported separately from vote share: how many people voted out of
// how many could have, never mixed into a candidate's percentage.
type Turnout struct {
	EligibleVoters int     `json:"eligible_voters"`
	BallotsCast    int     `json:"ballots_cast"`
	Percent        float64 `json:"percent"`
	Known          bool    `json:"known"`
}

// Result is a complete, reproducible tally of one election.
type Result struct {
	ElectionID string           `json:"election_id"`
	Turnout    Turnout          `json:"turnout"`
	Positions  []PositionResult `json:"positions"`
	// BallotsSpoiled counts ballots recorded as spoiled. They are not valid
	// votes anywhere and never enter a share denominator.
	BallotsSpoiled int `json:"ballots_spoiled"`
}

// Compute counts the ballots against the position rules.
//
// Rules applied:
//   - Only VALID ballots are counted. A spoiled ballot contributes nothing to
//     any position, but is reported in BallotsSpoiled.
//   - A ballot that names no candidate for a position is a blank answer for
//     that position; a ballot that answers only with ABSTAIN is an abstain.
//   - Vote share is votes for a candidate over valid votes for the position,
//     so multi-choice shares sum to 100.
//   - Turnout is cast ballots over eligible voters, reported separately.
//   - A tie at the top is never resolved by ordering: the outcome is
//     TIE_UNRESOLVED and no winner is declared.
//   - With Seats > 1, candidates strictly above the cut line win; a tie that
//     spans the cut line leaves the remaining seats unresolved.
func Compute(in Input) Result {
	out := Result{ElectionID: in.ElectionID}

	cast, spoiled := 0, 0
	for _, ballot := range in.Ballots {
		if strings.EqualFold(ballot.Status, "SPOILED") {
			spoiled++
			continue
		}
		cast++
	}
	out.BallotsSpoiled = spoiled
	out.Turnout = Turnout{
		EligibleVoters: in.EligibleVoters,
		BallotsCast:    cast,
		Known:          in.EligibleVoters > 0,
		Percent:        percent(cast, in.EligibleVoters),
	}

	positions := append([]PositionRule(nil), in.Positions...)
	sort.SliceStable(positions, func(i, j int) bool {
		if positions[i].DisplayOrder != positions[j].DisplayOrder {
			return positions[i].DisplayOrder < positions[j].DisplayOrder
		}
		return positions[i].ID < positions[j].ID
	})

	out.Positions = make([]PositionResult, 0, len(positions))
	for _, rule := range positions {
		out.Positions = append(out.Positions, countPosition(rule, in.Ballots, out.Turnout))
	}
	return out
}

// countPosition tallies one contest.
func countPosition(rule PositionRule, ballots []Ballot, turnout Turnout) PositionResult {
	names := make(map[string]string, len(rule.Candidates))
	valid := make(map[string]int, len(rule.Candidates))
	for _, candidate := range rule.Candidates {
		names[candidate.ID] = candidate.Name
		valid[candidate.ID] = 0
	}

	res := PositionResult{
		PositionID:    rule.ID,
		Title:         rule.Title,
		QuestionType:  rule.QuestionType,
		Seats:         rule.Seats,
		QuorumPercent: rule.QuorumPercent,
		QuorumMet:     rule.QuorumPercent <= 0 || (turnout.Known && turnout.Percent >= rule.QuorumPercent),
		Winners:       []string{},
	}

	for _, ballot := range ballots {
		if strings.EqualFold(ballot.Status, "SPOILED") {
			continue
		}
		answered := false
		abstained := false
		for _, selection := range ballot.Selections {
			if selection.PositionID != rule.ID {
				continue
			}
			answered = true
			if selection.CandidateID == "" || strings.EqualFold(selection.Choice, "ABSTAIN") {
				abstained = true
				continue
			}
			if _, known := valid[selection.CandidateID]; known {
				valid[selection.CandidateID]++
			}
		}
		switch {
		case !answered:
			res.BlankCount++
		case abstained && !anyValidSelection(ballot, rule.ID, valid):
			res.AbstainCount++
		default:
			res.BallotsAnswered++
		}
	}

	res.Rows = make([]ResultRow, 0, len(rule.Candidates))
	for _, candidate := range rule.Candidates {
		votes := valid[candidate.ID]
		res.ValidVotes += votes
		res.Rows = append(res.Rows, ResultRow{
			CandidateID:  candidate.ID,
			Name:         candidate.Name,
			Votes:        votes,
			SharePercent: percent(votes, 0), // replaced below once ValidVotes is known
		})
	}
	for i := range res.Rows {
		res.Rows[i].SharePercent = percent(res.Rows[i].Votes, res.ValidVotes)
	}
	// Deterministic order: most votes first, then name, then id. Never rely on
	// insertion order so a recount produces byte-identical output.
	sort.SliceStable(res.Rows, func(i, j int) bool {
		if res.Rows[i].Votes != res.Rows[j].Votes {
			return res.Rows[i].Votes > res.Rows[j].Votes
		}
		if res.Rows[i].Name != res.Rows[j].Name {
			return res.Rows[i].Name < res.Rows[j].Name
		}
		return res.Rows[i].CandidateID < res.Rows[j].CandidateID
	})

	// A ballot that answered the position counts once for BallotsAnswered.
	applyOutcome(&res)
	return res
}

// anyValidSelection reports whether the ballot named at least one known
// candidate for the position, so a mixed abstain-plus-choice answer is not
// counted as a pure abstain.
func anyValidSelection(ballot Ballot, positionID string, valid map[string]int) bool {
	for _, selection := range ballot.Selections {
		if selection.PositionID != positionID || selection.CandidateID == "" {
			continue
		}
		if _, known := valid[selection.CandidateID]; known {
			return true
		}
	}
	return false
}

// applyOutcome decides the position's outcome label and winners. It never
// picks an arbitrary winner from a tie.
func applyOutcome(res *PositionResult) {
	if res.ValidVotes == 0 {
		res.Outcome = OutcomeNoVotes
		return
	}
	if !res.QuorumMet {
		res.Outcome = OutcomeQuorumNotMet
		return
	}

	seats := res.Seats
	if seats < 1 {
		seats = 1
	}
	if len(res.Rows) < seats {
		seats = len(res.Rows)
	}

	top := res.Rows[0].Votes
	// Ties at the very top of a single-seat contest.
	if seats == 1 {
		var tied []string
		for _, row := range res.Rows {
			if row.Votes != top {
				break
			}
			tied = append(tied, row.CandidateID)
		}
		if len(tied) > 1 {
			markTies(res, top)
			res.Outcome = OutcomeTieUnresolved
			res.TiedCandidates = tied
			return
		}
		res.Outcome = OutcomeWinner
		res.Winners = []string{res.Rows[0].CandidateID}
		res.Rows[0].IsWinner = true
		return
	}

	// Multi-seat: everything strictly above the cut line wins. A tie that
	// spans the cut line leaves the remaining seats unresolved.
	cutLine := res.Rows[seats-1].Votes
	above := 0
	var tiedAtCut []string
	for _, row := range res.Rows {
		if row.Votes > cutLine {
			above++
			continue
		}
		if row.Votes == cutLine {
			tiedAtCut = append(tiedAtCut, row.CandidateID)
		}
	}
	remaining := seats - above
	if len(tiedAtCut) > remaining {
		for i := range res.Rows {
			if res.Rows[i].Votes == cutLine {
				res.Rows[i].IsTie = true
			}
		}
		res.Outcome = OutcomeTieUnresolved
		res.TiedCandidates = tiedAtCut
		for i := 0; i < above; i++ {
			res.Rows[i].IsWinner = true
			res.Winners = append(res.Winners, res.Rows[i].CandidateID)
		}
		return
	}
	res.Outcome = OutcomeWinner
	for i := 0; i < seats; i++ {
		res.Rows[i].IsWinner = true
		res.Winners = append(res.Winners, res.Rows[i].CandidateID)
	}
}

// markTies flags every row tied on the given vote count.
func markTies(res *PositionResult, votes int) {
	for i := range res.Rows {
		if res.Rows[i].Votes == votes {
			res.Rows[i].IsTie = true
		}
	}
}

// percent is share over total, rounded to two decimals so a recount hashes to
// the same string on every platform.
func percent(part, total int) float64 {
	if total <= 0 {
		return 0
	}
	return round2(float64(part) / float64(total) * 100)
}

func round2(value float64) float64 {
	return math.Round(value*100) / 100
}

// Hash returns the SHA-256 of the canonical JSON encoding of the result. The
// same result always hashes the same, which is what a certification record
// pins and what a recount compares against.
func Hash(result Result) string {
	payload, err := json.Marshal(result)
	if err != nil {
		// A Result is plain data; a marshal failure here would be a programming
		// error, and an empty hash must never certify anything.
		return ""
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

// Differences describes how a recount diverged from a prior result.
type Differences struct {
	Identical     bool           `json:"identical"`
	PositionDiffs []PositionDiff `json:"position_diffs,omitempty"`
	TurnoutDiff   *TurnoutDiff   `json:"turnout_diff,omitempty"`
}

// PositionDiff is the per-candidate movement of one contest.
type PositionDiff struct {
	PositionID    string          `json:"position_id"`
	Title         string          `json:"title"`
	OutcomeBefore string          `json:"outcome_before"`
	OutcomeAfter  string          `json:"outcome_after"`
	Candidates    []CandidateDiff `json:"candidates,omitempty"`
}

// CandidateDiff is one candidate's vote movement.
type CandidateDiff struct {
	CandidateID string `json:"candidate_id"`
	Name        string `json:"name"`
	Before      int    `json:"before"`
	After       int    `json:"after"`
	Delta       int    `json:"delta"`
}

// TurnoutDiff is turnout movement between two results.
type TurnoutDiff struct {
	Before float64 `json:"before"`
	After  float64 `json:"after"`
}

// Compare returns the differences between a prior result and a recount of the
// same ballots. Identical means byte-for-byte equal output, which is what the
// "ballots unchanged" statement on the recount screen is based on.
func Compare(before, after Result) Differences {
	diff := Differences{Identical: true}

	if before.Turnout.Percent != after.Turnout.Percent ||
		before.Turnout.BallotsCast != after.Turnout.BallotsCast {
		diff.Identical = false
		diff.TurnoutDiff = &TurnoutDiff{Before: before.Turnout.Percent, After: after.Turnout.Percent}
	}

	beforeByID := make(map[string]PositionResult, len(before.Positions))
	for _, position := range before.Positions {
		beforeByID[position.PositionID] = position
	}
	for _, afterPos := range after.Positions {
		beforePos, seen := beforeByID[afterPos.PositionID]
		if !seen {
			diff.Identical = false
			continue
		}
		if beforePos.Outcome != afterPos.Outcome {
			diff.Identical = false
		}
		var candidates []CandidateDiff
		beforeVotes := make(map[string]int, len(beforePos.Rows))
		beforeNames := make(map[string]string, len(beforePos.Rows))
		for _, row := range beforePos.Rows {
			beforeVotes[row.CandidateID] = row.Votes
			beforeNames[row.CandidateID] = row.Name
		}
		for _, row := range afterPos.Rows {
			prev, ok := beforeVotes[row.CandidateID]
			if !ok {
				prev = 0
			}
			if prev == row.Votes {
				continue
			}
			candidates = append(candidates, CandidateDiff{
				CandidateID: row.CandidateID,
				Name:        row.Name,
				Before:      prev,
				After:       row.Votes,
				Delta:       row.Votes - prev,
			})
		}
		if len(candidates) > 0 || beforePos.Outcome != afterPos.Outcome {
			diff.Identical = false
			diff.PositionDiffs = append(diff.PositionDiffs, PositionDiff{
				PositionID:    afterPos.PositionID,
				Title:         afterPos.Title,
				OutcomeBefore: beforePos.Outcome,
				OutcomeAfter:  afterPos.Outcome,
				Candidates:    candidates,
			})
		}
	}
	return diff
}
