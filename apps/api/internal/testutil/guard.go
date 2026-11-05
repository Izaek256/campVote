package testutil

import (
	"fmt"
	"net/url"
	"os"
	"strings"
)

var allowedHosts = map[string]bool{
	"localhost":   true,
	"127.0.0.1":   true,
	"postgres":    true,
}

// RequireTestDB aborts unless TEST_DATABASE_URL points at a disposable test DB.
func RequireTestDB() error {
	u := os.Getenv("TEST_DATABASE_URL")
	if u == "" {
		return fmt.Errorf("TEST_DATABASE_URL is required")
	}
	parsed, err := url.Parse(u)
	if err != nil {
		return fmt.Errorf("invalid TEST_DATABASE_URL: %w", err)
	}
	host := parsed.Hostname()
	if !allowedHosts[host] {
		return fmt.Errorf("refusing to run tests against host %q", host)
	}
	name := strings.TrimPrefix(parsed.Path, "/")
	if os.Getenv("CAMPVOTE_TEST_DB") != "1" && !strings.HasPrefix(name, "campvote_test_") {
		return fmt.Errorf("refusing to run tests against db %q (name must start with campvote_test_ or set CAMPVOTE_TEST_DB=1)", name)
	}
	return nil
}

