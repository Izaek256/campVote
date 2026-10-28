package handlers_test

import (
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

func lifecycleEngine(t *testing.T, db *gorm.DB, clock *time.Time) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	cfg := testkit.Config()
	auth := testkit.Auth(cfg, db)
	h := handlers.NewElectionLifecycleHandler(db)
	h.Now = func() time.Time { return *clock }
	bh := handlers.NewBallotHandler(db)
	bh.Now = func() time.Time { return *clock }

	r := gin.New()
	r.Use(policy.Declared())
	protected := r.Group("")
	protected.Use(middleware.APIKeyOrAuth(db, middleware.Auth(db, auth)))
	protected.Use(policy.Guard(db))
	protected.GET("/api/v1/elections/:id/lifecycle", h.Phase)
	protected.GET("/api/v1/elections/:id/ballot", bh.View)
	protected.POST("/api/v1/elections/:id/ballot/submit", bh.Submit)

	staff := r.Group("")
	staff.Use(middleware.APIKeyOrAuth(db, middleware.Auth(db, auth)))
	staff.Use(policy.Guard(db))
	staff.POST("/api/v1/elections", h.Create)
	staff.GET("/api/v1/elections/:id", h.Get)
	staff.PUT("/api/v1/elections/:id", h.Update)
	staff.POST("/api/v1/elections/:id/transitions", h.Transition)
	return r
}

type lifecycleElection struct {
	ID              string    `json:"id"`
	Title           string    `json:"title"`
	Status          string    `json:"status"`
	StartsAt        time.Time `json:"starts_at"`
	EndsAt          time.Time `json:"ends_at"`
	Version         int       `json:"version"`
	Phase           string    `json:"phase"`
	AcceptingVotes  bool      `json:"accepting_votes"`
	ServerTime      time.Time `json:"server_time"`
	DisplayTimezone string    `json:"display_timezone"`
}

func createLifecycleElection(t *testing.T, client *testkit.Client, token string, start, end time.Time) lifecycleElection {
	t.Helper()
	var created lifecycleElection
	client.As(token).Post("/api/v1/elections", map[string]any{
		"title":           "Synthetic Guild Election",
		"description":     "Test election",
		"category":        models.ElectionCategoryGuild,
		"organizing_unit": "MMU Electoral Commission",
		"starts_at":       start.Format(time.RFC3339),
		"ends_at":         end.Format(time.RFC3339),
		"result_policy":   models.ResultPolicyAfterClose,
		"allowed_domains": "@std.must.ac.ug",
	}).AssertCreated().BindData(&created)
	return created
}

func TestElectionLifecycleAPIValidatesSchedulesFreezesAndAudits(t *testing.T) {
	db := rbacDB(t)
	now := time.Date(2026, 11, 1, 8, 0, 0, 0, time.UTC)
	clock := now
	engine := lifecycleEngine(t, db, &clock)
	admin := rbacUser(t, db, "admin@must.ac.ug", models.RoleAdmin)
	token := testkit.SignIn(t, db, testkit.Auth(testkit.Config(), db), admin)
	client := testkit.New(t, engine)

	start := now.Add(time.Hour)
	end := start.Add(2 * time.Hour)
	e := createLifecycleElection(t, client, token, start, end)
	if e.Status != models.ElectionStatusDraft || e.Phase != models.ElectionStatusDraft {
		t.Fatalf("created election status = %s/%s, want DRAFT", e.Status, e.Phase)
	}
	if !e.StartsAt.Equal(start) || !e.EndsAt.Equal(end) {
		t.Errorf("schedule was not stored exactly: %s - %s", e.StartsAt, e.EndsAt)
	}

	// No positions yet: scheduling must be refused.
	client.As(token).Post("/api/v1/elections/"+e.ID+"/transitions", map[string]string{
		"to": models.ElectionStatusScheduled,
	}).AssertUnprocessable()

	// Build a valid ballot directly: position/candidate endpoints are ISSUE-06.
	position := &models.Position{
		ElectionID: e.ID, Title: "President", QuestionType: models.QuestionTypeSingleChoice,
		MinSelections: 1, MaxSelections: 1, Seats: 1,
	}
	if err := db.Create(position).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.Candidate{
		ElectionID: e.ID, PositionID: position.ID, FullName: "Synthetic Candidate",
		ApprovalStatus: models.CandidateStatusApproved,
	}).Error; err != nil {
		t.Fatal(err)
	}

	var scheduled lifecycleElection
	client.As(token).Post("/api/v1/elections/"+e.ID+"/transitions", map[string]string{
		"to": models.ElectionStatusScheduled,
	}).AssertOK().BindData(&scheduled)
	if scheduled.Status != models.ElectionStatusScheduled {
		t.Fatalf("status = %s, want SCHEDULED", scheduled.Status)
	}
	var scheduledAudit int64
	db.Model(&models.ElectionAuditLog{}).
		Where("election_id = ? AND action = ?", e.ID, models.AuditActionElectionScheduled).Count(&scheduledAudit)
	if scheduledAudit != 1 {
		t.Fatalf("scheduling audit rows = %d, want 1", scheduledAudit)
	}

	// Configuration is editable before the window opens.
	client.As(token).Put("/api/v1/elections/"+e.ID, map[string]any{
		"title": "Renamed Guild Election",
	}).AssertOK()

	// A scheduled election cannot have its end moved into the past; that would
	// silently close it on the next read.
	client.As(token).Put("/api/v1/elections/"+e.ID, map[string]any{
		"ends_at": now.Add(-time.Hour).Format(time.RFC3339),
	}).AssertUnprocessable()

	// Move server time to the start. A GET reconciles the schedule; no client
	// header is involved.
	clock = start
	var opened lifecycleElection
	client.As(token).Get("/api/v1/elections/" + e.ID).AssertOK().BindData(&opened)
	if opened.Status != models.ElectionStatusOpen || !opened.AcceptingVotes {
		t.Fatalf("status = %s accepting=%v, want OPEN/true", opened.Status, opened.AcceptingVotes)
	}
	var openedAudit int64
	db.Model(&models.ElectionAuditLog{}).
		Where("election_id = ? AND action = ?", e.ID, models.AuditActionElectionOpened).Count(&openedAudit)
	if openedAudit != 1 {
		t.Fatalf("opening audit rows = %d, want 1", openedAudit)
	}

	// After open the configuration is frozen, including the title.
	client.As(token).Put("/api/v1/elections/"+e.ID, map[string]any{
		"title": "Changed After Opening",
	}).AssertConflict()

	// An end-time extension is the audited exception; it needs a reason.
	client.As(token).Put("/api/v1/elections/"+e.ID, map[string]any{
		"ends_at": end.Add(time.Hour).Format(time.RFC3339),
	}).AssertBadRequest()
	client.As(token).Put("/api/v1/elections/"+e.ID, map[string]any{
		"ends_at": end.Add(time.Hour).Format(time.RFC3339),
		"reason":  "polling station outage",
	}).AssertOK()
	var extension models.ElectionAuditLog
	if err := db.Where("election_id = ? AND action = ?", e.ID, models.AuditActionElectionExtended).
		First(&extension).Error; err != nil {
		t.Fatal("the extension was not audited")
	}
	if extension.ActorID != admin.ID {
		t.Errorf("extension actor = %q, want %q", extension.ActorID, admin.ID)
	}
	if extension.Details == "" || !strings.Contains(extension.Details, "polling station outage") {
		t.Errorf("extension audit details %q do not carry the reason", extension.Details)
	}

	// Manual opening is refused: only the server clock opens an election.
	client.As(token).Post("/api/v1/elections/"+e.ID+"/transitions", map[string]string{
		"to": models.ElectionStatusOpen,
	}).AssertConflict()

	// Early close needs a reason and is audited.
	client.As(token).Post("/api/v1/elections/"+e.ID+"/transitions", map[string]string{
		"to": models.ElectionStatusClosed,
	}).AssertBadRequest()
	client.As(token).Post("/api/v1/elections/"+e.ID+"/transitions", map[string]string{
		"to": models.ElectionStatusClosed, "reason": "security incident",
	}).AssertOK()
	var closed models.ElectionAuditLog
	if err := db.Where("election_id = ? AND action = ?", e.ID, models.AuditActionElectionClosed).
		First(&closed).Error; err != nil {
		t.Fatal("the early close was not audited")
	}
	if !strings.Contains(closed.Details, "security incident") {
		t.Errorf("close audit details %q do not carry the reason", closed.Details)
	}
}

func TestOfficialCanEditButCannotCancelOrExtend(t *testing.T) {
	db := rbacDB(t)
	now := time.Date(2026, 11, 1, 8, 0, 0, 0, time.UTC)
	clock := now
	engine := lifecycleEngine(t, db, &clock)
	admin := rbacUser(t, db, "admin@must.ac.ug", models.RoleAdmin)
	official := rbacUser(t, db, "official@must.ac.ug", models.RoleElectionOfficial)
	adminToken := testkit.SignIn(t, db, testkit.Auth(testkit.Config(), db), admin)
	officialToken := testkit.SignIn(t, db, testkit.Auth(testkit.Config(), db), official)
	client := testkit.New(t, engine)

	e := createLifecycleElection(t, client, adminToken, now.Add(time.Hour), now.Add(3*time.Hour))
	rbacAssign(t, db, e.ID, official.ID, models.RoleElectionOfficial)

	client.As(officialToken).Put("/api/v1/elections/"+e.ID, map[string]any{
		"title": "Officer Edit",
	}).AssertOK()

	// Opening is simulated directly so the permission checks are what is tested.
	if err := db.Model(&models.Election{}).Where("id = ?", e.ID).
		Update("status", models.ElectionStatusOpen).Error; err != nil {
		t.Fatal(err)
	}
	client.As(officialToken).Put("/api/v1/elections/"+e.ID, map[string]any{
		"ends_at": now.Add(4 * time.Hour).Format(time.RFC3339),
		"reason":  "official wants more time",
	}).AssertForbidden()
	client.As(officialToken).Post("/api/v1/elections/"+e.ID+"/transitions", map[string]string{
		"to": models.ElectionStatusCancelled, "reason": "official decision",
	}).AssertForbidden()
}

func TestSpoofedDateHeaderCannotExtendVoting(t *testing.T) {
	db := rbacDB(t)
	now := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	clock := now
	engine := lifecycleEngine(t, db, &clock)

	voter := testkit.User(t, db, "listed@std.must.ac.ug", models.RoleStudentVoter)
	election := &models.Election{
		ID: "spoof", Title: "Spoof Test", Status: models.ElectionStatusOpen,
		StartsAt: now.Add(-2 * time.Hour), EndsAt: now.Add(-time.Minute),
		ResultPolicy: models.ResultPolicyAfterClose, AllowedDomains: "@std.must.ac.ug",
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
		ApprovalStatus: models.CandidateStatusApproved,
	}
	if err := db.Create(candidate).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.ElectionRoster{
		ElectionID: election.ID, StudentNumber: "22/U/9001", Email: voter.Email,
		FullName: voter.FirstName + " " + voter.LastName, Status: models.RosterStatusEligible,
	}).Error; err != nil {
		t.Fatal(err)
	}
	token := testkit.SignIn(t, db, testkit.Auth(testkit.Config(), db), voter)
	client := testkit.New(t, engine)

	// The header claims the election is still inside its window. Server time
	// says otherwise, so the ballot is refused.
	client.As(token).With("Date", now.Add(-time.Hour).Format(time.RFC1123)).
		Post("/api/v1/elections/"+election.ID+"/ballot/submit", map[string]any{
			"selections": []map[string]string{{"position_id": position.ID, "candidate_id": candidate.ID}},
		}).AssertForbidden()

	var lifecycleView struct {
		Status         string `json:"status"`
		AcceptingVotes bool   `json:"accepting_votes"`
	}
	client.As(token).Get("/api/v1/elections/" + election.ID + "/lifecycle").AssertOK().BindData(&lifecycleView)
	if lifecycleView.Status != models.ElectionStatusClosed || lifecycleView.AcceptingVotes {
		t.Errorf("server phase = %s accepting=%v, want CLOSED/false", lifecycleView.Status, lifecycleView.AcceptingVotes)
	}
}

func TestTimezoneOffsetsAreNormalizedAndNaiveTimesRejected(t *testing.T) {
	db := rbacDB(t)
	now := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	clock := now
	engine := lifecycleEngine(t, db, &clock)
	admin := rbacUser(t, db, "admin@must.ac.ug", models.RoleAdmin)
	token := testkit.SignIn(t, db, testkit.Auth(testkit.Config(), db), admin)
	client := testkit.New(t, engine)

	// US spring-forward: the wall clock jumps from 01:59 EST to 03:00 EDT.
	// The absolute window is 1.5 hours, which passes the one-hour minimum.
	var created lifecycleElection
	client.As(token).Post("/api/v1/elections", map[string]any{
		"title":           "DST Election",
		"category":        models.ElectionCategoryGuild,
		"starts_at":       "2027-03-14T01:30:00-05:00",
		"ends_at":         "2027-03-14T04:00:00-04:00",
		"result_policy":   models.ResultPolicyAfterClose,
		"allowed_domains": "@std.must.ac.ug",
	}).AssertCreated().BindData(&created)

	wantStart := time.Date(2027, 3, 14, 6, 30, 0, 0, time.UTC)
	wantEnd := time.Date(2027, 3, 14, 8, 0, 0, 0, time.UTC)
	if !created.StartsAt.Equal(wantStart) || !created.EndsAt.Equal(wantEnd) {
		t.Errorf("stored schedule = %s - %s, want %s - %s",
			created.StartsAt, created.EndsAt, wantStart, wantEnd)
	}
	var stored models.Election
	db.Where("id = ?", created.ID).First(&stored)
	if stored.StartsAt.Location() != time.UTC || stored.EndsAt.Location() != time.UTC {
		t.Errorf("stored locations = %s/%s, want UTC", stored.StartsAt.Location(), stored.EndsAt.Location())
	}

	// A local time without an offset is ambiguous and must be rejected.
	client.As(token).Post("/api/v1/elections", map[string]any{
		"title":     "Naive Election",
		"category":  models.ElectionCategoryGuild,
		"starts_at": "2027-03-14T01:30:00",
		"ends_at":   "2027-03-14T04:00:00",
	}).AssertUnprocessable()
}
