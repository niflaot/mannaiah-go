package application

import (
	"context"
	"fmt"
	"strings"

	"mannaiah/module/analytics/port"
)

// Seed executes best-effort initial data seeding behavior.
func (s *AnalyticsService) Seed(ctx context.Context) (*SeedSummary, error) {
	if !s.enabled {
		return nil, ErrDisabled
	}
	if s.store == nil {
		return nil, ErrBackendUnavailable
	}
	if err := s.store.EnsureSchema(ctx); err != nil {
		return nil, fmt.Errorf("ensure analytics schema: %w", err)
	}

	runID := ""
	if s.syncRecorder != nil {
		startedRunID, runErr := s.syncRecorder.StartRun(ctx, "analytics.seed", "manual")
		if runErr == nil {
			runID = startedRunID
		}
	}

	summary := &SeedSummary{}
	syncErrors := make([]port.SyncError, 0, 8)

	appendSyncError := func(errorType string, errorCode string, message string) {
		trimmedMessage := strings.TrimSpace(message)
		if trimmedMessage == "" {
			return
		}
		syncErrors = append(syncErrors, port.SyncError{
			Type:    strings.TrimSpace(errorType),
			Code:    strings.TrimSpace(errorCode),
			Message: trimmedMessage,
		})
	}

	finalizeSyncRecord := func(failed bool) {
		if strings.TrimSpace(runID) == "" || s.syncRecorder == nil {
			return
		}
		processed := int(summary.Contacts + summary.Orders + summary.OrderItems + summary.MembershipEvents)
		if failed {
			_ = s.syncRecorder.FailRun(ctx, runID, processed, processed, len(syncErrors), 0, syncErrors)
			return
		}
		_ = s.syncRecorder.CompleteRun(ctx, runID, processed, processed, 0, 0)
	}

	if err := s.seedContacts(ctx, summary); err != nil {
		appendSyncError("seed", "contacts_failed", err.Error())
		finalizeSyncRecord(true)
		return nil, err
	}
	if err := s.seedOrders(ctx, summary); err != nil {
		appendSyncError("seed", "orders_failed", err.Error())
		finalizeSyncRecord(true)
		return nil, err
	}
	if err := s.seedMembershipEvents(ctx, summary); err != nil {
		appendSyncError("seed", "membership_failed", err.Error())
		finalizeSyncRecord(true)
		return nil, err
	}
	if s.taxonomyStore != nil {
		if err := s.seedProductTaxonomy(ctx, s.taxonomyStore); err != nil {
			appendSyncError("seed", "product_taxonomy_failed", err.Error())
		}
		if err := s.seedVariationTaxonomy(ctx, s.taxonomyStore); err != nil {
			appendSyncError("seed", "variation_taxonomy_failed", err.Error())
		}
	}

	finalizeSyncRecord(false)
	return summary, nil
}
