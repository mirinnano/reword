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
		{name: "first correct skips to a week", answers: []Answer{{true, day(0)}}, wantStreak: 1, wantDue: day(7)},
		{name: "one wrong is due at once", answers: []Answer{{false, day(0)}}, wantStreak: 0, wantMissed: true, wantDue: day(0)},
		{
			name:       "spaced corrects climb the ladder",
			answers:    []Answer{{true, day(0)}, {true, day(7)}, {true, day(37)}},
			wantStreak: 3, wantDue: day(127), mastered: true,
		},
		{
			// External audit F4: cramming in one sitting must not master an item.
			name:       "same-day corrects change nothing",
			answers:    []Answer{{true, day(0)}, {true, day(0)}, {true, day(0)}, {true, day(0)}},
			wantStreak: 1, wantDue: day(7),
		},
		{
			// External audit F1: a past mistake must not mark the item forever.
			name:       "recovery after a mistake starts at the bottom",
			answers:    []Answer{{false, day(0)}, {true, day(0)}, {true, day(1)}, {true, day(5)}},
			wantStreak: 3, wantDue: day(15), mastered: true,
		},
		{
			name:       "a wrong answer resets a long streak",
			answers:    []Answer{{true, day(0)}, {true, day(7)}, {true, day(37)}, {false, day(127)}},
			wantStreak: 0, wantMissed: true, wantDue: day(127),
		},
		{
			// The skip must not count as progress toward mastery.
			name:       "one lucky guess is not mastery",
			answers:    []Answer{{true, day(0)}, {false, day(7)}},
			wantStreak: 0, wantMissed: true, wantDue: day(7),
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
	correctDue := Fold([]Answer{{true, day(20)}})              // due on day 27, so overdue
	correctLater := Fold([]Answer{{true, day(23)}})            // due on day 30, exactly now
	notDue := Fold([]Answer{{true, day(25)}, {true, day(27)}}) // an early review; due on day 32

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

func members(passages, each int) []Member {
	var out []Member
	id := uint64(0)
	for p := 1; p <= passages; p++ {
		// Added last so the input is not already in Seq order.
		for q := each; q >= 1; q-- {
			id++
			out = append(out, Member{ID: id, Key: fmt.Sprintf("p%d-%d", p, q), Group: fmt.Sprintf("p%d", p), Seq: q})
		}
	}
	return out
}

func TestSelectPassages(t *testing.T) {
	t.Parallel()
	ms := members(5, 3)
	byID := map[uint64]Member{}
	for _, m := range ms {
		byID[m.ID] = m
	}
	wrongOn := func(key string) map[string]State {
		s := map[string]State{}
		for _, m := range ms {
			if m.Group == "p2" {
				s[m.Key] = Fold([]Answer{{true, day(-1)}})
			}
		}
		s[key] = Fold([]Answer{{false, day(0)}})
		return s
	}
	tests := []struct {
		name   string
		states map[string]State
		n      int
		want   []string // passages that must be chosen
		count  int      // passages chosen
	}{
		{name: "unseen passages come whole", n: 2, count: 2},
		{name: "more than exist", n: 9, count: 5},
		{name: "one missed question brings its passage back", states: wrongOn("p2-2"), n: 1, want: []string{"p2"}, count: 1},
		{
			name:   "an unfinished passage comes back",
			states: map[string]State{"p4-1": Fold([]Answer{{true, day(0)}})},
			n:      2, want: []string{"p4"}, count: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			rng := rand.New(rand.NewPCG(1, 2))
			ids := SelectPassages(ms, tt.states, tt.n, day(0), rng)
			if len(ids) != tt.count*3 {
				t.Fatalf("got %d questions, want %d", len(ids), tt.count*3)
			}
			got := map[string]bool{}
			for i := 0; i < len(ids); i += 3 {
				g := byID[ids[i]].Group
				if got[g] {
					t.Fatalf("passage %s chosen twice", g)
				}
				got[g] = true
				for j := range 3 {
					if m := byID[ids[i+j]]; m.Group != g || m.Seq != j+1 {
						t.Fatalf("question %d of passage %s = %+v", j, g, m)
					}
				}
			}
			for _, w := range tt.want {
				if !got[w] {
					t.Errorf("passage %s not chosen; got %v", w, got)
				}
			}
		})
	}
}

func TestPassageState(t *testing.T) {
	t.Parallel()
	ms := members(1, 2)
	both := map[string]State{"p1-1": Fold([]Answer{{true, day(0)}, {true, day(7)}}), "p1-2": Fold([]Answer{{true, day(0)}})}
	s, seen := passageState(ms, both)
	if !seen || s.Streak != 1 || s.Step != firstCorrectStep || !s.Due.Equal(day(7)) || s.Missed {
		t.Errorf("passageState() = %+v, %v; want the weaker question's streak and due", s, seen)
	}
	if _, seen := passageState(ms, nil); seen {
		t.Error("an untouched passage must count as unseen")
	}
}
