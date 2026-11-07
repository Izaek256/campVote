package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"campvote/apps/api/internal/config"
	"campvote/apps/api/internal/migrations"
	"campvote/apps/api/internal/models"
	"campvote/apps/api/internal/services"
)

// fakeGoogleVerifier stands in for idtoken.Validate. The production verifier's
// job is to call Google's library; the handler's job is what happens after it
// answers, which is what these tests cover without network access.
type fakeGoogleVerifier struct {
	identity *services.GoogleIdentity
	err      error
	calls    int
}

func (f *fakeGoogleVerifier) Verify(context.Context, string) (*services.GoogleIdentity, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	return f.identity, nil
}

func googleTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)

	require.NoError(t, db.AutoMigrate(
		&models.User{},
		&models.Session{},
		&models.UserActivity{},
	))
	// Production gets this index from models.Migrate. The test database is
	// built table by table, so it asks for the same constraint explicitly.
	require.NoError(t, migrations.AddGoogleIDUniqueIndex(db))
	return db
}

func googleCfg() *config.Config {
	cfg := testCfg()
	cfg.GoogleOAuthClientID = "test-client.apps.googleusercontent.com"
	return cfg
}

func googleIdentity() *services.GoogleIdentity {
	return &services.GoogleIdentity{
		Subject:       "google-sub-1",
		Email:         "ada@example.com",
		EmailVerified: true,
		HostedDomain:  "example.com",
		FirstName:     "Ada",
		LastName:      "Lovelace",
		Name:          "Ada Lovelace",
		Picture:       "https://example.com/ada.png",
	}
}

func googleRouter(h *AuthHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/auth/google", h.GoogleSignIn)
	return r
}

func googleHandler(cfg *config.Config, db *gorm.DB, verifier services.GoogleVerifier) *AuthHandler {
	return &AuthHandler{
		DB:             db,
		AuthService:    newTestAuthSvc(cfg, db),
		Config:         cfg,
		GoogleVerifier: verifier,
	}
}

func postGoogle(t *testing.T, h *AuthHandler, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/google", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	googleRouter(h).ServeHTTP(w, req)
	return w
}

func TestGoogleSignIn_CreatesStudentAndSession(t *testing.T) {
	db := googleTestDB(t)
	cfg := googleCfg()
	verifier := &fakeGoogleVerifier{identity: googleIdentity()}
	h := googleHandler(cfg, db, verifier)

	w := postGoogle(t, h, `{"id_token":"a-google-id-token"}`)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	assert.Equal(t, 1, verifier.calls)

	var body authResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "ada@example.com", body.Data.User.Email)
	assert.Equal(t, models.RoleStudentVoter, body.Data.User.Role)
	assert.True(t, body.Data.User.Active)
	require.NotNil(t, body.Data.User.EmailVerifiedAt)
	assert.Empty(t, body.Data.User.Password, "a Google-created account has no password to leak")
	assert.NotEmpty(t, body.Data.Tokens.AccessToken)
	assert.NotEmpty(t, body.Data.Tokens.RefreshToken)

	// google_id is json:"-" like github_id and password: internal identity is
	// not part of the API surface, so the stored row is where it is checked.
	var stored models.User
	require.NoError(t, db.First(&stored, "id = ?", body.Data.User.ID).Error)
	assert.Equal(t, "google-sub-1", stored.GoogleID)

	var sessions int64
	require.NoError(t, db.Model(&models.Session{}).Count(&sessions).Error)
	assert.EqualValues(t, 1, sessions, "signing in must create a revocable session")

	// assert.Contains on a []string compares whole elements, and each element
	// is a full Set-Cookie line. Join them so the check is a substring match.
	cookies := strings.Join(w.Header().Values("Set-Cookie"), "\n")
	assert.Contains(t, cookies, "grit_access=")
	assert.Contains(t, cookies, "grit_refresh=")
}

func TestGoogleSignIn_RepeatedSignInIsIdempotent(t *testing.T) {
	db := googleTestDB(t)
	h := googleHandler(googleCfg(), db, &fakeGoogleVerifier{identity: googleIdentity()})

	first := postGoogle(t, h, `{"id_token":"token"}`)
	require.Equal(t, http.StatusCreated, first.Code)

	second := postGoogle(t, h, `{"id_token":"token"}`)
	require.Equal(t, http.StatusOK, second.Code)

	var users int64
	require.NoError(t, db.Model(&models.User{}).Count(&users).Error)
	assert.EqualValues(t, 1, users, "a repeated subject must not create a second account")
}

func TestGoogleSignIn_RejectsBadInputAndUnverifiedClaims(t *testing.T) {
	cases := []struct {
		name       string
		cfg        func(*config.Config)
		verifier   services.GoogleVerifier
		body       string
		wantStatus int
	}{
		{
			name:       "missing configuration",
			cfg:        func(c *config.Config) { c.GoogleOAuthClientID = "" },
			verifier:   &fakeGoogleVerifier{identity: googleIdentity()},
			body:       `{"id_token":"token"}`,
			wantStatus: http.StatusServiceUnavailable,
		},
		{
			name:       "missing verifier",
			verifier:   nil,
			body:       `{"id_token":"token"}`,
			wantStatus: http.StatusServiceUnavailable,
		},
		{
			name:       "missing token",
			verifier:   &fakeGoogleVerifier{identity: googleIdentity()},
			body:       `{}`,
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "malformed body",
			verifier:   &fakeGoogleVerifier{identity: googleIdentity()},
			body:       `{"id_token":`,
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:       "verification failure",
			verifier:   &fakeGoogleVerifier{err: errors.New("signature is invalid")},
			body:       `{"id_token":"bad"}`,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "unverified email",
			verifier:   &fakeGoogleVerifier{identity: &services.GoogleIdentity{Subject: "sub", Email: "ada@example.com", EmailVerified: false}},
			body:       `{"id_token":"token"}`,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "missing email",
			verifier:   &fakeGoogleVerifier{identity: &services.GoogleIdentity{Subject: "sub", EmailVerified: true}},
			body:       `{"id_token":"token"}`,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "not a student address",
			verifier:   &fakeGoogleVerifier{identity: &services.GoogleIdentity{Subject: "sub", Email: "ada@gmail.com", EmailVerified: true}},
			body:       `{"id_token":"token"}`,
			wantStatus: http.StatusForbidden,
		},
		{
			name: "hosted domain mismatch",
			cfg: func(c *config.Config) {
				c.GoogleOAuthHostedDomain = "must.ac.ug"
			},
			verifier:   &fakeGoogleVerifier{identity: &services.GoogleIdentity{Subject: "sub", Email: "ada@example.com", EmailVerified: true, HostedDomain: "evil.example"}},
			body:       `{"id_token":"token"}`,
			wantStatus: http.StatusForbidden,
		},
		{
			name: "hosted domain absent when required",
			cfg: func(c *config.Config) {
				c.GoogleOAuthHostedDomain = "must.ac.ug"
			},
			verifier:   &fakeGoogleVerifier{identity: &services.GoogleIdentity{Subject: "sub", Email: "ada@example.com", EmailVerified: true}},
			body:       `{"id_token":"token"}`,
			wantStatus: http.StatusForbidden,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := googleTestDB(t)
			cfg := googleCfg()
			if tc.cfg != nil {
				tc.cfg(cfg)
			}
			h := googleHandler(cfg, db, tc.verifier)

			w := postGoogle(t, h, tc.body)
			assert.Equal(t, tc.wantStatus, w.Code, w.Body.String())

			var users int64
			require.NoError(t, db.Model(&models.User{}).Count(&users).Error)
			assert.Zero(t, users, "a refused sign-in must not create an account")
		})
	}
}

func TestGoogleSignIn_LinksExistingVerifiedAccountWithoutChangingSecurityFields(t *testing.T) {
	db := googleTestDB(t)
	verified := time.Now().Add(-time.Hour).UTC()
	existing := models.User{
		FirstName:       "Ada",
		LastName:        "Lovelace",
		Email:           "ada@example.com",
		Password:        "existing-bcrypt-hash",
		Role:            models.RoleStudentVoter,
		Active:          true,
		Provider:        "local",
		EmailVerifiedAt: &verified,
	}
	require.NoError(t, db.Create(&existing).Error)
	// User.BeforeCreate bcrypt-hashes whatever password it is given, so the row
	// already holds a hash. That hash is what the link must leave untouched.
	passwordBeforeLink := existing.Password

	h := googleHandler(googleCfg(), db, &fakeGoogleVerifier{identity: googleIdentity()})
	w := postGoogle(t, h, `{"id_token":"token"}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var stored models.User
	require.NoError(t, db.First(&stored, "id = ?", existing.ID).Error)
	assert.Equal(t, existing.ID, stored.ID)
	assert.Equal(t, "google-sub-1", stored.GoogleID)
	assert.Equal(t, passwordBeforeLink, stored.Password, "a verified account keeps its password")
	assert.Equal(t, models.RoleStudentVoter, stored.Role)
	assert.Equal(t, "google", stored.Provider)

	var users int64
	require.NoError(t, db.Model(&models.User{}).Count(&users).Error)
	assert.EqualValues(t, 1, users)
}

func TestGoogleSignIn_UnverifiedExistingAccountLosesPasswordAndSessions(t *testing.T) {
	db := googleTestDB(t)
	existing := models.User{
		FirstName: "Ada",
		LastName:  "Lovelace",
		Email:     "ada@example.com",
		Password:  "attacker-chosen-hash",
		Role:      models.RoleStudentVoter,
		Active:    true,
	}
	require.NoError(t, db.Create(&existing).Error)
	require.NoError(t, db.Create(&models.Session{
		UserID:     existing.ID,
		TokenHash:  models.HashSessionToken("old-refresh-token"),
		LastSeenAt: time.Now(),
		ExpiresAt:  time.Now().Add(time.Hour),
	}).Error)

	h := googleHandler(googleCfg(), db, &fakeGoogleVerifier{identity: googleIdentity()})
	w := postGoogle(t, h, `{"id_token":"token"}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var stored models.User
	require.NoError(t, db.First(&stored, "id = ?", existing.ID).Error)
	assert.Empty(t, stored.Password, "Google proved ownership; the old password must stop working")
	require.NotNil(t, stored.EmailVerifiedAt)
	assert.Equal(t, "google-sub-1", stored.GoogleID)

	// The handler opens a fresh session for this sign-in, which is correct, so
	// the check is on the session the unverified account already held.
	var oldLive int64
	require.NoError(t, db.Model(&models.Session{}).
		Where("token_hash = ? AND revoked_at IS NULL", models.HashSessionToken("old-refresh-token")).
		Count(&oldLive).Error)
	assert.Zero(t, oldLive, "sessions opened before the address was verified must end")
}

func TestGoogleSignIn_RefusesConflictingLinkAndDisabledAccount(t *testing.T) {
	t.Run("email already linked to another subject", func(t *testing.T) {
		db := googleTestDB(t)
		require.NoError(t, db.Create(&models.User{
			FirstName: "Ada", LastName: "L", Email: "ada@example.com",
			Role: models.RoleStudentVoter, Active: true, GoogleID: "someone-else",
		}).Error)

		h := googleHandler(googleCfg(), db, &fakeGoogleVerifier{identity: googleIdentity()})
		w := postGoogle(t, h, `{"id_token":"token"}`)
		assert.Equal(t, http.StatusConflict, w.Code, w.Body.String())
	})

	t.Run("disabled account", func(t *testing.T) {
		db := googleTestDB(t)
		require.NoError(t, db.Create(&models.User{
			FirstName: "Ada", LastName: "L", Email: "ada@example.com",
			Role: models.RoleStudentVoter, Active: false,
		}).Error)

		h := googleHandler(googleCfg(), db, &fakeGoogleVerifier{identity: googleIdentity()})
		w := postGoogle(t, h, `{"id_token":"token"}`)
		assert.Equal(t, http.StatusForbidden, w.Code, w.Body.String())
	})
}

func TestGoogleSignIn_StaffSignsInOnlyIntoAnExistingAccount(t *testing.T) {
	// Distinct domains so the two policies can be told apart: the staff
	// address must never be self-registered.
	staffCfg := func() *config.Config {
		cfg := googleCfg()
		cfg.StudentEmailDomain = "students.example.com"
		cfg.AdminEmailDomain = "example.com"
		return cfg
	}
	staffIdentity := func() *services.GoogleIdentity {
		return &services.GoogleIdentity{
			Subject:       "google-staff-1",
			Email:         "registrar@example.com",
			EmailVerified: true,
			HostedDomain:  "example.com",
			FirstName:     "Grace",
			LastName:      "Hopper",
		}
	}

	t.Run("existing administrator is linked, role and password untouched", func(t *testing.T) {
		db := googleTestDB(t)
		verified := time.Now().Add(-time.Hour).UTC()
		existing := models.User{
			FirstName: "Grace", LastName: "Hopper", Email: "registrar@example.com",
			Password: "bootstrap-password", Role: models.RoleAdmin, Active: true,
			EmailVerifiedAt: &verified,
		}
		require.NoError(t, db.Create(&existing).Error)
		passwordBeforeLink := existing.Password

		h := googleHandler(staffCfg(), db, &fakeGoogleVerifier{identity: staffIdentity()})
		w := postGoogle(t, h, `{"id_token":"token"}`)
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())

		var stored models.User
		require.NoError(t, db.First(&stored, "id = ?", existing.ID).Error)
		assert.Equal(t, models.RoleAdmin, stored.Role, "a token must never change a role")
		assert.Equal(t, "google-staff-1", stored.GoogleID)
		assert.Equal(t, passwordBeforeLink, stored.Password)
	})

	t.Run("unknown staff address is refused and never created", func(t *testing.T) {
		db := googleTestDB(t)
		h := googleHandler(staffCfg(), db, &fakeGoogleVerifier{identity: staffIdentity()})
		w := postGoogle(t, h, `{"id_token":"token"}`)
		assert.Equal(t, http.StatusForbidden, w.Code, w.Body.String())

		var users int64
		require.NoError(t, db.Model(&models.User{}).Count(&users).Error)
		assert.Zero(t, users, "staff and administrator accounts are created by an administrator")
	})

	t.Run("student address still self-registers", func(t *testing.T) {
		db := googleTestDB(t)
		identity := staffIdentity()
		identity.Email = "ada@students.example.com"

		h := googleHandler(staffCfg(), db, &fakeGoogleVerifier{identity: identity})
		w := postGoogle(t, h, `{"id_token":"token"}`)
		require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

		var stored models.User
		require.NoError(t, db.First(&stored, "email = ?", "ada@students.example.com").Error)
		assert.Equal(t, models.RoleStudentVoter, stored.Role)
	})
}

func TestGoogleSignIn_SubjectWinsWhenGoogleEmailChanges(t *testing.T) {
	db := googleTestDB(t)
	existing := models.User{
		FirstName: "Ada", LastName: "L", Email: "old@example.com",
		Role: models.RoleStudentVoter, Active: true, GoogleID: "google-sub-1",
	}
	require.NoError(t, db.Create(&existing).Error)

	identity := googleIdentity()
	identity.Email = "new@example.com"
	h := googleHandler(googleCfg(), db, &fakeGoogleVerifier{identity: identity})

	w := postGoogle(t, h, `{"id_token":"token"}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var users int64
	require.NoError(t, db.Model(&models.User{}).Count(&users).Error)
	assert.EqualValues(t, 1, users, "the immutable subject keeps the same account")

	var stored models.User
	require.NoError(t, db.First(&stored, "id = ?", existing.ID).Error)
	assert.Equal(t, "old@example.com", stored.Email, "a renamed Google address must not silently move the account")
}
