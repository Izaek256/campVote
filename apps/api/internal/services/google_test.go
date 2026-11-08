package services

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/idtoken"
)

func TestIDTokenVerifier_MapsVerifiedClaims(t *testing.T) {
	verifier := &IDTokenVerifier{
		ClientID: "client.apps.googleusercontent.com",
		validate: func(_ context.Context, token, audience string) (*idtoken.Payload, error) {
			assert.Equal(t, "raw-token", token)
			assert.Equal(t, "client.apps.googleusercontent.com", audience)
			return &idtoken.Payload{
				Subject: "google-sub-1",
				Claims: map[string]interface{}{
					"email":          "Ada@Example.com",
					"email_verified": true,
					"hd":             "Example.com",
					"given_name":     "Ada",
					"family_name":    "Lovelace",
					"name":           "Ada Lovelace",
					"picture":        "https://example.com/ada.png",
				},
			}, nil
		},
	}

	identity, err := verifier.Verify(context.Background(), " raw-token ")
	require.NoError(t, err)
	assert.Equal(t, "google-sub-1", identity.Subject)
	assert.Equal(t, "ada@example.com", identity.Email)
	assert.True(t, identity.EmailVerified)
	assert.Equal(t, "example.com", identity.HostedDomain)
	assert.Equal(t, "Ada", identity.FirstName)
	assert.Equal(t, "Lovelace", identity.LastName)
}

func TestIDTokenVerifier_AcceptsStringEmailVerified(t *testing.T) {
	verifier := &IDTokenVerifier{
		ClientID: "client",
		validate: func(context.Context, string, string) (*idtoken.Payload, error) {
			return &idtoken.Payload{
				Subject: "sub",
				Claims:  map[string]interface{}{"email": "a@example.com", "email_verified": "true"},
			}, nil
		},
	}
	identity, err := verifier.Verify(context.Background(), "token")
	require.NoError(t, err)
	assert.True(t, identity.EmailVerified)
}

func TestIDTokenVerifier_RejectsMissingConfigurationAndSubject(t *testing.T) {
	_, err := (&IDTokenVerifier{}).Verify(context.Background(), "token")
	assert.ErrorIs(t, err, ErrGoogleNotConfigured)

	verifier := &IDTokenVerifier{
		ClientID: "client",
		validate: func(context.Context, string, string) (*idtoken.Payload, error) {
			return &idtoken.Payload{Claims: map[string]interface{}{"email": "a@example.com"}}, nil
		},
	}
	_, err = verifier.Verify(context.Background(), "token")
	assert.Error(t, err)
}

func TestIDTokenVerifier_PropagatesValidationFailure(t *testing.T) {
	verifier := &IDTokenVerifier{
		ClientID: "client",
		validate: func(context.Context, string, string) (*idtoken.Payload, error) {
			return nil, errors.New("token is expired")
		},
	}
	_, err := verifier.Verify(context.Background(), "token")
	assert.Error(t, err)
}
