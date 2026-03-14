package service

import (
	"context"
	"strings"
	"time"

	"go.uber.org/zap"
	"mannaiah/module/woocommerce/port"
)

// stampMembership stamps membership state from checker metadata when available.
func (s *ContactSyncService) stampMembership(ctx context.Context, command port.ContactSyncCommand) {
	if s == nil || s.membershipStamper == nil {
		return
	}

	email := normalizeEmailKey(command.Email)
	if email == "" {
		return
	}
	action, occurredAt, ok := resolveMembershipAction(command.Metadata)
	if !ok {
		return
	}

	if err := s.membershipStamper.StampByEmail(ctx, email, "all", action, "woocommerce_sync", occurredAt); err != nil {
		s.logger.Warn("woocommerce contact sync membership stamp failed", zap.Error(err), zap.String("email", email))
	}
}

// resolveMembershipAction resolves membership stamp payload values from contact metadata.
func resolveMembershipAction(metadata map[string]string) (port.MembershipAction, *time.Time, bool) {
	if len(metadata) == 0 {
		return "", nil, false
	}

	decision := strings.ToLower(strings.TrimSpace(metadata[circleOptInMetadataKey]))
	switch decision {
	case "yes":
		return port.MembershipActionOptIn, parseSyncOccurredAt(metadata[circleOptInMetadataKey+checkerMetadataAcceptedAtUTCSuffix], metadata[circleOptInMetadataKey+checkerMetadataAcceptedAtSuffix]), true
	case "no":
		return port.MembershipActionOptOut, parseSyncOccurredAt(metadata[circleOptInMetadataKey+checkerMetadataRejectedAtUTCSuffix], metadata[circleOptInMetadataKey+checkerMetadataRejectedAtSuffix]), true
	default:
		return "", nil, false
	}
}

// parseSyncOccurredAt parses UTC/local timestamps from sync metadata.
func parseSyncOccurredAt(utcValue string, localValue string) *time.Time {
	if parsedUTC, err := time.Parse(time.RFC3339, strings.TrimSpace(utcValue)); err == nil {
		value := parsedUTC.UTC()
		return &value
	}
	location, err := time.LoadLocation(checkerMetadataAcceptedAtLocalZone)
	if err != nil {
		location = time.FixedZone(checkerMetadataAcceptedAtFixedZone, checkerMetadataAcceptedAtFixedOffset)
	}
	if parsedLocal, parseErr := time.ParseInLocation(checkerMetadataAcceptedAtLocalLayout, strings.TrimSpace(localValue), location); parseErr == nil {
		value := parsedLocal.UTC()
		return &value
	}

	return nil
}
