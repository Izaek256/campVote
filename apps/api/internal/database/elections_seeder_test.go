package database_test

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"campvote/apps/api/internal/database"
	"campvote/apps/api/internal/models"
)

func TestSeedElectionsSkipsOutsideDevelopment(t *testing.T) {
	t.Setenv("APP_ENV", "production")

	db, err := gorm.Open(sqlite.Open("file:memdb_elections_seeder_prod_test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open in-memory database: %v", err)
	}
	if err := db.AutoMigrate(models.Models()...); err != nil {
		t.Fatalf("failed to run automigration: %v", err)
	}

	if err := database.SeedElections(db); err != nil {
		t.Fatalf("SeedElections should skip cleanly outside development: %v", err)
	}

	var elections, users int64
	db.Model(&models.Election{}).Count(&elections)
	db.Model(&models.User{}).Count(&users)
	if elections != 0 || users != 0 {
		t.Errorf("expected no demo data outside development, got %d elections and %d users", elections, users)
	}
}

func TestSeedElectionsSeedsNoAdministrator(t *testing.T) {
	t.Setenv("APP_ENV", "development")

	db, err := gorm.Open(sqlite.Open("file:memdb_elections_seeder_admin_test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open in-memory database: %v", err)
	}
	if err := db.AutoMigrate(models.Models()...); err != nil {
		t.Fatalf("failed to run automigration: %v", err)
	}
	if err := database.SeedElections(db); err != nil {
		t.Fatalf("SeedElections failed: %v", err)
	}

	var admins int64
	db.Model(&models.User{}).Where("role = ?", models.RoleAdmin).Count(&admins)
	if admins != 0 {
		t.Errorf("demo seeding must not create a known administrator, found %d", admins)
	}
}

func TestSeedElections(t *testing.T) {
	t.Setenv("APP_ENV", "development")

	db, err := gorm.Open(sqlite.Open("file:memdb_elections_seeder_test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open in-memory database: %v", err)
	}

	// 1. Run migrations
	if err := db.AutoMigrate(models.Models()...); err != nil {
		t.Fatalf("failed to run automigration: %v", err)
	}

	// 2. Run election seeder
	if err := database.SeedElections(db); err != nil {
		t.Fatalf("SeedElections failed: %v", err)
	}

	// 3. Verify Guild election exists
	var elections []models.Election
	if err := db.Find(&elections).Error; err != nil {
		t.Fatalf("failed to query elections: %v", err)
	}

	if len(elections) < 2 {
		t.Fatalf("expected at least 2 seeded elections, found %d", len(elections))
	}

	var guildElection *models.Election
	for i := range elections {
		if elections[i].Category == models.ElectionCategoryGuild {
			guildElection = &elections[i]
			break
		}
	}
	if guildElection == nil {
		t.Fatalf("expected to find a GUILD election")
	}
	if guildElection.Status != models.ElectionStatusOpen {
		t.Errorf("expected guild election status to be OPEN, got %s", guildElection.Status)
	}

	// 4. Verify positions for Guild election
	var positions []models.Position
	if err := db.Where("election_id = ?", guildElection.ID).Find(&positions).Error; err != nil {
		t.Fatalf("failed to query positions: %v", err)
	}
	if len(positions) != 3 {
		t.Errorf("expected 3 positions for guild election, got %d", len(positions))
	}

	// 5. Verify candidates
	var candidates []models.Candidate
	if err := db.Where("election_id = ?", guildElection.ID).Find(&candidates).Error; err != nil {
		t.Fatalf("failed to query candidates: %v", err)
	}
	if len(candidates) < 5 {
		t.Errorf("expected at least 5 candidates, got %d", len(candidates))
	}

	// 6. Verify roster entries
	var roster []models.ElectionRoster
	if err := db.Where("election_id = ?", guildElection.ID).Find(&roster).Error; err != nil {
		t.Fatalf("failed to query roster: %v", err)
	}
	if len(roster) != 2 {
		t.Errorf("expected 2 roster entries, got %d", len(roster))
	}

	// 7. Verify Idempotency - second run must not duplicate
	if err := database.SeedElections(db); err != nil {
		t.Fatalf("second run of SeedElections should be idempotent, failed with: %v", err)
	}
	var countAfter int64
	db.Model(&models.Election{}).Count(&countAfter)
	if int(countAfter) != len(elections) {
		t.Errorf("expected count to remain %d, got %d", len(elections), countAfter)
	}
}
