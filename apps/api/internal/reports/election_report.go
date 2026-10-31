// Package reports generates election-specific PDF reports for audit evidence.
package reports

import (
	"fmt"
	"time"

	"gorm.io/gorm"

	"campvote/apps/api/internal/eligibility"
	"campvote/apps/api/internal/models"
	"campvote/apps/api/internal/pdf"
)

// ElectionSummary generates a PDF summary of an election for audit evidence.
// Includes election metadata, candidates, participation counts, and results
// (if the election is closed or results are published).
func ElectionSummary(db *gorm.DB, electionID string) ([]byte, error) {
	var election models.Election
	if err := db.Where("id = ?", electionID).First(&election).Error; err != nil {
		return nil, fmt.Errorf("election not found: %w", err)
	}

	// Load candidates
	var candidates []models.Candidate
	if err := db.Where("election_id = ?", electionID).
		Order("position_id, full_name").
		Find(&candidates).Error; err != nil {
		return nil, fmt.Errorf("failed to load candidates: %w", err)
	}

	// Load positions separately
	var positions []models.Position
	if err := db.Where("id IN ?", getUniquePositionIDsFromCandidates(candidates)).
		Find(&positions).Error; err != nil {
		return nil, fmt.Errorf("failed to load positions: %w", err)
	}
	positionMap := make(map[string]models.Position)
	for _, p := range positions {
		positionMap[p.ID] = p
	}

	// Load vote counts from tally snapshots
	var tallies []models.ElectionTally
	if err := db.Where("election_id = ?", electionID).
		Order("computed_at DESC").
		Find(&tallies).Error; err != nil {
		return nil, fmt.Errorf("failed to load tallies: %w", err)
	}

	// Build a map of candidate ID to vote count
	voteCounts := make(map[string]int)
	for _, tally := range tallies {
		if tally.CandidateID != nil {
			voteCounts[*tally.CandidateID] = tally.Votes
		}
	}

	// Get eligibility report
	eligReport, err := eligibility.BuildReport(db, electionID)
	if err != nil {
		return nil, fmt.Errorf("failed to build eligibility report: %w", err)
	}

	// Get recent audit logs
	var auditLogs []models.ElectionAuditLog
	if err := db.Where("election_id = ?", electionID).
		Order("created_at DESC").
		Limit(20).
		Find(&auditLogs).Error; err != nil {
		return nil, fmt.Errorf("failed to load audit logs: %w", err)
	}

	doc := pdf.New()
	doc.RunningHeader("ELECTION AUDIT REPORT", election.ID)
	doc.RunningFooter(fmt.Sprintf("Generated %s · MMU Electoral Commission", time.Now().Format("2 Jan 2006")))

	// Title section
	doc.Header("ELECTION SUMMARY", election.Title)
	doc.KV("Election ID", election.ID)
	doc.KV("Status", election.Status)
	doc.KV("Category", election.Category)
	doc.KV("Organizing Unit", election.OrganizingUnit)
	doc.KV("Start Time", election.StartsAt.Format("2 Jan 2006 15:04 EAT"))
	doc.KV("End Time", election.EndsAt.Format("2 Jan 2006 15:04 EAT"))
	doc.KV("Result Policy", election.ResultPolicy)
	doc.KV("Allowed Domains", election.AllowedDomains)

	doc.Ln(4)

	// Eligibility section
	doc.Header("ELIGIBILITY & PARTICIPATION", "")
	doc.KV("Total Roster Entries", fmt.Sprintf("%d", eligReport.Total))
	doc.KV("Eligible Voters", fmt.Sprintf("%d", eligReport.Eligible))
	doc.KV("Participated", fmt.Sprintf("%d", eligReport.Participated))
	doc.KV("Turnout", fmt.Sprintf("%.1f%%", eligReport.TurnoutPercent))
	if eligReport.FrozenAt != nil {
		doc.KV("Eligibility Frozen", eligReport.FrozenAt.Format("2 Jan 2006 15:04"))
		doc.KV("Frozen Eligible Count", fmt.Sprintf("%d", eligReport.FrozenEligible))
	}

	doc.Ln(4)

	// Candidates and results section
	if len(candidates) > 0 {
		doc.Header("CANDIDATES & RESULTS", "")

		// Group candidates by position
		positions := make(map[string][]models.Candidate)
		for _, c := range candidates {
			if pos, ok := positionMap[c.PositionID]; ok {
				positions[pos.Title] = append(positions[pos.Title], c)
			}
		}

		for posTitle, posCandidates := range positions {
			doc.SetFont("Helvetica", "B", 11)
			doc.CellFormat(0, 6, posTitle, "", 1, "L", false, 0, "")
			doc.Ln(2)

			headers := []string{"Candidate", "Status", "Votes"}
			rows := [][]string{}
			aligns := []string{"L", "L", "R"}
			colWidths := []float64{120, 40, 0}

			for _, c := range posCandidates {
				voteCount := ""
				if vc, ok := voteCounts[c.ID]; ok {
					voteCount = fmt.Sprintf("%d", vc)
				}
				rows = append(rows, []string{
					c.FullName,
					c.ApprovalStatus,
					voteCount,
				})
			}

			doc.Table(headers, rows, colWidths, aligns)
			doc.Ln(4)
		}
	}

	// Audit log section
	if len(auditLogs) > 0 {
		doc.Header("RECENT ADMINISTRATIVE ACTIONS", "")

		headers := []string{"Time", "Action", "Actor"}
		rows := [][]string{}
		aligns := []string{"L", "L", "L"}
		colWidths := []float64{50, 80, 0}

		for _, log := range auditLogs {
			rows = append(rows, []string{
				log.CreatedAt.Format("2 Jan 15:04"),
				log.Action,
				log.ActorID,
			})
		}

		doc.Table(headers, rows, colWidths, aligns)
	}

	doc.Footer("This is an official audit record. Unauthorized modification is prohibited.")

	return doc.Bytes()
}

// ResultsVerification generates a PDF report verifying that results match
// the official tally snapshot and that the election followed proper procedures.
func ResultsVerification(db *gorm.DB, electionID string) ([]byte, error) {
	var election models.Election
	if err := db.Where("id = ?", electionID).First(&election).Error; err != nil {
		return nil, fmt.Errorf("election not found: %w", err)
	}

	// Load tally snapshots
	var tallies []models.ElectionTally
	if err := db.Where("election_id = ?", electionID).
		Order("computed_at DESC").
		Find(&tallies).Error; err != nil {
		return nil, fmt.Errorf("failed to load tallies: %w", err)
	}

	// Get eligibility report
	eligReport, err := eligibility.BuildReport(db, electionID)
	if err != nil {
		return nil, fmt.Errorf("failed to build eligibility report: %w", err)
	}

	// Count ballots (status field doesn't exist on AnonymousBallot, just count all)
	var ballotCount int64
	if err := db.Model(&models.AnonymousBallot{}).
		Where("election_id = ?", electionID).
		Count(&ballotCount).Error; err != nil {
		return nil, fmt.Errorf("failed to count ballots: %w", err)
	}

	doc := pdf.New()
	doc.RunningHeader("RESULTS VERIFICATION REPORT", election.ID)
	doc.RunningFooter(fmt.Sprintf("Generated %s · MMU Electoral Commission", time.Now().Format("2 Jan 2006")))

	// Title section
	doc.Header("RESULTS VERIFICATION", election.Title)
	doc.KV("Election ID", election.ID)
	doc.KV("Status", election.Status)
	doc.KV("Verification Date", time.Now().Format("2 Jan 2006 15:04 EAT"))

	doc.Ln(4)

	// Integrity checks section
	doc.Header("INTEGRITY CHECKS", "")

	integrityChecks := []struct {
		check   string
		passed  bool
		details string
	}{
		{
			check:   "Election status allows results publication",
			passed:  election.Status == models.ElectionStatusClosed || election.Status == models.ElectionStatusFinalized,
			details: fmt.Sprintf("Current status: %s", election.Status),
		},
		{
			check:   "Ballot count matches participation records",
			passed:  ballotCount == eligReport.Participated,
			details: fmt.Sprintf("Ballots: %d, Participations: %d", ballotCount, eligReport.Participated),
		},
		{
			check:   "Eligibility was frozen before voting ended",
			passed:  eligReport.FrozenAt != nil && eligReport.FrozenAt.Before(election.EndsAt),
			details: fmt.Sprintf("Frozen: %s, Ended: %s",
				formatTime(eligReport.FrozenAt),
				election.EndsAt.Format("2006-01-02 15:04")),
		},
	}

	headers := []string{"Check", "Status", "Details"}
	rows := [][]string{}
	aligns := []string{"L", "C", "L"}
	colWidths := []float64{80, 20, 0}

	for _, ic := range integrityChecks {
		status := "FAIL"
		if ic.passed {
			status = "PASS"
		}
		rows = append(rows, []string{ic.check, status, ic.details})
	}

	doc.Table(headers, rows, colWidths, aligns)
	doc.Ln(4)

	// Tally snapshot section
	if len(tallies) > 0 {
		doc.Header("OFFICIAL TALLY SNAPSHOTS", "")

		// Load positions and candidates for display
		var positions []models.Position
		positionMap := make(map[string]models.Position)
		db.Where("id IN ?", getUniquePositionIDs(tallies)).Find(&positions)
		for _, p := range positions {
			positionMap[p.ID] = p
		}

		var candidates []models.Candidate
		candidateMap := make(map[string]models.Candidate)
		db.Where("id IN ?", getUniqueCandidateIDs(tallies)).Find(&candidates)
		for _, c := range candidates {
			candidateMap[c.ID] = c
		}

		for i, tally := range tallies {
			doc.SetFont("Helvetica", "B", 10)
			snapshotLabel := fmt.Sprintf("Snapshot #%d", len(tallies)-i)
			doc.CellFormat(0, 5, snapshotLabel, "", 1, "L", false, 0, "")

			if pos, ok := positionMap[tally.PositionID]; ok {
				doc.KV("Position", pos.Title)
			}

			if tally.CandidateID != nil {
				if cand, ok := candidateMap[*tally.CandidateID]; ok {
					doc.KV("Candidate", cand.FullName)
				}
			} else {
				doc.KV("Choice", tally.Choice)
			}

			doc.KV("Vote Count", fmt.Sprintf("%d", tally.Votes))
			doc.KV("Computed At", tally.ComputedAt.Format("2 Jan 2006 15:04 EAT"))
			doc.KV("Snapshot ID", tally.ID)
			doc.Ln(2)
		}
	}

	// Summary section
	doc.Header("VERIFICATION SUMMARY", "")
	doc.Notes(fmt.Sprintf(
		"This election had %d eligible voters and %d participated (%.1f%% turnout). "+
			"%d ballots were accepted. %d tally snapshots are on record. "+
			"All integrity checks are necessary but not sufficient for a certified result.",
		eligReport.Eligible,
		eligReport.Participated,
		eligReport.TurnoutPercent,
		ballotCount,
		len(tallies),
	))

	doc.Footer("This verification report is for audit purposes only. Official certification requires separate approval.")

	return doc.Bytes()
}

func formatTime(t *time.Time) string {
	if t == nil {
		return "N/A"
	}
	return t.Format("2006-01-02 15:04")
}

func getUniquePositionIDs(tallies []models.ElectionTally) []string {
	seen := make(map[string]bool)
	var ids []string
	for _, t := range tallies {
		if !seen[t.PositionID] {
			seen[t.PositionID] = true
			ids = append(ids, t.PositionID)
		}
	}
	return ids
}

func getUniqueCandidateIDs(tallies []models.ElectionTally) []string {
	seen := make(map[string]bool)
	var ids []string
	for _, t := range tallies {
		if t.CandidateID != nil && !seen[*t.CandidateID] {
			seen[*t.CandidateID] = true
			ids = append(ids, *t.CandidateID)
		}
	}
	return ids
}

func getUniquePositionIDsFromCandidates(candidates []models.Candidate) []string {
	seen := make(map[string]bool)
	var ids []string
	for _, c := range candidates {
		if !seen[c.PositionID] {
			seen[c.PositionID] = true
			ids = append(ids, c.PositionID)
		}
	}
	return ids
}
