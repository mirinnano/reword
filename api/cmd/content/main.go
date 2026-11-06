// Command content validates, lints, audits and imports learning content.
//
//	go run ./cmd/content validate ../content
//	go run ./cmd/content lint ../content
//	go run ./cmd/content blind -out <dir> [-seed N] [-chunk 50] ../content
//	go run ./cmd/content review -out <dir> [-chunk 50] ../content
//	go run ./cmd/content hashes ../content
//	go run ./cmd/content import [-dry-run] ../content
//
// validate and import refuse grammar units without a matching audit record;
// lint, blind, review and hashes work on drafts.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mirinnano/reword/api/internal/content"
	"github.com/mirinnano/reword/api/internal/platform/db"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) < 2 {
		return fmt.Errorf("usage: content validate|lint|blind|review|hashes|import [flags] <dir>")
	}
	cmd := os.Args[1]
	fs := flag.NewFlagSet(cmd, flag.ExitOnError)
	dryRun := fs.Bool("dry-run", false, "import: show what would change without writing")
	out := fs.String("out", "", "blind: directory for the solver sets and the answer key")
	seed := fs.Uint64("seed", 1, "blind: shuffle seed (same seed, same sets)")
	chunk := fs.Int("chunk", 50, "blind: questions per solver set")
	only := fs.String("only", "", "blind, review: keep questions whose key has this prefix")
	_ = fs.Parse(os.Args[2:])
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: content %s [flags] <dir>", cmd)
	}

	set, err := content.Load(fs.Arg(0))
	if err != nil {
		return fmt.Errorf("load failed:\n%w", err)
	}
	if err := content.Validate(set); err != nil {
		return fmt.Errorf("validation failed:\n%w", err)
	}
	if cmd == "validate" || cmd == "import" {
		if err := content.CheckAudits(set); err != nil {
			return fmt.Errorf("grammar audit failed (docs/content-specs/grammar.md):\n%w", err)
		}
	}
	nq, np, nv := 0, 0, 0
	for _, items := range set.Questions {
		nq += len(items)
	}
	for _, items := range set.Passages {
		np += len(items)
		for _, l := range items {
			nq += len(l.Item.Questions)
		}
	}
	for _, items := range set.Vocab {
		nv += len(items)
	}
	fmt.Printf("valid: %d questions (%d passages), %d vocab\n", nq, np, nv)

	switch cmd {
	case "validate":
		return nil
	case "hashes":
		for _, l := range set.Grammar {
			fmt.Printf("%s %s\n", l.Item.Key, content.UnitHash(l.Item))
		}
		return nil
	case "lint":
		findings := content.Lint(set)
		for _, f := range findings {
			fmt.Println(f)
		}
		fmt.Printf("lint: %d findings\n", len(findings))
		return nil
	case "blind":
		if *out == "" || *chunk < 1 {
			return fmt.Errorf("blind needs -out and a positive -chunk")
		}
		return writeBlind(set, *out, *seed, *chunk, *only)
	case "review":
		if *out == "" || *chunk < 1 {
			return fmt.Errorf("review needs -out and a positive -chunk")
		}
		return writeReview(set, *out, *chunk, *only)
	case "import":
		dsn := os.Getenv("DATABASE_DSN")
		if dsn == "" {
			return fmt.Errorf("DATABASE_DSN is required")
		}
		ctx := context.Background()
		pool, err := db.Open(ctx, dsn)
		if err != nil {
			return err
		}
		defer pool.Close()
		r, err := content.Import(ctx, pool, set, content.ImportOptions{DryRun: *dryRun})
		if err != nil {
			return err
		}
		prefix := "imported"
		if *dryRun {
			prefix = "dry run (nothing written)"
		}
		fmt.Printf("%s: %s\n", prefix, r)
		return nil
	default:
		return fmt.Errorf("unknown command %q", cmd)
	}
}

// writeBlind writes set-NN.json files for solvers and key.json for scoring.
// Keep key.json away from solvers.
func writeBlind(set *content.Set, dir string, seed uint64, chunk int, only string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	allItems, allKeys := content.Blind(set, seed)
	var items []content.BlindItem
	var keys []content.BlindKey
	for i, k := range allKeys {
		if strings.HasPrefix(k.Key, only) {
			items, keys = append(items, allItems[i]), append(keys, k)
		}
	}
	write := func(name string, v any) error {
		b, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(dir, name), append(b, '\n'), 0o644)
	}
	n := 0
	for i := 0; i < len(items); i += chunk {
		n++
		if err := write(fmt.Sprintf("set-%02d.json", n), items[i:min(i+chunk, len(items))]); err != nil {
			return err
		}
	}
	if err := write("key.json", keys); err != nil {
		return err
	}
	fmt.Printf("blind: %d sets of up to %d questions, key in %s\n", n, chunk, filepath.Join(dir, "key.json"))
	return nil
}

// writeReview writes review-NN.json: each question with its answer,
// explanation and linked vocab, for reviewers who check content with the key.
func writeReview(set *content.Set, dir string, chunk int, only string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	var items []content.ReviewItem
	for _, it := range content.ReviewItems(set) {
		if strings.HasPrefix(it.Key, only) {
			items = append(items, it)
		}
	}
	// The explanation of a rule is what a grammar reviewer must read first.
	var units []content.GrammarUnit
	for _, l := range set.Grammar {
		if strings.HasPrefix(l.Item.Key, only) {
			units = append(units, l.Item)
		}
	}
	if len(units) > 0 {
		b, err := json.MarshalIndent(content.ReviewUnits(units), "", "  ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(dir, "units.json"), append(b, '\n'), 0o644); err != nil {
			return err
		}
	}
	n := 0
	for i := 0; i < len(items); i += chunk {
		n++
		b, err := json.MarshalIndent(items[i:min(i+chunk, len(items))], "", "  ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("review-%02d.json", n)), append(b, '\n'), 0o644); err != nil {
			return err
		}
	}
	fmt.Printf("review: %d sets of up to %d questions in %s\n", n, chunk, dir)
	return nil
}
