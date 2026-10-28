package practice

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/mirinnano/reword/api/internal/auth"
	"github.com/mirinnano/reword/api/internal/platform/httpx"
	"github.com/mirinnano/reword/api/internal/platform/store"
)

const (
	DefaultCount = 10
	MaxCount     = 30
	maxTimeMs    = 10 * 60 * 1000
)

type Handlers struct {
	DB  *sql.DB
	Q   *store.Queries
	Log *slog.Logger
	Now func() time.Time
}

func (h *Handlers) Routes(r chi.Router) {
	r.Get("/practice/types", h.types)
	r.Group(func(r chi.Router) {
		r.Use(requireProfile)
		r.Get("/practice/sessions", h.recent)
		r.Post("/practice/sessions", h.start)
		r.Get("/practice/sessions/{id}", h.get)
		r.Post("/practice/sessions/{id}/answers", h.answer)
		r.Post("/practice/sessions/{id}/finish", h.finish)
	})
}

// requireProfile: practice data belongs to the session's active learner profile.
func requireProfile(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s, ok := auth.FromContext(r.Context()); !ok || s.ActiveProfileID == 0 {
			httpx.WriteError(w, http.StatusConflict, "profile_required", "select a learner profile first")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func profileOf(r *http.Request) uint64 {
	s, _ := auth.FromContext(r.Context())
	return s.ActiveProfileID
}

func (h *Handlers) internal(w http.ResponseWriter, what string, err error) {
	h.Log.Error(what, "err", err)
	httpx.WriteError(w, http.StatusInternalServerError, "internal", "internal error")
}

// ---- views

type TypeView struct {
	Code          string `json:"code"`
	NameJa        string `json:"name_ja"`
	Format        string `json:"format"`
	GradeCode     string `json:"grade_code"`
	GradeNameJa   string `json:"grade_name_ja"`
	QuestionCount int64  `json:"question_count"`
}

type VocabView struct {
	Key        string   `json:"key"`
	Headword   string   `json:"headword"`
	POS        string   `json:"pos"`
	MeaningsJa []string `json:"meanings_ja"`
	ExampleEn  string   `json:"example_en"`
	ExampleJa  string   `json:"example_ja"`
}

// Feedback is present only once an item has been answered, so unanswered
// items never reveal the correct choice.
type Feedback struct {
	Selected      int         `json:"selected"`
	Correct       bool        `json:"correct"`
	CorrectChoice int         `json:"correct_choice"`
	ExplanationJa string      `json:"explanation_ja"`
	Vocab         []VocabView `json:"vocab"`
}

type ItemView struct {
	Position int       `json:"position"`
	Stem     string    `json:"stem"`
	Choices  []string  `json:"choices"`
	Feedback *Feedback `json:"feedback"`
}

type Progress struct {
	Total    int  `json:"total"`
	Answered int  `json:"answered"`
	Correct  int  `json:"correct"`
	Finished bool `json:"finished"`
}

type SessionView struct {
	ID        uint64     `json:"id"`
	TypeCode  string     `json:"type_code"`
	TypeName  string     `json:"type_name_ja"`
	Format    string     `json:"format"`
	StartedAt time.Time  `json:"started_at"`
	Progress  Progress   `json:"progress"`
	Items     []ItemView `json:"items"`
}

type mcqPayload struct {
	Stem    string   `json:"stem"`
	Choices []string `json:"choices"`
}

type mcqAnswer struct {
	Index int `json:"index"`
}

// ---- handlers

func (h *Handlers) types(w http.ResponseWriter, r *http.Request) {
	rows, err := h.Q.ListPracticeTypes(r.Context())
	if err != nil {
		h.internal(w, "list practice types", err)
		return
	}
	out := make([]TypeView, 0, len(rows))
	for _, t := range rows {
		out = append(out, TypeView{Code: t.Code, NameJa: t.NameJa, Format: string(t.Format), GradeCode: t.GradeCode, GradeNameJa: t.GradeNameJa, QuestionCount: t.QuestionCount})
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"types": out})
}

type startReq struct {
	Type  string `json:"type"`
	Count int    `json:"count"`
}

func (h *Handlers) start(w http.ResponseWriter, r *http.Request) {
	var req startReq
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	if req.Count == 0 {
		req.Count = DefaultCount
	}
	if req.Count < 1 || req.Count > MaxCount {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_count", "count must be 1 to 30")
		return
	}
	ctx := r.Context()
	profile := profileOf(r)
	qt, err := h.Q.GetQuestionTypeByCode(ctx, req.Type)
	if errors.Is(err, sql.ErrNoRows) {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_type", "unknown question type")
		return
	}
	if err != nil {
		h.internal(w, "get type", err)
		return
	}
	keys, err := h.Q.ListPublishedQuestionKeys(ctx, qt.ID)
	if err != nil {
		h.internal(w, "list questions", err)
		return
	}
	if len(keys) == 0 {
		httpx.WriteError(w, http.StatusConflict, "no_questions", "no questions available for this type")
		return
	}
	histRows, err := h.Q.ProfileHistoryByType(ctx, store.ProfileHistoryByTypeParams{ProfileID: profile, QuestionTypeID: qt.ID})
	if err != nil {
		h.internal(w, "history", err)
		return
	}
	cands := make([]Candidate, len(keys))
	for i, k := range keys {
		cands[i] = Candidate{ID: k.ID, Key: k.StableKey}
	}
	hist := make(map[string]History, len(histRows))
	for _, hr := range histRows {
		hist[hr.StableKey] = History{Answered: hr.Answered, Correct: hr.Correct, LastUnix: hr.LastAnsweredUnix}
	}
	rng := rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64()))
	ids := Select(cands, hist, req.Count, rng)

	tx, err := h.DB.BeginTx(ctx, nil)
	if err != nil {
		h.internal(w, "begin", err)
		return
	}
	defer tx.Rollback()
	q := h.Q.WithTx(tx)
	aid, err := q.CreateAttempt(ctx, store.CreateAttemptParams{
		ProfileID: profile, QuestionTypeID: qt.ID, ItemCount: uint16(len(ids)), StartedAt: h.Now(),
	})
	if err != nil {
		h.internal(w, "create attempt", err)
		return
	}
	for i, qid := range ids {
		order, _ := json.Marshal(ChoiceOrder(4, rng))
		if err := q.CreateAttemptItem(ctx, store.CreateAttemptItemParams{
			AttemptID: uint64(aid), Position: uint16(i + 1), QuestionID: qid, ChoiceOrder: order,
		}); err != nil {
			h.internal(w, "create item", err)
			return
		}
	}
	if err := tx.Commit(); err != nil {
		h.internal(w, "commit", err)
		return
	}
	view, err := h.view(ctx, uint64(aid), profile)
	if err != nil {
		h.internal(w, "load session", err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, view)
}

func attemptID(w http.ResponseWriter, r *http.Request) (uint64, bool) {
	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id == 0 {
		httpx.WriteError(w, http.StatusNotFound, "not_found", "session not found")
		return 0, false
	}
	return id, true
}

func (h *Handlers) get(w http.ResponseWriter, r *http.Request) {
	id, ok := attemptID(w, r)
	if !ok {
		return
	}
	view, err := h.view(r.Context(), id, profileOf(r))
	if errors.Is(err, sql.ErrNoRows) {
		httpx.WriteError(w, http.StatusNotFound, "not_found", "session not found")
		return
	}
	if err != nil {
		h.internal(w, "load session", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, view)
}

// view loads a session owned by profile. sql.ErrNoRows means not found / not theirs.
func (h *Handlers) view(ctx context.Context, id, profile uint64) (SessionView, error) {
	a, err := h.Q.GetAttempt(ctx, store.GetAttemptParams{ID: id, ProfileID: profile})
	if err != nil {
		return SessionView{}, err
	}
	rows, err := h.Q.ListAttemptItems(ctx, id)
	if err != nil {
		return SessionView{}, err
	}
	var vocabKeys []string
	for _, it := range rows {
		if it.AnsweredAt.Valid {
			var ks []string
			_ = json.Unmarshal(it.VocabKeys, &ks)
			vocabKeys = append(vocabKeys, ks...)
		}
	}
	vocab, err := h.vocab(ctx, vocabKeys)
	if err != nil {
		return SessionView{}, err
	}
	v := SessionView{
		ID: a.ID, TypeCode: a.TypeCode, TypeName: a.TypeNameJa, Format: string(a.TypeFormat), StartedAt: a.StartedAt,
		Progress: Progress{Total: int(a.ItemCount), Answered: int(a.AnsweredCount), Correct: int(a.CorrectCount), Finished: a.FinishedAt.Valid},
		Items:    make([]ItemView, 0, len(rows)),
	}
	for _, it := range rows {
		var p mcqPayload
		var order []int
		if err := json.Unmarshal(it.Payload, &p); err != nil {
			return SessionView{}, err
		}
		if err := json.Unmarshal(it.ChoiceOrder, &order); err != nil {
			return SessionView{}, err
		}
		iv := ItemView{Position: int(it.Position), Stem: p.Stem, Choices: displayChoices(p.Choices, order)}
		if it.AnsweredAt.Valid {
			fb, err := feedback(order, int(it.SelectedOriginal.Int16), it.Answer, it.ExplanationJa, it.VocabKeys, vocab)
			if err != nil {
				return SessionView{}, err
			}
			iv.Feedback = &fb
		}
		v.Items = append(v.Items, iv)
	}
	return v, nil
}

func displayChoices(choices []string, order []int) []string {
	out := make([]string, len(order))
	for i, o := range order {
		out[i] = choices[o]
	}
	return out
}

func feedback(order []int, selectedOriginal int, answerJSON json.RawMessage, expl string, vocabKeysJSON json.RawMessage, vocab map[string]VocabView) (Feedback, error) {
	var ans mcqAnswer
	if err := json.Unmarshal(answerJSON, &ans); err != nil {
		return Feedback{}, err
	}
	var keys []string
	_ = json.Unmarshal(vocabKeysJSON, &keys)
	fb := Feedback{
		Selected:      Displayed(order, selectedOriginal),
		Correct:       selectedOriginal == ans.Index,
		CorrectChoice: Displayed(order, ans.Index),
		ExplanationJa: expl,
		Vocab:         []VocabView{},
	}
	for _, k := range keys {
		if v, ok := vocab[k]; ok {
			fb.Vocab = append(fb.Vocab, v)
		}
	}
	return fb, nil
}

func (h *Handlers) vocab(ctx context.Context, keys []string) (map[string]VocabView, error) {
	out := map[string]VocabView{}
	if len(keys) == 0 {
		return out, nil
	}
	rows, err := h.Q.GetVocabByKeys(ctx, keys)
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		v := VocabView{Key: r.StableKey, Headword: r.Headword, POS: string(r.Pos)}
		var p struct {
			MeaningsJa []string `json:"meanings_ja"`
			ExampleEn  string   `json:"example_en"`
			ExampleJa  string   `json:"example_ja"`
		}
		if err := json.Unmarshal(r.Payload, &p); err != nil {
			return nil, err
		}
		v.MeaningsJa, v.ExampleEn, v.ExampleJa = p.MeaningsJa, p.ExampleEn, p.ExampleJa
		out[r.StableKey] = v
	}
	return out, nil
}

type answerReq struct {
	Position int `json:"position"`
	Choice   int `json:"choice"`
	TimeMs   int `json:"time_ms"`
}

// answer records one answer. Each item can be answered once; answering the
// last item finishes the session.
func (h *Handlers) answer(w http.ResponseWriter, r *http.Request) {
	id, ok := attemptID(w, r)
	if !ok {
		return
	}
	var req answerReq
	if !httpx.DecodeJSON(w, r, &req) {
		return
	}
	if req.Choice < 0 || req.Choice > 3 || req.Position < 1 {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_answer", "choice must be 0 to 3 and position at least 1")
		return
	}
	timeMs := min(max(req.TimeMs, 0), maxTimeMs)
	ctx := r.Context()
	profile := profileOf(r)

	tx, err := h.DB.BeginTx(ctx, nil)
	if err != nil {
		h.internal(w, "begin", err)
		return
	}
	defer tx.Rollback()
	q := h.Q.WithTx(tx)
	a, err := q.GetAttemptForUpdate(ctx, store.GetAttemptForUpdateParams{ID: id, ProfileID: profile})
	if errors.Is(err, sql.ErrNoRows) {
		httpx.WriteError(w, http.StatusNotFound, "not_found", "session not found")
		return
	}
	if err != nil {
		h.internal(w, "load attempt", err)
		return
	}
	if a.FinishedAt.Valid {
		httpx.WriteError(w, http.StatusConflict, "session_finished", "this session is already finished")
		return
	}
	it, err := q.GetAttemptItemForUpdate(ctx, store.GetAttemptItemForUpdateParams{AttemptID: id, Position: uint16(req.Position)})
	if errors.Is(err, sql.ErrNoRows) {
		httpx.WriteError(w, http.StatusNotFound, "not_found", "no item at that position")
		return
	}
	if err != nil {
		h.internal(w, "load item", err)
		return
	}
	if it.AnsweredAt.Valid {
		httpx.WriteError(w, http.StatusConflict, "already_answered", "this question was already answered")
		return
	}
	var order []int
	var ans mcqAnswer
	if err := json.Unmarshal(it.ChoiceOrder, &order); err != nil {
		h.internal(w, "decode order", err)
		return
	}
	if err := json.Unmarshal(it.Answer, &ans); err != nil {
		h.internal(w, "decode answer", err)
		return
	}
	if req.Choice >= len(order) {
		httpx.WriteError(w, http.StatusBadRequest, "invalid_answer", "choice out of range")
		return
	}
	selected := order[req.Choice]
	correct := selected == ans.Index
	now := h.Now()
	if err := q.AnswerAttemptItem(ctx, store.AnswerAttemptItemParams{
		SelectedOriginal: sql.NullInt16{Int16: int16(selected), Valid: true},
		IsCorrect:        sql.NullBool{Bool: correct, Valid: true},
		TimeMs:           sql.NullInt32{Int32: int32(timeMs), Valid: true},
		AnsweredAt:       sql.NullTime{Time: now, Valid: true},
		ID:               it.ID,
	}); err != nil {
		h.internal(w, "answer item", err)
		return
	}
	inc := uint16(0)
	if correct {
		inc = 1
	}
	if err := q.RecordAttemptAnswer(ctx, store.RecordAttemptAnswerParams{CorrectCount: inc, ID: id}); err != nil {
		h.internal(w, "record answer", err)
		return
	}
	finished := a.AnsweredCount+1 >= a.ItemCount
	if finished {
		if err := q.FinishAttempt(ctx, store.FinishAttemptParams{FinishedAt: sql.NullTime{Time: now, Valid: true}, ID: id}); err != nil {
			h.internal(w, "finish", err)
			return
		}
	}
	if err := tx.Commit(); err != nil {
		h.internal(w, "commit", err)
		return
	}

	var keys []string
	_ = json.Unmarshal(it.VocabKeys, &keys)
	vocab, err := h.vocab(ctx, keys)
	if err != nil {
		h.internal(w, "vocab", err)
		return
	}
	fb, err := feedback(order, selected, it.Answer, it.ExplanationJa, it.VocabKeys, vocab)
	if err != nil {
		h.internal(w, "feedback", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"feedback": fb,
		"progress": Progress{
			Total: int(a.ItemCount), Answered: int(a.AnsweredCount) + 1,
			Correct: int(a.CorrectCount) + int(inc), Finished: finished,
		},
	})
}

// finish ends a session early (unanswered items stay unanswered).
func (h *Handlers) finish(w http.ResponseWriter, r *http.Request) {
	id, ok := attemptID(w, r)
	if !ok {
		return
	}
	ctx := r.Context()
	profile := profileOf(r)
	if _, err := h.Q.GetAttempt(ctx, store.GetAttemptParams{ID: id, ProfileID: profile}); errors.Is(err, sql.ErrNoRows) {
		httpx.WriteError(w, http.StatusNotFound, "not_found", "session not found")
		return
	} else if err != nil {
		h.internal(w, "load attempt", err)
		return
	}
	if err := h.Q.FinishAttempt(ctx, store.FinishAttemptParams{FinishedAt: sql.NullTime{Time: h.Now(), Valid: true}, ID: id}); err != nil {
		h.internal(w, "finish", err)
		return
	}
	view, err := h.view(ctx, id, profile)
	if err != nil {
		h.internal(w, "load session", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, view)
}

type SummaryView struct {
	ID        uint64     `json:"id"`
	TypeCode  string     `json:"type_code"`
	TypeName  string     `json:"type_name_ja"`
	StartedAt time.Time  `json:"started_at"`
	EndedAt   *time.Time `json:"finished_at"`
	Progress  Progress   `json:"progress"`
}

func (h *Handlers) recent(w http.ResponseWriter, r *http.Request) {
	limit := 20
	if s := r.URL.Query().Get("limit"); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil || n < 1 || n > 100 {
			httpx.WriteError(w, http.StatusBadRequest, "invalid_limit", "limit must be 1 to 100")
			return
		}
		limit = n
	}
	rows, err := h.Q.ListRecentAttempts(r.Context(), store.ListRecentAttemptsParams{ProfileID: profileOf(r), Limit: int32(limit)})
	if err != nil {
		h.internal(w, "recent", err)
		return
	}
	out := make([]SummaryView, 0, len(rows))
	for _, a := range rows {
		s := SummaryView{
			ID: a.ID, TypeCode: a.TypeCode, TypeName: a.TypeNameJa, StartedAt: a.StartedAt,
			Progress: Progress{Total: int(a.ItemCount), Answered: int(a.AnsweredCount), Correct: int(a.CorrectCount), Finished: a.FinishedAt.Valid},
		}
		if a.FinishedAt.Valid {
			t := a.FinishedAt.Time
			s.EndedAt = &t
		}
		out = append(out, s)
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"sessions": out})
}
