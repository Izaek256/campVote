package models

import (
	"time"

	"gorm.io/gorm"

	"campvote/apps/api/internal/ids"
)

// Sex values a student profile may hold. Nothing else is accepted, in the
// handler and in the database.
const (
	SexMale   = "MALE"
	SexFemale = "FEMALE"
)

// ResidenceArea is a campus residence an administrator maintains.
//
// The table starts empty on purpose: areas are real places, so no demo rows are
// seeded. An area is never hard-deleted — deactivating it hides it from the
// profile form and clears the assignments that pointed at it.
type ResidenceArea struct {
	ID        string    `gorm:"primarykey;size:36" json:"id"`
	Name      string    `gorm:"size:120;not null;uniqueIndex" json:"name"`
	Active    bool      `gorm:"not null;default:true" json:"active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// BeforeCreate assigns a UUID.
func (r *ResidenceArea) BeforeCreate(tx *gorm.DB) error {
	if r.ID == "" {
		r.ID = ids.New()
	}
	return nil
}

// MissingProfileFields names what a voter still has to supply before voting.
// The ballot gate refuses until this is empty.
func (u *User) MissingProfileFields() []string {
	missing := make([]string, 0, 3)
	if u.Age == nil {
		missing = append(missing, "age")
	}
	if u.Sex == "" {
		missing = append(missing, "sex")
	}
	if u.ResidenceAreaID == nil || *u.ResidenceAreaID == "" {
		missing = append(missing, "residence")
	}
	return missing
}
