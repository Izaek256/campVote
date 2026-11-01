package handlers

import (
	"errors"
	"net/mail"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"campvote/apps/api/internal/eligibility"
	"campvote/apps/api/internal/models"
	"campvote/apps/api/internal/respond"
)

// CandidateHandler manages candidate nomination and approval workflow (ISSUE-06).
type CandidateHandler struct {
	DB  *gorm.DB
	Now func() time.Time
}

func NewCandidateHandler(db *gorm.DB) *CandidateHandler {
	return &CandidateHandler{DB: db, Now: time.Now}
}

func (h *CandidateHandler) now() time.Time {
	if h.Now == nil {
		return time.Now().UTC()
	}
	return h.Now().UTC()
}

// List returns candidates for an election, optionally filtered by status.
func (h *CandidateHandler) List(c *gin.Context) {
	electionID := c.Param("id")
	status := c.Query("status")

	query := h.DB.Where("election_id = ?", electionID)
	if status != "" {
		query = query.Where("approval_status = ?", status)
	}

	var candidates []models.Candidate
	if err := query.Order("created_at DESC").Find(&candidates).Error; err != nil {
		respond.Internal(c, err)
		return
	}

	respond.OK(c, gin.H{"candidates": candidates})
}

// Application is the voter's view of the candidate application form: the
// positions available, whether the window is open, and the caller's roster
// decision. It answers in one call so the page cannot show a form to somebody
// the server will refuse.
func (h *CandidateHandler) Application(c *gin.Context) {
	electionID := c.Param("id")
	var election models.Election
	if err := h.DB.Where("id = ?", electionID).First(&election).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respond.NotFound(c, "Election not found")
			return
		}
		respond.Internal(c, err)
		return
	}

	decision, err := eligibility.CheckRoster(h.DB, &election, requestUser(c))
	if err != nil {
		respond.Internal(c, err)
		return
	}
	open, reason := applicationWindow(&election, h.now())

	var positions []models.Position
	if err := h.DB.Where("election_id = ?", electionID).
		Order("display_order ASC, title ASC").Find(&positions).Error; err != nil {
		respond.Internal(c, err)
		return
	}

	respond.OK(c, gin.H{
		"election": gin.H{
			"id":         election.ID,
			"title":      election.Title,
			"status":     election.Status,
			"starts_at":  election.StartsAt,
			"ends_at":    election.EndsAt,
			"category":   election.Category,
			"unit":       election.OrganizingUnit,
		},
		"positions":            positions,
		"eligible":             decision.Eligible,
		"eligibility_reason":   decision.Reason,
		"accepting_applications": open,
		"application_reason":   reason,
	})
}

// Apply creates the caller's own candidacy. Officers nominate through Create;
// this path is the student-facing one and checks the roster itself.
func (h *CandidateHandler) Apply(c *gin.Context) {
	electionID := c.Param("id")
	user := requestUser(c)
	if user == nil {
		respond.Fail(c, respond.CodeUnauthorized, "Not signed in")
		return
	}

	var election models.Election
	if err := h.DB.Where("id = ?", electionID).First(&election).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respond.NotFound(c, "Election not found")
			return
		}
		respond.Internal(c, err)
		return
	}
	if open, reason := applicationWindow(&election, h.now()); !open {
		respond.Conflict(c, reason)
		return
	}
	decision, err := eligibility.CheckRoster(h.DB, &election, user)
	if err != nil {
		respond.Internal(c, err)
		return
	}
	if !decision.Eligible {
		respond.Forbidden(c, decision.Reason)
		return
	}

	var input struct {
		PositionID   string `json:"position_id" binding:"required"`
		FullName     string `json:"full_name"`
		Slogan       string `json:"slogan"`
		Bio          string `json:"bio"`
		Manifesto    string `json:"manifesto"`
		ManifestoURL string `json:"manifesto_url"`
		PhotoURL     string `json:"photo_url"`
		ContactEmail string `json:"contact_email"`
		ContactPhone string `json:"contact_phone"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		respond.BadRequest(c, "Choose a position and add your name")
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
	contactEmail := strings.TrimSpace(input.ContactEmail)
	if contactEmail != "" {
		if _, err := mail.ParseAddress(contactEmail); err != nil {
			respond.BadRequest(c, "Enter a valid contact email address")
			return
		}
	}
	if !storedFileURL(input.PhotoURL) || !storedFileURL(input.ManifestoURL) {
		respond.BadRequest(c, "Uploaded files must be links to this site or an https address")
		return
	}

	var position models.Position
	if err := h.DB.Where("id = ? AND election_id = ?", input.PositionID, electionID).
		First(&position).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respond.NotFound(c, "Position not found in this election")
			return
		}
		respond.Internal(c, err)
		return
	}

	var existing int64
	if err := h.DB.Model(&models.Candidate{}).
		Where("election_id = ? AND nominated_by = ? AND approval_status <> ?",
			electionID, user.ID, models.CandidateStatusWithdrawn).
		Count(&existing).Error; err != nil {
		respond.Internal(c, err)
		return
	}
	if existing > 0 {
		respond.Conflict(c, "You already have an application for this election")
		return
	}

	candidate := models.Candidate{
		ElectionID:     electionID,
		PositionID:     input.PositionID,
		FullName:       fullName,
		Slogan:         strings.TrimSpace(input.Slogan),
		PhotoURL:       strings.TrimSpace(input.PhotoURL),
		Manifesto:      strings.TrimSpace(input.Manifesto),
		ManifestoURL:   strings.TrimSpace(input.ManifestoURL),
		Bio:            strings.TrimSpace(input.Bio),
		ContactEmail:   contactEmail,
		ContactPhone:   strings.TrimSpace(input.ContactPhone),
		ApprovalStatus: models.CandidateStatusNominated,
		NominatedBy:    user.ID,
	}
	if err := h.DB.Create(&candidate).Error; err != nil {
		respond.Internal(c, err)
		return
	}
	h.audit(c, electionID, models.AuditActionCandidateNominated,
		"Application submitted: "+candidate.FullName+" for "+position.Title)
	respond.Created(c, candidate, "Application submitted. Officials will review it.")
}

// WithdrawMine withdraws the caller's own candidacy.
func (h *CandidateHandler) WithdrawMine(c *gin.Context) {
	userID := c.GetString("user_id")
	var input struct {
		Reason string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		respond.BadRequest(c, "A withdrawal reason is required")
		return
	}

	var candidate models.Candidate
	if err := h.DB.Where("id = ? AND nominated_by = ?", c.Param("candidate_id"), userID).
		First(&candidate).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respond.NotFound(c, "Application not found")
			return
		}
		respond.Internal(c, err)
		return
	}
	if candidate.ApprovalStatus == models.CandidateStatusWithdrawn {
		respond.Conflict(c, "This application is already withdrawn")
		return
	}
	if candidate.ApprovalStatus == models.CandidateStatusRejected {
		respond.Conflict(c, "A rejected application cannot be withdrawn")
		return
	}

	now := h.now()
	candidate.ApprovalStatus = models.CandidateStatusWithdrawn
	candidate.WithdrawnAt = &now
	candidate.WithdrawalReason = input.Reason
	if err := h.DB.Save(&candidate).Error; err != nil {
		respond.Internal(c, err)
		return
	}
	h.audit(c, candidate.ElectionID, models.AuditActionCandidateWithdrawn,
		"Application withdrawn: "+candidate.FullName+" - "+input.Reason)
	respond.OK(c, candidate, "Application withdrawn")
}

// applicationWindow decides whether candidates may still apply. Voting opening
// ends the window; a closed or cancelled election never reopens it.
func applicationWindow(election *models.Election, at time.Time) (bool, string) {
	switch election.Status {
	case models.ElectionStatusDraft:
		return false, "This election has not opened candidate applications yet"
	case models.ElectionStatusClosed, models.ElectionStatusReview,
		models.ElectionStatusFinalized, models.ElectionStatusArchived,
		models.ElectionStatusCancelled:
		return false, "Candidate applications are closed for this election"
	}
	if !at.Before(election.StartsAt) {
		return false, "Candidate applications closed when voting opened"
	}
	return true, ""
}

// storedFileURL accepts an empty value, an absolute http(s) URL, or a
// site-relative path produced by local storage. Anything else is refused so a
// stored field cannot become a javascript: link on the ballot.
func storedFileURL(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return true
	}
	if strings.HasPrefix(value, "/") && !strings.HasPrefix(value, "//") {
		return true
	}
	parsed, err := url.Parse(value)
	return err == nil && (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Host != ""
}

func (h *CandidateHandler) audit(c *gin.Context, electionID, action, details string) {
	h.DB.Create(&models.ElectionAuditLog{
		ElectionID: electionID,
		ActorID:    c.GetString("user_id"),
		Action:     action,
		Details:    details,
		IPAddress:  c.ClientIP(),
	})
}

// Create adds a new candidate nomination.
func (h *CandidateHandler) Create(c *gin.Context) {
	electionID := c.Param("id")
	userID := c.GetString("user_id")

	var input struct {
		PositionID string `json:"position_id" binding:"required"`
		FullName   string `json:"full_name" binding:"required"`
		PhotoURL   string `json:"photo_url"`
		Manifesto  string `json:"manifesto"`
		Bio        string `json:"bio"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		respond.BadRequest(c, "Invalid request body")
		return
	}

	// Verify position exists and belongs to election
	var position models.Position
	if err := h.DB.Where("id = ? AND election_id = ?", input.PositionID, electionID).First(&position).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respond.NotFound(c, "Position not found")
			return
		}
		respond.Internal(c, err)
		return
	}

	candidate := models.Candidate{
		ElectionID:     electionID,
		PositionID:     input.PositionID,
		FullName:       input.FullName,
		PhotoURL:       input.PhotoURL,
		Manifesto:      input.Manifesto,
		Bio:            input.Bio,
		ApprovalStatus: models.CandidateStatusNominated,
		NominatedBy:    userID,
	}

	if err := h.DB.Create(&candidate).Error; err != nil {
		respond.Internal(c, err)
		return
	}

	// Audit log
	audit := models.ElectionAuditLog{
		ElectionID: electionID,
		ActorID:    userID,
		Action:     models.AuditActionCandidateNominated,
		Details:    "Candidate nominated: " + candidate.FullName,
		IPAddress:  c.ClientIP(),
	}
	h.DB.Create(&audit)

	respond.Created(c, candidate, "Candidate nominated")
}

// Update allows a candidate to edit their own profile.
func (h *CandidateHandler) Update(c *gin.Context) {
	electionID := c.Param("id")
	candidateID := c.Param("candidate_id")

	var candidate models.Candidate
	if err := h.DB.Where("id = ? AND election_id = ?", candidateID, electionID).First(&candidate).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respond.NotFound(c, "Candidate not found")
			return
		}
		respond.Internal(c, err)
		return
	}

	// Only the candidate who nominated (or the candidate themselves in self-nomination) can edit
	// For ISSUE-06, we allow any officer to edit; candidate-specific access can be refined later
	// TODO: Add proper role check for officer access

	var input struct {
		FullName     string `json:"full_name"`
		Slogan       string `json:"slogan"`
		PhotoURL     string `json:"photo_url"`
		Manifesto    string `json:"manifesto"`
		ManifestoURL string `json:"manifesto_url"`
		Bio          string `json:"bio"`
		ContactEmail string `json:"contact_email"`
		ContactPhone string `json:"contact_phone"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		respond.BadRequest(c, "Invalid request body")
		return
	}

	updates := map[string]interface{}{}
	if input.FullName != "" {
		updates["full_name"] = input.FullName
	}
	if input.Slogan != "" {
		updates["slogan"] = input.Slogan
	}
	if input.PhotoURL != "" {
		updates["photo_url"] = input.PhotoURL
	}
	if input.Manifesto != "" {
		updates["manifesto"] = input.Manifesto
	}
	if input.ManifestoURL != "" {
		updates["manifesto_url"] = input.ManifestoURL
	}
	if input.Bio != "" {
		updates["bio"] = input.Bio
	}
	if input.ContactEmail != "" {
		updates["contact_email"] = input.ContactEmail
	}
	if input.ContactPhone != "" {
		updates["contact_phone"] = input.ContactPhone
	}

	if len(updates) > 0 {
		if err := h.DB.Model(&candidate).Updates(updates).Error; err != nil {
			respond.Internal(c, err)
			return
		}
	}

	respond.OK(c, candidate)
}

// Approve approves a candidate nomination.
func (h *CandidateHandler) Approve(c *gin.Context) {
	electionID := c.Param("id")
	candidateID := c.Param("candidate_id")
	userID := c.GetString("user_id")

	var input struct {
		Reason string `json:"reason" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		respond.BadRequest(c, "Reason is required")
		return
	}

	var candidate models.Candidate
	if err := h.DB.Where("id = ? AND election_id = ?", candidateID, electionID).First(&candidate).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respond.NotFound(c, "Candidate not found")
			return
		}
		respond.Internal(c, err)
		return
	}
	if candidate.ApprovalStatus != models.CandidateStatusNominated &&
		candidate.ApprovalStatus != models.CandidateStatusPending {
		respond.Conflict(c, "This application has already been reviewed")
		return
	}

	now := h.now()
	candidate.ApprovalStatus = models.CandidateStatusApproved
	candidate.ReviewedBy = userID
	candidate.ReviewedAt = &now
	candidate.ReviewNotes = input.Reason
	if candidate.BallotNumber == 0 {
		var highest int
		h.DB.Model(&models.Candidate{}).
			Where("election_id = ? AND position_id = ? AND id <> ?",
				electionID, candidate.PositionID, candidate.ID).
			Select("COALESCE(MAX(ballot_number), 0)").Scan(&highest)
		candidate.BallotNumber = highest + 1
	}

	if err := h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&candidate).Error; err != nil {
			return err
		}
		if err := tx.Create(&models.ElectionAuditLog{
			ElectionID: electionID,
			ActorID:    userID,
			Action:     models.AuditActionCandidateApproved,
			Details:    "Candidate approved: " + candidate.FullName + " - " + input.Reason,
			IPAddress:  c.ClientIP(),
		}).Error; err != nil {
			return err
		}
		return notifyCandidate(tx, candidate, "Candidate application approved",
			"Your application for this election has been approved. Review your profile before voting opens.")
	}); err != nil {
		respond.Internal(c, err)
		return
	}

	respond.OK(c, candidate, "Candidate approved")
}

// Reject rejects a candidate nomination.
func (h *CandidateHandler) Reject(c *gin.Context) {
	electionID := c.Param("id")
	candidateID := c.Param("candidate_id")
	userID := c.GetString("user_id")

	var input struct {
		Reason string `json:"reason" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		respond.BadRequest(c, "Reason is required")
		return
	}

	var candidate models.Candidate
	if err := h.DB.Where("id = ? AND election_id = ?", candidateID, electionID).First(&candidate).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respond.NotFound(c, "Candidate not found")
			return
		}
		respond.Internal(c, err)
		return
	}
	if candidate.ApprovalStatus != models.CandidateStatusNominated &&
		candidate.ApprovalStatus != models.CandidateStatusPending {
		respond.Conflict(c, "This application has already been reviewed")
		return
	}

	now := h.now()
	candidate.ApprovalStatus = models.CandidateStatusRejected
	candidate.ReviewedBy = userID
	candidate.ReviewedAt = &now
	candidate.ReviewNotes = input.Reason

	if err := h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&candidate).Error; err != nil {
			return err
		}
		if err := tx.Create(&models.ElectionAuditLog{
			ElectionID: electionID,
			ActorID:    userID,
			Action:     models.AuditActionCandidateRejected,
			Details:    "Candidate rejected: " + candidate.FullName + " - " + input.Reason,
			IPAddress:  c.ClientIP(),
		}).Error; err != nil {
			return err
		}
		return notifyCandidate(tx, candidate, "Candidate application not approved",
			"Your application was not approved. Reason: "+input.Reason)
	}); err != nil {
		respond.Internal(c, err)
		return
	}

	respond.OK(c, candidate, "Candidate rejected")
}

// notifyCandidate writes an in-app notification for the applicant. Email
// delivery is deferred: the outbox and mailer need a template decision.
// shortcut: in-app notification only, add email templates when comms signs off.
func notifyCandidate(tx *gorm.DB, candidate models.Candidate, title, body string) error {
	if candidate.NominatedBy == "" {
		return nil
	}
	return tx.Create(&models.Notification{
		UserID:   candidate.NominatedBy,
		Source:   "system",
		Severity: "info",
		Title:    title,
		Body:     body,
		Link:     "/candidate",
		Dedup:    "candidate:" + candidate.ID + ":" + candidate.ApprovalStatus,
	}).Error
}

// Withdraw handles candidate withdrawal.
func (h *CandidateHandler) Withdraw(c *gin.Context) {
	electionID := c.Param("id")
	candidateID := c.Param("candidate_id")
	userID := c.GetString("user_id")

	var input struct {
		Reason string `json:"reason" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		respond.BadRequest(c, "Reason is required")
		return
	}

	var candidate models.Candidate
	if err := h.DB.Where("id = ? AND election_id = ?", candidateID, electionID).First(&candidate).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respond.NotFound(c, "Candidate not found")
			return
		}
		respond.Internal(c, err)
		return
	}

	now := h.now()
	candidate.ApprovalStatus = models.CandidateStatusWithdrawn
	candidate.WithdrawnAt = &now
	candidate.WithdrawalReason = input.Reason

	if err := h.DB.Save(&candidate).Error; err != nil {
		respond.Internal(c, err)
		return
	}

	// Audit log
	audit := models.ElectionAuditLog{
		ElectionID: electionID,
		ActorID:    userID,
		Action:     models.AuditActionCandidateWithdrawn,
		Details:    "Candidate withdrawn: " + candidate.FullName + " - " + input.Reason,
		IPAddress:  c.ClientIP(),
	}
	h.DB.Create(&audit)

	respond.OK(c, candidate)
}

// GetMyCandidacies returns the current user's nominations.
func (h *CandidateHandler) GetMyCandidacies(c *gin.Context) {
	userID := c.GetString("user_id")

	var candidates []models.Candidate
	if err := h.DB.Where("nominated_by = ?", userID).
		Order("created_at DESC").
		Find(&candidates).Error; err != nil {
		respond.Internal(c, err)
		return
	}

	// The applicant needs the election and position names, not two opaque ids.
	electionIDs := make([]string, 0, len(candidates))
	positionIDs := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		electionIDs = append(electionIDs, candidate.ElectionID)
		positionIDs = append(positionIDs, candidate.PositionID)
	}
	electionNames := map[string]string{}
	positionNames := map[string]string{}
	if len(electionIDs) > 0 {
		var elections []models.Election
		h.DB.Where("id IN ?", electionIDs).Find(&elections)
		for _, election := range elections {
			electionNames[election.ID] = election.Title
		}
		var positions []models.Position
		h.DB.Where("id IN ?", positionIDs).Find(&positions)
		for _, position := range positions {
			positionNames[position.ID] = position.Title
		}
	}

	out := make([]gin.H, 0, len(candidates))
	for _, candidate := range candidates {
		out = append(out, gin.H{
			"candidate":       candidate,
			"election_title":  electionNames[candidate.ElectionID],
			"position_title":  positionNames[candidate.PositionID],
		})
	}
	respond.OK(c, gin.H{"candidates": out})
}
