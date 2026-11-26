package handlers

import (
	"errors"
	"fmt"
	"net/mail"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"campvote/apps/api/internal/eligibility"
	"campvote/apps/api/internal/models"
	"campvote/apps/api/internal/respond"
)

// A selected candidate's manifesto may be a PDF, an MP3 or an MP4. The client
// offers the same set; the server checks the stored upload again, because the
// file picker is not the control.
const manifestoMaxBytes = 100 << 20

var manifestoMIMEs = map[string]bool{
	"application/pdf": true,
	"audio/mpeg":      true,
	"audio/mp4":       true,
	"video/mp4":       true,
}

func manifestoTypeAllowed(mimeType string) bool {
	return manifestoMIMEs[strings.ToLower(strings.TrimSpace(mimeType))]
}

// selectionWindowOpen answers whether an invitation may be issued, accepted or
// submitted. The path closes when voting opens, and never reopens.
func (h *CandidateHandler) selectionWindowOpen(election *models.Election) (bool, string) {
	if election.Status == models.ElectionStatusClosed || !h.now().Before(election.StartsAt) {
		return false, "Selected candidates can only act before voting opens"
	}
	return true, ""
}

// InviteSelected is the administrator path: it names the only people who may
// stand for a position. It is idempotent per address and refuses to invite
// somebody who already has an active candidacy for the position.
func (h *CandidateHandler) InviteSelected(c *gin.Context) {
	electionID := c.Param("id")
	var position models.Position
	if err := h.DB.Where("id = ? AND election_id = ?", c.Param("position_id"), electionID).
		First(&position).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respond.NotFound(c, "Position not found in this election")
			return
		}
		respond.Internal(c, err)
		return
	}

	var election models.Election
	if err := h.DB.Where("id = ?", electionID).First(&election).Error; err != nil {
		respond.Internal(c, err)
		return
	}
	if open, reason := h.selectionWindowOpen(&election); !open {
		respond.Conflict(c, reason)
		return
	}

	var input struct {
		Emails []string `json:"emails" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || len(input.Emails) == 0 {
		respond.BadRequest(c, "Add at least one email address")
		return
	}

	created := make([]models.Candidate, 0, len(input.Emails))
	for _, raw := range input.Emails {
		email := strings.ToLower(strings.TrimSpace(raw))
		if email == "" {
			continue
		}
		if _, err := mail.ParseAddress(email); err != nil {
			respond.BadRequest(c, fmt.Sprintf("%q is not a valid email address", raw))
			return
		}
		if _, ok := eligibility.ParseEmail(email); !ok {
			respond.BadRequest(c, fmt.Sprintf("%q is not a student registration email", raw))
			return
		}

		var existing int64
		if err := h.DB.Model(&models.Candidate{}).
			Where("position_id = ? AND LOWER(invited_email) = ? AND approval_status NOT IN ?",
				position.ID, email, []string{models.CandidateStatusWithdrawn, models.CandidateStatusRejected}).
			Count(&existing).Error; err != nil {
			respond.Internal(c, err)
			return
		}
		if existing > 0 {
			continue
		}

		candidate := models.Candidate{
			ElectionID:     electionID,
			PositionID:     position.ID,
			FullName:       email,
			InvitedEmail:   email,
			ApprovalStatus: models.CandidateStatusSelected,
		}
		if err := h.DB.Create(&candidate).Error; err != nil {
			respond.Internal(c, err)
			return
		}
		created = append(created, candidate)
	}

	// Selecting is what makes the position closed to ordinary applications.
	if !position.AdminSelected {
		if err := h.DB.Model(&position).Update("admin_selected", true).Error; err != nil {
			respond.Internal(c, err)
			return
		}
	}
	h.audit(c, electionID, models.AuditActionCandidateNominated,
		fmt.Sprintf("Selected %d candidate(s) for %s", len(created), position.Title))
	respond.Created(c, gin.H{"candidates": created}, "Candidates selected")
}

// ListSelections returns the invited and accepted candidates for one position,
// so the administrator can see who has not responded yet.
func (h *CandidateHandler) ListSelections(c *gin.Context) {
	electionID := c.Param("id")
	var position models.Position
	if err := h.DB.Where("id = ? AND election_id = ?", c.Param("position_id"), electionID).
		First(&position).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respond.NotFound(c, "Position not found in this election")
			return
		}
		respond.Internal(c, err)
		return
	}

	var candidates []models.Candidate
	if err := h.DB.Where("position_id = ? AND invited_email <> ''", position.ID).
		Order("created_at ASC").Find(&candidates).Error; err != nil {
		respond.Internal(c, err)
		return
	}
	respond.OK(c, gin.H{"position": position, "candidates": candidates})
}

// AcceptSelection lets the signed-in account claim an invitation. Only the
// verified address the administrator named may accept, and only before voting
// opens.
func (h *CandidateHandler) AcceptSelection(c *gin.Context) {
	user := requestUser(c)
	if user == nil {
		respond.Fail(c, respond.CodeUnauthorized, "Not signed in")
		return
	}
	candidate, ok := h.loadInvitation(c, user)
	if !ok {
		return
	}
	if candidate.ApprovalStatus != models.CandidateStatusSelected {
		respond.Conflict(c, "This invitation has already been answered")
		return
	}

	var election models.Election
	if err := h.DB.Where("id = ?", candidate.ElectionID).First(&election).Error; err != nil {
		respond.Internal(c, err)
		return
	}
	if open, reason := h.selectionWindowOpen(&election); !open {
		respond.Conflict(c, reason)
		return
	}

	now := h.now()
	candidate.NominatedBy = user.ID
	candidate.UserID = user.ID
	candidate.AcceptedAt = &now
	candidate.ApprovalStatus = models.CandidateStatusAccepted
	if name := strings.TrimSpace(user.FirstName + " " + user.LastName); name != "" &&
		candidate.FullName == candidate.InvitedEmail {
		candidate.FullName = name
	}
	if err := h.DB.Save(&candidate).Error; err != nil {
		respond.Internal(c, err)
		return
	}
	h.audit(c, candidate.ElectionID, models.AuditActionCandidateUpdated,
		"Selected candidate accepted: "+candidate.InvitedEmail)
	respond.OK(c, candidate, "Invitation accepted. Add your details and manifesto.")
}

// SubmitSelection records the selected candidate's details and manifesto. It
// is also the edit path until voting opens. A manifesto file is mandatory and
// must be a PDF, MP3 or MP4 this account uploaded; no transcript is required
// and approval is automatic.
func (h *CandidateHandler) SubmitSelection(c *gin.Context) {
	user := requestUser(c)
	if user == nil {
		respond.Fail(c, respond.CodeUnauthorized, "Not signed in")
		return
	}
	candidate, ok := h.loadInvitation(c, user)
	if !ok {
		return
	}
	if candidate.ApprovalStatus != models.CandidateStatusAccepted &&
		candidate.ApprovalStatus != models.CandidateStatusApproved {
		respond.Conflict(c, "Accept the invitation before submitting your candidacy")
		return
	}

	var election models.Election
	if err := h.DB.Where("id = ?", candidate.ElectionID).First(&election).Error; err != nil {
		respond.Internal(c, err)
		return
	}
	if open, reason := h.selectionWindowOpen(&election); !open {
		respond.Conflict(c, reason)
		return
	}

	var input struct {
		FullName         string `json:"full_name"`
		Slogan           string `json:"slogan"`
		Bio              string `json:"bio"`
		Manifesto        string `json:"manifesto"`
		ManifestoURL     string `json:"manifesto_url" binding:"required"`
		PhotoURL         string `json:"photo_url"`
		PassportPhotoURL string `json:"passport_photo_url"`
		ContactEmail     string `json:"contact_email"`
		ContactPhone     string `json:"contact_phone"`
		DateOfBirth      string `json:"date_of_birth"`
		CampusResidence  string `json:"campus_residence"`
		Sex              string `json:"sex"`
		Religion         string `json:"religion"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		respond.BadRequest(c, "Add your details and a manifesto file")
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
		respond.BadRequest(c, "Your biography or manifesto text is too long")
		return
	}
	contactEmail := strings.TrimSpace(input.ContactEmail)
	if contactEmail != "" {
		if _, err := mail.ParseAddress(contactEmail); err != nil {
			respond.BadRequest(c, "Enter a valid contact email address")
			return
		}
	}
	if !storedFileURL(input.ManifestoURL) || !storedFileURL(input.PhotoURL) ||
		!storedFileURL(input.PassportPhotoURL) {
		respond.BadRequest(c, "Uploaded files must be links to this site or an https address")
		return
	}
	manifesto, message := h.manifestoUpload(user.ID, input.ManifestoURL)
	if manifesto == nil {
		respond.Validation(c, message, map[string]string{"manifesto_url": message})
		return
	}
	dateOfBirth, err := parseDateOfBirth(input.DateOfBirth)
	if err != nil {
		respond.Validation(c, "Your date of birth is not valid", map[string]string{
			"date_of_birth": "use YYYY-MM-DD",
		})
		return
	}
	sex := strings.ToUpper(strings.TrimSpace(input.Sex))
	if sex != "MALE" && sex != "FEMALE" {
		respond.Validation(c, "Choose your sex", map[string]string{
			"sex": "must be MALE or FEMALE",
		})
		return
	}
	passportPhoto := strings.TrimSpace(input.PassportPhotoURL)
	if passportPhoto == "" {
		respond.Validation(c, "A passport photo is required", map[string]string{
			"passport_photo_url": "upload a passport photo first",
		})
		return
	}
	photo := strings.TrimSpace(input.PhotoURL)
	if photo == "" {
		photo = passportPhoto
	}

	identity, parsed := eligibility.ParseEmail(user.Email)
	if !parsed {
		respond.Forbidden(c, "Your email address does not match the student registration format")
		return
	}

	now := h.now()
	candidate.FullName = fullName
	candidate.Slogan = strings.TrimSpace(input.Slogan)
	candidate.Bio = strings.TrimSpace(input.Bio)
	candidate.Manifesto = strings.TrimSpace(input.Manifesto)
	candidate.ManifestoURL = manifesto.URL
	candidate.PhotoURL = photo
	candidate.PassportPhotoURL = passportPhoto
	candidate.ContactEmail = contactEmail
	candidate.ContactPhone = strings.TrimSpace(input.ContactPhone)
	candidate.DateOfBirth = dateOfBirth
	candidate.CampusResidence = strings.TrimSpace(input.CampusResidence)
	candidate.Sex = sex
	candidate.Religion = strings.TrimSpace(input.Religion)
	candidate.RegistrationNumber = strings.ToUpper(identity.Year + identity.Course + identity.Number)
	candidate.Programme = identity.Programme()
	candidate.StudyYear = identity.StudyYear(now)
	candidate.ApprovalStatus = models.CandidateStatusApproved
	candidate.ReviewedAt = &now
	if strings.TrimSpace(candidate.ReviewNotes) == "" {
		candidate.ReviewNotes = "Selected by an administrator"
	}

	if err := h.DB.Transaction(func(tx *gorm.DB) error {
		if candidate.BallotNumber == 0 {
			var highest int
			tx.Model(&models.Candidate{}).
				Where("position_id = ? AND id <> ?", candidate.PositionID, candidate.ID).
				Select("COALESCE(MAX(ballot_number), 0)").Scan(&highest)
			candidate.BallotNumber = highest + 1
		}
		if err := tx.Save(&candidate).Error; err != nil {
			return err
		}
		return tx.Create(&models.ElectionAuditLog{
			ElectionID: candidate.ElectionID,
			ActorID:    user.ID,
			Action:     models.AuditActionCandidateApproved,
			Details:    "Selected candidate published: " + candidate.FullName,
			IPAddress:  c.ClientIP(),
		}).Error
	}); err != nil {
		respond.Internal(c, err)
		return
	}
	respond.OK(c, candidate, "Candidacy published")
}

// loadInvitation fetches an invitation addressed to the signed-in account. It
// writes the response itself when the id is not the caller's invitation.
func (h *CandidateHandler) loadInvitation(c *gin.Context, user *models.User) (models.Candidate, bool) {
	var candidate models.Candidate
	if err := h.DB.Where("id = ? AND invited_email <> ''", c.Param("candidate_id")).
		First(&candidate).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respond.NotFound(c, "Invitation not found")
			return candidate, false
		}
		respond.Internal(c, err)
		return candidate, false
	}
	if !strings.EqualFold(strings.TrimSpace(candidate.InvitedEmail), strings.TrimSpace(user.Email)) {
		respond.NotFound(c, "Invitation not found")
		return candidate, false
	}
	if user.EmailVerifiedAt == nil {
		respond.Forbidden(c, "Confirm your email address before accepting an invitation")
		return candidate, false
	}
	return candidate, true
}

// manifestoUpload resolves the caller's stored upload and checks its real type
// and size. A URL that is not one of this account's uploads is refused, so a
// candidate cannot point the ballot at somebody else's file or an outside page.
func (h *CandidateHandler) manifestoUpload(userID, rawURL string) (*models.Upload, string) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return nil, "Upload a manifesto PDF, MP3 or MP4"
	}
	var upload models.Upload
	err := h.DB.Where("user_id = ? AND (url = ? OR path = ?)", userID, rawURL, rawURL).
		First(&upload).Error
	if err != nil {
		return nil, "Upload your manifesto with the file picker first"
	}
	if !manifestoTypeAllowed(upload.MimeType) {
		return nil, "The manifesto must be a PDF, MP3 or MP4"
	}
	if upload.Size <= 0 || upload.Size > manifestoMaxBytes {
		return nil, fmt.Sprintf("The manifesto must be %d MB or smaller", manifestoMaxBytes/(1<<20))
	}
	return &upload, ""
}
