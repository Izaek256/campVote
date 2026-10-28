// Package lifecycle owns the election state machine and the schedule rules
// that decide whether voting is open. Server time is the only clock: callers
// pass it in so tests can move it and request headers can never affect it.
package lifecycle

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"

	"campvote/apps/api/internal/models"
)

// MinimumDuration is the shortest schedule an election may be given. A shorter
// window is almost always a timezone or typo mistake, not a real contest.
const MinimumDuration = time.Hour

// DisplayTimezone is the human-facing zone for MMU. Timestamps are stored and
// returned in UTC; clients format them in this zone for display only.
const DisplayTimezone = "Africa/Kampala"

// SystemActor is the actor recorded on transitions the scheduler or a lazy
// request made, rather than a person.
const SystemActor = "system"

var (
	// ErrIllegalTransition is returned for a state pair the machine forbids.
	ErrIllegalTransition = errors.New("illegal election transition")
	// ErrReasonRequired is returned for an exceptional action without a reason.
	ErrReasonRequired = errors.New("a reason is required for this change")
	// ErrConcurrentTransition means another request changed the same election
	// first; the caller should reload and decide again.
	ErrConcurrentTransition = errors.New("the election changed while this request was being applied")
)

// ConfigError carries per-field reasons so the HTTP layer can return 422.
type ConfigError struct {
	Fields map[string]string
}

func (e *ConfigError) Error() string { return "election configuration is not valid" }

// FieldErrors lets respond.WriteError render a 422 with the field map.
func (e *ConfigError) FieldErrors() map[string]string { return e.Fields }

// transitionMatrix is the whole state machine. OPEN and FINALIZED are reached
// by the scheduler and by certification respectively; the transition endpoint
// refuses to set them directly.
var transitionMatrix = map[string][]string{
	models.ElectionStatusDraft: {
		models.ElectionStatusRegistration,
		models.ElectionStatusPending,
		models.ElectionStatusScheduled,
		models.ElectionStatusCancelled,
	},
	models.ElectionStatusRegistration: {
		models.ElectionStatusPending,
		models.ElectionStatusScheduled,
		models.ElectionStatusCancelled,
	},
	models.ElectionStatusPending: {
		models.ElectionStatusRegistration,
		models.ElectionStatusScheduled,
		models.ElectionStatusCancelled,
	},
	models.ElectionStatusScheduled: {
		models.ElectionStatusOpen,
		models.ElectionStatusClosed,
		models.ElectionStatusCancelled,
	},
	models.ElectionStatusOpen: {
		models.ElectionStatusClosed,
		models.ElectionStatusCancelled,
	},
	models.ElectionStatusClosed: {
		models.ElectionStatusReview,
		models.ElectionStatusFinalized,
		models.ElectionStatusArchived,
	},
	models.ElectionStatusReview: {
		models.ElectionStatusFinalized,
		models.ElectionStatusArchived,
	},
	models.ElectionStatusFinalized: {
		models.ElectionStatusArchived,
	},
	models.ElectionStatusArchived:  {},
	models.ElectionStatusCancelled: {},
}

// CanTransition reports whether the state machine allows from -> to.
func CanTransition(from, to string) bool {
	for _, allowed := range transitionMatrix[from] {
		if allowed == to {
			return true
		}
	}
	return false
}

// AllowedTransitions returns the next states for an election, sorted.
func AllowedTransitions(from string) []string {
	out := append([]string(nil), transitionMatrix[from]...)
	sort.Strings(out)
	return out
}

// KnownStatus reports whether status is one this application recognises.
func KnownStatus(status string) bool {
	_, ok := transitionMatrix[status]
	return ok
}

// IsFrozen reports whether configuration edits need the audited exception.
// Configuration freezes when voting opens and stays frozen afterwards.
func IsFrozen(status string) bool {
	switch status {
	case models.ElectionStatusOpen, models.ElectionStatusClosed,
		models.ElectionStatusReview, models.ElectionStatusFinalized,
		models.ElectionStatusArchived, models.ElectionStatusCancelled:
		return true
	}
	return false
}

// Phase is the display-only view of where an election is in time.
type Phase struct {
	ElectionID      string    `json:"election_id"`
	Status          string    `json:"status"`
	AcceptingVotes  bool      `json:"accepting_votes"`
	ServerTime      time.Time `json:"server_time"`
	StartsAt        time.Time `json:"starts_at"`
	EndsAt          time.Time `json:"ends_at"`
	DisplayTimezone string    `json:"display_timezone"`
}

// CurrentPhase reports status and schedule from server time. Call Ensure first
// when the stored status may be stale.
func CurrentPhase(e *models.Election, at time.Time) Phase {
	at = at.UTC()
	return Phase{
		ElectionID:      e.ID,
		Status:          e.Status,
		AcceptingVotes:  e.IsOpen(at),
		ServerTime:      at,
		StartsAt:        e.StartsAt.UTC(),
		EndsAt:          e.EndsAt.UTC(),
		DisplayTimezone: DisplayTimezone,
	}
}

// ValidateBasics checks the election's own fields and schedule. It does not
// touch positions or candidates, so a draft can be saved before the ballot is
// built. scheduling additionally requires the end time to still be in future.
func ValidateBasics(e *models.Election, at time.Time, scheduling bool) error {
	fields := map[string]string{}
	if strings.TrimSpace(e.Title) == "" {
		fields["title"] = "a title is required"
	}
	if !validCategory(e.Category) {
		fields["category"] = "choose a known election category"
	}
	if !validResultPolicy(e.ResultPolicy) {
		fields["result_policy"] = "choose LIVE, AFTER_CLOSE or CERTIFIED_ONLY"
	}
	if !validDomains(e.AllowedDomains) {
		fields["allowed_domains"] = "use comma-separated institutional domains such as @std.must.ac.ug"
	}
	switch {
	case e.StartsAt.IsZero() || e.EndsAt.IsZero():
		fields["starts_at"] = "starts_at and ends_at are required"
	case !e.StartsAt.Before(e.EndsAt):
		fields["ends_at"] = "ends_at must be after starts_at"
	case e.EndsAt.Sub(e.StartsAt) < MinimumDuration:
		fields["ends_at"] = fmt.Sprintf("the schedule must be at least %s long", MinimumDuration)
	}
	if scheduling && !e.StartsAt.IsZero() && !e.EndsAt.IsZero() && !e.EndsAt.After(at.UTC()) {
		fields["ends_at"] = "ends_at must be in the future when the election is scheduled"
	}
	if len(fields) == 0 {
		return nil
	}
	return &ConfigError{Fields: fields}
}

// ValidateConfiguration checks everything that must be true before an election
// is scheduled or opened: schedule, policy, positions and approved candidates.
func ValidateConfiguration(db *gorm.DB, e *models.Election, at time.Time, scheduling bool) error {
	if err := ValidateBasics(e, at, scheduling); err != nil {
		return err
	}
	fields := map[string]string{}
	var positions []models.Position
	if err := db.Where("election_id = ?", e.ID).Find(&positions).Error; err != nil {
		return err
	}
	if len(positions) == 0 {
		fields["positions"] = "at least one position is required"
	}

	seenTitles := map[string]bool{}
	for _, position := range positions {
		title := strings.ToLower(strings.TrimSpace(position.Title))
		key := "positions." + position.ID
		if title == "" {
			fields[key] = "each position needs a title"
		} else if seenTitles[title] {
			fields[key] = "position titles must be unique within an election"
		}
		seenTitles[title] = true

		if !validQuestionType(position.QuestionType) {
			fields[key] = "unknown question type"
		}
		if position.MinSelections < 0 || position.MaxSelections < 1 {
			fields[key] = "selection limits must be positive"
		} else if position.MinSelections > position.MaxSelections {
			fields[key] = "min_selections cannot be greater than max_selections"
		} else if position.QuestionType != models.QuestionTypeMultiChoice && position.MaxSelections != 1 {
			fields[key] = "only a multi-choice position may accept more than one selection"
		}
	}

	if len(positions) > 0 {
		needsCandidates := make([]string, 0, len(positions))
		for _, position := range positions {
			if position.QuestionType == models.QuestionTypeSingleChoice ||
				position.QuestionType == models.QuestionTypeMultiChoice {
				needsCandidates = append(needsCandidates, position.ID)
			}
		}
		if len(needsCandidates) > 0 {
			type candidateCount struct {
				PositionID string
				Count      int64
			}
			var counts []candidateCount
			if err := db.Model(&models.Candidate{}).
				Select("position_id, count(*) as count").
				Where("election_id = ? AND approval_status = ? AND position_id IN ?",
					e.ID, models.CandidateStatusApproved, needsCandidates).
				Group("position_id").Scan(&counts).Error; err != nil {
				return err
			}
			approved := map[string]int64{}
			for _, row := range counts {
				approved[row.PositionID] = row.Count
			}
			for _, id := range needsCandidates {
				if approved[id] == 0 {
					fields["positions."+id] = "at least one approved candidate is required"
				}
			}
		}
	}

	if len(fields) == 0 {
		return nil
	}
	return &ConfigError{Fields: fields}
}

// ReadyToOpen is ValidateConfiguration plus the schedule window.
func ReadyToOpen(db *gorm.DB, e *models.Election, at time.Time) error {
	at = at.UTC()
	if err := ValidateConfiguration(db, e, at, false); err != nil {
		return err
	}
	if at.Before(e.StartsAt) {
		return errors.New("the election has not reached its start time")
	}
	if !at.Before(e.EndsAt) {
		return errors.New("the election has already ended")
	}
	return nil
}

// Apply validates and persists one transition, writing the election audit row
// in the same transaction. The caller has already checked the actor's role.
func Apply(db *gorm.DB, e *models.Election, to, actorID, reason, ip string, at time.Time) (*models.Election, error) {
	to = strings.ToUpper(strings.TrimSpace(to))
	if !KnownStatus(to) || !CanTransition(e.Status, to) {
		return nil, fmt.Errorf("%w: %s -> %s", ErrIllegalTransition, e.Status, to)
	}
	at = at.UTC()
	switch to {
	case models.ElectionStatusScheduled:
		if err := ValidateConfiguration(db, e, at, true); err != nil {
			return nil, err
		}
	case models.ElectionStatusOpen:
		if err := ReadyToOpen(db, e, at); err != nil {
			return nil, err
		}
	case models.ElectionStatusClosed:
		if at.Before(e.EndsAt) && strings.TrimSpace(reason) == "" {
			return nil, fmt.Errorf("%w: closing before the scheduled end", ErrReasonRequired)
		}
	case models.ElectionStatusCancelled:
		if strings.TrimSpace(reason) == "" {
			return nil, fmt.Errorf("%w: cancelling an election", ErrReasonRequired)
		}
	}
	return persist(db, e, to, actorID, reason, ip, at)
}

// Ensure reconciles the stored status with server time. It is the lazy safety
// net for a missed scheduler tick: after EndsAt it closes, and it never opens
// an election whose configuration is invalid.
func Ensure(db *gorm.DB, e *models.Election, at time.Time) error {
	at = at.UTC()
	switch e.Status {
	case models.ElectionStatusScheduled:
		if !at.Before(e.EndsAt) {
			return ensureApply(db, e, models.ElectionStatusClosed, "scheduled end reached without opening", at)
		}
		if !at.Before(e.StartsAt) {
			if err := ReadyToOpen(db, e, at); err != nil {
				var configErr *ConfigError
				if !errors.As(err, &configErr) {
					return err
				}
				log.Printf("lifecycle: election %s did not open: %v", e.ID, configErr.Fields)
				return nil
			}
			return ensureApply(db, e, models.ElectionStatusOpen, "start time reached", at)
		}
	case models.ElectionStatusOpen:
		if !at.Before(e.EndsAt) {
			return ensureApply(db, e, models.ElectionStatusClosed, "end time reached", at)
		}
	}
	return nil
}

func ensureApply(db *gorm.DB, e *models.Election, to, reason string, at time.Time) error {
	_, err := Apply(db, e, to, SystemActor, reason, "", at)
	if errors.Is(err, ErrConcurrentTransition) {
		// Another request won the race. Reload so the caller sees the new state.
		return db.Where("id = ?", e.ID).First(e).Error
	}
	return err
}

// SyncAll reconciles every election that could be due. The scheduler calls it
// once a minute; a missed tick is repaired by the next request or restart.
func SyncAll(db *gorm.DB, at time.Time) (int, error) {
	at = at.UTC()
	var due []models.Election
	if err := db.Where(
		"(status = ? AND starts_at <= ?) OR (status = ? AND ends_at <= ?)",
		models.ElectionStatusScheduled, at,
		models.ElectionStatusOpen, at,
	).Find(&due).Error; err != nil {
		return 0, err
	}
	changed := 0
	var failure error
	for i := range due {
		before := due[i].Status
		if err := Ensure(db, &due[i], at); err != nil {
			failure = errors.Join(failure, fmt.Errorf("election %s: %w", due[i].ID, err))
			continue
		}
		if due[i].Status != before {
			changed++
		}
	}
	return changed, failure
}

func persist(db *gorm.DB, e *models.Election, to, actorID, reason, ip string, at time.Time) (*models.Election, error) {
	from := e.Status
	details, _ := json.Marshal(map[string]string{
		"from":   from,
		"to":     to,
		"reason": strings.TrimSpace(reason),
	})
	err := db.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&models.Election{}).
			Where("id = ? AND status = ?", e.ID, from).
			Update("status", to)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrConcurrentTransition
		}
		return tx.Create(&models.ElectionAuditLog{
			ElectionID: e.ID,
			ActorID:    actorID,
			Action:     auditAction(to),
			Details:    string(details),
			IPAddress:  ip,
			CreatedAt:  at,
		}).Error
	})
	if err != nil {
		return nil, err
	}
	// Reload so Version and UpdatedAt describe the row that was written.
	if err := db.Where("id = ?", e.ID).First(e).Error; err != nil {
		return nil, err
	}
	return e, nil
}

func auditAction(to string) string {
	switch to {
	case models.ElectionStatusScheduled:
		return models.AuditActionElectionScheduled
	case models.ElectionStatusOpen:
		return models.AuditActionElectionOpened
	case models.ElectionStatusClosed:
		return models.AuditActionElectionClosed
	case models.ElectionStatusCancelled:
		return models.AuditActionElectionCancelled
	case models.ElectionStatusReview:
		return models.AuditActionElectionReview
	case models.ElectionStatusArchived:
		return models.AuditActionElectionArchived
	default:
		return models.AuditActionElectionUpdated
	}
}

func validCategory(value string) bool {
	switch value {
	case models.ElectionCategoryGuild, models.ElectionCategoryFaculty,
		models.ElectionCategoryDepartment, models.ElectionCategoryProgramme,
		models.ElectionCategoryClass, models.ElectionCategoryClub,
		models.ElectionCategoryAssociation, models.ElectionCategoryCommittee,
		models.ElectionCategoryReferendum:
		return true
	}
	return false
}

func validResultPolicy(value string) bool {
	switch value {
	case models.ResultPolicyLive, models.ResultPolicyAfterClose, models.ResultPolicyCertifiedOnly:
		return true
	}
	return false
}

func validQuestionType(value string) bool {
	switch value {
	case models.QuestionTypeSingleChoice, models.QuestionTypeMultiChoice,
		models.QuestionTypeYesNo, models.QuestionTypeReferendum:
		return true
	}
	return false
}

func validDomains(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return true // roster-only eligibility, decided by ISSUE-04
	}
	for _, domain := range strings.Split(value, ",") {
		domain = strings.TrimSpace(strings.TrimPrefix(domain, "@"))
		if domain == "" || strings.ContainsAny(domain, " @,") {
			return false
		}
	}
	return true
}
