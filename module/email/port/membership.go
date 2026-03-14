package port

import "context"

// MembershipStamper defines optional membership stamp behavior for complaint handling.
type MembershipStamper interface {
	// OptOutByEmail stamps email opt-out for one recipient.
	OptOutByEmail(ctx context.Context, email string, source string) error
}

// NoopMembershipStamper defines no-op membership stamping behavior.
type NoopMembershipStamper struct{}

// OptOutByEmail ignores membership stamp payload values.
func (NoopMembershipStamper) OptOutByEmail(ctx context.Context, email string, source string) error {
	return nil
}
