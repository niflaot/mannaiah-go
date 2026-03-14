package application

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"mannaiah/module/syncrecord/domain"
	"mannaiah/module/syncrecord/port"
)

var (
	// ErrNilRepository is returned when repository dependencies are nil.
	ErrNilRepository = errors.New("sync record repository must not be nil")
)

// ListResult defines paged sync-run query output values.
type ListResult struct {
	// Data defines run rows in the current page.
	Data []domain.SyncRun
	// Page defines current page number.
	Page int
	// Limit defines current page size.
	Limit int
	// Total defines total matching rows.
	Total int64
	// TotalPages defines total available pages.
	TotalPages int
}

// Service defines sync run use-cases and recorder behavior.
type Service interface {
	// StartRun starts a running sync run and returns its id.
	StartRun(ctx context.Context, input port.StartRunInput) (string, error)
	// CompleteRun marks a run as completed.
	CompleteRun(ctx context.Context, input port.FinishRunInput) error
	// FailRun marks a run as failed and persists child errors.
	FailRun(ctx context.Context, input port.FinishRunInput) error
	// GetRun retrieves one run by id.
	GetRun(ctx context.Context, runID string) (*domain.SyncRun, error)
	// ListRuns returns paged run rows using filter query values.
	ListRuns(ctx context.Context, query port.ListQuery) (*ListResult, error)
	// StatsSince returns aggregate stats since one timestamp.
	StatsSince(ctx context.Context, since time.Time) (*domain.RunStats, error)
	// CleanupBefore deletes runs older than cutoff.
	CleanupBefore(ctx context.Context, cutoff time.Time) (int64, error)
}

// RecorderService implements sync-run use-cases.
type RecorderService struct {
	// repository defines persistence dependency.
	repository port.Repository
}

var (
	// _ ensures RecorderService satisfies service contracts.
	_ Service = (*RecorderService)(nil)
	// _ ensures RecorderService satisfies recorder contracts.
	_ port.Recorder = (*RecorderService)(nil)
)

// NewService creates sync record services.
func NewService(repository port.Repository) (*RecorderService, error) {
	if repository == nil {
		return nil, ErrNilRepository
	}

	return &RecorderService{repository: repository}, nil
}

// StartRun starts a running sync run and returns its id.
func (s *RecorderService) StartRun(ctx context.Context, input port.StartRunInput) (string, error) {
	kind := domain.SyncKind(strings.TrimSpace(string(input.Kind)))
	if !kind.IsValid() {
		return "", domain.ErrInvalidKind
	}

	trigger := domain.SyncTrigger(strings.TrimSpace(string(input.Trigger)))
	if !trigger.IsValid() {
		return "", domain.ErrInvalidTrigger
	}

	startedAt := time.Now().UTC()
	if input.StartedAt != nil && !input.StartedAt.IsZero() {
		startedAt = input.StartedAt.UTC()
	}

	run := &domain.SyncRun{
		ID:        uuid.NewString(),
		Kind:      kind,
		Trigger:   trigger,
		Status:    domain.RunStatusRunning,
		StartedAt: startedAt,
		Metadata:  cloneMetadata(input.Metadata),
	}
	if err := s.repository.CreateRun(ctx, run); err != nil {
		return "", fmt.Errorf("create sync run: %w", err)
	}

	return run.ID, nil
}

// CompleteRun marks a run as completed.
func (s *RecorderService) CompleteRun(ctx context.Context, input port.FinishRunInput) error {
	trimmedRunID := strings.TrimSpace(input.RunID)
	if trimmedRunID == "" {
		return domain.ErrInvalidRunID
	}

	endedAt := time.Now().UTC()
	if input.EndedAt != nil && !input.EndedAt.IsZero() {
		endedAt = input.EndedAt.UTC()
	}

	if err := s.repository.CompleteRun(ctx, port.CompleteInput{
		RunID:     trimmedRunID,
		Status:    domain.RunStatusCompleted,
		EndedAt:   endedAt,
		Processed: input.Processed,
		Succeeded: input.Succeeded,
		Failed:    input.Failed,
		Skipped:   input.Skipped,
	}); err != nil {
		return fmt.Errorf("complete sync run: %w", err)
	}

	return nil
}

// FailRun marks a run as failed and persists child errors.
func (s *RecorderService) FailRun(ctx context.Context, input port.FinishRunInput) error {
	trimmedRunID := strings.TrimSpace(input.RunID)
	if trimmedRunID == "" {
		return domain.ErrInvalidRunID
	}

	endedAt := time.Now().UTC()
	if input.EndedAt != nil && !input.EndedAt.IsZero() {
		endedAt = input.EndedAt.UTC()
	}

	errorsToInsert := normalizeErrors(trimmedRunID, input.Errors, endedAt)
	if len(errorsToInsert) > 0 {
		if err := s.repository.AddRunErrors(ctx, errorsToInsert); err != nil {
			return fmt.Errorf("insert sync run errors: %w", err)
		}
	}

	if err := s.repository.CompleteRun(ctx, port.CompleteInput{
		RunID:     trimmedRunID,
		Status:    domain.RunStatusFailed,
		EndedAt:   endedAt,
		Processed: input.Processed,
		Succeeded: input.Succeeded,
		Failed:    input.Failed,
		Skipped:   input.Skipped,
	}); err != nil {
		return fmt.Errorf("fail sync run: %w", err)
	}

	return nil
}

// GetRun retrieves one run by id.
func (s *RecorderService) GetRun(ctx context.Context, runID string) (*domain.SyncRun, error) {
	trimmedRunID := strings.TrimSpace(runID)
	if trimmedRunID == "" {
		return nil, domain.ErrInvalidRunID
	}

	run, err := s.repository.GetRunByID(ctx, trimmedRunID)
	if err != nil {
		return nil, fmt.Errorf("get sync run: %w", err)
	}

	return run, nil
}

// ListRuns returns paged run rows using filter query values.
func (s *RecorderService) ListRuns(ctx context.Context, query port.ListQuery) (*ListResult, error) {
	page := query.Page
	if page <= 0 {
		page = 1
	}
	limit := query.Limit
	if limit <= 0 {
		limit = 50
	}

	normalized := query
	normalized.Page = page
	normalized.Limit = limit

	runs, total, err := s.repository.ListRuns(ctx, normalized)
	if err != nil {
		return nil, fmt.Errorf("list sync runs: %w", err)
	}

	totalPages := 0
	if total > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(limit)))
	}

	return &ListResult{Data: runs, Page: page, Limit: limit, Total: total, TotalPages: totalPages}, nil
}

// StatsSince returns aggregate stats since one timestamp.
func (s *RecorderService) StatsSince(ctx context.Context, since time.Time) (*domain.RunStats, error) {
	if since.IsZero() {
		since = time.Now().UTC().Add(-24 * time.Hour)
	}

	stats, err := s.repository.StatsSince(ctx, since.UTC())
	if err != nil {
		return nil, fmt.Errorf("sync run stats: %w", err)
	}

	return stats, nil
}

// CleanupBefore deletes runs older than cutoff.
func (s *RecorderService) CleanupBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	if cutoff.IsZero() {
		return 0, nil
	}

	deleted, err := s.repository.CleanupBefore(ctx, cutoff.UTC())
	if err != nil {
		return 0, fmt.Errorf("cleanup sync runs: %w", err)
	}

	return deleted, nil
}

// normalizeErrors applies defaults for run error rows.
func normalizeErrors(runID string, entries []domain.SyncRunError, timestamp time.Time) []domain.SyncRunError {
	if len(entries) == 0 {
		return nil
	}

	normalized := make([]domain.SyncRunError, 0, len(entries))
	for _, entry := range entries {
		message := strings.TrimSpace(entry.Message)
		if message == "" {
			continue
		}

		errorID := strings.TrimSpace(entry.ID)
		if errorID == "" {
			errorID = uuid.NewString()
		}
		createdAt := timestamp.UTC()
		if !entry.CreatedAt.IsZero() {
			createdAt = entry.CreatedAt.UTC()
		}

		normalized = append(normalized, domain.SyncRunError{
			ID:        errorID,
			RunID:     runID,
			ErrorType: strings.TrimSpace(entry.ErrorType),
			ErrorCode: strings.TrimSpace(entry.ErrorCode),
			Message:   message,
			CreatedAt: createdAt,
		})
	}

	return normalized
}

// cloneMetadata clones metadata maps.
func cloneMetadata(metadata map[string]string) map[string]string {
	if len(metadata) == 0 {
		return nil
	}

	cloned := make(map[string]string, len(metadata))
	for key, value := range metadata {
		cloned[key] = value
	}

	return cloned
}
