package migrations_test

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"campvote/apps/api/internal/migrations"
	"campvote/apps/api/internal/models"
)

// The old index was (election_id, idempotency_key), so one voter's key could
// block another voter's submission in the same election. The migration must
// widen the scope to the voter without touching the one-ballot-per-voter index.
func TestScopeParticipationIdempotencyKeyAllowsTheSameKeyForDifferentVoters(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:memdb_scope_idempotency_test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open in-memory database: %v", err)
	}
	if err := db.AutoMigrate(&models.VoterParticipation{}); err != nil {
		t.Fatal(err)
	}

	// Put the database back into the pre-migration shape.
	for _, statement := range []string{
		"DROP INDEX IF EXISTS idx_participation_idempotency",
		"CREATE UNIQUE INDEX idx_participation_idempotency ON voter_participations (election_id, idempotency_key)",
	} {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatal(err)
		}
	}

	first := models.VoterParticipation{ElectionID: "e-1", UserID: "user-1", IdempotencyKey: "shared"}
	if err := db.Create(&first).Error; err != nil {
		t.Fatalf("first participation was refused: %v", err)
	}
	second := models.VoterParticipation{ElectionID: "e-1", UserID: "user-2", IdempotencyKey: "shared"}
	if err := db.Create(&second).Error; err == nil {
		t.Fatal("old index unexpectedly allowed the same key for a second voter")
	}

	if err := migrations.ScopeParticipationIdempotencyKey(db); err != nil {
		t.Fatalf("migration failed: %v", err)
	}
	// Idempotent: a restart runs it again.
	if err := migrations.ScopeParticipationIdempotencyKey(db); err != nil {
		t.Fatalf("second migration run failed: %v", err)
	}

	if err := db.Create(&second).Error; err != nil {
		t.Fatalf("second voter was refused after the migration: %v", err)
	}

	// The one-ballot-per-voter rule is untouched.
	again := models.VoterParticipation{ElectionID: "e-1", UserID: "user-1", IdempotencyKey: "another"}
	if err := db.Create(&again).Error; err == nil {
		t.Fatal("second participation for one voter was accepted")
	}
}
