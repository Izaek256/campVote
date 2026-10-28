package handlers

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"campvote/apps/api/internal/eligibility"
	"campvote/apps/api/internal/models"
	"campvote/apps/api/internal/policy"
	"campvote/apps/api/internal/respond"
	"campvote/apps/api/internal/services"
)

// EligibilityHandler owns roster imports, revocation/override, freeze and the
// officer report. Every method resolves the election through policy.AuthorityFor
// first, so a cross-election request is 404 rather than a leak.
type EligibilityHandler struct {
	DB *gorm.DB
}

func NewEligibilityHandler(db *gorm.DB) *EligibilityHandler { return &EligibilityHandler{DB: db} }

// Mine returns the caller's own decision for one election. It is the voter's
// view: no roster rows and no other identities are exposed.
func (h *EligibilityHandler) Mine(c *gin.Context) {
	decision, err := eligibility.Check(h.DB, c.Param("id"), requestUser(c), time.Now().UTC())
	if errors.Is(err, gorm.ErrRecordNotFound) {
		respond.NotFound(c, "Election not found")
		return
	}
	if err != nil {
		respond.Internal(c, err)
		return
	}
	respond.OK(c, decision)
}

// Preview validates an uploaded CSV and reports what a commit would change.
func (h *EligibilityHandler) Preview(c *gin.Context) {
	if _, ok := h.authority(c, policy.PermRosterImport); !ok {
		return
	}
	body, name, err := rosterUpload(c)
	if err != nil {
		respond.BadRequest(c, err.Error())
		return
	}
	defer body.Close()

	plan, err := eligibility.Preview(h.DB, c.Param("id"), body, time.Now().UTC())
	if errors.Is(err, gorm.ErrRecordNotFound) {
		respond.NotFound(c, "Election not found")
		return
	}
	if err != nil {
		respond.BadRequest(c, err.Error())
		return
	}
	respond.OK(c, gin.H{"file_name": name, "preview": plan.View()})
}

// Import commits a validated roster CSV. A file with any invalid row is
// refused as a whole, so preview and commit cannot disagree row by row.
func (h *EligibilityHandler) Import(c *gin.Context) {
	authority, ok := h.authority(c, policy.PermRosterImport)
	if !ok {
		return
	}
	body, name, err := rosterUpload(c)
	if err != nil {
		respond.BadRequest(c, err.Error())
		return
	}
	defer body.Close()

	batch, plan, err := eligibility.Import(h.DB, authority.Election.ID, c.GetString("user_id"),
		models.RosterSourceCSVImport, name, body, time.Now().UTC())
	var invalid *eligibility.ValidationError
	if errors.As(err, &invalid) {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"error": gin.H{
				"code":    respond.CodeValidationError,
				"message": err.Error(),
				"details": gin.H{"rows": invalid.Plan.Errors},
			},
			"data": invalid.Plan.View(),
		})
		return
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		respond.NotFound(c, "Election not found")
		return
	}
	if err != nil {
		respond.Internal(c, err)
		return
	}
	h.audit(c, authority.Election.ID, "roster.import",
		"Imported roster "+batch.ID+" for "+authority.Election.Title)
	respond.Created(c, gin.H{"batch": batch, "preview": plan.View()}, "Roster imported")
}

type rosterStatusRequest struct {
	Status string `json:"status"`
	Reason string `json:"reason"`
}

// SetStatus revokes or restores one roster entry. A reason is mandatory and the
// change is written to the election audit log.
func (h *EligibilityHandler) SetStatus(c *gin.Context) {
	authority, ok := h.authority(c, policy.PermRosterEdit)
	if !ok {
		return
	}
	var in rosterStatusRequest
	if err := c.ShouldBindJSON(&in); err != nil {
		respond.BadRequest(c, "A status and reason are required")
		return
	}
	if strings.TrimSpace(in.Status) == "" {
		in.Status = models.RosterStatusIneligible
	}
	entry, err := eligibility.SetStatus(h.DB, authority.Election.ID, c.Param("rosterId"),
		c.GetString("user_id"), in.Status, in.Reason)
	if errors.Is(err, eligibility.ErrElectionClosed) {
		respond.Conflict(c, "Eligibility can no longer be changed after voting closes")
		return
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		respond.NotFound(c, "Roster entry not found")
		return
	}
	if err != nil {
		respond.BadRequest(c, err.Error())
		return
	}
	h.audit(c, authority.Election.ID, "roster.status_changed",
		"Changed roster status to "+entry.Status+" for "+authority.Election.Title)
	respond.OK(c, entry)
}

// Freeze snapshots the eligible count when voting opens. It is idempotent.
func (h *EligibilityHandler) Freeze(c *gin.Context) {
	authority, ok := h.authority(c, policy.PermRosterEdit)
	if !ok {
		return
	}
	election, err := eligibility.Freeze(h.DB, authority.Election.ID, c.GetString("user_id"), time.Now().UTC())
	if errors.Is(err, gorm.ErrRecordNotFound) {
		respond.NotFound(c, "Election not found")
		return
	}
	if err != nil {
		respond.Conflict(c, err.Error())
		return
	}
	h.audit(c, authority.Election.ID, "eligibility.frozen",
		"Froze eligibility for "+authority.Election.Title)
	respond.OK(c, gin.H{
		"election_id":  election.ID,
		"frozen_at":    election.EligibilityFrozenAt,
		"frozen_count": election.EligibilityFrozenCount,
	})
}

// Report returns counts only: no identities and no ballot linkage.
func (h *EligibilityHandler) Report(c *gin.Context) {
	authority, ok := h.authority(c, policy.PermRosterView)
	if !ok {
		return
	}
	report, err := eligibility.BuildReport(h.DB, authority.Election.ID)
	if err != nil {
		respond.Internal(c, err)
		return
	}
	respond.OK(c, report)
}

func (h *EligibilityHandler) authority(c *gin.Context, permission string) (*policy.Authority, bool) {
	actor := policy.ActorFrom(c)
	authority, err := policy.AuthorityFor(h.DB, actor, c.Param("id"))
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

func (h *EligibilityHandler) audit(c *gin.Context, electionID, action, summary string) {
	services.LogActivity(h.DB, c, services.ActivityArgs{
		Action:       action,
		Severity:     "info",
		Summary:      summary,
		ResourceType: "election",
		ResourceID:   electionID,
	})
}

// rosterUpload accepts the same file either as multipart/form-data (the admin
// upload path) or as a raw text/csv body (scripts and tests).
func rosterUpload(c *gin.Context) (io.ReadCloser, string, error) {
	if strings.HasPrefix(c.ContentType(), "multipart/form-data") {
		header, err := c.FormFile("file")
		if err != nil {
			return nil, "", errors.New("attach the roster CSV as the \"file\" field")
		}
		file, err := header.Open()
		if err != nil {
			return nil, "", errors.New("the uploaded file could not be read")
		}
		return file, header.Filename, nil
	}
	if c.Request.Body == nil {
		return nil, "", errors.New("a CSV body is required")
	}
	return io.NopCloser(c.Request.Body), "", nil
}
