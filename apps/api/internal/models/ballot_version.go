package models

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
	"campvote/apps/api/internal/ids"
)

// BallotVersion represents an immutable snapshot of a ballot definition at publication time.
//
// CRITICAL ARCHITECTURE (ISSUE-06):
// 1. Once published, a version is immutable. The ContentHash provides cryptographic proof.
// 2. Votes reference the BallotVersionID, ensuring they are interpreted against the exact
//    definition the voter saw when casting.
// 3. Post-publication edits create new versions; old versions remain retrievable for audit.
// 4. Status tracks whether this is the current published version or historical.
type BallotVersion struct {
	ID          string    `gorm:"primarykey;size:36" json:"id"`
	ElectionID  string    `gorm:"size:36;not null;index" json:"election_id"`
	Version     int       `gorm:"not null;index" json:"version"`
	ContentHash string    `gorm:"size:64;not null;index" json:"content_hash"`
	
	// Serialized ballot structure (positions, candidates, selection rules)
	// This is the canonical definition used to render ballots and validate votes.
	ContentJSON string    `gorm:"type:jsonb;not null" json:"content_json"`
	
	// Publication metadata
	PublishedBy string    `gorm:"size:36;not null;index" json:"published_by"`
	PublishedAt time.Time `gorm:"not null;index" json:"published_at"`
	Status      string    `gorm:"size:20;not null;default:'CURRENT';index" json:"status"`
	
	CreatedAt   time.Time `gorm:"index" json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// Ballot version status constants
const (
	BallotVersionStatusCurrent  = "CURRENT"
	BallotVersionStatusSuperseded = "SUPERSEDED"
)

// BallotContent represents the structure stored in ContentJSON
type BallotContent struct {
	Positions []BallotPositionContent `json:"positions"`
}

// BallotPositionContent represents a position in the ballot content
type BallotPositionContent struct {
	ID            string                    `json:"id"`
	Title         string                    `json:"title"`
	Description   string                    `json:"description"`
	QuestionType  string                    `json:"question_type"`
	MinSelections int                       `json:"min_selections"`
	MaxSelections int                       `json:"max_selections"`
	Seats         int                       `json:"seats"`
	AllowBlank    bool                      `json:"allow_blank"`
	DisplayOrder  int                       `json:"display_order"`
	Candidates    []BallotCandidateContent  `json:"candidates"`
}

// BallotCandidateContent represents a candidate in the ballot content
type BallotCandidateContent struct {
	ID           string `json:"id"`
	FullName     string `json:"full_name"`
	PhotoURL     string `json:"photo_url"`
	Manifesto    string `json:"manifesto"`
	Bio          string `json:"bio"`
	BallotNumber int    `json:"ballot_number"`
}

// BeforeCreate assigns a new UUID, computes content hash, and sets timestamps.
func (bv *BallotVersion) BeforeCreate(tx *gorm.DB) error {
	if bv.ID == "" {
		bv.ID = ids.New()
	}
	if bv.Status == "" {
		bv.Status = BallotVersionStatusCurrent
	}
	if bv.PublishedAt.IsZero() {
		bv.PublishedAt = time.Now().UTC()
	}
	if bv.ContentHash == "" && bv.ContentJSON != "" {
		hash := sha256.Sum256([]byte(bv.ContentJSON))
		bv.ContentHash = fmt.Sprintf("%x", hash)
	}
	return nil
}

// ComputeContentHash calculates SHA-256 hash of the ballot content.
func (bv *BallotVersion) ComputeContentHash() (string, error) {
	if bv.ContentJSON == "" {
		return "", fmt.Errorf("cannot compute hash of empty content")
	}
	hash := sha256.Sum256([]byte(bv.ContentJSON))
	return fmt.Sprintf("%x", hash), nil
}

// ValidateContent ensures the content JSON is well-formed and matches the hash.
func (bv *BallotVersion) ValidateContent() error {
	if bv.ContentJSON == "" {
		return fmt.Errorf("ballot content is empty")
	}
	
	// Verify JSON is well-formed
	var content BallotContent
	if err := json.Unmarshal([]byte(bv.ContentJSON), &content); err != nil {
		return fmt.Errorf("invalid ballot content JSON: %w", err)
	}
	
	// Verify hash matches
	computedHash, err := bv.ComputeContentHash()
	if err != nil {
		return err
	}
	if computedHash != bv.ContentHash {
		return fmt.Errorf("content hash mismatch: expected %s, computed %s", bv.ContentHash, computedHash)
	}
	
	return nil
}

// GetContent parses and returns the structured ballot content.
func (bv *BallotVersion) GetContent() (*BallotContent, error) {
	var content BallotContent
	if err := json.Unmarshal([]byte(bv.ContentJSON), &content); err != nil {
		return nil, fmt.Errorf("failed to parse ballot content: %w", err)
	}
	return &content, nil
}
