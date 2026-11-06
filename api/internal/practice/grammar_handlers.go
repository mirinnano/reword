package practice

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"net/http"
	"sort"

	"github.com/go-chi/chi/v5"

	"github.com/mirinnano/reword/api/internal/content"
	"github.com/mirinnano/reword/api/internal/platform/httpx"
	"github.com/mirinnano/reword/api/internal/platform/store"
)

// GrammarUnitView is one row of the unit list. Best is null until a sitting of
// the unit is finished.
type GrammarUnitView struct {
	Key   string `json:"key"`
	Group string `json:"group"`
	Title string `json:"title"`
	Best  *int   `json:"best"`
	Done  bool   `json:"done"`
}

// grammarUnits lists every unit in order. Next is the first unit not done: a
// suggestion, since nothing is locked.
func (h *Handlers) grammarUnits(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	rows, err := h.Q.ListGrammarUnits(ctx)
	if err != nil {
		h.internal(w, "list units", err)
		return
	}
	results, err := h.Q.ListGrammarResults(ctx, profileOf(r))
	if err != nil {
		h.internal(w, "grammar results", err)
		return
	}
	best := make(map[string]int, len(results))
	for _, b := range results {
		best[b.GrammarUnit.String] = int(b.Best)
	}
	out := make([]GrammarUnitView, len(rows))
	next := ""
	for i, u := range rows {
		v := GrammarUnitView{Key: u.StableKey, Group: u.GroupJa, Title: u.Title}
		if b, ok := best[u.StableKey]; ok {
			v.Best = &b
			v.Done = b >= content.GrammarPass
		}
		if !v.Done && next == "" {
			next = v.Key
		}
		out[i] = v
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"units": out, "next": next, "pass": content.GrammarPass})
}

func (h *Handlers) grammarUnit(w http.ResponseWriter, r *http.Request) {
	u, err := h.Q.GetGrammarUnit(r.Context(), chi.URLParam(r, "key"))
	if errors.Is(err, sql.ErrNoRows) {
		httpx.WriteError(w, http.StatusNotFound, "not_found", "no such grammar unit")
		return
	}
	if err != nil {
		h.internal(w, "get unit", err)
		return
	}
	var body struct {
		RuleJa   string          `json:"rule_ja"`
		Examples json.RawMessage `json:"examples"`
	}
	if err := json.Unmarshal(u.Payload, &body); err != nil {
		h.internal(w, "decode unit", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"key": u.StableKey, "group": u.GroupJa, "title": u.Title, "rule_ja": body.RuleJa, "examples": body.Examples,
	})
}

// grammarStart begins a sitting of a unit's check questions, in their authored
// order. A learner can sit a unit as often as they like; the best one counts.
func (h *Handlers) grammarStart(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	key := chi.URLParam(r, "key")
	if _, err := h.Q.GetGrammarUnit(ctx, key); errors.Is(err, sql.ErrNoRows) {
		httpx.WriteError(w, http.StatusNotFound, "not_found", "no such grammar unit")
		return
	} else if err != nil {
		h.internal(w, "get unit", err)
		return
	}
	id, err := h.createGrammarAttempt(ctx, profileOf(r), key)
	if errors.Is(err, errNoQuestions) {
		httpx.WriteError(w, http.StatusConflict, "no_questions", "no questions available for this type")
		return
	}
	if err != nil {
		h.internal(w, "start unit", err)
		return
	}
	view, err := h.view(ctx, id, profileOf(r))
	if err != nil {
		h.internal(w, "load session", err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, view)
}

func (h *Handlers) createGrammarAttempt(ctx context.Context, profile uint64, unit string) (uint64, error) {
	qt, err := h.Q.GetQuestionTypeByCode(ctx, content.GrammarType)
	if err != nil {
		return 0, err
	}
	rows, err := h.Q.ListPublishedQuestionPayloads(ctx, qt.ID)
	if err != nil {
		return 0, err
	}
	type member struct {
		id  uint64
		seq int
	}
	var members []member
	for _, row := range rows {
		var p mcqPayload
		if err := json.Unmarshal(row.Payload, &p); err != nil {
			return 0, fmt.Errorf("question %s: %w", row.StableKey, err)
		}
		if p.Group == unit {
			members = append(members, member{row.ID, p.Seq})
		}
	}
	if len(members) == 0 {
		return 0, errNoQuestions
	}
	sort.Slice(members, func(i, j int) bool { return members[i].seq < members[j].seq })
	picks := make([]Pick, len(members))
	for i, m := range members {
		picks[i] = Pick{ID: m.id}
	}
	tx, err := h.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	q := h.Q.WithTx(tx)
	aid, err := q.CreateGrammarAttempt(ctx, store.CreateGrammarAttemptParams{
		ProfileID: profile, QuestionTypeID: sql.NullInt16{Int16: int16(qt.ID), Valid: true},
		GrammarUnit: sql.NullString{String: unit, Valid: true}, ItemCount: uint16(len(picks)), StartedAt: h.Now(),
	})
	if err != nil {
		return 0, err
	}
	if err := createItems(ctx, q, uint64(aid), picks, rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64()))); err != nil {
		return 0, err
	}
	return uint64(aid), tx.Commit()
}
