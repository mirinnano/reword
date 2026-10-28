package practice

import (
	"fmt"
	"math/rand/v2"
	"testing"
)

func cands(n int) []Candidate {
	out := make([]Candidate, n)
	for i := range out {
		out[i] = Candidate{ID: uint64(i + 1), Key: fmt.Sprintf("k%02d", i+1)}
	}
	return out
}

func rng() *rand.Rand { return rand.New(rand.NewPCG(1, 2)) }

func set(ids []uint64) map[uint64]bool {
	m := map[uint64]bool{}
	for _, id := range ids {
		m[id] = true
	}
	return m
}

func TestSelectPrefersUnseen(t *testing.T) {
	hist := map[string]History{}
	for i := 1; i <= 10; i++ { // first 10 answered correctly
		hist[fmt.Sprintf("k%02d", i)] = History{Answered: 1, Correct: 1}
	}
	got := Select(cands(20), hist, 10, rng())
	if len(got) != 10 || len(set(got)) != 10 {
		t.Fatalf("want 10 distinct, got %v", got)
	}
	for _, id := range got {
		if id <= 10 {
			t.Fatalf("picked seen question %d while unseen ones remain", id)
		}
	}
}

func TestSelectMixesInMissed(t *testing.T) {
	hist := map[string]History{}
	for i := 1; i <= 5; i++ { // 5 missed
		hist[fmt.Sprintf("k%02d", i)] = History{Answered: 2, Correct: 1, LastUnix: int64(i)}
	}
	got := set(Select(cands(30), hist, 10, rng()))
	missed := 0
	for id := uint64(1); id <= 5; id++ {
		if got[id] {
			missed++
		}
	}
	if missed != 3 {
		t.Fatalf("want 3 missed questions (30%% of 10), got %d", missed)
	}
}

func TestSelectWeakestWhenExhausted(t *testing.T) {
	hist := map[string]History{
		"k01": {Answered: 4, Correct: 4},
		"k02": {Answered: 4, Correct: 0}, // weakest
		"k03": {Answered: 4, Correct: 3},
		"k04": {Answered: 4, Correct: 1}, // second weakest
	}
	got := set(Select(cands(4), hist, 2, rng()))
	if !got[2] || !got[4] {
		t.Fatalf("want the two weakest (2, 4), got %v", got)
	}
}

func TestSelectSmallPool(t *testing.T) {
	if got := Select(cands(3), nil, 10, rng()); len(got) != 3 {
		t.Fatalf("want all 3, got %v", got)
	}
}

func TestChoiceOrderRoundTrip(t *testing.T) {
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
