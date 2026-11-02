package results_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"campvote/apps/api/internal/models"
	"campvote/apps/api/internal/results"
	"campvote/apps/api/internal/tally"
)

func testDB(t *testing.T) *gorm.DB {
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
	return db
}

// closedElection builds a closed election with two rule-eligible verified
// accounts, one position, two candidates and one ballot for the first.
func closedElection(t *testing.T, db *gorm.DB) models.Election {
	t.Helper()
	now := time.Now().UTC()
	election := models.Election{
		ID: "e-auto", Title: "Auto tally", Status: models.ElectionStatusClosed,
		ResultPolicy: models.ResultPolicyAfterClose,
		StartsAt:     now.Add(-2 * time.Hour), EndsAt: now.Add(-time.Hour),
		AllowedDomain: "std.mast.ac.ug", AllowedYears: "2020",
	}
	if err := db.Create(&election).Error; err != nil {
		t.Fatal(err)
	}
	for _, email := range []string{"2020bse001@std.mast.ac.ug", "2020bse002@std.mast.ac.ug"} {
		verified := now
		user := models.User{
			FirstName: "Test", LastName: "Student", Email: email,
			Password: "x", Role: models.RoleStudentVoter, Active: true,
			EmailVerifiedAt: &verified,
		}
		if err := db.Create(&user).Error; err != nil {
			t.Fatal(err)
		}
	}
	position := models.Position{
		ID: "e-auto-pos", ElectionID: models.Ptr(election.ID), Title: "President",
		QuestionType: models.QuestionTypeSingleChoice, Seats: 1,
	}
	if err := db.Create(&position).Error; err != nil {
		t.Fatal(err)
	}
	candidate := models.Candidate{
		ID: "e-auto-cand", ElectionID: election.ID, PositionID: position.ID,
		FullName: "Ada",
	}
	if err := db.Create(&candidate).Error; err != nil {
		t.Fatal(err)
	}
	ballot := models.AnonymousBallot{ElectionID: election.ID}
	if err := db.Create(&ballot).Error; err != nil {
		t.Fatal(err)
	}
	choice := candidate.ID
	if err := db.Create(&models.BallotSelection{
		BallotID: ballot.ID, PositionID: position.ID, CandidateID: &choice, Choice: choice,
	}).Error; err != nil {
		t.Fatal(err)
	}
	return election
}

func TestEnsureOnCloseRecordsOneProvisionalSnapshot(t *testing.T) {
	db := testDB(t)
	election := closedElection(t, db)

	wrote, err := results.EnsureOnClose(db, &election, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if !wrote {
		t.Fatal("first close tally reported no snapshot")
	}
	// A repeated call, as the close hook and a Read retry can both make, does
	// not add a second version.
	wrote, err = results.EnsureOnClose(db, &election, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if wrote {
		t.Fatal("second close tally wrote another snapshot")
	}

	var snapshots []models.ResultSnapshot
	if err := db.Where("election_id = ?", election.ID).Find(&snapshots).Error; err != nil {
		t.Fatal(err)
	}
	if len(snapshots) != 1 {
		t.Fatalf("snapshots = %d, want 1", len(snapshots))
	}
	snapshot := snapshots[0]
	if snapshot.Version != 1 || snapshot.Status != models.ResultStatusProvisional {
		t.Fatalf("snapshot = version %d status %q, want 1/PROVISIONAL", snapshot.Version, snapshot.Status)
	}
	if snapshot.ComputedByID != results.SystemActor {
		t.Fatalf("computed by %q, want %q", snapshot.ComputedByID, results.SystemActor)
	}

	var content tally.Result
	if err := json.Unmarshal([]byte(snapshot.ContentJSON), &content); err != nil {
		t.Fatal(err)
	}
	if content.Turnout.EligibleVoters != 2 || content.Turnout.BallotsCast != 1 {
		t.Fatalf("turnout = %d/%d, want 1 cast of 2 eligible",
			content.Turnout.BallotsCast, content.Turnout.EligibleVoters)
	}
	if len(content.Positions) != 1 || content.Positions[0].Rows[0].Votes != 1 {
		t.Fatalf("tally did not count the one ballot: %+v", content.Positions)
	}

	var audits int64
	db.Model(&models.ElectionAuditLog{}).
		Where("election_id = ? AND action = ?", election.ID, models.AuditActionTallyRun).
		Count(&audits)
	if audits != 1 {
		t.Fatalf("tally audit rows = %d, want 1", audits)
	}
}

func TestEnsureOnCloseIgnoresAnOpenElection(t *testing.T) {
	db := testDB(t)
	election := closedElection(t, db)
	election.Status = models.ElectionStatusOpen

	wrote, err := results.EnsureOnClose(db, &election, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if wrote {
		t.Fatal("an open election was tallied on close")
	}
	var count int64
	db.Model(&models.ResultSnapshot{}).Where("election_id = ?", election.ID).Count(&count)
	if count != 0 {
		t.Fatalf("snapshots = %d, want none while open", count)
	}
}
