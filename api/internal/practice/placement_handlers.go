package practice

import (
	"context"
	"database/sql"
	"errors"
	"math/rand/v2"
	"net/http"

	"github.com/mirinnano/reword/api/internal/platform/httpx"
	"github.com/mirinnano/reword/api/internal/platform/store"
)

// PlacementView says where a learner stands. Stage is null until they finish a
// test; SessionID is set while one is in progress.
type PlacementView struct {
	Stage     *int   `json:"stage"`
	SessionID uint64 `json:"session_id,omitempty"`
}

func (h *Handlers) stage(ctx context.Context, profile uint64) (int, bool, error) {
	s, err := h.Q.GetPlacementStage(ctx, profile)
	if err != nil {
		return 0, false, err
	}
	return int(s.Int16), s.Valid, nil
}

func (h *Handlers) placementStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	profile := profileOf(r)
	var v PlacementView
	stage, placed, err := h.stage(ctx, profile)
	if err != nil {
		h.internal(w, "get stage", err)
		return
	}
	if placed {
		v.Stage = &stage
	}
	id, err := h.Q.GetOpenPlacement(ctx, profile)
	switch {
	case err == nil:
		v.SessionID = id
	case !errors.Is(err, sql.ErrNoRows):
		h.internal(w, "open placement", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, v)
}

// placementStart resumes the sitting in progress or begins a new one.
func (h *Handlers) placementStart(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	profile := profileOf(r)
	status := http.StatusOK
	id, err := h.Q.GetOpenPlacement(ctx, profile)
	if errors.Is(err, sql.ErrNoRows) {
		status = http.StatusCreated
		id, err = h.createPlacement(ctx, profile)
	}
	if errors.Is(err, errPlacementPool) {
		httpx.WriteError(w, http.StatusConflict, "no_questions", "no questions available for this type")
		return
	}
	if err != nil {
		h.internal(w, "start placement", err)
		return
	}
	view, err := h.view(ctx, id, profile)
	if err != nil {
		h.internal(w, "load placement", err)
		return
	}
	httpx.WriteJSON(w, status, view)
}

func (h *Handlers) createPlacement(ctx context.Context, profile uint64) (uint64, error) {
	rows, err := h.Q.ListPlacementCandidates(ctx)
	if err != nil {
		return 0, err
	}
	pool := make([]PlacementQuestion, len(rows))
	for i, row := range rows {
		pool[i] = PlacementQuestion{ID: row.ID, Type: row.TypeCode, Difficulty: int(row.Difficulty)}
	}
	rng := rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64()))
	picks, err := SelectPlacement(pool, rng)
	if err != nil {
		return 0, err
	}
	tx, err := h.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	q := h.Q.WithTx(tx)
	aid, err := q.CreatePlacementAttempt(ctx, store.CreatePlacementAttemptParams{
		ProfileID: profile, ItemCount: uint16(len(picks)), StartedAt: h.Now(),
	})
	if err != nil {
		return 0, err
	}
	if err := createItems(ctx, q, uint64(aid), picks, rng); err != nil {
		return 0, err
	}
	return uint64(aid), tx.Commit()
}

// scoreItems reads a sitting's items as ScorePlacement wants them.
func scoreItems(rows []store.ListAttemptItemsRow) PlacementResult {
	answers := make([]PlacementAnswer, len(rows))
	for i, it := range rows {
		answers[i] = PlacementAnswer{
			Answered: it.AnsweredAt.Valid,
			DontKnow: it.AnsweredAt.Valid && !it.SelectedOriginal.Valid,
			Correct:  it.IsCorrect.Bool,
		}
	}
	return ScorePlacement(answers)
}

// settlePlacement runs inside the answer's transaction. Once the score is
// decided it ends the sitting, however many questions are left, and records
// the stage on the profile.
func settlePlacement(ctx context.Context, q *store.Queries, attempt, profile uint64) (PlacementResult, error) {
	rows, err := q.ListAttemptItems(ctx, attempt)
	if err != nil {
		return PlacementResult{}, err
	}
	res := scoreItems(rows)
	if !res.Decided {
		return res, nil
	}
	return res, q.SetPlacementStage(ctx, store.SetPlacementStageParams{
		PlacementStage: sql.NullInt16{Int16: int16(res.Stage), Valid: true}, ID: profile,
	})
}
