package port

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrCustomerNotFound is returned when a Shopify customer cannot be resolved.
	ErrCustomerNotFound = errors.New("shopify customer not found")
	// ErrOrderNotFound is returned when a Shopify order cannot be resolved.
	ErrOrderNotFound = errors.New("shopify order not found")
)

// CustomerSource defines Shopify customer retrieval behavior.
type CustomerSource interface {
	// Validate verifies source connectivity and credentials.
	Validate(ctx context.Context) error
	// GetCustomer resolves one Shopify customer by identifier.
	GetCustomer(ctx context.Context, id string) (ShopifyCustomer, error)
	// FindCustomerByEmail resolves one Shopify customer by email address.
	FindCustomerByEmail(ctx context.Context, email string) (ShopifyCustomer, error)
	// ListCustomers returns up to limit customers with IDs greater than sinceID.
	// Pass empty sinceID to start from the beginning. hasMore is true when there may be additional pages.
	ListCustomers(ctx context.Context, sinceID string, limit int) (customers []ShopifyCustomer, hasMore bool, err error)
}

// OrderSource defines Shopify order retrieval behavior.
type OrderSource interface {
	// Validate verifies source connectivity and credentials.
	Validate(ctx context.Context) error
	// GetOrder resolves one Shopify order by identifier.
	GetOrder(ctx context.Context, id string) (ShopifyOrder, error)
	// ListOrders returns up to limit orders with IDs greater than sinceID.
	// Pass empty sinceID to start from the beginning. hasMore is true when there may be additional pages.
	ListOrders(ctx context.Context, sinceID string, limit int) (orders []ShopifyOrder, hasMore bool, err error)
}

// ShopifyNoteAttribute defines one normalized Shopify note attribute.
type ShopifyNoteAttribute struct {
	// Name defines note attribute names.
	Name string
	// Value defines note attribute values.
	Value string
}

// ShopifyAddress defines one normalized Shopify address payload.
type ShopifyAddress struct {
	// FirstName defines address first-name values.
	FirstName string
	// LastName defines address last-name values.
	LastName string
	// Company defines company values.
	Company string
	// Address1 defines address line 1 values.
	Address1 string
	// Address2 defines address line 2 values.
	Address2 string
	// City defines city values.
	City string
	// Province defines province/state values.
	Province string
	// Country defines country values.
	Country string
	// Zip defines postal-code values.
	Zip string
	// Phone defines phone values.
	Phone string
}

// ShopifyCustomer defines one normalized Shopify customer payload.
type ShopifyCustomer struct {
	// ShopDomain defines the Shopify store domain that owns this customer.
	ShopDomain string
	// ID defines Shopify customer identifiers.
	ID string
	// Email defines customer email values.
	Email string
	// FirstName defines customer first-name values.
	FirstName string
	// LastName defines customer last-name values.
	LastName string
	// Phone defines customer phone values.
	Phone string
	// Tags defines customer tag values.
	Tags string
	// Note defines customer note values.
	Note string
	// DefaultAddress defines optional default address values.
	DefaultAddress *ShopifyAddress
	// NoteAttributes defines structured note attribute values.
	NoteAttributes []ShopifyNoteAttribute
	// EmailMarketingState defines Shopify email marketing consent states.
	EmailMarketingState string
	// EmailMarketingConsentUpdatedAt defines Shopify email marketing consent timestamps.
	EmailMarketingConsentUpdatedAt *time.Time
	// SMSMarketingState defines Shopify SMS marketing consent states.
	SMSMarketingState string
	// SMSMarketingConsentUpdatedAt defines Shopify SMS marketing consent timestamps.
	SMSMarketingConsentUpdatedAt *time.Time
	// CreatedAt defines source creation timestamps.
	CreatedAt time.Time
}

// ShopifyLineItem defines one normalized Shopify order line item.
type ShopifyLineItem struct {
	// ID defines Shopify line-item identifiers.
	ID string
	// SKU defines SKU values.
	SKU string
	// Title defines product-title values.
	Title string
	// VariantTitle defines variant-title values.
	VariantTitle string
	// ProductID defines Shopify product identifiers.
	ProductID string
	// VariantID defines Shopify variant identifiers.
	VariantID string
	// MannaiahProductID defines explicit Mannaiah product identifiers from Shopify line-item properties.
	MannaiahProductID string
	// Quantity defines ordered quantity values.
	Quantity int
	// Price defines unit-price values.
	Price string
}

// ShopifyShippingLine defines one normalized Shopify shipping line.
type ShopifyShippingLine struct {
	// Code defines shipping code values.
	Code string
	// Title defines shipping title values.
	Title string
	// Price defines shipping price values.
	Price string
}

// ShopifyDiscountCode defines one normalized Shopify discount code.
type ShopifyDiscountCode struct {
	// Code defines discount code values.
	Code string
	// Amount defines discount amount values.
	Amount string
	// Type defines discount type values.
	Type string
}

// ShopifyOrder defines one normalized Shopify order payload.
type ShopifyOrder struct {
	// ShopDomain defines the Shopify store domain that owns this order.
	ShopDomain string
	// ID defines Shopify order identifiers.
	ID string
	// Name defines public Shopify order names.
	Name string
	// ContactEmail defines order email values.
	ContactEmail string
	// FinancialStatus defines Shopify financial-status values.
	FinancialStatus string
	// FulfillmentStatus defines Shopify fulfillment-status values.
	FulfillmentStatus string
	// Currency defines order currency values.
	Currency string
	// TotalPrice defines total order amount values.
	TotalPrice string
	// TotalTax defines total tax amount values.
	TotalTax string
	// TotalDiscounts defines total discount amount values.
	TotalDiscounts string
	// Note defines current order note values.
	Note string
	// Tags defines current order tags.
	Tags string
	// CancelReason defines cancellation reason values.
	CancelReason string
	// CancelledAt defines optional cancellation timestamps.
	CancelledAt *time.Time
	// PaymentGatewayNames defines payment gateway names.
	PaymentGatewayNames []string
	// Customer defines optional customer values.
	Customer *ShopifyCustomer
	// BillingAddress defines billing-address values.
	BillingAddress *ShopifyAddress
	// ShippingAddress defines shipping-address values.
	ShippingAddress *ShopifyAddress
	// NoteAttributes defines structured note attributes.
	NoteAttributes []ShopifyNoteAttribute
	// LineItems defines order item values.
	LineItems []ShopifyLineItem
	// ShippingLines defines shipping-charge values.
	ShippingLines []ShopifyShippingLine
	// DiscountCodes defines discount-code values.
	DiscountCodes []ShopifyDiscountCode
	// CreatedAt defines source creation timestamps.
	CreatedAt time.Time
}
