package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"campvote/apps/api/internal/models"
	"campvote/apps/api/internal/respond"
	"campvote/apps/api/internal/services"
)

// InviteAdmin creates an administrator who has no password yet, then emails a
// one-time set-password link. Following that link proves control of the
// mailbox, so the invitation cannot be used to plant an account on an address
// the inviter does not own.
type InviteAdminRequest struct {
	FirstName string `json:"first_name" binding:"required,min=2"`
	LastName  string `json:"last_name" binding:"required,min=2"`
	Email     string `json:"email" binding:"required,email"`
}

// InviteAdmin handles POST /staff/admins/invite.
func (h *AuthHandler) InviteAdmin(c *gin.Context) {
	var req InviteAdminRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Fail(c, respond.CodeValidationError, err.Error())
		return
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))

	if !h.Config.IsAdminEmail(email) {
		respond.Fail(c, respond.CodeValidationError, "Administrators must use an @"+h.Config.AdminDomain()+" address.")
		return
	}

	user := models.User{
		FirstName: strings.TrimSpace(req.FirstName),
		LastName:  strings.TrimSpace(req.LastName),
		Email:     email,
		Role:      models.RoleAdmin,
		Active:    true,
		// No password. Sign-in refuses an empty hash, and the reset link is the
		// only way in until the invited person chooses one.
		Password: "",
	}

	if err := services.CreateUser(c.Request.Context(), h.DB, &user); err != nil {
		if errors.Is(err, services.ErrEmailExists) {
			c.JSON(http.StatusConflict, gin.H{
				"error": gin.H{
					"code":    "EMAIL_EXISTS",
					"message": "An account with this email already exists",
				},
			})
			return
		}
		respond.Internal(c, err)
		return
	}

	// The reset flow owns token creation, expiry, hashing and delivery.
	h.deliverPasswordReset(c.Request.Context(), user, c.ClientIP())

	services.LogActivity(h.DB, c, services.ActivityArgs{
		Action:       "admin.invited",
		Severity:     "info",
		Summary:      "Administrator invited: " + email,
		ResourceType: "user",
		ResourceID:   user.ID,
	})

	c.JSON(http.StatusCreated, gin.H{
		"data":    user,
		"message": "Invitation sent. The administrator will choose a password from the emailed link.",
	})
}
