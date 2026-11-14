package handlers_test

import (
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"campvote/apps/api/internal/handlers"
	"campvote/apps/api/internal/middleware"
	"campvote/apps/api/internal/models"
	"campvote/apps/api/internal/policy"
	"campvote/apps/api/internal/testkit"
)

func residenceEngine(t *testing.T, db *gorm.DB) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	auth := testkit.Auth(testkit.Config(), db)
	users := &handlers.UserHandler{DB: db}
	areas := handlers.NewResidenceHandler(db)

	r := gin.New()
	r.Use(policy.Declared())

	protected := r.Group("")
	protected.Use(middleware.APIKeyOrAuth(db, middleware.Auth(db, auth)))
	protected.Use(policy.Guard(db))
	protected.PUT("/api/v1/profile", users.UpdateProfile)
	protected.GET("/api/v1/residence-areas", areas.List)

	staff := r.Group("")
	staff.Use(middleware.APIKeyOrAuth(db, middleware.Auth(db, auth)))
	staff.Use(policy.Guard(db))
	staff.POST("/api/v1/residence-areas", areas.Create)
	staff.DELETE("/api/v1/residence-areas/:id", areas.Deactivate)
	staff.PUT("/api/v1/residence-assignments", areas.CorrectResidence)
	return r
}

// clearProfile undoes testkit's complete profile for a test that needs to
// exercise the first-time form.
func clearProfile(t *testing.T, db *gorm.DB, user *models.User) {
	t.Helper()
	if err := db.Model(&models.User{}).Where("id = ?", user.ID).
		Updates(map[string]any{"age": nil, "sex": "", "residence_area_id": nil}).Error; err != nil {
		t.Fatal(err)
	}
	user.Age, user.Sex, user.ResidenceAreaID = nil, "", nil
}

func TestProfileIsSetOnceAndValidated(t *testing.T) {
	db := rbacDB(t)
	client := testkit.New(t, residenceEngine(t, db))
	admin := testkit.User(t, db, "areas-admin@must.ac.ug", models.RoleAdmin)
	adminToken := testkit.SignIn(t, db, testkit.Auth(testkit.Config(), db), admin)

	var area models.ResidenceArea
	client.As(adminToken).Post("/api/v1/residence-areas", map[string]any{"name": "Hall 3"}).
		AssertCreated().BindData(&area)
	if area.ID == "" || !area.Active {
		t.Fatalf("created area = %+v, want an active area", area)
	}
	client.As(adminToken).Post("/api/v1/residence-areas", map[string]any{"name": "   "}).
		AssertBadRequest()

	voter := testkit.User(t, db, "2022bse020@std.mast.ac.ug", models.RoleStudentVoter)
	token := testkit.SignIn(t, db, testkit.Auth(testkit.Config(), db), voter)
	clearProfile(t, db, voter)

	// Out-of-range and unknown values are refused, and nothing is stored.
	client.As(token).Put("/api/v1/profile", map[string]any{"age": 15}).AssertUnprocessable()
	client.As(token).Put("/api/v1/profile", map[string]any{"age": 101}).AssertUnprocessable()
	client.As(token).Put("/api/v1/profile", map[string]any{"sex": "OTHER"}).AssertUnprocessable()
	client.As(token).Put("/api/v1/profile", map[string]any{"residence_area_id": "missing"}).
		AssertUnprocessable()

	var fresh models.User
	if err := db.First(&fresh, "id = ?", voter.ID).Error; err != nil {
		t.Fatal(err)
	}
	if len(fresh.MissingProfileFields()) != 3 {
		t.Fatalf("a rejected update stored part of the profile: %+v", fresh)
	}

	client.As(token).Put("/api/v1/profile", map[string]any{
		"age": 21, "sex": "female", "residence_area_id": area.ID,
	}).AssertOK()

	if err := db.First(&fresh, "id = ?", voter.ID).Error; err != nil {
		t.Fatal(err)
	}
	if fresh.Age == nil || *fresh.Age != 21 || fresh.Sex != models.SexFemale ||
		fresh.ResidenceAreaID == nil || *fresh.ResidenceAreaID != area.ID {
		t.Fatalf("stored profile = %+v", fresh)
	}

	// Age, sex and residence are set once. A wrong residence is corrected by
	// an administrator, not by re-saving the profile.
	client.As(token).Put("/api/v1/profile", map[string]any{"age": 22}).AssertUnprocessable()
	client.As(token).Put("/api/v1/profile", map[string]any{"sex": "MALE"}).AssertUnprocessable()
	client.As(token).Put("/api/v1/profile", map[string]any{"residence_area_id": area.ID}).
		AssertOK()
}

func TestDeactivatingAResidenceClearsProfiles(t *testing.T) {
	db := rbacDB(t)
	client := testkit.New(t, residenceEngine(t, db))
	admin := testkit.User(t, db, "areas-admin2@must.ac.ug", models.RoleAdmin)
	adminToken := testkit.SignIn(t, db, testkit.Auth(testkit.Config(), db), admin)

	var area models.ResidenceArea
	client.As(adminToken).Post("/api/v1/residence-areas", map[string]any{"name": "Hall 4"}).
		AssertCreated().BindData(&area)

	voter := testkit.User(t, db, "2022bse021@std.mast.ac.ug", models.RoleStudentVoter)
	token := testkit.SignIn(t, db, testkit.Auth(testkit.Config(), db), voter)
	clearProfile(t, db, voter)
	client.As(token).Put("/api/v1/profile", map[string]any{
		"age": 20, "sex": "MALE", "residence_area_id": area.ID,
	}).AssertOK()

	var listed []models.ResidenceArea
	client.As(token).Get("/api/v1/residence-areas").AssertOK().BindData(&listed)
	if len(listed) != 1 {
		t.Fatalf("listed %d areas, want 1", len(listed))
	}

	client.As(adminToken).Delete("/api/v1/residence-areas/" + area.ID).AssertOK()

	var stored models.ResidenceArea
	if err := db.First(&stored, "id = ?", area.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Active {
		t.Error("the area is still active after removal")
	}
	var fresh models.User
	if err := db.First(&fresh, "id = ?", voter.ID).Error; err != nil {
		t.Fatal(err)
	}
	if fresh.ResidenceAreaID != nil {
		t.Error("the profile still points at a deactivated area")
	}
	client.As(token).Get("/api/v1/residence-areas").AssertOK().BindData(&listed)
	if len(listed) != 0 {
		t.Errorf("an inactive area is still offered: %+v", listed)
	}
}

func TestAdministratorCorrectsOnlyTheResidence(t *testing.T) {
	db := rbacDB(t)
	client := testkit.New(t, residenceEngine(t, db))
	admin := testkit.User(t, db, "areas-admin3@must.ac.ug", models.RoleAdmin)
	adminToken := testkit.SignIn(t, db, testkit.Auth(testkit.Config(), db), admin)

	var area models.ResidenceArea
	client.As(adminToken).Post("/api/v1/residence-areas", map[string]any{"name": "Hall 5"}).
		AssertCreated().BindData(&area)

	voter := testkit.User(t, db, "2022bse022@std.mast.ac.ug", models.RoleStudentVoter)
	token := testkit.SignIn(t, db, testkit.Auth(testkit.Config(), db), voter)
	clearProfile(t, db, voter)
	client.As(token).Put("/api/v1/profile", map[string]any{
		"age": 24, "sex": "FEMALE",
	}).AssertOK()

	client.As(adminToken).Put("/api/v1/residence-assignments", map[string]any{
		"email": voter.Email, "residence_area_id": area.ID,
	}).AssertOK()

	var fresh models.User
	if err := db.First(&fresh, "id = ?", voter.ID).Error; err != nil {
		t.Fatal(err)
	}
	if fresh.ResidenceAreaID == nil || *fresh.ResidenceAreaID != area.ID {
		t.Fatalf("residence not corrected: %+v", fresh)
	}
	if fresh.Age == nil || *fresh.Age != 24 || fresh.Sex != models.SexFemale {
		t.Error("the correction changed age or sex")
	}

	client.As(adminToken).Put("/api/v1/residence-assignments", map[string]any{
		"email": "nobody@std.mast.ac.ug", "residence_area_id": area.ID,
	}).AssertNotFound()
}
