package auth

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"

	mailer "github.com/mirinnano/reword/api/internal/mail"
	"github.com/mirinnano/reword/api/internal/platform/httpx"
	"github.com/mirinnano/reword/api/internal/platform/store"
)

// Versions of the legal documents a new account accepts. Bump when the
// published text changes; consents keep the version each user agreed to.
const (
	TermsVersion   = "2026-10-draft"
	PrivacyVersion = "2026-10-draft"
)

const (
	signupTTL       = 24 * time.Hour
	resetTTL        = 30 * time.Minute
	maxPendingPerEm = 3
	MaxDisplayName  = 40
)

type Handlers struct {
	DB        *sql.DB
	Q         *store.Queries
	Hasher    *Hasher
	Sessions  *Sessions
	Limiter   *Limiter
	Outbox    mailer.Outbox
	AppOrigin string
	Log       *slog.Logger
	Now       func() time.Time
	// AfterLogin builds the response body for a freshly signed-in user.
	AfterLogin func(ctx context.Context, userID, profileID uint64) (any, error)
}

func (h *Handlers) Routes(r chi.Router) {
	r.Post("/auth/signup", h.signup)
	r.Post("/auth/signup/verify", h.verifySignup)
	r.Post("/auth/login", h.login)
	r.Post("/auth/logout", h.logout)
	r.Post("/auth/password-reset", h.requestReset)
	r.Post("/auth/password-reset/confirm", h.confirmReset)
}

func NormalizeEmail(s string) (string, bool) {
	s = strings.ToLower(strings.TrimSpace(s))
	if len(s) > 254 {
		return "", false
	}
	a, err := mail.ParseAddress(s)
	if err != nil || a.Address != s || a.Name != "" || !strings.Contains(s[strings.LastIndex(s, "@")+1:], ".") {
		return "", false
	}
	return s, true
}

func ValidDisplayName(s string) (string, bool) {
	s = strings.TrimSpace(s)
	n := utf8.RuneCountInString(s)
	return s, n >= 1 && n <= MaxDisplayName
}

// limit applies l to key and writes 429 when exceeded.
func (h *Handlers) limit(w http.ResponseWriter, r *http.Request, l Limit, key string) bool {
	ok, err := h.Limiter.Allow(r.Context(), l, key)
	if err != nil {
		h.internal(w, "rate limit", err)
		return false
	}
	if !ok {
		httpx.WriteError(w, http.StatusTooManyRequests, "rate_limited", "too many requests, try again later")
		return false
	}
	return true
}

func (h *Handlers) internal(w http.ResponseWriter, what string, err error) {
	h.Log.Error(what, "err", err)
	httpx.WriteError(w, http.StatusInternalServerError, "internal", "internal error")
}

type signupReq struct {
	Email         string `json:"email"`
	Password      string `json:"password"`
	DisplayName   string `json:"display_name"`
	TargetGradeID *uint8 `json:"target_grade_id"`
	AcceptTerms   bool   `json:"accept_terms"`
}

// signup always answers 202 so it does not reveal whether an address is
// registered. The account itself is created by verifySignup.
func (h *Handlers) signup(w http.ResponseWriter, r *http.Request) {
	var req signupReq
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	email, ok := NormalizeEmail(req.Email)
	if !ok {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_email", "invalid email address")
		return
	}
	name, ok := ValidDisplayName(req.DisplayName)
	if !ok {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_display_name", "display name must be 1 to 40 characters")
		return
	}
	if err := ValidatePassword(req.Password); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "weak_password", err.Error())
		return
	}
	if !req.AcceptTerms {
		httpx.WriteError(w, http.StatusBadRequest, "terms_not_accepted", "terms and privacy policy must be accepted")
		return
	}
	ctx := r.Context()
	grade, ok := h.validGrade(ctx, w, req.TargetGradeID)
	if !ok {
		return
	}
	ip := httpx.ClientIP(ctx)
	if !h.limit(w, r, LimitSignupIP, ip) || !h.limit(w, r, LimitSignupEmail, email) {
		return
	}

	// Hash before the lookup so both branches cost the same.
	pwHash, err := h.Hasher.Hash(ctx, req.Password)
	if err != nil {
		h.internal(w, "hash password", err)
		return
	}
	accepted := func() { httpx.WriteJSON(w, http.StatusAccepted, map[string]string{"status": "verification_sent"}) }

	_, err = h.Q.GetUserByEmail(ctx, email)
	switch {
	case err == nil:
		if err := h.Outbox.Enqueue(ctx, mailer.AlreadyRegistered(email, h.AppOrigin)); err != nil {
			h.internal(w, "enqueue mail", err)
			return
		}
		accepted()
		return
	case !errors.Is(err, sql.ErrNoRows):
		h.internal(w, "lookup user", err)
		return
	}

	now := h.Now()
	pending, err := h.Q.CountPendingSignups(ctx, store.CountPendingSignupsParams{Email: email, ExpiresAt: now})
	if err != nil {
		h.internal(w, "count pending", err)
		return
	}
	if pending >= maxPendingPerEm {
		accepted() // silently drop; the rate limit already caps mail volume
		return
	}
	plain, hash, err := NewToken()
	if err != nil {
		h.internal(w, "token", err)
		return
	}
	if err := h.Q.CreatePendingSignup(ctx, store.CreatePendingSignupParams{
		TokenHash: hash, Email: email, PasswordHash: pwHash, DisplayName: name,
		TargetGradeID: grade, TermsVersion: TermsVersion, PrivacyVersion: PrivacyVersion,
		Ip: ip, ExpiresAt: now.Add(signupTTL),
	}); err != nil {
		h.internal(w, "create pending signup", err)
		return
	}
	if err := h.Outbox.Enqueue(ctx, mailer.VerifySignup(email, h.AppOrigin, plain)); err != nil {
		h.internal(w, "enqueue mail", err)
		return
	}
	accepted()
}

func (h *Handlers) validGrade(ctx context.Context, w http.ResponseWriter, id *uint8) (sql.NullInt16, bool) {
	if id == nil {
		return sql.NullInt16{}, true
	}
	active, err := h.Q.GradeIsActive(ctx, *id)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && !active) {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_grade", "unknown or unavailable grade")
		return sql.NullInt16{}, false
	}
	if err != nil {
		h.internal(w, "grade lookup", err)
		return sql.NullInt16{}, false
	}
	return sql.NullInt16{Int16: int16(*id), Valid: true}, true
}

type tokenReq struct {
	Token string `json:"token"`
}

// verifySignup turns a pending signup into an account and signs it in.
func (h *Handlers) verifySignup(w http.ResponseWriter, r *http.Request) {
	var req tokenReq
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	ctx := r.Context()
	if !h.limit(w, r, LimitTokenRedeemIP, httpx.ClientIP(ctx)) {
		return
	}
	invalid := func() {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_token", "this link is invalid or has expired")
	}
	if req.Token == "" {
		invalid()
		return
	}

	tx, err := h.DB.BeginTx(ctx, nil)
	if err != nil {
		h.internal(w, "begin", err)
		return
	}
	defer tx.Rollback()
	q := h.Q.WithTx(tx)
	now := h.Now()

	p, err := q.GetPendingSignupForUpdate(ctx, HashToken(req.Token))
	if errors.Is(err, sql.ErrNoRows) {
		invalid()
		return
	}
	if err != nil {
		h.internal(w, "load pending signup", err)
		return
	}
	if !p.ExpiresAt.After(now) {
		invalid()
		return
	}
	if _, err := q.GetUserByEmail(ctx, p.Email); err == nil {
		// Another pending signup for this address was verified first.
		_ = q.DeletePendingSignupsByEmail(ctx, p.Email)
		_ = tx.Commit()
		invalid()
		return
	} else if !errors.Is(err, sql.ErrNoRows) {
		h.internal(w, "lookup user", err)
		return
	}

	uid64, err := q.CreateUser(ctx, store.CreateUserParams{Email: p.Email, PasswordHash: p.PasswordHash})
	if err != nil {
		h.internal(w, "create user", err)
		return
	}
	uid := uint64(uid64)
	if err := q.MarkEmailVerified(ctx, store.MarkEmailVerifiedParams{
		EmailVerifiedAt: sql.NullTime{Time: now, Valid: true}, ID: uid,
	}); err != nil {
		h.internal(w, "verify email", err)
		return
	}
	pid64, err := q.CreateProfile(ctx, store.CreateProfileParams{
		UserID: uid, DisplayName: p.DisplayName, TargetGradeID: p.TargetGradeID,
	})
	if err != nil {
		h.internal(w, "create profile", err)
		return
	}
	for _, c := range []store.CreateConsentParams{
		{UserID: uid, Document: store.ConsentsDocumentTerms, Version: p.TermsVersion, Ip: p.Ip},
		{UserID: uid, Document: store.ConsentsDocumentPrivacy, Version: p.PrivacyVersion, Ip: p.Ip},
	} {
		if err := q.CreateConsent(ctx, c); err != nil {
			h.internal(w, "record consent", err)
			return
		}
	}
	if err := q.DeletePendingSignupsByEmail(ctx, p.Email); err != nil {
		h.internal(w, "delete pending signups", err)
		return
	}
	if err := tx.Commit(); err != nil {
		h.internal(w, "commit", err)
		return
	}
	h.startAndRespond(w, r, uid, uint64(pid64), http.StatusCreated)
}

type loginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handlers) login(w http.ResponseWriter, r *http.Request) {
	var req loginReq
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	ctx := r.Context()
	email, _ := NormalizeEmail(req.Email)
	if !h.limit(w, r, LimitLoginIP, httpx.ClientIP(ctx)) || !h.limit(w, r, LimitLoginEmail, email) {
		return
	}
	u, err := h.Q.GetUserByEmail(ctx, email)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		h.internal(w, "lookup user", err)
		return
	}
	// u.PasswordHash is "" for unknown users; Verify then checks the dummy hash.
	ok, verr := h.Hasher.Verify(ctx, req.Password, u.PasswordHash)
	if verr != nil {
		h.internal(w, "verify password", verr)
		return
	}
	if err != nil || !ok {
		httpx.WriteError(w, http.StatusUnauthorized, "invalid_credentials", "email or password is incorrect")
		return
	}
	profiles, err := h.Q.ListProfiles(ctx, u.ID)
	if err != nil {
		h.internal(w, "list profiles", err)
		return
	}
	var pid uint64
	if len(profiles) > 0 {
		pid = profiles[0].ID
	}
	// A fresh token on every login prevents session fixation.
	if s, ok := FromContext(ctx); ok {
		_ = h.Q.DeleteSession(ctx, s.TokenHash)
	}
	h.startAndRespond(w, r, u.ID, pid, http.StatusOK)
}

func (h *Handlers) startAndRespond(w http.ResponseWriter, r *http.Request, uid, pid uint64, status int) {
	ctx := r.Context()
	if err := h.Sessions.Start(ctx, w, r, uid, pid); err != nil {
		h.internal(w, "start session", err)
		return
	}
	body, err := h.AfterLogin(ctx, uid, pid)
	if err != nil {
		h.internal(w, "build me", err)
		return
	}
	httpx.WriteJSON(w, status, body)
}

func (h *Handlers) logout(w http.ResponseWriter, r *http.Request) {
	if s, ok := FromContext(r.Context()); ok {
		if err := h.Q.DeleteSession(r.Context(), s.TokenHash); err != nil {
			h.internal(w, "delete session", err)
			return
		}
	}
	h.Sessions.Clear(w)
	w.WriteHeader(http.StatusNoContent)
}

type resetReq struct {
	Email string `json:"email"`
}

// requestReset always answers 202 so it does not reveal registered addresses.
func (h *Handlers) requestReset(w http.ResponseWriter, r *http.Request) {
	var req resetReq
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	ctx := r.Context()
	accepted := func() { httpx.WriteJSON(w, http.StatusAccepted, map[string]string{"status": "reset_sent"}) }
	email, ok := NormalizeEmail(req.Email)
	if !ok {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_email", "invalid email address")
		return
	}
	if !h.limit(w, r, LimitResetIP, httpx.ClientIP(ctx)) || !h.limit(w, r, LimitResetEmail, email) {
		return
	}
	u, err := h.Q.GetUserByEmail(ctx, email)
	if errors.Is(err, sql.ErrNoRows) {
		accepted()
		return
	}
	if err != nil {
		h.internal(w, "lookup user", err)
		return
	}
	now := h.Now()
	plain, hash, err := NewToken()
	if err != nil {
		h.internal(w, "token", err)
		return
	}
	if err := h.Q.InvalidateEmailTokens(ctx, store.InvalidateEmailTokensParams{
		UsedAt: sql.NullTime{Time: now, Valid: true}, UserID: u.ID, Purpose: store.EmailTokensPurposeResetPassword,
	}); err != nil {
		h.internal(w, "invalidate tokens", err)
		return
	}
	if err := h.Q.CreateEmailToken(ctx, store.CreateEmailTokenParams{
		TokenHash: hash, UserID: u.ID, Purpose: store.EmailTokensPurposeResetPassword, ExpiresAt: now.Add(resetTTL),
	}); err != nil {
		h.internal(w, "create token", err)
		return
	}
	if err := h.Outbox.Enqueue(ctx, mailer.ResetPassword(email, h.AppOrigin, plain)); err != nil {
		h.internal(w, "enqueue mail", err)
		return
	}
	accepted()
}

type confirmResetReq struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

// confirmReset sets a new password and signs out every session of the user.
func (h *Handlers) confirmReset(w http.ResponseWriter, r *http.Request) {
	var req confirmResetReq
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	ctx := r.Context()
	if !h.limit(w, r, LimitTokenRedeemIP, httpx.ClientIP(ctx)) {
		return
	}
	if err := ValidatePassword(req.Password); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "weak_password", err.Error())
		return
	}
	invalid := func() {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_token", "this link is invalid or has expired")
	}
	if req.Token == "" {
		invalid()
		return
	}
	pwHash, err := h.Hasher.Hash(ctx, req.Password)
	if err != nil {
		h.internal(w, "hash password", err)
		return
	}

	tx, err := h.DB.BeginTx(ctx, nil)
	if err != nil {
		h.internal(w, "begin", err)
		return
	}
	defer tx.Rollback()
	q := h.Q.WithTx(tx)
	now := h.Now()
	t, err := q.GetEmailTokenForUpdate(ctx, HashToken(req.Token))
	if errors.Is(err, sql.ErrNoRows) {
		invalid()
		return
	}
	if err != nil {
		h.internal(w, "load token", err)
		return
	}
	if t.Purpose != store.EmailTokensPurposeResetPassword || t.UsedAt.Valid || !t.ExpiresAt.After(now) {
		invalid()
		return
	}
	u, err := q.GetUserByID(ctx, t.UserID)
	if err != nil {
		h.internal(w, "load user", err)
		return
	}
	if err := q.UpdatePasswordHash(ctx, store.UpdatePasswordHashParams{PasswordHash: pwHash, ID: t.UserID}); err != nil {
		h.internal(w, "update password", err)
		return
	}
	if err := q.InvalidateEmailTokens(ctx, store.InvalidateEmailTokensParams{
		UsedAt: sql.NullTime{Time: now, Valid: true}, UserID: t.UserID, Purpose: store.EmailTokensPurposeResetPassword,
	}); err != nil {
		h.internal(w, "invalidate tokens", err)
		return
	}
	if err := q.DeleteUserSessions(ctx, t.UserID); err != nil {
		h.internal(w, "revoke sessions", err)
		return
	}
	if err := tx.Commit(); err != nil {
		h.internal(w, "commit", err)
		return
	}
	if err := h.Outbox.Enqueue(ctx, mailer.PasswordChanged(u.Email, h.AppOrigin)); err != nil {
		h.Log.Warn("enqueue password changed mail", "err", err)
	}
	h.Sessions.Clear(w)
	w.WriteHeader(http.StatusNoContent)
}
