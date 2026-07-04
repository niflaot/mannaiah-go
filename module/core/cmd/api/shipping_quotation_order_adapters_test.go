package main

import (
	"context"
	"testing"

	contactdomain "mannaiah/module/contacts/domain"
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

// shippingOrderQuotationContactServiceStub defines contact lookup behavior for shipping quotation order adapter tests.
type shippingOrderQuotationContactServiceStub struct {
	// contact defines returned contact values.
	contact *contactdomain.Contact
}

// Get resolves one configured contact value.
func (s shippingOrderQuotationContactServiceStub) Get(ctx context.Context, id string) (*contactdomain.Contact, error) {
	return s.contact, nil
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
			name:            "cod payment method maps cod to full payable order total",
			paymentMethod:   "cod",
			expectedCOD:     326000,
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
						ShippingCharges: []ordersdomain.ShippingCharge{
							{MethodID: "envio", MethodTitle: "Envio", Price: 15000},
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

// TestCalculateOrderMonetaryTotals verifies item totals remain separate from payable order totals.
func TestCalculateOrderMonetaryTotals(t *testing.T) {
	t.Parallel()

	itemTotalValue, orderGrandTotal := calculateOrderMonetaryTotals(&ordersdomain.Order{
		Items: []ordersdomain.Item{
			{SKU: "sku-1", Quantity: 1, Value: 137000},
			{SKU: "sku-2", Quantity: 2, Value: 10000},
		},
		ShippingCharges: []ordersdomain.ShippingCharge{
			{MethodID: "envio", MethodTitle: "Envio", Price: 15000},
		},
	})
	if itemTotalValue != 157000 {
		t.Fatalf("itemTotalValue = %v, want 157000", itemTotalValue)
	}
	if orderGrandTotal != 172000 {
		t.Fatalf("orderGrandTotal = %v, want 172000", orderGrandTotal)
	}
}

// TestCalculateOrderMonetaryTotalsUsesFinalDiscountedShopifyTotal verifies COD totals use final payable order values.
func TestCalculateOrderMonetaryTotalsUsesFinalDiscountedShopifyTotal(t *testing.T) {
	t.Parallel()

	itemTotalValue, orderGrandTotal := calculateOrderMonetaryTotals(&ordersdomain.Order{
		Items: []ordersdomain.Item{
			{SKU: "journey-croma", Quantity: 1, Value: 145000},
			{SKU: "neceser-gift", Quantity: 1, Value: 45000},
		},
		ShippingCharges: []ordersdomain.ShippingCharge{
			{MethodID: "flat", MethodTitle: "Envios a todo Colombia", Price: 10000},
		},
		AppliedCoupons: []ordersdomain.AppliedCoupon{
			{Code: "SQUID_FGFGIGX", DiscountType: "fixed", DiscountAmount: 45000},
		},
		Metadata: map[string]string{"shopify_total_price": "155000"},
	})
	if itemTotalValue != 145000 {
		t.Fatalf("itemTotalValue = %v, want 145000", itemTotalValue)
	}
	if orderGrandTotal != 155000 {
		t.Fatalf("orderGrandTotal = %v, want 155000", orderGrandTotal)
	}
}

// TestCalculateOrderMonetaryTotalsSubtractsAppliedCoupons verifies non-Shopify rows still account for stored discounts.
func TestCalculateOrderMonetaryTotalsSubtractsAppliedCoupons(t *testing.T) {
	t.Parallel()

	itemTotalValue, orderGrandTotal := calculateOrderMonetaryTotals(&ordersdomain.Order{
		Items: []ordersdomain.Item{
			{SKU: "journey-croma", Quantity: 1, Value: 145000},
			{SKU: "neceser-gift", Quantity: 1, Value: 45000},
		},
		ShippingCharges: []ordersdomain.ShippingCharge{
			{MethodID: "flat", MethodTitle: "Envios a todo Colombia", Price: 10000},
		},
		AppliedCoupons: []ordersdomain.AppliedCoupon{
			{Code: "SQUID_FGFGIGX", DiscountType: "fixed", DiscountAmount: 45000},
		},
	})
	if itemTotalValue != 145000 {
		t.Fatalf("itemTotalValue = %v, want 145000", itemTotalValue)
	}
	if orderGrandTotal != 155000 {
		t.Fatalf("orderGrandTotal = %v, want 155000", orderGrandTotal)
	}
}

// TestShippingOrderQuotationSourceAdapterGetByIDOrIdentifierRecipientEnrichment verifies recipient fields are resolved from contact and shipping data.
func TestShippingOrderQuotationSourceAdapterGetByIDOrIdentifierRecipientEnrichment(t *testing.T) {
	t.Parallel()

	adapter := shippingOrderQuotationSourceAdapter{
		orders: shippingOrderQuotationServiceStub{
			order: &ordersdomain.Order{
				ID:            "order-1",
				Identifier:    "601205",
				ContactID:     "contact-1",
				PaymentMethod: "stripe",
				ShippingAddress: ordersdomain.ShippingAddress{
					Address:  "Calle 18 Sur # 24d - 46",
					Address2: "Piso 2",
					Phone:    "3057901484",
					CityCode: "11001",
				},
				Items: []ordersdomain.Item{{SKU: "sku-1", Quantity: 1, Value: 144000}},
			},
		},
		contacts: shippingOrderQuotationContactServiceStub{
			contact: &contactdomain.Contact{
				ID:             "contact-1",
				DocumentType:   contactdomain.DocumentTypeCC,
				DocumentNumber: "123456789",
				FirstName:      "Ian",
				LastName:       "Castaño",
				Email:          "coccostoreco@gmail.com",
				Phone:          "3001112233",
				Address:        "Fallback Address 1",
				AddressExtra:   "Fallback Address 2",
				CityCode:       "05001",
			},
		},
	}

	row, err := adapter.GetByIDOrIdentifier(context.Background(), "601205")
	if err != nil {
		t.Fatalf("GetByIDOrIdentifier() error = %v", err)
	}
	if row == nil {
		t.Fatal("GetByIDOrIdentifier() returned nil row")
	}
	if row.RecipientName != "Ian Castaño" {
		t.Fatalf("row.RecipientName = %q, want %q", row.RecipientName, "Ian Castaño")
	}
	if row.RecipientIDType != "CC" {
		t.Fatalf("row.RecipientIDType = %q, want %q", row.RecipientIDType, "CC")
	}
	if row.RecipientID != "123456789" {
		t.Fatalf("row.RecipientID = %q, want %q", row.RecipientID, "123456789")
	}
	if row.RecipientEmail != "coccostoreco@gmail.com" {
		t.Fatalf("row.RecipientEmail = %q, want %q", row.RecipientEmail, "coccostoreco@gmail.com")
	}
	if row.RecipientAddressLine != "Calle 18 Sur # 24d - 46" {
		t.Fatalf("row.RecipientAddressLine = %q, want shipping address", row.RecipientAddressLine)
	}
	if row.RecipientAddressLine2 != "Piso 2" {
		t.Fatalf("row.RecipientAddressLine2 = %q, want %q", row.RecipientAddressLine2, "Piso 2")
	}
	if row.RecipientPhone != "3057901484" {
		t.Fatalf("row.RecipientPhone = %q, want shipping phone", row.RecipientPhone)
	}
	if row.DestCityCode != "11001" {
		t.Fatalf("row.DestCityCode = %q, want shipping city", row.DestCityCode)
	}
	if row.RecipientCity != "11001" {
		t.Fatalf("row.RecipientCity = %q, want shipping city", row.RecipientCity)
	}
}

// TestShippingOrderQuotationSourceAdapterGetByIDOrIdentifierRecipientShippingFallback verifies recipient shipping fields fallback to contact when shipping fields are empty.
func TestShippingOrderQuotationSourceAdapterGetByIDOrIdentifierRecipientShippingFallback(t *testing.T) {
	t.Parallel()

	adapter := shippingOrderQuotationSourceAdapter{
		orders: shippingOrderQuotationServiceStub{
			order: &ordersdomain.Order{
				ID:            "order-2",
				Identifier:    "601206",
				ContactID:     "contact-2",
				PaymentMethod: "stripe",
				ShippingAddress: ordersdomain.ShippingAddress{
					Address:  "",
					Address2: "",
					Phone:    "",
					CityCode: "",
				},
				Items: []ordersdomain.Item{{SKU: "sku-1", Quantity: 1, Value: 144000}},
			},
		},
		contacts: shippingOrderQuotationContactServiceStub{
			contact: &contactdomain.Contact{
				ID:           "contact-2",
				FirstName:    "Ana",
				LastName:     "Rojas",
				Email:        "ana@example.com",
				Phone:        "3009991122",
				Address:      "Carrera 1 # 2-3",
				AddressExtra: "Apto 4",
				CityCode:     "05001",
			},
		},
	}

	row, err := adapter.GetByIDOrIdentifier(context.Background(), "601206")
	if err != nil {
		t.Fatalf("GetByIDOrIdentifier() error = %v", err)
	}
	if row == nil {
		t.Fatal("GetByIDOrIdentifier() returned nil row")
	}
	if row.RecipientAddressLine != "Carrera 1 # 2-3" {
		t.Fatalf("row.RecipientAddressLine = %q, want contact fallback address", row.RecipientAddressLine)
	}
	if row.RecipientAddressLine2 != "Apto 4" {
		t.Fatalf("row.RecipientAddressLine2 = %q, want contact fallback address 2", row.RecipientAddressLine2)
	}
	if row.RecipientPhone != "3009991122" {
		t.Fatalf("row.RecipientPhone = %q, want contact fallback phone", row.RecipientPhone)
	}
	if row.DestCityCode != "05001" {
		t.Fatalf("row.DestCityCode = %q, want contact fallback city", row.DestCityCode)
	}
	if row.RecipientCity != "05001" {
		t.Fatalf("row.RecipientCity = %q, want contact fallback city", row.RecipientCity)
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
