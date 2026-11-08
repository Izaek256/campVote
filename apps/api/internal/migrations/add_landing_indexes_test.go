package migrations_test

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"campvote/apps/api/internal/migrations"
	"campvote/apps/api/internal/models"
)

func TestAddLandingIndexesOnCleanDatabase(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:memdb_landing_index_test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open in-memory database: %v", err)
	}
	if err := db.AutoMigrate(&models.Election{}); err != nil {
		t.Fatalf("failed to create elections: %v", err)
	}

	if err := migrations.AddLandingIndexes(db); err != nil {
		t.Fatalf("AddLandingIndexes failed: %v", err)
	}
	// The migration is idempotent: running it again on the same schema is a
	// no-op, which is what a restart needs.
	if err := migrations.AddLandingIndexes(db); err != nil {
		t.Fatalf("second AddLandingIndexes run failed: %v", err)
	}

	var count int64
	if err := db.Raw(
		"SELECT COUNT(*) FROM sqlite_master WHERE type = 'index' AND name = 'idx_elections_public_upcoming'",
	).Scan(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("idx_elections_public_upcoming count = %d, want 1", count)
	}
}
