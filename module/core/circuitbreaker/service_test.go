package circuitbreaker

import (
	errorspkg "errors"
	"testing"
	"time"
)

// TestNewDefaults verifies default config normalization behavior.
func TestNewDefaults(t *testing.T) {
	breaker, err := New(Config{}, nil)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if breaker == nil {
		t.Fatalf("expected non-nil breaker")
	}
	if breaker.State() != StateClosed {
		t.Fatalf("breaker.State() = %q, want %q", breaker.State(), StateClosed)
	}
}

// TestExecuteValidation verifies execute validation behavior.
func TestExecuteValidation(t *testing.T) {
	breaker, err := New(Config{}, nil)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if executeErr := breaker.Execute(nil); !errorspkg.Is(executeErr, ErrNilOperation) {
		t.Fatalf("Execute(nil) error = %v, want ErrNilOperation", executeErr)
	}
}

// TestExecuteSuccess verifies successful execution behavior.
func TestExecuteSuccess(t *testing.T) {
	breaker, err := New(Config{}, nil)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if executeErr := breaker.Execute(func() error { return nil }); executeErr != nil {
		t.Fatalf("Execute() error = %v", executeErr)
	}
}

// TestExecuteOpenState verifies open-state rejection behavior.
func TestExecuteOpenState(t *testing.T) {
	breaker, err := New(Config{
		FailureThreshold: 1,
		TimeoutMS:        1000,
		IntervalMS:       1000,
	}, nil)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	failure := errorspkg.New("operation failed")
	if executeErr := breaker.Execute(func() error { return failure }); !errorspkg.Is(executeErr, failure) {
		t.Fatalf("Execute(first failure) error = %v, want failure", executeErr)
	}

	openErr := breaker.Execute(func() error { return nil })
	if !breaker.IsOpenError(openErr) {
		t.Fatalf("expected open-state error, got %v", openErr)
	}
	if breaker.State() != StateOpen {
		t.Fatalf("breaker.State() = %q, want %q", breaker.State(), StateOpen)
	}
}

// TestStateToString verifies private state string mapping behavior.
func TestStateToString(t *testing.T) {
	if stateToString(0) == "" {
		t.Fatalf("stateToString(default) should not be empty")
	}
}

// TestOpenToHalfOpenTransition verifies timeout-based transition behavior.
func TestOpenToHalfOpenTransition(t *testing.T) {
	breaker, err := New(Config{
		FailureThreshold: 1,
		TimeoutMS:        20,
		IntervalMS:       1000,
		MaxRequests:      1,
	}, nil)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	_ = breaker.Execute(func() error { return errorspkg.New("failed") })
	time.Sleep(30 * time.Millisecond)

	if executeErr := breaker.Execute(func() error { return nil }); executeErr != nil {
		t.Fatalf("Execute(half-open trial) error = %v", executeErr)
	}
	if breaker.State() != StateClosed {
		t.Fatalf("breaker.State() = %q, want %q", breaker.State(), StateClosed)
	}
}

// TestNewBreaker verifies abstract breaker factory behavior.
func TestNewBreaker(t *testing.T) {
	breaker, err := NewBreaker(Config{Name: "abstract"}, nil)
	if err != nil {
		t.Fatalf("NewBreaker() error = %v", err)
	}
	if breaker == nil {
		t.Fatalf("expected abstract breaker")
	}
}
