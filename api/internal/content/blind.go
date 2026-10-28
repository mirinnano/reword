package content

import (
	"fmt"
	"math/rand/v2"
	"sort"
)

// BlindItem is a question as an independent solver sees it: no answer, no
// explanation, no vocab link, choices reshuffled, and an opaque id so the
// key's numbering gives nothing away.
type BlindItem struct {
	ID      string   `json:"id"`
	Stem    string   `json:"stem"`
	Choices []string `json:"choices"`
}

// BlindKey maps a blind id back to the question and its correct choice in
// the shuffled order.
type BlindKey struct {
	ID         string `json:"id"`
	Key        string `json:"key"`
	File       string `json:"file"`
	Answer     int    `json:"answer"`
	Difficulty int    `json:"difficulty"`
}

// Blind prepares every question of every type for a solver audit. The same
// seed gives the same ids and orders, so audits can be re-run and compared.
func Blind(s *Set, seed uint64) ([]BlindItem, []BlindKey) {
	var all []Located[Question]
	for _, items := range s.Questions {
		all = append(all, items...)
	}
	sort.Slice(all, func(i, j int) bool { return all[i].Item.Key < all[j].Item.Key })
	rng := rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15))
	rng.Shuffle(len(all), func(i, j int) { all[i], all[j] = all[j], all[i] })

	items := make([]BlindItem, len(all))
	keys := make([]BlindKey, len(all))
	for i, l := range all {
		q := l.Item
		perm := rng.Perm(len(q.Choices))
		choices := make([]string, len(perm))
		answer := -1
		for d, orig := range perm {
			choices[d] = q.Choices[orig]
			if orig == *q.Answer {
				answer = d
			}
		}
		id := fmt.Sprintf("B%03d", i+1)
		items[i] = BlindItem{ID: id, Stem: q.Stem, Choices: choices}
		keys[i] = BlindKey{ID: id, Key: q.Key, File: l.File, Answer: answer, Difficulty: q.Difficulty}
	}
	return items, keys
}

// ReviewItem is a question with everything a reviewer needs to check it.
type ReviewItem struct {
	Key           string      `json:"key"`
	Stem          string      `json:"stem"`
	Choices       []string    `json:"choices"`
	Answer        int         `json:"answer"`
	ExplanationJa string      `json:"explanation_ja"`
	Difficulty    int         `json:"difficulty"`
	Vocab         []VocabView `json:"vocab"`
}

type VocabView struct {
	Key        string   `json:"key"`
	Headword   string   `json:"headword"`
	POS        string   `json:"pos"`
	MeaningsJa []string `json:"meanings_ja"`
	ExampleEn  string   `json:"example_en"`
	ExampleJa  string   `json:"example_ja"`
}

func ReviewItems(s *Set) []ReviewItem {
	vocab := map[string]VocabItem{}
	for _, items := range s.Vocab {
		for _, l := range items {
			vocab[l.Item.Key] = l.Item
		}
	}
	var out []ReviewItem
	for _, items := range s.Questions {
		for _, l := range items {
			q := l.Item
			r := ReviewItem{Key: q.Key, Stem: q.Stem, Choices: q.Choices, Answer: *q.Answer, ExplanationJa: q.ExplanationJa, Difficulty: q.Difficulty}
			for _, k := range q.Vocab {
				v := vocab[k]
				r.Vocab = append(r.Vocab, VocabView{Key: k, Headword: v.Headword, POS: v.POS, MeaningsJa: v.MeaningsJa, ExampleEn: v.ExampleEn, ExampleJa: v.ExampleJa})
			}
			out = append(out, r)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out
}
