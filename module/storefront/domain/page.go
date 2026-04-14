package domain

import (
	"encoding/json"
	"errors"
	"strings"
	"time"
)

var (
	// ErrStaticPageRenderableIDRequired is returned when page bindings are empty.
	ErrStaticPageRenderableIDRequired = errors.New("static page renderable id is required")
	// ErrStaticPageTitleRequired is returned when page titles are empty.
	ErrStaticPageTitleRequired = errors.New("static page title is required")
	// ErrStaticPageURLRequired is returned when page URLs are empty.
	ErrStaticPageURLRequired = errors.New("static page url is required")
	// ErrStaticPageSEOTagsInvalid is returned when SEO-tags JSON is invalid.
	ErrStaticPageSEOTagsInvalid = errors.New("static page seo tags must contain valid json")
)

// StaticPage defines one static storefront page bound to a renderable.
type StaticPage struct {
	// ID defines stable page identifiers.
	ID string `json:"id"`
	// RenderableID defines the bound renderable identifier.
	RenderableID string `json:"renderableId"`
	// Title defines display title values.
	Title string `json:"title"`
	// URL defines the resolved storefront URL path.
	URL string `json:"url"`
	// SEOTags defines frontend-provided SEO JSON.
	SEOTags json.RawMessage `json:"seoTags"`
	// CreatedAt defines creation timestamps.
	CreatedAt time.Time `json:"createdAt"`
	// UpdatedAt defines last mutation timestamps.
	UpdatedAt time.Time `json:"updatedAt"`
}

// Normalize trims canonical page string fields.
func (p *StaticPage) Normalize() {
	if p == nil {
		return
	}

	p.ID = strings.TrimSpace(p.ID)
	p.RenderableID = strings.TrimSpace(p.RenderableID)
	p.Title = strings.TrimSpace(p.Title)
	p.URL = strings.TrimSpace(p.URL)
}

// Validate verifies static-page invariants.
func (p StaticPage) Validate() error {
	if strings.TrimSpace(p.RenderableID) == "" {
		return ErrStaticPageRenderableIDRequired
	}
	if strings.TrimSpace(p.Title) == "" {
		return ErrStaticPageTitleRequired
	}
	if strings.TrimSpace(p.URL) == "" {
		return ErrStaticPageURLRequired
	}
	if !json.Valid(p.SEOTags) {
		return ErrStaticPageSEOTagsInvalid
	}

	return nil
}
