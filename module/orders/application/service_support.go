package application

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	ordersdomain "mannaiah/module/orders/domain"
	ordersport "mannaiah/module/orders/port"
)

const (
	// itemResolveWorkers defines max item-resolution worker count.
	itemResolveWorkers = 8
)

// resolveItems resolves order-item commands and attempts product matching.
func (s *OrderService) resolveItems(ctx context.Context, commands []CreateItemCommand) ([]ordersdomain.Item, error) {
	if len(commands) == 0 {
		return nil, nil
	}

	rows := make([]ordersdomain.Item, len(commands))
	workerCount := itemResolveWorkers
	if workerCount > len(commands) {
		workerCount = len(commands)
	}

	indexChannel := make(chan int, len(commands))
	errorChannel := make(chan error, 1)
	var waitGroup sync.WaitGroup

	for worker := 0; worker < workerCount; worker++ {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			for index := range indexChannel {
				item, err := s.resolveItem(ctx, commands[index])
				if err != nil {
					select {
					case errorChannel <- err:
					default:
					}
					return
				}
				rows[index] = item
			}
		}()
	}

	for index := range commands {
		select {
		case indexChannel <- index:
		case err := <-errorChannel:
			close(indexChannel)
			waitGroup.Wait()
			return nil, err
		}
	}
	close(indexChannel)
	waitGroup.Wait()

	select {
	case err := <-errorChannel:
		return nil, err
	default:
		return rows, nil
	}
}

// resolveItem resolves one order-item command with product lookup fallbacks.
func (s *OrderService) resolveItem(ctx context.Context, command CreateItemCommand) (ordersdomain.Item, error) {
	item := ordersdomain.Item{
		SKU:              strings.TrimSpace(command.SKU),
		AlternateName:    strings.TrimSpace(command.AlternateName),
		ProductID:        strings.TrimSpace(command.ProductID),
		Quantity:         command.Quantity,
		Value:            command.Value,
		ResolutionSource: ordersdomain.ItemResolutionSourceUnresolved,
	}
	if item.ProductID != "" {
		item.ResolutionSource = ordersdomain.ItemResolutionSourceSKU
		return item, nil
	}
	if s.productResolver == nil {
		return item, nil
	}

	resolution, err := s.productResolver.Resolve(ctx, item.SKU, item.AlternateName)
	if err != nil {
		return ordersdomain.Item{}, fmt.Errorf("resolve order item product (%s): %w", item.SKU, err)
	}
	if resolution == nil {
		return item, nil
	}

	item.ProductID = strings.TrimSpace(resolution.ProductID)
	switch strings.ToLower(strings.TrimSpace(resolution.MatchedBy)) {
	case "sku":
		item.ResolutionSource = ordersdomain.ItemResolutionSourceSKU
	case "alternate_name":
		item.ResolutionSource = ordersdomain.ItemResolutionSourceAlternateName
	default:
		item.ResolutionSource = ordersdomain.ItemResolutionSourceUnresolved
	}

	return item, nil
}

// applyShipping applies custom shipping rows and fallback billing values.
func applyShipping(order *ordersdomain.Order, customer *ordersport.Customer, shipping *ShippingAddressCommand) {
	if order == nil {
		return
	}

	billing := customerShipping(customer)
	if shipping == nil {
		order.ShippingAddress = billing
		order.HasCustomShippingAddress = false
		return
	}

	custom := normalizeShippingCommand(*shipping)
	order.ShippingAddress = custom
	order.HasCustomShippingAddress = true
}

// enrichShippingWithBilling applies billing fallbacks when custom shipping rows are absent.
func (s *OrderService) enrichShippingWithBilling(ctx context.Context, order *ordersdomain.Order) {
	if order == nil || order.HasCustomShippingAddress {
		return
	}

	customer, err := s.customerSource.GetByID(ctx, order.ContactID)
	if err != nil || customer == nil {
		return
	}

	order.ShippingAddress = customerShipping(customer)
}

// customerShipping maps customer billing-address values to shipping-address values.
func customerShipping(customer *ordersport.Customer) ordersdomain.ShippingAddress {
	if customer == nil {
		return ordersdomain.ShippingAddress{}
	}

	return ordersdomain.ShippingAddress{
		Address:  strings.TrimSpace(customer.Address),
		Address2: strings.TrimSpace(customer.AddressExtra),
		Phone:    strings.TrimSpace(customer.Phone),
		CityCode: strings.TrimSpace(customer.CityCode),
	}
}

// normalizeShippingCommand normalizes shipping commands.
func normalizeShippingCommand(value ShippingAddressCommand) ordersdomain.ShippingAddress {
	return ordersdomain.ShippingAddress{
		Address:  strings.TrimSpace(value.Address),
		Address2: strings.TrimSpace(value.Address2),
		Phone:    strings.TrimSpace(value.Phone),
		CityCode: strings.TrimSpace(value.CityCode),
	}
}

// shippingEqual reports whether shipping values are equivalent.
func shippingEqual(left ordersdomain.ShippingAddress, right ordersdomain.ShippingAddress) bool {
	return strings.TrimSpace(left.Address) == strings.TrimSpace(right.Address) &&
		strings.TrimSpace(left.Address2) == strings.TrimSpace(right.Address2) &&
		strings.TrimSpace(left.Phone) == strings.TrimSpace(right.Phone) &&
		strings.TrimSpace(left.CityCode) == strings.TrimSpace(right.CityCode)
}

// normalizeAppliedCoupons maps applied-coupon command values to domain applied-coupon values.
func normalizeAppliedCoupons(values []AppliedCouponCommand) []ordersdomain.AppliedCoupon {
	if len(values) == 0 {
		return nil
	}

	rows := make([]ordersdomain.AppliedCoupon, 0, len(values))
	for _, value := range values {
		code := strings.TrimSpace(value.Code)
		if code == "" {
			continue
		}
		appliedAt := time.Now().UTC()
		if value.AppliedAt != nil && !value.AppliedAt.IsZero() {
			appliedAt = value.AppliedAt.UTC()
		}
		rows = append(rows, ordersdomain.AppliedCoupon{
			CouponID:       strings.TrimSpace(value.CouponID),
			Code:           code,
			DiscountType:   strings.TrimSpace(value.DiscountType),
			DiscountAmount: value.DiscountAmount,
			AppliedAt:      appliedAt,
		})
	}
	if len(rows) == 0 {
		return nil
	}

	return rows
}

// normalizeShippingCharges normalizes shipping charge command values.
func normalizeShippingCharges(values []ShippingChargeCommand) []ordersdomain.ShippingCharge {
	if len(values) == 0 {
		return nil
	}

	rows := make([]ordersdomain.ShippingCharge, 0, len(values))
	for _, value := range values {
		methodID := strings.TrimSpace(value.MethodID)
		methodTitle := strings.TrimSpace(value.MethodTitle)
		if methodID == "" && methodTitle == "" && value.Price == 0 {
			continue
		}
		price := value.Price
		if price < 0 {
			price = 0
		}
		rows = append(rows, ordersdomain.ShippingCharge{
			MethodID:    methodID,
			MethodTitle: methodTitle,
			Price:       price,
		})
	}
	if len(rows) == 0 {
		return nil
	}

	return rows
}

// validateStatusEntry validates status-entry values.
func validateStatusEntry(entry ordersdomain.StatusEntry) error {
	entity := ordersdomain.Order{
		Identifier:    "validation-id",
		Realm:         "validation-realm",
		ContactID:     "validation-contact",
		Items:         []ordersdomain.Item{{SKU: "sku", Quantity: 1}},
		CurrentStatus: entry.Status,
		StatusHistory: []ordersdomain.StatusEntry{entry},
	}

	return entity.Validate()
}
