package models

import (
	"time"

	"gorm.io/gorm"
	"campvote/apps/api/internal/ids"
)

// ElectionTally stores aggregate results snapshot for positions and candidates (FR-TALLY-01, FR-TALLY-12).
type ElectionTally struct {
	ID                string    `gorm:"primarykey;size:36" json:"id"`
	ElectionID        string    `gorm:"size:36;not null;index" json:"election_id"`
	PositionID        string    `gorm:"size:36;not null;index" json:"position_id"`
	CandidateID       *string   `gorm:"size:36;index" json:"candidate_id,omitempty"`
	Choice            string    `gorm:"size:100;not null" json:"choice"`
	
	// Verified vote counts and explicit denominators (FR-TALLY-03, FR-DASH-08)
	Votes             int       `gorm:"not null;default:0" json:"votes"`
	ValidVotesTotal   int       `gorm:"not null;default:0" json:"valid_votes_total"`
	Percentage        float64   `gorm:"not null;default:0" json:"percentage"`
	TurnoutPercentage float64   `gorm:"not null;default:0" json:"turnout_percentage"`
	
	// Flags
	IsWinner          bool      `gorm:"default:false" json:"is_winner"`
	IsTie             bool      `gorm:"default:false" json:"is_tie"`
	
	ComputedAt        time.Time `gorm:"not null;index" json:"computed_at"`
}

// BeforeCreate assigns a new UUID and timestamp.
func (t *ElectionTally) BeforeCreate(tx *gorm.DB) error {
	if t.ID == "" {
		t.ID = ids.New()
	}
	if t.ComputedAt.IsZero() {
		t.ComputedAt = time.Now().UTC()
	}
	return nil
}
