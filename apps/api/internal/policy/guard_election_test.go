package policy_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"campvote/apps/api/internal/authz"
	"campvote/apps/api/internal/models"
	"campvote/apps/api/internal/policy"
)

// The route guard is the first gate. An election-scoped admin has no global
// grant, so the guard must admit them on their own election and nowhere else.
func TestGuardAdmitsScopedAdminOnlyForTheirElection(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := testDB(t)
	user(t, db, "scoped", models.RoleStudentVoter)
	user(t, db, "outsider", models.RoleStudentVoter)
	election(t, db, "e-a", models.ElectionStatusOpen, models.ResultPolicyAfterClose, "creator")
	election(t, db, "e-b", models.ElectionStatusOpen, models.ResultPolicyAfterClose, "creator")
	assign(t, db, "e-a", "scoped", models.RoleChiefElectionAdmin)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		id := c.GetHeader("X-Test-User")
		grants, _ := authz.GrantsFor(db, id)
		c.Set("user_id", id)
		c.Set("user_role", models.RoleStudentVoter)
		c.Set("user_grants", grants)
		c.Next()
	})
	r.Use(policy.Declared())
	r.Use(policy.Guard(db))
	r.GET("/api/v1/elections/:id", func(c *gin.Context) { c.Status(http.StatusOK) })

	cases := []struct {
		name     string
		user     string
		election string
		status   int
	}{
		{"assigned election", "scoped", "e-a", http.StatusOK},
		{"another election", "scoped", "e-b", http.StatusForbidden},
		{"no assignment", "outsider", "e-a", http.StatusForbidden},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/elections/"+tc.election, nil)
			req.Header.Set("X-Test-User", tc.user)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != tc.status {
				t.Fatalf("status = %d, want %d (%s)", w.Code, tc.status, w.Body.String())
			}
		})
	}
}

// A creator administers the election they made even without a scoped
// assignment, but that authority does not reach another election.
func TestCreatorAdministersTheirElectionWithoutAnAssignment(t *testing.T) {
	db := testDB(t)
	user(t, db, "creator", models.RoleElectionOfficial)
	user(t, db, "other", models.RoleElectionOfficial)
	election(t, db, "e-a", models.ElectionStatusOpen, models.ResultPolicyAfterClose, "creator")
	election(t, db, "e-b", models.ElectionStatusOpen, models.ResultPolicyAfterClose, "other")

	a, err := policy.AuthorityFor(db, actor(t, db, "creator"), "e-a")
	if err != nil {
		t.Fatalf("creator election: %v", err)
	}
	if a.Global {
		t.Error("a creator must not become a global administrator")
	}
	if !a.Can(policy.PermElectionRoles) {
		t.Error("the creator should be able to assign election admins")
	}
	if _, err := policy.AuthorityFor(db, actor(t, db, "creator"), "e-b"); err == nil {
		t.Error("the creator reached an election they do not administer")
	}
}
