package content

import (
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
)

// Grammar is taught, not only tested: a unit is a rule, examples and a short
// check, learned from the grammar screen rather than from the daily routine.
// The check questions are ordinary questions keyed "<unit key>-<id>", so they
// keep their history and come back through the review schedule.
//
// A wrong explanation of a rule teaches the wrong rule, so a unit cannot be
// imported until it has passed the audit: content/audits/grammar.yaml records
// the unit's hash, and any later change to the unit invalidates the record
// (docs/content-specs/grammar.md).
const (
	GrammarType      = "g2-gr"
	grammarQuestions = 6
	grammarExamples  = 3
	// GrammarPass is how many of a unit's check questions complete it.
	GrammarPass = 5
)

var grammarKeyRe = regexp.MustCompile(`^g2-gr-[a-z0-9]+(-[a-z0-9]+)*$`)

// GrammarFile is one unit, in content/<grade>/grammar/*.yaml.
type GrammarFile = GrammarUnit

type GrammarUnit struct {
	Grade     string            `yaml:"grade" json:"-"`
	Key       string            `yaml:"key" json:"-"`
	Group     string            `yaml:"group" json:"-"`
	Order     int               `yaml:"order" json:"-"`
	Title     string            `yaml:"title" json:"-"`
	RuleJa    string            `yaml:"rule_ja" json:"rule_ja"`
	Examples  []GrammarExample  `yaml:"examples" json:"examples"`
	Questions []GrammarQuestion `yaml:"questions" json:"-"`
}

type GrammarExample struct {
	En   string `yaml:"en" json:"en"`
	Ja   string `yaml:"ja" json:"ja"`
	Note string `yaml:"note,omitempty" json:"note,omitempty"`
}

type GrammarQuestion struct {
	// ID is permanent, like PassageQuestion.ID: it forms the stable key.
	ID            int      `yaml:"id"`
	Stem          string   `yaml:"stem"`
	Choices       []string `yaml:"choices"`
	Answer        *int     `yaml:"answer"`
	ExplanationJa string   `yaml:"explanation_ja"`
	Difficulty    int      `yaml:"difficulty"`
}

// GrammarAudit is one entry of content/audits/grammar.yaml.
type GrammarAudit struct {
	Unit     string   `yaml:"unit"`
	Hash     string   `yaml:"hash"`
	Solvers  []string `yaml:"solvers"`
	Reviewer string   `yaml:"reviewer"`
	Date     string   `yaml:"date"`
}

type GrammarAuditFile struct {
	Items []GrammarAudit `yaml:"items"`
}

// ExpandGrammar turns units into questions, in unit order.
func ExpandGrammar(s *Set) []Located[Question] {
	units := append([]Located[GrammarUnit](nil), s.Grammar...)
	sort.Slice(units, func(i, j int) bool { return units[i].Item.Order < units[j].Item.Order })
	var out []Located[Question]
	for _, l := range units {
		u := l.Item
		for i, gq := range u.Questions {
			out = append(out, Located[Question]{File: l.File, Item: Question{
				Key:           u.Key + "-" + strconv.Itoa(gq.ID),
				Stem:          gq.Stem,
				Choices:       gq.Choices,
				Answer:        gq.Answer,
				ExplanationJa: gq.ExplanationJa,
				Difficulty:    gq.Difficulty,
				Group:         u.Key,
				Seq:           i + 1,
			}})
		}
	}
	return out
}

// UnitHash covers everything a learner reads in a unit, so any change to the
// rule, an example or a question invalidates its audit.
func UnitHash(u GrammarUnit) string {
	return hex.EncodeToString(hashOf(u.Key, u.Group, u.Order, u.Title, u.RuleJa, u.Examples, u.Questions))
}

func validateGrammar(s *Set) []error {
	var errs []error
	keys := map[string]bool{}
	orders := map[int]string{}
	for _, l := range s.Grammar {
		u := l.Item
		fail := func(format string, args ...any) {
			errs = append(errs, fmt.Errorf("%s: %s: %s", l.File, u.Key, fmt.Sprintf(format, args...)))
		}
		switch {
		case !grammarKeyRe.MatchString(u.Key):
			fail("key must look like g2-gr-<name>")
		case keys[u.Key]:
			fail("duplicate key")
		}
		keys[u.Key] = true
		if u.Order < 1 {
			fail("order is required")
		} else if other, dup := orders[u.Order]; dup {
			fail("order %d is also used by %s", u.Order, other)
		}
		orders[u.Order] = u.Key
		if u.Group == "" || u.Title == "" || u.RuleJa == "" {
			fail("group, title and rule_ja are required")
		}
		if len(u.Examples) != grammarExamples {
			fail("needs %d examples, found %d", grammarExamples, len(u.Examples))
		}
		for i, e := range u.Examples {
			if e.En == "" || e.Ja == "" {
				fail("example %d needs en and ja", i+1)
			}
		}
		if len(u.Questions) != grammarQuestions {
			fail("needs %d questions, found %d", grammarQuestions, len(u.Questions))
		}
		ids := map[int]bool{}
		for _, gq := range u.Questions {
			if gq.ID < 1 || ids[gq.ID] {
				fail("question id %d must be positive and unique", gq.ID)
			}
			ids[gq.ID] = true
			validateMCQ(Question{Stem: gq.Stem, Choices: gq.Choices, Answer: gq.Answer}, func(format string, args ...any) {
				fail("question %d: %s", gq.ID, fmt.Sprintf(format, args...))
			})
			if gq.Difficulty < 1 || gq.Difficulty > 3 {
				fail("question %d: difficulty must be 1, 2 or 3", gq.ID)
			}
			if gq.ExplanationJa == "" {
				fail("question %d: explanation_ja is required", gq.ID)
			}
		}
	}
	return errs
}

// CheckAudits is the publishing gate: every unit needs a record, from at least
// two solvers and a reviewer, matching its current content. Lint, blind and
// review skip it, because they are how a draft reaches the audit.
func CheckAudits(s *Set) error {
	byUnit := map[string]GrammarAudit{}
	for _, a := range s.GrammarAudits {
		byUnit[a.Unit] = a
	}
	var errs []error
	for _, l := range s.Grammar {
		u := l.Item
		a, ok := byUnit[u.Key]
		switch {
		case !ok:
			errs = append(errs, fmt.Errorf("%s: %s: not audited", l.File, u.Key))
		case a.Hash != UnitHash(u):
			errs = append(errs, fmt.Errorf("%s: %s: changed since its audit, audit it again", l.File, u.Key))
		case len(a.Solvers) < 2 || a.Reviewer == "" || a.Date == "":
			errs = append(errs, fmt.Errorf("%s: %s: audit record needs two solvers, a reviewer and a date", l.File, u.Key))
		}
	}
	return errors.Join(errs...)
}

// ReviewUnit is a unit as a reviewer reads it, without its questions, which
// the question review covers.
type ReviewUnit struct {
	Key      string           `json:"key"`
	Group    string           `json:"group"`
	Title    string           `json:"title"`
	RuleJa   string           `json:"rule_ja"`
	Examples []GrammarExample `json:"examples"`
}

func ReviewUnits(units []GrammarUnit) []ReviewUnit {
	out := make([]ReviewUnit, len(units))
	for i, u := range units {
		out[i] = ReviewUnit{Key: u.Key, Group: u.Group, Title: u.Title, RuleJa: u.RuleJa, Examples: u.Examples}
	}
	return out
}
