package service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	syncdomain "mannaiah/module/falabella/domain/sync"
)

// TestResolvePendingFeedsAllResolved verifies all pending feeds resolve successfully.
func TestResolvePendingFeedsAllResolved(t *testing.T) {
	repo := &repoMock{
		entries: map[string]*syncdomain.SyncEntry{
			"feed-1": {FeedID: "feed-1", Status: syncdomain.SyncStatusPending, SyncedAt: time.Now()},
			"feed-2": {FeedID: "feed-2", Status: syncdomain.SyncStatusPending, SyncedAt: time.Now()},
		},
	}
	source := &sourceMock{payload: []byte(feedStatusFinishedSuccessXML)}

	svc, err := NewService(repo, source)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	result, resolveErr := svc.ResolvePendingFeeds(context.Background(), 50)
	if resolveErr != nil {
		t.Fatalf("ResolvePendingFeeds() error = %v", resolveErr)
	}
	if result.Checked != 2 {
		t.Fatalf("Checked = %d, want %d", result.Checked, 2)
	}
	if result.Resolved != 2 {
		t.Fatalf("Resolved = %d, want %d", result.Resolved, 2)
	}
	if result.StillPending != 0 {
		t.Fatalf("StillPending = %d, want %d", result.StillPending, 0)
	}
	if result.Errored != 0 {
		t.Fatalf("Errored = %d, want %d", result.Errored, 0)
	}
}

// TestResolvePendingFeedsStillPending verifies feeds that are not finished remain pending.
func TestResolvePendingFeedsStillPending(t *testing.T) {
	repo := &repoMock{
		entries: map[string]*syncdomain.SyncEntry{
			"feed-queued": {FeedID: "feed-queued", Status: syncdomain.SyncStatusPending, SyncedAt: time.Now()},
		},
	}
	source := &sourceMock{payload: []byte(feedStatusPendingXML)}

	svc, err := NewService(repo, source)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	result, resolveErr := svc.ResolvePendingFeeds(context.Background(), 50)
	if resolveErr != nil {
		t.Fatalf("ResolvePendingFeeds() error = %v", resolveErr)
	}
	if result.Checked != 1 {
		t.Fatalf("Checked = %d, want %d", result.Checked, 1)
	}
	if result.StillPending != 1 {
		t.Fatalf("StillPending = %d, want %d", result.StillPending, 1)
	}
	if result.Resolved != 0 {
		t.Fatalf("Resolved = %d, want %d", result.Resolved, 0)
	}
}

// TestResolvePendingFeedsSourceError verifies source errors increment the errored counter.
func TestResolvePendingFeedsSourceError(t *testing.T) {
	repo := &repoMock{
		entries: map[string]*syncdomain.SyncEntry{
			"feed-err": {FeedID: "feed-err", Status: syncdomain.SyncStatusPending, SyncedAt: time.Now()},
		},
	}
	source := &sourceMock{err: errors.New("upstream down")}

	svc, err := NewService(repo, source)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	result, resolveErr := svc.ResolvePendingFeeds(context.Background(), 50)
	if resolveErr != nil {
		t.Fatalf("ResolvePendingFeeds() error = %v", resolveErr)
	}
	if result.Errored != 1 {
		t.Fatalf("Errored = %d, want %d", result.Errored, 1)
	}
	if result.Resolved != 0 {
		t.Fatalf("Resolved = %d, want %d", result.Resolved, 0)
	}
}

// TestResolvePendingFeedsNoPending verifies empty pending set returns zero counts.
func TestResolvePendingFeedsNoPending(t *testing.T) {
	repo := &repoMock{entries: map[string]*syncdomain.SyncEntry{}}
	source := &sourceMock{payload: []byte(feedStatusFinishedSuccessXML)}

	svc, err := NewService(repo, source)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	result, resolveErr := svc.ResolvePendingFeeds(context.Background(), 50)
	if resolveErr != nil {
		t.Fatalf("ResolvePendingFeeds() error = %v", resolveErr)
	}
	if result.Checked != 0 {
		t.Fatalf("Checked = %d, want %d", result.Checked, 0)
	}
}

// TestResolvePendingFeedsRepoError verifies repository list errors are propagated.
func TestResolvePendingFeedsRepoError(t *testing.T) {
	repo := &repoMock{listPendingErr: errors.New("db down")}
	source := &sourceMock{payload: []byte(feedStatusFinishedSuccessXML)}

	svc, err := NewService(repo, source)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	if _, resolveErr := svc.ResolvePendingFeeds(context.Background(), 50); resolveErr == nil {
		t.Fatalf("ResolvePendingFeeds() expected error")
	}
}

// TestResolvePendingFeedsFailed verifies failed feeds are resolved with failed status.
func TestResolvePendingFeedsFailed(t *testing.T) {
	repo := &repoMock{
		entries: map[string]*syncdomain.SyncEntry{
			"feed-fail": {FeedID: "feed-fail", Status: syncdomain.SyncStatusPending, SyncedAt: time.Now()},
		},
	}
	source := &sourceMock{payload: []byte(feedStatusFinishedFailedXML)}

	svc, err := NewService(repo, source)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	result, resolveErr := svc.ResolvePendingFeeds(context.Background(), 50)
	if resolveErr != nil {
		t.Fatalf("ResolvePendingFeeds() error = %v", resolveErr)
	}
	if result.Resolved != 1 {
		t.Fatalf("Resolved = %d, want %d", result.Resolved, 1)
	}
	entry := repo.entries["feed-fail"]
	if entry.Status != syncdomain.SyncStatusFailed {
		t.Fatalf("Status = %q, want %q", entry.Status, syncdomain.SyncStatusFailed)
	}
}

// TestResolvePendingFeedsDefaultLimit verifies zero/negative limit defaults to 50.
func TestResolvePendingFeedsDefaultLimit(t *testing.T) {
	repo := &repoMock{entries: map[string]*syncdomain.SyncEntry{}}
	source := &sourceMock{payload: []byte(feedStatusFinishedSuccessXML)}

	svc, err := NewService(repo, source)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	result, resolveErr := svc.ResolvePendingFeeds(context.Background(), 0)
	if resolveErr != nil {
		t.Fatalf("ResolvePendingFeeds() error = %v", resolveErr)
	}
	if result.Checked != 0 {
		t.Fatalf("Checked = %d, want %d", result.Checked, 0)
	}
}

// TestNormalizePendingResolveLimit verifies default and cap behavior for pending resolution limits.
func TestNormalizePendingResolveLimit(t *testing.T) {
	if got := normalizePendingResolveLimit(0); got != defaultPendingResolveLimit {
		t.Fatalf("normalizePendingResolveLimit(0) = %d, want %d", got, defaultPendingResolveLimit)
	}
	if got := normalizePendingResolveLimit(-10); got != defaultPendingResolveLimit {
		t.Fatalf("normalizePendingResolveLimit(-10) = %d, want %d", got, defaultPendingResolveLimit)
	}
	if got := normalizePendingResolveLimit(10); got != 10 {
		t.Fatalf("normalizePendingResolveLimit(10) = %d, want %d", got, 10)
	}
	if got := normalizePendingResolveLimit(maxPendingResolveLimit + 1); got != maxPendingResolveLimit {
		t.Fatalf("normalizePendingResolveLimit(max+1) = %d, want %d", got, maxPendingResolveLimit)
	}
}

// concurrentPendingSourceMock defines delayed source behavior to verify concurrent pending resolution.
type concurrentPendingSourceMock struct {
	// mutex guards active counters.
	mutex sync.Mutex
	// active defines current in-flight GetFeedStatus calls.
	active int
	// maxActive defines max observed in-flight calls.
	maxActive int
}

// GetFeedStatus tracks active calls and returns finished responses.
func (m *concurrentPendingSourceMock) GetFeedStatus(ctx context.Context, feedID string) ([]byte, error) {
	m.mutex.Lock()
	m.active++
	if m.active > m.maxActive {
		m.maxActive = m.active
	}
	m.mutex.Unlock()

	time.Sleep(20 * time.Millisecond)

	m.mutex.Lock()
	m.active--
	m.mutex.Unlock()

	return []byte(feedStatusFinishedSuccessXML), nil
}

// TestResolvePendingFeedsUsesConcurrentWorkers verifies pending feed resolution executes concurrently.
func TestResolvePendingFeedsUsesConcurrentWorkers(t *testing.T) {
	repo := &repoMock{entries: map[string]*syncdomain.SyncEntry{
		"feed-1": {FeedID: "feed-1", Status: syncdomain.SyncStatusPending, SyncedAt: time.Now()},
		"feed-2": {FeedID: "feed-2", Status: syncdomain.SyncStatusPending, SyncedAt: time.Now()},
		"feed-3": {FeedID: "feed-3", Status: syncdomain.SyncStatusPending, SyncedAt: time.Now()},
		"feed-4": {FeedID: "feed-4", Status: syncdomain.SyncStatusPending, SyncedAt: time.Now()},
	}}
	source := &concurrentPendingSourceMock{}

	svc, err := NewService(repo, source)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	result, resolveErr := svc.ResolvePendingFeeds(context.Background(), 50)
	if resolveErr != nil {
		t.Fatalf("ResolvePendingFeeds() error = %v", resolveErr)
	}
	if result.Resolved != 4 {
		t.Fatalf("Resolved = %d, want %d", result.Resolved, 4)
	}
	if source.maxActive < 2 {
		t.Fatalf("maxActive = %d, want >= 2", source.maxActive)
	}
}

// TestNormalizePendingWorkerCount verifies worker count normalization behavior.
func TestNormalizePendingWorkerCount(t *testing.T) {
	if got := normalizePendingWorkerCount(0); got != 1 {
		t.Fatalf("normalizePendingWorkerCount(0) = %d, want %d", got, 1)
	}
	if got := normalizePendingWorkerCount(1); got != 1 {
		t.Fatalf("normalizePendingWorkerCount(1) = %d, want %d", got, 1)
	}
	if got := normalizePendingWorkerCount(2); got != 2 {
		t.Fatalf("normalizePendingWorkerCount(2) = %d, want %d", got, 2)
	}
	if got := normalizePendingWorkerCount(100); got != defaultPendingWorkers {
		t.Fatalf("normalizePendingWorkerCount(100) = %d, want %d", got, defaultPendingWorkers)
	}
}
