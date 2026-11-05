package database

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"gorm.io/gorm"

	"campvote/apps/api/internal/models"
	"campvote/apps/api/internal/password"
)

// BootstrapMinPasswordLength is longer than the ordinary user floor. This
// account can create every other account, so it is the one password worth
// making awkward to type.
const BootstrapMinPasswordLength = 12

// ErrAdministratorExists means bootstrap already ran. It is the one-time
// guarantee: there is no flag to force a second administrator, because a
// reusable bootstrap is a permanent known-account hole.
var ErrAdministratorExists = errors.New("an administrator already exists; bootstrap has already run")

// CreateFirstAdmin creates the first administrator of a fresh deployment.
//
// It is deliberately not a seeder. There is no default address, no default
// password, and it refuses as soon as any ADMIN exists. Credentials come from
// the caller, which reads them from the environment once and then removes them.
func CreateFirstAdmin(ctx context.Context, db *gorm.DB, email, plaintext, firstName, lastName string) (*models.User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return nil, errors.New("BOOTSTRAP_ADMIN_EMAIL is required")
	}
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email {
		return nil, fmt.Errorf("%q is not a valid email address", email)
	}
	if plaintext == "" {
		return nil, errors.New("BOOTSTRAP_ADMIN_PASSWORD is required")
	}
	if len([]rune(plaintext)) < BootstrapMinPasswordLength {
		return nil, fmt.Errorf("the administrator password must be at least %d characters", BootstrapMinPasswordLength)
	}
	if failed := password.Check(plaintext, email, firstName, lastName); len(failed) > 0 {
		return nil, errors.New(password.Message(failed))
	}

	firstName = strings.TrimSpace(firstName)
	lastName = strings.TrimSpace(lastName)
	if firstName == "" {
		firstName = "Administrator"
	}
	if lastName == "" {
		lastName = "User"
	}

	var admins int64
	// Unscoped: a soft-deleted administrator still counts. Resurrecting
	// bootstrap because the only admin was removed would hand out a fresh
	// account with a known password.
	if err := db.WithContext(ctx).Unscoped().Model(&models.User{}).
		Where("role = ?", models.RoleAdmin).Count(&admins).Error; err != nil {
		return nil, fmt.Errorf("checking for an existing administrator: %w", err)
	}
	if admins > 0 {
		return nil, ErrAdministratorExists
	}

	now := time.Now()
	admin := &models.User{
		FirstName: firstName,
		LastName:  lastName,
		Email:     email,
		Password:  plaintext,
		Role:      models.RoleAdmin,
		Active:    true,
		// The operator proved control of the deployment, not of the mailbox.
		// Requiring verification here would lock the first administrator out
		// of the only account that can configure mail delivery.
		EmailVerifiedAt: &now,
	}
	if err := db.WithContext(ctx).Create(admin).Error; err != nil {
		return nil, fmt.Errorf("creating the first administrator: %w", err)
	}
	return admin, nil
}
