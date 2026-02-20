package service

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	syncdomain "mannaiah/module/falabella/domain/sync"
	"mannaiah/module/falabella/port"
)

const (
	defaultPendingResolveLimit = 50
	maxPendingResolveLimit     = 200
	fallbackFeedStatusTimeout  = 5 * time.Second
	defaultPendingWorkers      = 4
	maxPendingWorkers          = 16
)

// ResolvePendingResult defines aggregate results from a batch pending-feed resolution pass.
type ResolvePendingResult struct {
	// Checked defines the total number of pending feeds evaluated.
	Checked int `json:"checked"`
	// Resolved defines the number of feeds that reached a terminal state.
	Resolved int `json:"resolved"`
	// StillPending defines the number of feeds that remain pending.
	StillPending int `json:"stillPending"`
	// Errored defines the number of feeds where the FeedStatus call itself failed.
	Errored int `json:"errored"`
}

// ResolvePendingFeeds resolves all pending feed entries by querying the Falabella FeedStatus API.
func (s *SyncStatusService) ResolvePendingFeeds(ctx context.Context, limit int) (*ResolvePendingResult, error) {
	resolvedLimit := normalizePendingResolveLimit(limit)

	pending, err := s.repo.ListPending(ctx, resolvedLimit)
	if err != nil {
		return nil, fmt.Errorf("list pending feeds: %w", err)
	}

	result := &ResolvePendingResult{Checked: len(pending)}
	workerCount := normalizePendingWorkerCount(len(pending))
	if workerCount == 1 {
		for _, entry := range pending {
			feedID := strings.TrimSpace(entry.FeedID)
			if feedID == "" {
				result.Errored++
				continue
			}

			resolveErr := s.resolveOneFeed(ctx, feedID)
			if resolveErr != nil {
				if errors.Is(resolveErr, ErrFeedNotFinished) {
					result.StillPending++
				} else {
					result.Errored++
				}
				continue
			}

			result.Resolved++
		}

		return result, nil
	}

	type pendingOutcome int
	const (
		pendingResolved pendingOutcome = iota
		pendingStillPending
		pendingErrored
	)

	jobs := make(chan string, len(pending))
	outcomes := make(chan pendingOutcome, len(pending))

	var waitGroup sync.WaitGroup
	for workerIndex := 0; workerIndex < workerCount; workerIndex++ {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			for feedID := range jobs {
				if strings.TrimSpace(feedID) == "" {
					outcomes <- pendingErrored
					continue
				}
				resolveErr := s.resolveOneFeed(ctx, feedID)
				if resolveErr == nil {
					outcomes <- pendingResolved
					continue
				}
				if errors.Is(resolveErr, ErrFeedNotFinished) {
					outcomes <- pendingStillPending
					continue
				}
				outcomes <- pendingErrored
			}
		}()
	}

	for _, entry := range pending {
		jobs <- strings.TrimSpace(entry.FeedID)
	}
	close(jobs)

	waitGroup.Wait()
	close(outcomes)

	for outcome := range outcomes {
		switch outcome {
		case pendingResolved:
			result.Resolved++
		case pendingStillPending:
			result.StillPending++
		default:
			result.Errored++
		}
	}

	return result, nil
}

// resolveOneFeed queries Falabella FeedStatus for a single feed and updates the local status.
func (s *SyncStatusService) resolveOneFeed(ctx context.Context, feedID string) error {
	requestCtx := ctx
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		requestCtx, cancel = context.WithTimeout(ctx, fallbackFeedStatusTimeout)
		defer cancel()
	}

	rawPayload, err := s.source.GetFeedStatus(requestCtx, feedID)
	if err != nil {
		return fmt.Errorf("get falabella feed status: %w", err)
	}

	var response syncdomain.FeedResponse
	if xmlErr := xml.Unmarshal(rawPayload, &response); xmlErr != nil {
		return fmt.Errorf("unmarshal falabella feed status: %w", xmlErr)
	}

	detail := response.Body.FeedDetail
	if !detail.IsFinished() {
		return fmt.Errorf("%w: current status is %q", ErrFeedNotFinished, detail.Status)
	}

	resolvedStatus := syncdomain.SyncStatusFinished
	if !detail.IsSuccess() {
		resolvedStatus = syncdomain.SyncStatusFailed
	}

	now := time.Now().UTC()
	if updateErr := s.repo.UpdateStatus(ctx, feedID, resolvedStatus, &now); updateErr != nil {
		if !errors.Is(updateErr, port.ErrSyncEntryNotFound) {
			return fmt.Errorf("update sync status: %w", updateErr)
		}
	}

	return nil
}

// normalizePendingResolveLimit resolves safe limits for pending-feed resolution batches.
func normalizePendingResolveLimit(limit int) int {
	if limit <= 0 {
		return defaultPendingResolveLimit
	}
	if limit > maxPendingResolveLimit {
		return maxPendingResolveLimit
	}

	return limit
}

// normalizePendingWorkerCount resolves bounded worker counts for pending-feed resolution.
func normalizePendingWorkerCount(totalPending int) int {
	if totalPending <= 1 {
		return 1
	}

	workers := defaultPendingWorkers
	if workers > maxPendingWorkers {
		workers = maxPendingWorkers
	}
	if workers > totalPending {
		workers = totalPending
	}
	if workers <= 0 {
		return 1
	}

	return workers
}
