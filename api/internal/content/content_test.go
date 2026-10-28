package content

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mirinnano/reword/api/internal/platform/db"
)

func write(t *testing.T, root, rel, body string) {
	t.Helper()
	p := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

const vocabOK = `grade: g2
items:
  - key: g2-v-abandon
    headword: abandon
    pos: verb
    meanings_ja: ["断念する"]
    example_en: They abandoned the plan.
    example_ja: 彼らは計画を断念した。
`

func question(key, stem, choices, answer string) string {
	return `grade: g2
type: g2-q1
items:
  - key: ` + key + `
    stem: "` + stem + `"
    choices: ` + choices + `
    answer: ` + answer + `
    explanation_ja: 説明
    vocab: [g2-v-abandon]
    difficulty: 2
`
}

func TestValidate(t *testing.T) {
	cases := []struct {
		name, file, body, wantErr string
	}{
		{"ok", "g2/questions/g2-q1/a.yaml", question("g2-q1-0001", "We had to ( ) it.", "[abandon, adopt, admit, adjust]", "0"), ""},
		{"no blank", "g2/questions/g2-q1/a.yaml", question("g2-q1-0001", "We had to stop it.", "[abandon, adopt, admit, adjust]", "0"), "exactly one blank"},
		{"two blanks", "g2/questions/g2-q1/a.yaml", question("g2-q1-0001", "( ) had to ( ) it.", "[abandon, adopt, admit, adjust]", "0"), "exactly one blank"},
		{"three choices", "g2/questions/g2-q1/a.yaml", question("g2-q1-0001", "We had to ( ) it.", "[abandon, adopt, admit]", "0"), "exactly 4"},
		{"duplicate choice", "g2/questions/g2-q1/a.yaml", question("g2-q1-0001", "We had to ( ) it.", "[abandon, Abandon, admit, adjust]", "0"), "duplicate"},
		{"answer out of range", "g2/questions/g2-q1/a.yaml", question("g2-q1-0001", "We had to ( ) it.", "[abandon, adopt, admit, adjust]", "4"), "out of range"},
		{"bad key", "g2/questions/g2-q1/a.yaml", question("q1-1", "We had to ( ) it.", "[abandon, adopt, admit, adjust]", "0"), "key must look like"},
		{"unknown field", "g2/vocab/b.yaml", "grade: g2\nitems:\n  - key: g2-v-x\n    color: red\n", "field color not found"},
		{"grade/dir mismatch", "g2/vocab/b.yaml", strings.Replace(vocabOK, "grade: g2", "grade: pre1", 1), "does not match directory"},
		{"stray file", "g2/notes.yaml", "x: 1\n", "unexpected location"},
		{"unknown vocab ref", "g2/questions/g2-q1/a.yaml", strings.Replace(question("g2-q1-0001", "We had to ( ) it.", "[abandon, adopt, admit, adjust]", "0"), "g2-v-abandon", "g2-v-missing", 1), "does not exist"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			write(t, root, "g2/vocab/a.yaml", vocabOK)
			write(t, root, tc.file, tc.body)
			s, err := Load(root)
			if err == nil {
				err = Validate(s)
			}
			switch {
			case tc.wantErr == "" && err != nil:
				t.Fatalf("unexpected error: %v", err)
			case tc.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tc.wantErr)):
				t.Fatalf("error %v, want it to contain %q", err, tc.wantErr)
			}
		})
	}
}

// The real content directory must always validate.
func TestRepositoryContentIsValid(t *testing.T) {
	s, err := Load("../../../content")
	if err != nil {
		t.Fatal(err)
	}
	if err := Validate(s); err != nil {
		t.Fatal(err)
	}
}

// Import against a real database, scoped by KeyPrefix to a test-only type
// and vocab keys so shared databases keep their real content.
func TestImportVersioning(t *testing.T) {
	dsn := os.Getenv("DATABASE_DSN")
	if dsn == "" {
		t.Skip("DATABASE_DSN not set")
	}
	ctx := context.Background()
	pool, err := db.Open(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	QuestionTypes = append(QuestionTypes, QuestionType{Code: "g2-q99", Grade: "g2", NameJa: "テスト", Format: "mcq", SortOrder: 99})
	t.Cleanup(func() {
		QuestionTypes = QuestionTypes[:len(QuestionTypes)-1]
		pool.Exec("DELETE FROM questions WHERE stable_key LIKE 'g2-q99-%'")
		pool.Exec("DELETE FROM question_types WHERE code = 'g2-q99'")
		pool.Exec("DELETE FROM vocab_items WHERE stable_key LIKE 'g2-v-zztest%'")
	})
	// Only our test keys may be touched or retired.
	scoped := func(s *Set) (Report, error) {
		r1, err := Import(ctx, pool, &Set{Questions: map[string][]Located[Question]{"g2-q99": s.Questions["g2-q99"]}}, ImportOptions{KeyPrefix: "g2-q99-"})
		if err != nil {
			return r1, err
		}
		r2, err := Import(ctx, pool, &Set{Vocab: map[string][]Located[VocabItem]{"g2": s.Vocab["g2"]}}, ImportOptions{KeyPrefix: "g2-v-zztest"})
		r1.VocabAdded, r1.VocabUpdated, r1.VocabRetired, r1.VocabUnchanged = r2.VocabAdded, r2.VocabUpdated, r2.VocabRetired, r2.VocabUnchanged
		return r1, err
	}

	ans := 0
	q := func(key, stem string) Located[Question] {
		a := ans
		return Located[Question]{Item: Question{Key: key, Stem: stem, Choices: []string{"a", "b", "c", "d"}, Answer: &a, ExplanationJa: "x", Difficulty: 1}}
	}
	v := func(key, meaning string) Located[VocabItem] {
		return Located[VocabItem]{Item: VocabItem{Key: key, Headword: "w", POS: "noun", MeaningsJa: []string{meaning}, ExampleEn: "e", ExampleJa: "j"}}
	}
	set := func(qs []Located[Question], vs []Located[VocabItem]) *Set {
		return &Set{Questions: map[string][]Located[Question]{"g2-q99": qs}, Vocab: map[string][]Located[VocabItem]{"g2": vs}}
	}
	check := func(step string, r Report, want Report) {
		t.Helper()
		if r != want {
			t.Fatalf("%s: got %+v\nwant %+v", step, r, want)
		}
	}

	r, err := scoped(set([]Located[Question]{q("g2-q99-0001", "A ( )"), q("g2-q99-0002", "B ( )")}, []Located[VocabItem]{v("g2-v-zztest-a", "意味")}))
	if err != nil {
		t.Fatal(err)
	}
	check("first import", r, Report{QuestionsAdded: 2, VocabAdded: 1})

	r, _ = scoped(set([]Located[Question]{q("g2-q99-0001", "A ( )"), q("g2-q99-0002", "B ( )")}, []Located[VocabItem]{v("g2-v-zztest-a", "意味")}))
	check("re-import unchanged", r, Report{QuestionsUnchanged: 2, VocabUnchanged: 1})

	r, _ = scoped(set([]Located[Question]{q("g2-q99-0001", "A changed ( )")}, []Located[VocabItem]{v("g2-v-zztest-a", "新しい意味")}))
	check("edit one, drop one", r, Report{QuestionsNewVersion: 1, QuestionsRetired: 1, VocabUpdated: 1})

	r, _ = scoped(set([]Located[Question]{q("g2-q99-0001", "A changed ( )"), q("g2-q99-0002", "B ( )")}, nil))
	check("restore dropped, drop vocab", r, Report{QuestionsUnchanged: 1, QuestionsRestored: 1, VocabRetired: 1})

	var published, total int
	if err := pool.QueryRow(`SELECT SUM(status='published'), COUNT(*) FROM questions WHERE stable_key = 'g2-q99-0001'`).Scan(&published, &total); err != nil {
		t.Fatal(err)
	}
	if published != 1 || total != 2 {
		t.Fatalf("g2-q99-0001: %d published of %d versions, want 1 of 2", published, total)
	}

	r, _ = Import(ctx, pool, set(nil, nil), ImportOptions{KeyPrefix: "g2-q99-", DryRun: true})
	if r.QuestionsRetired != 2 {
		t.Fatalf("dry run should report 2 retirements, got %+v", r)
	}
	if err := pool.QueryRow(`SELECT COUNT(*) FROM questions WHERE stable_key LIKE 'g2-q99-%' AND status='published'`).Scan(&published); err != nil || published != 2 {
		t.Fatalf("dry run wrote changes: %d published, err %v", published, err)
	}
}
