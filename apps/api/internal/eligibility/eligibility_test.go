package eligibility_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"campvote/apps/api/internal/eligibility"
	"campvote/apps/api/internal/models"
)

// testDB uses the pure-Go SQLite driver so the suite runs without a C
// compiler, the same choice the other election tests make.
func testDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(models.Models()...); err != nil {
		t.Fatal(err)
	}
	return db
}

func openElection(t *testing.T, db *gorm.DB, id string, start, end time.Time) *models.Election {
	t.Helper()
	election := &models.Election{
		ID: id, Title: id, Status: models.ElectionStatusOpen,
		StartsAt: start, EndsAt: end,
		AllowedDomains: "@std.must.ac.ug,@must.ac.ug",
		ResultPolicy:   models.ResultPolicyAfterClose,
	}
	if err := db.Create(election).Error; err != nil {
		t.Fatal(err)
	}
	return election
}

func rosterEntry(t *testing.T, db *gorm.DB, electionID, student, email string) *models.ElectionRoster {
	t.Helper()
	entry := &models.ElectionRoster{
		ElectionID: electionID, StudentNumber: student, Email: email, FullName: "Synthetic Student",
		Status: models.RosterStatusEligible, Source: models.RosterSourceCSVImport,
	}
	if err := db.Create(entry).Error; err != nil {
		t.Fatal(err)
	}
	return entry
}

// account is a signed-in user whose address has been verified. Eligibility
// reads only these two fields, so no row is needed.
func account(email string) *models.User {
	now := time.Now().UTC()
	return &models.User{Email: email, EmailVerifiedAt: &now}
}

func TestUnverifiedEmailCannotClaimARosterEntry(t *testing.T) {
	db := testDB(t)
	now := time.Now().UTC()
	openElection(t, db, "e-a", now.Add(-time.Hour), now.Add(time.Hour))
	rosterEntry(t, db, "e-a", "22/U/0001", "voter@std.must.ac.ug")

	decision, err := eligibility.Check(db, "e-a", &models.User{Email: "voter@std.must.ac.ug"}, now)
	if err != nil {
		t.Fatal(err)
	}
	if decision.Eligible {
		t.Fatal("an unverified address must not claim a roster entry")
	}
	if !strings.Contains(decision.Reason, "Confirm your email") {
		t.Errorf("reason = %q, want it to ask for verification", decision.Reason)
	}
}

func TestDomainAloneDoesNotGrantEligibility(t *testing.T) {
	db := testDB(t)
	now := time.Now().UTC()
	openElection(t, db, "e-a", now.Add(-time.Hour), now.Add(time.Hour))
	rosterEntry(t, db, "e-a", "22/U/0001", "listed@std.must.ac.ug")

	decision, err := eligibility.Check(db, "e-a", account("valid-domain-not-listed@std.must.ac.ug"), now)
	if err != nil {
		t.Fatal(err)
	}
	if decision.Eligible {
		t.Fatal("a valid institutional domain must not grant eligibility without a roster entry")
	}
	if !strings.Contains(decision.Reason, "roster") {
		t.Errorf("reason = %q, want it to point at the roster", decision.Reason)
	}

	listed, err := eligibility.Check(db, "e-a", account("LISTED@std.must.ac.ug"), now)
	if err != nil {
		t.Fatal(err)
	}
	if !listed.Eligible {
		t.Fatalf("a listed eligible voter was refused: %s", listed.Reason)
	}
}

func TestEligibilityIsPerElection(t *testing.T) {
	db := testDB(t)
	now := time.Now().UTC()
	openElection(t, db, "e-a", now.Add(-time.Hour), now.Add(time.Hour))
	openElection(t, db, "e-b", now.Add(-time.Hour), now.Add(time.Hour))
	rosterEntry(t, db, "e-a", "22/U/0001", "voter@std.must.ac.ug")

	inA, err := eligibility.Check(db, "e-a", account("voter@std.must.ac.ug"), now)
	if err != nil {
		t.Fatal(err)
	}
	inB, err := eligibility.Check(db, "e-b", account("voter@std.must.ac.ug"), now)
	if err != nil {
		t.Fatal(err)
	}
	if !inA.Eligible || inB.Eligible {
		t.Fatalf("eligibility leaked across elections: A=%v B=%v", inA.Eligible, inB.Eligible)
	}
}

func TestClockAndStatusGate(t *testing.T) {
	db := testDB(t)
	now := time.Now().UTC()
	openElection(t, db, "e-a", now.Add(time.Hour), now.Add(2*time.Hour))
	entry := rosterEntry(t, db, "e-a", "22/U/0001", "voter@std.must.ac.ug")

	before, _ := eligibility.Check(db, "e-a", account(entry.Email), now)
	if before.Eligible {
		t.Error("a voter must be refused before the election opens")
	}

	if err := db.Model(&models.Election{}).Where("id = ?", "e-a").
		Update("ends_at", now.Add(-time.Minute)).Error; err != nil {
		t.Fatal(err)
	}
	after, _ := eligibility.Check(db, "e-a", account(entry.Email), now)
	if after.Eligible {
		t.Error("a voter must be refused after the election closes")
	}
}

func TestFreezeRefusesNewEntriesButAllowsRevocation(t *testing.T) {
	db := testDB(t)
	now := time.Now().UTC()
	election := openElection(t, db, "e-a", now.Add(-time.Hour), now.Add(time.Hour))
	entry := rosterEntry(t, db, "e-a", "22/U/0001", "voter@std.must.ac.ug")

	if _, err := eligibility.Freeze(db, election.ID, "officer", now); err != nil {
		t.Fatal(err)
	}
	var frozen models.Election
	if err := db.Where("id = ?", election.ID).First(&frozen).Error; err != nil {
		t.Fatal(err)
	}
	if frozen.EligibilityFrozenAt == nil || frozen.EligibilityFrozenCount != 1 {
		t.Fatalf("freeze did not record the snapshot: %+v", frozen)
	}

	csv := "student_number,email,full_name\n22/U/0002,late@std.must.ac.ug,Late Arrival\n"
	if _, _, err := eligibility.Import(db, election.ID, "officer", models.RosterSourceCSVImport,
		"late.csv", strings.NewReader(csv), now); err == nil {
		t.Fatal("a new entry after the freeze should be refused")
	}

	if _, err := eligibility.SetStatus(db, election.ID, entry.ID, "officer",
		models.RosterStatusSuspended, "suspended by registrar"); err != nil {
		t.Fatal(err)
	}
	decision, err := eligibility.Check(db, election.ID, account(entry.Email), now)
	if err != nil {
		t.Fatal(err)
	}
	if decision.Eligible {
		t.Fatal("a revoked voter must be refused")
	}

	var audit models.ElectionAuditLog
	if err := db.Where("election_id = ? AND action = ?", election.ID, models.AuditActionRosterRevoked).
		First(&audit).Error; err != nil {
		t.Fatal("revocation should be audited")
	}

	// The import path must not restore eligibility after the freeze; restoring
	// is an explicit override with its own audit action.
	restore := "student_number,email,full_name,status\n22/U/0001,voter@std.must.ac.ug,Synthetic Student,ELIGIBLE\n"
	plan, err := eligibility.Preview(db, election.ID, strings.NewReader(restore), now)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Errors) == 0 {
		t.Fatal("import must not restore eligibility after the freeze")
	}
	if _, err := eligibility.SetStatus(db, election.ID, entry.ID, "officer",
		models.RosterStatusEligible, "registrar correction"); err != nil {
		t.Fatal(err)
	}
	var override models.ElectionAuditLog
	if err := db.Where("election_id = ? AND action = ?", election.ID, models.AuditActionEligibilityOverride).
		First(&override).Error; err != nil {
		t.Fatal("an override should be audited as such")
	}
}

func TestFreezeRefusesAfterCloseAndSetStatusRefusesAfterClose(t *testing.T) {
	db := testDB(t)
	now := time.Now().UTC()
	election := openElection(t, db, "e-a", now.Add(-2*time.Hour), now.Add(-time.Hour))
	entry := rosterEntry(t, db, "e-a", "22/U/0001", "voter@std.must.ac.ug")

	if _, err := eligibility.Freeze(db, election.ID, "officer", now); err == nil {
		t.Fatal("freeze must require an OPEN election")
	}
	if _, err := eligibility.SetStatus(db, election.ID, entry.ID, "officer",
		models.RosterStatusSuspended, "late revocation"); !errors.Is(err, eligibility.ErrElectionClosed) {
		t.Fatalf("post-close status change error = %v, want ErrElectionClosed", err)
	}
}

func TestImportPreviewRejectsBadAndDuplicateRowsWithoutWriting(t *testing.T) {
	db := testDB(t)
	now := time.Now().UTC()
	openElection(t, db, "e-a", now.Add(-time.Hour), now.Add(time.Hour))

	bad := strings.Join([]string{
		"student_number,email,full_name,study_year,status",
		"22/U/0001,one@std.must.ac.ug,One Student,3,ELIGIBLE",
		"22/U/0002,not-an-email,Two Student,2,ELIGIBLE",
		"22/U/0001,two@std.must.ac.ug,Duplicate Student,2,ELIGIBLE",
		"22/U/0003,one@std.must.ac.ug,Duplicate Identity,2,ELIGIBLE",
		"22/U/0004,four@std.must.ac.ug,Four Student,99,ELIGIBLE",
	}, "\n")

	plan, err := eligibility.Preview(db, "e-a", strings.NewReader(bad), now)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Errors) < 4 {
		t.Fatalf("expected at least 4 rejected rows, got %d: %+v", len(plan.Errors), plan.Errors)
	}
	if plan.Creates != 1 {
		t.Errorf("creates = %d, want 1 valid row", plan.Creates)
	}
	var count int64
	db.Model(&models.ElectionRoster{}).Where("election_id = ?", "e-a").Count(&count)
	if count != 0 {
		t.Errorf("preview wrote %d rows", count)
	}

	if _, _, err := eligibility.Import(db, "e-a", "officer", models.RosterSourceCSVImport,
		"bad.csv", strings.NewReader(bad), now); err == nil {
		t.Fatal("a file with bad rows must be refused as a whole")
	}
	db.Model(&models.ElectionRoster{}).Where("election_id = ?", "e-a").Count(&count)
	if count != 0 {
		t.Errorf("a refused import wrote %d rows", count)
	}
}

func TestImportCommitRecordsProvenance(t *testing.T) {
	db := testDB(t)
	now := time.Now().UTC()
	openElection(t, db, "e-a", now.Add(-time.Hour), now.Add(time.Hour))

	csv := "student_number,reg_number,email,full_name,faculty,department,programme,study_year,status\n" +
		"22/U/0001,MMU/22/1,one@std.must.ac.ug,One Student,Science,Computer Science,BSc CS,3,ELIGIBLE\n"
	batch, plan, err := eligibility.Import(db, "e-a", "officer-1", models.RosterSourceCSVImport,
		"roll.csv", strings.NewReader(csv), now)
	if err != nil {
		t.Fatal(err)
	}
	if batch == nil || plan == nil || batch.Created != 1 || batch.Total != 1 {
		t.Fatalf("unexpected batch: %+v plan: %+v", batch, plan)
	}
	var entry models.ElectionRoster
	if err := db.Where("election_id = ?", "e-a").First(&entry).Error; err != nil {
		t.Fatal(err)
	}
	if entry.ImportBatchID != batch.ID || entry.ImportedByID != "officer-1" || entry.Source != models.RosterSourceCSVImport {
		t.Errorf("roster provenance not recorded: %+v", entry)
	}
	var audit models.ElectionAuditLog
	if err := db.Where("election_id = ? AND action = ?", "e-a", models.AuditActionRosterImported).
		First(&audit).Error; err != nil {
		t.Fatal("import should be audited")
	}
}

func TestImportRejectsEmailAlreadyOwnedByAnotherStudent(t *testing.T) {
	db := testDB(t)
	now := time.Now().UTC()
	openElection(t, db, "e-a", now.Add(-time.Hour), now.Add(time.Hour))
	rosterEntry(t, db, "e-a", "22/U/0001", "one@std.must.ac.ug")

	csv := "student_number,email,full_name\n22/U/0002,one@std.must.ac.ug,Another Person\n"
	plan, err := eligibility.Preview(db, "e-a", strings.NewReader(csv), now)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Errors) != 1 || !strings.Contains(plan.Errors[0].Reason, "duplicate identity") {
		t.Fatalf("expected a duplicate-identity refusal, got %+v", plan.Errors)
	}
}

func TestReportHasCountsOnly(t *testing.T) {
	db := testDB(t)
	now := time.Now().UTC()
	openElection(t, db, "e-a", now.Add(-time.Hour), now.Add(time.Hour))
	rosterEntry(t, db, "e-a", "22/U/0001", "secret@std.must.ac.ug")
	rosterEntry(t, db, "e-a", "22/U/0002", "revoked@std.must.ac.ug")
	if err := db.Model(&models.ElectionRoster{}).Where("email = ?", "revoked@std.must.ac.ug").
		Update("status", models.RosterStatusSuspended).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.VoterParticipation{ElectionID: "e-a", UserID: "user-1"}).Error; err != nil {
		t.Fatal(err)
	}

	report, err := eligibility.BuildReport(db, "e-a")
	if err != nil {
		t.Fatal(err)
	}
	if report.Total != 2 || report.Eligible != 1 || report.Participated != 1 || report.TurnoutPercent != 100 {
		t.Errorf("unexpected report: %+v", report)
	}
	encoded, _ := json.Marshal(report)
	for _, forbidden := range []string{"secret@std.must.ac.ug", "22/U/0001", "user-1"} {
		if strings.Contains(string(encoded), forbidden) {
			t.Errorf("report leaked %q: %s", forbidden, encoded)
		}
	}
}

// A ballot already accepted stands when the voter is revoked afterwards. The
// revocation blocks any further ballot; it does not delete the vote, the
// participation record or the tally.
func TestRevocationAfterVotingLeavesTheBallotStanding(t *testing.T) {
	db := testDB(t)
	now := time.Now().UTC()
	openElection(t, db, "e-a", now.Add(-time.Hour), now.Add(time.Hour))
	entry := rosterEntry(t, db, "e-a", "22/U/0001", "voter@std.must.ac.ug")

	if err := db.Create(&models.VoterParticipation{ElectionID: "e-a", UserID: "user-1"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.AnonymousBallot{ElectionID: "e-a"}).Error; err != nil {
		t.Fatal(err)
	}

	if _, err := eligibility.SetStatus(db, "e-a", entry.ID, "officer-1",
		models.RosterStatusSuspended, "left the university"); err != nil {
		t.Fatal(err)
	}

	decision, err := eligibility.Check(db, "e-a", account("voter@std.must.ac.ug"), time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if decision.Eligible {
		t.Fatal("a revoked voter must not start another ballot")
	}

	var ballots int64
	if err := db.Model(&models.AnonymousBallot{}).Where("election_id = ?", "e-a").Count(&ballots).Error; err != nil {
		t.Fatal(err)
	}
	if ballots != 1 {
		t.Fatalf("the accepted ballot disappeared: %d remain", ballots)
	}
	var participation models.VoterParticipation
	if err := db.Where("election_id = ? AND user_id = ?", "e-a", "user-1").First(&participation).Error; err != nil {
		t.Fatalf("the participation record disappeared: %v", err)
	}
}
