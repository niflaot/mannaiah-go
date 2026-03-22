package application

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"mannaiah/module/campaign/domain"
)

// TestSend renders and delivers the campaign to a single override email for preview purposes.
// Campaign status and counters are not modified.
func (s *CampaignService) TestSend(ctx context.Context, campaignID string, command TestSendCommand) (*TestSendResult, error) {
	campaign, err := s.Get(ctx, campaignID)
	if err != nil {
		return nil, err
	}

	email := strings.ToLower(strings.TrimSpace(command.Email))
	if email == "" {
		return nil, domain.ErrInvalidTestEmail
	}

	if s.sender == nil {
		return nil, domain.ErrSenderNotConfigured
	}

	contactID := strings.TrimSpace(command.ContactID)
	htmlBody, textBody, renderErr := s.renderForContactStrict(ctx, campaign, contactID, email)
	if renderErr != nil {
		if errors.Is(renderErr, domain.ErrContactPersonalization) {
			return nil, renderErr
		}
		return nil, fmt.Errorf("%w: %v", domain.ErrInvalidTemplate, renderErr)
	}

	idempotencyKey := "test:" + campaign.ID + ":" + uuid.NewString()
	if err := s.sender.SendCampaignEmail(ctx, contactID, email, campaign.Subject, htmlBody, textBody, idempotencyKey); err != nil {
		return nil, fmt.Errorf("test send campaign email: %w", normalizeSenderError(err))
	}

	return &TestSendResult{
		Email:   email,
		Subject: campaign.Subject,
		Status:  "submitted",
	}, nil
}
