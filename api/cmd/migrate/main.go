// Command migrate applies schema migrations from db/migrations.
//
//	go run ./cmd/migrate up
//	go run ./cmd/migrate down 1
package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/go-sql-driver/mysql"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("usage: migrate up | down N | version")
	}
	dsn := os.Getenv("DATABASE_DSN")
	if dsn == "" {
		log.Fatal("DATABASE_DSN is required")
	}
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		log.Fatalf("parse dsn: %v", err)
	}
	// Migration files may contain several statements.
	cfg.MultiStatements = true

	dir := getenv("MIGRATIONS_DIR", "db/migrations")
	m, err := migrate.New("file://"+dir, "mysql://"+cfg.FormatDSN())
	if err != nil {
		log.Fatalf("init: %v", err)
	}
	defer m.Close()

	switch os.Args[1] {
	case "up":
		err = m.Up()
	case "down":
		n := 1
		if len(os.Args) > 2 {
			if n, err = strconv.Atoi(os.Args[2]); err != nil {
				log.Fatalf("down: %v", err)
			}
		}
		err = m.Steps(-n)
	case "version":
		v, dirty, verr := m.Version()
		if verr != nil && !errors.Is(verr, migrate.ErrNilVersion) {
			log.Fatal(verr)
		}
		fmt.Printf("version=%d dirty=%v\n", v, dirty)
		return
	default:
		log.Fatalf("unknown command %q", os.Args[1])
	}
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		log.Fatalf("%s: %v", os.Args[1], err)
	}
	log.Printf("%s: ok", os.Args[1])
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
