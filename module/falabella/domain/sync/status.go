package sync

import "time"

// SyncAction defines Falabella sync operation type values.
type SyncAction string

const (
	// SyncActionCreate defines product creation sync actions.
	SyncActionCreate SyncAction = "create"
	// SyncActionUpdate defines product update sync actions.
	SyncActionUpdate SyncAction = "update"
)

// SyncStatus defines Falabella feed resolution status values.
type SyncStatus string

const (
	// SyncStatusPending defines unresolved feed status values.
	SyncStatusPending SyncStatus = "pending"
	// SyncStatusFinished defines successfully resolved feed status values.
	SyncStatusFinished SyncStatus = "finished"
	// SyncStatusFailed defines failed feed status values.
	SyncStatusFailed SyncStatus = "failed"
)

// SyncEntry defines Falabella sync status domain entities.
type SyncEntry struct {
	// ProductID defines source product identifier values.
	ProductID string
	// SKU defines seller SKU values sent to Falabella.
	SKU string
	// FeedID defines Falabella feed identifier values returned on async submission.
	FeedID string
	// Action defines whether the sync was a creation or update.
	Action SyncAction
	// Status defines current feed resolution status values.
	Status SyncStatus
	// SyncedAt defines sync submission timestamp values.
	SyncedAt time.Time
	// ResolvedAt defines optional feed resolution timestamp values.
	ResolvedAt *time.Time
}

// IsValid reports whether sync action values are recognized.
func (a SyncAction) IsValid() bool {
	return a == SyncActionCreate || a == SyncActionUpdate
}

// String returns the string representation of sync action values.
func (a SyncAction) String() string {
	return string(a)
}

// IsValid reports whether sync status values are recognized.
func (s SyncStatus) IsValid() bool {
	return s == SyncStatusPending || s == SyncStatusFinished || s == SyncStatusFailed
}

// String returns the string representation of sync status values.
func (s SyncStatus) String() string {
	return string(s)
}

// IsResolved reports whether sync entries have been resolved.
func (e SyncEntry) IsResolved() bool {
	return e.Status == SyncStatusFinished || e.Status == SyncStatusFailed
}
