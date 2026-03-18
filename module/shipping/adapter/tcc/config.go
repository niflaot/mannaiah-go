package tcc

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var (
	// ErrBaseURLRequired is returned when base URL values are empty.
	ErrBaseURLRequired = errors.New("tcc base url is required")
	// ErrAccessTokenRequired is returned when access token values are empty.
	ErrAccessTokenRequired = errors.New("tcc access token is required")
	// ErrAccountRequired is returned when account values are empty.
	ErrAccountRequired = errors.New("tcc account is required")
	// ErrIdentifierRequired is returned when identifier values are empty.
	ErrIdentifierRequired = errors.New("tcc identifier is required")
)

// Config defines TCC adapter configuration values.
type Config struct {
	// BaseURL defines TCC API base URL values.
	BaseURL string
	// AccessToken defines TCC authentication token values.
	AccessToken string
	// Account defines TCC account values.
	Account string
	// Identifier defines TCC client identifier values.
	Identifier string
	// LegalName defines TCC legal-name values kept for future dispatch flows.
	LegalName string
	// Timeout defines TCC request timeout values.
	Timeout time.Duration
	// HTTPClient defines optional custom HTTP client dependencies.
	HTTPClient *http.Client
}

// normalizeConfig resolves and validates TCC configuration values.
func normalizeConfig(cfg Config) (Config, error) {
	resolved := cfg
	resolved.BaseURL = strings.TrimSpace(cfg.BaseURL)
	resolved.AccessToken = strings.TrimSpace(cfg.AccessToken)
	resolved.Account = strings.TrimSpace(cfg.Account)
	resolved.Identifier = strings.TrimSpace(cfg.Identifier)
	resolved.LegalName = strings.TrimSpace(cfg.LegalName)

	if resolved.BaseURL == "" {
		return Config{}, ErrBaseURLRequired
	}
	if _, err := url.ParseRequestURI(resolved.BaseURL); err != nil {
		return Config{}, fmt.Errorf("%w: %v", ErrBaseURLRequired, err)
	}
	if resolved.AccessToken == "" {
		return Config{}, ErrAccessTokenRequired
	}
	if resolved.Account == "" {
		return Config{}, ErrAccountRequired
	}
	if resolved.Identifier == "" {
		return Config{}, ErrIdentifierRequired
	}
	if resolved.Timeout <= 0 {
		resolved.Timeout = 5 * time.Second
	}

	return resolved, nil
}

// resolveHTTPClient resolves HTTP client dependencies from config values.
func resolveHTTPClient(cfg Config) *http.Client {
	if cfg.HTTPClient != nil {
		return cfg.HTTPClient
	}

	return &http.Client{Timeout: cfg.Timeout}
}
