package handlers

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"campvote/apps/api/internal/models"
	"campvote/apps/api/internal/respond"
	"campvote/apps/api/internal/services"
)

// GoogleSignInRequest is the browser's only input: the ID token Google
// Identity Services handed to the page. The backend decodes and verifies it;
// no claim from the browser is trusted.
type GoogleSignInRequest struct {
	IDToken string `json:"id_token" binding:"required,max=8192"`
}

var (
	// errGoogleAccountDisabled is a refusal that must happen before linking:
	// a disabled account must not acquire an identity it can sign in with.
	errGoogleAccountDisabled = errors.New("the account is disabled")
	// errGoogleAccountConflict means the verified address and the Google
	// subject point at different existing accounts. Resolving that is a
	// support decision, never an automatic merge.
	errGoogleAccountConflict = errors.New("the Google identity cannot be linked to this account")
	// errGoogleAccountUnknown means a verified staff address has no account.
	// Staff and administrator accounts are created by an administrator, so
	// signing in with Google must never create one.
	errGoogleAccountUnknown = errors.New("no account exists for this address")
)

// GoogleSignIn verifies a Google Identity Services ID token and signs the
// student in, creating a student voter account on first use.
//
// POST /api/v1/auth/google
func (h *AuthHandler) GoogleSignIn(c *gin.Context) {
	if h.Config == nil || h.Config.GoogleOAuthClientID == "" || h.GoogleVerifier == nil {
		// A clear refusal, not a fallback to password or OTP. A deployment that
		// forgot the client ID should be obvious in the browser and the log.
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": gin.H{
				"code":    "GOOGLE_NOT_CONFIGURED",
				"message": "Google sign-in is not configured on this server. An administrator must set GOOGLE_OAUTH_CLIENT_ID on the API and NEXT_PUBLIC_GOOGLE_CLIENT_ID on the web app.",
			},
		})
		return
	}

	var req GoogleSignInRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Fail(c, respond.CodeValidationError, "A Google ID token is required.")
		return
	}

	identity, err := h.GoogleVerifier.Verify(c.Request.Context(), req.IDToken)
	if err != nil {
		// Never log the token. The verifier's error names the check that
		// failed, which is enough for an operator and useless to a prober.
		log.Printf("google sign-in: token verification failed: %v", err)
		respond.Fail(c, respond.CodeInvalidCredentials, "Google sign-in could not be verified. Please try again.")
		return
	}
	if !identity.EmailVerified {
		respond.Fail(c, respond.CodeInvalidCredentials, "Google has not verified the email address on that account.")
		return
	}
	if identity.Email == "" {
		respond.Fail(c, respond.CodeInvalidCredentials, "That Google account did not share an email address.")
		return
	}

	// Google proves control of a Google account, not university enrolment.
	// Students may self-register on their campus domain; staff and
	// administrators may sign in only if an administrator already created the
	// account, so a token can never mint an elevated role.
	isStudent := h.Config.IsStudentEmail(identity.Email)
	isStaff := h.Config.IsAdminEmail(identity.Email)
	if !isStudent && !isStaff {
		respond.Fail(c, respond.CodeForbidden, "Only @"+h.Config.StudentDomain()+" student or @"+h.Config.AdminDomain()+" staff accounts can sign in with Google.")
		return
	}
	if hosted := h.Config.GoogleOAuthHostedDomain; hosted != "" && !strings.EqualFold(identity.HostedDomain, hosted) {
		respond.Fail(c, respond.CodeForbidden, "That Google account is not part of the university's Google Workspace.")
		return
	}

	user, created, err := h.resolveGoogleUser(c.Request.Context(), identity, c.ClientIP(), isStudent)
	switch {
	case errors.Is(err, errGoogleAccountDisabled):
		respond.Fail(c, respond.CodeAccountDisabled, "Your account has been disabled.")
		return
	case errors.Is(err, errGoogleAccountUnknown):
		respond.Fail(c, respond.CodeForbidden, "No CampVote account exists for that address. Staff and administrator accounts are created by an election administrator.")
		return
	case errors.Is(err, errGoogleAccountConflict):
		respond.Fail(c, respond.CodeConflict, "This Google account cannot be linked to that CampVote account. Ask an administrator for help.")
		return
	case err != nil:
		respond.Internal(c, err)
		return
	}
	if user == nil || !user.Active {
		respond.Fail(c, respond.CodeAccountDisabled, "Your account has been disabled.")
		return
	}

	tokens, err := h.AuthService.GenerateTokenPair(user.ID, user.Email, user.Role)
	if err != nil {
		respond.Fail(c, respond.CodeTokenError, "Failed to generate tokens")
		return
	}
	if _, err := services.CreateSession(h.DB, c, user.ID, tokens.RefreshToken); err != nil {
		log.Printf("google sign-in: failed to record session for %s: %v", user.ID, err)
	}
	h.AuthService.SetAuthCookies(c, tokens)

	if created {
		services.LogRegister(h.DB, c, user.ID, user.Email)
	} else {
		services.LogLogin(h.DB, c, user.ID, user.Email)
	}

	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	c.JSON(status, gin.H{
		"data": gin.H{
			"user":   user,
			"tokens": tokens,
		},
		"message": "Signed in with Google",
	})
}

// resolveGoogleUser finds the account a verified Google identity belongs to,
// or creates one when allowCreate is set.
//
// The lookup order is the security property: the immutable subject first, then
// the verified email. A subject already linked to another address keeps its
// account; an email already registered is adopted rather than duplicated.
func (h *AuthHandler) resolveGoogleUser(ctx context.Context, identity *services.GoogleIdentity, ip string, allowCreate bool) (*models.User, bool, error) {
	db := h.DB.WithContext(ctx)

	var user models.User
	err := db.Where("google_id = ?", identity.Subject).First(&user).Error
	if err == nil {
		if !user.Active {
			return nil, false, errGoogleAccountDisabled
		}
		return &user, false, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, err
	}

	err = db.Where("LOWER(email) = ?", identity.Email).First(&user).Error
	if err == nil {
		if !user.Active {
			return nil, false, errGoogleAccountDisabled
		}
		if user.GoogleID != "" && user.GoogleID != identity.Subject {
			return nil, false, errGoogleAccountConflict
		}
		return h.linkGoogleIdentity(ctx, &user, identity)
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, err
	}
	if !allowCreate {
		return nil, false, errGoogleAccountUnknown
	}

	now := time.Now().UTC()
	firstName := identity.FirstName
	if firstName == "" {
		firstName = identity.Name
	}
	if firstName == "" {
		firstName = "Student"
	}
	user = models.User{
		FirstName:       firstName,
		LastName:        identity.LastName,
		Email:           identity.Email,
		Avatar:          identity.Picture,
		Role:            models.RoleStudentVoter,
		Active:          true,
		Provider:        "google",
		GoogleID:        identity.Subject,
		EmailVerifiedAt: &now,
		IPAddress:       ip,
	}
	if err := services.CreateUser(ctx, h.DB, &user); err != nil {
		if errors.Is(err, services.ErrEmailExists) || services.IsDuplicateKey(err) {
			// Another request for the same subject or address won the insert.
			// Re-resolve once without creating; the unique indexes decided.
			existing, _, retryErr := h.resolveGoogleUser(ctx, identity, ip, false)
			if retryErr == nil && existing != nil {
				return existing, false, nil
			}
			return nil, false, errGoogleAccountConflict
		}
		return nil, false, err
	}
	return &user, true, nil
}

// linkGoogleIdentity attaches a subject to an existing account without
// changing its role, name, student identifiers or password.
//
// An account whose address was never verified may have been registered by
// somebody else. Google has now shown who controls the address, so the old
// password stops working and any session it opened ends before the link is
// written. This mirrors the redirect OAuth path's rule.
func (h *AuthHandler) linkGoogleIdentity(ctx context.Context, user *models.User, identity *services.GoogleIdentity) (*models.User, bool, error) {
	updates := map[string]interface{}{"google_id": identity.Subject}
	if user.Provider == "" || user.Provider == "local" {
		updates["provider"] = "google"
	}

	if user.EmailVerifiedAt == nil {
		now := time.Now().UTC()
		updates["email_verified_at"] = now
		updates["password"] = ""
		if err := h.DB.WithContext(ctx).Model(user).Updates(updates).Error; err != nil {
			return nil, false, err
		}
		if err := services.RevokeAllUserSessions(h.DB, user.ID, ""); err != nil {
			log.Printf("google sign-in: revoking sessions of unverified account %s: %v", user.ID, err)
		}
		user.Password = ""
		user.EmailVerifiedAt = &now
	} else if user.GoogleID == "" {
		if err := h.DB.WithContext(ctx).Model(user).Updates(updates).Error; err != nil {
			return nil, false, err
		}
	}

	user.GoogleID = identity.Subject
	if user.Provider == "" || user.Provider == "local" {
		user.Provider = "google"
	}
	return user, false, nil
}
