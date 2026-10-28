package handlers

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"campvote/apps/api/internal/lifecycle"
	"campvote/apps/api/internal/models"
	"campvote/apps/api/internal/policy"
	"campvote/apps/api/internal/respond"
)

// ElectionLifecycleHandler owns creating and editing election configuration and
// the explicit state transitions. Opening and closing by clock are done by
// lifecycle.Ensure and the scheduler, never by a client-supplied time.
type ElectionLifecycleHandler struct {
	DB  *gorm.DB
	Now func() time.Time
}

func NewElectionLifecycleHandler(db *gorm.DB) *ElectionLifecycleHandler {
	return &ElectionLifecycleHandler{DB: db, Now: time.Now}
}

func (h *ElectionLifecycleHandler) now() time.Time {
	if h.Now == nil {
		return time.Now().UTC()
	}
	return h.Now().UTC()
}

type electionView struct {
	models.Election
	Phase              string    `json:"phase"`
	ServerTime         time.Time `json:"server_time"`
	AcceptingVotes     bool      `json:"accepting_votes"`
	DisplayTimezone    string    `json:"display_timezone"`
	AllowedTransitions []string  `json:"allowed_transitions"`
}

func (h *ElectionLifecycleHandler) view(e *models.Election, now time.Time) electionView {
	return electionView{
		Election:           *e,
		Phase:              e.Status,
		ServerTime:         now.UTC(),
		AcceptingVotes:     e.IsOpen(now),
		DisplayTimezone:    lifecycle.DisplayTimezone,
		AllowedTransitions: lifecycle.AllowedTransitions(e.Status),
	}
}

type createElectionRequest struct {
	Title          string `json:"title"`
	Description    string `json:"description"`
	Category       string `json:"category"`
	OrganizingUnit string `json:"organizing_unit"`
	StartsAt       string `json:"starts_at"`
	EndsAt         string `json:"ends_at"`
	ResultPolicy   string `json:"result_policy"`
	AllowedDomains string `json:"allowed_domains"`
}

// Create writes a new DRAFT election. Scheduling is a separate, validated
// transition so a half-built ballot never reaches voters.
func (h *ElectionLifecycleHandler) Create(c *gin.Context) {
	var in createElectionRequest
	if err := c.ShouldBindJSON(&in); err != nil {
		respond.BadRequest(c, "Election details are required")
		return
	}
	now := h.now()
	startsAt, ok := scheduleField(c, "starts_at", in.StartsAt)
	if !ok {
		return
	}
	endsAt, ok := scheduleField(c, "ends_at", in.EndsAt)
	if !ok {
		return
	}

	election := &models.Election{
		Title:          strings.TrimSpace(in.Title),
		Description:    in.Description,
		Category:       strings.ToUpper(strings.TrimSpace(in.Category)),
		OrganizingUnit: strings.TrimSpace(in.OrganizingUnit),
		Status:         models.ElectionStatusDraft,
		StartsAt:       startsAt,
		EndsAt:         endsAt,
		ResultPolicy:   strings.ToUpper(strings.TrimSpace(in.ResultPolicy)),
		AllowedDomains: strings.TrimSpace(in.AllowedDomains),
		CreatedByID:    c.GetString("user_id"),
	}
	if election.Category == "" {
		election.Category = models.ElectionCategoryGuild
	}
	if election.OrganizingUnit == "" {
		election.OrganizingUnit = "MMU Electoral Commission"
	}
	if election.ResultPolicy == "" {
		election.ResultPolicy = models.ResultPolicyAfterClose
	}
	if election.AllowedDomains == "" {
		election.AllowedDomains = "@std.must.ac.ug,@must.ac.ug"
	}
	if err := lifecycle.ValidateBasics(election, now, false); err != nil {
		respond.WriteError(c, err, "Election configuration is invalid")
		return
	}

	details, _ := json.Marshal(map[string]string{
		"title":     election.Title,
		"starts_at": election.StartsAt.Format(time.RFC3339),
		"ends_at":   election.EndsAt.Format(time.RFC3339),
	})
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(election).Error; err != nil {
			return err
		}
		return tx.Create(&models.ElectionAuditLog{
			ElectionID: election.ID,
			ActorID:    c.GetString("user_id"),
			Action:     models.AuditActionElectionCreated,
			Details:    string(details),
			IPAddress:  c.ClientIP(),
			CreatedAt:  now,
		}).Error
	})
	if err != nil {
		respond.WriteError(c, err, "Election could not be created")
		return
	}
	respond.Created(c, h.view(election, now), "Election created")
}

// Get returns one election and its display-only phase.
func (h *ElectionLifecycleHandler) Get(c *gin.Context) {
	authority, ok := h.authority(c, policy.PermElectionView)
	if !ok {
		return
	}
	now := h.now()
	if err := lifecycle.Ensure(h.DB, &authority.Election, now); err != nil {
		respond.Internal(c, err)
		return
	}
	respond.OK(c, h.view(&authority.Election, now))
}

// Phase exposes the server's phase and clock to any signed-in caller for
// display. It contains no identity, roster or ballot data.
func (h *ElectionLifecycleHandler) Phase(c *gin.Context) {
	var election models.Election
	if err := h.DB.Where("id = ?", c.Param("id")).First(&election).Error; err != nil {
		respond.NotFound(c, "Election not found")
		return
	}
	now := h.now()
	if err := lifecycle.Ensure(h.DB, &election, now); err != nil {
		respond.Internal(c, err)
		return
	}
	respond.OK(c, lifecycle.CurrentPhase(&election, now))
}

type updateElectionRequest struct {
	Title          *string `json:"title"`
	Description    *string `json:"description"`
	Category       *string `json:"category"`
	OrganizingUnit *string `json:"organizing_unit"`
	StartsAt       *string `json:"starts_at"`
	EndsAt         *string `json:"ends_at"`
	ResultPolicy   *string `json:"result_policy"`
	AllowedDomains *string `json:"allowed_domains"`
	Reason         string  `json:"reason"`
}

// Update edits configuration before voting opens. After open, only an audited
// end-time extension is accepted, and it needs a reason and elections.publish.
func (h *ElectionLifecycleHandler) Update(c *gin.Context) {
	authority, ok := h.authority(c, policy.PermElectionEdit)
	if !ok {
		return
	}
	var in updateElectionRequest
	if err := c.ShouldBindJSON(&in); err != nil {
		respond.BadRequest(c, "Election details are required")
		return
	}
	now := h.now()
	if err := lifecycle.Ensure(h.DB, &authority.Election, now); err != nil {
		respond.Internal(c, err)
		return
	}
	current := authority.Election
	updated := current

	if in.Title != nil {
		updated.Title = strings.TrimSpace(*in.Title)
	}
	if in.Description != nil {
		updated.Description = *in.Description
	}
	if in.Category != nil {
		updated.Category = strings.ToUpper(strings.TrimSpace(*in.Category))
	}
	if in.OrganizingUnit != nil {
		updated.OrganizingUnit = strings.TrimSpace(*in.OrganizingUnit)
	}
	if in.ResultPolicy != nil {
		updated.ResultPolicy = strings.ToUpper(strings.TrimSpace(*in.ResultPolicy))
	}
	if in.AllowedDomains != nil {
		updated.AllowedDomains = strings.TrimSpace(*in.AllowedDomains)
	}
	if in.StartsAt != nil {
		startsAt, ok := scheduleField(c, "starts_at", *in.StartsAt)
		if !ok {
			return
		}
		updated.StartsAt = startsAt
	}
	if in.EndsAt != nil {
		endsAt, ok := scheduleField(c, "ends_at", *in.EndsAt)
		if !ok {
			return
		}
		updated.EndsAt = endsAt
	}

	changed := changedElectionFields(current, updated)
	if len(changed) == 0 {
		respond.OK(c, h.view(&current, now), "No changes")
		return
	}

	if lifecycle.IsFrozen(current.Status) {
		h.updateFrozen(c, authority, &current, &updated, changed, strings.TrimSpace(in.Reason), now)
		return
	}

	if err := lifecycle.ValidateBasics(&updated, now, false); err != nil {
		respond.WriteError(c, err, "Election configuration is invalid")
		return
	}
	if current.Status == models.ElectionStatusScheduled {
		// A scheduled election must stay schedulable: its end time cannot be
		// moved into the past, which would silently close it on the next read.
		if err := lifecycle.ValidateConfiguration(h.DB, &updated, now, true); err != nil {
			respond.WriteError(c, err, "Election configuration is invalid")
			return
		}
	}

	values := electionUpdateValues(&updated)
	if err := h.persistUpdate(c, &current, values, models.AuditActionElectionUpdated,
		updateAuditDetails(changed, strings.TrimSpace(in.Reason), current, updated), now); err != nil {
		h.writeUpdateError(c, err)
		return
	}
	h.respondUpdated(c, current.ID, now)
}

func (h *ElectionLifecycleHandler) updateFrozen(c *gin.Context, authority *policy.Authority, current, updated *models.Election, changed []string, reason string, now time.Time) {
	if current.Status != models.ElectionStatusOpen {
		respond.Conflict(c, "Election configuration is frozen and this state cannot be edited")
		return
	}
	onlyEndsAt := len(changed) == 1 && changed[0] == "ends_at"
	extension := onlyEndsAt && updated.EndsAt.After(current.EndsAt)
	if !onlyEndsAt {
		respond.Conflict(c, "Election configuration is frozen after voting opens")
		return
	}
	if !extension {
		respond.Conflict(c, "An open election can only be extended; close it to stop voting")
		return
	}
	if reason == "" {
		respond.BadRequest(c, "A reason is required for a schedule change after voting opens")
		return
	}
	if !authority.Can(policy.PermElectionPublish) {
		policy.LogDenial(h.DB, c, "requires "+policy.PermElectionPublish+" for a post-opening extension", current.ID)
		respond.Forbidden(c, "Only an authorised publisher may extend an open election")
		return
	}
	if !updated.EndsAt.After(now) || !updated.StartsAt.Before(updated.EndsAt) ||
		updated.EndsAt.Sub(updated.StartsAt) < lifecycle.MinimumDuration {
		respond.Validation(c, "The new end time is not a valid schedule", map[string]string{
			"ends_at": "must be after the start time, at least one hour later, and in the future",
		})
		return
	}

	values := map[string]interface{}{"ends_at": updated.EndsAt.UTC()}
	details, _ := json.Marshal(map[string]interface{}{
		"changed_fields": changed,
		"reason":         reason,
		"old_ends_at":    current.EndsAt.UTC().Format(time.RFC3339),
		"new_ends_at":    updated.EndsAt.UTC().Format(time.RFC3339),
		"exception":      true,
	})
	if err := h.persistUpdate(c, current, values, models.AuditActionElectionExtended, string(details), now); err != nil {
		h.writeUpdateError(c, err)
		return
	}
	h.respondUpdated(c, current.ID, now)
}

// Transition applies an explicit state change. Opening by clock and
// certification are deliberately not reachable here.
func (h *ElectionLifecycleHandler) Transition(c *gin.Context) {
	authority, ok := h.authority(c, policy.PermElectionEdit)
	if !ok {
		return
	}
	var in struct {
		To     string `json:"to"`
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		respond.BadRequest(c, "A target state is required")
		return
	}
	now := h.now()
	if err := lifecycle.Ensure(h.DB, &authority.Election, now); err != nil {
		respond.Internal(c, err)
		return
	}
	to := strings.ToUpper(strings.TrimSpace(in.To))
	reason := strings.TrimSpace(in.Reason)

	switch to {
	case models.ElectionStatusOpen:
		respond.Conflict(c, "Elections open automatically at their start time")
		return
	case models.ElectionStatusFinalized:
		respond.Conflict(c, "Certification is a separate audited operation")
		return
	}
	if !lifecycle.KnownStatus(to) || !lifecycle.CanTransition(authority.Election.Status, to) {
		respond.Conflict(c, "Illegal election transition from "+authority.Election.Status+" to "+to)
		return
	}
	if to == models.ElectionStatusCancelled && reason == "" {
		respond.BadRequest(c, "A reason is required to cancel an election")
		return
	}
	if to == models.ElectionStatusCancelled || to == models.ElectionStatusArchived {
		if !authority.Can(policy.PermElectionPublish) {
			policy.LogDenial(h.DB, c, "requires "+policy.PermElectionPublish+" to "+to, authority.Election.ID)
			respond.Forbidden(c, "Only an authorised publisher may cancel or archive an election")
			return
		}
	}
	if to == models.ElectionStatusClosed && now.Before(authority.Election.EndsAt) {
		if reason == "" {
			respond.BadRequest(c, "A reason is required to close an election early")
			return
		}
		if !authority.Can(policy.PermElectionPublish) {
			policy.LogDenial(h.DB, c, "requires "+policy.PermElectionPublish+" to close early", authority.Election.ID)
			respond.Forbidden(c, "Only an authorised publisher may close an election early")
			return
		}
	}

	if _, err := lifecycle.Apply(h.DB, &authority.Election, to,
		c.GetString("user_id"), reason, c.ClientIP(), now); err != nil {
		var configErr *lifecycle.ConfigError
		switch {
		case errors.As(err, &configErr):
			respond.WriteError(c, err, "Election configuration is invalid")
		case errors.Is(err, lifecycle.ErrReasonRequired):
			respond.BadRequest(c, err.Error())
		case errors.Is(err, lifecycle.ErrIllegalTransition), errors.Is(err, lifecycle.ErrConcurrentTransition):
			respond.Conflict(c, err.Error())
		default:
			respond.Internal(c, err)
		}
		return
	}
	var fresh models.Election
	if err := h.DB.Where("id = ?", authority.Election.ID).First(&fresh).Error; err != nil {
		respond.Internal(c, err)
		return
	}
	respond.OK(c, h.view(&fresh, now), "Election state changed")
}

func (h *ElectionLifecycleHandler) persistUpdate(c *gin.Context, current *models.Election, values map[string]interface{}, action, details string, now time.Time) error {
	return h.DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&models.Election{}).
			Where("id = ? AND version = ?", current.ID, current.Version).
			Updates(values)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return lifecycle.ErrConcurrentTransition
		}
		return tx.Create(&models.ElectionAuditLog{
			ElectionID: current.ID,
			ActorID:    c.GetString("user_id"),
			Action:     action,
			Details:    details,
			IPAddress:  c.ClientIP(),
			CreatedAt:  now,
		}).Error
	})
}

func (h *ElectionLifecycleHandler) respondUpdated(c *gin.Context, electionID string, now time.Time) {
	var fresh models.Election
	if err := h.DB.Where("id = ?", electionID).First(&fresh).Error; err != nil {
		respond.Internal(c, err)
		return
	}
	respond.OK(c, h.view(&fresh, now), "Election updated")
}

func (h *ElectionLifecycleHandler) writeUpdateError(c *gin.Context, err error) {
	if errors.Is(err, lifecycle.ErrConcurrentTransition) {
		respond.Conflict(c, "The election changed while this request was being applied; reload and try again")
		return
	}
	respond.WriteError(c, err, "Election could not be updated")
}

func (h *ElectionLifecycleHandler) authority(c *gin.Context, permission string) (*policy.Authority, bool) {
	authority, err := policy.AuthorityFor(h.DB, policy.ActorFrom(c), c.Param("id"))
	if err != nil {
		if errors.Is(err, policy.ErrNotVisible) {
			respond.NotFound(c, "Election not found")
			return nil, false
		}
		respond.Internal(c, err)
		return nil, false
	}
	if !authority.Can(permission) {
		policy.LogDenial(h.DB, c, "requires "+permission, authority.Election.ID)
		respond.Forbidden(c, "You do not have permission to manage this election")
		return nil, false
	}
	return authority, true
}

func scheduleField(c *gin.Context, field, value string) (time.Time, bool) {
	parsed, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(value))
	if err != nil {
		respond.Validation(c, "Use an RFC 3339 timestamp with an explicit offset", map[string]string{
			field: "for example 2026-11-01T09:00:00+03:00 or 2026-11-01T06:00:00Z",
		})
		return time.Time{}, false
	}
	return parsed.UTC(), true
}

func changedElectionFields(before, after models.Election) []string {
	var changed []string
	if before.Title != after.Title {
		changed = append(changed, "title")
	}
	if before.Description != after.Description {
		changed = append(changed, "description")
	}
	if before.Category != after.Category {
		changed = append(changed, "category")
	}
	if before.OrganizingUnit != after.OrganizingUnit {
		changed = append(changed, "organizing_unit")
	}
	if !before.StartsAt.Equal(after.StartsAt) {
		changed = append(changed, "starts_at")
	}
	if !before.EndsAt.Equal(after.EndsAt) {
		changed = append(changed, "ends_at")
	}
	if before.ResultPolicy != after.ResultPolicy {
		changed = append(changed, "result_policy")
	}
	if before.AllowedDomains != after.AllowedDomains {
		changed = append(changed, "allowed_domains")
	}
	return changed
}

func electionUpdateValues(e *models.Election) map[string]interface{} {
	return map[string]interface{}{
		"title":           e.Title,
		"description":     e.Description,
		"category":        e.Category,
		"organizing_unit": e.OrganizingUnit,
		"starts_at":       e.StartsAt.UTC(),
		"ends_at":         e.EndsAt.UTC(),
		"result_policy":   e.ResultPolicy,
		"allowed_domains": e.AllowedDomains,
	}
}

func updateAuditDetails(changed []string, reason string, before, after models.Election) string {
	details, _ := json.Marshal(map[string]interface{}{
		"changed_fields": changed,
		"reason":         reason,
		"old_starts_at":  before.StartsAt.UTC().Format(time.RFC3339),
		"new_starts_at":  after.StartsAt.UTC().Format(time.RFC3339),
		"old_ends_at":    before.EndsAt.UTC().Format(time.RFC3339),
		"new_ends_at":    after.EndsAt.UTC().Format(time.RFC3339),
	})
	return string(details)
}
