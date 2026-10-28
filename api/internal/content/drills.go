package content

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// Vocabulary drills are generated from the vocab files instead of being
// written by hand: every headword yields one "pick the meaning" and one
// "pick the word" question. They go through the same import and versioning
// as authored questions, so answers keep pointing at the exact version shown.

const (
	FormatMCQ          = "mcq"           // stem with a "( )" blank, English choices
	FormatVocabMeaning = "vocab_meaning" // English headword, Japanese choices
	FormatVocabWord    = "vocab_word"    // Japanese meaning, English choices
)

// drillDistractors is how many wrong choices each drill gets.
const drillDistractors = 3

// GenerateDrills returns the drill questions for every generated type, keyed
// by type code. Distractors are other headwords of the same grade and part
// of speech, chosen deterministically so that re-importing the same vocab
// yields identical questions (and so no new versions).
func GenerateDrills(s *Set) map[string][]Located[Question] {
	out := map[string][]Located[Question]{}
	for _, t := range QuestionTypes {
		if t.Format != FormatVocabMeaning && t.Format != FormatVocabWord {
			continue
		}
		items := s.Vocab[t.Grade]
		for _, l := range items {
			v := l.Item
			picked := pickDistractors(t.Code+"|"+v.Key, distractorPool(v, items))
			if len(picked) < drillDistractors {
				continue // too few same-POS words with distinct meanings
			}
			choices, answer := drillChoices(t.Format, v, picked)
			a := answer
			q := Question{
				Key:           t.Code + "-" + strings.TrimPrefix(v.Key, t.Grade+"-v-"),
				Stem:          drillStem(t.Format, v),
				Choices:       choices,
				Answer:        &a,
				ExplanationJa: drillExplanation(v),
				Vocab:         []string{v.Key},
				Difficulty:    1,
			}
			out[t.Code] = append(out[t.Code], Located[Question]{File: "(generated from " + l.File + ")", Item: q})
		}
	}
	return out
}

// distractorPool lists headwords that can stand next to v without creating
// a second correct answer: same part of speech, and no meaning in common
// (postpone and put off both mean 延期する, so neither may be the other's
// distractor).
func distractorPool(v VocabItem, items []Located[VocabItem]) []VocabItem {
	mine := meaningSet(v)
	var pool []VocabItem
	for _, l := range items {
		o := l.Item
		if o.Key == v.Key || o.POS != v.POS || strings.EqualFold(o.Headword, v.Headword) {
			continue
		}
		if overlaps(mine, meaningSet(o)) {
			continue
		}
		pool = append(pool, o)
	}
	return pool
}

var annotationRe = regexp.MustCompile(`（[^）]*）|\([^)]*\)`)

// meaningSet normalises glosses: usage notes in brackets are dropped and
// lists split on 、／ so "延期する、先延ばしにする" yields two meanings.
func meaningSet(v VocabItem) map[string]bool {
	m := map[string]bool{}
	for _, g := range v.MeaningsJa {
		g = annotationRe.ReplaceAllString(g, "")
		for _, part := range strings.FieldsFunc(g, func(r rune) bool { return r == '、' || r == '／' || r == '/' || r == '，' }) {
			if p := strings.TrimSpace(part); p != "" {
				m[p] = true
			}
		}
	}
	return m
}

// overlaps reports whether two meaning sets share a meaning, or one meaning
// contains another ("調べる" inside "詳しく調べる").
func overlaps(a, b map[string]bool) bool {
	for x := range a {
		for y := range b {
			if x == y || strings.Contains(x, y) || strings.Contains(y, x) {
				return true
			}
		}
	}
	return false
}

// pickDistractors ranks the pool by a hash of seed and headword and takes
// the first few whose meanings do not overlap each other, so no two choices
// read the same. Adding a word to the vocab only changes the drills where
// the new word ranks into the picks.
func pickDistractors(seed string, pool []VocabItem) []VocabItem {
	rank := func(v VocabItem) uint64 {
		h := sha256.Sum256([]byte(seed + "|" + v.Key))
		return binary.BigEndian.Uint64(h[:8])
	}
	sorted := append([]VocabItem(nil), pool...)
	sort.Slice(sorted, func(i, j int) bool { return rank(sorted[i]) < rank(sorted[j]) })
	var picked []VocabItem
	for _, c := range sorted {
		clash := false
		for _, p := range picked {
			if overlaps(meaningSet(c), meaningSet(p)) || strings.EqualFold(c.Headword, p.Headword) {
				clash = true
				break
			}
		}
		if !clash {
			picked = append(picked, c)
			if len(picked) == drillDistractors {
				break
			}
		}
	}
	return picked
}

// drillChoices places the answer among the distractors at a position derived
// from the headword, so answers are spread over all four positions.
func drillChoices(format string, v VocabItem, picked []VocabItem) ([]string, int) {
	label := func(x VocabItem) string {
		if format == FormatVocabMeaning {
			return primaryMeaning(x)
		}
		return x.Headword
	}
	h := sha256.Sum256([]byte("pos|" + v.Key))
	answer := int(h[0]) % (drillDistractors + 1)
	choices := make([]string, 0, drillDistractors+1)
	for _, p := range picked {
		choices = append(choices, label(p))
	}
	choices = append(choices[:answer], append([]string{label(v)}, choices[answer:]...)...)
	return choices, answer
}

func drillStem(format string, v VocabItem) string {
	if format == FormatVocabMeaning {
		return v.Headword
	}
	return strings.Join(v.MeaningsJa, "／")
}

// primaryMeaning is the first gloss without its usage note.
func primaryMeaning(v VocabItem) string {
	if len(v.MeaningsJa) == 0 {
		return ""
	}
	if m := strings.TrimSpace(annotationRe.ReplaceAllString(v.MeaningsJa[0], "")); m != "" {
		return m
	}
	return v.MeaningsJa[0]
}

func drillExplanation(v VocabItem) string {
	return fmt.Sprintf("%s は「%s」。例文：%s（%s）", v.Headword, strings.Join(v.MeaningsJa, "／"), v.ExampleEn, v.ExampleJa)
}
