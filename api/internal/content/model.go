// Package content loads, validates and imports learning content from the
// YAML files under content/ (format: docs/content-specs/README.md).
package content

// QuestionType is one kind of question (大問). This registry is the source of
// truth for the question_types table.
type QuestionType struct {
	Code      string
	Grade     string
	NameJa    string
	Format    string // FormatMCQ, FormatVocabMeaning or FormatVocabWord
	SortOrder uint16
}

// Generated reports whether the type's questions come from GenerateDrills
// rather than from YAML files.
func (t QuestionType) Generated() bool { return t.Format != FormatMCQ }

var QuestionTypes = []QuestionType{
	{Code: "g2-q1", Grade: "g2", NameJa: "語彙・熟語の空所補充", Format: FormatMCQ, SortOrder: 1},
	{Code: "g2-vm", Grade: "g2", NameJa: "単語：意味を選ぶ", Format: FormatVocabMeaning, SortOrder: 10},
	{Code: "g2-vw", Grade: "g2", NameJa: "単語：英語を選ぶ", Format: FormatVocabWord, SortOrder: 11},
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
}

// Set is everything loaded from one content directory.
type Set struct {
	Vocab     map[string][]Located[VocabItem] // by grade
	Questions map[string][]Located[Question]  // by question type code
}

// Located remembers where an item came from, for error messages.
type Located[T any] struct {
	File string
	Item T
}
