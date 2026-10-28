package database

import (
	"fmt"
	"log"
	"os"
	"time"

	"campvote/apps/api/internal/models"
	"gorm.io/gorm"
)

// SeedElections populates synthetic demo elections, positions, candidates, and rosters
// for local testing and demonstration at Mountains of the Moon University (MMU).
func SeedElections(db *gorm.DB) error {
	// Demo elections include demo voter accounts with a shared password. They
	// are development fixtures, never production data, so they follow the same
	// APP_ENV=development gate as the demo users.
	if !seedDevDefaults() {
		log.Printf("Skipping demo elections: APP_ENV is %q, not development", os.Getenv("APP_ENV"))
		return nil
	}

	// Only seed if no elections exist yet
	var count int64
	db.Model(&models.Election{}).Count(&count)
	if count > 0 {
		log.Println("Elections already seeded, skipping...")
		return nil
	}

	log.Println("Seeding MMU campus elections, positions, candidates, and eligibility roster...")

	now := time.Now().UTC()

	// 1. Seed MMU student demo accounts. No administrator is seeded: the first
	// administrator comes from cmd/bootstrap, so no demo run plants an ADMIN
	// with a known password.
	demoAccounts := []models.User{
		{
			FirstName: "Ivan",
			LastName:  "Mugisha",
			Email:     "ivan.mugisha@std.must.ac.ug",
			Password:  "admin123",
			Role:      models.RoleUser,
			Active:    true,
		},
		{
			FirstName: "Doreen",
			LastName:  "Kemigisa",
			Email:     "doreen.kemigisa@std.must.ac.ug",
			Password:  "admin123",
			Role:      models.RoleUser,
			Active:    true,
		},
	}

	for _, acc := range demoAccounts {
		var userCount int64
		db.Model(&models.User{}).Where("email = ?", acc.Email).Count(&userCount)
		if userCount == 0 {
			// The roster trusts this address, so the demo account owns it.
			acc.EmailVerifiedAt = &now
			if err := db.Create(&acc).Error; err != nil {
				log.Printf("Warning: failed to seed demo account %s: %v", acc.Email, err)
			} else {
				log.Printf("Created campus demo account: %s", acc.Email)
			}
		}
	}

	// 2. Active Guild Election
	guildElection := models.Election{
		Title:          "MMU Guild Representative Council Elections 2026/2027",
		Description:    "Annual democratic election of the MMU Guild President, Vice Guild President, and student referendum.",
		Category:       models.ElectionCategoryGuild,
		OrganizingUnit: "MMU Electoral Commission",
		Status:         models.ElectionStatusOpen,
		StartsAt:       now.Add(-24 * time.Hour),
		EndsAt:         now.Add(48 * time.Hour),
		ResultPolicy:   models.ResultPolicyAfterClose,
		AllowedDomains: "@std.must.ac.ug,@must.ac.ug",
		RequireRoster:  true,
	}

	if err := db.Create(&guildElection).Error; err != nil {
		return fmt.Errorf("creating guild election: %w", err)
	}

	// 3. Positions for Guild Election
	posPresident := models.Position{
		ElectionID:    guildElection.ID,
		Title:         "Guild President",
		Description:   "Chief executive officer of the MMU Guild Government.",
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
		ElectionID:    guildElection.ID,
		Title:         "Vice Guild President",
		Description:   "Principal deputy to the MMU Guild President.",
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
		ElectionID:    guildElection.ID,
		Title:         "Referendum: Guild Constitution Article 42 Amendment",
		Description:   "Do you approve the amendment to extend the electoral preparation window from 14 to 21 days?",
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

	// 4. Candidates for Positions
	candidates := []models.Candidate{
		{
			ElectionID:     guildElection.ID,
			PositionID:     posPresident.ID,
			FullName:       "Akiki Brenda",
			Manifesto:      "Empowering student innovation, reliable campus WiFi, and transparent guild bursary allocation.",
			Bio:            "Year 3, Bachelor of Science in Computer Science. Former Speaker of the Science Faculty.",
			BallotNumber:   1,
			ApprovalStatus: models.CandidateStatusApproved,
		},
		{
			ElectionID:     guildElection.ID,
			PositionID:     posPresident.ID,
			FullName:       "Mugisha Davis",
			Manifesto:      "Revitalizing student welfare, healthcare access on campus, and expanded sports facilities.",
			Bio:            "Year 3, Bachelor of Business Administration. President of the Debate Society.",
			BallotNumber:   2,
			ApprovalStatus: models.CandidateStatusApproved,
		},
		{
			ElectionID:     guildElection.ID,
			PositionID:     posPresident.ID,
			FullName:       "Tusiime Faith",
			Manifesto:      "Academic integrity, modernized library research resources, and active student representation.",
			Bio:            "Year 2, Bachelor of Education. Faculty representative.",
			BallotNumber:   3,
			ApprovalStatus: models.CandidateStatusApproved,
		},
		{
			ElectionID:     guildElection.ID,
			PositionID:     posVicePresident.ID,
			FullName:       "Katusabe Emmanuel",
			Manifesto:      "Fostering inter-faculty collaboration and transparent student accountability.",
			Bio:            "Year 2, Bachelor of Information Technology.",
			BallotNumber:   1,
			ApprovalStatus: models.CandidateStatusApproved,
		},
		{
			ElectionID:     guildElection.ID,
			PositionID:     posVicePresident.ID,
			FullName:       "Natukunda Peace",
			Manifesto:      "Promoting female student leadership and career mentorship partnerships.",
			Bio:            "Year 3, Bachelor of Public Administration.",
			BallotNumber:   2,
			ApprovalStatus: models.CandidateStatusApproved,
		},
		{
			ElectionID:     guildElection.ID,
			PositionID:     posReferendum.ID,
			FullName:       "YES (Approve Amendment)",
			Manifesto:      "Vote YES to approve extending the electoral preparation window.",
			BallotNumber:   1,
			ApprovalStatus: models.CandidateStatusApproved,
		},
		{
			ElectionID:     guildElection.ID,
			PositionID:     posReferendum.ID,
			FullName:       "NO (Reject Amendment)",
			Manifesto:      "Vote NO to retain the existing 14-day window.",
			BallotNumber:   2,
			ApprovalStatus: models.CandidateStatusApproved,
		},
	}

	for _, cand := range candidates {
		if err := db.Create(&cand).Error; err != nil {
			return fmt.Errorf("creating candidate %s: %w", cand.FullName, err)
		}
	}

	// 5. Seed sample eligibility roster entries
	rosterEntries := []models.ElectionRoster{
		{
			ElectionID:    guildElection.ID,
			StudentNumber: "22/U/1001/PS",
			RegNumber:     "MMU/22/01001",
			Email:         "ivan.mugisha@std.must.ac.ug",
			FullName:      "Ivan Mugisha",
			Faculty:       "Faculty of Science & Technology",
			Department:    "Computer Science",
			Programme:     "Bachelor of Science in Computer Science",
			StudyYear:     3,
			Status:        models.RosterStatusEligible,
		},
		{
			ElectionID:    guildElection.ID,
			StudentNumber: "22/U/1002/PS",
			RegNumber:     "MMU/22/01002",
			Email:         "doreen.kemigisa@std.must.ac.ug",
			FullName:      "Doreen Kemigisa",
			Faculty:       "Faculty of Business & Management",
			Department:    "Accounting and Finance",
			Programme:     "Bachelor of Business Administration",
			StudyYear:     2,
			Status:        models.RosterStatusEligible,
		},
	}

	for _, r := range rosterEntries {
		if err := db.Create(&r).Error; err != nil {
			log.Printf("Warning: creating roster entry %s: %v", r.StudentNumber, err)
		}
	}

	// 6. Seed a Scheduled Faculty Election
	facultyElection := models.Election{
		Title:          "Faculty of Science & Technology Student Representative",
		Description:    "Election of faculty representatives to the MMU Academic Board.",
		Category:       models.ElectionCategoryFaculty,
		OrganizingUnit: "Dean of Science & Technology",
		Status:         models.ElectionStatusScheduled,
		StartsAt:       now.Add(72 * time.Hour),
		EndsAt:         now.Add(120 * time.Hour),
		ResultPolicy:   models.ResultPolicyLive,
		AllowedDomains: "@std.must.ac.ug",
		RequireRoster:  true,
	}
	if err := db.Create(&facultyElection).Error; err != nil {
		log.Printf("Warning: creating faculty election: %v", err)
	}

	log.Println("MMU campus elections seeding completed successfully.")
	return nil
}
