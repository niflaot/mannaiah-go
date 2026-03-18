package service

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"mannaiah/module/falabella/port"
)

var (
	// ErrSKURequired is returned when product SKU values are missing.
	ErrSKURequired = errors.New("product sku is required")
	// ErrNameRequired is returned when falabella product names are missing.
	ErrNameRequired = errors.New("falabella product name is required")
	// ErrDescriptionRequired is returned when falabella product descriptions are missing.
	ErrDescriptionRequired = errors.New("falabella product description is required")
	// ErrVariantSKURequired is returned when variant SKU values are missing.
	ErrVariantSKURequired = errors.New("product variant sku is required")
)

const (
	// defaultFalabellaBrand defines fallback Falabella brand values.
	defaultFalabellaBrand = "GENERIC"
)

// mapProduct maps catalog products into Falabella sync payload values.
func mapProduct(product port.CatalogProduct, cfg Config) (port.SyncProductRequest, string, error) {
	trimmedSKU := strings.TrimSpace(product.SKU)
	if trimmedSKU == "" {
		return port.SyncProductRequest{}, "", ErrSKURequired
	}

	datasheet, ok := findDatasheetByRealm(product.Datasheets, cfg.Realm)
	if !ok {
		return port.SyncProductRequest{}, "missing_falabella_realm", nil
	}

	attributes := toStringMap(datasheet.Attributes)
	normalizeFalabellaAttributeKeys(attributes)
	brand := firstNonEmpty(attributes["Brand"], attributes["brand"])
	model := firstNonEmpty(attributes["Model"], attributes["model"])
	delete(attributes, "Brand")
	delete(attributes, "brand")
	delete(attributes, "Model")
	delete(attributes, "model")
	name := strings.TrimSpace(datasheet.Name)
	if strings.TrimSpace(name) == "" {
		return port.SyncProductRequest{}, "", ErrNameRequired
	}
	description := strings.TrimSpace(datasheet.Description)
	if description == "" {
		return port.SyncProductRequest{}, "", ErrDescriptionRequired
	}

	request := port.SyncProductRequest{
		SKU:             trimmedSKU,
		Name:            name,
		Brand:           firstNonEmpty(brand, defaultFalabellaBrand),
		Model:           strings.TrimSpace(model),
		Description:     description,
		PrimaryCategory: strings.TrimSpace(cfg.CategoryID),
		TaxClass:        strings.TrimSpace(attributes["TaxClass"]),
		Price:           strings.TrimSpace(attributes["PriceFalabella"]),
		SalePrice:       strings.TrimSpace(attributes["SalePriceFalabella"]),
		SaleStartDate:   strings.TrimSpace(attributes["SaleStartDateFalabella"]),
		SaleEndDate:     strings.TrimSpace(attributes["SaleEndDateFalabella"]),
		OperatorCode:    firstNonEmpty(strings.TrimSpace(cfg.OperatorCode), "FACO"),
		Attributes:      attributes,
	}
	applyRequestBusinessUnitFields(&request, attributes)

	if request.Attributes == nil {
		request.Attributes = map[string]string{}
	}
	if request.Brand == "" {
		request.Brand = defaultFalabellaBrand
	}

	return request, "", nil
}

// mapVariantProduct maps base product values and variant dimensions into Falabella sync payload values.
func mapVariantProduct(base port.SyncProductRequest, variant port.CatalogVariant, knownVariantSKUs map[string]struct{}) (port.SyncProductRequest, error) {
	parentSKU := strings.TrimSpace(base.SKU)
	variantSKU := strings.TrimSpace(variant.SKU)
	if variantSKU == "" {
		return port.SyncProductRequest{}, ErrVariantSKURequired
	}

	mapped := base
	mapped.SKU = variantSKU
	mapped.ParentSKU = parentSKU
	mapped.Variation = ""
	mapped.Attributes = copyAttributes(base.Attributes)
	applyVariantAttributes(mapped.Attributes, variant)
	applyVariantScopedAttributes(mapped.Attributes, base.Attributes, variantSKU, knownVariantSKUs)
	applyRequestBusinessUnitFields(&mapped, mapped.Attributes)

	return mapped, nil
}

// applyRequestBusinessUnitFields resolves product sync request business-unit values from mapped attributes.
func applyRequestBusinessUnitFields(request *port.SyncProductRequest, attributes map[string]string) {
	if request == nil {
		return
	}

	request.TaxClass = firstNonEmpty(
		request.TaxClass,
		strings.TrimSpace(attributes["TaxClass"]),
	)
	request.Price = firstNonEmpty(
		request.Price,
		strings.TrimSpace(attributes["PriceFalabella"]),
		strings.TrimSpace(attributes["Price"]),
	)
	request.SalePrice = firstNonEmpty(
		request.SalePrice,
		strings.TrimSpace(attributes["SalePriceFalabella"]),
		strings.TrimSpace(attributes["SpecialPrice"]),
	)
	request.SaleStartDate = firstNonEmpty(
		request.SaleStartDate,
		strings.TrimSpace(attributes["SaleStartDateFalabella"]),
		strings.TrimSpace(attributes["SpecialFromDate"]),
	)
	request.SaleEndDate = firstNonEmpty(
		request.SaleEndDate,
		strings.TrimSpace(attributes["SaleEndDateFalabella"]),
		strings.TrimSpace(attributes["SpecialToDate"]),
	)
}

// findDatasheetByRealm resolves datasheets for configured realm values.
func findDatasheetByRealm(datasheets []port.CatalogDatasheet, realm string) (port.CatalogDatasheet, bool) {
	trimmedRealm := strings.TrimSpace(realm)
	for _, datasheet := range datasheets {
		if strings.EqualFold(strings.TrimSpace(datasheet.Realm), trimmedRealm) {
			return datasheet, true
		}
	}

	return port.CatalogDatasheet{}, false
}

// toStringMap converts generic attribute maps into string-key/string-value maps.
func toStringMap(attributes map[string]any) map[string]string {
	if len(attributes) == 0 {
		return map[string]string{}
	}

	mapped := make(map[string]string, len(attributes))
	for key, value := range attributes {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" || value == nil {
			continue
		}
		mapped[trimmedKey] = strings.TrimSpace(fmt.Sprint(value))
	}

	return mapped
}

// copyAttributes copies string map values.
func copyAttributes(attributes map[string]string) map[string]string {
	if len(attributes) == 0 {
		return map[string]string{}
	}

	copied := make(map[string]string, len(attributes))
	for key, value := range attributes {
		copied[key] = value
	}

	return copied
}

// applyVariantAttributes maps variant variation values into Falabella attributes.
func applyVariantAttributes(attributes map[string]string, variant port.CatalogVariant) {
	if attributes == nil {
		return
	}

	if len(variant.Variations) == 0 {
		return
	}

	for _, item := range variant.Variations {
		key := variationAttributeKey(item)
		value := strings.TrimSpace(item.Value)
		if key == "" || value == "" {
			continue
		}

		attributes[key] = value
		if strings.EqualFold(strings.TrimSpace(item.Definition), "COLOR") {
			if strings.TrimSpace(attributes["ColorBasico"]) == "" {
				attributes["ColorBasico"] = value
			}
		}
	}
}

// variationAttributeKey maps variation values into preferred Falabella attribute keys.
func variationAttributeKey(variation port.CatalogVariation) string {
	switch strings.ToUpper(strings.TrimSpace(variation.Definition)) {
	case "COLOR":
		return "Color"
	case "SIZE":
		return "Talla"
	default:
		name := strings.TrimSpace(variation.Name)
		if name == "" {
			return "Variation"
		}
		return name
	}
}

// resolveImageURLs resolves filtered image URL values for realm and variation selection.
func resolveImageURLs(images []port.CatalogImage, realm string, variantVariationIDs []string) []string {
	if len(images) == 0 {
		return nil
	}

	normalizedVariantVariationIDs := normalizeTrimmedSet(variantVariationIDs)
	isVariantSelection := len(variantVariationIDs) > 0
	orderedImages := sortCatalogImages(images, isVariantSelection)
	result := make([]string, 0, len(images))
	seen := make(map[string]struct{}, len(images))
	for _, item := range orderedImages {
		if !isRealmIncluded(item.IncludedRealms, realm) {
			continue
		}
		if isVariantSelection {
			if !isSubsetOfNormalized(item.VariationIDs, normalizedVariantVariationIDs) {
				continue
			}
		} else if len(item.VariationIDs) > 0 {
			continue
		}

		url := strings.TrimSpace(item.URL)
		if url == "" {
			continue
		}
		if _, ok := seen[url]; ok {
			continue
		}
		seen[url] = struct{}{}
		result = append(result, url)
	}

	return result
}

type orderedCatalogImage struct {
	// image defines catalog image values.
	image port.CatalogImage
	// sourceIndex defines original source-order values used for stable fallback ordering.
	sourceIndex int
}

// sortCatalogImages resolves deterministic catalog-image ordering for base and variation-specific sync flows.
func sortCatalogImages(images []port.CatalogImage, variantSelection bool) []port.CatalogImage {
	if len(images) <= 1 {
		return append([]port.CatalogImage(nil), images...)
	}

	ordered := make([]orderedCatalogImage, 0, len(images))
	for index, image := range images {
		ordered = append(ordered, orderedCatalogImage{image: image, sourceIndex: index})
	}

	sort.SliceStable(ordered, func(left, right int) bool {
		leftImage := ordered[left]
		rightImage := ordered[right]

		if variantSelection {
			leftBucket := resolveVariationBucket(leftImage.image)
			rightBucket := resolveVariationBucket(rightImage.image)
			if leftBucket != rightBucket {
				return leftBucket < rightBucket
			}

			leftVariationPosition := resolveVariationSortPosition(leftImage.image, leftImage.sourceIndex)
			rightVariationPosition := resolveVariationSortPosition(rightImage.image, rightImage.sourceIndex)
			if leftVariationPosition != rightVariationPosition {
				return leftVariationPosition < rightVariationPosition
			}
		}

		leftPosition := resolveGallerySortPosition(leftImage.image, leftImage.sourceIndex)
		rightPosition := resolveGallerySortPosition(rightImage.image, rightImage.sourceIndex)
		if leftPosition != rightPosition {
			return leftPosition < rightPosition
		}

		return leftImage.sourceIndex < rightImage.sourceIndex
	})

	result := make([]port.CatalogImage, 0, len(ordered))
	for _, item := range ordered {
		result = append(result, item.image)
	}

	return result
}

// resolveVariationBucket resolves ordering buckets for variation-specific image sync.
func resolveVariationBucket(image port.CatalogImage) int {
	if len(image.VariationIDs) > 0 {
		return 0
	}

	return 1
}

// resolveGallerySortPosition resolves stable gallery sort positions with source-index fallback values.
func resolveGallerySortPosition(image port.CatalogImage, fallback int) int {
	if image.Position == nil {
		return fallback
	}
	if *image.Position < 0 {
		return 0
	}

	return *image.Position
}

// resolveVariationSortPosition resolves variation-scoped sort positions with gallery-position fallback values.
func resolveVariationSortPosition(image port.CatalogImage, fallback int) int {
	if image.VariationPosition == nil {
		return resolveGallerySortPosition(image, fallback)
	}
	if *image.VariationPosition < 0 {
		return 0
	}

	return *image.VariationPosition
}

// normalizeTrimmedSet resolves normalized string set values.
func normalizeTrimmedSet(values []string) map[string]struct{} {
	if len(values) == 0 {
		return map[string]struct{}{}
	}

	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		set[trimmed] = struct{}{}
	}

	return set
}

// uniqueTrimmedValues resolves deduplicated, non-empty string values preserving first occurrence order.
func uniqueTrimmedValues(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}

	return result
}

// isSubsetOfNormalized reports whether candidate values are subset of pre-normalized values.
func isSubsetOfNormalized(candidate []string, normalizedSuperset map[string]struct{}) bool {
	if len(candidate) == 0 {
		return true
	}
	if len(normalizedSuperset) == 0 {
		return false
	}

	for _, value := range candidate {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := normalizedSuperset[trimmed]; !ok {
			return false
		}
	}

	return true
}

// isRealmIncluded reports whether the provided realm is allowed for image sync.
// An empty includedRealms list means the image is visible in all realms.
func isRealmIncluded(includedRealms []string, realm string) bool {
	if len(includedRealms) == 0 {
		return true
	}

	trimmedRealm := strings.TrimSpace(realm)
	for _, includedRealm := range includedRealms {
		if strings.EqualFold(strings.TrimSpace(includedRealm), trimmedRealm) {
			return true
		}
	}

	return false
}

// firstNonEmpty resolves the first non-empty value from provided candidates.
func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}

	return ""
}
