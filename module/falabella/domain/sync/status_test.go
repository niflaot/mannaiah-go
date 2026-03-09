package sync

import "testing"

// TestSyncActionIsValid verifies sync action validation behavior.
func TestSyncActionIsValid(t *testing.T) {
	tests := []struct {
		name  string
		value SyncAction
		want  bool
	}{
		{name: "create is valid", value: SyncActionCreate, want: true},
		{name: "update is valid", value: SyncActionUpdate, want: true},
		{name: "empty is invalid", value: "", want: false},
		{name: "unknown is invalid", value: "delete", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.value.IsValid(); got != tt.want {
				t.Fatalf("SyncAction(%q).IsValid() = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

// TestSyncActionString verifies sync action string conversion behavior.
func TestSyncActionString(t *testing.T) {
	if got := SyncActionCreate.String(); got != "create" {
		t.Fatalf("SyncActionCreate.String() = %q, want %q", got, "create")
	}
	if got := SyncActionUpdate.String(); got != "update" {
		t.Fatalf("SyncActionUpdate.String() = %q, want %q", got, "update")
	}
}

// TestSyncStatusIsValid verifies sync status validation behavior.
func TestSyncStatusIsValid(t *testing.T) {
	tests := []struct {
		name  string
		value SyncStatus
		want  bool
	}{
		{name: "pending is valid", value: SyncStatusPending, want: true},
		{name: "finished is valid", value: SyncStatusFinished, want: true},
		{name: "failed is valid", value: SyncStatusFailed, want: true},
		{name: "empty is invalid", value: "", want: false},
		{name: "unknown is invalid", value: "cancelled", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.value.IsValid(); got != tt.want {
				t.Fatalf("SyncStatus(%q).IsValid() = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

// TestSyncStepTask verifies step-to-task mapping behavior.
func TestSyncStepTask(t *testing.T) {
	if got := SyncStepProduct.Task(); got != SyncTaskData {
		t.Fatalf("SyncStepProduct.Task() = %q, want %q", got, SyncTaskData)
	}
	if got := SyncStepImage.Task(); got != SyncTaskImage {
		t.Fatalf("SyncStepImage.Task() = %q, want %q", got, SyncTaskImage)
	}
}

// TestSyncTaskIsValid verifies sync task validation behavior.
func TestSyncTaskIsValid(t *testing.T) {
	if !SyncTaskData.IsValid() {
		t.Fatalf("SyncTaskData.IsValid() = false, want true")
	}
	if !SyncTaskImage.IsValid() {
		t.Fatalf("SyncTaskImage.IsValid() = false, want true")
	}
	if SyncTask("").IsValid() {
		t.Fatalf("SyncTask(\"\").IsValid() = true, want false")
	}
}

// TestSyncStatusString verifies sync status string conversion behavior.
func TestSyncStatusString(t *testing.T) {
	if got := SyncStatusPending.String(); got != "pending" {
		t.Fatalf("SyncStatusPending.String() = %q, want %q", got, "pending")
	}
	if got := SyncStatusFinished.String(); got != "finished" {
		t.Fatalf("SyncStatusFinished.String() = %q, want %q", got, "finished")
	}
	if got := SyncStatusFailed.String(); got != "failed" {
		t.Fatalf("SyncStatusFailed.String() = %q, want %q", got, "failed")
	}
}

// TestSyncEntryIsResolved verifies sync entry resolution detection behavior.
func TestSyncEntryIsResolved(t *testing.T) {
	tests := []struct {
		name   string
		status SyncStatus
		want   bool
	}{
		{name: "pending is not resolved", status: SyncStatusPending, want: false},
		{name: "finished is resolved", status: SyncStatusFinished, want: true},
		{name: "failed is resolved", status: SyncStatusFailed, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := SyncEntry{Status: tt.status}
			if got := entry.IsResolved(); got != tt.want {
				t.Fatalf("SyncEntry{Status: %q}.IsResolved() = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}
