package content

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var (
	vocabKeyRe    = regexp.MustCompile(`^[a-z0-9]+-v-[a-z0-9]+(-[a-z0-9]+)*$`)
	questionKeyRe = regexp.MustCompile(`^[a-z0-9]+-q[0-9]+-[0-9]{4,}$`)
	blankRe       = regexp.MustCompile(`\(\s*\)`)
)

// Validate checks every rule in docs/content-specs/README.md and returns all
// problems at once, each prefixed with its file and key.
func Validate(s *Set) error {
	var errs []error
	fail := func(file, key, format string, args ...any) {
		errs = append(errs, fmt.Errorf("%s: %s: %s", file, key, fmt.Sprintf(format, args...)))
	}

	vocabKeys := map[string]bool{}
	for grade, items := range s.Vocab {
		if !Grades[grade] {
			errs = append(errs, fmt.Errorf("vocab: unknown grade %q", grade))
		}
		for _, l := range items {
			it := l.Item
			switch {
			case !vocabKeyRe.MatchString(it.Key) || !strings.HasPrefix(it.Key, grade+"-v-"):
				fail(l.File, it.Key, "key must look like %s-v-<word>", grade)
			case vocabKeys[it.Key]:
				fail(l.File, it.Key, "duplicate key")
			}
			vocabKeys[it.Key] = true
			if strings.TrimSpace(it.Headword) == "" || len(it.Headword) > 100 {
				fail(l.File, it.Key, "headword is required (max 100 chars)")
			}
			if !PartsOfSpeech[it.POS] {
				fail(l.File, it.Key, "pos %q is not one of noun|verb|adjective|adverb|phrase|other", it.POS)
			}
			if len(it.MeaningsJa) == 0 {
				fail(l.File, it.Key, "meanings_ja needs at least one meaning")
			}
			for _, m := range it.MeaningsJa {
				if strings.TrimSpace(m) == "" {
					fail(l.File, it.Key, "meanings_ja has an empty entry")
				}
			}
			if strings.TrimSpace(it.ExampleEn) == "" || strings.TrimSpace(it.ExampleJa) == "" {
				fail(l.File, it.Key, "example_en and example_ja are required")
			}
		}
	}

	questionKeys := map[string]bool{}
	for code, items := range s.Questions {
		t, ok := questionType(code)
		if !ok {
			errs = append(errs, fmt.Errorf("questions: unknown type %q (register it in internal/content/model.go)", code))
			continue
		}
		for _, l := range items {
			q := l.Item
			switch {
			case !questionKeyRe.MatchString(q.Key) || !strings.HasPrefix(q.Key, code+"-"):
				fail(l.File, q.Key, "key must look like %s-0001", code)
			case questionKeys[q.Key]:
				fail(l.File, q.Key, "duplicate key")
			}
			questionKeys[q.Key] = true
			if t.Format == "mcq" {
				validateMCQ(q, func(format string, args ...any) { fail(l.File, q.Key, format, args...) })
			}
			if q.Difficulty < 1 || q.Difficulty > 3 {
				fail(l.File, q.Key, "difficulty must be 1, 2 or 3")
			}
			if strings.TrimSpace(q.ExplanationJa) == "" {
				fail(l.File, q.Key, "explanation_ja is required")
			}
		}
	}

	// Vocab references must resolve.
	for _, items := range s.Questions {
		for _, l := range items {
			for _, v := range l.Item.Vocab {
				if !vocabKeys[v] {
					fail(l.File, l.Item.Key, "vocab %q does not exist", v)
				}
			}
		}
	}
	return errors.Join(errs...)
}

func validateMCQ(q Question, fail func(string, ...any)) {
	if n := len(blankRe.FindAllString(q.Stem, -1)); n != 1 {
		fail("stem must contain exactly one blank \"( )\", found %d", n)
	}
	if len(q.Choices) != 4 {
		fail("choices must have exactly 4 entries, got %d", len(q.Choices))
	}
	seen := map[string]bool{}
	for _, c := range q.Choices {
		c = strings.TrimSpace(c)
		if c == "" {
			fail("choices has an empty entry")
		}
		if seen[strings.ToLower(c)] {
			fail("choices has duplicate %q", c)
		}
		seen[strings.ToLower(c)] = true
	}
	if q.Answer == nil {
		fail("answer is required")
	} else if *q.Answer < 0 || *q.Answer >= len(q.Choices) {
		fail("answer %d is out of range", *q.Answer)
	}
}
