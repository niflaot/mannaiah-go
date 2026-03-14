package e2e_test

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	corecron "mannaiah/module/core/cron"
	"mannaiah/module/syncrecord"
	syncrecorddomain "mannaiah/module/syncrecord/domain"
	syncrecordport "mannaiah/module/syncrecord/port"
	"mannaiah/module/woocommerce"
	wooevent "mannaiah/module/woocommerce/adapter/event"
	woocommerceport "mannaiah/module/woocommerce/port"
)

// TestSyncRecordWooCommerceE2E verifies WooCommerce sync run recording behavior.
func TestSyncRecordWooCommerceE2E(t *testing.T) {
	harness := newContactsE2EHarness(t)
	defer harness.Close(t)

	harness.tracer.Step("start woocommerce mock server")
	wooServer := newWooOrdersServer(t)
	defer wooServer.Close()

	harness.tracer.Step("initialize sync record module")
	syncModule, err := syncrecord.New(syncrecord.Config{Enabled: true, CleanupEnabled: false}, harness.db)
	if err != nil {
		t.Fatalf("syncrecord.New() error = %v", err)
	}
	syncModule.SetAuthorizer(harness.authModule)
	harness.server.RegisterRoutes(syncModule.RegisterRoutes)

	harness.tracer.Step("initialize woocommerce scheduler")
	scheduler, err := corecron.NewScheduler(corecron.Config{Location: "UTC"}, harness.tracer.logger)
	if err != nil {
		t.Fatalf("corecron.NewScheduler() error = %v", err)
	}

	harness.tracer.Step("initialize woocommerce event publisher")
	publisher, err := wooevent.NewPublisher(harness.messaging.Publisher())
	if err != nil {
		t.Fatalf("wooevent.NewPublisher() error = %v", err)
	}

	harness.tracer.Step("initialize woocommerce module")
	module, err := woocommerce.New(woocommerce.Config{
		URL:                 wooServer.URL,
		ConsumerKey:         "key",
		ConsumerSecret:      "secret",
		SyncContacts:        true,
		SyncContactsCron:    "0 0 * * *",
		SyncPageSize:        2,
		SyncWorkers:         4,
		RequestTimeoutMS:    2000,
		ValidationTimeoutMS: 1000,
		VerifySSL:           true,
	}, harness.contactsModule.Service(), harness.ordersModule.Service(), scheduler, harness.tracer.logger, publisher)
	if err != nil {
		t.Fatalf("woocommerce.New() error = %v", err)
	}
	module.SetSyncRecorder(e2eWooSyncRecorderAdapter{recorder: syncModule.Recorder()})
	module.SetAuthorizer(harness.authModule)
	harness.server.RegisterRoutes(module.RegisterRoutes)
	if err := module.Start(context.Background()); err != nil {
		t.Fatalf("module.Start() error = %v", err)
	}
	defer func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = module.Stop(stopCtx)
	}()

	manageToken := harness.SignToken(t, "contacts:manage marketing:manage")

	harness.tracer.Step("trigger woocommerce contacts sync")
	status, payload := harness.DoJSONRequest(t, http.MethodPost, "/woo/sync/contacts", manageToken, nil)
	if status != http.StatusOK {
		t.Fatalf("status = %d, want %d", status, http.StatusOK)
	}
	if payload["processed"] != float64(2) {
		t.Fatalf("payload.processed = %v, want %v", payload["processed"], float64(2))
	}

	harness.tracer.Step("query sync record runs")
	status, payload = harness.DoJSONRequest(t, http.MethodGet, "/syncrecord/runs?kind=woocommerce.contacts", manageToken, nil)
	if status != http.StatusOK {
		t.Fatalf("status = %d, want %d", status, http.StatusOK)
	}

	dataPayload := payload["Data"]
	if dataPayload == nil {
		dataPayload = payload["data"]
	}
	runs, ok := dataPayload.([]any)
	if !ok || len(runs) == 0 {
		t.Fatalf("expected sync run list payload, got %v", dataPayload)
	}

	run, ok := runs[0].(map[string]any)
	if !ok {
		t.Fatalf("expected first run object")
	}
	if run["kind"] != "woocommerce.contacts" {
		t.Fatalf("run.kind = %v, want %q", run["kind"], "woocommerce.contacts")
	}
	if run["status"] != "completed" {
		t.Fatalf("run.status = %v, want %q", run["status"], "completed")
	}
	if run["processed"] != float64(2) {
		t.Fatalf("run.processed = %v, want %v", run["processed"], float64(2))
	}

	harness.tracer.Step("assert e2e trace logs")
	harness.tracer.AssertStepCount(8)
}

// TestSyncRecordRetentionE2E verifies sync run retention cleanup behavior.
func TestSyncRecordRetentionE2E(t *testing.T) {
	harness := newContactsE2EHarness(t)
	defer harness.Close(t)

	harness.tracer.Step("initialize sync record module")
	syncModule, err := syncrecord.New(syncrecord.Config{Enabled: true, CleanupEnabled: false}, harness.db)
	if err != nil {
		t.Fatalf("syncrecord.New() error = %v", err)
	}

	recorder := syncModule.Recorder()

	harness.tracer.Step("create old sync run")
	oldStartedAt := time.Now().UTC().Add(-120 * 24 * time.Hour)
	oldRunID, err := recorder.StartRun(context.Background(), syncrecordport.StartRunInput{
		Kind:      syncrecorddomain.KindMembershipMigration,
		Trigger:   syncrecorddomain.TriggerMigration,
		StartedAt: &oldStartedAt,
	})
	if err != nil {
		t.Fatalf("recorder.StartRun(old) error = %v", err)
	}
	oldEndedAt := oldStartedAt.Add(2 * time.Minute)
	if err := recorder.CompleteRun(context.Background(), syncrecordport.FinishRunInput{
		RunID:     oldRunID,
		EndedAt:   &oldEndedAt,
		Processed: 10,
		Succeeded: 10,
		Failed:    0,
		Skipped:   0,
	}); err != nil {
		t.Fatalf("recorder.CompleteRun(old) error = %v", err)
	}

	harness.tracer.Step("create recent sync run")
	recentStartedAt := time.Now().UTC().Add(-2 * time.Hour)
	recentRunID, err := recorder.StartRun(context.Background(), syncrecordport.StartRunInput{
		Kind:      syncrecorddomain.KindMembershipMigration,
		Trigger:   syncrecorddomain.TriggerManual,
		StartedAt: &recentStartedAt,
	})
	if err != nil {
		t.Fatalf("recorder.StartRun(recent) error = %v", err)
	}
	recentEndedAt := recentStartedAt.Add(3 * time.Minute)
	if err := recorder.CompleteRun(context.Background(), syncrecordport.FinishRunInput{
		RunID:     recentRunID,
		EndedAt:   &recentEndedAt,
		Processed: 2,
		Succeeded: 2,
		Failed:    0,
		Skipped:   0,
	}); err != nil {
		t.Fatalf("recorder.CompleteRun(recent) error = %v", err)
	}

	harness.tracer.Step("cleanup expired runs")
	deleted, err := syncModule.Service().CleanupBefore(context.Background(), time.Now().UTC().Add(-90*24*time.Hour))
	if err != nil {
		t.Fatalf("CleanupBefore() error = %v", err)
	}
	if deleted < 1 {
		t.Fatalf("deleted = %d, want at least 1", deleted)
	}

	harness.tracer.Step("assert old run was removed")
	_, err = syncModule.Service().GetRun(context.Background(), oldRunID)
	if err == nil {
		t.Fatalf("expected old run to be deleted")
	}
	if !errors.Is(err, syncrecorddomain.ErrRunNotFound) && !strings.Contains(strings.ToLower(err.Error()), "not found") {
		t.Fatalf("unexpected old run lookup error = %v", err)
	}

	harness.tracer.Step("assert recent run is preserved")
	run, err := syncModule.Service().GetRun(context.Background(), recentRunID)
	if err != nil {
		t.Fatalf("GetRun(recent) error = %v", err)
	}
	if run == nil || run.ID != recentRunID {
		t.Fatalf("unexpected recent run payload = %v", run)
	}

	harness.tracer.Step("assert e2e trace logs")
	harness.tracer.AssertStepCount(7)
}

// e2eWooSyncRecorderAdapter adapts sync recorder behavior for WooCommerce module tests.
type e2eWooSyncRecorderAdapter struct {
	// recorder defines sync recorder dependencies.
	recorder syncrecordport.Recorder
}

// StartRun starts one synchronization run and returns a run identifier.
func (a e2eWooSyncRecorderAdapter) StartRun(ctx context.Context, kind string, trigger string) (string, error) {
	if a.recorder == nil {
		return "", nil
	}

	return a.recorder.StartRun(ctx, syncrecordport.StartRunInput{
		Kind:    syncrecorddomain.SyncKind(kind),
		Trigger: syncrecorddomain.SyncTrigger(trigger),
	})
}

// CompleteRun marks one synchronization run as completed.
func (a e2eWooSyncRecorderAdapter) CompleteRun(ctx context.Context, runID string, processed int, succeeded int, failed int, skipped int) error {
	if a.recorder == nil {
		return nil
	}

	return a.recorder.CompleteRun(ctx, syncrecordport.FinishRunInput{
		RunID:     runID,
		Processed: processed,
		Succeeded: succeeded,
		Failed:    failed,
		Skipped:   skipped,
	})
}

// FailRun marks one synchronization run as failed.
func (a e2eWooSyncRecorderAdapter) FailRun(ctx context.Context, runID string, processed int, succeeded int, failed int, skipped int, syncErrors []woocommerceport.SyncError) error {
	if a.recorder == nil {
		return nil
	}

	errorsPayload := make([]syncrecorddomain.SyncRunError, 0, len(syncErrors))
	for _, syncErr := range syncErrors {
		errorsPayload = append(errorsPayload, syncrecorddomain.SyncRunError{
			ErrorType: strings.TrimSpace(syncErr.Type),
			ErrorCode: strings.TrimSpace(syncErr.Code),
			Message:   strings.TrimSpace(syncErr.Message),
		})
	}

	return a.recorder.FailRun(ctx, syncrecordport.FinishRunInput{
		RunID:     runID,
		Processed: processed,
		Succeeded: succeeded,
		Failed:    failed,
		Skipped:   skipped,
		Errors:    errorsPayload,
	})
}
