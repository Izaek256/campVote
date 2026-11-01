package config

// EligibilityDomains is kept for backward compatibility only.
// New code uses the AllowedDomain, AllowedYears, and AllowedCourses fields
// on each Election model record instead of this global map.
//
// Deprecated: Do not add entries here. Configure eligibility per-election.
var EligibilityDomains = map[string][]string{}

// GetAllDomains returns all configured domains as a flat list.
// Deprecated: Use eligibility.RuleAllows instead.
func GetAllDomains() []string {
	return nil
}

// GetDomainsByYear returns domains for a specific academic year.
// Deprecated: Use eligibility.RuleAllows instead.
func GetDomainsByYear(year string) []string {
	return nil
}

// IsDomainEligible checks if an email matches any configured domain.
// Deprecated: Use eligibility.ParseEmail + eligibility.RuleAllows instead.
func IsDomainEligible(email string) bool {
	return false
}
