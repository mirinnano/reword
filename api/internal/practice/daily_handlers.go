package practice

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"net/http"
	"time"

	"github.com/go-sql-driver/mysql"

	"github.com/mirinnano/reword/api/internal/platform/httpx"
	"github.com/mirinnano/reword/api/internal/platform/store"
)

const (
	// DailyCode stands in for the type code and format of a daily session.
	DailyCode = "daily"
	dailyName = "今日の日課"

	mysqlDuplicateKey = 1062
)

// DailyView is the home screen's question: is today's routine done, and if
// not, what is in it. Number counts routines ever started and never resets.
type DailyView struct {
	Date    string `json:"date"`
	Number  int    `json:"number"`
	Total   int    `json:"total"`
	Reviews int    `json:"reviews"`
	New     int    `json:"new"`
	Passage bool   `json:"passage"`
	// Session is today's routine once it has been started.
	Session *Progress `json:"session"`
	ID      uint64    `json:"session_id,omitempty"`
}

// dailyPool is what a routine can draw on: the singles, and the passages.
type dailyPool struct {
	singles []Candidate
	members []Member
}

// loadDailyPool keeps what suits the learner's stage. A question they have
// already answered stays whatever the stage, so its review still comes back.
func (h *Handlers) loadDailyPool(ctx context.Context, stage int, states map[string]State) (dailyPool, error) {
	rows, err := h.Q.ListDailyCandidates(ctx)
	if err != nil {
		return dailyPool{}, err
	}
	var p dailyPool
	for _, row := range rows {
		rank, keep := newRank(stage, row.GradeCode, int(row.Difficulty))
		if _, seen := states[row.StableKey]; !keep && !seen {
			continue
		}
		if !isPassage(row.Format) {
			p.singles = append(p.singles, Candidate{ID: row.ID, Key: row.StableKey, Rank: rank})
			continue
		}
		var m mcqPayload
		if err := json.Unmarshal(row.Payload, &m); err != nil {
			return dailyPool{}, fmt.Errorf("question %s: %w", row.StableKey, err)
		}
		p.members = append(p.members, Member{ID: row.ID, Key: row.StableKey, Group: m.Group, Seq: m.Seq})
	}
	return p, nil
}

func (h *Handlers) allStates(ctx context.Context, profile uint64) (map[string]State, error) {
	rows, err := h.Q.ProfileAnswers(ctx, profile)
	if err != nil {
		return nil, err
	}
	byKey := map[string][]Answer{}
	for _, r := range rows {
		byKey[r.StableKey] = append(byKey[r.StableKey], Answer{Correct: r.IsCorrect.Bool, At: time.Unix(r.AnsweredUnix, 0).UTC()})
	}
	out := make(map[string]State, len(byKey))
	for k, as := range byKey {
		out[k] = Fold(as)
	}
	return out, nil
}

func (h *Handlers) todaysAttempt(ctx context.Context, profile uint64, day time.Time) (uint64, error) {
	return h.Q.GetDailyAttempt(ctx, store.GetDailyAttemptParams{
		ProfileID: profile, DailyDate: sql.NullTime{Time: day, Valid: true},
	})
}

func (h *Handlers) dailyStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	profile := profileOf(r)
	now := h.Now()
	day := StudyDay(now)
	count, err := h.Q.CountDailyAttempts(ctx, profile)
	if err != nil {
		h.internal(w, "count daily", err)
		return
	}
	v := DailyView{Date: day.Format(time.DateOnly), Number: int(count) + 1}

	id, err := h.todaysAttempt(ctx, profile, day)
	switch {
	case err == nil:
		view, err := h.view(ctx, id, profile)
		if err != nil {
			h.internal(w, "load daily", err)
			return
		}
		v.Number = int(count)
		v.ID, v.Session = id, &view.Progress
		v.Total = len(view.Items)
		for _, it := range view.Items {
			switch {
			case it.Passage != "":
				v.Passage = true
			case it.Review:
				v.Reviews++
			default:
				v.New++
			}
		}
	case errors.Is(err, sql.ErrNoRows):
		picks, err := h.pickDaily(ctx, profile, int(count), day, now)
		if err != nil {
			h.internal(w, "pick daily", err)
			return
		}
		v.Total = len(picks)
		for _, p := range picks {
			switch {
			case p.Passage:
				v.Passage = true
			case p.Review:
				v.Reviews++
			default:
				v.New++
			}
		}
	default:
		h.internal(w, "get daily", err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, v)
}

// dailyStart returns today's routine, creating it the first time, so a double
// tap or a second device lands on the same session.
func (h *Handlers) dailyStart(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	profile := profileOf(r)
	now := h.Now()
	day := StudyDay(now)

	id, err := h.todaysAttempt(ctx, profile, day)
	status := http.StatusOK
	switch {
	case err == nil:
	case errors.Is(err, sql.ErrNoRows):
		status = http.StatusCreated
		id, err = h.createDaily(ctx, profile, day, now)
		var dup *mysql.MySQLError
		if errors.As(err, &dup) && dup.Number == mysqlDuplicateKey {
			status = http.StatusOK
			id, err = h.todaysAttempt(ctx, profile, day)
		}
		if errors.Is(err, errNoQuestions) {
			httpx.WriteError(w, http.StatusConflict, "no_questions", "no questions available for this type")
			return
		}
		if err != nil {
			h.internal(w, "create daily", err)
			return
		}
	default:
		h.internal(w, "get daily", err)
		return
	}
	view, err := h.view(ctx, id, profile)
	if err != nil {
		h.internal(w, "load daily", err)
		return
	}
	httpx.WriteJSON(w, status, view)
}

var errNoQuestions = errors.New("no questions")

// pickDaily seeds the choice with the profile and the day, so the plan the
// home screen shows is the routine that starting it creates.
func (h *Handlers) pickDaily(ctx context.Context, profile uint64, count int, day, now time.Time) ([]Pick, error) {
	stage, placed, err := h.stage(ctx, profile)
	if err != nil {
		return nil, err
	}
	if !placed {
		stage = StageDefault
	}
	states, err := h.allStates(ctx, profile)
	if err != nil {
		return nil, err
	}
	pool, err := h.loadDailyPool(ctx, stage, states)
	if err != nil {
		return nil, err
	}
	var members []Member
	if PassageDue(count) && PassagesAllowed(stage) {
		members = pool.members
	}
	rng := rand.New(rand.NewPCG(profile, uint64(day.Unix())))
	return SelectDaily(pool.singles, members, states, DailySize, now, rng), nil
}

func (h *Handlers) createDaily(ctx context.Context, profile uint64, day, now time.Time) (uint64, error) {
	count, err := h.Q.CountDailyAttempts(ctx, profile)
	if err != nil {
		return 0, err
	}
	picks, err := h.pickDaily(ctx, profile, int(count), day, now)
	if err != nil {
		return 0, err
	}
	if len(picks) == 0 {
		return 0, errNoQuestions
	}
	rng := rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64()))

	tx, err := h.DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	q := h.Q.WithTx(tx)
	aid, err := q.CreateDailyAttempt(ctx, store.CreateDailyAttemptParams{
		ProfileID: profile, DailyDate: sql.NullTime{Time: day, Valid: true}, ItemCount: uint16(len(picks)), StartedAt: now,
	})
	if err != nil {
		return 0, err
	}
	if err := createItems(ctx, q, uint64(aid), picks, rng); err != nil {
		return 0, err
	}
	return uint64(aid), tx.Commit()
}
