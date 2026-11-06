package content

import (
	"strings"
	"testing"
)

func validUnit() GrammarUnit {
	a := 0
	u := GrammarUnit{
		Grade: "g2", Key: "g2-gr-perfect", Group: "時制", Order: 1, Title: "現在完了", RuleJa: "規則",
		Examples: []GrammarExample{{En: "a", Ja: "あ"}, {En: "b", Ja: "い"}, {En: "c", Ja: "う"}},
	}
	for i := 1; i <= 6; i++ {
		u.Questions = append(u.Questions, GrammarQuestion{
			ID: i, Stem: "I ( ) it.", Choices: []string{"did", "do", "does", "doing"}, Answer: &a, ExplanationJa: "解説", Difficulty: 1,
		})
	}
	return u
}

func TestAuditGate(t *testing.T) {
	u := validUnit()
	good := GrammarAudit{Unit: u.Key, Hash: UnitHash(u), Solvers: []string{"a", "b"}, Reviewer: "c", Date: "2026-11-06"}
	set := func(u GrammarUnit, a ...GrammarAudit) *Set {
		return &Set{Grammar: []Located[GrammarUnit]{{File: "f", Item: u}}, GrammarAudits: a}
	}
	if err := CheckAudits(set(u, good)); err != nil {
		t.Fatalf("audited unit rejected: %v", err)
	}
	if err := CheckAudits(set(u)); err == nil || !strings.Contains(err.Error(), "not audited") {
		t.Fatalf("unaudited unit: %v", err)
	}
	oneSolver := good
	oneSolver.Solvers = []string{"a"}
	if err := CheckAudits(set(u, oneSolver)); err == nil {
		t.Fatal("one solver must not count as an audit")
	}
	noReviewer := good
	noReviewer.Reviewer = ""
	if err := CheckAudits(set(u, noReviewer)); err == nil {
		t.Fatal("an audit without a reviewer must not count")
	}
	// Every part a learner reads is covered by the hash.
	for name, change := range map[string]func(*GrammarUnit){
		"rule":    func(u *GrammarUnit) { u.RuleJa += "。" },
		"example": func(u *GrammarUnit) { u.Examples = append([]GrammarExample{{En: "z", Ja: "ん"}}, u.Examples[1:]...) },
		"explanation": func(u *GrammarUnit) {
			u.Questions = append([]GrammarQuestion{}, u.Questions...)
			u.Questions[0].ExplanationJa += "。"
		},
		"answer": func(u *GrammarUnit) {
			one := 1
			u.Questions = append([]GrammarQuestion{}, u.Questions...)
			u.Questions[0].Answer = &one
		},
	} {
		c := validUnit()
		change(&c)
		if err := CheckAudits(set(c, good)); err == nil || !strings.Contains(err.Error(), "changed") {
			t.Errorf("%s changed after audit: %v", name, err)
		}
	}
}

func TestValidateGrammar(t *testing.T) {
	if errs := validateGrammar(&Set{Grammar: []Located[GrammarUnit]{{File: "f", Item: validUnit()}}}); len(errs) != 0 {
		t.Fatalf("valid unit: %v", errs)
	}
	short := validUnit()
	short.Questions = short.Questions[:5]
	dupID := validUnit()
	dupID.Questions[1].ID = 1
	badKey := validUnit()
	badKey.Key = "perfect"
	for name, u := range map[string]GrammarUnit{"five questions": short, "duplicate id": dupID, "bad key": badKey} {
		if errs := validateGrammar(&Set{Grammar: []Located[GrammarUnit]{{File: "f", Item: u}}}); len(errs) == 0 {
			t.Errorf("%s must fail", name)
		}
	}
	two := []Located[GrammarUnit]{{File: "f", Item: validUnit()}, {File: "g", Item: validUnit()}}
	if errs := validateGrammar(&Set{Grammar: two}); len(errs) == 0 {
		t.Error("a duplicate unit must fail")
	}
}

func TestExpandGrammar(t *testing.T) {
	first, second := validUnit(), validUnit()
	second.Key, second.Order = "g2-gr-past", 2
	first.Order = 3
	qs := ExpandGrammar(&Set{Grammar: []Located[GrammarUnit]{{Item: first}, {Item: second}}})
	if len(qs) != 12 || qs[0].Item.Key != "g2-gr-past-1" || qs[6].Item.Group != "g2-gr-perfect" || qs[5].Item.Seq != 6 {
		t.Fatalf("expanded: %+v", qs[0].Item)
	}
}
