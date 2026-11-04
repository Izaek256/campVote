package eligibility

import (
	"sort"
	"strings"

	"gorm.io/gorm"

	"campvote/apps/api/internal/models"
)

// FacultyUnspecified labels eligible voters and cast ballots whose account the
// election roster does not name, or whose roster row carries no faculty.
// Keeping them in their own row is what lets the faculty columns add up to the
// headline turnout instead of silently dropping people.
const FacultyUnspecified = "Not on roster"

// FacultyTurnout is one row of the participation-by-faculty breakdown. It is
// aggregate only: counts of people and ballots, never a name, an address or a
// link to how anybody voted.
type FacultyTurnout struct {
	Faculty string `json:"faculty"`
	// Eligible and Ballots are the published figures. Both are zero when
	// Suppressed is set, so a group too small to be anonymous is never
	// published even in the raw response.
	Eligible   int     `json:"eligible"`
	Ballots    int     `json:"ballots"`
	Percent    float64 `json:"percent"`
	Suppressed bool    `json:"suppressed,omitempty"`
}

// FacultyTurnoutBreakdown counts eligible voters and cast ballots per faculty
// for one election.
//
// Eligible voters are the same rule-based electorate the tally counts, so a row
// counts accounts rather than roster lines and the rows add up to the eligible
// figure the results headline reports. A voter's faculty is read from the
// election roster, which is the only record that carries it; an account the
// roster does not name is reported under FacultyUnspecified.
//
// Ballots are attributed from participation records alone, so they add up to
// the headline ballot count even for somebody who has since been removed from
// the roster. Nothing here can be joined back to a choice.
//
// A faculty with fewer than threshold eligible voters is reported with
// Suppressed set and no figures. A threshold of zero publishes every group.
func FacultyTurnoutBreakdown(db *gorm.DB, election *models.Election, threshold int) ([]FacultyTurnout, error) {
	type rosterRow struct {
		Email   string
		Faculty string
		Status  string
	}
	var roster []rosterRow
	if err := db.Model(&models.ElectionRoster{}).
		Select("email, faculty, status").
		Where("election_id = ?", election.ID).
		Scan(&roster).Error; err != nil {
		return nil, err
	}

	facultyOf := make(map[string]string, len(roster))
	blocked := make(map[string]bool)
	for _, row := range roster {
		email := strings.ToLower(strings.TrimSpace(row.Email))
		if email == "" {
			continue
		}
		if _, seen := facultyOf[email]; !seen {
			facultyOf[email] = strings.TrimSpace(row.Faculty)
		}
		if row.Status != models.RosterStatusEligible {
			blocked[email] = true
		}
	}

	// Faculty is only ever recorded against a roster line, so an account is
	// placed by the address it signs in with.
	facultyFor := func(email string) string {
		faculty, ok := facultyOf[strings.ToLower(strings.TrimSpace(email))]
		if !ok || faculty == "" {
			return FacultyUnspecified
		}
		return faculty
	}

	var users []models.User
	if err := db.Model(&models.User{}).
		Select("id", "email", "email_verified_at").Find(&users).Error; err != nil {
		return nil, err
	}
	eligibleByFaculty := make(map[string]int)
	facultyByUser := make(map[string]string, len(users))
	for _, user := range users {
		email := strings.ToLower(strings.TrimSpace(user.Email))
		facultyByUser[user.ID] = facultyFor(email)
		if user.EmailVerifiedAt == nil || blocked[email] {
			continue
		}
		identity, parsed := ParseEmail(user.Email)
		if !parsed || !RuleAllows(election, identity) {
			continue
		}
		eligibleByFaculty[facultyByUser[user.ID]]++
	}

	var voterIDs []string
	if err := db.Model(&models.VoterParticipation{}).
		Where("election_id = ?", election.ID).
		Pluck("user_id", &voterIDs).Error; err != nil {
		return nil, err
	}
	ballotsByFaculty := make(map[string]int)
	for _, userID := range voterIDs {
		faculty, ok := facultyByUser[userID]
		if !ok {
			faculty = FacultyUnspecified
		}
		ballotsByFaculty[faculty]++
	}

	faculties := make([]string, 0, len(eligibleByFaculty))
	for faculty := range eligibleByFaculty {
		faculties = append(faculties, faculty)
	}
	sort.Slice(faculties, func(i, j int) bool {
		if eligibleByFaculty[faculties[i]] != eligibleByFaculty[faculties[j]] {
			return eligibleByFaculty[faculties[i]] > eligibleByFaculty[faculties[j]]
		}
		return faculties[i] < faculties[j]
	})

	rows := make([]FacultyTurnout, 0, len(faculties))
	for _, faculty := range faculties {
		eligible := eligibleByFaculty[faculty]
		if threshold > 0 && eligible < threshold {
			rows = append(rows, FacultyTurnout{Faculty: faculty, Suppressed: true})
			continue
		}
		ballots := ballotsByFaculty[faculty]
		row := FacultyTurnout{Faculty: faculty, Eligible: eligible, Ballots: ballots}
		if eligible > 0 {
			row.Percent = float64(ballots) / float64(eligible) * 100
		}
		rows = append(rows, row)
	}
	return rows, nil
}
