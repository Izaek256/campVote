package handlers

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"campvote/apps/api/internal/lifecycle"
	"campvote/apps/api/internal/models"
	"campvote/apps/api/internal/policy"
	"campvote/apps/api/internal/respond"
)

// cloneElectionRequest is the only input a re-run needs: a new schedule, and
// optionally a new title. Everything else is copied from the source election.
type cloneElectionRequest struct {
	Title    string `json:"title"`
	StartsAt string `json:"starts_at"`
	EndsAt   string `json:"ends_at"`
}

// Clone creates a new DRAFT election from an existing one.
//
// It copies the configuration, eligibility rule, positions, candidates and
// roster, so a recurring poll does not have to be rebuilt by hand. It copies no
// votes, results, ballot versions or role assignments: the new election is a
// fresh contest, and the caller becomes its creator. Withdrawn (and rejected)
// candidates are left behind because they are not part of the ballot.
func (h *ElectionLifecycleHandler) Clone(c *gin.Context) {
	authority, ok := h.authority(c, policy.PermElectionCreate)
	if !ok {
		return
	}
	var in cloneElectionRequest
	if err := c.ShouldBindJSON(&in); err != nil {
		respond.BadRequest(c, "A start and end time are required for the copy")
		return
	}
	startsAt, ok := scheduleField(c, "starts_at", in.StartsAt)
	if !ok {
		return
	}
	endsAt, ok := scheduleField(c, "ends_at", in.EndsAt)
	if !ok {
		return
	}
	now := h.now()
	source := authority.Election

	title := strings.TrimSpace(in.Title)
	if title == "" {
		title = source.Title + " (copy)"
	}
	clone := &models.Election{
		Title:          title,
		Description:    source.Description,
		Category:       source.Category,
		OrganizingUnit: source.OrganizingUnit,
		Status:         models.ElectionStatusDraft,
		StartsAt:       startsAt,
		EndsAt:         endsAt,
		ResultPolicy:   source.ResultPolicy,
		AllowedDomain:  source.AllowedDomain,
		AllowedYears:   source.AllowedYears,
		AllowedCourses: source.AllowedCourses,
		RequireRoster:  source.RequireRoster,
		PublicListing:  source.PublicListing,
		CreatedByID:    c.GetString("user_id"),
	}
	if err := lifecycle.ValidateBasics(clone, now, false); err != nil {
		respond.WriteError(c, err, "The copied schedule is not valid")
		return
	}

	err := h.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(clone).Error; err != nil {
			return err
		}
		positions, candidates, err := cloneBallot(tx, source.ID, clone.ID)
		if err != nil {
			return err
		}
		if err := cloneRoster(tx, source.ID, clone.ID); err != nil {
			return err
		}
		details, _ := json.Marshal(map[string]interface{}{
			"source_election_id": source.ID,
			"source_title":       source.Title,
			"starts_at":          clone.StartsAt.Format(time.RFC3339),
			"ends_at":            clone.EndsAt.Format(time.RFC3339),
			"positions":          positions,
			"candidates":         candidates,
		})
		return tx.Create(&models.ElectionAuditLog{
			ElectionID: clone.ID,
			ActorID:    c.GetString("user_id"),
			Action:     models.AuditActionElectionCloned,
			Details:    string(details),
			IPAddress:  c.ClientIP(),
			CreatedAt:  now,
		}).Error
	})
	if err != nil {
		respond.WriteError(c, err, "The election could not be copied")
		return
	}
	respond.Created(c, h.view(clone, now), "Election copied")
}

// cloneBallot copies the source positions and the candidates still in the
// contest, remapping each candidate to its new position.
func cloneBallot(tx *gorm.DB, sourceID, cloneID string) (int, int, error) {
	var sourcePositions []models.Position
	if err := tx.Where("election_id = ?", sourceID).Order("display_order, created_at").
		Find(&sourcePositions).Error; err != nil {
		return 0, 0, err
	}
	positionMap := make(map[string]string, len(sourcePositions))
	for i := range sourcePositions {
		old := sourcePositions[i]
		next := old
		next.ID = ""
		next.ElectionID = &cloneID
		next.CreatedAt, next.UpdatedAt, next.DeletedAt = time.Time{}, time.Time{}, gorm.DeletedAt{}
		next.Candidates = nil
		if err := tx.Create(&next).Error; err != nil {
			return 0, 0, err
		}
		positionMap[old.ID] = next.ID
	}

	var sourceCandidates []models.Candidate
	if err := tx.Where("election_id = ? AND approval_status NOT IN ?", sourceID,
		[]string{models.CandidateStatusWithdrawn, models.CandidateStatusRejected}).
		Order("created_at").Find(&sourceCandidates).Error; err != nil {
		return 0, 0, err
	}
	copied := 0
	for i := range sourceCandidates {
		old := sourceCandidates[i]
		positionID, ok := positionMap[old.PositionID]
		if !ok {
			// The candidate belongs to a position outside this election; it is
			// not on the ballot being copied.
			continue
		}
		next := old
		next.ID = ""
		next.ElectionID = cloneID
		next.PositionID = positionID
		next.CreatedAt, next.UpdatedAt, next.DeletedAt = time.Time{}, time.Time{}, gorm.DeletedAt{}
		// Review provenance belongs to the previous contest; the copied record
		// keeps the decision but is not attributed to the old reviewer.
		next.ReviewedBy, next.ReviewedAt = "", nil
		if err := tx.Create(&next).Error; err != nil {
			return 0, 0, err
		}
		copied++
	}
	return len(sourcePositions), copied, nil
}

// cloneRoster copies the voter list. Provenance is reset because the rows were
// not imported for this election; the status (including a revocation) is kept.
func cloneRoster(tx *gorm.DB, sourceID, cloneID string) error {
	var rows []models.ElectionRoster
	if err := tx.Where("election_id = ?", sourceID).Find(&rows).Error; err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}
	copies := make([]models.ElectionRoster, 0, len(rows))
	for i := range rows {
		next := rows[i]
		next.ID = ""
		next.ElectionID = cloneID
		next.Source = models.RosterSourceManual
		next.ImportBatchID, next.ImportedByID = "", ""
		next.CreatedAt, next.UpdatedAt, next.DeletedAt = time.Time{}, time.Time{}, gorm.DeletedAt{}
		copies = append(copies, next)
	}
	return tx.Create(&copies).Error
}
