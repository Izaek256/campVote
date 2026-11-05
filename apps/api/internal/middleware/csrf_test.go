package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// AutoCSRF is the guard the router installs globally. These tests cover the
// three behaviours it relies on: a cookie-authenticated mutation needs the
// double-submit token, a bearer mutation does not, and the token cookie is
// readable by JavaScript but still Secure when the connection is.

func csrfRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(AutoCSRF())
	ok := func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) }
	r.GET("/api/users", ok)
	r.POST("/api/users", ok)
	r.POST("/api/auth/login", ok) // bootstrap route: exempt
	return r
}

func csrfRequest(method, path string, mod func(*http.Request)) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	if mod != nil {
		mod(req)
	}
	w := httptest.NewRecorder()
	csrfRouter().ServeHTTP(w, req)
	return w
}

func TestAutoCSRF_CookieMutationNeedsMatchingHeader(t *testing.T) {
	access := &http.Cookie{Name: "grit_access", Value: "token"}
	csrf := &http.Cookie{Name: "grit_csrf", Value: "csrf-123"}

	missing := csrfRequest(http.MethodPost, "/api/users", func(req *http.Request) {
		req.AddCookie(access)
	})
	assert.Equal(t, http.StatusForbidden, missing.Code)

	mismatch := csrfRequest(http.MethodPost, "/api/users", func(req *http.Request) {
		req.AddCookie(access)
		req.AddCookie(csrf)
		req.Header.Set("X-CSRF-Token", "another-token")
	})
	assert.Equal(t, http.StatusForbidden, mismatch.Code)

	ok := csrfRequest(http.MethodPost, "/api/users", func(req *http.Request) {
		req.AddCookie(access)
		req.AddCookie(csrf)
		req.Header.Set("X-CSRF-Token", csrf.Value)
	})
	assert.Equal(t, http.StatusOK, ok.Code)
}

func TestAutoCSRF_BearerMutationIsExempt(t *testing.T) {
	w := csrfRequest(http.MethodPost, "/api/users", func(req *http.Request) {
		// A native client can hold a stray grit_access cookie from login; the
		// explicit bearer header is what decides this is not a forged request.
		req.AddCookie(&http.Cookie{Name: "grit_access", Value: "token"})
		req.Header.Set("Authorization", "Bearer access-token")
	})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAutoCSRF_BootstrapLoginIsExempt(t *testing.T) {
	w := csrfRequest(http.MethodPost, "/api/auth/login", func(req *http.Request) {
		req.AddCookie(&http.Cookie{Name: "grit_access", Value: "token"})
	})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAutoCSRF_SafeRequestSetsReadableTokenCookie(t *testing.T) {
	w := csrfRequest(http.MethodGet, "/api/users", nil)
	require.Equal(t, http.StatusOK, w.Code)

	var token *http.Cookie
	for _, c := range w.Result().Cookies() {
		if c.Name == "grit_csrf" {
			token = c
		}
	}
	require.NotNil(t, token, "a safe request must bootstrap the CSRF cookie")
	assert.NotEmpty(t, token.Value)
	assert.False(t, token.HttpOnly, "the SPA must be able to read this token")
	assert.Equal(t, http.SameSiteLaxMode, token.SameSite)
	assert.False(t, token.Secure)
}

func TestAutoCSRF_SecureCookieBehindTrustedProxy(t *testing.T) {
	w := csrfRequest(http.MethodGet, "/api/users", func(req *http.Request) {
		req.Header.Set("X-Forwarded-Proto", "https")
	})

	for _, c := range w.Result().Cookies() {
		if c.Name == "grit_csrf" {
			assert.True(t, c.Secure, "the CSRF cookie must be Secure when the proxy terminated TLS")
			return
		}
	}
	t.Fatal("no grit_csrf cookie was set")
}
