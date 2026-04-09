package tcc

import (
	"time"

	"mannaiah/module/shipping/domain"
)

// resolveGlobalTrackingStatus prioritizes business-relevant statuses over raw recency.
func resolveGlobalTrackingStatus(events []domain.TrackingEvent) domain.TrackingStatus {
	bestStatus := domain.TrackingStatusProcessing
	bestPriority := trackingStatusPriority(bestStatus)
	bestDate := time.Time{}

	for _, event := range events {
		priority := trackingStatusPriority(event.Status)
		if priority > bestPriority || (priority == bestPriority && event.Date.After(bestDate)) {
			bestStatus = event.Status
			bestPriority = priority
			bestDate = event.Date
		}
	}

	return bestStatus
}

// trackingStatusPriority assigns global-status precedence for mixed carrier histories.
func trackingStatusPriority(status domain.TrackingStatus) int {
	switch status {
	case domain.TrackingStatusReturn:
		return 600
	case domain.TrackingStatusVoided:
		return 550
	case domain.TrackingStatusCompleted:
		return 500
	case domain.TrackingStatusIncidence:
		return 450
	case domain.TrackingStatusProcessing:
		return 400
	case domain.TrackingStatusOrigin:
		return 300
	default:
		return 100
	}
}
