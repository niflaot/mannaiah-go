package application

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"mannaiah/module/email/domain"
	"mannaiah/module/email/port"
)

type snsWebhookRepositoryStub struct {
	mutex             sync.Mutex
	deliveriesByID    map[string]*domain.Delivery
	providerToID      map[string]string
	statusEntriesByID map[string][]domain.StatusEntry
}

// CreateDelivery persists one in-memory delivery row.
func (s *snsWebhookRepositoryStub) CreateDelivery(ctx context.Context, delivery *domain.Delivery) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.deliveriesByID == nil {
		s.deliveriesByID = map[string]*domain.Delivery{}
	}
	if s.providerToID == nil {
		s.providerToID = map[string]string{}
	}
	copy := *delivery
	s.deliveriesByID[copy.ID] = &copy
	if strings.TrimSpace(copy.ProviderMessageID) != "" {
		s.providerToID[strings.TrimSpace(copy.ProviderMessageID)] = copy.ID
	}

	return nil
}

// UpdateDeliveryStatus updates one in-memory delivery status row.
func (s *snsWebhookRepositoryStub) UpdateDeliveryStatus(ctx context.Context, deliveryID string, status domain.DeliveryStatus, providerMessageID string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	delivery, exists := s.deliveriesByID[deliveryID]
	if !exists {
		return domain.ErrNotFound
	}
	delivery.Status = status
	trimmedProviderMessageID := strings.TrimSpace(providerMessageID)
	if trimmedProviderMessageID != "" {
		if old := strings.TrimSpace(delivery.ProviderMessageID); old != "" {
			delete(s.providerToID, old)
		}
		delivery.ProviderMessageID = trimmedProviderMessageID
		s.providerToID[trimmedProviderMessageID] = deliveryID
	}

	return nil
}

// AddStatusEntry appends one in-memory status-entry row.
func (s *snsWebhookRepositoryStub) AddStatusEntry(ctx context.Context, entry *domain.StatusEntry) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.statusEntriesByID == nil {
		s.statusEntriesByID = map[string][]domain.StatusEntry{}
	}
	s.statusEntriesByID[entry.DeliveryID] = append(s.statusEntriesByID[entry.DeliveryID], *entry)

	return nil
}

// CountStatusEntries counts in-memory status entries for one delivery and status.
func (s *snsWebhookRepositoryStub) CountStatusEntries(ctx context.Context, deliveryID string, status domain.DeliveryStatus) (int64, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	var total int64
	for _, entry := range s.statusEntriesByID[deliveryID] {
		if entry.Status == status {
			total++
		}
	}

	return total, nil
}

// GetByID retrieves one in-memory delivery row by id.
func (s *snsWebhookRepositoryStub) GetByID(ctx context.Context, id string) (*domain.Delivery, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	delivery, exists := s.deliveriesByID[id]
	if !exists {
		return nil, domain.ErrNotFound
	}
	copy := *delivery

	return &copy, nil
}

// GetByProviderMessageID retrieves one in-memory delivery row by provider message id.
func (s *snsWebhookRepositoryStub) GetByProviderMessageID(ctx context.Context, providerMessageID string) (*domain.Delivery, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	id, exists := s.providerToID[providerMessageID]
	if !exists {
		return nil, domain.ErrNotFound
	}
	delivery, exists := s.deliveriesByID[id]
	if !exists {
		return nil, domain.ErrNotFound
	}
	copy := *delivery

	return &copy, nil
}

// ListByEmail retrieves in-memory delivery rows by recipient email ordered by created time descending.
func (s *snsWebhookRepositoryStub) ListByEmail(ctx context.Context, email string) ([]*domain.Delivery, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	normalizedEmail := strings.ToLower(strings.TrimSpace(email))
	rows := make([]*domain.Delivery, 0)
	for _, delivery := range s.deliveriesByID {
		if strings.ToLower(strings.TrimSpace(delivery.Email)) == normalizedEmail {
			copy := *delivery
			rows = append(rows, &copy)
		}
	}

	sort.Slice(rows, func(i int, j int) bool {
		if rows[i].CreatedAt.Equal(rows[j].CreatedAt) {
			return rows[i].ID > rows[j].ID
		}

		return rows[i].CreatedAt.After(rows[j].CreatedAt)
	})

	return rows, nil
}

type snsWebhookProviderStub struct {
	send func(ctx context.Context, request port.SendRequest) (string, error)
}

// Send dispatches one in-memory provider send request.
func (s snsWebhookProviderStub) Send(ctx context.Context, request port.SendRequest) (string, error) {
	if s.send != nil {
		return s.send(ctx, request)
	}

	return "provider-message-id", nil
}

type snsWebhookMembershipStub struct {
	mutex sync.Mutex
	calls []struct {
		email  string
		source string
	}
}

// OptOutByEmail records one in-memory membership opt-out call.
func (s *snsWebhookMembershipStub) OptOutByEmail(ctx context.Context, email string, source string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.calls = append(s.calls, struct {
		email  string
		source string
	}{email: strings.TrimSpace(email), source: strings.TrimSpace(source)})

	return nil
}

type snsWebhookVerifierStub struct {
	err error
}

// Verify returns the preconfigured verifier result.
func (s snsWebhookVerifierStub) Verify(ctx context.Context, message port.SNSMessage) error {
	return s.err
}

// TestHandleWebhookSNSComplaintOptsOut verifies complaint webhook handling updates delivery status and membership opt-out.
func TestHandleWebhookSNSComplaintOptsOut(t *testing.T) {
	t.Parallel()

	repository := &snsWebhookRepositoryStub{
		deliveriesByID: map[string]*domain.Delivery{
			"d-1": {ID: "d-1", Email: "user@example.com", ProviderMessageID: "msg-1", Status: domain.StatusSubmitted},
		},
		providerToID: map[string]string{"msg-1": "d-1"},
	}
	membership := &snsWebhookMembershipStub{}
	service, err := NewService(repository, snsWebhookProviderStub{}, membership)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	service.SetWebhookPolicy("arn:aws:sns:us-east-1:123456789012:ses-events", 0, 0)
	service.SetSNSMessageVerifier(snsWebhookVerifierStub{})

	webhookErr := service.HandleWebhook(context.Background(), WebhookCommand{
		MessageType: "Notification",
		TopicARN:    "arn:aws:sns:us-east-1:123456789012:ses-events",
		Message:     `{"notificationType":"Complaint","mail":{"messageId":"msg-1","destination":["user@example.com"]},"complaint":{"complaintFeedbackType":"abuse","complainedRecipients":[{"emailAddress":"user@example.com"}]}}`,
	})
	if webhookErr != nil {
		t.Fatalf("HandleWebhook() error = %v", webhookErr)
	}

	delivery, getErr := repository.GetByID(context.Background(), "d-1")
	if getErr != nil {
		t.Fatalf("GetByID() error = %v", getErr)
	}
	if delivery.Status != domain.StatusComplained {
		t.Fatalf("delivery.Status = %q, want %q", delivery.Status, domain.StatusComplained)
	}

	membership.mutex.Lock()
	callCount := len(membership.calls)
	call := membership.calls[0]
	membership.mutex.Unlock()
	if callCount != 1 {
		t.Fatalf("membership opt-out calls = %d, want 1", callCount)
	}
	if call.email != "user@example.com" || call.source != "ses_complaint" {
		t.Fatalf("membership call = %+v, want user@example.com + ses_complaint", call)
	}
}

// TestHandleWebhookSNSPermanentBounceOptsOut verifies permanent-bounce webhook handling updates delivery status and membership opt-out.
func TestHandleWebhookSNSPermanentBounceOptsOut(t *testing.T) {
	t.Parallel()

	repository := &snsWebhookRepositoryStub{
		deliveriesByID: map[string]*domain.Delivery{
			"d-2": {ID: "d-2", Email: "user2@example.com", ProviderMessageID: "msg-2", Status: domain.StatusSubmitted},
		},
		providerToID: map[string]string{"msg-2": "d-2"},
	}
	membership := &snsWebhookMembershipStub{}
	service, err := NewService(repository, snsWebhookProviderStub{}, membership)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	service.SetSNSMessageVerifier(snsWebhookVerifierStub{})

	webhookErr := service.HandleWebhook(context.Background(), WebhookCommand{
		MessageType: "Notification",
		Message:     `{"notificationType":"Bounce","mail":{"messageId":"msg-2","destination":["user2@example.com"]},"bounce":{"bounceType":"Permanent","bounceSubType":"General","bouncedRecipients":[{"emailAddress":"user2@example.com"}]}}`,
	})
	if webhookErr != nil {
		t.Fatalf("HandleWebhook() error = %v", webhookErr)
	}

	delivery, getErr := repository.GetByID(context.Background(), "d-2")
	if getErr != nil {
		t.Fatalf("GetByID() error = %v", getErr)
	}
	if delivery.Status != domain.StatusBounced {
		t.Fatalf("delivery.Status = %q, want %q", delivery.Status, domain.StatusBounced)
	}

	membership.mutex.Lock()
	callCount := len(membership.calls)
	call := membership.calls[0]
	membership.mutex.Unlock()
	if callCount != 1 {
		t.Fatalf("membership opt-out calls = %d, want 1", callCount)
	}
	if call.email != "user2@example.com" || call.source != "ses_bounce_permanent" {
		t.Fatalf("membership call = %+v, want user2@example.com + ses_bounce_permanent", call)
	}
}

// TestHandleWebhookSNSTransientBounceRetries verifies transient-bounce webhook handling schedules retry sends.
func TestHandleWebhookSNSTransientBounceRetries(t *testing.T) {
	t.Parallel()

	sendTriggered := make(chan struct{}, 1)
	repository := &snsWebhookRepositoryStub{
		deliveriesByID: map[string]*domain.Delivery{
			"d-3": {
				ID:                "d-3",
				Email:             "user3@example.com",
				Subject:           "Hello",
				HTMLBody:          "<html><body>Hello</body></html>",
				TextBody:          "Hello",
				IdempotencyKey:    "idem-3",
				ProviderMessageID: "msg-3",
				Status:            domain.StatusSubmitted,
			},
		},
		providerToID:      map[string]string{"msg-3": "d-3"},
		statusEntriesByID: map[string][]domain.StatusEntry{},
	}
	service, err := NewService(repository, snsWebhookProviderStub{
		send: func(ctx context.Context, request port.SendRequest) (string, error) {
			sendTriggered <- struct{}{}
			return "msg-3-retry", nil
		},
	})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	service.SetWebhookPolicy("", 0, 1)
	service.SetSNSMessageVerifier(snsWebhookVerifierStub{})

	webhookErr := service.HandleWebhook(context.Background(), WebhookCommand{
		MessageType: "Notification",
		Message:     `{"notificationType":"Bounce","mail":{"messageId":"msg-3","destination":["user3@example.com"]},"bounce":{"bounceType":"Transient","bounceSubType":"General","bouncedRecipients":[{"emailAddress":"user3@example.com"}]}}`,
	})
	if webhookErr != nil {
		t.Fatalf("HandleWebhook() error = %v", webhookErr)
	}

	select {
	case <-sendTriggered:
	case <-time.After(2 * time.Second):
		t.Fatalf("soft-bounce retry send was not triggered")
	}

	deadline := time.Now().Add(2 * time.Second)
	for {
		delivery, getErr := repository.GetByID(context.Background(), "d-3")
		if getErr != nil {
			t.Fatalf("GetByID() error = %v", getErr)
		}
		if delivery.Status == domain.StatusSubmitted && delivery.ProviderMessageID == "msg-3-retry" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("delivery status/provider message id not updated after retry: status=%q providerMessageID=%q", delivery.Status, delivery.ProviderMessageID)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// TestHandleWebhookSNSInvalidSignature verifies signature-validation failures map to invalid webhook signature errors.
func TestHandleWebhookSNSInvalidSignature(t *testing.T) {
	t.Parallel()

	service, err := NewService(&snsWebhookRepositoryStub{}, snsWebhookProviderStub{})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	service.SetSNSMessageVerifier(snsWebhookVerifierStub{err: errors.New("invalid signature")})

	webhookErr := service.HandleWebhook(context.Background(), WebhookCommand{
		MessageType: "Notification",
		Message:     `{"notificationType":"Delivery","mail":{"messageId":"msg-404"}}`,
	})
	if webhookErr == nil {
		t.Fatalf("HandleWebhook() error = nil, want invalid signature error")
	}
	if !errors.Is(webhookErr, domain.ErrInvalidWebhookSignature) {
		t.Fatalf("HandleWebhook() error = %v, want ErrInvalidWebhookSignature", webhookErr)
	}
}

// TestHandleWebhookSNSSubscriptionConfirmation verifies subscription confirmations are acknowledged via SubscribeURL.
func TestHandleWebhookSNSSubscriptionConfirmation(t *testing.T) {
	t.Parallel()

	server := httptest.NewTLSServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	serverURL, parseErr := url.Parse(server.URL)
	if parseErr != nil {
		t.Fatalf("Parse(server.URL) error = %v", parseErr)
	}

	service, err := NewService(&snsWebhookRepositoryStub{}, snsWebhookProviderStub{})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	service.SetSNSMessageVerifier(snsWebhookVerifierStub{})
	service.webhookHTTPClient = &http.Client{
		Timeout: 2 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			DialContext: func(ctx context.Context, network string, address string) (net.Conn, error) {
				return (&net.Dialer{Timeout: 2 * time.Second}).DialContext(ctx, network, serverURL.Host)
			},
		},
	}

	webhookErr := service.HandleWebhook(context.Background(), WebhookCommand{
		MessageType:  "SubscriptionConfirmation",
		SubscribeURL: "https://sns.us-east-1.amazonaws.com/?Action=ConfirmSubscription&TopicArn=arn",
		MessageID:    "message-id",
		Timestamp:    "2026-03-22T20:20:20.000Z",
		TopicARN:     "arn:aws:sns:us-east-1:123456789012:ses-events",
		Token:        "token",
		Message:      "confirm",
	})
	if webhookErr != nil {
		t.Fatalf("HandleWebhook() error = %v", webhookErr)
	}
}
