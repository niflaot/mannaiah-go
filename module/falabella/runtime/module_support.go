package runtime

import (
	"context"
	"strings"
	"time"

	falabellahttp "mannaiah/module/falabella/adapter/http"

	"go.uber.org/zap"
)

// resolveContext resolves nil contexts to background defaults.
func resolveContext(ctx context.Context) context.Context {
	if ctx != nil {
		return ctx
	}

	return context.Background()
}

// resolveLogger resolves nil loggers to no-op defaults.
func resolveLogger(providedLogger *zap.Logger) *zap.Logger {
	if providedLogger != nil {
		return providedLogger
	}

	return zap.NewNop()
}

// resolveValidationTimeout resolves startup validation timeout values.
func resolveValidationTimeout(timeoutMS int) time.Duration {
	if timeoutMS <= 0 {
		return 3 * time.Second
	}

	return time.Duration(timeoutMS) * time.Millisecond
}

// resolveRequestTimeout resolves Falabella request timeout values.
func resolveRequestTimeout(timeoutMS int) time.Duration {
	if timeoutMS <= 0 {
		return 5 * time.Second
	}

	return time.Duration(timeoutMS) * time.Millisecond
}

// resolveImageTranscodeTimeout resolves image transcode source-request timeout values.
func resolveImageTranscodeTimeout(timeoutMS int) time.Duration {
	if timeoutMS <= 0 {
		return 15 * time.Second
	}

	return time.Duration(timeoutMS) * time.Millisecond
}

// resolveImageTranscodeAllowedPrefixes resolves allowed source URL prefixes for image transcode endpoints.
func resolveImageTranscodeAllowedPrefixes(cfg Config) []string {
	value := strings.TrimSpace(cfg.ProductImageTranscodeAllowedPrefixes)
	if value == "" {
		fallback := strings.TrimRight(strings.TrimSpace(cfg.ProductImageBaseURL), "/")
		if fallback == "" {
			return nil
		}

		return []string{fallback}
	}

	parts := strings.Split(value, ",")
	prefixes := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimRight(strings.TrimSpace(part), "/")
		if trimmed == "" {
			continue
		}
		prefixes = append(prefixes, trimmed)
	}
	if len(prefixes) == 0 {
		return nil
	}

	return prefixes
}

// resolveImageTranscodeConfig resolves runtime values used by Falabella image-transcode HTTP handlers.
func resolveImageTranscodeConfig(cfg Config) falabellahttp.ImageTranscodeConfig {
	return falabellahttp.ImageTranscodeConfig{
		Enabled:               cfg.ProductImageTranscodeEnabled,
		AllowedSourcePrefixes: resolveImageTranscodeAllowedPrefixes(cfg),
		RequestTimeout:        resolveImageTranscodeTimeout(cfg.ProductImageTranscodeTimeoutMS),
	}
}
