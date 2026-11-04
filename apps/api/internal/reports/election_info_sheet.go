package reports

import (
	"fmt"
	"strings"

	"gorm.io/gorm"

	"campvote/apps/api/internal/models"
	"campvote/apps/api/internal/pdf"
)

// ElectionInfoSheet renders the one-page notice for boards and emails: key
// facts, the timeline, the approved candidates, how to vote, the secrecy
// notice and a QR slot.
//
// It carries only configuration that is already public — never a ballot choice
// and never a voter.
func ElectionInfoSheet(db *gorm.DB, election models.Election) ([]byte, error) {
	positions, err := infoSheetPositions(db, election.ID)
	if err != nil {
		return nil, err
	}
	candidates, err := infoSheetCandidates(db, election.ID)
	if err != nil {
		return nil, err
	}
	byPosition := make(map[string][]models.Candidate)
	for _, candidate := range candidates {
		byPosition[candidate.PositionID] = append(byPosition[candidate.PositionID], candidate)
	}

	doc := newKitDoc()
	doc.RunningFooter(fmt.Sprintf("campVote Electoral Commission - election %s - page {nb}", election.ID))

	inkBand(doc,
		election.Title,
		"ELECTION INFORMATION - "+categoryLabel(election.Category)+" - "+election.OrganizingUnit,
		"ELECTION ID "+election.ID+" - STATUS "+election.Status,
	)

	if strings.TrimSpace(election.Description) != "" {
		doc.SetFont("Helvetica", "", 10)
		doc.SetTextColor(kitInk2R, kitInk2G, kitInk2B)
		doc.MultiCell(0, 5, truncate(election.Description, 420), "", "L", false)
		doc.Ln(4)
		doc.SetTextColor(kitInkR, kitInkG, kitInkB)
	}

	statStrip(doc, [][2]string{
		{"OPENS", eatTime(election.StartsAt)},
		{"CLOSES", eatTime(election.EndsAt)},
		{"POSITIONS", fmt.Sprintf("%d", len(positions))},
		{"APPROVED", fmt.Sprintf("%d", len(candidates))},
	})

	doc.TwoColumnKV("Category", categoryLabel(election.Category), "Organizing unit", valueOr(election.OrganizingUnit, "-"))
	doc.TwoColumnKV("Who may vote", eligibilitySummary(election), "Results published", resultPolicyLabel(election.ResultPolicy))

	sectionTitle(doc, "Timeline")
	doc.Table(
		[]string{"STAGE", "WHEN", "WHAT HAPPENS"},
		[][]string{
			{"Nominations", eatTime(election.StartsAt), "Applications and manifestos are read before polls open."},
			{"Voting", eatTime(election.StartsAt) + " to " + eatTime(election.EndsAt), "One ballot per eligible voter, one choice per position."},
			{"Results", eatTime(election.EndsAt), "The count is published and certified by the returning officer."},
		},
		[]float64{30, 60, 0},
		[]string{"L", "L", "L"},
	)

	sectionTitle(doc, "Approved candidates")
	if len(positions) == 0 {
		doc.Notes("No positions have been published for this election yet.")
	} else {
		for _, position := range positions {
			doc.SetFont("Helvetica", "B", 10)
			doc.SetTextColor(kitInkR, kitInkG, kitInkB)
			doc.CellFormat(0, 6, truncate(position.Title, 60), "", 1, "L", false, 0, "")
			rows := make([][]string, 0, len(byPosition[position.ID]))
			for _, candidate := range byPosition[position.ID] {
				rows = append(rows, []string{
					truncate(candidate.FullName, 34),
					ballotPositionLabel(candidate.BallotNumber),
					truncate(valueOr(candidate.Slogan, candidate.Manifesto), 62),
				})
			}
			if len(rows) == 0 {
				doc.SetFont("Helvetica", "", 9)
				doc.SetTextColor(kitInk3R, kitInk3G, kitInk3B)
				doc.CellFormat(0, 5, "No approved candidates for this position.", "", 1, "L", false, 0, "")
				doc.SetTextColor(kitInkR, kitInkG, kitInkB)
				doc.Ln(3)
				continue
			}
			doc.Table(
				[]string{"CANDIDATE", "ON BALLOT", "SLOGAN / MANIFESTO"},
				rows,
				[]float64{50, 24, 0},
				[]string{"L", "L", "L"},
			)
			doc.Ln(4)
		}
	}

	sectionTitle(doc, "How to vote")
	howToVote(doc)

	sectionTitle(doc, "Secrecy notice")
	doc.Notes("Your ballot is secret. The system never stores your name next to your choices, so nobody " +
		"can tell how you voted — not the candidates, not the officers and not the administrators. " +
		"Only whether you have voted is recorded, so a second ballot can be refused.")

	qrSlot(doc, election.ID)

	doc.Footer("Check the official notice board or the campVote web app for any change to these times.")
	return doc.Bytes()
}

// howToVote writes the numbered voting steps.
func howToVote(doc *pdf.Doc) {
	steps := []string{
		"Sign in with your campus account and open the election from your dashboard.",
		"Read each candidate's manifesto before you decide.",
		"Choose one candidate for every position you want to vote in; leave the rest blank if you prefer.",
		"Submit your ballot. It is accepted once — you cannot vote twice and you cannot edit a submitted ballot.",
		"You will get a confirmation. A receipt proves your ballot was accepted without revealing your choice.",
	}
	for i, step := range steps {
		doc.SetFont("Helvetica", "B", 9)
		doc.SetTextColor(kitPriR, kitPriG, kitPriB)
		doc.CellFormat(8, 5, fmt.Sprintf("%d.", i+1), "", 0, "L", false, 0, "")
		doc.SetFont("Helvetica", "", 9)
		doc.SetTextColor(kitInkR, kitInkG, kitInkB)
		doc.MultiCell(0, 5, step, "", "L", false)
	}
	doc.Ln(4)
}

// qrSlot draws the print-safe placeholder for the ballot QR code. The code
// itself is added by the web app; the sheet only reserves the space so a
// printed notice always has it in the same corner.
func qrSlot(doc *pdf.Doc, electionID string) {
	const size = 30.0
	left, _, _, _ := doc.GetMargins()
	x, y := doc.GetXY()
	doc.SetDrawColor(kitInkR, kitInkG, kitInkB)
	doc.SetLineWidth(0.4)
	doc.Rect(x, y, size, size, "D")
	doc.SetFont("Helvetica", "", 7)
	doc.SetTextColor(kitInk3R, kitInk3G, kitInk3B)
	doc.SetXY(x+2, y+size/2-4)
	doc.MultiCell(size-4, 3, "scan to open the ballot", "", "C", false)

	doc.SetXY(x+size+5, y+6)
	doc.SetFont("Helvetica", "B", 9)
	doc.SetTextColor(kitInkR, kitInkG, kitInkB)
	doc.CellFormat(0, 5, "Open the ballot", "", 1, "L", false, 0, "")
	doc.SetX(x + size + 5)
	doc.SetFont("Helvetica", "", 8)
	doc.SetTextColor(kitInk3R, kitInk3G, kitInk3B)
	doc.MultiCell(0, 4, fmt.Sprintf("Scan the code, or sign in and search for election %s. "+
		"Do not share the code with anybody outside the eligible register.", electionID), "", "L", false)
	doc.SetY(y+size+6)
	doc.SetTextColor(kitInkR, kitInkG, kitInkB)
	_ = left
}

func infoSheetPositions(db *gorm.DB, electionID string) ([]models.Position, error) {
	var positions []models.Position
	if err := db.Where("election_id = ?", electionID).
		Order("display_order ASC, title ASC").Find(&positions).Error; err != nil {
		return nil, fmt.Errorf("failed to load positions: %w", err)
	}
	return positions, nil
}

func infoSheetCandidates(db *gorm.DB, electionID string) ([]models.Candidate, error) {
	var candidates []models.Candidate
	if err := db.Where("election_id = ? AND approval_status = ?", electionID, models.CandidateStatusApproved).
		Order("position_id, ballot_number, full_name").Find(&candidates).Error; err != nil {
		return nil, fmt.Errorf("failed to load approved candidates: %w", err)
	}
	return candidates, nil
}

// eligibilitySummary describes an election's voter rule in one line. An empty
// rule means every student, which is stated plainly rather than left blank.
func eligibilitySummary(election models.Election) string {
	parts := make([]string, 0, 3)
	if domain := strings.TrimSpace(election.AllowedDomain); domain != "" {
		parts = append(parts, strings.ToLower(strings.TrimPrefix(domain, "@"))+" accounts")
	}
	if years := strings.TrimSpace(election.AllowedYears); years != "" {
		parts = append(parts, "enrolment year "+years)
	}
	if courses := strings.TrimSpace(election.AllowedCourses); courses != "" {
		parts = append(parts, strings.ToUpper(courses)+" students")
	}
	if len(parts) == 0 {
		return "All registered students"
	}
	return strings.Join(parts, ", ")
}

func resultPolicyLabel(policy string) string {
	switch policy {
	case models.ResultPolicyLive:
		return "Live during voting"
	case models.ResultPolicyCertifiedOnly:
		return "After certification"
	case models.ResultPolicyAfterClose:
		return "After polls close"
	}
	if policy == "" {
		return "After polls close"
	}
	return policy
}

func ballotPositionLabel(number int) string {
	if number <= 0 {
		return "-"
	}
	return fmt.Sprintf("No. %d", number)
}

func valueOr(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
