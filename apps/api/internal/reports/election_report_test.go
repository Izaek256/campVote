package reports

import (
	"fmt"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"campvote/apps/api/internal/models"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	if err := db.AutoMigrate(models.Models()...); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

func TestElectionSummary(t *testing.T) {
	db := setupTestDB(t)

	// Create a test election
	election := &models.Election{
		Title:          "Test Guild Election",
		Description:    "Test election for unit testing",
		Category:       models.ElectionCategoryGuild,
		OrganizingUnit: "Test Commission",
		Status:         models.ElectionStatusDraft,
		StartsAt:       time.Now().UTC().Add(24 * time.Hour),
		EndsAt:         time.Now().UTC().Add(48 * time.Hour),
		ResultPolicy:   models.ResultPolicyAfterClose,
		AllowedDomains: "@std.must.ac.ug",
	}
	if err := db.Create(election).Error; err != nil {
		t.Fatalf("failed to create election: %v", err)
	}

	// Create a position
	position := &models.Position{
		ElectionID: election.ID,
		Title:      "President",
		Description: "Guild President",
		MaxSelections: 1,
	}
	if err := db.Create(position).Error; err != nil {
		t.Fatalf("failed to create position: %v", err)
	}

	// Create a candidate
	candidate := &models.Candidate{
		ElectionID:     election.ID,
		PositionID:     position.ID,
		FullName:       "Test Candidate",
		ApprovalStatus: models.CandidateStatusApproved,
	}
	if err := db.Create(candidate).Error; err != nil {
		t.Fatalf("failed to create candidate: %v", err)
	}

	// Create roster entries
	for i := 0; i < 100; i++ {
		roster := &models.ElectionRoster{
			ElectionID:    election.ID,
			Email:         fmt.Sprintf("student%d@example.com", i),
			StudentNumber: fmt.Sprintf("22/U/100%d/PS", i),
			Status:        models.RosterStatusEligible,
		}
		if err := db.Create(roster).Error; err != nil {
			t.Fatalf("failed to create roster entry: %v", err)
		}
	}

	// Create some participations
	for i := 0; i < 50; i++ {
		participation := &models.VoterParticipation{
			ElectionID: election.ID,
			UserID:     "user-" + string(rune(i)),
		}
		if err := db.Create(participation).Error; err != nil {
			t.Fatalf("failed to create participation: %v", err)
		}
	}

	// Generate the report
	pdfBytes, err := ElectionSummary(db, election.ID)
	if err != nil {
		t.Fatalf("ElectionSummary failed: %v", err)
	}

	if len(pdfBytes) == 0 {
		t.Error("ElectionSummary returned empty PDF")
	}

	// Verify PDF starts with PDF signature
	if len(pdfBytes) < 5 || string(pdfBytes[:5]) != "%PDF-" {
		t.Error("ElectionSummary did not return a valid PDF")
	}
}

func TestResultsVerification(t *testing.T) {
	db := setupTestDB(t)

	// Create a closed election
	now := time.Now().UTC()
	election := &models.Election{
		Title:          "Test Guild Election",
		Description:    "Test election for unit testing",
		Category:       models.ElectionCategoryGuild,
		OrganizingUnit: "Test Commission",
		Status:         models.ElectionStatusClosed,
		StartsAt:       now.Add(-48 * time.Hour),
		EndsAt:         now.Add(-24 * time.Hour),
		ResultPolicy:   models.ResultPolicyAfterClose,
		AllowedDomains: "@std.must.ac.ug",
		EligibilityFrozenAt: func() *time.Time {
			t := now.Add(-45 * time.Hour)
			return &t
		}(),
		EligibilityFrozenCount: 100,
	}
	if err := db.Create(election).Error; err != nil {
		t.Fatalf("failed to create election: %v", err)
	}

	// Create a position
	position := &models.Position{
		ElectionID: election.ID,
		Title:      "President",
		Description: "Guild President",
		MaxSelections: 1,
	}
	if err := db.Create(position).Error; err != nil {
		t.Fatalf("failed to create position: %v", err)
	}

	// Create a candidate
	candidate := &models.Candidate{
		ElectionID:     election.ID,
		PositionID:     position.ID,
		FullName:       "Test Candidate",
		ApprovalStatus: models.CandidateStatusApproved,
	}
	if err := db.Create(candidate).Error; err != nil {
		t.Fatalf("failed to create candidate: %v", err)
	}

	// Create a tally snapshot
	tally := &models.ElectionTally{
		ElectionID:  election.ID,
		PositionID:  position.ID,
		CandidateID: &candidate.ID,
		Choice:      candidate.FullName,
		Votes:       42,
	}
	if err := db.Create(tally).Error; err != nil {
		t.Fatalf("failed to create tally: %v", err)
	}

	// Create roster entries
	for i := 0; i < 100; i++ {
		roster := &models.ElectionRoster{
			ElectionID:    election.ID,
			Email:         fmt.Sprintf("student%d@example.com", i),
			StudentNumber: fmt.Sprintf("22/U/100%d/PS", i),
			Status:        models.RosterStatusEligible,
		}
		if err := db.Create(roster).Error; err != nil {
			t.Fatalf("failed to create roster entry: %v", err)
		}
	}

	// Create participations and ballots
	for i := 0; i < 42; i++ {
		participation := &models.VoterParticipation{
			ElectionID: election.ID,
			UserID:     "user-" + string(rune(i)),
		}
		if err := db.Create(participation).Error; err != nil {
			t.Fatalf("failed to create participation: %v", err)
		}

		ballot := &models.AnonymousBallot{
			ElectionID: election.ID,
			Status:     models.BallotStatusValid,
			CastAt:     time.Now().UTC(),
		}
		if err := db.Create(ballot).Error; err != nil {
			t.Fatalf("failed to create ballot: %v", err)
		}
	}

	// Generate the report
	pdfBytes, err := ResultsVerification(db, election.ID)
	if err != nil {
		t.Fatalf("ResultsVerification failed: %v", err)
	}

	if len(pdfBytes) == 0 {
		t.Error("ResultsVerification returned empty PDF")
	}

	// Verify PDF starts with PDF signature
	if len(pdfBytes) < 5 || string(pdfBytes[:5]) != "%PDF-" {
		t.Error("ResultsVerification did not return a valid PDF")
	}
}
