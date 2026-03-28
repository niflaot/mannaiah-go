package main

import (
	"context"
	"testing"

	ordersapplication "mannaiah/module/orders/application"
	ordersdomain "mannaiah/module/orders/domain"
)

// shippingOrderQuotationServiceStub defines order lookup behavior for shipping quotation order adapter tests.
type shippingOrderQuotationServiceStub struct {
	// order defines the order returned by Get.
	order *ordersdomain.Order
	// list defines fallback list result values.
	list *ordersapplication.ListResult
}

// Get resolves one configured order value.
func (s shippingOrderQuotationServiceStub) Get(ctx context.Context, id string) (*ordersdomain.Order, error) {
	return s.order, nil
}

// List resolves configured fallback list result values.
func (s shippingOrderQuotationServiceStub) List(ctx context.Context, query ordersapplication.ListQuery) (*ordersapplication.ListResult, error) {
	if s.list != nil {
		return s.list, nil
	}

	return &ordersapplication.ListResult{}, nil
}

// TestShippingOrderQuotationSourceAdapterGetByIDOrIdentifierResolvesCODByPaymentMethod verifies COD amount mapping from payment methods.
func TestShippingOrderQuotationSourceAdapterGetByIDOrIdentifierResolvesCODByPaymentMethod(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name            string
		paymentMethod   string
		expectedCOD     float64
		expectedTotal   float64
		expectedCity    string
		expectedOrderID string
	}{
		{
			name:            "non cod payment method maps cod to zero",
			paymentMethod:   "stripe",
			expectedCOD:     0,
			expectedTotal:   311000,
			expectedCity:    "11001",
			expectedOrderID: "order-1",
		},
		{
			name:            "cod payment method maps cod to order total",
			paymentMethod:   "cod",
			expectedCOD:     311000,
			expectedTotal:   311000,
			expectedCity:    "11001",
			expectedOrderID: "order-1",
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			adapter := shippingOrderQuotationSourceAdapter{
				orders: shippingOrderQuotationServiceStub{
					order: &ordersdomain.Order{
						ID:            "order-1",
						Identifier:    "1024590",
						PaymentMethod: testCase.paymentMethod,
						ShippingAddress: ordersdomain.ShippingAddress{
							CityCode: "11001",
						},
						Items: []ordersdomain.Item{
							{SKU: "sku-1", Quantity: 1, Value: 157000},
							{SKU: "sku-2", Quantity: 1, Value: 154000},
						},
					},
				},
			}

			row, err := adapter.GetByIDOrIdentifier(context.Background(), "1024590")
			if err != nil {
				t.Fatalf("GetByIDOrIdentifier() error = %v", err)
			}
			if row == nil {
				t.Fatal("GetByIDOrIdentifier() returned nil row")
			}
			if row.OrderID != testCase.expectedOrderID {
				t.Fatalf("row.OrderID = %q, want %q", row.OrderID, testCase.expectedOrderID)
			}
			if row.DestCityCode != testCase.expectedCity {
				t.Fatalf("row.DestCityCode = %q, want %q", row.DestCityCode, testCase.expectedCity)
			}
			if row.TotalValue != testCase.expectedTotal {
				t.Fatalf("row.TotalValue = %v, want %v", row.TotalValue, testCase.expectedTotal)
			}
			if row.CollectOnDeliveryAmount != testCase.expectedCOD {
				t.Fatalf("row.CollectOnDeliveryAmount = %v, want %v", row.CollectOnDeliveryAmount, testCase.expectedCOD)
			}
		})
	}
}

// TestResolveOrderCollectOnDeliveryAmount verifies COD amount resolution behavior.
func TestResolveOrderCollectOnDeliveryAmount(t *testing.T) {
	t.Parallel()

	if value := resolveOrderCollectOnDeliveryAmount(157000, "cash on delivery"); value != 157000 {
		t.Fatalf("resolveOrderCollectOnDeliveryAmount() = %v, want 157000", value)
	}
	if value := resolveOrderCollectOnDeliveryAmount(157000, "wompi"); value != 0 {
		t.Fatalf("resolveOrderCollectOnDeliveryAmount() = %v, want 0", value)
	}
	if value := resolveOrderCollectOnDeliveryAmount(0, "cod"); value != 0 {
		t.Fatalf("resolveOrderCollectOnDeliveryAmount() = %v, want 0", value)
	}
}
