// Package sqlite provides SQLite-backed implementations of all repository
// and infrastructure ports defined in internal/core/ports.
package sqlite

import (
	"database/sql"
	"embed"
	"fmt"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Open opens a SQLite database at the given DSN, applies PRAGMAs (WAL,
// foreign_keys, busy_timeout), runs migrations and seed data via embed.FS.
func Open(dsn string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("sqlite.Open: %w", err)
	}

	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA foreign_keys=ON",
		"PRAGMA busy_timeout=5000",
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			db.Close()
			return nil, fmt.Errorf("sqlite.Open pragma %q: %w", p, err)
		}
	}

	// Run migration.
	mig, err := migrationsFS.ReadFile("migrations/001_init.sql")
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("sqlite.Open read migration: %w", err)
	}
	if _, err := db.Exec(string(mig)); err != nil {
		db.Close()
		return nil, fmt.Errorf("sqlite.Open run migration: %w", err)
	}

	// Run seed.
	seed, err := migrationsFS.ReadFile("migrations/seed.sql")
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("sqlite.Open read seed: %w", err)
	}
	if _, err := db.Exec(string(seed)); err != nil {
		db.Close()
		return nil, fmt.Errorf("sqlite.Open run seed: %w", err)
	}

	return db, nil
}
