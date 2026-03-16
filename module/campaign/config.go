package campaign

import (
	campaignport "mannaiah/module/campaign/port"
	campaignruntime "mannaiah/module/campaign/runtime"
)

// Config defines campaign runtime configuration values.
type Config = campaignruntime.Config

// SegmentResolver defines segment resolution dependencies.
type SegmentResolver = campaignport.SegmentResolver

// EmailSender defines email dispatch dependencies.
type EmailSender = campaignport.EmailSender

// IntegrationEventPublisher defines campaign event publication behavior.
type IntegrationEventPublisher = campaignport.IntegrationEventPublisher

// DeliveryReader defines campaign delivery read behavior.
type DeliveryReader = campaignport.DeliveryReader
