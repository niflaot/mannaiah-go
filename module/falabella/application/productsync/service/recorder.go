package service

import (
	"context"
	"strings"
	"time"

	syncdomain "mannaiah/module/falabella/domain/sync"

	"go.uber.org/zap"
)

// SyncStatusRecorder defines optional sync status entry recording behavior.
type SyncStatusRecorder interface {
	// RecordEntry persists a new sync status entry.
	RecordEntry(ctx context.Context, entry *syncdomain.SyncEntry) error
}

// SetRecorder configures optional sync status recording dependencies.
func (s *ProductSyncService) SetRecorder(recorder SyncStatusRecorder) {
	if s == nil {
		return
	}

	s.recorder = recorder
}

// SetLogger configures structured logging dependencies.
func (s *ProductSyncService) SetLogger(logger *zap.Logger) {
	if s == nil {
		return
	}

	if logger != nil {
		s.logger = logger
	}
}

// recordSyncEntry persists sync status entries when a recorder is configured.
func (s *ProductSyncService) recordSyncEntry(ctx context.Context, productID, sku, feedID string, actionResp *syncdomain.ActionResponse) {
	if s.recorder == nil || strings.TrimSpace(feedID) == "" {
		return
	}

	action := syncdomain.SyncActionCreate
	if actionResp != nil {
		action = actionResp.SyncAction()
	}

	entry := &syncdomain.SyncEntry{
		ProductID: strings.TrimSpace(productID),
		SKU:       strings.TrimSpace(sku),
		FeedID:    strings.TrimSpace(feedID),
		Action:    action,
		Status:    syncdomain.SyncStatusPending,
		SyncedAt:  time.Now().UTC(),
	}

	if err := s.recorder.RecordEntry(ctx, entry); err != nil {
		s.logger.Warn("falabella sync status recording failed",
			zap.String("feed_id", feedID),
			zap.String("product_id", productID),
			zap.String("sku", sku),
			zap.Error(err),
		)
	}
}

// parseSyncResponse extracts ActionResponse from Falabella sync response payloads.
func parseSyncResponse(data []byte) *syncdomain.ActionResponse {
	if len(data) == 0 {
		return nil
	}

	parsed, err := syncdomain.ParseActionResponse(data)
	if err != nil || parsed == nil {
		return nil
	}

	return parsed
}
