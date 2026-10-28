package practice

import (
	"fmt"
	"math/rand/v2"
	"testing"
	"time"
)

var t0 = time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)

func day(n int) time.Time { return t0.Add(time.Duration(n) * 24 * time.Hour) }

func TestFold(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		answers    []Answer
		wantStreak int
		wantMissed bool
		wantDue    time.Time
		mastered   bool
	}{
		{name: "one correct", answers: []Answer{{true, day(0)}}, wantStreak: 1, wantDue: day(1)},
		{name: "one wrong is due at once", answers: []Answer{{false, day(0)}}, wantStreak: 0, wantMissed: true, wantDue: day(0)},
		{
			name:       "spaced corrects grow the interval",
			answers:    []Answer{{true, day(0)}, {true, day(1)}, {true, day(4)}},
			wantStreak: 3, wantDue: day(11), mastered: true,
		},
		{
			// External audit F4: cramming in one sitting must not master an item.
			name:       "same-day corrects do not raise the streak",
			answers:    []Answer{{true, day(0)}, {true, day(0)}, {true, day(0)}, {true, day(0)}},
			wantStreak: 1, wantDue: day(1),
		},
		{
			// External audit F1: a past mistake must not mark the item forever.
			name:       "recovery after a mistake",
			answers:    []Answer{{false, day(0)}, {true, day(0)}, {true, day(1)}, {true, day(4)}},
			wantStreak: 3, wantDue: day(11), mastered: true,
		},
		{
			name:       "a wrong answer resets a long streak",
			answers:    []Answer{{true, day(0)}, {true, day(1)}, {true, day(4)}, {false, day(11)}},
			wantStreak: 0, wantMissed: true, wantDue: day(11),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := Fold(tt.answers)
			if s.Streak != tt.wantStreak || s.Missed != tt.wantMissed || !s.Due.Equal(tt.wantDue) || s.Mastered() != tt.mastered {
				t.Errorf("Fold() = streak %d missed %v due %v mastered %v; want %d %v %v %v",
					s.Streak, s.Missed, s.Due, s.Mastered(), tt.wantStreak, tt.wantMissed, tt.wantDue, tt.mastered)
			}
		})
	}
}

func cands(n int) []Candidate {
	out := make([]Candidate, n)
	for i := range out {
		out[i] = Candidate{ID: uint64(i + 1), Key: fmt.Sprintf("k%03d", i+1)}
	}
	return out
}

func key(id int) string { return fmt.Sprintf("k%03d", id) }

func rng() *rand.Rand { return rand.New(rand.NewPCG(1, 2)) }

func count(ids []uint64, in func(uint64) bool) int {
	n := 0
	for _, id := range ids {
		if in(id) {
			n++
		}
	}
	return n
}

func TestSelect(t *testing.T) {
	t.Parallel()
	now := day(30)
	missedAt := Fold([]Answer{{false, day(29)}})
	correctDue := Fold([]Answer{{true, day(20)}})              // due on day 21, so overdue
	correctLater := Fold([]Answer{{true, day(29)}})            // due on day 30, exactly now
	notDue := Fold([]Answer{{true, day(25)}, {true, day(27)}}) // streak 2, due on day 30 + 3

	t.Run("reviews capped at half while unseen remain", func(t *testing.T) {
		t.Parallel()
		states := map[string]State{}
		for i := 1; i <= 10; i++ {
			states[key(i)] = missedAt
		}
		got := Select(cands(30), states, 10, now, rng())
		if reviews := count(got, func(id uint64) bool { return id <= 10 }); reviews != 5 {
			t.Errorf("reviews = %d, want 5", reviews)
		}
	})

	t.Run("five-question session keeps the cap (audit F17)", func(t *testing.T) {
		t.Parallel()
		states := map[string]State{}
		for i := 1; i <= 10; i++ {
			states[key(i)] = missedAt
		}
		got := Select(cands(30), states, 5, now, rng())
		if reviews := count(got, func(id uint64) bool { return id <= 10 }); reviews != 2 {
			t.Errorf("reviews = %d, want 2", reviews)
		}
	})

	t.Run("missed reviews come before correct ones", func(t *testing.T) {
		t.Parallel()
		states := map[string]State{key(1): correctDue, key(2): correctDue, key(3): missedAt, key(4): missedAt}
		got := Select(cands(20), states, 4, now, rng())
		if count(got, func(id uint64) bool { return id == 3 || id == 4 }) != 2 {
			t.Errorf("Select() = %v, want both missed items (3, 4)", got)
		}
	})

	t.Run("correct items return once unseen run out (audit F2)", func(t *testing.T) {
		t.Parallel()
		states := map[string]State{}
		for i := 1; i <= 10; i++ {
			states[key(i)] = missedAt
		}
		for i := 11; i <= 20; i++ {
			states[key(i)] = correctDue
		}
		got := Select(cands(20), states, 20, now, rng())
		if c := count(got, func(id uint64) bool { return id > 10 }); c != 10 {
			t.Errorf("due correct items selected = %d, want 10", c)
		}
	})

	t.Run("not-yet-due items fill the rest, earliest first", func(t *testing.T) {
		t.Parallel()
		states := map[string]State{key(1): notDue, key(2): correctLater, key(3): notDue}
		got := Select(cands(3), states, 2, now, rng())
		if len(got) != 2 || count(got, func(id uint64) bool { return id == 2 }) != 1 {
			t.Errorf("Select() = %v, want 2 items including the due one (2)", got)
		}
	})

	t.Run("distinct ids and short pool", func(t *testing.T) {
		t.Parallel()
		got := Select(cands(3), nil, 10, now, rng())
		seen := map[uint64]bool{}
		for _, id := range got {
			if seen[id] {
				t.Fatalf("duplicate id %d in %v", id, got)
			}
			seen[id] = true
		}
		if len(got) != 3 {
			t.Errorf("len = %d, want 3", len(got))
		}
	})
}

func TestChoiceOrderRoundTrip(t *testing.T) {
	t.Parallel()
	r := rng()
	for range 50 {
		order := ChoiceOrder(4, r)
		for orig := range 4 {
			if order[Displayed(order, orig)] != orig {
				t.Fatalf("round trip failed for %v", order)
			}
		}
	}
}
