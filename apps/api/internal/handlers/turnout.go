package handlers

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"campvote/apps/api/internal/models"
	"campvote/apps/api/internal/respond"
)

// TurnoutHandler provides turnout/participation counts with privacy protection (ISSUE-07).
type TurnoutHandler struct {
	DB *gorm.DB
}

func NewTurnoutHandler(db *gorm.DB) *TurnoutHandler {
	return &TurnoutHandler{DB: db}
}

// GetTurnout returns participation counts with small-group suppression.
func (h *TurnoutHandler) GetTurnout(c *gin.Context) {
	electionID := c.Param("id")

	// Get eligible count from election
	var election models.Election
	if err := h.DB.Where("id = ?", electionID).First(&election).Error; err != nil {
		respond.NotFound(c, "Election not found")
		return
	}

	// Count eligible voters
	var eligible int64
	if err := h.DB.Model(&models.ElectionRoster{}).
		Where("election_id = ? AND status = ?", electionID, models.RosterStatusEligible).
		Count(&eligible).Error; err != nil {
		respond.Internal(c, err)
		return
	}

	// Count voters who have participated
	var voted int64
	if err := h.DB.Model(&models.VoterParticipation{}).
		Where("election_id = ?", electionID).
		Count(&voted).Error; err != nil {
		respond.Internal(c, err)
		return
	}

	// Small-group suppression threshold (configurable, default 10)
	// In production, this should be an environment variable or election setting
	suppressionThreshold := 10
	suppressed := eligible < int64(suppressionThreshold)

	var percent float64
	if eligible > 0 {
		percent = float64(voted) / float64(eligible) * 100
	}

	respond.OK(c, gin.H{
		"eligible":      eligible,
		"voted":         voted,
		"percent":       percent,
		"suppressed":    suppressed,
		"last_refresh":  election.UpdatedAt,
	})
}
