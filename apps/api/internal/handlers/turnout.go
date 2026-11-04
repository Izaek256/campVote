package handlers

import (
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"campvote/apps/api/internal/models"
	"campvote/apps/api/internal/respond"
	"campvote/apps/api/internal/results"
)

// TurnoutHandler reports participation counts. It never exposes a voter's
// identity or their choices, and it never includes per-candidate totals: those
// belong to the results endpoint, which applies the publication policy.
type TurnoutHandler struct {
	DB *gorm.DB
}

func NewTurnoutHandler(db *gorm.DB) *TurnoutHandler {
	return &TurnoutHandler{DB: db}
}

// positionParticipation is the candidate-free part of a position's count: how
// many ballots answered it and how many left it blank or abstained.
type positionParticipation struct {
	PositionID      string `json:"position_id"`
	Title           string `json:"title"`
	BallotsAnswered int    `json:"ballots_answered"`
	BlankCount      int    `json:"blank_count"`
	AbstainCount    int    `json:"abstain_count"`
}

// GetTurnout returns participation counts with optional small-group
// suppression.
func (h *TurnoutHandler) GetTurnout(c *gin.Context) {
	electionID := c.Param("id")

	var election models.Election
	if err := h.DB.Where("id = ?", electionID).First(&election).Error; err != nil {
		respond.NotFound(c, "Election not found")
		return
	}

	// The tally already counts the rule-based electorate and the stored
	// ballots without loading identity. Reuse it so turnout and the final
	// result cannot disagree about who was eligible.
	result, err := results.Compute(h.DB, electionID)
	if err != nil {
		respond.Internal(c, err)
		return
	}

	eligible := result.Turnout.EligibleVoters
	voted := result.Turnout.BallotsCast

	threshold := turnoutPrivacyThreshold()
	suppressed := threshold > 0 && eligible < threshold

	positions := make([]positionParticipation, 0, len(result.Positions))
	for _, position := range result.Positions {
		positions = append(positions, positionParticipation{
			PositionID:      position.PositionID,
			Title:           position.Title,
			BallotsAnswered: position.BallotsAnswered,
			BlankCount:      position.BlankCount,
			AbstainCount:    position.AbstainCount,
		})
	}

	var percent float64
	if eligible > 0 {
		percent = float64(voted) / float64(eligible) * 100
	}

	respond.OK(c, gin.H{
		"eligible":     eligible,
		"voted":        voted,
		"percent":      percent,
		"suppressed":   suppressed,
		"positions":    positions,
		"last_refresh": election.UpdatedAt,
	})
}

// turnoutPrivacyThreshold is the smallest group whose participation may be
// published. A threshold of zero, the MVP default, publishes turnout for a
// class-sized election; set TURNOUT_PRIVACY_THRESHOLD above zero to withhold
// counts for anything smaller. It applies to the whole election and, through
// eligibility.FacultyTurnoutBreakdown, to each faculty within it.
func turnoutPrivacyThreshold() int {
	raw := strings.TrimSpace(os.Getenv("TURNOUT_PRIVACY_THRESHOLD"))
	if raw == "" {
		return 0
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed < 0 {
		return 0
	}
	return parsed
}
