package orders

import (
	"context"
	"strings"

	couponservice "mannaiah/module/coupons/application/coupon/service"
)

// couponUsageSyncServiceMock defines coupon-usage sync behavior for order upserter tests.
type couponUsageSyncServiceMock struct {
	// syncErr defines sync-operation errors.
	syncErr error
	// resolveErr defines coupon-reference lookup errors.
	resolveErr error
	// commands stores synchronized usage commands.
	commands []couponservice.SyncUsageByCodeCommand
	// references stores local coupon references keyed by uppercased code.
	references map[string]CouponReference
}

// SyncUsageByCode records synchronized coupon-usage commands.
func (m *couponUsageSyncServiceMock) SyncUsageByCode(ctx context.Context, cmd couponservice.SyncUsageByCodeCommand) error {
	_ = ctx
	m.commands = append(m.commands, cmd)
	return m.syncErr
}

// ResolveCouponByCode resolves local coupon references by coupon code.
func (m *couponUsageSyncServiceMock) ResolveCouponByCode(ctx context.Context, code string) (*CouponReference, error) {
	_ = ctx
	if m.resolveErr != nil {
		return nil, m.resolveErr
	}
	if m.references == nil {
		return nil, nil
	}

	reference, ok := m.references[strings.ToUpper(strings.TrimSpace(code))]
	if !ok {
		return nil, nil
	}

	copied := reference
	return &copied, nil
}
