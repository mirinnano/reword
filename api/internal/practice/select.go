// Package practice runs question practice sessions for a learner profile.
package practice

import (
	"math/rand/v2"
	"sort"
	"time"
)

// Spaced review. Each question key has a streak: correct answers in a row,
// counted only when given on or after the due time. A wrong answer resets it
// to zero. The next due time grows with the streak, so a question answered
// correctly comes back after 1, 3, 7, 16 and then 35 days, and a missed one
// comes back in the next session.
var intervals = []time.Duration{
	0,                   // streak 0: missed last time, due at once
	24 * time.Hour,      // 1
	3 * 24 * time.Hour,  // 2
	7 * 24 * time.Hour,  // 3
	16 * 24 * time.Hour, // 4
	35 * 24 * time.Hour, // 5 and more
}

// MasteredStreak is the streak at which a question counts as mastered: three
// spaced correct answers in a row, so one lucky guess (1 in 4) is not enough.
const MasteredStreak = 3

// reviewShare caps how much of a session goes to due reviews while unseen
// questions remain, so new material keeps flowing in every session.
const reviewShare = 0.5

type Answer struct {
	Correct bool
	At      time.Time
}

// State is one profile's standing on one question key.
type State struct {
	Streak  int
	Last    time.Time
	Due     time.Time
	Missed  bool // the most recent answer was wrong
	Answers int
}

func (s State) Mastered() bool { return s.Streak >= MasteredStreak }

func interval(streak int) time.Duration {
	return intervals[min(streak, len(intervals)-1)]
}

// Fold replays answers (oldest first) into a State. A correct answer given
// before the item was due (an early review) keeps the streak but does not
// raise it, so cramming the same item in one sitting cannot "master" it.
func Fold(answers []Answer) State {
	var s State
	for _, a := range answers {
		switch {
		case !a.Correct:
			s.Streak = 0
			s.Missed = true
		case s.Answers == 0 || !a.At.Before(s.Due):
			s.Streak++
			s.Missed = false
		default:
			s.Missed = false
		}
		s.Answers++
		s.Last = a.At
		s.Due = a.At.Add(interval(s.Streak))
	}
	return s
}

type Candidate struct {
	ID  uint64
	Key string
}

// Select picks up to n question IDs at time now:
//  1. due reviews, missed ones first, then the most overdue relative to their
//     interval, up to half the session while unseen questions remain;
//  2. unseen questions at random;
//  3. more due reviews if the unseen pool has run out;
//  4. not-yet-due questions, earliest due first, so a session is never short
//     just because everything has been reviewed recently.
func Select(cands []Candidate, states map[string]State, n int, now time.Time, rng *rand.Rand) []uint64 {
	var unseen, due, later []Candidate
	for _, c := range cands {
		s, seen := states[c.Key]
		switch {
		case !seen:
			unseen = append(unseen, c)
		case !now.Before(s.Due):
			due = append(due, c)
		default:
			later = append(later, c)
		}
	}
	rng.Shuffle(len(unseen), func(i, j int) { unseen[i], unseen[j] = unseen[j], unseen[i] })
	// Random tie-break, then stable sort by priority.
	rng.Shuffle(len(due), func(i, j int) { due[i], due[j] = due[j], due[i] })
	sort.SliceStable(due, func(i, j int) bool {
		a, b := states[due[i].Key], states[due[j].Key]
		if a.Missed != b.Missed {
			return a.Missed
		}
		return overdue(a, now) > overdue(b, now)
	})
	sort.SliceStable(later, func(i, j int) bool { return states[later[i].Key].Due.Before(states[later[j].Key].Due) })

	out := make([]uint64, 0, n)
	take := func(cs []Candidate, max int) []Candidate {
		for len(cs) > 0 && len(out) < n && max > 0 {
			out = append(out, cs[0].ID)
			cs = cs[1:]
			max--
		}
		return cs
	}
	reviewCap := n
	if len(unseen) > 0 {
		reviewCap = max(1, int(float64(n)*reviewShare))
	}
	due = take(due, reviewCap)
	take(unseen, n)
	take(due, n)
	take(later, n)
	rng.Shuffle(len(out), func(i, j int) { out[i], out[j] = out[j], out[i] })
	return out
}

// overdue is how late a review is, relative to its interval, so a 35-day item
// one day late ranks below a 1-day item one day late.
func overdue(s State, now time.Time) float64 {
	iv := interval(s.Streak)
	if iv == 0 {
		iv = time.Hour
	}
	return float64(now.Sub(s.Due)) / float64(iv)
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
