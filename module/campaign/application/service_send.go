package application

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"mannaiah/module/campaign/domain"
	"mannaiah/module/campaign/port"
)

// Send starts asynchronous campaign fan-out and returns accepted campaign states.
func (s *CampaignService) Send(ctx context.Context, id string) (*domain.Campaign, error) {
	campaign, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if campaign.Status == domain.StatusProcessing || campaign.Status == domain.StatusSent {
		return nil, domain.ErrSendConflict
	}

	s.mutex.Lock()
	if _, busy := s.activeSends[campaign.ID]; busy {
		s.mutex.Unlock()
		return nil, domain.ErrSendConflict
	}
	s.activeSends[campaign.ID] = struct{}{}
	s.mutex.Unlock()

	campaign.Status = domain.StatusProcessing
	campaign.TotalRecipients = 0
	campaign.SentCount = 0
	campaign.FailedCount = 0
	if updateErr := s.repository.Update(ctx, campaign); updateErr != nil {
		s.mutex.Lock()
		delete(s.activeSends, campaign.ID)
		s.mutex.Unlock()
		return nil, fmt.Errorf("mark campaign processing: %w", updateErr)
	}

	runID := ""
	if s.syncRecorder != nil {
		startedRunID, runErr := s.syncRecorder.StartRun(ctx, "campaign.send", "manual")
		if runErr == nil {
			runID = startedRunID
		}
	}

	go s.processCampaignSend(context.Background(), campaign.ID, runID)
	return campaign, nil
}

// processCampaignSend runs asynchronous audience resolution and fan-out send behavior.
func (s *CampaignService) processCampaignSend(ctx context.Context, campaignID string, runID string) {
	defer func() {
		s.mutex.Lock()
		delete(s.activeSends, campaignID)
		s.mutex.Unlock()
	}()

	syncErrors := make([]port.SyncError, 0, 16)
	appendSyncError := func(errorType string, errorCode string, message string) {
		trimmedMessage := strings.TrimSpace(message)
		if trimmedMessage == "" {
			return
		}
		syncErrors = append(syncErrors, port.SyncError{
			Type:    strings.TrimSpace(errorType),
			Code:    strings.TrimSpace(errorCode),
			Message: trimmedMessage,
		})
	}
	finalizeSyncRecord := func(campaign *domain.Campaign) {
		if strings.TrimSpace(runID) == "" || s.syncRecorder == nil || campaign == nil {
			return
		}
		if campaign.FailedCount > 0 {
			_ = s.syncRecorder.FailRun(ctx, runID, campaign.TotalRecipients, campaign.SentCount, campaign.FailedCount, 0, syncErrors)
			return
		}
		_ = s.syncRecorder.CompleteRun(ctx, runID, campaign.TotalRecipients, campaign.SentCount, campaign.FailedCount, 0)
	}

	campaign, err := s.repository.GetByID(ctx, campaignID)
	if err != nil {
		appendSyncError("repository", "load_campaign", err.Error())
		if strings.TrimSpace(runID) != "" && s.syncRecorder != nil {
			_ = s.syncRecorder.FailRun(ctx, runID, 0, 0, 1, 0, syncErrors)
		}
		return
	}

	if s.resolver == nil || s.sender == nil {
		campaign.Status = domain.StatusFailed
		appendSyncError("dependency", "missing_dependency", "campaign resolver or sender is not configured")
		_ = s.repository.Update(ctx, campaign)
		finalizeSyncRecord(campaign)
		return
	}

	contactIDs := make([]string, 0, 1024)
	page := 1
	for {
		rows, resolveErr := s.resolver.ResolveSegment(ctx, campaign.SegmentID, page, 1000)
		if resolveErr != nil {
			campaign.Status = domain.StatusFailed
			appendSyncError("resolver", "resolve_segment", resolveErr.Error())
			_ = s.repository.Update(ctx, campaign)
			finalizeSyncRecord(campaign)
			return
		}
		if len(rows) == 0 {
			break
		}
		contactIDs = append(contactIDs, rows...)
		if len(rows) < 1000 {
			break
		}
		page++
	}

	emailMap, emailErr := s.resolver.ResolveEmails(ctx, contactIDs)
	if emailErr != nil {
		campaign.Status = domain.StatusFailed
		appendSyncError("resolver", "resolve_emails", emailErr.Error())
		_ = s.repository.Update(ctx, campaign)
		finalizeSyncRecord(campaign)
		return
	}

	campaign.TotalRecipients = len(contactIDs)
	if campaign.TotalRecipients == 0 {
		campaign.Status = domain.StatusSent
		_ = s.repository.Update(ctx, campaign)
		finalizeSyncRecord(campaign)
		return
	}

	type job struct {
		contactID string
		email     string
	}
	type sendResult struct {
		contactID string
		email     string
		err       error
	}
	jobs := make(chan job, len(contactIDs))
	results := make(chan sendResult, len(contactIDs))

	workerCount := s.workers
	if workerCount > len(contactIDs) {
		workerCount = len(contactIDs)
	}

	var waitGroup sync.WaitGroup
	for worker := 0; worker < workerCount; worker++ {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			for item := range jobs {
				idempotencyKey := campaign.ID + ":" + item.contactID
				sendErr := s.sender.SendCampaignEmail(ctx, item.contactID, item.email, campaign.Subject, campaign.HTMLBody, campaign.TextBody, idempotencyKey)
				results <- sendResult{contactID: item.contactID, email: item.email, err: sendErr}
			}
		}()
	}

	for _, contactID := range contactIDs {
		email := strings.TrimSpace(emailMap[contactID])
		if email == "" {
			campaign.FailedCount++
			appendSyncError("validation", "missing_email", "contact "+contactID+" has no email")
			s.publishDeliveryEvent(ctx, campaign.ID, contactID, campaign.Channel, "skipped_ineligible")
			continue
		}
		jobs <- job{contactID: contactID, email: email}
	}
	close(jobs)

	go func() {
		waitGroup.Wait()
		close(results)
	}()

	for result := range results {
		if result.err == nil {
			campaign.SentCount++
			s.publishDeliveryEvent(ctx, campaign.ID, result.contactID, campaign.Channel, "submitted_to_provider")
		} else {
			campaign.FailedCount++
			appendSyncError("delivery", "send_failed", "contact "+result.contactID+" email "+result.email+": "+result.err.Error())
			s.publishDeliveryEvent(ctx, campaign.ID, result.contactID, campaign.Channel, "failed")
		}
	}

	if campaign.FailedCount > 0 && campaign.SentCount == 0 {
		campaign.Status = domain.StatusFailed
	} else {
		campaign.Status = domain.StatusSent
	}
	_ = s.repository.Update(ctx, campaign)
	finalizeSyncRecord(campaign)
}

// publishDeliveryEvent publishes campaign delivery integration events.
func (s *CampaignService) publishDeliveryEvent(ctx context.Context, campaignID string, contactID string, channel string, status string) {
	if s == nil || s.publisher == nil {
		return
	}

	_ = s.publisher.Publish(ctx, buildCampaignDeliveryIntegrationEvent(campaignID, contactID, channel, status, 1, time.Now().UTC()))
}
