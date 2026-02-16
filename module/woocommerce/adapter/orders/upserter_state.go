package orders

import ordersdomain "mannaiah/module/orders/domain"

// hasStatusMutation reports whether status-history state changed after append-status behavior.
func hasStatusMutation(previous ordersdomain.Order, next ordersdomain.Order) bool {
	if previous.CurrentStatus != next.CurrentStatus {
		return true
	}

	return len(previous.StatusHistory) != len(next.StatusHistory)
}

// hasCommentMutation reports whether comment-history state changed after append-comment behavior.
func hasCommentMutation(previous ordersdomain.Order, next ordersdomain.Order) bool {
	return len(previous.Comments) != len(next.Comments)
}
