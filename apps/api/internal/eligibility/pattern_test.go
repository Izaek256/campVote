// Package eligibility — tests for legacy pattern matching behaviour.
// Now replaced by ParseEmail + RuleAllows (see identity_test.go).
// This file documents what the old patterns did; new tests use RuleAllows.
package eligibility

import (
	"testing"
)

// The legacy matchesPattern and domainAllowed functions have been removed.
// Their behaviour is now covered by ParseEmail + RuleAllows in identity_test.go.
// This placeholder exists so the file is not empty.
func TestRuleAllows_MatchesLegacyDomainBehaviour(t *testing.T) {
	cases := []struct {
		email  string
		domain string
		years  string
		course string
		want   bool
		desc   string
	}{
		// Student domain — any course, any year
		{"2026bse001@std.mast.ac.ug", "std.mast.ac.ug", "", "", true, "student domain match"},
		{"2026bse001@gmail.com", "std.mast.ac.ug", "", "", false, "wrong domain"},
		// BSE course filter
		{"2026bse001@std.mast.ac.ug", "std.mast.ac.ug", "", "bse", true, "BSE course match"},
		{"2026bcs001@std.mast.ac.ug", "std.mast.ac.ug", "", "bse", false, "BCS rejected for BSE filter"},
		// Year filter
		{"2026bse001@std.mast.ac.ug", "std.mast.ac.ug", "2026,2025", "", true, "year in list"},
		{"2024bse001@std.mast.ac.ug", "std.mast.ac.ug", "2026,2025", "", false, "year not in list"},
	}
	for _, tc := range cases {
		id, ok := ParseEmail(tc.email)
		if !ok {
			if tc.want {
				t.Errorf("ParseEmail(%q) failed, expected success", tc.email)
			}
			continue
		}
		e := makeElection(tc.domain, tc.years, tc.course)
		got := RuleAllows(e, id)
		if got != tc.want {
			t.Errorf("[%s] RuleAllows(%q domain=%q years=%q course=%q) = %v, want %v",
				tc.desc, tc.email, tc.domain, tc.years, tc.course, got, tc.want)
		}
	}
}
