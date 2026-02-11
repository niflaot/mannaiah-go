package circuitbreaker

import (
	"errors"
	"time"

	gobreaker "github.com/sony/gobreaker"
	"go.uber.org/zap"
)

var (
	// ErrNilOperation is returned when nil operations are executed.
	ErrNilOperation = errors.New("circuit breaker operation must not be nil")
)

// State defines circuit breaker state values.
type State string

const (
	// StateClosed defines closed-state values.
	StateClosed State = "closed"
	// StateHalfOpen defines half-open-state values.
	StateHalfOpen State = "half_open"
	// StateOpen defines open-state values.
	StateOpen State = "open"
)

// Config defines circuit breaker configuration values.
type Config struct {
	// Name defines circuit breaker names.
	Name string
	// MaxRequests defines half-open trial request limits.
	MaxRequests uint32
	// IntervalMS defines closed-state rolling counter reset intervals in milliseconds.
	IntervalMS int
	// TimeoutMS defines open-state timeout windows in milliseconds.
	TimeoutMS int
	// FailureThreshold defines consecutive-failure thresholds that open breakers.
	FailureThreshold uint32
}

// Breaker defines circuit breaker behavior.
type Breaker interface {
	// Execute runs operations through the circuit breaker.
	Execute(operation func() error) error
	// State reports current circuit breaker state values.
	State() State
	// IsOpenError reports whether errors represent open-circuit rejections.
	IsOpenError(err error) bool
}

// Service defines gobreaker-backed circuit breaker behavior.
type Service struct {
	// name defines circuit breaker names.
	name string
	// breaker defines underlying gobreaker dependencies.
	breaker *gobreaker.CircuitBreaker
	// logger defines structured logger dependencies.
	logger *zap.Logger
}

var (
	// _ ensures Service satisfies Breaker contracts.
	_ Breaker = (*Service)(nil)
)

// New creates gobreaker-backed circuit breaker services.
func New(cfg Config, providedLogger *zap.Logger) (*Service, error) {
	resolved := normalizeConfig(cfg)
	logger := resolveLogger(providedLogger)

	service := &Service{
		name:   resolved.Name,
		logger: logger,
	}

	service.breaker = gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        resolved.Name,
		MaxRequests: resolved.MaxRequests,
		Interval:    time.Duration(resolved.IntervalMS) * time.Millisecond,
		Timeout:     time.Duration(resolved.TimeoutMS) * time.Millisecond,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= resolved.FailureThreshold
		},
		OnStateChange: service.onStateChange,
	})

	return service, nil
}

// NewBreaker creates abstract circuit breaker dependencies.
func NewBreaker(cfg Config, providedLogger *zap.Logger) (Breaker, error) {
	return New(cfg, providedLogger)
}

// Execute runs operations through the circuit breaker.
func (s *Service) Execute(operation func() error) error {
	if operation == nil {
		return ErrNilOperation
	}

	_, err := s.breaker.Execute(func() (interface{}, error) {
		return nil, operation()
	})
	return err
}

// State reports current circuit breaker state values.
func (s *Service) State() State {
	switch s.breaker.State() {
	case gobreaker.StateOpen:
		return StateOpen
	case gobreaker.StateHalfOpen:
		return StateHalfOpen
	default:
		return StateClosed
	}
}

// IsOpenError reports whether errors represent open-circuit rejections.
func (s *Service) IsOpenError(err error) bool {
	return errors.Is(err, gobreaker.ErrOpenState) || errors.Is(err, gobreaker.ErrTooManyRequests)
}

// onStateChange logs circuit breaker state transition values.
func (s *Service) onStateChange(name string, from gobreaker.State, to gobreaker.State) {
	s.logger.Warn(
		"circuit breaker state changed",
		zap.String("name", name),
		zap.String("from", stateToString(from)),
		zap.String("to", stateToString(to)),
	)
}

// normalizeConfig resolves default circuit breaker configuration values.
func normalizeConfig(cfg Config) Config {
	resolved := cfg
	if resolved.Name == "" {
		resolved.Name = "circuit-breaker"
	}
	if resolved.MaxRequests == 0 {
		resolved.MaxRequests = 1
	}
	if resolved.IntervalMS <= 0 {
		resolved.IntervalMS = 60000
	}
	if resolved.TimeoutMS <= 0 {
		resolved.TimeoutMS = 30000
	}
	if resolved.FailureThreshold == 0 {
		resolved.FailureThreshold = 5
	}

	return resolved
}

// resolveLogger resolves nil loggers to no-op defaults.
func resolveLogger(providedLogger *zap.Logger) *zap.Logger {
	if providedLogger != nil {
		return providedLogger
	}

	return zap.NewNop()
}

// stateToString maps gobreaker state values to log-friendly values.
func stateToString(value gobreaker.State) string {
	switch value {
	case gobreaker.StateOpen:
		return string(StateOpen)
	case gobreaker.StateHalfOpen:
		return string(StateHalfOpen)
	default:
		return string(StateClosed)
	}
}
