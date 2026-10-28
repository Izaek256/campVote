package models

import (
	"time"

	"campvote/apps/api/internal/ids"
	"gorm.io/gorm"
)

// Significant administrative audit actions (FR-AUDIT-01)
const (
	AuditActionElectionCreated     = "ELECTION_CREATED"
	AuditActionElectionUpdated     = "ELECTION_UPDATED"
	AuditActionElectionScheduled   = "ELECTION_SCHEDULED"
	AuditActionElectionOpened      = "ELECTION_OPENED"
	AuditActionElectionClosed      = "ELECTION_CLOSED"
	AuditActionElectionReview      = "ELECTION_REVIEW_STARTED"
	AuditActionElectionArchived    = "ELECTION_ARCHIVED"
	AuditActionElectionExtended    = "ELECTION_EXTENDED"
	AuditActionElectionCancelled   = "ELECTION_CANCELLED"
	AuditActionRosterImported      = "ROSTER_IMPORTED"
	AuditActionRosterRevoked       = "ROSTER_REVOKED"
	AuditActionEligibilityFrozen   = "ELIGIBILITY_FROZEN"
	AuditActionEligibilityOverride = "ELIGIBILITY_OVERRIDE"
	AuditActionCandidateApproved   = "CANDIDATE_APPROVED"
	AuditActionCandidateRejected   = "CANDIDATE_REJECTED"
	AuditActionRecountTriggered    = "RECOUNT_TRIGGERED"
	AuditActionResultsCertified    = "RESULTS_CERTIFIED"
)

// ElectionAuditLog tracks tamper-evident administrative events for an election.
// NOTE (FR-AUDIT-03): Audit records MUST NEVER contain secret ballot choices or voter-choice mappings.
type ElectionAuditLog struct {
	ID         string    `gorm:"primarykey;size:36" json:"id"`
	ElectionID string    `gorm:"size:36;not null;index" json:"election_id"`
	ActorID    string    `gorm:"size:36;not null;index" json:"actor_id"`
	Action     string    `gorm:"size:100;not null;index" json:"action"`
	Details    string    `gorm:"type:text" json:"details"`
	IPAddress  string    `gorm:"size:45" json:"ip_address"`
	CreatedAt  time.Time `gorm:"index" json:"created_at"`
}

// BeforeCreate assigns a new UUID and timestamp.
func (a *ElectionAuditLog) BeforeCreate(tx *gorm.DB) error {
	if a.ID == "" {
		a.ID = ids.New()
	}
	if a.CreatedAt.IsZero() {
		a.CreatedAt = time.Now().UTC()
	}
	return nil
}
