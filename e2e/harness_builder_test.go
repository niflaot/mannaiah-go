package e2e_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"strings"
	"testing"
	"time"

	"mannaiah/module/assets"
	assetevent "mannaiah/module/assets/adapter/event"
	assetsapplication "mannaiah/module/assets/application"
	"mannaiah/module/auth"
	"mannaiah/module/contacts"
	contactevent "mannaiah/module/contacts/adapter/event"
	contactapplication "mannaiah/module/contacts/application"
	coredatabase "mannaiah/module/core/database"
	coredatabasemigration "mannaiah/module/core/database/migration"
	corehttp "mannaiah/module/core/http"
	coremsgplatform "mannaiah/module/core/messaging/platform"
	corewatermill "mannaiah/module/core/messaging/watermill"
	"mannaiah/module/orders"
	ordercontacts "mannaiah/module/orders/adapter/contacts"
	orderevent "mannaiah/module/orders/adapter/event"
	orderproducts "mannaiah/module/orders/adapter/products"
	"mannaiah/module/products"
)

// newContactsE2EHarness creates a fully wired contacts/auth/event E2E runtime harness.
func newContactsE2EHarness(t *testing.T) *contactsE2EHarness {
	t.Helper()

	tracer := newStepTracer(t)
	tracer.Step("generate jwt signing key")
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey() error = %v", err)
	}

	tracer.Step("start jwks server")
	jwksServer := newJWKSServer(t, key.PublicKey)

	tracer.Step("initialize auth module")
	authModule, err := auth.New(auth.Config{
		Issuer:                 strings.TrimSuffix(jwksServer.URL, e2eIssuerSuffix),
		Audience:               e2eAudience,
		JWKSRateLimitPerMinute: 5,
		JWKSCacheTTLMS:         60000,
		JWKSHTTPTimeoutMS:      2000,
	}, "production", tracer.logger)
	if err != nil {
		t.Fatalf("auth.New() error = %v", err)
	}

	tracer.Step("open sqlite database")
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := coredatabase.Open(coredatabase.Config{
		Driver:       "sqlite",
		DSN:          dsn,
		MaxOpenConns: 1,
	}, tracer.logger)
	if err != nil {
		t.Fatalf("coredatabase.Open() error = %v", err)
	}
	if err := coredatabasemigration.Apply(context.Background(), db, coredatabasemigration.Config{Enabled: true, Driver: "sqlite", Table: "schema_migrations"}, tracer.logger); err != nil {
		t.Fatalf("coredatabasemigration.Apply() error = %v", err)
	}

	tracer.Step("initialize in-memory messaging platform")
	messaging, err := corewatermill.NewInMemoryPlatform(coremsgplatform.Config{}, tracer.logger)
	if err != nil {
		t.Fatalf("corewatermill.NewInMemoryPlatform() error = %v", err)
	}

	createdEvents := make(chan contactEventRecord, harnessEventBufferSize)
	updatedEvents := make(chan contactEventRecord, harnessEventBufferSize)
	assetCreatedEvents := make(chan contactEventRecord, harnessEventBufferSize)
	assetUpdatedEvents := make(chan contactEventRecord, harnessEventBufferSize)
	assetDeletedEvents := make(chan contactEventRecord, harnessEventBufferSize)

	tracer.Step("register event listeners")
	registerContactTopicHandler(t, messaging, contactapplication.TopicContactCreated, createdEvents)
	registerContactTopicHandler(t, messaging, contactapplication.TopicContactUpdated, updatedEvents)
	registerContactTopicHandler(t, messaging, assetsapplication.TopicAssetCreated, assetCreatedEvents)
	registerContactTopicHandler(t, messaging, assetsapplication.TopicAssetUpdated, assetUpdatedEvents)
	registerContactTopicHandler(t, messaging, assetsapplication.TopicAssetDeleted, assetDeletedEvents)

	messagingCtx, messagingCancel := context.WithCancel(context.Background())
	messagingErrs := make(chan error, 1)

	tracer.Step("run messaging router")
	go func() {
		messagingErrs <- messaging.Run(messagingCtx)
	}()

	select {
	case <-messaging.Running():
	case <-time.After(2 * time.Second):
		t.Fatalf("messaging router did not start")
	}

	tracer.Step("initialize contacts module")
	publisher, err := contactevent.NewPublisher(messaging.Publisher())
	if err != nil {
		t.Fatalf("contactevent.NewPublisher() error = %v", err)
	}
	assetPublisher, err := assetevent.NewPublisher(messaging.Publisher())
	if err != nil {
		t.Fatalf("assetevent.NewPublisher() error = %v", err)
	}
	orderPublisher, err := orderevent.NewPublisher(messaging.Publisher())
	if err != nil {
		t.Fatalf("orderevent.NewPublisher() error = %v", err)
	}

	contactsModule, err := contacts.New(db, publisher)
	if err != nil {
		t.Fatalf("contacts.New() error = %v", err)
	}
	contactsModule.SetAuthorizer(authModule)

	tracer.Step("initialize assets module")
	assetsModule, err := assets.New(db, newInMemoryAssetStorage(), assetPublisher)
	if err != nil {
		t.Fatalf("assets.New() error = %v", err)
	}
	assetsModule.SetAuthorizer(authModule)

	tracer.Step("initialize products module")
	productsModule, err := products.New(db, assetsModule.Service())
	if err != nil {
		t.Fatalf("products.New() error = %v", err)
	}
	productsModule.SetAuthorizer(authModule)

	tracer.Step("initialize orders module")
	orderCustomerSource, err := ordercontacts.NewSource(contactsModule.Service())
	if err != nil {
		t.Fatalf("ordercontacts.NewSource() error = %v", err)
	}
	orderProductResolver, err := orderproducts.NewResolver(db)
	if err != nil {
		t.Fatalf("orderproducts.NewResolver() error = %v", err)
	}
	ordersModule, err := orders.NewWithPublisher(db, orderCustomerSource, orderPublisher, orderProductResolver)
	if err != nil {
		t.Fatalf("orders.New() error = %v", err)
	}
	ordersModule.SetAuthorizer(authModule)

	tracer.Step("initialize http server")
	server, err := corehttp.New(corehttp.Config{Host: "127.0.0.1", Port: 8011}, tracer.logger)
	if err != nil {
		t.Fatalf("corehttp.New() error = %v", err)
	}
	server.RegisterRoutes(contactsModule.RegisterRoutes)
	server.RegisterRoutes(authModule.RegisterRoutes)
	server.RegisterRoutes(assetsModule.RegisterRoutes)
	server.RegisterRoutes(productsModule.RegisterRoutes)
	server.RegisterRoutes(ordersModule.RegisterRoutes)

	return &contactsE2EHarness{
		tracer:             tracer,
		key:                key,
		jwksServer:         jwksServer,
		authModule:         authModule,
		db:                 db,
		messaging:          messaging,
		messagingCancel:    messagingCancel,
		messagingErrs:      messagingErrs,
		server:             server,
		contactsModule:     contactsModule,
		assetsModule:       assetsModule,
		productsModule:     productsModule,
		ordersModule:       ordersModule,
		createdEvents:      createdEvents,
		updatedEvents:      updatedEvents,
		assetCreatedEvents: assetCreatedEvents,
		assetUpdatedEvents: assetUpdatedEvents,
		assetDeletedEvents: assetDeletedEvents,
	}
}
