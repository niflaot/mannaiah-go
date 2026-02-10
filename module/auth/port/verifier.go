package port

import (
	"context"

	"mannaiah/module/auth/domain"
)

// TokenVerifier defines token validation behavior required by auth use cases.
type TokenVerifier interface {
	// Verify validates JWT tokens and returns normalized claims.
	Verify(ctx context.Context, token string) (*domain.Claims, error)
}
