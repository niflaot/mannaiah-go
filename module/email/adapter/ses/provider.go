package ses

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
	"mannaiah/module/email/port"
)

// sesTagInvalidChars matches characters not allowed in SES email tag values.
var sesTagInvalidChars = regexp.MustCompile(`[^a-zA-Z0-9_\-.@]`)

var (
	// ErrSenderRequired is returned when sender values are blank.
	ErrSenderRequired = errors.New("email sender address is required")
)

// Config defines SES provider configuration values.
type Config struct {
	// Region defines AWS region values.
	Region string
	// Sender defines SES sender email values.
	Sender string
	// SenderName defines the display name shown alongside the sender address (e.g. "Flock").
	// When set, the From header becomes "SenderName <Sender>".
	SenderName string
	// AccessKeyID defines static AWS access key values.
	AccessKeyID string
	// SecretAccessKey defines static AWS secret key values.
	SecretAccessKey string
}

// Provider defines SES-backed email delivery behavior.
type Provider struct {
	// client defines SES API dependencies.
	client *sesv2.Client
	// sender defines sender email values.
	sender string
}

var (
	// _ ensures Provider satisfies provider contracts.
	_ port.Provider = (*Provider)(nil)
)

// NewProvider creates SES-backed provider dependencies.
func NewProvider(ctx context.Context, cfg Config) (*Provider, error) {
	sender := strings.TrimSpace(cfg.Sender)
	if sender == "" {
		return nil, ErrSenderRequired
	}

	loaderOpts := []func(*awsconfig.LoadOptions) error{
		func(options *awsconfig.LoadOptions) error {
			if strings.TrimSpace(cfg.Region) != "" {
				options.Region = strings.TrimSpace(cfg.Region)
			}
			return nil
		},
	}
	if strings.TrimSpace(cfg.AccessKeyID) != "" && strings.TrimSpace(cfg.SecretAccessKey) != "" {
		loaderOpts = append(loaderOpts, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				strings.TrimSpace(cfg.AccessKeyID),
				strings.TrimSpace(cfg.SecretAccessKey),
				"",
			),
		))
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, loaderOpts...)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	from := sender
	if name := strings.TrimSpace(cfg.SenderName); name != "" {
		from = fmt.Sprintf("%s <%s>", name, sender)
	}
	return &Provider{client: sesv2.NewFromConfig(awsCfg), sender: from}, nil
}

// Send submits one email request and returns provider message ids.
func (p *Provider) Send(ctx context.Context, request port.SendRequest) (string, error) {
	result, err := p.client.SendEmail(ctx, &sesv2.SendEmailInput{
		FromEmailAddress: &p.sender,
		Destination: &types.Destination{
			ToAddresses: []string{strings.TrimSpace(request.To)},
		},
		Content: &types.EmailContent{
			Simple: &types.Message{
				Subject: &types.Content{Data: &request.Subject},
				Body: &types.Body{
					Html: &types.Content{Data: &request.HTMLBody},
					Text: &types.Content{Data: &request.TextBody},
				},
			},
		},
		EmailTags: []types.MessageTag{{Name: ptr("idempotency_key"), Value: ptr(sanitizeSESTagValue(request.IdempotencyKey))}},
	})
	if err != nil {
		return "", fmt.Errorf("send ses email: %w", err)
	}
	if result == nil || result.MessageId == nil {
		return "", nil
	}

	return strings.TrimSpace(*result.MessageId), nil
}

// sanitizeSESTagValue replaces characters not allowed in SES tag values with '_'.
func sanitizeSESTagValue(value string) string {
	return sesTagInvalidChars.ReplaceAllString(strings.TrimSpace(value), "_")
}

// ptr resolves string pointers for SES payload values.
func ptr(value string) *string {
	resolved := value
	return &resolved
}
