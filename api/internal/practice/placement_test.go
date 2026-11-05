package practice

import (
	"math/rand/v2"
	"testing"
	"time"
)

func answers(blocks ...[3]int) []PlacementAnswer {
	var out []PlacementAnswer
	for _, b := range blocks {
		for range b[0] {
			out = append(out, PlacementAnswer{Answered: true, Correct: true})
		}
		for range b[1] {
			out = append(out, PlacementAnswer{Answered: true})
		}
		for range b[2] {
			out = append(out, PlacementAnswer{Answered: true, DontKnow: true})
		}
	}
	return out
}

func TestScorePlacement(t *testing.T) {
	tests := []struct {
		name    string
		a       []PlacementAnswer
		stage   int
		decided bool
	}{
		{"nothing yet", make([]PlacementAnswer, PlacementSize), 0, false},
		{"all don't know stops at the first block", answers([3]int{0, 0, 10}), 0, true},
		{"seven right and the rest don't know pass", answers([3]int{7, 0, 3}), 1, false},
		{"seven right and one wrong fall short", answers([3]int{7, 1, 2}), 0, true},
		{"eight right and two wrong pass", answers([3]int{8, 2, 0}), 1, false},
		{"second block fails", answers([3]int{10, 0, 0}, [3]int{3, 7, 0}), 1, true},
		{"all three pass", answers([3]int{10, 0, 0}, [3]int{9, 0, 1}, [3]int{8, 1, 1}), 3, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := append(tt.a, make([]PlacementAnswer, max(0, PlacementSize-len(tt.a)))...)
			r := ScorePlacement(a)
			if r.Decided != tt.decided || (r.Decided && r.Stage != tt.stage) {
				t.Fatalf("got stage %d decided %v, want %d %v (%+v)", r.Stage, r.Decided, tt.stage, tt.decided, r.Blocks)
			}
		})
	}
}

// Guessing every question must almost never beat saying "don't know": the
// correction for wrong answers is what the test's honesty rests on.
func TestGuessingDoesNotPass(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	passed := 0
	const runs = 20000
	for range runs {
		a := make([]PlacementAnswer, PlacementSize)
		for i := range a {
			a[i] = PlacementAnswer{Answered: true, Correct: rng.IntN(4) == 0}
		}
		if ScorePlacement(a).Stage > 0 {
			passed++
		}
	}
	if float64(passed)/runs > 0.001 {
		t.Fatalf("a pure guesser passed the first block %d of %d times", passed, runs)
	}
}

func TestSelectPlacement(t *testing.T) {
	var pool []PlacementQuestion
	id := uint64(0)
	add := func(typ string, diff, n int) {
		for range n {
			id++
			pool = append(pool, PlacementQuestion{ID: id, Type: typ, Difficulty: diff})
		}
	}
	add("pre2-q1", 1, 12)
	add("g2-q1", 1, 12)
	add("g2-q1", 2, 6)
	add("g2-q1", 3, 6)
	add("g2-vm", 1, 50)
	tier := map[uint64]int{}
	for _, q := range pool {
		tier[q.ID] = placementTier(q)
	}
	picks, err := SelectPlacement(pool, rand.New(rand.NewPCG(1, 2)))
	if err != nil || len(picks) != PlacementSize {
		t.Fatalf("picks = %d, err = %v", len(picks), err)
	}
	seen := map[uint64]bool{}
	for i, p := range picks {
		if want := i / PlacementBlock; tier[p.ID] != want {
			t.Fatalf("position %d is from tier %d, want %d", i, tier[p.ID], want)
		}
		if seen[p.ID] {
			t.Fatalf("question %d picked twice", p.ID)
		}
		seen[p.ID] = true
	}
	if _, err := SelectPlacement(pool[:15], rand.New(rand.NewPCG(1, 2))); err == nil {
		t.Fatal("a short pool must be an error, not a short test")
	}
}

func TestStageOrdersNewQuestions(t *testing.T) {
	type q struct {
		key   string
		grade string
		diff  int
	}
	qs := []q{{"pre", "pre2", 1}, {"easy", "g2", 1}, {"hard", "g2", 2}}
	first := func(stage int) map[string]bool {
		var cands []Candidate
		for i, c := range qs {
			if rank, keep := newRank(stage, c.grade, c.diff); keep {
				cands = append(cands, Candidate{ID: uint64(i + 1), Key: c.key, Rank: rank})
			}
		}
		// With nothing due, the routine's one place goes to a new question.
		picks := SelectDaily(cands, nil, nil, 1, time.Unix(0, 0), rand.New(rand.NewPCG(3, 4)))
		out := map[string]bool{}
		for _, p := range picks {
			out[qs[p.ID-1].key] = true
		}
		return out
	}
	for stage, want := range map[int]string{0: "pre", 1: "easy", 3: "hard"} {
		if got := first(stage); !got[want] {
			t.Errorf("stage %d started on %v, want %s", stage, got, want)
		}
	}
	if _, keep := newRank(2, "pre2", 1); keep {
		t.Error("a learner past 準2級 must not get 準2級 questions as new")
	}
	if PassagesAllowed(0) || !PassagesAllowed(1) {
		t.Error("passages wait for the 準2級 block")
	}
}
