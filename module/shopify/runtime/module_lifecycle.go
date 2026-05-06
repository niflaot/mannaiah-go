package runtime

import (
	"context"
	"fmt"
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
	if m.orderConsumer != nil && m.registrar != nil && !m.consumerRegistered {
		if err := m.orderConsumer.Register(m.registrar); err != nil {
			return fmt.Errorf("register shopify order consumer: %w", err)
		}
		m.consumerRegistered = true
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
