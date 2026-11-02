// Package eligibility decides whether an authenticated account may vote in a
// particular election.
package eligibility

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"campvote/apps/api/internal/models"
)

// regNoPattern matches the local part of a registration-number email:
// e.g. "2024bse088" → year=2024, course=bse, number=088
var regNoPattern = regexp.MustCompile(`^(\d{4})([a-z][a-z-]*)(\d{1,5})$`)

// Identity holds the parsed components of a campus student registration email.
type Identity struct {
	Year   string // Enrolment year, e.g. "2024"
	Course string // Lower-cased course code, e.g. "bse"
	Number string // Student serial within course-year, e.g. "088"
	Domain string // Email domain, e.g. "std.mast.ac.ug"
}

// ParseEmail lowercases and trims the email, then extracts the Identity from a
// registration-number email such as 2024BSE088@std.mast.ac.ug.
// Returns (Identity, true) on success, ({}, false) if the email is not in the
// expected format (e.g. staff or external addresses).
func ParseEmail(email string) (Identity, bool) {
	email = strings.ToLower(strings.TrimSpace(email))
	atIdx := strings.LastIndex(email, "@")
	if atIdx <= 0 || atIdx == len(email)-1 {
		return Identity{}, false
	}
	local := email[:atIdx]
	domain := email[atIdx+1:]
	m := regNoPattern.FindStringSubmatch(local)
	if m == nil {
		return Identity{}, false
	}
	return Identity{
		Year:   m[1],
		Course: m[2],
		Number: m[3],
		Domain: domain,
	}, true
}

// RuleAllows reports whether the identity is permitted by the election's
// eligibility rules:
//   - AllowedDomain must match id.Domain (leading "@" is stripped if present).
//   - AllowedYears is a comma-separated list; empty means any year.
//   - AllowedCourses is a comma-separated list; empty means any course.
//     Matching is case-insensitive.
func RuleAllows(e *models.Election, id Identity) bool {
	// Domain check — required
	wantDomain := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(e.AllowedDomain), "@"))
	if wantDomain != "" && id.Domain != wantDomain {
		return false
	}

	// Year check — optional (empty = any)
	if e.AllowedYears != "" {
		if !inList(id.Year, e.AllowedYears) {
			return false
		}
	}

	// Course check — optional (empty = any)
	if e.AllowedCourses != "" {
		if !inList(id.Course, e.AllowedCourses) {
			return false
		}
	}

	return true
}

// Programme returns the upper-cased course code, e.g. "BSE".
func (id Identity) Programme() string {
	return strings.ToUpper(id.Course)
}

// StudyYear derives the year of study from the enrolment year and the current
// date. The academic year starts in August, so in November 2026 a student who
// enrolled in 2024 is in year 3. A future enrolment year is clamped to 1.
func (id Identity) StudyYear(at time.Time) int {
	year, err := strconv.Atoi(id.Year)
	if err != nil {
		return 0
	}
	at = at.UTC()
	academicStart := at.Year()
	if at.Month() < time.August {
		academicStart--
	}
	studyYear := academicStart - year + 1
	if studyYear < 1 {
		return 1
	}
	return studyYear
}

// inList checks whether val appears in a comma-separated list, case-insensitively.
func inList(val, list string) bool {
	val = strings.ToLower(strings.TrimSpace(val))
	for _, item := range strings.Split(list, ",") {
		if strings.ToLower(strings.TrimSpace(item)) == val {
			return true
		}
	}
	return false
}
