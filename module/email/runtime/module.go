package runtime

import (
	"context"
	"strings"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"gorm.io/gorm"
	corehttp "mannaiah/module/core/http"
	emailhttp "mannaiah/module/email/adapter/http"
	"mannaiah/module/email/adapter/ses"
	emailstore "mannaiah/module/email/adapter/store"
	"mannaiah/module/email/application"
	"mannaiah/module/email/port"
)

// Loader defines bootstrap hooks required by email modules.
type Loader interface {
	// RegisterRoutes registers module route handlers.
	RegisterRoutes(register func(router corehttp.Router))
	// AddOpenAPISpec merges module OpenAPI specs.
	AddOpenAPISpec(spec *openapi3.T) error
}

// Module defines composition-root wiring for email endpoints.
type Module struct {
	// cfg defines runtime configuration values.
	cfg Config
	// service defines email use-case dependencies.
	service *application.EmailService
	// handler defines HTTP route adapter dependencies.
	handler *emailhttp.Handler
	// repository defines email persistence dependencies.
	repository port.Repository
}

// New creates email modules with adapter wiring.
func New(cfg Config, db *gorm.DB) (*Module, error) {
	repository, err := emailstore.NewRepository(db)
	if err != nil {
		return nil, err
	}

	var provider port.Provider
	region := strings.TrimSpace(cfg.SESRegion)
	if region == "" {
		region = strings.TrimSpace(cfg.AWSRegion)
	}
	sender := strings.TrimSpace(cfg.SESFromAddress)
	if sender == "" {
		sender = strings.TrimSpace(cfg.SenderAddress)
	}
	if cfg.Enabled && strings.EqualFold(strings.TrimSpace(cfg.Provider), "ses") && sender != "" {
		sesProvider, providerErr := ses.NewProvider(context.Background(), ses.Config{
			Region:          region,
			Sender:          sender,
			AccessKeyID:     strings.TrimSpace(cfg.SESAccessKeyID),
			SecretAccessKey: strings.TrimSpace(cfg.SESSecretAccessKey),
		})
		if providerErr == nil {
			provider = sesProvider
		}
	}

	service, err := application.NewService(repository, provider)
	if err != nil {
		return nil, err
	}
	service.SetTrackingBaseURL(resolveTrackingBaseURL(cfg.TrackingBaseURL, sender))
	service.SetWebhookPolicy(
		cfg.WebhookSNSTopicARN,
		time.Duration(cfg.WebhookSoftBounceRetryDelaySeconds)*time.Second,
		cfg.WebhookSoftBounceMaxRetries,
	)
	if cfg.WebhookSNSVerifySignature {
		service.SetSNSMessageVerifier(ses.NewSNSMessageVerifier(ses.SNSMessageVerifierConfig{
			RequestTimeout: time.Duration(cfg.WebhookSNSRequestTimeoutMS) * time.Millisecond,
		}))
	}

	handler, err := emailhttp.NewHandler(service)
	if err != nil {
		return nil, err
	}

	return &Module{cfg: cfg, service: service, handler: handler, repository: repository}, nil
}

// resolveTrackingBaseURL resolves open-tracking base URLs from explicit config or sender-domain fallback.
func resolveTrackingBaseURL(configuredBaseURL string, senderEmail string) string {
	trimmedConfigured := strings.TrimSpace(configuredBaseURL)
	if trimmedConfigured != "" {
		return trimmedConfigured
	}

	trimmedSender := strings.TrimSpace(senderEmail)
	at := strings.LastIndex(trimmedSender, "@")
	if at <= 0 || at+1 >= len(trimmedSender) {
		return ""
	}
	domain := strings.TrimSpace(trimmedSender[at+1:])
	if domain == "" {
		return ""
	}

	return "https://" + domain
}

// RegisterRoutes registers email routes on the provided router.
func (m *Module) RegisterRoutes(router corehttp.Router) {
	if m == nil || m.handler == nil {
		return
	}

	m.handler.RegisterRoutes(router)
}

// SetAuthorizer configures endpoint authentication and permission dependencies.
func (m *Module) SetAuthorizer(authorizer emailhttp.Authorizer) {
	if m == nil || m.handler == nil {
		return
	}

	m.handler.SetAuthorizer(authorizer)
}

// SetMembershipStamper configures optional membership stamp dependencies.
func (m *Module) SetMembershipStamper(stamper port.MembershipStamper) {
	if m == nil || m.service == nil {
		return
	}

	m.service.SetMembershipStamper(stamper)
}

// Repository returns email repository dependencies.
func (m *Module) Repository() port.Repository {
	if m == nil {
		return nil
	}

	return m.repository
}

// Service returns email application service dependencies.
func (m *Module) Service() *application.EmailService {
	if m == nil {
		return nil
	}

	return m.service
}

// OpenAPISpec returns email-module OpenAPI documentation.
func (m *Module) OpenAPISpec() *openapi3.T {
	return OpenAPISpec()
}

// Load mounts all module routes/specs into the provided startup loader.
func (m *Module) Load(loader Loader) error {
	if m == nil || loader == nil {
		return nil
	}

	loader.RegisterRoutes(m.RegisterRoutes)
	if err := loader.AddOpenAPISpec(m.OpenAPISpec()); err != nil {
		return err
	}

	return nil
}
