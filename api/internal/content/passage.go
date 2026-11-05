package content

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const (
	minPassageWords = 120
	maxPassageWords = 450
	minPassageQs    = 2
	maxPassageQs    = 5
)

// numberedBlankRe matches the "( 1 )" markers of a passage_blank passage. Two
// digits at most, so a parenthesised year such as "(1998)" stays plain text.
var numberedBlankRe = regexp.MustCompile(`\(\s*(\d{1,2})\s*\)`)

// ExpandPassages turns every passage into one question per blank or prompt,
// keyed "<passage key>-<id>". They import, version and are answered like any
// other question; Group and Seq tie them back to their passage, which practice
// sessions always present whole and in order.
func ExpandPassages(s *Set) map[string][]Located[Question] {
	out := map[string][]Located[Question]{}
	for code, items := range s.Passages {
		t, _ := questionType(code)
		for _, l := range items {
			p := l.Item
			for i, pq := range p.Questions {
				q := Question{
					Key:           p.Key + "-" + strconv.Itoa(pq.ID),
					Choices:       pq.Choices,
					Answer:        pq.Answer,
					ExplanationJa: pq.ExplanationJa,
					Vocab:         pq.Vocab,
					Difficulty:    p.Difficulty,
					Group:         p.Key,
					Title:         p.Title,
					Passage:       strings.TrimSpace(p.Passage),
					Seq:           i + 1,
				}
				if t.Format == FormatPassageQuestion {
					q.Stem = pq.Prompt
				}
				out[code] = append(out[code], Located[Question]{File: l.File, Item: q})
			}
		}
	}
	for code := range out {
		sort.SliceStable(out[code], func(i, j int) bool { return out[code][i].Item.Key < out[code][j].Item.Key })
	}
	return out
}

func validatePassages(s *Set, vocabKeys map[string]bool) []error {
	var errs []error
	seen := map[string]bool{}
	for code, items := range s.Passages {
		t, ok := questionType(code)
		if !ok || !t.Passage() {
			errs = append(errs, fmt.Errorf("passages: %q is not a passage type", code))
			continue
		}
		for _, l := range items {
			p := l.Item
			fail := func(format string, args ...any) {
				errs = append(errs, fmt.Errorf("%s: %s: %s", l.File, p.Key, fmt.Sprintf(format, args...)))
			}
			switch {
			case !questionKeyRe.MatchString(p.Key) || !strings.HasPrefix(p.Key, code+"-"):
				fail("key must look like %s-0001", code)
			case seen[p.Key]:
				fail("duplicate key")
			}
			seen[p.Key] = true
			if strings.TrimSpace(p.Title) == "" {
				fail("title is required")
			}
			if n := len(strings.Fields(numberedBlankRe.ReplaceAllString(p.Passage, "blank"))); n < minPassageWords || n > maxPassageWords {
				fail("passage has %d words (want %d-%d)", n, minPassageWords, maxPassageWords)
			}
			if n := len(p.Questions); n < minPassageQs || n > maxPassageQs {
				fail("needs %d-%d questions, got %d", minPassageQs, maxPassageQs, n)
			}
			if p.Difficulty < 1 || p.Difficulty > 3 {
				fail("difficulty must be 1, 2 or 3")
			}

			if blankRe.MatchString(p.Passage) {
				fail("passage has an unnumbered blank \"( )\"")
			}
			marks := numberedBlankRe.FindAllStringSubmatch(p.Passage, -1)
			if t.Format == FormatPassageBlank {
				if len(marks) != len(p.Questions) {
					fail("passage has %d numbered blanks for %d questions", len(marks), len(p.Questions))
				}
				for i, m := range marks {
					if m[1] != strconv.Itoa(i+1) {
						fail("blanks must be numbered ( 1 ), ( 2 ), ... in order; found ( %s ) at position %d", m[1], i+1)
						break
					}
				}
			} else if len(marks) > 0 {
				fail("passage_question passages must not contain numbered blanks")
			}

			ids := map[int]bool{}
			for i, q := range p.Questions {
				qfail := func(format string, args ...any) { fail("question %d: %s", i+1, fmt.Sprintf(format, args...)) }
				if q.ID < 1 {
					qfail("id is required and must be positive")
				} else if ids[q.ID] {
					qfail("duplicate id %d", q.ID)
				}
				ids[q.ID] = true
				if t.Format == FormatPassageQuestion && strings.TrimSpace(q.Prompt) == "" {
					qfail("prompt is required")
				}
				if t.Format == FormatPassageBlank && q.Prompt != "" {
					qfail("blank questions take no prompt")
				}
				validateChoices(q.Choices, q.Answer, qfail)
				if strings.TrimSpace(q.ExplanationJa) == "" {
					qfail("explanation_ja is required")
				}
				for _, v := range q.Vocab {
					if !vocabKeys[v] {
						qfail("vocab %q does not exist", v)
					}
				}
			}
		}
	}
	return errs
}

// validateChoices applies the four-choice rules shared by every format.
func validateChoices(choices []string, answer *int, fail func(string, ...any)) {
	if len(choices) != 4 {
		fail("choices must have exactly 4 entries, got %d", len(choices))
	}
	seen := map[string]bool{}
	for _, c := range choices {
		c = strings.TrimSpace(c)
		if c == "" {
			fail("choices has an empty entry")
		}
		if seen[strings.ToLower(c)] {
			fail("choices has duplicate %q", c)
		}
		seen[strings.ToLower(c)] = true
	}
	if answer == nil {
		fail("answer is required")
	} else if *answer < 0 || *answer >= len(choices) {
		fail("answer %d is out of range", *answer)
	}
}
