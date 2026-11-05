package services

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"time"

	"gorm.io/gorm"

	"campvote/apps/api/internal/models"
)

// EmailVerificationTTL is deliberately short: a six-digit code is easy to
// type and easy to guess, so it must not stay live for long.
var EmailVerificationTTL = 10 * time.Minute

var (
	ErrVerificationTokenInvalid = errors.New("email verification token is not valid")
	ErrEmailAlreadyVerified     = errors.New("email is already verified")
)

// GenerateVerificationCode returns a zero-padded six-digit code from the
// cryptographic source. The leading zeros matter: the code is a string, not a
// number to be trimmed.
func GenerateVerificationCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", fmt.Errorf("generating verification code: %w", err)
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

// CreateEmailVerificationToken issues a code for a user, invalidating any
// earlier unused ones. Storing only the keyed hash means this function is the
// single place the raw code exists — the caller must mail it and then forget it.
func CreateEmailVerificationToken(db *gorm.DB, userID, email, code, secret string) (*models.EmailVerificationToken, error) {
	row := &models.EmailVerificationToken{
		UserID:    userID,
		Email:     email,
		TokenHash: models.HashVerificationToken(code, secret),
		ExpiresAt: time.Now().Add(EmailVerificationTTL),
	}

	err := db.Transaction(func(tx *gorm.DB) error {
		// Burn outstanding tokens: two live links for one address means the
		// older one still works after the user re-requests, which is exactly
		// what someone who intercepted the first mail wants.
		now := time.Now()
		if err := tx.Model(&models.EmailVerificationToken{}).
			Where("user_id = ? AND used_at IS NULL", userID).
			Update("used_at", now).Error; err != nil {
			return err
		}
		return tx.Create(row).Error
	})
	if err != nil {
		return nil, fmt.Errorf("creating verification token: %w", err)
	}
	return row, nil
}

// ConsumeEmailVerificationToken marks the user's email verified and returns
// their id. Single-use is enforced with a conditional UPDATE, so two
// concurrent requests cannot both succeed.
func ConsumeEmailVerificationToken(db *gorm.DB, code, secret string) (string, error) {
	var userID string

	err := db.Transaction(func(tx *gorm.DB) error {
		var row models.EmailVerificationToken
		if err := tx.Where("token_hash = ?", models.HashVerificationToken(code, secret)).
			First(&row).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrVerificationTokenInvalid
			}
			return err
		}

		if row.UsedAt != nil || time.Now().After(row.ExpiresAt) {
			return ErrVerificationTokenInvalid
		}

		now := time.Now()
		res := tx.Model(&models.EmailVerificationToken{}).
			Where("id = ? AND used_at IS NULL", row.ID).
			Update("used_at", now)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrVerificationTokenInvalid // lost the race
		}

		// Only verify the address the token was issued for. If the user
		// changed their email in the meantime, this matches nothing and the
		// stale link cannot verify the new address.
		upd := tx.Model(&models.User{}).
			Where("id = ? AND email = ?", row.UserID, row.Email).
			Update("email_verified_at", now)
		if upd.Error != nil {
			return upd.Error
		}
		if upd.RowsAffected == 0 {
			return ErrVerificationTokenInvalid
		}

		userID = row.UserID
		return nil
	})
	if err != nil {
		return "", err
	}
	return userID, nil
}

// PurgeExpiredVerificationTokens removes spent and expired rows.
func PurgeExpiredVerificationTokens(db *gorm.DB, olderThan time.Duration) (int64, error) {
	cutoff := time.Now().Add(-olderThan)
	res := db.Where("expires_at < ? OR (used_at IS NOT NULL AND used_at < ?)", cutoff, cutoff).
		Delete(&models.EmailVerificationToken{})
	return res.RowsAffected, res.Error
}
