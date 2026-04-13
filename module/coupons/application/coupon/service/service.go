// Package service defines coupon management use-case behavior.
package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	couponevent "mannaiah/module/coupons/application/coupon/event"
	"mannaiah/module/coupons/domain"
	"mannaiah/module/coupons/port"
)

var (
	// ErrNilRepository is returned when required repository dependencies are nil.
	ErrNilRepository = errors.New("coupon repository must not be nil")
	// ErrNilUsageRepository is returned when required usage repository dependencies are nil.
	ErrNilUsageRepository = errors.New("coupon usage repository must not be nil")
	// ErrCouponNotFound is returned when a coupon cannot be found by the provided identifier.
	ErrCouponNotFound = errors.New("coupon not found")
	// ErrCouponCodeConflict is returned when a manually-provided code is already in use.
	ErrCouponCodeConflict = errors.New("coupon code already exists")
	// ErrCouponExhausted is returned when global usage limits are reached.
	ErrCouponExhausted = errors.New("coupon usage limit reached")
	// ErrCouponExhaustedPerEmail is returned when per-email usage limits are reached.
	ErrCouponExhaustedPerEmail = errors.New("coupon usage limit per email reached")
	// ErrCouponExpired is returned when coupons are past their expiry date.
	ErrCouponExpired = errors.New("coupon has expired")
	// ErrCouponInactive is returned when coupons are marked inactive.
	ErrCouponInactive = errors.New("coupon is inactive")
	// ErrCouponAlreadyUsedOnOrder is returned when the same coupon is applied twice to one order.
	ErrCouponAlreadyUsedOnOrder = errors.New("coupon already applied to this order")
)

// CreateCommand defines coupon creation input values.
type CreateCommand struct {
	// Code defines optional explicit coupon codes. When empty a random code is generated.
	Code string
	// Origin defines the coupon source (e.g., "manual", "campaign").
	Origin string
	// DiscountType defines the discount calculation method.
	DiscountType domain.DiscountType
	// DiscountAmount defines the discount value.
	DiscountAmount float64
	// MaxUsagesGlobal defines the optional global usage limit.
	MaxUsagesGlobal *int
	// MaxUsagesPerEmail defines the optional per-email usage limit.
	MaxUsagesPerEmail *int
	// Active defines the initial active state.
	Active bool
	// ExpiresAt defines the optional expiry timestamp.
	ExpiresAt *time.Time
	// AssignedEmails defines the optional list of authorized emails.
	AssignedEmails []string
	// AssignedContactIDs defines the optional list of authorized contact identifiers.
	AssignedContactIDs []string
	// IncludedProductIDs defines the optional product scope.
	IncludedProductIDs []string
	// IncludedCategoryIDs defines the optional category scope.
	IncludedCategoryIDs []string
	// IncludedTagIDs defines the optional tag scope.
	IncludedTagIDs []string
	// WooCommerceID defines an optional WooCommerce coupon identifier for deduplication.
	WooCommerceID *int
}

// UpdateCommand defines coupon mutation input values.
type UpdateCommand struct {
	// ID defines the coupon to update.
	ID string
	// Origin defines the updated origin value.
	Origin string
	// DiscountType defines the updated discount type.
	DiscountType domain.DiscountType
	// DiscountAmount defines the updated discount amount.
	DiscountAmount float64
	// MaxUsagesGlobal defines the updated global usage limit. Nil clears the limit.
	MaxUsagesGlobal *int
	// MaxUsagesPerEmail defines the updated per-email usage limit. Nil clears the limit.
	MaxUsagesPerEmail *int
	// Active defines the updated active state.
	Active bool
	// ExpiresAt defines the updated expiry timestamp. Nil clears the expiry.
	ExpiresAt *time.Time
	// AssignedEmails replaces the list of authorized emails.
	AssignedEmails []string
	// AssignedContactIDs replaces the list of authorized contact identifiers.
	AssignedContactIDs []string
	// IncludedProductIDs replaces the product scope.
	IncludedProductIDs []string
	// IncludedCategoryIDs replaces the category scope.
	IncludedCategoryIDs []string
	// IncludedTagIDs replaces the tag scope.
	IncludedTagIDs []string
}

// RecordUsageCommand defines coupon usage recording input values.
type RecordUsageCommand struct {
	// CouponID defines the coupon to mark as used.
	CouponID string
	// OrderID defines the order where the coupon was applied.
	OrderID string
	// Email defines the email of the redeemer.
	Email string
}

// Service defines coupon management use-case behavior.
type Service struct {
	// repository defines coupon persistence dependencies.
	repository port.CouponRepository
	// usageRepository defines coupon usage persistence dependencies.
	usageRepository port.CouponUsageRepository
	// publisher defines integration event publication dependencies.
	publisher port.IntegrationEventPublisher
}

// NewService creates coupon management services.
func NewService(repo port.CouponRepository, usageRepo port.CouponUsageRepository, publisher port.IntegrationEventPublisher) (*Service, error) {
	if repo == nil {
		return nil, ErrNilRepository
	}
	if usageRepo == nil {
		return nil, ErrNilUsageRepository
	}

	return &Service{
		repository:      repo,
		usageRepository: usageRepo,
		publisher:       couponevent.ResolvePublisher(publisher),
	}, nil
}

// Create creates a new coupon and emits an integration event.
func (s *Service) Create(ctx context.Context, cmd CreateCommand) (*domain.Coupon, error) {
	code := strings.ToUpper(strings.TrimSpace(cmd.Code))
	if code == "" {
		generated, err := s.generateUniqueCode(ctx)
		if err != nil {
			return nil, fmt.Errorf("generate coupon code: %w", err)
		}
		code = generated
	} else {
		exists, err := s.repository.CodeExists(ctx, code)
		if err != nil {
			return nil, fmt.Errorf("check coupon code: %w", err)
		}
		if exists {
			return nil, ErrCouponCodeConflict
		}
	}

	coupon := domain.Coupon{
		ID:                  uuid.NewString(),
		Code:                code,
		Origin:              strings.TrimSpace(cmd.Origin),
		DiscountType:        cmd.DiscountType,
		DiscountAmount:      cmd.DiscountAmount,
		MaxUsagesGlobal:     cmd.MaxUsagesGlobal,
		MaxUsagesPerEmail:   cmd.MaxUsagesPerEmail,
		Active:              cmd.Active,
		ExpiresAt:           cmd.ExpiresAt,
		AssignedEmails:      cmd.AssignedEmails,
		AssignedContactIDs:  cmd.AssignedContactIDs,
		IncludedProductIDs:  cmd.IncludedProductIDs,
		IncludedCategoryIDs: cmd.IncludedCategoryIDs,
		IncludedTagIDs:      cmd.IncludedTagIDs,
		WooCommerceID:       cmd.WooCommerceID,
		CreatedAt:           time.Now().UTC(),
		UpdatedAt:           time.Now().UTC(),
	}
	coupon.Normalize()
	if err := coupon.Validate(); err != nil {
		return nil, err
	}

	if err := s.repository.Create(ctx, &coupon); err != nil {
		return nil, fmt.Errorf("persist coupon: %w", err)
	}

	_ = s.publisher.Publish(ctx, couponevent.NewCouponCreatedEvent(coupon))

	return &coupon, nil
}

// GetByID retrieves a coupon by its unique identifier.
func (s *Service) GetByID(ctx context.Context, id string) (*domain.Coupon, error) {
	coupon, err := s.repository.GetByID(ctx, strings.TrimSpace(id))
	if err != nil {
		return nil, fmt.Errorf("get coupon: %w", err)
	}
	if coupon == nil {
		return nil, ErrCouponNotFound
	}

	return coupon, nil
}

// GetByCode retrieves a coupon by its unique code.
func (s *Service) GetByCode(ctx context.Context, code string) (*domain.Coupon, error) {
	coupon, err := s.repository.GetByCode(ctx, strings.ToUpper(strings.TrimSpace(code)))
	if err != nil {
		return nil, fmt.Errorf("get coupon by code: %w", err)
	}
	if coupon == nil {
		return nil, ErrCouponNotFound
	}

	return coupon, nil
}

// GetByWooCommerceID retrieves a coupon by its WooCommerce identifier.
func (s *Service) GetByWooCommerceID(ctx context.Context, wooID int) (*domain.Coupon, error) {
	coupon, err := s.repository.GetByWooCommerceID(ctx, wooID)
	if err != nil {
		return nil, fmt.Errorf("get coupon by woocommerce id: %w", err)
	}

	return coupon, nil
}

// Update applies mutations to an existing coupon and emits an integration event.
func (s *Service) Update(ctx context.Context, cmd UpdateCommand) (*domain.Coupon, error) {
	coupon, err := s.repository.GetByID(ctx, strings.TrimSpace(cmd.ID))
	if err != nil {
		return nil, fmt.Errorf("get coupon for update: %w", err)
	}
	if coupon == nil {
		return nil, ErrCouponNotFound
	}

	coupon.Origin = strings.TrimSpace(cmd.Origin)
	coupon.DiscountType = cmd.DiscountType
	coupon.DiscountAmount = cmd.DiscountAmount
	coupon.MaxUsagesGlobal = cmd.MaxUsagesGlobal
	coupon.MaxUsagesPerEmail = cmd.MaxUsagesPerEmail
	coupon.Active = cmd.Active
	coupon.ExpiresAt = cmd.ExpiresAt
	coupon.AssignedEmails = cmd.AssignedEmails
	coupon.AssignedContactIDs = cmd.AssignedContactIDs
	coupon.IncludedProductIDs = cmd.IncludedProductIDs
	coupon.IncludedCategoryIDs = cmd.IncludedCategoryIDs
	coupon.IncludedTagIDs = cmd.IncludedTagIDs
	coupon.UpdatedAt = time.Now().UTC()
	coupon.Normalize()
	if err := coupon.Validate(); err != nil {
		return nil, err
	}

	if err := s.repository.Update(ctx, coupon); err != nil {
		return nil, fmt.Errorf("persist coupon update: %w", err)
	}

	_ = s.publisher.Publish(ctx, couponevent.NewCouponUpdatedEvent(*coupon))

	return coupon, nil
}

// Delete soft-deletes a coupon and emits an integration event.
func (s *Service) Delete(ctx context.Context, id string) error {
	coupon, err := s.repository.GetByID(ctx, strings.TrimSpace(id))
	if err != nil {
		return fmt.Errorf("get coupon for delete: %w", err)
	}
	if coupon == nil {
		return ErrCouponNotFound
	}

	if err := s.repository.Delete(ctx, coupon.ID); err != nil {
		return fmt.Errorf("delete coupon: %w", err)
	}

	_ = s.publisher.Publish(ctx, couponevent.NewCouponDeletedEvent(*coupon))

	return nil
}

// List retrieves paginated coupons matching the provided query.
func (s *Service) List(ctx context.Context, query port.ListQuery) ([]domain.Coupon, int64, error) {
	return s.repository.List(ctx, query)
}

// RecordUsage validates limits and records a coupon redemption event.
func (s *Service) RecordUsage(ctx context.Context, cmd RecordUsageCommand) error {
	coupon, err := s.repository.GetByID(ctx, strings.TrimSpace(cmd.CouponID))
	if err != nil {
		return fmt.Errorf("get coupon for usage: %w", err)
	}
	if coupon == nil {
		return ErrCouponNotFound
	}

	if !coupon.Active {
		return ErrCouponInactive
	}
	if coupon.ExpiresAt != nil && time.Now().UTC().After(*coupon.ExpiresAt) {
		return ErrCouponExpired
	}

	alreadyUsed, err := s.usageRepository.UsageExistsForOrder(ctx, coupon.ID, strings.TrimSpace(cmd.OrderID))
	if err != nil {
		return fmt.Errorf("check coupon order usage: %w", err)
	}
	if alreadyUsed {
		return ErrCouponAlreadyUsedOnOrder
	}

	if coupon.MaxUsagesGlobal != nil {
		count, countErr := s.usageRepository.CountGlobalUsage(ctx, coupon.ID)
		if countErr != nil {
			return fmt.Errorf("count global coupon usage: %w", countErr)
		}
		if count >= int64(*coupon.MaxUsagesGlobal) {
			return ErrCouponExhausted
		}
	}

	email := strings.ToLower(strings.TrimSpace(cmd.Email))
	if coupon.MaxUsagesPerEmail != nil && email != "" {
		count, countErr := s.usageRepository.CountUsageByEmail(ctx, coupon.ID, email)
		if countErr != nil {
			return fmt.Errorf("count email coupon usage: %w", countErr)
		}
		if count >= int64(*coupon.MaxUsagesPerEmail) {
			return ErrCouponExhaustedPerEmail
		}
	}

	usedAt := time.Now().UTC()
	if err := s.usageRepository.RecordUsage(ctx, port.UsageRecord{
		CouponID: coupon.ID,
		OrderID:  strings.TrimSpace(cmd.OrderID),
		Email:    email,
		UsedAt:   usedAt,
	}); err != nil {
		return fmt.Errorf("record coupon usage: %w", err)
	}

	_ = s.publisher.Publish(ctx, couponevent.NewCouponUsedEvent(coupon.ID, coupon.Code, cmd.OrderID, email, usedAt))

	return nil
}
