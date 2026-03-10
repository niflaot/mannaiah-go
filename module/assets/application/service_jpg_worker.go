package application

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"path"
	"strings"

	_ "golang.org/x/image/webp"
	_ "image/png"
	"mannaiah/module/assets/domain"
	"mannaiah/module/assets/port"
)

const (
	defaultJPGWorkerJPEGQuality = 90
	minJPGWorkerJPEGQuality     = 1
	maxJPGWorkerJPEGQuality     = 100
)

// RunJPGWorker converts selected tagged assets to JPG and replaces storage keys.
func (s *AssetService) RunJPGWorker(ctx context.Context, command JPGWorkerCommand) (*JPGWorkerResult, error) {
	if err := s.ensureStorage(); err != nil {
		return nil, err
	}

	tagNames := normalizeWorkerTags(command.Tags)
	if len(tagNames) == 0 {
		return nil, ErrJPGWorkerTagsRequired
	}

	assets, err := s.repository.ListByTagNames(ctx, tagNames, command.BatchSize)
	if err != nil {
		return nil, fmt.Errorf("list jpg worker assets: %w", err)
	}

	result := &JPGWorkerResult{Scanned: len(assets)}
	jpegQuality := resolveWorkerJPEGQuality(command.JPEGQuality)

	for _, entity := range assets {
		if err := ctx.Err(); err != nil {
			return result, err
		}

		converted, convertErr := s.convertAssetToJPG(ctx, entity, jpegQuality)
		if convertErr != nil {
			result.Failed++
			continue
		}
		if converted {
			result.Converted++
			continue
		}

		result.Skipped++
	}

	return result, nil
}

// convertAssetToJPG converts one asset to JPG and replaces storage/database references.
func (s *AssetService) convertAssetToJPG(ctx context.Context, entity domain.Asset, quality int) (bool, error) {
	unlock := s.locks.Lock("asset:" + strings.TrimSpace(entity.ID))
	defer unlock()

	current, err := s.repository.GetByID(ctx, entity.ID)
	if err != nil {
		return false, fmt.Errorf("load asset for jpg worker: %w", err)
	}
	if isJPGAsset(*current) {
		return false, nil
	}

	sourceBody, err := s.storage.Download(ctx, current.Key)
	if err != nil {
		return false, fmt.Errorf("download source asset object: %w", err)
	}

	decoded, _, err := image.Decode(bytes.NewReader(sourceBody))
	if err != nil {
		return false, fmt.Errorf("decode source asset image: %w", err)
	}

	buffer := bytes.Buffer{}
	if err := jpeg.Encode(&buffer, decoded, &jpeg.Options{Quality: quality}); err != nil {
		return false, fmt.Errorf("encode jpg asset image: %w", err)
	}

	newKey := replaceWithJPGExtension(current.Key)
	newOriginalName := replaceWithJPGExtension(current.OriginalName)
	if strings.TrimSpace(newOriginalName) == "" {
		newOriginalName = strings.TrimSpace(current.ID) + ".jpg"
	}
	if strings.TrimSpace(newKey) == "" {
		return false, fmt.Errorf("resolve jpg storage key: %w", domain.ErrKeyRequired)
	}

	oldState := *current
	jpgPayload := buffer.Bytes()
	if err := s.storage.Upload(ctx, port.UploadRequest{
		Key:         newKey,
		ContentType: "image/jpeg",
		Body:        jpgPayload,
	}); err != nil {
		return false, fmt.Errorf("upload jpg asset object: %w", err)
	}

	updated, err := s.repository.UpdateBinary(ctx, current.ID, port.AssetBinaryUpdate{
		Key:          newKey,
		OriginalName: newOriginalName,
		MimeType:     "image/jpeg",
		Size:         int64(len(jpgPayload)),
	})
	if err != nil {
		if newKey != oldState.Key {
			_ = s.storage.Delete(ctx, newKey)
		}
		return false, fmt.Errorf("update asset jpg metadata: %w", err)
	}

	if newKey != oldState.Key {
		if err := s.storage.Delete(ctx, oldState.Key); err != nil {
			if _, rollbackErr := s.repository.UpdateBinary(ctx, oldState.ID, port.AssetBinaryUpdate{
				Key:          oldState.Key,
				OriginalName: oldState.OriginalName,
				MimeType:     oldState.MimeType,
				Size:         oldState.Size,
			}); rollbackErr != nil {
				return false, fmt.Errorf("delete old asset object: %w (rollback metadata: %v)", err, rollbackErr)
			}
			_ = s.storage.Delete(ctx, newKey)
			return false, fmt.Errorf("delete old asset object: %w", err)
		}
	}

	if publishErr := s.publisher.Publish(ctx, buildAssetUpdatedIntegrationEvent(*updated)); publishErr != nil {
		return false, fmt.Errorf("publish asset updated event: %w", publishErr)
	}

	return true, nil
}

// normalizeWorkerTags normalizes and deduplicates tag names.
func normalizeWorkerTags(tags []string) []string {
	seen := map[string]struct{}{}
	normalized := make([]string, 0, len(tags))

	for _, raw := range tags {
		trimmed := strings.ToLower(strings.TrimSpace(raw))
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		normalized = append(normalized, trimmed)
	}

	return normalized
}

// resolveWorkerJPEGQuality normalizes jpg encoder quality values.
func resolveWorkerJPEGQuality(quality int) int {
	if quality < minJPGWorkerJPEGQuality || quality > maxJPGWorkerJPEGQuality {
		return defaultJPGWorkerJPEGQuality
	}

	return quality
}

// isJPGAsset reports whether binary metadata already points to canonical jpg payloads.
func isJPGAsset(entity domain.Asset) bool {
	return strings.EqualFold(strings.TrimSpace(entity.MimeType), "image/jpeg") &&
		strings.EqualFold(path.Ext(strings.TrimSpace(entity.Key)), ".jpg") &&
		strings.EqualFold(path.Ext(strings.TrimSpace(entity.OriginalName)), ".jpg")
}

// replaceWithJPGExtension replaces file extensions with .jpg.
func replaceWithJPGExtension(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}

	extension := path.Ext(trimmed)
	if extension == "" {
		return trimmed + ".jpg"
	}

	return strings.TrimSuffix(trimmed, extension) + ".jpg"
}
