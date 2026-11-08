package models

import (
	"time"

	"campvote/apps/api/internal/ids"

	"gorm.io/gorm"
)

// LandingSettingsID is the fixed primary key of the single landing-settings
// row. A landing page has exactly one set of words, so the table holds one row
// and the id makes that explicit rather than relying on "the only row".
const LandingSettingsID = "landing"

// LandingSettings is the editable copy of the public landing page: the hero,
// the announcement banner, the ticker, the "How it works" steps and the footer.
//
// It is the DRAFT. The public endpoint never reads these columns; it reads
// PublishedJSON, which Publish fills from the draft. That is what lets an
// administrator edit and preview without the change going live.
type LandingSettings struct {
	ID string `gorm:"primarykey;size:36" json:"id"`

	HeroHeadline       string `gorm:"type:text" json:"hero_headline"`
	HeroSubtext        string `gorm:"type:text" json:"hero_subtext"`
	HeroPrimaryLabel   string `gorm:"size:100" json:"hero_primary_label"`
	HeroSecondaryLabel string `gorm:"size:100" json:"hero_secondary_label"`

	BannerEnabled  bool       `gorm:"default:false" json:"banner_enabled"`
	BannerText     string     `gorm:"type:text" json:"banner_text"`
	BannerLink     string     `gorm:"size:500" json:"banner_link"`
	BannerStartsAt *time.Time `json:"banner_starts_at,omitempty"`
	BannerEndsAt   *time.Time `json:"banner_ends_at,omitempty"`

	// TickerPhrases is one phrase per line.
	TickerPhrases string `gorm:"type:text" json:"ticker_phrases"`
	// HowSteps is a JSON array of {title,text}.
	HowSteps string `gorm:"type:text" json:"how_steps"`

	FooterText   string `gorm:"type:text" json:"footer_text"`
	SupportEmail string `gorm:"size:254" json:"support_email"`

	// PublishedJSON is the last published snapshot of the WHOLE landing
	// document (settings + features + faqs). Empty until the first publish, so
	// the public endpoint falls back to the built-in defaults.
	PublishedJSON string     `gorm:"type:text" json:"-"`
	PublishedByID string     `gorm:"size:36" json:"published_by_id,omitempty"`
	PublishedAt   *time.Time `json:"published_at,omitempty"`

	UpdatedByID string    `gorm:"size:36" json:"updated_by_id,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// BeforeCreate pins the single-row id so callers never generate a second row.
func (s *LandingSettings) BeforeCreate(tx *gorm.DB) error {
	if s.ID == "" {
		s.ID = LandingSettingsID
	}
	return nil
}

// LandingFeature is one card in the draft "Features" grid.
type LandingFeature struct {
	ID          string    `gorm:"primarykey;size:36" json:"id"`
	IconKey     string    `gorm:"size:50;not null" json:"icon_key"`
	Title       string    `gorm:"size:200;not null" json:"title"`
	Body        string    `gorm:"type:text" json:"body"`
	SortOrder   int       `gorm:"not null;default:0;index" json:"sort_order"`
	Visible     bool      `gorm:"default:true" json:"visible"`
	UpdatedByID string    `gorm:"size:36" json:"updated_by_id,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// BeforeCreate assigns a UUID.
func (f *LandingFeature) BeforeCreate(tx *gorm.DB) error {
	if f.ID == "" {
		f.ID = ids.New()
	}
	return nil
}

// LandingFAQ is one draft question in the FAQ list.
type LandingFAQ struct {
	ID          string    `gorm:"primarykey;size:36" json:"id"`
	Question    string    `gorm:"size:300;not null" json:"question"`
	Answer      string    `gorm:"type:text" json:"answer"`
	SortOrder   int       `gorm:"not null;default:0;index" json:"sort_order"`
	Visible     bool      `gorm:"default:true" json:"visible"`
	UpdatedByID string    `gorm:"size:36" json:"updated_by_id,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// BeforeCreate assigns a UUID.
func (f *LandingFAQ) BeforeCreate(tx *gorm.DB) error {
	if f.ID == "" {
		f.ID = ids.New()
	}
	return nil
}
