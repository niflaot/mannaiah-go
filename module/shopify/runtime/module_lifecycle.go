package runtime

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/zap"
)

// Start runs Shopify startup logic.
func (m *Module) Start(ctx context.Context) error {
	if m == nil {
		return ErrModuleNotInitialized
	}
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if m.started {
		return nil
	}
	if m.installationResolver != nil {
		if err := m.installationResolver.Refresh(ctx); err != nil {
			return fmt.Errorf("refresh shopify installations: %w", err)
		}
	}
	m.logger.Info(
		"shopify startup",
		zap.Bool("sync_contacts_enabled", m.cfg.SyncContacts),
		zap.Bool("sync_orders_enabled", m.cfg.SyncOrders),
	)
	if m.scheduler == nil || (!m.cfg.SyncContacts && !m.cfg.SyncOrders) {
		m.started = true
		return nil
	}
	if m.cfg.SyncContacts {
		entryID, err := m.scheduler.AddFunc(strings.TrimSpace(m.cfg.SyncContactsCron), func() {
			syncCtx, cancel := context.WithTimeout(context.Background(), resolveSyncTimeout(m.cfg.SyncTimeoutMS))
			defer cancel()
			if _, syncErr := m.contactSyncService.SyncContacts(syncCtx, "cron"); syncErr != nil {
				m.logger.Warn("shopify cron contacts sync failed", zap.Error(syncErr))
			}
		})
		if err != nil {
			return fmt.Errorf("register shopify contacts sync cron: %w", err)
		}
		m.contactsSchedulerEntryID = entryID
	}
	if m.cfg.SyncOrders {
		entryID, err := m.scheduler.AddFunc(strings.TrimSpace(m.cfg.SyncOrdersCron), func() {
			syncCtx, cancel := context.WithTimeout(context.Background(), resolveSyncTimeout(m.cfg.SyncTimeoutMS))
			defer cancel()
			if _, syncErr := m.orderSyncService.SyncOrders(syncCtx, "cron"); syncErr != nil {
				m.logger.Warn("shopify cron orders sync failed", zap.Error(syncErr))
			}
		})
		if err != nil {
			return fmt.Errorf("register shopify orders sync cron: %w", err)
		}
		m.ordersSchedulerEntryID = entryID
	}
	m.scheduler.Start()
	m.started = true
	return nil
}

// Stop stops Shopify runtime lifecycle resources.
func (m *Module) Stop(ctx context.Context) error {
	if m == nil {
		return nil
	}
	m.mutex.Lock()
	contactsEntryID := m.contactsSchedulerEntryID
	ordersEntryID := m.ordersSchedulerEntryID
	scheduler := m.scheduler
	started := m.started
	m.contactsSchedulerEntryID = 0
	m.ordersSchedulerEntryID = 0
	m.started = false
	m.mutex.Unlock()

	if started && scheduler != nil {
		if contactsEntryID != 0 {
			scheduler.Remove(contactsEntryID)
		}
		if ordersEntryID != 0 {
			scheduler.Remove(ordersEntryID)
		}
		if err := scheduler.Stop(resolveContext(ctx)); err != nil {
			return fmt.Errorf("stop shopify scheduler: %w", err)
		}
	}
	if m.processor != nil {
		return m.processor.Stop(ctx)
	}
	return nil
}
