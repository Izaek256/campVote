package config

// EligibilityDomains defines the structured email domains for voter eligibility
// These domains are organized by academic year for MMU students
var EligibilityDomains = map[string][]string{
	"YEAR_2026": {
		"2026.mbr@std.must.ac.ug", "2026.pha@std.must.ac.ug", "2026.phs@std.must.ac.ug",
		"2026.bns@std.must.ac.ug", "2026.bnc@std.must.ac.ug", "2026.mls@std.must.ac.ug",
		"2026.mlc@std.must.ac.ug", "2026.bsp@std.must.ac.ug", "2026.bspc@std.must.ac.ug",
		"2026.bba@std.must.ac.ug", "2026.bpsm@std.must.ac.ug", "2026.bsaf@std.must.ac.ug",
		"2026.eco@std.must.ac.ug", "2026.bpcd@std.must.ac.ug", "2026.bsal@std.must.ac.ug",
		"2026.bit@std.must.ac.ug", "2026.bcs@std.must.ac.ug", "2026.bse@std.must.ac.ug",
		"2026.bs-chem-maths@std.must.ac.ug", "2026.bs-biological@std.must.ac.ug",
		"2026.bs-physical@std.must.ac.ug", "2026.bme@std.must.ac.ug", "2026.eee@std.must.ac.ug",
		"2026.pem@std.must.ac.ug", "2026.civ@std.must.ac.ug", "2026.mie@std.must.ac.ug",
		"2026.dlt@std.must.ac.ug",
	},
	"YEAR_2025": {
		"2025.mbr@std.must.ac.ug", "2025.pha@std.must.ac.ug", "2025.phs@std.must.ac.ug",
		"2025.bns@std.must.ac.ug", "2025.bnc@std.must.ac.ug", "2025.mls@std.must.ac.ug",
		"2025.mlc@std.must.ac.ug", "2025.bsp@std.must.ac.ug", "2025.bspc@std.must.ac.ug",
		"2025.bba@std.must.ac.ug", "2025.bsaf@std.must.ac.ug", "2025.bpsm@std.must.ac.ug",
		"2025.eco@std.must.ac.ug", "2025.bpcd@std.must.ac.ug", "2025.bsal@std.must.ac.ug",
		"2025.bit@std.must.ac.ug", "2025.bcs@std.must.ac.ug", "2025.bse@std.must.ac.ug",
		"2025.bs-biological@std.must.ac.ug", "2025.bs-chem-maths@std.must.ac.ug",
		"2025.bs-physical@std.must.ac.ug", "2025.bme@std.must.ac.ug", "2025.pem@std.must.ac.ug",
		"2025.eee@std.must.ac.ug", "2025.cve@std.must.ac.ug", "2025.mie@std.must.ac.ug",
		"2025.dlt@std.must.ac.ug",
	},
	"YEAR_2024": {
		"2024.mbr@std.must.ac.ug", "2024.pha@std.must.ac.ug", "2024.phs@std.must.ac.ug",
		"2024.bns@std.must.ac.ug", "2024.bnc@std.must.ac.ug", "2024.mls@std.must.ac.ug",
		"2024.mlc@std.must.ac.ug", "2024.bsp@std.must.ac.ug", "2024.bba@std.must.ac.ug",
		"2024.baf@std.must.ac.ug", "2024.bpsm@std.must.ac.ug", "2024.eco@std.must.ac.ug",
		"2024.bpcd@std.must.ac.ug", "2024.bsal@std.must.ac.ug", "2024.bgwh@std.must.ac.ug",
		"2024.bit@std.must.ac.ug", "2024.bcs@std.must.ac.ug", "2024.bse@std.must.ac.ug",
		"2024.bs@std.must.ac.ug", "2024.bme@std.must.ac.ug", "2024.pem@std.must.ac.ug",
		"2024.eee@std.must.ac.ug", "2024.mie@std.must.ac.ug", "2024.cve@std.must.ac.ug",
	},
	"YEAR_2023": {
		"2023.mbr@std.must.ac.ug", "2023.pha@std.must.ac.ug", "2023.bns@std.must.ac.ug",
		"2023.mls@std.must.ac.ug", "2023.bsp@std.must.ac.ug", "2023.bsal@std.must.ac.ug",
		"2023.bme@std.must.ac.ug", "2023.pem@std.must.ac.ug", "2023.eee@std.must.ac.ug",
		"2023.bse@std.must.ac.ug", "2023.mie@std.must.ac.ug", "2023.cve@std.must.ac.ug",
	},
	"YEAR_2022": {
		"2022.mbr@std.must.ac.ug",
	},
	"STAFF": {
		"@must.ac.ug",
	},
}

// GetAllDomains returns all configured domains as a flat list
func GetAllDomains() []string {
	var domains []string
	for _, yearDomains := range EligibilityDomains {
		domains = append(domains, yearDomains...)
	}
	return domains
}

// GetDomainsByYear returns domains for a specific academic year
func GetDomainsByYear(year string) []string {
	return EligibilityDomains[year]
}

// IsDomainEligible checks if an email domain is in the configured eligible domains
func IsDomainEligible(email string) bool {
	for _, yearDomains := range EligibilityDomains {
		for _, domain := range yearDomains {
			if len(email) > len(domain) && email[len(email)-len(domain):] == domain {
				return true
			}
		}
	}
	return false
}
