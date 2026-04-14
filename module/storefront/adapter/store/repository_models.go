package store

import "time"

// renderableRecord defines renderable root persistence rows.
type renderableRecord struct {
	// ID defines stable renderable identifiers.
	ID string `gorm:"primaryKey;size:64"`
	// Kind defines renderable kind values.
	Kind string `gorm:"size:64;not null;index:idx_storefront_renderables_kind_draft,priority:1"`
	// MetadataJSON defines compact metadata JSON values.
	MetadataJSON string `gorm:"column:metadata_json;type:longtext;not null"`
	// ContentJSON defines compact content JSON values.
	ContentJSON string `gorm:"column:content_json;type:longtext;not null"`
	// SnapshotHash defines stable snapshot hash values.
	SnapshotHash string `gorm:"column:snapshot_hash;size:64;not null"`
	// Draft defines current draft-state values.
	Draft bool `gorm:"not null;index:idx_storefront_renderables_kind_draft,priority:2"`
	// LatestPublishedVersionID defines latest published version identifiers.
	LatestPublishedVersionID *string `gorm:"column:latest_published_version_id;size:64;default:null"`
	// LatestPublishedAt defines latest published timestamps.
	LatestPublishedAt *time.Time `gorm:"column:latest_published_at;default:null;index"`
	// CreatedAt defines creation timestamps.
	CreatedAt time.Time
	// UpdatedAt defines update timestamps.
	UpdatedAt time.Time
}

// renderableVersionRecord defines immutable published snapshot rows.
type renderableVersionRecord struct {
	// ID defines stable version identifiers.
	ID string `gorm:"primaryKey;size:64"`
	// RenderableID defines owning renderable identifiers.
	RenderableID string `gorm:"column:renderable_id;size:64;not null;index:idx_storefront_renderable_versions_renderable_published,priority:1;index:idx_storefront_renderable_versions_renderable_hash,priority:1"`
	// SourceVersionID defines source version identifiers for rollback snapshots.
	SourceVersionID *string `gorm:"column:source_version_id;size:64;default:null"`
	// MetadataJSON defines immutable snapshot metadata JSON.
	MetadataJSON string `gorm:"column:metadata_json;type:longtext;not null"`
	// ContentJSON defines immutable snapshot content JSON.
	ContentJSON string `gorm:"column:content_json;type:longtext;not null"`
	// SnapshotHash defines stable snapshot hash values.
	SnapshotHash string `gorm:"column:snapshot_hash;size:64;not null;index:idx_storefront_renderable_versions_renderable_hash,priority:2"`
	// PublishedAt defines publish timestamps.
	PublishedAt time.Time `gorm:"column:published_at;not null;index:idx_storefront_renderable_versions_renderable_published,priority:2"`
	// CreatedAt defines creation timestamps.
	CreatedAt time.Time
}

// staticPageRecord defines storefront static-page persistence rows.
type staticPageRecord struct {
	// ID defines stable static-page identifiers.
	ID string `gorm:"primaryKey;size:64"`
	// RenderableID defines bound renderable identifiers.
	RenderableID string `gorm:"column:renderable_id;size:64;not null;uniqueIndex"`
	// Title defines page title values.
	Title string `gorm:"size:255;not null;index"`
	// URL defines unique storefront URL values.
	URL string `gorm:"column:url;size:512;not null;uniqueIndex"`
	// SEOTagsJSON defines compact SEO JSON values.
	SEOTagsJSON string `gorm:"column:seo_tags_json;type:longtext;not null"`
	// ArchivedAt defines archive timestamps when present.
	ArchivedAt *time.Time `gorm:"column:archived_at;default:null;index"`
	// CreatedAt defines creation timestamps.
	CreatedAt time.Time
	// UpdatedAt defines update timestamps.
	UpdatedAt time.Time
}

// TableName defines renderable storage table names.
func (renderableRecord) TableName() string { return "storefront_renderables" }

// TableName defines renderable-version storage table names.
func (renderableVersionRecord) TableName() string { return "storefront_renderable_versions" }

// TableName defines static-page storage table names.
func (staticPageRecord) TableName() string { return "storefront_static_pages" }