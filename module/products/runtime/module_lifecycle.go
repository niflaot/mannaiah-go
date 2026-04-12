package runtime

import (
	"context"
	"errors"
	"fmt"
	"time"

	corecron "mannaiah/module/core/cron"

	"go.uber.org/zap"
)

var (
	// ErrModuleNotInitialized is returned when lifecycle methods are called on nil receivers.
	ErrModuleNotInitialized = errors.New("products module is not initialized")
)

// ConfigureScheduler configures the cron scheduler for storefront navigation refresh.
func (m *Module) ConfigureScheduler(scheduler corecron.Scheduler) {
	if m == nil {
		return
	}

	m.scheduler = scheduler
}

// Start registers and starts the storefront navigation regeneration cron job.
func (m *Module) Start(_ context.Context) error {
	if m == nil {
		return ErrModuleNotInitialized
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.started {
		return nil
	}
	if m.storefrontService == nil || m.scheduler == nil || !m.cfg.StorefrontNavigationEnabled {
		m.started = true
		return nil
	}

	entryID, err := m.scheduler.AddFunc(m.storefrontRefreshSpec(), func() {
		regenerationCtx, cancel := context.WithTimeout(context.Background(), m.storefrontRegenerationTimeout())
		defer cancel()

		if _, regenerateErr := m.storefrontService.Regenerate(regenerationCtx); regenerateErr != nil {
			m.logger.Warn("products storefront navigation regeneration failed", zap.Error(regenerateErr))
		}
	})
	if err != nil {
		return fmt.Errorf("register storefront navigation cron: %w", err)
	}

	m.storefrontRefreshEntryID = entryID
	m.scheduler.Start()
	m.started = true
	return nil
}

// Stop stops the storefront navigation scheduler and removes its cron entry.
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
	entryID := m.storefrontRefreshEntryID
	m.storefrontRefreshEntryID = 0
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
