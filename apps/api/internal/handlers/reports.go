package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"campvote/apps/api/internal/policy"
	"campvote/apps/api/internal/respond"
	"campvote/apps/api/internal/reports"
)

// ReportsHandler handles election report generation endpoints.
type ReportsHandler struct {
	DB *gorm.DB
}

// NewReportsHandler creates a new ReportsHandler.
func NewReportsHandler(db *gorm.DB) *ReportsHandler {
	return &ReportsHandler{DB: db}
}

// ElectionSummaryPDF generates a PDF summary of an election.
// GET /api/v1/elections/:id/reports/summary.pdf
func (h *ReportsHandler) ElectionSummaryPDF(c *gin.Context) {
	authority, ok := h.authority(c, policy.PermElectionView)
	if !ok {
		return
	}

	pdfBytes, err := reports.ElectionSummary(h.DB, authority.Election.ID)
	if err != nil {
		respond.WriteError(c, err, "Failed to generate election summary")
		return
	}

	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", `attachment; filename="election-summary-`+authority.Election.ID+`.pdf"`)
	c.Data(http.StatusOK, "application/pdf", pdfBytes)
}

// ResultsVerificationPDF generates a PDF results verification report.
// GET /api/v1/elections/:id/reports/verification.pdf
func (h *ReportsHandler) ResultsVerificationPDF(c *gin.Context) {
	authority, ok := h.authority(c, policy.PermElectionView)
	if !ok {
		return
	}

	pdfBytes, err := reports.ResultsVerification(h.DB, authority.Election.ID)
	if err != nil {
		respond.WriteError(c, err, "Failed to generate verification report")
		return
	}

	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", `attachment; filename="results-verification-`+authority.Election.ID+`.pdf"`)
	c.Data(http.StatusOK, "application/pdf", pdfBytes)
}

// authority resolves the election and the permission, writing the refusal.
func (h *ReportsHandler) authority(c *gin.Context, permission string) (*policy.Authority, bool) {
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
		respond.Forbidden(c, "You do not have permission to view election reports")
		return nil, false
	}
	return authority, true
}
