package database

import (
	"fmt"
	"log"
	"os"
	"time"

	"campvote/apps/api/internal/models"

	"gorm.io/gorm"
)

// SeedElections is DISABLED for production use.
// Set SEED_DEMO_DATA=true in .env (and APP_ENV=development) to activate.
func SeedElections(db *gorm.DB) error {
	// Demo data gate 1: explicit opt-in
	if os.Getenv("SEED_DEMO_DATA") != "true" {
		log.Println("Skipping demo elections: set SEED_DEMO_DATA=true to enable (development only).")
		return nil
	}

	// Demo data gate 2: must not be production
	if !seedDevDefaults() {
		log.Printf("Skipping demo elections: APP_ENV is %q, not development", os.Getenv("APP_ENV"))
		return nil
	}

	// Idempotent: skip if elections already exist
	var count int64
	db.Model(&models.Election{}).Count(&count)
	if count > 0 {
		log.Println("Elections already seeded, skipping...")
		return nil
	}

	log.Println("Seeding campus demo elections, positions, and candidates...")

	now := time.Now().UTC()

	// ── Demo voter accounts ────────────────────────────────────────────────────
	// Emails use the registration-number format: YYYY<course><serial>@<domain>
	// Domain: std.mast.ac.ug (as per the prompt). No admin is seeded here;
	// the first admin comes from cmd/bootstrap.
	demoAccounts := []models.User{
		{
			FirstName: "Ivan",
			LastName:  "Mugisha",
			Email:     "2022bse001@std.mast.ac.ug",
			Password:  "Demo1234!",
			Role:      models.RoleUser,
			Active:    true,
		},
		{
			FirstName: "Doreen",
			LastName:  "Kemigisa",
			Email:     "2022bba002@std.mast.ac.ug",
			Password:  "Demo1234!",
			Role:      models.RoleUser,
			Active:    true,
		},
		{
			FirstName: "Faith",
			LastName:  "Tusiime",
			Email:     "2023bed003@std.mast.ac.ug",
			Password:  "Demo1234!",
			Role:      models.RoleUser,
			Active:    true,
		},
		{
			FirstName: "Emmanuel",
			LastName:  "Katusabe",
			Email:     "2023bit004@std.mast.ac.ug",
			Password:  "Demo1234!",
			Role:      models.RoleUser,
			Active:    true,
		},
	}

	for i := range demoAccounts {
		acc := &demoAccounts[i]
		var userCount int64
		db.Model(&models.User{}).Where("email = ?", acc.Email).Count(&userCount)
		if userCount == 0 {
			// Mark as verified so they can vote immediately in the demo.
			acc.EmailVerifiedAt = &now
			if err := db.Create(acc).Error; err != nil {
				log.Printf("Warning: failed to seed demo account %s: %v", acc.Email, err)
			} else {
				log.Printf("Created demo voter account: %s", acc.Email)
			}
		}
	}

	// ── Guild Election (OPEN) ─────────────────────────────────────────────────
	guildElection := models.Election{
		Title:          "Guild Representative Council Elections 2026/2027",
		Description:    "Annual democratic election of the Guild President, Vice Guild President, and student referendum.",
		Category:       models.ElectionCategoryGuild,
		OrganizingUnit: "Campus Electoral Commission",
		Status:         models.ElectionStatusOpen,
		StartsAt:       now.Add(-24 * time.Hour),
		EndsAt:         now.Add(48 * time.Hour),
		ResultPolicy:   models.ResultPolicyAfterClose,
		AllowedDomain:  "std.mast.ac.ug",
		AllowedYears:   "", // any year
		AllowedCourses: "", // any course
	}
	if err := db.Create(&guildElection).Error; err != nil {
		return fmt.Errorf("creating guild election: %w", err)
	}

	// ── Positions for Guild Election ──────────────────────────────────────────
	posPresident := models.Position{
		ElectionID:    models.Ptr(guildElection.ID),
		Title:         "Guild President",
		Description:   "Chief executive officer of the Guild Government.",
		QuestionType:  models.QuestionTypeSingleChoice,
		MinSelections: 1,
		MaxSelections: 1,
		Seats:         1,
		AllowBlank:    false,
		DisplayOrder:  1,
	}
	if err := db.Create(&posPresident).Error; err != nil {
		return fmt.Errorf("creating guild president position: %w", err)
	}

	posVicePresident := models.Position{
		ElectionID:    models.Ptr(guildElection.ID),
		Title:         "Vice Guild President",
		Description:   "Principal deputy to the Guild President.",
		QuestionType:  models.QuestionTypeSingleChoice,
		MinSelections: 1,
		MaxSelections: 1,
		Seats:         1,
		AllowBlank:    false,
		DisplayOrder:  2,
	}
	if err := db.Create(&posVicePresident).Error; err != nil {
		return fmt.Errorf("creating vice guild president position: %w", err)
	}

	posReferendum := models.Position{
		ElectionID:    models.Ptr(guildElection.ID),
		Title:         "Referendum: Guild Constitution Article 42 Amendment",
		Description:   "Do you approve extending the electoral preparation window from 14 to 21 days?",
		QuestionType:  models.QuestionTypeReferendum,
		MinSelections: 1,
		MaxSelections: 1,
		Seats:         1,
		AllowBlank:    true,
		DisplayOrder:  3,
	}
	if err := db.Create(&posReferendum).Error; err != nil {
		return fmt.Errorf("creating referendum position: %w", err)
	}

	// ── Candidates ────────────────────────────────────────────────────────────
	candidates := []models.Candidate{
		{
			ElectionID:     guildElection.ID,
			PositionID:     posPresident.ID,
			FullName:       "Akiki Brenda",
			Manifesto:      "Empowering student innovation, reliable campus WiFi, and transparent bursary allocation.",
			Bio:            "Year 4, BSE. Former Speaker of the Science Faculty.",
			BallotNumber:   1,
			ApprovalStatus: models.CandidateStatusApproved,
		},
		{
			ElectionID:     guildElection.ID,
			PositionID:     posPresident.ID,
			FullName:       "Mugisha Davis",
			Manifesto:      "Revitalizing student welfare, healthcare access, and expanded sports facilities.",
			Bio:            "Year 4, BBA. President of the Debate Society.",
			BallotNumber:   2,
			ApprovalStatus: models.CandidateStatusApproved,
		},
		{
			ElectionID:     guildElection.ID,
			PositionID:     posPresident.ID,
			FullName:       "Tusiime Faith",
			Manifesto:      "Academic integrity, modernized library resources, and active student representation.",
			Bio:            "Year 3, BED. Faculty representative.",
			BallotNumber:   3,
			ApprovalStatus: models.CandidateStatusApproved,
		},
		{
			ElectionID:     guildElection.ID,
			PositionID:     posVicePresident.ID,
			FullName:       "Katusabe Emmanuel",
			Manifesto:      "Fostering inter-faculty collaboration and transparent student accountability.",
			Bio:            "Year 3, BIT.",
			BallotNumber:   1,
			ApprovalStatus: models.CandidateStatusApproved,
		},
		{
			ElectionID:     guildElection.ID,
			PositionID:     posVicePresident.ID,
			FullName:       "Natukunda Peace",
			Manifesto:      "Promoting female student leadership and career mentorship partnerships.",
			Bio:            "Year 4, BPA.",
			BallotNumber:   2,
			ApprovalStatus: models.CandidateStatusApproved,
		},
		{
			ElectionID:     guildElection.ID,
			PositionID:     posReferendum.ID,
			FullName:       "YES — Approve Amendment",
			Manifesto:      "Vote YES to approve extending the electoral preparation window.",
			BallotNumber:   1,
			ApprovalStatus: models.CandidateStatusApproved,
		},
		{
			ElectionID:     guildElection.ID,
			PositionID:     posReferendum.ID,
			FullName:       "NO — Reject Amendment",
			Manifesto:      "Vote NO to retain the existing 14-day window.",
			BallotNumber:   2,
			ApprovalStatus: models.CandidateStatusApproved,
		},
	}

	for i := range candidates {
		if err := db.Create(&candidates[i]).Error; err != nil {
			return fmt.Errorf("creating candidate %s: %w", candidates[i].FullName, err)
		}
	}

	// ── Faculty Election (OPEN, published for a future window) ───────────────
	facultyElection := models.Election{
		Title:          "Faculty of Science & Technology Student Representative 2026",
		Description:    "Election of faculty representatives to the Academic Board.",
		Category:       models.ElectionCategoryFaculty,
		OrganizingUnit: "Dean of Science & Technology",
		Status:         models.ElectionStatusOpen,
		StartsAt:       now.Add(72 * time.Hour),
		EndsAt:         now.Add(120 * time.Hour),
		ResultPolicy:   models.ResultPolicyLive,
		AllowedDomain:  "std.mast.ac.ug",
		AllowedCourses: "bse,bcs,bit,bist",
		AllowedYears:   "",
	}
	if err := db.Create(&facultyElection).Error; err != nil {
		log.Printf("Warning: creating faculty election: %v", err)
	}

	log.Println("Campus demo elections seeded successfully.")
	return nil
}
