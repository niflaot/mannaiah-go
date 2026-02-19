package runtime

import (
	"context"
	"fmt"
	"strings"
	"time"

	corecron "mannaiah/module/core/cron"

	"go.uber.org/zap"
)

// ConfigureScheduler configures the cron scheduler for periodic feed status resolution.
func (m *Module) ConfigureScheduler(scheduler corecron.Scheduler) {
	if m == nil {
		return
	}

	m.scheduler = scheduler
}

// Start registers and starts the feed status resolution cron job.
func (m *Module) Start(_ context.Context) error {
	if m == nil {
		return ErrModuleNotInitialized
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.started {
		return nil
	}

	cronSpec := strings.TrimSpace(m.cfg.SyncStatusCron)
	if m.syncStatusService == nil || m.scheduler == nil || cronSpec == "" {
		m.started = true
		return nil
	}

	batchSize := m.cfg.SyncStatusBatchSize
	if batchSize <= 0 {
		batchSize = 50
	}

	entryID, err := m.scheduler.AddFunc(cronSpec, func() {
		tickCtx, cancel := context.WithTimeout(context.Background(), resolveRequestTimeout(m.cfg.RequestTimeoutMS))
		defer cancel()

		result, resolveErr := m.syncStatusService.ResolvePendingFeeds(tickCtx, batchSize)
		if resolveErr != nil {
			m.logger.Warn("falabella cron feed status resolution failed", zap.Error(resolveErr))
			return
		}

		m.logger.Debug("falabella cron feed status resolution completed",
			zap.Int("checked", result.Checked),
			zap.Int("resolved", result.Resolved),
			zap.Int("still_pending", result.StillPending),
			zap.Int("errored", result.Errored),
		)
	})
	if err != nil {
		return fmt.Errorf("register falabella feed status cron: %w", err)
	}

	m.schedulerEntryID = entryID
	m.scheduler.Start()
	m.started = true
	return nil
}

// Stop stops the cron scheduler and removes the feed status resolution job.
func (m *Module) Stop(ctx context.Context) error {
	if m == nil {
		return nil
	}

	m.mutex.Lock()
	if !m.started {
		m.mutex.Unlock()
		return nil
	}

	m.started = false
	entryID := m.schedulerEntryID
	m.schedulerEntryID = 0
	scheduler := m.scheduler
	m.mutex.Unlock()

	if scheduler == nil {
		return nil
	}

	if entryID != 0 {
		scheduler.Remove(entryID)
	}

	stopCtx := ctx
	if stopCtx == nil {
		var cancel context.CancelFunc
		stopCtx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
	}

	return scheduler.Stop(stopCtx)
}
