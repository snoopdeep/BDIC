package api

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"

	"bdic/backend/internal/audit"
	"bdic/backend/internal/auth"
	"bdic/backend/internal/httpx"
	"bdic/backend/internal/store"
)

// handleHealth reports whether the API and its database are up. Used by the
// deployment's health check, so it must be cheap and must actually touch the
// database rather than only proving the process is running.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := contextWithTimeout(r, 3*time.Second)
	defer cancel()

	status := "ok"
	dbStatus := "ok"
	if err := s.store.Pool().Ping(ctx); err != nil {
		status = "degraded"
		dbStatus = "unreachable"
		s.logger.Error("health check: database unreachable", "error", err)
	}

	code := http.StatusOK
	if status != "ok" {
		code = http.StatusServiceUnavailable
	}

	httpx.JSON(w, code, map[string]any{
		"status":   status,
		"service":  "bdic-api",
		"database": dbStatus,
		"env":      s.cfg.Env,
		"time":     time.Now().Format(time.RFC3339),
	})
}

type loginRequest struct {
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
}

type loginResponse struct {
	AccessToken string        `json:"accessToken"`
	ExpiresAt   string        `json:"expiresAt"`
	User        auth.Identity `json:"user"`
}

// handleLogin signs a user in.
//
// One identifier field accepts an email, a phone number, an employee code, or
// an admission number, because that is how the four kinds of user think of
// themselves. A wrong identifier and a wrong password produce the same message
// and the same timing, so the endpoint cannot be used to discover which
// accounts exist.
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var input loginRequest
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.Fail(w, r, err)
		return
	}

	identifier := strings.TrimSpace(input.Identifier)
	if identifier == "" || input.Password == "" {
		httpx.Fail(w, r, httpx.ErrBadRequest(
			"Enter your email, phone number, or admission number, and your password.",
			"अपना ईमेल, फ़ोन नंबर या अनुक्रमांक और पासवर्ड दर्ज करें।"))
		return
	}

	// Two limits. The per-identifier one stops someone working through a
	// password list against the Principal's account from many addresses; the
	// per-address one stops one machine working through many accounts.
	clientIP := auth.ClientIP(r, s.cfg.IsProduction())
	identifierKey := "id:" + strings.ToLower(identifier)
	addressKey := "ip:" + clientIP

	if !s.loginLimiter.Allow(identifierKey) || !s.loginLimiter.Allow(addressKey) {
		s.audit.RecordLoginFailure(r.Context(), r, identifier, "rate limited")
		httpx.Fail(w, r, httpx.ErrTooManyRequests())
		return
	}

	credentials, err := s.store.FindCredentials(r.Context(), identifier)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			// Spend roughly the same time as a real bcrypt comparison would,
			// so a missing account is not detectable by how fast we answer.
			auth.VerifyPassword(
				"$2a$12$C6UzMDM.H6dfI/f/IKcEe.7ZtVKBRzuGO0k6ZTUCiKx4J/1oZ3Yuy",
				input.Password)
			s.audit.RecordLoginFailure(r.Context(), r, identifier, "no such user")
			httpx.Fail(w, r, invalidCredentials())
			return
		}
		httpx.Fail(w, r, httpx.ErrInternal().WithInternal(err))
		return
	}

	if !auth.VerifyPassword(credentials.PasswordHash, input.Password) {
		s.audit.RecordLoginFailure(r.Context(), r, identifier, "wrong password")
		httpx.Fail(w, r, invalidCredentials())
		return
	}

	if credentials.Identity.Status != "ACTIVE" {
		s.audit.RecordLoginFailure(r.Context(), r, identifier, "account not active")
		httpx.Fail(w, r, httpx.NewError(http.StatusForbidden, "ACCOUNT_INACTIVE",
			"This account has been suspended. Please contact the school office.",
			"यह खाता निलंबित कर दिया गया है। कृपया विद्यालय कार्यालय से संपर्क करें।"))
		return
	}

	token, expiresAt, err := s.signer.Issue(
		credentials.Identity.UserID,
		credentials.Identity.Role,
		credentials.Identity.TokenVersion,
	)
	if err != nil {
		httpx.Fail(w, r, httpx.ErrInternal().WithInternal(err))
		return
	}

	// The password is in hand for this one moment, so an old hash gets
	// upgraded to the current cost here or never.
	upgradedHash := ""
	if auth.NeedsRehash(credentials.PasswordHash) {
		if rehashed, hashErr := auth.HashPassword(input.Password); hashErr == nil {
			upgradedHash = rehashed
		}
	}
	if err := s.store.RecordSignIn(r.Context(), credentials.Identity.UserID, upgradedHash); err != nil {
		// Not worth failing the sign-in over: the user is authenticated and
		// the only loss is an accurate last_login_at.
		s.logger.Warn("could not record sign-in", "error", err)
	}

	s.loginLimiter.Reset(identifierKey)
	s.loginLimiter.Reset(addressKey)

	ctx := auth.WithIdentity(r.Context(), credentials.Identity)
	s.audit.Record(ctx, r, audit.Entry{
		Action:     audit.ActionLogin,
		EntityType: "user",
		EntityID:   credentials.Identity.UserID,
		Summary:    credentials.Identity.FullNameEN + " signed in",
	})

	httpx.JSON(w, http.StatusOK, loginResponse{
		AccessToken: token,
		ExpiresAt:   expiresAt.Format(time.RFC3339),
		User:        credentials.Identity,
	})
}

// invalidCredentials is one message for both "no such user" and "wrong
// password". Telling them apart is a gift to whoever is guessing.
func invalidCredentials() *httpx.APIError {
	return httpx.NewError(http.StatusUnauthorized, "INVALID_CREDENTIALS",
		"Those details do not match an account.",
		"ये विवरण किसी खाते से मेल नहीं खाते।")
}

// handleLogout is recorded for the audit trail. Tokens are stateless, so the
// client discards its copy; a token that must die immediately is killed by
// bumping token_version, which is what a password change does.
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	identity := auth.MustFromContext(r.Context())
	s.audit.Record(r.Context(), r, audit.Entry{
		Action:     audit.ActionLogout,
		EntityType: "user",
		EntityID:   identity.UserID,
		Summary:    identity.FullNameEN + " signed out",
	})
	httpx.NoContent(w)
}

// handleMe returns the signed-in user plus the context every portal screen
// needs on load: the school, the current session, and the settings. One request
// instead of four on a slow connection.
func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	identity := auth.MustFromContext(r.Context())

	school, err := s.store.GetSchool(r.Context())
	if err != nil {
		httpx.Fail(w, r, httpx.ErrInternal().WithInternal(err))
		return
	}

	session, err := s.store.CurrentSession(r.Context())
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		httpx.Fail(w, r, httpx.ErrInternal().WithInternal(err))
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"user":           identity,
		"school":         school,
		"currentSession": session,
	})
}

type setLocaleRequest struct {
	Locale string `json:"locale"`
}

// handleSetLocale remembers the language the user picked, so the choice
// survives signing out and signing in on another device.
func (s *Server) handleSetLocale(w http.ResponseWriter, r *http.Request) {
	identity := auth.MustFromContext(r.Context())

	var input setLocaleRequest
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	if input.Locale != "en" && input.Locale != "hi" {
		httpx.Fail(w, r, httpx.ErrBadRequest(
			"Choose either English or Hindi.",
			"अंग्रेज़ी या हिंदी चुनें।").WithField("locale", "Must be en or hi"))
		return
	}

	if err := s.store.SetLocale(r.Context(), identity.UserID, input.Locale); err != nil {
		httpx.Fail(w, r, httpx.ErrInternal().WithInternal(err))
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"locale": input.Locale})
}

type changePasswordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

// handleChangePassword changes the signed-in user's own password.
//
// The current password is required even though the caller is already
// authenticated: it is what stops someone who walks up to an unlocked phone in
// a staff room from taking the account over.
func (s *Server) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	identity := auth.MustFromContext(r.Context())

	var input changePasswordRequest
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.Fail(w, r, err)
		return
	}

	credentials, err := s.store.FindCredentialsByUserID(r.Context(), identity.UserID)
	if err != nil {
		httpx.Fail(w, r, httpx.ErrInternal().WithInternal(err))
		return
	}
	if !auth.VerifyPassword(credentials.PasswordHash, input.CurrentPassword) {
		httpx.Fail(w, r, httpx.ErrBadRequest(
			"Your current password is not correct.",
			"आपका वर्तमान पासवर्ड सही नहीं है।").
			WithField("currentPassword", "Incorrect"))
		return
	}
	if input.CurrentPassword == input.NewPassword {
		httpx.Fail(w, r, httpx.ErrBadRequest(
			"The new password must be different from the current one.",
			"नया पासवर्ड वर्तमान पासवर्ड से भिन्न होना चाहिए।").
			WithField("newPassword", "Must be different"))
		return
	}

	hashed, err := auth.HashPassword(input.NewPassword)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	if err := s.store.SetPassword(r.Context(), identity.UserID, hashed, false); err != nil {
		httpx.Fail(w, r, httpx.ErrInternal().WithInternal(err))
		return
	}

	s.audit.Record(r.Context(), r, audit.Entry{
		Action:     audit.ActionPasswordReset,
		EntityType: "user",
		EntityID:   identity.UserID,
		Summary:    identity.FullNameEN + " changed their own password",
	})

	// token_version was bumped, so every other device is now signed out. The
	// caller has to sign in again too, which is the honest behaviour.
	httpx.JSON(w, http.StatusOK, map[string]any{
		"changed":      true,
		"signedOutAll": true,
		"messageEn":    "Password changed. Please sign in again.",
		"messageHi":    "पासवर्ड बदल गया। कृपया दोबारा साइन इन करें।",
	})
}

type forgotPasswordRequest struct {
	Identifier string `json:"identifier"`
}

// handleForgotPassword starts a reset.
//
// It always answers the same way, whether or not the identifier matches an
// account, so the endpoint cannot be used to test which phone numbers belong to
// parents at this school.
//
// The code is not sent anywhere yet. No SMS provider is connected, so a DRAFT
// row goes into the outbox where the office can read the code out to the parent
// over the counter or on the phone. Once the school opens an SMS account and
// fills SMS_API_KEY in .env, the same draft is what the dispatcher sends.
func (s *Server) handleForgotPassword(w http.ResponseWriter, r *http.Request) {
	var input forgotPasswordRequest
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.Fail(w, r, err)
		return
	}

	identifier := strings.TrimSpace(input.Identifier)
	sameAnswer := map[string]string{
		"messageEn": "If those details match an account, the school office can give you a reset code.",
		"messageHi": "यदि ये विवरण किसी खाते से मेल खाते हैं, तो विद्यालय कार्यालय आपको रीसेट कोड दे सकता है।",
	}

	if identifier == "" {
		httpx.JSON(w, http.StatusAccepted, sameAnswer)
		return
	}

	clientIP := auth.ClientIP(r, s.cfg.IsProduction())
	if !s.publicLimiter.Allow("forgot:" + clientIP) {
		httpx.Fail(w, r, httpx.ErrTooManyRequests())
		return
	}

	credentials, err := s.store.FindCredentials(r.Context(), identifier)
	if err != nil {
		// Deliberately the same response as success.
		httpx.JSON(w, http.StatusAccepted, sameAnswer)
		return
	}

	code, err := generateResetCode()
	if err != nil {
		httpx.Fail(w, r, httpx.ErrInternal().WithInternal(err))
		return
	}

	const resetCodeTTL = 15 * time.Minute
	if err := s.store.CreatePasswordResetCode(
		r.Context(), credentials.Identity.UserID, hashResetCode(code), resetCodeTTL,
	); err != nil {
		httpx.Fail(w, r, httpx.ErrInternal().WithInternal(err))
		return
	}

	recipient := credentials.Identity.Phone
	channel := store.ChannelSMS
	if recipient == "" {
		recipient = credentials.Identity.Email
		channel = store.ChannelEmail
	}
	if recipient != "" {
		body := fmt.Sprintf(
			"BDIC: your password reset code is %s. It is valid for 15 minutes. Do not share it.\n"+
				"BDIC: आपका पासवर्ड रीसेट कोड %s है। यह 15 मिनट के लिए मान्य है। इसे किसी से साझा न करें।",
			code, code)
		if _, err := s.store.QueueOutbound(r.Context(), store.OutboundDraft{
			Channel:         channel,
			Recipient:       recipient,
			RecipientUserID: credentials.Identity.UserID,
			TemplateKey:     "PASSWORD_RESET_CODE",
			Subject:         "BDIC password reset code",
			Body:            body,
			RelatedType:     "user",
			RelatedID:       credentials.Identity.UserID,
		}); err != nil {
			s.logger.Error("could not queue reset code", "error", err)
		}
	}

	// In development the code is logged so the build can be tested without a
	// provider. This is gated on APP_ENV, so production never logs a code.
	if !s.cfg.IsProduction() {
		s.logger.Warn("development only: password reset code issued",
			"identifier", identifier, "code", code)
	}

	httpx.JSON(w, http.StatusAccepted, sameAnswer)
}

type resetPasswordRequest struct {
	Identifier  string `json:"identifier"`
	Code        string `json:"code"`
	NewPassword string `json:"newPassword"`
}

// handleResetPassword completes a reset with a code from the office.
func (s *Server) handleResetPassword(w http.ResponseWriter, r *http.Request) {
	var input resetPasswordRequest
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.Fail(w, r, err)
		return
	}

	identifier := strings.TrimSpace(input.Identifier)
	code := strings.TrimSpace(input.Code)
	if identifier == "" || code == "" || input.NewPassword == "" {
		httpx.Fail(w, r, httpx.ErrBadRequest(
			"Enter your details, the code you were given, and a new password.",
			"अपना विवरण, आपको दिया गया कोड और नया पासवर्ड दर्ज करें।"))
		return
	}

	clientIP := auth.ClientIP(r, s.cfg.IsProduction())
	if !s.loginLimiter.Allow("reset:" + clientIP) {
		httpx.Fail(w, r, httpx.ErrTooManyRequests())
		return
	}

	invalidCode := httpx.ErrBadRequest(
		"That code is not valid or has expired. Ask the school office for a new one.",
		"वह कोड मान्य नहीं है या समाप्त हो गया है। कार्यालय से नया कोड लें।")

	credentials, err := s.store.FindCredentials(r.Context(), identifier)
	if err != nil {
		httpx.Fail(w, r, invalidCode)
		return
	}

	// Strength is checked before the code is consumed, so a weak password does
	// not burn the user's one-time code.
	hashed, err := auth.HashPassword(input.NewPassword)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}

	if err := s.store.ConsumePasswordResetCode(
		r.Context(), credentials.Identity.UserID, hashResetCode(code),
	); err != nil {
		httpx.Fail(w, r, invalidCode)
		return
	}

	if err := s.store.SetPassword(r.Context(), credentials.Identity.UserID, hashed, false); err != nil {
		httpx.Fail(w, r, httpx.ErrInternal().WithInternal(err))
		return
	}

	ctx := auth.WithIdentity(r.Context(), credentials.Identity)
	s.audit.Record(ctx, r, audit.Entry{
		Action:     audit.ActionPasswordReset,
		EntityType: "user",
		EntityID:   credentials.Identity.UserID,
		Summary:    "Password reset using a one-time code",
	})

	httpx.JSON(w, http.StatusOK, map[string]any{
		"reset":     true,
		"messageEn": "Password reset. Please sign in with your new password.",
		"messageHi": "पासवर्ड रीसेट हो गया। कृपया अपने नए पासवर्ड से साइन इन करें।",
	})
}

// generateResetCode returns a six-digit code from a cryptographic source.
// math/rand would be predictable, and a predictable reset code is no protection
// at all.
func generateResetCode() (string, error) {
	upper := big.NewInt(1_000_000)
	value, err := rand.Int(rand.Reader, upper)
	if err != nil {
		return "", fmt.Errorf("generate reset code: %w", err)
	}
	return fmt.Sprintf("%06d", value.Int64()), nil
}

// hashResetCode stores only a hash of the code, so the table is of no use to
// anyone who reads it.
func hashResetCode(code string) string {
	sum := sha256.Sum256([]byte(code))
	return hex.EncodeToString(sum[:])
}
