package routes

import "testing"

// Google sign-in mints a session, so it gets the same flood control as the
// password and OTP sign-in routes. A missing entry is a public endpoint with
// no limit at all, which is the failure this test exists to catch.
func TestGoogleSignInHasRateLimit(t *testing.T) {
	for _, dev := range []bool{false, true} {
		limits := authRouteLimits(dev)
		limit, ok := limits["/api/v1/auth/google"]
		if !ok {
			t.Fatalf("dev=%v: /api/v1/auth/google has no rate limit", dev)
		}
		if limit.Requests <= 0 || limit.Window <= 0 {
			t.Fatalf("dev=%v: invalid limit %+v", dev, limit)
		}
	}
}
