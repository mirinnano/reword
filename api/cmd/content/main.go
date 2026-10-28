// Command content validates and imports learning content.
//
//	go run ./cmd/content validate ../content
//	go run ./cmd/content import [--dry-run] ../content
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

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
		return fmt.Errorf("usage: content validate|import [--dry-run] <dir>")
	}
	cmd := os.Args[1]
	fs := flag.NewFlagSet(cmd, flag.ExitOnError)
	dryRun := fs.Bool("dry-run", false, "show what import would change without writing")
	_ = fs.Parse(os.Args[2:])
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: content %s <dir>", cmd)
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
