package orders

import (
	"context"

	couponservice "mannaiah/module/coupons/application/coupon/service"
)

// couponUsageSyncServiceMock defines coupon-usage sync behavior for order upserter tests.
type couponUsageSyncServiceMock struct {
	// syncErr defines sync-operation errors.
	syncErr error
	// commands stores synchronized usage commands.
	commands []couponservice.SyncUsageByCodeCommand
}

// SyncUsageByCode records synchronized coupon-usage commands.
func (m *couponUsageSyncServiceMock) SyncUsageByCode(ctx context.Context, cmd couponservice.SyncUsageByCodeCommand) error {
	_ = ctx
	m.commands = append(m.commands, cmd)
	return m.syncErr
}
