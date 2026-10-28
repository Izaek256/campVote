package models

import (
	"time"

	"campvote/apps/api/internal/ids"
	"gorm.io/gorm"
)

// Roster membership statuses (FR-ELIG-02, FR-ELIG-07)
const (
	RosterStatusEligible   = "ELIGIBLE"
	RosterStatusIneligible = "INELIGIBLE"
	RosterStatusGraduated  = "GRADUATED"
	RosterStatusSuspended  = "SUSPENDED"
)

// Provenance of a roster entry. A domain-only email is never one of these.
const (
	RosterSourceManual    = "MANUAL"
	RosterSourceCSVImport = "CSV_IMPORT"
	RosterSourceSIS       = "SIS"
)

// ElectionRoster represents authorized institutional roster entries for an election.
// It is the authoritative eligibility evidence: two entries with the same
// student number or the same institutional email cannot exist in one election.
type ElectionRoster struct {
	ID            string `gorm:"primarykey;size:36" json:"id"`
	ElectionID    string `gorm:"size:36;not null;index:idx_roster_election_email,unique;index:idx_roster_election_student,unique" json:"election_id"`
	StudentNumber string `gorm:"size:100;not null;index:idx_roster_election_student,unique" json:"student_number"`
	RegNumber     string `gorm:"size:100" json:"reg_number"`
	Email         string `gorm:"size:255;not null;index:idx_roster_election_email,unique" json:"email"`
	FullName      string `gorm:"size:255;not null" json:"full_name"`
	Faculty       string `gorm:"size:150;index" json:"faculty"`
	Department    string `gorm:"size:150" json:"department"`
	Programme     string `gorm:"size:150" json:"programme"`
	StudyYear     int    `gorm:"default:1" json:"study_year"`
	Status        string `gorm:"size:50;not null;default:'ELIGIBLE';index" json:"status"`

	// Provenance. ImportBatchID points at the EligibilityImportBatch that wrote
	// this row; ImportedByID is the officer who ran it.
	Source          string     `gorm:"size:20;not null;default:'MANUAL'" json:"source"`
	ImportBatchID   string     `gorm:"size:36;index" json:"import_batch_id,omitempty"`
	ImportedByID    string     `gorm:"size:36;index" json:"imported_by_id,omitempty"`
	StatusReason    string     `gorm:"size:255" json:"status_reason,omitempty"`
	StatusChangedAt *time.Time `gorm:"index" json:"status_changed_at,omitempty"`

	CreatedAt time.Time      `gorm:"index" json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// IsEligibleStatus reports whether status permits voting.
func IsEligibleStatus(status string) bool {
	return status == RosterStatusEligible
}

// BeforeCreate assigns a new UUID.
func (r *ElectionRoster) BeforeCreate(tx *gorm.DB) error {
	if r.ID == "" {
		r.ID = ids.New()
	}
	if r.Status == "" {
		r.Status = RosterStatusEligible
	}
	if r.Source == "" {
		r.Source = RosterSourceManual
	}
	return nil
}
