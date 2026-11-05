package models

import (
	"time"

	"gorm.io/gorm"

	"campvote/apps/api/internal/ids"
)

// Per-election roles from SRS §2.1. A role here scopes authority to one
// election; the user's global role says what they may do everywhere.
const (
	ElectionRoleChiefAdmin = RoleChiefElectionAdmin
	ElectionRoleOfficial   = RoleElectionOfficial
	ElectionRoleCandidate  = RoleCandidate
	ElectionRoleAuditor    = RoleElectionAuditor
)

// ElectionRole assigns a SRS role to a user for one election. The unique
// index makes an assignment idempotent and keeps two rows from granting the
// same role twice.
type ElectionRole struct {
	ID           string    `gorm:"primarykey;size:36" json:"id"`
	ElectionID   string    `gorm:"size:36;not null;uniqueIndex:idx_election_role" json:"election_id"`
	UserID       string    `gorm:"size:36;not null;uniqueIndex:idx_election_role" json:"user_id"`
	Role         string    `gorm:"size:50;not null;uniqueIndex:idx_election_role" json:"role"`
	AssignedByID string    `gorm:"size:36" json:"assigned_by_id"`
	CreatedAt    time.Time `json:"created_at"`
}

// BeforeCreate assigns a UUID.
func (er *ElectionRole) BeforeCreate(tx *gorm.DB) error {
	if er.ID == "" {
		er.ID = ids.New()
	}
	return nil
}

// IsElectionAdminRole reports whether the per-election role may administer an
// election. Candidates and auditors are deliberately excluded.
func IsElectionAdminRole(role string) bool {
	return role == ElectionRoleOfficial || role == ElectionRoleChiefAdmin
}
