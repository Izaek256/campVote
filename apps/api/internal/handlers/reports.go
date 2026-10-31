package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"campvote/apps/api/internal/authz"
	"campvote/apps/api/internal/handlers/respond"
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
	electionID := c.Param("id")

	// Authorization: require perm:elections.view or ADMIN
	if !authz.Check(c, "perm:elections.view") && !authz.Check(c, "role:ADMIN") {
		respond.Forbidden(c, "You do not have permission to view election reports")
		return
	}

	pdfBytes, err := reports.ElectionSummary(h.DB, electionID)
	if err != nil {
		respond.WriteError(c, err, "Failed to generate election summary")
		return
	}

	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", `attachment; filename="election-summary-`+electionID+`.pdf"`)
	c.Data(http.StatusOK, "application/pdf", pdfBytes)
}

// ResultsVerificationPDF generates a PDF results verification report.
// GET /api/v1/elections/:id/reports/verification.pdf
func (h *ReportsHandler) ResultsVerificationPDF(c *gin.Context) {
	electionID := c.Param("id")

	// Authorization: require perm:elections.view or ADMIN
	if !authz.Check(c, "perm:elections.view") && !authz.Check(c, "role:ADMIN") {
		respond.Forbidden(c, "You do not have permission to view election reports")
		return
	}

	pdfBytes, err := reports.ResultsVerification(h.DB, electionID)
	if err != nil {
		respond.WriteError(c, err, "Failed to generate verification report")
		return
	}

	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", `attachment; filename="results-verification-`+electionID+`.pdf"`)
	c.Data(http.StatusOK, "application/pdf", pdfBytes)
}
