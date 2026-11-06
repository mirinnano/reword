// Package content loads, validates and imports learning content from the
// YAML files under content/ (format: docs/content-specs/README.md).
package content

// QuestionType is one kind of question (大問). This registry is the source of
// truth for the question_types table.
type QuestionType struct {
	Code      string
	Grade     string
	NameJa    string
	Format    string // one of the Format* constants
	SortOrder uint16
}

// Generated reports whether the type's questions come from GenerateDrills
// rather than from YAML files.
func (t QuestionType) Generated() bool {
	return t.Format == FormatVocabMeaning || t.Format == FormatVocabWord
}

// Passage reports whether the type is written as passages with several
// questions each (PassageFile) rather than one question per item.
func (t QuestionType) Passage() bool {
	return t.Format == FormatPassageBlank || t.Format == FormatPassageQuestion
}

var QuestionTypes = []QuestionType{
	{Code: "g2-q1", Grade: "g2", NameJa: "語彙・熟語の空所補充", Format: FormatMCQ, SortOrder: 1},
	{Code: "g2-q2", Grade: "g2", NameJa: "長文の語句空所補充", Format: FormatPassageBlank, SortOrder: 2},
	{Code: "g2-q3", Grade: "g2", NameJa: "長文の内容一致選択", Format: FormatPassageQuestion, SortOrder: 3},
	{Code: "g2-vm", Grade: "g2", NameJa: "単語：意味を選ぶ", Format: FormatVocabMeaning, SortOrder: 10},
	{Code: "g2-vw", Grade: "g2", NameJa: "単語：英語を選ぶ", Format: FormatVocabWord, SortOrder: 11},
	{Code: GrammarType, Grade: "g2", NameJa: "文法", Format: FormatMCQ, SortOrder: 20},
	{Code: "pre2-q1", Grade: "pre2", NameJa: "語彙・熟語の空所補充", Format: FormatMCQ, SortOrder: 1},
}

func questionType(code string) (QuestionType, bool) {
	for _, t := range QuestionTypes {
		if t.Code == code {
			return t, true
		}
	}
	return QuestionType{}, false
}

// Grades that content may target (codes from the grades table).
var Grades = map[string]bool{"pre2": true, "pre2plus": true, "g2": true, "pre1": true, "g1": true}

var PartsOfSpeech = map[string]bool{
	"noun": true, "verb": true, "adjective": true, "adverb": true, "phrase": true, "other": true,
}

// File shapes as written in YAML.

type VocabFile struct {
	Grade string      `yaml:"grade"`
	Items []VocabItem `yaml:"items"`
}

type VocabItem struct {
	Key        string   `yaml:"key" json:"-"`
	Headword   string   `yaml:"headword" json:"-"`
	POS        string   `yaml:"pos" json:"-"`
	MeaningsJa []string `yaml:"meanings_ja" json:"meanings_ja"`
	ExampleEn  string   `yaml:"example_en" json:"example_en"`
	ExampleJa  string   `yaml:"example_ja" json:"example_ja"`
}

type QuestionFile struct {
	Grade string     `yaml:"grade"`
	Type  string     `yaml:"type"`
	Items []Question `yaml:"items"`
}

type Question struct {
	Key           string   `yaml:"key"`
	Stem          string   `yaml:"stem"`
	Choices       []string `yaml:"choices"`
	Answer        *int     `yaml:"answer"`
	ExplanationJa string   `yaml:"explanation_ja"`
	Vocab         []string `yaml:"vocab"`
	Difficulty    int      `yaml:"difficulty"`

	// Set only for questions expanded from a passage (ExpandPassages).
	Group   string `yaml:"-"` // the passage key
	Title   string `yaml:"-"`
	Passage string `yaml:"-"`
	Seq     int    `yaml:"-"` // position in the passage; passage_blank: fills "( Seq )"
}

// PassageFile holds passage-based questions: one passage, several questions.
type PassageFile struct {
	Grade string    `yaml:"grade"`
	Type  string    `yaml:"type"`
	Items []Passage `yaml:"items"`
}

type Passage struct {
	Key        string            `yaml:"key"`
	Title      string            `yaml:"title"`
	Passage    string            `yaml:"passage"`
	Difficulty int               `yaml:"difficulty"`
	Questions  []PassageQuestion `yaml:"questions"`
}

type PassageQuestion struct {
	// ID is the question's permanent number inside its passage. It forms the
	// stable key, so history stays with the question when others are added,
	// removed or reordered. Never change or reuse one.
	ID            int      `yaml:"id"`
	Prompt        string   `yaml:"prompt"` // passage_question only
	Choices       []string `yaml:"choices"`
	Answer        *int     `yaml:"answer"`
	ExplanationJa string   `yaml:"explanation_ja"`
	Vocab         []string `yaml:"vocab"`
}

// Set is everything loaded from one content directory.
type Set struct {
	Vocab     map[string][]Located[VocabItem] // by grade
	Questions map[string][]Located[Question]  // by question type code
	Passages  map[string][]Located[Passage]   // by question type code
	Grammar   []Located[GrammarUnit]
	// GrammarAudits is content/audits/grammar.yaml.
	GrammarAudits []GrammarAudit
}

// Located remembers where an item came from, for error messages.
type Located[T any] struct {
	File string
	Item T
}
