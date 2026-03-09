package service

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"strings"
	"time"

	syncdomain "mannaiah/module/falabella/domain/sync"
	"mannaiah/module/falabella/port"
)

var (
	// ErrNilRepository is returned when repository dependencies are nil.
	ErrNilRepository = errors.New("falabella sync status repository must not be nil")
	// ErrNilSource is returned when source dependencies are nil.
	ErrNilSource = errors.New("falabella source must not be nil")
	// ErrInvalidFeedID is returned when feed identifier values are empty.
	ErrInvalidFeedID = errors.New("falabella feed id is required")
	// ErrInvalidProductID is returned when product identifier values are empty.
	ErrInvalidProductID = errors.New("falabella product id is required")
	// ErrInvalidExecutionID is returned when execution identifier values are empty.
	ErrInvalidExecutionID = errors.New("falabella execution id is required")
	// ErrFeedNotFinished is returned when feed status resolution is attempted on unfinished feeds.
	ErrFeedNotFinished = errors.New("falabella feed is not finished")
)

// Repository defines sync status persistence behavior required by this service.
type Repository interface {
	// EnsureSchema migrates sync status persistence schema.
	EnsureSchema(ctx context.Context) error
	// CreateExecution persists one sync execution parent record.
	CreateExecution(ctx context.Context, execution *syncdomain.SyncExecution) error
	// Create persists a new sync status entry.
	Create(ctx context.Context, entry *syncdomain.SyncEntry) error
	// GetExecutionByID retrieves one sync execution by identifier.
	GetExecutionByID(ctx context.Context, executionID string) (*syncdomain.SyncExecution, error)
	// GetByFeedID retrieves a sync status entry by Falabella feed identifier.
	GetByFeedID(ctx context.Context, feedID string) (*syncdomain.SyncEntry, error)
	// ListByExecutionID retrieves child feed rows by execution identifier.
	ListByExecutionID(ctx context.Context, executionID string) ([]syncdomain.SyncEntry, error)
	// GetByProductID retrieves sync status entries by source product identifier.
	GetByProductID(ctx context.Context, productID string) ([]syncdomain.SyncEntry, error)
	// ListPending retrieves unresolved sync status entries ordered by submission time.
	ListPending(ctx context.Context, limit int) ([]syncdomain.SyncEntry, error)
	// UpdateStatus updates the status and resolution timestamp of a sync status entry.
	UpdateStatus(ctx context.Context, feedID string, status syncdomain.SyncStatus, resolvedAt *time.Time) error
}

// Source defines Falabella feed status retrieval behavior required by this service.
type Source interface {
	// GetFeedStatus retrieves Falabella feed status by feed identifier.
	GetFeedStatus(ctx context.Context, feedID string) ([]byte, error)
}

// Service defines sync status use-case behavior.
type Service interface {
	// RecordEntry persists a new sync status entry.
	RecordEntry(ctx context.Context, entry *syncdomain.SyncEntry) error
	// GetExecutionByID retrieves one sync execution by identifier.
	GetExecutionByID(ctx context.Context, executionID string) (*syncdomain.SyncExecution, error)
	// GetByFeedID retrieves a sync status entry by Falabella feed identifier.
	GetByFeedID(ctx context.Context, feedID string) (*syncdomain.SyncEntry, error)
	// GetByExecutionID retrieves child feed rows by execution identifier.
	GetByExecutionID(ctx context.Context, executionID string) ([]syncdomain.SyncEntry, error)
	// GetByProductID retrieves sync status entries by source product identifier.
	GetByProductID(ctx context.Context, productID string) ([]syncdomain.SyncEntry, error)
	// ResolveFeedStatus queries Falabella feed status and updates the entry resolution.
	ResolveFeedStatus(ctx context.Context, feedID string) (*ResolveResult, error)
	// ResolvePendingFeeds resolves all pending feed entries by querying Falabella FeedStatus API.
	ResolvePendingFeeds(ctx context.Context, limit int) (*ResolvePendingResult, error)
}

// ResolveResult defines feed status resolution result values.
type ResolveResult struct {
	// FeedID defines the Falabella feed identifier.
	FeedID string `json:"feedId"`
	// Step defines the logical sync step for this feed (product/image) when known.
	Step string `json:"step,omitempty"`
	// Task defines high-level sync task category values (data/image) when known.
	Task string `json:"task,omitempty"`
	// Status defines the resolved sync status.
	Status string `json:"status"`
	// Action defines the feed action type.
	Action string `json:"action"`
	// TotalRecords defines total record count.
	TotalRecords int `json:"totalRecords"`
	// ProcessedRecords defines processed record count.
	ProcessedRecords int `json:"processedRecords"`
	// FailedRecords defines failed record count.
	FailedRecords int `json:"failedRecords"`
	// Errors defines per-record error messages.
	Errors []FeedErrorDetail `json:"errors,omitempty"`
}

// FeedErrorDetail defines feed error detail values.
type FeedErrorDetail struct {
	// Code defines error code values.
	Code int `json:"code"`
	// Message defines error message values.
	Message string `json:"message"`
	// SellerSku defines affected seller SKU values.
	SellerSku string `json:"sellerSku,omitempty"`
}

// SyncStatusService implements sync status use cases.
type SyncStatusService struct {
	// repo defines sync status persistence dependencies.
	repo Repository
	// source defines Falabella feed status source dependencies.
	source Source
}

var (
	// _ ensures SyncStatusService satisfies service contracts.
	_ Service = (*SyncStatusService)(nil)
)

// NewService creates sync status services.
func NewService(repo Repository, source Source) (*SyncStatusService, error) {
	if repo == nil {
		return nil, ErrNilRepository
	}
	if source == nil {
		return nil, ErrNilSource
	}

	return &SyncStatusService{repo: repo, source: source}, nil
}

// RecordEntry persists a new sync status entry.
func (s *SyncStatusService) RecordEntry(ctx context.Context, entry *syncdomain.SyncEntry) error {
	if entry == nil {
		return errors.New("sync entry must not be nil")
	}

	return s.repo.Create(ctx, entry)
}

// GetExecutionByID retrieves one sync execution by identifier.
func (s *SyncStatusService) GetExecutionByID(ctx context.Context, executionID string) (*syncdomain.SyncExecution, error) {
	trimmed := strings.TrimSpace(executionID)
	if trimmed == "" {
		return nil, ErrInvalidExecutionID
	}

	return s.repo.GetExecutionByID(ctx, trimmed)
}

// GetByFeedID retrieves a sync status entry by Falabella feed identifier.
func (s *SyncStatusService) GetByFeedID(ctx context.Context, feedID string) (*syncdomain.SyncEntry, error) {
	trimmed := strings.TrimSpace(feedID)
	if trimmed == "" {
		return nil, ErrInvalidFeedID
	}

	return s.repo.GetByFeedID(ctx, trimmed)
}

// GetByExecutionID retrieves child feed rows by execution identifier.
func (s *SyncStatusService) GetByExecutionID(ctx context.Context, executionID string) ([]syncdomain.SyncEntry, error) {
	trimmed := strings.TrimSpace(executionID)
	if trimmed == "" {
		return nil, ErrInvalidExecutionID
	}

	return s.repo.ListByExecutionID(ctx, trimmed)
}

// GetByProductID retrieves sync status entries by source product identifier.
func (s *SyncStatusService) GetByProductID(ctx context.Context, productID string) ([]syncdomain.SyncEntry, error) {
	trimmed := strings.TrimSpace(productID)
	if trimmed == "" {
		return nil, ErrInvalidProductID
	}

	return s.repo.GetByProductID(ctx, trimmed)
}

// ResolveFeedStatus queries Falabella feed status API and updates the entry resolution.
func (s *SyncStatusService) ResolveFeedStatus(ctx context.Context, feedID string) (*ResolveResult, error) {
	trimmed := strings.TrimSpace(feedID)
	if trimmed == "" {
		return nil, ErrInvalidFeedID
	}
	var (
		entryStep string
		entryTask string
	)
	if entry, entryErr := s.repo.GetByFeedID(ctx, trimmed); entryErr == nil && entry != nil {
		entryStep = entry.Step.String()
		entryTask = entry.Task.String()
	} else if entryErr != nil && !errors.Is(entryErr, port.ErrSyncEntryNotFound) {
		return nil, fmt.Errorf("get sync status entry: %w", entryErr)
	}

	rawPayload, err := s.source.GetFeedStatus(ctx, trimmed)
	if err != nil {
		return nil, fmt.Errorf("get falabella feed status: %w", err)
	}

	var response syncdomain.FeedResponse
	if xmlErr := xml.Unmarshal(rawPayload, &response); xmlErr != nil {
		return nil, fmt.Errorf("unmarshal falabella feed status: %w", xmlErr)
	}

	detail := response.Body.FeedDetail
	if !detail.IsFinished() {
		return nil, fmt.Errorf("%w: current status is %q", ErrFeedNotFinished, detail.Status)
	}

	resolvedStatus := syncdomain.SyncStatusFinished
	if !detail.IsSuccess() {
		resolvedStatus = syncdomain.SyncStatusFailed
	}

	now := time.Now().UTC()
	if updateErr := s.repo.UpdateStatus(ctx, trimmed, resolvedStatus, &now); updateErr != nil {
		if !errors.Is(updateErr, port.ErrSyncEntryNotFound) {
			return nil, fmt.Errorf("update sync status: %w", updateErr)
		}
	}

	feedErrors := make([]FeedErrorDetail, 0, len(detail.FeedErrors.Errors))
	for _, e := range detail.FeedErrors.Errors {
		feedErrors = append(feedErrors, FeedErrorDetail{
			Code:      e.Code,
			Message:   strings.TrimSpace(e.Message),
			SellerSku: strings.TrimSpace(e.SellerSku),
		})
	}

	return &ResolveResult{
		FeedID:           detail.Feed,
		Step:             entryStep,
		Task:             entryTask,
		Status:           detail.Status,
		Action:           detail.Action,
		TotalRecords:     detail.TotalRecords,
		ProcessedRecords: detail.ProcessedRecords,
		FailedRecords:    detail.FailedRecords,
		Errors:           feedErrors,
	}, nil
}
