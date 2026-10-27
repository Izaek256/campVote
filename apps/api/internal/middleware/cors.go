package middleware

import (
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

// refusedOrigins remembers which origins have been reported already. A page
// that retries would otherwise write the same line on every request.
var refusedOrigins sync.Map

// reportRefusedOrigin says, once per origin, that the browser was turned away.
//
// A request from an origin outside the allowlist gets a preflight with no
// Access-Control-Allow-Origin, which is the correct answer and an invisible
// one: the browser never sends the real request, so no handler runs, no status
// is logged, and the screen shows a button that does nothing. Moving the admin
// to a different port is enough to produce it.
func reportRefusedOrigin(origin string) {
	if origin == "" {
		return
	}
	if _, seen := refusedOrigins.LoadOrStore(origin, struct{}{}); seen {
		return
	}
	log.Printf("CORS: refused %s. The browser drops every request from that origin without reporting an error, so the page will look like it is doing nothing. Add it to CORS_ORIGINS in .env (comma separated) and restart.", origin)
}

// isWailsOrigin reports whether the request came from the Wails desktop
// webview, whose origin is not stably enumerable:
//
//	wails dev   (Windows)     http://wails.localhost:34115   <- port from wails.json
//	wails build (Windows)     http://wails.localhost
//	wails build (mac/linux)   wails://wails
//
// The dev-server port is configurable, so pinning exact origins in
// CORS_ORIGINS is fragile: change the port and the desktop login silently
// starts failing with an opaque "Network Error". Match on the host instead.
//
// Safe by construction: "wails.localhost" is a virtual host the webview
// resolves internally, so a page on the public internet cannot be served
// from it and cannot forge this origin. Every other origin still has to be
// in the explicit CORS_ORIGINS allowlist.
func isWailsOrigin(origin string) bool {
	if origin == "wails://wails" {
		return true
	}
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}
	return u.Hostname() == "wails.localhost"
}

// allowedRequestHeaders is what the browser is told the page may send.
//
// A header missing from this list is not refused with an error: it is stripped
// during preflight, so the request never arrives, no handler runs, nothing is
// logged, and the same call works perfectly under curl. That is the worst shape
// a bug can have, and it is why this is a list anything can add to rather than
// a string literal somebody has to remember to edit.
//
// X-CSRF-Token and Idempotency-Key are injected by the web and admin axios
// clients on every unsafe method; without them a request either fails the
// AutoCSRF check or replays with no idempotency guarantee. Authorization stays
// for native bearer clients. X-API-Key is how a storefront calls the public
// endpoints cross-origin.
var allowedRequestHeaders = []string{
	"Origin",
	"Content-Type",
	"Accept",
	"Authorization",
	"X-API-Key",
	"X-CSRF-Token",
	"Idempotency-Key",
	"X-Public-IP-Hint",
}

// allowedRequestHeadersValue is the joined header value, built once.
var allowedRequestHeadersValue = strings.Join(allowedRequestHeaders, ", ")

// AllowRequestHeader adds a header to that list.
//
// Call it from an init() in the file that introduces the header, so the feature
// and its CORS entry arrive and leave together instead of the second one being
// a thing to remember. internal/middleware/tenant.go does exactly this for
// X-Organization-ID, which is why removing the multitenant plugin also removes
// the header without touching this file.
//
// init() only. Package-level variables are initialised before any init() runs,
// and every init() finishes before main, so by the time a request can read
// allowedRequestHeadersValue the list is final and nothing synchronises it.
// Calling this from a handler would be a data race.
func AllowRequestHeader(name string) {
	name = strings.TrimSpace(name)
	if name == "" {
		return
	}
	for _, existing := range allowedRequestHeaders {
		if strings.EqualFold(existing, name) {
			return
		}
	}
	allowedRequestHeaders = append(allowedRequestHeaders, name)
	allowedRequestHeadersValue = strings.Join(allowedRequestHeaders, ", ")
}

// CORS creates a CORS middleware with a fixed allowlist.
//
// Kept for callers that genuinely have a static list. Anything user-facing
// should use CORSDynamic, so adding a domain does not need a redeploy.
func CORS(allowedOrigins []string) gin.HandlerFunc {
	return CORSDynamic(func() []string { return allowedOrigins })
}

// CORSDynamic resolves the allowlist per request.
//
// Per request rather than at construction, because the point of putting
// origins in settings is that somebody adds a domain at 9pm and it works.
// Capturing the slice at boot would mean the setting existed and did nothing
// until the next deploy, which is worse than not offering it.
//
// The cost is a map build per request over a list with single digits of
// entries, against a settings store that is already cached in memory. That is
// not the thing to optimise.
func CORSDynamic(resolve func() []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		allowed := false
		for _, candidate := range resolve() {
			if candidate == origin && origin != "" {
				allowed = true
				break
			}
		}
		if allowed || isWailsOrigin(origin) {
			c.Header("Access-Control-Allow-Origin", origin)
		} else {
			reportRefusedOrigin(origin)
		}

		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", allowedRequestHeadersValue)
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
