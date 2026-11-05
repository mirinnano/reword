package practice

import (
	"errors"
	"math/rand/v2"
)

// The placement test climbs three blocks of fill-in-the-blank vocabulary, from
// 準2級 to the harder half of 2級, and stops at the first block not passed. A
// learner who is out of their depth stops early, and nobody answers more than
// 30 questions. The stage is the number of blocks passed.
//
// Questions are not calibrated: a block is what the authors rated it. The block
// size and the pass mark come from a simulation of hypothetical learners, which
// placed about four in five in the right stage (docs/PLAN.md). The stage only
// changes the order of new questions and whether passages are in the routine,
// so a wrong stage costs little and retaking the test undoes it.
const (
	PlacementBlocks = 3
	PlacementBlock  = 10
	PlacementSize   = PlacementBlocks * PlacementBlock

	// PlacementPass is the net score a block needs. Each wrong answer takes
	// away a third of a point, which cancels the 1 in 4 lucky guess, so
	// guessing never scores better than choosing "don't know".
	PlacementPass = 7.0

	// StageDefault is how a learner who skipped the test is treated.
	StageDefault = 2

	PlacementCode = "placement"
	placementName = "実力診断"
)

// BlockScore is one block as the learner sat it.
type BlockScore struct {
	Total    int  `json:"total"`
	Correct  int  `json:"correct"`
	Wrong    int  `json:"wrong"`
	DontKnow int  `json:"dont_know"`
	Passed   bool `json:"passed"`
}

// PlacementResult is the test so far. Stage means nothing until Decided.
type PlacementResult struct {
	Stage   int          `json:"stage"`
	Decided bool         `json:"decided"`
	Blocks  []BlockScore `json:"blocks"`
}

// PlacementAnswer is one question of a sitting, in position order.
type PlacementAnswer struct {
	Answered bool
	DontKnow bool
	Correct  bool
}

// ScorePlacement reads the blocks in order and stops at the first one that is
// unfinished or failed.
func ScorePlacement(answers []PlacementAnswer) PlacementResult {
	var r PlacementResult
	for b := 0; b < PlacementBlocks; b++ {
		start := b * PlacementBlock
		if start+PlacementBlock > len(answers) {
			return r
		}
		s := BlockScore{Total: PlacementBlock}
		for _, a := range answers[start : start+PlacementBlock] {
			switch {
			case !a.Answered:
			case a.DontKnow:
				s.DontKnow++
			case a.Correct:
				s.Correct++
			default:
				s.Wrong++
			}
		}
		// Compared in thirds so the mark is exact: 8 right and 2 wrong pass, 7
		// right and 1 wrong do not.
		s.Passed = 3*s.Correct-s.Wrong >= int(3*PlacementPass)
		r.Blocks = append(r.Blocks, s)
		if s.Correct+s.Wrong+s.DontKnow < PlacementBlock {
			return r
		}
		if !s.Passed {
			r.Decided = true
			return r
		}
		r.Stage++
	}
	r.Decided = true
	return r
}

// PlacementQuestion is a candidate for the test.
type PlacementQuestion struct {
	ID         uint64
	Type       string
	Difficulty int
}

// placementTier says which block a question belongs to, or -1.
func placementTier(q PlacementQuestion) int {
	switch {
	case q.Type == "pre2-q1":
		return 0
	case q.Type == "g2-q1" && q.Difficulty <= 1:
		return 1
	case q.Type == "g2-q1":
		return 2
	default:
		return -1
	}
}

var errPlacementPool = errors.New("too few questions for a placement test")

// SelectPlacement draws each block at random, so a retake mostly differs.
func SelectPlacement(pool []PlacementQuestion, rng *rand.Rand) ([]Pick, error) {
	tiers := make([][]PlacementQuestion, PlacementBlocks)
	for _, q := range pool {
		if t := placementTier(q); t >= 0 {
			tiers[t] = append(tiers[t], q)
		}
	}
	out := make([]Pick, 0, PlacementSize)
	for _, t := range tiers {
		if len(t) < PlacementBlock {
			return nil, errPlacementPool
		}
		rng.Shuffle(len(t), func(i, j int) { t[i], t[j] = t[j], t[i] })
		for _, q := range t[:PlacementBlock] {
			out = append(out, Pick{ID: q.ID})
		}
	}
	return out, nil
}

// newRank orders a learner's unseen questions, lowest first. keep is false for
// a question below the learner's stage, which is not worth a place in their
// routine. Stage 0 starts on 準2級 material, stage 1 on the easy half of 2級,
// stage 2 is the whole of 2級 at random, and stage 3 goes to the hard half.
func newRank(stage int, grade string, difficulty int) (rank int, keep bool) {
	pre2 := grade == "pre2"
	if pre2 && stage > 0 {
		return 0, false
	}
	switch stage {
	case 0:
		switch {
		case pre2:
			return 0, true
		case difficulty <= 1:
			return 1, true
		}
		return 2, true
	case 1:
		if difficulty <= 1 {
			return 0, true
		}
		return 1, true
	case 3:
		if difficulty >= 2 {
			return 0, true
		}
		return 1, true
	}
	return 0, true
}

// PassagesAllowed reports whether a stage takes passages in the routine. The
// 2級 passages are out of reach until the 準2級 block is passed; they stay open
// in format practice.
func PassagesAllowed(stage int) bool { return stage > 0 }
