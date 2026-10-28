// Package practice runs question practice sessions for a learner profile.
package practice

import (
	"math/rand/v2"
	"sort"
)

type Candidate struct {
	ID  uint64
	Key string
}

// History is one profile's record for one question key, across versions.
type History struct {
	Answered, Correct int64
	LastUnix          int64
}

func (h History) missed() bool { return h.Correct < h.Answered }

// reviewShare caps how much of a session goes to questions answered wrongly
// before, so new material keeps flowing while mistakes come back early.
const reviewShare = 0.3

// Select picks n question IDs: up to 30% previously-missed questions (weakest
// and least recent first), then unseen ones at random, then the weakest of
// the rest if the pool of unseen questions has run out.
func Select(cands []Candidate, hist map[string]History, n int, rng *rand.Rand) []uint64 {
	var unseen, missed, rest []Candidate
	for _, c := range cands {
		h, seen := hist[c.Key]
		switch {
		case !seen:
			unseen = append(unseen, c)
		case h.missed():
			missed = append(missed, c)
		default:
			rest = append(rest, c)
		}
	}
	rng.Shuffle(len(unseen), func(i, j int) { unseen[i], unseen[j] = unseen[j], unseen[i] })
	weakestFirst := func(cs []Candidate) {
		rng.Shuffle(len(cs), func(i, j int) { cs[i], cs[j] = cs[j], cs[i] }) // random tie-break
		sort.SliceStable(cs, func(i, j int) bool {
			a, b := hist[cs[i].Key], hist[cs[j].Key]
			// accuracy a.C/a.A < b.C/b.A without division
			if l, r := a.Correct*b.Answered, b.Correct*a.Answered; l != r {
				return l < r
			}
			return a.LastUnix < b.LastUnix
		})
	}
	weakestFirst(missed)
	weakestFirst(rest)

	out := make([]uint64, 0, n)
	take := func(cs []Candidate, max int) []Candidate {
		for len(cs) > 0 && len(out) < n && max > 0 {
			out = append(out, cs[0].ID)
			cs = cs[1:]
			max--
		}
		return cs
	}
	missed = take(missed, int(float64(n)*reviewShare+0.5))
	take(unseen, n)
	take(missed, n)
	take(rest, n)
	rng.Shuffle(len(out), func(i, j int) { out[i], out[j] = out[j], out[i] })
	return out
}

// ChoiceOrder returns a random permutation: displayed index i shows the
// original choice order[i].
func ChoiceOrder(n int, rng *rand.Rand) []int {
	return rng.Perm(n)
}

// Displayed converts an original choice index to its displayed index.
func Displayed(order []int, original int) int {
	for i, o := range order {
		if o == original {
			return i
		}
	}
	return -1
}
