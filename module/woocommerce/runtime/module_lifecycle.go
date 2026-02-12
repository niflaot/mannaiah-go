package runtime

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"go.uber.org/zap"
	woocontactservice "mannaiah/module/woocommerce/application/contact/service"
)

// Start runs startup checks and cron scheduler registration.
func (m *Module) Start(ctx context.Context) error {
	if m == nil {
		return ErrModuleNotInitialized
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.started {
		return nil
	}

	m.validateAtStartup(resolveContext(ctx))
	if !m.cfg.SyncContacts {
		m.started = true
		return nil
	}

	entryID, err := m.scheduler.AddFunc(strings.TrimSpace(m.cfg.SyncContactsCron), func() {
		syncCtx, cancel := context.WithTimeout(context.Background(), resolveValidationTimeout(m.cfg.ValidationTimeoutMS))
		defer cancel()

		if _, syncErr := m.contactsSyncService.SyncContacts(syncCtx, "cron"); syncErr != nil {
			m.logger.Warn("woocommerce cron contacts sync failed", zap.Error(syncErr))
		}
	})
	if err != nil {
		return fmt.Errorf("register woocommerce contacts sync cron: %w", err)
	}

	m.schedulerEntryID = entryID
	m.scheduler.Start()
	m.started = true
	return nil
}

// Stop stops cron scheduling and removes registered jobs.
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
	if err := scheduler.Stop(ctx); err != nil {
		return fmt.Errorf("stop woocommerce scheduler: %w", err)
	}

	return nil
}

// validateAtStartup verifies integration availability and logs startup warnings.
func (m *Module) validateAtStartup(ctx context.Context) {
	validationCtx, cancel := context.WithTimeout(ctx, resolveValidationTimeout(m.cfg.ValidationTimeoutMS))
	defer cancel()

	if err := m.contactsSyncService.ValidateIntegration(validationCtx); err != nil {
		if !errors.Is(err, woocontactservice.ErrSyncDisabled) {
			m.logger.Warn(
				"woocommerce integration unavailable; endpoints remain documented and return 503 until integration recovers",
				zap.Error(err),
			)
		}
	}
}
