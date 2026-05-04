package service_test

import (
	"context"
	"sync"
	"time"

	"mannaiah/module/coupons/domain"
	"mannaiah/module/coupons/port"
)

// mockRepository defines an in-memory coupon repository stub.
type mockRepository struct {
	mu       sync.Mutex
	coupons  map[string]*domain.Coupon
	codes    map[string]string
	wooIDs   map[int]string
	createFn func(ctx context.Context, coupon *domain.Coupon) error
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		coupons: make(map[string]*domain.Coupon),
		codes:   make(map[string]string),
		wooIDs:  make(map[int]string),
	}
}

func (m *mockRepository) Create(ctx context.Context, coupon *domain.Coupon) error {
	if m.createFn != nil {
		return m.createFn(ctx, coupon)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	copy := *coupon
	m.coupons[coupon.ID] = &copy
	m.codes[coupon.Code] = coupon.ID
	if coupon.WooCommerceID != nil {
		m.wooIDs[*coupon.WooCommerceID] = coupon.ID
	}
	return nil
}

func (m *mockRepository) GetByID(_ context.Context, id string) (*domain.Coupon, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.coupons[id]
	if !ok {
		return nil, nil
	}
	copy := *c
	return &copy, nil
}

func (m *mockRepository) GetByCode(_ context.Context, code string) (*domain.Coupon, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id, ok := m.codes[code]
	if !ok {
		return nil, nil
	}
	c, ok := m.coupons[id]
	if !ok {
		return nil, nil
	}
	copy := *c
	return &copy, nil
}

func (m *mockRepository) GetByWooCommerceID(_ context.Context, wooID int) (*domain.Coupon, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id, ok := m.wooIDs[wooID]
	if !ok {
		return nil, nil
	}
	c, ok := m.coupons[id]
	if !ok {
		return nil, nil
	}
	copy := *c
	return &copy, nil
}

func (m *mockRepository) Update(_ context.Context, coupon *domain.Coupon) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	copy := *coupon
	m.coupons[coupon.ID] = &copy
	return nil
}

func (m *mockRepository) Delete(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.coupons[id]
	if ok {
		delete(m.codes, c.Code)
		if c.WooCommerceID != nil {
			delete(m.wooIDs, *c.WooCommerceID)
		}
	}
	delete(m.coupons, id)
	return nil
}

func (m *mockRepository) List(_ context.Context, _ port.ListQuery) ([]domain.Coupon, int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]domain.Coupon, 0, len(m.coupons))
	for _, c := range m.coupons {
		result = append(result, *c)
	}
	return result, int64(len(result)), nil
}

func (m *mockRepository) Search(_ context.Context, _ port.SearchQuery) ([]domain.Coupon, int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]domain.Coupon, 0, len(m.coupons))
	for _, c := range m.coupons {
		result = append(result, *c)
	}
	return result, int64(len(result)), nil
}

func (m *mockRepository) CodeExists(_ context.Context, code string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.codes[code]
	return ok, nil
}

// mockUsageRepository defines an in-memory coupon usage stub.
type mockUsageRepository struct {
	mu     sync.Mutex
	usages []port.UsageRecord
}

func newMockUsageRepository() *mockUsageRepository {
	return &mockUsageRepository{}
}

func (m *mockUsageRepository) RecordUsage(_ context.Context, record port.UsageRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.usages = append(m.usages, record)
	return nil
}

func (m *mockUsageRepository) CountGlobalUsage(_ context.Context, couponID string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var count int64
	for _, u := range m.usages {
		if u.CouponID == couponID {
			count++
		}
	}
	return count, nil
}

func (m *mockUsageRepository) CountUsageByEmail(_ context.Context, couponID string, email string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var count int64
	for _, u := range m.usages {
		if u.CouponID == couponID && u.Email == email {
			count++
		}
	}
	return count, nil
}

func (m *mockUsageRepository) UsageExistsForOrder(_ context.Context, couponID string, orderID string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, u := range m.usages {
		if u.CouponID == couponID && u.OrderID == orderID {
			return true, nil
		}
	}
	return false, nil
}

// mockPublisher defines a test stub for IntegrationEventPublisher.
type mockPublisher struct {
	mu     sync.Mutex
	events []port.IntegrationEvent
}

func (p *mockPublisher) Publish(_ context.Context, ev port.IntegrationEvent) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.events = append(p.events, ev)
	return nil
}

func (p *mockPublisher) published() []port.IntegrationEvent {
	p.mu.Lock()
	defer p.mu.Unlock()
	result := make([]port.IntegrationEvent, len(p.events))
	copy(result, p.events)
	return result
}

// ptr returns a pointer to a value of any type.
func ptr[T any](v T) *T { return &v }

// futureTime returns a time well in the future.
func futureTime() *time.Time {
	t := time.Now().Add(24 * time.Hour * 365)
	return &t
}

// pastTime returns a time well in the past.
func pastTime() *time.Time {
	t := time.Now().Add(-24 * time.Hour)
	return &t
}
