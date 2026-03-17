package e2e_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	analyticruntime "mannaiah/module/analytics/runtime"
	coredatabase "mannaiah/module/core/database"
	coredatabasemigration "mannaiah/module/core/database/migration"
	corehttp "mannaiah/module/core/http"
)

// newAffinityHarness creates a minimal harness for affinity E2E tests.
func newAffinityHarness(t *testing.T) (*corehttp.Server, *analyticruntime.Module, func()) {
	t.Helper()

	dsn := fmt.Sprintf("file:%s_aff?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := coredatabase.Open(coredatabase.Config{Driver: "sqlite", DSN: dsn, MaxOpenConns: 1}, nil)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := coredatabasemigration.Apply(context.Background(), db, coredatabasemigration.Config{
		Enabled: true, Driver: "sqlite", Table: "schema_migrations",
	}, nil); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	analyticsModule, err := analyticruntime.New(analyticruntime.Config{Enabled: false}, db, nil)
	if err != nil {
		t.Fatalf("analytics.New() error = %v", err)
	}

	server, err := corehttp.New(corehttp.Config{Host: "127.0.0.1", Port: 9022}, nil)
	if err != nil {
		t.Fatalf("corehttp.New() error = %v", err)
	}
	server.RegisterRoutes(analyticsModule.RegisterRoutes)

	cleanup := func() {
		_ = analyticsModule.Stop()
		sqlDB, _ := db.DB()
		if sqlDB != nil {
			_ = sqlDB.Close()
		}
	}

	return server, analyticsModule, cleanup
}

// TestAffinity_GetProfile_NoClickHouse verifies that the profile endpoint returns a valid empty
// profile when ClickHouse is not configured (noop store).
func TestAffinity_GetProfile_NoClickHouse(t *testing.T) {
	server, _, cleanup := newAffinityHarness(t)
	defer cleanup()

	status, payload, _, err := doJSONRequestRaw(server, "GET", "/analytics/affinity/contacts/c-1?limit=10&minScore=0", "", nil)
	if err != nil {
		t.Fatalf("GET affinity profile error = %v", err)
	}
	if status != 200 {
		t.Fatalf("GET affinity profile status = %d, want 200 (payload: %v)", status, payload)
	}
	// Domain struct fields serialize without json tags: ContactID → "ContactID".
	if payload["ContactID"] != "c-1" {
		t.Errorf("profile ContactID = %v, want c-1 (payload: %v)", payload["ContactID"], payload)
	}
}

// TestAffinity_GetTagAffinity_NoClickHouse verifies the tag affinity endpoint returns 200 with
// an empty result when ClickHouse is not configured.
func TestAffinity_GetTagAffinity_NoClickHouse(t *testing.T) {
	server, _, cleanup := newAffinityHarness(t)
	defer cleanup()

	status, _, _, err := doJSONRequestRaw(server, "GET", "/analytics/affinity/contacts/c-1/tags?limit=10", "", nil)
	if err != nil {
		t.Fatalf("GET tag affinity error = %v", err)
	}
	if status != 200 {
		t.Errorf("GET tag affinity status = %d, want 200", status)
	}
}

// TestAffinity_GetCategoryAffinity_NoClickHouse verifies the category affinity endpoint.
func TestAffinity_GetCategoryAffinity_NoClickHouse(t *testing.T) {
	server, _, cleanup := newAffinityHarness(t)
	defer cleanup()

	status, _, _, err := doJSONRequestRaw(server, "GET", "/analytics/affinity/contacts/c-1/categories", "", nil)
	if err != nil {
		t.Fatalf("GET category affinity error = %v", err)
	}
	if status != 200 {
		t.Errorf("GET category affinity status = %d, want 200", status)
	}
}

// TestAffinity_GetVariationAffinity_NoClickHouse verifies the variation affinity endpoint.
func TestAffinity_GetVariationAffinity_NoClickHouse(t *testing.T) {
	server, _, cleanup := newAffinityHarness(t)
	defer cleanup()

	status, _, _, err := doJSONRequestRaw(server, "GET", "/analytics/affinity/contacts/c-1/variations?limit=10&minScore=0", "", nil)
	if err != nil {
		t.Fatalf("GET variation affinity error = %v", err)
	}
	if status != 200 {
		t.Errorf("GET variation affinity status = %d, want 200", status)
	}
}

// TestAffinity_RefreshAll_NoClickHouse verifies refresh endpoint returns 200 when analytics disabled.
func TestAffinity_RefreshAll_NoClickHouse(t *testing.T) {
	server, _, cleanup := newAffinityHarness(t)
	defer cleanup()

	status, _, _, err := doJSONRequestRaw(server, "POST", "/analytics/affinity/refresh", "", nil)
	if err != nil {
		t.Fatalf("POST affinity refresh error = %v", err)
	}
	if status != 200 {
		t.Errorf("POST affinity refresh status = %d, want 200", status)
	}
}
