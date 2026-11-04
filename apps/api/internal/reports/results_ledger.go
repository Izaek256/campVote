package reports

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	"campvote/apps/api/internal/lifecycle"
	"campvote/apps/api/internal/models"
	"campvote/apps/api/internal/pdf"
	"campvote/apps/api/internal/tally"
)

// The campVote UI kit palette. These are the only colours a report may use:
// ink for structure, one green for emphasis, sunken/line for tables. Times are
// rendered in EAT and every count is an aggregate — no report in this package
// carries a voter's identity or an individual ballot.
const (
	kitInkR, kitInkG, kitInkB          = 10, 15, 12    // #0A0F0C
	kitInk2R, kitInk2G, kitInk2B       = 56, 66, 61    // #38423D
	kitInk3R, kitInk3G, kitInk3B       = 91, 103, 97   // #5B6761
	kitLineR, kitLineG, kitLineB       = 211, 217, 214 // #D3D9D6
	kitSunkenR, kitSunkenG, kitSunkenB = 243, 246, 244 // #F3F6F4
	kitPriR, kitPriG, kitPriB          = 18, 120, 63   // #12783F
	kitWhiteR, kitWhiteG, kitWhiteB    = 255, 255, 255 // #FFFFFF
	ledgerMarginX                      = 16.0
	ledgerContentW                     = 210.0 - 2*ledgerMarginX
)

// eatLocation is the zone every human-facing timestamp is rendered in. Uganda
// has no daylight saving, so a fixed +03:00 offset is a correct fallback when
// the tz database is not installed on the host.
var eatLocation = func() *time.Location {
	if loc, err := time.LoadLocation(lifecycle.DisplayTimezone); err == nil {
		return loc
	}
	return time.FixedZone("EAT", 3*60*60)
}()

func eatTime(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.In(eatLocation).Format("2 Jan 2006 15:04") + " EAT"
}

// ResultsLedger renders design option A, the "certified ledger": an ink header
// band, a stat strip, one tally block per position with share bars, a turnout
// table, the administrative audit trail and signature lines.
//
// It draws only what the stored snapshot already publishes. The word "Elected"
// is used only when that snapshot is CERTIFIED; until then the leading
// candidate is labelled "Leading". Nothing in the document identifies a voter.
func ResultsLedger(db *gorm.DB, election models.Election, snapshot models.ResultSnapshot) ([]byte, error) {
	var content tally.Result
	if err := json.Unmarshal([]byte(snapshot.ContentJSON), &content); err != nil {
		return nil, fmt.Errorf("stored tally content could not be read: %w", err)
	}
	certified := snapshot.Status == models.ResultStatusCertified

	audit, err := ledgerAuditTrail(db, election.ID)
	if err != nil {
		return nil, err
	}

	doc := newKitDoc()
	doc.RunningFooter(fmt.Sprintf("%s - election %s - page {nb}", "campVote Electoral Commission", election.ID))

	inkBand(doc,
		election.Title,
		"OFFICIAL RESULTS LEDGER - "+categoryLabel(election.Category)+" - "+election.OrganizingUnit,
		"ELECTION ID "+election.ID+" - "+statusLabel(snapshot.Status, election.Status),
	)

	statStrip(doc, [][2]string{
		{"ELIGIBLE VOTERS", thousands(snapshot.EligibleVoters)},
		{"BALLOTS ACCEPTED", thousands(snapshot.BallotsCast)},
		{"TURNOUT", fmt.Sprintf("%.1f%%", snapshot.TurnoutPercent)},
		{"SPOILED", thousands(content.BallotsSpoiled)},
	})

	doc.KV("Count computed", eatTime(snapshot.ComputedAt))
	if certified {
		doc.KV("Certified", fmt.Sprintf("%s by %s", eatTime(snapshot.CertifiedAt), snapshot.CertifiedByID))
	}
	doc.KV("Content hash (SHA-256)", snapshot.ContentHash)
	if !certified {
		doc.Notes("This count is provisional. The leading candidate in each position is labelled " +
			"\"Leading\" and is not yet an elected officer; the label becomes \"Elected\" once the " +
			"returning officer certifies the snapshot.")
	}

	sectionTitle(doc, "Results by position")
	for _, position := range content.Positions {
		positionBlock(doc, position, certified)
	}

	sectionTitle(doc, "Turnout")
	doc.Table(
		[]string{"MEASURE", "COUNT", "SHARE"},
		[][]string{
			{"Eligible voters (denominator of turnout)", thousands(snapshot.EligibleVoters), "-"},
			{"Ballots accepted", thousands(snapshot.BallotsCast), shareLabel(snapshot.BallotsCast, snapshot.EligibleVoters)},
			{"Ballots spoiled (excluded from every share)", thousands(content.BallotsSpoiled), "-"},
			{"Turnout", "-", fmt.Sprintf("%.1f%%", snapshot.TurnoutPercent)},
		},
		[]float64{ledgerContentW - 90, 45, 45},
		[]string{"L", "R", "R"},
	)
	doc.Notes("Turnout is cast ballots over eligible voters. It is reported separately and is never " +
		"mixed into a candidate's share, which is votes over valid votes for that position.")

	if len(audit) > 0 {
		sectionTitle(doc, "Administrative audit trail")
		rows := make([][]string, 0, len(audit))
		for _, entry := range audit {
			rows = append(rows, []string{eatTime(entry.CreatedAt), entry.Action, truncate(entry.Details, 60)})
		}
		doc.Table(
			[]string{"TIME (EAT)", "ACTION", "REFERENCE"},
			rows,
			[]float64{38, 52, 0},
			[]string{"L", "L", "L"},
		)
		doc.Notes("Administrative actions only. A ballot choice is never recorded against a voter, so " +
			"no line here can be traced back to how anybody voted.")
	}

	signatureLines(doc, []string{"Returning officer", "Election auditor", "Guild representative"})

	doc.Footer("This ledger reproduces the published aggregate count and contains no voter identity " +
		"and no individual ballot.")
	return doc.Bytes()
}

// ledgerAuditTrail loads the most recent administrative actions for an
// election. It is deliberately capped: the ledger is evidence, not a log dump.
func ledgerAuditTrail(db *gorm.DB, electionID string) ([]models.ElectionAuditLog, error) {
	var entries []models.ElectionAuditLog
	if err := db.Where("election_id = ?", electionID).
		Order("created_at DESC").Limit(12).Find(&entries).Error; err != nil {
		return nil, fmt.Errorf("failed to load audit trail: %w", err)
	}
	return entries, nil
}

// newKitDoc returns an A4 portrait document themed with the campVote UI kit.
func newKitDoc() *pdf.Doc {
	doc := pdf.New()
	doc.Accent = [3]int{kitPriR, kitPriG, kitPriB}
	doc.Muted = [3]int{kitInk3R, kitInk3G, kitInk3B}
	doc.SetMargins(ledgerMarginX, 18, ledgerMarginX)
	doc.SetAutoPageBreak(true, 26)
	doc.SetXY(ledgerMarginX, 18)
	return doc
}

// inkBand draws the solid header block that opens every document in the kit.
func inkBand(doc *pdf.Doc, title, kicker, reference string) {
	w, _ := doc.GetPageSize()
	left, _, right, _ := doc.GetMargins()
	bandW := w - left - right
	const bandH = 30.0

	x, y := doc.GetXY()
	doc.SetFillColor(kitInkR, kitInkG, kitInkB)
	doc.Rect(x, y, bandW, bandH, "F")

	doc.SetTextColor(kitWhiteR, kitWhiteG, kitWhiteB)
	doc.SetXY(x+5, y+4)
	doc.SetFont("Helvetica", "B", 16)
	doc.CellFormat(bandW-10, 8, strings.ToUpper(truncate(title, 56)), "", 1, "L", false, 0, "")

	doc.SetXY(x+5, y+13)
	doc.SetFont("Helvetica", "", 9)
	doc.CellFormat(bandW-10, 5, truncate(kicker, 78), "", 1, "L", false, 0, "")

	doc.SetXY(x+5, y+21)
	doc.SetFont("Helvetica", "", 8)
	doc.CellFormat(bandW-10, 5, truncate(reference, 84), "", 1, "L", false, 0, "")

	doc.SetY(y + bandH + 8)
	doc.SetTextColor(kitInkR, kitInkG, kitInkB)
}

// statStrip draws the row of headline figures under the header band.
func statStrip(doc *pdf.Doc, stats [][2]string) {
	if len(stats) == 0 {
		return
	}
	w, _ := doc.GetPageSize()
	left, _, right, _ := doc.GetMargins()
	bandW := w - left - right
	const gap = 3.0
	const boxH = 20.0
	boxW := (bandW - gap*float64(len(stats)-1)) / float64(len(stats))

	x, y := doc.GetXY()
	for i, stat := range stats {
		bx := x + float64(i)*(boxW+gap)
		doc.SetFillColor(kitSunkenR, kitSunkenG, kitSunkenB)
		doc.Rect(bx, y, boxW, boxH, "F")
		doc.SetDrawColor(kitLineR, kitLineG, kitLineB)
		doc.SetLineWidth(0.3)
		doc.Rect(bx, y, boxW, boxH, "D")

		doc.SetXY(bx+3, y+3)
		doc.SetFont("Helvetica", "B", 7)
		doc.SetTextColor(kitInk3R, kitInk3G, kitInk3B)
		doc.CellFormat(boxW-6, 4, stat[0], "", 1, "L", false, 0, "")

		doc.SetXY(bx+3, y+8)
		doc.SetFont("Helvetica", "B", 15)
		doc.SetTextColor(kitInkR, kitInkG, kitInkB)
		doc.CellFormat(boxW-6, 9, stat[1], "", 1, "L", false, 0, "")
	}
	doc.SetY(y + boxH + 8)
	doc.SetTextColor(kitInkR, kitInkG, kitInkB)
}

// sectionTitle writes a small-caps heading with a hairline rule beneath it.
func sectionTitle(doc *pdf.Doc, text string) {
	if doc.GetY() > 250 {
		doc.AddPage()
	}
	doc.Ln(2)
	doc.SetFont("Helvetica", "B", 9)
	doc.SetTextColor(kitInk3R, kitInk3G, kitInk3B)
	doc.CellFormat(0, 6, strings.ToUpper(text), "", 1, "L", false, 0, "")

	doc.SetDrawColor(kitLineR, kitLineG, kitLineB)
	doc.SetLineWidth(0.4)
	left, _, right, _ := doc.GetMargins()
	w, _ := doc.GetPageSize()
	y := doc.GetY()
	doc.Line(left, y, w-right, y)
	doc.Ln(4)
	doc.SetTextColor(kitInkR, kitInkG, kitInkB)
}

// positionBlock renders one contest: its title bar, candidate rows with share
// bars, and the blank/abstain footnote.
func positionBlock(doc *pdf.Doc, position tally.PositionResult, certified bool) {
	if doc.GetY() > 240 {
		doc.AddPage()
	}
	w, _ := doc.GetPageSize()
	left, _, right, _ := doc.GetMargins()
	bandW := w - left - right

	x, y := doc.GetXY()
	doc.SetFillColor(kitSunkenR, kitSunkenG, kitSunkenB)
	doc.Rect(x, y, bandW, 9, "F")

	doc.SetXY(x+3, y+1.5)
	doc.SetFont("Helvetica", "B", 11)
	doc.SetTextColor(kitInkR, kitInkG, kitInkB)
	doc.CellFormat(bandW-6-72, 6, truncate(position.Title, 44), "", 0, "L", false, 0, "")
	doc.SetFont("Helvetica", "", 8)
	doc.SetTextColor(kitInk3R, kitInk3G, kitInk3B)
	doc.CellFormat(72, 6, fmt.Sprintf("%s - %s - %s VALID",
		seatLabel(position.Seats),
		outcomeLabel(position.Outcome),
		thousands(position.ValidVotes)), "", 1, "R", false, 0, "")
	doc.SetY(y + 12)

	const (
		colName   = 74.0
		colVotes  = 18.0
		colShare  = 16.0
		colBar    = 48.0
		colResult = 22.0
	)

	for _, row := range position.Rows {
		_, ry := doc.GetXY()
		doc.SetFont("Helvetica", "", 10)
		doc.SetTextColor(kitInkR, kitInkG, kitInkB)
		doc.CellFormat(colName, 7, truncate(row.Name, 42), "", 0, "L", false, 0, "")
		doc.CellFormat(colVotes, 7, thousands(row.Votes), "", 0, "R", false, 0, "")
		doc.CellFormat(colShare, 7, fmt.Sprintf("%.1f%%", row.SharePercent), "", 0, "R", false, 0, "")

		bx, _ := doc.GetXY()
		doc.SetFillColor(kitLineR, kitLineG, kitLineB)
		doc.Rect(bx+2, ry+2.4, colBar-6, 2.2, "F")
		if row.SharePercent > 0 {
			if row.IsWinner {
				doc.SetFillColor(kitPriR, kitPriG, kitPriB)
			} else {
				doc.SetFillColor(kitInk2R, kitInk2G, kitInk2B)
			}
			doc.Rect(bx+2, ry+2.4, (colBar-6)*row.SharePercent/100, 2.2, "F")
		}
		doc.SetX(bx + colBar)

		doc.SetFont("Helvetica", "B", 8)
		if row.IsWinner {
			doc.SetTextColor(kitPriR, kitPriG, kitPriB)
		} else {
			doc.SetTextColor(kitInk3R, kitInk3G, kitInk3B)
		}
		doc.CellFormat(colResult, 7, rowLabel(row, certified), "", 1, "R", false, 0, "")

		doc.SetDrawColor(kitLineR, kitLineG, kitLineB)
		doc.SetLineWidth(0.2)
		doc.Line(left, doc.GetY(), w-right, doc.GetY())
	}

	doc.Ln(2)
	doc.SetFont("Helvetica", "", 8)
	doc.SetTextColor(kitInk3R, kitInk3G, kitInk3B)
	quorum := "quorum not required"
	if position.QuorumPercent > 0 {
		quorum = fmt.Sprintf("%.0f%% quorum %s", position.QuorumPercent, metWord(position.QuorumMet))
	}
	doc.CellFormat(0, 5, fmt.Sprintf("Blank %s - abstain %s - %s",
		thousands(position.BlankCount), thousands(position.AbstainCount), quorum), "", 1, "L", false, 0, "")
	doc.Ln(5)
	doc.SetTextColor(kitInkR, kitInkG, kitInkB)
}

// signatureLines draws the certifying signature row.
func signatureLines(doc *pdf.Doc, labels []string) {
	if len(labels) == 0 {
		return
	}
	if doc.GetY() > 235 {
		doc.AddPage()
	}
	sectionTitle(doc, "Certification")

	w, _ := doc.GetPageSize()
	left, _, right, _ := doc.GetMargins()
	bandW := w - left - right
	const gap = 6.0
	colW := (bandW - gap*float64(len(labels)-1)) / float64(len(labels))

	x, y := doc.GetXY()
	for i, label := range labels {
		bx := x + float64(i)*(colW+gap)
		doc.SetDrawColor(kitInk3R, kitInk3G, kitInk3B)
		doc.SetLineWidth(0.4)
		doc.Line(bx, y+10, bx+colW, y+10)

		doc.SetXY(bx, y+12)
		doc.SetFont("Helvetica", "B", 8)
		doc.SetTextColor(kitInk3R, kitInk3G, kitInk3B)
		doc.CellFormat(colW, 4, strings.ToUpper(label), "", 1, "L", false, 0, "")

		doc.SetXY(bx, y+16)
		doc.SetFont("Helvetica", "", 8)
		doc.CellFormat(colW, 4, "name / signature / date", "", 1, "L", false, 0, "")
	}
	doc.SetY(y + 24)
	doc.SetTextColor(kitInkR, kitInkG, kitInkB)
}

// rowLabel is the kit's wording rule: only a certified snapshot may call a
// candidate elected.
func rowLabel(row tally.ResultRow, certified bool) string {
	switch {
	case row.IsTie:
		return "TIE"
	case row.IsWinner && certified:
		return "ELECTED"
	case row.IsWinner:
		return "LEADING"
	}
	return ""
}

func outcomeLabel(outcome string) string {
	switch outcome {
	case tally.OutcomeWinner:
		return "DECIDED"
	case tally.OutcomeTieUnresolved:
		return "TIE - UNRESOLVED"
	case tally.OutcomeQuorumNotMet:
		return "QUORUM NOT MET"
	case tally.OutcomeNoVotes:
		return "NO VOTES"
	}
	if outcome == "" {
		return "NOT COUNTED"
	}
	return outcome
}

func statusLabel(snapshotStatus, electionStatus string) string {
	switch snapshotStatus {
	case models.ResultStatusCertified:
		return "CERTIFIED RESULT"
	case models.ResultStatusSuperseded:
		return "SUPERSEDED RESULT"
	case models.ResultStatusProvisional:
		return "PROVISIONAL RESULT"
	}
	return electionStatus
}

func categoryLabel(category string) string {
	if category == "" {
		return "ELECTION"
	}
	return strings.ToUpper(category)
}

func seatLabel(seats int) string {
	if seats <= 1 {
		return "1 SEAT"
	}
	return fmt.Sprintf("%d SEATS", seats)
}

func metWord(met bool) string {
	if met {
		return "MET"
	}
	return "NOT MET"
}

func shareLabel(part, total int) string {
	if total <= 0 {
		return "-"
	}
	return fmt.Sprintf("%.1f%%", float64(part)/float64(total)*100)
}

// thousands formats an integer with comma separators.
func thousands(n int) string {
	digits := strconv.Itoa(n)
	sign := ""
	if strings.HasPrefix(digits, "-") {
		sign = "-"
		digits = digits[1:]
	}
	var out strings.Builder
	for i, c := range digits {
		if i > 0 && (len(digits)-i)%3 == 0 {
			out.WriteByte(',')
		}
		out.WriteRune(c)
	}
	return sign + out.String()
}

// truncate shortens display text so a long title cannot break a fixed column.
func truncate(s string, max int) string {
	s = strings.TrimSpace(s)
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	if max <= 3 {
		return string(runes[:max])
	}
	return string(runes[:max-3]) + "..."
}
