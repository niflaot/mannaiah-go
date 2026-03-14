package store

import (
	"context"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"mannaiah/module/membership/domain"
	"mannaiah/module/membership/port"
)

// TestRepositoryResolvesEffectiveStatusFromStamps verifies latest-status resolution from immutable stamps.
func TestRepositoryResolvesEffectiveStatusFromStamps(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:membership_repo_status?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if execErr := db.Exec(`CREATE TABLE membership_stamps (
		id TEXT PRIMARY KEY,
		contact_id TEXT NOT NULL,
		channel TEXT NOT NULL,
		action TEXT NOT NULL,
		source TEXT NOT NULL,
		occurred_at DATETIME NOT NULL,
		created_at DATETIME NOT NULL
	);`).Error; execErr != nil {
		t.Fatalf("create membership_stamps: %v", execErr)
	}

	repository, err := NewRepository(db)
	if err != nil {
		t.Fatalf("NewRepository() error = %v", err)
	}

	ctx := context.Background()
	firstAt := time.Date(2026, time.March, 14, 18, 0, 0, 0, time.UTC)
	first, saveErr := repository.SaveStamp(ctx, port.StampInput{
		ContactID:  "contact-1",
		Channel:    domain.ChannelAll,
		Action:     domain.ActionOptIn,
		Source:     "woocommerce_sync",
		OccurredAt: firstAt,
	})
	if saveErr != nil {
		t.Fatalf("SaveStamp(all opt_in) error = %v", saveErr)
	}
	if first == nil || !first.Created {
		t.Fatalf("first.Created = %v, want true", first)
	}

	duplicate, duplicateErr := repository.SaveStamp(ctx, port.StampInput{
		ContactID:  "contact-1",
		Channel:    domain.ChannelAll,
		Action:     domain.ActionOptIn,
		Source:     "woocommerce_sync",
		OccurredAt: firstAt.Add(1 * time.Hour),
	})
	if duplicateErr != nil {
		t.Fatalf("SaveStamp(duplicate all opt_in) error = %v", duplicateErr)
	}
	if duplicate == nil || duplicate.Created {
		t.Fatalf("duplicate.Created = %v, want false", duplicate)
	}

	emailStatus, statusErr := repository.GetStatus(ctx, "contact-1", domain.ChannelEmail)
	if statusErr != nil {
		t.Fatalf("GetStatus(email) error = %v", statusErr)
	}
	if emailStatus.Channel != domain.ChannelEmail {
		t.Fatalf("emailStatus.Channel = %q, want %q", emailStatus.Channel, domain.ChannelEmail)
	}
	if emailStatus.Action != domain.ActionOptIn {
		t.Fatalf("emailStatus.Action = %q, want %q", emailStatus.Action, domain.ActionOptIn)
	}

	_, saveErr = repository.SaveStamp(ctx, port.StampInput{
		ContactID:  "contact-1",
		Channel:    domain.ChannelEmail,
		Action:     domain.ActionOptOut,
		Source:     "api",
		OccurredAt: firstAt.Add(2 * time.Hour),
	})
	if saveErr != nil {
		t.Fatalf("SaveStamp(email opt_out) error = %v", saveErr)
	}

	emailStatus, statusErr = repository.GetStatus(ctx, "contact-1", domain.ChannelEmail)
	if statusErr != nil {
		t.Fatalf("GetStatus(email after opt_out) error = %v", statusErr)
	}
	if emailStatus.Action != domain.ActionOptOut {
		t.Fatalf("emailStatus.Action = %q, want %q", emailStatus.Action, domain.ActionOptOut)
	}
}

// TestRepositoryGetStatuses verifies channel-agnostic status queries.
func TestRepositoryGetStatuses(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:membership_repo_statuses?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if execErr := db.Exec(`CREATE TABLE membership_stamps (
		id TEXT PRIMARY KEY,
		contact_id TEXT NOT NULL,
		channel TEXT NOT NULL,
		action TEXT NOT NULL,
		source TEXT NOT NULL,
		occurred_at DATETIME NOT NULL,
		created_at DATETIME NOT NULL
	);`).Error; execErr != nil {
		t.Fatalf("create membership_stamps: %v", execErr)
	}

	repository, err := NewRepository(db)
	if err != nil {
		t.Fatalf("NewRepository() error = %v", err)
	}

	ctx := context.Background()
	at := time.Date(2026, time.March, 14, 18, 0, 0, 0, time.UTC)
	if _, saveErr := repository.SaveStamp(ctx, port.StampInput{ContactID: "contact-2", Channel: domain.ChannelAll, Action: domain.ActionOptIn, Source: "woocommerce_sync", OccurredAt: at}); saveErr != nil {
		t.Fatalf("SaveStamp(all opt_in) error = %v", saveErr)
	}
	if _, saveErr := repository.SaveStamp(ctx, port.StampInput{ContactID: "contact-2", Channel: domain.Channel("sms"), Action: domain.ActionOptOut, Source: "api", OccurredAt: at.Add(1 * time.Hour)}); saveErr != nil {
		t.Fatalf("SaveStamp(sms opt_out) error = %v", saveErr)
	}

	statuses, statusesErr := repository.GetStatuses(ctx, "contact-2")
	if statusesErr != nil {
		t.Fatalf("GetStatuses() error = %v", statusesErr)
	}
	if len(statuses) != 2 {
		t.Fatalf("len(statuses) = %d, want 2", len(statuses))
	}

	foundEmail := false
	foundSMS := false
	for _, status := range statuses {
		switch status.Channel {
		case domain.ChannelEmail:
			foundEmail = status.Action == domain.ActionOptIn
		case domain.Channel("sms"):
			foundSMS = status.Action == domain.ActionOptOut
		}
	}
	if !foundEmail {
		t.Fatalf("expected email status from all-channel fallback")
	}
	if !foundSMS {
		t.Fatalf("expected sms status from explicit stamp")
	}
}
