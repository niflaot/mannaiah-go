// Package coupons provides the coupon management feature module.
package coupons

import (
	"mannaiah/module/core/messaging/bus"
	couponruntime "mannaiah/module/coupons/runtime"

	"gorm.io/gorm"
)

// Module defines composition-root wiring for coupon endpoints and services.
type Module = couponruntime.Module

// Loader defines bootstrap hooks required by the coupon module.
type Loader = couponruntime.Loader

// New creates a coupon module with the provided database and optional bus publisher.
func New(db *gorm.DB, busPublisher bus.Publisher) (*Module, error) {
	return couponruntime.NewWithMessaging(db, busPublisher)
}
