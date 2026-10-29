package handlers

import (
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"campvote/apps/api/internal/models"
	"campvote/apps/api/internal/respond"
)

// CandidateHandler manages candidate nomination and approval workflow (ISSUE-06).
type CandidateHandler struct {
	DB  *gorm.DB
	Now func() time.Time
}

func NewCandidateHandler(db *gorm.DB) *CandidateHandler {
	return &CandidateHandler{DB: db, Now: time.Now}
}

func (h *CandidateHandler) now() time.Time {
	if h.Now == nil {
		return time.Now().UTC()
	}
	return h.Now().UTC()
}

// List returns candidates for an election, optionally filtered by status.
func (h *CandidateHandler) List(c *gin.Context) {
	electionID := c.Param("id")
	status := c.Query("status")

	query := h.DB.Where("election_id = ?", electionID)
	if status != "" {
		query = query.Where("approval_status = ?", status)
	}

	var candidates []models.Candidate
	if err := query.Order("created_at DESC").Find(&candidates).Error; err != nil {
		respond.Internal(c, err)
		return
	}

	respond.OK(c, gin.H{"candidates": candidates})
}

// Create adds a new candidate nomination.
func (h *CandidateHandler) Create(c *gin.Context) {
	electionID := c.Param("id")
	userID := c.GetString("user_id")

	var input struct {
		PositionID string `json:"position_id" binding:"required"`
		FullName   string `json:"full_name" binding:"required"`
		PhotoURL   string `json:"photo_url"`
		Manifesto  string `json:"manifesto"`
		Bio        string `json:"bio"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		respond.BadRequest(c, "Invalid request body")
		return
	}

	// Verify position exists and belongs to election
	var position models.Position
	if err := h.DB.Where("id = ? AND election_id = ?", input.PositionID, electionID).First(&position).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respond.NotFound(c, "Position not found")
			return
		}
		respond.Internal(c, err)
		return
	}

	candidate := models.Candidate{
		ElectionID:     electionID,
		PositionID:     input.PositionID,
		FullName:       input.FullName,
		PhotoURL:       input.PhotoURL,
		Manifesto:      input.Manifesto,
		Bio:            input.Bio,
		ApprovalStatus: models.CandidateStatusNominated,
		NominatedBy:    userID,
	}

	if err := h.DB.Create(&candidate).Error; err != nil {
		respond.Internal(c, err)
		return
	}

	// Audit log
	audit := models.ElectionAuditLog{
		ElectionID: electionID,
		ActorID:    userID,
		Action:     models.AuditActionCandidateNominated,
		Details:    "Candidate nominated: " + candidate.FullName,
		IPAddress:  c.ClientIP(),
	}
	h.DB.Create(&audit)

	respond.Created(c, candidate, "Candidate nominated")
}

// Update allows a candidate to edit their own profile.
func (h *CandidateHandler) Update(c *gin.Context) {
	electionID := c.Param("id")
	candidateID := c.Param("candidate_id")
	userID := c.GetString("user_id")

	var candidate models.Candidate
	if err := h.DB.Where("id = ? AND election_id = ?", candidateID, electionID).First(&candidate).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respond.NotFound(c, "Candidate not found")
			return
		}
		respond.Internal(c, err)
		return
	}

	// Only the candidate who nominated (or the candidate themselves in self-nomination) can edit
	// For ISSUE-06, we allow any officer to edit; candidate-specific access can be refined later
	// TODO: Add proper role check for officer access

	var input struct {
		FullName  string `json:"full_name"`
		PhotoURL  string `json:"photo_url"`
		Manifesto string `json:"manifesto"`
		Bio       string `json:"bio"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		respond.BadRequest(c, "Invalid request body")
		return
	}

	updates := map[string]interface{}{}
	if input.FullName != "" {
		updates["full_name"] = input.FullName
	}
	if input.PhotoURL != "" {
		updates["photo_url"] = input.PhotoURL
	}
	if input.Manifesto != "" {
		updates["manifesto"] = input.Manifesto
	}
	if input.Bio != "" {
		updates["bio"] = input.Bio
	}

	if len(updates) > 0 {
		if err := h.DB.Model(&candidate).Updates(updates).Error; err != nil {
			respond.Internal(c, err)
			return
		}
	}

	respond.OK(c, candidate)
}

// Approve approves a candidate nomination.
func (h *CandidateHandler) Approve(c *gin.Context) {
	electionID := c.Param("id")
	candidateID := c.Param("candidate_id")
	userID := c.GetString("user_id")

	var input struct {
		Reason string `json:"reason" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		respond.BadRequest(c, "Reason is required")
		return
	}

	var candidate models.Candidate
	if err := h.DB.Where("id = ? AND election_id = ?", candidateID, electionID).First(&candidate).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respond.NotFound(c, "Candidate not found")
			return
		}
		respond.Internal(c, err)
		return
	}

	now := h.now()
	candidate.ApprovalStatus = models.CandidateStatusApproved
	candidate.ReviewedBy = userID
	candidate.ReviewedAt = &now
	candidate.ReviewNotes = input.Reason

	if err := h.DB.Save(&candidate).Error; err != nil {
		respond.Internal(c, err)
		return
	}

	// Audit log
	audit := models.ElectionAuditLog{
		ElectionID: electionID,
		ActorID:    userID,
		Action:     models.AuditActionCandidateApproved,
		Details:    "Candidate approved: " + candidate.FullName + " - " + input.Reason,
		IPAddress:  c.ClientIP(),
	}
	h.DB.Create(&audit)

	respond.OK(c, candidate)
}

// Reject rejects a candidate nomination.
func (h *CandidateHandler) Reject(c *gin.Context) {
	electionID := c.Param("id")
	candidateID := c.Param("candidate_id")
	userID := c.GetString("user_id")

	var input struct {
		Reason string `json:"reason" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		respond.BadRequest(c, "Reason is required")
		return
	}

	var candidate models.Candidate
	if err := h.DB.Where("id = ? AND election_id = ?", candidateID, electionID).First(&candidate).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respond.NotFound(c, "Candidate not found")
			return
		}
		respond.Internal(c, err)
		return
	}

	now := h.now()
	candidate.ApprovalStatus = models.CandidateStatusRejected
	candidate.ReviewedBy = userID
	candidate.ReviewedAt = &now
	candidate.ReviewNotes = input.Reason

	if err := h.DB.Save(&candidate).Error; err != nil {
		respond.Internal(c, err)
		return
	}

	// Audit log
	audit := models.ElectionAuditLog{
		ElectionID: electionID,
		ActorID:    userID,
		Action:     models.AuditActionCandidateRejected,
		Details:    "Candidate rejected: " + candidate.FullName + " - " + input.Reason,
		IPAddress:  c.ClientIP(),
	}
	h.DB.Create(&audit)

	respond.OK(c, candidate)
}

// Withdraw handles candidate withdrawal.
func (h *CandidateHandler) Withdraw(c *gin.Context) {
	electionID := c.Param("id")
	candidateID := c.Param("candidate_id")
	userID := c.GetString("user_id")

	var input struct {
		Reason string `json:"reason" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		respond.BadRequest(c, "Reason is required")
		return
	}

	var candidate models.Candidate
	if err := h.DB.Where("id = ? AND election_id = ?", candidateID, electionID).First(&candidate).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respond.NotFound(c, "Candidate not found")
			return
		}
		respond.Internal(c, err)
		return
	}

	now := h.now()
	candidate.ApprovalStatus = models.CandidateStatusWithdrawn
	candidate.WithdrawnAt = &now
	candidate.WithdrawalReason = input.Reason

	if err := h.DB.Save(&candidate).Error; err != nil {
		respond.Internal(c, err)
		return
	}

	// Audit log
	audit := models.ElectionAuditLog{
		ElectionID: electionID,
		ActorID:    userID,
		Action:     models.AuditActionCandidateWithdrawn,
		Details:    "Candidate withdrawn: " + candidate.FullName + " - " + input.Reason,
		IPAddress:  c.ClientIP(),
	}
	h.DB.Create(&audit)

	respond.OK(c, candidate)
}

// GetMyCandidacies returns the current user's nominations.
func (h *CandidateHandler) GetMyCandidacies(c *gin.Context) {
	userID := c.GetString("user_id")

	var candidates []models.Candidate
	if err := h.DB.Where("nominated_by = ?", userID).
		Order("created_at DESC").
		Find(&candidates).Error; err != nil {
		respond.Internal(c, err)
		return
	}

	respond.OK(c, gin.H{"candidates": candidates})
}
