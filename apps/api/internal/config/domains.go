package config

import "strings"

// Default campus domains. They are the only domains that may sign in:
// students self-register with the student domain, and administrators are
// invited on the staff domain.
const (
	DefaultStudentEmailDomain = "std.must.ac.ug"
	DefaultAdminEmailDomain   = "must.ac.ug"
)

// NormalizeDomain lowercases a domain and drops any leading "@".
func NormalizeDomain(domain string) string {
	return strings.ToLower(strings.TrimPrefix(strings.TrimSpace(domain), "@"))
}

// EmailDomain returns the part of an email strictly after the final "@",
// lowercased. It returns "" when the address has no local part or no domain.
func EmailDomain(email string) string {
	email = strings.TrimSpace(email)
	at := strings.LastIndex(email, "@")
	if at <= 0 || at == len(email)-1 {
		return ""
	}
	return strings.ToLower(email[at+1:])
}

// MatchesEmailDomain reports whether email's exact final domain is domain.
// A suffix check is deliberately not used: "x@evilstd.must.ac.ug" must not
// match "std.must.ac.ug".
func MatchesEmailDomain(email, domain string) bool {
	want := NormalizeDomain(domain)
	return want != "" && EmailDomain(email) == want
}

// StudentDomain returns the configured student domain, falling back to the
// documented default so a zero-value Config in a test still behaves sensibly.
func (c *Config) StudentDomain() string {
	if c != nil && NormalizeDomain(c.StudentEmailDomain) != "" {
		return NormalizeDomain(c.StudentEmailDomain)
	}
	return DefaultStudentEmailDomain
}

// AdminDomain returns the configured administrator domain.
func (c *Config) AdminDomain() string {
	if c != nil && NormalizeDomain(c.AdminEmailDomain) != "" {
		return NormalizeDomain(c.AdminEmailDomain)
	}
	return DefaultAdminEmailDomain
}

// IsStudentEmail reports whether email belongs to the student domain.
func (c *Config) IsStudentEmail(email string) bool {
	return MatchesEmailDomain(email, c.StudentDomain())
}

// IsAdminEmail reports whether email belongs to the administrator domain.
func (c *Config) IsAdminEmail(email string) bool {
	return MatchesEmailDomain(email, c.AdminDomain())
}
