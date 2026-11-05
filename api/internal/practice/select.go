// Package practice runs question practice sessions for a learner profile.
package practice

import (
	"math/rand/v2"
	"sort"
	"time"
)

// Spaced review. Each question key moves up a ladder of intervals: a correct
// answer given on or after the due time climbs one step and a wrong answer
// goes back to the bottom, so a question comes back after 1, 4, 10, 30 and
// then 90 days, and a missed one comes back in the next session. A question
// answered correctly the first time it is seen skips the short steps and
// returns after a week: most learners already know many words, and the early
// steps are wasted on them. A lucky guess is caught at that return.
var intervals = []time.Duration{
	0,                   // step 0: missed last time, due at once
	24 * time.Hour,      // 1
	4 * 24 * time.Hour,  // 2
	10 * 24 * time.Hour, // 3
	30 * 24 * time.Hour, // 4
	90 * 24 * time.Hour, // 5 and more
}

const (
	firstCorrectStep = 3
	firstCorrectGap  = 7 * 24 * time.Hour
)

// MasteredStreak is the streak at which a question counts as mastered: three
// spaced correct answers in a row, so one lucky guess (1 in 4) is not enough.
// The streak is kept apart from the ladder step: skipping steps on a first
// correct answer must not count as mastering anything.
const MasteredStreak = 3

// reviewShare caps how much of a practice session goes to due reviews while
// unseen questions remain, so new material keeps flowing in every session.
const reviewShare = 0.5

type Answer struct {
	Correct bool
	At      time.Time
}

// State is one profile's standing on one question key.
type State struct {
	Streak  int // spaced correct answers in a row
	Step    int // position on the interval ladder
	Last    time.Time
	Due     time.Time
	Missed  bool // the most recent answer was wrong
	Answers int
}

func (s State) Mastered() bool { return s.Streak >= MasteredStreak }

func interval(step int) time.Duration {
	return intervals[min(step, len(intervals)-1)]
}

// Fold replays answers (oldest first) into a State. A correct answer given
// before the item was due (an early review) changes nothing but Missed, so
// cramming the same item in one sitting cannot "master" it or push it away.
func Fold(answers []Answer) State {
	var s State
	for _, a := range answers {
		switch {
		case !a.Correct:
			s.Streak, s.Step = 0, 0
			s.Missed = true
			s.Due = a.At
		case s.Answers == 0:
			s.Streak, s.Step = 1, firstCorrectStep
			s.Due = a.At.Add(firstCorrectGap)
		case !a.At.Before(s.Due):
			s.Streak++
			s.Step++
			s.Missed = false
			s.Due = a.At.Add(interval(s.Step))
		default:
			s.Missed = false
		}
		s.Answers++
		s.Last = a.At
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
	unseen, due, later := sortCandidates(cands, states, now, rng)
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

// sortCandidates splits candidates into unseen (shuffled), due (missed first,
// then most overdue) and not yet due (earliest first).
func sortCandidates(cands []Candidate, states map[string]State, now time.Time, rng *rand.Rand) (unseen, due, later []Candidate) {
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
	return unseen, due, later
}

// Member is one question of a passage.
type Member struct {
	ID    uint64
	Key   string
	Group string // the passage
	Seq   int    // position in the passage
}

// SelectPassages picks up to n passages by the same rules as Select and
// returns all their questions, passage by passage and in passage order: a
// passage is read once, so it is never split across or within a session.
func SelectPassages(members []Member, states map[string]State, n int, now time.Time, rng *rand.Rand) []uint64 {
	groups := map[string][]Member{}
	for _, m := range members {
		groups[m.Group] = append(groups[m.Group], m)
	}
	names := make([]string, 0, len(groups))
	for name := range groups {
		names = append(names, name)
	}
	sort.Strings(names)

	cands := make([]Candidate, len(names))
	passageStates := make(map[string]State, len(names))
	for i, name := range names {
		cands[i] = Candidate{ID: uint64(i), Key: name}
		if s, seen := passageState(groups[name], states); seen {
			passageStates[name] = s
		}
	}

	var out []uint64
	for _, i := range Select(cands, passageStates, n, now, rng) {
		ms := groups[names[i]]
		sort.Slice(ms, func(a, b int) bool { return ms[a].Seq < ms[b].Seq })
		for _, m := range ms {
			out = append(out, m.ID)
		}
	}
	return out
}

// passageState is a passage's standing, led by its weakest question: it is
// missed if any question was and due when the first one is. A passage left
// unfinished is due at once, so the learner comes back to it.
func passageState(ms []Member, states map[string]State) (State, bool) {
	var out State
	seen := 0
	for _, m := range ms {
		s, ok := states[m.Key]
		if !ok {
			continue
		}
		if seen == 0 {
			out = s
		} else {
			out.Streak = min(out.Streak, s.Streak)
			out.Step = min(out.Step, s.Step)
			out.Missed = out.Missed || s.Missed
			if s.Due.Before(out.Due) {
				out.Due = s.Due
			}
		}
		seen++
	}
	if seen > 0 && seen < len(ms) {
		out.Due = time.Time{}
	}
	return out, seen > 0
}

// overdue is how late a review is, relative to its interval, so a 35-day item
// one day late ranks below a 1-day item one day late.
func overdue(s State, now time.Time) float64 {
	iv := interval(s.Step)
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
