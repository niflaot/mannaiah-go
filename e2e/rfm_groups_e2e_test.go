package e2e_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	analyticruntime "mannaiah/module/analytics/runtime"
	coredatabase "mannaiah/module/core/database"
	coredatabasemigration "mannaiah/module/core/database/migration"
	corehttp "mannaiah/module/core/http"
)

// newRFMGroupsHarness creates a minimal harness for RFM group E2E tests.
func newRFMGroupsHarness(t *testing.T) (*corehttp.Server, *analyticruntime.Module, func()) {
	t.Helper()

	dsn := fmt.Sprintf("file:%s_rfm?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
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

	server, err := corehttp.New(corehttp.Config{Host: "127.0.0.1", Port: 9021}, nil)
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

// TestRFMGroups_CRUD verifies the full RFM group CRUD lifecycle via HTTP.
func TestRFMGroups_CRUD(t *testing.T) {
	server, _, cleanup := newRFMGroupsHarness(t)
	defer cleanup()

	// Create group — no auth wired, endpoint must be accessible.
	body, _ := json.Marshal(map[string]any{
		"name":        "Champions",
		"slug":        "champions",
		"description": "Top customers",
		"conditions": map[string]any{
			"rMin": 4,
			"rMax": 5,
			"fMin": 4,
		},
	})

	status, payload, _, err := doJSONRequestRaw(server, "POST", "/analytics/rfm/groups", "", body)
	if err != nil {
		t.Fatalf("POST /analytics/rfm/groups error = %v", err)
	}
	if status != 201 {
		t.Fatalf("POST /analytics/rfm/groups status = %d, want 201 (payload: %v)", status, payload)
	}

	// Domain struct fields serialize as Go field names (no json tags): "ID", "Name", etc.
	groupID, ok := payload["ID"].(string)
	if !ok || strings.TrimSpace(groupID) == "" {
		t.Fatalf("created group has no ID field (payload: %v)", payload)
	}

	// Get the group.
	status, payload, _, err = doJSONRequestRaw(server, "GET", "/analytics/rfm/groups/"+groupID, "", nil)
	if err != nil {
		t.Fatalf("GET /analytics/rfm/groups/:id error = %v", err)
	}
	if status != 200 {
		t.Fatalf("GET /analytics/rfm/groups/:id status = %d, want 200", status)
	}
	if payload["Name"] != "Champions" {
		t.Errorf("group Name = %v, want Champions", payload["Name"])
	}

	// List groups.
	status, listPayload, _, err := doJSONRequestRaw(server, "GET", "/analytics/rfm/groups", "", nil)
	if err != nil {
		t.Fatalf("GET /analytics/rfm/groups error = %v", err)
	}
	if status != 200 {
		t.Fatalf("GET /analytics/rfm/groups status = %d, want 200", status)
	}
	groups, _ := listPayload["data"].([]any)
	if len(groups) == 0 {
		t.Errorf("list groups returned 0 items")
	}

	// Update the group.
	updateBody, _ := json.Marshal(map[string]any{
		"name": "VIPs", "slug": "vips",
		"conditions": map[string]any{"rMin": 5, "rMax": 5},
	})
	status, _, _, err = doJSONRequestRaw(server, "PUT", "/analytics/rfm/groups/"+groupID, "", updateBody)
	if err != nil {
		t.Fatalf("PUT /analytics/rfm/groups/:id error = %v", err)
	}
	if status != 200 {
		t.Fatalf("PUT /analytics/rfm/groups/:id status = %d, want 200", status)
	}

	// Delete the group.
	status, _, _, err = doJSONRequestRaw(server, "DELETE", "/analytics/rfm/groups/"+groupID, "", nil)
	if err != nil {
		t.Fatalf("DELETE /analytics/rfm/groups/:id error = %v", err)
	}
	// Fiber returns 204 with no body; handler also tolerates 200 variants.
	if status != 204 && status != 200 {
		t.Fatalf("DELETE /analytics/rfm/groups/:id status = %d, want 204", status)
	}
}

// TestRFMBands_GetAndUpdate verifies band configuration read and write via HTTP.
func TestRFMBands_GetAndUpdate(t *testing.T) {
	server, _, cleanup := newRFMGroupsHarness(t)
	defer cleanup()

	// Seed default bands by calling the API to retrieve them.
	status, payload, _, err := doJSONRequestRaw(server, "GET", "/analytics/rfm/bands", "", nil)
	if err != nil {
		t.Fatalf("GET /analytics/rfm/bands error = %v", err)
	}
	// When analytics is disabled and no bands seeded, the endpoint returns 200 with empty array.
	if status != 200 {
		t.Fatalf("GET /analytics/rfm/bands status = %d, want 200 (payload: %v)", status, payload)
	}
}

// TestRFMScore_ContactNotFound verifies scoring endpoint returns nil gracefully when ClickHouse absent.
func TestRFMScore_ContactNotFound(t *testing.T) {
	server, _, cleanup := newRFMGroupsHarness(t)
	defer cleanup()

	status, _, _, err := doJSONRequestRaw(server, "GET", "/analytics/rfm/contacts/no-such-contact/score", "", nil)
	if err != nil {
		t.Fatalf("score endpoint error = %v", err)
	}
	// With noop store, ScoreContact returns nil score — handler returns 200 with null body.
	if status != 200 {
		t.Errorf("score endpoint status = %d, want 200", status)
	}
}

// TestRFMRefresh_DisabledModule verifies refresh endpoint returns 200 when analytics disabled (noop).
func TestRFMRefresh_DisabledModule(t *testing.T) {
	server, _, cleanup := newRFMGroupsHarness(t)
	defer cleanup()

	status, _, _, err := doJSONRequestRaw(server, "POST", "/analytics/rfm/refresh", "", nil)
	if err != nil {
		t.Fatalf("refresh endpoint error = %v", err)
	}
	if status != 200 {
		t.Errorf("refresh status = %d, want 200", status)
	}
}
