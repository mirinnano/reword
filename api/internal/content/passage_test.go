package content

import (
	"strings"
	"testing"
)

// body returns a passage body of n words with the given blanks spliced in.
func body(n int, blanks ...string) string {
	w := make([]string, n)
	for i := range w {
		w[i] = "word"
	}
	return strings.Join(append(w, blanks...), " ")
}

// pq builds a valid question; ids are assigned by qs.
func pq(prompt string) PassageQuestion {
	a := 0
	return PassageQuestion{ID: 1, Prompt: prompt, Choices: []string{"a", "b", "c", "d"}, Answer: &a, ExplanationJa: "説明"}
}

func qs(prompts ...string) []PassageQuestion {
	out := make([]PassageQuestion, len(prompts))
	for i, p := range prompts {
		out[i] = pq(p)
		out[i].ID = i + 1
	}
	return out
}

func passageSet(code string, p Passage) *Set {
	return &Set{Passages: map[string][]Located[Passage]{code: {{File: "t.yaml", Item: p}}}}
}

func TestValidatePassages(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		code    string
		p       Passage
		wantErr string
	}{
		{name: "blank ok", code: "g2-q2", p: Passage{Key: "g2-q2-0001", Title: "T", Difficulty: 2,
			Passage: body(150, "( 1 )", "( 2 )", "( 3 )"), Questions: qs("", "", "")}},
		{name: "question ok", code: "g2-q3", p: Passage{Key: "g2-q3-0001", Title: "T", Difficulty: 2,
			Passage: body(200), Questions: qs("Why?", "What?")}},
		{name: "blank numbering skips", code: "g2-q2", p: Passage{Key: "g2-q2-0001", Title: "T", Difficulty: 2,
			Passage: body(150, "( 1 )", "( 3 )"), Questions: qs("", "")}, wantErr: "numbered ( 1 ), ( 2 )"},
		{name: "blanks and questions differ", code: "g2-q2", p: Passage{Key: "g2-q2-0001", Title: "T", Difficulty: 2,
			Passage: body(150, "( 1 )", "( 2 )"), Questions: qs("", "", "")}, wantErr: "2 numbered blanks for 3 questions"},
		{name: "question passage with blanks", code: "g2-q3", p: Passage{Key: "g2-q3-0001", Title: "T", Difficulty: 2,
			Passage: body(150, "( 1 )"), Questions: qs("Why?", "What?")}, wantErr: "must not contain numbered blanks"},
		{name: "missing prompt", code: "g2-q3", p: Passage{Key: "g2-q3-0001", Title: "T", Difficulty: 2,
			Passage: body(150), Questions: qs("Why?", "")}, wantErr: "question 2: prompt is required"},
		{name: "too short", code: "g2-q3", p: Passage{Key: "g2-q3-0001", Title: "T", Difficulty: 2,
			Passage: body(50), Questions: qs("Why?", "What?")}, wantErr: "passage has 50 words"},
		{name: "parenthesised year is not a blank", code: "g2-q3", p: Passage{Key: "g2-q3-0001", Title: "T", Difficulty: 2,
			Passage: body(150, "(1998)"), Questions: qs("Why?", "What?")}},
		{name: "unnumbered blank", code: "g2-q2", p: Passage{Key: "g2-q2-0001", Title: "T", Difficulty: 2,
			Passage: body(150, "( 1 )", "( 2 )", "( )"), Questions: qs("", "")}, wantErr: "unnumbered blank"},
		{name: "blank markers are not words", code: "g2-q2", p: Passage{Key: "g2-q2-0001", Title: "T", Difficulty: 2,
			Passage: body(117, "( 1 )", "( 2 )"), Questions: qs("", "")}, wantErr: "passage has 119 words"},
		{name: "duplicate question id", code: "g2-q3", p: Passage{Key: "g2-q3-0001", Title: "T", Difficulty: 2,
			Passage: body(150), Questions: []PassageQuestion{pq("Why?"), pq("What?")}}, wantErr: "question 2: duplicate id 1"},
		{name: "missing question id", code: "g2-q3", p: Passage{Key: "g2-q3-0001", Title: "T", Difficulty: 2,
			Passage: body(150), Questions: []PassageQuestion{{Prompt: "Why?", Choices: pq("").Choices, Answer: pq("").Answer, ExplanationJa: "説明"}, pq("What?")}}, wantErr: "question 1: id is required"},
		{name: "bad key", code: "g2-q2", p: Passage{Key: "g2-q3-0001", Title: "T", Difficulty: 2,
			Passage: body(150, "( 1 )", "( 2 )"), Questions: qs("", "")}, wantErr: "key must look like g2-q2-0001"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			errs := validatePassages(passageSet(tt.code, tt.p), map[string]bool{})
			got := ""
			for _, e := range errs {
				got += e.Error() + "\n"
			}
			switch {
			case tt.wantErr == "" && got != "":
				t.Errorf("validatePassages() unexpected errors:\n%s", got)
			case tt.wantErr != "" && !strings.Contains(got, tt.wantErr):
				t.Errorf("validatePassages() = %q, want it to contain %q", got, tt.wantErr)
			}
		})
	}
}

func TestExpandPassages(t *testing.T) {
	t.Parallel()
	s := passageSet("g2-q2", Passage{Key: "g2-q2-0001", Title: "Night Trains", Difficulty: 2,
		Passage: "  " + body(150, "( 1 )", "( 2 )") + "\n", Questions: qs("", "")})
	s.Passages["g2-q3"] = []Located[Passage]{{Item: Passage{Key: "g2-q3-0001", Title: "T", Difficulty: 1,
		Passage: body(150), Questions: qs("Why?", "What?")}}}
	got := ExpandPassages(s)

	blanks := got["g2-q2"]
	if len(blanks) != 2 || blanks[0].Item.Key != "g2-q2-0001-1" || blanks[1].Item.Seq != 2 {
		t.Fatalf("g2-q2 expansion = %+v", blanks)
	}
	if q := blanks[0].Item; q.Group != "g2-q2-0001" || q.Title != "Night Trains" || q.Stem != "" || strings.HasPrefix(q.Passage, " ") {
		t.Errorf("g2-q2 question fields = %+v", q)
	}
	q3 := got["g2-q3"]
	if len(q3) != 2 || q3[1].Item.Stem != "What?" || q3[1].Item.Seq != 2 || q3[1].Item.Difficulty != 1 {
		t.Errorf("g2-q3 expansion = %+v", q3)
	}
}

// A key belongs to its question: adding one in front must not hand the
// existing questions' history to something else.
func TestExpandPassagesKeepsKeysWhenQuestionsMove(t *testing.T) {
	t.Parallel()
	first := qs("Why?", "What?")
	added := pq("How?")
	added.ID = 3
	s := passageSet("g2-q3", Passage{Key: "g2-q3-0001", Questions: []PassageQuestion{added, first[0], first[1]}})

	got := map[string]string{}
	seq := map[string]int{}
	for _, l := range ExpandPassages(s)["g2-q3"] {
		got[l.Item.Key], seq[l.Item.Key] = l.Item.Stem, l.Item.Seq
	}
	if got["g2-q3-0001-1"] != "Why?" || got["g2-q3-0001-2"] != "What?" || got["g2-q3-0001-3"] != "How?" {
		t.Errorf("keys moved with position: %v", got)
	}
	if seq["g2-q3-0001-3"] != 1 || seq["g2-q3-0001-1"] != 2 {
		t.Errorf("Seq must follow file order: %v", seq)
	}
}

func TestLintCoversPassages(t *testing.T) {
	t.Parallel()
	p := pq("Why?")
	p.Choices = []string{"go", "a long phrase", "c", "d"}
	findings := Lint(passageSet("g2-q3", Passage{Key: "g2-q3-0001", Passage: body(150), Questions: []PassageQuestion{p}}))
	if len(findings) != 1 || findings[0].Rule != "choice-shape" {
		t.Errorf("Lint() = %v, want one choice-shape finding and no stem rules", findings)
	}
}
