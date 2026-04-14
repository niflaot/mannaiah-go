package store

import (
	"encoding/json"

	"mannaiah/module/storefront/domain"
)

// toRenderableRecord maps renderable entities into persistence rows.
func toRenderableRecord(renderable domain.Renderable) renderableRecord {
	return renderableRecord{
		ID:                       renderable.ID,
		Kind:                     renderable.Kind,
		MetadataJSON:             string(renderable.Metadata),
		ContentJSON:              string(renderable.Content),
		SnapshotHash:             renderable.SnapshotHash,
		Draft:                    renderable.Draft,
		LatestPublishedVersionID: stringPtrOrEmpty(renderable.LatestPublishedVersionID),
		LatestPublishedAt:        renderable.LatestPublishedAt,
		CreatedAt:                renderable.CreatedAt,
		UpdatedAt:                renderable.UpdatedAt,
	}
}

// toRenderableEntity maps renderable persistence rows into domain entities.
func toRenderableEntity(record renderableRecord) domain.Renderable {
	entity := domain.Renderable{
		ID:           record.ID,
		Kind:         record.Kind,
		Metadata:     json.RawMessage(record.MetadataJSON),
		Content:      json.RawMessage(record.ContentJSON),
		Draft:        record.Draft,
		SnapshotHash: record.SnapshotHash,
		CreatedAt:    record.CreatedAt,
		UpdatedAt:    record.UpdatedAt,
	}
	if record.LatestPublishedVersionID != nil {
		entity.LatestPublishedVersionID = *record.LatestPublishedVersionID
	}
	entity.LatestPublishedAt = record.LatestPublishedAt
	entity.Normalize()
	return entity
}

// toRenderableVersionRecord maps renderable-version entities into persistence rows.
func toRenderableVersionRecord(version domain.RenderableVersion) renderableVersionRecord {
	return renderableVersionRecord{
		ID:              version.ID,
		RenderableID:    version.RenderableID,
		SourceVersionID: stringPtrOrEmpty(version.SourceVersionID),
		MetadataJSON:    string(version.Metadata),
		ContentJSON:     string(version.Content),
		SnapshotHash:    version.SnapshotHash,
		PublishedAt:     version.PublishedAt,
		CreatedAt:       version.PublishedAt,
	}
}

// toRenderableVersionEntity maps renderable-version persistence rows into domain entities.
func toRenderableVersionEntity(record renderableVersionRecord) domain.RenderableVersion {
	entity := domain.RenderableVersion{
		ID:           record.ID,
		RenderableID: record.RenderableID,
		Metadata:     json.RawMessage(record.MetadataJSON),
		Content:      json.RawMessage(record.ContentJSON),
		SnapshotHash: record.SnapshotHash,
		PublishedAt:  record.PublishedAt,
	}
	if record.SourceVersionID != nil {
		entity.SourceVersionID = *record.SourceVersionID
	}
	entity.Normalize()
	return entity
}

// toStaticPageRecord maps static-page entities into persistence rows.
func toStaticPageRecord(page domain.StaticPage) staticPageRecord {
	return staticPageRecord{
		ID:           page.ID,
		RenderableID: page.RenderableID,
		Title:        page.Title,
		URL:          page.URL,
		SEOTagsJSON:  string(page.SEOTags),
		ArchivedAt:   page.ArchivedAt,
		CreatedAt:    page.CreatedAt,
		UpdatedAt:    page.UpdatedAt,
	}
}

// toStaticPageEntity maps static-page persistence rows into domain entities.
func toStaticPageEntity(record staticPageRecord) domain.StaticPage {
	entity := domain.StaticPage{
		ID:           record.ID,
		RenderableID: record.RenderableID,
		Title:        record.Title,
		URL:          record.URL,
		SEOTags:      json.RawMessage(record.SEOTagsJSON),
		ArchivedAt:   record.ArchivedAt,
		CreatedAt:    record.CreatedAt,
		UpdatedAt:    record.UpdatedAt,
	}
	entity.Normalize()
	return entity
}

// stringPtrOrEmpty converts string values into nullable persistence pointers.
func stringPtrOrEmpty(value string) *string {
	if value == "" {
		return nil
	}
	resolved := value
	return &resolved
}
