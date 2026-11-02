package models

import (
	"time"

	"campvote/apps/api/internal/ids"
	"gorm.io/gorm"
)

// Question / contest types (FR-BALLOT-03)
const (
	QuestionTypeSingleChoice = "SINGLE_CHOICE"
	QuestionTypeMultiChoice  = "MULTI_CHOICE"
	QuestionTypeYesNo        = "YES_NO"
	QuestionTypeReferendum   = "REFERENDUM"
)

// Position application lifecycle. A position opens for applications as soon as
// it is created and is closed by an administrator once the candidate list is
// settled (or earlier, if they choose).
const (
	PositionStatusOpen   = "OPEN"
	PositionStatusClosed = "CLOSED"
)

// Position represents an elected office, contest, or ballot proposition.
//
// Positions are created first and linked to an election afterwards: a position
// can collect applications on its own, and only once its candidates are chosen
// does an administrator attach it to the election it will appear on.
type Position struct {
	ID string `gorm:"primarykey;size:36" json:"id"`
	// ElectionID is optional. A position created before its election has no
	// election id yet; SetElection attaches it later.
	ElectionID  *string `gorm:"size:36;index" json:"election_id,omitempty"`
	Title       string  `gorm:"size:255;not null" json:"title" binding:"required"`
	Description string  `gorm:"type:text" json:"description"`
	// Requirements is the eligibility/experience text shown to applicants
	// before they apply and to officials while vetting.
	Requirements string `gorm:"type:text" json:"requirements"`
	QuestionType string `gorm:"size:50;not null;default:'SINGLE_CHOICE'" json:"question_type"`

	// Status is the application window: OPEN collects applications, CLOSED
	// stops them. It is independent of the election's own lifecycle.
	Status string `gorm:"size:50;not null;default:'OPEN';index" json:"status"`

	// MaxCandidates caps how many applications may be approved for this
	// position. 0 means no cap.
	MaxCandidates int `gorm:"not null;default:0" json:"max_candidates"`

	// Eligibility rules for who may stand for this position. The same shape as
	// an election's voter eligibility: a required email domain, plus optional
	// year and course allow-lists. Empty year/course lists mean "any".
	AllowedDomain  string `gorm:"size:255" json:"allowed_domain"`
	AllowedYears   string `gorm:"size:255" json:"allowed_years"`
	AllowedCourses string `gorm:"size:255" json:"allowed_courses"`

	// Rules for ballot validity (FR-BALLOT-05)
	MinSelections int  `gorm:"not null;default:1" json:"min_selections"`
	MaxSelections int  `gorm:"not null;default:1" json:"max_selections"`
	Seats         int  `gorm:"not null;default:1" json:"seats"`
	AllowBlank    bool `gorm:"default:false" json:"allow_blank"`
	DisplayOrder  int  `gorm:"not null;default:0" json:"display_order"`

	CreatedAt time.Time      `gorm:"index" json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Associations
	Candidates []Candidate `gorm:"foreignKey:PositionID;constraint:OnDelete:CASCADE" json:"candidates,omitempty"`
}

// BeforeCreate assigns a new UUID and fills in the defaults a zero value would
// otherwise leave as an empty string.
func (p *Position) BeforeCreate(tx *gorm.DB) error {
	if p.ID == "" {
		p.ID = ids.New()
	}
	if p.QuestionType == "" {
		p.QuestionType = QuestionTypeSingleChoice
	}
	if p.Status == "" {
		p.Status = PositionStatusOpen
	}
	if p.MaxSelections == 0 {
		p.MaxSelections = 1
	}
	if p.Seats == 0 {
		p.Seats = 1
	}
	return nil
}

// SetElection attaches the position to an election, or detaches it when id is
// empty. It is the only write path for the nullable column, so callers never
// have to take the address of a local themselves.
func (p *Position) SetElection(id string) {
	if id == "" {
		p.ElectionID = nil
		return
	}
	p.ElectionID = &id
}

// ElectionRef returns the linked election id, or "" when the position is not
// attached to an election yet.
func (p *Position) ElectionRef() string {
	if p.ElectionID == nil {
		return ""
	}
	return *p.ElectionID
}

// AcceptingApplications reports whether the position's application window is open.
func (p *Position) AcceptingApplications() bool {
	return p.Status == PositionStatusOpen
}
