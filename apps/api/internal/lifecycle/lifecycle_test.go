package lifecycle_test

import (
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"campvote/apps/api/internal/lifecycle"
	"campvote/apps/api/internal/models"
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

var statuses = []string{
	models.ElectionStatusDraft,
	models.ElectionStatusRegistration,
	models.ElectionStatusPending,
	models.ElectionStatusScheduled,
	models.ElectionStatusOpen,
	models.ElectionStatusClosed,
	models.ElectionStatusReview,
	models.ElectionStatusFinalized,
	models.ElectionStatusArchived,
	models.ElectionStatusCancelled,
}

// TestTransitionMatrix checks every from/from pair. The expected table is
// deliberately written out rather than derived from the implementation.
func TestTransitionMatrix(t *testing.T) {
	allowed := map[string]map[string]bool{
		models.ElectionStatusDraft: {
			models.ElectionStatusRegistration: true,
			models.ElectionStatusPending:      true,
			models.ElectionStatusScheduled:    true,
			models.ElectionStatusCancelled:    true,
		},
		models.ElectionStatusRegistration: {
			models.ElectionStatusPending:   true,
			models.ElectionStatusScheduled: true,
			models.ElectionStatusCancelled: true,
		},
		models.ElectionStatusPending: {
			models.ElectionStatusRegistration: true,
			models.ElectionStatusScheduled:    true,
			models.ElectionStatusCancelled:    true,
		},
		models.ElectionStatusScheduled: {
			models.ElectionStatusOpen:      true,
			models.ElectionStatusClosed:    true,
			models.ElectionStatusCancelled: true,
		},
		models.ElectionStatusOpen: {
			models.ElectionStatusClosed:    true,
			models.ElectionStatusCancelled: true,
		},
		models.ElectionStatusClosed: {
			models.ElectionStatusReview:    true,
			models.ElectionStatusFinalized: true,
			models.ElectionStatusArchived:  true,
		},
		models.ElectionStatusReview: {
			models.ElectionStatusFinalized: true,
			models.ElectionStatusArchived:  true,
		},
		models.ElectionStatusFinalized: {
			models.ElectionStatusArchived: true,
		},
		models.ElectionStatusArchived:  {},
		models.ElectionStatusCancelled: {},
	}

	for _, from := range statuses {
		for _, to := range statuses {
			want := allowed[from][to]
			if got := lifecycle.CanTransition(from, to); got != want {
				t.Errorf("CanTransition(%s, %s) = %v, want %v", from, to, got, want)
			}
		}
	}
}

func validElection(t *testing.T, db *gorm.DB, status string, start, end time.Time) *models.Election {
	t.Helper()
	e := &models.Election{
		Title: "Synthetic Guild Election", Category: models.ElectionCategoryGuild,
		OrganizingUnit: "MMU Electoral Commission", Status: status,
		StartsAt: start, EndsAt: end, ResultPolicy: models.ResultPolicyAfterClose,
		AllowedDomains: "@std.must.ac.ug",
	}
	if err := db.Create(e).Error; err != nil {
		t.Fatal(err)
	}
	if status != models.ElectionStatusDraft {
		addPositionAndCandidate(t, db, e.ID)
	}
	return e
}

func addPositionAndCandidate(t *testing.T, db *gorm.DB, electionID string) {
	t.Helper()
	position := &models.Position{
		ElectionID: electionID, Title: "President", QuestionType: models.QuestionTypeSingleChoice,
		MinSelections: 1, MaxSelections: 1, Seats: 1,
	}
	if err := db.Create(position).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.Candidate{
		ElectionID: electionID, PositionID: position.ID, FullName: "Synthetic Candidate",
		ApprovalStatus: models.CandidateStatusApproved,
	}).Error; err != nil {
		t.Fatal(err)
	}
}

// TestOpenAndCloseBoundaries moves server time around the window. Start is
// inclusive and end is exclusive; that choice is documented in
// docs/election-lifecycle.md.
func TestOpenAndCloseBoundaries(t *testing.T) {
	base := time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC)
	start := base.Add(time.Hour)
	end := start.Add(2 * time.Hour)

	cases := []struct {
		name   string
		at     time.Time
		status string
		open   bool
	}{
		{"just before start", start.Add(-time.Nanosecond), models.ElectionStatusScheduled, false},
		{"at start", start, models.ElectionStatusOpen, true},
		{"just before end", end.Add(-time.Nanosecond), models.ElectionStatusOpen, true},
		{"at end", end, models.ElectionStatusClosed, false},
		{"after end", end.Add(time.Hour), models.ElectionStatusClosed, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := testDB(t)
			e := validElection(t, db, models.ElectionStatusScheduled, start, end)
			if err := lifecycle.Ensure(db, e, tc.at); err != nil {
				t.Fatal(err)
			}
			if e.Status != tc.status {
				t.Errorf("status at %s = %s, want %s", tc.at, e.Status, tc.status)
			}
			if got := e.IsOpen(tc.at); got != tc.open {
				t.Errorf("IsOpen at %s = %v, want %v", tc.at, got, tc.open)
			}
		})
	}
}

// TestMissedSchedulerTickCannotExtendVoting starts from an OPEN election whose
// end has passed and never runs the scheduler. The lazy check must close it and
// IsOpen must already be false before that.
func TestMissedSchedulerTickCannotExtendVoting(t *testing.T) {
	db := testDB(t)
	now := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	e := validElection(t, db, models.ElectionStatusOpen, now.Add(-2*time.Hour), now.Add(-time.Minute))

	if e.IsOpen(now) {
		t.Fatal("an election past its end time must not accept votes")
	}
	if err := lifecycle.Ensure(db, e, now); err != nil {
		t.Fatal(err)
	}
	if e.Status != models.ElectionStatusClosed {
		t.Errorf("status = %s, want CLOSED after a missed tick", e.Status)
	}
	var audit models.ElectionAuditLog
	if err := db.Where("election_id = ? AND action = ?", e.ID, models.AuditActionElectionClosed).
		First(&audit).Error; err != nil {
		t.Fatal("the lazy close was not audited")
	}
	if audit.ActorID != lifecycle.SystemActor {
		t.Errorf("audit actor = %q, want %q", audit.ActorID, lifecycle.SystemActor)
	}
}

func TestEnsureDoesNotOpenInvalidConfiguration(t *testing.T) {
	db := testDB(t)
	now := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	e := validElection(t, db, models.ElectionStatusScheduled, now.Add(-time.Hour), now.Add(time.Hour))
	if err := db.Where("election_id = ?", e.ID).Delete(&models.Position{}).Error; err != nil {
		t.Fatal(err)
	}

	if err := lifecycle.Ensure(db, e, now); err != nil {
		t.Fatal(err)
	}
	if e.Status != models.ElectionStatusScheduled {
		t.Errorf("status = %s, want SCHEDULED when configuration is incomplete", e.Status)
	}
	var opened int64
	db.Model(&models.ElectionAuditLog{}).
		Where("election_id = ? AND action = ?", e.ID, models.AuditActionElectionOpened).Count(&opened)
	if opened != 0 {
		t.Error("an invalid election must not be recorded as opened")
	}
}

func TestSchedulerRestartReconcilesMissedTransitions(t *testing.T) {
	db := testDB(t)
	now := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	due := validElection(t, db, models.ElectionStatusScheduled, now.Add(-time.Minute), now.Add(time.Hour))
	ended := validElection(t, db, models.ElectionStatusOpen, now.Add(-3*time.Hour), now.Add(-time.Hour))

	changed, err := lifecycle.SyncAll(db, now)
	if err != nil {
		t.Fatal(err)
	}
	if changed != 2 {
		t.Errorf("changed = %d, want 2", changed)
	}
	var dueAfter, endedAfter models.Election
	db.Where("id = ?", due.ID).First(&dueAfter)
	db.Where("id = ?", ended.ID).First(&endedAfter)
	if dueAfter.Status != models.ElectionStatusOpen {
		t.Errorf("due election = %s, want OPEN", dueAfter.Status)
	}
	if endedAfter.Status != models.ElectionStatusClosed {
		t.Errorf("ended election = %s, want CLOSED", endedAfter.Status)
	}
}

func TestValidateConfiguration(t *testing.T) {
	now := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name   string
		mutate func(*models.Election)
	}{
		{"start not before end", func(e *models.Election) {
			e.StartsAt = now.Add(2 * time.Hour)
			e.EndsAt = now.Add(time.Hour)
		}},
		{"shorter than the minimum", func(e *models.Election) {
			e.StartsAt = now
			e.EndsAt = now.Add(30 * time.Minute)
		}},
		{"already ended when scheduling", func(e *models.Election) {
			e.StartsAt = now.Add(-3 * time.Hour)
			e.EndsAt = now.Add(-2 * time.Hour)
		}},
		{"unknown category", func(e *models.Election) { e.Category = "SPACE" }},
		{"unknown result policy", func(e *models.Election) { e.ResultPolicy = "SOMETIME" }},
		{"bad domain", func(e *models.Election) { e.AllowedDomains = "@std.must.ac.ug,not a domain" }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db := testDB(t)
			e := validElection(t, db, models.ElectionStatusDraft, now.Add(time.Hour), now.Add(2*time.Hour))
			addPositionAndCandidate(t, db, e.ID)
			tc.mutate(e)
			if err := lifecycle.ValidateConfiguration(db, e, now, true); err == nil {
				t.Error("expected validation to fail")
			}
		})
	}
}

func TestValidateConfigurationRequiresPositionsAndCandidates(t *testing.T) {
	db := testDB(t)
	now := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	e := validElection(t, db, models.ElectionStatusDraft, now.Add(time.Hour), now.Add(2*time.Hour))

	if err := lifecycle.ValidateConfiguration(db, e, now, false); err == nil {
		t.Error("an election with no positions must not be ready")
	}
	addPositionAndCandidate(t, db, e.ID)
	if err := lifecycle.ValidateConfiguration(db, e, now, false); err != nil {
		t.Fatalf("valid election refused: %v", err)
	}

	var candidate models.Candidate
	db.Where("election_id = ?", e.ID).First(&candidate)
	db.Model(&candidate).Update("approval_status", models.CandidateStatusPending)
	if err := lifecycle.ValidateConfiguration(db, e, now, false); err == nil {
		t.Error("a position without an approved candidate must not be ready")
	}
}

func TestExceptionalCloseAndCancelRequireAReason(t *testing.T) {
	db := testDB(t)
	now := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	open := validElection(t, db, models.ElectionStatusOpen, now.Add(-time.Hour), now.Add(time.Hour))

	if _, err := lifecycle.Apply(db, open, models.ElectionStatusClosed, "actor", "", "", now); err == nil {
		t.Error("early close without a reason must be refused")
	}
	if _, err := lifecycle.Apply(db, open, models.ElectionStatusClosed, "actor", "security concern", "", now); err != nil {
		t.Fatalf("early close with a reason: %v", err)
	}

	draft := validElection(t, db, models.ElectionStatusDraft, now.Add(time.Hour), now.Add(2*time.Hour))
	if _, err := lifecycle.Apply(db, draft, models.ElectionStatusCancelled, "actor", "", "", now); err == nil {
		t.Error("cancellation without a reason must be refused")
	}
	if _, err := lifecycle.Apply(db, draft, models.ElectionStatusCancelled, "actor", "no candidates", "", now); err != nil {
		t.Fatalf("cancellation with a reason: %v", err)
	}
}

func TestIllegalTransitionHasAClearError(t *testing.T) {
	db := testDB(t)
	now := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	draft := validElection(t, db, models.ElectionStatusDraft, now.Add(time.Hour), now.Add(2*time.Hour))
	_, err := lifecycle.Apply(db, draft, models.ElectionStatusOpen, "actor", "", "", now)
	if err == nil {
		t.Fatal("DRAFT -> OPEN must be refused")
	}
	if !strings.Contains(err.Error(), "illegal election transition") || !strings.Contains(err.Error(), "DRAFT") {
		t.Errorf("error %q does not name the refused transition", err)
	}
}
