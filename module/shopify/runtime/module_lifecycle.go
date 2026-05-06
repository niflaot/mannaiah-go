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
		"shopify consumers startup",
		zap.Bool("sync_contacts_enabled", m.cfg.SyncContacts),
		zap.Bool("sync_orders_enabled", m.cfg.SyncOrders),
		zap.Bool("has_registrar", m.registrar != nil),
		zap.Bool("has_contact_consumer", m.contactConsumer != nil),
		zap.Bool("has_order_consumer", m.orderConsumer != nil),
	)
	if m.registrar != nil && !m.consumerRegistered {
		if m.contactConsumer != nil {
			m.logger.Info("registering shopify contact consumer")
			if err := m.contactConsumer.Register(m.registrar); err != nil {
				return fmt.Errorf("register shopify contact consumer: %w", err)
			}
		} else if m.cfg.SyncContacts {
			m.logger.Warn("shopify contacts sync enabled but contact consumer is unavailable")
		}
		if m.orderConsumer != nil {
			m.logger.Info("registering shopify order consumer")
			if err := m.orderConsumer.Register(m.registrar); err != nil {
				return fmt.Errorf("register shopify order consumer: %w", err)
			}
		} else if m.cfg.SyncOrders {
			m.logger.Warn("shopify orders sync enabled but order consumer is unavailable")
		}
		m.consumerRegistered = true
	} else if m.registrar == nil {
		m.logger.Info("shopify consumers not registered because messaging registrar is unavailable")
	} else if m.consumerRegistered {
		m.logger.Info("shopify consumers already registered")
	}
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
