package http

import (
	"fmt"
	"strings"
	"time"

	"mannaiah/module/contacts/application"
	"mannaiah/module/contacts/domain"
	"mannaiah/module/contacts/port"
	corehttp "mannaiah/module/core/http"
)

const (
	// circleOptInMetadataKey defines contact metadata key used for circle opt-in decision values.
	circleOptInMetadataKey = "flock_checker_circle_optin"
	// circleOptInAcceptedAtMetadataKey defines contact metadata key used for local accepted-at timestamp values.
	circleOptInAcceptedAtMetadataKey = "flock_checker_circle_optin_accepted_at"
	// circleOptInAcceptedAtUTCMetadataKey defines contact metadata key used for UTC accepted-at timestamp values.
	circleOptInAcceptedAtUTCMetadataKey = "flock_checker_circle_optin_accepted_at_utc"
	// circleOptInRejectedAtMetadataKey defines contact metadata key used for local rejected-at timestamp values.
	circleOptInRejectedAtMetadataKey = "flock_checker_circle_optin_rejected_at"
	// circleOptInRejectedAtUTCMetadataKey defines contact metadata key used for UTC rejected-at timestamp values.
	circleOptInRejectedAtUTCMetadataKey = "flock_checker_circle_optin_rejected_at_utc"
	// circleOptInLocalTimestampLayout defines layout values used by local accepted-at metadata keys.
	circleOptInLocalTimestampLayout = "2006-01-02 15:04:05"
	// circleOptInLocalTimezoneName defines timezone values used by local accepted-at metadata keys.
	circleOptInLocalTimezoneName = "America/Bogota"
	// circleOptInFallbackTimezoneName defines fixed timezone labels used when timezone loading fails.
	circleOptInFallbackTimezoneName = "UTC-05"
	// circleOptInFallbackTimezoneOffset defines fixed timezone offsets used when timezone loading fails.
	circleOptInFallbackTimezoneOffset = -5 * 60 * 60
	// circleOptInYesValue defines the opt-in accepted value.
	circleOptInYesValue = "yes"
	// circleOptInNoValue defines the opt-in rejected value.
	circleOptInNoValue = "no"
)

// consentByEmailRequest defines request payload for by-email consent updates.
type consentByEmailRequest struct {
	// Email defines target contact email values.
	Email string `json:"email"`
}

// optInByEmail handles contact circle opt-in update endpoints.
func (h *Handler) optInByEmail(ctx corehttp.Context) error {
	return h.updateCircleOptInByEmail(ctx, circleOptInYesValue)
}

// optOutByEmail handles contact circle opt-out update endpoints.
func (h *Handler) optOutByEmail(ctx corehttp.Context) error {
	return h.updateCircleOptInByEmail(ctx, circleOptInNoValue)
}

// updateCircleOptInByEmail updates circle opt-in metadata values by contact email.
func (h *Handler) updateCircleOptInByEmail(ctx corehttp.Context, decision string) error {
	var request consentByEmailRequest
	if err := ctx.BodyParser(&request); err != nil {
		return corehttp.NewAppError(400, "invalid_payload", err)
	}

	email := strings.ToLower(strings.TrimSpace(request.Email))
	if email == "" {
		return corehttp.NewAppError(400, "invalid_payload", domain.ErrEmailRequired)
	}

	contact, err := h.findByEmail(ctx, email)
	if err != nil {
		return h.mapError(err)
	}

	metadata := cloneMetadata(contact.Metadata)
	localAcceptedAt, utcAcceptedAt := consentAcceptedAtValues(h.currentTime())
	metadata[circleOptInMetadataKey] = decision
	switch decision {
	case circleOptInYesValue:
		metadata[circleOptInAcceptedAtMetadataKey] = localAcceptedAt
		metadata[circleOptInAcceptedAtUTCMetadataKey] = utcAcceptedAt
		delete(metadata, circleOptInRejectedAtMetadataKey)
		delete(metadata, circleOptInRejectedAtUTCMetadataKey)
	case circleOptInNoValue:
		metadata[circleOptInRejectedAtMetadataKey] = localAcceptedAt
		metadata[circleOptInRejectedAtUTCMetadataKey] = utcAcceptedAt
		delete(metadata, circleOptInAcceptedAtMetadataKey)
		delete(metadata, circleOptInAcceptedAtUTCMetadataKey)
	}

	updated, err := h.service.Update(ctx.Context(), contact.ID, application.UpdateCommand{Metadata: &metadata})
	if err != nil {
		return h.mapError(err)
	}

	return ctx.Status(200).JSON(updated)
}

// findByEmail resolves one contact by normalized email values.
func (h *Handler) findByEmail(ctx corehttp.Context, email string) (*domain.Contact, error) {
	result, err := h.service.List(ctx.Context(), port.ListQuery{
		Page:  1,
		Limit: 1,
		Email: email,
	})
	if err != nil {
		return nil, fmt.Errorf("find contact by email: %w", err)
	}
	if len(result.Data) == 0 {
		return nil, port.ErrNotFound
	}

	contact := result.Data[0]
	return &contact, nil
}

// currentTime resolves current UTC timestamps from handler clock dependencies.
func (h *Handler) currentTime() time.Time {
	if h == nil || h.now == nil {
		return time.Now().UTC()
	}

	return h.now().UTC()
}

// consentAcceptedAtValues maps one source timestamp into local and UTC consent metadata values.
func consentAcceptedAtValues(value time.Time) (string, string) {
	location, err := time.LoadLocation(circleOptInLocalTimezoneName)
	if err != nil {
		location = time.FixedZone(circleOptInFallbackTimezoneName, circleOptInFallbackTimezoneOffset)
	}

	utc := value.UTC()
	return utc.In(location).Format(circleOptInLocalTimestampLayout), utc.Format(time.RFC3339)
}

// cloneMetadata clones metadata maps and normalizes empty maps to initialized values.
func cloneMetadata(metadata map[string]string) map[string]string {
	if len(metadata) == 0 {
		return map[string]string{}
	}

	cloned := make(map[string]string, len(metadata))
	for key, value := range metadata {
		cloned[key] = value
	}

	return cloned
}
