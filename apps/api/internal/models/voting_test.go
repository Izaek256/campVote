package models_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"campvote/apps/api/internal/models"
)

// TestBallotPrivacyDecoupling enforces FR-PRIV-01, FR-PRIV-02, and FR-PRIV-03 via structural reflection.
// It verifies that AnonymousBallot and BallotSelection never have any field that could link to voter identity.
func TestBallotPrivacyDecoupling(t *testing.T) {
	forbiddenVoterFields := []string{
		"UserID", "UserId", "user_id",
		"VoterID", "VoterId", "voter_id",
		"Email", "email",
		"StudentNumber", "student_number",
		"RegNumber", "reg_number",
		"IPAddress", "ip_address",
		"SessionID", "session_id",
	}

	ballotType := reflect.TypeOf(models.AnonymousBallot{})
	for _, field := range forbiddenVoterFields {
		if _, found := ballotType.FieldByName(field); found {
			t.Fatalf("CRITICAL PRIVACY VIOLATION: AnonymousBallot contains forbidden voter identification field: %s", field)
		}
	}

	selectionType := reflect.TypeOf(models.BallotSelection{})
	for _, field := range forbiddenVoterFields {
		if _, found := selectionType.FieldByName(field); found {
			t.Fatalf("CRITICAL PRIVACY VIOLATION: BallotSelection contains forbidden voter identification field: %s", field)
		}
	}

	forbiddenBallotFieldsInParticipation := []string{
		"BallotID", "BallotId", "ballot_id",
		"CandidateID", "CandidateId", "candidate_id",
		"Choice", "choice",
		"Selections", "selections",
	}

	participationType := reflect.TypeOf(models.VoterParticipation{})
	for _, field := range forbiddenBallotFieldsInParticipation {
		if _, found := participationType.FieldByName(field); found {
			t.Fatalf("CRITICAL PRIVACY VIOLATION: VoterParticipation contains forbidden ballot choice field: %s", field)
		}
	}
}

// TestElectionLifecycle verifies schedule checks and state transitions.
func TestElectionLifecycle(t *testing.T) {
	now := time.Now().UTC()

	election := models.Election{
		Title:    "Guild Presidential Election",
		Status:   models.ElectionStatusOpen,
		StartsAt: now.Add(-1 * time.Hour),
		EndsAt:   now.Add(1 * time.Hour),
	}

	if !election.IsOpen(now) {
		t.Errorf("expected election to be open at %v, got false", now)
	}
	if !election.CanAcceptVotes(now) {
		t.Errorf("expected election to accept votes at %v, got false", now)
	}

	// Election before start time
	if election.IsOpen(now.Add(-2 * time.Hour)) {
		t.Errorf("election should NOT be open before StartsAt")
	}

	// Election after end time
	if election.IsOpen(now.Add(2 * time.Hour)) {
		t.Errorf("election should NOT be open after EndsAt")
	}

	// Election in draft status
	election.Status = models.ElectionStatusDraft
	if election.IsOpen(now) {
		t.Errorf("election in DRAFT status must NOT be open")
	}
}

// TestDuplicateVotingConstraint verifies database-level enforcement of one vote per voter per election.
func TestDuplicateVotingConstraint(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:memdb_voting_test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	if err := db.AutoMigrate(models.Models()...); err != nil {
		t.Fatalf("failed to automigrate models: %v", err)
	}

	electionID := "elec-test-001"
	userID := "user-test-voter-001"

	// 1. Record first participation
	p1 := models.VoterParticipation{
		ElectionID: electionID,
		UserID:     userID,
	}
	if err := db.Create(&p1).Error; err != nil {
		t.Fatalf("failed to create initial voter participation: %v", err)
	}

	if p1.ConfirmationToken == "" {
		t.Errorf("expected confirmation token to be generated automatically")
	}

	// 2. Attempt duplicate participation for same election and user
	p2 := models.VoterParticipation{
		ElectionID: electionID,
		UserID:     userID,
	}
	err = db.Create(&p2).Error
	if err == nil {
		t.Fatalf("CRITICAL SECURITY FAILURE: duplicate voter participation was accepted; database unique constraint failed!")
	}

	// 3. Different election with same user MUST succeed
	p3 := models.VoterParticipation{
		ElectionID: "elec-test-002",
		UserID:     userID,
	}
	if err := db.Create(&p3).Error; err != nil {
		t.Fatalf("expected voter to participate in different election, got error: %v", err)
	}
}
