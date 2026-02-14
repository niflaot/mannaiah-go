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

// appendCommentStatuses appends comment-derived status entries while preventing duplicate history rows.
func (u *Upserter) appendCommentStatuses(
	ctx context.Context,
	order ordersdomain.Order,
	status ordersdomain.Status,
	comments []port.OrderSyncComment,
) (changed bool, latest ordersdomain.Order, err error) {
	if len(comments) == 0 {
		return false, order, nil
	}

	sorted := sortedComments(comments)
	current := order
	hasChanges := false

	for _, comment := range sorted {
		noteOwner := strings.TrimSpace(comment.Owner)
		if noteOwner == "" {
			noteOwner = strings.TrimSpace(comment.Author)
		}
		if noteOwner == "" {
			noteOwner = syncNoteOwner
		}

		note := strings.TrimSpace(comment.Note)
		if note == "" {
			note = strings.TrimSpace(comment.Description)
		}
		if note == "" {
			continue
		}

		occurredAt := comment.OccurredAt.UTC()
		if occurredAt.IsZero() {
			occurredAt = time.Now().UTC()
		}
		if hasStatusEntry(current.StatusHistory, status, syncStatusAuthor, syncStatusDescription, noteOwner, note, occurredAt) {
			continue
		}

		next, updateErr := u.orderService.UpdateStatus(ctx, current.ID, ordersapplication.UpdateStatusCommand{
			Status:      status,
			Author:      syncStatusAuthor,
			Description: syncStatusDescription,
			NoteOwner:   noteOwner,
			Note:        note,
			OccurredAt:  &occurredAt,
		})
		if updateErr != nil {
			return false, order, fmt.Errorf("append woocommerce order comment status: %w", updateErr)
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

// hasStatusEntry reports whether status history already contains the same status-author-description-timestamp tuple.
func hasStatusEntry(
	history []ordersdomain.StatusEntry,
	status ordersdomain.Status,
	author string,
	description string,
	noteOwner string,
	note string,
	occurredAt time.Time,
) bool {
	for _, entry := range history {
		if entry.Status != status {
			continue
		}
		if strings.TrimSpace(entry.Author) != strings.TrimSpace(author) {
			continue
		}
		if strings.TrimSpace(entry.Description) != strings.TrimSpace(description) {
			continue
		}
		if strings.TrimSpace(entry.NoteOwner) != strings.TrimSpace(noteOwner) {
			continue
		}
		if strings.TrimSpace(entry.Note) != strings.TrimSpace(note) {
			continue
		}
		if entry.OccurredAt.UTC().Equal(occurredAt.UTC()) {
			return true
		}
	}

	return false
}
