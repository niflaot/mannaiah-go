package store

import (
	errorspkg "errors"
	"testing"

	"mannaiah/module/assets/domain"
)

// TestTagMetadataNormalization validates normalized tag/metadata mapping behavior.
func TestTagMetadataNormalization(t *testing.T) {
	assetTags, err := toAssetTagRecords("a-1", []domain.Tag{{Name: "hero", Color: "#ff0000"}})
	if err != nil {
		t.Fatalf("toAssetTagRecords() error = %v", err)
	}
	if len(assetTags) != 1 {
		t.Fatalf("len(assetTags) = %d, want 1", len(assetTags))
	}

	folderTags, err := toFolderTagRecords("f-1", []domain.Tag{{Name: "cover", Color: "#00aa11"}})
	if err != nil {
		t.Fatalf("toFolderTagRecords() error = %v", err)
	}
	if len(folderTags) != 1 {
		t.Fatalf("len(folderTags) = %d, want 1", len(folderTags))
	}

	metadata, err := toAssetMetadataRecords("a-1", map[string]string{" alt ": " value "})
	if err != nil {
		t.Fatalf("toAssetMetadataRecords() error = %v", err)
	}
	if len(metadata) != 1 {
		t.Fatalf("len(metadata) = %d, want 1", len(metadata))
	}
	if metadata[0].Key != "alt" || metadata[0].Value != "value" {
		t.Fatalf("metadata[0] = %#v, want key/value alt/value", metadata[0])
	}
}

// TestMapperValidationErrors validates mapping validation error behavior.
func TestMapperValidationErrors(t *testing.T) {
	if _, err := toAssetTagRecords("a-1", []domain.Tag{{Name: "BAD", Color: "#ff0000"}}); !errorspkg.Is(err, domain.ErrInvalidTagName) {
		t.Fatalf("toAssetTagRecords(invalid name) error = %v, want domain.ErrInvalidTagName", err)
	}
	if _, err := toAssetMetadataRecords("a-1", map[string]string{"key": string(make([]byte, 2049))}); !errorspkg.Is(err, domain.ErrInvalidMetadata) {
		t.Fatalf("toAssetMetadataRecords(long value) error = %v, want domain.ErrInvalidMetadata", err)
	}
}

// TestToRecordMappings validates core record mapping helpers.
func TestToRecordMappings(t *testing.T) {
	assetRecord, err := toAssetRecord(domain.Asset{
		ID:           "a-1",
		Key:          "assets/a-1.png",
		Name:         "Hero",
		OriginalName: "hero.png",
		FolderID:     "f-1",
		MimeType:     "image/png",
		Size:         100,
		Tags:         []domain.Tag{{Name: "hero", Color: "#ff0000"}},
		Metadata:     map[string]string{"alt": "hero"},
	})
	if err != nil {
		t.Fatalf("toAssetRecord() error = %v", err)
	}
	assetDomain, err := toAssetDomain(assetRecord, []domain.Tag{{Name: "hero", Color: "#ff0000"}}, map[string]string{"alt": "hero"})
	if err != nil {
		t.Fatalf("toAssetDomain() error = %v", err)
	}
	if assetDomain.FolderID != "f-1" {
		t.Fatalf("assetDomain.FolderID = %q, want %q", assetDomain.FolderID, "f-1")
	}

	folderRecord, err := toFolderRecord(domain.Folder{
		ID:             "f-1",
		Name:           "Catalog",
		Slug:           "catalog",
		ParentFolderID: "root",
		Tags:           []domain.Tag{{Name: "hero", Color: "#ff0000"}},
	})
	if err != nil {
		t.Fatalf("toFolderRecord() error = %v", err)
	}
	folderDomain, err := toFolderDomain(folderRecord, []domain.Tag{{Name: "hero", Color: "#ff0000"}})
	if err != nil {
		t.Fatalf("toFolderDomain() error = %v", err)
	}
	if folderDomain.ParentFolderID != "root" {
		t.Fatalf("folderDomain.ParentFolderID = %q, want %q", folderDomain.ParentFolderID, "root")
	}
}
