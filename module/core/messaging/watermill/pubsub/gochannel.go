package pubsub

import (
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"go.uber.org/zap"
	"mannaiah/module/core/messaging/platform"
	watermilllogger "mannaiah/module/core/messaging/watermill/logger"
)

// NewGoChannel creates an in-memory Watermill GoChannel pubsub.
func NewGoChannel(cfg platform.Config, providedLogger *zap.Logger) *gochannel.GoChannel {
	normalizedCfg := cfg.Normalized()

	return gochannel.NewGoChannel(
		gochannel.Config{
			OutputChannelBuffer:            normalizedCfg.GoChannelBuffer,
			BlockPublishUntilSubscriberAck: true,
		},
		watermilllogger.NewZapAdapter(providedLogger),
	)
}
