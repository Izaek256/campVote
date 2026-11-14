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

func applicationPayload(positionID, name string) map[string]any {
	return map[string]any{
		"position_id":            positionID,
		"full_name":              name,
		"date_of_birth":          "2003-05-14",
		"sex":                    "MALE",
		"passport_photo_url":     "/uploads/passport.jpg",
		"id_document_url":        "/uploads/student-id.jpg",
		"partial_transcript_url": "/uploads/transcript.pdf",
	}
}

func candidateEngine(t *testing.T, db *gorm.DB) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	auth := testkit.Auth(testkit.Config(), db)
	h := handlers.NewCandidateHandler(db)

	r := gin.New()
	r.Use(policy.Declared())
	protected := r.Group("")
	protected.Use(middleware.APIKeyOrAuth(db, middleware.Auth(db, auth)))
	protected.Use(policy.Guard(db))
	protected.GET("/api/v1/elections/:id/candidate-application", h.Application)
	protected.POST("/api/v1/elections/:id/candidates/apply", h.Apply)
	protected.GET("/api/v1/candidates/my", h.GetMyCandidacies)
	protected.POST("/api/v1/candidates/my/:candidate_id/withdraw", h.WithdrawMine)

	staff := r.Group("")
	staff.Use(middleware.APIKeyOrAuth(db, middleware.Auth(db, auth)))
	staff.Use(middleware.RequireStaff(db))
	staff.Use(policy.Guard(db))
	staff.POST("/api/v1/elections/:id/candidates/:candidate_id/approve", h.Approve)
	staff.POST("/api/v1/elections/:id/candidates/:candidate_id/reject", h.Reject)
	staff.POST("/api/v1/elections/:id/candidates/:candidate_id/withdraw", h.Withdraw)
	return r
}

func candidateElection(t *testing.T, db *gorm.DB, status string, startsAt time.Time) (*models.Election, *models.Position) {
	t.Helper()
	election := &models.Election{
		ID: "e-cand", Title: "Guild Election", Status: status,
		StartsAt: startsAt, EndsAt: startsAt.Add(24 * time.Hour),
		AllowedDomain: "std.mast.ac.ug", ResultPolicy: models.ResultPolicyAfterClose,
	}
	if err := db.Create(election).Error; err != nil {
		t.Fatal(err)
	}
	position := &models.Position{
		ElectionID: models.Ptr(election.ID), Title: "President",
		QuestionType:  models.QuestionTypeSingleChoice,
		MinSelections: 1, MaxSelections: 1, Seats: 1,
	}
	if err := db.Create(position).Error; err != nil {
		t.Fatal(err)
	}
	return election, position
}

func TestCandidateApplicationChecksRosterAndWindow(t *testing.T) {
	db := rbacDB(t)
	now := time.Now().UTC()
	election, position := candidateElection(t, db, models.ElectionStatusOpen, now.Add(24*time.Hour))
	if err := db.Model(&models.Election{}).Where("id = ?", election.ID).
		Update("allowed_courses", "bse").Error; err != nil {
		t.Fatal(err)
	}
	election.AllowedCourses = "bse"

	voter := testkit.User(t, db, "2022bse001@std.mast.ac.ug", models.RoleStudentVoter)
	intruder := testkit.User(t, db, "2022bba001@std.mast.ac.ug", models.RoleStudentVoter)
	listed(t, db, election.ID, "22/U/0100", voter.Email)

	client := testkit.New(t, candidateEngine(t, db))
	voterToken := testkit.SignIn(t, db, testkit.Auth(testkit.Config(), db), voter)

	application := client.As(voterToken).Get("/api/v1/elections/e-cand/candidate-application").AssertOK()
	var view struct {
		Eligible  bool `json:"eligible"`
		Accepting bool `json:"accepting_applications"`
		Positions []struct {
			ID string `json:"id"`
		} `json:"positions"`
	}
	application.BindData(&view)
	if !view.Eligible || !view.Accepting || len(view.Positions) != 1 {
		t.Fatalf("application view = %+v, want eligible and open", view)
	}

	client.As(voterToken).Post("/api/v1/elections/e-cand/candidates/apply", applicationPayload(position.ID, "Synthetic Applicant")).AssertCreated()

	// One active application per position; a second position is allowed.
	client.As(voterToken).Post("/api/v1/elections/e-cand/candidates/apply", applicationPayload(position.ID, "Synthetic Applicant")).AssertConflict()

	second := models.Position{
		ElectionID: models.Ptr(election.ID), Title: "Secretary", QuestionType: models.QuestionTypeSingleChoice,
		MinSelections: 1, MaxSelections: 1, Seats: 1,
	}
	if err := db.Create(&second).Error; err != nil {
		t.Fatal(err)
	}
	client.As(voterToken).Post("/api/v1/elections/e-cand/candidates/apply", applicationPayload(second.ID, "Synthetic Applicant")).AssertCreated()

	intruderToken := testkit.SignIn(t, db, testkit.Auth(testkit.Config(), db), intruder)
	client.As(intruderToken).Post("/api/v1/elections/e-cand/candidates/apply", applicationPayload(position.ID, "Not Listed")).AssertForbidden()
}

func TestCandidateApplicationClosesWhenVotingOpens(t *testing.T) {
	db := rbacDB(t)
	now := time.Now().UTC()
	election, position := candidateElection(t, db, models.ElectionStatusOpen, now.Add(-time.Hour))
	voter := testkit.User(t, db, "2022bcs001@std.mast.ac.ug", models.RoleStudentVoter)
	listed(t, db, election.ID, "22/U/0101", voter.Email)

	client := testkit.New(t, candidateEngine(t, db))
	token := testkit.SignIn(t, db, testkit.Auth(testkit.Config(), db), voter)
	client.As(token).Post("/api/v1/elections/e-cand/candidates/apply", applicationPayload(position.ID, "Late Applicant")).AssertConflict()
}

func TestCandidateApprovalNotifiesAndAssignsBallotNumber(t *testing.T) {
	db := rbacDB(t)
	now := time.Now().UTC()
	election, position := candidateElection(t, db, models.ElectionStatusOpen, now.Add(24*time.Hour))
	voter := testkit.User(t, db, "2023bit001@std.mast.ac.ug", models.RoleStudentVoter)
	listed(t, db, election.ID, "22/U/0102", voter.Email)
	officer := testkit.User(t, db, "chief@must.ac.ug", models.RoleChiefElectionAdmin)

	client := testkit.New(t, candidateEngine(t, db))
	voterToken := testkit.SignIn(t, db, testkit.Auth(testkit.Config(), db), voter)
	officerToken := testkit.SignIn(t, db, testkit.Auth(testkit.Config(), db), officer)

	var created models.Candidate
	client.As(voterToken).Post("/api/v1/elections/e-cand/candidates/apply", applicationPayload(position.ID, "Synthetic Applicant")).AssertCreated().BindData(&created)
	if created.Programme != "BIT" || created.StudyYear < 1 {
		t.Fatalf("derived identity = %q/year %d, want BIT and a positive study year", created.Programme, created.StudyYear)
	}

	client.As(officerToken).Post(
		"/api/v1/elections/e-cand/candidates/"+created.ID+"/approve",
		map[string]any{"reason": "Documents verified"},
	).AssertOK()

	var approved models.Candidate
	if err := db.Where("id = ?", created.ID).First(&approved).Error; err != nil {
		t.Fatal(err)
	}
	if approved.ApprovalStatus != models.CandidateStatusApproved || approved.BallotNumber != 1 {
		t.Fatalf("approved = %s ballot %d, want APPROVED/1", approved.ApprovalStatus, approved.BallotNumber)
	}
	var notifications int64
	db.Model(&models.Notification{}).Where("user_id = ?", voter.ID).Count(&notifications)
	if notifications != 1 {
		t.Fatalf("notifications = %d, want 1", notifications)
	}

	// A reviewed application cannot be rejected afterwards.
	client.As(officerToken).Post(
		"/api/v1/elections/e-cand/candidates/"+created.ID+"/reject",
		map[string]any{"reason": "Too late"},
	).AssertConflict()
}

func TestApprovedCandidateCanWithdrawOnlyBeforeTheElectionOpens(t *testing.T) {
	db := rbacDB(t)
	now := time.Now().UTC()
	election, position := candidateElection(t, db, models.ElectionStatusDraft, now.Add(24*time.Hour))
	voter := testkit.User(t, db, "2023bed002@std.mast.ac.ug", models.RoleStudentVoter)
	listed(t, db, election.ID, "22/U/0105", voter.Email)
	client := testkit.New(t, candidateEngine(t, db))
	voterToken := testkit.SignIn(t, db, testkit.Auth(testkit.Config(), db), voter)

	// Applications only open with the poll, so an approved candidate is seeded
	// directly to test the withdrawal window.
	created := models.Candidate{
		ElectionID: election.ID, PositionID: position.ID, FullName: "Synthetic Applicant",
		NominatedBy: voter.ID, ApprovalStatus: models.CandidateStatusApproved,
	}
	if err := db.Create(&created).Error; err != nil {
		t.Fatal(err)
	}

	// A reason or a signed letter is required.
	client.As(voterToken).Post("/api/v1/candidates/my/"+created.ID+"/withdraw",
		map[string]any{}).AssertBadRequest()
	client.As(voterToken).Post("/api/v1/candidates/my/"+created.ID+"/withdraw",
		map[string]any{"document_url": "/uploads/withdrawal-letter.pdf"}).AssertOK()

	var withdrawn models.Candidate
	db.Where("id = ?", created.ID).First(&withdrawn)
	if withdrawn.ApprovalStatus != models.CandidateStatusWithdrawn ||
		withdrawn.WithdrawalDocumentURL != "/uploads/withdrawal-letter.pdf" {
		t.Fatalf("withdrawn = %+v", withdrawn)
	}
	var audited int64
	db.Model(&models.ElectionAuditLog{}).
		Where("election_id = ? AND action = ?", election.ID, models.AuditActionCandidateWithdrawn).
		Count(&audited)
	if audited != 1 {
		t.Fatalf("withdrawal audit rows = %d, want 1", audited)
	}

	// Once the poll opens the ballot is fixed.
	if err := db.Model(&models.Election{}).Where("id = ?", election.ID).
		Update("status", models.ElectionStatusOpen).Error; err != nil {
		t.Fatal(err)
	}
	late := models.Candidate{
		ElectionID: election.ID, PositionID: position.ID, FullName: "Late Applicant",
		NominatedBy: voter.ID, ApprovalStatus: models.CandidateStatusApproved,
	}
	if err := db.Create(&late).Error; err != nil {
		t.Fatal(err)
	}
	client.As(voterToken).Post("/api/v1/candidates/my/"+late.ID+"/withdraw",
		map[string]any{"reason": "Changed my mind"}).AssertConflict()
}

func TestAdministratorConfirmsAWithdrawalBeforeTheElectionOpens(t *testing.T) {
	db := rbacDB(t)
	now := time.Now().UTC()
	election, position := candidateElection(t, db, models.ElectionStatusDraft, now.Add(24*time.Hour))
	admin := rbacUser(t, db, "candidate-admin@must.ac.ug", models.RoleAdmin)
	voter := testkit.User(t, db, "2023bed003@std.mast.ac.ug", models.RoleStudentVoter)
	listed(t, db, election.ID, "22/U/0106", voter.Email)
	client := testkit.New(t, candidateEngine(t, db))
	adminToken := testkit.SignIn(t, db, testkit.Auth(testkit.Config(), db), admin)

	created := models.Candidate{
		ElectionID: election.ID, PositionID: position.ID, FullName: "Synthetic Applicant",
		NominatedBy: voter.ID, ApprovalStatus: models.CandidateStatusNominated,
	}
	if err := db.Create(&created).Error; err != nil {
		t.Fatal(err)
	}

	client.As(adminToken).Post("/api/v1/elections/e-cand/candidates/"+created.ID+"/withdraw",
		map[string]any{}).AssertBadRequest()
	client.As(adminToken).Post("/api/v1/elections/e-cand/candidates/"+created.ID+"/withdraw",
		map[string]any{"reason": "Signed letter received"}).AssertOK()

	if err := db.Model(&models.Election{}).Where("id = ?", election.ID).
		Update("status", models.ElectionStatusOpen).Error; err != nil {
		t.Fatal(err)
	}
	late := models.Candidate{
		ElectionID: election.ID, PositionID: position.ID, FullName: "Late Applicant",
		NominatedBy: voter.ID, ApprovalStatus: models.CandidateStatusNominated,
	}
	if err := db.Create(&late).Error; err != nil {
		t.Fatal(err)
	}
	client.As(adminToken).Post("/api/v1/elections/e-cand/candidates/"+late.ID+"/withdraw",
		map[string]any{"reason": "Late"}).AssertConflict()
}

func TestCandidateWithdrawalIsOwnedByApplicant(t *testing.T) {
	db := rbacDB(t)
	now := time.Now().UTC()
	election, position := candidateElection(t, db, models.ElectionStatusOpen, now.Add(24*time.Hour))
	voter := testkit.User(t, db, "2023bed001@std.mast.ac.ug", models.RoleStudentVoter)
	other := testkit.User(t, db, "2023bba001@std.mast.ac.ug", models.RoleStudentVoter)
	listed(t, db, election.ID, "22/U/0103", voter.Email)
	listed(t, db, election.ID, "22/U/0104", other.Email)

	client := testkit.New(t, candidateEngine(t, db))
	voterToken := testkit.SignIn(t, db, testkit.Auth(testkit.Config(), db), voter)
	otherToken := testkit.SignIn(t, db, testkit.Auth(testkit.Config(), db), other)

	var created models.Candidate
	client.As(voterToken).Post("/api/v1/elections/e-cand/candidates/apply", applicationPayload(position.ID, "Synthetic Applicant")).AssertCreated().BindData(&created)

	// Another applicant cannot withdraw somebody else's application.
	client.As(otherToken).Post("/api/v1/candidates/my/"+created.ID+"/withdraw",
		map[string]any{"reason": "Not mine"}).AssertNotFound()

	client.As(voterToken).Post("/api/v1/candidates/my/"+created.ID+"/withdraw",
		map[string]any{"reason": "Personal reasons"}).AssertOK()

	var withdrawn models.Candidate
	db.Where("id = ?", created.ID).First(&withdrawn)
	if withdrawn.ApprovalStatus != models.CandidateStatusWithdrawn || withdrawn.WithdrawalReason != "Personal reasons" {
		t.Fatalf("withdrawn = %+v", withdrawn)
	}
}
