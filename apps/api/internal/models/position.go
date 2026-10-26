package models

import (
	"time"

	"gorm.io/gorm"
	"campvote/apps/api/internal/ids"
)

// Question / contest types (FR-BALLOT-03)
const (
	QuestionTypeSingleChoice = "SINGLE_CHOICE"
	QuestionTypeMultiChoice  = "MULTI_CHOICE"
	QuestionTypeYesNo        = "YES_NO"
	QuestionTypeReferendum   = "REFERENDUM"
)

// Position represents an elected office, contest, or ballot proposition.
type Position struct {
	ID            string         `gorm:"primarykey;size:36" json:"id"`
	ElectionID    string         `gorm:"size:36;not null;index" json:"election_id"`
	Title         string         `gorm:"size:255;not null" json:"title" binding:"required"`
	Description   string         `gorm:"type:text" json:"description"`
	QuestionType  string         `gorm:"size:50;not null;default:'SINGLE_CHOICE'" json:"question_type"`
	
	// Rules for ballot validity (FR-BALLOT-05)
	MinSelections int            `gorm:"not null;default:1" json:"min_selections"`
	MaxSelections int            `gorm:"not null;default:1" json:"max_selections"`
	Seats         int            `gorm:"not null;default:1" json:"seats"`
	AllowBlank    bool           `gorm:"default:false" json:"allow_blank"`
	DisplayOrder  int            `gorm:"not null;default:0" json:"display_order"`
	
	CreatedAt     time.Time      `gorm:"index" json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`

	// Associations
	Candidates    []Candidate    `gorm:"foreignKey:PositionID;constraint:OnDelete:CASCADE" json:"candidates,omitempty"`
}

// BeforeCreate assigns a new UUID.
func (p *Position) BeforeCreate(tx *gorm.DB) error {
	if p.ID == "" {
		p.ID = ids.New()
	}
	if p.QuestionType == "" {
		p.QuestionType = QuestionTypeSingleChoice
	}
	if p.MaxSelections == 0 {
		p.MaxSelections = 1
	}
	if p.Seats == 0 {
		p.Seats = 1
	}
	return nil
}
