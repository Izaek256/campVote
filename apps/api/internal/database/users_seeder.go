package database

import (
	"log"
	"os"

	"gorm.io/gorm"

	"campvote/apps/api/internal/models"
)

// SeedUsers creates the synthetic demo accounts for local development.
//
// It does not create an administrator. The first administrator comes from
// cmd/bootstrap, with credentials the operator supplies once, because a
// seeded admin is a known account and a known password in every deployment
// that runs the seeder.
func SeedUsers(db *gorm.DB) error {
	return seedDemoUsers(db)
}

// seedDemoUsers creates sample non-administrator accounts for development.
// All demo users share the password "admin123" — the same as before — so the
// Concepts course / first-look lesson works without remembering a second
// password.
func seedDemoUsers(db *gorm.DB) error {
	// Demo users are development fixtures sharing a weak password, so they are
	// seeded with APP_ENV=development and nowhere else.
	if !seedDevDefaults() {
		log.Printf("Skipping demo users: APP_ENV is %q, not development", os.Getenv("APP_ENV"))
		return nil
	}
	users := []models.User{
		{FirstName: "Jane", LastName: "Cooper", Email: "jane@example.com", Password: "admin123", Role: "EDITOR", Active: true},
		{FirstName: "Robert", LastName: "Fox", Email: "robert@example.com", Password: "admin123", Role: "USER", Active: true},
		{FirstName: "Emily", LastName: "Davis", Email: "emily@example.com", Password: "admin123", Role: "USER", Active: true},
		{FirstName: "Michael", LastName: "Chen", Email: "michael@example.com", Password: "admin123", Role: "USER", Active: false},
	}

	for _, u := range users {
		var count int64
		db.Model(&models.User{}).Where("email = ?", u.Email).Count(&count)
		if count > 0 {
			continue
		}

		if err := db.Create(&u).Error; err != nil {
			log.Printf("Warning: failed to create user %s: %v", u.Email, err)
			continue
		}
		log.Printf("Created demo user: %s / admin123 (development only)", u.Email)
	}

	return nil
}

// seedDevDefaults reports whether the weak development accounts may be seeded:
// only when APP_ENV says development. Unset, staging and prod are not.
func seedDevDefaults() bool {
	return os.Getenv("APP_ENV") == "development"
}
