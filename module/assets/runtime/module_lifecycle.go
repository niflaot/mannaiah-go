package runtime

import (
	"context"
	"fmt"
	assetsapplication "mannaiah/module/assets/application"
	"mannaiah/module/assets/port"
	corecron "mannaiah/module/core/cron"
	"strings"
	"time"

	"go.uber.org/zap"
)

// ConfigureScheduler configures the cron scheduler for periodic JPG worker execution.
func (m *Module) ConfigureScheduler(scheduler corecron.Scheduler) {
	if m == nil {
		return
	}

	m.scheduler = scheduler
}

// Start registers and starts the JPG worker cron job.
func (m *Module) Start(_ context.Context) error {
	if m == nil {
		return ErrModuleNotInitialized
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.started {
		return nil
	}
	if !m.cfg.JPGWorkerEnabled {
		m.started = true
		return nil
	}
	if m.scheduler == nil {
		return ErrNilSchedulerWhenEnabled
	}

	workerTags := resolveWorkerTags(m.cfg.JPGWorkerTags)
	if len(workerTags) == 0 {
		m.logger.Warn("assets jpg worker disabled because no tags were configured")
		m.started = true
		return nil
	}

	cronSpec := strings.TrimSpace(m.cfg.JPGWorkerCron)
	if cronSpec == "" {
		m.logger.Warn("assets jpg worker disabled because cron spec is empty")
		m.started = true
		return nil
	}

	entryID, err := m.scheduler.AddFunc(cronSpec, func() {
		tickCtx, cancel := context.WithTimeout(context.Background(), resolveWorkerTimeout(m.cfg.JPGWorkerTimeoutMS))
		defer cancel()
		runID := ""
		if m.syncRecorder != nil {
			recordRunID, recordErr := m.syncRecorder.StartRun(tickCtx, "assets.jpg_conversion", "cron")
			if recordErr != nil {
				m.logger.Warn("assets jpg worker sync recorder start failed", zap.Error(recordErr))
			}
			runID = recordRunID
		}

		result, runErr := m.service.RunJPGWorker(tickCtx, assetsapplication.JPGWorkerCommand{
			Tags:        workerTags,
			BatchSize:   m.cfg.JPGWorkerBatchSize,
			JPEGQuality: m.cfg.JPGWorkerQuality,
		})
		if runErr != nil {
			m.logger.Warn("assets jpg worker failed", zap.Error(runErr))
			if m.syncRecorder != nil && strings.TrimSpace(runID) != "" {
				_ = m.syncRecorder.FailRun(tickCtx, runID, 0, 0, 0, 0, []port.SyncError{{
					Type:    "worker",
					Code:    "assets_jpg_worker_failed",
					Message: runErr.Error(),
				}})
			}
			return
		}
		if m.syncRecorder != nil && strings.TrimSpace(runID) != "" {
			succeeded := result.Converted + result.Skipped
			_ = m.syncRecorder.CompleteRun(tickCtx, runID, result.Scanned, succeeded, result.Failed, result.Skipped)
		}

		m.logger.Debug("assets jpg worker completed",
			zap.Int("scanned", result.Scanned),
			zap.Int("converted", result.Converted),
			zap.Int("skipped", result.Skipped),
			zap.Int("failed", result.Failed),
		)
	})
	if err != nil {
		return fmt.Errorf("register assets jpg worker cron: %w", err)
	}

	m.schedulerEntryID = entryID
	m.scheduler.Start()
	m.started = true
	return nil
}

// Stop stops the cron scheduler and removes the JPG worker job.
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
