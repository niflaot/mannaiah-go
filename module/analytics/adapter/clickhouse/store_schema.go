package clickhouse

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"sort"
	"strings"
)

//go:embed migrations/*.up.sql
var embeddedMigrations embed.FS

// EnsureSchema applies analytical schema dependencies.
func (s *StoreAdapter) EnsureSchema(ctx context.Context) error {
	if s == nil || s.client == nil || s.client.db == nil {
		return nil
	}

	entries, err := embeddedMigrations.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("read clickhouse migrations: %w", err)
	}
	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		name := strings.TrimSpace(entry.Name())
		if entry.IsDir() || !strings.HasSuffix(name, ".up.sql") {
			continue
		}
		files = append(files, name)
	}
	sort.Strings(files)

	return withTx(ctx, s.client.db, func(tx *sql.Tx) error {
		for _, file := range files {
			statement, err := embeddedMigrations.ReadFile("migrations/" + file)
			if err != nil {
				return fmt.Errorf("read clickhouse migration %q: %w", file, err)
			}
			for _, query := range splitMigrationStatements(string(statement)) {
				if _, err := tx.ExecContext(ctx, query); err != nil {
					return fmt.Errorf("apply clickhouse migration %q: %w", file, err)
				}
			}
		}

		return nil
	})
}

// splitMigrationStatements splits one migration file into executable statements.
func splitMigrationStatements(content string) []string {
	parts := strings.Split(content, ";")
	statements := make([]string, 0, len(parts))
	for _, part := range parts {
		statement := strings.TrimSpace(part)
		if statement == "" {
			continue
		}
		statements = append(statements, statement)
	}

	return statements
}

func withTx(ctx context.Context, db *sql.DB, fn func(tx *sql.Tx) error) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin clickhouse transaction: %w", err)
	}

	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit clickhouse transaction: %w", err)
	}

	return nil
}
