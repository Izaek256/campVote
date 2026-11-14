package handlers

import (
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"campvote/apps/api/internal/models"
	"campvote/apps/api/internal/respond"
	"campvote/apps/api/internal/services"
)

// ResidenceHandler owns the administrator-managed list of campus residences
// and the one correction an administrator may make to a student profile.
//
// The table starts empty and is never seeded with invented places. Removing an
// area deactivates it and clears the assignments that pointed at it, in one
// transaction, so a profile cannot keep a dangling reference.
type ResidenceHandler struct {
	DB *gorm.DB
}

func NewResidenceHandler(db *gorm.DB) *ResidenceHandler {
	return &ResidenceHandler{DB: db}
}

// List returns the active areas, in name order, for the profile form.
func (h *ResidenceHandler) List(c *gin.Context) {
	var areas []models.ResidenceArea
	if err := h.DB.Where("active = ?", true).Order("name").Find(&areas).Error; err != nil {
		respond.Internal(c, err)
		return
	}
	respond.OK(c, areas)
}

type createResidenceRequest struct {
	Name string `json:"name"`
}

// Create adds an area. The unique index on the name is the duplicate check.
func (h *ResidenceHandler) Create(c *gin.Context) {
	var in createResidenceRequest
	if err := c.ShouldBindJSON(&in); err != nil {
		respond.BadRequest(c, "A residence name is required")
		return
	}
	name := strings.TrimSpace(in.Name)
	if len([]rune(name)) < 2 || len([]rune(name)) > 120 {
		respond.BadRequest(c, "A residence name must be between 2 and 120 characters")
		return
	}

	area := models.ResidenceArea{Name: name, Active: true}
	if err := h.DB.Create(&area).Error; err != nil {
		respond.WriteError(c, err, "The residence could not be added")
		return
	}
	services.LogCreate(h.DB, c, "residence_area", area.Name, area.ID, "Added residence area")
	respond.Created(c, area, "Residence area added")
}

// Deactivate removes an area from the profile form without deleting history.
// Every profile pointing at it is cleared in the same transaction.
func (h *ResidenceHandler) Deactivate(c *gin.Context) {
	id := c.Param("id")
	var area models.ResidenceArea
	if err := h.DB.Where("id = ?", id).First(&area).Error; err != nil {
		respond.NotFound(c, "Residence area not found")
		return
	}
	if !area.Active {
		respond.OK(c, area, "Residence area is already inactive")
		return
	}

	var cleared int64
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.ResidenceArea{}).Where("id = ?", area.ID).
			Update("active", false).Error; err != nil {
			return err
		}
		result := tx.Model(&models.User{}).Where("residence_area_id = ?", area.ID).
			Update("residence_area_id", nil)
		cleared = result.RowsAffected
		return result.Error
	})
	if err != nil {
		respond.Internal(c, err)
		return
	}

	services.LogDelete(h.DB, c, "residence_area", area.Name, area.ID)
	respond.OK(c, gin.H{"id": area.ID, "cleared_profiles": cleared}, "Residence area deactivated")
}

type correctResidenceRequest struct {
	Email           string  `json:"email"`
	ResidenceAreaID *string `json:"residence_area_id"`
}

// CorrectResidence is the only profile field an administrator edits: the
// student's age and sex stay as the student supplied them. The student is named
// by email so the console needs no id lookup.
func (h *ResidenceHandler) CorrectResidence(c *gin.Context) {
	var in correctResidenceRequest
	if err := c.ShouldBindJSON(&in); err != nil {
		respond.BadRequest(c, "A student email and residence area are required")
		return
	}
	email := strings.ToLower(strings.TrimSpace(in.Email))
	var user models.User
	if err := h.DB.Where("LOWER(email) = ?", email).First(&user).Error; err != nil {
		respond.NotFound(c, "User not found")
		return
	}

	var area models.ResidenceArea
	if in.ResidenceAreaID != nil && *in.ResidenceAreaID != "" {
		if err := h.DB.Where("id = ? AND active = ?", *in.ResidenceAreaID, true).
			First(&area).Error; err != nil {
			respond.BadRequest(c, "That residence area is not available")
			return
		}
	}

	updates := map[string]interface{}{"residence_area_id": nil}
	if area.ID != "" {
		updates["residence_area_id"] = area.ID
	}
	result := h.DB.Model(&models.User{}).Where("id = ?", user.ID).Updates(updates)
	if result.Error != nil {
		respond.Internal(c, result.Error)
		return
	}

	services.LogUpdate(h.DB, c, "user", user.ID, user.ID, "Corrected residence area")
	respond.OK(c, gin.H{"user_id": user.ID, "residence_area_id": updates["residence_area_id"]}, "Residence updated")
}
