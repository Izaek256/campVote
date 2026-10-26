package models

import (
	"time"

	"gorm.io/gorm"
	"campvote/apps/api/internal/ids"
)

// Roster membership statuses (FR-ELIG-02, FR-ELIG-07)
const (
	RosterStatusEligible   = "ELIGIBLE"
	RosterStatusIneligible = "INELIGIBLE"
	RosterStatusGraduated  = "GRADUATED"
	RosterStatusSuspended  = "SUSPENDED"
)

// ElectionRoster represents authorized institutional roster entries for an election.
type ElectionRoster struct {
	ID            string         `gorm:"primarykey;size:36" json:"id"`
	ElectionID    string         `gorm:"size:36;not null;index:idx_roster_election_email,unique;index:idx_roster_election_student,unique" json:"election_id"`
	StudentNumber string         `gorm:"size:100;not null;index:idx_roster_election_student,unique" json:"student_number"`
	RegNumber     string         `gorm:"size:100" json:"reg_number"`
	Email         string         `gorm:"size:255;not null;index:idx_roster_election_email,unique" json:"email"`
	FullName      string         `gorm:"size:255;not null" json:"full_name"`
	Faculty       string         `gorm:"size:150;index" json:"faculty"`
	Department    string         `gorm:"size:150" json:"department"`
	Programme     string         `gorm:"size:150" json:"programme"`
	StudyYear     int            `gorm:"default:1" json:"study_year"`
	Status        string         `gorm:"size:50;not null;default:'ELIGIBLE';index" json:"status"`
	
	CreatedAt     time.Time      `gorm:"index" json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

// BeforeCreate assigns a new UUID.
func (r *ElectionRoster) BeforeCreate(tx *gorm.DB) error {
	if r.ID == "" {
		r.ID = ids.New()
	}
	if r.Status == "" {
		r.Status = RosterStatusEligible
	}
	return nil
}
