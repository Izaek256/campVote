// Package landing owns the shape, the safe defaults and the validation of the
// editable copy on the public landing page ("/").
//
// It is deliberately small. The landing page has a fixed layout in code; only
// the words are editable, and only through plain text. Security claims and the
// illustrations stay in code so an administrator can never accidentally
// publish a false security statement or inject markup.
package landing

import (
	"strings"
	"time"
)

// IconKeys is the allowlist of feature illustrations the admin may choose from.
// Each key names one of the eight animated illustrations that ship in the web
// app. A feature whose icon is not in this list is rejected, so the CMS can
// never point at an illustration that does not exist.
var IconKeys = []string{
	"elections",
	"applications",
	"builder",
	"eligibility",
	"roles",
	"turnout",
	"results",
	"notifications",
}

// Document is the whole editable landing-page content. It is what the public
// endpoint serves and what the admin editor reads and writes.
type Document struct {
	Hero     Hero      `json:"hero"`
	Banner   Banner    `json:"banner"`
	Ticker   []string  `json:"ticker"`
	How      How       `json:"how"`
	Features []Feature `json:"features"`
	FAQs     []FAQ     `json:"faqs"`
	Footer   Footer    `json:"footer"`
}

// Hero is the top-of-page headline block.
type Hero struct {
	Headline       string `json:"headline"`
	Subtext        string `json:"subtext"`
	PrimaryLabel   string `json:"primary_label"`
	SecondaryLabel string `json:"secondary_label"`
}

// Banner is the announcement strip above the navigation. When either bound is
// set the banner only shows inside that window.
type Banner struct {
	Enabled  bool       `json:"enabled"`
	Text     string     `json:"text"`
	Link     string     `json:"link"`
	StartsAt *time.Time `json:"starts_at,omitempty"`
	EndsAt   *time.Time `json:"ends_at,omitempty"`
}

// Step is one card in the "How it works" strip.
type Step struct {
	Title string `json:"title"`
	Text  string `json:"text"`
}

// How is the "How it works" section.
type How struct {
	Steps []Step `json:"steps"`
}

// Feature is one card in the "Features" grid. Icon is a key from IconKeys.
type Feature struct {
	Icon  string `json:"icon"`
	Title string `json:"title"`
	Text  string `json:"text"`
}

// FAQ is one item in the question list.
type FAQ struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

// Footer is the closing line and the support address.
type Footer struct {
	Text         string `json:"text"`
	SupportEmail string `json:"support_email"`
}

// Field limits. Small on purpose: a landing page is a shop window, not a wiki,
// and a bounded payload keeps the free-text fields from becoming a storage
// problem or an injection surface.
const (
	MaxHeadline    = 160
	MaxSubtext     = 400
	MaxLabel       = 60
	MaxBannerText  = 200
	MaxLink        = 500
	MaxTickerItems = 12
	MaxTickerItem  = 60
	MaxSteps       = 6
	MaxStepTitle   = 80
	MaxStepText    = 200
	MaxFeatures    = 12
	MaxFeatureText = 200
	MaxFAQs        = 20
	MaxQuestion    = 200
	MaxAnswer      = 1000
	MaxFooterText  = 300
	MaxEmail       = 254
)

// Default returns the landing content the site ships with. It mirrors the
// original prototype, so a fresh install and a database with nothing published
// both render a complete, honest page rather than a blank one.
//
// The figures the prototype showed as "Eligible voters", "Ballots accepted"
// and "Turnout" are deliberately absent: they were sample data, and this page
// must never print a number it cannot source from the running election.
func Default() Document {
	return Document{
		Hero: Hero{
			// The asterisks are the hero's only formatting: the web renderer
			// colours that word green and interprets nothing else.
			Headline:       "Vote once.\nKeep it *secret.*\nTrust the result.",
			Subtext:        "campVote runs guild and campus elections end to end: eligibility, secret ballots, live turnout, and certified results, on any phone.",
			PrimaryLabel:   "Sign in with campus Google",
			SecondaryLabel: "See how voting works",
		},
		Banner: Banner{
			Enabled: true,
			Text:    "Voting is open · check the elections you are eligible for",
			Link:    "/elections",
		},
		Ticker: []string{
			"One voter", "One ballot", "Secret by design", "Auditable",
			"Mobile-first", "Certified results",
		},
		How: How{
			Steps: []Step{
				{Title: "Sign in", Text: "Use your campus Google account. Staff use a separate sign-in."},
				{Title: "We check eligibility", Text: "Your place on the voter roster is verified for each election."},
				{Title: "Choose", Text: "Pick one candidate per position, or leave a position blank."},
				{Title: "Confirm and keep your receipt", Text: "Your ballot is recorded. The receipt never shows your choices."},
			},
		},
		Features: []Feature{
			{Icon: "elections", Title: "Many elections at once", Text: "Guild, faculty, class or club polls run side by side, each with its own roster and results."},
			{Icon: "applications", Title: "Candidate applications", Text: "Upload documents, get reviewed and approved by administrators, then appear on the ballot."},
			{Icon: "builder", Title: "Custom ballot builder", Text: "Positions, options and instructions configured per election, then versioned and locked."},
			{Icon: "eligibility", Title: "Eligibility rules", Text: "Restrict by faculty, programme, year or an imported roster. Only eligible voters ever see the ballot."},
			{Icon: "roles", Title: "Clear roles", Text: "Administrators, election officers, candidates, observers and voters each see only their part."},
			{Icon: "turnout", Title: "Live turnout", Text: "Watch participation by faculty as voting runs, with results hidden until the policy says otherwise."},
			{Icon: "results", Title: "Certified results and PDFs", Text: "Review, certify and export a signed-off results report, plus an information sheet for each election."},
			{Icon: "notifications", Title: "Email notifications", Text: "Codes, invitations, reminders and receipts, sent in a consistent campVote style."},
		},
		FAQs: []FAQ{
			{Question: "Can anyone see how I voted?", Answer: "No. Officials, candidates and administrators cannot see individual choices. The system stores that you voted separately from the ballot itself."},
			{Question: "Who can vote?", Answer: "Students on the voter roster for that election, signed in with their campus Google account. Eligibility is checked on the server every time."},
			{Question: "When are results published?", Answer: "Each election has its own visibility policy. Final results are shown after review and certification by the election administrators."},
			{Question: "How do administrators sign in?", Answer: "Administrators have a separate staff sign-in reachable from the footer."},
		},
		Footer: Footer{
			Text:         "A university project. Built for campus elections.",
			SupportEmail: "",
		},
	}
}

// IconAllowed reports whether key is one of the eight illustrations.
func IconAllowed(key string) bool {
	for _, k := range IconKeys {
		if k == key {
			return true
		}
	}
	return false
}

// Normalize trims surrounding whitespace on every text field so a value the
// admin typed with trailing spaces compares equal to the same value without
// them. It never silently drops content.
func Normalize(d Document) Document {
	d.Hero.Headline = strings.TrimSpace(d.Hero.Headline)
	d.Hero.Subtext = strings.TrimSpace(d.Hero.Subtext)
	d.Hero.PrimaryLabel = strings.TrimSpace(d.Hero.PrimaryLabel)
	d.Hero.SecondaryLabel = strings.TrimSpace(d.Hero.SecondaryLabel)
	d.Banner.Text = strings.TrimSpace(d.Banner.Text)
	d.Banner.Link = strings.TrimSpace(d.Banner.Link)
	d.Footer.Text = strings.TrimSpace(d.Footer.Text)
	d.Footer.SupportEmail = strings.TrimSpace(d.Footer.SupportEmail)

	for i := range d.Ticker {
		d.Ticker[i] = strings.TrimSpace(d.Ticker[i])
	}
	for i := range d.How.Steps {
		d.How.Steps[i].Title = strings.TrimSpace(d.How.Steps[i].Title)
		d.How.Steps[i].Text = strings.TrimSpace(d.How.Steps[i].Text)
	}
	for i := range d.Features {
		d.Features[i].Title = strings.TrimSpace(d.Features[i].Title)
		d.Features[i].Text = strings.TrimSpace(d.Features[i].Text)
	}
	for i := range d.FAQs {
		d.FAQs[i].Question = strings.TrimSpace(d.FAQs[i].Question)
		d.FAQs[i].Answer = strings.TrimSpace(d.FAQs[i].Answer)
	}
	return d
}
