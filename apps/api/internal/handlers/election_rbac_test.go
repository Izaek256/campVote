package handlers_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"campvote/apps/api/internal/authz"
	"campvote/apps/api/internal/handlers"
	"campvote/apps/api/internal/middleware"
	"campvote/apps/api/internal/models"
	"campvote/apps/api/internal/policy"
	"campvote/apps/api/internal/testkit"
)

func rbacDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(models.Models()...); err != nil {
		t.Fatal(err)
	}
	if err := models.SeedRoles(db); err != nil {
		t.Fatal(err)
	}
	return db
}

func rbacUser(t *testing.T, db *gorm.DB, email, role string) *models.User {
	t.Helper()
	u := &models.User{FirstName: "Test", LastName: "User", Email: email, Password: testkit.Password, Role: role, Active: true}
	if err := db.Create(u).Error; err != nil {
		t.Fatal(err)
	}
	return u
}

func rbacElection(t *testing.T, db *gorm.DB, id, status, resultPolicy, creator string) {
	t.Helper()
	now := time.Now().UTC()
	if err := db.Create(&models.Election{
		ID: id, Title: id, Status: status, ResultPolicy: resultPolicy,
		StartsAt: now.Add(-time.Hour), EndsAt: now.Add(time.Hour), CreatedByID: creator,
	}).Error; err != nil {
		t.Fatal(err)
	}
}

func rbacAssign(t *testing.T, db *gorm.DB, electionID, userID, role string) {
	t.Helper()
	if err := db.Create(&models.ElectionRole{ElectionID: electionID, UserID: userID, Role: role}).Error; err != nil {
		t.Fatal(err)
	}
}

func rbacEngine(t *testing.T, db *gorm.DB) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	cfg := testkit.Config()
	auth := testkit.Auth(cfg, db)
	h := handlers.NewElectionRBACHandler(db)

	r := gin.New()
	r.Use(policy.Declared())
	protected := r.Group("")
	protected.Use(middleware.APIKeyOrAuth(db, middleware.Auth(db, auth)))
	protected.Use(policy.Guard(db))
	protected.GET("/api/v1/elections/:id/roles", h.ListRoles)
	protected.PUT("/api/v1/elections/:id/roles/:userId", h.AssignRole)
	protected.DELETE("/api/v1/elections/:id/roles/:userId/:role", h.RemoveRole)
	// Results routes (ISSUE-09), served from the protected group so a voter
	// reaches the same endpoint an officer does.
	results := handlers.NewResultsHandler(db)
	protected.GET("/api/v1/elections/:id/results", results.Read)
	protected.GET("/api/v1/elections/:id/results/history", results.History)
	protected.POST("/api/v1/elections/:id/results/tally", results.RunTally)
	protected.POST("/api/v1/elections/:id/results/recount", results.Recount)
	protected.POST("/api/v1/elections/:id/results/certify", results.Certify)
	// An admin endpoint, to prove a voter token cannot reach it.
	protected.GET("/api/v1/users", func(c *gin.Context) { c.Status(http.StatusOK) })
	return r
}

func TestRoutePolicyMatrixForElectionEndpoints(t *testing.T) {
	db := rbacDB(t)
	cfg := testkit.Config()
	auth := testkit.Auth(cfg, db)
	engine := rbacEngine(t, db)

	admin := rbacUser(t, db, "admin@must.ac.ug", models.RoleAdmin)
	official := rbacUser(t, db, "official@must.ac.ug", models.RoleElectionOfficial)
	official2 := rbacUser(t, db, "official2@must.ac.ug", models.RoleElectionOfficial)
	candidate := rbacUser(t, db, "candidate@std.must.ac.ug", models.RoleCandidate)
	voter := rbacUser(t, db, "voter@std.must.ac.ug", models.RoleStudentVoter)

	rbacElection(t, db, "e-a", models.ElectionStatusOpen, models.ResultPolicyLive, "creator")
	rbacElection(t, db, "e-b", models.ElectionStatusOpen, models.ResultPolicyLive, "creator")
	rbacElection(t, db, "e-c", models.ElectionStatusClosed, models.ResultPolicyAfterClose, "creator")
	rbacElection(t, db, "e-d", models.ElectionStatusClosed, models.ResultPolicyCertifiedOnly, "creator")
	rbacAssign(t, db, "e-a", official.ID, models.RoleElectionOfficial)
	rbacAssign(t, db, "e-a", candidate.ID, models.RoleCandidate)
	rbacAssign(t, db, "e-c", candidate.ID, models.RoleCandidate)
	rbacAssign(t, db, "e-d", official.ID, models.RoleElectionOfficial)
	rbacAssign(t, db, "e-d", official2.ID, models.RoleElectionOfficial)

	adminToken := testkit.SignIn(t, db, auth, admin)
	officialToken := testkit.SignIn(t, db, auth, official)
	candidateToken := testkit.SignIn(t, db, auth, candidate)
	voterToken := testkit.SignIn(t, db, auth, voter)

	client := testkit.New(t, engine)
	cases := []struct {
		name   string
		token  string
		method string
		path   string
		status int
	}{
		{"unauthenticated results", "", http.MethodGet, "/api/v1/elections/e-a/results", http.StatusUnauthorized},
		{"admin voter token reaches admin route", voterToken, http.MethodGet, "/api/v1/users", http.StatusForbidden},
		{"admin reaches admin route", adminToken, http.MethodGet, "/api/v1/users", http.StatusOK},
		{"voter cannot read provisional results", voterToken, http.MethodGet, "/api/v1/elections/e-a/results", http.StatusForbidden},
		{"candidate cannot read provisional results", candidateToken, http.MethodGet, "/api/v1/elections/e-a/results", http.StatusForbidden},
		{"candidate reads published results after close", candidateToken, http.MethodGet, "/api/v1/elections/e-c/results", http.StatusOK},
		{"official reads provisional results", officialToken, http.MethodGet, "/api/v1/elections/e-a/results", http.StatusOK},
		{"official gets 404 on another election", officialToken, http.MethodGet, "/api/v1/elections/e-b/results", http.StatusNotFound},
		{"admin reads results", adminToken, http.MethodGet, "/api/v1/elections/e-b/results", http.StatusOK},
		{"candidate cannot list roles", candidateToken, http.MethodGet, "/api/v1/elections/e-a/roles", http.StatusForbidden},
		{"official cannot assign roles", officialToken, http.MethodPut, "/api/v1/elections/e-a/roles/" + voter.ID, http.StatusForbidden},
		{"admin cannot read certified-only before certification", adminToken, http.MethodGet, "/api/v1/elections/e-d/results", http.StatusForbidden},
		{"official cannot certify", officialToken, http.MethodPost, "/api/v1/elections/e-d/results/certify", http.StatusForbidden},
		{"certify requires a snapshot id", adminToken, http.MethodPost, "/api/v1/elections/e-d/results/certify", http.StatusNotFound},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := client
			if tc.token != "" {
				c = c.As(tc.token)
			}
			var body any
			if tc.method == http.MethodPut {
				body = map[string]string{"role": models.RoleElectionOfficial}
			}
			if tc.path == "/api/v1/elections/e-d/results/certify" {
				// Certification names a snapshot and always states a reason.
				body = map[string]string{"snapshot_id": "does-not-exist", "reason": "checked the count"}
			}
			c.Do(tc.method, tc.path, body).AssertStatus(tc.status)
		})
	}

	// Certification pins a specific snapshot, so a tally runs first. The
	// snapshot is what the certifier's decision is recorded against.
	client.As(adminToken).Post("/api/v1/elections/e-d/results/tally", nil).AssertCreated()
	var history struct {
		Snapshots []struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"snapshots"`
	}
	client.As(adminToken).Get("/api/v1/elections/e-d/results/history").
		AssertOK().BindData(&history)
	if len(history.Snapshots) == 0 || history.Snapshots[0].ID == "" {
		t.Fatal("expected a tally snapshot on the results history")
	}
	if history.Snapshots[0].Status != models.ResultStatusProvisional {
		t.Errorf("first snapshot status = %q, want PROVISIONAL", history.Snapshots[0].Status)
	}

	// Certified-only: still forbidden before certification, for everyone.
	client.As(adminToken).Get("/api/v1/elections/e-d/results").
		AssertStatus(http.StatusForbidden)

	client.As(adminToken).Post("/api/v1/elections/e-d/results/certify", map[string]string{
		"snapshot_id": history.Snapshots[0].ID,
		"reason":      "count checked against the roster",
	}).AssertOK()

	// Once certified, the same election is readable and labelled CERTIFIED.
	var certified struct {
		Label    string `json:"label"`
		Snapshot struct {
			Status        string `json:"status"`
			CertifyReason string `json:"certify_reason"`
			ContentHash   string `json:"content_hash"`
		} `json:"snapshot"`
	}
	client.As(adminToken).Get("/api/v1/elections/e-d/results").AssertOK().BindData(&certified)
	if certified.Label != "CERTIFIED" {
		t.Errorf("certified results label = %q, want CERTIFIED", certified.Label)
	}
	if certified.Snapshot.Status != models.ResultStatusCertified || certified.Snapshot.ContentHash == "" {
		t.Errorf("certified snapshot = %+v, want CERTIFIED status with a content hash", certified.Snapshot)
	}

	// Denials are auditable, with the actor and the route, never a ballot.
	var denied int64
	db.Model(&models.UserActivity{}).Where("user_id = ? AND action = ?", voter.ID, "authz.denied").Count(&denied)
	if denied == 0 {
		t.Error("a denied attempt should write an authz.denied activity row")
	}
}

func TestRoleChangeAppliesToExistingSessionAndCrossElection(t *testing.T) {
	db := rbacDB(t)
	cfg := testkit.Config()
	auth := testkit.Auth(cfg, db)
	engine := rbacEngine(t, db)

	chief := rbacUser(t, db, "chief@must.ac.ug", models.RoleChiefElectionAdmin)
	target := rbacUser(t, db, "target@must.ac.ug", models.RoleElectionOfficial)
	rbacElection(t, db, "e-a", models.ElectionStatusOpen, models.ResultPolicyLive, "creator")
	rbacElection(t, db, "e-b", models.ElectionStatusOpen, models.ResultPolicyLive, "creator")

	chiefToken := testkit.SignIn(t, db, auth, chief)
	// The target signs in BEFORE the role is granted. The same token must see
	// the new authority on its next request, not after signing in again.
	targetToken := testkit.SignIn(t, db, auth, target)
	client := testkit.New(t, engine)

	client.As(targetToken).Get("/api/v1/elections/e-a/results").AssertStatus(http.StatusNotFound)

	client.As(chiefToken).Put("/api/v1/elections/e-a/roles/"+target.ID, map[string]string{
		"role": models.RoleElectionOfficial,
	}).AssertStatus(http.StatusCreated)

	client.As(targetToken).Get("/api/v1/elections/e-a/results").AssertStatus(http.StatusOK)
	client.As(targetToken).Get("/api/v1/elections/e-b/results").AssertStatus(http.StatusNotFound)

	// Separation of duties: a candidate cannot be made an official in the same
	// election they stand in.
	candidate := rbacUser(t, db, "cand@std.must.ac.ug", models.RoleCandidate)
	rbacAssign(t, db, "e-a", candidate.ID, models.RoleCandidate)
	client.As(chiefToken).Put("/api/v1/elections/e-a/roles/"+candidate.ID, map[string]string{
		"role": models.RoleElectionOfficial,
	}).AssertStatus(http.StatusForbidden)

	// Removing the role takes effect on the next request with the old token.
	client.As(chiefToken).Delete("/api/v1/elections/e-a/roles/" + target.ID + "/" + models.RoleElectionOfficial).
		AssertStatus(http.StatusOK)
	client.As(targetToken).Get("/api/v1/elections/e-a/results").AssertStatus(http.StatusNotFound)

	// The grant cache is invalidated as part of the write; keep the assertion so
	// a future refactor cannot quietly reintroduce stale authorization.
	authz.Invalidate()
}
