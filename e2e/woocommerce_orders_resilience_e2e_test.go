package e2e_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	corecron "mannaiah/module/core/cron"
	"mannaiah/module/woocommerce"
)

// TestWooCommerceOrdersInvalidIntegrationE2E verifies controlled unavailable behavior for invalid WooCommerce integration config.
func TestWooCommerceOrdersInvalidIntegrationE2E(t *testing.T) {
	harness := newContactsE2EHarness(t)
	defer harness.Close(t)

	harness.tracer.Step("initialize woocommerce scheduler")
	scheduler, err := corecron.NewScheduler(corecron.Config{Location: "UTC"}, harness.tracer.logger)
	if err != nil {
		t.Fatalf("corecron.NewScheduler() error = %v", err)
	}

	harness.tracer.Step("initialize woocommerce module with invalid integration config")
	module, err := woocommerce.New(woocommerce.Config{
		SyncOrders:     true,
		SyncOrdersCron: "0 0 * * *",
	}, harness.contactsModule.Service(), harness.ordersModule.Service(), scheduler, harness.tracer.logger)
	if err != nil {
		t.Fatalf("woocommerce.New() error = %v", err)
	}
	module.SetAuthorizer(harness.authModule)
	harness.server.RegisterRoutes(module.RegisterRoutes)
	if err := module.Start(context.Background()); err != nil {
		t.Fatalf("module.Start() error = %v", err)
	}
	defer func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = module.Stop(stopCtx)
	}()

	ordersManageToken := harness.SignToken(t, "orders:manage")

	harness.tracer.Step("trigger manual woocommerce orders sync with invalid integration")
	status, payload := harness.DoJSONRequest(t, http.MethodPost, "/woo/sync/orders", ordersManageToken, nil)
	if status != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", status, http.StatusServiceUnavailable)
	}
	if payload["message"] != "woocommerce_integration_unavailable" {
		t.Fatalf("payload.message = %v, want %q", payload["message"], "woocommerce_integration_unavailable")
	}

	harness.tracer.Step("assert e2e trace logs")
	harness.tracer.AssertStepCount(4)
}

// TestWooCommerceOrdersSyncDisabledE2E verifies controlled disabled behavior when orders sync is turned off.
func TestWooCommerceOrdersSyncDisabledE2E(t *testing.T) {
	harness := newContactsE2EHarness(t)
	defer harness.Close(t)

	harness.tracer.Step("initialize woocommerce module with orders sync disabled")
	module, err := woocommerce.New(woocommerce.Config{
		SyncOrders: false,
	}, harness.contactsModule.Service(), harness.ordersModule.Service(), nil, harness.tracer.logger)
	if err != nil {
		t.Fatalf("woocommerce.New() error = %v", err)
	}
	module.SetAuthorizer(harness.authModule)
	harness.server.RegisterRoutes(module.RegisterRoutes)
	if err := module.Start(context.Background()); err != nil {
		t.Fatalf("module.Start() error = %v", err)
	}
	defer func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = module.Stop(stopCtx)
	}()

	ordersManageToken := harness.SignToken(t, "orders:manage")

	harness.tracer.Step("trigger manual woocommerce orders sync while disabled")
	status, payload := harness.DoJSONRequest(t, http.MethodPost, "/woo/sync/orders", ordersManageToken, nil)
	if status != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", status, http.StatusServiceUnavailable)
	}
	if payload["message"] != "woocommerce_orders_sync_disabled" {
		t.Fatalf("payload.message = %v, want %q", payload["message"], "woocommerce_orders_sync_disabled")
	}

	harness.tracer.Step("assert e2e trace logs")
	harness.tracer.AssertStepCount(3)
}
