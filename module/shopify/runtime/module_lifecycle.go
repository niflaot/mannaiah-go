package runtime

import (
	"context"
	"fmt"

	"go.uber.org/zap"
)

// Start runs Shopify startup logic.
func (m *Module) Start(ctx context.Context) error {
	if m == nil {
		return ErrModuleNotInitialized
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
	_ = ctx
	return nil
}

// Stop stops Shopify runtime lifecycle resources.
func (m *Module) Stop(ctx context.Context) error {
	if m == nil {
		return nil
	}
	if m.processor != nil {
		return m.processor.Stop(ctx)
	}
	return nil
}
