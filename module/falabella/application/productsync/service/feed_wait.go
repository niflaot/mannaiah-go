package service

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	syncdomain "mannaiah/module/falabella/domain/sync"
	"strings"
	"time"
)

var (
	// ErrProductFeedNotResolved is returned when product feed status is not terminal-success before image sync.
	ErrProductFeedNotResolved = errors.New("falabella product feed is not resolved")
)

// syncImagesAfterProductFeedResolved synchronizes product images only after product-feed resolution succeeds.
func (s *ProductSyncService) syncImagesAfterProductFeedResolved(
	ctx context.Context,
	sku string,
	urls []string,
	productFeedID string,
) (*syncdomain.ActionResponse, error) {
	normalized := uniqueTrimmedValues(urls)
	if len(normalized) == 0 {
		return nil, nil
	}

	if err := s.waitForProductFeedResolution(ctx, productFeedID); err != nil {
		return nil, fmt.Errorf("resolve product feed before image sync: %w", err)
	}

	return s.syncImages(ctx, sku, normalized)
}

// waitForProductFeedResolution waits until the product feed reaches terminal-success status.
func (s *ProductSyncService) waitForProductFeedResolution(ctx context.Context, feedID string) error {
	trimmedFeedID := strings.TrimSpace(feedID)
	if trimmedFeedID == "" {
		return fmt.Errorf("%w: empty feed id", ErrProductFeedNotResolved)
	}

	attempts := s.cfg.FeedResolutionAttempts
	if attempts <= 0 {
		attempts = defaultFeedResolutionAttempts
	}

	lastState := "unknown"
	for attempt := 1; attempt <= attempts; attempt++ {
		detail, err := s.getFeedDetail(ctx, trimmedFeedID)
		if err == nil {
			lastState = strings.TrimSpace(detail.Status)
			if detail.IsFinished() {
				if detail.IsSuccess() {
					return nil
				}

				return fmt.Errorf("%w: feed %q finished with %d failed records", ErrProductFeedNotResolved, trimmedFeedID, detail.FailedRecords)
			}
		} else {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return err
			}
			lastState = err.Error()
		}

		if attempt == attempts {
			break
		}

		if err := waitWithContext(ctx, resolveFeedResolutionBackoff(s.cfg.FeedResolutionBackoffMS, attempt)); err != nil {
			return err
		}
	}

	return fmt.Errorf("%w: feed %q was not finished after %d attempts (last state: %s)", ErrProductFeedNotResolved, trimmedFeedID, attempts, lastState)
}

// getFeedDetail retrieves and parses one Falabella feed status payload.
func (s *ProductSyncService) getFeedDetail(ctx context.Context, feedID string) (syncdomain.FeedDetail, error) {
	requestCtx := ctx
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		timeout := resolveFeedResolutionRequestTimeout(s.cfg.FeedResolutionRequestTimeoutMS)
		var cancel context.CancelFunc
		requestCtx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	rawPayload, err := s.source.GetFeedStatus(requestCtx, feedID)
	if err != nil {
		return syncdomain.FeedDetail{}, fmt.Errorf("get feed status: %w", err)
	}

	var response syncdomain.FeedResponse
	if decodeErr := xml.Unmarshal(rawPayload, &response); decodeErr != nil {
		return syncdomain.FeedDetail{}, fmt.Errorf("decode feed status: %w", decodeErr)
	}

	return response.Body.FeedDetail, nil
}

// resolveFeedResolutionBackoff resolves one attempt-indexed polling backoff duration.
func resolveFeedResolutionBackoff(backoffMS int, attempt int) time.Duration {
	if attempt <= 0 {
		attempt = 1
	}

	resolvedBackoff := backoffMS
	if resolvedBackoff <= 0 {
		resolvedBackoff = defaultFeedResolutionBackoffMS
	}
	if resolvedBackoff > maxFeedResolutionBackoffMS {
		resolvedBackoff = maxFeedResolutionBackoffMS
	}

	return time.Duration(resolvedBackoff*attempt) * time.Millisecond
}

// resolveFeedResolutionRequestTimeout resolves timeout values used by feed-status polling requests.
func resolveFeedResolutionRequestTimeout(timeoutMS int) time.Duration {
	resolvedTimeout := timeoutMS
	if resolvedTimeout <= 0 {
		resolvedTimeout = defaultFeedResolutionRequestTimeoutMS
	}
	if resolvedTimeout > maxFeedResolutionRequestTimeoutMS {
		resolvedTimeout = maxFeedResolutionRequestTimeoutMS
	}

	return time.Duration(resolvedTimeout) * time.Millisecond
}

// waitWithContext sleeps for the provided duration or exits early when context cancellation happens.
func waitWithContext(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
