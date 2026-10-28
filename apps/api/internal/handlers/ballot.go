package handlers

import (
	"errors"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"campvote/apps/api/internal/eligibility"
	"campvote/apps/api/internal/models"
	"campvote/apps/api/internal/policy"
	"campvote/apps/api/internal/respond"
)

// BallotHandler is the voter-facing side of ISSUE-04: viewing, starting and
// submitting a ballot all pass through eligibility.CheckElection on the server.
//
// The submission writes the existing privacy-separated models (participation
// and an anonymous ballot) so the eligibility gate can be tested end to end.
// Richer ballot versioning/tallying remains ISSUE-06 and ISSUE-08.
type BallotHandler struct {
	DB *gorm.DB
}

func NewBallotHandler(db *gorm.DB) *BallotHandler { return &BallotHandler{DB: db} }

// requestUser reads the account the auth middleware loaded. Both Auth and
// APIKeyOrAuth set it; a route that reaches here without it is treated as
// ineligible rather than trusted.
func requestUser(c *gin.Context) *models.User {
	value, ok := c.Get("user")
	if !ok {
		return nil
	}
	user, ok := value.(models.User)
	if !ok {
		return nil
	}
	return &user
}

var (
	errAlreadyVoted  = errors.New("you have already voted in this election")
	errBallotRefused = errors.New("ballot refused")
)

type ballotPositionView struct {
	ID            string            `json:"id"`
	Title         string            `json:"title"`
	Description   string            `json:"description"`
	QuestionType  string            `json:"question_type"`
	MinSelections int               `json:"min_selections"`
	MaxSelections int               `json:"max_selections"`
	AllowBlank    bool              `json:"allow_blank"`
	DisplayOrder  int               `json:"display_order"`
	Candidates    []ballotCandidate `json:"candidates"`
}

type ballotCandidate struct {
	ID           string `json:"id"`
	FullName     string `json:"full_name"`
	Manifesto    string `json:"manifesto"`
	BallotNumber int    `json:"ballot_number"`
}

// View returns the ballot only to an eligible voter while the election is open.
func (h *BallotHandler) View(c *gin.Context) {
	election, ok := h.gate(c)
	if !ok {
		return
	}
	var positions []models.Position
	if err := h.DB.Where("election_id = ?", election.ID).
		Order("display_order ASC, title ASC").Find(&positions).Error; err != nil {
		respond.Internal(c, err)
		return
	}
	out := make([]ballotPositionView, 0, len(positions))
	for _, position := range positions {
		var candidates []models.Candidate
		if err := h.DB.Where("election_id = ? AND position_id = ? AND approval_status = ?",
			election.ID, position.ID, models.CandidateStatusApproved).
			Order("ballot_number ASC, full_name ASC").Find(&candidates).Error; err != nil {
			respond.Internal(c, err)
			return
		}
		view := ballotPositionView{
			ID: position.ID, Title: position.Title, Description: position.Description,
			QuestionType: position.QuestionType, MinSelections: position.MinSelections,
			MaxSelections: position.MaxSelections, AllowBlank: position.AllowBlank,
			DisplayOrder: position.DisplayOrder,
		}
		for _, candidate := range candidates {
			view.Candidates = append(view.Candidates, ballotCandidate{
				ID: candidate.ID, FullName: candidate.FullName,
				Manifesto: candidate.Manifesto, BallotNumber: candidate.BallotNumber,
			})
		}
		out = append(out, view)
	}
	respond.OK(c, gin.H{
		"election_id": election.ID,
		"title":       election.Title,
		"starts_at":   election.StartsAt,
		"ends_at":     election.EndsAt,
		"positions":   out,
	})
}

// Start rechecks eligibility and refuses a second ballot before the voter has
// filled anything in. Starting is stateless: participation is recorded only on
// submission.
func (h *BallotHandler) Start(c *gin.Context) {
	election, ok := h.gate(c)
	if !ok {
		return
	}
	var count int64
	if err := h.DB.Model(&models.VoterParticipation{}).
		Where("election_id = ? AND user_id = ?", election.ID, c.GetString("user_id")).Count(&count).Error; err != nil {
		respond.Internal(c, err)
		return
	}
	if count > 0 {
		respond.Conflict(c, errAlreadyVoted.Error())
		return
	}
	respond.OK(c, gin.H{
		"election_id": election.ID,
		"starts_at":   election.StartsAt,
		"ends_at":     election.EndsAt,
		"eligible":    true,
	})
}

type ballotSelectionInput struct {
	PositionID  string  `json:"position_id"`
	CandidateID *string `json:"candidate_id"`
	Choice      string  `json:"choice"`
}

type submitBallotRequest struct {
	Selections []ballotSelectionInput `json:"selections"`
}

// Submit is the final eligibility gate. The whole ballot is validated before
// the transaction, then eligibility and open status are rechecked inside it.
func (h *BallotHandler) Submit(c *gin.Context) {
	election, ok := h.gate(c)
	if !ok {
		return
	}
	user := requestUser(c)
	var in submitBallotRequest
	if err := c.ShouldBindJSON(&in); err != nil {
		respond.BadRequest(c, "A ballot with selections is required")
		return
	}
	selections, reason := h.validateSelections(election, in.Selections)
	if reason != "" {
		respond.BadRequest(c, reason)
		return
	}

	var token string
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		var current models.Election
		if err := tx.Where("id = ?", election.ID).First(&current).Error; err != nil {
			return err
		}
		decision, err := eligibility.CheckElection(tx, &current, user, time.Now().UTC())
		if err != nil {
			return err
		}
		if !decision.Eligible {
			return errBallotRefused
		}
		var count int64
		if err := tx.Model(&models.VoterParticipation{}).
			Where("election_id = ? AND user_id = ?", current.ID, c.GetString("user_id")).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return errAlreadyVoted
		}
		participation := models.VoterParticipation{ElectionID: current.ID, UserID: c.GetString("user_id")}
		if err := tx.Create(&participation).Error; err != nil {
			return err
		}
		token = participation.ConfirmationToken
		return tx.Create(&models.AnonymousBallot{ElectionID: current.ID, Selections: selections}).Error
	})
	switch {
	case errors.Is(err, errAlreadyVoted):
		respond.Conflict(c, errAlreadyVoted.Error())
	case errors.Is(err, errBallotRefused):
		respond.Forbidden(c, "You are no longer eligible to vote in this election")
	case err != nil:
		respond.Internal(c, err)
	default:
		respond.Created(c, gin.H{
			"election_id":        election.ID,
			"confirmation_token": token,
			"message":            "Ballot accepted",
		}, "Ballot accepted")
	}
}

func (h *BallotHandler) gate(c *gin.Context) (*models.Election, bool) {
	var election models.Election
	if err := h.DB.Where("id = ?", c.Param("id")).First(&election).Error; err != nil {
		respond.NotFound(c, "Election not found")
		return nil, false
	}
	decision, err := eligibility.CheckElection(h.DB, &election, requestUser(c), time.Now().UTC())
	if err != nil {
		respond.Internal(c, err)
		return nil, false
	}
	if !decision.Eligible {
		policy.LogDenial(h.DB, c, decision.Reason, election.ID)
		respond.Forbidden(c, decision.Reason)
		return nil, false
	}
	return &election, true
}

func (h *BallotHandler) validateSelections(election *models.Election, input []ballotSelectionInput) ([]models.BallotSelection, string) {
	var positions []models.Position
	if err := h.DB.Where("election_id = ?", election.ID).Find(&positions).Error; err != nil {
		return nil, "the ballot could not be loaded"
	}
	var candidates []models.Candidate
	if err := h.DB.Where("election_id = ? AND approval_status = ?", election.ID,
		models.CandidateStatusApproved).Find(&candidates).Error; err != nil {
		return nil, "the ballot could not be loaded"
	}
	positionByID := map[string]models.Position{}
	for _, position := range positions {
		positionByID[position.ID] = position
	}
	candidateByID := map[string]models.Candidate{}
	for _, candidate := range candidates {
		candidateByID[candidate.ID] = candidate
	}

	seen := map[string]bool{}
	abstained := map[string]bool{}
	out := make([]models.BallotSelection, 0, len(input))
	for _, choice := range input {
		position, exists := positionByID[choice.PositionID]
		if !exists {
			return nil, "a selection names a position that is not on this ballot"
		}

		selection := models.BallotSelection{PositionID: position.ID}
		if choice.CandidateID != nil && strings.TrimSpace(*choice.CandidateID) != "" {
			candidate, exists := candidateByID[strings.TrimSpace(*choice.CandidateID)]
			if !exists || candidate.PositionID != position.ID {
				return nil, "a selection names a candidate that is not on this ballot"
			}
			selection.CandidateID = &candidate.ID
			selection.Choice = candidate.ID
		} else {
			value := strings.ToUpper(strings.TrimSpace(choice.Choice))
			switch position.QuestionType {
			case models.QuestionTypeSingleChoice, models.QuestionTypeMultiChoice:
				if value == "ABSTAIN" && position.AllowBlank {
					selection.Choice = value
					break
				}
				return nil, "this position requires a candidate"
			default:
				if value != "YES" && value != "NO" && !(value == "ABSTAIN" && position.AllowBlank) {
					return nil, "this position accepts YES, NO or an allowed abstention"
				}
				selection.Choice = value
			}
		}
		// A multi-choice position may hold several choices, but not the same
		// one twice. The key is position plus choice, not position alone.
		key := position.ID + "|" + selection.Choice
		if seen[key] {
			return nil, "the same choice was submitted more than once"
		}
		seen[key] = true
		if selection.Choice == "ABSTAIN" {
			abstained[position.ID] = true
		}
		out = append(out, selection)
	}

	counts := map[string]int{}
	for _, selection := range out {
		counts[selection.PositionID]++
	}
	for _, position := range positions {
		count := counts[position.ID]
		if count == 0 {
			if position.AllowBlank || position.MinSelections == 0 {
				continue
			}
			return nil, "every position must be answered"
		}
		if abstained[position.ID] && count > 1 {
			return nil, "an abstention must be the only choice for " + position.Title
		}
		max := position.MaxSelections
		// Only a multi-choice contest may take more than one answer; a
		// misconfigured max must not turn a single-choice race into one.
		if position.QuestionType != models.QuestionTypeMultiChoice && max > 1 {
			max = 1
		}
		if count < position.MinSelections {
			return nil, "not enough selections for " + position.Title
		}
		if count > max {
			return nil, "too many selections for " + position.Title
		}
	}
	return out, ""
}
