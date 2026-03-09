package service

import (
	"context"
	"sort"
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
func (s *ProductSyncService) recordSyncEntry(
	ctx context.Context,
	executionID string,
	productID string,
	sku string,
	feedID string,
	variationIDs []string,
	step syncdomain.SyncStep,
	action syncdomain.SyncAction,
) {
	if s.recorder == nil || strings.TrimSpace(feedID) == "" {
		return
	}
	if !step.IsValid() {
		step = syncdomain.SyncStepProduct
	}
	if !action.IsValid() {
		action = syncdomain.SyncActionCreate
	}

	entry := &syncdomain.SyncEntry{
		ExecutionID:  strings.TrimSpace(executionID),
		ProductID:    strings.TrimSpace(productID),
		SKU:          strings.TrimSpace(sku),
		VariationIDs: normalizeVariationIDs(variationIDs),
		FeedID:       strings.TrimSpace(feedID),
		Step:         step,
		Action:       action,
		Status:       syncdomain.SyncStatusPending,
		SyncedAt:     time.Now().UTC(),
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

// normalizeVariationIDs resolves sorted, deduplicated, trimmed variation identifier values.
func normalizeVariationIDs(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	seen := map[string]struct{}{}
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		normalized = append(normalized, trimmed)
	}

	if len(normalized) == 0 {
		return nil
	}

	sort.Strings(normalized)
	return normalized
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
