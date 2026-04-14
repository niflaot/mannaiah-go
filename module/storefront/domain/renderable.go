package domain

import (
	"encoding/json"
	"errors"
	"strings"
	"time"
)

var (
	// ErrRenderableKindRequired is returned when renderable kinds are empty.
	ErrRenderableKindRequired = errors.New("renderable kind is required")
	// ErrRenderableMetadataInvalid is returned when metadata JSON is invalid.
	ErrRenderableMetadataInvalid = errors.New("renderable metadata must contain valid json")
	// ErrRenderableContentInvalid is returned when content JSON is invalid.
	ErrRenderableContentInvalid = errors.New("renderable content must contain valid json")
)

const (
	// StaticPageRenderableKind defines the renderable kind used by storefront static pages.
	StaticPageRenderableKind = "static_page"
)

// Renderable defines one reusable storefront content unit.
type Renderable struct {
	// ID defines stable renderable identifiers.
	ID string `json:"id"`
	// Kind defines the renderable child/resource family.
	Kind string `json:"kind"`
	// Metadata defines renderable metadata JSON.
	Metadata json.RawMessage `json:"metadata"`
	// Content defines renderable editor JSON.
	Content json.RawMessage `json:"content"`
	// Draft reports whether the current root snapshot is still unpublished.
	Draft bool `json:"draft"`
	// SnapshotHash defines a compact snapshot checksum for current draft values.
	SnapshotHash string `json:"snapshotHash"`
	// LatestPublishedVersionID defines the current published snapshot identifier when present.
	LatestPublishedVersionID string `json:"latestPublishedVersionId,omitempty"`
	// LatestPublishedAt defines the latest published snapshot timestamp when present.
	LatestPublishedAt *time.Time `json:"latestPublishedAt,omitempty"`
	// CreatedAt defines creation timestamps.
	CreatedAt time.Time `json:"createdAt"`
	// UpdatedAt defines last mutation timestamps.
	UpdatedAt time.Time `json:"updatedAt"`
}

// RenderableVersion defines one immutable published renderable snapshot.
type RenderableVersion struct {
	// ID defines stable version identifiers.
	ID string `json:"id"`
	// RenderableID defines the owning renderable identifier.
	RenderableID string `json:"renderableId"`
	// SourceVersionID defines the source version used by rollback snapshots when present.
	SourceVersionID string `json:"sourceVersionId,omitempty"`
	// Metadata defines immutable snapshot metadata JSON.
	Metadata json.RawMessage `json:"metadata"`
	// Content defines immutable snapshot content JSON.
	Content json.RawMessage `json:"content"`
	// SnapshotHash defines the compact snapshot checksum.
	SnapshotHash string `json:"snapshotHash"`
	// PublishedAt defines publish timestamps.
	PublishedAt time.Time `json:"publishedAt"`
}

// Normalize trims canonical renderable string fields.
func (r *Renderable) Normalize() {
	if r == nil {
		return
	}

	r.ID = strings.TrimSpace(r.ID)
	r.Kind = strings.ToLower(strings.TrimSpace(r.Kind))
	r.SnapshotHash = strings.TrimSpace(r.SnapshotHash)
	r.LatestPublishedVersionID = strings.TrimSpace(r.LatestPublishedVersionID)
}

// Validate verifies renderable invariants.
func (r Renderable) Validate() error {
	if strings.TrimSpace(r.Kind) == "" {
		return ErrRenderableKindRequired
	}
	if !json.Valid(r.Metadata) {
		return ErrRenderableMetadataInvalid
	}
	if !json.Valid(r.Content) {
		return ErrRenderableContentInvalid
	}

	return nil
}

// Normalize trims canonical version string fields.
func (v *RenderableVersion) Normalize() {
	if v == nil {
		return
	}

	v.ID = strings.TrimSpace(v.ID)
	v.RenderableID = strings.TrimSpace(v.RenderableID)
	v.SourceVersionID = strings.TrimSpace(v.SourceVersionID)
	v.SnapshotHash = strings.TrimSpace(v.SnapshotHash)
}

// Validate verifies version invariants.
func (v RenderableVersion) Validate() error {
	if strings.TrimSpace(v.RenderableID) == "" {
		return ErrRenderableKindRequired
	}
	if !json.Valid(v.Metadata) {
		return ErrRenderableMetadataInvalid
	}
	if !json.Valid(v.Content) {
		return ErrRenderableContentInvalid
	}

	return nil
}
