package models

import (
	"time"

	"gorm.io/gorm"

	"campvote/apps/api/internal/ids"
)

// EligibilityImportBatch is the provenance record for one committed roster
// import. It records who imported what and the aggregate outcome; the roster
// rows themselves point back at it through ImportBatchID. The CSV content is
// not stored, only its hash.
type EligibilityImportBatch struct {
	ID           string    `gorm:"primarykey;size:36" json:"id"`
	ElectionID   string    `gorm:"size:36;not null;index" json:"election_id"`
	ImportedByID string    `gorm:"size:36;not null;index" json:"imported_by_id"`
	Source       string    `gorm:"size:20;not null;default:'CSV_IMPORT'" json:"source"`
	FileName     string    `gorm:"size:255" json:"file_name"`
	ContentHash  string    `gorm:"size:64;index" json:"content_hash"`
	Total        int       `json:"total"`
	Created      int       `json:"created"`
	Updated      int       `json:"updated"`
	Revoked      int       `json:"revoked"`
	CreatedAt    time.Time `gorm:"index" json:"created_at"`
}

// BeforeCreate assigns an opaque id.
func (b *EligibilityImportBatch) BeforeCreate(tx *gorm.DB) error {
	if b.ID == "" {
		b.ID = ids.New()
	}
	if b.Source == "" {
		b.Source = RosterSourceCSVImport
	}
	return nil
}
