package port

import (
	"context"
	"time"
)

// MembershipAction defines membership action values.
type MembershipAction string

const (
	// MembershipActionOptIn defines opt-in action values.
	MembershipActionOptIn MembershipAction = "opt_in"
	// MembershipActionOptOut defines opt-out action values.
	MembershipActionOptOut MembershipAction = "opt_out"
)

// MembershipStamper defines optional membership stamp behavior for WooCommerce sync flows.
type MembershipStamper interface {
	// StampByEmail stamps membership state by contact email.
	StampByEmail(ctx context.Context, email string, channel string, action MembershipAction, source string, occurredAt *time.Time) error
}

// NoopMembershipStamper defines no-op membership stamp behavior.
type NoopMembershipStamper struct{}

// StampByEmail ignores membership stamp payload values.
func (NoopMembershipStamper) StampByEmail(ctx context.Context, email string, channel string, action MembershipAction, source string, occurredAt *time.Time) error {
	return nil
}
