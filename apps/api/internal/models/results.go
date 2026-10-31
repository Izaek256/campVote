package models

import (
	"time"

	"campvote/apps/api/internal/ids"
	"gorm.io/gorm"
)

// Result status labels. A snapshot moves forward only: PROVISIONAL is
// superseded by a correction, and CERTIFIED is pinned to its content hash.
const (
	ResultStatusProvisional = "PROVISIONAL"
	ResultStatusCertified   = "CERTIFIED"
	ResultStatusSuperseded  = "SUPERSEDED"
)

// ResultSnapshot is one immutable, reproducible tally of an election.
//
// It exists so certification pins a specific count rather than a mutable table,
// so a recount can be compared to what was certified, and so a correction is a
// new record that keeps the original visible instead of an edit in place.
type ResultSnapshot struct {
	ID         string `gorm:"primarykey;size:36" json:"id"`
	ElectionID string `gorm:"size:36;not null;index" json:"election_id"`
	// Version numbers this election's snapshots 1, 2, 3... The latest
	// non-superseded one is the current result.
	Version int    `gorm:"not null;default:1;uniqueIndex:idx_result_snapshot_election_version" json:"version"`
	Status  string `gorm:"size:20;not null;default:'PROVISIONAL';index" json:"status"`

	// ContentHash pins the exact tally content. Certification records it, and
	// a recount recomputes it, so tampering shows up as a hash mismatch.
	ContentHash string `gorm:"size:64;not null" json:"content_hash"`
	// ContentJSON is the full tally result, the same shape the public result
	// view renders. It carries no voter identity by construction.
	ContentJSON string `gorm:"type:text" json:"content_json"`

	// Turnout headline, denormalised for list views. Vote share lives in
	// ContentJSON per position and is never mixed into this.
	EligibleVoters int     `gorm:"not null;default:0" json:"eligible_voters"`
	BallotsCast    int     `gorm:"not null;default:0" json:"ballots_cast"`
	TurnoutPercent float64 `gorm:"not null;default:0" json:"turnout_percent"`

	ComputedByID string    `gorm:"size:36" json:"computed_by_id"`
	ComputedAt   time.Time `gorm:"not null;index" json:"computed_at"`

	CertifiedByID string    `gorm:"size:36" json:"certified_by_id,omitempty"`
	CertifiedAt   time.Time `json:"certified_at,omitempty"`
	CertifyReason string    `gorm:"size:500" json:"certify_reason,omitempty"`

	// SupersedesID names the snapshot this one replaces; SupersededByID is
	// set on the snapshot being retired. Both keep the chain auditable.
	SupersedesID   *string `gorm:"size:36;index" json:"supersedes_id,omitempty"`
	SupersededByID *string `gorm:"size:36" json:"superseded_by_id,omitempty"`
}

// BeforeCreate assigns a UUID and defaults.
func (s *ResultSnapshot) BeforeCreate(tx *gorm.DB) error {
	if s.ID == "" {
		s.ID = ids.New()
	}
	if s.Status == "" {
		s.Status = ResultStatusProvisional
	}
	if s.ComputedAt.IsZero() {
		s.ComputedAt = time.Now().UTC()
	}
	return nil
}

// ResultRecount records one recount run and how it compared to a prior
// snapshot. It never mutates ballots or the snapshot it compares against.
type ResultRecount struct {
	ID         string `gorm:"primarykey;size:36" json:"id"`
	ElectionID string `gorm:"size:36;not null;index" json:"election_id"`
	// ComparedSnapshotID names the snapshot the recount was checked against.
	ComparedSnapshotID string `gorm:"size:36;index" json:"compared_snapshot_id"`
	// RecomputedHash is the hash of the recount's own tally; if it equals the
	// compared snapshot's ContentHash the count reproduced exactly.
	RecomputedHash string    `gorm:"size:64;not null" json:"recomputed_hash"`
	Identical      bool      `gorm:"not null;default:false" json:"identical"`
	Differences    string    `gorm:"type:text" json:"differences,omitempty"`
	Reason         string    `gorm:"size:500" json:"reason"`
	RunByID        string    `gorm:"size:36" json:"run_by_id"`
	RunAt          time.Time `gorm:"not null;index" json:"run_at"`
}

// BeforeCreate assigns a UUID and timestamp.
func (r *ResultRecount) BeforeCreate(tx *gorm.DB) error {
	if r.ID == "" {
		r.ID = ids.New()
	}
	if r.RunAt.IsZero() {
		r.RunAt = time.Now().UTC()
	}
	return nil
}
