package domain

import "strings"

// SyncTrigger defines why a sync was initiated.
type SyncTrigger string

const (
	// TriggerCron defines scheduled sync runs.
	TriggerCron SyncTrigger = "cron"
	// TriggerManual defines manually-triggered sync runs.
	TriggerManual SyncTrigger = "manual"
	// TriggerEvent defines event-triggered sync runs.
	TriggerEvent SyncTrigger = "event"
	// TriggerStartup defines startup-triggered sync runs.
	TriggerStartup SyncTrigger = "startup"
	// TriggerMigration defines migration-triggered sync runs.
	TriggerMigration SyncTrigger = "migration"
)

// IsValid reports whether a sync trigger is recognized.
func (t SyncTrigger) IsValid() bool {
	switch SyncTrigger(strings.TrimSpace(string(t))) {
	case TriggerCron, TriggerManual, TriggerEvent, TriggerStartup, TriggerMigration:
		return true
	default:
		return false
	}
}
