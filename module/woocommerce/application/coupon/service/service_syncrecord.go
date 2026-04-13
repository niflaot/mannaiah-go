package service

import (
	"context"
	"errors"
	"strings"

	"go.uber.org/zap"
	"mannaiah/module/woocommerce/port"
)

// startSyncRunRecord starts sync recorder runs and ignores recorder failures.
func (s *CouponSyncService) startSyncRunRecord(ctx context.Context, trigger string) string {
	if s == nil || s.syncRecorder == nil {
		return ""
	}

	runID, err := s.syncRecorder.StartRun(ctx, "woocommerce.coupons", trigger)
	if err != nil {
		s.logger.Warn("woocommerce coupons sync recorder start failed", zap.Error(err))
		return ""
	}

	return strings.TrimSpace(runID)
}

// finishSyncRunRecord completes or fails sync recorder runs and ignores recorder failures.
func (s *CouponSyncService) finishSyncRunRecord(ctx context.Context, runID string, summary *SyncSummary, syncErr error) {
	if s == nil || s.syncRecorder == nil || strings.TrimSpace(runID) == "" || summary == nil {
		return
	}

	succeeded := summary.Created + summary.Updated + summary.Unchanged
	if syncErr == nil {
		if err := s.syncRecorder.CompleteRun(ctx, runID, summary.Processed, succeeded, summary.Failed, summary.Skipped); err != nil {
			s.logger.Warn("woocommerce coupons sync recorder complete failed", zap.Error(err))
		}
		return
	}

	syncErrors := []port.SyncError{{
		Type:    classifyCouponSyncError(syncErr),
		Code:    "coupons_sync_failed",
		Message: syncErr.Error(),
	}}
	if errors.Is(syncErr, context.Canceled) || errors.Is(syncErr, context.DeadlineExceeded) {
		syncErrors[0].Type = "timeout"
		syncErrors[0].Code = "coupons_sync_timeout"
	}
	if err := s.syncRecorder.FailRun(ctx, runID, summary.Processed, succeeded, summary.Failed, summary.Skipped, syncErrors); err != nil {
		s.logger.Warn("woocommerce coupons sync recorder fail failed", zap.Error(err))
	}
}

// classifyCouponSyncError maps sync errors into stable error type values.
func classifyCouponSyncError(err error) string {
	if err == nil {
		return "unknown"
	}
	if errors.Is(err, ErrIntegrationUnavailable) || errors.Is(err, ErrUpsertUnavailable) {
		return "dependency"
	}

	return "sync"
}