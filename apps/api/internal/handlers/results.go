package handlers

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"campvote/apps/api/internal/models"
	"campvote/apps/api/internal/policy"
	"campvote/apps/api/internal/respond"
	"campvote/apps/api/internal/tally"
)

// ResultsHandler is ISSUE-09: the count, the snapshot it is pinned to, and the
// chain of records that lets a certified result be trusted and corrected
// without ever being edited in place.
//
// Everything a results screen shows comes from a stored snapshot. The server
// computes shares, outcomes and turnout; no client arithmetic decides meaning.
type ResultsHandler struct {
	DB *gorm.DB
	// Now is injectable for boundary tests. Never read from a request.
	Now func() time.Time
}

func NewResultsHandler(db *gorm.DB) *ResultsHandler {
	return &ResultsHandler{DB: db, Now: time.Now}
}

func (h *ResultsHandler) now() time.Time {
	if h.Now == nil {
		return time.Now().UTC()
	}
	return h.Now().UTC()
}

// snapshotView is what every results surface renders: a snapshot and its
// content, with no identity of any voter anywhere in it.
type snapshotView struct {
	ID             string                 `json:"id"`
	Version        int                    `json:"version"`
	Status         string                 `json:"status"`
	ContentHash    string                 `json:"content_hash"`
	ComputedAt     time.Time              `json:"computed_at"`
	ComputedByID   string                 `json:"computed_by_id"`
	CertifiedByID  string                 `json:"certified_by_id,omitempty"`
	CertifiedAt    *time.Time             `json:"certified_at,omitempty"`
	CertifyReason  string                 `json:"certify_reason,omitempty"`
	SupersedesID   *string                `json:"supersedes_id,omitempty"`
	SupersededByID *string                `json:"superseded_by_id,omitempty"`
	EligibleVoters int                    `json:"eligible_voters"`
	BallotsCast    int                    `json:"ballots_cast"`
	TurnoutPercent float64                `json:"turnout_percent"`
	Turnout        tally.Turnout          `json:"turnout"`
	Positions      []tally.PositionResult `json:"positions"`
	BallotsSpoiled int                    `json:"ballots_spoiled"`
}

func viewOf(snapshot models.ResultSnapshot) (snapshotView, bool) {
	var content tally.Result
	if err := json.Unmarshal([]byte(snapshot.ContentJSON), &content); err != nil {
		return snapshotView{}, false
	}
	view := snapshotView{
		ID:             snapshot.ID,
		Version:        snapshot.Version,
		Status:         snapshot.Status,
		ContentHash:    snapshot.ContentHash,
		ComputedAt:     snapshot.ComputedAt,
		ComputedByID:   snapshot.ComputedByID,
		CertifiedByID:  snapshot.CertifiedByID,
		CertifyReason:  snapshot.CertifyReason,
		SupersedesID:   snapshot.SupersedesID,
		SupersededByID: snapshot.SupersededByID,
		EligibleVoters: snapshot.EligibleVoters,
		BallotsCast:    snapshot.BallotsCast,
		TurnoutPercent: snapshot.TurnoutPercent,
		Turnout:        content.Turnout,
		Positions:      content.Positions,
		BallotsSpoiled: content.BallotsSpoiled,
	}
	if !snapshot.CertifiedAt.IsZero() {
		at := snapshot.CertifiedAt
		view.CertifiedAt = &at
	}
	return view, true
}

// compute tallies the election's stored ballots against its positions.
func (h *ResultsHandler) compute(electionID string) (tally.Result, error) {
	var eligible int64
	if err := h.DB.Model(&models.ElectionRoster{}).
		Where("election_id = ? AND status = ?", electionID, models.RosterStatusEligible).
		Count(&eligible).Error; err != nil {
		return tally.Result{}, err
	}

	var positions []models.Position
	if err := h.DB.Where("election_id = ?", electionID).
		Preload("Candidates").Order("display_order ASC, id ASC").
		Find(&positions).Error; err != nil {
		return tally.Result{}, err
	}
	rules := make([]tally.PositionRule, 0, len(positions))
	for _, position := range positions {
		rule := tally.PositionRule{
			ID: position.ID, Title: position.Title, QuestionType: position.QuestionType,
			Seats: position.Seats, DisplayOrder: position.DisplayOrder,
		}
		for _, candidate := range position.Candidates {
			rule.Candidates = append(rule.Candidates, tally.CandidateRef{
				ID: candidate.ID, Name: candidate.FullName,
			})
		}
		rules = append(rules, rule)
	}

	// Only aggregate fields are loaded. Ballot rows carry no identity, so
	// there is nothing here that could link a voter to a choice.
	var stored []models.AnonymousBallot
	if err := h.DB.Where("election_id = ?", electionID).
		Preload("Selections").Find(&stored).Error; err != nil {
		return tally.Result{}, err
	}

	ballots := make([]tally.Ballot, 0, len(stored))
	for _, row := range stored {
		ballot := tally.Ballot{ID: row.ID, Status: row.Status}
		for _, selection := range row.Selections {
			s := tally.Selection{PositionID: selection.PositionID, Choice: selection.Choice}
			if selection.CandidateID != nil {
				s.CandidateID = *selection.CandidateID
			}
			ballot.Selections = append(ballot.Selections, s)
		}
		ballots = append(ballots, ballot)
	}

	return tally.Compute(tally.Input{
		ElectionID:     electionID,
		EligibleVoters: int(eligible),
		Positions:      rules,
		Ballots:        ballots,
	}), nil
}

// election loads the election by id. A missing election writes 404. It also
// returns the caller's authority, which is nil when the caller has no
// relationship with this election: a voter has none, and that is not an error
// for reading published results.
func (h *ResultsHandler) election(c *gin.Context) (models.Election, *policy.Authority, bool) {
	var election models.Election
	if err := h.DB.Where("id = ?", c.Param("id")).First(&election).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respond.NotFound(c, "Election not found")
			return models.Election{}, nil, false
		}
		respond.Internal(c, err)
		return models.Election{}, nil, false
	}
	authority, err := policy.AuthorityFor(h.DB, policy.ActorFrom(c), election.ID)
	if errors.Is(err, policy.ErrNotVisible) {
		// A scoped campus role — official, candidate, auditor — with no
		// assignment for this election cannot see it at all, so it is 404 and
		// never a hint that the id exists. A plain voter has no relationship
		// with any election, and Read serves voters, so theirs is not a
		// failure: the publication policy inside decides what they see.
		if scopedCampusRole(policy.ActorFrom(c).Role) {
			respond.NotFound(c, "Election not found")
			return models.Election{}, nil, false
		}
		return election, nil, true
	}
	if err != nil {
		respond.Internal(c, err)
		return models.Election{}, nil, false
	}
	return election, authority, true
}

// scopedCampusRole reports whether the actor's own role is one that is only
// ever meaningful with a per-election assignment.
func scopedCampusRole(role string) bool {
	switch role {
	case models.RoleElectionOfficial, models.RoleCandidate, models.RoleElectionAuditor:
		return true
	}
	return false
}

// staffOnly writes the refusal unless the caller may act on results at all.
func staffOnly(c *gin.Context, authority *policy.Authority, ok bool, electionID string) bool {
	if ok && authority != nil && (authority.Can(policy.PermResultsView) || authority.Can(policy.PermResultsViewLive)) {
		return true
	}
	policy.LogDenial(nil, c, "requires results.view", electionID)
	respond.Forbidden(c, "You are not authorised to read results for this election")
	return false
}

// RunTally POST /api/v1/elections/:id/results/tally
//
// Post-close only. A first tally is PROVISIONAL. If a snapshot has already
// been certified, a new tally is a correction: it requires a reason, is stored
// as a new version, and marks the certified one SUPERSEDED while keeping it
// visible. Ballots are never touched.
func (h *ResultsHandler) RunTally(c *gin.Context) {
	election, authority, ok := h.election(c)
	if !ok {
		return
	}
	if !authority.Can(policy.PermElectionEdit) {
		policy.LogDenial(h.DB, c, "requires "+policy.PermElectionEdit, election.ID)
		respond.Forbidden(c, "You do not have permission to run a tally for this election")
		return
	}
	if !closedOrLater(election.Status) {
		respond.Conflict(c, "A tally can only be run after voting has closed")
		return
	}

	var body struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&body)

	var certified models.ResultSnapshot
	certifiedErr := h.DB.Where("election_id = ? AND status = ?",
		election.ID, models.ResultStatusCertified).
		Order("version DESC").First(&certified).Error
	isCorrection := certifiedErr == nil
	if isCorrection && len(body.Reason) == 0 {
		respond.Validation(c, "A correction to certified results requires a stated reason",
			map[string]string{"reason": "required"})
		return
	}

	result, err := h.compute(election.ID)
	if err != nil {
		respond.Internal(c, err)
		return
	}
	payload, err := json.Marshal(result)
	if err != nil {
		respond.Internal(c, err)
		return
	}

	latest, _ := h.latestVersion(election.ID)
	snapshot := models.ResultSnapshot{
		ElectionID:     election.ID,
		Version:        latest + 1,
		Status:         models.ResultStatusProvisional,
		ContentHash:    tally.Hash(result),
		ContentJSON:    string(payload),
		EligibleVoters: result.Turnout.EligibleVoters,
		BallotsCast:    result.Turnout.BallotsCast,
		TurnoutPercent: result.Turnout.Percent,
		ComputedByID:   policy.ActorFrom(c).UserID,
	}
	if isCorrection {
		snapshot.SupersedesID = &certified.ID
	}

	err = h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&snapshot).Error; err != nil {
			return err
		}
		if isCorrection {
			if err := tx.Model(&models.ResultSnapshot{}).Where("id = ?", certified.ID).
				Update("superseded_by_id", snapshot.ID).Error; err != nil {
				return err
			}
			if err := tx.Model(&models.ResultSnapshot{}).Where("id = ?", certified.ID).
				Update("status", models.ResultStatusSuperseded).Error; err != nil {
				return err
			}
		}
		details, _ := json.Marshal(map[string]any{
			"snapshot_id": snapshot.ID,
			"version":     snapshot.Version,
			"hash":        snapshot.ContentHash,
			"reason":      body.Reason,
		})
		action := models.AuditActionTallyRun
		if isCorrection {
			action = models.AuditActionResultsCorrected
		}
		return tx.Create(&models.ElectionAuditLog{
			ElectionID: election.ID,
			ActorID:    policy.ActorFrom(c).UserID,
			Action:     action,
			Details:    string(details),
			IPAddress:  c.ClientIP(),
		}).Error
	})
	if err != nil {
		respond.Internal(c, err)
		return
	}
	respond.Created(c, gin.H{
		"election_id": election.ID,
		"snapshot":    mustView(snapshot, result),
		"correction":  isCorrection,
	}, "Tally recorded")
}

// Read GET /api/v1/elections/:id/results
//
// One handler serves the officer workspace and the voter/public page. The
// election's publication policy is applied to every caller — an officer is not
// exempt from a CERTIFIED_ONLY election that has not been certified — and
// before it allows publication the answer carries no result data at all.
//
// While an election is still open its tally is provisional, and provisional
// figures are restricted to roles holding results.view_live. Once the policy
// allows publication, any signed-in caller sees the same published result; the
// only difference left is which snapshot they are shown.
func (h *ResultsHandler) Read(c *gin.Context) {
	election, authority, ok := h.election(c)
	if !ok {
		return
	}
	// Holding results.view is enough to read a published result. A tally that
	// is still provisional — the election is open — needs results.view_live,
	// which a candidate deliberately does not hold.
	provisionalReader := authority != nil && authority.Can(policy.PermResultsViewLive)
	staff := provisionalReader || (authority != nil && authority.Can(policy.PermResultsView))

	visible, reason := policy.ResultsVisible(election, h.now())
	if !visible {
		policy.LogDenial(h.DB, c, reason, election.ID)
		respond.Forbidden(c, reason)
		return
	}
	if election.Status == models.ElectionStatusOpen && !provisionalReader {
		policy.LogDenial(h.DB, c, "Provisional results are restricted to authorised election roles", election.ID)
		respond.Forbidden(c, "Provisional results are restricted to authorised election roles")
		return
	}

	snapshot, err := h.currentSnapshot(election.ID, staff)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respond.OK(c, gin.H{
				"election_id": election.ID,
				"available":   false,
				"label":       "UNAVAILABLE",
				"reason":      "Results have not been tallied for this election yet",
			})
			return
		}
		respond.Internal(c, err)
		return
	}
	view, ok := viewOf(snapshot)
	if !ok {
		respond.Internal(c, errors.New("stored tally content could not be read"))
		return
	}
	respond.OK(c, gin.H{
		"election_id":     election.ID,
		"election_status": election.Status,
		"available":       true,
		"label":           snapshot.Status,
		"snapshot":        view,
	})
}

// History GET /api/v1/elections/:id/results/history
//
// Every snapshot of this election, newest first, so a correction shows the
// original and the superseding version side by side. Nothing here can be
// edited or deleted from this API.
func (h *ResultsHandler) History(c *gin.Context) {
	election, authority, ok := h.election(c)
	if !staffOnly(c, authority, ok, election.ID) {
		return
	}
	var snapshots []models.ResultSnapshot
	if err := h.DB.Where("election_id = ?", election.ID).
		Order("version DESC").Find(&snapshots).Error; err != nil {
		respond.Internal(c, err)
		return
	}
	var recounts []models.ResultRecount
	if err := h.DB.Where("election_id = ?", election.ID).
		Order("run_at DESC").Find(&recounts).Error; err != nil {
		respond.Internal(c, err)
		return
	}

	views := make([]snapshotView, 0, len(snapshots))
	for _, snapshot := range snapshots {
		if view, ok := viewOf(snapshot); ok {
			views = append(views, view)
		}
	}
	respond.OK(c, gin.H{
		"election_id": election.ID,
		"snapshots":   views,
		"recounts":    recounts,
	})
}

// Recount POST /api/v1/elections/:id/results/recount
//
// Re-runs the tally from stored ballots and compares it to a named snapshot.
// It writes a comparison record and never mutates ballots or the snapshot.
func (h *ResultsHandler) Recount(c *gin.Context) {
	election, authority, ok := h.election(c)
	if !ok {
		return
	}
	if !authority.Can(policy.PermElectionEdit) {
		policy.LogDenial(h.DB, c, "requires "+policy.PermElectionEdit, election.ID)
		respond.Forbidden(c, "You do not have permission to run a recount for this election")
		return
	}
	if !closedOrLater(election.Status) {
		respond.Conflict(c, "A recount can only be run after voting has closed")
		return
	}

	var body struct {
		SnapshotID string `json:"snapshot_id"`
		Reason     string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&body)

	comparison, err := h.snapshotByID(election.ID, body.SnapshotID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respond.NotFound(c, "Snapshot not found for this election")
			return
		}
		respond.Internal(c, err)
		return
	}
	var prior tally.Result
	if err := json.Unmarshal([]byte(comparison.ContentJSON), &prior); err != nil {
		respond.Internal(c, err)
		return
	}

	recounted, err := h.compute(election.ID)
	if err != nil {
		respond.Internal(c, err)
		return
	}
	diff := tally.Compare(prior, recounted)
	details, _ := json.Marshal(diff)

	record := models.ResultRecount{
		ElectionID:         election.ID,
		ComparedSnapshotID: comparison.ID,
		RecomputedHash:     tally.Hash(recounted),
		Identical:          diff.Identical,
		Differences:        string(details),
		Reason:             body.Reason,
		RunByID:            policy.ActorFrom(c).UserID,
	}
	if err := h.DB.Create(&record).Error; err != nil {
		respond.Internal(c, err)
		return
	}
	auditDetails, _ := json.Marshal(map[string]any{
		"snapshot_id": comparison.ID,
		"identical":   diff.Identical,
		"hash":        record.RecomputedHash,
		"reason":      body.Reason,
	})
	_ = h.DB.Create(&models.ElectionAuditLog{
		ElectionID: election.ID,
		ActorID:    policy.ActorFrom(c).UserID,
		Action:     models.AuditActionRecountTriggered,
		Details:    string(auditDetails),
		IPAddress:  c.ClientIP(),
	}).Error

	respond.OK(c, gin.H{
		"recount":            record,
		"comparison_version": comparison.Version,
		"identical":          diff.Identical,
		"differences":        diff,
		"ballots_unchanged":  true,
	})
}

// Certify POST /api/v1/elections/:id/results/certify
//
// Pins a specific snapshot: hash, timestamp, certifier, reason. Separation of
// duties comes from policy.CertifyDecision. The snapshot is then immutable —
// a later correction is a new version that supersedes it, never an edit.
func (h *ResultsHandler) Certify(c *gin.Context) {
	election, authority, ok := h.election(c)
	if !ok {
		return
	}
	actor := policy.ActorFrom(c)
	allowed, reason := policy.CertifyDecision(h.DB, authority, actor)
	if !allowed {
		policy.LogDenial(h.DB, c, reason, election.ID)
		respond.Forbidden(c, reason)
		return
	}

	var body struct {
		SnapshotID string `json:"snapshot_id"`
		Reason     string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || len(body.Reason) == 0 {
		respond.Validation(c, "Certification requires a stated reason",
			map[string]string{"reason": "required"})
		return
	}

	snapshot, err := h.snapshotByID(election.ID, body.SnapshotID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respond.NotFound(c, "Snapshot not found for this election")
			return
		}
		respond.Internal(c, err)
		return
	}
	if snapshot.Status != models.ResultStatusProvisional {
		respond.Conflict(c, "Only a provisional snapshot can be certified")
		return
	}
	latest, err := h.latestVersion(election.ID)
	if err != nil || snapshot.Version != latest {
		respond.Conflict(c, "A newer tally exists; certify the current snapshot")
		return
	}

	err = h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.ResultSnapshot{}).
			Where("id = ? AND status = ?", snapshot.ID, models.ResultStatusProvisional).
			Updates(map[string]any{
				"status":          models.ResultStatusCertified,
				"certified_by_id": actor.UserID,
				"certified_at":    h.now(),
				"certify_reason":  body.Reason,
			}).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.Election{}).Where("id = ?", election.ID).
			Update("status", models.ElectionStatusFinalized).Error; err != nil {
			return err
		}
		details, _ := json.Marshal(map[string]any{
			"snapshot_id": snapshot.ID,
			"version":     snapshot.Version,
			"hash":        snapshot.ContentHash,
			"reason":      body.Reason,
		})
		return tx.Create(&models.ElectionAuditLog{
			ElectionID: election.ID,
			ActorID:    actor.UserID,
			Action:     models.AuditActionResultsCertified,
			Details:    string(details),
			IPAddress:  c.ClientIP(),
		}).Error
	})
	if err != nil {
		respond.Internal(c, err)
		return
	}
	snapshot.Status = models.ResultStatusCertified
	snapshot.CertifiedByID = actor.UserID
	snapshot.CertifiedAt = h.now()
	snapshot.CertifyReason = body.Reason
	view, _ := viewOf(snapshot)
	respond.OK(c, gin.H{
		"election_id": election.ID,
		"status":      models.ElectionStatusFinalized,
		"snapshot":    view,
	}, "Results certified")
}

// currentSnapshot picks what this caller may see: staff see the newest live
// snapshot, and everyone else sees only what the publication policy permits.
func (h *ResultsHandler) currentSnapshot(electionID string, staff bool) (models.ResultSnapshot, error) {
	query := h.DB.Where("election_id = ? AND superseded_by_id IS NULL", electionID)
	if !staff {
		query = query.Where("status = ?", models.ResultStatusCertified)
	}
	var snapshot models.ResultSnapshot
	err := query.Order("version DESC").First(&snapshot).Error
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) && !staff {
		// A voter may still see a provisional result when the election allows
		// publication after close, as long as one exists.
		return h.snapshotByStatus(electionID, models.ResultStatusProvisional)
	}
	return snapshot, err
}

func (h *ResultsHandler) snapshotByStatus(electionID, status string) (models.ResultSnapshot, error) {
	var snapshot models.ResultSnapshot
	err := h.DB.Where("election_id = ? AND status = ?", electionID, status).
		Order("version DESC").First(&snapshot).Error
	return snapshot, err
}

func (h *ResultsHandler) snapshotByID(electionID, snapshotID string) (models.ResultSnapshot, error) {
	var snapshot models.ResultSnapshot
	query := h.DB.Where("election_id = ?", electionID)
	if snapshotID != "" {
		query = query.Where("id = ?", snapshotID)
	} else {
		query = query.Where("superseded_by_id IS NULL").Order("version DESC")
	}
	err := query.First(&snapshot).Error
	return snapshot, err
}

func (h *ResultsHandler) latestVersion(electionID string) (int, error) {
	var latest models.ResultSnapshot
	err := h.DB.Where("election_id = ?", electionID).
		Order("version DESC").First(&latest).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, nil
		}
		return 0, err
	}
	return latest.Version, nil
}

// closedOrLater reports whether voting has ended for this election.
func closedOrLater(status string) bool {
	switch status {
	case models.ElectionStatusClosed, models.ElectionStatusReview,
		models.ElectionStatusFinalized, models.ElectionStatusArchived:
		return true
	}
	return false
}

func mustView(snapshot models.ResultSnapshot, result tally.Result) snapshotView {
	view, ok := viewOf(snapshot)
	if ok {
		return view
	}
	return snapshotView{
		ID: snapshot.ID, Version: snapshot.Version, Status: snapshot.Status,
		ContentHash: snapshot.ContentHash, ComputedAt: snapshot.ComputedAt,
		EligibleVoters: result.Turnout.EligibleVoters, BallotsCast: result.Turnout.BallotsCast,
		TurnoutPercent: result.Turnout.Percent, Turnout: result.Turnout,
		Positions: result.Positions, BallotsSpoiled: result.BallotsSpoiled,
	}
}
