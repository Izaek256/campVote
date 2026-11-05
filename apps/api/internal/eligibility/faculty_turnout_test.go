package eligibility_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"

	"campvote/apps/api/internal/eligibility"
	"campvote/apps/api/internal/models"
)

// rosterWithFaculty adds one roster line that names a faculty. The faculty is
// only ever recorded here, which is why the breakdown reads it from the roster.
func rosterWithFaculty(t *testing.T, db *gorm.DB, electionID, student, email, faculty string) {
	t.Helper()
	entry := &models.ElectionRoster{
		ElectionID: electionID, StudentNumber: student, Email: email,
		FullName: "Synthetic Student", Faculty: faculty,
		Status: models.RosterStatusEligible, Source: models.RosterSourceCSVImport,
	}
	if err := db.Create(entry).Error; err != nil {
		t.Fatal(err)
	}
}

// facultyFixture builds an election with two named faculties, one eligible
// student the roster does not name, and three cast ballots.
func facultyFixture(t *testing.T, db *gorm.DB) (*models.Election, map[string]*models.User) {
	t.Helper()
	now := time.Now().UTC()
	election := openElection(t, db, "e-a", now.Add(-time.Hour), now.Add(time.Hour))

	people := []struct{ key, email, faculty, student string }{
		{"science-1", studentEmail("2022", "bse", "001"), "Science", "22/U/0001"},
		{"science-2", studentEmail("2022", "bse", "002"), "Science", "22/U/0002"},
		{"business-1", studentEmail("2022", "bse", "003"), "Business", "22/U/0003"},
		{"unlisted", studentEmail("2022", "bse", "004"), "", ""},
	}
	users := make(map[string]*models.User, len(people))
	for _, person := range people {
		user := account(person.email)
		if err := db.Create(user).Error; err != nil {
			t.Fatal(err)
		}
		users[person.key] = user
		if person.faculty != "" {
			rosterWithFaculty(t, db, "e-a", person.student, person.email, person.faculty)
		}
	}
	for _, key := range []string{"science-1", "business-1", "unlisted"} {
		if err := db.Create(&models.VoterParticipation{
			ElectionID: "e-a", UserID: users[key].ID,
		}).Error; err != nil {
			t.Fatal(err)
		}
	}
	return election, users
}

func rowFor(rows []eligibility.FacultyTurnout, faculty string) (eligibility.FacultyTurnout, bool) {
	for _, row := range rows {
		if row.Faculty == faculty {
			return row, true
		}
	}
	return eligibility.FacultyTurnout{}, false
}

func TestFacultyTurnoutAddsUpToTheHeadline(t *testing.T) {
	db := testDB(t)
	election, _ := facultyFixture(t, db)

	rows, err := eligibility.FacultyTurnoutBreakdown(db, election, 0)
	if err != nil {
		t.Fatal(err)
	}

	science, ok := rowFor(rows, "Science")
	if !ok {
		t.Fatalf("no Science row in %+v", rows)
	}
	if science.Eligible != 2 || science.Ballots != 1 || science.Percent != 50 {
		t.Errorf("Science = %+v, want 2 eligible, 1 ballot, 50%%", science)
	}
	business, ok := rowFor(rows, "Business")
	if !ok {
		t.Fatalf("no Business row in %+v", rows)
	}
	if business.Eligible != 1 || business.Ballots != 1 {
		t.Errorf("Business = %+v, want 1 eligible, 1 ballot", business)
	}
	// An eligible account the roster does not name must still be reported, or
	// the columns would not add up to the headline turnout.
	unlisted, ok := rowFor(rows, eligibility.FacultyUnspecified)
	if !ok {
		t.Fatalf("no %q row in %+v", eligibility.FacultyUnspecified, rows)
	}
	if unlisted.Eligible != 1 || unlisted.Ballots != 1 {
		t.Errorf("%s = %+v, want 1 eligible, 1 ballot", eligibility.FacultyUnspecified, unlisted)
	}

	// The rows are the headline figure split up, not a second count of it.
	totalEligible, totalBallots := 0, 0
	for _, row := range rows {
		totalEligible += row.Eligible
		totalBallots += row.Ballots
	}
	eligible, err := eligibility.EligibleCount(db, election)
	if err != nil {
		t.Fatal(err)
	}
	if int(eligible) != totalEligible || totalEligible != 4 {
		t.Errorf("rows total %d eligible, headline says %d", totalEligible, eligible)
	}
	var cast int64
	db.Model(&models.VoterParticipation{}).Where("election_id = ?", "e-a").Count(&cast)
	if int(cast) != totalBallots || totalBallots != 3 {
		t.Errorf("rows total %d ballots, recorded %d", totalBallots, cast)
	}
}

func TestFacultyTurnoutOrdersBySize(t *testing.T) {
	db := testDB(t)
	election, _ := facultyFixture(t, db)

	rows, err := eligibility.FacultyTurnoutBreakdown(db, election, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 {
		t.Fatalf("got %d rows, want 3: %+v", len(rows), rows)
	}
	// Largest faculty first, then alphabetical for the equal-sized remainder,
	// so the table reads the same way twice.
	if rows[0].Faculty != "Science" || rows[1].Faculty != "Business" ||
		rows[2].Faculty != eligibility.FacultyUnspecified {
		t.Errorf("unexpected order: %q, %q, %q", rows[0].Faculty, rows[1].Faculty, rows[2].Faculty)
	}
}

func TestFacultyTurnoutWithholdsGroupsBelowTheThreshold(t *testing.T) {
	db := testDB(t)
	election, _ := facultyFixture(t, db)

	rows, err := eligibility.FacultyTurnoutBreakdown(db, election, 2)
	if err != nil {
		t.Fatal(err)
	}

	science, _ := rowFor(rows, "Science")
	if science.Suppressed || science.Eligible != 2 {
		t.Errorf("Science should be published at a threshold of 2: %+v", science)
	}
	for _, faculty := range []string{"Business", eligibility.FacultyUnspecified} {
		row, ok := rowFor(rows, faculty)
		if !ok {
			t.Fatalf("suppression dropped the %s row entirely: %+v", faculty, rows)
		}
		if !row.Suppressed {
			t.Errorf("%s should be suppressed: %+v", faculty, row)
		}
		// Withholding has to mean the figures are absent, not just labelled.
		if row.Eligible != 0 || row.Ballots != 0 || row.Percent != 0 {
			t.Errorf("%s published figures under suppression: %+v", faculty, row)
		}
	}
}

func TestFacultyTurnoutCarriesNoIdentity(t *testing.T) {
	db := testDB(t)
	election, users := facultyFixture(t, db)

	rows, err := eligibility.FacultyTurnoutBreakdown(db, election, 0)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(rows)
	if err != nil {
		t.Fatal(err)
	}
	for key, user := range users {
		for _, secret := range []string{key, user.ID, user.Email} {
			if strings.Contains(string(encoded), secret) {
				t.Errorf("breakdown leaked %q: %s", secret, encoded)
			}
		}
	}
	if strings.Contains(string(encoded), "user_id") || strings.Contains(string(encoded), "email") {
		t.Errorf("breakdown exposes an identity field: %s", encoded)
	}
}
