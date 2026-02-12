package product

import productdomain "mannaiah/module/products/domain/product"

// CopyDatasheets creates a shallow copy of datasheet slices.
func CopyDatasheets(values []productdomain.Datasheet) []productdomain.Datasheet {
	if len(values) == 0 {
		return nil
	}

	copied := make([]productdomain.Datasheet, len(values))
	copy(copied, values)
	return copied
}
