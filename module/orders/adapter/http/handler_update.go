package http

import (
	corehttp "mannaiah/module/core/http"
	ordersapplication "mannaiah/module/orders/application"
)

// updateRequest defines request payload for mutable order updates.
type updateRequest struct {
	// Items defines optional order item payload values.
	Items *[]createItemRequest `json:"items"`
	// ShippingAddress defines optional explicit shipping-address values.
	ShippingAddress *shippingAddressRequest `json:"shippingAddress"`
	// ShippingCharges defines optional shipping charge values.
	ShippingCharges *[]shippingChargeRequest `json:"shippingCharges"`
	// CouponCode defines optional order-level coupon attribution code values.
	CouponCode *string `json:"couponCode"`
	// CouponDiscountAmount defines optional order-level coupon attribution amount values.
	CouponDiscountAmount *float64 `json:"couponDiscountAmount"`
	// CouponDiscountType defines optional order-level coupon attribution type values.
	CouponDiscountType *string `json:"couponDiscountType"`
	// Source defines optional mutation source values.
	Source string `json:"source,omitempty"`
}

// update handles mutable order update endpoints.
func (h *Handler) update(ctx corehttp.Context) error {
	var request updateRequest
	if err := ctx.BodyParser(&request); err != nil {
		return corehttp.NewAppError(400, "invalid_payload", err)
	}

	command := ordersapplication.UpdateCommand{
		Items:                mapOptionalCreateItems(request.Items),
		ShippingAddress:      mapOptionalShippingAddress(request.ShippingAddress),
		ShippingCharges:      mapOptionalShippingCharges(request.ShippingCharges),
		CouponCode:           request.CouponCode,
		CouponDiscountAmount: request.CouponDiscountAmount,
		CouponDiscountType:   request.CouponDiscountType,
		Source:               resolveCommandSource(ctx, request.Source),
	}
	entity, err := h.service.Update(ctx.Context(), ctx.Params("id"), command)
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(entity)
}

// mapOptionalCreateItems maps optional request item payloads to application command values.
func mapOptionalCreateItems(items *[]createItemRequest) *[]ordersapplication.CreateItemCommand {
	if items == nil {
		return nil
	}

	rows := mapCreateItems(*items)
	return &rows
}

// mapOptionalShippingAddress maps optional shipping-address payload values to application command values.
func mapOptionalShippingAddress(value *shippingAddressRequest) *ordersapplication.ShippingAddressCommand {
	if value == nil {
		return nil
	}

	return &ordersapplication.ShippingAddressCommand{
		Address:  value.Address,
		Address2: value.Address2,
		Phone:    value.Phone,
		CityCode: value.CityCode,
	}
}

// mapOptionalShippingCharges maps optional shipping charge payload values to application command values.
func mapOptionalShippingCharges(values *[]shippingChargeRequest) *[]ordersapplication.ShippingChargeCommand {
	if values == nil {
		return nil
	}

	rows := mapShippingCharges(*values)
	return &rows
}
