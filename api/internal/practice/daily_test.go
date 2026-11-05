package practice

import (
	"testing"
	"time"
)

func TestStudyDay(t *testing.T) {
	t.Parallel()
	jstAt := func(d, h int) time.Time { return time.Date(2026, 11, d, h, 0, 0, 0, jst) }
	tests := []struct {
		name string
		at   time.Time
		want int
	}{
		{"evening", jstAt(5, 21), 5},
		{"after midnight still counts for the day before", jstAt(6, 3), 5},
		{"at the boundary a new day starts", jstAt(6, 4), 6},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := StudyDay(tt.at); got.Day() != tt.want {
				t.Errorf("StudyDay(%v) = %v, want day %d", tt.at, got, tt.want)
			}
		})
	}
}

func TestSplit(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name               string
		due, unseen, n     int
		wantReviews, wantN int
	}{
		{"quiet queue fills up with new", 2, 100, 12, 2, 10},
		{"reviews fit in one routine: a third new", 12, 100, 12, 8, 4},
		{"reviews fit in two: a sixth new", 24, 100, 12, 10, 2},
		{"crowded queue: reviews only", 25, 100, 12, 12, 0},
		{"nothing new left", 3, 0, 12, 3, 0},
		{"no reviews", 0, 100, 12, 0, 12},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if r, f := split(tt.due, tt.unseen, tt.n); r != tt.wantReviews || f != tt.wantN {
				t.Errorf("split() = %d reviews, %d new; want %d, %d", r, f, tt.wantReviews, tt.wantN)
			}
		})
	}
}

func TestSelectDaily(t *testing.T) {
	t.Parallel()
	now := day(30)
	missed := Fold([]Answer{{false, day(29)}})

	t.Run("passage comes whole at the end and counts toward the size", func(t *testing.T) {
		t.Parallel()
		ms := members(3, 4)
		got := SelectDaily(cands(40), ms, nil, DailySize, now, rng())
		if len(got) != DailySize {
			t.Fatalf("len = %d, want %d", len(got), DailySize)
		}
		for i, p := range got {
			if p.Passage != (i >= DailySize-4) {
				t.Fatalf("position %d passage = %v: %+v", i, p.Passage, got)
			}
		}
	})

	t.Run("reviews are marked, new questions are not", func(t *testing.T) {
		t.Parallel()
		states := map[string]State{}
		for i := 1; i <= 6; i++ {
			states[key(i)] = missed
		}
		reviews := 0
		for _, p := range SelectDaily(cands(40), nil, states, DailySize, now, rng()) {
			if p.Review != (p.ID <= 6) {
				t.Fatalf("pick %+v", p)
			}
			if p.Review {
				reviews++
			}
		}
		if reviews != 6 {
			t.Errorf("reviews = %d, want 6", reviews)
		}
	})

	t.Run("a crowded queue brings nothing new", func(t *testing.T) {
		t.Parallel()
		states := map[string]State{}
		for i := 1; i <= 30; i++ {
			states[key(i)] = missed
		}
		for _, p := range SelectDaily(cands(40), nil, states, DailySize, now, rng()) {
			if !p.Review {
				t.Fatalf("new question %+v in a crowded queue", p)
			}
		}
	})

	t.Run("seen everything: still a full routine, distinct", func(t *testing.T) {
		t.Parallel()
		states := map[string]State{}
		for i := 1; i <= 20; i++ {
			states[key(i)] = Fold([]Answer{{true, day(29)}})
		}
		got := SelectDaily(cands(20), nil, states, DailySize, now, rng())
		seen := map[uint64]bool{}
		for _, p := range got {
			if seen[p.ID] {
				t.Fatalf("duplicate %d", p.ID)
			}
			seen[p.ID] = true
		}
		if len(got) != DailySize {
			t.Errorf("len = %d, want %d", len(got), DailySize)
		}
	})
}

// meanQueue replays a learner answering 70% of questions for half a year and
// returns the average review queue over the last 120 days.
func meanQueue(pick func(cs []Candidate, states map[string]State, now time.Time) []uint64) int {
	r := rng()
	answers := map[string][]Answer{}
	cs := cands(1000)
	sum := 0
	for d := range 180 {
		now := day(d)
		states := map[string]State{}
		for k, as := range answers {
			states[k] = Fold(as)
		}
		if _, due, _ := sortCandidates(cs, states, now, r); d >= 60 {
			sum += len(due)
		}
		for _, id := range pick(cs, states, now) {
			k := key(int(id))
			answers[k] = append(answers[k], Answer{r.Float64() < 0.7, now})
		}
	}
	return sum / 120
}

// The reason this rule exists: under a fixed review share the queue grows
// without end, because each new question costs several reviews. The fixed
// share is run too, so this test can tell the two apart.
func TestDailyQueueStaysBounded(t *testing.T) {
	t.Parallel()
	r := rng()
	daily := meanQueue(func(cs []Candidate, states map[string]State, now time.Time) []uint64 {
		var ids []uint64
		for _, p := range SelectDaily(cs, nil, states, DailySize, now, r) {
			ids = append(ids, p.ID)
		}
		return ids
	})
	fixed := meanQueue(func(cs []Candidate, states map[string]State, now time.Time) []uint64 {
		return Select(cs, states, DailySize, now, r)
	})
	if daily > 4*DailySize {
		t.Errorf("mean queue with the daily rule = %d, want at most %d", daily, 4*DailySize)
	}
	if fixed < 3*daily {
		t.Errorf("mean queue with a fixed share = %d, daily rule = %d; the test no longer tells them apart", fixed, daily)
	}
	t.Logf("mean queue: daily rule %d, fixed share %d", daily, fixed)
}
