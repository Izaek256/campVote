package handlers_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"campvote/apps/api/internal/handlers"
	"campvote/apps/api/internal/landing"
	"campvote/apps/api/internal/middleware"
	"campvote/apps/api/internal/models"
	"campvote/apps/api/internal/policy"
	"campvote/apps/api/internal/testkit"
)

// landingEngine mounts the landing routes the way the real router does: the
// public reads are truly open, and the CMS routes sit behind the same
// APIKeyOrAuth + policy.Guard + RequireRole("ADMIN") stack the admin group
// uses. policy.Guard is on the public group too, so the test also proves those
// paths are declared Public rather than accidentally protected.
func landingEngine(t *testing.T, db *gorm.DB) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	auth := testkit.Auth(testkit.Config(), db)
	h := handlers.NewLandingHandler(db)

	r := gin.New()
	r.Use(policy.Declared())

	pub := r.Group("")
	pub.Use(policy.Guard(db))
	pub.GET("/api/v1/public/landing", h.Public)
	pub.GET("/api/v1/public/elections/upcoming", h.UpcomingElections)

	admin := r.Group("")
	admin.Use(middleware.APIKeyOrAuth(db, middleware.Auth(db, auth)))
	admin.Use(policy.Guard(db))
	admin.Use(middleware.RequireRole(models.RoleAdmin))
	admin.GET("/api/v1/admin/landing", h.AdminGet)
	admin.PUT("/api/v1/admin/landing", h.AdminSave)
	admin.POST("/api/v1/admin/landing/publish", h.AdminPublish)
	admin.POST("/api/v1/admin/landing/revert", h.AdminRevert)

	return r
}

// publicElection writes an election with the fields the landing query reads.
func publicElection(t *testing.T, db *gorm.DB, id, status string, startsAt time.Time, listed bool) *models.Election {
	t.Helper()
	e := &models.Election{
		ID:             id,
		Title:          id + " title",
		Category:       models.ElectionCategoryGuild,
		OrganizingUnit: "Electoral Commission",
		Status:         status,
		StartsAt:       startsAt,
		EndsAt:         startsAt.Add(48 * time.Hour),
		AllowedDomain:  "std.mast.ac.ug",
		ResultPolicy:   models.ResultPolicyAfterClose,
		PublicListing:  listed,
		CreatedByID:    "creator",
	}
	if err := db.Create(e).Error; err != nil {
		t.Fatal(err)
	}
	return e
}

// landingBody is a valid draft payload, as the admin screen would send it.
func landingBody(headline string) map[string]any {
	return map[string]any{
		"hero": map[string]any{
			"headline":        headline,
			"subtext":         "A plain-text subtext for the test.",
			"primary_label":   "Sign in",
			"secondary_label": "Learn more",
		},
		"banner": map[string]any{"enabled": false, "text": "", "link": ""},
		"ticker": []string{"One voter", "One ballot"},
		"how": map[string]any{"steps": []map[string]any{
			{"title": "Sign in", "text": "Use your campus account."},
		}},
		"features": []map[string]any{
			{"icon": "elections", "title": "Many elections", "text": "Run several at once."},
		},
		"faqs": []map[string]any{
			{"question": "Can anyone see my vote?", "answer": "No, choices are stored separately."},
		},
		"footer": map[string]any{"text": "A university project.", "support_email": ""},
	}
}

func TestLandingUpcomingOnlyReturnsListedFutureDrafts(t *testing.T) {
	db := rbacDB(t)
	now := time.Now().UTC()

	publicElection(t, db, "listed-future", models.ElectionStatusDraft, now.Add(48*time.Hour), true)
	publicElection(t, db, "not-listed", models.ElectionStatusDraft, now.Add(72*time.Hour), false)
	publicElection(t, db, "open", models.ElectionStatusOpen, now.Add(96*time.Hour), true)
	publicElection(t, db, "closed", models.ElectionStatusClosed, now.Add(-24*time.Hour), true)
	publicElection(t, db, "listed-past", models.ElectionStatusDraft, now.Add(-time.Hour), true)

	client := testkit.New(t, landingEngine(t, db))
	res := client.Get("/api/v1/public/elections/upcoming").AssertOK()
	res.AssertHeader("Cache-Control", "public, max-age=60")
	res.JSON().Count("data", 1).Where("data.0.id", "listed-future")
}

func TestLandingUpcomingServesWhitelistedAggregatesOnly(t *testing.T) {
	db := rbacDB(t)
	now := time.Now().UTC()

	// Roster-backed, opening soonest: its aggregate eligible count is allowed.
	earlier := publicElection(t, db, "earlier", models.ElectionStatusDraft, now.Add(24*time.Hour), true)
	if err := db.Model(&models.Election{}).Where("id = ?", earlier.ID).
		Update("require_roster", true).Error; err != nil {
		t.Fatal(err)
	}
	listed(t, db, earlier.ID, "22/U/0001", "one@std.mast.ac.ug")
	listed(t, db, earlier.ID, "22/U/0002", "two@std.mast.ac.ug")

	// Open to all, opening later: no count can be computed, so it is omitted.
	publicElection(t, db, "later", models.ElectionStatusDraft, now.Add(72*time.Hour), true)

	client := testkit.New(t, landingEngine(t, db))
	res := client.Get("/api/v1/public/elections/upcoming").AssertOK()
	j := res.JSON()

	j.Count("data", 2)
	// Ordered by opening date, soonest first.
	j.Where("data.0.id", "earlier").Where("data.1.id", "later")

	// Only the whitelisted fields are present.
	for _, path := range []string{
		"data.0.id", "data.0.title", "data.0.category", "data.0.organizing_unit",
		"data.0.positions", "data.0.opens_at", "data.0.closes_at", "data.0.status",
	} {
		j.Has(path)
	}

	// Nothing internal, and nothing that could identify a voter.
	for _, path := range []string{
		"data.0.created_by_id", "data.0.allowed_domain", "data.0.allowed_courses",
		"data.0.require_roster", "data.0.public_listing", "data.0.result_policy",
		"data.0.description", "data.0.email", "data.0.roster",
	} {
		j.Missing(path)
	}

	// The count is an aggregate, published only for the roster-backed election.
	j.Where("data.0.eligible_count", 2).Missing("data.1.eligible_count")
}

func TestLandingPublicContentDefaultsThenPublished(t *testing.T) {
	db := rbacDB(t)
	client := testkit.New(t, landingEngine(t, db))
	admin := testkit.User(t, db, "admin@must.ac.ug", models.RoleAdmin)
	token := testkit.SignIn(t, db, testkit.Auth(testkit.Config(), db), admin)

	// Nothing published yet: the built-in defaults keep the page whole.
	var fresh landing.Document
	client.Get("/api/v1/public/landing").AssertOK().
		AssertHeader("Cache-Control", "public, max-age=60").BindData(&fresh)
	if fresh.Hero.Headline != landing.Default().Hero.Headline {
		t.Fatalf("public landing headline = %q, want the built-in default", fresh.Hero.Headline)
	}

	// Saving a draft does not change the public page.
	client.As(token).Put("/api/v1/admin/landing", landingBody("Edited headline")).AssertOK()
	var stillDefault landing.Document
	client.Get("/api/v1/public/landing").AssertOK().BindData(&stillDefault)
	if stillDefault.Hero.Headline == "Edited headline" {
		t.Fatal("a saved draft leaked to the public page before publishing")
	}

	// Publishing flips the public page to the draft.
	client.As(token).Post("/api/v1/admin/landing/publish", nil).AssertOK()
	var published landing.Document
	client.Get("/api/v1/public/landing").AssertOK().BindData(&published)
	if published.Hero.Headline != "Edited headline" {
		t.Fatalf("public landing headline = %q, want the published edit", published.Hero.Headline)
	}
}

func TestLandingAdminRequiresAdminRole(t *testing.T) {
	db := rbacDB(t)
	client := testkit.New(t, landingEngine(t, db))
	admin := testkit.User(t, db, "admin@must.ac.ug", models.RoleAdmin)
	voter := testkit.User(t, db, "2022bse001@std.mast.ac.ug", models.RoleStudentVoter)
	auth := testkit.Auth(testkit.Config(), db)
	adminToken := testkit.SignIn(t, db, auth, admin)
	voterToken := testkit.SignIn(t, db, auth, voter)

	client.Get("/api/v1/admin/landing").AssertUnauthorized()
	client.As(voterToken).Get("/api/v1/admin/landing").AssertForbidden()
	client.As(adminToken).Get("/api/v1/admin/landing").AssertOK()

	client.Put("/api/v1/admin/landing", landingBody("x")).AssertUnauthorized()
	client.As(voterToken).Put("/api/v1/admin/landing", landingBody("x")).AssertForbidden()
	client.As(adminToken).Put("/api/v1/admin/landing", landingBody("x")).AssertOK()

	client.Post("/api/v1/admin/landing/publish", nil).AssertUnauthorized()
	client.As(voterToken).Post("/api/v1/admin/landing/publish", nil).AssertForbidden()

	client.Post("/api/v1/admin/landing/revert", nil).AssertUnauthorized()
	client.As(voterToken).Post("/api/v1/admin/landing/revert", nil).AssertForbidden()
}

// assertValidationField fails unless the response is a 422 validation error
// carrying a message for field. The field names contain dots ("hero.headline"),
// which the body's path helper cannot address, so the details map is read
// directly.
func assertValidationField(t *testing.T, res *testkit.Response, field string) {
	t.Helper()
	res.AssertUnprocessable().AssertCode("VALIDATION_ERROR")
	var body struct {
		Error struct {
			Details map[string]string `json:"details"`
		} `json:"error"`
	}
	res.Bind(&body)
	if _, ok := body.Error.Details[field]; !ok {
		t.Errorf("validation details = %v, want a message for %q\n%s", body.Error.Details, field, res.Body())
	}
}

func TestLandingAdminSaveRejectsMarkupBadIconAndUnsafeLink(t *testing.T) {
	db := rbacDB(t)
	client := testkit.New(t, landingEngine(t, db))
	admin := testkit.User(t, db, "admin@must.ac.ug", models.RoleAdmin)
	token := testkit.SignIn(t, db, testkit.Auth(testkit.Config(), db), admin)

	// Markup is refused: the CMS stores plain text only.
	assertValidationField(t,
		client.As(token).Put("/api/v1/admin/landing", landingBody("<script>alert(1)</script>")),
		"hero.headline")

	// An icon outside the allowlist is refused.
	badIcon := landingBody("Fine headline")
	badIcon["features"].([]map[string]any)[0]["icon"] = "not-a-real-icon"
	assertValidationField(t,
		client.As(token).Put("/api/v1/admin/landing", badIcon),
		"features.0.icon")

	// A javascript: link is refused; only /path or https:// is allowed.
	badLink := landingBody("Fine headline")
	badLink["banner"].(map[string]any)["enabled"] = true
	badLink["banner"].(map[string]any)["text"] = "Notice"
	badLink["banner"].(map[string]any)["link"] = "javascript:alert(1)"
	assertValidationField(t,
		client.As(token).Put("/api/v1/admin/landing", badLink),
		"banner.link")
}

func TestLandingPublishAndRevertTrackDraftState(t *testing.T) {
	db := rbacDB(t)
	client := testkit.New(t, landingEngine(t, db))
	admin := testkit.User(t, db, "admin@must.ac.ug", models.RoleAdmin)
	token := testkit.SignIn(t, db, testkit.Auth(testkit.Config(), db), admin)

	// Publish a first version.
	client.As(token).Put("/api/v1/admin/landing", landingBody("Version one")).AssertOK()
	client.As(token).Post("/api/v1/admin/landing/publish", nil).AssertOK()

	// Edit the draft only: it is now dirty relative to the published copy.
	client.As(token).Put("/api/v1/admin/landing", landingBody("Version two")).AssertOK()

	var dirty struct {
		Draft struct {
			Hero landing.Hero `json:"hero"`
		} `json:"draft"`
		Published struct {
			Hero landing.Hero `json:"hero"`
		} `json:"published"`
		HasPublished bool `json:"has_published"`
		Dirty        bool `json:"dirty"`
	}
	client.As(token).Get("/api/v1/admin/landing").AssertOK().BindData(&dirty)
	if !dirty.HasPublished || !dirty.Dirty {
		t.Fatalf("after an unpublished edit: has_published=%v dirty=%v, want both true", dirty.HasPublished, dirty.Dirty)
	}
	if dirty.Draft.Hero.Headline != "Version two" || dirty.Published.Hero.Headline != "Version one" {
		t.Fatalf("draft=%q published=%q, want the draft ahead of the published copy",
			dirty.Draft.Hero.Headline, dirty.Published.Hero.Headline)
	}

	// Revert discards the draft edit and restores the published version.
	var reverted struct {
		Draft struct {
			Hero landing.Hero `json:"hero"`
		} `json:"draft"`
		Dirty bool `json:"dirty"`
	}
	client.As(token).Post("/api/v1/admin/landing/revert", nil).AssertOK().BindData(&reverted)
	if reverted.Dirty || reverted.Draft.Hero.Headline != "Version one" {
		t.Fatalf("after revert: dirty=%v headline=%q, want the published version and no pending changes",
			reverted.Dirty, reverted.Draft.Hero.Headline)
	}

	// The public page still serves the published version.
	var published landing.Document
	client.Get("/api/v1/public/landing").AssertOK().BindData(&published)
	if published.Hero.Headline != "Version one" {
		t.Fatalf("public headline = %q, want the published version", published.Hero.Headline)
	}

	// Every change is recorded in the activity log.
	for _, action := range []string{"landing.update", "landing.publish", "landing.revert"} {
		var n int64
		db.Model(&models.UserActivity{}).Where("action = ?", action).Count(&n)
		if n == 0 {
			t.Errorf("expected an activity row for %s", action)
		}
	}
}

// TestLandingPublishOmitsHiddenItems covers the filter that keeps a hidden
// feature or FAQ off the public snapshot. Visibility is not exposed through the
// save payload, so the rows are set directly, the way a future visibility
// toggle would.
func TestLandingPublishOmitsHiddenItems(t *testing.T) {
	db := rbacDB(t)
	client := testkit.New(t, landingEngine(t, db))
	admin := testkit.User(t, db, "admin@must.ac.ug", models.RoleAdmin)
	token := testkit.SignIn(t, db, testkit.Auth(testkit.Config(), db), admin)

	client.As(token).Put("/api/v1/admin/landing", landingBody("Publish me")).AssertOK()

	// Hide the single feature, then publish.
	if err := db.Model(&models.LandingFeature{}).Where("1 = 1").
		Update("visible", false).Error; err != nil {
		t.Fatal(err)
	}
	client.As(token).Post("/api/v1/admin/landing/publish", nil).AssertOK()

	var published landing.Document
	client.Get("/api/v1/public/landing").AssertOK().BindData(&published)
	if len(published.Features) != 0 {
		t.Fatalf("published features = %d, want a hidden feature to be left out", len(published.Features))
	}
	if len(published.FAQs) != 1 {
		t.Fatalf("published faqs = %d, want the visible FAQ kept", len(published.FAQs))
	}
}

// TestLandingAdminSavePersistsVisibilityAndOrder proves the editor's two list
// controls survive a save: hidden items stay in the draft but not on the
// public page, and the order the editor sends is the order published.
func TestLandingAdminSavePersistsVisibilityAndOrder(t *testing.T) {
	db := rbacDB(t)
	client := testkit.New(t, landingEngine(t, db))
	admin := testkit.User(t, db, "admin@must.ac.ug", models.RoleAdmin)
	token := testkit.SignIn(t, db, testkit.Auth(testkit.Config(), db), admin)

	body := landingBody("Visibility test")
	body["features"] = []map[string]any{
		{"icon": "turnout", "title": "Second card", "text": "Hidden from the page.", "visible": false},
		{"icon": "elections", "title": "First card", "text": "The one that remains.", "visible": true},
	}
	client.As(token).Put("/api/v1/admin/landing", body).AssertOK()

	var state struct {
		Draft struct {
			Features []struct {
				Title   string `json:"title"`
				Visible bool   `json:"visible"`
			} `json:"features"`
		} `json:"draft"`
		Dirty bool `json:"dirty"`
	}
	client.As(token).Get("/api/v1/admin/landing").AssertOK().BindData(&state)
	if len(state.Draft.Features) != 2 {
		t.Fatalf("draft features = %d, want both rows kept", len(state.Draft.Features))
	}
	if state.Draft.Features[0].Title != "Second card" || state.Draft.Features[0].Visible {
		t.Fatalf("first draft row = %+v, want the hidden card in its saved position", state.Draft.Features[0])
	}
	if state.Draft.Features[1].Title != "First card" || !state.Draft.Features[1].Visible {
		t.Fatalf("second draft row = %+v, want the visible card in its saved position", state.Draft.Features[1])
	}

	client.As(token).Post("/api/v1/admin/landing/publish", nil).AssertOK()
	var published landing.Document
	client.Get("/api/v1/public/landing").AssertOK().BindData(&published)
	if len(published.Features) != 1 || published.Features[0].Title != "First card" {
		t.Fatalf("published features = %+v, want only the visible card", published.Features)
	}
}

// TestLandingUpcomingIsOpenWithoutCredentials pins the promise that the landing
// reads need no token: an anonymous request must not be refused.
func TestLandingUpcomingIsOpenWithoutCredentials(t *testing.T) {
	db := rbacDB(t)
	now := time.Now().UTC()
	publicElection(t, db, "listed", models.ElectionStatusDraft, now.Add(24*time.Hour), true)

	client := testkit.New(t, landingEngine(t, db))
	client.Get("/api/v1/public/landing").AssertStatus(http.StatusOK)
	client.Get("/api/v1/public/elections/upcoming").AssertStatus(http.StatusOK)
}
