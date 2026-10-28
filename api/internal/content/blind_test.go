package content

import "testing"

// Blind sets carry no answers and map back to every question exactly once.
func TestBlind(t *testing.T) {
	t.Parallel()
	a := 2
	set := &Set{Questions: map[string][]Located[Question]{"g2-q1": {
		{Item: Question{Key: "g2-q1-0001", Stem: "x ( )", Choices: []string{"a", "b", "c", "d"}, Answer: &a}},
		{Item: Question{Key: "g2-q1-0002", Stem: "y ( )", Choices: []string{"e", "f", "g", "h"}, Answer: &a}},
	}}}
	items, keys := Blind(set, 7)
	if len(items) != 2 || len(keys) != 2 {
		t.Fatalf("Blind() = %d items, %d keys, want 2 and 2", len(items), len(keys))
	}
	orig := map[string]string{"g2-q1-0001": "c", "g2-q1-0002": "g"}
	for i, k := range keys {
		if items[i].ID != k.ID || items[i].Choices[k.Answer] != orig[k.Key] {
			t.Errorf("key %+v does not point at the original answer in %v", k, items[i].Choices)
		}
	}
	again, _ := Blind(set, 7)
	if again[0].Choices[0] != items[0].Choices[0] || again[0].ID != items[0].ID {
		t.Error("same seed must give the same sets")
	}
}

func TestReviewItems(t *testing.T) {
	t.Parallel()
	a := 1
	set := &Set{
		Questions: map[string][]Located[Question]{"g2-q1": {
			{Item: Question{Key: "g2-q1-0002", Stem: "y ( )", Choices: []string{"e", "f", "g", "h"}, Answer: &a, Vocab: []string{"g2-v-f"}}},
			{Item: Question{Key: "g2-q1-0001", Stem: "x ( )", Choices: []string{"a", "b", "c", "d"}, Answer: &a}},
		}},
		Vocab: map[string][]Located[VocabItem]{"g2": {{Item: VocabItem{Key: "g2-v-f", Headword: "f", MeaningsJa: []string{"エフ"}}}}},
	}
	got := ReviewItems(set)
	if len(got) != 2 || got[0].Key != "g2-q1-0001" {
		t.Fatalf("ReviewItems() not sorted by key: %+v", got)
	}
	if got[1].Answer != 1 || len(got[1].Vocab) != 1 || got[1].Vocab[0].MeaningsJa[0] != "エフ" {
		t.Errorf("ReviewItems() lost the answer or vocab: %+v", got[1])
	}
}
