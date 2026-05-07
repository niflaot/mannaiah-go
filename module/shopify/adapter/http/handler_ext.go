package http

import (
	"context"
	"strings"
	"time"

	contactsdomain "mannaiah/module/contacts/domain"
	corehttp "mannaiah/module/core/http"
	shopifycontactservice "mannaiah/module/shopify/application/contact/service"
	shopifyorderservice "mannaiah/module/shopify/application/order/service"
	shopifyport "mannaiah/module/shopify/port"
)

const extensionOrigin = "https://admin.shopify.com"

// ExtensionOrderSummary defines Shopify Admin extension order-summary payload values.
type ExtensionOrderSummary struct {
	Linked       bool       `json:"linked"`
	MannaiahID   string     `json:"mannaiahId,omitempty"`
	CreatedAt    *time.Time `json:"createdAt,omitempty"`
	Status       string     `json:"status,omitempty"`
	ContactName  string     `json:"contactName,omitempty"`
	LastSyncedAt *time.Time `json:"lastSyncedAt,omitempty"`
}

// ExtensionContactSummary defines Shopify Admin extension contact-summary payload values.
type ExtensionContactSummary struct {
	Linked       bool       `json:"linked"`
	MannaiahID   string     `json:"mannaiahId,omitempty"`
	DisplayName  string     `json:"displayName,omitempty"`
	CreatedAt    *time.Time `json:"createdAt,omitempty"`
	LastSyncedAt *time.Time `json:"lastSyncedAt,omitempty"`
}

func (h *Handler) handleExtensionOptions(ctx corehttp.Context) error {
	setExtensionCORSHeaders(ctx)
	return ctx.SendStatus(204)
}

func (h *Handler) protectExtension(next corehttp.Handler) corehttp.Handler {
	return func(ctx corehttp.Context) error {
		setExtensionCORSHeaders(ctx)
		shopDomain, err := authenticateSessionToken(ctx.Context(), ctx.GetHeader("Authorization"), h.clientSecret, h.installationResolver)
		if err != nil {
			return h.mapError(err)
		}
		ctx.Locals(extensionShopDomainLocal, shopDomain)
		return next(ctx)
	}
}

func (h *Handler) getExtensionOrder(ctx corehttp.Context) error {
	setExtensionCORSHeaders(ctx)
	shopifyOrderID := strings.TrimSpace(ctx.Params("shopifyOrderId", ""))
	if shopifyOrderID == "" {
		return h.mapError(shopifyorderservice.ErrInvalidOrderID)
	}
	requestCtx := extensionRequestContext(ctx)
	shopDomain := extensionShopDomain(ctx)

	link, err := h.links.GetLinkByShopifyID(requestCtx, shopifyport.SyncKindOrder, shopDomain, shopifyOrderID)
	if err != nil {
		return h.mapError(err)
	}
	if link == nil {
		return ctx.Status(200).JSON(ExtensionOrderSummary{Linked: false})
	}

	order, err := h.ordersLookup.Get(requestCtx, link.MannaiahID)
	if err != nil {
		return h.mapError(err)
	}
	contactName := ""
	if strings.TrimSpace(order.ContactID) != "" {
		contact, contactErr := h.contactsLookup.Get(requestCtx, order.ContactID)
		if contactErr == nil && contact != nil {
			contactName = buildContactDisplayName(*contact)
		}
	}

	return ctx.Status(200).JSON(ExtensionOrderSummary{
		Linked:       true,
		MannaiahID:   strings.TrimSpace(order.ID),
		CreatedAt:    extensionTimePointer(order.CreatedAt),
		Status:       strings.TrimSpace(string(order.CurrentStatus)),
		ContactName:  contactName,
		LastSyncedAt: link.LastSyncedAt,
	})
}

func (h *Handler) getExtensionContact(ctx corehttp.Context) error {
	setExtensionCORSHeaders(ctx)
	shopifyCustomerID := strings.TrimSpace(ctx.Params("shopifyCustomerId", ""))
	if shopifyCustomerID == "" {
		return h.mapError(shopifycontactservice.ErrInvalidCustomerID)
	}
	requestCtx := extensionRequestContext(ctx)
	shopDomain := extensionShopDomain(ctx)

	link, err := h.links.GetLinkByShopifyID(requestCtx, shopifyport.SyncKindContact, shopDomain, shopifyCustomerID)
	if err != nil {
		return h.mapError(err)
	}
	if link == nil {
		return ctx.Status(200).JSON(ExtensionContactSummary{Linked: false})
	}

	contact, err := h.contactsLookup.Get(requestCtx, link.MannaiahID)
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(ExtensionContactSummary{
		Linked:       true,
		MannaiahID:   strings.TrimSpace(contact.ID),
		DisplayName:  buildContactDisplayName(*contact),
		CreatedAt:    extensionTimePointer(contact.CreatedAt),
		LastSyncedAt: link.LastSyncedAt,
	})
}

func extensionRequestContext(ctx corehttp.Context) context.Context {
	return shopifyport.WithShopDomain(ctx.Context(), extensionShopDomain(ctx))
}

func extensionShopDomain(ctx corehttp.Context) string {
	value, _ := ctx.Locals(extensionShopDomainLocal).(string)
	return shopifyport.NormalizeShopDomain(value)
}

func setExtensionCORSHeaders(ctx corehttp.Context) {
	ctx.SetHeader("Access-Control-Allow-Origin", extensionOrigin)
	ctx.SetHeader("Access-Control-Allow-Headers", "Authorization, Content-Type")
	ctx.SetHeader("Access-Control-Allow-Methods", "GET, OPTIONS")
	ctx.SetHeader("Vary", "Origin")
}

func buildContactDisplayName(contact contactsdomain.Contact) string {
	if strings.TrimSpace(contact.LegalName) != "" {
		return strings.TrimSpace(contact.LegalName)
	}
	fullName := strings.TrimSpace(strings.TrimSpace(contact.FirstName) + " " + strings.TrimSpace(contact.LastName))
	if fullName != "" {
		return fullName
	}

	return strings.TrimSpace(contact.Email)
}

func extensionTimePointer(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}
	resolvedValue := value.UTC()
	return &resolvedValue
}
