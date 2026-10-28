package handlers_test

import (
	"net/http"
	"strings"
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

func eligibilityEngine(t *testing.T, db *gorm.DB) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	cfg := testkit.Config()
	auth := testkit.Auth(cfg, db)
	eh := handlers.NewEligibilityHandler(db)
	bh := handlers.NewBallotHandler(db)
	userHandler := &handlers.UserHandler{DB: db, AuthService: auth}

	r := gin.New()
	r.Use(policy.Declared())
	protected := r.Group("")
	protected.Use(middleware.APIKeyOrAuth(db, middleware.Auth(db, auth)))
	protected.Use(policy.Guard(db))
	protected.GET("/api/v1/elections/:id/eligibility", eh.Mine)
	protected.GET("/api/v1/elections/:id/ballot", bh.View)
	protected.POST("/api/v1/elections/:id/ballot/start", bh.Start)
	protected.POST("/api/v1/elections/:id/ballot/submit", bh.Submit)
	protected.PUT("/api/v1/profile", userHandler.UpdateProfile)

	staff := r.Group("")
	staff.Use(middleware.APIKeyOrAuth(db, middleware.Auth(db, auth)))
	staff.Use(policy.Guard(db))
	staff.POST("/api/v1/elections/:id/roster/preview", eh.Preview)
	staff.POST("/api/v1/elections/:id/roster/import", eh.Import)
	staff.POST("/api/v1/elections/:id/roster/:rosterId/status", eh.SetStatus)
	staff.POST("/api/v1/elections/:id/eligibility/freeze", eh.Freeze)
	staff.GET("/api/v1/elections/:id/eligibility/report", eh.Report)
	return r
}

type ballotFixture struct {
	election  *models.Election
	position  *models.Position
	candidate *models.Candidate
}

func ballotElection(t *testing.T, db *gorm.DB, id string, start, end time.Time) ballotFixture {
	t.Helper()
	election := &models.Election{
		ID: id, Title: id, Status: models.ElectionStatusOpen,
		StartsAt: start, EndsAt: end, AllowedDomains: "@std.must.ac.ug,@must.ac.ug",
		ResultPolicy: models.ResultPolicyAfterClose,
	}
	if err := db.Create(election).Error; err != nil {
		t.Fatal(err)
	}
	position := &models.Position{
		ElectionID: election.ID, Title: "President", QuestionType: models.QuestionTypeSingleChoice,
		MinSelections: 1, MaxSelections: 1, Seats: 1,
	}
	if err := db.Create(position).Error; err != nil {
		t.Fatal(err)
	}
	candidate := &models.Candidate{
		ElectionID: election.ID, PositionID: position.ID, FullName: "Synthetic Candidate",
		ApprovalStatus: models.CandidateStatusApproved, BallotNumber: 1,
	}
	if err := db.Create(candidate).Error; err != nil {
		t.Fatal(err)
	}
	return ballotFixture{election: election, position: position, candidate: candidate}
}

func TestEligibilityGateOnBallotViewStartAndSubmit(t *testing.T) {
	db := rbacDB(t)
	now := time.Now().UTC()
	a := ballotElection(t, db, "e-a", now.Add(-time.Hour), now.Add(time.Hour))
	b := ballotElection(t, db, "e-b", now.Add(-time.Hour), now.Add(time.Hour))
	c := ballotElection(t, db, "e-c", now.Add(time.Hour), now.Add(2*time.Hour))

	voter := testkit.User(t, db, "listed@std.must.ac.ug", models.RoleStudentVoter)
	intruder := testkit.User(t, db, "valid-domain@std.must.ac.ug", models.RoleStudentVoter)
	other := testkit.User(t, db, "other@std.must.ac.ug", models.RoleStudentVoter)
	listed(t, db, a.election.ID, "22/U/0001", voter.Email)
	listed(t, db, b.election.ID, "22/U/0002", other.Email)
	listed(t, db, c.election.ID, "22/U/0003", voter.Email)

	cfg := testkit.Config()
	auth := testkit.Auth(cfg, db)
	client := testkit.New(t, eligibilityEngine(t, db))
	voterToken := testkit.SignIn(t, db, auth, voter)
	intruderToken := testkit.SignIn(t, db, auth, intruder)

	client.As(intruderToken).Get("/api/v1/elections/e-a/ballot").AssertForbidden()
	client.As(intruderToken).Post("/api/v1/elections/e-a/ballot/start", nil).AssertForbidden()
	client.As(intruderToken).Post("/api/v1/elections/e-a/ballot/submit", map[string]any{}).AssertForbidden()

	client.As(voterToken).Get("/api/v1/elections/e-a/ballot").AssertOK()
	client.As(voterToken).Get("/api/v1/elections/e-b/ballot").AssertForbidden()
	client.As(voterToken).Get("/api/v1/elections/e-c/ballot").AssertForbidden() // not open yet
	client.As(voterToken).Post("/api/v1/elections/e-a/ballot/start", nil).AssertOK()
	client.As(voterToken).Post("/api/v1/elections/e-b/ballot/start", nil).AssertForbidden()

	// Tamper: a token eligible for e-a cannot submit into e-b.
	client.As(voterToken).Post("/api/v1/elections/e-b/ballot/submit", map[string]any{
		"selections": []map[string]string{{"position_id": b.position.ID, "candidate_id": b.candidate.ID}},
	}).AssertForbidden()

	// Tamper: a candidate from another election is not on this ballot.
	client.As(voterToken).Post("/api/v1/elections/e-a/ballot/submit", map[string]any{
		"selections": []map[string]string{{"position_id": a.position.ID, "candidate_id": b.candidate.ID}},
	}).AssertBadRequest()

	// A valid submission records participation and an anonymous ballot.
	client.As(voterToken).Post("/api/v1/elections/e-a/ballot/submit", map[string]any{
		"selections": []map[string]string{{"position_id": a.position.ID, "candidate_id": a.candidate.ID}},
	}).AssertCreated()
	client.As(voterToken).Post("/api/v1/elections/e-a/ballot/start", nil).AssertConflict()

	var ballots int64
	db.Model(&models.AnonymousBallot{}).Where("election_id = ?", a.election.ID).Count(&ballots)
	if ballots != 1 {
		t.Errorf("accepted ballots = %d, want 1", ballots)
	}
	var participation models.VoterParticipation
	if err := db.Where("election_id = ? AND user_id = ?", a.election.ID, voter.ID).First(&participation).Error; err != nil {
		t.Fatal("participation was not recorded")
	}
	var ballot models.AnonymousBallot
	if err := db.Where("election_id = ?", a.election.ID).First(&ballot).Error; err != nil {
		t.Fatal(err)
	}
	if ballot.CastAt.IsZero() {
		t.Error("ballot has no cast time")
	}
}

func TestVoterCannotChangeOwnEligibility(t *testing.T) {
	db := rbacDB(t)
	now := time.Now().UTC()
	a := ballotElection(t, db, "e-a", now.Add(-time.Hour), now.Add(time.Hour))
	voter := testkit.User(t, db, "listed@std.must.ac.ug", models.RoleStudentVoter)
	entry := listed(t, db, a.election.ID, "22/U/0001", voter.Email)

	cfg := testkit.Config()
	auth := testkit.Auth(cfg, db)
	client := testkit.New(t, eligibilityEngine(t, db))
	voterToken := testkit.SignIn(t, db, auth, voter)

	// Trying to "become" another student or to change faculty/status through the
	// profile is ignored; the roster is the only eligibility record.
	client.As(voterToken).Put("/api/v1/profile", map[string]any{
		"student_number": "22/U/9999", "faculty": "Invented", "status": models.RosterStatusEligible,
	}).AssertOK()

	var after models.ElectionRoster
	if err := db.Where("id = ?", entry.ID).First(&after).Error; err != nil {
		t.Fatal(err)
	}
	if after.StudentNumber != "22/U/0001" || after.Faculty != "" || after.Status != models.RosterStatusEligible {
		t.Errorf("profile update changed eligibility fields: %+v", after)
	}

	// A voter cannot reach the roster import or status endpoints at all.
	client.As(voterToken).Post("/api/v1/elections/e-a/roster/preview", nil).AssertForbidden()
	client.As(voterToken).Post("/api/v1/elections/e-a/roster/import", nil).AssertForbidden()
	client.As(voterToken).Post("/api/v1/elections/e-a/roster/"+entry.ID+"/status", map[string]string{
		"status": models.RosterStatusSuspended, "reason": "self",
	}).AssertForbidden()
	client.As(voterToken).Get("/api/v1/elections/e-a/eligibility/report").AssertForbidden()
}

func TestUnverifiedEmailChangeCannotBorrowARosterEntry(t *testing.T) {
	db := rbacDB(t)
	now := time.Now().UTC()
	a := ballotElection(t, db, "e-a", now.Add(-time.Hour), now.Add(time.Hour))
	// A roster entry for a student who has not registered an account.
	listed(t, db, a.election.ID, "22/U/0001", "unclaimed@std.must.ac.ug")

	attacker := testkit.User(t, db, "attacker@std.must.ac.ug", models.RoleStudentVoter)
	// What UpdateProfile does: the address changes and verification is cleared.
	if err := db.Model(&models.User{}).Where("id = ?", attacker.ID).Updates(map[string]any{
		"email": "unclaimed@std.must.ac.ug", "email_verified_at": nil,
	}).Error; err != nil {
		t.Fatal(err)
	}
	attacker.Email = "unclaimed@std.must.ac.ug"

	cfg := testkit.Config()
	auth := testkit.Auth(cfg, db)
	client := testkit.New(t, eligibilityEngine(t, db))
	token := testkit.SignIn(t, db, auth, attacker)

	// A typed address is not proof of owning it.
	client.As(token).Get("/api/v1/elections/e-a/ballot").AssertForbidden()

	// Once the address is verified, the same account may use its roster entry.
	if err := db.Model(&models.User{}).Where("id = ?", attacker.ID).
		Update("email_verified_at", now).Error; err != nil {
		t.Fatal(err)
	}
	client.As(token).Get("/api/v1/elections/e-a/ballot").AssertOK()
}

func TestMultiChoiceBallotAcceptsDistinctChoicesAndRejectsDuplicates(t *testing.T) {
	db := rbacDB(t)
	now := time.Now().UTC()
	a := ballotElection(t, db, "e-a", now.Add(-time.Hour), now.Add(time.Hour))
	multi := &models.Position{
		ElectionID: a.election.ID, Title: "Senate", QuestionType: models.QuestionTypeMultiChoice,
		MinSelections: 1, MaxSelections: 2, Seats: 2,
	}
	if err := db.Create(multi).Error; err != nil {
		t.Fatal(err)
	}
	var picks []*models.Candidate
	for i, name := range []string{"First", "Second"} {
		candidate := &models.Candidate{
			ElectionID: a.election.ID, PositionID: multi.ID, FullName: name,
			ApprovalStatus: models.CandidateStatusApproved, BallotNumber: i + 1,
		}
		if err := db.Create(candidate).Error; err != nil {
			t.Fatal(err)
		}
		picks = append(picks, candidate)
	}

	voter := testkit.User(t, db, "listed@std.must.ac.ug", models.RoleStudentVoter)
	listed(t, db, a.election.ID, "22/U/0001", voter.Email)
	cfg := testkit.Config()
	auth := testkit.Auth(cfg, db)
	client := testkit.New(t, eligibilityEngine(t, db))
	token := testkit.SignIn(t, db, auth, voter)

	// The same candidate twice is not two answers.
	client.As(token).Post("/api/v1/elections/e-a/ballot/submit", map[string]any{
		"selections": []map[string]string{
			{"position_id": a.position.ID, "candidate_id": a.candidate.ID},
			{"position_id": multi.ID, "candidate_id": picks[0].ID},
			{"position_id": multi.ID, "candidate_id": picks[0].ID},
		},
	}).AssertBadRequest()

	// Two distinct choices are one valid multi-choice answer.
	client.As(token).Post("/api/v1/elections/e-a/ballot/submit", map[string]any{
		"selections": []map[string]string{
			{"position_id": a.position.ID, "candidate_id": a.candidate.ID},
			{"position_id": multi.ID, "candidate_id": picks[0].ID},
			{"position_id": multi.ID, "candidate_id": picks[1].ID},
		},
	}).AssertCreated()
}

func TestRosterImportPreviewCommitFreezeAndReport(t *testing.T) {
	db := rbacDB(t)
	now := time.Now().UTC()
	a := ballotElection(t, db, "e-a", now.Add(-time.Hour), now.Add(time.Hour))

	chief := testkit.User(t, db, "chief@must.ac.ug", models.RoleChiefElectionAdmin)
	voter := testkit.User(t, db, "listed@std.must.ac.ug", models.RoleStudentVoter)
	cfg := testkit.Config()
	auth := testkit.Auth(cfg, db)
	client := testkit.New(t, eligibilityEngine(t, db))
	chiefToken := testkit.SignIn(t, db, auth, chief)
	voterToken := testkit.SignIn(t, db, auth, voter)

	good := "student_number,email,full_name,study_year,status\n" +
		"22/U/0001,listed@std.must.ac.ug,Listed Voter,3,ELIGIBLE\n"
	bad := "student_number,email,full_name\n" +
		"22/U/0002,not-an-email,Bad Email\n"
	late := "student_number,email,full_name\n22/U/0009,late@std.must.ac.ug,Late Voter\n"

	preview := postCSV(t, client, chiefToken, "/api/v1/elections/e-a/roster/preview", good, "roll.csv")
	preview.AssertStatus(http.StatusOK).JSON().Where("data.preview.creates", 1)
	postCSV(t, client, chiefToken, "/api/v1/elections/e-a/roster/import", bad, "bad.csv").
		AssertStatus(http.StatusUnprocessableEntity).JSON().Where("error.code", "VALIDATION_ERROR")
	postCSV(t, client, chiefToken, "/api/v1/elections/e-a/roster/import", good, "roll.csv").
		AssertStatus(http.StatusCreated).JSON().Where("data.batch.created", 1)

	client.As(chiefToken).Post("/api/v1/elections/e-a/eligibility/freeze", nil).AssertOK()
	// After the freeze the electorate is fixed: a new entry is refused.
	postCSV(t, client, chiefToken, "/api/v1/elections/e-a/roster/import", late, "late.csv").
		AssertStatus(http.StatusUnprocessableEntity)

	// Revocation is allowed and the voter loses the ballot immediately.
	var entry models.ElectionRoster
	if err := db.Where("election_id = ?", a.election.ID).First(&entry).Error; err != nil {
		t.Fatal(err)
	}
	client.As(chiefToken).Post("/api/v1/elections/e-a/roster/"+entry.ID+"/status", map[string]string{
		"status": models.RosterStatusSuspended, "reason": "registrar confirmed suspension",
	}).AssertOK()
	client.As(voterToken).Get("/api/v1/elections/e-a/ballot").AssertForbidden()

	report := client.As(chiefToken).Get("/api/v1/elections/e-a/eligibility/report").AssertOK()
	report.JSON().Where("data.total", 1).Where("data.eligible", 0)
	body := report.Body()
	for _, forbidden := range []string{"listed@std.must.ac.ug", "22/U/0001"} {
		if strings.Contains(body, forbidden) {
			t.Errorf("eligibility report leaked %q: %s", forbidden, body)
		}
	}

	var audits int64
	db.Model(&models.ElectionAuditLog{}).Where("election_id = ?", a.election.ID).
		Where("action IN ?", []string{models.AuditActionRosterImported, models.AuditActionEligibilityFrozen,
			models.AuditActionRosterRevoked}).Count(&audits)
	if audits < 3 {
		t.Errorf("audit events = %d, want import, freeze and revocation", audits)
	}
}

func listed(t *testing.T, db *gorm.DB, electionID, student, email string) *models.ElectionRoster {
	t.Helper()
	entry := &models.ElectionRoster{
		ElectionID: electionID, StudentNumber: student, Email: email, FullName: "Synthetic Student",
		Status: models.RosterStatusEligible, Source: models.RosterSourceCSVImport,
	}
	if err := db.Create(entry).Error; err != nil {
		t.Fatal(err)
	}
	return entry
}

func postCSV(t *testing.T, client *testkit.Client, token, path, csv, _ string) *testkit.Response {
	t.Helper()
	return client.As(token).DoRaw(http.MethodPost, path, "text/csv", []byte(csv))
}
