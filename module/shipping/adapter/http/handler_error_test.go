package http

import (
	"context"
	"errors"
	"testing"

	corehttp "mannaiah/module/core/http"
	dispatchservice "mannaiah/module/shipping/application/dispatch/service"
	markservice "mannaiah/module/shipping/application/mark/service"
	quotationservice "mannaiah/module/shipping/application/quotation/service"
	"mannaiah/module/shipping/domain"
	"mannaiah/module/shipping/port"
)

type quotationServiceStub struct{}

func (quotationServiceStub) Quote(ctx context.Context, command quotationservice.QuoteCommand) (*domain.QuotationResult, error) {
	return &domain.QuotationResult{}, nil
}
func (quotationServiceStub) ListByOrderID(ctx context.Context, orderID string) ([]port.QuotationRecord, error) {
	return nil, nil
}

type markServiceStub struct{}

func (markServiceStub) Generate(ctx context.Context, command markservice.GenerateCommand) (*domain.ShippingMark, error) {
	return &domain.ShippingMark{}, nil
}
func (markServiceStub) Get(ctx context.Context, id string) (*domain.ShippingMark, error) {
	return &domain.ShippingMark{}, nil
}
func (markServiceStub) List(ctx context.Context, query markservice.ListQuery) ([]domain.ShippingMark, int64, error) {
	return nil, 0, nil
}
func (markServiceStub) Void(ctx context.Context, id string, reason string) (*domain.ShippingMark, error) {
	return &domain.ShippingMark{}, nil
}
func (markServiceStub) QueryDispatch(ctx context.Context, query markservice.DispatchQuery) (*markservice.DispatchResult, error) {
	return &markservice.DispatchResult{}, nil
}
func (markServiceStub) Related(ctx context.Context, id string) ([]domain.ShippingMark, error) {
	return nil, nil
}

type dispatchServiceStub struct{}

func (dispatchServiceStub) Create(ctx context.Context, command dispatchservice.CreateBatchCommand) (*domain.DispatchBatch, error) {
	return &domain.DispatchBatch{}, nil
}
func (dispatchServiceStub) Get(ctx context.Context, id string) (*domain.DispatchBatch, error) {
	return &domain.DispatchBatch{}, nil
}
func (dispatchServiceStub) List(ctx context.Context, query dispatchservice.ListQuery) ([]domain.DispatchBatch, int64, error) {
	return nil, 0, nil
}
func (dispatchServiceStub) DraftMark(ctx context.Context, command dispatchservice.DraftMarkCommand) (*domain.ShippingMark, error) {
	return &domain.ShippingMark{}, nil
}
func (dispatchServiceStub) RemoveDraftMark(ctx context.Context, batchID string, markID string) (*domain.DispatchBatch, error) {
	return &domain.DispatchBatch{}, nil
}
func (dispatchServiceStub) Close(ctx context.Context, batchID string) (*domain.DispatchBatch, error) {
	return &domain.DispatchBatch{}, nil
}
func (dispatchServiceStub) ManifestDocument(ctx context.Context, batchID string) ([]byte, error) {
	return []byte("%PDF"), nil
}

type trackingServiceStub struct{}

func (trackingServiceStub) Get(ctx context.Context, carrierID string, trackingNumber string) (*domain.TrackingHistory, error) {
	return &domain.TrackingHistory{}, nil
}

type carrierServiceStub struct{}

func (carrierServiceStub) List(ctx context.Context) ([]domain.Carrier, error) { return nil, nil }
func (carrierServiceStub) Get(ctx context.Context, id string) (*domain.Carrier, error) {
	return &domain.Carrier{}, nil
}

// TestMapError verifies shipping error-to-http mapping.
func TestMapError(t *testing.T) {
	handler, err := NewHandler(quotationServiceStub{}, markServiceStub{}, dispatchServiceStub{}, trackingServiceStub{}, carrierServiceStub{})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	cases := []struct {
		err        error
		statusCode int
		code       string
	}{
		{err: domain.ErrCarrierNotSupported, statusCode: 400, code: "carrier_not_supported"},
		{err: domain.ErrBatchClosed, statusCode: 409, code: "batch_closed"},
		{err: domain.ErrInvalidBatchStatus, statusCode: 409, code: "batch_status_invalid"},
		{err: domain.ErrNotFound, statusCode: 404, code: "shipping_resource_not_found"},
		{err: errors.New("boom"), statusCode: 500, code: "internal_server_error"},
	}
	for _, testCase := range cases {
		mapped := handler.mapError(testCase.err)
		appErr := &corehttp.AppError{}
		if !errors.As(mapped, &appErr) {
			t.Fatalf("mapError() type = %T", mapped)
		}
		if appErr.Status != testCase.statusCode || appErr.Message != testCase.code {
			t.Fatalf("mapError() = (%d,%q), want (%d,%q)", appErr.Status, appErr.Message, testCase.statusCode, testCase.code)
		}
	}
}
