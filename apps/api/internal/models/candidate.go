package models

import (
	"time"

	"campvote/apps/api/internal/ids"

	"gorm.io/gorm"
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
	ID         string `gorm:"primarykey;size:36" json:"id"`
	ElectionID string `gorm:"size:36;not null;index" json:"election_id"`
	PositionID string `gorm:"size:36;not null;index" json:"position_id"`
	FullName   string `gorm:"size:255;not null" json:"full_name" binding:"required"`
	Slogan     string `gorm:"size:200" json:"slogan,omitempty"`
	PhotoURL   string `gorm:"size:500" json:"photo_url"`
	Manifesto  string `gorm:"type:text" json:"manifesto"`
	// ManifestoURL is the uploaded PDF, kept beside the short manifesto text
	// voters read on the ballot.
	ManifestoURL   string `gorm:"size:500" json:"manifesto_url,omitempty"`
	Bio            string `gorm:"type:text" json:"bio"`
	ContactEmail   string `gorm:"size:255" json:"contact_email,omitempty"`
	ContactPhone   string `gorm:"size:50" json:"contact_phone,omitempty"`
	BallotNumber   int    `gorm:"not null;default:0" json:"ballot_number"`
	ApprovalStatus string `gorm:"size:50;not null;default:'NOMINATED';index" json:"approval_status"`
	ReviewNotes    string `gorm:"type:text" json:"review_notes,omitempty"`

	// Nomination tracking (ISSUE-06)
	NominatedBy      string     `gorm:"size:36;index" json:"nominated_by,omitempty"` // User ID who nominated
	ReviewedBy       string     `gorm:"size:36;index" json:"reviewed_by,omitempty"`  // User ID who reviewed
	UserID           string     `gorm:"size:36;index" json:"user_id,omitempty"`      // User ID of the candidate
	ReviewedAt       *time.Time `json:"reviewed_at,omitempty"`
	WithdrawnAt      *time.Time `json:"withdrawn_at,omitempty"`
	WithdrawalReason string     `gorm:"type:text" json:"withdrawal_reason,omitempty"`

	// Application details for candidate screening
	DateOfBirth        *time.Time `json:"date_of_birth,omitempty"`
	RegistrationNumber string     `gorm:"size:100" json:"registration_number,omitempty"`
	CampusResidence    string     `gorm:"size:255" json:"campus_residence,omitempty"`
	Sex                string     `gorm:"size:20" json:"sex,omitempty"`
	Religion           string     `gorm:"size:100" json:"religion,omitempty"`
	IDDocumentURL      string     `gorm:"size:500" json:"id_document_url,omitempty"`
	PassportPhotoURL   string     `gorm:"size:500" json:"passport_photo_url,omitempty"`
	// BallotPhotoURL is the optional replacement an approved candidate uploads
	// for the ballot. When empty the passport photo is shown instead.
	BallotPhotoURL string `gorm:"size:500" json:"ballot_photo_url,omitempty"`
	// PartialTranscriptURL is the uploaded academic transcript an official
	// inspects during screening.
	PartialTranscriptURL string `gorm:"size:500" json:"partial_transcript_url,omitempty"`
	// Programme and StudyYear are derived from the registration-number email
	// (2024BSE088 → BSE, year of study in the current academic year).
	Programme string `gorm:"size:50" json:"programme,omitempty"`
	StudyYear int    `gorm:"not null;default:0" json:"study_year,omitempty"`

	CreatedAt time.Time      `gorm:"index" json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
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
