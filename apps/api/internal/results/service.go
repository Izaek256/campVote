// Package results turns stored ballots into snapshots and decides when an
// election's first snapshot is recorded.
//
// It exists so the lifecycle package can ask for a tally when an election
// closes without importing the HTTP handlers, and so the handler and the close
// hook share exactly one counting path. A tally never reads voter identity:
// ballots are anonymous rows and the eligible count is an aggregate.
package results

import (
	"encoding/json"
	"errors"
	"time"

	"gorm.io/gorm"

	"campvote/apps/api/internal/eligibility"
	"campvote/apps/api/internal/models"
	"campvote/apps/api/internal/tally"
)

// SystemActor is recorded on snapshots that the close hook wrote, rather than
// a person.
const SystemActor = "system"

// Closed reports whether voting has ended for this election status.
func Closed(status string) bool {
	return status == models.ElectionStatusClosed
}

// Compute counts the stored ballots of one election into an aggregate result.
//
// The eligible count is the rule-based electorate minus explicit roster
// exclusions, the same number the officer eligibility report shows. Loading
// ballots only brings aggregate fields, so there is nothing here that could
// link a voter to a choice.
func Compute(db *gorm.DB, electionID string) (tally.Result, error) {
	var election models.Election
	if err := db.Where("id = ?", electionID).First(&election).Error; err != nil {
		return tally.Result{}, err
	}
	eligible, err := eligibility.EligibleCount(db, &election)
	if err != nil {
		return tally.Result{}, err
	}

	var positions []models.Position
	if err := db.Where("election_id = ?", electionID).
		Preload("Candidates").Order("display_order ASC, id ASC").
		Find(&positions).Error; err != nil {
		return tally.Result{}, err
	}
	rules := make([]tally.PositionRule, 0, len(positions))
	for _, position := range positions {
		rule := tally.PositionRule{
			ID: position.ID, Title: position.Title, QuestionType: position.QuestionType,
			Seats: position.Seats, DisplayOrder: position.DisplayOrder,
		}
		for _, candidate := range position.Candidates {
			rule.Candidates = append(rule.Candidates, tally.CandidateRef{
				ID: candidate.ID, Name: candidate.FullName,
			})
		}
		rules = append(rules, rule)
	}

	var stored []models.AnonymousBallot
	if err := db.Where("election_id = ?", electionID).
		Preload("Selections").Find(&stored).Error; err != nil {
		return tally.Result{}, err
	}

	ballots := make([]tally.Ballot, 0, len(stored))
	for _, row := range stored {
		ballot := tally.Ballot{ID: row.ID, Status: row.Status}
		for _, selection := range row.Selections {
			s := tally.Selection{PositionID: selection.PositionID, Choice: selection.Choice}
			if selection.CandidateID != nil {
				s.CandidateID = *selection.CandidateID
			}
			ballot.Selections = append(ballot.Selections, s)
		}
		ballots = append(ballots, ballot)
	}

	return tally.Compute(tally.Input{
		ElectionID:     electionID,
		EligibleVoters: int(eligible),
		Positions:      rules,
		Ballots:        ballots,
	}), nil
}

// NewSnapshot builds the provisional snapshot for a computed result. Version is
// supplied by the caller, which already knows the latest snapshot.
func NewSnapshot(electionID string, version int, result tally.Result, actorID string, at time.Time) (models.ResultSnapshot, error) {
	payload, err := json.Marshal(result)
	if err != nil {
		return models.ResultSnapshot{}, err
	}
	return models.ResultSnapshot{
		ElectionID:     electionID,
		Version:        version,
		Status:         models.ResultStatusProvisional,
		ContentHash:    tally.Hash(result),
		ContentJSON:    string(payload),
		EligibleVoters: result.Turnout.EligibleVoters,
		BallotsCast:    result.Turnout.BallotsCast,
		TurnoutPercent: result.Turnout.Percent,
		ComputedByID:   actorID,
		ComputedAt:     at.UTC(),
	}, nil
}

// LatestVersion returns the highest snapshot version for an election, or 0 when
// none exists.
func LatestVersion(db *gorm.DB, electionID string) (int, error) {
	var latest models.ResultSnapshot
	err := db.Where("election_id = ?", electionID).
		Order("version DESC").First(&latest).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, nil
		}
		return 0, err
	}
	return latest.Version, nil
}

// EnsureOnClose records the first provisional snapshot when an election has
// closed. It is idempotent, so it is safe to call from the close hook and
// again from Read when a close raced or an earlier attempt failed.
//
// It returns true when this call wrote the snapshot.
func EnsureOnClose(db *gorm.DB, election *models.Election, at time.Time) (bool, error) {
	if election == nil || !Closed(election.Status) {
		return false, nil
	}
	var existing int64
	if err := db.Model(&models.ResultSnapshot{}).
		Where("election_id = ?", election.ID).Count(&existing).Error; err != nil {
		return false, err
	}
	if existing > 0 {
		return false, nil
	}

	result, err := Compute(db, election.ID)
	if err != nil {
		return false, err
	}
	version, err := LatestVersion(db, election.ID)
	if err != nil {
		return false, err
	}
	snapshot, err := NewSnapshot(election.ID, version+1, result, SystemActor, at)
	if err != nil {
		return false, err
	}
	details, _ := json.Marshal(map[string]any{
		"snapshot_id": snapshot.ID,
		"version":     snapshot.Version,
		"hash":        snapshot.ContentHash,
		"reason":      "automatic tally when voting closed",
	})
	err = db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&snapshot).Error; err != nil {
			return err
		}
		return tx.Create(&models.ElectionAuditLog{
			ElectionID: election.ID,
			ActorID:    SystemActor,
			Action:     models.AuditActionTallyRun,
			Details:    string(details),
			CreatedAt:  at.UTC(),
		}).Error
	})
	if err != nil {
		// A concurrent close could have written version 1 first. That is the
		// outcome we wanted, so treat an existing snapshot as success.
		var raced int64
		if countErr := db.Model(&models.ResultSnapshot{}).
			Where("election_id = ?", election.ID).Count(&raced).Error; countErr == nil && raced > 0 {
			return false, nil
		}
		return false, err
	}
	NotifySnapshot(election.ID, snapshot.ID)
	return true, nil
}

// OnClose is the lifecycle hook form of EnsureOnClose: the close is already
// durable, so the hook reports only whether the tally could be written.
func OnClose(db *gorm.DB, election *models.Election, at time.Time) error {
	_, err := EnsureOnClose(db, election, at)
	return err
}

// OnSnapshot is called after a snapshot is written, including the automatic
// one at close. The HTTP layer registers it to broadcast; it carries no voter
// data and no candidate count, only the fact that a new snapshot exists.
var OnSnapshot func(electionID, snapshotID string)

func NotifySnapshot(electionID, snapshotID string) {
	if OnSnapshot != nil {
		OnSnapshot(electionID, snapshotID)
	}
}
