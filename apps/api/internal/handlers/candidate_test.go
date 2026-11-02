package handlers

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"campvote/apps/api/internal/eligibility"
	"campvote/apps/api/internal/models"
)

func TestCandidateApplicationWithDocuments(t *testing.T) {
	db := setupTestDB(t)

	// Create test election and position
	election := models.Election{
		Title:        "Test Election",
		Status:       models.ElectionStatusDraft,
		StartsAt:     time.Now().Add(24 * time.Hour),
		EndsAt:       time.Now().Add(48 * time.Hour),
		Category:     models.ElectionCategoryGuild,
		ResultPolicy: models.ResultPolicyAfterClose,
	}
	require.NoError(t, db.Create(&election).Error)

	position := models.Position{
		ElectionID:    election.ID,
		Title:         "Test Position",
		QuestionType:  models.QuestionTypeSingleChoice,
		MinSelections: 1,
		MaxSelections: 1,
		Seats:         1,
	}
	require.NoError(t, db.Create(&position).Error)

	// Test candidate application with new fields
	dob := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	candidate := models.Candidate{
		ElectionID:         election.ID,
		PositionID:         position.ID,
		FullName:           "Test Candidate",
		RegistrationNumber: "22/U/1001/PS",
		CampusResidence:    "Block A, Room 101",
		DateOfBirth:        &dob,
		IDDocumentURL:      "https://example.com/id.pdf",
		PassportPhotoURL:   "https://example.com/photo.jpg",
		ApprovalStatus:     models.CandidateStatusNominated,
	}

	err := db.Create(&candidate).Error
	require.NoError(t, err)

	// Verify the candidate was created with all fields
	var retrieved models.Candidate
	err = db.First(&retrieved, candidate.ID).Error
	require.NoError(t, err)

	assert.Equal(t, candidate.FullName, retrieved.FullName)
	assert.Equal(t, candidate.RegistrationNumber, retrieved.RegistrationNumber)
	assert.Equal(t, candidate.CampusResidence, retrieved.CampusResidence)
	assert.NotNil(t, retrieved.DateOfBirth)
	assert.Equal(t, candidate.IDDocumentURL, retrieved.IDDocumentURL)
	assert.Equal(t, candidate.PassportPhotoURL, retrieved.PassportPhotoURL)
}

func TestCandidateScreeningWorkflow(t *testing.T) {
	db := setupTestDB(t)

	// Create test election, position, and candidate
	election := models.Election{
		Title:        "Test Election",
		Status:       models.ElectionStatusDraft,
		StartsAt:     time.Now().Add(24 * time.Hour),
		EndsAt:       time.Now().Add(48 * time.Hour),
		Category:     models.ElectionCategoryGuild,
		ResultPolicy: models.ResultPolicyAfterClose,
	}
	require.NoError(t, db.Create(&election).Error)

	position := models.Position{
		ElectionID:    election.ID,
		Title:         "Test Position",
		QuestionType:  models.QuestionTypeSingleChoice,
		MinSelections: 1,
		MaxSelections: 1,
		Seats:         1,
	}
	require.NoError(t, db.Create(&position).Error)

	candidate := models.Candidate{
		ElectionID:     election.ID,
		PositionID:     position.ID,
		FullName:       "Test Candidate",
		ApprovalStatus: models.CandidateStatusNominated,
	}
	require.NoError(t, db.Create(&candidate).Error)

	// Test approval workflow
	now := time.Now().UTC()
	candidate.ApprovalStatus = models.CandidateStatusApproved
	candidate.ReviewedBy = "admin-user-id"
	candidate.ReviewedAt = &now
	candidate.ReviewNotes = "Candidate meets all requirements"

	err := db.Save(&candidate).Error
	require.NoError(t, err)

	// Verify approval
	var retrieved models.Candidate
	err = db.First(&retrieved, candidate.ID).Error
	require.NoError(t, err)

	assert.Equal(t, models.CandidateStatusApproved, retrieved.ApprovalStatus)
	assert.Equal(t, "admin-user-id", retrieved.ReviewedBy)
	assert.NotNil(t, retrieved.ReviewedAt)
	assert.Equal(t, "Candidate meets all requirements", retrieved.ReviewNotes)

	// Test rejection workflow
	candidate.ApprovalStatus = models.CandidateStatusRejected
	candidate.ReviewNotes = "Missing required documents"

	err = db.Save(&candidate).Error
	require.NoError(t, err)

	err = db.First(&retrieved, candidate.ID).Error
	require.NoError(t, err)

	assert.Equal(t, models.CandidateStatusRejected, retrieved.ApprovalStatus)
	assert.Equal(t, "Missing required documents", retrieved.ReviewNotes)
}

func TestEligibilityParsing(t *testing.T) {
	// Test that the new registration-number email parser works.
	cases := []struct {
		email string
		valid bool
		year  string
		crs   string
	}{
		{"2024bse088@std.mast.ac.ug", true, "2024", "bse"},
		{"2022bba001@std.mast.ac.ug", true, "2022", "bba"},
		{"staff@must.ac.ug", false, "", ""},
		{"student@external.com", false, "", ""},
	}
	for _, tc := range cases {
		id, ok := eligibility.ParseEmail(tc.email)
		if ok != tc.valid {
			t.Errorf("ParseEmail(%q) valid=%v, want %v", tc.email, ok, tc.valid)
			continue
		}
		if ok && (id.Year != tc.year || id.Course != tc.crs) {
			t.Errorf("ParseEmail(%q) = {%s %s}, want {%s %s}", tc.email, id.Year, id.Course, tc.year, tc.crs)
		}
	}
}

func setupTestDB(t *testing.T) *gorm.DB {
	// This is a placeholder - in a real test setup, you would use
	// a test database connection
	// For now, this test will be skipped in CI until proper test DB is configured
	t.Skip("Test database not configured - implement test DB setup")
	return nil
}
