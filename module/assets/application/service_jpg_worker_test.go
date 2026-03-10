package application

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"testing"

	"mannaiah/module/assets/domain"
	"mannaiah/module/assets/port"
)

// TestRunJPGWorkerRequiresTags verifies worker-tag validation behavior.
func TestRunJPGWorkerRequiresTags(t *testing.T) {
	service, err := NewService(newRepositoryMock(), newStorageMock())
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	if _, runErr := service.RunJPGWorker(context.Background(), JPGWorkerCommand{}); runErr != ErrJPGWorkerTagsRequired {
		t.Fatalf("RunJPGWorker() error = %v, want %v", runErr, ErrJPGWorkerTagsRequired)
	}
}

// TestRunJPGWorkerConvertsAndReplaces verifies worker conversion and replacement behavior.
func TestRunJPGWorkerConvertsAndReplaces(t *testing.T) {
	source := domain.Asset{
		ID:           "a-1",
		Key:          "assets/a-1-front.webp",
		Name:         "Asset One",
		OriginalName: "front.webp",
		MimeType:     "image/webp",
		Size:         10,
		Tags:         []domain.Tag{{Name: "marketplaces", Color: "#000000"}},
	}

	var downloadedKey string
	var uploadedKey string
	var uploadedType string
	var deletedKey string
	var publishedTopic string
	var updatedBinary port.AssetBinaryUpdate

	service, err := NewService(
		newRepositoryMockWith(repositoryMock{
			listByTagNamesFn: func(ctx context.Context, tagNames []string, limit int) ([]domain.Asset, error) {
				return []domain.Asset{source}, nil
			},
			getByIDFn: func(ctx context.Context, id string) (*domain.Asset, error) {
				return &source, nil
			},
			updateBinaryFn: func(ctx context.Context, id string, update port.AssetBinaryUpdate) (*domain.Asset, error) {
				updatedBinary = update
				updated := source
				updated.Key = update.Key
				updated.OriginalName = update.OriginalName
				updated.MimeType = update.MimeType
				updated.Size = update.Size
				return &updated, nil
			},
		}),
		storageMock{
			uploadFn: func(ctx context.Context, request port.UploadRequest) error {
				uploadedKey = request.Key
				uploadedType = request.ContentType
				return nil
			},
			downloadFn: func(ctx context.Context, key string) ([]byte, error) {
				downloadedKey = key
				return samplePNG(t), nil
			},
			deleteFn: func(ctx context.Context, key string) error {
				deletedKey = key
				return nil
			},
			existsFn: func(ctx context.Context, key string) (bool, error) { return true, nil },
		},
		publisherMock{publishFn: func(ctx context.Context, event port.IntegrationEvent) error {
			publishedTopic = event.Topic
			return nil
		}},
	)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	result, runErr := service.RunJPGWorker(context.Background(), JPGWorkerCommand{
		Tags:        []string{"marketplaces"},
		BatchSize:   5,
		JPEGQuality: 90,
	})
	if runErr != nil {
		t.Fatalf("RunJPGWorker() error = %v", runErr)
	}
	if result.Converted != 1 || result.Skipped != 0 || result.Failed != 0 {
		t.Fatalf("result = %#v, want converted=1 skipped=0 failed=0", result)
	}
	if downloadedKey != source.Key {
		t.Fatalf("downloadedKey = %q, want %q", downloadedKey, source.Key)
	}
	if uploadedKey != "assets/a-1-front.jpg" {
		t.Fatalf("uploadedKey = %q, want %q", uploadedKey, "assets/a-1-front.jpg")
	}
	if uploadedType != "image/jpeg" {
		t.Fatalf("uploadedType = %q, want %q", uploadedType, "image/jpeg")
	}
	if deletedKey != source.Key {
		t.Fatalf("deletedKey = %q, want %q", deletedKey, source.Key)
	}
	if updatedBinary.Key != "assets/a-1-front.jpg" {
		t.Fatalf("updatedBinary.Key = %q, want %q", updatedBinary.Key, "assets/a-1-front.jpg")
	}
	if updatedBinary.OriginalName != "front.jpg" {
		t.Fatalf("updatedBinary.OriginalName = %q, want %q", updatedBinary.OriginalName, "front.jpg")
	}
	if updatedBinary.MimeType != "image/jpeg" {
		t.Fatalf("updatedBinary.MimeType = %q, want %q", updatedBinary.MimeType, "image/jpeg")
	}
	if updatedBinary.Size <= 0 {
		t.Fatalf("updatedBinary.Size = %d, want > 0", updatedBinary.Size)
	}
	if publishedTopic != TopicAssetUpdated {
		t.Fatalf("publishedTopic = %q, want %q", publishedTopic, TopicAssetUpdated)
	}
}

// TestRunJPGWorkerSkipsCanonicalJPG verifies skip behavior for already-converted assets.
func TestRunJPGWorkerSkipsCanonicalJPG(t *testing.T) {
	source := domain.Asset{
		ID:           "a-1",
		Key:          "assets/a-1-front.jpg",
		Name:         "Asset One",
		OriginalName: "front.jpg",
		MimeType:     "image/jpeg",
		Size:         10,
	}

	downloaded := false
	service, err := NewService(
		newRepositoryMockWith(repositoryMock{
			listByTagNamesFn: func(ctx context.Context, tagNames []string, limit int) ([]domain.Asset, error) {
				return []domain.Asset{source}, nil
			},
			getByIDFn: func(ctx context.Context, id string) (*domain.Asset, error) {
				return &source, nil
			},
		}),
		storageMock{
			uploadFn: func(ctx context.Context, request port.UploadRequest) error { return nil },
			downloadFn: func(ctx context.Context, key string) ([]byte, error) {
				downloaded = true
				return nil, nil
			},
			deleteFn: func(ctx context.Context, key string) error { return nil },
			existsFn: func(ctx context.Context, key string) (bool, error) { return true, nil },
		},
	)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	result, runErr := service.RunJPGWorker(context.Background(), JPGWorkerCommand{
		Tags: []string{"marketplaces"},
	})
	if runErr != nil {
		t.Fatalf("RunJPGWorker() error = %v", runErr)
	}
	if result.Skipped != 1 || result.Converted != 0 {
		t.Fatalf("result = %#v, want skipped=1 converted=0", result)
	}
	if downloaded {
		t.Fatalf("download should not be called for canonical jpg assets")
	}
}

// samplePNG builds a small PNG payload for conversion tests.
func samplePNG(t *testing.T) []byte {
	t.Helper()

	imagePayload := image.NewRGBA(image.Rect(0, 0, 2, 2))
	imagePayload.Set(0, 0, color.RGBA{R: 255, G: 0, B: 0, A: 255})
	imagePayload.Set(1, 0, color.RGBA{R: 0, G: 255, B: 0, A: 255})
	imagePayload.Set(0, 1, color.RGBA{R: 0, G: 0, B: 255, A: 255})
	imagePayload.Set(1, 1, color.RGBA{R: 255, G: 255, B: 255, A: 255})

	buffer := bytes.Buffer{}
	if err := png.Encode(&buffer, imagePayload); err != nil {
		t.Fatalf("png.Encode() error = %v", err)
	}

	return buffer.Bytes()
}
