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
	// Now is injectable so boundary tests can move the server clock. It is
	// never read from a request header or the client.
	Now func() time.Time
}

func NewBallotHandler(db *gorm.DB) *BallotHandler {
	return &BallotHandler{DB: db, Now: time.Now}
}

func (h *BallotHandler) now() time.Time {
	if h.Now == nil {
		return time.Now().UTC()
	}
	return h.Now().UTC()
}

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
	errStaleBallot   = errors.New("the ballot has changed; reload it and choose again")
	errVotingClosed  = errors.New("voting has closed for this election")
)

// currentBallotVersion returns the published definition voters should be
// using, or nil when the election has never published a ballot.
func currentBallotVersion(db *gorm.DB, electionID string) (*models.BallotVersion, error) {
	var version models.BallotVersion
	err := db.Where("election_id = ? AND status = ?", electionID, models.BallotVersionStatusCurrent).
		Order("version DESC").First(&version).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &version, nil
}

// View returns the ballot only to an eligible voter while the election is open.
func (h *BallotHandler) View(c *gin.Context) {
	election, ok := h.gate(c)
	if !ok {
		return
	}
	version, err := currentBallotVersion(h.DB, election.ID)
	if err != nil {
		respond.Internal(c, err)
		return
	}

	var content models.BallotContent
	versionID := any(nil)
	versionNumber := any(nil)
	if version != nil {
		parsed, err := version.GetContent()
		if err != nil {
			respond.Internal(c, err)
			return
		}
		content = *parsed
		versionID = version.ID
		versionNumber = version.Version
	} else {
		// Legacy path: an open election with no published version still
		// renders its approved positions. Publication is enforced for new
		// elections by the officer workflow (ISSUE-06).
		content, err = BallotContentFor(h.DB, election.ID)
		if err != nil {
			respond.Internal(c, err)
			return
		}
	}

	respond.OK(c, gin.H{
		"election_id":       election.ID,
		"title":             election.Title,
		"starts_at":         election.StartsAt,
		"ends_at":           election.EndsAt,
		"ballot_version_id": versionID,
		"ballot_version":    versionNumber,
		"positions":         content.Positions,
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
	version, err := currentBallotVersion(h.DB, election.ID)
	if err != nil {
		respond.Internal(c, err)
		return
	}
	versionID := any(nil)
	if version != nil {
		versionID = version.ID
	}
	respond.OK(c, gin.H{
		"election_id":       election.ID,
		"starts_at":         election.StartsAt,
		"ends_at":           election.EndsAt,
		"eligible":          true,
		"ballot_version_id": versionID,
	})
}

type ballotSelectionInput struct {
	PositionID  string  `json:"position_id"`
	CandidateID *string `json:"candidate_id"`
	Choice      string  `json:"choice"`
}

type submitBallotRequest struct {
	Selections      []ballotSelectionInput `json:"selections"`
	BallotVersionID string                 `json:"ballot_version_id"`
	IdempotencyKey  string                 `json:"idempotency_key"`
}

// Submit is the final eligibility gate. The whole ballot is validated against
// the published definition before the transaction, then eligibility, open
// status, the version and the single-participation rule are rechecked inside
// it. Participation and the anonymous ballot are written together or not at
// all (ISSUE-08).
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

	version, err := currentBallotVersion(h.DB, election.ID)
	if err != nil {
		respond.Internal(c, err)
		return
	}
	var versionID *string
	var content models.BallotContent
	if version != nil {
		// The voter must submit against the definition they were shown.
		if strings.TrimSpace(in.BallotVersionID) != version.ID {
			respond.Conflict(c, errStaleBallot.Error())
			return
		}
		parsed, err := version.GetContent()
		if err != nil {
			respond.Internal(c, err)
			return
		}
		content = *parsed
		versionID = &version.ID
	} else {
		content, err = BallotContentFor(h.DB, election.ID)
		if err != nil {
			respond.Internal(c, err)
			return
		}
	}

	selections, reason := validateSelectionsAgainst(content, in.Selections)
	if reason != "" {
		respond.BadRequest(c, reason)
		return
	}

	key := strings.TrimSpace(in.IdempotencyKey)

	var token string
	var replay bool
	err = h.DB.Transaction(func(tx *gorm.DB) error {
		var current models.Election
		if err := tx.Where("id = ?", election.ID).First(&current).Error; err != nil {
			return err
		}
		decision, err := eligibility.CheckElection(tx, &current, user, h.now())
		if err != nil {
			return err
		}
		if !decision.Eligible {
			// Closing-time race rule: acceptance is decided at commit time,
			// inside this transaction, by the server clock. A request that
			// started before ends_at but reaches here after it is refused.
			if !current.IsOpen(h.now()) {
				return errVotingClosed
			}
			return errBallotRefused
		}

		participation := models.VoterParticipation{
			ElectionID: current.ID, UserID: c.GetString("user_id"), IdempotencyKey: key,
		}
		if err := tx.Create(&participation).Error; err != nil {
			// Either the single-participation constraint or the idempotency
			// index fired. Read what won and answer consistently.
			var existing models.VoterParticipation
			found := tx.Where("election_id = ? AND user_id = ?", current.ID, c.GetString("user_id")).
				First(&existing).Error == nil
			if found && key != "" && existing.IdempotencyKey == key {
				token = existing.ConfirmationToken
				replay = true
				return nil
			}
			if found {
				return errAlreadyVoted
			}
			return err
		}
		token = participation.ConfirmationToken

		ballot := models.AnonymousBallot{
			ElectionID: current.ID, BallotVersionID: versionID, Selections: selections,
		}
		if err := tx.Create(&ballot).Error; err != nil {
			return err
		}
		// The audit references participation only. It must never carry a
		// choice, and this row carries none.
		return tx.Create(&models.ElectionAuditLog{
			ElectionID: current.ID,
			ActorID:    c.GetString("user_id"),
			Action:     models.AuditActionVoteCast,
			Details:    "Participation " + participation.ID + " recorded",
			IPAddress:  c.ClientIP(),
			CreatedAt:  h.now(),
		}).Error
	})
	switch {
	case errors.Is(err, errAlreadyVoted):
		respond.Conflict(c, errAlreadyVoted.Error())
	case errors.Is(err, errVotingClosed):
		respond.Conflict(c, errVotingClosed.Error())
	case errors.Is(err, errBallotRefused):
		respond.Forbidden(c, "You are no longer eligible to vote in this election")
	case err != nil:
		respond.Internal(c, err)
	case replay:
		respond.OK(c, gin.H{
			"election_id":        election.ID,
			"confirmation_token": token,
			"message":            "This ballot was already accepted",
		}, "This ballot was already accepted")
	default:
		respond.Created(c, gin.H{
			"election_id":        election.ID,
			"confirmation_token": token,
			"message":            "Ballot accepted",
		}, "Ballot accepted")
	}
}

// VoteStatus lets a voter recover from a lost response. It returns only
// whether a participation record exists and a coarse time, never a choice.
func (h *BallotHandler) VoteStatus(c *gin.Context) {
	var election models.Election
	if err := h.DB.Where("id = ?", c.Param("id")).First(&election).Error; err != nil {
		respond.NotFound(c, "Election not found")
		return
	}
	var participation models.VoterParticipation
	err := h.DB.Where("election_id = ? AND user_id = ?", election.ID, c.GetString("user_id")).
		First(&participation).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		respond.Internal(c, err)
		return
	}
	now := h.now()
	out := gin.H{
		"election_id": election.ID,
		"voted":       err == nil,
		"phase":       election.Status,
		"server_time": now,
	}
	if err == nil {
		// 15-minute buckets: enough for the voter to recognise the session,
		// too coarse to correlate with ballot storage order.
		coarse := participation.VotedAt.UTC().Truncate(15 * time.Minute)
		out["recorded_at"] = coarse
	}
	respond.OK(c, out)
}

func (h *BallotHandler) gate(c *gin.Context) (*models.Election, bool) {
	var election models.Election
	if err := h.DB.Where("id = ?", c.Param("id")).First(&election).Error; err != nil {
		respond.NotFound(c, "Election not found")
		return nil, false
	}
	decision, err := eligibility.CheckElection(h.DB, &election, requestUser(c), h.now())
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

// validateSelectionsAgainst applies the selection rules of one ballot
// definition. The same function serves a published version and the legacy
// live-definition path, so a rule cannot drift between what is shown and what
// is accepted.
func validateSelectionsAgainst(content models.BallotContent, input []ballotSelectionInput) ([]models.BallotSelection, string) {
	positions := content.Positions
	positionByID := map[string]models.BallotPositionContent{}
	for _, position := range positions {
		positionByID[position.ID] = position
	}
	candidateByID := map[string]models.BallotCandidateContent{}
	for _, position := range positions {
		for _, candidate := range position.Candidates {
			candidateByID[candidate.ID] = candidate
		}
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
			if !exists || !positionHasCandidate(position, candidate.ID) {
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

func positionHasCandidate(position models.BallotPositionContent, candidateID string) bool {
	for _, candidate := range position.Candidates {
		if candidate.ID == candidateID {
			return true
		}
	}
	return false
}
