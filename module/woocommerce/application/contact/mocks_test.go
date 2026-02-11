package contact

import (
	"context"
	"sync"

	"mannaiah/module/woocommerce/port"
)

// sourceMock defines order source behavior for sync tests.
type sourceMock struct {
	// validateErr defines validation errors.
	validateErr error
	// pages defines paginated order responses.
	pages [][]port.WooOrder
	// listErrAtPage defines page numbers that should return list errors.
	listErrAtPage map[int]error
}

// Validate verifies source connectivity.
func (m *sourceMock) Validate(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	return m.validateErr
}

// ListOrders retrieves paginated order values.
func (m *sourceMock) ListOrders(ctx context.Context, page int, pageSize int) (orders []port.WooOrder, hasNext bool, err error) {
	if err := ctx.Err(); err != nil {
		return nil, false, err
	}
	if listErr, hasError := m.listErrAtPage[page]; hasError {
		return nil, false, listErr
	}
	if page <= 0 || page > len(m.pages) {
		return nil, false, nil
	}

	items := m.pages[page-1]
	return items, page < len(m.pages), nil
}

// targetMock defines contact sync target behavior for sync tests.
type targetMock struct {
	// mu guards state mutation for concurrent workers.
	mu sync.Mutex
	// outcomes defines upsert outcomes keyed by email.
	outcomes map[string]port.UpsertOutcome
	// errors defines upsert errors keyed by email.
	errors map[string]error
	// commands stores received upsert commands.
	commands []port.ContactSyncCommand
}

// UpsertByEmail creates or updates contacts by email.
func (m *targetMock) UpsertByEmail(ctx context.Context, command port.ContactSyncCommand) (outcome port.UpsertOutcome, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.commands = append(m.commands, command)
	if err := m.errors[command.Email]; err != nil {
		return "", err
	}
	if outcome, ok := m.outcomes[command.Email]; ok {
		return outcome, nil
	}

	return port.UpsertOutcomeUpdated, nil
}

// publisherMock defines integration event publication behavior for sync tests.
type publisherMock struct {
	// events stores published integration events.
	events []port.IntegrationEvent
	// mu guards state mutation for concurrent event publication.
	mu sync.Mutex
}

// Publish captures integration events.
func (m *publisherMock) Publish(ctx context.Context, event port.IntegrationEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.events = append(m.events, event)
	return nil
}

// circuitBreakerMock defines circuit-breaker behavior for sync tests.
type circuitBreakerMock struct {
	// executeErr defines forced execute errors.
	executeErr error
	// openError defines whether executeErr should be classified as open-state.
	openError bool
	// executions defines executed operation counts.
	executions int
}

// Execute runs operations with controlled error injection.
func (m *circuitBreakerMock) Execute(operation func() error) error {
	if m.executeErr != nil {
		return m.executeErr
	}

	m.executions++
	return operation()
}

// IsOpenError reports open-state classifications.
func (m *circuitBreakerMock) IsOpenError(err error) bool {
	return m.openError
}
