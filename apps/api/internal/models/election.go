package models

import (
	"time"

	"gorm.io/gorm"
	"campvote/apps/api/internal/ids"
)

// Election category constants (FR-CMS-03)
const (
	ElectionCategoryGuild       = "GUILD"
	ElectionCategoryFaculty     = "FACULTY"
	ElectionCategoryDepartment  = "DEPARTMENT"
	ElectionCategoryProgramme   = "PROGRAMME"
	ElectionCategoryClass       = "CLASS"
	ElectionCategoryClub        = "CLUB"
	ElectionCategoryAssociation = "ASSOCIATION"
	ElectionCategoryCommittee   = "COMMITTEE"
	ElectionCategoryReferendum  = "REFERENDUM"
)

// Election lifecycle status constants (FR-CMS-05)
const (
	ElectionStatusDraft         = "DRAFT"
	ElectionStatusRegistration  = "REGISTRATION"
	ElectionStatusPending       = "PENDING_APPROVAL"
	ElectionStatusScheduled     = "SCHEDULED"
	ElectionStatusOpen          = "OPEN"
	ElectionStatusClosed        = "CLOSED"
	ElectionStatusReview        = "RESULTS_UNDER_REVIEW"
	ElectionStatusFinalized     = "FINALIZED"
	ElectionStatusArchived      = "ARCHIVED"
	ElectionStatusCancelled     = "CANCELLED"
)

// Result visibility policy constants (FR-DASH-06, FR-DASH-07)
const (
	ResultPolicyLive          = "LIVE"           // Provisional live results updated during voting
	ResultPolicyAfterClose    = "AFTER_CLOSE"    // Results published only after polls close
	ResultPolicyCertifiedOnly = "CERTIFIED_ONLY" // Results published only after formal certification
)

// Election represents a campus voting event with schedule, rules, and status.
type Election struct {
	ID             string         `gorm:"primarykey;size:36" json:"id"`
	Title          string         `gorm:"size:255;not null" json:"title" binding:"required"`
	Description    string         `gorm:"type:text" json:"description"`
	Category       string         `gorm:"size:50;not null;default:'GUILD'" json:"category"`
	OrganizingUnit string         `gorm:"size:255;not null;default:'MMU Electoral Commission'" json:"organizing_unit"`
	Status         string         `gorm:"size:50;not null;default:'DRAFT';index" json:"status"`
	
	// Server-enforced schedule (UTC stored; displayed in EAT UTC+3)
	StartsAt       time.Time      `gorm:"not null;index" json:"starts_at"`
	EndsAt         time.Time      `gorm:"not null;index" json:"ends_at"`
	
	// Policies and access rules
	ResultPolicy   string         `gorm:"size:50;not null;default:'AFTER_CLOSE'" json:"result_policy"`
	AllowedDomains string         `gorm:"size:255;default:'@std.must.ac.ug,@must.ac.ug'" json:"allowed_domains"`
	RequireRoster  bool           `gorm:"default:false" json:"require_roster"`
	
	// Metadata
	CreatedByID    string         `gorm:"size:36;index" json:"created_by_id"`
	Version        int            `gorm:"not null;default:1" json:"version"`
	CreatedAt      time.Time      `gorm:"index" json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`

	// Associations
	Positions      []Position     `gorm:"foreignKey:ElectionID;constraint:OnDelete:CASCADE" json:"positions,omitempty"`
}

// BeforeCreate assigns a new UUID and initial version.
func (e *Election) BeforeCreate(tx *gorm.DB) error {
	if e.ID == "" {
		e.ID = ids.New()
	}
	if e.Version == 0 {
		e.Version = 1
	}
	if e.Status == "" {
		e.Status = ElectionStatusDraft
	}
	if e.ResultPolicy == "" {
		e.ResultPolicy = ResultPolicyAfterClose
	}
	return nil
}

// BeforeUpdate increments Version for optimistic locking.
func (e *Election) BeforeUpdate(tx *gorm.DB) error {
	tx.Statement.SetColumn("version", gorm.Expr("version + 1"))
	return nil
}

// IsOpen checks if the election is in OPEN status and current time is within schedule.
func (e *Election) IsOpen(at time.Time) bool {
	if e.Status != ElectionStatusOpen {
		return false
	}
	return !at.Before(e.StartsAt) && at.Before(e.EndsAt)
}

// CanAcceptVotes verifies if ballot submissions are permitted.
func (e *Election) CanAcceptVotes(at time.Time) bool {
	return e.IsOpen(at)
}
