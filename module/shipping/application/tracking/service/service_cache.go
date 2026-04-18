package service

import (
	"strings"
	"time"

	"mannaiah/module/shipping/domain"
)

const trackingHistoryCacheTTL = 30 * time.Second

// trackingHistoryCacheEntry defines one cached tracking-history lookup.
type trackingHistoryCacheEntry struct {
	// history defines the cached normalized tracking history.
	history domain.TrackingHistory
	// expiresAt defines cache expiration timestamps.
	expiresAt time.Time
}

// trackingHistoryCacheKey resolves one stable cache key for tracking history.
func trackingHistoryCacheKey(carrierID string, trackingNumber string) string {
	return strings.ToLower(strings.TrimSpace(carrierID)) + "|" + strings.TrimSpace(trackingNumber)
}

// getCachedTrackingHistory resolves one cached tracking history when it is still fresh.
func (s *Service) getCachedTrackingHistory(carrierID string, trackingNumber string) (*domain.TrackingHistory, bool) {
	if s == nil {
		return nil, false
	}
	cacheKey := trackingHistoryCacheKey(carrierID, trackingNumber)
	now := time.Now().UTC()

	s.trackingHistoryCacheMu.RLock()
	entry, exists := s.trackingHistoryCache[cacheKey]
	s.trackingHistoryCacheMu.RUnlock()
	if !exists || now.After(entry.expiresAt) {
		if exists {
			s.trackingHistoryCacheMu.Lock()
			delete(s.trackingHistoryCache, cacheKey)
			s.trackingHistoryCacheMu.Unlock()
		}

		return nil, false
	}

	return cloneTrackingHistory(entry.history), true
}

// putCachedTrackingHistory stores one normalized tracking history for a short reuse window.
func (s *Service) putCachedTrackingHistory(carrierID string, trackingNumber string, history *domain.TrackingHistory) {
	if s == nil || history == nil {
		return
	}
	cacheKey := trackingHistoryCacheKey(carrierID, trackingNumber)
	cloned := cloneTrackingHistory(*history)
	if cloned == nil {
		return
	}

	s.trackingHistoryCacheMu.Lock()
	s.trackingHistoryCache[cacheKey] = trackingHistoryCacheEntry{
		history:   *cloned,
		expiresAt: time.Now().UTC().Add(trackingHistoryCacheTTL),
	}
	s.trackingHistoryCacheMu.Unlock()
}

// cloneTrackingHistory returns a defensive copy of one tracking history payload.
func cloneTrackingHistory(history domain.TrackingHistory) *domain.TrackingHistory {
	clonedHistory := history
	if len(history.History) > 0 {
		clonedHistory.History = append([]domain.TrackingEvent(nil), history.History...)
	}

	return &clonedHistory
}
