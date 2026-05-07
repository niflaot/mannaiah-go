package contacts

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	contactsapplication "mannaiah/module/contacts/application"
	contactsport "mannaiah/module/contacts/port"
	"mannaiah/module/exports/port"
)

const pageSize = 500

const (
	checkerCircleOptInMetadataKey      = "flock_checker_circle_optin"
	checkerPrivacyAcceptMetadataKey    = "flock_checker_privacy_accept"
	checkerAcceptedAtSuffix            = "_accepted_at"
	checkerAcceptedAtUTCSuffix         = "_accepted_at_utc"
	checkerRejectedAtSuffix            = "_rejected_at"
	checkerRejectedAtUTCSuffix         = "_rejected_at_utc"
	checkerAcceptedAtLocalLayout       = "2006-01-02 15:04:05"
	checkerAcceptedAtLocalZone         = "America/Bogota"
	checkerAcceptedAtFixedZone         = "UTC-05"
	checkerAcceptedAtFixedOffset       = -5 * 60 * 60
	checkerAcceptedValueConfirmed      = "yes"
	checkerRejectedValueConfirmed      = "no"
	membershipActionOptIn              = "opt_in"
	membershipActionOptOut             = "opt_out"
	membershipPreferredChannelAll      = "all"
	membershipPreferredChannelEmail    = "email"
	membershipPreferredChannelFallback = ""
)

var (
	// ErrNilService is returned when contact services are nil.
	ErrNilService = errors.New("contacts service must not be nil")
)

// Source adapts contact application services to export source ports.
type Source struct {
	// service defines contact query dependencies.
	service contactsapplication.Service
	// consentSource defines optional contact consent lookups.
	consentSource port.ContactConsentSource
}

var (
	// _ ensures Source satisfies export source ports.
	_ port.ContactSource = (*Source)(nil)
)

// NewSource creates contact export source adapters.
func NewSource(service contactsapplication.Service, consentSources ...port.ContactConsentSource) (*Source, error) {
	if service == nil {
		return nil, ErrNilService
	}

	var consentSource port.ContactConsentSource
	if len(consentSources) > 0 {
		consentSource = consentSources[0]
	}

	return &Source{service: service, consentSource: consentSource}, nil
}

// ListContacts returns all contacts to export.
func (s *Source) ListContacts(ctx context.Context) ([]port.ContactRow, error) {
	rows := []port.ContactRow{}
	for page := 1; ; page++ {
		result, err := s.service.List(ctx, contactsport.ListQuery{
			Page:     page,
			Limit:    pageSize,
			OrderBy:  "createdAt",
			OrderDir: "asc",
		})
		if err != nil {
			return nil, fmt.Errorf("list contacts page %d: %w", page, err)
		}
		for _, contact := range result.Data {
			metadata := cloneMetadata(contact.Metadata)
			membershipOptIn, membershipOptInAt, err := s.resolveMembership(ctx, contact.ID, metadata)
			if err != nil {
				return nil, fmt.Errorf("resolve contact membership %s: %w", contact.ID, err)
			}
			privacyAccepted, privacyAcceptedAt := resolvePrivacy(metadata)
			rows = append(rows, port.ContactRow{
				ID:                contact.ID,
				DocumentType:      string(contact.DocumentType),
				DocumentNumber:    contact.DocumentNumber,
				LegalName:         contact.LegalName,
				FirstName:         contact.FirstName,
				LastName:          contact.LastName,
				Email:             contact.Email,
				Phone:             contact.Phone,
				Address:           contact.Address,
				AddressExtra:      contact.AddressExtra,
				CityCode:          contact.CityCode,
				MembershipOptIn:   membershipOptIn,
				MembershipOptInAt: membershipOptInAt,
				PrivacyAccepted:   privacyAccepted,
				PrivacyAcceptedAt: privacyAcceptedAt,
				Metadata:          metadata,
				CreatedAt:         contact.CreatedAt,
				UpdatedAt:         contact.UpdatedAt,
			})
		}
		if result.TotalPages == 0 || page >= result.TotalPages || len(result.Data) == 0 {
			break
		}
	}

	return rows, nil
}

// resolveMembership resolves latest membership opt-in state from status sources or metadata.
func (s *Source) resolveMembership(ctx context.Context, contactID string, metadata map[string]string) (bool, time.Time, error) {
	if s != nil && s.consentSource != nil {
		statuses, err := s.consentSource.GetContactStatuses(ctx, contactID)
		if err != nil {
			return false, time.Time{}, err
		}
		if len(statuses) > 0 {
			return membershipFromStatuses(statuses)
		}
	}

	return membershipFromMetadata(metadata)
}

// membershipFromStatuses resolves one exportable membership state from status rows.
func membershipFromStatuses(statuses []port.ContactConsentStatus) (bool, time.Time, error) {
	selected := port.ContactConsentStatus{}
	for _, preferredChannel := range []string{membershipPreferredChannelAll, membershipPreferredChannelEmail, membershipPreferredChannelFallback} {
		for _, status := range statuses {
			channel := strings.ToLower(strings.TrimSpace(status.Channel))
			if preferredChannel != membershipPreferredChannelFallback && channel != preferredChannel {
				continue
			}
			if preferredChannel == membershipPreferredChannelFallback && channel == membershipPreferredChannelAll {
				continue
			}
			selected = status
			break
		}
		if selected.Action != "" {
			break
		}
	}

	action := strings.ToLower(strings.TrimSpace(selected.Action))
	switch action {
	case membershipActionOptIn:
		return true, selected.OccurredAt.UTC(), nil
	case membershipActionOptOut:
		return false, selected.OccurredAt.UTC(), nil
	default:
		return false, time.Time{}, nil
	}
}

// membershipFromMetadata resolves legacy checker membership metadata values.
func membershipFromMetadata(metadata map[string]string) (bool, time.Time, error) {
	decision := normalizeDecision(metadata[checkerCircleOptInMetadataKey])
	switch decision {
	case checkerAcceptedValueConfirmed:
		return true, parseCheckerTime(metadata[checkerCircleOptInMetadataKey+checkerAcceptedAtUTCSuffix], metadata[checkerCircleOptInMetadataKey+checkerAcceptedAtSuffix]), nil
	case checkerRejectedValueConfirmed:
		return false, parseCheckerTime(metadata[checkerCircleOptInMetadataKey+checkerRejectedAtUTCSuffix], metadata[checkerCircleOptInMetadataKey+checkerRejectedAtSuffix]), nil
	default:
		return false, time.Time{}, nil
	}
}

// resolvePrivacy resolves privacy acceptance from legacy checker metadata.
func resolvePrivacy(metadata map[string]string) (bool, time.Time) {
	decision := normalizeDecision(metadata[checkerPrivacyAcceptMetadataKey])
	if decision != checkerAcceptedValueConfirmed {
		return false, time.Time{}
	}

	return true, parseCheckerTime(metadata[checkerPrivacyAcceptMetadataKey+checkerAcceptedAtUTCSuffix], metadata[checkerPrivacyAcceptMetadataKey+checkerAcceptedAtSuffix])
}

// normalizeDecision normalizes checker yes/no decisions.
func normalizeDecision(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	switch normalized {
	case "yes", "true", "1", "accepted", "accept":
		return checkerAcceptedValueConfirmed
	case "no", "false", "0", "rejected", "reject":
		return checkerRejectedValueConfirmed
	default:
		return ""
	}
}

// parseCheckerTime parses UTC/local checker timestamp values.
func parseCheckerTime(utcValue string, localValue string) time.Time {
	if parsedUTC, err := time.Parse(time.RFC3339, strings.TrimSpace(utcValue)); err == nil {
		return parsedUTC.UTC()
	}
	location, err := time.LoadLocation(checkerAcceptedAtLocalZone)
	if err != nil {
		location = time.FixedZone(checkerAcceptedAtFixedZone, checkerAcceptedAtFixedOffset)
	}
	if parsedLocal, parseErr := time.ParseInLocation(checkerAcceptedAtLocalLayout, strings.TrimSpace(localValue), location); parseErr == nil {
		return parsedLocal.UTC()
	}

	return time.Time{}
}

// cloneMetadata copies metadata maps.
func cloneMetadata(input map[string]string) map[string]string {
	if len(input) == 0 {
		return nil
	}
	output := make(map[string]string, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}
