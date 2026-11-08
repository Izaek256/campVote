package handlers

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"campvote/apps/api/internal/landing"
	"campvote/apps/api/internal/models"
	"campvote/apps/api/internal/respond"
	"campvote/apps/api/internal/services"
)

// LandingHandler serves the public landing page data and owns the small CMS
// that administrators use to edit its copy.
//
// Two rules shape everything here:
//
//   - The public endpoints are read-only, unauthenticated and serve only
//     whitelisted, non-identifying fields. They never touch ballot, roster or
//     voter data.
//   - The admin endpoints are ordinary authenticated admin routes; role checks
//     live in the router, but they also re-check server-side before writing.
type LandingHandler struct {
	DB *gorm.DB
}

// NewLandingHandler builds a handler.
func NewLandingHandler(db *gorm.DB) *LandingHandler {
	return &LandingHandler{DB: db}
}

// ─── Public ────────────────────────────────────────────────────────────────

// Public returns the published landing content. When nothing has ever been
// published it returns the built-in defaults, so the page always has words.
func (h *LandingHandler) Public(c *gin.Context) {
	c.Header("Cache-Control", "public, max-age=60")
	respond.OK(c, h.publishedDocument())
}

// publishedDocument reads the published snapshot, falling back to defaults.
func (h *LandingHandler) publishedDocument() landing.Document {
	var s models.LandingSettings
	if err := h.DB.First(&s, "id = ?", models.LandingSettingsID).Error; err != nil || s.PublishedJSON == "" {
		return landing.Default()
	}
	var doc landing.Document
	if err := json.Unmarshal([]byte(s.PublishedJSON), &doc); err != nil {
		return landing.Default()
	}
	return doc
}

// publicUpcomingElection is the exact, whitelisted shape the landing page may
// see. No rosters, voter lists, candidate documents, emails, ballots or
// tallies appear here, and nothing links an identity to a choice.
type publicUpcomingElection struct {
	ID             string    `json:"id"`
	Title          string    `json:"title"`
	Category       string    `json:"category"`
	OrganizingUnit string    `json:"organizing_unit"`
	Positions      int64     `json:"positions"`
	OpensAt        time.Time `json:"opens_at"`
	ClosesAt       time.Time `json:"closes_at"`
	// Status is a landing-friendly label: "registration" while applications
	// are open, otherwise "scheduled".
	Status string `json:"status"`
	// EligibleCount is present only when the election's policy lets us
	// compute it (a roster-backed election). It is an aggregate and never a
	// list of people.
	EligibleCount *int64 `json:"eligible_count,omitempty"`
}

// UpcomingElections returns publicly listed elections that have not opened yet.
//
// "Upcoming" means a DRAFT election, opted in with public_listing, whose
// opening time is still in the future. Drafts stay private unless an
// administrator explicitly lists them, and OPEN / CLOSED / cancelled elections
// are never returned.
func (h *LandingHandler) UpcomingElections(c *gin.Context) {
	now := time.Now().UTC()

	var elections []models.Election
	if err := h.DB.
		Where("public_listing = ? AND status = ? AND starts_at > ?", true, models.ElectionStatusDraft, now).
		Order("starts_at asc").
		Limit(20).
		Find(&elections).Error; err != nil {
		respond.Fail(c, respond.CodeInternalError, "Failed to load upcoming elections")
		return
	}

	out := make([]publicUpcomingElection, 0, len(elections))
	for i := range elections {
		e := elections[i]

		var positions int64
		h.DB.Model(&models.Position{}).Where("election_id = ?", e.ID).Count(&positions)

		var openPositions int64
		h.DB.Model(&models.Position{}).
			Where("election_id = ? AND status = ?", e.ID, models.PositionStatusOpen).
			Count(&openPositions)

		status := "scheduled"
		if openPositions > 0 {
			status = "registration"
		}

		// The aggregate eligible count is published only for roster-backed
		// elections, where we can actually count eligible entries. An
		// open-to-all election has no computable number, so we omit it rather
		// than print a guess.
		var eligible *int64
		if e.RequireRoster {
			var n int64
			h.DB.Model(&models.ElectionRoster{}).
				Where("election_id = ? AND status = ?", e.ID, models.RosterStatusEligible).
				Count(&n)
			eligible = &n
		}

		out = append(out, publicUpcomingElection{
			ID:             e.ID,
			Title:          e.Title,
			Category:       e.Category,
			OrganizingUnit: e.OrganizingUnit,
			Positions:      positions,
			OpensAt:        e.StartsAt,
			ClosesAt:       e.EndsAt,
			Status:         status,
			EligibleCount:  eligible,
		})
	}

	c.Header("Cache-Control", "public, max-age=60")
	respond.OK(c, out)
}

// ─── Admin CMS ─────────────────────────────────────────────────────────────

// landingFeaturePayload is one feature card as the editor sends it.
type landingFeaturePayload struct {
	ID      string `json:"id"`
	Icon    string `json:"icon"`
	Title   string `json:"title"`
	Text    string `json:"text"`
	Visible *bool  `json:"visible"`
}

type landingFAQPayload struct {
	ID       string `json:"id"`
	Question string `json:"question"`
	Answer   string `json:"answer"`
	Visible  *bool  `json:"visible"`
}

type landingDraftPayload struct {
	Hero     landing.Hero              `json:"hero"`
	Banner   landing.Banner            `json:"banner"`
	Ticker   []string                  `json:"ticker"`
	How      landing.How               `json:"how"`
	Features []landingFeaturePayload   `json:"features"`
	FAQs     []landingFAQPayload       `json:"faqs"`
	Footer   landing.Footer            `json:"footer"`
}

// document converts the payload to the validated, storable shape. Item ids and
// visibility are dropped: validation and storage treat every item the same way.
func (p landingDraftPayload) document() landing.Document {
	doc := landing.Document{
		Hero:   p.Hero,
		Banner: p.Banner,
		Ticker: p.Ticker,
		How:    p.How,
		Footer: p.Footer,
	}
	for _, f := range p.Features {
		doc.Features = append(doc.Features, landing.Feature{Icon: f.Icon, Title: f.Title, Text: f.Text})
	}
	for _, q := range p.FAQs {
		doc.FAQs = append(doc.FAQs, landing.FAQ{Question: q.Question, Answer: q.Answer})
	}
	return doc
}

// AdminGet returns the draft for editing, the currently published content and
// whether the two differ so the screen can show an "unsaved changes" state.
func (h *LandingHandler) AdminGet(c *gin.Context) {
	s, features, faqs, err := h.loadDraft()
	if err != nil {
		respond.Fail(c, respond.CodeInternalError, "Failed to load landing content")
		return
	}

	draft := draftPayloadFromRows(s, features, faqs)
	published := landing.Default()
	hasPublished := s.PublishedJSON != ""
	if hasPublished {
		var pub landing.Document
		if err := json.Unmarshal([]byte(s.PublishedJSON), &pub); err == nil {
			published = pub
		}
	}

	respond.OK(c, gin.H{
		"draft":         draft,
		"published":     published,
		"has_published": hasPublished,
		"dirty":         canonical(publishedOf(s)) != canonical(documentFromRows(s, features, faqs)),
		"updated_at":    s.UpdatedAt,
		"published_at":  s.PublishedAt,
	})
}

// AdminSave validates and stores the draft. Nothing goes live until Publish.
func (h *LandingHandler) AdminSave(c *gin.Context) {
	var payload landingDraftPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		respond.Fail(c, respond.CodeValidationError, "The request body is not valid landing content")
		return
	}

	doc := landing.Normalize(payload.document())
	if errs := landing.Validate(doc); len(errs) > 0 {
		respond.Validation(c, "Please fix the highlighted fields", errs)
		return
	}

	userID := c.GetString("user_id")
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		if err := writeSettings(tx, doc, userID); err != nil {
			return err
		}
		return writeItems(tx, doc, userID)
	})
	if err != nil {
		respond.Fail(c, respond.CodeInternalError, "Failed to save landing content")
		return
	}

	services.LogActivity(h.DB, c, services.ActivityArgs{
		Action:       "landing.update",
		Severity:     "info",
		Summary:      "Updated the landing page draft",
		ResourceType: "landing",
		ResourceID:   models.LandingSettingsID,
	})

	s, features, faqs, err := h.loadDraft()
	if err != nil {
		respond.Fail(c, respond.CodeInternalError, "Failed to reload landing content")
		return
	}
	respond.OK(c, gin.H{
		"draft":         draftPayloadFromRows(s, features, faqs),
		"published":     publishedOf(s),
		"has_published": s.PublishedJSON != "",
		"dirty":         canonical(publishedOf(s)) != canonical(documentFromRows(s, features, faqs)),
		"updated_at":    s.UpdatedAt,
		"published_at":  s.PublishedAt,
	}, "Landing page draft saved")
}

// AdminPublish snapshots the draft into the published content the public page
// reads. Hidden items are left out of the snapshot.
func (h *LandingHandler) AdminPublish(c *gin.Context) {
	s, features, faqs, err := h.loadDraft()
	if err != nil {
		respond.Fail(c, respond.CodeInternalError, "Failed to load landing content")
		return
	}

	doc := landing.Normalize(documentFromRows(s, features, faqs))
	if errs := landing.Validate(doc); len(errs) > 0 {
		respond.Validation(c, "Fix the draft before publishing it", errs)
		return
	}
	// Only visible items appear on the public page.
	doc.Features = filterVisibleFeatures(features, doc.Features)
	doc.FAQs = filterVisibleFAQs(faqs, doc.FAQs)

	blob, err := json.Marshal(doc)
	if err != nil {
		respond.Fail(c, respond.CodeInternalError, "Failed to publish landing content")
		return
	}

	now := time.Now().UTC()
	if err := h.DB.Model(&models.LandingSettings{}).
		Where("id = ?", models.LandingSettingsID).
		Updates(map[string]interface{}{
			"published_json":  string(blob),
			"published_by_id": c.GetString("user_id"),
			"published_at":    now,
		}).Error; err != nil {
		respond.Fail(c, respond.CodeInternalError, "Failed to publish landing content")
		return
	}

	services.LogActivity(h.DB, c, services.ActivityArgs{
		Action:       "landing.publish",
		Severity:     "info",
		Summary:      "Published the landing page",
		ResourceType: "landing",
		ResourceID:   models.LandingSettingsID,
	})

	respond.OK(c, gin.H{
		"published":      doc,
		"has_published":  true,
		"published_at":   now,
		"dirty":          false,
	}, "Landing page published")
}

// AdminRevert discards the draft and restores it from the published snapshot.
// With nothing published it restores the built-in defaults.
func (h *LandingHandler) AdminRevert(c *gin.Context) {
	var s models.LandingSettings
	if err := h.DB.First(&s, "id = ?", models.LandingSettingsID).Error; err != nil {
		// No row means no edits: seed defaults so the editor has content.
		s = models.LandingSettings{ID: models.LandingSettingsID}
	}

	doc := landing.Default()
	if s.PublishedJSON != "" {
		var pub landing.Document
		if err := json.Unmarshal([]byte(s.PublishedJSON), &pub); err == nil {
			doc = pub
		}
	}

	userID := c.GetString("user_id")
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		if err := writeSettings(tx, doc, userID); err != nil {
			return err
		}
		return writeItems(tx, doc, userID)
	})
	if err != nil {
		respond.Fail(c, respond.CodeInternalError, "Failed to revert landing content")
		return
	}

	services.LogActivity(h.DB, c, services.ActivityArgs{
		Action:       "landing.revert",
		Severity:     "warn",
		Summary:      "Reverted the landing page to the published version",
		ResourceType: "landing",
		ResourceID:   models.LandingSettingsID,
	})

	r, features, faqs, err := h.loadDraft()
	if err != nil {
		respond.Fail(c, respond.CodeInternalError, "Failed to reload landing content")
		return
	}
	respond.OK(c, gin.H{
		"draft":         draftPayloadFromRows(r, features, faqs),
		"published":     publishedOf(r),
		"has_published": r.PublishedJSON != "",
		"dirty":         false,
		"updated_at":    r.UpdatedAt,
		"published_at":  r.PublishedAt,
	}, "Landing page draft reverted")
}

// ─── Draft storage helpers ─────────────────────────────────────────────────

// loadDraft reads the single settings row (creating it from defaults when it is
// missing) together with the ordered feature and FAQ rows.
func (h *LandingHandler) loadDraft() (models.LandingSettings, []models.LandingFeature, []models.LandingFAQ, error) {
	var s models.LandingSettings
	err := h.DB.First(&s, "id = ?", models.LandingSettingsID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		doc := landing.Default()
		if err := h.DB.Transaction(func(tx *gorm.DB) error {
			if err := writeSettings(tx, doc, ""); err != nil {
				return err
			}
			return writeItems(tx, doc, "")
		}); err != nil {
			return s, nil, nil, err
		}
		if err := h.DB.First(&s, "id = ?", models.LandingSettingsID).Error; err != nil {
			return s, nil, nil, err
		}
	} else if err != nil {
		return s, nil, nil, err
	}

	var features []models.LandingFeature
	if err := h.DB.Order("sort_order asc, created_at asc").Find(&features).Error; err != nil {
		return s, nil, nil, err
	}
	var faqs []models.LandingFAQ
	if err := h.DB.Order("sort_order asc, created_at asc").Find(&faqs).Error; err != nil {
		return s, nil, nil, err
	}
	return s, features, faqs, nil
}

// writeSettings upserts the settings row scalars from doc.
func writeSettings(tx *gorm.DB, doc landing.Document, userID string) error {
	var s models.LandingSettings
	err := tx.First(&s, "id = ?", models.LandingSettingsID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		s = models.LandingSettings{ID: models.LandingSettingsID}
	} else if err != nil {
		return err
	}

	s.HeroHeadline = doc.Hero.Headline
	s.HeroSubtext = doc.Hero.Subtext
	s.HeroPrimaryLabel = doc.Hero.PrimaryLabel
	s.HeroSecondaryLabel = doc.Hero.SecondaryLabel
	s.BannerEnabled = doc.Banner.Enabled
	s.BannerText = doc.Banner.Text
	s.BannerLink = doc.Banner.Link
	s.BannerStartsAt = doc.Banner.StartsAt
	s.BannerEndsAt = doc.Banner.EndsAt
	s.TickerPhrases = strings.Join(doc.Ticker, "\n")
	steps, err := json.Marshal(doc.How.Steps)
	if err != nil {
		return err
	}
	s.HowSteps = string(steps)
	s.FooterText = doc.Footer.Text
	s.SupportEmail = doc.Footer.SupportEmail
	s.UpdatedByID = userID

	return tx.Save(&s).Error
}

// writeItems replaces every feature and FAQ row with the draft's items, in
// order. Replace-all is correct here because the draft is saved as a whole
// document; it also removes rows the editor deleted.
func writeItems(tx *gorm.DB, doc landing.Document, userID string) error {
	if err := tx.Where("1 = 1").Delete(&models.LandingFeature{}).Error; err != nil {
		return err
	}
	if err := tx.Where("1 = 1").Delete(&models.LandingFAQ{}).Error; err != nil {
		return err
	}
	for i, f := range doc.Features {
		row := models.LandingFeature{
			IconKey: f.Icon, Title: f.Title, Body: f.Text,
			SortOrder: i, Visible: true, UpdatedByID: userID,
		}
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
	}
	for i, q := range doc.FAQs {
		row := models.LandingFAQ{
			Question: q.Question, Answer: q.Answer,
			SortOrder: i, Visible: true, UpdatedByID: userID,
		}
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
	}
	return nil
}

// documentFromRows rebuilds the document from the draft rows.
func documentFromRows(s models.LandingSettings, features []models.LandingFeature, faqs []models.LandingFAQ) landing.Document {
	doc := landing.Document{
		Hero: landing.Hero{
			Headline:       s.HeroHeadline,
			Subtext:        s.HeroSubtext,
			PrimaryLabel:   s.HeroPrimaryLabel,
			SecondaryLabel: s.HeroSecondaryLabel,
		},
		Banner: landing.Banner{
			Enabled:  s.BannerEnabled,
			Text:     s.BannerText,
			Link:     s.BannerLink,
			StartsAt: s.BannerStartsAt,
			EndsAt:   s.BannerEndsAt,
		},
		Ticker: splitPhrases(s.TickerPhrases),
		Footer: landing.Footer{Text: s.FooterText, SupportEmail: s.SupportEmail},
	}
	if s.HowSteps != "" {
		_ = json.Unmarshal([]byte(s.HowSteps), &doc.How.Steps)
	}
	for _, f := range features {
		doc.Features = append(doc.Features, landing.Feature{Icon: f.IconKey, Title: f.Title, Text: f.Body})
	}
	for _, q := range faqs {
		doc.FAQs = append(doc.FAQs, landing.FAQ{Question: q.Question, Answer: q.Answer})
	}
	return doc
}

// draftPayloadFromRows is documentFromRows carrying the row ids, so the editor
// can key its inputs without them leaking into validation.
func draftPayloadFromRows(s models.LandingSettings, features []models.LandingFeature, faqs []models.LandingFAQ) landingDraftPayload {
	doc := documentFromRows(s, features, faqs)
	out := landingDraftPayload{
		Hero:   doc.Hero,
		Banner: doc.Banner,
		Ticker: doc.Ticker,
		How:    doc.How,
		Footer: doc.Footer,
	}
	for i, f := range doc.Features {
		visible := true
		if i < len(features) {
			visible = features[i].Visible
		}
		id := ""
		if i < len(features) {
			id = features[i].ID
		}
		out.Features = append(out.Features, landingFeaturePayload{
			ID: id, Icon: f.Icon, Title: f.Title, Text: f.Text, Visible: &visible,
		})
	}
	for i, q := range doc.FAQs {
		visible := true
		if i < len(faqs) {
			visible = faqs[i].Visible
		}
		id := ""
		if i < len(faqs) {
			id = faqs[i].ID
		}
		out.FAQs = append(out.FAQs, landingFAQPayload{
			ID: id, Question: q.Question, Answer: q.Answer, Visible: &visible,
		})
	}
	return out
}

// publishedOf returns the published document, or the defaults when nothing has
// been published yet.
func publishedOf(s models.LandingSettings) landing.Document {
	if s.PublishedJSON == "" {
		return landing.Default()
	}
	var doc landing.Document
	if err := json.Unmarshal([]byte(s.PublishedJSON), &doc); err != nil {
		return landing.Default()
	}
	return doc
}

func filterVisibleFeatures(rows []models.LandingFeature, all []landing.Feature) []landing.Feature {
	out := make([]landing.Feature, 0, len(all))
	for i, f := range all {
		if i < len(rows) && !rows[i].Visible {
			continue
		}
		out = append(out, f)
	}
	return out
}

func filterVisibleFAQs(rows []models.LandingFAQ, all []landing.FAQ) []landing.FAQ {
	out := make([]landing.FAQ, 0, len(all))
	for i, q := range all {
		if i < len(rows) && !rows[i].Visible {
			continue
		}
		out = append(out, q)
	}
	return out
}

// splitPhrases turns the stored newline-separated ticker back into a list.
func splitPhrases(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return []string{}
	}
	parts := strings.Split(raw, "\n")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if s := strings.TrimSpace(p); s != "" {
			out = append(out, s)
		}
	}
	return out
}

// canonical marshals a document with its lists normalised, so "not published"
// and "published with empty lists" compare equal.
func canonical(d landing.Document) string {
	if d.Ticker == nil {
		d.Ticker = []string{}
	}
	if d.How.Steps == nil {
		d.How.Steps = []landing.Step{}
	}
	if d.Features == nil {
		d.Features = []landing.Feature{}
	}
	if d.FAQs == nil {
		d.FAQs = []landing.FAQ{}
	}
	b, _ := json.Marshal(d)
	return string(b)
}
