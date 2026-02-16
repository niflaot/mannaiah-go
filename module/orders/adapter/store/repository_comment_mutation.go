package store

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	ordersdomain "mannaiah/module/orders/domain"
	ordersport "mannaiah/module/orders/port"
)

// UpdateComment updates comment rows for order identifiers.
func (r *Repository) UpdateComment(ctx context.Context, id string, commentID string, comment ordersdomain.Comment) (*ordersdomain.Order, error) {
	trimmedID := strings.TrimSpace(id)
	resolvedCommentID, err := parseCommentID(commentID)
	if err != nil {
		return nil, err
	}

	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&orderRecord{}, "id = ?", trimmedID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ordersport.ErrNotFound
			}
			return fmt.Errorf("lock order record: %w", err)
		}

		var row orderCommentRecord
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&row, "order_id = ? AND id = ?", trimmedID, resolvedCommentID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ordersport.ErrCommentNotFound
			}
			return fmt.Errorf("lock order comment record: %w", err)
		}

		row.Author = strings.TrimSpace(comment.Author)
		row.Comment = strings.TrimSpace(comment.Comment)
		row.Internal = comment.Internal
		if !comment.OccurredAt.IsZero() {
			row.OccurredAt = comment.OccurredAt.UTC()
		}

		if err := tx.Save(&row).Error; err != nil {
			return fmt.Errorf("update order comment record: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return r.GetByID(ctx, trimmedID)
}

// DeleteComment deletes comment rows for order identifiers.
func (r *Repository) DeleteComment(ctx context.Context, id string, commentID string) (*ordersdomain.Order, error) {
	trimmedID := strings.TrimSpace(id)
	resolvedCommentID, err := parseCommentID(commentID)
	if err != nil {
		return nil, err
	}

	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&orderRecord{}, "id = ?", trimmedID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ordersport.ErrNotFound
			}
			return fmt.Errorf("lock order record: %w", err)
		}

		deleteTx := tx.Where("order_id = ? AND id = ?", trimmedID, resolvedCommentID).Delete(&orderCommentRecord{})
		if deleteTx.Error != nil {
			return fmt.Errorf("delete order comment record: %w", deleteTx.Error)
		}
		if deleteTx.RowsAffected == 0 {
			return ordersport.ErrCommentNotFound
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return r.GetByID(ctx, trimmedID)
}

// parseCommentID parses persisted comment identifier values.
func parseCommentID(value string) (uint, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0, ordersport.ErrCommentNotFound
	}

	resolved, err := strconv.ParseUint(trimmed, 10, 64)
	if err != nil || resolved == 0 {
		return 0, ordersport.ErrCommentNotFound
	}

	return uint(resolved), nil
}
