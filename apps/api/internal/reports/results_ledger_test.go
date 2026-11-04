package reports

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"

	"campvote/apps/api/internal/models"
	"campvote/apps/api/internal/tally"
)

// resultsFixture creates a closed election with one decided position, a stored
// snapshot in the given status, one audit entry and one position. It returns
// the shared in-memory database along with the records.
func resultsFixture(t *testing.T, snapshotStatus string) (*gorm.DB, models.Election, models.ResultSnapshot) {
	t.Helper()
	db := setupTestDB(t)
	now := time.Now().UTC()

	election := models.Election{
		Title:          "Guild President 2026",
		Category:       models.ElectionCategoryGuild,
		OrganizingUnit: "MMU Electoral Commission",
		Status:         models.ElectionStatusClosed,
		StartsAt:       now.Add(-48 * time.Hour),
		EndsAt:         now.Add(-24 * time.Hour),
		ResultPolicy:   models.ResultPolicyAfterClose,
	}
	if err := db.Create(&election).Error; err != nil {
		t.Fatalf("create election: %v", err)
	}

	result := tally.Result{
		ElectionID: election.ID,
		Turnout:    tally.Turnout{EligibleVoters: 100, BallotsCast: 60, Percent: 60, Known: true},
		Positions: []tally.PositionResult{{
			PositionID: "pos-1",
			Title:      "President",
			Seats:      1,
			Outcome:    tally.OutcomeWinner,
			ValidVotes: 60,
			Rows: []tally.ResultRow{
				{CandidateID: "cand-1", Name: "Amina Nakato", Votes: 40, SharePercent: 66.67, IsWinner: true},
				{CandidateID: "cand-2", Name: "Bosco Okello", Votes: 20, SharePercent: 33.33},
			},
		}},
	}
	payload, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}

	snapshot := models.ResultSnapshot{
		ElectionID:     election.ID,
		Version:        1,
		Status:         snapshotStatus,
		ContentHash:    tally.Hash(result),
		ContentJSON:    string(payload),
		EligibleVoters: 100,
		BallotsCast:    60,
		TurnoutPercent: 60,
		ComputedAt:     now.Add(-23 * time.Hour),
	}
	if snapshotStatus == models.ResultStatusCertified {
		snapshot.CertifiedByID = "officer-1"
		snapshot.CertifiedAt = now.Add(-22 * time.Hour)
	}
	if err := db.Create(&snapshot).Error; err != nil {
		t.Fatalf("create snapshot: %v", err)
	}

	// The ledger also prints the audit trail and the info sheet the approved
	// candidates, so seed one of each.
	if err := db.Create(&models.ElectionAuditLog{
		ElectionID: election.ID,
		ActorID:    "officer-1",
		Action:     models.AuditActionTallyRun,
		CreatedAt:  now.Add(-23 * time.Hour),
	}).Error; err != nil {
		t.Fatalf("create audit log: %v", err)
	}
	if err := db.Create(&models.Position{
		ElectionID: models.Ptr(election.ID),
		Title:      "President",
	}).Error; err != nil {
		t.Fatalf("create position: %v", err)
	}

	return db, election, snapshot
}

func TestResultsLedgerRenders(t *testing.T) {
	for _, status := range []string{models.ResultStatusProvisional, models.ResultStatusCertified} {
		t.Run(status, func(t *testing.T) {
			db, election, snapshot := resultsFixture(t, status)
			body, err := ResultsLedger(db, election, snapshot)
			if err != nil {
				t.Fatalf("ResultsLedger: %v", err)
			}
			if len(body) < 5 || string(body[:5]) != "%PDF-" {
				t.Fatalf("ResultsLedger did not return a valid PDF")
			}
		})
	}
}

// TestRowLabelOnlyElectsWhenCertified pins the kit's wording rule: a leading
// candidate is only "Elected" once the returning officer has certified.
func TestRowLabelOnlyElectsWhenCertified(t *testing.T) {
	winner := tally.ResultRow{Name: "A", Votes: 10, IsWinner: true}
	tied := tally.ResultRow{Name: "B", Votes: 10, IsTie: true}
	runnerUp := tally.ResultRow{Name: "C", Votes: 4}

	cases := []struct {
		name      string
		row       tally.ResultRow
		certified bool
		want      string
	}{
		{"winner before certification leads", winner, false, "LEADING"},
		{"winner after certification is elected", winner, true, "ELECTED"},
		{"tie never elects", tied, true, "TIE"},
		{"runner-up has no label", runnerUp, true, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := rowLabel(tc.row, tc.certified); got != tc.want {
				t.Errorf("rowLabel = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestElectionInfoSheetRenders(t *testing.T) {
	db, election, _ := resultsFixture(t, models.ResultStatusCertified)
	if err := db.Create(&models.Candidate{
		ElectionID:     election.ID,
		PositionID:     "pos-1",
		FullName:       "Amina Nakato",
		Slogan:         "Books before ballots",
		ApprovalStatus: models.CandidateStatusApproved,
	}).Error; err != nil {
		t.Fatalf("create candidate: %v", err)
	}

	body, err := ElectionInfoSheet(db, election)
	if err != nil {
		t.Fatalf("ElectionInfoSheet: %v", err)
	}
	if len(body) < 5 || string(body[:5]) != "%PDF-" {
		t.Fatalf("ElectionInfoSheet did not return a valid PDF")
	}
}

func TestEligibilitySummaryStatesTheOpenCase(t *testing.T) {
	if got := eligibilitySummary(models.Election{}); got != "All registered students" {
		t.Errorf("empty rule = %q, want the open case stated plainly", got)
	}
	scoped := models.Election{AllowedDomain: "@std.must.ac.ug", AllowedYears: "2024,2025", AllowedCourses: "bse"}
	got := eligibilitySummary(scoped)
	for _, want := range []string{"std.must.ac.ug", "2024,2025", "BSE"} {
		if !strings.Contains(got, want) {
			t.Errorf("eligibilitySummary = %q, missing %q", got, want)
		}
	}
}

func TestThousands(t *testing.T) {
	cases := map[int]string{0: "0", 7: "7", 999: "999", 1000: "1,000", 1234567: "1,234,567"}
	for in, want := range cases {
		if got := thousands(in); got != want {
			t.Errorf("thousands(%d) = %q, want %q", in, got, want)
		}
	}
}
