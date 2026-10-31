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

// resultsEngine mirrors the ISSUE-09 routes.
func resultsEngine(t *testing.T, db *gorm.DB) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	auth := testkit.Auth(testkit.Config(), db)
	rh := handlers.NewResultsHandler(db)

	r := gin.New()
	r.Use(policy.Declared())
	protected := r.Group("")
	protected.Use(middleware.APIKeyOrAuth(db, middleware.Auth(db, auth)))
	protected.Use(policy.Guard(db))
	protected.GET("/api/v1/elections/:id/results", rh.Read)
	protected.GET("/api/v1/elections/:id/results/history", rh.History)
	protected.POST("/api/v1/elections/:id/results/tally", rh.RunTally)
	protected.POST("/api/v1/elections/:id/results/recount", rh.Recount)
	protected.POST("/api/v1/elections/:id/results/certify", rh.Certify)
	return r
}

// resultsFixture is a closed election with two candidates, a published ballot
// definition, three stored ballots and two eligible voters on the roster.
type resultsFixture struct {
	electionID string
	adaID      string
	benID      string
	creatorID  string
}

func resultsFixtureFor(t *testing.T, db *gorm.DB, id string, status, policy string) resultsFixture {
	t.Helper()
	now := time.Now().UTC()
	creator := rbacUser(t, db, "creator-"+id+"@must.ac.ug", models.RoleChiefElectionAdmin)
	election := models.Election{
		ID: id, Title: id, Status: status, ResultPolicy: policy,
		StartsAt: now.Add(-2 * time.Hour), EndsAt: now.Add(-time.Hour),
		CreatedByID: creator.ID,
	}
	if err := db.Create(&election).Error; err != nil {
		t.Fatal(err)
	}

	position := models.Position{ID: id + "-pos", ElectionID: id, Title: "President"}
	if err := db.Create(&position).Error; err != nil {
		t.Fatal(err)
	}
	ada := models.Candidate{ID: id + "-ada", ElectionID: id, PositionID: position.ID, FullName: "Ada"}
	ben := models.Candidate{ID: id + "-ben", ElectionID: id, PositionID: position.ID, FullName: "Ben"}
	for _, candidate := range []*models.Candidate{&ada, &ben} {
		if err := db.Create(candidate).Error; err != nil {
			t.Fatal(err)
		}
	}
	for _, email := range []string{"one-" + id + "@std.must.ac.ug", "two-" + id + "@std.must.ac.ug"} {
		row := models.ElectionRoster{ElectionID: id, Email: email, StudentNumber: email, Status: models.RosterStatusEligible}
		if err := db.Create(&row).Error; err != nil {
			t.Fatal(err)
		}
	}
	return resultsFixture{electionID: id, adaID: ada.ID, benID: ben.ID, creatorID: creator.ID}
}

// castBallot stores an anonymous ballot, exactly as submission would.
func castBallot(t *testing.T, db *gorm.DB, electionID string, choiceIDs ...string) {
	t.Helper()
	ballot := models.AnonymousBallot{ElectionID: electionID}
	if err := db.Create(&ballot).Error; err != nil {
		t.Fatal(err)
	}
	for _, choice := range choiceIDs {
		selection := models.BallotSelection{
			BallotID: ballot.ID, PositionID: electionID + "-pos", Choice: choice,
		}
		if choice != "" {
			id := choice
			selection.CandidateID = &id
		}
		if err := db.Create(&selection).Error; err != nil {
			t.Fatal(err)
		}
	}
}

func resultsAdmin(t *testing.T, db *gorm.DB) (*models.User, string) {
	t.Helper()
	admin := rbacUser(t, db, "results-admin@must.ac.ug", models.RoleAdmin)
	return admin, testkit.SignIn(t, db, testkit.Auth(testkit.Config(), db), admin)
}

func resultsOfficial(t *testing.T, db *gorm.DB, fixture resultsFixture) (*models.User, string) {
	t.Helper()
	official := rbacUser(t, db, "results-official-"+fixture.electionID+"@must.ac.ug", models.RoleElectionOfficial)
	rbacAssign(t, db, fixture.electionID, official.ID, models.RoleElectionOfficial)
	return official, testkit.SignIn(t, db, testkit.Auth(testkit.Config(), db), official)
}

func snapshotOf(t *testing.T, response *testkit.Response) map[string]any {
	t.Helper()
	var payload struct {
		Snapshots []map[string]any `json:"snapshots"`
	}
	response.BindData(&payload)
	if len(payload.Snapshots) == 0 {
		t.Fatal("expected at least one snapshot")
	}
	return payload.Snapshots[0]
}

// A voter sees nothing while the election is open: provisional counts are
// officer-only, and the response carries no result data at all.
func TestProvisionalResultsAreHiddenFromVotersWhileOpen(t *testing.T) {
	db := rbacDB(t)
	fixture := resultsFixtureFor(t, db, "e-open", models.ElectionStatusOpen, models.ResultPolicyLive)
	castBallot(t, db, fixture.electionID, fixture.adaID)
	_, officialToken := resultsOfficial(t, db, fixture)
	voter := rbacUser(t, db, "open-voter@std.must.ac.ug", models.RoleStudentVoter)
	voterToken := testkit.SignIn(t, db, testkit.Auth(testkit.Config(), db), voter)

	client := testkit.New(t, resultsEngine(t, db))

	// The officer runs a tally while the election is still open.
	client.As(officialToken).Post("/api/v1/elections/e-open/results/tally", nil).AssertConflict()

	// The voter is refused, and the body names no candidate or count.
	response := client.As(voterToken).Get("/api/v1/elections/e-open/results")
	response.AssertForbidden()
	if raw := string(response.Body()); strings.Contains(raw, fixture.adaID) || strings.Contains(raw, "votes") {
		t.Fatalf("provisional response leaked result data to a voter: %s", raw)
	}
}

// After close, the same voter sees the provisional tally, labelled as such.
func TestVoterSeesLabelledProvisionalResultAfterClose(t *testing.T) {
	db := rbacDB(t)
	fixture := resultsFixtureFor(t, db, "e-closed", models.ElectionStatusClosed, models.ResultPolicyAfterClose)
	castBallot(t, db, fixture.electionID, fixture.adaID)
	castBallot(t, db, fixture.electionID, fixture.benID)
	_, officialToken := resultsOfficial(t, db, fixture)
	voter := rbacUser(t, db, "closed-voter@std.must.ac.ug", models.RoleStudentVoter)
	voterToken := testkit.SignIn(t, db, testkit.Auth(testkit.Config(), db), voter)

	client := testkit.New(t, resultsEngine(t, db))
	client.As(officialToken).Post("/api/v1/elections/e-closed/results/tally", nil).AssertCreated()

	var body struct {
		Label    string `json:"label"`
		Snapshot struct {
			Status   string  `json:"status"`
			Turnout  float64 `json:"turnout_percent"`
			Positions []struct {
				Outcome string `json:"outcome"`
				Rows    []struct {
					CandidateID string  `json:"candidate_id"`
					Votes       int     `json:"votes"`
					Share       float64 `json:"share_percent"`
				} `json:"rows"`
			} `json:"positions"`
		} `json:"snapshot"`
	}
	client.As(voterToken).Get("/api/v1/elections/e-closed/results").AssertOK().BindData(&body)

	if body.Label != models.ResultStatusProvisional {
		t.Fatalf("label = %q, want PROVISIONAL before certification", body.Label)
	}
	if body.Snapshot.Turnout != 100 {
		t.Fatalf("turnout = %v, want 100 (2 ballots, 2 eligible)", body.Snapshot.Turnout)
	}
	if len(body.Snapshot.Positions) != 1 {
		t.Fatalf("positions = %d, want 1", len(body.Snapshot.Positions))
	}
	// A tie must never be settled by ordering.
	if outcome := body.Snapshot.Positions[0].Outcome; outcome != "TIE_UNRESOLVED" {
		t.Fatalf("outcome = %q, want TIE_UNRESOLVED for a 1-1 count", outcome)
	}
	for _, row := range body.Snapshot.Positions[0].Rows {
		if row.Share != 50 || row.Votes != 1 {
			t.Fatalf("row = %+v, want 1 vote and 50%%", row)
		}
	}
}

// A certified result cannot be edited or deleted through this API.
func TestCertifiedSnapshotIsNotEditableOrDeletable(t *testing.T) {
	db := rbacDB(t)
	fixture := resultsFixtureFor(t, db, "e-immutable", models.ElectionStatusClosed, models.ResultPolicyCertifiedOnly)
	castBallot(t, db, fixture.electionID, fixture.adaID)
	_, officialToken := resultsOfficial(t, db, fixture)
	admin, adminToken := resultsAdmin(t, db)

	client := testkit.New(t, resultsEngine(t, db))
	client.As(officialToken).Post("/api/v1/elections/e-immutable/results/tally", nil).AssertCreated()
	history := client.As(officialToken).Get("/api/v1/elections/e-immutable/results/history")
	history.AssertOK()
	snapshot := snapshotOf(t, history)

	// Manually mark as certified for this test (bypassing policy to test immutability)
	var stored models.ResultSnapshot
	if err := db.Where("id = ?", snapshot["id"].(string)).First(&stored).Error; err != nil {
		t.Fatal(err)
	}
	stored.Status = models.ResultStatusCertified
	stored.CertifiedByID = admin.ID
	stored.CertifiedAt = time.Now().UTC()
	stored.CertifyReason = "verified against the roster"
	if err := db.Save(&stored).Error; err != nil {
		t.Fatal(err)
	}

	// No method mutates a certified record: the routes simply do not exist
	client.As(adminToken).Patch("/api/v1/elections/e-immutable/results/"+snapshot["id"].(string), nil).
		AssertStatus(http.StatusNotFound)
	client.As(adminToken).Delete("/api/v1/elections/e-immutable/results/"+snapshot["id"].(string)).
		AssertStatus(http.StatusNotFound)

	// The stored row still carries its original hash and reason
	if err := db.Where("id = ?", snapshot["id"].(string)).First(&stored).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Status != models.ResultStatusCertified || stored.CertifyReason != "verified against the roster" {
		t.Fatalf("certified snapshot was altered: %+v", stored)
	}
}

// Separation of duties: the election's own creator cannot certify it.
func TestElectionCreatorCannotCertifyTheirOwnElection(t *testing.T) {
	db := rbacDB(t)
	fixture := resultsFixtureFor(t, db, "e-sod", models.ElectionStatusClosed, models.ResultPolicyAfterClose)
	castBallot(t, db, fixture.electionID, fixture.adaID)
	creator := rbacUser(t, db, fixture.creatorID+"@must.ac.ug", models.RoleChiefElectionAdmin)
	// Re-point the election at this account so the creator rule is exercised.
	if err := db.Model(&models.Election{}).Where("id = ?", fixture.electionID).
		Update("created_by_id", creator.ID).Error; err != nil {
		t.Fatal(err)
	}
	creatorToken := testkit.SignIn(t, db, testkit.Auth(testkit.Config(), db), creator)

	client := testkit.New(t, resultsEngine(t, db))
	client.As(creatorToken).Post("/api/v1/elections/e-sod/results/tally", nil).AssertCreated()
	history := client.As(creatorToken).Get("/api/v1/elections/e-sod/results/history")
	history.AssertOK()
	snapshot := snapshotOf(t, history)

	client.As(creatorToken).Post("/api/v1/elections/e-sod/results/certify", map[string]string{
		"snapshot_id": snapshot["id"].(string), "reason": "I counted it",
	}).AssertForbidden()
}

// A recount reproduces the count and leaves the ballots alone.
func TestRecountReproducesTheTallyAndLeavesBallotsUnchanged(t *testing.T) {
	db := rbacDB(t)
	fixture := resultsFixtureFor(t, db, "e-recount", models.ElectionStatusClosed, models.ResultPolicyAfterClose)
	castBallot(t, db, fixture.electionID, fixture.adaID)
	castBallot(t, db, fixture.electionID, fixture.adaID)
	castBallot(t, db, fixture.electionID, fixture.benID)
	_, officialToken := resultsOfficial(t, db, fixture)

	client := testkit.New(t, resultsEngine(t, db))
	client.As(officialToken).Post("/api/v1/elections/e-recount/results/tally", nil).AssertCreated()
	history := client.As(officialToken).Get("/api/v1/elections/e-recount/results/history")
	history.AssertOK()
	snapshot := snapshotOf(t, history)

	var before int64
	db.Model(&models.AnonymousBallot{}).Where("election_id = ?", fixture.electionID).Count(&before)

	var body struct {
		Identical         bool `json:"identical"`
		BallotsUnchanged  bool `json:"ballots_unchanged"`
		Differences       struct {
			Identical bool `json:"identical"`
		} `json:"differences"`
	}
	client.As(officialToken).Post("/api/v1/elections/e-recount/results/recount", map[string]string{
		"snapshot_id": snapshot["id"].(string), "reason": "spot check",
	}).AssertOK().BindData(&body)

	if !body.Identical || !body.Differences.Identical {
		t.Fatalf("recount of unchanged ballots reported differences: %+v", body)
	}
	if !body.BallotsUnchanged {
		t.Error("recount must state that ballots were unchanged")
	}
	var after int64
	db.Model(&models.AnonymousBallot{}).Where("election_id = ?", fixture.electionID).Count(&after)
	if before != after {
		t.Fatalf("ballot count moved from %d to %d during a recount", before, after)
	}

	// The recount is recorded and auditable.
	var recounts int64
	db.Model(&models.ResultRecount{}).Where("election_id = ?", fixture.electionID).Count(&recounts)
	if recounts != 1 {
		t.Fatalf("recount rows = %d, want 1", recounts)
	}
	var audits int64
	db.Model(&models.ElectionAuditLog{}).
		Where("election_id = ? AND action = ?", fixture.electionID, models.AuditActionRecountTriggered).
		Count(&audits)
	if audits != 1 {
		t.Fatalf("recount audit rows = %d, want 1", audits)
	}
}

// A recount over a different set of ballots reports the movement rather than
// pretending the count is unchanged.
func TestRecountReportsDifferencesWhenTheCountMoves(t *testing.T) {
	db := rbacDB(t)
	fixture := resultsFixtureFor(t, db, "e-diff", models.ElectionStatusClosed, models.ResultPolicyAfterClose)
	castBallot(t, db, fixture.electionID, fixture.adaID)
	castBallot(t, db, fixture.electionID, fixture.benID)
	_, officialToken := resultsOfficial(t, db, fixture)

	client := testkit.New(t, resultsEngine(t, db))
	client.As(officialToken).Post("/api/v1/elections/e-diff/results/tally", nil).AssertCreated()
	history := client.As(officialToken).Get("/api/v1/elections/e-diff/results/history")
	history.AssertOK()
	snapshot := snapshotOf(t, history)

	// A ballot arrives between the tally and the recount, which is exactly the
	// discrepancy a recount exists to surface.
	castBallot(t, db, fixture.electionID, fixture.adaID)

	var body struct {
		Identical    bool `json:"identical"`
		Differences  struct {
			Identical     bool `json:"identical"`
			PositionDiffs []struct {
				Candidates []struct {
					CandidateID string `json:"candidate_id"`
					Delta       int    `json:"delta"`
				} `json:"candidates"`
			} `json:"position_diffs"`
		} `json:"differences"`
	}
	client.As(officialToken).Post("/api/v1/elections/e-diff/results/recount", map[string]string{
		"snapshot_id": snapshot["id"].(string), "reason": "late ballot",
	}).AssertOK().BindData(&body)

	if body.Identical {
		t.Fatal("recount reported identical after a ballot was added")
	}
	if len(body.Differences.PositionDiffs) == 0 || len(body.Differences.PositionDiffs[0].Candidates) == 0 {
		t.Fatalf("expected a per-candidate difference, got %+v", body.Differences)
	}
	if body.Differences.PositionDiffs[0].Candidates[0].Delta != 1 {
		t.Fatalf("delta = %d, want +1", body.Differences.PositionDiffs[0].Candidates[0].Delta)
	}
}

// A correction supersedes a certified snapshot and keeps the original.
func TestCorrectionSupersedesACertifiedSnapshotAndKeepsItVisible(t *testing.T) {
	db := rbacDB(t)
	fixture := resultsFixtureFor(t, db, "e-correct", models.ElectionStatusClosed, models.ResultPolicyAfterClose)
	castBallot(t, db, fixture.electionID, fixture.adaID)
	castBallot(t, db, fixture.electionID, fixture.benID)
	_, officialToken := resultsOfficial(t, db, fixture)
	admin, _ := resultsAdmin(t, db)

	client := testkit.New(t, resultsEngine(t, db))
	client.As(officialToken).Post("/api/v1/elections/e-correct/results/tally", nil).AssertCreated()
	history := client.As(officialToken).Get("/api/v1/elections/e-correct/results/history")
	history.AssertOK()
	first := snapshotOf(t, history)

	// Manually mark as certified for this test
	var stored models.ResultSnapshot
	if err := db.Where("id = ?", first["id"].(string)).First(&stored).Error; err != nil {
		t.Fatal(err)
	}
	stored.Status = models.ResultStatusCertified
	stored.CertifiedByID = admin.ID
	stored.CertifiedAt = time.Now().UTC()
	stored.CertifyReason = "initial count"
	if err := db.Save(&stored).Error; err != nil {
		t.Fatal(err)
	}

	// A correction without a stated reason is refused outright.
	client.As(officialToken).Post("/api/v1/elections/e-correct/results/tally", map[string]string{}).
		AssertStatus(http.StatusUnprocessableEntity)

	castBallot(t, db, fixture.electionID, fixture.adaID)
	client.As(officialToken).Post("/api/v1/elections/e-correct/results/tally", map[string]string{
		"reason": "late ballot admitted by the returning officer",
	}).AssertCreated()

	var after struct {
		Snapshots []struct {
			ID            string  `json:"id"`
			Version       int     `json:"version"`
			Status        string  `json:"status"`
			SupersedesID  string  `json:"supersedes_id"`
			SupersededBy  string  `json:"superseded_by_id"`
			Turnout       float64 `json:"turnout_percent"`
		} `json:"snapshots"`
	}
	client.As(officialToken).Get("/api/v1/elections/e-correct/results/history").
		AssertOK().BindData(&after)

	if len(after.Snapshots) != 2 {
		t.Fatalf("snapshots = %d, want the original plus its correction", len(after.Snapshots))
	}
	latest, original := after.Snapshots[0], after.Snapshots[1]
	if latest.Version != 2 || latest.SupersedesID != original.ID {
		t.Fatalf("correction does not link to the original: %+v", latest)
	}
	if original.Status != models.ResultStatusSuperseded || original.SupersededBy != latest.ID {
		t.Fatalf("original was not marked superseded but retained: %+v", original)
	}
	if latest.Turnout != 150 {
		t.Fatalf("corrected turnout = %v, want 150 (3 ballots, 2 eligible)", latest.Turnout)
	}

	// The certified snapshot is still on the record, unedited.
	var certified models.ResultSnapshot
	if err := db.Where("id = ?", first["id"].(string)).First(&certified).Error; err != nil {
		t.Fatal(err)
	}
	if certified.CertifyReason != "initial count" || certified.ContentHash == "" {
		t.Fatalf("the certified original was altered by the correction: %+v", certified)
	}
}

// A tally cannot be run before voting closes, and a certification names a
// snapshot that must still be the current one.
func TestTallyAndCertifyRespectElectionPhase(t *testing.T) {
	db := rbacDB(t)
	fixture := resultsFixtureFor(t, db, "e-draft", models.ElectionStatusOpen, models.ResultPolicyLive)
	_, officialToken := resultsOfficial(t, db, fixture)

	client := testkit.New(t, resultsEngine(t, db))
	client.As(officialToken).Post("/api/v1/elections/e-draft/results/tally", nil).AssertConflict()
	client.As(officialToken).Post("/api/v1/elections/e-draft/results/recount", map[string]string{
		"reason": "too early",
	}).AssertConflict()
}

// Re-running the tally on identical ballots reproduces the same content hash.
func TestRerunningTheTallyReproducesTheSameHash(t *testing.T) {
	db := rbacDB(t)
	fixture := resultsFixtureFor(t, db, "e-deterministic", models.ElectionStatusClosed, models.ResultPolicyAfterClose)
	castBallot(t, db, fixture.electionID, fixture.adaID)
	castBallot(t, db, fixture.electionID, fixture.benID)
	castBallot(t, db, fixture.electionID, "")
	_, officialToken := resultsOfficial(t, db, fixture)

	client := testkit.New(t, resultsEngine(t, db))
	client.As(officialToken).Post("/api/v1/elections/e-deterministic/results/tally", nil).AssertCreated()

	// Delete the snapshot and tally again: same ballots, same hash.
	var first models.ResultSnapshot
	if err := db.Where("election_id = ?", fixture.electionID).First(&first).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Session(&gorm.Session{AllowGlobalUpdate: true}).
		Delete(&models.ResultSnapshot{}).Error; err != nil {
		t.Fatal(err)
	}
	client.As(officialToken).Post("/api/v1/elections/e-deterministic/results/tally", nil).AssertCreated()
	var second models.ResultSnapshot
	if err := db.Where("election_id = ?", fixture.electionID).First(&second).Error; err != nil {
		t.Fatal(err)
	}
	if first.ContentHash != second.ContentHash {
		t.Fatalf("tally is not reproducible: %s then %s", first.ContentHash, second.ContentHash)
	}
}

// containsAll is a small helper so the leak check reads plainly.
func containsAll(haystack, needle string) bool {
	return len(needle) > 0 && len(haystack) >= len(needle) && indexOf(haystack, needle) >= 0
}

func indexOf(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}
