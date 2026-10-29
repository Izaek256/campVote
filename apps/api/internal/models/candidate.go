package models

import (
	"time"

	"gorm.io/gorm"
	"campvote/apps/api/internal/ids"
)

// Candidate approval lifecycle statuses (ISSUE-06)
const (
	CandidateStatusNominated = "NOMINATED" // Initial nomination by candidate or officer
	CandidateStatusPending   = "PENDING"   // Under review
	CandidateStatusApproved  = "APPROVED"  // Approved and will appear on ballot
	CandidateStatusRejected  = "REJECTED"  // Rejected with reason
	CandidateStatusWithdrawn = "WITHDRAWN" // Candidate withdrew (policy-dependent effect on ballot)
)

// Candidate represents a candidate running for a position in an election.
type Candidate struct {
	ID             string         `gorm:"primarykey;size:36" json:"id"`
	ElectionID     string         `gorm:"size:36;not null;index" json:"election_id"`
	PositionID     string         `gorm:"size:36;not null;index" json:"position_id"`
	FullName       string         `gorm:"size:255;not null" json:"full_name" binding:"required"`
	PhotoURL       string         `gorm:"size:500" json:"photo_url"`
	Manifesto      string         `gorm:"type:text" json:"manifesto"`
	Bio            string         `gorm:"type:text" json:"bio"`
	BallotNumber   int            `gorm:"not null;default:0" json:"ballot_number"`
	ApprovalStatus string         `gorm:"size:50;not null;default:'NOMINATED';index" json:"approval_status"`
	ReviewNotes    string         `gorm:"type:text" json:"review_notes,omitempty"`
	
	// Nomination tracking (ISSUE-06)
	NominatedBy    string         `gorm:"size:36;index" json:"nominated_by,omitempty"` // User ID who nominated
	ReviewedBy     string         `gorm:"size:36;index" json:"reviewed_by,omitempty"`  // User ID who reviewed
	ReviewedAt     *time.Time     `json:"reviewed_at,omitempty"`
	WithdrawnAt    *time.Time     `json:"withdrawn_at,omitempty"`
	WithdrawalReason string       `gorm:"type:text" json:"withdrawal_reason,omitempty"`
	
	CreatedAt      time.Time      `gorm:"index" json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

// BeforeCreate assigns a new UUID and sets default status.
func (c *Candidate) BeforeCreate(tx *gorm.DB) error {
	if c.ID == "" {
		c.ID = ids.New()
	}
	if c.ApprovalStatus == "" {
		c.ApprovalStatus = CandidateStatusNominated
	}
	return nil
}
