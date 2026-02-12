package e2e_test

import (
	"context"
	"errors"
	"testing"
	"time"
)

// Close releases harness resources.
func (h *contactsE2EHarness) Close(t *testing.T) {
	t.Helper()

	h.tracer.Step("shutdown messaging context")
	h.messagingCancel()

	select {
	case err := <-h.messagingErrs:
		if err != nil && !errors.Is(err, context.Canceled) {
			t.Fatalf("messaging.Run() error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("messaging shutdown timeout")
	}

	h.tracer.Step("close messaging platform")
	if err := h.messaging.Close(); err != nil {
		t.Fatalf("messaging.Close() error = %v", err)
	}

	h.tracer.Step("close database handle")
	h.CloseDatabase(t)

	h.tracer.Step("close jwks server")
	h.jwksServer.Close()
}

// CloseDatabase closes the harness database handle and tolerates double-close behavior.
func (h *contactsE2EHarness) CloseDatabase(t *testing.T) {
	t.Helper()

	if h == nil || h.db == nil || h.dbClosed {
		return
	}

	sqlDB, err := h.db.DB()
	if err != nil {
		t.Fatalf("db.DB() error = %v", err)
	}
	if err := sqlDB.Close(); err != nil && !isClosedDBError(err) {
		t.Fatalf("sqlDB.Close() error = %v", err)
	}

	h.dbClosed = true
}
