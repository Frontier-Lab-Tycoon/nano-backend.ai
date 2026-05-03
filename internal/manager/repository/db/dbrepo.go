package db

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"sort"

	"github.com/jmoiron/sqlx"
	"github.com/seedspirit/nano-backend.ai/internal/manager/errordef"
	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

// Open opens the SQLite ledger database and applies idempotent migrations.
//
// The modernc.org/sqlite driver is pure Go, which keeps local development and
// CI builds reproducible without requiring a C compiler.
func Open(ctx context.Context, args Args) (*sqlx.DB, error) {
	if args.DBPath == "" {
		return nil, errordef.Errorf(errordef.InvalidInput, "db path is required")
	}

	dbx, err := sqlx.Open("sqlite", args.DBPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite db: %w", err)
	}
	dbx.SetMaxOpenConns(1)

	if _, err := dbx.ExecContext(ctx, "PRAGMA foreign_keys = ON"); err != nil {
		_ = dbx.Close()
		return nil, fmt.Errorf("enable sqlite foreign keys: %w", err)
	}
	if err := dbx.PingContext(ctx); err != nil {
		_ = dbx.Close()
		return nil, fmt.Errorf("ping sqlite db: %w", err)
	}
	if err := Migrate(ctx, dbx); err != nil {
		_ = dbx.Close()
		return nil, err
	}

	return dbx, nil
}

// Migrate applies all embedded SQLite migrations in lexical order.
func Migrate(ctx context.Context, dbx *sqlx.DB) error {
	files, err := fs.Glob(migrationFS, "migrations/*.sql")
	if err != nil {
		return fmt.Errorf("list migrations: %w", err)
	}
	sort.Strings(files)

	for _, file := range files {
		sql, err := migrationFS.ReadFile(file)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", file, err)
		}
		if _, err := dbx.ExecContext(ctx, string(sql)); err != nil {
			return fmt.Errorf("apply migration %s: %w", file, err)
		}
	}

	return nil
}
