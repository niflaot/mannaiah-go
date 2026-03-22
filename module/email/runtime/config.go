package runtime

// Config defines email runtime configuration values.
type Config struct {
	// Enabled defines whether email module wiring should be active.
	Enabled bool `mapstructure:"EMAIL_ENABLED" default:"false"`
	// Provider defines provider labels (ses).
	Provider string `mapstructure:"EMAIL_PROVIDER" default:"ses"`
	// SESRegion defines AWS SES region values.
	SESRegion string `mapstructure:"EMAIL_SES_REGION" default:""`
	// SESAccessKeyID defines AWS SES access key values.
	SESAccessKeyID string `mapstructure:"EMAIL_SES_ACCESS_KEY_ID" default:""`
	// SESSecretAccessKey defines AWS SES secret key values.
	SESSecretAccessKey string `mapstructure:"EMAIL_SES_SECRET_ACCESS_KEY" default:""`
	// SESFromAddress defines SES sender address values.
	SESFromAddress string `mapstructure:"EMAIL_SES_FROM_ADDRESS" default:""`
	// SESConfigurationSet defines SES configuration set values.
	SESConfigurationSet string `mapstructure:"EMAIL_SES_CONFIGURATION_SET" default:""`
	// SESMaxSendRate defines SES maximum send-rate values per second.
	SESMaxSendRate int `mapstructure:"EMAIL_SES_MAX_SEND_RATE" default:"14"`
	// AWSRegion defines legacy AWS region values.
	AWSRegion string `mapstructure:"EMAIL_AWS_REGION" default:""`
	// SenderAddress defines legacy sender address values.
	SenderAddress string `mapstructure:"EMAIL_SENDER_ADDRESS" default:""`
	// TrackingBaseURL defines the public base URL used to build open-tracking pixel URLs.
	// When empty, open tracking pixel injection is disabled.
	TrackingBaseURL string `mapstructure:"EMAIL_TRACKING_BASE_URL" default:""`
	// WebhookSNSTopicARN defines expected SNS topic arn values for SES webhook notifications.
	WebhookSNSTopicARN string `mapstructure:"EMAIL_WEBHOOK_SNS_TOPIC_ARN" default:""`
	// WebhookSNSVerifySignature enables SNS signature verification for webhook requests.
	WebhookSNSVerifySignature bool `mapstructure:"EMAIL_WEBHOOK_SNS_VERIFY_SIGNATURE" default:"true"`
	// WebhookSNSRequestTimeoutMS defines request timeout values for SNS signature/cert and subscription confirmation HTTP calls.
	WebhookSNSRequestTimeoutMS int `mapstructure:"EMAIL_WEBHOOK_SNS_REQUEST_TIMEOUT_MS" default:"5000"`
	// WebhookSoftBounceRetryDelaySeconds defines retry delay values for transient-bounce retry attempts.
	WebhookSoftBounceRetryDelaySeconds int `mapstructure:"EMAIL_WEBHOOK_SOFT_BOUNCE_RETRY_DELAY_SECONDS" default:"300"`
	// WebhookSoftBounceMaxRetries defines max retry attempts for transient-bounce handling.
	WebhookSoftBounceMaxRetries int `mapstructure:"EMAIL_WEBHOOK_SOFT_BOUNCE_MAX_RETRIES" default:"1"`
}
