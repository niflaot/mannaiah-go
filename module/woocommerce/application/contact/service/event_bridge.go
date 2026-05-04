package service

import (
	"context"

	woocontactevent "mannaiah/module/woocommerce/application/contact/event"
	"mannaiah/module/woocommerce/port"
)

// publishEvent publishes sync integration events and ignores publication failures.
func (s *ContactSyncService) publishEvent(ctx context.Context, integrationEvent port.IntegrationEvent) {
	_ = s.publisher.Publish(ctx, integrationEvent)
}

// toEventSummary maps service-level sync summaries to event package summary contracts.
func toEventSummary(summary SyncSummary) woocontactevent.Summary {
	return woocontactevent.Summary{
		Trigger:   summary.Trigger,
		Processed: summary.Processed,
		Created:   summary.Created,
		Updated:   summary.Updated,
		Unchanged: summary.Unchanged,
		Skipped:   summary.Skipped,
		Failed:    summary.Failed,
	}
}
