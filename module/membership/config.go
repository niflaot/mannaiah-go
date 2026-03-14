package membership

import (
	membershipport "mannaiah/module/membership/port"
	membershipruntime "mannaiah/module/membership/runtime"
)

// Config defines membership runtime configuration values.
type Config = membershipruntime.Config

// ContactLookup defines contact lookup behavior required by membership module.
type ContactLookup = membershipport.ContactLookup

// IntegrationEventPublisher defines membership event publication behavior.
type IntegrationEventPublisher = membershipport.IntegrationEventPublisher
