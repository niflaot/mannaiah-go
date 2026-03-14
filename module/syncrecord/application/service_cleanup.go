package application

import (
	"context"
	"time"
)

// CleanupExpired removes runs older than retention days and returns deleted rows.
func (s *RecorderService) CleanupExpired(ctx context.Context, retentionDays int) (int64, error) {
	if retentionDays <= 0 {
		return 0, nil
	}

	cutoff := time.Now().UTC().Add(-time.Duration(retentionDays) * 24 * time.Hour)
	return s.CleanupBefore(ctx, cutoff)
}
