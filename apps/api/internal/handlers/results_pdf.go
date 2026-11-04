package handlers

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"campvote/apps/api/internal/eligibility"
	"campvote/apps/api/internal/models"
	"campvote/apps/api/internal/policy"
	"campvote/apps/api/internal/reports"
	"campvote/apps/api/internal/respond"
	"campvote/apps/api/internal/results"
)

// ResultsPDF GET /api/v1/elections/:id/results.pdf
//
// The downloadable form of what /results already publishes. It is refused with
// the same word a reader of /results would see until the publication policy
// allows the figures, and it renders the same snapshot that caller is shown, so
// a download can never reveal more than the screen.
func (h *ResultsHandler) ResultsPDF(c *gin.Context) {
	election, authority, ok := h.election(c)
	if !ok {
		return
	}

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
		if results.Closed(election.Status) {
			if _, tallyErr := results.EnsureOnClose(h.DB, &election, h.now()); tallyErr == nil {
				snapshot, err = h.currentSnapshot(election.ID, staff)
			}
		}
	}
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respond.Conflict(c, "Results have not been tallied for this election yet")
			return
		}
		respond.Internal(c, err)
		return
	}

	pdfBytes, err := reports.ResultsLedger(h.DB, election, snapshot)
	if err != nil {
		respond.WriteError(c, err, "Failed to generate the results ledger")
		return
	}
	writePDF(c, fmt.Sprintf("results-%s.pdf", election.ID), pdfBytes)
}

// InfoPDF GET /api/v1/elections/:id/info.pdf
//
// The printed notice: times, who may vote, approved candidates and how to
// vote. It contains public configuration only. It obeys the same visibility
// rule as the election list, so a draft is downloadable only by the students
// it will admit.
func (h *ResultsHandler) InfoPDF(c *gin.Context) {
	var election models.Election
	if err := h.DB.Where("id = ?", c.Param("id")).First(&election).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respond.NotFound(c, "Election not found")
			return
		}
		respond.Internal(c, err)
		return
	}

	actor := policy.ActorFrom(c)
	if !actor.Admin {
		var callerID eligibility.Identity
		var callerParsed bool
		var user models.User
		if err := h.DB.Where("id = ?", actor.UserID).First(&user).Error; err == nil {
			callerID, callerParsed = eligibility.ParseEmail(user.Email)
		}
		if !electionVisibleTo(h.DB, &election, actor.UserID, callerID, callerParsed) {
			respond.NotFound(c, "Election not found")
			return
		}
	}

	pdfBytes, err := reports.ElectionInfoSheet(h.DB, election)
	if err != nil {
		respond.WriteError(c, err, "Failed to generate the election information sheet")
		return
	}
	writePDF(c, fmt.Sprintf("election-info-%s.pdf", election.ID), pdfBytes)
}

// writePDF streams a generated document as a download.
func writePDF(c *gin.Context, filename string, body []byte) {
	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", `attachment; filename="`+filename+`"`)
	c.Data(http.StatusOK, "application/pdf", body)
}
