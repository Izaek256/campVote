package handlers

import (
	"errors"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"campvote/apps/api/internal/models"
	"campvote/apps/api/internal/respond"
)

// PositionHandler manages position CRUD for elections (ISSUE-06).
type PositionHandler struct {
	DB *gorm.DB
}

func NewPositionHandler(db *gorm.DB) *PositionHandler {
	return &PositionHandler{DB: db}
}

// List returns all positions for an election.
func (h *PositionHandler) List(c *gin.Context) {
	electionID := c.Param("id")

	var positions []models.Position
	if err := h.DB.Where("election_id = ?", electionID).
		Order("display_order ASC, title ASC").
		Find(&positions).Error; err != nil {
		respond.Internal(c, err)
		return
	}

	respond.OK(c, gin.H{"positions": positions})
}

// Create adds a new position to an election.
func (h *PositionHandler) Create(c *gin.Context) {
	electionID := c.Param("id")

	var input struct {
		Title         string `json:"title" binding:"required"`
		Description   string `json:"description"`
		QuestionType  string `json:"question_type" binding:"required"`
		MinSelections int    `json:"min_selections"`
		MaxSelections int    `json:"max_selections"`
		Seats         int    `json:"seats"`
		AllowBlank    bool   `json:"allow_blank"`
		DisplayOrder  int    `json:"display_order"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		respond.BadRequest(c, "Invalid request body")
		return
	}

	// Validate question type
	validTypes := map[string]bool{
		models.QuestionTypeSingleChoice: true,
		models.QuestionTypeMultiChoice:  true,
		models.QuestionTypeYesNo:        true,
		models.QuestionTypeReferendum:   true,
	}
	if !validTypes[input.QuestionType] {
		respond.BadRequest(c, "Invalid question type")
		return
	}

	// Validate selection rules
	if input.MinSelections < 0 {
		respond.BadRequest(c, "Minimum selections cannot be negative")
		return
	}
	if input.MaxSelections < input.MinSelections {
		respond.BadRequest(c, "Maximum selections must be at least minimum selections")
		return
	}
	if input.Seats < 1 {
		respond.BadRequest(c, "Seats must be at least 1")
		return
	}

	position := models.Position{
		ElectionID:    electionID,
		Title:         input.Title,
		Description:   input.Description,
		QuestionType:  input.QuestionType,
		MinSelections: input.MinSelections,
		MaxSelections: input.MaxSelections,
		Seats:         input.Seats,
		AllowBlank:    input.AllowBlank,
		DisplayOrder:  input.DisplayOrder,
	}

	if err := h.DB.Create(&position).Error; err != nil {
		respond.Internal(c, err)
		return
	}

	// Audit log
	userID := c.GetString("user_id")
	audit := models.ElectionAuditLog{
		ElectionID: electionID,
		ActorID:    userID,
		Action:     models.AuditActionPositionCreated,
		Details:    "Position created: " + position.Title,
		IPAddress:  c.ClientIP(),
	}
	h.DB.Create(&audit)

	respond.Created(c, position, "Position created")
}

// Update updates a position (only allowed in draft state).
func (h *PositionHandler) Update(c *gin.Context) {
	electionID := c.Param("id")
	positionID := c.Param("position_id")

	var position models.Position
	if err := h.DB.Where("id = ? AND election_id = ?", positionID, electionID).First(&position).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respond.NotFound(c, "Position not found")
			return
		}
		respond.Internal(c, err)
		return
	}

	// Check if election is still in draft
	var election models.Election
	if err := h.DB.Where("id = ?", electionID).First(&election).Error; err != nil {
		respond.Internal(c, err)
		return
	}
	if election.Status != models.ElectionStatusDraft {
		respond.Forbidden(c, "Cannot edit positions after election is published")
		return
	}

	var input struct {
		Title         *string `json:"title"`
		Description   *string `json:"description"`
		QuestionType  *string `json:"question_type"`
		MinSelections *int    `json:"min_selections"`
		MaxSelections *int    `json:"max_selections"`
		Seats         *int    `json:"seats"`
		AllowBlank    *bool   `json:"allow_blank"`
		DisplayOrder  *int    `json:"display_order"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		respond.BadRequest(c, "Invalid request body")
		return
	}

	updates := map[string]interface{}{}
	if input.Title != nil {
		updates["title"] = *input.Title
	}
	if input.Description != nil {
		updates["description"] = *input.Description
	}
	if input.QuestionType != nil {
		updates["question_type"] = *input.QuestionType
	}
	if input.MinSelections != nil {
		updates["min_selections"] = *input.MinSelections
	}
	if input.MaxSelections != nil {
		updates["max_selections"] = *input.MaxSelections
	}
	if input.Seats != nil {
		updates["seats"] = *input.Seats
	}
	if input.AllowBlank != nil {
		updates["allow_blank"] = *input.AllowBlank
	}
	if input.DisplayOrder != nil {
		updates["display_order"] = *input.DisplayOrder
	}

	if len(updates) > 0 {
		if err := h.DB.Model(&position).Updates(updates).Error; err != nil {
			respond.Internal(c, err)
			return
		}

		// Audit log
		userID := c.GetString("user_id")
		audit := models.ElectionAuditLog{
			ElectionID: electionID,
			ActorID:    userID,
			Action:     models.AuditActionPositionUpdated,
			Details:    "Position updated: " + position.Title,
			IPAddress:  c.ClientIP(),
		}
		h.DB.Create(&audit)
	}

	respond.OK(c, position)
}

// Delete removes a position (only allowed in draft state).
func (h *PositionHandler) Delete(c *gin.Context) {
	electionID := c.Param("id")
	positionID := c.Param("position_id")

	var position models.Position
	if err := h.DB.Where("id = ? AND election_id = ?", positionID, electionID).First(&position).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respond.NotFound(c, "Position not found")
			return
		}
		respond.Internal(c, err)
		return
	}

	// Check if election is still in draft
	var election models.Election
	if err := h.DB.Where("id = ?", electionID).First(&election).Error; err != nil {
		respond.Internal(c, err)
		return
	}
	if election.Status != models.ElectionStatusDraft {
		respond.Forbidden(c, "Cannot delete positions after election is published")
		return
	}

	if err := h.DB.Delete(&position).Error; err != nil {
		respond.Internal(c, err)
		return
	}

	// Audit log
	userID := c.GetString("user_id")
	audit := models.ElectionAuditLog{
		ElectionID: electionID,
		ActorID:    userID,
		Action:     models.AuditActionPositionDeleted,
		Details:    "Position deleted: " + position.Title,
		IPAddress:  c.ClientIP(),
	}
	h.DB.Create(&audit)

	respond.OK(c, gin.H{"message": "Position deleted"})
}
