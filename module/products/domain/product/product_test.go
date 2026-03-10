package product

import "testing"

// TestNormalize verifies canonicalization behavior.
func TestNormalize(t *testing.T) {
	position := -2
	variationPosition := -9
	entity := &Product{
		SKU: " SKU-1 ",
		Gallery: []GalleryItem{{
			AssetID:           " asset-1 ",
			Position:          &position,
			VariationPosition: &variationPosition,
		}},
		Datasheets: []Datasheet{{Realm: " default ", Name: " Tee "}},
	}
	entity.Normalize()

	if entity.SKU != "SKU-1" {
		t.Fatalf("entity.SKU = %q, want %q", entity.SKU, "SKU-1")
	}
	if entity.Gallery[0].AssetID != "asset-1" {
		t.Fatalf("entity.Gallery[0].AssetID = %q, want %q", entity.Gallery[0].AssetID, "asset-1")
	}
	if entity.Gallery[0].Position == nil || *entity.Gallery[0].Position != 0 {
		t.Fatalf("entity.Gallery[0].Position = %v, want 0", entity.Gallery[0].Position)
	}
	if entity.Gallery[0].VariationPosition == nil || *entity.Gallery[0].VariationPosition != 0 {
		t.Fatalf("entity.Gallery[0].VariationPosition = %v, want 0", entity.Gallery[0].VariationPosition)
	}
	if entity.Datasheets[0].Realm != "default" {
		t.Fatalf("entity.Datasheets[0].Realm = %q, want %q", entity.Datasheets[0].Realm, "default")
	}
}

// TestValidate verifies product invariants.
func TestValidate(t *testing.T) {
	if err := (Product{}).Validate(); err != ErrSKURequired {
		t.Fatalf("Validate() error = %v, want ErrSKURequired", err)
	}
	if err := (Product{SKU: "SKU", Gallery: []GalleryItem{{}}}).Validate(); err != ErrGalleryAssetIDRequired {
		t.Fatalf("Validate() error = %v, want ErrGalleryAssetIDRequired", err)
	}
	if err := (Product{SKU: "SKU", Datasheets: []Datasheet{{Realm: ""}}}).Validate(); err != ErrDatasheetRealmRequired {
		t.Fatalf("Validate() error = %v, want ErrDatasheetRealmRequired", err)
	}
	if err := (Product{SKU: "SKU", Gallery: []GalleryItem{{AssetID: "asset-1"}}, Datasheets: []Datasheet{{Realm: "default"}}}).Validate(); err != nil {
		t.Fatalf("Validate() error = %v, want nil", err)
	}
}

// TestMergeDatasheets verifies datasheet merge behavior.
func TestMergeDatasheets(t *testing.T) {
	existing := []Datasheet{{Realm: "default", Name: "Old"}}
	incoming := []Datasheet{{Realm: "default", Name: "New"}, {Realm: "b2b", Name: "Bulk"}}
	merged := MergeDatasheets(existing, incoming)

	if len(merged) != 2 {
		t.Fatalf("len(merged) = %d, want %d", len(merged), 2)
	}
	if merged[0].Name != "New" {
		t.Fatalf("merged[0].Name = %q, want %q", merged[0].Name, "New")
	}
}
