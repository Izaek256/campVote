package eligibility

import (
	"bytes"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/mail"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	"campvote/apps/api/internal/models"
)

// MaxImportBytes caps a roster upload. A real roll is far smaller; the cap is
// there so a malformed or hostile upload cannot consume the process.
const MaxImportBytes = 8 << 20

// Row is one accepted roster line, already trimmed and normalized.
type Row struct {
	StudentNumber string
	RegNumber     string
	Email         string
	FullName      string
	Faculty       string
	Department    string
	Programme     string
	StudyYear     int
	Status        string
}

// RowError names the line and the reason it was refused. It carries enough
// identity to let an officer fix the file, and is returned only to callers with
// rosters.import.
type RowError struct {
	Line          int    `json:"line"`
	StudentNumber string `json:"student_number,omitempty"`
	Email         string `json:"email,omitempty"`
	Reason        string `json:"reason"`
}

// PlanRow is one valid row plus what the commit will do with it.
type PlanRow struct {
	Row
	ExistingID    string
	Update        bool
	Revoked       bool
	StatusChanged bool
}

// Plan is the result of validating a file without writing it.
type Plan struct {
	Election    models.Election `json:"-"`
	Rows        []PlanRow       `json:"-"`
	Errors      []RowError      `json:"errors"`
	Creates     int             `json:"creates"`
	Updates     int             `json:"updates"`
	Unchanged   int             `json:"unchanged"`
	Revokes     int             `json:"revokes"`
	ContentHash string          `json:"content_hash"`
}

// ValidationError means the file is not safe to commit. Nothing was written.
type ValidationError struct{ Plan *Plan }

func (e *ValidationError) Error() string {
	return fmt.Sprintf("roster import has %d invalid row(s); nothing was committed", len(e.Plan.Errors))
}

// Preview parses and validates an upload against the live roster.
func Preview(db *gorm.DB, electionID string, r io.Reader, now time.Time) (*Plan, error) {
	var election models.Election
	if err := db.Where("id = ?", electionID).First(&election).Error; err != nil {
		return nil, err
	}
	return BuildPlan(db, &election, r, now)
}

// BuildPlan validates every row, detects duplicate identities in the file and
// against the existing roster, and records what a commit would change.
func BuildPlan(db *gorm.DB, election *models.Election, r io.Reader, now time.Time) (*Plan, error) {
	raw, err := io.ReadAll(io.LimitReader(r, MaxImportBytes+1))
	if err != nil {
		return nil, err
	}
	if len(raw) > MaxImportBytes {
		return nil, fmt.Errorf("roster file is larger than %d bytes", MaxImportBytes)
	}
	sum := sha256.Sum256(raw)

	records, err := csv.NewReader(bytes.NewReader(raw)).ReadAll()
	if err != nil {
		return nil, fmt.Errorf("read CSV: %w", err)
	}
	if len(records) == 0 {
		return nil, errors.New("the roster file is empty")
	}

	index, err := headerIndex(records[0])
	if err != nil {
		return nil, err
	}

	plan := &Plan{Election: *election, ContentHash: hex.EncodeToString(sum[:])}
	var existing []models.ElectionRoster
	if err := db.Where("election_id = ?", election.ID).Find(&existing).Error; err != nil {
		return nil, err
	}
	byStudent := map[string]*models.ElectionRoster{}
	usedEmail := map[string]*models.ElectionRoster{}
	for i := range existing {
		byStudent[strings.ToLower(existing[i].StudentNumber)] = &existing[i]
		usedEmail[strings.ToLower(existing[i].Email)] = &existing[i]
	}

	seenStudent := map[string]int{}
	seenEmail := map[string]int{}
	for i, record := range records[1:] {
		line := i + 2
		row, reason := parseRow(record, index)
		if reason != "" {
			plan.Errors = append(plan.Errors, RowError{Line: line, StudentNumber: row.StudentNumber, Email: row.Email, Reason: reason})
			continue
		}
		if first, dup := seenStudent[strings.ToLower(row.StudentNumber)]; dup {
			plan.Errors = append(plan.Errors, RowError{Line: line, StudentNumber: row.StudentNumber, Email: row.Email,
				Reason: fmt.Sprintf("duplicate student number; first seen on line %d", first)})
			continue
		}
		if first, dup := seenEmail[strings.ToLower(row.Email)]; dup {
			plan.Errors = append(plan.Errors, RowError{Line: line, StudentNumber: row.StudentNumber, Email: row.Email,
				Reason: fmt.Sprintf("duplicate identity: this email is already on line %d", first)})
			continue
		}
		seenStudent[strings.ToLower(row.StudentNumber)] = line
		seenEmail[strings.ToLower(row.Email)] = line

		current := byStudent[strings.ToLower(row.StudentNumber)]
		if current == nil {
			if other := usedEmail[strings.ToLower(row.Email)]; other != nil {
				plan.Errors = append(plan.Errors, RowError{Line: line, StudentNumber: row.StudentNumber, Email: row.Email,
					Reason: "duplicate identity: this email already belongs to another student number in this election"})
				continue
			}
			if election.EligibilityFrozenAt != nil {
				plan.Errors = append(plan.Errors, RowError{Line: line, StudentNumber: row.StudentNumber, Email: row.Email,
					Reason: "roster is frozen; new entries are not accepted after voting opens"})
				continue
			}
			plan.Rows = append(plan.Rows, PlanRow{Row: row})
			plan.Creates++
			if row.Status != models.RosterStatusEligible {
				plan.Revokes++
			}
			continue
		}

		if other := usedEmail[strings.ToLower(row.Email)]; other != nil && other.ID != current.ID {
			plan.Errors = append(plan.Errors, RowError{Line: line, StudentNumber: row.StudentNumber, Email: row.Email,
				Reason: "duplicate identity: this email already belongs to another student number in this election"})
			continue
		}
		if election.EligibilityFrozenAt != nil {
			if !sameIdentity(*current, row) {
				plan.Errors = append(plan.Errors, RowError{Line: line, StudentNumber: row.StudentNumber, Email: row.Email,
					Reason: "roster is frozen; only a status change is accepted after voting opens"})
				continue
			}
			if row.Status == models.RosterStatusEligible && current.Status != models.RosterStatusEligible {
				plan.Errors = append(plan.Errors, RowError{Line: line, StudentNumber: row.StudentNumber, Email: row.Email,
					Reason: "roster is frozen; eligibility cannot be restored by import (use an explicit override)"})
				continue
			}
		}

		update := !sameRow(*current, row)
		statusChanged := current.Status != row.Status
		plan.Rows = append(plan.Rows, PlanRow{
			Row: row, ExistingID: current.ID, Update: update, StatusChanged: statusChanged,
		})
		if update {
			plan.Updates++
		} else {
			plan.Unchanged++
		}
		if row.Status != models.RosterStatusEligible && current.Status == models.RosterStatusEligible {
			plan.Revokes++
			plan.Rows[len(plan.Rows)-1].Revoked = true
		}
		if !strings.EqualFold(current.Email, row.Email) {
			delete(usedEmail, strings.ToLower(current.Email))
			usedEmail[strings.ToLower(row.Email)] = current
		}
	}
	return plan, nil
}

// Import validates again inside a transaction and commits every row or none.
// The second validation closes the gap between preview and commit.
func Import(db *gorm.DB, electionID, actorID, source, fileName string, r io.Reader, now time.Time) (*models.EligibilityImportBatch, *Plan, error) {
	if source == "" {
		source = models.RosterSourceCSVImport
	}
	var batch *models.EligibilityImportBatch
	var plan *Plan
	err := db.Transaction(func(tx *gorm.DB) error {
		var election models.Election
		if err := tx.Where("id = ?", electionID).First(&election).Error; err != nil {
			return err
		}
		built, err := BuildPlan(tx, &election, r, now)
		if err != nil {
			return err
		}
		plan = built
		if len(plan.Errors) > 0 {
			return &ValidationError{Plan: plan}
		}

		batch = &models.EligibilityImportBatch{
			ElectionID:   election.ID,
			ImportedByID: actorID,
			Source:       source,
			FileName:     strings.TrimSpace(fileName),
			ContentHash:  plan.ContentHash,
			Total:        len(plan.Rows) + len(plan.Errors),
			Created:      plan.Creates,
			Updated:      plan.Updates,
			Revoked:      plan.Revokes,
		}
		if err := tx.Create(batch).Error; err != nil {
			return err
		}

		var revokedIDs []string
		for _, row := range plan.Rows {
			statusChangedAt := any(nil)
			statusReason := ""
			if row.Status != models.RosterStatusEligible {
				statusChangedAt = now.UTC()
				statusReason = "imported as " + strings.ToLower(row.Status)
			}
			if row.ExistingID == "" {
				entry := models.ElectionRoster{
					ElectionID:    election.ID,
					StudentNumber: row.StudentNumber,
					RegNumber:     row.RegNumber,
					Email:         row.Email,
					FullName:      row.FullName,
					Faculty:       row.Faculty,
					Department:    row.Department,
					Programme:     row.Programme,
					StudyYear:     row.StudyYear,
					Status:        row.Status,
					Source:        source,
					ImportBatchID: batch.ID,
					ImportedByID:  actorID,
					StatusReason:  statusReason,
				}
				if row.Status != models.RosterStatusEligible {
					at := now.UTC()
					entry.StatusChangedAt = &at
				}
				if err := tx.Create(&entry).Error; err != nil {
					return err
				}
				if row.Status != models.RosterStatusEligible {
					revokedIDs = append(revokedIDs, entry.ID)
				}
				continue
			}

			updates := map[string]interface{}{
				"reg_number":      row.RegNumber,
				"email":           row.Email,
				"full_name":       row.FullName,
				"faculty":         row.Faculty,
				"department":      row.Department,
				"programme":       row.Programme,
				"study_year":      row.StudyYear,
				"status":          row.Status,
				"source":          source,
				"import_batch_id": batch.ID,
				"imported_by_id":  actorID,
			}
			if row.StatusChanged {
				updates["status_reason"] = statusReason
				updates["status_changed_at"] = statusChangedAt
			}
			if err := tx.Model(&models.ElectionRoster{}).Where("id = ?", row.ExistingID).Updates(updates).Error; err != nil {
				return err
			}
			if row.Revoked {
				revokedIDs = append(revokedIDs, row.ExistingID)
			}
		}

		details, _ := json.Marshal(map[string]interface{}{
			"batch_id":     batch.ID,
			"source":       source,
			"file_name":    batch.FileName,
			"content_hash": plan.ContentHash,
			"created":      plan.Creates,
			"updated":      plan.Updates,
			"revoked":      plan.Revokes,
		})
		if err := tx.Create(&models.ElectionAuditLog{
			ElectionID: election.ID,
			ActorID:    actorID,
			Action:     models.AuditActionRosterImported,
			Details:    string(details),
		}).Error; err != nil {
			return err
		}
		for _, id := range revokedIDs {
			eventDetails, _ := json.Marshal(map[string]string{"roster_id": id, "reason": "imported revocation"})
			if err := tx.Create(&models.ElectionAuditLog{
				ElectionID: election.ID,
				ActorID:    actorID,
				Action:     models.AuditActionRosterRevoked,
				Details:    string(eventDetails),
			}).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		var invalid *ValidationError
		if errors.As(err, &invalid) {
			return nil, invalid.Plan, err
		}
		return nil, plan, err
	}
	return batch, plan, nil
}

// PreviewView is the safe response shape for an import preview.
type PreviewView struct {
	Creates     int        `json:"creates"`
	Updates     int        `json:"updates"`
	Unchanged   int        `json:"unchanged"`
	Revokes     int        `json:"revokes"`
	Errors      []RowError `json:"errors"`
	ContentHash string     `json:"content_hash"`
}

func (p *Plan) View() PreviewView {
	return PreviewView{
		Creates: p.Creates, Updates: p.Updates, Unchanged: p.Unchanged,
		Revokes: p.Revokes, Errors: p.Errors, ContentHash: p.ContentHash,
	}
}

func headerIndex(header []string) (map[string]int, error) {
	index := map[string]int{}
	for i, name := range header {
		name = strings.TrimPrefix(strings.TrimSpace(name), "\uFEFF")
		name = strings.ToLower(strings.ReplaceAll(name, " ", "_"))
		index[name] = i
	}
	for _, required := range []string{"student_number", "email", "full_name"} {
		if _, ok := index[required]; !ok {
			return nil, fmt.Errorf("missing required column %q", required)
		}
	}
	return index, nil
}

func parseRow(record []string, index map[string]int) (Row, string) {
	field := func(name string) string {
		i, ok := index[name]
		if !ok || i >= len(record) {
			return ""
		}
		return strings.TrimSpace(record[i])
	}
	row := Row{
		StudentNumber: field("student_number"),
		RegNumber:     field("reg_number"),
		Email:         strings.ToLower(field("email")),
		FullName:      field("full_name"),
		Faculty:       field("faculty"),
		Department:    field("department"),
		Programme:     field("programme"),
		Status:        strings.ToUpper(field("status")),
	}
	if row.StudentNumber == "" && row.Email == "" && row.FullName == "" {
		return row, "row is empty"
	}
	if row.StudentNumber == "" {
		return row, "student_number is required"
	}
	if row.FullName == "" {
		return row, "full_name is required"
	}
	if row.Email == "" {
		return row, "email is required"
	}
	address, err := mail.ParseAddress(row.Email)
	if err != nil || !strings.EqualFold(address.Address, row.Email) || !strings.Contains(row.Email, "@") {
		return row, "email is not a valid address"
	}
	if raw := field("study_year"); raw != "" {
		year, err := strconv.Atoi(raw)
		if err != nil || year < 1 || year > 10 {
			return row, "study_year must be a number between 1 and 10"
		}
		row.StudyYear = year
	} else {
		row.StudyYear = 1
	}
	if row.Status == "" {
		row.Status = models.RosterStatusEligible
	}
	if !validStatus(row.Status) {
		return row, "status must be ELIGIBLE, INELIGIBLE, GRADUATED or SUSPENDED"
	}
	return row, ""
}

func sameIdentity(current models.ElectionRoster, row Row) bool {
	return strings.EqualFold(current.StudentNumber, row.StudentNumber) &&
		strings.EqualFold(current.Email, row.Email) &&
		current.FullName == row.FullName &&
		current.RegNumber == row.RegNumber &&
		current.Faculty == row.Faculty &&
		current.Department == row.Department &&
		current.Programme == row.Programme &&
		current.StudyYear == row.StudyYear
}

func sameRow(current models.ElectionRoster, row Row) bool {
	return sameIdentity(current, row) && current.Status == row.Status
}
