package cron

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	cronv3 "github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

var (
	// ErrInvalidLocation is returned when CRON_LOCATION cannot be resolved.
	ErrInvalidLocation = errors.New("cron location is invalid")
	// ErrEmptySpec is returned when cron expressions are blank.
	ErrEmptySpec = errors.New("cron spec must not be empty")
	// ErrInvalidSpec is returned when cron expressions cannot be parsed.
	ErrInvalidSpec = errors.New("cron spec is invalid")
	// ErrNilJob is returned when a nil job interface is provided.
	ErrNilJob = errors.New("cron job must not be nil")
	// ErrNilFunc is returned when a nil function job is provided.
	ErrNilFunc = errors.New("cron function must not be nil")
)

// Job defines scheduler-executable units independent of provider types.
type Job interface {
	// Run executes the scheduled workload.
	Run()
}

// JobFunc adapts function values into the abstract Job contract.
type JobFunc func()

// Run executes a function-backed job value.
func (f JobFunc) Run() {
	f()
}

// EntryID defines provider-agnostic scheduled entry identifiers.
type EntryID int

// Entry defines provider-agnostic scheduled entry metadata.
type Entry struct {
	// ID defines the scheduled entry identifier.
	ID EntryID
	// Spec defines the original cron expression used for registration.
	Spec string
	// Next defines the next planned run timestamp.
	Next time.Time
	// Prev defines the previous run timestamp.
	Prev time.Time
}

// Scheduler defines provider-agnostic scheduling capabilities.
type Scheduler interface {
	// Add registers a job under a cron expression and returns an entry identifier.
	Add(spec string, job Job) (EntryID, error)
	// AddFunc registers a function job under a cron expression and returns an entry identifier.
	AddFunc(spec string, job func()) (EntryID, error)
	// Remove removes a scheduled entry by identifier.
	Remove(id EntryID)
	// Entries returns current scheduled entries.
	Entries() []Entry
	// Start starts asynchronous cron scheduling.
	Start()
	// Run starts blocking cron scheduling until the scheduler is stopped.
	Run()
	// Stop requests scheduler shutdown and waits for running jobs or context cancellation.
	Stop(ctx context.Context) error
}

var (
	// _ ensures Service satisfies the abstract Scheduler contract.
	_ Scheduler = (*Service)(nil)
)

// Service defines a robfig-backed scheduler adapter.
type Service struct {
	// instance defines the underlying robfig cron engine.
	instance *cronv3.Cron
	// logger defines structured logging dependency.
	logger *zap.Logger
	// mutex guards entry metadata state.
	mutex sync.RWMutex
	// specs stores original expressions keyed by entry id.
	specs map[EntryID]string
}

// New creates a scheduler service from config and optional logger.
func New(cfg Config, providedLogger *zap.Logger) (*Service, error) {
	location, err := loadLocation(cfg.Location)
	if err != nil {
		return nil, err
	}

	options := []cronv3.Option{cronv3.WithLocation(location)}
	if cfg.WithSeconds {
		options = append(options, cronv3.WithSeconds())
	}

	return &Service{
		instance: cronv3.New(options...),
		logger:   resolveLogger(providedLogger),
		specs:    make(map[EntryID]string),
	}, nil
}

// NewScheduler creates a provider-agnostic scheduler.
func NewScheduler(cfg Config, providedLogger *zap.Logger) (Scheduler, error) {
	return New(cfg, providedLogger)
}

// Add registers a job under a cron expression and returns an entry identifier.
func (s *Service) Add(spec string, job Job) (EntryID, error) {
	normalizedSpec, err := normalizeSpec(spec)
	if err != nil {
		return 0, err
	}
	if job == nil {
		return 0, ErrNilJob
	}

	id, addErr := s.instance.AddJob(normalizedSpec, s.wrapJob(job))
	if addErr != nil {
		return 0, fmt.Errorf("%w: %v", ErrInvalidSpec, addErr)
	}

	entryID := EntryID(id)
	s.mutex.Lock()
	s.specs[entryID] = normalizedSpec
	s.mutex.Unlock()

	return entryID, nil
}

// AddFunc registers a function job under a cron expression and returns an entry identifier.
func (s *Service) AddFunc(spec string, job func()) (EntryID, error) {
	if job == nil {
		return 0, ErrNilFunc
	}

	return s.Add(spec, JobFunc(job))
}

// Remove removes a scheduled entry by identifier.
func (s *Service) Remove(id EntryID) {
	s.instance.Remove(cronv3.EntryID(id))
	s.mutex.Lock()
	delete(s.specs, id)
	s.mutex.Unlock()
}

// Entries returns current scheduled entries.
func (s *Service) Entries() []Entry {
	rawEntries := s.instance.Entries()
	entries := make([]Entry, 0, len(rawEntries))

	s.mutex.RLock()
	for _, raw := range rawEntries {
		entryID := EntryID(raw.ID)
		entries = append(entries, Entry{
			ID:   entryID,
			Spec: s.specs[entryID],
			Next: raw.Next,
			Prev: raw.Prev,
		})
	}
	s.mutex.RUnlock()

	return entries
}

// Start starts asynchronous cron scheduling.
func (s *Service) Start() {
	s.instance.Start()
}

// Run starts blocking cron scheduling until the scheduler is stopped.
func (s *Service) Run() {
	s.instance.Run()
}

// Stop requests scheduler shutdown and waits for running jobs or context cancellation.
func (s *Service) Stop(ctx context.Context) error {
	done := s.instance.Stop()
	stopContext := resolveStopContext(ctx)

	select {
	case <-done.Done():
		return nil
	case <-stopContext.Done():
		return stopContext.Err()
	}
}

// resolveLogger resolves nil loggers to no-op defaults.
func resolveLogger(providedLogger *zap.Logger) *zap.Logger {
	if providedLogger != nil {
		return providedLogger
	}

	return zap.NewNop()
}

// loadLocation resolves scheduler locations and returns startup validation errors.
func loadLocation(location string) (*time.Location, error) {
	normalized := strings.TrimSpace(location)
	if normalized == "" {
		normalized = "UTC"
	}

	resolved, err := time.LoadLocation(normalized)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidLocation, normalized)
	}

	return resolved, nil
}

// normalizeSpec validates and normalizes cron expression values.
func normalizeSpec(spec string) (string, error) {
	normalized := strings.TrimSpace(spec)
	if normalized == "" {
		return "", ErrEmptySpec
	}

	return normalized, nil
}

// resolveStopContext normalizes nil stop contexts.
func resolveStopContext(ctx context.Context) context.Context {
	if ctx != nil {
		return ctx
	}

	return context.Background()
}

// wrapJob applies panic recovery and structured logs to scheduled jobs.
func (s *Service) wrapJob(job Job) cronv3.Job {
	return cronv3.FuncJob(func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				s.logger.Error(
					"cron job panic recovered",
					zap.Any("panic", recovered),
				)
			}
		}()

		job.Run()
	})
}
