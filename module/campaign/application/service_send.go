package application

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	campaigntemplate "mannaiah/module/campaign/application/template"
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
				htmlBody, textBody := s.renderForContact(ctx, campaign, item.contactID, item.email)
				idempotencyKey := campaign.ID + ":" + item.contactID
				sendErr := normalizeSenderError(s.sender.SendCampaignEmail(ctx, item.contactID, item.email, campaign.Subject, htmlBody, textBody, idempotencyKey))
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

// renderForContact builds a per-contact template context and renders HTML and text bodies.
// Falls back to the raw campaign bodies on any enrichment or render failure (fail-open).
func (s *CampaignService) renderForContact(ctx context.Context, campaign *domain.Campaign, contactID string, email string) (htmlBody string, textBody string) {
	htmlBody, textBody, err := s.renderForContactStrict(ctx, campaign, contactID, email)
	if err != nil {
		return campaign.HTMLBody, campaign.TextBody
	}

	return htmlBody, textBody
}

// renderForContactStrict renders one campaign template context and returns render errors to callers.
func (s *CampaignService) renderForContactStrict(ctx context.Context, campaign *domain.Campaign, contactID string, email string) (htmlBody string, textBody string, err error) {
	htmlBody = campaign.HTMLBody
	textBody = campaign.TextBody

	if s.templateRenderer == nil {
		return htmlBody, textBody, nil
	}

	tplCtx := s.buildTemplateContext(ctx, campaign, contactID, email)

	renderedHTML, renderHTMLErr := s.templateRenderer.Render("html:"+campaign.ID, campaign.HTMLBody, tplCtx)
	if renderHTMLErr != nil {
		return "", "", fmt.Errorf("render html template: %w", renderHTMLErr)
	}
	renderedText, renderTextErr := s.templateRenderer.Render("text:"+campaign.ID, campaign.TextBody, tplCtx)
	if renderTextErr != nil {
		return "", "", fmt.Errorf("render text template: %w", renderTextErr)
	}

	return campaigntemplate.RewriteLinks(renderedHTML, campaign.ID, campaign.Slug), renderedText, nil
}

// buildTemplateContext builds one per-contact template context value.
func (s *CampaignService) buildTemplateContext(ctx context.Context, campaign *domain.Campaign, contactID string, email string) domain.TemplateContext {
	// Build contact data (fail-open: use defaults on error).
	contactData, _ := s.contactDataProvider.GetContactData(ctx, contactID)
	if contactData.Name == "" {
		contactData.Name = email
	}

	// Build product blocks (fail-open: skip failed blocks).
	products := make(map[string][]domain.TemplateProduct, len(campaign.ProductBlocks))
	for _, block := range campaign.ProductBlocks {
		if strings.TrimSpace(block.ID) == "" || !hasProductSource(block) {
			continue
		}
		items, err := s.affinityProductProvider.GetProducts(ctx, contactID, block)
		if err != nil {
			continue
		}
		if items == nil {
			items = []domain.TemplateProduct{}
		}
		products[block.ID] = items
	}

	return domain.TemplateContext{
		Contact: domain.ContactTemplateData{
			Name:         contactData.Name,
			FullName:     contactData.Name,
			FirstName:    campaigntemplate.ExtractFirstName(contactData.Name),
			Email:        email,
			LastSaleDate: contactData.LastSaleDate,
		},
		Custom:   campaign.TemplateVars,
		Products: products,
	}
}

// hasProductSource reports whether a product block has at least one source capable of resolving products.
func hasProductSource(block domain.ProductBlock) bool {
	if strings.TrimSpace(block.BaseTag) != "" {
		return true
	}
	for _, tag := range block.BaseTags {
		if strings.TrimSpace(tag) != "" {
			return true
		}
	}

	return len(block.PinnedProductIDs) > 0
}

// publishDeliveryEvent publishes campaign delivery integration events.
func (s *CampaignService) publishDeliveryEvent(ctx context.Context, campaignID string, contactID string, channel string, status string) {
	if s == nil || s.publisher == nil {
		return
	}

	_ = s.publisher.Publish(ctx, buildCampaignDeliveryIntegrationEvent(campaignID, contactID, channel, status, 1, time.Now().UTC()))
}
