// Package eligibility decides whether an authenticated account may vote in a
// particular election. An institutional email domain is a prerequisite, never
// proof: the roster entry is the evidence.
package eligibility

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"campvote/apps/api/internal/lifecycle"
	"campvote/apps/api/internal/models"
)

// ErrElectionClosed means the election's server-enforced schedule does not
// permit ballot access or eligibility changes.
var ErrElectionClosed = errors.New("eligibility: voting is not open")

// Decision is the answer for one account in one election. Reason is safe to
// show to the voter: it never names another person or exposes the roster.
type Decision struct {
	Eligible bool   `json:"eligible"`
	Reason   string `json:"reason,omitempty"`
	RosterID string `json:"-"`
}

// Check loads the election and evaluates the account against its roster.
func Check(db *gorm.DB, electionID string, user *models.User, at time.Time) (Decision, error) {
	var election models.Election
	if err := db.Where("id = ?", electionID).First(&election).Error; err != nil {
		return Decision{}, err
	}
	return CheckElection(db, &election, user, at)
}

// CheckElection is the single server-side eligibility gate. It is used by the
// ballot view, start and submit paths, so a hidden button can never be the only
// control.
//
// The account's email is the identity key, so it must be verified: an
// unverified address is one the account holder typed, and changing it to an
// unregistered student's roster address would otherwise borrow their vote.
func CheckElection(db *gorm.DB, election *models.Election, user *models.User, at time.Time) (Decision, error) {
	if election == nil {
		return Decision{Reason: "Election not found"}, nil
	}
	// A missed scheduler tick must not keep a stale status: reconcile the
	// schedule before deciding anything. This is a no-op while the window is
	// unchanged and is safe to run inside the submission transaction.
	if err := lifecycle.Ensure(db, election, at); err != nil {
		return Decision{}, err
	}
	if !election.IsOpen(at) {
		return Decision{Reason: "Voting is not open for this election"}, nil
	}
	return CheckRoster(db, election, user)
}

// CheckRoster answers whether this account may participate in the election,
// without requiring the voting window to be open. Candidate applications happen
// before polls open, so they need eligibility evidence but not the ballot gate.
//
// Decision logic:
//  1. Email must be verified.
//  2. ParseEmail must succeed and RuleAllows must return true for the election.
//  3. If a roster row exists with a non-ELIGIBLE status, access is denied
//     (explicit exclusion). No row means allowed (inclusion by rule).
func CheckRoster(db *gorm.DB, election *models.Election, user *models.User) (Decision, error) {
	if election == nil {
		return Decision{Reason: "Election not found"}, nil
	}
	if user == nil || strings.TrimSpace(user.Email) == "" {
		return Decision{Reason: "Sign in with your institutional email to participate"}, nil
	}
	if user.EmailVerifiedAt == nil {
		return Decision{Reason: "Confirm your email address before participating"}, nil
	}

	// Parse the email to extract enrolment year, course, and domain.
	id, ok := ParseEmail(user.Email)
	if !ok {
		return Decision{Reason: "Your email address does not match the expected student registration format (e.g. 2024BSE088@std.mast.ac.ug)"}, nil
	}

	// Check the election rule.
	if !RuleAllows(election, id) {
		reason := "This election is restricted"
		if election.AllowedYears != "" && election.AllowedCourses != "" {
			reason = fmt.Sprintf("This election is for %s / %s students",
				election.AllowedYears, strings.ToUpper(election.AllowedCourses))
		} else if election.AllowedYears != "" {
			reason = fmt.Sprintf("This election is for enrolment year %s students", election.AllowedYears)
		} else if election.AllowedCourses != "" {
			reason = fmt.Sprintf("This election is for %s students", strings.ToUpper(election.AllowedCourses))
		} else if election.AllowedDomain != "" {
			reason = fmt.Sprintf("This election is for %s email accounts", election.AllowedDomain)
		}
		return Decision{Reason: reason}, nil
	}

	// Check for an explicit roster exclusion. A non-ELIGIBLE status (e.g.
	// SUSPENDED, GRADUATED, INELIGIBLE) denies even when the rule matches.
	// No roster row → allowed by rule.
	var entry models.ElectionRoster
	err := db.Where("election_id = ? AND LOWER(email) = ?", election.ID,
		strings.ToLower(strings.TrimSpace(user.Email))).
		First(&entry).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return Decision{}, err
	}
	if err == nil {
		// Row found — check its status.
		if !models.IsEligibleStatus(entry.Status) {
			return Decision{Reason: "Your eligibility for this election is " + strings.ToLower(entry.Status)}, nil
		}
		if election.EligibilityFrozenAt != nil && entry.CreatedAt.After(*election.EligibilityFrozenAt) {
			return Decision{Reason: "Your eligibility evidence was added after voting opened"}, nil
		}
		return Decision{Eligible: true, RosterID: entry.ID}, nil
	}

	// No roster row and rule allows → eligible.
	return Decision{Eligible: true}, nil
}


// Freeze snapshots the eligible count when an election opens. After it, new
// roster entries are refused and only revocations are accepted; that keeps the
// electorate fixed while still allowing suspension or graduation to take effect
// before polls close.
func Freeze(db *gorm.DB, electionID, actorID string, at time.Time) (*models.Election, error) {
	var election models.Election
	if err := db.Where("id = ?", electionID).First(&election).Error; err != nil {
		return nil, err
	}
	if err := lifecycle.Ensure(db, &election, at); err != nil {
		return nil, err
	}
	if election.Status != models.ElectionStatusOpen || !election.IsOpen(at) {
		return nil, fmt.Errorf("eligibility can only be frozen while the election is open")
	}
	if election.EligibilityFrozenAt != nil {
		return &election, nil
	}
	// Count rule-matching users from the users table. We count all verified
	// users whose email matches the election rule. Roster exclusions are not
	// subtracted here (they are a minority and hard to count efficiently);
	// the frozen count is advisory for the officer report.
	var eligible int64
	var allUsers []models.User
	if err := db.Where("email_verified_at IS NOT NULL").Find(&allUsers).Error; err != nil {
		return nil, err
	}
	for _, u := range allUsers {
		id, ok := ParseEmail(u.Email)
		if ok && RuleAllows(&election, id) {
			eligible++
		}
	}

	err := db.Transaction(func(tx *gorm.DB) error {
		now := at.UTC()
		if err := tx.Model(&models.Election{}).Where("id = ?", election.ID).
			Updates(map[string]interface{}{
				"eligibility_frozen_at":    now,
				"eligibility_frozen_count": int(eligible),
			}).Error; err != nil {
			return err
		}
		details, _ := json.Marshal(map[string]interface{}{
			"eligible":  int(eligible),
			"frozen_at": now.Format(time.RFC3339),
		})
		return tx.Create(&models.ElectionAuditLog{
			ElectionID: election.ID,
			ActorID:    actorID,
			Action:     models.AuditActionEligibilityFrozen,
			Details:    string(details),
		}).Error
	})
	if err != nil {
		return nil, err
	}
	election.EligibilityFrozenAt = &at
	election.EligibilityFrozenCount = int(eligible)
	return &election, nil
}

// SetStatus changes one roster entry's status and audits it. Before the freeze
// this is an ordinary correction; after it, restoring eligibility is an
// override and is labelled as one. It refuses once voting has closed.
func SetStatus(db *gorm.DB, electionID, rosterID, actorID, status, reason string) (*models.ElectionRoster, error) {
	status = strings.ToUpper(strings.TrimSpace(status))
	if !validStatus(status) {
		return nil, fmt.Errorf("unknown roster status %q", status)
	}
	if strings.TrimSpace(reason) == "" {
		return nil, fmt.Errorf("a reason is required for an eligibility change")
	}

	var election models.Election
	if err := db.Where("id = ?", electionID).First(&election).Error; err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	if err := lifecycle.Ensure(db, &election, now); err != nil {
		return nil, err
	}
	switch election.Status {
	case models.ElectionStatusClosed, models.ElectionStatusReview,
		models.ElectionStatusFinalized, models.ElectionStatusArchived,
		models.ElectionStatusCancelled:
		return nil, ErrElectionClosed
	}
	if !now.Before(election.EndsAt) {
		return nil, ErrElectionClosed
	}

	var entry models.ElectionRoster
	if err := db.Where("id = ? AND election_id = ?", rosterID, election.ID).First(&entry).Error; err != nil {
		return nil, err
	}

	action := models.AuditActionRosterRevoked
	if status == models.RosterStatusEligible {
		action = models.AuditActionEligibilityOverride
	}
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.ElectionRoster{}).Where("id = ?", entry.ID).
			Updates(map[string]interface{}{
				"status":            status,
				"status_reason":     strings.TrimSpace(reason),
				"status_changed_at": now,
			}).Error; err != nil {
			return err
		}
		details, _ := json.Marshal(map[string]string{
			"roster_id": entry.ID,
			"status":    status,
			"reason":    strings.TrimSpace(reason),
		})
		return tx.Create(&models.ElectionAuditLog{
			ElectionID: election.ID,
			ActorID:    actorID,
			Action:     action,
			Details:    string(details),
		}).Error
	})
	if err != nil {
		return nil, err
	}
	entry.Status = status
	entry.StatusReason = strings.TrimSpace(reason)
	entry.StatusChangedAt = &now
	return &entry, nil
}

// Report is the officer-facing eligibility summary. Counts only: it has no
// email, student number, name, or ballot linkage.
type Report struct {
	ElectionID      string           `json:"election_id"`
	Status          string           `json:"status"`
	Total           int64            `json:"total"`
	Eligible        int64            `json:"eligible"`
	ByStatus        map[string]int64 `json:"by_status"`
	Participated    int64            `json:"participated"`
	TurnoutPercent  float64          `json:"turnout_percent"`
	FrozenAt        *time.Time       `json:"frozen_at,omitempty"`
	FrozenEligible  int              `json:"frozen_eligible"`
	LastImportAt    *time.Time       `json:"last_import_at,omitempty"`
	LastImportBatch string           `json:"last_import_batch_id,omitempty"`
}

// BuildReport aggregates the roster and participation counts for an election.
func BuildReport(db *gorm.DB, electionID string) (*Report, error) {
	var election models.Election
	if err := db.Where("id = ?", electionID).First(&election).Error; err != nil {
		return nil, err
	}

	type statusCount struct {
		Status string
		Count  int64
	}
	var rows []statusCount
	if err := db.Model(&models.ElectionRoster{}).
		Select("status, count(*) as count").
		Where("election_id = ?", election.ID).
		Group("status").Scan(&rows).Error; err != nil {
		return nil, err
	}

	report := &Report{
		ElectionID:     election.ID,
		Status:         election.Status,
		ByStatus:       map[string]int64{},
		FrozenAt:       election.EligibilityFrozenAt,
		FrozenEligible: election.EligibilityFrozenCount,
	}
	for _, row := range rows {
		report.ByStatus[row.Status] = row.Count
		report.Total += row.Count
	}

	// Count rule-eligible users (the authoritative eligible number).
	// Use the frozen count if already frozen, otherwise derive from users table.
	if election.EligibilityFrozenAt != nil {
		report.Eligible = int64(election.EligibilityFrozenCount)
	} else {
		var allUsers []models.User
		if err := db.Where("email_verified_at IS NOT NULL").Find(&allUsers).Error; err != nil {
			return nil, err
		}
		for _, u := range allUsers {
			id, ok := ParseEmail(u.Email)
			if ok && RuleAllows(&election, id) {
				report.Eligible++
			}
		}
	}
	report.Total = report.Eligible // total eligible, not total roster rows

	if err := db.Model(&models.VoterParticipation{}).
		Where("election_id = ?", election.ID).Count(&report.Participated).Error; err != nil {
		return nil, err
	}
	if report.Eligible > 0 {
		report.TurnoutPercent = float64(report.Participated) / float64(report.Eligible) * 100
	}
	var last models.EligibilityImportBatch
	if err := db.Where("election_id = ?", election.ID).
		Order("created_at DESC").First(&last).Error; err == nil {
		at := last.CreatedAt
		report.LastImportAt = &at
		report.LastImportBatch = last.ID
	}
	return report, nil
}


func validStatus(status string) bool {
	switch status {
	case models.RosterStatusEligible, models.RosterStatusIneligible,
		models.RosterStatusGraduated, models.RosterStatusSuspended:
		return true
	}
	return false
}
