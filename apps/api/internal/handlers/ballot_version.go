package handlers

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"campvote/apps/api/internal/models"
	"campvote/apps/api/internal/respond"
)

// BallotVersionHandler manages ballot versioning and publication (ISSUE-06).
type BallotVersionHandler struct {
	DB  *gorm.DB
	Now func() time.Time
}

func NewBallotVersionHandler(db *gorm.DB) *BallotVersionHandler {
	return &BallotVersionHandler{DB: db, Now: time.Now}
}

func (h *BallotVersionHandler) now() time.Time {
	if h.Now == nil {
		return time.Now().UTC()
	}
	return h.Now().UTC()
}

// BuildBallotContent constructs the ballot content JSON from positions and candidates.
func (h *BallotVersionHandler) BuildBallotContent(electionID string) (string, error) {
	var positions []models.Position
	if err := h.DB.Where("election_id = ?", electionID).
		Order("display_order ASC, title ASC").Find(&positions).Error; err != nil {
		return "", err
	}

	content := models.BallotContent{
		Positions: make([]models.BallotPositionContent, 0, len(positions)),
	}

	for _, position := range positions {
		var candidates []models.Candidate
		if err := h.DB.Where("election_id = ? AND position_id = ? AND approval_status = ?",
			electionID, position.ID, models.CandidateStatusApproved).
			Order("ballot_number ASC, full_name ASC").Find(&candidates).Error; err != nil {
			return "", err
		}

		posContent := models.BallotPositionContent{
			ID:            position.ID,
			Title:         position.Title,
			Description:   position.Description,
			QuestionType:  position.QuestionType,
			MinSelections: position.MinSelections,
			MaxSelections: position.MaxSelections,
			Seats:         position.Seats,
			AllowBlank:    position.AllowBlank,
			DisplayOrder:  position.DisplayOrder,
			Candidates:    make([]models.BallotCandidateContent, 0, len(candidates)),
		}

		for _, candidate := range candidates {
			posContent.Candidates = append(posContent.Candidates, models.BallotCandidateContent{
				ID:           candidate.ID,
				FullName:     candidate.FullName,
				PhotoURL:     candidate.PhotoURL,
				Manifesto:    candidate.Manifesto,
				Bio:          candidate.Bio,
				BallotNumber: candidate.BallotNumber,
			})
		}

		content.Positions = append(content.Positions, posContent)
	}

	jsonBytes, err := json.Marshal(content)
	if err != nil {
		return "", err
	}

	return string(jsonBytes), nil
}

// ValidateBallot checks if the ballot meets all publication requirements.
func (h *BallotVersionHandler) ValidateBallot(electionID string) ([]string, error) {
	var errors []string

	var positions []models.Position
	if err := h.DB.Where("election_id = ?", electionID).Find(&positions).Error; err != nil {
		return nil, err
	}

	if len(positions) == 0 {
		errors = append(errors, "Ballot must have at least one position")
	}

	for _, position := range positions {
		// Check minimum candidates
		var count int64
		if err := h.DB.Model(&models.Candidate{}).
			Where("election_id = ? AND position_id = ? AND approval_status = ?",
				electionID, position.ID, models.CandidateStatusApproved).
			Count(&count).Error; err != nil {
			return nil, err
		}

		if count == 0 {
			errors = append(errors, "Position '"+position.Title+"' has no approved candidates")
		}

		// Check selection rules
		if position.MinSelections < 0 {
			errors = append(errors, "Position '"+position.Title+"' has invalid minimum selections")
		}
		if position.MaxSelections < position.MinSelections {
			errors = append(errors, "Position '"+position.Title+"' has maximum selections less than minimum")
		}
		if position.Seats < 1 {
			errors = append(errors, "Position '"+position.Title+"' must have at least one seat")
		}

		// Check for duplicate ballot numbers
		var candidates []models.Candidate
		if err := h.DB.Where("election_id = ? AND position_id = ? AND approval_status = ?",
			electionID, position.ID, models.CandidateStatusApproved).
			Find(&candidates).Error; err != nil {
			return nil, err
		}

		seenNumbers := make(map[int]bool)
		for _, candidate := range candidates {
			if seenNumbers[candidate.BallotNumber] {
				errors = append(errors, "Position '"+position.Title+"' has duplicate ballot number "+string(rune(candidate.BallotNumber)))
			}
			seenNumbers[candidate.BallotNumber] = true
		}
	}

	return errors, nil
}

// Publish creates a new immutable ballot version.
func (h *BallotVersionHandler) Publish(c *gin.Context) {
	electionID := c.Param("id")
	userID := c.GetString("user_id")

	// Validate election exists and is in appropriate state
	var election models.Election
	if err := h.DB.Where("id = ?", electionID).First(&election).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respond.NotFound(c, "Election not found")
			return
		}
		respond.Internal(c, err)
		return
	}

	// Validate ballot structure
	validationErrors, err := h.ValidateBallot(electionID)
	if err != nil {
		respond.Internal(c, err)
		return
	}
	if len(validationErrors) > 0 {
		respond.BadRequest(c, "Ballot validation failed")
		c.JSON(400, gin.H{"errors": validationErrors})
		return
	}

	// Build content
	contentJSON, err := h.BuildBallotContent(electionID)
	if err != nil {
		respond.Internal(c, err)
		return
	}

	// Calculate version number
	var maxVersion int
	h.DB.Model(&models.BallotVersion{}).
		Where("election_id = ?", electionID).
		Select("COALESCE(MAX(version), 0)").
		Scan(&maxVersion)
	newVersion := maxVersion + 1

	// Mark previous versions as superseded
	h.DB.Model(&models.BallotVersion{}).
		Where("election_id = ? AND status = ?", electionID, models.BallotVersionStatusCurrent).
		Update("status", models.BallotVersionStatusSuperseded)

	// Create new version
	version := models.BallotVersion{
		ElectionID:  electionID,
		Version:     newVersion,
		ContentJSON: contentJSON,
		PublishedBy: userID,
		Status:      models.BallotVersionStatusCurrent,
	}

	if err := h.DB.Create(&version).Error; err != nil {
		respond.Internal(c, err)
		return
	}

	// Audit log
	audit := models.ElectionAuditLog{
		ElectionID: electionID,
		ActorID:    userID,
		Action:     models.AuditActionBallotPublished,
		Details:    "Ballot version " + string(rune(newVersion)) + " published",
		IPAddress:  c.ClientIP(),
	}
	h.DB.Create(&audit)

	respond.Created(c, gin.H{
		"id":           version.ID,
		"version":      version.Version,
		"content_hash": version.ContentHash,
		"published_at": version.PublishedAt,
		"published_by": version.PublishedBy,
	}, "Ballot version published")
}

// ListVersions returns all ballot versions for an election.
func (h *BallotVersionHandler) ListVersions(c *gin.Context) {
	electionID := c.Param("id")

	var versions []models.BallotVersion
	if err := h.DB.Where("election_id = ?", electionID).
		Order("version DESC").
		Find(&versions).Error; err != nil {
		respond.Internal(c, err)
		return
	}

	respond.OK(c, gin.H{"versions": versions})
}

// GetVersion returns a specific ballot version.
func (h *BallotVersionHandler) GetVersion(c *gin.Context) {
	id := c.Param("version_id")

	var version models.BallotVersion
	if err := h.DB.Where("id = ?", id).First(&version).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respond.NotFound(c, "Ballot version not found")
			return
		}
		respond.Internal(c, err)
		return
	}

	// Validate content integrity
	if err := version.ValidateContent(); err != nil {
		respond.Internal(c, err)
		return
	}

	content, err := version.GetContent()
	if err != nil {
		respond.Internal(c, err)
		return
	}

	respond.OK(c, gin.H{
		"version":      version,
		"content":     content,
		"content_hash": version.ContentHash,
	})
}

// CompareVersions returns a diff between two ballot versions.
func (h *BallotVersionHandler) CompareVersions(c *gin.Context) {
	versionID1 := c.Query("v1")
	versionID2 := c.Query("v2")

	if versionID1 == "" || versionID2 == "" {
		respond.BadRequest(c, "Both v1 and v2 parameters are required")
		return
	}

	var v1, v2 models.BallotVersion
	if err := h.DB.Where("id = ?", versionID1).First(&v1).Error; err != nil {
		respond.NotFound(c, "Version v1 not found")
		return
	}
	if err := h.DB.Where("id = ?", versionID2).First(&v2).Error; err != nil {
		respond.NotFound(c, "Version v2 not found")
		return
	}

	content1, err := v1.GetContent()
	if err != nil {
		respond.Internal(c, err)
		return
	}
	content2, err := v2.GetContent()
	if err != nil {
		respond.Internal(c, err)
		return
	}

	respond.OK(c, gin.H{
		"v1":       v1,
		"v1_content": content1,
		"v2":       v2,
		"v2_content": content2,
		"same_hash": v1.ContentHash == v2.ContentHash,
	})
}
