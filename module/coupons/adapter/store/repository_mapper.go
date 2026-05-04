package store

import (
	"strings"

	"mannaiah/module/coupons/domain"
)

// toCouponRecord maps a coupon aggregate to a root storage record.
func toCouponRecord(c domain.Coupon) couponRecord {
	return couponRecord{
		ID:                strings.TrimSpace(c.ID),
		Code:              strings.TrimSpace(c.Code),
		Origin:            strings.TrimSpace(c.Origin),
		DiscountType:      strings.TrimSpace(string(c.DiscountType)),
		DiscountAmount:    c.DiscountAmount,
		MaxUsagesGlobal:   c.MaxUsagesGlobal,
		MaxUsagesPerEmail: c.MaxUsagesPerEmail,
		Active:            c.Active,
		ExpiresAt:         c.ExpiresAt,
		WooCommerceID:     c.WooCommerceID,
		CreatedAt:         c.CreatedAt,
		UpdatedAt:         c.UpdatedAt,
	}
}

// toCouponEntity maps root and child rows to a coupon aggregate.
func toCouponEntity(
	record couponRecord,
	emails []couponAssignedEmailRecord,
	contacts []couponAssignedContactRecord,
	products []couponIncludedProductRecord,
	categories []couponIncludedCategoryRecord,
	tags []couponIncludedTagRecord,
) domain.Coupon {
	coupon := domain.Coupon{
		ID:                strings.TrimSpace(record.ID),
		Code:              strings.TrimSpace(record.Code),
		Origin:            strings.TrimSpace(record.Origin),
		DiscountType:      domain.DiscountType(strings.TrimSpace(record.DiscountType)),
		DiscountAmount:    record.DiscountAmount,
		MaxUsagesGlobal:   record.MaxUsagesGlobal,
		MaxUsagesPerEmail: record.MaxUsagesPerEmail,
		Active:            record.Active,
		ExpiresAt:         record.ExpiresAt,
		WooCommerceID:     record.WooCommerceID,
		CreatedAt:         record.CreatedAt,
		UpdatedAt:         record.UpdatedAt,
	}

	coupon.AssignedEmails = make([]string, 0, len(emails))
	for _, row := range emails {
		if v := strings.TrimSpace(row.Email); v != "" {
			coupon.AssignedEmails = append(coupon.AssignedEmails, v)
		}
	}

	coupon.AssignedContactIDs = make([]string, 0, len(contacts))
	for _, row := range contacts {
		if v := strings.TrimSpace(row.ContactID); v != "" {
			coupon.AssignedContactIDs = append(coupon.AssignedContactIDs, v)
		}
	}

	coupon.IncludedProductIDs = make([]string, 0, len(products))
	for _, row := range products {
		if v := strings.TrimSpace(row.ProductID); v != "" {
			coupon.IncludedProductIDs = append(coupon.IncludedProductIDs, v)
		}
	}

	coupon.IncludedCategoryIDs = make([]string, 0, len(categories))
	for _, row := range categories {
		if v := strings.TrimSpace(row.CategoryID); v != "" {
			coupon.IncludedCategoryIDs = append(coupon.IncludedCategoryIDs, v)
		}
	}

	coupon.IncludedTagIDs = make([]string, 0, len(tags))
	for _, row := range tags {
		if v := strings.TrimSpace(row.TagID); v != "" {
			coupon.IncludedTagIDs = append(coupon.IncludedTagIDs, v)
		}
	}

	return coupon
}

// toAssignedEmailRecords maps email strings to child rows.
func toAssignedEmailRecords(couponID string, emails []string) []couponAssignedEmailRecord {
	rows := make([]couponAssignedEmailRecord, 0, len(emails))
	for _, email := range emails {
		if v := strings.TrimSpace(email); v != "" {
			rows = append(rows, couponAssignedEmailRecord{CouponID: couponID, Email: v})
		}
	}
	return rows
}

// toAssignedContactRecords maps contact ID strings to child rows.
func toAssignedContactRecords(couponID string, contactIDs []string) []couponAssignedContactRecord {
	rows := make([]couponAssignedContactRecord, 0, len(contactIDs))
	for _, id := range contactIDs {
		if v := strings.TrimSpace(id); v != "" {
			rows = append(rows, couponAssignedContactRecord{CouponID: couponID, ContactID: v})
		}
	}
	return rows
}

// toIncludedProductRecords maps product ID strings to child rows.
func toIncludedProductRecords(couponID string, productIDs []string) []couponIncludedProductRecord {
	rows := make([]couponIncludedProductRecord, 0, len(productIDs))
	for _, id := range productIDs {
		if v := strings.TrimSpace(id); v != "" {
			rows = append(rows, couponIncludedProductRecord{CouponID: couponID, ProductID: v})
		}
	}
	return rows
}

// toIncludedCategoryRecords maps category ID strings to child rows.
func toIncludedCategoryRecords(couponID string, categoryIDs []string) []couponIncludedCategoryRecord {
	rows := make([]couponIncludedCategoryRecord, 0, len(categoryIDs))
	for _, id := range categoryIDs {
		if v := strings.TrimSpace(id); v != "" {
			rows = append(rows, couponIncludedCategoryRecord{CouponID: couponID, CategoryID: v})
		}
	}
	return rows
}

// toIncludedTagRecords maps tag ID strings to child rows.
func toIncludedTagRecords(couponID string, tagIDs []string) []couponIncludedTagRecord {
	rows := make([]couponIncludedTagRecord, 0, len(tagIDs))
	for _, id := range tagIDs {
		if v := strings.TrimSpace(id); v != "" {
			rows = append(rows, couponIncludedTagRecord{CouponID: couponID, TagID: v})
		}
	}
	return rows
}
