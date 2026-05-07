package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"mannaiah/module/exports/domain"
	"mannaiah/module/exports/port"
)

const (
	reportContentType = "text/csv"
	reportStorageRoot = "exports"
)

var (
	// ErrNilRepository is returned when report repositories are nil.
	ErrNilRepository = errors.New("exports repository must not be nil")
	// ErrNilStorage is returned when object storage dependencies are nil.
	ErrNilStorage = errors.New("exports storage must not be nil")
	// ErrNilContactSource is returned when contact source dependencies are nil.
	ErrNilContactSource = errors.New("exports contact source must not be nil")
	// ErrNilOrderSource is returned when order source dependencies are nil.
	ErrNilOrderSource = errors.New("exports order source must not be nil")
)

// ListResult defines paginated export report registry results.
type ListResult struct {
	// Data defines current-page report rows.
	Data []domain.Report `json:"data"`
	// Page defines current page values.
	Page int `json:"page"`
	// Limit defines current page-size values.
	Limit int `json:"limit"`
	// Total defines filtered total values.
	Total int64 `json:"total"`
	// TotalPages defines total page values.
	TotalPages int `json:"totalPages"`
}

// Service defines export generation and registry use cases.
type Service interface {
	// GenerateContacts creates a contact CSV report.
	GenerateContacts(ctx context.Context) (*domain.Report, error)
	// GenerateOrders creates an order CSV report.
	GenerateOrders(ctx context.Context) (*domain.Report, error)
	// GetReport retrieves one report by id.
	GetReport(ctx context.Context, id string) (*domain.Report, error)
	// ListReports returns paginated reports.
	ListReports(ctx context.Context, query port.ListQuery) (*ListResult, error)
	// SearchReports returns paginated reports using filter criteria.
	SearchReports(ctx context.Context, query port.ListQuery) (*ListResult, error)
}

// ExportService implements CSV export generation use cases.
type ExportService struct {
	// repository defines report registry persistence dependencies.
	repository port.Repository
	// storage defines report object storage dependencies.
	storage port.Storage
	// contacts defines contact export source dependencies.
	contacts port.ContactSource
	// orders defines order export source dependencies.
	orders port.OrderSource
	// now resolves generation timestamps.
	now func() time.Time
}

var (
	// _ ensures ExportService satisfies Service contracts.
	_ Service = (*ExportService)(nil)
)

// NewService creates export generation services.
func NewService(repository port.Repository, storage port.Storage, contacts port.ContactSource, orders port.OrderSource) (*ExportService, error) {
	if repository == nil {
		return nil, ErrNilRepository
	}
	if storage == nil {
		return nil, ErrNilStorage
	}
	if contacts == nil {
		return nil, ErrNilContactSource
	}
	if orders == nil {
		return nil, ErrNilOrderSource
	}

	return &ExportService{
		repository: repository,
		storage:    storage,
		contacts:   contacts,
		orders:     orders,
		now:        func() time.Time { return time.Now().UTC() },
	}, nil
}

// SetClock configures deterministic generation timestamps.
func (s *ExportService) SetClock(now func() time.Time) {
	if s == nil || now == nil {
		return
	}
	s.now = now
}

// GenerateContacts creates a contact CSV report.
func (s *ExportService) GenerateContacts(ctx context.Context) (*domain.Report, error) {
	rows, err := s.contacts.ListContacts(ctx)
	if err != nil {
		return nil, fmt.Errorf("list contacts for export: %w", err)
	}
	body, err := buildContactsCSV(rows)
	if err != nil {
		return nil, fmt.Errorf("build contacts csv: %w", err)
	}

	return s.persistReport(ctx, domain.ReportTypeContacts, len(rows), body)
}

// GenerateOrders creates an order CSV report.
func (s *ExportService) GenerateOrders(ctx context.Context) (*domain.Report, error) {
	rows, err := s.orders.ListOrders(ctx)
	if err != nil {
		return nil, fmt.Errorf("list orders for export: %w", err)
	}
	body, err := buildOrdersCSV(rows)
	if err != nil {
		return nil, fmt.Errorf("build orders csv: %w", err)
	}

	return s.persistReport(ctx, domain.ReportTypeOrders, len(rows), body)
}

// GetReport retrieves one report by id.
func (s *ExportService) GetReport(ctx context.Context, id string) (*domain.Report, error) {
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return nil, domain.ErrInvalidReportID
	}

	return s.repository.GetByID(ctx, trimmedID)
}

// ListReports returns paginated reports.
func (s *ExportService) ListReports(ctx context.Context, query port.ListQuery) (*ListResult, error) {
	return s.list(ctx, query)
}

// SearchReports returns paginated reports using filter criteria.
func (s *ExportService) SearchReports(ctx context.Context, query port.ListQuery) (*ListResult, error) {
	return s.list(ctx, query)
}

// persistReport uploads generated CSV bytes and stores registry metadata.
func (s *ExportService) persistReport(ctx context.Context, reportType domain.ReportType, rowCount int, body []byte) (*domain.Report, error) {
	if err := s.storage.AvailabilityError(); err != nil {
		return nil, fmt.Errorf("export storage unavailable: %w", err)
	}

	generatedAt := s.now().UTC()
	stamp := generatedAt.Format("20060102T150405Z")
	sum := sha256.Sum256(body)
	hash := hex.EncodeToString(sum[:])
	normalizedType := reportType.Normalize()
	fileName := fmt.Sprintf("%s-export-%s.csv", normalizedType, stamp)
	storageKey := fmt.Sprintf("%s/%s/%s-%s.csv", reportStorageRoot, normalizedType, stamp, hash)

	if err := s.storage.Upload(ctx, port.UploadRequest{
		Key:         storageKey,
		ContentType: reportContentType,
		Body:        body,
	}); err != nil {
		return nil, fmt.Errorf("upload export report: %w", err)
	}

	report := &domain.Report{
		ID:          uuid.NewString(),
		Type:        normalizedType,
		Status:      domain.ReportStatusCompleted,
		Stamp:       stamp,
		FileName:    fileName,
		StorageKey:  storageKey,
		SHA256:      hash,
		ContentType: reportContentType,
		RowCount:    rowCount,
		ByteSize:    int64(len(body)),
		GeneratedAt: generatedAt,
		CreatedAt:   generatedAt,
		UpdatedAt:   generatedAt,
	}
	report.Normalize()
	if err := report.Validate(); err != nil {
		return nil, err
	}
	if err := s.repository.Create(ctx, report); err != nil {
		return nil, fmt.Errorf("create export registry entry: %w", err)
	}

	return report, nil
}

// list returns paginated report registry values.
func (s *ExportService) list(ctx context.Context, query port.ListQuery) (*ListResult, error) {
	normalized := normalizeListQuery(query)
	if normalized.Type != "" && !normalized.Type.Normalize().IsValid() {
		return nil, domain.ErrInvalidReportType
	}

	data, total, err := s.repository.List(ctx, normalized)
	if err != nil {
		return nil, fmt.Errorf("list export reports: %w", err)
	}

	totalPages := 0
	if total > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(normalized.Limit)))
	}

	return &ListResult{
		Data:       data,
		Page:       normalized.Page,
		Limit:      normalized.Limit,
		Total:      total,
		TotalPages: totalPages,
	}, nil
}

// normalizeListQuery applies report list defaults.
func normalizeListQuery(query port.ListQuery) port.ListQuery {
	normalized := query
	normalized.Type = normalized.Type.Normalize()
	if normalized.Page <= 0 {
		normalized.Page = 1
	}
	if normalized.Limit <= 0 {
		normalized.Limit = 50
	}
	if normalized.Limit > 500 {
		normalized.Limit = 500
	}
	return normalized
}
