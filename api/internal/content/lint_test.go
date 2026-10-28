package content

import (
	"strings"
	"testing"
)

// lintSet lints questions whose answers link to vocab of the given POS
// (key "v-<pos>"); an empty POS leaves the question unlinked.
func lintSet(pos string, qs ...Question) []Finding {
	ans := 0
	locs := make([]Located[Question], len(qs))
	for i, q := range qs {
		if q.Answer == nil {
			q.Answer = &ans
		}
		if pos != "" {
			q.Vocab = []string{"v-" + pos}
		}
		locs[i] = Located[Question]{File: "t.yaml", Item: q}
	}
	set := &Set{Questions: map[string][]Located[Question]{"g2-q1": locs}, Vocab: map[string][]Located[VocabItem]{}}
	if pos != "" {
		// The headword matches the first choice so answer-vocab stays quiet.
		set.Vocab["g2"] = []Located[VocabItem]{{Item: VocabItem{Key: "v-" + pos, Headword: qs[0].Choices[0], POS: pos}}}
	}
	return Lint(set)
}

func rules(fs []Finding) string {
	var r []string
	for _, f := range fs {
		r = append(r, f.Key+":"+f.Rule)
	}
	return strings.Join(r, ",")
}

func TestLint(t *testing.T) {
	t.Parallel()
	stem := "The museum decided to ( ) the exhibition for two more weeks because many people came."
	q := func(choices ...string) Question { return Question{Key: "q1", Stem: stem, Choices: choices} }
	tests := []struct {
		name string
		pos  string
		qs   []Question
		want string
	}{
		{name: "clean", pos: "verb", qs: []Question{q("extend", "punish", "swallow", "injure")}},
		{name: "phrases of different length are fine", pos: "phrase", qs: []Question{q("put off", "make up for", "look into", "carry out")}},
		{name: "irregular past is past", pos: "verb", qs: []Question{q("stole", "wasted", "delayed", "donated")}},
		{name: "forgot and died are past", pos: "verb", qs: []Question{q("forgot", "died", "rested", "stayed")}},
		{name: "same-form verbs match past phrases", pos: "phrase", qs: []Question{q("took over", "set up", "ran out of", "put off")}},
		{name: "same-form verbs match base phrases", pos: "phrase", qs: []Question{q("take over", "set up", "run out of", "put off")}},
		{name: "-ed adjectives are not tense", pos: "adjective", qs: []Question{q("afraid", "tired", "capable", "proud")}},
		{name: "-ing nouns are not tense", pos: "noun", qs: []Question{q("furniture", "clothing", "equipment", "machinery")}},
		{name: "word mixed with phrases", pos: "verb", qs: []Question{q("extend", "put off", "look into", "carry out")}, want: "q1:choice-shape"},
		{name: "verb tense mismatch", pos: "verb", qs: []Question{q("extended", "punish", "swallow", "injure")}, want: "q1:choice-inflection"},
		{name: "unlinked answer is checked as a verb", qs: []Question{q("extended", "punish", "swallow", "injure")}, want: "q1:choice-inflection"},
		{name: "short stem", pos: "verb", qs: []Question{{Key: "q1", Stem: "He ( ) it.", Choices: []string{"extend", "punish", "swallow", "injure"}}}, want: "q1:stem-length"},
		{name: "near duplicate", pos: "verb", qs: []Question{
			q("extend", "punish", "swallow", "injure"),
			{Key: "q2", Stem: "The museum decided to ( ) the exhibition for two more weeks because it was popular.", Choices: []string{"extend", "punish", "swallow", "injure"}},
		}, want: "q1:near-duplicate"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := rules(lintSet(tt.pos, tt.qs...)); got != tt.want {
				t.Errorf("Lint() rules = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSameLemma(t *testing.T) {
	t.Parallel()
	tests := []struct {
		form, head string
		want       bool
	}{
		{"conducted", "conduct", true},
		{"carried out", "carry out", true},
		{"figured out", "figure out", true},
		{"took over", "take over", true},
		{"studies", "study", true},
		{"stopped", "stop", true},
		{"conduct", "consume", false},
		{"took part in", "take over", false},
	}
	for _, tt := range tests {
		t.Run(tt.form+"/"+tt.head, func(t *testing.T) {
			t.Parallel()
			if got := sameLemma(tt.form, tt.head); got != tt.want {
				t.Errorf("sameLemma(%q, %q) = %v, want %v", tt.form, tt.head, got, tt.want)
			}
		})
	}
}
