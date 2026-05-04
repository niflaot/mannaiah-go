package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"go.uber.org/zap"
	ordersport "mannaiah/module/orders/port"
	"mannaiah/module/woocommerce/port"
)

var (
	// ErrNilDestination is returned when a nil WooCommerce destination is used.
	ErrNilDestination = errors.New("woocommerce order destination must not be nil")
)

// MainstreamUpdateService defines mainstream-origin order update synchronization behavior.
type MainstreamUpdateService struct {
	// destination defines WooCommerce order destination dependencies.
	destination port.OrderDestination
	// logger defines structured log dependencies.
	logger *zap.Logger
	// breaker guards outbound destination operations.
	breaker CircuitBreaker
}

// NewMainstreamUpdateService creates mainstream-origin WooCommerce order update services.
func NewMainstreamUpdateService(destination port.OrderDestination, providedLogger *zap.Logger, breakers ...CircuitBreakers) (*MainstreamUpdateService, error) {
	if destination == nil {
		return nil, ErrNilDestination
	}

	resolvedBreakers := resolveCircuitBreakers(breakers)
	return &MainstreamUpdateService{
		destination: destination,
		logger:      resolveLogger(providedLogger),
		breaker:     resolvedBreakers.Source,
	}, nil
}

// HandleOrderEvent handles order integration event payload values for mainstream-origin WooCommerce updates.
func (s *MainstreamUpdateService) HandleOrderEvent(ctx context.Context, payload ordersport.OrderEventPayload) error {
	if !isWooRealm(payload.Realm) || isLoopSource(payload.Source) {
		return nil
	}
	if strings.TrimSpace(payload.Identifier) == "" {
		return nil
	}
	if !isWooNumericIdentifier(payload.Identifier) {
		s.logger.Debug(
			"skip woocommerce outbound update for non-numeric identifier",
			zap.String("identifier", strings.TrimSpace(payload.Identifier)),
			zap.String("realm", strings.TrimSpace(payload.Realm)),
		)
		return nil
	}

	validateErr := executeOptionalBreaker(s.breaker, func() error {
		return s.destination.Validate(ctx)
	})
	if validateErr != nil {
		return fmt.Errorf("validate woocommerce destination: %w", validateErr)
	}

	command := mapMainstreamCommand(payload)
	updateErr := executeOptionalBreaker(s.breaker, func() error {
		return s.destination.UpdateOrderFromMainstream(ctx, command)
	})
	if updateErr != nil {
		return fmt.Errorf("update woocommerce order from mainstream event: %w", updateErr)
	}

	return nil
}

// mapMainstreamCommand maps order integration event payload values to WooCommerce update command values.
func mapMainstreamCommand(payload ordersport.OrderEventPayload) port.MainstreamOrderUpdateCommand {
	items := make([]port.OrderSyncItem, 0, len(payload.Items))
	for _, row := range payload.Items {
		items = append(items, port.OrderSyncItem{
			SKU:      strings.TrimSpace(row.SKU),
			Name:     strings.TrimSpace(row.AlternateName),
			Quantity: row.Quantity,
			Value:    row.Value,
		})
	}

	charges := make([]port.OrderSyncShippingCharge, 0, len(payload.ShippingCharges))
	for _, row := range payload.ShippingCharges {
		charges = append(charges, port.OrderSyncShippingCharge{
			MethodID:    strings.TrimSpace(row.MethodID),
			MethodTitle: strings.TrimSpace(row.MethodTitle),
			Price:       row.Price,
		})
	}

	command := port.MainstreamOrderUpdateCommand{
		Identifier:      strings.TrimSpace(payload.Identifier),
		Status:          mapMainstreamStatus(firstNonEmptyStatus(payload.LatestStatus.Status, payload.CurrentStatus)),
		ShippingCharges: charges,
		Items:           items,
	}
	if payload.ShippingAddress.Address != "" || payload.ShippingAddress.Address2 != "" || payload.ShippingAddress.Phone != "" || payload.ShippingAddress.CityCode != "" {
		command.ShippingAddress = &port.OrderSyncShippingAddress{
			Address:  strings.TrimSpace(payload.ShippingAddress.Address),
			Address2: strings.TrimSpace(payload.ShippingAddress.Address2),
			Phone:    strings.TrimSpace(payload.ShippingAddress.Phone),
			CityCode: strings.TrimSpace(payload.ShippingAddress.CityCode),
		}
	}

	return command
}

// firstNonEmptyStatus resolves the first non-empty status value.
func firstNonEmptyStatus(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}

	return ""
}

// mapMainstreamStatus maps order-domain/mainstream status values to WooCommerce status values.
func mapMainstreamStatus(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "created", "processing":
		return "processing"
	case "pending", "pending-payment":
		return "pending"
	case "hold", "on_hold", "on-hold":
		return "on-hold"
	case "completed", "complete":
		return "completed"
	case "cancelled", "canceled", "failed":
		return "cancelled"
	default:
		return ""
	}
}

// isWooRealm reports whether realm values target WooCommerce integration.
func isWooRealm(realm string) bool {
	return strings.EqualFold(strings.TrimSpace(realm), "woocommerce")
}

// isLoopSource reports whether source values indicate WooCommerce-origin updates that should not be republished.
func isLoopSource(source string) bool {
	normalized := strings.ToLower(strings.TrimSpace(source))
	if normalized == "" {
		return false
	}

	return strings.HasPrefix(normalized, "woocommerce")
}

// isWooNumericIdentifier reports whether identifier values are valid WooCommerce numeric order identifiers.
func isWooNumericIdentifier(identifier string) bool {
	trimmed := strings.TrimSpace(identifier)
	if trimmed == "" {
		return false
	}

	value, err := strconv.Atoi(trimmed)
	return err == nil && value > 0
}

// executeOptionalBreaker executes operations behind optional circuit breakers.
func executeOptionalBreaker(breaker CircuitBreaker, operation func() error) error {
	if breaker == nil {
		return operation()
	}

	return breaker.Execute(operation)
}
