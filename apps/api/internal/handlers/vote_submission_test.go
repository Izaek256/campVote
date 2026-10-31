package handlers_test

import (
	"fmt"
	"strings"
	"sync"
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

// voteEngine mirrors the protected routes ISSUE-08 touches.
func voteEngine(t *testing.T, db *gorm.DB) *gin.Engine {
	t.Helper()
	return voteEngineWithHandler(t, db, handlers.NewBallotHandler(db))
}

func voteEngineWithHandler(t *testing.T, db *gorm.DB, bh *handlers.BallotHandler) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	auth := testkit.Auth(testkit.Config(), db)

	r := gin.New()
	r.Use(policy.Declared())
	protected := r.Group("")
	protected.Use(middleware.APIKeyOrAuth(db, middleware.Auth(db, auth)))
	protected.Use(policy.Guard(db))
	protected.GET("/api/v1/elections/:id/ballot", bh.View)
	protected.GET("/api/v1/elections/:id/my-vote-status", bh.VoteStatus)
	protected.POST("/api/v1/elections/:id/ballot/start", bh.Start)
	protected.POST("/api/v1/elections/:id/ballot/submit", bh.Submit)
	return r
}

func publishVersion(t *testing.T, db *gorm.DB, electionID, publisher string) *models.BallotVersion {
	t.Helper()
	content, err := handlers.NewBallotVersionHandler(db).BuildBallotContent(electionID)
	if err != nil {
		t.Fatal(err)
	}
	version := &models.BallotVersion{
		ElectionID: electionID, Version: 1, ContentJSON: content,
		PublishedBy: publisher, Status: models.BallotVersionStatusCurrent,
	}
	if err := db.Create(version).Error; err != nil {
		t.Fatal(err)
	}
	return version
}

func voteCounts(t *testing.T, db *gorm.DB, electionID string) (participants, ballots int64) {
	t.Helper()
	db.Model(&models.VoterParticipation{}).Where("election_id = ?", electionID).Count(&participants)
	db.Model(&models.AnonymousBallot{}).Where("election_id = ?", electionID).Count(&ballots)
	return participants, ballots
}

func TestVoteSubmissionIsIdempotentAndRecordsOneBallot(t *testing.T) {
	db := rbacDB(t)
	now := time.Now().UTC()
	fixture := ballotElection(t, db, "e-idem", now.Add(-time.Hour), now.Add(time.Hour))
	voter := testkit.User(t, db, "idem@std.must.ac.ug", models.RoleStudentVoter)
	listed(t, db, fixture.election.ID, "22/U/0100", voter.Email)

	officer := testkit.User(t, db, "officer@must.ac.ug", models.RoleElectionOfficial)
	version := publishVersion(t, db, fixture.election.ID, officer.ID)

	auth := testkit.Auth(testkit.Config(), db)
	token := testkit.SignIn(t, db, auth, voter)
	client := testkit.New(t, voteEngine(t, db))

	ballot := client.As(token).Get("/api/v1/elections/e-idem/ballot").AssertOK()
	var view struct {
		BallotVersionID string `json:"ballot_version_id"`
		Positions       []struct {
			ID string `json:"id"`
		} `json:"positions"`
	}
	ballot.BindData(&view)
	if view.BallotVersionID != version.ID {
		t.Fatalf("ballot version = %q, want %q", view.BallotVersionID, version.ID)
	}

	selections := []map[string]string{{"position_id": fixture.position.ID, "candidate_id": fixture.candidate.ID}}

	// A payload built against a different definition is refused, and nothing
	// is recorded.
	client.As(token).Post("/api/v1/elections/e-idem/ballot/submit", map[string]any{
		"selections": selections, "ballot_version_id": "stale", "idempotency_key": "key-1",
	}).AssertConflict()
	if p, b := voteCounts(t, db, fixture.election.ID); p != 0 || b != 0 {
		t.Fatalf("stale submission wrote participation=%d ballots=%d", p, b)
	}

	client.As(token).Post("/api/v1/elections/e-idem/ballot/submit", map[string]any{
		"selections": selections, "ballot_version_id": version.ID, "idempotency_key": "key-1",
	}).AssertCreated()

	// The same key is a retry, not a second vote.
	client.As(token).Post("/api/v1/elections/e-idem/ballot/submit", map[string]any{
		"selections": selections, "ballot_version_id": version.ID, "idempotency_key": "key-1",
	}).AssertOK()

	// A new key is a second attempt at voting and is refused.
	client.As(token).Post("/api/v1/elections/e-idem/ballot/submit", map[string]any{
		"selections": selections, "ballot_version_id": version.ID, "idempotency_key": "key-2",
	}).AssertConflict()

	if p, b := voteCounts(t, db, fixture.election.ID); p != 1 || b != 1 {
		t.Fatalf("after retries participation=%d ballots=%d, want 1/1", p, b)
	}

	// The audit event names participation only; it must not contain a choice.
	var audit models.ElectionAuditLog
	if err := db.Where("election_id = ? AND action = ?", fixture.election.ID, models.AuditActionVoteCast).
		First(&audit).Error; err != nil {
		t.Fatalf("vote audit event missing: %v", err)
	}
	if containsChoice(audit.Details, fixture.candidate.ID) {
		t.Fatalf("audit details contain a ballot choice: %q", audit.Details)
	}

	status := client.As(token).Get("/api/v1/elections/e-idem/my-vote-status").AssertOK()
	var voteStatus map[string]any
	status.BindData(&voteStatus)
	if voted, _ := voteStatus["voted"].(bool); !voted {
		t.Fatalf("vote status = %v, want voted", voteStatus)
	}
	if _, ok := voteStatus["recorded_at"]; !ok {
		t.Fatalf("vote status has no recorded_at: %v", voteStatus)
	}
}

func TestParallelSubmissionsByOneVoterRecordExactlyOneBallot(t *testing.T) {
	db := rbacDB(t)
	// ":memory:" gives every pooled connection its own empty database. One
	// connection serialises the parallel requests the way Postgres would, so
	// the constraint is what decides, not the harness.
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)

	now := time.Now().UTC()
	fixture := ballotElection(t, db, "e-race", now.Add(-time.Hour), now.Add(time.Hour))
	voter := testkit.User(t, db, "race@std.must.ac.ug", models.RoleStudentVoter)
	listed(t, db, fixture.election.ID, "22/U/0101", voter.Email)
	version := publishVersion(t, db, fixture.election.ID, testkit.User(t, db, "race-officer@must.ac.ug", models.RoleElectionOfficial).ID)

	auth := testkit.Auth(testkit.Config(), db)
	token := testkit.SignIn(t, db, auth, voter)
	client := testkit.New(t, voteEngine(t, db))

	const attempts = 8
	codes := make([]int, attempts)
	var wg sync.WaitGroup
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			response := client.As(token).Post("/api/v1/elections/e-race/ballot/submit", map[string]any{
				"ballot_version_id": version.ID,
				"idempotency_key":   fmt.Sprintf("parallel-%d", index),
				"selections": []map[string]string{{
					"position_id": fixture.position.ID, "candidate_id": fixture.candidate.ID,
				}},
			})
			codes[index] = response.Code()
		}(i)
	}
	wg.Wait()

	accepted := 0
	for _, code := range codes {
		switch code {
		case 201:
			accepted++
		case 409:
		default:
			t.Fatalf("unexpected status %d in %v", code, codes)
		}
	}
	if accepted != 1 {
		t.Fatalf("accepted submissions = %d, want 1 (codes %v)", accepted, codes)
	}
	if p, b := voteCounts(t, db, fixture.election.ID); p != 1 || b != 1 {
		t.Fatalf("participation=%d ballots=%d, want 1/1", p, b)
	}
}

func TestRejectedBallotWritesNothing(t *testing.T) {
	db := rbacDB(t)
	now := time.Now().UTC()
	fixture := ballotElection(t, db, "e-reject", now.Add(-time.Hour), now.Add(time.Hour))
	voter := testkit.User(t, db, "reject@std.must.ac.ug", models.RoleStudentVoter)
	listed(t, db, fixture.election.ID, "22/U/0102", voter.Email)
	token := testkit.SignIn(t, db, testkit.Auth(testkit.Config(), db), voter)
	client := testkit.New(t, voteEngine(t, db))

	client.As(token).Post("/api/v1/elections/e-reject/ballot/submit", map[string]any{
		"selections": []map[string]string{{"position_id": fixture.position.ID}},
	}).AssertBadRequest()

	if p, b := voteCounts(t, db, fixture.election.ID); p != 0 || b != 0 {
		t.Fatalf("rejected submission wrote participation=%d ballots=%d", p, b)
	}
}

// Closing-time race rule: acceptance is decided at commit time inside the
// submission transaction by the server clock. A request whose clock crosses
// ends_at between the gate check and the transaction is refused and writes
// nothing.
func TestSubmissionRefusedWhenServerClockCrossesCloseDuringRequest(t *testing.T) {
	db := rbacDB(t)
	now := time.Now().UTC()
	closes := now.Add(30 * time.Minute)
	fixture := ballotElection(t, db, "e-close-race", now.Add(-time.Hour), closes)
	voter := testkit.User(t, db, "closerace@std.must.ac.ug", models.RoleStudentVoter)
	listed(t, db, fixture.election.ID, "22/U/0109", voter.Email)
	version := publishVersion(t, db, fixture.election.ID, testkit.User(t, db, "closerace-officer@must.ac.ug", models.RoleElectionOfficial).ID)
	token := testkit.SignIn(t, db, testkit.Auth(testkit.Config(), db), voter)

	handler := handlers.NewBallotHandler(db)
	var calls int
	handler.Now = func() time.Time {
		calls++
		if calls == 1 {
			return now.Add(20 * time.Minute) // gate: still open
		}
		return now.Add(time.Hour) // transaction: past ends_at
	}
	client := testkit.New(t, voteEngineWithHandler(t, db, handler))

	response := client.As(token).Post("/api/v1/elections/e-close-race/ballot/submit", map[string]any{
		"ballot_version_id": version.ID,
		"selections": []map[string]string{{
			"position_id": fixture.position.ID, "candidate_id": fixture.candidate.ID,
		}},
	})
	response.AssertConflict()
	if p, b := voteCounts(t, db, fixture.election.ID); p != 0 || b != 0 {
		t.Fatalf("closed-election submission wrote participation=%d ballots=%d", p, b)
	}
}

func TestVoteStatusBeforeVotingIsFalseAndCarriesNoChoiceField(t *testing.T) {
	db := rbacDB(t)
	now := time.Now().UTC()
	fixture := ballotElection(t, db, "e-status", now.Add(-time.Hour), now.Add(time.Hour))
	voter := testkit.User(t, db, "status@std.must.ac.ug", models.RoleStudentVoter)
	listed(t, db, fixture.election.ID, "22/U/0103", voter.Email)
	token := testkit.SignIn(t, db, testkit.Auth(testkit.Config(), db), voter)
	client := testkit.New(t, voteEngine(t, db))

	var body map[string]any
	client.As(token).Get("/api/v1/elections/e-status/my-vote-status").AssertOK().BindData(&body)
	if voted, _ := body["voted"].(bool); voted {
		t.Fatalf("vote status = %v, want not voted", body)
	}
	for _, forbidden := range []string{"selections", "candidate_id", "choice", "ballot"} {
		if _, ok := body[forbidden]; ok {
			t.Fatalf("vote status leaks %q: %v", forbidden, body)
		}
	}
}

func containsChoice(details, candidateID string) bool {
	return candidateID != "" && strings.Contains(details, candidateID)
}
