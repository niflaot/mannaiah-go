package service

import (
	"context"
	"fmt"

	"mannaiah/module/coupons/domain"
)

const maxCodeGenAttempts = 10

// generateUniqueCode generates a random coupon code that is not already in use.
// It retries up to maxCodeGenAttempts times before returning an error.
func (s *Service) generateUniqueCode(ctx context.Context) (string, error) {
	for attempt := range maxCodeGenAttempts {
		code := domain.GenerateCode()
		exists, err := s.repository.CodeExists(ctx, code)
		if err != nil {
			return "", fmt.Errorf("check generated code (attempt %d): %w", attempt+1, err)
		}
		if !exists {
			return code, nil
		}
	}

	return "", fmt.Errorf("failed to generate unique code after %d attempts", maxCodeGenAttempts)
}
