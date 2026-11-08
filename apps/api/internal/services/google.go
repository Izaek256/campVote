package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"google.golang.org/api/idtoken"
)

// ErrGoogleNotConfigured means this deployment has no Google OAuth client, so
// no ID token can be accepted. The handler turns it into a 503 with an
// operator-facing message; it is never a fallback to another sign-in method.
var ErrGoogleNotConfigured = errors.New("Google sign-in is not configured")

// GoogleIdentity is the part of a verified Google ID token this application
// uses. Every field comes from a token whose signature, issuer, audience and
// expiry Google's library has already checked; nothing here is trusted from
// the browser.
type GoogleIdentity struct {
	Subject       string
	Email         string
	EmailVerified bool
	HostedDomain  string
	FirstName     string
	LastName      string
	Name          string
	Picture       string
}

// GoogleVerifier verifies a Google ID token. The production implementation is
// IDTokenVerifier; tests replace it at this boundary so they never call Google.
type GoogleVerifier interface {
	Verify(ctx context.Context, rawToken string) (*GoogleIdentity, error)
}

// IDTokenVerifier validates Google Identity Services ID tokens with Google's
// supported library. idtoken.Validate checks the signature against Google's
// published keys, the issuer, the audience and the expiry before returning any
// claim.
type IDTokenVerifier struct {
	ClientID string

	// validate is idtoken.Validate in production. A field, not a package call,
	// so a test can exercise the claim mapping without network access.
	validate func(ctx context.Context, idToken, audience string) (*idtoken.Payload, error)
}

// NewGoogleVerifier returns a verifier for one OAuth client ID.
func NewGoogleVerifier(clientID string) *IDTokenVerifier {
	return &IDTokenVerifier{ClientID: strings.TrimSpace(clientID), validate: idtoken.Validate}
}

// Verify returns the verified identity, or an error that says nothing about
// why the token failed. The caller logs the detail; the browser gets a generic
// refusal so token probing learns nothing.
func (v *IDTokenVerifier) Verify(ctx context.Context, rawToken string) (*GoogleIdentity, error) {
	if v == nil || v.ClientID == "" {
		return nil, ErrGoogleNotConfigured
	}
	rawToken = strings.TrimSpace(rawToken)
	if rawToken == "" {
		return nil, errors.New("empty Google ID token")
	}

	validate := v.validate
	if validate == nil {
		validate = idtoken.Validate
	}
	payload, err := validate(ctx, rawToken, v.ClientID)
	if err != nil {
		return nil, fmt.Errorf("verifying Google ID token: %w", err)
	}
	if payload == nil {
		return nil, errors.New("verifying Google ID token: empty payload")
	}

	identity := &GoogleIdentity{
		Subject:       strings.TrimSpace(payload.Subject),
		Email:         strings.ToLower(strings.TrimSpace(claimString(payload.Claims, "email"))),
		EmailVerified: claimBool(payload.Claims, "email_verified"),
		HostedDomain:  strings.ToLower(strings.TrimSpace(claimString(payload.Claims, "hd"))),
		FirstName:     strings.TrimSpace(claimString(payload.Claims, "given_name")),
		LastName:      strings.TrimSpace(claimString(payload.Claims, "family_name")),
		Name:          strings.TrimSpace(claimString(payload.Claims, "name")),
		Picture:       strings.TrimSpace(claimString(payload.Claims, "picture")),
	}
	if identity.Subject == "" {
		return nil, errors.New("verifying Google ID token: missing subject")
	}
	return identity, nil
}

func claimString(claims map[string]interface{}, name string) string {
	if claims == nil {
		return ""
	}
	value, ok := claims[name]
	if !ok || value == nil {
		return ""
	}
	if s, ok := value.(string); ok {
		return s
	}
	return fmt.Sprint(value)
}

// claimBool accepts both shapes Google has used for email_verified: a JSON
// boolean and the string "true".
func claimBool(claims map[string]interface{}, name string) bool {
	if claims == nil {
		return false
	}
	switch value := claims[name].(type) {
	case bool:
		return value
	case string:
		return strings.EqualFold(value, "true")
	default:
		return false
	}
}
