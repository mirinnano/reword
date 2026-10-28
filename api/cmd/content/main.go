// Command content validates, lints, audits and imports learning content.
//
//	go run ./cmd/content validate ../content
//	go run ./cmd/content lint ../content
//	go run ./cmd/content blind -out <dir> [-seed N] [-chunk 50] ../content
//	go run ./cmd/content review -out <dir> [-chunk 50] ../content
//	go run ./cmd/content import [-dry-run] ../content
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

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
		return fmt.Errorf("usage: content validate|lint|blind|review|import [flags] <dir>")
	}
	cmd := os.Args[1]
	fs := flag.NewFlagSet(cmd, flag.ExitOnError)
	dryRun := fs.Bool("dry-run", false, "import: show what would change without writing")
	out := fs.String("out", "", "blind: directory for the solver sets and the answer key")
	seed := fs.Uint64("seed", 1, "blind: shuffle seed (same seed, same sets)")
	chunk := fs.Int("chunk", 50, "blind: questions per solver set")
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
	nq, nv := 0, 0
	for _, items := range set.Questions {
		nq += len(items)
	}
	for _, items := range set.Vocab {
		nv += len(items)
	}
	fmt.Printf("valid: %d questions, %d vocab\n", nq, nv)

	switch cmd {
	case "validate":
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
		return writeBlind(set, *out, *seed, *chunk)
	case "review":
		if *out == "" || *chunk < 1 {
			return fmt.Errorf("review needs -out and a positive -chunk")
		}
		return writeReview(set, *out, *chunk)
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
func writeBlind(set *content.Set, dir string, seed uint64, chunk int) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	items, keys := content.Blind(set, seed)
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
func writeReview(set *content.Set, dir string, chunk int) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	items := content.ReviewItems(set)
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
