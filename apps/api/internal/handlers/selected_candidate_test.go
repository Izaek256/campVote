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

// selectedEngine mounts the invitation, acceptance and submission routes with
// the same guard the API runs.
func selectedEngine(t *testing.T, db *gorm.DB) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	auth := testkit.Auth(testkit.Config(), db)
	h := handlers.NewCandidateHandler(db)

	r := gin.New()
	r.Use(policy.Declared())

	protected := r.Group("")
	protected.Use(middleware.APIKeyOrAuth(db, middleware.Auth(db, auth)))
	protected.Use(policy.Guard(db))
	protected.GET("/api/v1/candidates/my", h.GetMyCandidacies)
	protected.POST("/api/v1/elections/:id/candidates/apply", h.Apply)
	protected.POST("/api/v1/candidates/my/:candidate_id/accept", h.AcceptSelection)
	protected.POST("/api/v1/candidates/my/:candidate_id/submit", h.SubmitSelection)

	staff := r.Group("")
	staff.Use(middleware.APIKeyOrAuth(db, middleware.Auth(db, auth)))
	staff.Use(middleware.RequireStaff(db))
	staff.Use(policy.Guard(db))
	staff.GET("/api/v1/elections/:id/positions/:position_id/selections", h.ListSelections)
	staff.POST("/api/v1/elections/:id/positions/:position_id/selections", h.InviteSelected)
	return r
}

func selectionPayload(name string) map[string]any {
	return map[string]any{
		"full_name":          name,
		"date_of_birth":      "2003-05-14",
		"sex":                "MALE",
		"passport_photo_url": "/uploads/passport.jpg",
		"manifesto_url":      "/uploads/manifesto.mp3",
	}
}

func TestSelectedCandidateAcceptsAndPublishesWithAManifesto(t *testing.T) {
	db := rbacDB(t)
	now := time.Now().UTC()
	election, position := candidateElection(t, db, models.ElectionStatusOpen, now.Add(24*time.Hour))
	if err := db.Model(position).Update("admin_selected", true).Error; err != nil {
		t.Fatal(err)
	}
	position.AdminSelected = true

	admin := testkit.User(t, db, "admin@must.ac.ug", models.RoleAdmin)
	student := testkit.User(t, db, "2024bse088@std.mast.ac.ug", models.RoleStudentVoter)
	intruder := testkit.User(t, db, "2024bse089@std.mast.ac.ug", models.RoleStudentVoter)

	client := testkit.New(t, selectedEngine(t, db))
	adminToken := testkit.SignIn(t, db, testkit.Auth(testkit.Config(), db), admin)
	studentToken := testkit.SignIn(t, db, testkit.Auth(testkit.Config(), db), student)
	intruderToken := testkit.SignIn(t, db, testkit.Auth(testkit.Config(), db), intruder)

	// Administrator names the only email that may stand.
	client.As(adminToken).
		Post("/api/v1/elections/"+election.ID+"/positions/"+position.ID+"/selections",
			map[string]any{"emails": []string{student.Email}}).
		AssertCreated()

	var invitation models.Candidate
	if err := db.Where("position_id = ?", position.ID).First(&invitation).Error; err != nil {
		t.Fatal(err)
	}
	if invitation.ApprovalStatus != models.CandidateStatusSelected ||
		invitation.InvitedEmail != student.Email {
		t.Fatalf("invitation = %+v", invitation)
	}

	// Ordinary applications are closed for an administrator-selected position.
	client.As(studentToken).
		Post("/api/v1/elections/"+election.ID+"/candidates/apply",
			applicationPayload(position.ID, "Synthetic Applicant")).
		AssertForbidden()

	// The invitation is listed before it is accepted.
	var mine struct {
		Candidates []struct {
			Candidate models.Candidate `json:"candidate"`
		} `json:"candidates"`
	}
	client.As(studentToken).Get("/api/v1/candidates/my").AssertOK().BindData(&mine)
	if len(mine.Candidates) != 1 || mine.Candidates[0].Candidate.ID != invitation.ID {
		t.Fatalf("invitation not listed: %+v", mine)
	}

	// Only the named account may accept.
	client.As(intruderToken).
		Post("/api/v1/candidates/my/"+invitation.ID+"/accept", nil).
		AssertNotFound()
	client.As(studentToken).
		Post("/api/v1/candidates/my/"+invitation.ID+"/accept", nil).
		AssertOK()

	// A manifesto is mandatory, must be this account's upload, and must be an
	// accepted type. The file picker is not the control.
	client.As(studentToken).
		Post("/api/v1/candidates/my/"+invitation.ID+"/submit", selectionPayload("Selected Student")).
		AssertStatus(422)

	if err := db.Create(&models.Upload{
		ID: "upload-manifesto", UserID: student.ID,
		OriginalName: "manifesto.mp3", Filename: "manifesto.mp3",
		Path: "uploads/manifesto.mp3", URL: "/uploads/manifesto.mp3",
		MimeType: "audio/mpeg", Size: 2 << 20,
	}).Error; err != nil {
		t.Fatal(err)
	}
	client.As(studentToken).
		Post("/api/v1/candidates/my/"+invitation.ID+"/submit", selectionPayload("Selected Student")).
		AssertOK()

	var published models.Candidate
	if err := db.Where("id = ?", invitation.ID).First(&published).Error; err != nil {
		t.Fatal(err)
	}
	if published.ApprovalStatus != models.CandidateStatusApproved ||
		published.ManifestoURL != "/uploads/manifesto.mp3" ||
		published.BallotNumber == 0 {
		t.Fatalf("published candidate = %+v", published)
	}

	// The same path edits the details until voting opens.
	edit := selectionPayload("Edited Name")
	edit["bio"] = "Updated before the poll opened."
	client.As(studentToken).
		Post("/api/v1/candidates/my/"+invitation.ID+"/submit", edit).
		AssertOK()
	var edited models.Candidate
	if err := db.Where("id = ?", invitation.ID).First(&edited).Error; err != nil {
		t.Fatal(err)
	}
	if edited.FullName != "Edited Name" || edited.Bio != "Updated before the poll opened." {
		t.Fatalf("edit did not stick: %+v", edited)
	}
}
