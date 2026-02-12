package s3

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	awss3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"go.uber.org/zap"
	corebreaker "mannaiah/module/core/circuitbreaker"
)

var (
	// ErrUnavailable is returned when storage integration is disabled or unavailable.
	ErrUnavailable = errors.New("s3 storage is unavailable")
	// ErrInvalidKey is returned when storage keys are empty.
	ErrInvalidKey = errors.New("s3 key must not be empty")
	// ErrEmptyBody is returned when upload bodies are empty.
	ErrEmptyBody = errors.New("s3 upload body must not be empty")
)

// Client defines S3-backed storage behavior.
type Client struct {
	// api defines AWS S3 API dependencies.
	api s3API
	// bucketName defines target S3 bucket names.
	bucketName string
	// requestTimeout defines operation timeout values.
	requestTimeout time.Duration
	// logger defines structured logging dependencies.
	logger *zap.Logger
	// breaker defines optional operation circuit breaker behavior.
	breaker corebreaker.Breaker
	// availabilityErr defines disabled/unavailable integration reasons.
	availabilityErr error
}

// s3API defines required AWS S3 client behavior.
type s3API interface {
	// PutObject uploads object bytes.
	PutObject(ctx context.Context, params *awss3.PutObjectInput, optFns ...func(*awss3.Options)) (*awss3.PutObjectOutput, error)
	// DeleteObject removes objects by key.
	DeleteObject(ctx context.Context, params *awss3.DeleteObjectInput, optFns ...func(*awss3.Options)) (*awss3.DeleteObjectOutput, error)
	// HeadObject checks object existence by key.
	HeadObject(ctx context.Context, params *awss3.HeadObjectInput, optFns ...func(*awss3.Options)) (*awss3.HeadObjectOutput, error)
}

// New creates S3-backed storage dependencies.
func New(cfg Config, providedLogger *zap.Logger) *Client {
	logger := resolveLogger(providedLogger)
	if !cfg.Enabled {
		reason := errors.New("storage integration disabled")
		logger.Warn("s3 storage disabled", zap.Error(reason))
		return Disabled(reason)
	}

	resolved, validationErr := validateConfig(cfg)
	if validationErr != nil {
		logger.Warn("s3 storage configuration invalid; integration disabled", zap.Error(validationErr))
		return Disabled(validationErr)
	}

	awsCfgOptions := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(resolved.Region),
		awsconfig.WithBaseEndpoint(resolved.Endpoint),
	}
	if resolved.AccessKey != "" || resolved.SecretKey != "" {
		awsCfgOptions = append(awsCfgOptions, awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			resolved.AccessKey,
			resolved.SecretKey,
			"",
		)))
	}

	awsCfg, awsCfgErr := awsconfig.LoadDefaultConfig(context.Background(), awsCfgOptions...)
	if awsCfgErr != nil {
		logger.Warn("s3 storage configuration load failed; integration disabled", zap.Error(awsCfgErr))
		return Disabled(fmt.Errorf("load s3 aws config: %w", awsCfgErr))
	}

	api := awss3.NewFromConfig(awsCfg, func(options *awss3.Options) {
		options.UsePathStyle = resolved.ForcePathStyle
	})

	client := &Client{
		api:            api,
		bucketName:     resolved.BucketName,
		requestTimeout: time.Duration(resolved.RequestTimeoutMS) * time.Millisecond,
		logger:         logger,
	}

	if resolved.CircuitBreakerEnabled {
		breaker, breakerErr := corebreaker.NewBreaker(corebreaker.Config{
			Name:             "s3-storage",
			MaxRequests:      resolved.CircuitBreakerMaxRequests,
			IntervalMS:       resolved.CircuitBreakerIntervalMS,
			TimeoutMS:        resolved.CircuitBreakerTimeoutMS,
			FailureThreshold: resolved.CircuitBreakerFailureThreshold,
		}, logger)
		if breakerErr != nil {
			logger.Warn("s3 storage circuit breaker initialization failed; breaker disabled", zap.Error(breakerErr))
		} else {
			client.breaker = breaker
		}
	}

	return client
}

// Disabled creates a disabled storage store with a fixed availability reason.
func Disabled(reason error) *Client {
	resolvedReason := reason
	if resolvedReason == nil {
		resolvedReason = errors.New("storage integration unavailable")
	}

	return &Client{
		logger:          zap.NewNop(),
		availabilityErr: resolvedReason,
	}
}

// Upload uploads object bytes to storage.
func (c *Client) Upload(ctx context.Context, request UploadRequest) error {
	if err := c.validateAvailability(); err != nil {
		return err
	}

	key := strings.TrimSpace(request.Key)
	if key == "" {
		return ErrInvalidKey
	}
	if len(request.Body) == 0 {
		return ErrEmptyBody
	}

	err := c.execute(ctx, func(operationCtx context.Context) error {
		_, operationErr := c.api.PutObject(operationCtx, &awss3.PutObjectInput{
			Bucket:      &c.bucketName,
			Key:         &key,
			Body:        bytes.NewReader(request.Body),
			ContentType: optionalString(request.ContentType),
		})
		if operationErr != nil {
			return fmt.Errorf("put object key %q: %w", key, operationErr)
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

// Delete removes object keys from storage.
func (c *Client) Delete(ctx context.Context, key string) error {
	if err := c.validateAvailability(); err != nil {
		return err
	}

	trimmedKey := strings.TrimSpace(key)
	if trimmedKey == "" {
		return ErrInvalidKey
	}

	err := c.execute(ctx, func(operationCtx context.Context) error {
		_, operationErr := c.api.DeleteObject(operationCtx, &awss3.DeleteObjectInput{
			Bucket: &c.bucketName,
			Key:    &trimmedKey,
		})
		if operationErr != nil {
			return fmt.Errorf("delete object key %q: %w", trimmedKey, operationErr)
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

// Exists verifies whether object keys exist.
func (c *Client) Exists(ctx context.Context, key string) (bool, error) {
	if err := c.validateAvailability(); err != nil {
		return false, err
	}

	trimmedKey := strings.TrimSpace(key)
	if trimmedKey == "" {
		return false, ErrInvalidKey
	}

	var exists bool
	err := c.execute(ctx, func(operationCtx context.Context) error {
		_, operationErr := c.api.HeadObject(operationCtx, &awss3.HeadObjectInput{
			Bucket: &c.bucketName,
			Key:    &trimmedKey,
		})
		if operationErr != nil {
			if isNotFoundError(operationErr) {
				exists = false
				return nil
			}
			return fmt.Errorf("head object key %q: %w", trimmedKey, operationErr)
		}

		exists = true
		return nil
	})
	if err != nil {
		return false, err
	}

	return exists, nil
}

// AvailabilityError reports storage availability failures when disabled/unavailable.
func (c *Client) AvailabilityError() error {
	if c == nil {
		return fmt.Errorf("%w: storage client is nil", ErrUnavailable)
	}
	if c.availabilityErr == nil {
		return nil
	}

	return fmt.Errorf("%w: %v", ErrUnavailable, c.availabilityErr)
}

// execute runs operation calls with timeout and optional circuit breaker handling.
func (c *Client) execute(ctx context.Context, operation func(operationCtx context.Context) error) error {
	if c.breaker == nil {
		return c.executeWithTimeout(ctx, operation)
	}

	err := c.breaker.Execute(func() error {
		return c.executeWithTimeout(ctx, operation)
	})
	if err != nil {
		if c.breaker.IsOpenError(err) {
			return fmt.Errorf("%w: %v", ErrUnavailable, err)
		}
		return err
	}

	return nil
}

// executeWithTimeout runs operation calls under configured timeout values.
func (c *Client) executeWithTimeout(ctx context.Context, operation func(operationCtx context.Context) error) error {
	timeout := c.requestTimeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	operationCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	return operation(operationCtx)
}

// validateAvailability verifies storage availability state.
func (c *Client) validateAvailability() error {
	if c == nil {
		return fmt.Errorf("%w: storage client is nil", ErrUnavailable)
	}
	if c.availabilityErr != nil {
		return fmt.Errorf("%w: %v", ErrUnavailable, c.availabilityErr)
	}
	if c.api == nil {
		return fmt.Errorf("%w: s3 api is nil", ErrUnavailable)
	}
	if strings.TrimSpace(c.bucketName) == "" {
		return fmt.Errorf("%w: s3 bucket is empty", ErrUnavailable)
	}

	return nil
}

// validateConfig verifies required config values.
func validateConfig(cfg Config) (Config, error) {
	resolved := cfg
	resolved.Endpoint = strings.TrimSpace(resolved.Endpoint)
	resolved.Region = strings.TrimSpace(resolved.Region)
	resolved.BucketName = strings.TrimSpace(resolved.BucketName)
	resolved.AccessKey = strings.TrimSpace(resolved.AccessKey)
	resolved.SecretKey = strings.TrimSpace(resolved.SecretKey)
	if resolved.RequestTimeoutMS <= 0 {
		resolved.RequestTimeoutMS = 5000
	}

	if resolved.Endpoint == "" {
		return Config{}, errors.New("STORAGE_ENDPOINT is required when STORAGE_ENABLED=true")
	}
	if resolved.Region == "" {
		return Config{}, errors.New("STORAGE_REGION is required when STORAGE_ENABLED=true")
	}
	if resolved.BucketName == "" {
		return Config{}, errors.New("STORAGE_BUCKET_NAME is required when STORAGE_ENABLED=true")
	}

	return resolved, nil
}

// optionalString returns nil for empty values and pointer for non-empty values.
func optionalString(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}

	return &trimmed
}

// isNotFoundError reports whether head-object errors indicate missing objects.
func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	var notFound *awss3types.NotFound
	if errors.As(err, &notFound) {
		return true
	}

	value := strings.ToLower(strings.TrimSpace(err.Error()))
	return strings.Contains(value, "not found") || strings.Contains(value, "status code: 404") || strings.Contains(value, "nosuchkey")
}

// resolveLogger resolves nil loggers to no-op defaults.
func resolveLogger(providedLogger *zap.Logger) *zap.Logger {
	if providedLogger != nil {
		return providedLogger
	}

	return zap.NewNop()
}
