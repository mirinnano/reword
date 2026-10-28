package content

import (
	"fmt"
	"sort"
	"strings"
	"unicode"
)

// Lint finds likely quality problems that Validate does not treat as errors:
// they need a human (or a solver audit) to decide. Each finding names the
// question and the rule.
type Finding struct {
	Key  string
	File string
	Rule string
	Msg  string
}

func (f Finding) String() string { return fmt.Sprintf("%s: %s [%s] %s", f.File, f.Key, f.Rule, f.Msg) }

const (
	minStemWords = 8
	maxStemWords = 30
	// Two stems sharing this share of word trigrams are near-duplicates.
	nearDupThreshold = 0.35
)

func Lint(s *Set) []Finding {
	var out []Finding
	vocab := map[string]VocabItem{}
	for _, items := range s.Vocab {
		for _, l := range items {
			vocab[l.Item.Key] = l.Item
		}
	}
	type stem struct {
		key, file string
		grams     map[string]bool
	}
	var stems []stem

	for _, items := range s.Questions {
		for _, l := range items {
			q := l.Item
			add := func(rule, format string, args ...any) {
				out = append(out, Finding{Key: q.Key, File: l.File, Rule: rule, Msg: fmt.Sprintf(format, args...)})
			}
			words := strings.Fields(blankRe.ReplaceAllString(q.Stem, "BLANK"))
			if n := len(words); n < minStemWords || n > maxStemWords {
				add("stem-length", "stem has %d words (want %d-%d)", n, minStemWords, maxStemWords)
			}

			// Choices should share shape: same number of words, same inflection.
			// Phrases of different lengths are normal ("put off", "make up for");
			// only a mix of single words and phrases is a giveaway.
			single, multi := 0, 0
			shapes := map[string]int{}
			for _, c := range q.Choices {
				if len(strings.Fields(c)) > 1 {
					multi++
				} else {
					single++
				}
				if sh := inflection(c); sh != "" {
					shapes[sh]++
				}
			}
			if single > 0 && multi > 0 {
				add("choice-shape", "choices mix single words and phrases: %q", q.Choices)
			}
			// Inflection only applies to verbs: "-ed"/"-ing" on adjectives and
			// nouns ("tired", "clothing") is not tense.
			if len(shapes) > 1 && answerIsVerb(q, vocab) {
				add("choice-inflection", "choices differ in inflection (%s): %q", describe(shapes), q.Choices)
			}

			// The answer should be a form of the linked headword.
			if q.Answer != nil && *q.Answer < len(q.Choices) {
				ans := strings.ToLower(q.Choices[*q.Answer])
				for _, k := range q.Vocab {
					v, ok := vocab[k]
					if ok && !sameLemma(ans, strings.ToLower(v.Headword)) {
						add("answer-vocab", "answer %q does not look like a form of linked headword %q", ans, v.Headword)
					}
				}
			}
			stems = append(stems, stem{key: q.Key, file: l.File, grams: trigrams(words)})
		}
	}

	sort.Slice(stems, func(i, j int) bool { return stems[i].key < stems[j].key })
	for i := range stems {
		for j := i + 1; j < len(stems); j++ {
			if sim := jaccard(stems[i].grams, stems[j].grams); sim >= nearDupThreshold {
				out = append(out, Finding{
					Key: stems[i].key, File: stems[i].file, Rule: "near-duplicate",
					Msg: fmt.Sprintf("stem overlaps %s (%.0f%% of word trigrams)", stems[j].key, sim*100),
				})
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Key != out[j].Key {
			return out[i].Key < out[j].Key
		}
		return out[i].Rule < out[j].Rule
	})
	return out
}

// inflection classifies a choice as past, -ing or base form by its first
// word (phrasal verbs inflect there: "carried out"). It only catches clear
// mismatches such as "conducted" next to "consume"; finer grammar is left to
// the solver audit.
func inflection(c string) string {
	f := strings.Fields(strings.ToLower(c))
	if len(f) == 0 {
		return "empty"
	}
	w := f[0]
	switch {
	case sameInAllForms[w]:
		return "" // "put", "set": base and past look alike, so they match anything
	case pastForms[w]:
		return "past"
	case strings.HasSuffix(w, "ing") && len(w) > 5:
		return "-ing"
	case strings.HasSuffix(w, "ed") && len(w) > 3:
		return "past"
	default:
		return "base"
	}
}

// answerIsVerb reports whether the question's linked headword is a verb or a
// phrasal verb; without a link it assumes so, keeping the check on.
func answerIsVerb(q Question, vocab map[string]VocabItem) bool {
	for _, k := range q.Vocab {
		if v, ok := vocab[k]; ok {
			return v.POS == "verb" || v.POS == "phrase"
		}
	}
	return true
}

var sameInAllForms = map[string]bool{
	"put": true, "set": true, "cut": true, "hit": true, "hurt": true, "let": true,
	"shut": true, "spread": true, "cost": true, "quit": true, "read": true,
}

var pastForms = func() map[string]bool {
	m := map[string]bool{}
	for _, w := range strings.Fields(`stole stolen withdrew withdrawn broke broken took taken ran stood
		dealt came made got gotten caught kept lost gave given brought held led sought wrote written
		began begun chose chosen drove driven fell fallen felt found grew grown hid hidden knew known
		left meant met paid sold sent spent spoke spoken stuck taught thought threw thrown told understood
		woke won wore worn forgave forgiven forbade forbidden overcame undertook bore born
		rose risen forgot forgotten`) {
		m[w] = true
	}
	return m
}()

func describe(m map[string]int) string {
	var parts []string
	for k, v := range m {
		parts = append(parts, fmt.Sprintf("%s×%d", k, v))
	}
	sort.Strings(parts)
	return strings.Join(parts, ", ")
}

// sameLemma accepts common inflections of a headword, including phrasal
// verbs inflected on their first word ("figured out" for "figure out").
func sameLemma(form, head string) bool {
	if form == head {
		return true
	}
	fw, hw := strings.Fields(form), strings.Fields(head)
	if len(fw) != len(hw) || len(fw) == 0 {
		return false
	}
	for i := 1; i < len(fw); i++ {
		if fw[i] != hw[i] {
			return false
		}
	}
	f, h := fw[0], hw[0]
	stem := strings.TrimSuffix(h, "e")
	stemY := strings.TrimSuffix(h, "y")
	for _, cand := range []string{
		h + "s", h + "es", h + "d", h + "ed", h + "ing", stem + "ing", stem + "ed",
		stemY + "ied", stemY + "ies", h + string(h[len(h)-1]) + "ed", h + string(h[len(h)-1]) + "ing",
	} {
		if f == cand {
			return true
		}
	}
	// Irregular past forms the lint cannot derive are left to the solver audit.
	return irregular[h] != "" && strings.Contains(irregular[h], f)
}

var irregular = map[string]string{
	"take": "took taken takes taking", "run": "ran runs running", "stand": "stood stands standing",
	"break": "broke broken breaks breaking", "turn": "turned", "get": "got gotten gets getting",
	"come": "came comes coming", "make": "made makes making", "put": "puts putting",
	"catch": "caught catches catching", "deal": "dealt deals dealing", "hand": "handed",
	"keep": "kept keeps keeping", "lose": "lost loses losing", "carry": "carried carries carrying",
	"seek": "sought", "lead": "led", "bring": "brought", "hold": "held", "give": "gave given",
}

func trigrams(words []string) map[string]bool {
	norm := make([]string, 0, len(words))
	for _, w := range words {
		w = strings.ToLower(strings.TrimFunc(w, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) }))
		if w != "" {
			norm = append(norm, w)
		}
	}
	g := map[string]bool{}
	for i := 0; i+2 < len(norm); i++ {
		g[norm[i]+" "+norm[i+1]+" "+norm[i+2]] = true
	}
	return g
}

func jaccard(a, b map[string]bool) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	inter := 0
	for k := range a {
		if b[k] {
			inter++
		}
	}
	return float64(inter) / float64(len(a)+len(b)-inter)
}
