package content

import (
	"fmt"
	"math/rand/v2"
	"reflect"
	"testing"
)

// randomVocab builds a vocab set from a seed: a few parts of speech, and
// meanings drawn from a small pool so that overlaps actually happen.
func randomVocab(seed uint64, n int) *Set {
	rng := rand.New(rand.NewPCG(seed, seed^0xabcdef))
	posList := []string{"verb", "noun", "adjective"}
	pool := []string{"延期する", "先延ばしにする", "調べる", "詳しく調べる", "保存する", "減らす", "増やす", "許可", "量", "速い", "正確な", "希少な"}
	items := make([]Located[VocabItem], n)
	for i := range items {
		m := []string{pool[rng.IntN(len(pool))]}
		if rng.IntN(2) == 0 {
			m = append(m, "（注記）"+pool[rng.IntN(len(pool))])
		}
		items[i] = Located[VocabItem]{File: "t.yaml", Item: VocabItem{
			Key: fmt.Sprintf("g2-v-w%d", i), Headword: fmt.Sprintf("w%d", i),
			POS: posList[rng.IntN(len(posList))], MeaningsJa: m, ExampleEn: "e", ExampleJa: "j",
		}}
	}
	return &Set{Vocab: map[string][]Located[VocabItem]{"g2": items}}
}

func FuzzGenerateDrills(f *testing.F) {
	for _, s := range []uint64{1, 2, 3, 42, 1000} {
		f.Add(s, uint8(30))
	}
	f.Fuzz(func(t *testing.T, seed uint64, size uint8) {
		set := randomVocab(seed, int(size%60)+4)
		byKey := map[string]VocabItem{}
		byHead := map[string]VocabItem{}
		for _, l := range set.Vocab["g2"] {
			byKey[l.Item.Key] = l.Item
			byHead[l.Item.Headword] = l.Item
		}
		drills := GenerateDrills(set)
		for _, t2 := range QuestionTypes {
			if !t2.Generated() {
				continue
			}
			for _, l := range drills[t2.Code] {
				q := l.Item
				if len(q.Choices) != 4 {
					t.Fatalf("%s: %d choices", q.Key, len(q.Choices))
				}
				seen := map[string]bool{}
				for _, c := range q.Choices {
					if seen[c] {
						t.Fatalf("%s: duplicate choice %q in %q", q.Key, c, q.Choices)
					}
					seen[c] = true
				}
				own := byKey[q.Vocab[0]]
				want := own.Headword
				if t2.Format == FormatVocabMeaning {
					want = primaryMeaning(own)
				}
				if q.Choices[*q.Answer] != want {
					t.Fatalf("%s: answer %q, want %q", q.Key, q.Choices[*q.Answer], want)
				}
				if t2.Format == FormatVocabWord {
					for i, c := range q.Choices {
						if i == *q.Answer {
							continue
						}
						d := byHead[c]
						if d.POS != own.POS {
							t.Fatalf("%s: distractor %q has POS %s, answer %s", q.Key, c, d.POS, own.POS)
						}
						if overlaps(meaningSet(d), meaningSet(own)) {
							t.Fatalf("%s: distractor %q shares a meaning with %q", q.Key, c, own.Headword)
						}
					}
				}
			}
		}
		if again := GenerateDrills(set); !reflect.DeepEqual(again, drills) {
			t.Fatal("GenerateDrills is not deterministic")
		}
	})
}
