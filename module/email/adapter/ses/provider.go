package ses

import (
	"context"
	"errors"
	"fmt"
	"strings"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
	"mannaiah/module/email/port"
)

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

	return &Provider{client: sesv2.NewFromConfig(awsCfg), sender: sender}, nil
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
		EmailTags: []types.MessageTag{{Name: ptr("idempotency_key"), Value: ptr(strings.TrimSpace(request.IdempotencyKey))}},
	})
	if err != nil {
		return "", fmt.Errorf("send ses email: %w", err)
	}
	if result == nil || result.MessageId == nil {
		return "", nil
	}

	return strings.TrimSpace(*result.MessageId), nil
}

// ptr resolves string pointers for SES payload values.
func ptr(value string) *string {
	resolved := value
	return &resolved
}
