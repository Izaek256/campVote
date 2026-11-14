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
	protected.GET("/api/v1/elections", h.List)
	protected.GET("/api/v1/elections/:id/lifecycle", h.Phase)
	protected.GET("/api/v1/elections/:id/ballot", bh.View)
	protected.POST("/api/v1/elections/:id/ballot/submit", bh.Submit)

	staff := r.Group("")
	staff.Use(middleware.APIKeyOrAuth(db, middleware.Auth(db, auth)))
	staff.Use(policy.Guard(db))
	staff.POST("/api/v1/elections", h.Create)
	staff.POST("/api/v1/elections/:id/clone", h.Clone)
	staff.GET("/api/v1/elections/:id", h.Get)
	staff.PUT("/api/v1/elections/:id", h.Update)
	staff.DELETE("/api/v1/elections/:id", h.Delete)
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
		"allowed_domains": "@std.mast.ac.ug",
	}).AssertCreated().BindData(&created)
	return created
}

func TestCloneCopiesTheBallotAndListButNoHistory(t *testing.T) {
	db := rbacDB(t)
	now := time.Date(2026, 11, 1, 8, 0, 0, 0, time.UTC)
	clock := now
	engine := lifecycleEngine(t, db, &clock)
	admin := rbacUser(t, db, "clone-admin@must.ac.ug", models.RoleAdmin)
	officer := rbacUser(t, db, "clone-officer@must.ac.ug", models.RoleStudentVoter)
	adminToken := testkit.SignIn(t, db, testkit.Auth(testkit.Config(), db), admin)
	client := testkit.New(t, engine)

	source := createLifecycleElection(t, client, adminToken,
		now.Add(24*time.Hour), now.Add(30*time.Hour))
	position := &models.Position{
		ElectionID: models.Ptr(source.ID), Title: "President",
		QuestionType: models.QuestionTypeSingleChoice, MinSelections: 1, MaxSelections: 1, Seats: 1,
	}
	if err := db.Create(position).Error; err != nil {
		t.Fatal(err)
	}
	candidates := []models.Candidate{
		{ElectionID: source.ID, PositionID: position.ID, FullName: "Still Standing",
			ApprovalStatus: models.CandidateStatusApproved, BallotNumber: 1,
			ManifestoURL: "/uploads/manifesto.pdf", PassportPhotoURL: "/uploads/photo.jpg"},
		{ElectionID: source.ID, PositionID: position.ID, FullName: "Stepped Down",
			ApprovalStatus: models.CandidateStatusWithdrawn},
		{ElectionID: source.ID, PositionID: position.ID, FullName: "Turned Down",
			ApprovalStatus: models.CandidateStatusRejected},
	}
	for i := range candidates {
		if err := db.Create(&candidates[i]).Error; err != nil {
			t.Fatal(err)
		}
	}
	for _, row := range []models.ElectionRoster{
		{StudentNumber: "24/U/0001", Email: "one@std.must.ac.ug", FullName: "One"},
		{StudentNumber: "24/U/0002", Email: "two@std.must.ac.ug", FullName: "Two"},
	} {
		row.ElectionID = source.ID
		if err := db.Create(&row).Error; err != nil {
			t.Fatal(err)
		}
	}
	// The copy must not inherit who administers the source election.
	if err := db.Create(&models.ElectionRole{
		ElectionID: source.ID, UserID: officer.ID, Role: models.RoleChiefElectionAdmin,
	}).Error; err != nil {
		t.Fatal(err)
	}

	var clone lifecycleElection
	client.As(adminToken).Post("/api/v1/elections/"+source.ID+"/clone", map[string]any{
		"title":     "Guild Election 2027",
		"starts_at": now.Add(48 * time.Hour).Format(time.RFC3339),
		"ends_at":   now.Add(54 * time.Hour).Format(time.RFC3339),
	}).AssertCreated().BindData(&clone)

	if clone.ID == source.ID || clone.Status != models.ElectionStatusDraft ||
		clone.Title != "Guild Election 2027" {
		t.Fatalf("clone = %+v, want a new DRAFT titled Guild Election 2027", clone)
	}
	var stored models.Election
	if err := db.Where("id = ?", clone.ID).First(&stored).Error; err != nil {
		t.Fatal(err)
	}
	if stored.CreatedByID != admin.ID || !stored.StartsAt.Equal(now.Add(48*time.Hour)) {
		t.Fatalf("clone row = %+v, want the caller as creator and the new schedule", stored)
	}
	if stored.EligibilityFrozenAt != nil {
		t.Error("the clone inherited a frozen eligibility snapshot")
	}

	var clonedPositions []models.Position
	db.Where("election_id = ?", clone.ID).Find(&clonedPositions)
	if len(clonedPositions) != 1 || clonedPositions[0].Title != "President" {
		t.Fatalf("cloned positions = %+v, want one President", clonedPositions)
	}
	var clonedCandidates []models.Candidate
	db.Where("election_id = ?", clone.ID).Find(&clonedCandidates)
	if len(clonedCandidates) != 1 || clonedCandidates[0].FullName != "Still Standing" {
		t.Fatalf("cloned candidates = %+v, want only the approved candidate", clonedCandidates)
	}
	if clonedCandidates[0].PositionID != clonedPositions[0].ID ||
		clonedCandidates[0].ManifestoURL != "/uploads/manifesto.pdf" ||
		clonedCandidates[0].PassportPhotoURL != "/uploads/photo.jpg" {
		t.Errorf("cloned candidate = %+v, want it remapped with its documents", clonedCandidates[0])
	}

	var clonedRoster int64
	db.Model(&models.ElectionRoster{}).Where("election_id = ?", clone.ID).Count(&clonedRoster)
	if clonedRoster != 2 {
		t.Errorf("cloned roster rows = %d, want 2", clonedRoster)
	}
	var clonedRoles int64
	db.Model(&models.ElectionRole{}).Where("election_id = ?", clone.ID).Count(&clonedRoles)
	if clonedRoles != 0 {
		t.Errorf("cloned role assignments = %d, want 0", clonedRoles)
	}
	var ballots int64
	db.Model(&models.AnonymousBallot{}).Where("election_id = ?", clone.ID).Count(&ballots)
	if ballots != 0 {
		t.Errorf("cloned ballots = %d, want 0", ballots)
	}
	var versions int64
	db.Model(&models.BallotVersion{}).Where("election_id = ?", clone.ID).Count(&versions)
	if versions != 0 {
		t.Errorf("cloned ballot versions = %d, want 0", versions)
	}
	var audit int64
	db.Model(&models.ElectionAuditLog{}).
		Where("election_id = ? AND action = ?", clone.ID, models.AuditActionElectionCloned).
		Count(&audit)
	if audit != 1 {
		t.Errorf("clone audit rows = %d, want 1", audit)
	}

	// A plain student cannot copy an election they merely see.
	outsider := rbacUser(t, db, "2024bse097@std.must.ac.ug", models.RoleStudentVoter)
	outsiderToken := testkit.SignIn(t, db, testkit.Auth(testkit.Config(), db), outsider)
	client.As(outsiderToken).Post("/api/v1/elections/"+source.ID+"/clone", map[string]any{
		"starts_at": now.Add(48 * time.Hour).Format(time.RFC3339),
		"ends_at":   now.Add(54 * time.Hour).Format(time.RFC3339),
	}).AssertForbidden()
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

	// A client cannot force an election open; only the server clock does that.
	client.As(token).Post("/api/v1/elections/"+e.ID+"/transitions", map[string]string{
		"to": models.ElectionStatusOpen,
	}).AssertConflict()

	// Build a valid ballot directly: position/candidate endpoints are ISSUE-06.
	position := &models.Position{
		ElectionID: models.Ptr(e.ID), Title: "President", QuestionType: models.QuestionTypeSingleChoice,
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

	// Configuration is editable while the election is still a draft.
	client.As(token).Put("/api/v1/elections/"+e.ID, map[string]any{
		"title": "Renamed Guild Election",
	}).AssertOK()

	// An end moved before the start leaves the window invalid.
	client.As(token).Put("/api/v1/elections/"+e.ID, map[string]any{
		"ends_at": now.Add(-time.Hour).Format(time.RFC3339),
	}).AssertUnprocessable()

	// Move server time to the start. A GET reconciles the schedule; the clock
	// opens the fully configured draft with no client involvement.
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

	// A live election exposes only its words and a later end time; everything
	// else is locked.
	client.As(token).Put("/api/v1/elections/"+e.ID, map[string]any{
		"title": "Changed After Opening",
	}).AssertOK()
	client.As(token).Put("/api/v1/elections/"+e.ID, map[string]any{
		"description": "A wording fix that does not change the ballot.",
	}).AssertOK()
	client.As(token).Put("/api/v1/elections/"+e.ID, map[string]any{
		"allowed_domain": "std.must.ac.ug",
	}).AssertConflict()
	client.As(token).Put("/api/v1/elections/"+e.ID, map[string]any{
		"category": models.ElectionCategoryFaculty,
	}).AssertConflict()
	var liveEdit models.ElectionAuditLog
	if err := db.Where("election_id = ? AND action = ?", e.ID, models.AuditActionElectionUpdated).
		Order("created_at DESC").First(&liveEdit).Error; err != nil {
		t.Fatal("the live edit was not audited")
	}
	if !strings.Contains(liveEdit.Details, "live_edit") {
		t.Errorf("live edit audit details %q do not mark it as live", liveEdit.Details)
	}

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
		"to": models.ElectionStatusClosed, "reason": "official decision",
	}).AssertForbidden()
}

func TestPlainStudentCannotCreateElection(t *testing.T) {
	db := rbacDB(t)
	now := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	clock := now
	engine := lifecycleEngine(t, db, &clock)
	student := rbacUser(t, db, "2024bse088@std.mast.ac.ug", models.RoleStudentVoter)
	token := testkit.SignIn(t, db, testkit.Auth(testkit.Config(), db), student)
	client := testkit.New(t, engine)

	client.As(token).Post("/api/v1/elections", map[string]any{
		"title":         "Student-run election",
		"starts_at":     now.Add(time.Hour).Format(time.RFC3339),
		"ends_at":       now.Add(3 * time.Hour).Format(time.RFC3339),
		"result_policy": models.ResultPolicyAfterClose,
	}).AssertForbidden()
}

func TestSpoofedDateHeaderCannotExtendVoting(t *testing.T) {
	db := rbacDB(t)
	now := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	clock := now
	engine := lifecycleEngine(t, db, &clock)

	voter := testkit.User(t, db, "2022bse001@std.mast.ac.ug", models.RoleStudentVoter)
	election := &models.Election{
		ID: "spoof", Title: "Spoof Test", Status: models.ElectionStatusOpen,
		StartsAt: now.Add(-2 * time.Hour), EndsAt: now.Add(-time.Minute),
		ResultPolicy: models.ResultPolicyAfterClose, AllowedDomain: "std.mast.ac.ug",
	}
	if err := db.Create(election).Error; err != nil {
		t.Fatal(err)
	}
	position := &models.Position{
		ElectionID: models.Ptr(election.ID), Title: "President", QuestionType: models.QuestionTypeSingleChoice,
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
		"allowed_domains": "@std.mast.ac.ug",
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

func TestElectionListHidesOtherPeoplesDrafts(t *testing.T) {
	db := rbacDB(t)
	now := time.Date(2026, 11, 1, 8, 0, 0, 0, time.UTC)
	clock := now
	engine := lifecycleEngine(t, db, &clock)
	auth := testkit.Auth(testkit.Config(), db)
	admin := rbacUser(t, db, "list-admin@must.ac.ug", models.RoleAdmin)
	voter := rbacUser(t, db, "list-voter@std.mast.ac.ug", models.RoleStudentVoter)
	adminToken := testkit.SignIn(t, db, auth, admin)
	voterToken := testkit.SignIn(t, db, auth, voter)
	client := testkit.New(t, engine)

	created := createLifecycleElection(t, client, adminToken,
		now.Add(24*time.Hour), now.Add(30*time.Hour))

	var mine []lifecycleElection
	client.As(adminToken).Get("/api/v1/elections").AssertOK().BindData(&mine)
	if len(mine) != 1 || mine[0].ID != created.ID {
		t.Fatalf("admin list = %+v, want the draft they created", mine)
	}

	var hidden []lifecycleElection
	client.As(voterToken).Get("/api/v1/elections").AssertOK().BindData(&hidden)
	if len(hidden) != 0 {
		t.Fatalf("voter list = %+v, want no other user's draft", hidden)
	}

	if err := db.Model(&models.Election{}).Where("id = ?", created.ID).
		Update("status", models.ElectionStatusOpen).Error; err != nil {
		t.Fatal(err)
	}
	var visible []lifecycleElection
	client.As(voterToken).Get("/api/v1/elections").AssertOK().BindData(&visible)
	if len(visible) != 1 || visible[0].ID != created.ID {
		t.Fatalf("voter list = %+v, want the open election", visible)
	}
}

func TestRestrictedElectionIsHiddenFromUnlistedStudents(t *testing.T) {
	db := rbacDB(t)
	now := time.Date(2026, 11, 1, 8, 0, 0, 0, time.UTC)
	clock := now
	engine := lifecycleEngine(t, db, &clock)
	auth := testkit.Auth(testkit.Config(), db)
	admin := rbacUser(t, db, "restricted-admin@must.ac.ug", models.RoleAdmin)
	voter := rbacUser(t, db, "2024bse099@std.must.ac.ug", models.RoleStudentVoter)
	adminToken := testkit.SignIn(t, db, auth, admin)
	voterToken := testkit.SignIn(t, db, auth, voter)
	client := testkit.New(t, engine)

	created := createLifecycleElection(t, client, adminToken,
		now.Add(24*time.Hour), now.Add(30*time.Hour))
	if err := db.Model(&models.Election{}).Where("id = ?", created.ID).
		Updates(map[string]any{
			"status":         models.ElectionStatusOpen,
			"require_roster": true,
		}).Error; err != nil {
		t.Fatal(err)
	}

	var list []lifecycleElection
	client.As(voterToken).Get("/api/v1/elections").AssertOK().BindData(&list)
	if len(list) != 0 {
		t.Fatalf("an unlisted student sees a restricted election: %+v", list)
	}

	if err := db.Create(&models.ElectionRoster{
		ElectionID: created.ID, StudentNumber: "24/U/0099", Email: voter.Email,
		FullName: "Listed Voter", Status: models.RosterStatusEligible,
	}).Error; err != nil {
		t.Fatal(err)
	}
	client.As(voterToken).Get("/api/v1/elections").AssertOK().BindData(&list)
	if len(list) != 1 || list[0].ID != created.ID {
		t.Fatalf("a listed student does not see the election: %+v", list)
	}

	// Closing it does not publish the list to everyone.
	if err := db.Model(&models.Election{}).Where("id = ?", created.ID).
		Update("status", models.ElectionStatusClosed).Error; err != nil {
		t.Fatal(err)
	}
	other := rbacUser(t, db, "2024bse098@std.must.ac.ug", models.RoleStudentVoter)
	otherToken := testkit.SignIn(t, db, auth, other)
	client.As(otherToken).Get("/api/v1/elections").AssertOK().BindData(&list)
	if len(list) != 0 {
		t.Fatalf("a closed restricted election leaked to an unlisted student: %+v", list)
	}
}

// TestElectionCreateLeavesDomainOpenToAnyStudent guards the bug where creating
// an election silently defaulted the voter domain to one campus address space,
// hiding the election from every voter whose email used a different domain.
func TestElectionCreateLeavesDomainOpenToAnyStudent(t *testing.T) {
	db := rbacDB(t)
	now := time.Date(2026, 11, 1, 8, 0, 0, 0, time.UTC)
	clock := now
	engine := lifecycleEngine(t, db, &clock)
	auth := testkit.Auth(testkit.Config(), db)
	admin := rbacUser(t, db, "domain-admin@must.ac.ug", models.RoleAdmin)
	voter := rbacUser(t, db, "2024bse088@std.must.ac.ug", models.RoleStudentVoter)
	adminToken := testkit.SignIn(t, db, auth, admin)
	voterToken := testkit.SignIn(t, db, auth, voter)
	client := testkit.New(t, engine)

	created := createLifecycleElection(t, client, adminToken,
		now.Add(24*time.Hour), now.Add(30*time.Hour))

	var stored models.Election
	if err := db.Where("id = ?", created.ID).First(&stored).Error; err != nil {
		t.Fatal(err)
	}
	if stored.AllowedDomain != "" {
		t.Fatalf("allowed_domain = %q, want empty so any registration email qualifies", stored.AllowedDomain)
	}

	// Drafts are creator-only, so publish the election to test the visibility
	// rule itself rather than the draft filter.
	if err := db.Model(&models.Election{}).Where("id = ?", created.ID).
		Update("status", models.ElectionStatusOpen).Error; err != nil {
		t.Fatal(err)
	}
	var visible []lifecycleElection
	client.As(voterToken).Get("/api/v1/elections").AssertOK().BindData(&visible)
	if len(visible) != 1 || visible[0].ID != created.ID {
		t.Fatalf("voter list = %+v, want the domain-agnostic election", visible)
	}
}

// TestAdminCanDeleteDraftButNotAnElectionWithBallots covers the delete rule:
// a draft that has recorded no ballot may be removed, but an election that
// already holds ballots must be closed instead so the evidence survives.
func TestAdminCanDeleteDraftButNotAnElectionWithBallots(t *testing.T) {
	db := rbacDB(t)
	now := time.Date(2026, 11, 1, 8, 0, 0, 0, time.UTC)
	clock := now
	engine := lifecycleEngine(t, db, &clock)
	auth := testkit.Auth(testkit.Config(), db)
	admin := rbacUser(t, db, "delete-admin@must.ac.ug", models.RoleAdmin)
	token := testkit.SignIn(t, db, auth, admin)
	client := testkit.New(t, engine)

	created := createLifecycleElection(t, client, token, now.Add(24*time.Hour), now.Add(30*time.Hour))
	position := &models.Position{
		ElectionID: models.Ptr(created.ID), Title: "President", QuestionType: models.QuestionTypeSingleChoice,
		MinSelections: 1, MaxSelections: 1, Seats: 1,
	}
	if err := db.Create(position).Error; err != nil {
		t.Fatal(err)
	}

	client.As(token).Delete("/api/v1/elections/" + created.ID).AssertOK()
	var remaining int64
	db.Model(&models.Election{}).Where("id = ?", created.ID).Count(&remaining)
	if remaining != 0 {
		t.Fatalf("election rows after delete = %d, want 0", remaining)
	}
	var audit models.ElectionAuditLog
	if err := db.Where("election_id = ? AND action = ?", created.ID, models.AuditActionElectionDeleted).
		First(&audit).Error; err != nil {
		t.Fatal("the deletion was not audited")
	}

	// An election that has already recorded a ballot is refused.
	open := &models.Election{
		ID: "e-open", Title: "Open Election", Status: models.ElectionStatusOpen,
		StartsAt: now.Add(-time.Hour), EndsAt: now.Add(time.Hour),
		ResultPolicy: models.ResultPolicyAfterClose, AllowedDomain: "std.mast.ac.ug",
	}
	if err := db.Create(open).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.AnonymousBallot{ElectionID: open.ID, Status: "VALID", CastAt: now}).Error; err != nil {
		t.Fatal(err)
	}
	client.As(token).Delete("/api/v1/elections/" + open.ID).AssertConflict()

	var kept int64
	db.Model(&models.Election{}).Where("id = ?", open.ID).Count(&kept)
	if kept != 1 {
		t.Fatalf("election with ballots was removed: count = %d, want 1", kept)
	}
}
