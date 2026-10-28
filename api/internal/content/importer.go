package content

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/mirinnano/reword/api/internal/platform/store"
)

// Report counts what an import changed (or would change, in a dry run).
type Report struct {
	QuestionsAdded, QuestionsNewVersion, QuestionsRetired, QuestionsRestored, QuestionsUnchanged int
	VocabAdded, VocabUpdated, VocabRetired, VocabUnchanged                                       int
}

func (r Report) String() string {
	return fmt.Sprintf("questions: +%d new, %d new versions, %d retired, %d restored, %d unchanged; "+
		"vocab: +%d new, %d updated, %d retired, %d unchanged",
		r.QuestionsAdded, r.QuestionsNewVersion, r.QuestionsRetired, r.QuestionsRestored, r.QuestionsUnchanged,
		r.VocabAdded, r.VocabUpdated, r.VocabRetired, r.VocabUnchanged)
}

type ImportOptions struct {
	// DryRun rolls the transaction back after computing the report.
	DryRun bool
	// KeyPrefix limits the sync to stable keys with this prefix: rows outside
	// it are neither updated nor retired. Tests use it so they never retire
	// real content in a shared database. Empty means everything.
	KeyPrefix string
}

// Import makes the database match s in one transaction. Items missing from s
// are retired, never deleted.
func Import(ctx context.Context, db *sql.DB, s *Set, opt ImportOptions) (Report, error) {
	var r Report
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return r, err
	}
	defer tx.Rollback()
	q := store.New(db).WithTx(tx)

	gradeIDs := map[string]uint8{}
	gradeID := func(code string) (uint8, error) {
		if id, ok := gradeIDs[code]; ok {
			return id, nil
		}
		id, err := q.GetGradeIDByCode(ctx, code)
		if err != nil {
			return 0, fmt.Errorf("grade %q: %w", code, err)
		}
		gradeIDs[code] = id
		return id, nil
	}

	for _, t := range QuestionTypes {
		gid, err := gradeID(t.Grade)
		if err != nil {
			return r, err
		}
		if err := q.UpsertQuestionType(ctx, store.UpsertQuestionTypeParams{
			Code: t.Code, GradeID: gid, NameJa: t.NameJa, Format: store.QuestionTypesFormat(t.Format), SortOrder: t.SortOrder,
		}); err != nil {
			return r, fmt.Errorf("upsert type %s: %w", t.Code, err)
		}
		if err := importQuestions(ctx, q, t, s.Questions[t.Code], opt.KeyPrefix, &r); err != nil {
			return r, fmt.Errorf("type %s: %w", t.Code, err)
		}
	}

	grades := make([]string, 0, len(Grades))
	for g := range Grades {
		grades = append(grades, g)
	}
	sort.Strings(grades)
	for _, g := range grades {
		gid, err := gradeID(g)
		if err != nil {
			return r, err
		}
		if err := importVocab(ctx, q, gid, s.Vocab[g], opt.KeyPrefix, &r); err != nil {
			return r, fmt.Errorf("vocab %s: %w", g, err)
		}
	}

	if opt.DryRun {
		return r, nil
	}
	return r, tx.Commit()
}

type mcqPayload struct {
	Stem    string   `json:"stem"`
	Choices []string `json:"choices"`
}

type mcqAnswer struct {
	Index int `json:"index"`
}

func importQuestions(ctx context.Context, q *store.Queries, t QuestionType, items []Located[Question], prefix string, r *Report) error {
	typeID, err := q.GetQuestionTypeID(ctx, t.Code)
	if err != nil {
		return err
	}
	rows, err := q.ListQuestionVersionsByType(ctx, typeID)
	if err != nil {
		return err
	}
	// Rows are ordered newest version first per key.
	latest := map[string]store.ListQuestionVersionsByTypeRow{}
	for _, row := range rows {
		if _, ok := latest[row.StableKey]; !ok && strings.HasPrefix(row.StableKey, prefix) {
			latest[row.StableKey] = row
		}
	}

	inFiles := map[string]bool{}
	for _, l := range items {
		it := l.Item
		if !strings.HasPrefix(it.Key, prefix) {
			continue
		}
		inFiles[it.Key] = true
		payload, _ := json.Marshal(mcqPayload{Stem: it.Stem, Choices: it.Choices})
		answer, _ := json.Marshal(mcqAnswer{Index: *it.Answer})
		vocab := it.Vocab
		if vocab == nil {
			vocab = []string{}
		}
		vocabJSON, _ := json.Marshal(vocab)
		hash := hashOf(payload, answer, it.ExplanationJa, it.Difficulty, vocabJSON)

		prev, exists := latest[it.Key]
		switch {
		case exists && bytes.Equal(prev.ContentHash, hash):
			if prev.Status == store.QuestionsStatusRetired {
				if err := q.SetQuestionStatus(ctx, store.SetQuestionStatusParams{Status: store.QuestionsStatusPublished, ID: prev.ID}); err != nil {
					return err
				}
				r.QuestionsRestored++
			} else {
				r.QuestionsUnchanged++
			}
			continue
		case exists:
			if prev.Status == store.QuestionsStatusPublished {
				if err := q.SetQuestionStatus(ctx, store.SetQuestionStatusParams{Status: store.QuestionsStatusRetired, ID: prev.ID}); err != nil {
					return err
				}
			}
			r.QuestionsNewVersion++
		default:
			r.QuestionsAdded++
		}
		version := uint32(1)
		if exists {
			version = prev.Version + 1
		}
		if err := q.InsertQuestion(ctx, store.InsertQuestionParams{
			StableKey: it.Key, Version: version, QuestionTypeID: typeID,
			Payload: payload, Answer: answer, ExplanationJa: it.ExplanationJa,
			Difficulty: uint8(it.Difficulty), VocabKeys: vocabJSON, ContentHash: hash,
		}); err != nil {
			return fmt.Errorf("%s: %w", it.Key, err)
		}
	}

	for key, row := range latest {
		if !inFiles[key] && row.Status == store.QuestionsStatusPublished {
			if err := q.SetQuestionStatus(ctx, store.SetQuestionStatusParams{Status: store.QuestionsStatusRetired, ID: row.ID}); err != nil {
				return err
			}
			r.QuestionsRetired++
		}
	}
	return nil
}

func importVocab(ctx context.Context, q *store.Queries, gradeID uint8, items []Located[VocabItem], prefix string, r *Report) error {
	rows, err := q.ListVocabByGrade(ctx, gradeID)
	if err != nil {
		return err
	}
	existing := map[string]store.ListVocabByGradeRow{}
	for _, row := range rows {
		if strings.HasPrefix(row.StableKey, prefix) {
			existing[row.StableKey] = row
		}
	}
	inFiles := map[string]bool{}
	for _, l := range items {
		it := l.Item
		if !strings.HasPrefix(it.Key, prefix) {
			continue
		}
		inFiles[it.Key] = true
		payload, _ := json.Marshal(it)
		hash := hashOf(it.Headword, it.POS, payload)
		prev, ok := existing[it.Key]
		switch {
		case !ok:
			if err := q.InsertVocab(ctx, store.InsertVocabParams{
				StableKey: it.Key, GradeID: gradeID, Headword: it.Headword,
				Pos: store.VocabItemsPos(it.POS), Payload: payload, ContentHash: hash,
			}); err != nil {
				return fmt.Errorf("%s: %w", it.Key, err)
			}
			r.VocabAdded++
		case !bytes.Equal(prev.ContentHash, hash) || prev.Status != store.VocabItemsStatusPublished:
			if err := q.UpdateVocab(ctx, store.UpdateVocabParams{
				Headword: it.Headword, Pos: store.VocabItemsPos(it.POS), Payload: payload, ContentHash: hash, ID: prev.ID,
			}); err != nil {
				return fmt.Errorf("%s: %w", it.Key, err)
			}
			r.VocabUpdated++
		default:
			r.VocabUnchanged++
		}
	}
	for key, row := range existing {
		if !inFiles[key] && row.Status == store.VocabItemsStatusPublished {
			if err := q.SetVocabStatus(ctx, store.SetVocabStatusParams{Status: store.VocabItemsStatusRetired, ID: row.ID}); err != nil {
				return err
			}
			r.VocabRetired++
		}
	}
	return nil
}

// hashOf fingerprints content so unchanged items are skipped on re-import.
func hashOf(parts ...any) []byte {
	h := sha256.New()
	for _, p := range parts {
		b, _ := json.Marshal(p)
		h.Write(b)
		h.Write([]byte{0})
	}
	return h.Sum(nil)
}
