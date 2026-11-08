package landing

import (
	"net/mail"
	"net/url"
	"strings"
	"unicode"
)

// Validate checks a landing document and returns per-field messages, keyed by a
// dotted path the admin form can put under the matching input. An empty map
// means the document is safe to store and render.
//
// Everything here enforces "plain text only": no markup, no control characters
// beyond line breaks, bounded lengths, and links limited to a site-relative
// path or https. The renderer escapes as well, so this is defence in depth
// rather than the only line of defence.
func Validate(d Document) map[string]string {
	errs := map[string]string{}

	// Hero
	requireText(errs, "hero.headline", d.Hero.Headline, MaxHeadline, true)
	requireText(errs, "hero.subtext", d.Hero.Subtext, MaxSubtext, true)
	requireText(errs, "hero.primary_label", d.Hero.PrimaryLabel, MaxLabel, true)
	requireText(errs, "hero.secondary_label", d.Hero.SecondaryLabel, MaxLabel, false)

	// Banner
	if d.Banner.Enabled {
		requireText(errs, "banner.text", d.Banner.Text, MaxBannerText, true)
	} else {
		requireText(errs, "banner.text", d.Banner.Text, MaxBannerText, false)
	}
	if d.Banner.Link != "" {
		if !safeLink(d.Banner.Link) {
			errs["banner.link"] = "Use a site-relative path (starting with /) or an https:// address"
		} else if len(d.Banner.Link) > MaxLink {
			errs["banner.link"] = "Too long"
		}
	}
	if d.Banner.StartsAt != nil && d.Banner.EndsAt != nil && d.Banner.EndsAt.Before(*d.Banner.StartsAt) {
		errs["banner.ends_at"] = "The end time must be after the start time"
	}

	// Ticker
	if len(d.Ticker) > MaxTickerItems {
		errs["ticker"] = "Too many phrases"
	}
	for i, p := range d.Ticker {
		key := "ticker." + itoa(i)
		requireText(errs, key, p, MaxTickerItem, false)
	}

	// How-it-works steps
	if len(d.How.Steps) == 0 {
		errs["how.steps"] = "Add at least one step"
	}
	if len(d.How.Steps) > MaxSteps {
		errs["how.steps"] = "Too many steps"
	}
	for i, s := range d.How.Steps {
		requireText(errs, "how.steps."+itoa(i)+".title", s.Title, MaxStepTitle, true)
		requireText(errs, "how.steps."+itoa(i)+".text", s.Text, MaxStepText, true)
	}

	// Features
	if len(d.Features) == 0 {
		errs["features"] = "Add at least one feature"
	}
	if len(d.Features) > MaxFeatures {
		errs["features"] = "Too many features"
	}
	for i, f := range d.Features {
		if !IconAllowed(f.Icon) {
			errs["features."+itoa(i)+".icon"] = "Choose one of the available illustrations"
		}
		requireText(errs, "features."+itoa(i)+".title", f.Title, MaxLabel, true)
		requireText(errs, "features."+itoa(i)+".text", f.Text, MaxFeatureText, true)
	}

	// FAQs
	if len(d.FAQs) > MaxFAQs {
		errs["faqs"] = "Too many questions"
	}
	for i, q := range d.FAQs {
		requireText(errs, "faqs."+itoa(i)+".question", q.Question, MaxQuestion, true)
		requireText(errs, "faqs."+itoa(i)+".answer", q.Answer, MaxAnswer, true)
	}

	// Footer
	requireText(errs, "footer.text", d.Footer.Text, MaxFooterText, false)
	if d.Footer.SupportEmail != "" {
		if len(d.Footer.SupportEmail) > MaxEmail {
			errs["footer.support_email"] = "Too long"
		} else if _, err := mail.ParseAddress(d.Footer.SupportEmail); err != nil {
			errs["footer.support_email"] = "That is not a valid email address"
		}
	}

	return errs
}

// requireText records a message when a value is missing (if required), too
// long, or contains characters that plain text must not.
func requireText(errs map[string]string, key, value string, max int, required bool) {
	if value == "" {
		if required {
			errs[key] = "This field is required"
		}
		return
	}
	if len(value) > max {
		errs[key] = "Too long"
		return
	}
	if !plainText(value) {
		errs[key] = "Plain text only — no markup or control characters"
	}
}

// plainText reports whether value is safe free text. Line breaks are allowed;
// tabs and other control characters are not, and angle brackets are refused so
// nothing that looks like markup can be stored at all.
func plainText(value string) bool {
	if strings.ContainsAny(value, "<>") {
		return false
	}
	for _, r := range value {
		if r == '\n' {
			continue
		}
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}

// safeLink allows a site-relative path or an https URL and nothing else. It
// blocks javascript:, data:, protocol-relative //host and plain http.
func safeLink(raw string) bool {
	if strings.HasPrefix(raw, "/") && !strings.HasPrefix(raw, "//") {
		return true
	}
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	return u.Scheme == "https" && u.Host != ""
}

// itoa avoids pulling strconv into this file's signature just for field keys.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
