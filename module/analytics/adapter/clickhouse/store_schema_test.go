package clickhouse

import (
	"errors"
	"testing"
)

// TestSplitMigrationStatements keeps multi-statement migration files executable.
func TestSplitMigrationStatements(t *testing.T) {
	content := `
DROP VIEW IF EXISTS variation_affinity_mv;

DROP TABLE IF EXISTS variation_affinity_mv;
DROP TABLE IF EXISTS campaign_events;
`

	statements := splitMigrationStatements(content)

	if len(statements) != 3 {
		t.Fatalf("expected 3 statements, got %d", len(statements))
	}

	if statements[0] != "DROP VIEW IF EXISTS variation_affinity_mv" {
		t.Fatalf("unexpected first statement: %q", statements[0])
	}

	if statements[2] != "DROP TABLE IF EXISTS campaign_events" {
		t.Fatalf("unexpected last statement: %q", statements[2])
	}
}

// TestShouldIgnoreDropTypeMismatch keeps cleanup migrations tolerant to legacy object kinds.
func TestShouldIgnoreDropTypeMismatch(t *testing.T) {
	t.Run("drop view against table", func(t *testing.T) {
		if !shouldIgnoreDropTypeMismatch("DROP VIEW IF EXISTS variation_affinity_mv", errors.New("code: 80, message: Table mannaiah.variation_affinity_mv is not a View")) {
			t.Fatal("expected view/table mismatch to be ignored")
		}
	})

	t.Run("drop table against view", func(t *testing.T) {
		if !shouldIgnoreDropTypeMismatch("DROP TABLE IF EXISTS variation_affinity_mv", errors.New("code: 80, message: Table mannaiah.variation_affinity_mv is not a table")) {
			t.Fatal("expected table/view mismatch to be ignored")
		}
	})

	t.Run("non drop mismatch", func(t *testing.T) {
		if shouldIgnoreDropTypeMismatch("CREATE TABLE variation_affinity_mv (id UInt64)", errors.New("is not a table")) {
			t.Fatal("expected non-drop statement errors to surface")
		}
	})

	t.Run("unrelated drop error", func(t *testing.T) {
		if shouldIgnoreDropTypeMismatch("DROP TABLE IF EXISTS variation_affinity_mv", errors.New("permission denied")) {
			t.Fatal("expected unrelated drop errors to surface")
		}
	})
}
