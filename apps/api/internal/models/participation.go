package models

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"gorm.io/gorm"
	"campvote/apps/api/internal/ids"
)

// VoterParticipation records that an eligible voter participated in an election.
//
// CRITICAL PRIVACY ARCHITECTURE (FR-PRIV-01, FR-PRIV-02, FR-PRIV-03, FR-VOTE-07):
// 1. This model stores ONLY who voted and when.
// 2. It has ABSOLUTELY NO foreign key, token, or relationship pointing to AnonymousBallot or BallotSelection.
// 3. The unique index on (ElectionID, UserID) enforces that an authenticated user can vote at most once.
// 4. The ConfirmationToken is a randomly generated secret receipt given to the voter confirming participation
//    without revealing or encoding any candidate choices (FR-PRIV-05).
type VoterParticipation struct {
	ID                string    `gorm:"primarykey;size:36" json:"id"`
	ElectionID        string    `gorm:"size:36;not null;index:idx_participation_election_user,unique;index" json:"election_id"`
	UserID            string    `gorm:"size:36;not null;index:idx_participation_election_user,unique;index" json:"user_id"`
	ConfirmationToken string    `gorm:"size:64;not null;index" json:"confirmation_token"`
	VotedAt           time.Time `gorm:"not null;index" json:"voted_at"`
}

// BeforeCreate assigns a UUID, generation of random confirmation token, and timestamp.
func (vp *VoterParticipation) BeforeCreate(tx *gorm.DB) error {
	if vp.ID == "" {
		vp.ID = ids.New()
	}
	if vp.ConfirmationToken == "" {
		tokenBytes := make([]byte, 16)
		if _, err := rand.Read(tokenBytes); err == nil {
			vp.ConfirmationToken = "MMU-VOTE-" + hex.EncodeToString(tokenBytes)
		} else {
			vp.ConfirmationToken = "MMU-VOTE-" + ids.New()
		}
	}
	if vp.VotedAt.IsZero() {
		vp.VotedAt = time.Now().UTC()
	}
	return nil
}
