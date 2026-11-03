package models

import (
	"time"

	"campvote/apps/api/internal/ids"

	"gorm.io/gorm"
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

// Election lifecycle status constants (FR-CMS-05).
//
// The lifecycle has three stages only:
//   - DRAFT:  being built; private to its creator.
//   - OPEN:   published. Voting is accepted only within [StartsAt, EndsAt).
//   - CLOSED: voting has ended; the final tally is available.
const (
	ElectionStatusDraft  = "DRAFT"
	ElectionStatusOpen   = "OPEN"
	ElectionStatusClosed = "CLOSED"
)

// Result visibility policy constants (FR-DASH-06, FR-DASH-07)
const (
	ResultPolicyLive          = "LIVE"           // Provisional live results updated during voting
	ResultPolicyAfterClose    = "AFTER_CLOSE"    // Results published only after polls close
	ResultPolicyCertifiedOnly = "CERTIFIED_ONLY" // Results published only after formal certification
)

// Election represents a campus voting event with schedule, rules, and status.
type Election struct {
	ID             string `gorm:"primarykey;size:36" json:"id"`
	Title          string `gorm:"size:255;not null" json:"title" binding:"required"`
	Description    string `gorm:"type:text" json:"description"`
	Category       string `gorm:"size:50;not null;default:'GUILD'" json:"category"`
	OrganizingUnit string `gorm:"size:255;not null;default:'Electoral Commission'" json:"organizing_unit"`
	Status         string `gorm:"size:50;not null;default:'DRAFT';index" json:"status"`

	// Server-enforced schedule (UTC stored; displayed in EAT UTC+3)
	StartsAt time.Time `gorm:"not null;index" json:"starts_at"`
	EndsAt   time.Time `gorm:"not null;index" json:"ends_at"`

	// Composite index for querying open elections by time
	// This optimizes queries like: WHERE status='OPEN' AND starts_at <= NOW() AND ends_at > NOW()

	// Policies and access rules
	ResultPolicy string `gorm:"size:50;not null;default:'AFTER_CLOSE'" json:"result_policy"`

	// AllowedDomain is the optional student email domain for this election.
	// Leading "@" is accepted and stripped. Empty means any student with a
	// parseable registration email. It must not carry a database default: a
	// wrong campus domain silently hides the election from every other voter.
	AllowedDomain string `gorm:"size:255" json:"allowed_domain"`
	// AllowedYears is a comma-separated list of enrolment years, e.g. "2024,2025".
	// Empty means any year.
	AllowedYears string `gorm:"size:255" json:"allowed_years"`
	// AllowedCourses is a comma-separated list of course codes, e.g. "bse,bcs".
	// Empty means any course.
	AllowedCourses string `gorm:"size:255" json:"allowed_courses"`

	// AllowedDomains is kept for backward compatibility with existing rows.
	// New code uses AllowedDomain + AllowedYears + AllowedCourses.
	AllowedDomains string `gorm:"size:255" json:"allowed_domains,omitempty"`
	RequireRoster  bool   `gorm:"default:false" json:"require_roster"`

	// EligibilityFrozenAt is set when voting opens. The roster is the evidence
	// of who may vote; after this point new entries are refused and only
	// revocations (with audit) are accepted. FrozenCount is the eligible count
	// at that moment, for the officer report.
	EligibilityFrozenAt    *time.Time `gorm:"index" json:"eligibility_frozen_at,omitempty"`
	EligibilityFrozenCount int        `gorm:"not null;default:0" json:"eligibility_frozen_count"`

	// Metadata
	CreatedByID string         `gorm:"size:36;index" json:"created_by_id"`
	Version     int            `gorm:"not null;default:1" json:"version"`
	CreatedAt   time.Time      `gorm:"index" json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	// Associations
	Positions []Position `gorm:"foreignKey:ElectionID;constraint:OnDelete:CASCADE" json:"positions,omitempty"`
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
