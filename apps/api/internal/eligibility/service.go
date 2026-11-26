// Package eligibility decides whether an authenticated account may vote in a
// particular election. An institutional email domain is a prerequisite, never
// proof: the roster entry is the evidence.
package eligibility

import (
	"encoding/json"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"sync"
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
	decision, err := CheckRoster(db, election, user)
	if err != nil || !decision.Eligible {
		return decision, err
	}
	// Voting also needs the campus profile. The web app opens the profile form;
	// this is the server-side half, so a hidden button is never the only
	// control. CheckRoster is left alone because candidate applications do not
	// need the profile.
	if missing := user.MissingProfileFields(); len(missing) > 0 {
		return Decision{Reason: "Complete your profile (" + strings.Join(missing, ", ") + ") before voting"}, nil
	}
	return decision, nil
}

// CheckRoster answers whether this account may participate in the election,
// without requiring the voting window to be open. Candidate applications happen
// before polls open, so they need eligibility evidence but not the ballot gate.
//
// Decision logic:
//  1. Email must be verified.
//  2. ParseEmail must succeed and RuleAllows must return true for the election.
//  3. If a roster row exists with a non-ELIGIBLE status, access is denied
//     (explicit exclusion). No row means allowed (inclusion by rule) unless
//     the election requires a roster, in which case the row is required.
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

	// A listed email is an individual inclusion. It admits the account even
	// when the election's criteria (year, course, sex or residence) do not
	// match. A non-ELIGIBLE row remains an explicit exclusion and denies even
	// when the rule would otherwise admit the account.
	var entry models.ElectionRoster
	err := db.Where("election_id = ? AND LOWER(email) = ?", election.ID,
		strings.ToLower(strings.TrimSpace(user.Email))).
		First(&entry).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return Decision{}, err
	}
	if err == nil {
		if !models.IsEligibleStatus(entry.Status) {
			return Decision{Reason: "Your eligibility for this election is " + strings.ToLower(entry.Status)}, nil
		}
		// A restricted election's list is its electorate, so an officer may add
		// a name while polls are open. The rule-based freeze still applies.
		if election.EligibilityFrozenAt != nil && !election.RequireRoster && entry.CreatedAt.After(*election.EligibilityFrozenAt) {
			return Decision{Reason: "Your eligibility evidence was added after voting opened"}, nil
		}
		return Decision{Eligible: true, RosterID: entry.ID}, nil
	}

	// No roster row. A restricted election is defined by its list, so the
	// criteria cannot admit somebody the list leaves out.
	if election.RequireRoster {
		return Decision{Reason: "This election is limited to the voters on its list"}, nil
	}

	// Otherwise the election's criteria are the second, independent path in.
	if !RuleAllows(election, id) {
		return Decision{Reason: ruleReason(election)}, nil
	}

	// Sex and residence are profile facts, not email facts, so they are
	// checked separately and only when the election restricts them.
	profileAllowed, err := ProfileAllows(db, election, user)
	if err != nil {
		return Decision{}, err
	}
	if !profileAllowed {
		return Decision{Reason: RuleForElection(election).ProfileReason()}, nil
	}

	return Decision{Eligible: true}, nil
}

// ruleReason is the voter-safe explanation for a criteria mismatch.
func ruleReason(election *models.Election) string {
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
	return reason
}

// IsListed reports whether an email is an eligible individual inclusion for an
// election. Revoked rows do not count.
func IsListed(db *gorm.DB, electionID, email string) (bool, error) {
	var count int64
	err := db.Model(&models.ElectionRoster{}).
		Where("election_id = ? AND LOWER(email) = ? AND status = ?", electionID,
			strings.ToLower(strings.TrimSpace(email)), models.RosterStatusEligible).
		Count(&count).Error
	return count > 0, err
}

// ProfileAllows applies an election's sex and residence restrictions to a
// signed-in account. A blank rule admits everyone without a query; otherwise
// the residence area name is resolved from the account's residence_area_id.
func ProfileAllows(db *gorm.DB, election *models.Election, user *models.User) (bool, error) {
	rule := RuleForElection(election)
	if rule.AllowedSexes == "" && rule.AllowedResidences == "" {
		return true, nil
	}
	if user == nil {
		return false, nil
	}
	residence := ""
	if user.ResidenceAreaID != nil && *user.ResidenceAreaID != "" {
		var area models.ResidenceArea
		err := db.Where("id = ?", *user.ResidenceAreaID).First(&area).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return false, err
		}
		residence = area.Name
	}
	return rule.AllowsProfile(user.Sex, residence), nil
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
	// Snapshot the rule-based electorate, minus explicit roster exclusions.
	eligible, err := EligibleCount(db, &election)
	if err != nil {
		return nil, err
	}

	err = db.Transaction(func(tx *gorm.DB) error {
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
	case models.ElectionStatusClosed:
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

// EligibleCount returns the number of verified accounts matching the election
// rule, minus accounts with an explicit non-ELIGIBLE roster row. The roster is
// an exclusion list under rule-based eligibility, not an inclusion list.
func EligibleCount(db *gorm.DB, election *models.Election) (int64, error) {
	_, eligible, err := EligibleCounts(db, election)
	return eligible, err
}

// EligibleCounts returns both the rule-matching account count (total) and the
// count after roster exclusions (eligible), for the officer report.
func EligibleCounts(db *gorm.DB, election *models.Election) (total, eligible int64, err error) {
	return CountForRule(db, election.ID, RuleForElection(election), election.RequireRoster)
}

// countBatch is how many accounts one worker evaluates at a time. The rows are
// read in batches so a large roll never has to fit in memory at once.
const countBatch = 500

// CountForRule counts verified accounts admitted by either eligibility path,
// in parallel. It is the single counter behind the report, the freeze snapshot
// and the live preview, so those three can never disagree.
//
// A restricted election's roster is its electorate: only an ELIGIBLE row counts,
// regardless of the criteria. Under rule-based eligibility the two paths are a
// union: an account counts when the rule admits it or its email is listed, and
// an ELIGIBLE row is never counted twice. A non-ELIGIBLE row still excludes an
// account the rule would admit.
func CountForRule(db *gorm.DB, electionID string, rule Rule, requireRoster bool) (total, eligible int64, err error) {
	residences, err := residenceNames(db)
	if err != nil {
		return 0, 0, err
	}
	allowed, blocked, err := rosterSets(db, electionID)
	if err != nil {
		return 0, 0, err
	}

	workers := runtime.NumCPU()
	if workers < 1 {
		workers = 1
	}
	// Bound the concurrent batches: the work is CPU-bound per row, but each
	// batch also holds a database cursor open.
	sem := make(chan struct{}, workers)
	var (
		mu sync.Mutex
		wg sync.WaitGroup
	)

	var batch []models.User
	feedErr := db.Model(&models.User{}).Where("email_verified_at IS NOT NULL").
		FindInBatches(&batch, countBatch, func(_ *gorm.DB, _ int) error {
			// FindInBatches reuses the slice, so each worker needs its own copy.
			chunk := make([]models.User, len(batch))
			copy(chunk, batch)

			wg.Add(1)
			sem <- struct{}{}
			go func(chunk []models.User) {
				defer wg.Done()
				defer func() { <-sem }()
				var found, kept int64
				for _, u := range chunk {
					id, ok := ParseEmail(u.Email)
					if !ok {
						continue
					}
					key := strings.ToLower(strings.TrimSpace(u.Email))
					listed := allowed[key]
					if requireRoster {
						if !listed {
							continue
						}
						found++
						kept++
						continue
					}
					ruleMatch := rule.Allows(id) &&
						rule.AllowsProfile(u.Sex, residences[userResidence(u)])
					if !ruleMatch && !listed {
						continue
					}
					found++
					if !blocked[key] {
						kept++
					}
				}
				mu.Lock()
				total += found
				eligible += kept
				mu.Unlock()
			}(chunk)
			return nil
		}).Error
	wg.Wait()
	if feedErr != nil {
		return 0, 0, feedErr
	}
	return total, eligible, nil
}

// rosterSets splits an election's roster into the addresses it admits and the
// addresses it explicitly excludes.
func rosterSets(db *gorm.DB, electionID string) (allowed, blocked map[string]bool, err error) {
	var rows []models.ElectionRoster
	if err := db.Select("email", "status").Where("election_id = ?", electionID).
		Find(&rows).Error; err != nil {
		return nil, nil, err
	}
	allowed = make(map[string]bool, len(rows))
	blocked = make(map[string]bool)
	for _, row := range rows {
		email := strings.ToLower(strings.TrimSpace(row.Email))
		if models.IsEligibleStatus(row.Status) {
			allowed[email] = true
		} else {
			blocked[email] = true
		}
	}
	return allowed, blocked, nil
}

// residenceNames maps residence_area_id to its display name. The table is
// small and administrator-maintained, so one query is cheaper than a join per
// account.
func residenceNames(db *gorm.DB) (map[string]string, error) {
	var areas []models.ResidenceArea
	if err := db.Find(&areas).Error; err != nil {
		return nil, err
	}
	names := make(map[string]string, len(areas))
	for _, area := range areas {
		names[area.ID] = area.Name
	}
	return names, nil
}

func userResidence(u models.User) string {
	if u.ResidenceAreaID == nil {
		return ""
	}
	return *u.ResidenceAreaID
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

	// Rule-based electorate, minus explicit exclusions.
	total, eligible, err := EligibleCounts(db, &election)
	if err != nil {
		return nil, err
	}
	report.Total = total
	report.Eligible = eligible

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
