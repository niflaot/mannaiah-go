package service

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"strings"
	"time"

	syncdomain "mannaiah/module/falabella/domain/sync"
	"mannaiah/module/falabella/port"
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
	resolvedLimit := limit
	if resolvedLimit <= 0 {
		resolvedLimit = 50
	}

	pending, err := s.repo.ListPending(ctx, resolvedLimit)
	if err != nil {
		return nil, fmt.Errorf("list pending feeds: %w", err)
	}

	result := &ResolvePendingResult{Checked: len(pending)}
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

// resolveOneFeed queries Falabella FeedStatus for a single feed and updates the local status.
func (s *SyncStatusService) resolveOneFeed(ctx context.Context, feedID string) error {
	rawPayload, err := s.source.GetFeedStatus(ctx, feedID)
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
