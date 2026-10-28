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
	if user == nil || strings.TrimSpace(user.Email) == "" {
		return Decision{Reason: "Sign in with your institutional email to vote"}, nil
	}
	if user.EmailVerifiedAt == nil {
		return Decision{Reason: "Confirm your email address before voting"}, nil
	}
	email := user.Email
	if !domainAllowed(election.AllowedDomains, email) {
		return Decision{Reason: "Your institutional email is not allowed in this election"}, nil
	}

	var entry models.ElectionRoster
	err := db.Where("election_id = ? AND LOWER(email) = ?", election.ID, strings.ToLower(strings.TrimSpace(email))).
		First(&entry).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// This is the acceptance case: a valid domain is not enough.
		return Decision{Reason: "You are not on this election's eligibility roster"}, nil
	}
	if err != nil {
		return Decision{}, err
	}
	if !models.IsEligibleStatus(entry.Status) {
		return Decision{Reason: "Your eligibility for this election is " + strings.ToLower(entry.Status)}, nil
	}
	if election.EligibilityFrozenAt != nil && entry.CreatedAt.After(*election.EligibilityFrozenAt) {
		// Defence in depth: import refuses this, but a direct database write
		// must not become a vote after the snapshot.
		return Decision{Reason: "Your eligibility evidence was added after voting opened"}, nil
	}
	return Decision{Eligible: true, RosterID: entry.ID}, nil
}

// domainAllowed implements the SRS domain prerequisite. An empty list does not
// grant access; it simply leaves the roster as the only evidence.
func domainAllowed(allowed, email string) bool {
	email = strings.ToLower(strings.TrimSpace(email))
	allowed = strings.TrimSpace(allowed)
	if allowed == "" {
		return true
	}
	for _, domain := range strings.Split(allowed, ",") {
		domain = strings.ToLower(strings.TrimSpace(domain))
		if domain == "" {
			continue
		}
		if !strings.HasPrefix(domain, "@") {
			domain = "@" + domain
		}
		if strings.HasSuffix(email, domain) {
			return true
		}
	}
	return false
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
	var eligible int64
	if err := db.Model(&models.ElectionRoster{}).
		Where("election_id = ? AND status = ?", election.ID, models.RosterStatusEligible).
		Count(&eligible).Error; err != nil {
		return nil, err
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
		if row.Status == models.RosterStatusEligible {
			report.Eligible = row.Count
		}
	}
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
