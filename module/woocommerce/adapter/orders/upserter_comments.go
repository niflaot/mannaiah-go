package orders

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	ordersapplication "mannaiah/module/orders/application"
	ordersdomain "mannaiah/module/orders/domain"
	"mannaiah/module/woocommerce/port"
)

// appendComments appends WooCommerce order comments while preventing duplicates.
func (u *Upserter) appendComments(
	ctx context.Context,
	order ordersdomain.Order,
	comments []port.OrderSyncComment,
) (changed bool, latest ordersdomain.Order, err error) {
	if len(comments) == 0 {
		return false, order, nil
	}

	sorted := sortedComments(comments)
	current := order
	hasChanges := false

	for _, value := range sorted {
		author := strings.TrimSpace(value.Author)
		if author == "" {
			author = syncCommentAuthor
		}

		comment := strings.TrimSpace(value.Comment)
		if comment == "" {
			continue
		}

		occurredAt := value.OccurredAt.UTC()
		if occurredAt.IsZero() {
			occurredAt = time.Now().UTC()
		}

		if hasCommentEntry(current.Comments, author, comment, value.Internal, occurredAt) {
			continue
		}

		next, addErr := u.orderService.AddComment(ctx, current.ID, ordersapplication.AddCommentCommand{
			Author:     author,
			Comment:    comment,
			Internal:   value.Internal,
			OccurredAt: &occurredAt,
			Source:     syncStatusAuthor,
		})
		if addErr != nil {
			return false, order, fmt.Errorf("append woocommerce order comment: %w", addErr)
		}

		current = *next
		hasChanges = true
	}

	return hasChanges, current, nil
}

// sortedComments sorts comment values by timestamp and preserves stable input order for ties.
func sortedComments(values []port.OrderSyncComment) []port.OrderSyncComment {
	if len(values) == 0 {
		return nil
	}

	sorted := make([]port.OrderSyncComment, len(values))
	copy(sorted, values)
	sort.SliceStable(sorted, func(left int, right int) bool {
		leftTime := sorted[left].OccurredAt.UTC()
		rightTime := sorted[right].OccurredAt.UTC()

		return leftTime.Before(rightTime)
	})

	return sorted
}

// hasCommentEntry reports whether comments already contain the same author-comment-internal-timestamp tuple.
func hasCommentEntry(comments []ordersdomain.Comment, author string, comment string, internal bool, occurredAt time.Time) bool {
	for _, entry := range comments {
		if strings.TrimSpace(entry.Author) != strings.TrimSpace(author) {
			continue
		}
		if strings.TrimSpace(entry.Comment) != strings.TrimSpace(comment) {
			continue
		}
		if entry.Internal != internal {
			continue
		}
		if entry.OccurredAt.UTC().Equal(occurredAt.UTC()) {
			return true
		}
	}

	return false
}
