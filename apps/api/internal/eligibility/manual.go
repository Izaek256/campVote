package eligibility

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"gorm.io/gorm"

	"campvote/apps/api/internal/models"
)

// ErrRosterFrozen means the rule-based electorate is fixed until the election
// closes, so identity fields may not be added or changed.
var ErrRosterFrozen = errors.New("eligibility: roster is frozen")

// ensureRosterMutable refuses edits once voting has closed. A restricted
// election's list is its electorate, so the freeze does not apply to it.
func ensureRosterMutable(election *models.Election, now time.Time) error {
	if election.Status == models.ElectionStatusClosed || !now.Before(election.EndsAt) {
		return ErrElectionClosed
	}
	if election.EligibilityFrozenAt != nil && !election.RequireRoster {
		return ErrRosterFrozen
	}
	return nil
}

// cleanEmails splits a pasted list on commas, semicolons and whitespace,
// trims and lower-cases each address, and drops blanks and repeats.
func cleanEmails(input string) []string {
	parts := strings.FieldsFunc(input, func(r rune) bool {
		return r == ',' || r == ';' || r == '\n' || r == '\r' || r == '\t' || r == ' '
	})
	seen := map[string]bool{}
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		email := strings.ToLower(strings.TrimSpace(part))
		if email == "" || seen[email] {
			continue
		}
		seen[email] = true
		out = append(out, email)
	}
	return out
}

// AddEmails adds manually entered addresses to an election's list. Only the
// address is required: the student number is its local part, upper-cased, and
// the name starts as that same local part. Every address is validated before
// anything is written, so a partly valid paste adds nothing.
func AddEmails(db *gorm.DB, electionID, actorID, input string, now time.Time) ([]models.ElectionRoster, error) {
	emails := cleanEmails(input)
	if len(emails) == 0 {
		return nil, errors.New("enter at least one email address")
	}

	var election models.Election
	if err := db.Where("id = ?", electionID).First(&election).Error; err != nil {
		return nil, err
	}
	if err := ensureRosterMutable(&election, now); err != nil {
		return nil, err
	}

	var existing []models.ElectionRoster
	if err := db.Where("election_id = ?", election.ID).Find(&existing).Error; err != nil {
		return nil, err
	}
	byEmail := make(map[string]bool, len(existing))
	byStudent := make(map[string]bool, len(existing))
	for _, row := range existing {
		byEmail[strings.ToLower(strings.TrimSpace(row.Email))] = true
		byStudent[strings.ToUpper(strings.TrimSpace(row.StudentNumber))] = true
	}

	problems := make([]string, 0)
	entries := make([]models.ElectionRoster, 0, len(emails))
	for _, email := range emails {
		address, err := mail.ParseAddress(email)
		if err != nil || !strings.EqualFold(address.Address, email) || !strings.Contains(email, "@") {
			problems = append(problems, fmt.Sprintf("%s is not a valid email address", email))
			continue
		}
		studentNumber := strings.ToUpper(email[:strings.LastIndex(email, "@")])
		if byEmail[email] {
			problems = append(problems, fmt.Sprintf("%s is already on this list", email))
			continue
		}
		if byStudent[studentNumber] {
			problems = append(problems, fmt.Sprintf("%s conflicts with a student number already on this list", email))
			continue
		}
		byEmail[email] = true
		byStudent[studentNumber] = true
		entries = append(entries, models.ElectionRoster{
			ElectionID:    election.ID,
			StudentNumber: studentNumber,
			Email:         email,
			FullName:      email[:strings.LastIndex(email, "@")],
			Status:        models.RosterStatusEligible,
			Source:        models.RosterSourceManual,
			ImportedByID:  actorID,
		})
	}
	if len(problems) > 0 {
		return nil, errors.New(strings.Join(problems, "; "))
	}

	err := db.Transaction(func(tx *gorm.DB) error {
		for i := range entries {
			if err := tx.Create(&entries[i]).Error; err != nil {
				return err
			}
		}
		details, _ := json.Marshal(map[string]interface{}{
			"source":  models.RosterSourceManual,
			"created": len(entries),
		})
		return tx.Create(&models.ElectionAuditLog{
			ElectionID: election.ID,
			ActorID:    actorID,
			Action:     models.AuditActionRosterAdded,
			Details:    string(details),
		}).Error
	})
	if err != nil {
		return nil, err
	}
	return entries, nil
}

// EntryUpdate carries the identity fields an officer may correct. A nil field
// is left unchanged.
type EntryUpdate struct {
	Email         *string
	FullName      *string
	StudentNumber *string
}

// UpdateEntry corrects one roster row and audits the change. It keeps the
// election's uniqueness rules: an address or student number already used by
// another row is refused.
func UpdateEntry(db *gorm.DB, electionID, rosterID, actorID string, in EntryUpdate, now time.Time) (*models.ElectionRoster, error) {
	var election models.Election
	if err := db.Where("id = ?", electionID).First(&election).Error; err != nil {
		return nil, err
	}
	if err := ensureRosterMutable(&election, now); err != nil {
		return nil, err
	}
	var entry models.ElectionRoster
	if err := db.Where("id = ? AND election_id = ?", rosterID, election.ID).First(&entry).Error; err != nil {
		return nil, err
	}

	updates := map[string]interface{}{}
	changed := make([]string, 0, 3)
	if in.Email != nil {
		email := strings.ToLower(strings.TrimSpace(*in.Email))
		address, err := mail.ParseAddress(email)
		if err != nil || !strings.EqualFold(address.Address, email) || !strings.Contains(email, "@") {
			return nil, fmt.Errorf("%s is not a valid email address", email)
		}
		var taken int64
		if err := db.Model(&models.ElectionRoster{}).
			Where("election_id = ? AND LOWER(email) = ? AND id <> ?", election.ID, email, entry.ID).
			Count(&taken).Error; err != nil {
			return nil, err
		}
		if taken > 0 {
			return nil, fmt.Errorf("%s is already on this list", email)
		}
		if email != entry.Email {
			updates["email"] = email
			changed = append(changed, "email")
		}
	}
	if in.StudentNumber != nil {
		number := strings.ToUpper(strings.TrimSpace(*in.StudentNumber))
		if number == "" {
			return nil, errors.New("a student number cannot be blank")
		}
		var taken int64
		if err := db.Model(&models.ElectionRoster{}).
			Where("election_id = ? AND UPPER(student_number) = ? AND id <> ?", election.ID, number, entry.ID).
			Count(&taken).Error; err != nil {
			return nil, err
		}
		if taken > 0 {
			return nil, fmt.Errorf("student number %s is already on this list", number)
		}
		if number != entry.StudentNumber {
			updates["student_number"] = number
			changed = append(changed, "student_number")
		}
	}
	if in.FullName != nil {
		name := strings.TrimSpace(*in.FullName)
		if name == "" {
			return nil, errors.New("a name cannot be blank")
		}
		if name != entry.FullName {
			updates["full_name"] = name
			changed = append(changed, "full_name")
		}
	}
	if len(updates) == 0 {
		return &entry, nil
	}

	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.ElectionRoster{}).Where("id = ?", entry.ID).Updates(updates).Error; err != nil {
			return err
		}
		details, _ := json.Marshal(map[string]interface{}{
			"roster_id":      entry.ID,
			"changed_fields": changed,
		})
		return tx.Create(&models.ElectionAuditLog{
			ElectionID: election.ID,
			ActorID:    actorID,
			Action:     models.AuditActionRosterUpdated,
			Details:    string(details),
		}).Error
	})
	if err != nil {
		return nil, err
	}
	if err := db.Where("id = ?", entry.ID).First(&entry).Error; err != nil {
		return nil, err
	}
	return &entry, nil
}
