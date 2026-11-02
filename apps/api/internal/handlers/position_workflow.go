package handlers

import (
	"errors"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"campvote/apps/api/internal/eligibility"
	"campvote/apps/api/internal/models"
	"campvote/apps/api/internal/respond"
	"campvote/apps/api/internal/services"
)

// errBadPositionStatus is returned for a status that is not OPEN or CLOSED.
var errBadPositionStatus = errors.New("status must be OPEN or CLOSED")

// normalizePositionStatus maps the caller's value onto a known position status.
// An empty value means OPEN: a new position accepts applications immediately.
func normalizePositionStatus(value string) (string, error) {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "", models.PositionStatusOpen:
		return models.PositionStatusOpen, nil
	case models.PositionStatusClosed:
		return models.PositionStatusClosed, nil
	default:
		return "", errBadPositionStatus
	}
}

// normalizeDomain lower-cases an email domain and drops a leading "@", so
// "@std.mast.ac.ug" and "STD.MAST.AC.UG" compare equal.
func normalizeDomain(value string) string {
	return strings.ToLower(strings.TrimPrefix(strings.TrimSpace(value), "@"))
}

// validatePositionRules checks the contest shape shared by every position
// create path, and returns a ready-to-show message, or "" when the input is valid.
func validatePositionRules(questionType string, minSelections, maxSelections, seats, maxCandidates int) string {
	validTypes := map[string]bool{
		models.QuestionTypeSingleChoice: true,
		models.QuestionTypeMultiChoice:  true,
		models.QuestionTypeYesNo:        true,
		models.QuestionTypeReferendum:   true,
	}
	if !validTypes[questionType] {
		return "Choose a valid question type"
	}
	if minSelections < 0 {
		return "Minimum selections cannot be negative"
	}
	if maxSelections < minSelections {
		return "Maximum selections must be at least minimum selections"
	}
	if seats < 1 {
		return "Seats must be at least 1"
	}
	if maxCandidates < 0 {
		return "Maximum candidates cannot be negative"
	}
	return ""
}

// positionWithMeta is a position plus the counts the admin list shows.
type positionWithMeta struct {
	models.Position
	ElectionTitle    string `json:"election_title,omitempty"`
	ApplicationCount int64  `json:"application_count"`
	ApprovedCount    int64  `json:"approved_count"`
}

// eligiblePosition is a position as an applicant sees it.
type eligiblePosition struct {
	models.Position
	ElectionTitle string `json:"election_title,omitempty"`
	Applied       bool   `json:"applied"`
}

// audit records a position change. A position linked to an election also gets
// that election's tamper-evident audit row; an unlinked one appears in the
// site-wide activity log only, because there is no election to file it under.
func (h *PositionHandler) audit(c *gin.Context, electionID, action, details string) {
	userID := c.GetString("user_id")
	if electionID != "" {
		h.DB.Create(&models.ElectionAuditLog{
			ElectionID: electionID,
			ActorID:    userID,
			Action:     action,
			Details:    details,
			IPAddress:  c.ClientIP(),
		})
	}
	services.LogActivity(h.DB, c, services.ActivityArgs{
		UserID:       userID,
		Action:       action,
		Severity:     "info",
		Summary:      details,
		ResourceType: "position",
		ResourceID:   c.Param("position_id"),
	})
}

// electionTitles answers a position_id → election title map for a set of
// positions, so a list shows names rather than opaque ids.
func (h *PositionHandler) electionTitles(positions []models.Position) map[string]string {
	ids := make([]string, 0, len(positions))
	for i := range positions {
		if ref := positions[i].ElectionRef(); ref != "" {
			ids = append(ids, ref)
		}
	}
	titles := map[string]string{}
	if len(ids) == 0 {
		return titles
	}
	var elections []models.Election
	h.DB.Where("id IN ?", ids).Find(&elections)
	for _, election := range elections {
		titles[election.ID] = election.Title
	}
	return titles
}

// ListAll returns every position, attached to an election or not, for the
// admin positions page. Filters: election_id, status, unlinked=1.
func (h *PositionHandler) ListAll(c *gin.Context) {
	query := h.DB.Model(&models.Position{})
	if value := c.Query("election_id"); value != "" {
		query = query.Where("election_id = ?", value)
	}
	if c.Query("unlinked") == "1" {
		query = query.Where("election_id IS NULL")
	}
	if value := c.Query("status"); value != "" {
		query = query.Where("status = ?", strings.ToUpper(strings.TrimSpace(value)))
	}

	var positions []models.Position
	if err := query.Order("created_at DESC").Find(&positions).Error; err != nil {
		respond.Internal(c, err)
		return
	}

	titles := h.electionTitles(positions)

	counts := map[string][2]int64{}
	if len(positions) > 0 {
		ids := make([]string, 0, len(positions))
		for i := range positions {
			ids = append(ids, positions[i].ID)
		}
		var rows []struct {
			PositionID string
			Total      int64
			Approved   int64
		}
		if err := h.DB.Model(&models.Candidate{}).
			Select("position_id, COUNT(*) AS total, "+
				"COALESCE(SUM(CASE WHEN approval_status = ? THEN 1 ELSE 0 END), 0) AS approved",
				models.CandidateStatusApproved).
			Where("position_id IN ?", ids).
			Group("position_id").
			Scan(&rows).Error; err != nil {
			respond.Internal(c, err)
			return
		}
		for _, row := range rows {
			counts[row.PositionID] = [2]int64{row.Total, row.Approved}
		}
	}

	out := make([]positionWithMeta, 0, len(positions))
	for i := range positions {
		meta := positionWithMeta{
			Position:      positions[i],
			ElectionTitle: titles[positions[i].ElectionRef()],
		}
		if pair, ok := counts[positions[i].ID]; ok {
			meta.ApplicationCount = pair[0]
			meta.ApprovedCount = pair[1]
		}
		out = append(out, meta)
	}
	respond.OK(c, gin.H{"positions": out})
}

// Eligible returns the positions the caller may apply for: open for
// applications, and permitted by the position's own eligibility rule. A
// position the caller has already applied for is returned with applied=true so
// the page can say so rather than hiding it.
func (h *PositionHandler) Eligible(c *gin.Context) {
	user := requestUser(c)
	if user == nil {
		respond.Fail(c, respond.CodeUnauthorized, "Not signed in")
		return
	}

	identity, parsed := eligibility.ParseEmail(user.Email)
	if !parsed {
		// A staff or external address cannot stand for a student position.
		respond.OK(c, gin.H{"positions": []eligiblePosition{}})
		return
	}

	var positions []models.Position
	if err := h.DB.Where("status = ?", models.PositionStatusOpen).
		Order("display_order ASC, created_at DESC").Find(&positions).Error; err != nil {
		respond.Internal(c, err)
		return
	}

	applied := map[string]bool{}
	ids := make([]string, 0, len(positions))
	for i := range positions {
		ids = append(ids, positions[i].ID)
	}
	if len(ids) > 0 {
		var mine []models.Candidate
		if err := h.DB.Where("nominated_by = ? AND position_id IN ? AND approval_status <> ?",
			user.ID, ids, models.CandidateStatusWithdrawn).Find(&mine).Error; err != nil {
			respond.Internal(c, err)
			return
		}
		for _, candidate := range mine {
			applied[candidate.PositionID] = true
		}
	}

	titles := h.electionTitles(positions)

	out := make([]eligiblePosition, 0, len(positions))
	for i := range positions {
		position := positions[i]
		if !eligibility.RuleForPosition(&position).Allows(identity) {
			continue
		}
		out = append(out, eligiblePosition{
			Position:      position,
			ElectionTitle: titles[position.ElectionRef()],
			Applied:       applied[position.ID],
		})
	}
	respond.OK(c, gin.H{"positions": out})
}

// CreateStandalone creates a position that is not yet attached to any
// election. An administrator creates positions first, collects applications,
// and links the settled position to its election afterwards.
func (h *PositionHandler) CreateStandalone(c *gin.Context) {
	var input struct {
		Title          string `json:"title" binding:"required"`
		Description    string `json:"description"`
		Requirements   string `json:"requirements"`
		QuestionType   string `json:"question_type" binding:"required"`
		MinSelections  int    `json:"min_selections"`
		MaxSelections  int    `json:"max_selections"`
		Seats          int    `json:"seats"`
		AllowBlank     bool   `json:"allow_blank"`
		DisplayOrder   int    `json:"display_order"`
		Status         string `json:"status"`
		MaxCandidates  int    `json:"max_candidates"`
		AllowedDomain  string `json:"allowed_domain"`
		AllowedYears   string `json:"allowed_years"`
		AllowedCourses string `json:"allowed_courses"`
		ElectionID     string `json:"election_id"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		respond.BadRequest(c, "Invalid request body")
		return
	}
	if message := validatePositionRules(input.QuestionType, input.MinSelections,
		input.MaxSelections, input.Seats, input.MaxCandidates); message != "" {
		respond.BadRequest(c, message)
		return
	}
	status, err := normalizePositionStatus(input.Status)
	if err != nil {
		respond.BadRequest(c, err.Error())
		return
	}

	electionID := strings.TrimSpace(input.ElectionID)
	if electionID != "" {
		var election models.Election
		if err := h.DB.Where("id = ?", electionID).First(&election).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				respond.NotFound(c, "Election not found")
				return
			}
			respond.Internal(c, err)
			return
		}
	}

	position := models.Position{
		Title:          input.Title,
		Description:    input.Description,
		Requirements:   input.Requirements,
		QuestionType:   input.QuestionType,
		MinSelections:  input.MinSelections,
		MaxSelections:  input.MaxSelections,
		Seats:          input.Seats,
		AllowBlank:     input.AllowBlank,
		DisplayOrder:   input.DisplayOrder,
		Status:         status,
		MaxCandidates:  input.MaxCandidates,
		AllowedDomain:  normalizeDomain(input.AllowedDomain),
		AllowedYears:   strings.TrimSpace(input.AllowedYears),
		AllowedCourses: strings.TrimSpace(input.AllowedCourses),
	}
	position.SetElection(electionID)

	if err := h.DB.Create(&position).Error; err != nil {
		respond.Internal(c, err)
		return
	}
	h.audit(c, position.ElectionRef(), models.AuditActionPositionCreated,
		"Position created: "+position.Title)
	respond.Created(c, position, "Position created")
}

// UpdateStandalone edits a position that is not tied to a locked election.
func (h *PositionHandler) UpdateStandalone(c *gin.Context) {
	var position models.Position
	if err := h.DB.Where("id = ?", c.Param("position_id")).First(&position).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respond.NotFound(c, "Position not found")
			return
		}
		respond.Internal(c, err)
		return
	}

	var input struct {
		Title          *string `json:"title"`
		Description    *string `json:"description"`
		Requirements   *string `json:"requirements"`
		QuestionType   *string `json:"question_type"`
		MinSelections  *int    `json:"min_selections"`
		MaxSelections  *int    `json:"max_selections"`
		Seats          *int    `json:"seats"`
		AllowBlank     *bool   `json:"allow_blank"`
		DisplayOrder   *int    `json:"display_order"`
		Status         *string `json:"status"`
		MaxCandidates  *int    `json:"max_candidates"`
		AllowedDomain  *string `json:"allowed_domain"`
		AllowedYears   *string `json:"allowed_years"`
		AllowedCourses *string `json:"allowed_courses"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		respond.BadRequest(c, "Invalid request body")
		return
	}

	updates := map[string]interface{}{}
	if input.Title != nil {
		if strings.TrimSpace(*input.Title) == "" {
			respond.BadRequest(c, "Give the position a title")
			return
		}
		updates["title"] = *input.Title
	}
	if input.Description != nil {
		updates["description"] = *input.Description
	}
	if input.Requirements != nil {
		updates["requirements"] = *input.Requirements
	}
	if input.QuestionType != nil {
		updates["question_type"] = *input.QuestionType
	}
	if input.MinSelections != nil {
		updates["min_selections"] = *input.MinSelections
	}
	if input.MaxSelections != nil {
		updates["max_selections"] = *input.MaxSelections
	}
	if input.Seats != nil {
		updates["seats"] = *input.Seats
	}
	if input.AllowBlank != nil {
		updates["allow_blank"] = *input.AllowBlank
	}
	if input.DisplayOrder != nil {
		updates["display_order"] = *input.DisplayOrder
	}
	if input.Status != nil {
		status, err := normalizePositionStatus(*input.Status)
		if err != nil {
			respond.BadRequest(c, err.Error())
			return
		}
		updates["status"] = status
	}
	if input.MaxCandidates != nil {
		if *input.MaxCandidates < 0 {
			respond.BadRequest(c, "Maximum candidates cannot be negative")
			return
		}
		updates["max_candidates"] = *input.MaxCandidates
	}
	if input.AllowedDomain != nil {
		updates["allowed_domain"] = normalizeDomain(*input.AllowedDomain)
	}
	if input.AllowedYears != nil {
		updates["allowed_years"] = strings.TrimSpace(*input.AllowedYears)
	}
	if input.AllowedCourses != nil {
		updates["allowed_courses"] = strings.TrimSpace(*input.AllowedCourses)
	}

	if len(updates) > 0 {
		if err := h.DB.Model(&position).Updates(updates).Error; err != nil {
			respond.Internal(c, err)
			return
		}
		h.audit(c, position.ElectionRef(), models.AuditActionPositionUpdated,
			"Position updated: "+position.Title)
	}
	respond.OK(c, position)
}

// DeleteStandalone removes a position and, by the schema's cascade, its applications.
func (h *PositionHandler) DeleteStandalone(c *gin.Context) {
	var position models.Position
	if err := h.DB.Where("id = ?", c.Param("position_id")).First(&position).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respond.NotFound(c, "Position not found")
			return
		}
		respond.Internal(c, err)
		return
	}
	if err := h.DB.Delete(&position).Error; err != nil {
		respond.Internal(c, err)
		return
	}
	h.audit(c, position.ElectionRef(), models.AuditActionPositionDeleted,
		"Position deleted: "+position.Title)
	respond.OK(c, gin.H{"message": "Position deleted"})
}

// Applications lists the applications received for one position, for review.
func (h *PositionHandler) Applications(c *gin.Context) {
	var position models.Position
	if err := h.DB.Where("id = ?", c.Param("position_id")).First(&position).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respond.NotFound(c, "Position not found")
			return
		}
		respond.Internal(c, err)
		return
	}

	query := h.DB.Where("position_id = ?", position.ID)
	if status := c.Query("status"); status != "" {
		query = query.Where("approval_status = ?", strings.ToUpper(strings.TrimSpace(status)))
	}
	var candidates []models.Candidate
	if err := query.Order("created_at DESC").Find(&candidates).Error; err != nil {
		respond.Internal(c, err)
		return
	}

	approved := int64(0)
	for _, candidate := range candidates {
		if candidate.ApprovalStatus == models.CandidateStatusApproved {
			approved++
		}
	}
	respond.OK(c, gin.H{
		"position": position,
		"applications": candidates,
		"approved_count": approved,
	})
}

// Apply records the caller's application to a standalone position. This is the
// student-facing path: the position's own eligibility rule and application
// window are checked here, on the server.
func (h *PositionHandler) Apply(c *gin.Context) {
	user := requestUser(c)
	if user == nil {
		respond.Fail(c, respond.CodeUnauthorized, "Not signed in")
		return
	}

	var position models.Position
	if err := h.DB.Where("id = ?", c.Param("position_id")).First(&position).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respond.NotFound(c, "Position not found")
			return
		}
		respond.Internal(c, err)
		return
	}
	if !position.AcceptingApplications() {
		respond.Conflict(c, "This position is not accepting applications")
		return
	}

	identity, parsed := eligibility.ParseEmail(user.Email)
	if !parsed {
		respond.Forbidden(c, "Your email address does not match the student registration format")
		return
	}
	if !eligibility.RuleForPosition(&position).Allows(identity) {
		respond.Forbidden(c, "You are not eligible to stand for this position")
		return
	}

	var input struct {
		FullName     string `json:"full_name"`
		Slogan       string `json:"slogan"`
		Bio          string `json:"bio"`
		Manifesto    string `json:"manifesto"`
		ContactEmail string `json:"contact_email"`
		ContactPhone string `json:"contact_phone"`
	}
	// An empty body is allowed: the applicant then stands under their account
	// name. A malformed body is still rejected.
	if err := c.ShouldBindJSON(&input); err != nil && err.Error() != "EOF" {
		respond.BadRequest(c, "Invalid request body")
		return
	}

	fullName := strings.TrimSpace(input.FullName)
	if fullName == "" {
		fullName = strings.TrimSpace(user.FirstName + " " + user.LastName)
	}
	if len(fullName) < 2 || len(fullName) > 255 {
		respond.BadRequest(c, "Your name must be between 2 and 255 characters")
		return
	}
	if len(strings.TrimSpace(input.Slogan)) > 200 {
		respond.BadRequest(c, "Your slogan must be 200 characters or fewer")
		return
	}
	if len(input.Bio) > 5000 || len(input.Manifesto) > 20000 {
		respond.BadRequest(c, "Your biography or manifesto is too long")
		return
	}

	var existing int64
	if err := h.DB.Model(&models.Candidate{}).
		Where("position_id = ? AND nominated_by = ? AND approval_status <> ?",
			position.ID, user.ID, models.CandidateStatusWithdrawn).
		Count(&existing).Error; err != nil {
		respond.Internal(c, err)
		return
	}
	if existing > 0 {
		respond.Conflict(c, "You already have an application for this position")
		return
	}

	candidate := models.Candidate{
		ElectionID:     position.ElectionRef(),
		PositionID:     position.ID,
		FullName:       fullName,
		Slogan:         strings.TrimSpace(input.Slogan),
		Bio:            strings.TrimSpace(input.Bio),
		Manifesto:      strings.TrimSpace(input.Manifesto),
		ContactEmail:   strings.TrimSpace(input.ContactEmail),
		ContactPhone:   strings.TrimSpace(input.ContactPhone),
		ApprovalStatus: models.CandidateStatusNominated,
		NominatedBy:    user.ID,
		// Registration identity comes from the verified email, not from input.
		RegistrationNumber: strings.ToUpper(identity.Year + identity.Course + identity.Number),
		Programme:          identity.Programme(),
		StudyYear:          identity.StudyYear(h.now()),
	}
	if err := h.DB.Create(&candidate).Error; err != nil {
		respond.Internal(c, err)
		return
	}
	h.audit(c, position.ElectionRef(), models.AuditActionCandidateNominated,
		"Application submitted: "+candidate.FullName+" for "+position.Title)
	respond.Created(c, candidate, "Application submitted. Officials will review it.")
}

// ApproveApplication approves one application. When the position sets a
// maximum number of candidates, approval is refused once that many are already
// approved, so the cap is enforced by the server rather than the page.
func (h *PositionHandler) ApproveApplication(c *gin.Context) {
	position, candidate, ok := h.loadApplication(c)
	if !ok {
		return
	}
	if candidate.ApprovalStatus != models.CandidateStatusNominated &&
		candidate.ApprovalStatus != models.CandidateStatusPending {
		respond.Conflict(c, "This application has already been reviewed")
		return
	}

	if position.MaxCandidates > 0 {
		var approved int64
		if err := h.DB.Model(&models.Candidate{}).
			Where("position_id = ? AND approval_status = ? AND id <> ?",
				position.ID, models.CandidateStatusApproved, candidate.ID).
			Count(&approved).Error; err != nil {
			respond.Internal(c, err)
			return
		}
		if approved >= int64(position.MaxCandidates) {
			respond.Conflict(c, "This position already has the maximum number of candidates")
			return
		}
	}

	var input struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&input)
	reason := strings.TrimSpace(input.Reason)
	if reason == "" {
		reason = "Approved by an administrator"
	}

	now := h.now()
	candidate.ApprovalStatus = models.CandidateStatusApproved
	candidate.ReviewedBy = c.GetString("user_id")
	candidate.ReviewedAt = &now
	candidate.ReviewNotes = reason
	if candidate.BallotNumber == 0 {
		var highest int
		h.DB.Model(&models.Candidate{}).
			Where("position_id = ? AND id <> ?", position.ID, candidate.ID).
			Select("COALESCE(MAX(ballot_number), 0)").Scan(&highest)
		candidate.BallotNumber = highest + 1
	}

	if err := h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&candidate).Error; err != nil {
			return err
		}
		return notifyCandidate(tx, candidate, "Application approved",
			"Your application for "+position.Title+" has been approved.")
	}); err != nil {
		respond.Internal(c, err)
		return
	}
	h.audit(c, position.ElectionRef(), models.AuditActionCandidateApproved,
		"Candidate approved: "+candidate.FullName+" for "+position.Title+" - "+reason)
	respond.OK(c, candidate, "Application approved")
}

// RejectApplication rejects one application with a reason the applicant can read.
func (h *PositionHandler) RejectApplication(c *gin.Context) {
	position, candidate, ok := h.loadApplication(c)
	if !ok {
		return
	}
	if candidate.ApprovalStatus != models.CandidateStatusNominated &&
		candidate.ApprovalStatus != models.CandidateStatusPending {
		respond.Conflict(c, "This application has already been reviewed")
		return
	}

	var input struct {
		Reason string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		respond.BadRequest(c, "A reason is required")
		return
	}

	now := h.now()
	candidate.ApprovalStatus = models.CandidateStatusRejected
	candidate.ReviewedBy = c.GetString("user_id")
	candidate.ReviewedAt = &now
	candidate.ReviewNotes = strings.TrimSpace(input.Reason)

	if err := h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&candidate).Error; err != nil {
			return err
		}
		return notifyCandidate(tx, candidate, "Application not approved",
			"Your application for "+position.Title+" was not approved. Reason: "+candidate.ReviewNotes)
	}); err != nil {
		respond.Internal(c, err)
		return
	}
	h.audit(c, position.ElectionRef(), models.AuditActionCandidateRejected,
		"Candidate rejected: "+candidate.FullName+" for "+position.Title+" - "+candidate.ReviewNotes)
	respond.OK(c, candidate, "Application rejected")
}

// loadApplication fetches the position named in the URL and one of its
// applications, writing the response itself when either is missing.
func (h *PositionHandler) loadApplication(c *gin.Context) (models.Position, models.Candidate, bool) {
	var position models.Position
	if err := h.DB.Where("id = ?", c.Param("position_id")).First(&position).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respond.NotFound(c, "Position not found")
			return position, models.Candidate{}, false
		}
		respond.Internal(c, err)
		return position, models.Candidate{}, false
	}
	var candidate models.Candidate
	if err := h.DB.Where("id = ? AND position_id = ?", c.Param("candidate_id"), position.ID).
		First(&candidate).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respond.NotFound(c, "Application not found")
			return position, candidate, false
		}
		respond.Internal(c, err)
		return position, candidate, false
	}
	return position, candidate, true
}

// Link attaches a position to the election it will appear on, and moves its
// applications across so they belong to that election too. Passing an empty
// election_id detaches the position again.
func (h *PositionHandler) Link(c *gin.Context) {
	var position models.Position
	if err := h.DB.Where("id = ?", c.Param("position_id")).First(&position).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respond.NotFound(c, "Position not found")
			return
		}
		respond.Internal(c, err)
		return
	}

	var input struct {
		ElectionID string `json:"election_id"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		respond.BadRequest(c, "Invalid request body")
		return
	}
	electionID := strings.TrimSpace(input.ElectionID)

	if electionID != "" {
		var election models.Election
		if err := h.DB.Where("id = ?", electionID).First(&election).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				respond.NotFound(c, "Election not found")
				return
			}
			respond.Internal(c, err)
			return
		}
		switch election.Status {
		case models.ElectionStatusFinalized, models.ElectionStatusArchived, models.ElectionStatusCancelled:
			respond.Forbidden(c, "This election can no longer take new positions")
			return
		}
	}

	previous := position.ElectionRef()
	if err := h.DB.Transaction(func(tx *gorm.DB) error {
		position.SetElection(electionID)
		if err := tx.Save(&position).Error; err != nil {
			return err
		}
		// Applications follow their position, so tallying sees them under the
		// election they are actually contesting.
		return tx.Model(&models.Candidate{}).
			Where("position_id = ?", position.ID).
			Update("election_id", electionID).Error
	}); err != nil {
		respond.Internal(c, err)
		return
	}

	details := "Position linked to election: " + position.Title
	if electionID == "" {
		details = "Position detached from its election: " + position.Title
	} else if previous != "" && previous != electionID {
		details = "Position moved to another election: " + position.Title
	}
	h.audit(c, electionID, models.AuditActionPositionUpdated, details)
	respond.OK(c, position, "Position linked")
}

// now returns the handler's clock, matching CandidateHandler so tests can pin it.
func (h *PositionHandler) now() time.Time {
	return time.Now().UTC()
}
