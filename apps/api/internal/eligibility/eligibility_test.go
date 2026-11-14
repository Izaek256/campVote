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

// openElection creates an OPEN election that allows the std.mast.ac.ug domain
// (registration-number format). No roster is required under the new rules.
func openElection(t *testing.T, db *gorm.DB, id string, start, end time.Time) *models.Election {
	t.Helper()
	election := &models.Election{
		ID: id, Title: id, Status: models.ElectionStatusOpen,
		StartsAt:      start,
		EndsAt:        end,
		AllowedDomain: "std.mast.ac.ug",
		ResultPolicy:  models.ResultPolicyAfterClose,
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

// account returns a verified user with a registration-number email and a
// complete voting profile.
func account(email string) *models.User {
	now := time.Now().UTC()
	return &models.User{Email: email, EmailVerifiedAt: &now,
		Age: models.Ptr(20), Sex: models.SexMale, ResidenceAreaID: models.Ptr("area-1")}
}

// studentEmail returns a valid std.mast.ac.ug registration-number email.
func studentEmail(year, course, n string) string {
	return year + course + n + "@std.mast.ac.ug"
}

func TestUnverifiedEmailIsRefused(t *testing.T) {
	db := testDB(t)
	now := time.Now().UTC()
	openElection(t, db, "e-a", now.Add(-time.Hour), now.Add(time.Hour))

	decision, err := eligibility.Check(db, "e-a", &models.User{
		Email: studentEmail("2022", "bse", "001"),
		// no EmailVerifiedAt
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	if decision.Eligible {
		t.Fatal("an unverified address must not be eligible")
	}
	if !strings.Contains(decision.Reason, "Confirm your email") {
		t.Errorf("reason = %q, want it to ask for verification", decision.Reason)
	}
}

func TestRuleMatchingStudentIsEligibleWithoutRosterEntry(t *testing.T) {
	db := testDB(t)
	now := time.Now().UTC()
	openElection(t, db, "e-a", now.Add(-time.Hour), now.Add(time.Hour))

	// No roster entry created — eligible by rule alone.
	decision, err := eligibility.Check(db, "e-a", account(studentEmail("2022", "bse", "001")), now)
	if err != nil {
		t.Fatal(err)
	}
	if !decision.Eligible {
		t.Fatalf("a rule-matching student should be eligible without a roster entry: %s", decision.Reason)
	}
}

func TestVotingNeedsACompleteProfile(t *testing.T) {
	db := testDB(t)
	now := time.Now().UTC()
	openElection(t, db, "e-profile", now.Add(-time.Hour), now.Add(time.Hour))

	user := account(studentEmail("2022", "bse", "099"))
	user.Age = nil
	decision, err := eligibility.Check(db, "e-profile", user, now)
	if err != nil {
		t.Fatal(err)
	}
	if decision.Eligible || !strings.Contains(decision.Reason, "profile") {
		t.Fatalf("an incomplete profile was allowed: eligible=%v reason=%q", decision.Eligible, decision.Reason)
	}

	// The roster check used by candidate applications does not need it.
	roster, err := eligibility.CheckRoster(db, &models.Election{ID: "e-profile", AllowedDomain: "std.mast.ac.ug"}, user)
	if err != nil {
		t.Fatal(err)
	}
	if !roster.Eligible {
		t.Fatalf("CheckRoster refused the same account: %s", roster.Reason)
	}
}

func TestRestrictedElectionNeedsARosterEntry(t *testing.T) {
	db := testDB(t)
	now := time.Now().UTC()
	election := openElection(t, db, "e-restricted", now.Add(-time.Hour), now.Add(time.Hour))
	if err := db.Model(election).Update("require_roster", true).Error; err != nil {
		t.Fatal(err)
	}
	election.RequireRoster = true

	studentAddr := studentEmail("2022", "bse", "001")
	decision, err := eligibility.Check(db, election.ID, account(studentAddr), now)
	if err != nil {
		t.Fatal(err)
	}
	if decision.Eligible {
		t.Fatal("a student who is not on the restricted list must not be eligible")
	}

	rosterEntry(t, db, election.ID, "22/U/0001", studentAddr)
	decision, err = eligibility.Check(db, election.ID, account(studentAddr), now)
	if err != nil {
		t.Fatal(err)
	}
	if !decision.Eligible {
		t.Fatalf("a listed student must be eligible: %s", decision.Reason)
	}
}

func TestRestrictedRosterCanChangeUntilClose(t *testing.T) {
	db := testDB(t)
	now := time.Now().UTC()
	election := openElection(t, db, "e-restricted", now.Add(-time.Hour), now.Add(time.Hour))
	if err := db.Model(election).Update("require_roster", true).Error; err != nil {
		t.Fatal(err)
	}
	election.RequireRoster = true
	if _, err := eligibility.Freeze(db, election.ID, "officer", now); err != nil {
		t.Fatal(err)
	}

	// The freeze keeps a rule-based electorate fixed. A restricted election is
	// defined by its list, so the officer must be able to add names while the
	// poll is open.
	csv := "student_number,email,full_name\n22/U/0002,late@std.mast.ac.ug,Late Arrival\n"
	if _, _, err := eligibility.Import(db, election.ID, "officer", models.RosterSourceCSVImport,
		"late.csv", strings.NewReader(csv), now); err != nil {
		t.Fatalf("adding a listed voter while open: %v", err)
	}

	if _, _, err := eligibility.Import(db, election.ID, "officer", models.RosterSourceCSVImport,
		"late.csv", strings.NewReader(csv), election.EndsAt.Add(time.Minute)); !errors.Is(err, eligibility.ErrElectionClosed) {
		t.Fatalf("post-close import error = %v, want ErrElectionClosed", err)
	}
}

func TestWrongDomainEmailIsRefused(t *testing.T) {
	db := testDB(t)
	now := time.Now().UTC()
	openElection(t, db, "e-a", now.Add(-time.Hour), now.Add(time.Hour))

	// Non-registration-number email: not parseable by ParseEmail.
	decision, err := eligibility.Check(db, "e-a", account("staff@must.ac.ug"), now)
	if err != nil {
		t.Fatal(err)
	}
	if decision.Eligible {
		t.Fatal("a non-student email must not be eligible in a student election")
	}
}

func TestExplicitRosterExclusionOverridesRule(t *testing.T) {
	db := testDB(t)
	now := time.Now().UTC()
	election := openElection(t, db, "e-a", now.Add(-time.Hour), now.Add(time.Hour))

	email := studentEmail("2022", "bse", "001")
	// By rule: eligible. Add a SUSPENDED roster entry to deny.
	if err := db.Create(&models.ElectionRoster{
		ElectionID:    election.ID,
		StudentNumber: "22/U/0001",
		Email:         email,
		FullName:      "Synthetic Student",
		Status:        models.RosterStatusSuspended,
		Source:        models.RosterSourceCSVImport,
	}).Error; err != nil {
		t.Fatal(err)
	}

	decision, err := eligibility.Check(db, "e-a", account(email), now)
	if err != nil {
		t.Fatal(err)
	}
	if decision.Eligible {
		t.Fatal("a suspended roster entry should deny even when the rule matches")
	}
	if !strings.Contains(decision.Reason, "suspended") {
		t.Errorf("reason = %q, want it to mention the suspension", decision.Reason)
	}
}

func TestEligibilityIsPerElection(t *testing.T) {
	db := testDB(t)
	now := time.Now().UTC()
	// Election A allows std.mast.ac.ug; election B restricts to bba course only.
	openElection(t, db, "e-a", now.Add(-time.Hour), now.Add(time.Hour))
	db.Create(&models.Election{
		ID: "e-b", Title: "B", Status: models.ElectionStatusOpen,
		StartsAt:       now.Add(-time.Hour),
		EndsAt:         now.Add(time.Hour),
		AllowedDomain:  "std.mast.ac.ug",
		AllowedCourses: "bba", // BSE student is excluded
		ResultPolicy:   models.ResultPolicyAfterClose,
	})

	bseStudent := account(studentEmail("2022", "bse", "001"))
	inA, err := eligibility.Check(db, "e-a", bseStudent, now)
	if err != nil {
		t.Fatal(err)
	}
	inB, err := eligibility.Check(db, "e-b", bseStudent, now)
	if err != nil {
		t.Fatal(err)
	}
	if !inA.Eligible {
		t.Fatalf("BSE student should be eligible in election A: %s", inA.Reason)
	}
	if inB.Eligible {
		t.Fatalf("BSE student must not be eligible in election B (BBA-only): %s", inB.Reason)
	}
}

func TestClockAndStatusGate(t *testing.T) {
	db := testDB(t)
	now := time.Now().UTC()
	openElection(t, db, "e-a", now.Add(time.Hour), now.Add(2*time.Hour))
	student := account(studentEmail("2022", "bse", "001"))

	// Before polling opens: Check still evaluates eligibility but the
	// clock gate (IsOpen) must deny. Check calls lifecycle.Ensure.
	before, _ := eligibility.Check(db, "e-a", student, now)
	if before.Eligible {
		t.Error("a voter must be refused before the election opens")
	}
}

func TestFreezeRefusesNewEntriesButAllowsRevocation(t *testing.T) {
	db := testDB(t)
	now := time.Now().UTC()
	election := openElection(t, db, "e-a", now.Add(-time.Hour), now.Add(time.Hour))

	// One verified student in the DB so Freeze counts them.
	studentAddr := studentEmail("2022", "bse", "001")
	db.Create(&models.User{
		Email:           studentAddr,
		EmailVerifiedAt: &now,
	})

	_, err := eligibility.Freeze(db, election.ID, "officer", now)
	if err != nil {
		t.Fatal(err)
	}
	var frozen models.Election
	if err := db.Where("id = ?", election.ID).First(&frozen).Error; err != nil {
		t.Fatal(err)
	}
	// FrozenCount should be ≥ 1 (the student we created).
	if frozen.EligibilityFrozenAt == nil || frozen.EligibilityFrozenCount < 1 {
		t.Fatalf("freeze did not record the snapshot: %+v", frozen)
	}

	// New CSV import after freeze must be refused.
	csv := "student_number,email,full_name\n22/U/0002,late@std.mast.ac.ug,Late Arrival\n"
	if _, _, err := eligibility.Import(db, election.ID, "officer", models.RosterSourceCSVImport,
		"late.csv", strings.NewReader(csv), now); err == nil {
		t.Fatal("a new entry after the freeze should be refused")
	}

	// SetStatus can still revoke an existing roster entry.
	entry := rosterEntry(t, db, election.ID, "22/U/0001", studentAddr)
	if _, err := eligibility.SetStatus(db, election.ID, entry.ID, "officer",
		models.RosterStatusSuspended, "suspended by registrar"); err != nil {
		t.Fatal(err)
	}
	decision, err := eligibility.Check(db, election.ID, account(studentAddr), now)
	if err != nil {
		t.Fatal(err)
	}
	if decision.Eligible {
		t.Fatal("a revoked voter must be refused")
	}
}

func TestFreezeRefusesAfterCloseAndSetStatusRefusesAfterClose(t *testing.T) {
	db := testDB(t)
	now := time.Now().UTC()
	election := openElection(t, db, "e-a", now.Add(-2*time.Hour), now.Add(-time.Hour))
	entry := rosterEntry(t, db, "e-a", "22/U/0001", studentEmail("2022", "bse", "001"))

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
		"22/U/0001,one@std.mast.ac.ug,One Student,3,ELIGIBLE",
		"22/U/0002,not-an-email,Two Student,2,ELIGIBLE",
		"22/U/0001,two@std.mast.ac.ug,Duplicate Student,2,ELIGIBLE",
		"22/U/0003,one@std.mast.ac.ug,Duplicate Identity,2,ELIGIBLE",
		"22/U/0004,four@std.mast.ac.ug,Four Student,99,ELIGIBLE",
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
}

func TestImportCommitRecordsProvenance(t *testing.T) {
	db := testDB(t)
	now := time.Now().UTC()
	openElection(t, db, "e-a", now.Add(-time.Hour), now.Add(time.Hour))

	csv := "student_number,reg_number,email,full_name,faculty,department,programme,study_year,status\n" +
		"22/U/0001,REG/22/1,one@std.mast.ac.ug,One Student,Science,CS,BSc CS,3,ELIGIBLE\n"
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
}

func TestReportHasCountsOnly(t *testing.T) {
	db := testDB(t)
	now := time.Now().UTC()
	openElection(t, db, "e-a", now.Add(-time.Hour), now.Add(time.Hour))

	// Add two verified students.
	db.Create(&models.User{Email: studentEmail("2022", "bse", "001"), EmailVerifiedAt: &now})
	db.Create(&models.User{Email: studentEmail("2022", "bse", "002"), EmailVerifiedAt: &now})

	if err := db.Create(&models.VoterParticipation{ElectionID: "e-a", UserID: "user-1"}).Error; err != nil {
		t.Fatal(err)
	}

	report, err := eligibility.BuildReport(db, "e-a")
	if err != nil {
		t.Fatal(err)
	}
	// 2 eligible students, 1 participated
	if report.Eligible != 2 || report.Participated != 1 {
		t.Errorf("unexpected report: eligible=%d participated=%d", report.Eligible, report.Participated)
	}
	// Report must not expose individual identity.
	encoded, _ := json.Marshal(report)
	for _, forbidden := range []string{"user-1"} {
		if strings.Contains(string(encoded), forbidden) {
			t.Errorf("report leaked %q: %s", forbidden, encoded)
		}
	}
}

func TestRevocationAfterVotingLeavesTheBallotStanding(t *testing.T) {
	db := testDB(t)
	now := time.Now().UTC()
	openElection(t, db, "e-a", now.Add(-time.Hour), now.Add(time.Hour))
	email := studentEmail("2022", "bse", "001")
	entry := rosterEntry(t, db, "e-a", "22/U/0001", email)

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

	decision, err := eligibility.Check(db, "e-a", account(email), time.Now().UTC())
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
