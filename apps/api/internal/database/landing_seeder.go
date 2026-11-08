package database

import (
	"encoding/json"
	"log"
	"strings"

	"gorm.io/gorm"

	"campvote/apps/api/internal/landing"
	"campvote/apps/api/internal/models"
)

// SeedLanding fills the landing-page CMS with its built-in defaults so a fresh
// install has real content to edit. It only runs on an empty table: landing
// copy is meant to be edited by hand, and re-seeding would silently discard an
// administrator's work.
//
// It seeds the DRAFT only. The public endpoint already falls back to the same
// defaults when nothing has been published, so seeding a published snapshot
// would add a second copy of the same words with nothing to gain.
func SeedLanding(db *gorm.DB) error {
	var count int64
	if err := db.Model(&models.LandingSettings{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		log.Println("Landing content already seeded, skipping...")
		return nil
	}

	doc := landing.Default()
	steps, err := json.Marshal(doc.How.Steps)
	if err != nil {
		return err
	}

	return db.Transaction(func(tx *gorm.DB) error {
		settings := models.LandingSettings{
			ID:                 models.LandingSettingsID,
			HeroHeadline:       doc.Hero.Headline,
			HeroSubtext:        doc.Hero.Subtext,
			HeroPrimaryLabel:   doc.Hero.PrimaryLabel,
			HeroSecondaryLabel: doc.Hero.SecondaryLabel,
			BannerEnabled:      doc.Banner.Enabled,
			BannerText:         doc.Banner.Text,
			BannerLink:         doc.Banner.Link,
			TickerPhrases:      strings.Join(doc.Ticker, "\n"),
			HowSteps:           string(steps),
			FooterText:         doc.Footer.Text,
			SupportEmail:       doc.Footer.SupportEmail,
		}
		if err := tx.Create(&settings).Error; err != nil {
			return err
		}

		for i, f := range doc.Features {
			row := models.LandingFeature{
				IconKey: f.Icon, Title: f.Title, Body: f.Text,
				SortOrder: i, Visible: true,
			}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		}
		for i, q := range doc.FAQs {
			row := models.LandingFAQ{
				Question: q.Question, Answer: q.Answer,
				SortOrder: i, Visible: true,
			}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
