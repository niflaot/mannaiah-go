package tcc

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"mannaiah/module/shipping/domain"
)

const (
	guardrailCODPaymentFormRule            = "tcc_cod_formapago_must_be_2"
	guardrailCODCollectAmountRule          = "tcc_cod_recaudoproducto_required_positive"
	guardrailCODTotalProductValueRule      = "tcc_cod_totalvalorproducto_required_positive"
	guardrailNonCODPaymentFormRule         = "tcc_non_cod_formapago_must_be_1"
	guardrailNonCODCollectAmountAbsentRule = "tcc_non_cod_recaudoproducto_must_not_exist"
	guardrailNonCODTotalValueAbsentRule    = "tcc_non_cod_totalvalorproducto_must_not_exist"
)

// validateDispatchGuardrails enforces TCC COD/non-COD dispatch invariants before carrier submission.
func validateDispatchGuardrails(mark domain.ShippingMark, request DispatchRequest) error {
	isCOD := mark.CollectOnDeliveryChargedAmount > 0 || mark.CollectOnDeliveryAmount > 0

	if isCOD {
		if strings.TrimSpace(request.PaymentForm) != "2" {
			return newDispatchGuardrailViolation(mark, guardrailCODPaymentFormRule, request)
		}
		if !isPositivePointerFloat(request.CollectOnDeliveryAmount) {
			return newDispatchGuardrailViolation(mark, guardrailCODCollectAmountRule, request)
		}
		if !isPositivePointerFloat(request.TotalProductValue) {
			return newDispatchGuardrailViolation(mark, guardrailCODTotalProductValueRule, request)
		}

		return nil
	}

	if strings.TrimSpace(request.PaymentForm) != "1" {
		return newDispatchGuardrailViolation(mark, guardrailNonCODPaymentFormRule, request)
	}
	if request.CollectOnDeliveryAmount != nil {
		return newDispatchGuardrailViolation(mark, guardrailNonCODCollectAmountAbsentRule, request)
	}
	if request.TotalProductValue != nil {
		return newDispatchGuardrailViolation(mark, guardrailNonCODTotalValueAbsentRule, request)
	}

	return nil
}

func newDispatchGuardrailViolation(mark domain.ShippingMark, rule string, request DispatchRequest) error {
	previewBytes, err := json.Marshal(request)
	preview := "{}"
	if err == nil {
		preview = strings.TrimSpace(string(previewBytes))
	}

	return &domain.GuardrailViolationError{
		CarrierID:      "tcc",
		MarkID:         strings.TrimSpace(mark.ID),
		OrderID:        strings.TrimSpace(mark.OrderID),
		Rule:           strings.TrimSpace(rule),
		RequestPreview: preview,
	}
}

func isPositivePointerFloat(value *string) bool {
	if value == nil {
		return false
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return false
	}
	parsed, err := strconv.ParseFloat(trimmed, 64)
	if err != nil {
		return false
	}

	return parsed > 0
}

// formatDispatchCODAmountPointer converts COD amount values into pointer payload fields.
func formatDispatchCODAmountPointer(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	copy := fmt.Sprintf("%s", trimmed)

	return &copy
}
