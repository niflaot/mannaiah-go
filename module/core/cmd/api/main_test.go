package main

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"gorm.io/gorm"
	coredatabase "mannaiah/module/core/database"
	corehttp "mannaiah/module/core/http"
	coremsgplatform "mannaiah/module/core/messaging/platform"
	corewatermill "mannaiah/module/core/messaging/watermill"
)

// TestRegisterCoreStatusRoute verifies status route registration behavior.
func TestRegisterCoreStatusRoute(t *testing.T) {
	server, err := corehttp.New(corehttp.Config{Host: "127.0.0.1", Port: 8031}, nil)
	if err != nil {
		t.Fatalf("corehttp.New() error = %v", err)
	}

	server.RegisterRoutes(registerCoreStatusRoute)

	request, _ := http.NewRequest(http.MethodGet, "/status", nil)
	response, err := server.App().Test(request)
	if err != nil {
		t.Fatalf("App().Test() error = %v", err)
	}
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}
}

// TestWaitForShutdownReturnsServerError verifies server-error propagation behavior.
func TestWaitForShutdownReturnsServerError(t *testing.T) {
	serverErrors := make(chan error, 1)
	messagingErrors := make(chan error, 1)
	serverErrors <- errors.New("listen failed")

	err := waitForShutdown(context.Background(), nil, nil, nil, serverErrors, messagingErrors)
	if err == nil {
		t.Fatalf("expected waitForShutdown() error")
	}
	if !strings.Contains(err.Error(), "http server stopped") {
		t.Fatalf("error = %v, want http server stopped", err)
	}
}

// TestWaitForShutdownReturnsMessagingError verifies messaging-error propagation behavior.
func TestWaitForShutdownReturnsMessagingError(t *testing.T) {
	serverErrors := make(chan error, 1)
	messagingErrors := make(chan error, 1)
	messagingErrors <- errors.New("router failed")

	err := waitForShutdown(context.Background(), nil, nil, nil, serverErrors, messagingErrors)
	if err == nil {
		t.Fatalf("expected waitForShutdown() error")
	}
	if !strings.Contains(err.Error(), "messaging router stopped") {
		t.Fatalf("error = %v, want messaging router stopped", err)
	}
}

// TestWaitForShutdownHandlesServerStopWithoutError verifies graceful shutdown on nil server stop errors.
func TestWaitForShutdownHandlesServerStopWithoutError(t *testing.T) {
	db, server, messaging := newRuntimeResourcesForTest(t)

	serverErrors := make(chan error, 1)
	messagingErrors := make(chan error, 1)
	serverErrors <- nil

	err := waitForShutdown(context.Background(), db, server, messaging, serverErrors, messagingErrors)
	if err != nil {
		t.Fatalf("waitForShutdown() error = %v", err)
	}
}

// TestWaitForShutdownHandlesCanceledMessaging verifies graceful shutdown on canceled messaging errors.
func TestWaitForShutdownHandlesCanceledMessaging(t *testing.T) {
	db, server, messaging := newRuntimeResourcesForTest(t)

	serverErrors := make(chan error, 1)
	messagingErrors := make(chan error, 1)
	messagingErrors <- context.Canceled

	err := waitForShutdown(context.Background(), db, server, messaging, serverErrors, messagingErrors)
	if err != nil {
		t.Fatalf("waitForShutdown() error = %v", err)
	}
}

// newRuntimeResourcesForTest creates runtime resources for wait/shutdown tests.
func newRuntimeResourcesForTest(t *testing.T) (*gorm.DB, *corehttp.Server, *corewatermill.InMemoryPlatform) {
	t.Helper()

	db, err := coredatabase.Open(coredatabase.Config{Driver: "sqlite", DSN: "file::memory:?cache=shared"}, nil)
	if err != nil {
		t.Fatalf("coredatabase.Open() error = %v", err)
	}

	server, err := corehttp.New(corehttp.Config{Host: "127.0.0.1", Port: 8032}, nil)
	if err != nil {
		t.Fatalf("corehttp.New() error = %v", err)
	}

	messaging, err := corewatermill.NewInMemoryPlatform(coremsgplatform.Config{}, nil)
	if err != nil {
		t.Fatalf("corewatermill.NewInMemoryPlatform() error = %v", err)
	}

	return db, server, messaging
}
