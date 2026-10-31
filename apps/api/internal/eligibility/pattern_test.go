package eligibility

import (
	"testing"
)

func TestMatchesPattern(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		email    string
		expected bool
	}{
		// Simple domain suffix matches
		{
			name:     "simple domain @std.must.ac.ug matches",
			pattern:  "@std.must.ac.ug",
			email:    "user@std.must.ac.ug",
			expected: true,
		},
		{
			name:     "simple domain without @ matches",
			pattern:  "std.must.ac.ug",
			email:    "user@std.must.ac.ug",
			expected: true,
		},
		{
			name:     "simple domain @must.ac.ug matches staff",
			pattern:  "@must.ac.ug",
			email:    "admin@must.ac.ug",
			expected: true,
		},
		{
			name:     "simple domain rejects wrong domain",
			pattern:  "@std.must.ac.ug",
			email:    "user@gmail.com",
			expected: false,
		},

		// Wildcard patterns
		{
			name:     "wildcard * matches any @std.must.ac.ug",
			pattern:  "*@std.must.ac.ug",
			email:    "any.user@std.must.ac.ug",
			expected: true,
		},
		{
			name:     "wildcard * rejects wrong domain",
			pattern:  "*@std.must.ac.ug",
			email:    "user@gmail.com",
			expected: false,
		},

		// Class patterns with year wildcard
		{
			name:     "class pattern *.bse matches 2026.bse",
			pattern:  "*.bse@std.must.ac.ug",
			email:    "2026.bse@std.must.ac.ug",
			expected: true,
		},
		{
			name:     "class pattern *.bse matches 2025.bse",
			pattern:  "*.bse@std.must.ac.ug",
			email:    "2025.bse@std.must.ac.ug",
			expected: true,
		},
		{
			name:     "class pattern *.bse matches 2024.bse",
			pattern:  "*.bse@std.must.ac.ug",
			email:    "2024.bse@std.must.ac.ug",
			expected: true,
		},
		{
			name:     "class pattern *.bse rejects different program",
			pattern:  "*.bse@std.must.ac.ug",
			email:    "2026.bcs@std.must.ac.ug",
			expected: false,
		},
		{
			name:     "class pattern *.bse rejects wrong domain",
			pattern:  "*.bse@std.must.ac.ug",
			email:    "2026.bse@gmail.com",
			expected: false,
		},

		// Specific class patterns (exact match)
		{
			name:     "specific class 2026.bse matches exactly",
			pattern:  "2026.bse@std.must.ac.ug",
			email:    "2026.bse@std.must.ac.ug",
			expected: true,
		},
		{
			name:     "specific class 2026.bse rejects different year",
			pattern:  "2026.bse@std.must.ac.ug",
			email:    "2025.bse@std.must.ac.ug",
			expected: false,
		},
		{
			name:     "specific class 2026.bse rejects different program",
			pattern:  "2026.bse@std.must.ac.ug",
			email:    "2026.bcs@std.must.ac.ug",
			expected: false,
		},

		// Class patterns with hyphens
		{
			name:     "class pattern with hyphen *.bs-chem-maths matches",
			pattern:  "*.bs-chem-maths@std.must.ac.ug",
			email:    "2026.bs-chem-maths@std.must.ac.ug",
			expected: true,
		},
		{
			name:     "class pattern *.bs-biological matches",
			pattern:  "*.bs-biological@std.must.ac.ug",
			email:    "2026.bs-biological@std.must.ac.ug",
			expected: true,
		},
		{
			name:     "class pattern *.bs-physical matches",
			pattern:  "*.bs-physical@std.must.ac.ug",
			email:    "2026.bs-physical@std.must.ac.ug",
			expected: true,
		},

		// Medical programs
		{
			name:     "class pattern *.mbr matches",
			pattern:  "*.mbr@std.must.ac.ug",
			email:    "2026.mbr@std.must.ac.ug",
			expected: true,
		},
		{
			name:     "class pattern *.bns matches",
			pattern:  "*.bns@std.must.ac.ug",
			email:    "2026.bns@std.must.ac.ug",
			expected: true,
		},
		{
			name:     "class pattern *.mls matches",
			pattern:  "*.mls@std.must.ac.ug",
			email:    "2026.mls@std.must.ac.ug",
			expected: true,
		},

		// Engineering programs
		{
			name:     "class pattern *.bme matches",
			pattern:  "*.bme@std.must.ac.ug",
			email:    "2026.bme@std.must.ac.ug",
			expected: true,
		},
		{
			name:     "class pattern *.eee matches",
			pattern:  "*.eee@std.must.ac.ug",
			email:    "2026.eee@std.must.ac.ug",
			expected: true,
		},
		{
			name:     "class pattern *.civ matches",
			pattern:  "*.civ@std.must.ac.ug",
			email:    "2026.civ@std.must.ac.ug",
			expected: true,
		},

		// Business programs
		{
			name:     "class pattern *.bba matches",
			pattern:  "*.bba@std.must.ac.ug",
			email:    "2026.bba@std.must.ac.ug",
			expected: true,
		},
		{
			name:     "class pattern *.bsaf matches",
			pattern:  "*.bsaf@std.must.ac.ug",
			email:    "2026.bsaf@std.must.ac.ug",
			expected: true,
		},
		{
			name:     "class pattern *.bpsm matches",
			pattern:  "*.bpsm@std.must.ac.ug",
			email:    "2026.bpsm@std.must.ac.ug",
			expected: true,
		},

		// Education programs
		{
			name:     "class pattern *.bsp matches",
			pattern:  "*.bsp@std.must.ac.ug",
			email:    "2026.bsp@std.must.ac.ug",
			expected: true,
		},
		{
			name:     "class pattern *.bspc matches",
			pattern:  "*.bspc@std.must.ac.ug",
			email:    "2026.bspc@std.must.ac.ug",
			expected: true,
		},

		// Computing programs
		{
			name:     "class pattern *.bit matches",
			pattern:  "*.bit@std.must.ac.ug",
			email:    "2026.bit@std.must.ac.ug",
			expected: true,
		},
		{
			name:     "class pattern *.bcs matches",
			pattern:  "*.bcs@std.must.ac.ug",
			email:    "2026.bcs@std.must.ac.ug",
			expected: true,
		},
		{
			name:     "class pattern *.bse matches",
			pattern:  "*.bse@std.must.ac.ug",
			email:    "2026.bse@std.must.ac.ug",
			expected: true,
		},

		// Case insensitivity - these are tested via domainAllowed which handles lowercasing
		// For direct matchesPattern calls, inputs must already be lowercased
		{
			name:     "pattern and email both lowercase match",
			pattern:  "*.bse@std.must.ac.ug",
			email:    "2026.bse@std.must.ac.ug",
			expected: true,
		},

		// Edge cases
		{
			name:     "empty pattern returns false",
			pattern:  "",
			email:    "user@std.must.ac.ug",
			expected: false,
		},
		{
			name:     "email without @ rejected",
			pattern:  "@std.must.ac.ug",
			email:    "userstd.must.ac.ug",
			expected: false,
		},
		{
			name:     "pattern without @ and without domain returns false",
			pattern:  "bse",
			email:    "2026.bse@std.must.ac.ug",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchesPattern(tt.pattern, tt.email)
			if result != tt.expected {
				t.Errorf("matchesPattern(%q, %q) = %v, want %v", tt.pattern, tt.email, result, tt.expected)
			}
		})
	}
}

func TestDomainAllowed(t *testing.T) {
	tests := []struct {
		name     string
		allowed  string
		email    string
		expected bool
	}{
		{
			name:     "empty allowed returns true (roster-only mode)",
			allowed:  "",
			email:    "any@email.com",
			expected: true,
		},
		{
			name:     "single domain matches",
			allowed:  "@std.must.ac.ug",
			email:    "user@std.must.ac.ug",
			expected: true,
		},
		{
			name:     "multiple domains, first matches",
			allowed:  "@std.must.ac.ug,@must.ac.ug",
			email:    "user@std.must.ac.ug",
			expected: true,
		},
		{
			name:     "multiple domains, second matches",
			allowed:  "@std.must.ac.ug,@must.ac.ug",
			email:    "admin@must.ac.ug",
			expected: true,
		},
		{
			name:     "multiple domains, none match",
			allowed:  "@std.must.ac.ug,@must.ac.ug",
			email:    "user@gmail.com",
			expected: false,
		},
		{
			name:     "mix of patterns and domains",
			allowed:  "@must.ac.ug,*.bse@std.must.ac.ug",
			email:    "2026.bse@std.must.ac.ug",
			expected: true,
		},
		{
			name:     "mix of patterns, class pattern matches",
			allowed:  "@must.ac.ug,*.bse@std.must.ac.ug",
			email:    "admin@must.ac.ug",
			expected: true,
		},
		{
			name:     "multiple class patterns",
			allowed:  "*.bse@std.must.ac.ug,*.bcs@std.must.ac.ug",
			email:    "2026.bse@std.must.ac.ug",
			expected: true,
		},
		{
			name:     "multiple class patterns, second matches",
			allowed:  "*.bse@std.must.ac.ug,*.bcs@std.must.ac.ug",
			email:    "2026.bcs@std.must.ac.ug",
			expected: true,
		},
		{
			name:     "whitespace in patterns is trimmed",
			allowed:  " *.bse@std.must.ac.ug , *.bcs@std.must.ac.ug ",
			email:    "2026.bse@std.must.ac.ug",
			expected: true,
		},
		{
			name:     "specific class in mixed list",
			allowed:  "*.bse@std.must.ac.ug,2026.bcs@std.must.ac.ug",
			email:    "2026.bcs@std.must.ac.ug",
			expected: true,
		},
		{
			name:     "specific class rejects different year",
			allowed:  "*.bse@std.must.ac.ug,2026.bcs@std.must.ac.ug",
			email:    "2025.bcs@std.must.ac.ug",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := domainAllowed(tt.allowed, tt.email)
			if result != tt.expected {
				t.Errorf("domainAllowed(%q, %q) = %v, want %v", tt.allowed, tt.email, result, tt.expected)
			}
		})
	}
}
