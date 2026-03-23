package shipping

import (
	shippingport "mannaiah/module/shipping/port"
	shippingruntime "mannaiah/module/shipping/runtime"
)

// Config defines shipping runtime configuration values.
type Config = shippingruntime.Config

// IntegrationEventPublisher defines shipping integration event publication behavior.
type IntegrationEventPublisher = shippingport.IntegrationEventPublisher
