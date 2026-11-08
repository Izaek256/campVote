package handlers

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"campvote/apps/api/internal/eligibility"
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
	Phase              string          `json:"phase"`
	ServerTime         time.Time       `json:"server_time"`
	AcceptingVotes     bool            `json:"accepting_votes"`
	DisplayTimezone    string          `json:"display_timezone"`
	AllowedTransitions []string        `json:"allowed_transitions"`
	MyVoteStatus       *voteStatusView `json:"my_vote_status,omitempty"`
}

// voteStatusView is the caller's own participation state for one election. It
// is display data only: the server still enforces every rule.
type voteStatusView struct {
	Voted      bool       `json:"voted"`
	RecordedAt *time.Time `json:"recorded_at,omitempty"`
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
	// New eligibility fields (replaces AllowedDomains)
	AllowedDomain  string `json:"allowed_domain"`
	AllowedYears   string `json:"allowed_years"`
	AllowedCourses string `json:"allowed_courses"`
	// PublicListing opts the election into the public landing page's upcoming
	// list. Off by default: a draft is private until an administrator says
	// otherwise.
	PublicListing bool `json:"public_listing"`
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
		AllowedDomain:  strings.ToLower(strings.TrimPrefix(strings.TrimSpace(in.AllowedDomain), "@")),
		AllowedYears:   strings.TrimSpace(in.AllowedYears),
		AllowedCourses: strings.ToLower(strings.TrimSpace(in.AllowedCourses)),
		PublicListing:  in.PublicListing,
		CreatedByID:    c.GetString("user_id"),
	}
	if election.Category == "" {
		election.Category = models.ElectionCategoryGuild
	}
	if election.OrganizingUnit == "" {
		election.OrganizingUnit = "Electoral Commission"
	}
	if election.ResultPolicy == "" {
		election.ResultPolicy = models.ResultPolicyAfterClose
	}
	// A blank domain means "any student with a parseable registration email".
	// Never default it to one campus domain: a wrong default silently hides the
	// election from every voter whose domain differs.
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

// List returns the elections a signed-in caller may see.
//
// Visibility here is deliberately wider than the right to vote. A draft is
// shown to the students it will admit, so the candidates and their manifestos
// can be read before polls open. A closed election is shown to every signed-in
// student, not only those who could have voted in it, so the outcome is
// transparent across campus. Whether a caller may actually cast a ballot is
// still decided per election by the ballot handler.
func (h *ElectionLifecycleHandler) List(c *gin.Context) {
	actor := policy.ActorFrom(c)
	var elections []models.Election
	if err := h.DB.Order("starts_at DESC").Limit(100).Find(&elections).Error; err != nil {
		respond.Internal(c, err)
		return
	}
	now := h.now()

	// Parse the caller's email once for eligibility filtering.
	var callerID eligibility.Identity
	var callerParsed bool
	if !actor.Admin {
		var user models.User
		if err := h.DB.Where("id = ?", actor.UserID).First(&user).Error; err == nil {
			callerID, callerParsed = eligibility.ParseEmail(user.Email)
		}
	}

	out := make([]electionView, 0, len(elections))
	for i := range elections {
		if err := lifecycle.Ensure(h.DB, &elections[i], now); err != nil {
			respond.Internal(c, err)
			return
		}
		if !actor.Admin && !h.visibleTo(&elections[i], actor.UserID, callerID, callerParsed) {
			continue
		}
		out = append(out, h.view(&elections[i], now))
	}
	h.attachVoteStatus(c, out)
	respond.OK(c, out)
}

// visibleTo decides whether a signed-in non-admin caller may see one election.
// A caller holding a role on the election always may. A closed election is
// public to every signed-in student. A draft is previewed only by the students
// its rule admits, and an open election keeps the established behaviour: shown
// to the students the rule admits, or to everyone when the caller's own
// address is one the domain parser does not recognise.
func (h *ElectionLifecycleHandler) visibleTo(
	election *models.Election, userID string,
	callerID eligibility.Identity, callerParsed bool,
) bool {
	return electionVisibleTo(h.DB, election, userID, callerID, callerParsed)
}

// electionVisibleTo is the single visibility rule for one election, shared by
// the list and by the downloadable information notice so the two cannot drift.
func electionVisibleTo(
	db *gorm.DB, election *models.Election, userID string,
	callerID eligibility.Identity, callerParsed bool,
) bool {
	if election.Status == models.ElectionStatusClosed {
		return true
	}
	var roleCount int64
	db.Model(&models.ElectionRole{}).
		Where("election_id = ? AND user_id = ?", election.ID, userID).
		Count(&roleCount)
	if roleCount > 0 {
		return true
	}
	if election.Status == models.ElectionStatusDraft {
		return callerParsed && eligibility.RuleAllows(election, callerID)
	}
	return !callerParsed || eligibility.RuleAllows(election, callerID)
}

// attachVoteStatus adds the caller's own participation flag to the list. It is
// the only identity-linked field in the response and carries no choice data.
func (h *ElectionLifecycleHandler) attachVoteStatus(c *gin.Context, elections []electionView) {
	userID := c.GetString("user_id")
	if userID == "" || len(elections) == 0 {
		return
	}
	ids := make([]string, 0, len(elections))
	for _, election := range elections {
		ids = append(ids, election.ID)
	}
	var rows []models.VoterParticipation
	if err := h.DB.Where("user_id = ? AND election_id IN ?", userID, ids).Find(&rows).Error; err != nil {
		return
	}
	byElection := make(map[string]models.VoterParticipation, len(rows))
	for _, row := range rows {
		byElection[row.ElectionID] = row
	}
	for i := range elections {
		status := &voteStatusView{}
		if row, ok := byElection[elections[i].ID]; ok {
			status.Voted = true
			coarse := row.VotedAt.UTC().Truncate(15 * time.Minute)
			status.RecordedAt = &coarse
		}
		elections[i].MyVoteStatus = status
	}
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

// Delete removes an election whose voting has not started. An election with
// any stored ballot is refused: those ballots are the evidence of a contest,
// and deleting the election would orphan them. Close it instead.
func (h *ElectionLifecycleHandler) Delete(c *gin.Context) {
	authority, ok := h.authority(c, policy.PermElectionEdit)
	if !ok {
		return
	}
	election := authority.Election
	now := h.now()
	if err := lifecycle.Ensure(h.DB, &election, now); err != nil {
		respond.Internal(c, err)
		return
	}

	var ballots int64
	if err := h.DB.Model(&models.AnonymousBallot{}).
		Where("election_id = ?", election.ID).Count(&ballots).Error; err != nil {
		respond.Internal(c, err)
		return
	}
	if ballots > 0 {
		respond.Conflict(c, "This election already has recorded ballots and cannot be deleted; close it instead")
		return
	}

	details, _ := json.Marshal(map[string]string{
		"title":  election.Title,
		"status": election.Status,
	})
	err := h.DB.Transaction(func(tx *gorm.DB) error {
		// Record the deletion before the election leaves the live table.
		if err := tx.Create(&models.ElectionAuditLog{
			ElectionID: election.ID,
			ActorID:    c.GetString("user_id"),
			Action:     models.AuditActionElectionDeleted,
			Details:    string(details),
			IPAddress:  c.ClientIP(),
			CreatedAt:  now,
		}).Error; err != nil {
			return err
		}
		if err := tx.Where("election_id = ?", election.ID).Delete(&models.Candidate{}).Error; err != nil {
			return err
		}
		if err := tx.Where("election_id = ?", election.ID).Delete(&models.Position{}).Error; err != nil {
			return err
		}
		return tx.Delete(&models.Election{}, "id = ?", election.ID).Error
	})
	if err != nil {
		respond.Internal(c, err)
		return
	}
	respond.OK(c, gin.H{"id": election.ID, "message": "Election deleted"})
}

// Readiness reports whether the election can be opened. The reasons are
// safe to show to an official: field-level configuration problems, never voter
// identities.
func (h *ElectionLifecycleHandler) Readiness(c *gin.Context) {
	authority, ok := h.authority(c, policy.PermElectionEdit)
	if !ok {
		return
	}
	now := h.now()
	if err := lifecycle.Ensure(h.DB, &authority.Election, now); err != nil {
		respond.Internal(c, err)
		return
	}
	ready := true
	problems := map[string]string{}
	if err := lifecycle.ValidateConfiguration(h.DB, &authority.Election, now, true); err != nil {
		ready = false
		var configErr *lifecycle.ConfigError
		if errors.As(err, &configErr) {
			problems = configErr.Fields
		} else {
			problems["configuration"] = err.Error()
		}
	}
	respond.OK(c, gin.H{
		"ready":    ready,
		"status":   authority.Election.Status,
		"problems": problems,
	})
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
	AllowedDomain  *string `json:"allowed_domain"`
	AllowedYears   *string `json:"allowed_years"`
	AllowedCourses *string `json:"allowed_courses"`
	PublicListing  *bool   `json:"public_listing"`
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
	if in.AllowedDomain != nil {
		updated.AllowedDomain = strings.ToLower(strings.TrimPrefix(strings.TrimSpace(*in.AllowedDomain), "@"))
	}
	if in.AllowedYears != nil {
		updated.AllowedYears = strings.TrimSpace(*in.AllowedYears)
	}
	if in.AllowedCourses != nil {
		updated.AllowedCourses = strings.ToLower(strings.TrimSpace(*in.AllowedCourses))
	}
	if in.PublicListing != nil {
		updated.PublicListing = *in.PublicListing
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

// Transition applies an explicit state change. Opening is deliberately not
// reachable here: only the server clock opens an election at its start time,
// so a client can never force a contest open early.
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

	if !lifecycle.KnownStatus(to) || !lifecycle.CanTransition(authority.Election.Status, to) {
		respond.Conflict(c, "Illegal election transition from "+authority.Election.Status+" to "+to)
		return
	}
	if to == models.ElectionStatusOpen {
		respond.Conflict(c, "Elections open automatically at their start time")
		return
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
	if before.AllowedDomain != after.AllowedDomain {
		changed = append(changed, "allowed_domain")
	}
	if before.AllowedYears != after.AllowedYears {
		changed = append(changed, "allowed_years")
	}
	if before.AllowedCourses != after.AllowedCourses {
		changed = append(changed, "allowed_courses")
	}
	if before.PublicListing != after.PublicListing {
		changed = append(changed, "public_listing")
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
		"allowed_domain":  e.AllowedDomain,
		"allowed_years":   e.AllowedYears,
		"allowed_courses": e.AllowedCourses,
		"public_listing":  e.PublicListing,
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
