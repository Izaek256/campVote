package handlers_test

import (
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"campvote/apps/api/internal/handlers"
	"campvote/apps/api/internal/middleware"
	"campvote/apps/api/internal/models"
	"campvote/apps/api/internal/policy"
	"campvote/apps/api/internal/testkit"
)

// positionEngine mounts the standalone positions-first routes with the same
// middleware the API runs: declaration check, authentication, then the guard
// that enforces the declared policy. A student reaches only the eligible list
// and the apply action; everything else sits behind RequireStaff.
func positionEngine(t *testing.T, db *gorm.DB) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	auth := testkit.Auth(testkit.Config(), db)
	h := handlers.NewPositionHandler(db)

	r := gin.New()
	r.Use(policy.Declared())

	protected := r.Group("")
	protected.Use(middleware.APIKeyOrAuth(db, middleware.Auth(db, auth)))
	protected.Use(policy.Guard(db))
	protected.GET("/api/v1/positions/eligible", h.Eligible)
	protected.POST("/api/v1/positions/:position_id/apply", h.Apply)

	staff := r.Group("")
	staff.Use(middleware.APIKeyOrAuth(db, middleware.Auth(db, auth)))
	staff.Use(middleware.RequireStaff())
	staff.Use(policy.Guard(db))
	staff.GET("/api/v1/positions", h.ListAll)
	staff.POST("/api/v1/positions", h.CreateStandalone)
	staff.PUT("/api/v1/positions/:position_id", h.UpdateStandalone)
	staff.DELETE("/api/v1/positions/:position_id", h.DeleteStandalone)
	staff.GET("/api/v1/positions/:position_id/applications", h.Applications)
	staff.POST("/api/v1/positions/:position_id/applications/:candidate_id/approve", h.ApproveApplication)
	staff.POST("/api/v1/positions/:position_id/applications/:candidate_id/reject", h.RejectApplication)
	staff.POST("/api/v1/positions/:position_id/link", h.Link)
	return r
}

// positionApplicationPayload is a complete, valid student application. The
// apply path requires a date of birth, sex, a passport photo and a partial
// transcript, so a name alone is no longer accepted.
func positionApplicationPayload(name string) map[string]any {
	return map[string]any{
		"full_name":              name,
		"date_of_birth":          "2003-05-14",
		"sex":                    "MALE",
		"passport_photo_url":     "/uploads/passport.jpg",
		"partial_transcript_url": "/uploads/transcript.pdf",
	}
}

// openPosition writes a standalone position that accepts applications and
// admits only BSE students on the student domain.
func openPosition(t *testing.T, db *gorm.DB, title string, maxCandidates int) *models.Position {
	t.Helper()
	position := &models.Position{
		Title:          title,
		QuestionType:   models.QuestionTypeSingleChoice,
		Status:         models.PositionStatusOpen,
		MinSelections:  1,
		MaxSelections:  1,
		Seats:          1,
		MaxCandidates:  maxCandidates,
		AllowedDomain:  "std.mast.ac.ug",
		AllowedCourses: "bse",
	}
	if err := db.Create(position).Error; err != nil {
		t.Fatal(err)
	}
	return position
}

func TestEligiblePositionsAreScopedToTheStudent(t *testing.T) {
	db := rbacDB(t)
	openPosition(t, db, "Guild President", 0)
	// A position that has been closed to new applications must not appear.
	closed := openPosition(t, db, "Sports Minister", 0)
	if err := db.Model(closed).Update("status", models.PositionStatusClosed).Error; err != nil {
		t.Fatal(err)
	}

	eligible := testkit.User(t, db, "2022bse001@std.mast.ac.ug", models.RoleStudentVoter)
	outsider := testkit.User(t, db, "2022bba001@std.mast.ac.ug", models.RoleStudentVoter)

	client := testkit.New(t, positionEngine(t, db))
	eligibleToken := testkit.SignIn(t, db, testkit.Auth(testkit.Config(), db), eligible)
	outsiderToken := testkit.SignIn(t, db, testkit.Auth(testkit.Config(), db), outsider)

	var list struct {
		Positions []struct {
			ID      string `json:"id"`
			Applied bool   `json:"applied"`
		} `json:"positions"`
	}
	client.As(eligibleToken).Get("/api/v1/positions/eligible").AssertOK().BindData(&list)
	if len(list.Positions) != 1 || list.Positions[0].Applied {
		t.Fatalf("eligible list = %+v, want one unapplied position", list.Positions)
	}

	// The BBA student does not match the BSE-only course list.
	client.As(outsiderToken).Get("/api/v1/positions/eligible").
		AssertOK().JSON().Count("data.positions", 0)
}

func TestApplyToPositionChecksEligibilityAndRejectsDuplicates(t *testing.T) {
	db := rbacDB(t)
	position := openPosition(t, db, "Guild President", 0)

	eligible := testkit.User(t, db, "2022bse001@std.mast.ac.ug", models.RoleStudentVoter)
	outsider := testkit.User(t, db, "2022bba001@std.mast.ac.ug", models.RoleStudentVoter)

	client := testkit.New(t, positionEngine(t, db))
	eligibleToken := testkit.SignIn(t, db, testkit.Auth(testkit.Config(), db), eligible)
	outsiderToken := testkit.SignIn(t, db, testkit.Auth(testkit.Config(), db), outsider)

	var created models.Candidate
	client.As(eligibleToken).Post("/api/v1/positions/"+position.ID+"/apply",
		positionApplicationPayload("Synthetic Applicant")).AssertCreated().BindData(&created)
	if created.ApprovalStatus != models.CandidateStatusNominated || created.Programme != "BSE" {
		t.Fatalf("application = %s/%s, want NOMINATED/BSE", created.ApprovalStatus, created.Programme)
	}

	// One live application per applicant per position.
	client.As(eligibleToken).Post("/api/v1/positions/"+position.ID+"/apply",
		positionApplicationPayload("Synthetic Applicant")).AssertConflict()

	// A student outside the position's rule is refused by the server.
	client.As(outsiderToken).Post("/api/v1/positions/"+position.ID+"/apply",
		positionApplicationPayload("Not Eligible")).AssertForbidden()
}

func TestApprovalEnforcesMaxCandidates(t *testing.T) {
	db := rbacDB(t)
	position := openPosition(t, db, "Guild President", 1)

	first := testkit.User(t, db, "2022bse001@std.mast.ac.ug", models.RoleStudentVoter)
	second := testkit.User(t, db, "2022bse002@std.mast.ac.ug", models.RoleStudentVoter)
	admin := testkit.User(t, db, "registrar@must.ac.ug", models.RoleAdmin)

	client := testkit.New(t, positionEngine(t, db))
	firstToken := testkit.SignIn(t, db, testkit.Auth(testkit.Config(), db), first)
	secondToken := testkit.SignIn(t, db, testkit.Auth(testkit.Config(), db), second)
	adminToken := testkit.SignIn(t, db, testkit.Auth(testkit.Config(), db), admin)

	var a, b models.Candidate
	client.As(firstToken).Post("/api/v1/positions/"+position.ID+"/apply",
		positionApplicationPayload("First Applicant")).AssertCreated().BindData(&a)
	client.As(secondToken).Post("/api/v1/positions/"+position.ID+"/apply",
		positionApplicationPayload("Second Applicant")).AssertCreated().BindData(&b)

	client.As(adminToken).Post("/api/v1/positions/"+position.ID+"/applications/"+a.ID+"/approve",
		map[string]any{"reason": "Documents verified"}).AssertOK()

	// The second approval breaches the cap and is refused by the server.
	client.As(adminToken).Post("/api/v1/positions/"+position.ID+"/applications/"+b.ID+"/approve",
		map[string]any{"reason": "Too many"}).AssertConflict()

	var approved int64
	db.Model(&models.Candidate{}).
		Where("position_id = ? AND approval_status = ?", position.ID, models.CandidateStatusApproved).
		Count(&approved)
	if approved != 1 {
		t.Fatalf("approved = %d, want 1", approved)
	}
}

func TestLinkingPositionAdoptsItsApplications(t *testing.T) {
	db := rbacDB(t)
	now := time.Now().UTC()
	election := &models.Election{
		ID: "e-link", Title: "Guild Election", Status: models.ElectionStatusDraft,
		StartsAt: now.Add(24 * time.Hour), EndsAt: now.Add(48 * time.Hour),
		AllowedDomain: "std.mast.ac.ug", ResultPolicy: models.ResultPolicyAfterClose,
	}
	if err := db.Create(election).Error; err != nil {
		t.Fatal(err)
	}
	position := openPosition(t, db, "Guild President", 0)

	applicant := testkit.User(t, db, "2022bse001@std.mast.ac.ug", models.RoleStudentVoter)
	admin := testkit.User(t, db, "registrar@must.ac.ug", models.RoleAdmin)

	client := testkit.New(t, positionEngine(t, db))
	applicantToken := testkit.SignIn(t, db, testkit.Auth(testkit.Config(), db), applicant)
	adminToken := testkit.SignIn(t, db, testkit.Auth(testkit.Config(), db), admin)

	var application models.Candidate
	client.As(applicantToken).Post("/api/v1/positions/"+position.ID+"/apply",
		positionApplicationPayload("Synthetic Applicant")).AssertCreated().BindData(&application)

	client.As(adminToken).Post("/api/v1/positions/"+position.ID+"/link",
		map[string]any{"election_id": election.ID}).AssertOK()

	var storedPosition models.Position
	if err := db.First(&storedPosition, "id = ?", position.ID).Error; err != nil {
		t.Fatal(err)
	}
	if storedPosition.ElectionRef() != election.ID {
		t.Fatalf("position election = %q, want %q", storedPosition.ElectionRef(), election.ID)
	}

	// The application follows the position so tallying sees it under the
	// election it is contesting.
	var storedCandidate models.Candidate
	if err := db.First(&storedCandidate, "id = ?", application.ID).Error; err != nil {
		t.Fatal(err)
	}
	if storedCandidate.ElectionID != election.ID {
		t.Fatalf("application election = %q, want %q", storedCandidate.ElectionID, election.ID)
	}
}

func TestStandalonePositionCreationIsAdminOnly(t *testing.T) {
	db := rbacDB(t)
	student := testkit.User(t, db, "2022bse001@std.mast.ac.ug", models.RoleStudentVoter)
	admin := testkit.User(t, db, "registrar@must.ac.ug", models.RoleAdmin)

	client := testkit.New(t, positionEngine(t, db))
	studentToken := testkit.SignIn(t, db, testkit.Auth(testkit.Config(), db), student)
	adminToken := testkit.SignIn(t, db, testkit.Auth(testkit.Config(), db), admin)

	payload := map[string]any{
		"title":          "Guild President",
		"question_type":  models.QuestionTypeSingleChoice,
		"min_selections": 1,
		"max_selections": 1,
		"seats":          1,
		"allowed_domain": "std.mast.ac.ug",
	}

	// A student cannot create a position.
	client.As(studentToken).Post("/api/v1/positions", payload).AssertForbidden()

	// An administrator can, and it starts unlinked and open.
	var created models.Position
	client.As(adminToken).Post("/api/v1/positions", payload).AssertCreated().BindData(&created)
	if created.ElectionID != nil {
		t.Fatalf("new position election = %v, want nil (unlinked)", created.ElectionID)
	}
	if created.Status != models.PositionStatusOpen {
		t.Fatalf("new position status = %q, want OPEN", created.Status)
	}

	// An invalid question type is rejected.
	bad := map[string]any{"title": "Bad", "question_type": "NONSENSE"}
	client.As(adminToken).Post("/api/v1/positions", bad).AssertBadRequest()
}
