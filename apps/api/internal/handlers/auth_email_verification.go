package handlers

import (
	"context"
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

	code := h.deliverVerificationEmail(c.Request.Context(), *user)

	body := gin.H{"message": "Verification code sent. It is valid for 10 minutes."}

	// In development the code comes back with the response, so the banner can
	// show it. Verifying an email in development otherwise means opening the
	// log or digging a .html file out of storage/mail, which is enough friction
	// that the warning banner simply stays up forever in every dev project.
	// Guarded on APP_ENV: anywhere else this would hand a working code to
	// whoever asked, which is the whole secret.
	if h.Config.AppEnv == "development" && code != "" {
		body["verify_code"] = code
		body["message"] = "Development: the code is below rather than in your inbox."
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
func (h *AuthHandler) deliverVerificationEmail(ctx context.Context, user models.User) string {
	var code string
	issued := false
	for attempt := 0; attempt < 5 && !issued; attempt++ {
		generated, err := services.GenerateVerificationCode()
		if err != nil {
			log.Printf("email verification: generating code for %s: %v", user.Email, err)
			return ""
		}
		code = generated
		if _, err := services.CreateEmailVerificationToken(h.DB, user.ID, user.Email, code, h.Config.JWTSecret); err != nil {
			log.Printf("email verification: storing code for %s: %v", user.Email, err)
			continue
		}
		issued = true
	}
	if !issued {
		return ""
	}

	if h.Jobs != nil || h.Mailer != nil {
		// Keyed on the code, so one enqueue per mail: a client that retries
		// the request mints a new code and gets a new key, and a proxy that
		// replays the same one does not send the mail twice.
		if err := dispatchMail(ctx, h.Mailer, h.Jobs, "verify:"+user.ID+":"+code, mail.SendOptions{
			To:       user.Email,
			Subject:  "Confirm your email address",
			Template: "email-verification",
			Data: map[string]interface{}{
				"AppName": h.Config.AppName,
				"Title":   "Confirm your email address",
				"Message": "Enter this code on the verification page to confirm this address.",
				"Code":    code,
				"Minutes": 10,
				"Year":    time.Now().Year(),
			},
		}); err != nil {
			log.Printf("email verification: sending to %s: %v", user.Email, err)
		}
		return code
	}

	if h.Config.AppEnv == "production" {
		log.Printf("email verification: NO MAILER CONFIGURED: %s cannot receive a code. Set MAIL_MAILER (see .env.example).", user.Email)
		return ""
	}

	log.Printf("email verification code for %s: %s", user.Email, code)
	return code
}
