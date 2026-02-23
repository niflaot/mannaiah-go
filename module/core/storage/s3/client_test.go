package s3

import (
	"context"
	errorspkg "errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	awss3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	corebreaker "mannaiah/module/core/circuitbreaker"
	coretelemetry "mannaiah/module/core/telemetry"
)

// apiMock defines S3 API behavior for tests.
type apiMock struct {
	// putFn defines put-object behavior.
	putFn func(ctx context.Context, params *awss3.PutObjectInput, optFns ...func(*awss3.Options)) (*awss3.PutObjectOutput, error)
	// deleteFn defines delete-object behavior.
	deleteFn func(ctx context.Context, params *awss3.DeleteObjectInput, optFns ...func(*awss3.Options)) (*awss3.DeleteObjectOutput, error)
	// headFn defines head-object behavior.
	headFn func(ctx context.Context, params *awss3.HeadObjectInput, optFns ...func(*awss3.Options)) (*awss3.HeadObjectOutput, error)
}

// PutObject executes configured put-object behavior.
func (m apiMock) PutObject(ctx context.Context, params *awss3.PutObjectInput, optFns ...func(*awss3.Options)) (*awss3.PutObjectOutput, error) {
	return m.putFn(ctx, params, optFns...)
}

// DeleteObject executes configured delete-object behavior.
func (m apiMock) DeleteObject(ctx context.Context, params *awss3.DeleteObjectInput, optFns ...func(*awss3.Options)) (*awss3.DeleteObjectOutput, error) {
	return m.deleteFn(ctx, params, optFns...)
}

// HeadObject executes configured head-object behavior.
func (m apiMock) HeadObject(ctx context.Context, params *awss3.HeadObjectInput, optFns ...func(*awss3.Options)) (*awss3.HeadObjectOutput, error) {
	return m.headFn(ctx, params, optFns...)
}

// breakerMock defines circuit breaker behavior for tests.
type breakerMock struct {
	// executeFn defines execute behavior.
	executeFn func(operation func() error) error
	// isOpenErrorFn defines open-error classification behavior.
	isOpenErrorFn func(err error) bool
}

// Execute executes configured breaker behavior.
func (m breakerMock) Execute(operation func() error) error {
	return m.executeFn(operation)
}

// State returns closed-state values for tests.
func (m breakerMock) State() corebreaker.State {
	return corebreaker.StateClosed
}

// IsOpenError executes configured open-error classification behavior.
func (m breakerMock) IsOpenError(err error) bool {
	return m.isOpenErrorFn(err)
}

// TestNewDisabled verifies disabled new-client behavior.
func TestNewDisabled(t *testing.T) {
	client := New(Config{Enabled: false}, nil)
	if client == nil {
		t.Fatalf("expected disabled client")
	}
	if client.AvailabilityError() == nil {
		t.Fatalf("expected availability error")
	}
}

// TestNewInvalidConfig verifies invalid config fallback behavior.
func TestNewInvalidConfig(t *testing.T) {
	client := New(Config{Enabled: true}, nil)
	if client == nil {
		t.Fatalf("expected disabled client")
	}
	if client.AvailabilityError() == nil {
		t.Fatalf("expected availability error")
	}
}

// TestUploadSuccess verifies upload behavior.
func TestUploadSuccess(t *testing.T) {
	var gotKey string
	client := &Client{
		api: apiMock{
			putFn: func(ctx context.Context, params *awss3.PutObjectInput, optFns ...func(*awss3.Options)) (*awss3.PutObjectOutput, error) {
				gotKey = *params.Key
				return &awss3.PutObjectOutput{}, nil
			},
			deleteFn: func(ctx context.Context, params *awss3.DeleteObjectInput, optFns ...func(*awss3.Options)) (*awss3.DeleteObjectOutput, error) {
				return &awss3.DeleteObjectOutput{}, nil
			},
			headFn: func(ctx context.Context, params *awss3.HeadObjectInput, optFns ...func(*awss3.Options)) (*awss3.HeadObjectOutput, error) {
				return &awss3.HeadObjectOutput{}, nil
			},
		},
		bucketName:     "bucket",
		requestTimeout: 100 * time.Millisecond,
	}

	err := client.Upload(context.Background(), UploadRequest{Key: "assets/a.png", ContentType: "image/png", Body: []byte("data")})
	if err != nil {
		t.Fatalf("Upload() error = %v", err)
	}
	if gotKey != "assets/a.png" {
		t.Fatalf("got key = %q, want %q", gotKey, "assets/a.png")
	}
}

// TestUploadValidation verifies upload validation behavior.
func TestUploadValidation(t *testing.T) {
	client := &Client{api: apiMock{}, bucketName: "bucket", requestTimeout: 10 * time.Millisecond}

	if err := client.Upload(context.Background(), UploadRequest{}); !errorspkg.Is(err, ErrInvalidKey) {
		t.Fatalf("Upload(empty key) error = %v, want ErrInvalidKey", err)
	}
	if err := client.Upload(context.Background(), UploadRequest{Key: "key", Body: nil}); !errorspkg.Is(err, ErrEmptyBody) {
		t.Fatalf("Upload(empty body) error = %v, want ErrEmptyBody", err)
	}
}

// TestDeleteAndExists verifies delete and exists behavior.
func TestDeleteAndExists(t *testing.T) {
	client := &Client{
		api: apiMock{
			putFn: func(ctx context.Context, params *awss3.PutObjectInput, optFns ...func(*awss3.Options)) (*awss3.PutObjectOutput, error) {
				return &awss3.PutObjectOutput{}, nil
			},
			deleteFn: func(ctx context.Context, params *awss3.DeleteObjectInput, optFns ...func(*awss3.Options)) (*awss3.DeleteObjectOutput, error) {
				return &awss3.DeleteObjectOutput{}, nil
			},
			headFn: func(ctx context.Context, params *awss3.HeadObjectInput, optFns ...func(*awss3.Options)) (*awss3.HeadObjectOutput, error) {
				if *params.Key == "missing" {
					return nil, &awss3types.NotFound{}
				}
				return &awss3.HeadObjectOutput{}, nil
			},
		},
		bucketName:     "bucket",
		requestTimeout: 100 * time.Millisecond,
	}

	if err := client.Delete(context.Background(), "assets/a.png"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	exists, err := client.Exists(context.Background(), "assets/a.png")
	if err != nil {
		t.Fatalf("Exists(found) error = %v", err)
	}
	if !exists {
		t.Fatalf("Exists(found) = false, want true")
	}

	exists, err = client.Exists(context.Background(), "missing")
	if err != nil {
		t.Fatalf("Exists(missing) error = %v", err)
	}
	if exists {
		t.Fatalf("Exists(missing) = true, want false")
	}
}

// TestDeleteValidation verifies delete validation behavior.
func TestDeleteValidation(t *testing.T) {
	client := &Client{api: apiMock{}, bucketName: "bucket", requestTimeout: 10 * time.Millisecond}
	if err := client.Delete(context.Background(), ""); !errorspkg.Is(err, ErrInvalidKey) {
		t.Fatalf("Delete(empty key) error = %v, want ErrInvalidKey", err)
	}
}

// TestBreakerOpenError verifies breaker open-state mapping behavior.
func TestBreakerOpenError(t *testing.T) {
	openErr := errorspkg.New("open")
	client := &Client{
		api:            apiMock{},
		bucketName:     "bucket",
		requestTimeout: 100 * time.Millisecond,
		breaker: breakerMock{
			executeFn: func(operation func() error) error { return openErr },
			isOpenErrorFn: func(err error) bool {
				return errorspkg.Is(err, openErr)
			},
		},
	}

	err := client.Upload(context.Background(), UploadRequest{Key: "k", Body: []byte("v")})
	if !errorspkg.Is(err, ErrUnavailable) {
		t.Fatalf("Upload() error = %v, want ErrUnavailable", err)
	}
}

// TestAvailabilityAndHelpers verifies helper behavior.
func TestAvailabilityAndHelpers(t *testing.T) {
	disabled := Disabled(errorspkg.New("disabled"))
	if disabled.AvailabilityError() == nil {
		t.Fatalf("expected disabled availability error")
	}

	if optionalString("") != nil {
		t.Fatalf("optionalString(\"\") should be nil")
	}
	if optionalString(" value ") == nil {
		t.Fatalf("optionalString(non-empty) should return pointer")
	}

	if !isNotFoundError(errorspkg.New("status code: 404")) {
		t.Fatalf("expected 404 string to be not found")
	}
	if isNotFoundError(nil) {
		t.Fatalf("nil error should not be not found")
	}

	if resolveLogger(nil) == nil {
		t.Fatalf("resolveLogger(nil) should return logger")
	}
}

// TestValidateConfig verifies config validation behavior.
func TestValidateConfig(t *testing.T) {
	if _, err := validateConfig(Config{}); err == nil {
		t.Fatalf("expected missing endpoint error")
	}

	resolved, err := validateConfig(Config{Endpoint: "http://localhost:9000", Region: "us-east-1", BucketName: "bucket"})
	if err != nil {
		t.Fatalf("validateConfig() error = %v", err)
	}
	if resolved.RequestTimeoutMS <= 0 {
		t.Fatalf("expected positive request timeout")
	}
}

// TestS3OperationsEmitDependencyMetrics verifies S3 operations emit telemetry dependency metrics.
func TestS3OperationsEmitDependencyMetrics(t *testing.T) {
	provider, err := coretelemetry.Init(context.Background(), coretelemetry.Config{
		Enabled:        true,
		MetricsEnabled: true,
		TracesEnabled:  false,
	}, nil)
	if err != nil {
		t.Fatalf("coretelemetry.Init() error = %v", err)
	}
	defer func() {
		_ = provider.Shutdown(context.Background())
		coretelemetry.SetActive(nil)
	}()

	client := &Client{
		api: apiMock{
			putFn: func(ctx context.Context, params *awss3.PutObjectInput, optFns ...func(*awss3.Options)) (*awss3.PutObjectOutput, error) {
				return &awss3.PutObjectOutput{}, nil
			},
			deleteFn: func(ctx context.Context, params *awss3.DeleteObjectInput, optFns ...func(*awss3.Options)) (*awss3.DeleteObjectOutput, error) {
				return &awss3.DeleteObjectOutput{}, nil
			},
			headFn: func(ctx context.Context, params *awss3.HeadObjectInput, optFns ...func(*awss3.Options)) (*awss3.HeadObjectOutput, error) {
				return &awss3.HeadObjectOutput{}, nil
			},
		},
		bucketName:     "bucket",
		requestTimeout: 100 * time.Millisecond,
	}

	if err := client.Upload(context.Background(), UploadRequest{Key: "assets/telemetry.png", Body: []byte("data")}); err != nil {
		t.Fatalf("Upload() error = %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	recorder := httptest.NewRecorder()
	provider.MetricsHandler().ServeHTTP(recorder, request)

	body := recorder.Body.String()
	if !strings.Contains(body, `dependency="s3"`) {
		t.Fatalf("expected s3 dependency labels in metrics output")
	}
}
