package falabella

import (
	"time"

	"go.uber.org/zap"
)

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
	// UserAgent defines Falabella API User-Agent header values.
	UserAgent string
	// Timeout defines Falabella request timeout values.
	Timeout time.Duration
	// Version defines Falabella API version values.
	Version string
	// Logger defines optional adapter debug logger values.
	Logger *zap.Logger
}
