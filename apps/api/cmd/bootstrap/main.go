package main

import (
	"context"
	"errors"
	"log"
	"os"

	"campvote/apps/api/internal/config"
	"campvote/apps/api/internal/database"
	"campvote/apps/api/internal/models"
)

// bootstrap-admin creates the first administrator of a fresh deployment.
//
// It is one-time and has no defaults: set BOOTSTRAP_ADMIN_EMAIL and
// BOOTSTRAP_ADMIN_PASSWORD for this run, then remove them. A second run after
// an administrator exists is refused.
func main() {
	email := os.Getenv("BOOTSTRAP_ADMIN_EMAIL")
	plaintext := os.Getenv("BOOTSTRAP_ADMIN_PASSWORD")
	if email == "" || plaintext == "" {
		log.Fatal("Set BOOTSTRAP_ADMIN_EMAIL and BOOTSTRAP_ADMIN_PASSWORD to create the first administrator.")
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	if !db.Migrator().HasTable(&models.User{}) {
		log.Fatal("The users table does not exist. Run the migrations first: go run ./cmd/migrate")
	}

	admin, err := database.CreateFirstAdmin(
		context.Background(), db, email, plaintext,
		os.Getenv("BOOTSTRAP_ADMIN_FIRST_NAME"), os.Getenv("BOOTSTRAP_ADMIN_LAST_NAME"),
	)
	if err != nil {
		if errors.Is(err, database.ErrAdministratorExists) {
			log.Fatalf("%v. Use the existing administrator, or reset its password with a one-time link.", err)
		}
		log.Fatalf("Bootstrap failed: %v", err)
	}

	log.Printf("Created the first administrator: %s", admin.Email)
	log.Println("Remove BOOTSTRAP_ADMIN_EMAIL and BOOTSTRAP_ADMIN_PASSWORD from this environment now.")
}
