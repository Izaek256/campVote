package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"campvote/apps/api/internal/mail/mailtest"
	"campvote/apps/api/internal/middleware"
	"campvote/apps/api/internal/models"
	"campvote/apps/api/internal/respond"
	"campvote/apps/api/internal/services"
)

// The tests in this file exercise the auth hardening from ISSUE-02 over HTTP:
// registration, sign-in, verification, reset, logout, deactivation, cookies and
// brute-force lockout. They drive real gin routes so the middleware and the
// handler answer together, the way a client sees them.

type authResponse struct {
	Data struct {
		User   models.User        `json:"user"`
		Tokens services.TokenPair `json:"tokens"`
	} `json:"data"`
	Message string `json:"message"`
}

func securityRouter(h *AuthHandler, db *gorm.DB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	pub := r.Group("/api/auth")
	pub.POST("/register", h.Register)
	pub.POST("/login", h.Login)
	pub.POST("/refresh", h.Refresh)
	pub.POST("/forgot-password", h.ForgotPassword)
	pub.POST("/reset-password", h.ResetPassword)
	pub.POST("/verify-email", h.VerifyEmail)

	authed := r.Group("/api", middleware.Auth(db, h.AuthService))
	authed.POST("/auth/logout", h.Logout)
	authed.GET("/me", h.Me)

	return r
}

func secRequest(tb testing.TB, r http.Handler, method, path string, body any, mod func(*http.Request)) *httptest.ResponseRecorder {
	tb.Helper()
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		require.NoError(tb, err)
		reader = bytes.NewReader(raw)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if mod != nil {
		mod(req)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func errorCode(tb testing.TB, w *httptest.ResponseRecorder) string {
	tb.Helper()
	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	require.NoError(tb, json.Unmarshal(w.Body.Bytes(), &body))
	return body.Error.Code
}

func loginTokens(tb testing.TB, r http.Handler, email, password string) services.TokenPair {
	tb.Helper()
	w := secRequest(tb, r, http.MethodPost, "/api/auth/login", map[string]string{
		"email":    email,
		"password": password,
	}, nil)
	require.Equal(tb, http.StatusOK, w.Code, "login failed: %s", w.Body.String())

	var resp authResponse
	require.NoError(tb, json.Unmarshal(w.Body.Bytes(), &resp))
	require.NotEmpty(tb, resp.Data.Tokens.RefreshToken)
	return resp.Data.Tokens
}

func cookieNamed(tb testing.TB, w *httptest.ResponseRecorder, name string) *http.Cookie {
	tb.Helper()
	for _, c := range w.Result().Cookies() {
		if c.Name == name {
			return c
		}
	}
	tb.Fatalf("cookie %q not set; got %v", name, w.Result().Cookies())
	return nil
}

func seedVerifiedUser(t *testing.T, db *gorm.DB, email, password string) models.User {
	t.Helper()
	now := time.Now()
	user := models.User{
		FirstName:       "Jane",
		LastName:        "Doe",
		Email:           email,
		Password:        password,
		Role:            models.RoleUser,
		Active:          true,
		EmailVerifiedAt: &now,
	}
	require.NoError(t, db.Create(&user).Error)
	return user
}

func activityCount(t *testing.T, db *gorm.DB, action string) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.Model(&models.UserActivity{}).Where("action = ?", action).Count(&n).Error)
	return n
}

// Registration must store a strong hash and never echo the password back.
func TestRegister_StoresHashAndDoesNotLeakPassword(t *testing.T) {
	const plaintext = "harbour-lamp-97"
	cfg := testCfg()
	db := newTestDB(t)
	h := &AuthHandler{DB: db, AuthService: newTestAuthSvc(cfg, db), Config: cfg}
	r := securityRouter(h, db)

	w := secRequest(t, r, http.MethodPost, "/api/auth/register", map[string]string{
		"first_name": "Jane",
		"last_name":  "Doe",
		"email":      "jane@example.com",
		"password":   plaintext,
	}, nil)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	assert.NotContains(t, w.Body.String(), plaintext, "the response must not echo the password")
	assert.NotContains(t, w.Body.String(), `"password"`, "the user JSON must not carry a password field")

	var user models.User
	require.NoError(t, db.Where("email = ?", "jane@example.com").First(&user).Error)
	assert.NotEqual(t, plaintext, user.Password)
	assert.True(t, strings.HasPrefix(user.Password, "$2"), "expected a bcrypt hash")
	assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(plaintext)))
}

// Browser sessions must keep tokens out of JavaScript's reach and refuse them
// over plain HTTP once TLS (or a trusted proxy saying so) is in play.
func TestAuthCookies_AreHardened(t *testing.T) {
	db := newTestDB(t)
	cfg := testCfg()
	h := &AuthHandler{DB: db, AuthService: newTestAuthSvc(cfg, db), Config: cfg}
	r := securityRouter(h, db)

	// Register rather than seed so the cookies are set by a real sign-in path.
	w := secRequest(t, r, http.MethodPost, "/api/auth/register", map[string]string{
		"first_name": "Jane",
		"last_name":  "Doe",
		"email":      "jane@example.com",
		"password":   "harbour-lamp-97",
	}, func(req *http.Request) { req.Header.Set("X-Forwarded-Proto", "https") })
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	access := cookieNamed(t, w, "grit_access")
	refresh := cookieNamed(t, w, "grit_refresh")

	for _, c := range []*http.Cookie{access, refresh} {
		assert.True(t, c.HttpOnly, "%s must be HttpOnly", c.Name)
		assert.Equal(t, http.SameSiteLaxMode, c.SameSite, "%s must be SameSite=Lax", c.Name)
		assert.True(t, c.Secure, "%s must be Secure behind HTTPS", c.Name)
	}
	assert.Equal(t, "/", access.Path)
	assert.Equal(t, services.RefreshCookiePath, refresh.Path,
		"the refresh cookie must only be sent to the auth routes")
}

// An unknown address, a wrong password and a locked account all get one answer.
func TestLogin_GenericErrorsForUnknownEmailAndWrongPassword(t *testing.T) {
	db := newTestDB(t)
	cfg := testCfg()
	h := &AuthHandler{DB: db, AuthService: newTestAuthSvc(cfg, db), Config: cfg}
	r := securityRouter(h, db)
	seedVerifiedUser(t, db, "jane@example.com", "harbour-lamp-97")

	wrong := secRequest(t, r, http.MethodPost, "/api/auth/login", map[string]string{
		"email": "jane@example.com", "password": "not-the-password",
	}, nil)
	unknown := secRequest(t, r, http.MethodPost, "/api/auth/login", map[string]string{
		"email": "nobody@example.com", "password": "not-the-password",
	}, nil)

	assert.Equal(t, http.StatusUnauthorized, wrong.Code)
	assert.Equal(t, http.StatusUnauthorized, unknown.Code)
	assert.Equal(t, errorCode(t, wrong), errorCode(t, unknown))
	assert.Equal(t, wrong.Body.String(), unknown.Body.String(),
		"the two failures must be indistinguishable")
}

// Repeated wrong passwords lock the account; the correct password is refused
// until the window has passed.
func TestLogin_LocksAfterRepeatedFailuresAndRecoversAfterWindow(t *testing.T) {
	db := newTestDB(t)
	cfg := testCfg()
	cfg.LoginMaxAttempts = 3
	cfg.LoginLockoutWindow = 15 * time.Minute
	h := &AuthHandler{DB: db, AuthService: newTestAuthSvc(cfg, db), Config: cfg}
	r := securityRouter(h, db)
	user := seedVerifiedUser(t, db, "jane@example.com", "harbour-lamp-97")
	// LockAccount also clears pending two-factor tokens, so this table has to
	// exist for the lock (and its audit row) to complete.
	require.NoError(t, db.AutoMigrate(&models.TOTPPendingToken{}))

	for i := 0; i < cfg.LoginMaxAttempts; i++ {
		w := secRequest(t, r, http.MethodPost, "/api/auth/login", map[string]string{
			"email": "jane@example.com", "password": "wrong-password",
		}, nil)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	}

	var locked models.User
	require.NoError(t, db.First(&locked, "id = ?", user.ID).Error)
	require.NotNil(t, locked.LockedUntil, "the account should be locked")
	assert.True(t, locked.LockedUntil.After(time.Now()))
	assert.Equal(t, int64(1), activityCount(t, db, "auth.account_locked"))

	// The right password must not bypass the lock.
	w := secRequest(t, r, http.MethodPost, "/api/auth/login", map[string]string{
		"email": "jane@example.com", "password": "harbour-lamp-97",
	}, nil)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Equal(t, string(respond.CodeInvalidCredentials), errorCode(t, w))

	// Time travel past the window instead of sleeping through it.
	require.NoError(t, db.Model(&models.User{}).Where("id = ?", user.ID).
		Update("locked_until", time.Now().Add(-time.Minute)).Error)

	w = secRequest(t, r, http.MethodPost, "/api/auth/login", map[string]string{
		"email": "jane@example.com", "password": "harbour-lamp-97",
	}, nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var cleared models.User
	require.NoError(t, db.First(&cleared, "id = ?", user.ID).Error)
	assert.Zero(t, cleared.FailedLoginCount, "a completed sign-in must clear the failure count")
	assert.Nil(t, cleared.LockedUntil)
}

// When the verification gate is on, an unverified account cannot sign in; the
// code works once, verifies the address, and replaying it fails.
func TestLogin_RequiresVerifiedEmailAndVerifyCodeIsSingleUse(t *testing.T) {
	db := newTestDB(t)
	cfg := testCfg()
	cfg.RequireEmailVerification = true
	h := &AuthHandler{DB: db, AuthService: newTestAuthSvc(cfg, db), Config: cfg}
	r := securityRouter(h, db)
	seedVerifiedUser(t, db, "unverified@example.com", "harbour-lamp-97")
	require.NoError(t, db.Model(&models.User{}).Where("email = ?", "unverified@example.com").
		Update("email_verified_at", nil).Error)

	refused := secRequest(t, r, http.MethodPost, "/api/auth/login", map[string]string{
		"email": "unverified@example.com", "password": "harbour-lamp-97",
	}, nil)
	require.Equal(t, http.StatusForbidden, refused.Code)
	assert.Equal(t, string(respond.CodeEmailNotVerified), errorCode(t, refused))

	var user models.User
	require.NoError(t, db.Where("email = ?", "unverified@example.com").First(&user).Error)
	code, err := services.GenerateVerificationCode()
	require.NoError(t, err)
	_, err = services.CreateEmailVerificationToken(db, user.ID, user.Email, code, cfg.JWTSecret)
	require.NoError(t, err)

	verified := secRequest(t, r, http.MethodPost, "/api/auth/verify-email", map[string]string{"code": code}, nil)
	require.Equal(t, http.StatusOK, verified.Code, verified.Body.String())

	replay := secRequest(t, r, http.MethodPost, "/api/auth/verify-email", map[string]string{"code": code}, nil)
	assert.Equal(t, http.StatusBadRequest, replay.Code)
	assert.Equal(t, string(respond.CodeInvalidLink), errorCode(t, replay))

	loginTokens(t, r, "unverified@example.com", "harbour-lamp-97")
}

// Verification codes expire. Moving the clock past the stored expiry must be
// refused exactly like a spent or unknown code.
func TestVerifyEmail_ExpiredCodeIsRefused(t *testing.T) {
	db := newTestDB(t)
	cfg := testCfg()
	h := &AuthHandler{DB: db, AuthService: newTestAuthSvc(cfg, db), Config: cfg}
	r := securityRouter(h, db)
	user := seedVerifiedUser(t, db, "jane@example.com", "harbour-lamp-97")
	require.NoError(t, db.Model(&models.User{}).Where("id = ?", user.ID).
		Update("email_verified_at", nil).Error)

	token, err := services.GenerateVerificationCode()
	require.NoError(t, err)
	_, err = services.CreateEmailVerificationToken(db, user.ID, user.Email, token, cfg.JWTSecret)
	require.NoError(t, err)
	require.NoError(t, db.Model(&models.EmailVerificationToken{}).
		Where("user_id = ?", user.ID).
		Update("expires_at", time.Now().Add(-time.Minute)).Error)

	w := secRequest(t, r, http.MethodPost, "/api/auth/verify-email", map[string]string{"code": token}, nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, string(respond.CodeInvalidLink), errorCode(t, w))

	var reloaded models.User
	require.NoError(t, db.First(&reloaded, "id = ?", user.ID).Error)
	assert.Nil(t, reloaded.EmailVerifiedAt)
}

// A reset: the mailed link changes the password, kills every live session, and
// cannot be used twice.
func TestPasswordReset_RevokesSessionsAndCannotBeReplayed(t *testing.T) {
	db := newTestDB(t)
	require.NoError(t, db.AutoMigrate(&models.PasswordResetToken{}))
	cfg := testCfg()
	cfg.OAuthFrontendURL = "http://localhost:3001"
	fake := mailtest.New()
	h := &AuthHandler{DB: db, AuthService: newTestAuthSvc(cfg, db), Config: cfg, Mailer: fake.Mailer()}
	r := securityRouter(h, db)

	user := seedVerifiedUser(t, db, "jane@example.com", "old-harbour-97")
	oldTokens := loginTokens(t, r, "jane@example.com", "old-harbour-97")

	w := secRequest(t, r, http.MethodPost, "/api/auth/forgot-password", map[string]string{
		"email": "jane@example.com",
	}, nil)
	require.Equal(t, http.StatusOK, w.Code)

	msg := fake.AssertSentWithin(t, 5*time.Second, "jane@example.com", "Reset your password")
	match := regexp.MustCompile(`reset-password\?token=([0-9a-f]+)`).FindStringSubmatch(msg.HTML)
	require.Len(t, match, 2, "the reset email must contain a token")
	resetToken := match[1]

	reset := secRequest(t, r, http.MethodPost, "/api/auth/reset-password", map[string]string{
		"token": resetToken, "password": "new-harbour-99",
	}, nil)
	require.Equal(t, http.StatusOK, reset.Code, reset.Body.String())

	var reloaded models.User
	require.NoError(t, db.First(&reloaded, "id = ?", user.ID).Error)
	assert.True(t, reloaded.CheckPassword("new-harbour-99"))
	assert.False(t, reloaded.CheckPassword("old-harbour-97"))

	// Every device is signed out by the reset.
	refresh := secRequest(t, r, http.MethodPost, "/api/auth/refresh", map[string]string{
		"refresh_token": oldTokens.RefreshToken,
	}, nil)
	assert.Equal(t, http.StatusUnauthorized, refresh.Code)
	assert.Equal(t, string(respond.CodeSessionRevoked), errorCode(t, refresh))

	replay := secRequest(t, r, http.MethodPost, "/api/auth/reset-password", map[string]string{
		"token": resetToken, "password": "another-harbour-99",
	}, nil)
	assert.Equal(t, http.StatusBadRequest, replay.Code)
	assert.Equal(t, string(respond.CodeInvalidLink), errorCode(t, replay))

	// The old password no longer signs in; the new one does.
	oldLogin := secRequest(t, r, http.MethodPost, "/api/auth/login", map[string]string{
		"email": "jane@example.com", "password": "old-harbour-97",
	}, nil)
	assert.Equal(t, http.StatusUnauthorized, oldLogin.Code)
	loginTokens(t, r, "jane@example.com", "new-harbour-99")
}

// Logging out with the browser cookie kills the refresh session, not just the
// cookies in this browser.
func TestLogout_RevokesCookieSession(t *testing.T) {
	db := newTestDB(t)
	cfg := testCfg()
	h := &AuthHandler{DB: db, AuthService: newTestAuthSvc(cfg, db), Config: cfg}
	r := securityRouter(h, db)
	seedVerifiedUser(t, db, "jane@example.com", "harbour-lamp-97")

	login := secRequest(t, r, http.MethodPost, "/api/auth/login", map[string]string{
		"email": "jane@example.com", "password": "harbour-lamp-97",
	}, nil)
	require.Equal(t, http.StatusOK, login.Code)
	access := cookieNamed(t, login, "grit_access")
	refresh := cookieNamed(t, login, "grit_refresh")

	logout := secRequest(t, r, http.MethodPost, "/api/auth/logout", nil, func(req *http.Request) {
		req.AddCookie(access)
		req.AddCookie(refresh)
	})
	require.Equal(t, http.StatusOK, logout.Code, logout.Body.String())
	assert.Empty(t, cookieNamed(t, logout, "grit_access").Value, "logout must clear the access cookie")

	reuse := secRequest(t, r, http.MethodPost, "/api/auth/refresh", nil, func(req *http.Request) {
		req.AddCookie(refresh)
	})
	assert.Equal(t, http.StatusUnauthorized, reuse.Code)
	assert.Equal(t, string(respond.CodeSessionRevoked), errorCode(t, reuse))
}

// Native clients never carry the refresh cookie. Logging out with a bearer
// token must still revoke the session it names.
func TestLogout_RevokesBearerSession(t *testing.T) {
	db := newTestDB(t)
	cfg := testCfg()
	h := &AuthHandler{DB: db, AuthService: newTestAuthSvc(cfg, db), Config: cfg}
	r := securityRouter(h, db)
	seedVerifiedUser(t, db, "jane@example.com", "harbour-lamp-97")
	tokens := loginTokens(t, r, "jane@example.com", "harbour-lamp-97")

	logout := secRequest(t, r, http.MethodPost, "/api/auth/logout", nil, func(req *http.Request) {
		req.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	})
	require.Equal(t, http.StatusOK, logout.Code, logout.Body.String())

	reuse := secRequest(t, r, http.MethodPost, "/api/auth/refresh", map[string]string{
		"refresh_token": tokens.RefreshToken,
	}, nil)
	assert.Equal(t, http.StatusUnauthorized, reuse.Code)
	assert.Equal(t, string(respond.CodeSessionRevoked), errorCode(t, reuse))
}

// Deactivating an account kills its sessions immediately and the next request
// with an otherwise-valid access token is refused.
func TestDeactivation_RevokesSessionsAndBlocksNextRequest(t *testing.T) {
	db := newTestDB(t)
	cfg := testCfg()
	authSvc := newTestAuthSvc(cfg, db)
	h := &AuthHandler{DB: db, AuthService: authSvc, Config: cfg}
	r := securityRouter(h, db)
	user := seedVerifiedUser(t, db, "jane@example.com", "harbour-lamp-97")
	tokens := loginTokens(t, r, "jane@example.com", "harbour-lamp-97")

	users := &UserHandler{DB: db, AuthService: authSvc}
	routerWithUserUpdate := gin.New()
	// The route guard sets this on a real request; it decides whether the
	// caller may change an administrator's account.
	routerWithUserUpdate.PUT("/api/users/:id", func(c *gin.Context) {
		c.Set("user_role", models.RoleAdmin)
		users.Update(c)
	})

	w := secRequest(t, routerWithUserUpdate, http.MethodPut, "/api/users/"+user.ID, map[string]any{
		"active": false,
	}, nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	assert.Equal(t, int64(1), activityCount(t, db, "user.deactivated"))

	var live int64
	require.NoError(t, db.Model(&models.Session{}).
		Where("user_id = ? AND revoked_at IS NULL", user.ID).Count(&live).Error)
	assert.Zero(t, live, "deactivation must revoke every live session")

	next := secRequest(t, r, http.MethodGet, "/api/me", nil, func(req *http.Request) {
		req.Header.Set("Authorization", "Bearer "+tokens.AccessToken)
	})
	// Deactivation revokes the session before the disabled-account check runs,
	// so the token is refused as no longer valid. Either way the old token is
	// dead; the middleware's ACCOUNT_DISABLED branch covers rows deactivated
	// outside this handler.
	assert.Equal(t, http.StatusUnauthorized, next.Code)
	assert.Equal(t, string(respond.CodeUnauthorized), errorCode(t, next))
}
