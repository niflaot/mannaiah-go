package service

import (
	"context"
	"errors"
	"testing"
	"time"

	syncdomain "mannaiah/module/falabella/domain/sync"
	"mannaiah/module/falabella/port"
)

// repoMock defines sync status repository behavior for service tests.
type repoMock struct {
	// executions defines stored execution parent values by execution ID.
	executions map[string]*syncdomain.SyncExecution
	// entries defines stored entries by feed ID.
	entries map[string]*syncdomain.SyncEntry
	// productEntries defines stored entries by product ID.
	productEntries map[string][]syncdomain.SyncEntry
	// createErr defines Create() errors.
	createErr error
	// updateErr defines UpdateStatus() errors.
	updateErr error
	// listPendingErr defines ListPending() errors.
	listPendingErr error
}

// EnsureSchema returns nil.
func (m *repoMock) EnsureSchema(ctx context.Context) error { return nil }

// CreateExecution persists parent execution records.
func (m *repoMock) CreateExecution(ctx context.Context, execution *syncdomain.SyncExecution) error {
	if execution == nil {
		return errors.New("execution must not be nil")
	}
	if m.executions == nil {
		m.executions = map[string]*syncdomain.SyncExecution{}
	}
	m.executions[execution.ExecutionID] = execution
	return nil
}

// Create persists entries or returns configured errors.
func (m *repoMock) Create(ctx context.Context, entry *syncdomain.SyncEntry) error {
	if m.createErr != nil {
		return m.createErr
	}
	if m.entries == nil {
		m.entries = map[string]*syncdomain.SyncEntry{}
	}
	m.entries[entry.FeedID] = entry
	return nil
}

// GetExecutionByID retrieves parent execution values.
func (m *repoMock) GetExecutionByID(ctx context.Context, executionID string) (*syncdomain.SyncExecution, error) {
	if m.executions == nil {
		return nil, port.ErrSyncExecutionNotFound
	}
	execution, ok := m.executions[executionID]
	if !ok {
		return nil, port.ErrSyncExecutionNotFound
	}
	return execution, nil
}

// GetByFeedID retrieves entries by feed ID.
func (m *repoMock) GetByFeedID(ctx context.Context, feedID string) (*syncdomain.SyncEntry, error) {
	if m.entries == nil {
		return nil, port.ErrSyncEntryNotFound
	}
	entry, ok := m.entries[feedID]
	if !ok {
		return nil, port.ErrSyncEntryNotFound
	}
	return entry, nil
}

// ListByExecutionID retrieves child feed rows for one execution id.
func (m *repoMock) ListByExecutionID(ctx context.Context, executionID string) ([]syncdomain.SyncEntry, error) {
	entries := make([]syncdomain.SyncEntry, 0)
	for _, entry := range m.entries {
		if entry != nil && entry.ExecutionID == executionID {
			entries = append(entries, *entry)
		}
	}
	return entries, nil
}

// GetByProductID retrieves entries by product ID.
func (m *repoMock) GetByProductID(ctx context.Context, productID string) ([]syncdomain.SyncEntry, error) {
	if m.productEntries == nil {
		return nil, nil
	}
	return m.productEntries[productID], nil
}

// UpdateStatus updates entry status.
func (m *repoMock) UpdateStatus(ctx context.Context, feedID string, status syncdomain.SyncStatus, resolvedAt *time.Time) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	if m.entries == nil {
		return port.ErrSyncEntryNotFound
	}
	entry, ok := m.entries[feedID]
	if !ok {
		return port.ErrSyncEntryNotFound
	}
	entry.Status = status
	entry.ResolvedAt = resolvedAt
	return nil
}

// ListPending retrieves pending entries.
func (m *repoMock) ListPending(ctx context.Context, limit int) ([]syncdomain.SyncEntry, error) {
	if m.listPendingErr != nil {
		return nil, m.listPendingErr
	}

	var pending []syncdomain.SyncEntry
	for _, entry := range m.entries {
		if entry.Status == syncdomain.SyncStatusPending {
			pending = append(pending, *entry)
		}
	}
	if limit > 0 && len(pending) > limit {
		pending = pending[:limit]
	}

	return pending, nil
}

// sourceMock defines feed status source behavior for service tests.
type sourceMock struct {
	// payload defines GetFeedStatus() payload values.
	payload []byte
	// err defines GetFeedStatus() errors.
	err error
}

// GetFeedStatus returns configured payload/errors.
func (m *sourceMock) GetFeedStatus(ctx context.Context, feedID string) ([]byte, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.payload, nil
}

const feedStatusFinishedSuccessXML = `<?xml version="1.0" encoding="UTF-8"?>
<SuccessResponse>
  <Head>
    <RequestId/>
    <RequestAction>FeedStatus</RequestAction>
    <ResponseType>FeedDetail</ResponseType>
  </Head>
  <Body>
    <FeedDetail>
      <Feed>feed-abc</Feed>
      <Status>Finished</Status>
      <Action>ProductCreate</Action>
      <TotalRecords>1</TotalRecords>
      <ProcessedRecords>1</ProcessedRecords>
      <FailedRecords>0</FailedRecords>
    </FeedDetail>
  </Body>
</SuccessResponse>`

const feedStatusFinishedFailedXML = `<?xml version="1.0" encoding="UTF-8"?>
<SuccessResponse>
  <Head>
    <RequestId/>
    <RequestAction>FeedStatus</RequestAction>
    <ResponseType>FeedDetail</ResponseType>
  </Head>
  <Body>
    <FeedDetail>
      <Feed>feed-abc</Feed>
      <Status>Finished</Status>
      <Action>ProductCreate</Action>
      <TotalRecords>1</TotalRecords>
      <ProcessedRecords>1</ProcessedRecords>
      <FailedRecords>1</FailedRecords>
      <FeedErrors>
        <Error>
          <Code>0</Code>
          <Message>Invalid brand</Message>
          <SellerSku>SKU-001</SellerSku>
        </Error>
      </FeedErrors>
    </FeedDetail>
  </Body>
</SuccessResponse>`

const feedStatusPendingXML = `<?xml version="1.0" encoding="UTF-8"?>
<SuccessResponse>
  <Head>
    <RequestId/>
    <RequestAction>FeedStatus</RequestAction>
    <ResponseType>FeedDetail</ResponseType>
  </Head>
  <Body>
    <FeedDetail>
      <Feed>feed-abc</Feed>
      <Status>Queued</Status>
      <Action>ProductCreate</Action>
      <TotalRecords>1</TotalRecords>
      <ProcessedRecords>0</ProcessedRecords>
      <FailedRecords>0</FailedRecords>
    </FeedDetail>
  </Body>
</SuccessResponse>`

// TestNewServiceValidation verifies constructor validation behavior.
func TestNewServiceValidation(t *testing.T) {
	if _, err := NewService(nil, &sourceMock{}); !errors.Is(err, ErrNilRepository) {
		t.Fatalf("NewService(nil repo) error = %v, want ErrNilRepository", err)
	}
	if _, err := NewService(&repoMock{}, nil); !errors.Is(err, ErrNilSource) {
		t.Fatalf("NewService(nil source) error = %v, want ErrNilSource", err)
	}
}

// TestRecordEntry verifies entry persistence behavior.
func TestRecordEntry(t *testing.T) {
	repo := &repoMock{}
	svc, err := NewService(repo, &sourceMock{})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	entry := &syncdomain.SyncEntry{
		ProductID: "prod-1",
		SKU:       "SKU-001",
		FeedID:    "feed-abc",
		Action:    syncdomain.SyncActionCreate,
		Status:    syncdomain.SyncStatusPending,
		SyncedAt:  time.Now().UTC(),
	}
	if recordErr := svc.RecordEntry(context.Background(), entry); recordErr != nil {
		t.Fatalf("RecordEntry() error = %v", recordErr)
	}
}

// TestRecordEntryNil verifies nil entry error behavior.
func TestRecordEntryNil(t *testing.T) {
	svc, _ := NewService(&repoMock{}, &sourceMock{})
	if err := svc.RecordEntry(context.Background(), nil); err == nil {
		t.Fatalf("RecordEntry(nil) expected error")
	}
}

// TestGetByFeedID verifies feed ID lookup behavior.
func TestGetByFeedID(t *testing.T) {
	repo := &repoMock{entries: map[string]*syncdomain.SyncEntry{
		"feed-abc": {FeedID: "feed-abc", ProductID: "prod-1"},
	}}
	svc, _ := NewService(repo, &sourceMock{})

	entry, err := svc.GetByFeedID(context.Background(), "feed-abc")
	if err != nil {
		t.Fatalf("GetByFeedID() error = %v", err)
	}
	if entry.ProductID != "prod-1" {
		t.Fatalf("ProductID = %q, want %q", entry.ProductID, "prod-1")
	}
}

// TestGetExecutionByID verifies execution lookup behavior.
func TestGetExecutionByID(t *testing.T) {
	repo := &repoMock{executions: map[string]*syncdomain.SyncExecution{"exec-1": {ExecutionID: "exec-1", StartedAt: time.Now().UTC()}}}
	svc, _ := NewService(repo, &sourceMock{})

	execution, err := svc.GetExecutionByID(context.Background(), "exec-1")
	if err != nil {
		t.Fatalf("GetExecutionByID() error = %v", err)
	}
	if execution.ExecutionID != "exec-1" {
		t.Fatalf("ExecutionID = %q, want %q", execution.ExecutionID, "exec-1")
	}
}

// TestGetByExecutionID verifies child feed lookup by execution behavior.
func TestGetByExecutionID(t *testing.T) {
	repo := &repoMock{entries: map[string]*syncdomain.SyncEntry{
		"feed-1": {ExecutionID: "exec-1", FeedID: "feed-1", ProductID: "prod-1"},
		"feed-2": {ExecutionID: "exec-1", FeedID: "feed-2", ProductID: "prod-1"},
		"feed-3": {ExecutionID: "exec-2", FeedID: "feed-3", ProductID: "prod-1"},
	}}
	svc, _ := NewService(repo, &sourceMock{})

	entries, err := svc.GetByExecutionID(context.Background(), "exec-1")
	if err != nil {
		t.Fatalf("GetByExecutionID() error = %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("len(entries) = %d, want %d", len(entries), 2)
	}
}

// TestGetByFeedIDEmpty verifies empty feed ID validation.
func TestGetByFeedIDEmpty(t *testing.T) {
	svc, _ := NewService(&repoMock{}, &sourceMock{})
	if _, err := svc.GetByFeedID(context.Background(), "  "); !errors.Is(err, ErrInvalidFeedID) {
		t.Fatalf("GetByFeedID(empty) error = %v, want ErrInvalidFeedID", err)
	}
}

// TestGetByProductID verifies product ID lookup behavior.
func TestGetByProductID(t *testing.T) {
	repo := &repoMock{productEntries: map[string][]syncdomain.SyncEntry{
		"prod-1": {{FeedID: "feed-abc", ProductID: "prod-1"}},
	}}
	svc, _ := NewService(repo, &sourceMock{})

	entries, err := svc.GetByProductID(context.Background(), "prod-1")
	if err != nil {
		t.Fatalf("GetByProductID() error = %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("len(entries) = %d, want %d", len(entries), 1)
	}
}

// TestGetByProductIDEmpty verifies empty product ID validation.
func TestGetByProductIDEmpty(t *testing.T) {
	svc, _ := NewService(&repoMock{}, &sourceMock{})
	if _, err := svc.GetByProductID(context.Background(), "  "); !errors.Is(err, ErrInvalidProductID) {
		t.Fatalf("GetByProductID(empty) error = %v, want ErrInvalidProductID", err)
	}
}

// TestResolveFeedStatusSuccess verifies successful feed resolution behavior.
func TestResolveFeedStatusSuccess(t *testing.T) {
	repo := &repoMock{entries: map[string]*syncdomain.SyncEntry{
		"feed-abc": {FeedID: "feed-abc", Status: syncdomain.SyncStatusPending},
	}}
	svc, _ := NewService(repo, &sourceMock{payload: []byte(feedStatusFinishedSuccessXML)})

	result, err := svc.ResolveFeedStatus(context.Background(), "feed-abc")
	if err != nil {
		t.Fatalf("ResolveFeedStatus() error = %v", err)
	}
	if result.FeedID != "feed-abc" {
		t.Fatalf("FeedID = %q, want %q", result.FeedID, "feed-abc")
	}
	if result.Status != "Finished" {
		t.Fatalf("Status = %q, want %q", result.Status, "Finished")
	}
	if result.FailedRecords != 0 {
		t.Fatalf("FailedRecords = %d, want %d", result.FailedRecords, 0)
	}

	entry := repo.entries["feed-abc"]
	if entry.Status != syncdomain.SyncStatusFinished {
		t.Fatalf("persisted status = %q, want %q", entry.Status, syncdomain.SyncStatusFinished)
	}
	if entry.ResolvedAt == nil {
		t.Fatalf("ResolvedAt should not be nil")
	}
}

// TestResolveFeedStatusFailed verifies failed feed resolution behavior.
func TestResolveFeedStatusFailed(t *testing.T) {
	repo := &repoMock{entries: map[string]*syncdomain.SyncEntry{
		"feed-abc": {FeedID: "feed-abc", Status: syncdomain.SyncStatusPending},
	}}
	svc, _ := NewService(repo, &sourceMock{payload: []byte(feedStatusFinishedFailedXML)})

	result, err := svc.ResolveFeedStatus(context.Background(), "feed-abc")
	if err != nil {
		t.Fatalf("ResolveFeedStatus() error = %v", err)
	}
	if result.FailedRecords != 1 {
		t.Fatalf("FailedRecords = %d, want %d", result.FailedRecords, 1)
	}
	if len(result.Errors) != 1 {
		t.Fatalf("len(Errors) = %d, want %d", len(result.Errors), 1)
	}
	if result.Errors[0].Message != "Invalid brand" {
		t.Fatalf("Errors[0].Message = %q, want %q", result.Errors[0].Message, "Invalid brand")
	}

	entry := repo.entries["feed-abc"]
	if entry.Status != syncdomain.SyncStatusFailed {
		t.Fatalf("persisted status = %q, want %q", entry.Status, syncdomain.SyncStatusFailed)
	}
}

// TestResolveFeedStatusPending verifies pending feed error behavior.
func TestResolveFeedStatusPending(t *testing.T) {
	svc, _ := NewService(&repoMock{}, &sourceMock{payload: []byte(feedStatusPendingXML)})

	_, err := svc.ResolveFeedStatus(context.Background(), "feed-abc")
	if !errors.Is(err, ErrFeedNotFinished) {
		t.Fatalf("ResolveFeedStatus(pending) error = %v, want ErrFeedNotFinished", err)
	}
}

// TestResolveFeedStatusEmpty verifies empty feed ID validation.
func TestResolveFeedStatusEmpty(t *testing.T) {
	svc, _ := NewService(&repoMock{}, &sourceMock{})
	if _, err := svc.ResolveFeedStatus(context.Background(), " "); !errors.Is(err, ErrInvalidFeedID) {
		t.Fatalf("ResolveFeedStatus(empty) error = %v, want ErrInvalidFeedID", err)
	}
}

// TestResolveFeedStatusSourceError verifies source error propagation behavior.
func TestResolveFeedStatusSourceError(t *testing.T) {
	svc, _ := NewService(&repoMock{}, &sourceMock{err: errors.New("api down")})
	if _, err := svc.ResolveFeedStatus(context.Background(), "feed-abc"); err == nil {
		t.Fatalf("ResolveFeedStatus() expected source error")
	}
}

// TestResolveFeedStatusInvalidXML verifies invalid XML error behavior.
func TestResolveFeedStatusInvalidXML(t *testing.T) {
	svc, _ := NewService(&repoMock{}, &sourceMock{payload: []byte("not-xml")})
	if _, err := svc.ResolveFeedStatus(context.Background(), "feed-abc"); err == nil {
		t.Fatalf("ResolveFeedStatus() expected unmarshal error")
	}
}

// TestResolveFeedStatusEntryNotFound verifies resolution proceeds when entry is not persisted.
func TestResolveFeedStatusEntryNotFound(t *testing.T) {
	svc, _ := NewService(&repoMock{}, &sourceMock{payload: []byte(feedStatusFinishedSuccessXML)})

	result, err := svc.ResolveFeedStatus(context.Background(), "feed-abc")
	if err != nil {
		t.Fatalf("ResolveFeedStatus() error = %v", err)
	}
	if result.FeedID != "feed-abc" {
		t.Fatalf("FeedID = %q, want %q", result.FeedID, "feed-abc")
	}
}

// TestResolveFeedStatusUpdateError verifies non-not-found update error propagation.
func TestResolveFeedStatusUpdateError(t *testing.T) {
	repo := &repoMock{
		entries:   map[string]*syncdomain.SyncEntry{"feed-abc": {FeedID: "feed-abc"}},
		updateErr: errors.New("db crash"),
	}
	svc, _ := NewService(repo, &sourceMock{payload: []byte(feedStatusFinishedSuccessXML)})

	if _, err := svc.ResolveFeedStatus(context.Background(), "feed-abc"); err == nil {
		t.Fatalf("ResolveFeedStatus() expected update error")
	}
}