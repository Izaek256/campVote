package policy

import (
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"campvote/apps/api/internal/authz"
	"campvote/apps/api/internal/models"
	"campvote/apps/api/internal/services"
)

// Permission keys used by the election policies. They match the catalog in
// internal/authz/permissions.go.
const (
	PermElectionView    = "elections.view"
	PermElectionEdit    = "elections.edit"
	PermElectionPublish = "elections.publish"
	PermElectionRoles   = "elections.roles"
	PermElectionCertify = "elections.certify"
	PermCandidateReview = "candidates.review"
	PermResultsView     = "results.view"
	PermResultsViewLive = "results.view_live"
	PermAuditView       = "audit.view"
	PermRosterView      = "rosters.view"
	PermRosterEdit      = "rosters.edit"
	PermRosterImport    = "rosters.import"
	PermBallotView      = "ballots.view"
	PermBallotEdit      = "ballots.edit"
)

// ErrNotVisible means the election does not exist, or this actor has no
// relationship to it. Both are answered 404 so a guessed id cannot be used to
// discover that an election exists.
var ErrNotVisible = errors.New("policy: election not found or not visible")

// Actor is the authenticated caller plus the grants the auth middleware
// already resolved.
type Actor struct {
	UserID string
	Role   string
	Grants []string
	Admin  bool
}

// ActorFrom reads the caller the auth middleware set on the request.
func ActorFrom(c *gin.Context) Actor {
	grants, _ := c.Get("user_grants")
	list, _ := grants.([]string)
	return Actor{
		UserID: c.GetString("user_id"),
		Role:   c.GetString("user_role"),
		Grants: list,
		Admin:  authz.IsAdmin(c),
	}
}

// ActorFor loads a user's role and grants. Background jobs and tests use it
// where there is no request context.
func ActorFor(db *gorm.DB, userID string) (Actor, error) {
	if userID == "" {
		return Actor{}, nil
	}
	var user models.User
	if err := db.Where("id = ?", userID).First(&user).Error; err != nil {
		return Actor{}, err
	}
	grants, err := authz.GrantsFor(db, userID)
	if err != nil {
		return Actor{}, err
	}
	return Actor{UserID: user.ID, Role: user.Role, Grants: grants, Admin: user.Role == models.RoleAdmin}, nil
}

func (a Actor) has(permission string) bool {
	if a.Admin {
		return true
	}
	return authz.Granted(a.Grants, permission)
}

// roleGrants is the static half of the matrix for a per-election role. It
// mirrors DefaultRoles; keeping it here means a scoped assignment needs no
// second trip through the roles table and cannot be widened by editing a role
// row that means something else globally.
func roleGrants(role string) []string {
	switch role {
	case models.RoleChiefElectionAdmin:
		return []string{"elections.*", "candidates.*", "rosters.*", "ballots.*", "results.*", "audit.view"}
	case models.RoleElectionOfficial:
		return []string{PermElectionView, PermElectionEdit, "candidates.*", PermRosterView, PermBallotView, PermResultsView, PermResultsViewLive}
	case models.RoleElectionAuditor:
		return []string{PermElectionView, "candidates.view", PermResultsView, PermAuditView}
	case models.RoleCandidate:
		// Viewing a provisional tally needs results.view_live, which a candidate
		// never holds, so this grants published results only.
		return []string{PermElectionView, "candidates.view", PermResultsView}
	default:
		return nil
	}
}

// Authority is what an actor may do to one election.
type Authority struct {
	Election models.Election
	// Global means the actor's role applies to every election. Officials,
	// candidates and auditors are scoped instead, even if they also hold a
	// global grant for the same key.
	Global bool
	Roles  []string
	Grants []string
}

// AuthorityFor loads the election and the actor's relationship to it. A
// missing election and an election the actor has no relationship with both
// return ErrNotVisible.
func AuthorityFor(db *gorm.DB, actor Actor, electionID string) (*Authority, error) {
	var election models.Election
	if err := db.Where("id = ?", electionID).First(&election).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotVisible
		}
		return nil, err
	}

	var assignments []models.ElectionRole
	if actor.UserID != "" {
		if err := db.Where("election_id = ? AND user_id = ?", electionID, actor.UserID).
			Find(&assignments).Error; err != nil {
			return nil, err
		}
	}

	authority := &Authority{Election: election, Grants: append([]string(nil), actor.Grants...)}
	for _, a := range assignments {
		authority.Roles = append(authority.Roles, a.Role)
		for _, g := range roleGrants(a.Role) {
			authority.Grants = append(authority.Grants, g)
		}
	}

	// Scoped campus roles never become global by holding a grant. A custom role
	// with elections.view is global, because only a global role can be assigned
	// outside an election. The global role name alone is enough to know that an
	// official or candidate is scoped: otherwise their role grants would make
	// every election visible.
	scoped := actor.Role == models.RoleElectionOfficial ||
		actor.Role == models.RoleCandidate ||
		actor.Role == models.RoleElectionAuditor
	for _, r := range authority.Roles {
		if r == models.RoleElectionOfficial || r == models.RoleCandidate || r == models.RoleElectionAuditor {
			scoped = true
		}
	}
	if actor.Admin || actor.Role == models.RoleChiefElectionAdmin {
		authority.Global = true
	} else if !scoped && authz.Granted(actor.Grants, PermElectionView) {
		authority.Global = true
	}

	if !authority.Global && len(authority.Roles) == 0 {
		return nil, ErrNotVisible
	}
	return authority, nil
}

// Can reports whether the authority holds permission for this election.
func (a *Authority) Can(permission string) bool {
	if a == nil {
		return false
	}
	return authz.Granted(a.Grants, permission)
}

// IsCandidate reports whether the actor stands in this election.
func (a *Authority) IsCandidate() bool {
	for _, r := range a.Roles {
		if r == models.RoleCandidate {
			return true
		}
	}
	return false
}

// ResultsVisible applies the election's publication policy. It says nothing
// about who may ask; CanReadResults does that.
func ResultsVisible(e models.Election, now time.Time) (bool, string) {
	switch e.ResultPolicy {
	case models.ResultPolicyLive:
		return true, ""
	case models.ResultPolicyAfterClose:
		switch e.Status {
		case models.ElectionStatusClosed, models.ElectionStatusReview,
			models.ElectionStatusFinalized, models.ElectionStatusArchived:
			return true, ""
		}
		return false, "Results are published after voting closes"
	case models.ResultPolicyCertifiedOnly:
		switch e.Status {
		case models.ElectionStatusFinalized, models.ElectionStatusArchived:
			return true, ""
		}
		return false, "Results are published only after certification"
	default:
		// An unrecognised policy is not permission to publish.
		return false, "This election has no recognised results policy"
	}
}

// CanReadResults combines the actor's grant with the election's publication
// policy. Provisional tallies while voting is open need results.view_live;
// after close or certification results.view is enough.
func (a *Authority) CanReadResults(now time.Time) (bool, string) {
	visible, reason := ResultsVisible(a.Election, now)
	if !visible {
		return false, reason
	}
	provisional := a.Election.Status == models.ElectionStatusOpen
	if provisional {
		if !a.Can(PermResultsViewLive) {
			return false, "Provisional results are restricted to authorised election roles"
		}
		return true, ""
	}
	if !a.Can(PermResultsView) {
		return false, "You are not authorised to read results for this election"
	}
	return true, ""
}

// RoleAssignmentConflict enforces separation of duties. A candidate cannot also
// administer or observe the election they stand in, and an official cannot be
// made a candidate in an election they administer.
func RoleAssignmentConflict(role string, existing []string) string {
	candidate := false
	staff := false
	for _, r := range existing {
		if r == models.RoleCandidate {
			candidate = true
		}
		if models.IsElectionAdminRole(r) || r == models.RoleElectionAuditor {
			staff = true
		}
	}
	switch role {
	case models.RoleCandidate:
		if staff {
			return "a candidate cannot also administer or observe the same election"
		}
	case models.RoleElectionOfficial, models.RoleChiefElectionAdmin, models.RoleElectionAuditor:
		if candidate {
			return "a candidate cannot also administer or observe the election they stand in"
		}
	}
	return ""
}

// ValidElectionRole reports whether role may be assigned per election.
func ValidElectionRole(role string) bool {
	switch role {
	case models.RoleChiefElectionAdmin, models.RoleElectionOfficial,
		models.RoleCandidate, models.RoleElectionAuditor:
		return true
	}
	return false
}

// CertifyDecision checks the certification rule for an authority: the actor
// holds elections.certify, is not the election's creator, and is not the only
// election official. The last part is the separation-of-duties requirement
// that one person does not both run and certify the count.
func CertifyDecision(db *gorm.DB, a *Authority, actor Actor) (bool, string) {
	if !a.Can(PermElectionCertify) {
		return false, "You are not authorised to certify results"
	}
	if actor.UserID != "" && actor.UserID == a.Election.CreatedByID {
		return false, "the election creator cannot certify its results"
	}
	switch a.Election.Status {
	case models.ElectionStatusClosed, models.ElectionStatusReview:
	default:
		return false, "results can be certified only after voting has closed and before finalisation"
	}

	var officials []string
	if err := db.Model(&models.ElectionRole{}).
		Where("election_id = ? AND role IN ?", a.Election.ID,
			[]string{models.RoleElectionOfficial, models.RoleChiefElectionAdmin}).
		Distinct().Pluck("user_id", &officials).Error; err != nil {
		return false, "the election officials could not be checked"
	}
	if len(officials) < 2 {
		return false, "a second election official must be assigned before results can be certified"
	}

	isOfficial := false
	for _, id := range officials {
		if id == actor.UserID {
			isOfficial = true
		}
	}
	if !isOfficial && !a.Global {
		return false, "only an assigned election official may certify"
	}
	return true, ""
}

// LogDenial writes an object-level refusal. The path and action are recorded;
// request bodies never are, so ballot choices cannot leak into the audit log.
func LogDenial(db *gorm.DB, c *gin.Context, reason, resourceID string) {
	if db == nil {
		return
	}
	services.LogActivity(db, c, services.ActivityArgs{
		UserID:       c.GetString("user_id"),
		Action:       "authz.denied",
		Severity:     "warn",
		Summary:      "Denied " + c.Request.Method + " " + c.FullPath() + ": " + reason,
		ResourceType: "election",
		ResourceID:   resourceID,
	})
}
