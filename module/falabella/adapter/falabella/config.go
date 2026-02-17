package falabella

import "time"

const (
	// defaultVersion defines Falabella API version values.
	defaultVersion = "1.0"
)

// Config defines Falabella client configuration values.
type Config struct {
	// URL defines Falabella base URL values.
	URL string
	// UserID defines Falabella API user values.
	UserID string
	// APIKey defines Falabella API key values.
	APIKey string
	// Timeout defines Falabella request timeout values.
	Timeout time.Duration
	// Version defines Falabella API version values.
	Version string
}
