package application

import (
	"context"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strings"
	"sync"
	"time"

	"mannaiah/module/campaign/domain"
	"mannaiah/module/campaign/port"
)

var (
	// ErrNilRepository is returned when nil repository dependencies are provided.
	ErrNilRepository = errors.New("campaign repository must not be nil")
)

var slugPattern = regexp.MustCompile(`^[a-z0-9-]+$`)

// CreateCommand defines campaign creation payload values.
type CreateCommand struct {
	// Name defines campaign names.
	Name string
	// Slug defines campaign slugs.
	Slug string
	// Channel defines target channel values.
	Channel string
	// SegmentID defines target segment identifier values.
	SegmentID string
	// Subject defines email subject values.
	Subject string
	// HTMLBody defines html content values.
	HTMLBody string
	// TextBody defines text content values.
	TextBody string
}

// UpdateCommand defines campaign update payload values.
type UpdateCommand struct {
	// Name defines optional campaign names.
	Name *string
	// Slug defines optional campaign slugs.
	Slug *string
	// Channel defines optional target channel values.
	Channel *string
	// SegmentID defines optional target segment identifier values.
	SegmentID *string
	// Subject defines optional email subject values.
	Subject *string
	// HTMLBody defines optional html content values.
	HTMLBody *string
	// TextBody defines optional text content values.
	TextBody *string
}

// ListResult defines paged campaign query output values.
type ListResult struct {
	// Data defines campaign rows in the current page.
	Data []domain.Campaign `json:"data"`
	// Page defines current page number.
	Page int `json:"page"`
	// Limit defines current page size.
	Limit int `json:"limit"`
	// Total defines total matching rows.
	Total int64 `json:"total"`
	// TotalPages defines total available pages.
	TotalPages int `json:"totalPages"`
}

// Service defines campaign use-case behavior.
type Service interface {
	// Create persists campaign rows.
	Create(ctx context.Context, command CreateCommand) (*domain.Campaign, error)
	// Get retrieves one campaign by id.
	Get(ctx context.Context, id string) (*domain.Campaign, error)
	// List retrieves paged campaign rows.
	List(ctx context.Context, page int, limit int) (*ListResult, error)
	// Update persists campaign row updates.
	Update(ctx context.Context, id string, command UpdateCommand) (*domain.Campaign, error)
	// Delete removes one campaign by id.
	Delete(ctx context.Context, id string) error
	// Send starts asynchronous campaign fan-out and returns accepted campaign states.
	Send(ctx context.Context, id string) (*domain.Campaign, error)
}

// CampaignService implements campaign use-cases.
type CampaignService struct {
	// repository defines campaign persistence dependencies.
	repository port.Repository
	// resolver defines segment audience resolution dependencies.
	resolver port.SegmentResolver
	// sender defines outbound email sender dependencies.
	sender port.EmailSender
	// workers defines bounded fan-out worker counts.
	workers int
	// syncRecorder defines optional sync run recording dependencies.
	syncRecorder port.SyncRecorder
	// publisher defines integration event publication dependencies.
	publisher port.IntegrationEventPublisher
	// mutex guards active send guards.
	mutex sync.Mutex
	// activeSends prevents duplicate in-memory sends per campaign.
	activeSends map[string]struct{}
}

// NewService creates campaign services.
func NewService(repository port.Repository, resolver port.SegmentResolver, sender port.EmailSender, workers int, publisher port.IntegrationEventPublisher) (*CampaignService, error) {
	if repository == nil {
		return nil, ErrNilRepository
	}
	if workers <= 0 {
		workers = 8
	}

	return &CampaignService{
		repository:   repository,
		resolver:     resolver,
		sender:       sender,
		workers:      workers,
		syncRecorder: port.NoopSyncRecorder{},
		publisher:    resolvePublisher(publisher),
		activeSends:  map[string]struct{}{},
	}, nil
}

// SetSyncRecorder configures optional sync run recording dependencies.
func (s *CampaignService) SetSyncRecorder(recorder port.SyncRecorder) {
	if s == nil {
		return
	}
	if recorder == nil {
		s.syncRecorder = port.NoopSyncRecorder{}
		return
	}

	s.syncRecorder = recorder
}

// Create persists campaign rows.
func (s *CampaignService) Create(ctx context.Context, command CreateCommand) (*domain.Campaign, error) {
	name := strings.TrimSpace(command.Name)
	if name == "" {
		return nil, domain.ErrInvalidName
	}
	slug := strings.TrimSpace(strings.ToLower(command.Slug))
	if slug == "" || !slugPattern.MatchString(slug) {
		return nil, domain.ErrInvalidSlug
	}

	campaign := &domain.Campaign{
		Name:      name,
		Slug:      slug,
		Channel:   strings.TrimSpace(command.Channel),
		SegmentID: strings.TrimSpace(command.SegmentID),
		Subject:   strings.TrimSpace(command.Subject),
		HTMLBody:  command.HTMLBody,
		TextBody:  command.TextBody,
		Status:    domain.StatusPlanned,
	}
	if campaign.Channel == "" {
		campaign.Channel = "email"
	}

	if err := s.repository.Create(ctx, campaign); err != nil {
		return nil, fmt.Errorf("create campaign: %w", err)
	}

	return campaign, nil
}

// Get retrieves one campaign by id.
func (s *CampaignService) Get(ctx context.Context, id string) (*domain.Campaign, error) {
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return nil, domain.ErrInvalidID
	}

	campaign, err := s.repository.GetByID(ctx, trimmedID)
	if err != nil {
		return nil, fmt.Errorf("get campaign: %w", err)
	}

	return campaign, nil
}

// List retrieves paged campaign rows.
func (s *CampaignService) List(ctx context.Context, page int, limit int) (*ListResult, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}

	rows, total, err := s.repository.List(ctx, page, limit)
	if err != nil {
		return nil, fmt.Errorf("list campaigns: %w", err)
	}
	totalPages := 0
	if total > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(limit)))
	}

	return &ListResult{Data: rows, Page: page, Limit: limit, Total: total, TotalPages: totalPages}, nil
}

// Update persists campaign row updates.
func (s *CampaignService) Update(ctx context.Context, id string, command UpdateCommand) (*domain.Campaign, error) {
	campaign, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if campaign.Status == domain.StatusProcessing || campaign.Status == domain.StatusSent {
		return nil, domain.ErrSendConflict
	}

	if command.Name != nil {
		name := strings.TrimSpace(*command.Name)
		if name == "" {
			return nil, domain.ErrInvalidName
		}
		campaign.Name = name
	}
	if command.Slug != nil {
		slug := strings.TrimSpace(strings.ToLower(*command.Slug))
		if slug == "" || !slugPattern.MatchString(slug) {
			return nil, domain.ErrInvalidSlug
		}
		campaign.Slug = slug
	}
	if command.Channel != nil {
		campaign.Channel = strings.TrimSpace(*command.Channel)
	}
	if command.SegmentID != nil {
		campaign.SegmentID = strings.TrimSpace(*command.SegmentID)
	}
	if command.Subject != nil {
		campaign.Subject = strings.TrimSpace(*command.Subject)
	}
	if command.HTMLBody != nil {
		campaign.HTMLBody = *command.HTMLBody
	}
	if command.TextBody != nil {
		campaign.TextBody = *command.TextBody
	}

	if err := s.repository.Update(ctx, campaign); err != nil {
		return nil, fmt.Errorf("update campaign: %w", err)
	}

	return campaign, nil
}

// Delete removes one campaign by id.
func (s *CampaignService) Delete(ctx context.Context, id string) error {
	trimmedID := strings.TrimSpace(id)
	if trimmedID == "" {
		return domain.ErrInvalidID
	}

	if err := s.repository.Delete(ctx, trimmedID); err != nil {
		return fmt.Errorf("delete campaign: %w", err)
	}

	return nil
}

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
				err := s.sender.SendCampaignEmail(ctx, item.contactID, item.email, campaign.Subject, campaign.HTMLBody, campaign.TextBody, idempotencyKey)
				results <- sendResult{contactID: item.contactID, email: item.email, err: err}
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
