package practice

import (
	"math/rand/v2"
	"time"
)

// The daily routine: a fixed-size mix of reviews, new questions and, every
// other day, one passage. A fixed review share would pile up reviews without
// end, because each new question costs several reviews before it sticks.
// So reviews go first and new questions are rationed by how busy the queue is:
// a third of the routine while reviews fit in one routine, a sixth while they
// fit in two, none beyond that. Reviews left over simply wait; nothing is lost
// and the learner is never shown a debt. The rule and its numbers come from
// comparing candidates under a forgetting-curve simulation (docs/PLAN.md).
const (
	DailySize = 12

	// A learner's study day ends at 04:00 Japan time, so a late-night routine
	// counts for the day it was started in.
	dayBoundary = 4 * time.Hour
)

var jst = time.FixedZone("JST", 9*60*60)

// StudyDay is the date of the learner's study day containing t, as midnight UTC.
func StudyDay(t time.Time) time.Time {
	y, m, d := t.In(jst).Add(-dayBoundary).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// PassageDue reports whether the routine after count earlier ones takes a
// passage. Every other routine does, starting with the second, so the first
// day stays short.
func PassageDue(count int) bool { return count%2 == 1 }

func newBudget(dueCount, n int) int {
	switch {
	case dueCount <= n:
		return n / 3
	case dueCount <= 2*n:
		return n / 6
	default:
		return 0
	}
}

// split says how many of n places go to reviews and to new questions.
func split(dueCount, unseen, n int) (reviews, fresh int) {
	fresh = min(max(newBudget(dueCount, n), n-dueCount), unseen)
	return min(n-fresh, dueCount), fresh
}

// Pick is one question of a routine.
type Pick struct {
	ID      uint64
	Review  bool
	Passage bool
}

// SelectDaily picks a routine of about n questions: the singles shuffled, then
// the passage, whole and in order. members is empty on a day without one.
func SelectDaily(cands []Candidate, members []Member, states map[string]State, n int, now time.Time, rng *rand.Rand) []Pick {
	var passage []Pick
	if len(members) > 0 {
		key := make(map[uint64]string, len(members))
		for _, m := range members {
			key[m.ID] = m.Key
		}
		for _, id := range SelectPassages(members, states, 1, now, rng) {
			_, seen := states[key[id]]
			passage = append(passage, Pick{ID: id, Review: seen, Passage: true})
		}
	}
	singles := max(n-len(passage), 0)
	unseen, due, later := sortCandidates(cands, states, now, rng)
	reviews, fresh := split(len(due), len(unseen), singles)

	out := make([]Pick, 0, n)
	for _, c := range due[:reviews] {
		out = append(out, Pick{ID: c.ID, Review: true})
	}
	for _, c := range unseen[:fresh] {
		out = append(out, Pick{ID: c.ID})
	}
	// A learner who has seen everything still gets a full routine.
	for _, c := range later {
		if len(out) >= singles {
			break
		}
		out = append(out, Pick{ID: c.ID, Review: true})
	}
	rng.Shuffle(len(out), func(i, j int) { out[i], out[j] = out[j], out[i] })
	return append(out, passage...)
}
