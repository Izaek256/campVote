package database_test

import (
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"campvote/apps/api/internal/database"
	"campvote/apps/api/internal/landing"
	"campvote/apps/api/internal/models"
)

// TestSeedLandingOnCleanDatabase is the migration-and-seed check: a database
// with no tables becomes a usable CMS with the prototype's default copy, and a
// second run does not duplicate or overwrite it.
func TestSeedLandingOnCleanDatabase(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:memdb_landing_seeder_test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open in-memory database: %v", err)
	}
	if err := db.AutoMigrate(models.Models()...); err != nil {
		t.Fatalf("failed to run automigration: %v", err)
	}

	if err := database.SeedLanding(db); err != nil {
		t.Fatalf("SeedLanding failed: %v", err)
	}

	var settings models.LandingSettings
	if err := db.First(&settings, "id = ?", models.LandingSettingsID).Error; err != nil {
		t.Fatalf("landing settings row missing: %v", err)
	}
	if settings.PublishedJSON != "" {
		t.Error("seeding must leave publishing to an administrator")
	}
	if settings.HeroHeadline != landing.Default().Hero.Headline {
		t.Errorf("hero headline = %q, want the built-in default", settings.HeroHeadline)
	}

	var features, faqs int64
	db.Model(&models.LandingFeature{}).Count(&features)
	db.Model(&models.LandingFAQ{}).Count(&faqs)
	if int(features) != len(landing.Default().Features) {
		t.Errorf("features = %d, want %d", features, len(landing.Default().Features))
	}
	if int(faqs) != len(landing.Default().FAQs) {
		t.Errorf("faqs = %d, want %d", faqs, len(landing.Default().FAQs))
	}

	if err := database.SeedLanding(db); err != nil {
		t.Fatalf("second SeedLanding run should be idempotent: %v", err)
	}
	var featuresAfter, faqsAfter, settingsAfter int64
	db.Model(&models.LandingFeature{}).Count(&featuresAfter)
	db.Model(&models.LandingFAQ{}).Count(&faqsAfter)
	db.Model(&models.LandingSettings{}).Count(&settingsAfter)
	if featuresAfter != features || faqsAfter != faqs || settingsAfter != 1 {
		t.Errorf("second run duplicated rows: settings=%d features=%d faqs=%d",
			settingsAfter, featuresAfter, faqsAfter)
	}
}
