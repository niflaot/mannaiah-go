package orders

import (
	"time"

	ordersdomain "mannaiah/module/orders/domain"
)

// resolveStatusOccurredAt resolves status timestamps that always move status history forward.
func resolveStatusOccurredAt(order ordersdomain.Order, source *time.Time) *time.Time {
	latest := latestStatusOccurredAt(order.StatusHistory)
	if source != nil && !source.IsZero() {
		candidate := source.UTC()
		if latest.IsZero() || candidate.After(latest) {
			return &candidate
		}
	}

	now := time.Now().UTC()
	return &now
}

// latestStatusOccurredAt resolves the latest status timestamp for an order history sequence.
func latestStatusOccurredAt(history []ordersdomain.StatusEntry) time.Time {
	var latest time.Time
	for _, value := range history {
		occurredAt := value.OccurredAt.UTC()
		if occurredAt.After(latest) {
			latest = occurredAt
		}
	}

	return latest
}
