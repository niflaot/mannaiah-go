package orders

import (
	"math"
	"strconv"
	"strings"
	"time"

	ordersapplication "mannaiah/module/orders/application"
	ordersdomain "mannaiah/module/orders/domain"
	"mannaiah/module/woocommerce/port"
)

// toCreateCommand maps order sync command values to order creation payload values.
func toCreateCommand(
	command port.OrderSyncCommand,
	contactID string,
	realm string,
	status ordersdomain.Status,
	references map[string]CouponReference,
) ordersapplication.CreateCommand {
	return ordersapplication.CreateCommand{
		Identifier:      strings.TrimSpace(command.Identifier),
		Realm:           strings.TrimSpace(realm),
		ContactID:       strings.TrimSpace(contactID),
		Items:           toCreateItems(command.Items),
		InitialStatus:   &status,
		Author:          syncStatusAuthor,
		Description:     syncStatusDescription,
		ShippingAddress: toShippingAddress(command.ShippingAddress),
		ShippingCharges: toShippingCharges(command.ShippingCharges),
		PaymentMethod:   strings.TrimSpace(command.PaymentMethod),
		AppliedCoupons:  toAppliedCoupons(command.AppliedCoupons, command.CreatedAt, nil, references),
		Metadata:        normalizeMetadata(command.Metadata),
		CreatedAt:       command.CreatedAt,
		Source:          syncStatusAuthor,
	}
}

// toUpdateCommand maps order sync command values to mutable order update payload values.
func toUpdateCommand(command port.OrderSyncCommand, existing ordersdomain.Order, references map[string]CouponReference) ordersapplication.UpdateCommand {
	items := toCreateItems(command.Items)
	shippingCharges := toUpdateShippingCharges(command.ShippingCharges)
	appliedCoupons := toUpdateAppliedCoupons(command.AppliedCoupons, command.CreatedAt, existing.AppliedCoupons, references)

	return ordersapplication.UpdateCommand{
		Items:           optionalCreateItems(items),
		ShippingAddress: toShippingAddress(command.ShippingAddress),
		ShippingCharges: shippingCharges,
		AppliedCoupons:  appliedCoupons,
		Source:          syncStatusAuthor,
	}
}

// toCreateItems maps sync item values to create command item values.
func toCreateItems(items []port.OrderSyncItem) []ordersapplication.CreateItemCommand {
	result := make([]ordersapplication.CreateItemCommand, 0, len(items))
	for _, item := range items {
		sku := strings.TrimSpace(item.SKU)
		alternateName := strings.TrimSpace(item.Name)
		if sku == "" && alternateName == "" {
			continue
		}

		quantity := item.Quantity
		if quantity <= 0 {
			quantity = 1
		}

		result = append(result, ordersapplication.CreateItemCommand{
			SKU:           sku,
			AlternateName: alternateName,
			Quantity:      quantity,
			Value:         item.Value,
		})
	}

	return result
}

// toShippingAddress maps optional sync shipping address values to create command shipping values.
func toShippingAddress(value *port.OrderSyncShippingAddress) *ordersapplication.ShippingAddressCommand {
	if value == nil {
		return nil
	}

	command := &ordersapplication.ShippingAddressCommand{
		Address:  strings.TrimSpace(value.Address),
		Address2: strings.TrimSpace(value.Address2),
		Phone:    strings.TrimSpace(value.Phone),
		CityCode: strings.TrimSpace(value.CityCode),
	}
	if command.Address == "" && command.Address2 == "" && command.Phone == "" && command.CityCode == "" {
		return nil
	}

	return command
}

// toShippingCharges maps sync shipping charge values to create command shipping-charge values.
func toShippingCharges(values []port.OrderSyncShippingCharge) []ordersapplication.ShippingChargeCommand {
	if len(values) == 0 {
		return nil
	}

	rows := make([]ordersapplication.ShippingChargeCommand, 0, len(values))
	for _, value := range values {
		methodID := strings.TrimSpace(value.MethodID)
		methodTitle := strings.TrimSpace(value.MethodTitle)
		if methodID == "" && methodTitle == "" && value.Price == 0 {
			continue
		}
		rows = append(rows, ordersapplication.ShippingChargeCommand{
			MethodID:    methodID,
			MethodTitle: methodTitle,
			Price:       value.Price,
		})
	}
	if len(rows) == 0 {
		return nil
	}

	return rows
}

// toUpdateShippingCharges maps sync shipping charge values to mutable update payloads.
func toUpdateShippingCharges(values []port.OrderSyncShippingCharge) *[]ordersapplication.ShippingChargeCommand {
	rows := make([]ordersapplication.ShippingChargeCommand, 0, len(values))
	for _, value := range values {
		methodID := strings.TrimSpace(value.MethodID)
		methodTitle := strings.TrimSpace(value.MethodTitle)
		if methodID == "" && methodTitle == "" && value.Price == 0 {
			continue
		}
		rows = append(rows, ordersapplication.ShippingChargeCommand{
			MethodID:    methodID,
			MethodTitle: methodTitle,
			Price:       value.Price,
		})
	}

	return &rows
}

// toAppliedCoupons maps WooCommerce coupon-line values to applied-coupon command values.
func toAppliedCoupons(values []port.OrderSyncAppliedCoupon, defaultAppliedAt *time.Time, existing []ordersdomain.AppliedCoupon, references map[string]CouponReference) []ordersapplication.AppliedCouponCommand {
	if len(values) == 0 {
		return nil
	}

	existingByCode := make(map[string]ordersdomain.AppliedCoupon, len(existing))
	for _, coupon := range existing {
		code := strings.ToUpper(strings.TrimSpace(coupon.Code))
		if code == "" {
			continue
		}
		existingByCode[code] = coupon
	}

	rows := make([]ordersapplication.AppliedCouponCommand, 0, len(values))
	for _, value := range values {
		code := strings.TrimSpace(value.Code)
		if code == "" {
			continue
		}
		normalizedCode := strings.ToUpper(code)
		amount, _ := strconv.ParseFloat(strings.TrimSpace(value.Discount), 64)
		existingCoupon, hasExisting := existingByCode[normalizedCode]
		reference, hasReference := references[normalizedCode]
		appliedAt := time.Now().UTC()
		if hasExisting && !existingCoupon.AppliedAt.IsZero() {
			appliedAt = existingCoupon.AppliedAt.UTC()
		}
		if defaultAppliedAt != nil && !defaultAppliedAt.IsZero() {
			appliedAt = defaultAppliedAt.UTC()
		}
		couponID := strings.TrimSpace(existingCoupon.CouponID)
		if hasReference && strings.TrimSpace(reference.ID) != "" {
			couponID = strings.TrimSpace(reference.ID)
		}
		discountType := strings.TrimSpace(existingCoupon.DiscountType)
		if discountType == "" && hasReference {
			discountType = strings.TrimSpace(reference.DiscountType)
		}
		rows = append(rows, ordersapplication.AppliedCouponCommand{
			CouponID:       couponID,
			Code:           code,
			DiscountType:   discountType,
			DiscountAmount: amount,
			AppliedAt:      &appliedAt,
		})
	}
	if len(rows) == 0 {
		return nil
	}

	return rows
}

// toUpdateAppliedCoupons maps sync applied coupon values to mutable update payloads.
func toUpdateAppliedCoupons(values []port.OrderSyncAppliedCoupon, defaultAppliedAt *time.Time, existing []ordersdomain.AppliedCoupon, references map[string]CouponReference) *[]ordersapplication.AppliedCouponCommand {
	rows := toAppliedCoupons(values, defaultAppliedAt, existing, references)
	if rows == nil {
		empty := []ordersapplication.AppliedCouponCommand{}
		return &empty
	}

	return &rows
}

// optionalCreateItems maps populated item rows to optional update payload pointers.
func optionalCreateItems(items []ordersapplication.CreateItemCommand) *[]ordersapplication.CreateItemCommand {
	if len(items) == 0 {
		return nil
	}

	return &items
}

// hasMutableOrderStateChanges reports whether mutable order state differs between two orders.
func hasMutableOrderStateChanges(left ordersdomain.Order, right ordersdomain.Order) bool {
	if len(left.Items) != len(right.Items) {
		return true
	}
	for index := range left.Items {
		if !itemsEqual(left.Items[index], right.Items[index]) {
			return true
		}
	}
	if left.HasCustomShippingAddress != right.HasCustomShippingAddress {
		return true
	}
	if !shippingAddressEqual(left.ShippingAddress, right.ShippingAddress) {
		return true
	}
	if len(left.ShippingCharges) != len(right.ShippingCharges) {
		return true
	}
	for index := range left.ShippingCharges {
		if !shippingChargeEqual(left.ShippingCharges[index], right.ShippingCharges[index]) {
			return true
		}
	}
	if len(left.AppliedCoupons) != len(right.AppliedCoupons) {
		return true
	}
	for index := range left.AppliedCoupons {
		if !appliedCouponEqual(left.AppliedCoupons[index], right.AppliedCoupons[index]) {
			return true
		}
	}

	return false
}

// itemsEqual reports whether mutable order item state is equivalent.
func itemsEqual(left ordersdomain.Item, right ordersdomain.Item) bool {
	return strings.EqualFold(strings.TrimSpace(left.SKU), strings.TrimSpace(right.SKU)) &&
		strings.EqualFold(strings.TrimSpace(left.AlternateName), strings.TrimSpace(right.AlternateName)) &&
		left.Quantity == right.Quantity &&
		math.Abs(left.Value-right.Value) <= 0.000001 &&
		strings.EqualFold(strings.TrimSpace(left.ProductID), strings.TrimSpace(right.ProductID)) &&
		strings.EqualFold(strings.TrimSpace(string(left.ResolutionSource)), strings.TrimSpace(string(right.ResolutionSource)))
}

// shippingAddressEqual reports whether mutable shipping-address state is equivalent.
func shippingAddressEqual(left ordersdomain.ShippingAddress, right ordersdomain.ShippingAddress) bool {
	return strings.TrimSpace(left.Address) == strings.TrimSpace(right.Address) &&
		strings.TrimSpace(left.Address2) == strings.TrimSpace(right.Address2) &&
		strings.TrimSpace(left.Phone) == strings.TrimSpace(right.Phone) &&
		strings.TrimSpace(left.CityCode) == strings.TrimSpace(right.CityCode)
}

// shippingChargeEqual reports whether mutable shipping-charge state is equivalent.
func shippingChargeEqual(left ordersdomain.ShippingCharge, right ordersdomain.ShippingCharge) bool {
	return strings.EqualFold(strings.TrimSpace(left.MethodID), strings.TrimSpace(right.MethodID)) &&
		strings.EqualFold(strings.TrimSpace(left.MethodTitle), strings.TrimSpace(right.MethodTitle)) &&
		math.Abs(left.Price-right.Price) <= 0.000001
}

// appliedCouponEqual reports whether mutable applied-coupon state is equivalent.
func appliedCouponEqual(left ordersdomain.AppliedCoupon, right ordersdomain.AppliedCoupon) bool {
	return strings.EqualFold(strings.TrimSpace(left.CouponID), strings.TrimSpace(right.CouponID)) &&
		strings.EqualFold(strings.TrimSpace(left.Code), strings.TrimSpace(right.Code)) &&
		strings.EqualFold(strings.TrimSpace(left.DiscountType), strings.TrimSpace(right.DiscountType)) &&
		math.Abs(left.DiscountAmount-right.DiscountAmount) <= 0.000001 &&
		left.AppliedAt.UTC().Equal(right.AppliedAt.UTC())
}

// mapOrderStatus maps WooCommerce source status values to order-domain status values.
func mapOrderStatus(value string) ordersdomain.Status {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "cancelled", "canceled", "failed":
		return ordersdomain.StatusCancelled
	case "processing", "created":
		return ordersdomain.StatusCreated
	case "on-hold", "hold":
		return ordersdomain.StatusHold
	case "pending", "pending-payment":
		return ordersdomain.StatusPending
	case "completed", "complete":
		return ordersdomain.StatusCompleted
	default:
		return ordersdomain.StatusCreated
	}
}
