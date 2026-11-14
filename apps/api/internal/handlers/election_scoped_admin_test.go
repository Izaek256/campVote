package handlers

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"campvote/apps/api/internal/models"
)

// A staff account that is not a global ADMIN can still use email and password
// when it has been assigned to administer an election. The current-user
// payload names that assignment so the client can offer the staff area for
// exactly that election.
func TestScopedStaffAdminSignsInWithPasswordAndSeesTheirAssignment(t *testing.T) {
	db := newTestDB(t)
	require.NoError(t, db.AutoMigrate(&models.Election{}, &models.ElectionRole{}))
	cfg := testCfg()
	h := &AuthHandler{DB: db, AuthService: newTestAuthSvc(cfg, db), Config: cfg}
	r := securityRouter(h, db)

	verified := time.Now()
	creator := models.User{
		FirstName: "Global", LastName: "Admin", Email: "creator@example.com",
		Password: "harbour-lamp-97", Role: models.RoleAdmin, Active: true, EmailVerifiedAt: &verified,
	}
	require.NoError(t, db.Create(&creator).Error)
	staff := models.User{
		FirstName: "Election", LastName: "Staff", Email: "staff@example.com",
		Password: "harbour-lamp-97", Role: models.RoleElectionOfficial, Active: true, EmailVerifiedAt: &verified,
	}
	require.NoError(t, db.Create(&staff).Error)

	election := models.Election{
		ID: "e-scoped", Title: "Scoped election", Status: models.ElectionStatusOpen,
		StartsAt: verified.Add(-time.Hour), EndsAt: verified.Add(time.Hour),
		ResultPolicy: models.ResultPolicyAfterClose, CreatedByID: creator.ID,
	}
	require.NoError(t, db.Create(&election).Error)
	require.NoError(t, db.Create(&models.ElectionRole{
		ElectionID: election.ID, UserID: staff.ID, Role: models.RoleChiefElectionAdmin,
	}).Error)

	tokens := loginTokens(t, r, staff.Email, "harbour-lamp-97")
	w := secRequest(t, r, http.MethodGet, "/api/me", nil, func(req *http.Request) {
		req.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var body struct {
		Data struct {
			Role          string                `json:"role"`
			ElectionRoles []models.ElectionRole `json:"election_roles"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, models.RoleElectionOfficial, body.Data.Role, "the global role stays scoped")
	require.Len(t, body.Data.ElectionRoles, 1)
	assert.Equal(t, election.ID, body.Data.ElectionRoles[0].ElectionID)
	assert.Equal(t, models.RoleChiefElectionAdmin, body.Data.ElectionRoles[0].Role)
}
