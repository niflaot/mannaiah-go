package store

import (
	errorspkg "errors"
	"testing"

	"mannaiah/module/assets/domain"
)

// TestMapperEncodeDecode validates mapper serialization helpers.
func TestMapperEncodeDecode(t *testing.T) {
	tags := []domain.Tag{{Name: "hero", Color: "#ff0000"}}
	encodedTags, err := encodeTags(tags)
	if err != nil {
		t.Fatalf("encodeTags() error = %v", err)
	}
	decodedTags, err := decodeTags(encodedTags)
	if err != nil {
		t.Fatalf("decodeTags() error = %v", err)
	}
	if len(decodedTags) != 1 || decodedTags[0].Name != "hero" {
		t.Fatalf("decodedTags = %#v, want one hero tag", decodedTags)
	}

	metadata := map[string]string{"alt": "hero image"}
	encodedMetadata, err := encodeMetadata(metadata)
	if err != nil {
		t.Fatalf("encodeMetadata() error = %v", err)
	}
	decodedMetadata, err := decodeMetadata(encodedMetadata)
	if err != nil {
		t.Fatalf("decodeMetadata() error = %v", err)
	}
	if decodedMetadata["alt"] != "hero image" {
		t.Fatalf("decodedMetadata[alt] = %q, want %q", decodedMetadata["alt"], "hero image")
	}
}

// TestMapperDecodeErrorPaths validates mapper decode failure behavior.
func TestMapperDecodeErrorPaths(t *testing.T) {
	if _, err := decodeTags("[bad json"); err == nil {
		t.Fatalf("expected decodeTags json error")
	}
	if _, err := decodeMetadata("{bad json"); err == nil {
		t.Fatalf("expected decodeMetadata json error")
	}
	if _, err := encodeTags([]domain.Tag{{Name: "BAD", Color: "#ff0000"}}); !errorspkg.Is(err, domain.ErrInvalidTagName) {
		t.Fatalf("encodeTags(invalid name) error = %v, want domain.ErrInvalidTagName", err)
	}
	if _, err := encodeMetadata(map[string]string{"long": string(make([]byte, 2049))}); !errorspkg.Is(err, domain.ErrInvalidMetadata) {
		t.Fatalf("encodeMetadata(long value) error = %v, want domain.ErrInvalidMetadata", err)
	}
}

// TestToRecordMappings validates record mapping helpers.
func TestToRecordMappings(t *testing.T) {
	asset, err := toAssetRecord(domain.Asset{
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
	mappedAsset, err := toAssetDomain(asset)
	if err != nil {
		t.Fatalf("toAssetDomain() error = %v", err)
	}
	if mappedAsset.FolderID != "f-1" {
		t.Fatalf("mappedAsset.FolderID = %q, want %q", mappedAsset.FolderID, "f-1")
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
	mappedFolder, err := toFolderDomain(folderRecord)
	if err != nil {
		t.Fatalf("toFolderDomain() error = %v", err)
	}
	if mappedFolder.Slug != "catalog" {
		t.Fatalf("mappedFolder.Slug = %q, want %q", mappedFolder.Slug, "catalog")
	}
	if mappedFolder.ParentFolderID != "root" {
		t.Fatalf("mappedFolder.ParentFolderID = %q, want %q", mappedFolder.ParentFolderID, "root")
	}
}
