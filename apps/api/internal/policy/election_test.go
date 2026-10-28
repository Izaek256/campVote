package policy_test

import (
	"errors"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"campvote/apps/api/internal/models"
	"campvote/apps/api/internal/policy"
)

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
	if err := models.SeedRoles(db); err != nil {
		t.Fatal(err)
	}
	return db
}

func user(t *testing.T, db *gorm.DB, id, role string) {
	t.Helper()
	if err := db.Create(&models.User{
		ID: id, FirstName: "Test", LastName: id, Email: id + "@must.ac.ug",
		Password: "not-used-here", Role: role, Active: true,
	}).Error; err != nil {
		t.Fatal(err)
	}
}

func election(t *testing.T, db *gorm.DB, id, status, resultPolicy, creator string) {
	t.Helper()
	now := time.Now().UTC()
	e := models.Election{
		ID: id, Title: id, Status: status, ResultPolicy: resultPolicy,
		StartsAt: now.Add(-time.Hour), EndsAt: now.Add(time.Hour), CreatedByID: creator,
	}
	if err := db.Create(&e).Error; err != nil {
		t.Fatal(err)
	}
}

func assign(t *testing.T, db *gorm.DB, electionID, userID, role string) {
	t.Helper()
	if err := db.Create(&models.ElectionRole{ElectionID: electionID, UserID: userID, Role: role}).Error; err != nil {
		t.Fatal(err)
	}
}

func actor(t *testing.T, db *gorm.DB, id string) policy.Actor {
	t.Helper()
	a, err := policy.ActorFor(db, id)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func TestAuthorityScopesOfficialToAssignedElection(t *testing.T) {
	db := testDB(t)
	user(t, db, "official", models.RoleElectionOfficial)
	election(t, db, "e-a", models.ElectionStatusOpen, models.ResultPolicyAfterClose, "creator")
	election(t, db, "e-b", models.ElectionStatusOpen, models.ResultPolicyAfterClose, "creator")
	assign(t, db, "e-a", "official", models.RoleElectionOfficial)

	a, err := policy.AuthorityFor(db, actor(t, db, "official"), "e-a")
	if err != nil {
		t.Fatalf("assigned election: %v", err)
	}
	if !a.Can(policy.PermElectionEdit) {
		t.Error("official should edit their election")
	}
	if _, err := policy.AuthorityFor(db, actor(t, db, "official"), "e-b"); !errors.Is(err, policy.ErrNotVisible) {
		t.Errorf("cross-election access = %v, want ErrNotVisible", err)
	}
	if _, err := policy.AuthorityFor(db, actor(t, db, "official"), "missing"); !errors.Is(err, policy.ErrNotVisible) {
		t.Errorf("unknown election = %v, want ErrNotVisible", err)
	}
}

func TestCandidateCannotAdministerOwnElection(t *testing.T) {
	db := testDB(t)
	user(t, db, "candidate", models.RoleCandidate)
	election(t, db, "e-a", models.ElectionStatusOpen, models.ResultPolicyAfterClose, "creator")
	assign(t, db, "e-a", "candidate", models.RoleCandidate)

	a, err := policy.AuthorityFor(db, actor(t, db, "candidate"), "e-a")
	if err != nil {
		t.Fatal(err)
	}
	if !a.IsCandidate() {
		t.Fatal("candidate assignment not recognised")
	}
	if a.Can(policy.PermElectionEdit) {
		t.Error("a candidate must not administer the election they stand in")
	}
	if a.Can(policy.PermElectionRoles) {
		t.Error("a candidate must not manage election roles")
	}
}

func TestRoleAssignmentConflict(t *testing.T) {
	cases := []struct {
		name     string
		role     string
		existing []string
		conflict bool
	}{
		{"candidate then official", models.RoleElectionOfficial, []string{models.RoleCandidate}, true},
		{"official then candidate", models.RoleCandidate, []string{models.RoleElectionOfficial}, true},
		{"auditor then candidate", models.RoleCandidate, []string{models.RoleElectionAuditor}, true},
		{"two officials", models.RoleElectionOfficial, []string{models.RoleChiefElectionAdmin}, false},
		{"candidate alone", models.RoleCandidate, nil, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := policy.RoleAssignmentConflict(tc.role, tc.existing) != ""
			if got != tc.conflict {
				t.Errorf("conflict = %v, want %v", got, tc.conflict)
			}
		})
	}
}

func TestResultsVisibilityFollowsElectionPolicy(t *testing.T) {
	db := testDB(t)
	user(t, db, "official", models.RoleElectionOfficial)
	user(t, db, "candidate", models.RoleCandidate)
	user(t, db, "voter", models.RoleStudentVoter)
	election(t, db, "live", models.ElectionStatusOpen, models.ResultPolicyLive, "creator")
	election(t, db, "after-open", models.ElectionStatusOpen, models.ResultPolicyAfterClose, "creator")
	election(t, db, "after-closed", models.ElectionStatusClosed, models.ResultPolicyAfterClose, "creator")
	election(t, db, "cert-review", models.ElectionStatusReview, models.ResultPolicyCertifiedOnly, "creator")
	election(t, db, "cert-final", models.ElectionStatusFinalized, models.ResultPolicyCertifiedOnly, "creator")

	assign(t, db, "live", "official", models.RoleElectionOfficial)
	assign(t, db, "live", "candidate", models.RoleCandidate)
	assign(t, db, "after-open", "official", models.RoleElectionOfficial)
	assign(t, db, "after-closed", "official", models.RoleElectionOfficial)
	assign(t, db, "cert-review", "official", models.RoleElectionOfficial)
	assign(t, db, "cert-final", "official", models.RoleElectionOfficial)

	cases := []struct {
		name     string
		user     string
		election string
		allowed  bool
	}{
		{"official sees live provisional", "official", "live", true},
		{"candidate cannot see live provisional", "candidate", "live", false},
		{"voter cannot see live provisional", "voter", "live", false},
		{"official cannot see early after-close tally", "official", "after-open", false},
		{"official sees after close", "official", "after-closed", true},
		{"official cannot see review-stage certified-only tally", "official", "cert-review", false},
		{"official sees certified-only after finalisation", "official", "cert-final", true},
	}
	now := time.Now().UTC()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a, err := policy.AuthorityFor(db, actor(t, db, tc.user), tc.election)
			if err != nil {
				if errors.Is(err, policy.ErrNotVisible) && !tc.allowed {
					return
				}
				t.Fatal(err)
			}
			got, reason := a.CanReadResults(now)
			if got != tc.allowed {
				t.Errorf("allowed = %v, want %v (%s)", got, tc.allowed, reason)
			}
		})
	}
}

func TestRoleChangeAppliesToNextRequest(t *testing.T) {
	db := testDB(t)
	user(t, db, "official", models.RoleElectionOfficial)
	election(t, db, "e-a", models.ElectionStatusOpen, models.ResultPolicyAfterClose, "creator")
	assign(t, db, "e-a", "official", models.RoleElectionOfficial)

	if _, err := policy.AuthorityFor(db, actor(t, db, "official"), "e-a"); err != nil {
		t.Fatalf("before removal: %v", err)
	}
	if err := db.Where("election_id = ? AND user_id = ?", "e-a", "official").
		Delete(&models.ElectionRole{}).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := policy.AuthorityFor(db, actor(t, db, "official"), "e-a"); !errors.Is(err, policy.ErrNotVisible) {
		t.Errorf("after removal = %v, want ErrNotVisible", err)
	}
}

func TestCertifyRequiresTwoOfficialsAndADifferentActor(t *testing.T) {
	db := testDB(t)
	user(t, db, "creator", models.RoleChiefElectionAdmin)
	user(t, db, "one", models.RoleElectionOfficial)
	user(t, db, "two", models.RoleElectionOfficial)
	election(t, db, "e-a", models.ElectionStatusClosed, models.ResultPolicyAfterClose, "creator")
	assign(t, db, "e-a", "one", models.RoleChiefElectionAdmin)
	assign(t, db, "e-a", "two", models.RoleElectionOfficial)

	a, err := policy.AuthorityFor(db, actor(t, db, "one"), "e-a")
	if err != nil {
		t.Fatal(err)
	}
	if ok, reason := policy.CertifyDecision(db, a, actor(t, db, "one")); !ok {
		t.Errorf("assigned non-creator official should certify: %s", reason)
	}

	if err := db.Where("election_id = ? AND user_id = ?", "e-a", "two").
		Delete(&models.ElectionRole{}).Error; err != nil {
		t.Fatal(err)
	}
	if ok, _ := policy.CertifyDecision(db, a, actor(t, db, "one")); ok {
		t.Error("one official alone must not certify")
	}

	assign(t, db, "e-a", "two", models.RoleElectionOfficial)
	creatorAuthority, err := policy.AuthorityFor(db, actor(t, db, "creator"), "e-a")
	if err != nil {
		t.Fatal(err)
	}
	if ok, _ := policy.CertifyDecision(db, creatorAuthority, actor(t, db, "creator")); ok {
		t.Error("the election creator must not certify its own result")
	}
}
