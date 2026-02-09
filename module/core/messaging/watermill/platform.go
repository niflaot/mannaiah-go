package watermill

import (
	wmmsg "github.com/ThreeDotsLabs/watermill/message"
	"go.uber.org/zap"
	"mannaiah/module/core/messaging/platform"
	watermillpublisher "mannaiah/module/core/messaging/watermill/publisher"
	watermillrouter "mannaiah/module/core/messaging/watermill/router"
)

var (
	// ErrNilHandler is returned when a nil handler is provided.
	ErrNilHandler = watermillrouter.ErrNilHandler
	// ErrEmptyTopic is returned when a topic argument is empty.
	ErrEmptyTopic = watermillrouter.ErrEmptyTopic
	// ErrNilPublisher is returned when a nil Watermill publisher is provided.
	ErrNilPublisher = watermillpublisher.ErrNilPublisher
	// ErrMessageIDRequired is returned when a message id is not provided.
	ErrMessageIDRequired = watermillpublisher.ErrMessageIDRequired
	// ErrMessageTopicRequired is returned when a message topic is not provided.
	ErrMessageTopicRequired = watermillpublisher.ErrMessageTopicRequired
)

// InMemoryPlatform aliases the nested router platform type.
type InMemoryPlatform = watermillrouter.InMemoryPlatform

// PublisherAdapter aliases the nested publisher adapter type.
type PublisherAdapter = watermillpublisher.Adapter

// NewInMemoryPlatform creates an in-memory messaging platform with Watermill adapters.
func NewInMemoryPlatform(cfg platform.Config, providedLogger *zap.Logger) (*InMemoryPlatform, error) {
	return watermillrouter.NewInMemoryPlatform(cfg, providedLogger)
}

// NewPublisherAdapter creates a bus publisher adapter over a Watermill publisher.
func NewPublisherAdapter(publisher wmmsg.Publisher) (*PublisherAdapter, error) {
	return watermillpublisher.NewAdapter(publisher)
}
