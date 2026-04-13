package runtime

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"go.uber.org/zap"
	woocouponservice "mannaiah/module/woocommerce/application/coupon/service"
	woocontactservice "mannaiah/module/woocommerce/application/contact/service"
	wooorderservice "mannaiah/module/woocommerce/application/order/service"
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
	if !m.cfg.SyncContacts && !m.cfg.SyncOrders && !m.cfg.SyncCoupons {
		m.started = true
		return nil
	}

	if m.cfg.SyncContacts {
		entryID, err := m.scheduler.AddFunc(strings.TrimSpace(m.cfg.SyncContactsCron), func() {
			syncCtx, cancel := context.WithTimeout(context.Background(), resolveSyncTimeout(m.cfg.SyncTimeoutMS))
			defer cancel()

			if _, syncErr := m.contactsSyncService.SyncContacts(syncCtx, "cron"); syncErr != nil {
				m.logger.Warn("woocommerce cron contacts sync failed", zap.Error(syncErr))
			}
		})
		if err != nil {
			return fmt.Errorf("register woocommerce contacts sync cron: %w", err)
		}

		m.contactsSchedulerEntryID = entryID
	}

	if m.cfg.SyncOrders {
		entryID, err := m.scheduler.AddFunc(strings.TrimSpace(m.cfg.SyncOrdersCron), func() {
			syncCtx, cancel := context.WithTimeout(context.Background(), resolveSyncTimeout(m.cfg.SyncTimeoutMS))
			defer cancel()

			if _, syncErr := m.ordersSyncService.SyncOrders(syncCtx, "cron"); syncErr != nil {
				m.logger.Warn("woocommerce cron orders sync failed", zap.Error(syncErr))
			}
		})
		if err != nil {
			return fmt.Errorf("register woocommerce orders sync cron: %w", err)
		}

		m.ordersSchedulerEntryID = entryID
	}

	if m.cfg.SyncCoupons && m.couponsSyncService != nil {
		entryID, err := m.scheduler.AddFunc(strings.TrimSpace(m.cfg.SyncCouponsCron), func() {
			syncCtx, cancel := context.WithTimeout(context.Background(), resolveSyncTimeout(m.cfg.SyncTimeoutMS))
			defer cancel()

			if _, syncErr := m.couponsSyncService.SyncCoupons(syncCtx, "cron"); syncErr != nil {
				m.logger.Warn("woocommerce cron coupons sync failed", zap.Error(syncErr))
			}
		})
		if err != nil {
			return fmt.Errorf("register woocommerce coupons sync cron: %w", err)
		}

		m.couponsSchedulerEntryID = entryID
	}

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
	contactsEntryID := m.contactsSchedulerEntryID
	m.contactsSchedulerEntryID = 0
	ordersEntryID := m.ordersSchedulerEntryID
	m.ordersSchedulerEntryID = 0
	couponsEntryID := m.couponsSchedulerEntryID
	m.couponsSchedulerEntryID = 0
	scheduler := m.scheduler
	m.mutex.Unlock()

	if scheduler == nil {
		return nil
	}
	if contactsEntryID != 0 {
		scheduler.Remove(contactsEntryID)
	}
	if ordersEntryID != 0 {
		scheduler.Remove(ordersEntryID)
	}
	if couponsEntryID != 0 {
		scheduler.Remove(couponsEntryID)
	}
	if err := scheduler.Stop(ctx); err != nil {
		return fmt.Errorf("stop woocommerce scheduler: %w", err)
	}

	return nil
}

// validateAtStartup verifies integration availability and logs startup warnings.
func (m *Module) validateAtStartup(ctx context.Context) {
	validate := func(run func(ctx context.Context) error, disabledErr error) {
		validationCtx, cancel := context.WithTimeout(ctx, resolveValidationTimeout(m.cfg.ValidationTimeoutMS))
		defer cancel()

		if err := run(validationCtx); err != nil {
			if errors.Is(err, disabledErr) {
				return
			}
			m.logger.Warn(
				"woocommerce integration unavailable; endpoints remain documented and return 503 until integration recovers",
				zap.Error(err),
			)
		}
	}

	validate(m.contactsSyncService.ValidateIntegration, woocontactservice.ErrSyncDisabled)
	validate(m.ordersSyncService.ValidateIntegration, wooorderservice.ErrSyncDisabled)
	if m.couponsSyncService != nil {
		validate(func(ctx context.Context) error {
			if !m.cfg.SyncCoupons {
				return woocouponservice.ErrSyncDisabled
			}
			return nil
		}, woocouponservice.ErrSyncDisabled)
	}
}
