package models

import (
	"time"

	"gorm.io/gorm"
	"campvote/apps/api/internal/ids"
)

// Ballot status constants (FR-TALLY-08)
const (
	BallotStatusValid   = "VALID"
	BallotStatusSpoiled = "SPOILED"
)

// AnonymousBallot represents an accepted anonymous ballot in an election.
//
// CRITICAL PRIVACY ARCHITECTURE (FR-PRIV-01, FR-PRIV-02, FR-PRIV-03, FR-PRIV-10):
// 1. This table contains NO user_id, email, student_number, IP address, session ID, or user-agent.
// 2. It is impossible to join this table to the users or voter_participations table.
// 3. CastAt is stored with second precision and not correlated with access logs.
type AnonymousBallot struct {
	ID         string            `gorm:"primarykey;size:36" json:"id"`
	ElectionID string            `gorm:"size:36;not null;index" json:"election_id"`
	Status     string            `gorm:"size:20;not null;default:'VALID';index" json:"status"`
	CastAt     time.Time         `gorm:"not null;index" json:"cast_at"`

	// Selections for the ballot
	Selections []BallotSelection `gorm:"foreignKey:BallotID;constraint:OnDelete:CASCADE" json:"selections,omitempty"`
}

// BallotSelection represents a single choice made on a ballot for a position or question.
type BallotSelection struct {
	ID          string    `gorm:"primarykey;size:36" json:"id"`
	BallotID    string    `gorm:"size:36;not null;index" json:"ballot_id"`
	PositionID  string    `gorm:"size:36;not null;index" json:"position_id"`
	CandidateID *string   `gorm:"size:36;index" json:"candidate_id,omitempty"`
	Choice      string    `gorm:"size:100;not null" json:"choice"` // "YES", "NO", "ABSTAIN", or candidate ID
	CreatedAt   time.Time `gorm:"index" json:"created_at"`
}

// BeforeCreate assigns a new UUID for AnonymousBallot.
func (b *AnonymousBallot) BeforeCreate(tx *gorm.DB) error {
	if b.ID == "" {
		b.ID = ids.New()
	}
	if b.Status == "" {
		b.Status = BallotStatusValid
	}
	if b.CastAt.IsZero() {
		b.CastAt = time.Now().UTC()
	}
	return nil
}

// BeforeCreate assigns a new UUID for BallotSelection.
func (s *BallotSelection) BeforeCreate(tx *gorm.DB) error {
	if s.ID == "" {
		s.ID = ids.New()
	}
	if s.CreatedAt.IsZero() {
		s.CreatedAt = time.Now().UTC()
	}
	return nil
}
