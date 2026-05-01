package clickhouse

import "testing"

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
