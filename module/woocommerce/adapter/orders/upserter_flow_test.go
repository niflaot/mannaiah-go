package orders

import (
	"context"
	"testing"
	"time"

	ordersdomain "mannaiah/module/orders/domain"
	"mannaiah/module/woocommerce/port"
)

// TestUpsertByIdentifierCreate verifies create-path sync behavior.
func TestUpsertByIdentifierCreate(t *testing.T) {
	contactService := newContactServiceMock()
	orderService := newOrdersServiceMock()
	upserter, err := NewUpserter(orderService, contactService)
	if err != nil {
		t.Fatalf("NewUpserter() error = %v", err)
	}

	createdAt := time.Date(2026, time.February, 10, 12, 0, 0, 0, time.UTC)
	outcome, err := upserter.UpsertByIdentifier(context.Background(), port.OrderSyncCommand{
		Identifier: "1001",
		Realm:      "",
		Status:     "processing",
		CreatedAt:  &createdAt,
		Contact: port.ContactSyncCommand{
			Email:     "woo.one@example.com",
			FirstName: "Woo",
			LastName:  "One",
		},
		ShippingAddress: &port.OrderSyncShippingAddress{
			Address:  "Ship Street",
			Address2: "Suite",
			Phone:    "+573001112233",
			CityCode: "05001",
		},
		Items: []port.OrderSyncItem{
			{
				SKU:      "SKU-1",
				Name:     "Blue Shirt",
				Quantity: 2,
				Value:    25000,
			},
			{
				SKU:      "",
				Name:     "Cuota 1/3",
				Quantity: 1,
			},
		},
		Metadata: map[string]string{"integration.source": "woocommerce"},
	})
	if err != nil {
		t.Fatalf("UpsertByIdentifier() error = %v", err)
	}
	if outcome != port.UpsertOutcomeCreated {
		t.Fatalf("outcome = %q, want %q", outcome, port.UpsertOutcomeCreated)
	}
	if len(orderService.createCommands) != 1 {
		t.Fatalf("len(createCommands) = %d, want 1", len(orderService.createCommands))
	}

	command := orderService.createCommands[0]
	if command.Realm != defaultRealm {
		t.Fatalf("createCommand.Realm = %q, want %q", command.Realm, defaultRealm)
	}
	if command.InitialStatus == nil || *command.InitialStatus != ordersdomain.StatusCreated {
		t.Fatalf("createCommand.InitialStatus = %v, want %q", command.InitialStatus, ordersdomain.StatusCreated)
	}
	if command.Author != syncStatusAuthor || command.Description != syncStatusDescription {
		t.Fatalf("createCommand author/description = %q/%q, want %q/%q", command.Author, command.Description, syncStatusAuthor, syncStatusDescription)
	}
	if command.CreatedAt == nil || !command.CreatedAt.UTC().Equal(createdAt) {
		t.Fatalf("createCommand.CreatedAt = %v, want %v", command.CreatedAt, createdAt)
	}
	if len(command.Items) != 2 {
		t.Fatalf("len(createCommand.Items) = %d, want 2", len(command.Items))
	}
	if command.Items[0].SKU != "SKU-1" || command.Items[0].AlternateName != "Blue Shirt" {
		t.Fatalf("createCommand.Items[0] = %+v, want sku and alternateName mapping", command.Items[0])
	}
	if command.Items[1].SKU != "" || command.Items[1].AlternateName != "Cuota 1/3" {
		t.Fatalf("createCommand.Items[1] = %+v, want quota/non-sku item mapping", command.Items[1])
	}
	if command.ShippingAddress == nil || command.ShippingAddress.Address != "Ship Street" {
		t.Fatalf("createCommand.ShippingAddress = %+v, want mapped shipping address", command.ShippingAddress)
	}
}

// TestUpsertByIdentifierUpdate verifies update-path status and comment behavior.
func TestUpsertByIdentifierUpdate(t *testing.T) {
	contactService := newContactServiceMock()
	seedContact(contactService, "contact-1", "woo.one@example.com")

	orderService := newOrdersServiceMock()
	orderService.orders["order-1"] = ordersdomain.Order{
		ID:            "order-1",
		Identifier:    "1001",
		Realm:         defaultRealm,
		ContactID:     "contact-1",
		CurrentStatus: ordersdomain.StatusCreated,
		StatusHistory: []ordersdomain.StatusEntry{
			{Status: ordersdomain.StatusCreated, Author: "system", Description: syncStatusDescription, OccurredAt: time.Date(2026, time.January, 1, 10, 0, 0, 0, time.UTC)},
		},
	}

	upserter, err := NewUpserter(orderService, contactService)
	if err != nil {
		t.Fatalf("NewUpserter() error = %v", err)
	}

	createdAt := time.Date(2026, time.February, 10, 12, 0, 0, 0, time.UTC)
	commentAt := time.Date(2026, time.February, 10, 13, 0, 0, 0, time.UTC)
	outcome, err := upserter.UpsertByIdentifier(context.Background(), port.OrderSyncCommand{
		Identifier: "1001",
		Realm:      defaultRealm,
		Status:     "completed",
		CreatedAt:  &createdAt,
		Contact: port.ContactSyncCommand{
			Email:     "woo.one@example.com",
			FirstName: "Woo",
			LastName:  "One",
		},
		Items: []port.OrderSyncItem{{SKU: "SKU-1", Quantity: 1}},
		Comments: []port.OrderSyncComment{
			{Author: "agent-1", Comment: "Order delivered", OccurredAt: commentAt},
		},
	})
	if err != nil {
		t.Fatalf("UpsertByIdentifier() error = %v", err)
	}
	if outcome != port.UpsertOutcomeUpdated {
		t.Fatalf("outcome = %q, want %q", outcome, port.UpsertOutcomeUpdated)
	}
	if len(orderService.updateStatusCommands) != 1 {
		t.Fatalf("len(updateStatusCommands) = %d, want 1", len(orderService.updateStatusCommands))
	}
	if orderService.updateStatusCommands[0].Status != ordersdomain.StatusCompleted {
		t.Fatalf("first status update = %q, want %q", orderService.updateStatusCommands[0].Status, ordersdomain.StatusCompleted)
	}
	if orderService.updateStatusCommands[0].OccurredAt == nil || !orderService.updateStatusCommands[0].OccurredAt.UTC().Equal(createdAt) {
		t.Fatalf("first status occurredAt = %v, want %v", orderService.updateStatusCommands[0].OccurredAt, createdAt)
	}
	if len(orderService.addCommentCommands) != 1 {
		t.Fatalf("len(addCommentCommands) = %d, want 1", len(orderService.addCommentCommands))
	}
	if orderService.addCommentCommands[0].Author != "agent-1" {
		t.Fatalf("comment author = %q, want %q", orderService.addCommentCommands[0].Author, "agent-1")
	}
	if orderService.addCommentCommands[0].Comment != "Order delivered" {
		t.Fatalf("comment text = %q, want %q", orderService.addCommentCommands[0].Comment, "Order delivered")
	}
}

// TestUpsertByIdentifierUnchanged verifies unchanged-path behavior.
func TestUpsertByIdentifierUnchanged(t *testing.T) {
	contactService := newContactServiceMock()
	seedContact(contactService, "contact-1", "woo.one@example.com")

	commentAt := time.Date(2026, time.February, 10, 13, 0, 0, 0, time.UTC)
	orderService := newOrdersServiceMock()
	orderService.orders["order-1"] = ordersdomain.Order{
		ID:            "order-1",
		Identifier:    "1001",
		Realm:         defaultRealm,
		ContactID:     "contact-1",
		CurrentStatus: ordersdomain.StatusCompleted,
		StatusHistory: []ordersdomain.StatusEntry{
			{Status: ordersdomain.StatusCompleted, Author: syncStatusAuthor, Description: syncStatusDescription, OccurredAt: time.Date(2026, time.February, 10, 12, 0, 0, 0, time.UTC)},
		},
		Comments: []ordersdomain.Comment{
			{
				Author:     "agent-1",
				Comment:    "Order delivered",
				Internal:   false,
				OccurredAt: commentAt,
			},
		},
	}

	upserter, err := NewUpserter(orderService, contactService)
	if err != nil {
		t.Fatalf("NewUpserter() error = %v", err)
	}

	outcome, err := upserter.UpsertByIdentifier(context.Background(), port.OrderSyncCommand{
		Identifier: "1001",
		Realm:      defaultRealm,
		Status:     "completed",
		Contact: port.ContactSyncCommand{
			Email:     "woo.one@example.com",
			FirstName: "Woo",
			LastName:  "One",
		},
		Items: []port.OrderSyncItem{{SKU: "SKU-1", Quantity: 1}},
		Comments: []port.OrderSyncComment{
			{Author: "agent-1", Comment: "Order delivered", OccurredAt: commentAt},
		},
	})
	if err != nil {
		t.Fatalf("UpsertByIdentifier() error = %v", err)
	}
	if outcome != port.UpsertOutcomeUnchanged {
		t.Fatalf("outcome = %q, want %q", outcome, port.UpsertOutcomeUnchanged)
	}
	if len(orderService.updateStatusCommands) != 0 {
		t.Fatalf("expected no status updates on unchanged sync")
	}
	if len(orderService.addCommentCommands) != 0 {
		t.Fatalf("expected no comment updates on unchanged sync")
	}
}

// TestUpsertByIdentifierUpdateIgnoredByOrderService verifies unchanged outcomes when Woo-origin updates are ignored by order policies.
func TestUpsertByIdentifierUpdateIgnoredByOrderService(t *testing.T) {
	contactService := newContactServiceMock()
	seedContact(contactService, "contact-1", "woo.one@example.com")

	commentAt := time.Date(2026, time.February, 10, 13, 0, 0, 0, time.UTC)
	orderService := newOrdersServiceMock()
	orderService.ignoreWooSourceMutations = true
	orderService.orders["order-1"] = ordersdomain.Order{
		ID:            "order-1",
		Identifier:    "1001",
		Realm:         defaultRealm,
		ContactID:     "contact-1",
		CurrentStatus: ordersdomain.StatusCreated,
		StatusHistory: []ordersdomain.StatusEntry{
			{Status: ordersdomain.StatusCreated, Author: syncStatusAuthor, Description: syncStatusDescription, OccurredAt: time.Date(2026, time.February, 10, 12, 0, 0, 0, time.UTC)},
		},
	}

	upserter, err := NewUpserter(orderService, contactService)
	if err != nil {
		t.Fatalf("NewUpserter() error = %v", err)
	}

	outcome, err := upserter.UpsertByIdentifier(context.Background(), port.OrderSyncCommand{
		Identifier: "1001",
		Realm:      defaultRealm,
		Status:     "completed",
		Contact: port.ContactSyncCommand{
			Email:     "woo.one@example.com",
			FirstName: "Woo",
			LastName:  "One",
		},
		Items: []port.OrderSyncItem{{SKU: "SKU-1", Quantity: 1}},
		Comments: []port.OrderSyncComment{
			{Author: "agent-1", Comment: "Order delivered", OccurredAt: commentAt},
		},
	})
	if err != nil {
		t.Fatalf("UpsertByIdentifier() error = %v", err)
	}
	if outcome != port.UpsertOutcomeUnchanged {
		t.Fatalf("outcome = %q, want %q", outcome, port.UpsertOutcomeUnchanged)
	}
	if len(orderService.updateStatusCommands) != 1 {
		t.Fatalf("len(updateStatusCommands) = %d, want 1", len(orderService.updateStatusCommands))
	}
	if len(orderService.addCommentCommands) != 1 {
		t.Fatalf("len(addCommentCommands) = %d, want 1", len(orderService.addCommentCommands))
	}
}

// TestUpsertByIdentifierCreateSyncsCouponUsage verifies Woo order coupon usage backfill behavior.
func TestUpsertByIdentifierCreateSyncsCouponUsage(t *testing.T) {
	contactService := newContactServiceMock()
	orderService := newOrdersServiceMock()
	couponService := &couponUsageSyncServiceMock{}
	upserter, err := NewUpserter(orderService, contactService)
	if err != nil {
		t.Fatalf("NewUpserter() error = %v", err)
	}
	upserter.SetCouponUsageSyncService(couponService)

	createdAt := time.Date(2026, time.April, 13, 15, 0, 0, 0, time.UTC)
	_, err = upserter.UpsertByIdentifier(context.Background(), port.OrderSyncCommand{
		Identifier: "1002",
		Realm:      defaultRealm,
		Status:     "processing",
		CreatedAt:  &createdAt,
		Contact: port.ContactSyncCommand{
			Email:     "woo.two@example.com",
			FirstName: "Woo",
			LastName:  "Two",
		},
		Items: []port.OrderSyncItem{{SKU: "SKU-2", Quantity: 1}},
		AppliedCoupons: []port.OrderSyncAppliedCoupon{{
			Code:     "WELCOME10",
			Discount: "15000",
		}},
	})
	if err != nil {
		t.Fatalf("UpsertByIdentifier() error = %v", err)
	}
	if len(couponService.commands) != 1 {
		t.Fatalf("len(couponUsageCommands) = %d, want 1", len(couponService.commands))
	}
	if couponService.commands[0].Code != "WELCOME10" {
		t.Fatalf("coupon usage code = %q, want %q", couponService.commands[0].Code, "WELCOME10")
	}
	if couponService.commands[0].OrderID != "order-1002" {
		t.Fatalf("coupon usage order id = %q, want %q", couponService.commands[0].OrderID, "order-1002")
	}
	if couponService.commands[0].Email != "woo.two@example.com" {
		t.Fatalf("coupon usage email = %q, want %q", couponService.commands[0].Email, "woo.two@example.com")
	}
	if couponService.commands[0].UsedAt == nil || !couponService.commands[0].UsedAt.UTC().Equal(createdAt) {
		t.Fatalf("coupon usage usedAt = %v, want %v", couponService.commands[0].UsedAt, createdAt)
	}
}
