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
}
