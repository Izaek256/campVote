package models

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"gorm.io/gorm"
	"campvote/apps/api/internal/ids"
)

// EmailVerificationToken is a single-use, expiring proof that someone can read
// mail at an address. Only the hash is stored, so leaking this table does not
// let an attacker verify addresses they do not control.
//
// The code is six digits, so the hash is keyed with the application secret.
// A plain SHA-256 of 1,000,000 possible codes would be reversible instantly
// from a database dump.
type EmailVerificationToken struct {
	ID     string `gorm:"primarykey;size:36" json:"id"`
	UserID string `gorm:"size:36;index;not null" json:"user_id"`

	TokenHash string `gorm:"size:64;uniqueIndex;not null" json:"-"`

	// Email is recorded as it was when the code was issued. If the user
	// changes their address before entering the code, the code must not
	// verify the new one — that is how you verify an address you never owned.
	Email string `gorm:"size:255;not null" json:"email"`

	ExpiresAt time.Time  `gorm:"index" json:"expires_at"`
	UsedAt    *time.Time `gorm:"index" json:"used_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

func (t *EmailVerificationToken) BeforeCreate(tx *gorm.DB) error {
	if t.ID == "" {
		t.ID = ids.New()
	}
	return nil
}

// HashVerificationToken returns the storage form of a verification code.
// secret is the application's signing secret, so a database leak alone cannot
// enumerate the six-digit space.
func HashVerificationToken(token, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(token))
	return hex.EncodeToString(mac.Sum(nil))
}
