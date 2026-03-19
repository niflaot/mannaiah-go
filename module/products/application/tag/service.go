package tag

import (
	"context"
	"errors"
	"fmt"
	"strings"

	tagdomain "mannaiah/module/products/domain/tag"
	tagport "mannaiah/module/products/port/tag"
)

var (
	// ErrNilRepository is returned when repository dependencies are nil.
	ErrNilRepository = errors.New("tags repository must not be nil")
	// ErrInvalidTagName is returned when tag names are empty.
	ErrInvalidTagName = errors.New("tag name is required")
	// ErrInvalidSourceTag is returned when source tag names are empty.
	ErrInvalidSourceTag = errors.New("source tag is required")
	// ErrInvalidTargetTag is returned when target tag names are empty.
	ErrInvalidTargetTag = errors.New("target tag is required")
	// ErrProbabilityRange is returned when probability is outside 0.00–100.00.
	ErrProbabilityRange = errors.New("probability must be between 0.00 and 100.00")
	// ErrSelfCorrelation is returned when source and target tags are identical.
	ErrSelfCorrelation = errors.New("source and target tags must differ")
)

// CreateCorrelationCommand defines create-correlation command payloads.
type CreateCorrelationCommand struct {
	// SourceTag defines the source tag name.
	SourceTag string
	// TargetTag defines the target tag name.
	TargetTag string
	// Probability defines cross-sell purchase probability (0.00–100.00).
	Probability float64
	// Notes defines optional marketing notes.
	Notes string
}

// UpdateCorrelationCommand defines update-correlation command payloads.
type UpdateCorrelationCommand struct {
	// Probability defines optional updated probability.
	Probability *float64
	// Notes defines optional updated notes.
	Notes *string
	// HasProbability reports whether Probability was provided.
	HasProbability bool
	// HasNotes reports whether Notes was provided.
	HasNotes bool
}

// Service defines tag application use cases.
type Service interface {
	// EnsureAll creates missing tags and reintegrates soft-deleted ones.
	EnsureAll(ctx context.Context, names []string) error
	// List returns all non-deleted tags.
	List(ctx context.Context) ([]tagdomain.Tag, error)
	// SoftDelete soft-deletes a tag by name and cascades to product_tags.
	SoftDelete(ctx context.Context, name string) error
	// ListCorrelations returns all tag correlations.
	ListCorrelations(ctx context.Context) ([]tagdomain.TagCorrelation, error)
	// ListCorrelationsBySource returns correlations for a specific source tag.
	ListCorrelationsBySource(ctx context.Context, sourceTag string) ([]tagdomain.TagCorrelation, error)
	// CreateCorrelation creates a new tag correlation.
	CreateCorrelation(ctx context.Context, cmd CreateCorrelationCommand) (*tagdomain.TagCorrelation, error)
	// UpdateCorrelation updates an existing tag correlation by ID.
	UpdateCorrelation(ctx context.Context, id uint, cmd UpdateCorrelationCommand) (*tagdomain.TagCorrelation, error)
	// DeleteCorrelation deletes a tag correlation by ID.
	DeleteCorrelation(ctx context.Context, id uint) error
}

// TagService implements tag use cases.
type TagService struct {
	// repository defines persistence dependencies.
	repository tagport.Repository
}

var (
	// _ ensures TagService satisfies Service contracts.
	_ Service = (*TagService)(nil)
)

// NewService creates tag services.
func NewService(repository tagport.Repository) (*TagService, error) {
	if repository == nil {
		return nil, ErrNilRepository
	}

	return &TagService{repository: repository}, nil
}

// EnsureAll creates missing tags and reintegrates soft-deleted ones.
func (s *TagService) EnsureAll(ctx context.Context, names []string) error {
	if err := s.repository.EnsureAll(ctx, names); err != nil {
		return fmt.Errorf("ensure tags: %w", err)
	}

	return nil
}

// List returns all non-deleted tags.
func (s *TagService) List(ctx context.Context) ([]tagdomain.Tag, error) {
	tags, err := s.repository.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list tags: %w", err)
	}

	return tags, nil
}

// SoftDelete soft-deletes a tag by name.
func (s *TagService) SoftDelete(ctx context.Context, name string) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return ErrInvalidTagName
	}

	if err := s.repository.SoftDelete(ctx, trimmed); err != nil {
		return fmt.Errorf("delete tag: %w", err)
	}

	return nil
}

// ListCorrelations returns all correlations.
func (s *TagService) ListCorrelations(ctx context.Context) ([]tagdomain.TagCorrelation, error) {
	correlations, err := s.repository.ListCorrelations(ctx)
	if err != nil {
		return nil, fmt.Errorf("list correlations: %w", err)
	}

	return correlations, nil
}

// ListCorrelationsBySource returns correlations for a specific source tag.
func (s *TagService) ListCorrelationsBySource(ctx context.Context, sourceTag string) ([]tagdomain.TagCorrelation, error) {
	trimmed := strings.TrimSpace(sourceTag)
	if trimmed == "" {
		return nil, ErrInvalidSourceTag
	}

	correlations, err := s.repository.ListCorrelationsBySource(ctx, trimmed)
	if err != nil {
		return nil, fmt.Errorf("list correlations by source: %w", err)
	}

	return correlations, nil
}

// CreateCorrelation creates a new tag correlation.
func (s *TagService) CreateCorrelation(ctx context.Context, cmd CreateCorrelationCommand) (*tagdomain.TagCorrelation, error) {
	source := strings.TrimSpace(cmd.SourceTag)
	target := strings.TrimSpace(cmd.TargetTag)

	if source == "" {
		return nil, ErrInvalidSourceTag
	}
	if target == "" {
		return nil, ErrInvalidTargetTag
	}
	if source == target {
		return nil, ErrSelfCorrelation
	}
	if cmd.Probability < 0 || cmd.Probability > 100 {
		return nil, ErrProbabilityRange
	}

	// Normalize pair: always store the lexicographically smaller tag as source so that
	// (A, B) and (B, A) are treated as the same correlation.
	if source > target {
		source, target = target, source
	}

	correlation := &tagdomain.TagCorrelation{
		SourceTag:   source,
		TargetTag:   target,
		Probability: cmd.Probability,
		Notes:       strings.TrimSpace(cmd.Notes),
	}

	if err := s.repository.CreateCorrelation(ctx, correlation); err != nil {
		return nil, fmt.Errorf("create correlation: %w", err)
	}

	return correlation, nil
}

// UpdateCorrelation updates a tag correlation by ID.
func (s *TagService) UpdateCorrelation(ctx context.Context, id uint, cmd UpdateCorrelationCommand) (*tagdomain.TagCorrelation, error) {
	var prob *float64
	var notes *string

	if cmd.HasProbability {
		if cmd.Probability == nil || *cmd.Probability < 0 || *cmd.Probability > 100 {
			return nil, ErrProbabilityRange
		}
		prob = cmd.Probability
	}
	if cmd.HasNotes {
		notes = cmd.Notes
	}

	updated, err := s.repository.UpdateCorrelation(ctx, id, prob, notes)
	if err != nil {
		return nil, fmt.Errorf("update correlation: %w", err)
	}

	return updated, nil
}

// DeleteCorrelation deletes a tag correlation by ID.
func (s *TagService) DeleteCorrelation(ctx context.Context, id uint) error {
	if err := s.repository.DeleteCorrelation(ctx, id); err != nil {
		return fmt.Errorf("delete correlation: %w", err)
	}

	return nil
}
