package service

import (
	"context"
	"errors"
	"testing"

	syncdomain "mannaiah/module/falabella/domain/sync"

	"go.uber.org/zap"
)

// TestParseSyncResponseValid verifies ActionResponse extraction from valid XML payloads.
func TestParseSyncResponseValid(t *testing.T) {
	resp := parseSyncResponse([]byte(testSyncResponseXML))
	if resp == nil {
		t.Fatalf("parseSyncResponse() returned nil for valid XML")
	}
	if resp.RequestID != "feed-abc-123" {
		t.Fatalf("RequestID = %q, want %q", resp.RequestID, "feed-abc-123")
	}
	if resp.RequestAction != "ProductCreate" {
		t.Fatalf("RequestAction = %q, want %q", resp.RequestAction, "ProductCreate")
	}
}

// TestParseSyncResponseEmpty verifies nil return for empty payloads.
func TestParseSyncResponseEmpty(t *testing.T) {
	if resp := parseSyncResponse(nil); resp != nil {
		t.Fatalf("parseSyncResponse(nil) = %v, want nil", resp)
	}
	if resp := parseSyncResponse([]byte{}); resp != nil {
		t.Fatalf("parseSyncResponse(empty) = %v, want nil", resp)
	}
}

// TestParseSyncResponseInvalidXML verifies nil return for invalid XML payloads.
func TestParseSyncResponseInvalidXML(t *testing.T) {
	if resp := parseSyncResponse([]byte("not-xml")); resp != nil {
		t.Fatalf("parseSyncResponse(invalid) = %v, want nil", resp)
	}
}

// TestParseSyncResponseWarnings verifies warning extraction from XML payloads.
func TestParseSyncResponseWarnings(t *testing.T) {
	resp := parseSyncResponse([]byte(testSyncResponseWithWarningsXML))
	if resp == nil {
		t.Fatalf("parseSyncResponse() returned nil for warnings XML")
	}
	if !resp.HasWarnings() {
		t.Fatalf("HasWarnings() = false, want true")
	}
	if !resp.HasRequiredFieldViolations() {
		t.Fatalf("HasRequiredFieldViolations() = false, want true")
	}
}

// recorderProbe defines recorder behavior for recorder unit tests.
type recorderProbe struct {
	// entries defines recorded entries.
	entries []*syncdomain.SyncEntry
	// err defines configured error.
	err error
}

// RecordEntry stores entries or returns configured errors.
func (p *recorderProbe) RecordEntry(ctx context.Context, entry *syncdomain.SyncEntry) error {
	if p.err != nil {
		return p.err
	}
	p.entries = append(p.entries, entry)
	return nil
}

// TestRecordSyncEntryWithRecorder verifies entry recording on successful sync.
func TestRecordSyncEntryWithRecorder(t *testing.T) {
	probe := &recorderProbe{}
	svc := &ProductSyncService{recorder: probe, logger: zap.NewNop()}

	svc.recordSyncEntry(
		context.Background(),
		"exec-1",
		"prod-1",
		"SKU-001",
		"feed-abc",
		[]string{"v-color", " v-size ", "v-color"},
		syncdomain.SyncStepProduct,
		syncdomain.SyncActionCreate,
	)

	if len(probe.entries) != 1 {
		t.Fatalf("len(entries) = %d, want %d", len(probe.entries), 1)
	}
	if probe.entries[0].ProductID != "prod-1" {
		t.Fatalf("ProductID = %q, want %q", probe.entries[0].ProductID, "prod-1")
	}
	if probe.entries[0].ExecutionID != "exec-1" {
		t.Fatalf("ExecutionID = %q, want %q", probe.entries[0].ExecutionID, "exec-1")
	}
	if probe.entries[0].SKU != "SKU-001" {
		t.Fatalf("SKU = %q, want %q", probe.entries[0].SKU, "SKU-001")
	}
	if probe.entries[0].Step != syncdomain.SyncStepProduct {
		t.Fatalf("Step = %q, want %q", probe.entries[0].Step, syncdomain.SyncStepProduct)
	}
	if probe.entries[0].FeedID != "feed-abc" {
		t.Fatalf("FeedID = %q, want %q", probe.entries[0].FeedID, "feed-abc")
	}
	if probe.entries[0].Action != syncdomain.SyncActionCreate {
		t.Fatalf("Action = %q, want %q", probe.entries[0].Action, syncdomain.SyncActionCreate)
	}
	if probe.entries[0].Status != syncdomain.SyncStatusPending {
		t.Fatalf("Status = %q, want %q", probe.entries[0].Status, syncdomain.SyncStatusPending)
	}
	if len(probe.entries[0].VariationIDs) != 2 {
		t.Fatalf("len(VariationIDs) = %d, want %d", len(probe.entries[0].VariationIDs), 2)
	}
	if probe.entries[0].VariationIDs[0] != "v-color" || probe.entries[0].VariationIDs[1] != "v-size" {
		t.Fatalf("VariationIDs = %#v, want %#v", probe.entries[0].VariationIDs, []string{"v-color", "v-size"})
	}
}

// TestRecordSyncEntryUpdateAction verifies update action detection from response.
func TestRecordSyncEntryUpdateAction(t *testing.T) {
	probe := &recorderProbe{}
	svc := &ProductSyncService{recorder: probe, logger: zap.NewNop()}

	svc.recordSyncEntry(context.Background(), "exec-1", "prod-1", "SKU-001", "feed-upd", nil, syncdomain.SyncStepImage, syncdomain.SyncActionUpdate)

	if len(probe.entries) != 1 {
		t.Fatalf("len(entries) = %d, want %d", len(probe.entries), 1)
	}
	if probe.entries[0].Action != syncdomain.SyncActionUpdate {
		t.Fatalf("Action = %q, want %q", probe.entries[0].Action, syncdomain.SyncActionUpdate)
	}
	if probe.entries[0].Step != syncdomain.SyncStepImage {
		t.Fatalf("Step = %q, want %q", probe.entries[0].Step, syncdomain.SyncStepImage)
	}
}

// TestRecordSyncEntryNilRecorder verifies no-op when recorder is nil.
func TestRecordSyncEntryNilRecorder(t *testing.T) {
	svc := &ProductSyncService{logger: zap.NewNop()}
	svc.recordSyncEntry(context.Background(), "exec-1", "prod-1", "SKU-001", "feed-abc", nil, syncdomain.SyncStepProduct, syncdomain.SyncActionCreate)
}

// TestRecordSyncEntryEmptyFeedID verifies no-op when feed ID is empty.
func TestRecordSyncEntryEmptyFeedID(t *testing.T) {
	probe := &recorderProbe{}
	svc := &ProductSyncService{recorder: probe, logger: zap.NewNop()}

	svc.recordSyncEntry(context.Background(), "exec-1", "prod-1", "SKU-001", "", nil, syncdomain.SyncStepProduct, syncdomain.SyncActionCreate)
	if len(probe.entries) != 0 {
		t.Fatalf("len(entries) = %d, want %d", len(probe.entries), 0)
	}
}

// TestRecordSyncEntryRecorderError verifies recording errors are logged without panicking.
func TestRecordSyncEntryRecorderError(t *testing.T) {
	probe := &recorderProbe{err: errors.New("db down")}
	svc := &ProductSyncService{recorder: probe, logger: zap.NewNop()}

	svc.recordSyncEntry(context.Background(), "exec-1", "prod-1", "SKU-001", "feed-abc", nil, syncdomain.SyncStepProduct, syncdomain.SyncActionCreate)
}

// TestSetRecorderNilService verifies SetRecorder on nil service is safe.
func TestSetRecorderNilService(t *testing.T) {
	var svc *ProductSyncService
	svc.SetRecorder(&recorderProbe{})
}
