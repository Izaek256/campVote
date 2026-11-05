package eligibility

import (
	"testing"
	"time"

	"campvote/apps/api/internal/models"
)

// ── ParseEmail ────────────────────────────────────────────────────────────────

func TestParseEmail_Valid(t *testing.T) {
	cases := []struct {
		email  string
		year   string
		course string
		number string
		domain string
	}{
		{"2024BSE088@std.mast.ac.ug", "2024", "bse", "088", "std.mast.ac.ug"},
		{"2024bse088@std.mast.ac.ug", "2024", "bse", "088", "std.mast.ac.ug"},
		{"2022BBA002@std.mast.ac.ug", "2022", "bba", "002", "std.mast.ac.ug"},
		{"2023bit004@std.mast.ac.ug", "2023", "bit", "004", "std.mast.ac.ug"},
		{"2025bed99999@std.mast.ac.ug", "2025", "bed", "99999", "std.mast.ac.ug"},
		// With extra whitespace
		{"  2024BSE001@std.mast.ac.ug  ", "2024", "bse", "001", "std.mast.ac.ug"},
	}
	for _, tc := range cases {
		id, ok := ParseEmail(tc.email)
		if !ok {
			t.Errorf("ParseEmail(%q) returned false, want true", tc.email)
			continue
		}
		if id.Year != tc.year || id.Course != tc.course || id.Number != tc.number || id.Domain != tc.domain {
			t.Errorf("ParseEmail(%q) = {%s %s %s %s}, want {%s %s %s %s}",
				tc.email, id.Year, id.Course, id.Number, id.Domain,
				tc.year, tc.course, tc.number, tc.domain)
		}
	}
}

func TestParseEmail_Invalid(t *testing.T) {
	cases := []string{
		// Staff / non-registration emails
		"admin@must.ac.ug",
		"john.doe@gmail.com",
		"staff@std.mast.ac.ug",        // no leading year
		"2024@std.mast.ac.ug",          // no course, no number
		"2024BSE@std.mast.ac.ug",       // no number
		"noatsign",
		"",
		"@std.mast.ac.ug",
	}
	for _, tc := range cases {
		_, ok := ParseEmail(tc)
		if ok {
			t.Errorf("ParseEmail(%q) returned true, want false", tc)
		}
	}
}

// ── RuleAllows ────────────────────────────────────────────────────────────────

func makeElection(domain, years, courses string) *models.Election {
	return &models.Election{
		AllowedDomain:  domain,
		AllowedYears:   years,
		AllowedCourses: courses,
	}
}

func TestRuleAllows_DomainOnly(t *testing.T) {
	e := makeElection("std.mast.ac.ug", "", "")
	id, _ := ParseEmail("2024bse001@std.mast.ac.ug")

	if !RuleAllows(e, id) {
		t.Error("RuleAllows: expected eligible for matching domain, got false")
	}
}

func TestRuleAllows_WrongDomain(t *testing.T) {
	e := makeElection("std.mast.ac.ug", "", "")
	id := Identity{Year: "2024", Course: "bse", Number: "001", Domain: "std.other.ac.ug"}

	if RuleAllows(e, id) {
		t.Error("RuleAllows: expected ineligible for wrong domain, got true")
	}
}

func TestRuleAllows_DomainWithAtSign(t *testing.T) {
	// AllowedDomain may carry a leading @ (old data); it should be stripped.
	e := makeElection("@std.mast.ac.ug", "", "")
	id, _ := ParseEmail("2024bse001@std.mast.ac.ug")
	if !RuleAllows(e, id) {
		t.Error("RuleAllows: expected eligible when domain has leading @")
	}
}

func TestRuleAllows_YearFilter_Match(t *testing.T) {
	e := makeElection("std.mast.ac.ug", "2024,2025", "")
	id, _ := ParseEmail("2024bse001@std.mast.ac.ug")
	if !RuleAllows(e, id) {
		t.Error("RuleAllows: expected eligible for year in allowed list")
	}
}

func TestRuleAllows_YearFilter_NoMatch(t *testing.T) {
	e := makeElection("std.mast.ac.ug", "2025", "")
	id, _ := ParseEmail("2024bse001@std.mast.ac.ug")
	if RuleAllows(e, id) {
		t.Error("RuleAllows: expected ineligible for year not in allowed list")
	}
}

func TestRuleAllows_CourseFilter_Match(t *testing.T) {
	e := makeElection("std.mast.ac.ug", "", "bse,bcs")
	id, _ := ParseEmail("2024bse001@std.mast.ac.ug")
	if !RuleAllows(e, id) {
		t.Error("RuleAllows: expected eligible for course in allowed list")
	}
}

func TestRuleAllows_CourseFilter_NoMatch(t *testing.T) {
	e := makeElection("std.mast.ac.ug", "", "bba")
	id, _ := ParseEmail("2024bse001@std.mast.ac.ug")
	if RuleAllows(e, id) {
		t.Error("RuleAllows: expected ineligible for course not in list")
	}
}

func TestRuleAllows_CourseCaseInsensitive(t *testing.T) {
	e := makeElection("std.mast.ac.ug", "", "BSE") // upper-cased in config
	id, _ := ParseEmail("2024BSE001@std.mast.ac.ug")
	if !RuleAllows(e, id) {
		t.Error("RuleAllows: course match should be case-insensitive")
	}
}

func TestRuleAllows_YearAndCourse_BothMustMatch(t *testing.T) {
	e := makeElection("std.mast.ac.ug", "2024", "bse")
	// Right year, wrong course
	wrongCourse, _ := ParseEmail("2024bba001@std.mast.ac.ug")
	if RuleAllows(e, wrongCourse) {
		t.Error("RuleAllows: expected false when course does not match despite year matching")
	}
	// Right course, wrong year
	wrongYear, _ := ParseEmail("2023bse001@std.mast.ac.ug")
	if RuleAllows(e, wrongYear) {
		t.Error("RuleAllows: expected false when year does not match despite course matching")
	}
	// Both match
	both, _ := ParseEmail("2024bse001@std.mast.ac.ug")
	if !RuleAllows(e, both) {
		t.Error("RuleAllows: expected true when year and course both match")
	}
}

func TestRuleAllows_EmptyDomainAllowsAll(t *testing.T) {
	// An election with no AllowedDomain set defaults to no domain restriction.
	e := makeElection("", "", "")
	id := Identity{Year: "2024", Course: "bse", Number: "001", Domain: "anything.ac.ug"}
	if !RuleAllows(e, id) {
		t.Error("RuleAllows: empty AllowedDomain should allow any domain")
	}
}

func TestIdentityProgrammeAndStudyYear(t *testing.T) {
	id, ok := ParseEmail("2024BSE088@std.mast.ac.ug")
	if !ok {
		t.Fatal("expected the registration email to parse")
	}
	if id.Programme() != "BSE" {
		t.Errorf("Programme() = %q, want BSE", id.Programme())
	}
	cases := []struct {
		at   string
		want int
	}{
		{"2024-10-01T00:00:00Z", 1}, // enrolled in the academic year that just started
		{"2025-07-31T00:00:00Z", 1}, // still the 2024/25 academic year
		{"2025-08-01T00:00:00Z", 2}, // new academic year
		{"2026-11-02T00:00:00Z", 3},
	}
	for _, tc := range cases {
		at, err := time.Parse(time.RFC3339, tc.at)
		if err != nil {
			t.Fatal(err)
		}
		if got := id.StudyYear(at); got != tc.want {
			t.Errorf("StudyYear(%s) = %d, want %d", tc.at, got, tc.want)
		}
	}
}
