// Package account serves the signed-in user's own data: who they are and
// the learner profiles they switch between.
package account

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/mirinnano/reword/api/internal/auth"
	"github.com/mirinnano/reword/api/internal/platform/httpx"
	"github.com/mirinnano/reword/api/internal/platform/store"
)

const MaxProfiles = 5

type Handlers struct {
	Q   *store.Queries
	Log *slog.Logger
}

// Public routes.
func (h *Handlers) PublicRoutes(r chi.Router) {
	r.Get("/grades", h.grades)
}

// Routes that require a session.
func (h *Handlers) Routes(r chi.Router) {
	r.Get("/me", h.me)
	r.Get("/profiles", h.listProfiles)
	r.Post("/profiles", h.createProfile)
	r.Patch("/profiles/{id}", h.updateProfile)
	r.Delete("/profiles/{id}", h.deleteProfile)
	r.Post("/profiles/{id}/select", h.selectProfile)
}

type Grade struct {
	ID     uint8  `json:"id"`
	Code   string `json:"code"`
	NameJa string `json:"name_ja"`
}

type Profile struct {
	ID            uint64 `json:"id"`
	DisplayName   string `json:"display_name"`
	TargetGradeID *uint8 `json:"target_grade_id"`
}

type User struct {
	ID    uint64 `json:"id"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

type Me struct {
	User            User      `json:"user"`
	Profiles        []Profile `json:"profiles"`
	ActiveProfileID *uint64   `json:"active_profile_id"`
}

func (h *Handlers) internal(w http.ResponseWriter, what string, err error) {
	h.Log.Error(what, "err", err)
	httpx.WriteError(w, http.StatusInternalServerError, "internal", "internal error")
}

func (h *Handlers) grades(w http.ResponseWriter, r *http.Request) {
	rows, err := h.Q.ListGrades(r.Context())
	if err != nil {
		h.internal(w, "list grades", err)
		return
	}
	out := []Grade{}
	for _, g := range rows {
		if g.IsActive {
			out = append(out, Grade{ID: g.ID, Code: g.Code, NameJa: g.NameJa})
		}
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"grades": out})
}

// BuildMe is shared with the login and signup-verify responses.
func (h *Handlers) BuildMe(ctx context.Context, userID, activeProfileID uint64) (any, error) {
	u, err := h.Q.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	profiles, err := h.profiles(ctx, userID)
	if err != nil {
		return nil, err
	}
	me := Me{User: User{ID: u.ID, Email: u.Email, Role: string(u.Role)}, Profiles: profiles}
	if activeProfileID != 0 {
		me.ActiveProfileID = &activeProfileID
	}
	return me, nil
}

func (h *Handlers) profiles(ctx context.Context, userID uint64) ([]Profile, error) {
	rows, err := h.Q.ListProfiles(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]Profile, 0, len(rows))
	for _, p := range rows {
		out = append(out, toProfile(p.ID, p.DisplayName, p.TargetGradeID))
	}
	return out, nil
}

func toProfile(id uint64, name string, grade sql.NullInt16) Profile {
	p := Profile{ID: id, DisplayName: name}
	if grade.Valid {
		g := uint8(grade.Int16)
		p.TargetGradeID = &g
	}
	return p
}

func (h *Handlers) me(w http.ResponseWriter, r *http.Request) {
	s, _ := auth.FromContext(r.Context())
	body, err := h.BuildMe(r.Context(), s.UserID, s.ActiveProfileID)
	if err != nil {
		h.internal(w, "build me", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, body)
}

func (h *Handlers) listProfiles(w http.ResponseWriter, r *http.Request) {
	s, _ := auth.FromContext(r.Context())
	profiles, err := h.profiles(r.Context(), s.UserID)
	if err != nil {
		h.internal(w, "list profiles", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"profiles": profiles})
}

type profileReq struct {
	DisplayName   string `json:"display_name"`
	TargetGradeID *uint8 `json:"target_grade_id"`
}

func (h *Handlers) parseProfile(w http.ResponseWriter, r *http.Request) (string, sql.NullInt16, bool) {
	var req profileReq
	if !httpx.DecodeJSON(w, r, &req) {
		return "", sql.NullInt16{}, false
	}
	name, ok := auth.ValidDisplayName(req.DisplayName)
	if !ok {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_display_name", "display name must be 1 to 40 characters")
		return "", sql.NullInt16{}, false
	}
	if req.TargetGradeID == nil {
		return name, sql.NullInt16{}, true
	}
	active, err := h.Q.GradeIsActive(r.Context(), *req.TargetGradeID)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && !active) {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_grade", "unknown or unavailable grade")
		return "", sql.NullInt16{}, false
	}
	if err != nil {
		h.internal(w, "grade lookup", err)
		return "", sql.NullInt16{}, false
	}
	return name, sql.NullInt16{Int16: int16(*req.TargetGradeID), Valid: true}, true
}

func (h *Handlers) createProfile(w http.ResponseWriter, r *http.Request) {
	s, _ := auth.FromContext(r.Context())
	name, grade, ok := h.parseProfile(w, r)
	if !ok {
		return
	}
	n, err := h.Q.CountProfiles(r.Context(), s.UserID)
	if err != nil {
		h.internal(w, "count profiles", err)
		return
	}
	if n >= MaxProfiles {
		httpx.WriteError(w, http.StatusConflict, "profile_limit", "up to 5 profiles per account")
		return
	}
	id, err := h.Q.CreateProfile(r.Context(), store.CreateProfileParams{UserID: s.UserID, DisplayName: name, TargetGradeID: grade})
	if err != nil {
		h.internal(w, "create profile", err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, toProfile(uint64(id), name, grade))
}

// profileID parses {id}; ownership is enforced by every query's user_id filter.
func profileID(w http.ResponseWriter, r *http.Request) (uint64, bool) {
	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id == 0 {
		httpx.WriteError(w, http.StatusNotFound, "not_found", "profile not found")
		return 0, false
	}
	return id, true
}

func notFound(w http.ResponseWriter) {
	httpx.WriteError(w, http.StatusNotFound, "not_found", "profile not found")
}

func (h *Handlers) updateProfile(w http.ResponseWriter, r *http.Request) {
	s, _ := auth.FromContext(r.Context())
	id, ok := profileID(w, r)
	if !ok {
		return
	}
	name, grade, ok := h.parseProfile(w, r)
	if !ok {
		return
	}
	if _, err := h.Q.GetProfile(r.Context(), store.GetProfileParams{ID: id, UserID: s.UserID}); errors.Is(err, sql.ErrNoRows) {
		notFound(w)
		return
	} else if err != nil {
		h.internal(w, "get profile", err)
		return
	}
	if _, err := h.Q.UpdateProfile(r.Context(), store.UpdateProfileParams{
		DisplayName: name, TargetGradeID: grade, ID: id, UserID: s.UserID,
	}); err != nil {
		h.internal(w, "update profile", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toProfile(id, name, grade))
}

func (h *Handlers) deleteProfile(w http.ResponseWriter, r *http.Request) {
	s, _ := auth.FromContext(r.Context())
	id, ok := profileID(w, r)
	if !ok {
		return
	}
	n, err := h.Q.CountProfiles(r.Context(), s.UserID)
	if err != nil {
		h.internal(w, "count profiles", err)
		return
	}
	if n <= 1 {
		httpx.WriteError(w, http.StatusConflict, "last_profile", "an account needs at least one profile")
		return
	}
	rows, err := h.Q.DeleteProfile(r.Context(), store.DeleteProfileParams{ID: id, UserID: s.UserID})
	if err != nil {
		h.internal(w, "delete profile", err)
		return
	}
	if rows == 0 {
		notFound(w)
		return
	}
	// sessions.active_profile_id is cleared by ON DELETE SET NULL.
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) selectProfile(w http.ResponseWriter, r *http.Request) {
	s, _ := auth.FromContext(r.Context())
	id, ok := profileID(w, r)
	if !ok {
		return
	}
	p, err := h.Q.GetProfile(r.Context(), store.GetProfileParams{ID: id, UserID: s.UserID})
	if errors.Is(err, sql.ErrNoRows) {
		notFound(w)
		return
	}
	if err != nil {
		h.internal(w, "get profile", err)
		return
	}
	if err := h.Q.SetSessionProfile(r.Context(), store.SetSessionProfileParams{
		ActiveProfileID: sql.NullInt64{Int64: int64(p.ID), Valid: true}, TokenHash: s.TokenHash,
	}); err != nil {
		h.internal(w, "select profile", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toProfile(p.ID, p.DisplayName, p.TargetGradeID))
}
