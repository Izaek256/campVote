package handlers

import (
	"errors"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"campvote/apps/api/internal/authz"
	"campvote/apps/api/internal/models"
	"campvote/apps/api/internal/policy"
	"campvote/apps/api/internal/respond"
	"campvote/apps/api/internal/services"
)

// ElectionRBACHandler owns the election-scoped role assignments and the two
// result operations whose visibility is a policy rather than a fixed rule.
//
// Every method resolves the election through policy.AuthorityFor first. That is
// what turns a cross-election request into 404 rather than 403: the caller is
// not allowed to know the other election exists.
type ElectionRBACHandler struct {
	DB *gorm.DB
}

func NewElectionRBACHandler(db *gorm.DB) *ElectionRBACHandler {
	return &ElectionRBACHandler{DB: db}
}

type electionRoleView struct {
	UserID    string    `json:"user_id"`
	Email     string    `json:"email,omitempty"`
	FullName  string    `json:"full_name,omitempty"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

// ListRoles returns the per-election assignments.
func (h *ElectionRBACHandler) ListRoles(c *gin.Context) {
	authority, ok := h.authority(c, policy.PermElectionRoles)
	if !ok {
		return
	}

	var rows []models.ElectionRole
	if err := h.DB.Where("election_id = ?", authority.Election.ID).
		Order("role ASC, user_id ASC").Find(&rows).Error; err != nil {
		respond.Internal(c, err)
		return
	}
	out := make([]electionRoleView, 0, len(rows))
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.UserID)
	}
	users := map[string]models.User{}
	if len(ids) > 0 {
		var list []models.User
		if err := h.DB.Where("id IN ?", ids).Find(&list).Error; err != nil {
			respond.Internal(c, err)
			return
		}
		for _, user := range list {
			users[user.ID] = user
		}
	}
	for _, row := range rows {
		user := users[row.UserID]
		out = append(out, electionRoleView{
			UserID: row.UserID, Email: user.Email,
			FullName: strings.TrimSpace(user.FirstName + " " + user.LastName),
			Role:     row.Role, CreatedAt: row.CreatedAt,
		})
	}
	respond.OK(c, out)
}

// LookupUser resolves an exact email to a user id so the role panel can assign
// a role without asking an official to copy database identifiers around. The
// lookup is scoped to an election the caller can already manage roles for.
func (h *ElectionRBACHandler) LookupUser(c *gin.Context) {
	authority, ok := h.authority(c, policy.PermElectionRoles)
	if !ok {
		return
	}
	email := strings.ToLower(strings.TrimSpace(c.Query("email")))
	if email == "" {
		respond.BadRequest(c, "An email address is required")
		return
	}
	var user models.User
	if err := h.DB.Where("LOWER(email) = ?", email).First(&user).Error; err != nil {
		respond.NotFound(c, "No account uses that email address")
		return
	}
	respond.OK(c, gin.H{
		"user_id":     user.ID,
		"email":       user.Email,
		"full_name":   strings.TrimSpace(user.FirstName + " " + user.LastName),
		"election_id": authority.Election.ID,
	})
}

type assignElectionRoleRequest struct {
	Role string `json:"role" binding:"required"`
}

// AssignRole grants one SRS role for one election.
func (h *ElectionRBACHandler) AssignRole(c *gin.Context) {
	authority, ok := h.authority(c, policy.PermElectionRoles)
	if !ok {
		return
	}

	var in assignElectionRoleRequest
	if err := c.ShouldBindJSON(&in); err != nil {
		respond.BadRequest(c, "A role is required")
		return
	}
	role := strings.TrimSpace(in.Role)
	if !policy.ValidElectionRole(role) {
		respond.BadRequest(c, "Unknown election role")
		return
	}

	userID := c.Param("userId")
	var target models.User
	if err := h.DB.Where("id = ?", userID).First(&target).Error; err != nil {
		// Same answer for an unknown account and one outside this election.
		respond.NotFound(c, "User not found")
		return
	}

	var existing []models.ElectionRole
	if err := h.DB.Where("election_id = ? AND user_id = ?", authority.Election.ID, userID).
		Find(&existing).Error; err != nil {
		respond.Internal(c, err)
		return
	}
	roles := make([]string, 0, len(existing))
	for _, row := range existing {
		if row.Role == role {
			respond.OK(c, electionRoleView{UserID: userID, Role: role, CreatedAt: row.CreatedAt}, "Role already assigned")
			return
		}
		roles = append(roles, row.Role)
	}
	if reason := policy.RoleAssignmentConflict(role, roles); reason != "" {
		policy.LogDenial(h.DB, c, reason, authority.Election.ID)
		respond.Forbidden(c, reason)
		return
	}

	assignment := models.ElectionRole{
		ElectionID:   authority.Election.ID,
		UserID:       userID,
		Role:         role,
		AssignedByID: c.GetString("user_id"),
	}
	if err := h.DB.Create(&assignment).Error; err != nil {
		respond.Internal(c, err)
		return
	}
	// The assignment is read per request, but dropping the cache makes a
	// global grant change visible immediately too.
	authz.Invalidate()
	h.audit(c, authority.Election.ID, "election.role_assigned",
		"Assigned "+role+" to "+target.Email+" for "+authority.Election.Title)
	respond.Created(c, electionRoleView{UserID: userID, Role: role, CreatedAt: assignment.CreatedAt}, "Role assigned")
}

// RemoveRole removes one per-election assignment.
func (h *ElectionRBACHandler) RemoveRole(c *gin.Context) {
	authority, ok := h.authority(c, policy.PermElectionRoles)
	if !ok {
		return
	}
	role := c.Param("role")
	if !policy.ValidElectionRole(role) {
		respond.BadRequest(c, "Unknown election role")
		return
	}

	result := h.DB.Where("election_id = ? AND user_id = ? AND role = ?",
		authority.Election.ID, c.Param("userId"), role).Delete(&models.ElectionRole{})
	if result.Error != nil {
		respond.Internal(c, result.Error)
		return
	}
	if result.RowsAffected == 0 {
		respond.NotFound(c, "Role assignment not found")
		return
	}
	authz.Invalidate()
	h.audit(c, authority.Election.ID, "election.role_removed",
		"Removed "+role+" for "+authority.Election.Title)
	respond.OK(c, gin.H{"removed": true})
}

// authority resolves the election and the permission, writing the refusal.
func (h *ElectionRBACHandler) authority(c *gin.Context, permission string) (*policy.Authority, bool) {
	actor := policy.ActorFrom(c)
	authority, err := policy.AuthorityFor(h.DB, actor, c.Param("id"))
	if err != nil {
		if errors.Is(err, policy.ErrNotVisible) {
			respond.NotFound(c, "Election not found")
			return nil, false
		}
		respond.Internal(c, err)
		return nil, false
	}
	if !authority.Can(permission) {
		policy.LogDenial(h.DB, c, "requires "+permission, authority.Election.ID)
		respond.Forbidden(c, "You do not have permission to manage this election")
		return nil, false
	}
	return authority, true
}

func (h *ElectionRBACHandler) audit(c *gin.Context, electionID, action, summary string) {
	services.LogActivity(h.DB, c, services.ActivityArgs{
		Action:       action,
		Severity:     "info",
		Summary:      summary,
		ResourceType: "election",
		ResourceID:   electionID,
	})
}
