package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"campvote/apps/api/internal/mail"
	"campvote/apps/api/internal/models"
	"campvote/apps/api/internal/respond"
	"campvote/apps/api/internal/services"
	"time"

	"github.com/gin-gonic/gin"
)

// Email verification: send the six-digit code, and redeem it.

// SendVerificationEmail issues a fresh verification code for the signed-in
// user. Authenticated on purpose: an unauthenticated "send a code to this
// address" endpoint is a spam cannon aimed at whoever you name.
func (h *AuthHandler) SendVerificationEmail(c *gin.Context) {
	userID := c.GetString("user_id")

	user, err := h.AuthService.UserByID(c.Request.Context(), userID)
	if err != nil {
		respond.Fail(c, respond.CodeNotFound, "User not found")
		return
	}

	if user.EmailVerifiedAt != nil {
		respond.Fail(c, respond.CodeAlreadyVerified, "This email is already verified")
		return
	}

	code, err := h.deliverVerificationEmail(c.Request.Context(), *user)
	if err != nil {
		// In development the caller still gets the code below, so a missing or
		// failing mailer does not block local testing. In production a code
		// nobody received is not a success, and the response must say so.
		log.Printf("email verification: sending to %s: %v", user.Email, err)
		if h.Config.AppEnv == "production" || code == "" {
			respond.Fail(c, respond.CodeMailFailed, "We could not send the verification email. Try again in a moment.")
			return
		}
	}

	body := gin.H{"message": "Verification code sent. It is valid for 10 minutes."}
	// Dev-only fallback, gated behind MAIL_ECHO_CODES=true: when delivery
	// actually failed (no mailer or send error) the code is returned here so
	// the flow can still be completed without a working mail server. A
	// development deploy with working SMTP must not leak the code, which
	// would let anyone verify any address.
	if h.Config.Mail.EchoCodes && h.Config.AppEnv != "production" && err != nil && code != "" {
		body["verify_code"] = code
	}

	c.JSON(http.StatusOK, body)
}

// The six-digit code from the verification email.

func (h *AuthHandler) VerifyEmail(c *gin.Context) {
	var req VerifyEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond.Fail(c, respond.CodeValidationError, err.Error())
		return
	}

	if _, err := services.ConsumeEmailVerificationToken(h.DB, req.Code, h.Config.JWTSecret); err != nil {
		// One message for expired, spent, unknown and address-changed. Telling
		// them apart tells an attacker which codes once existed.
		respond.Fail(c, respond.CodeInvalidLink, "That verification code is invalid or has expired. Request a new one.")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Email verified"})
}

// deliverVerificationEmail mints a six-digit code and queues the mail.
//
// The code is retried on a storage collision: only one million codes exist, so
// a duplicate is expected occasionally and must not fail the request.
//
// It used to run in a goroutine the request started, so a slow SMTP call could
// not hold the response open. The mail goes on the background queue now, which
// keeps the response just as short and, unlike a goroutine, retries a provider
// that is briefly down and survives a deploy. Without a queue the send is
// inline, which is what development without Redis does.
func (h *AuthHandler) deliverVerificationEmail(ctx context.Context, user models.User) (string, error) {
	var code string
	issued := false
	for attempt := 0; attempt < 5 && !issued; attempt++ {
		generated, err := services.GenerateVerificationCode()
		if err != nil {
			return "", fmt.Errorf("generating a code for %s: %w", user.Email, err)
		}
		code = generated
		if _, err := services.CreateEmailVerificationToken(h.DB, user.ID, user.Email, code, h.Config.JWTSecret); err != nil {
			log.Printf("email verification: storing code for %s: %v", user.Email, err)
			continue
		}
		issued = true
	}
	if !issued {
		return "", fmt.Errorf("storing a verification code for %s", user.Email)
	}

	if h.Jobs != nil || h.Mailer != nil {
		// Keyed on the code, so one enqueue per mail: a client that retries
		// the request mints a new code and gets a new key, and a proxy that
		// replays the same one does not send the mail twice.
		err := dispatchMail(ctx, h.Mailer, h.Jobs, "verify:"+user.ID+":"+code, mail.SendOptions{
			To:       user.Email,
			Subject:  "Your CampVote sign-in code",
			Template: "email-verification",
			Data: map[string]interface{}{
				"AppName":   h.Config.AppName,
				"FirstName": user.FirstName,
				"Title":     "Your verification code",
				"Message":   "Enter this code in " + h.Config.AppName + " to finish signing in.",
				"Code":      code,
				"Minutes":   10,
				"Year":      time.Now().Year(),
			},
		})
		if err != nil {
			return code, err
		}
		return code, nil
	}

	if h.Config.AppEnv == "production" {
		return code, ErrNoMailer
	}

	log.Printf("email verification code for %s: %s", user.Email, code)
	return code, ErrNoMailer
}
