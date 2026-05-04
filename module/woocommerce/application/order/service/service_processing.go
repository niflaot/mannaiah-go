package service

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"sync"

	"go.uber.org/zap"
	"mannaiah/module/woocommerce/port"
)

// collectCommandsFromOrders maps order payload values into deduplicated order commands.
func collectCommandsFromOrders(
	orders []port.WooOrder,
	commandIndexByIdentifier map[string]int,
	commands []port.OrderSyncCommand,
	summary *SyncSummary,
	logger *zap.Logger,
) []port.OrderSyncCommand {
	if commandIndexByIdentifier == nil {
		commandIndexByIdentifier = map[string]int{}
	}

	for _, order := range orders {
		command, shouldProcess, reason := mapOrderToCommand(order)
		if !shouldProcess {
			logSkippedOrder(logger, order, reason)
			summary.Skipped++
			continue
		}

		key := strings.TrimSpace(command.Realm) + "::" + strings.TrimSpace(command.Identifier)
		if index, seen := commandIndexByIdentifier[key]; seen {
			mergeOrderSyncCommand(&commands[index], command)
			logSkippedOrder(logger, order, "duplicate_identifier_merged")
			summary.Skipped++
			continue
		}

		commandIndexByIdentifier[key] = len(commands)
		commands = append(commands, command)
	}

	return commands
}

// logSkippedOrder logs order skip reasons with non-sensitive diagnostic fields.
func logSkippedOrder(logger *zap.Logger, order port.WooOrder, reason mapOrderSkipReason) {
	if logger == nil {
		return
	}

	reasonValue := strings.TrimSpace(string(reason))
	if reasonValue == "" {
		reasonValue = "unknown_skip_reason"
	}
	logger.Warn(
		"woocommerce order skipped",
		zap.String("reason", reasonValue),
		zap.String("order_ref", resolveOrderReference(order)),
		zap.String("status", strings.TrimSpace(order.Status)),
		zap.Int("item_count", len(order.Items)),
	)
}

// resolveOrderReference resolves order reference values for non-sensitive diagnostics.
func resolveOrderReference(order port.WooOrder) string {
	if order.ID > 0 {
		return strconv.Itoa(order.ID)
	}

	value := strings.TrimSpace(order.Metadata["integration.woocommerce.order_id"])
	if value != "" {
		return value
	}

	return "unknown"
}

// mergeOrderSyncCommand merges duplicate order commands with latest status and additive comments/metadata.
func mergeOrderSyncCommand(existing *port.OrderSyncCommand, candidate port.OrderSyncCommand) {
	if existing == nil {
		return
	}

	if candidate.CreatedAt != nil && (existing.CreatedAt == nil || candidate.CreatedAt.UTC().Before(existing.CreatedAt.UTC())) {
		resolved := candidate.CreatedAt.UTC()
		existing.CreatedAt = &resolved
	}
	if candidate.Status != "" {
		existing.Status = strings.TrimSpace(candidate.Status)
	}
	if len(candidate.Items) > 0 {
		existing.Items = candidate.Items
	}
	if candidate.ShippingAddress != nil {
		existing.ShippingAddress = candidate.ShippingAddress
	}
	existing.Metadata = mergeMetadata(existing.Metadata, candidate.Metadata)
	existing.Comments = append(existing.Comments, candidate.Comments...)
}

// processCommands applies concurrent upsert behavior for prepared sync command values.
func (s *OrderSyncService) processCommands(ctx context.Context, commands []port.OrderSyncCommand, summary *SyncSummary) error {
	if len(commands) == 0 {
		return nil
	}

	workerCount := s.cfg.WorkerCount
	if workerCount > len(commands) {
		workerCount = len(commands)
	}

	channelSize := workerCount * 2
	if channelSize < 1 {
		channelSize = 1
	}

	workChannel := make(chan port.OrderSyncCommand, channelSize)
	resultChannel := make(chan upsertResult, channelSize)
	dispatchErrChannel := make(chan error, 1)
	var workerWait sync.WaitGroup

	for workerIndex := 0; workerIndex < workerCount; workerIndex++ {
		workerWait.Add(1)
		go func() {
			defer workerWait.Done()
			for command := range workChannel {
				if err := ctx.Err(); err != nil {
					resultChannel <- upsertResult{err: err}
					continue
				}

				var outcome port.UpsertOutcome
				upsertErr := s.executeWithBreaker(s.upsertBreaker, ErrUpsertUnavailable, func() error {
					var operationErr error
					outcome, operationErr = s.target.UpsertByIdentifier(ctx, command)
					return operationErr
				})
				resultChannel <- upsertResult{outcome: outcome, err: upsertErr}
			}
		}()
	}

	go func() {
		defer close(workChannel)

		for _, command := range commands {
			if err := ctx.Err(); err != nil {
				dispatchErrChannel <- err
				return
			}

			select {
			case workChannel <- command:
			case <-ctx.Done():
				dispatchErrChannel <- ctx.Err()
				return
			}
		}

		dispatchErrChannel <- nil
	}()

	go func() {
		workerWait.Wait()
		close(resultChannel)
	}()

	var canceledErr error
	for result := range resultChannel {
		if errors.Is(result.err, context.Canceled) || errors.Is(result.err, context.DeadlineExceeded) {
			if canceledErr == nil {
				canceledErr = result.err
			}
			continue
		}
		if canceledErr != nil {
			continue
		}

		summary.Processed++
		if result.err != nil {
			summary.Failed++
			s.logger.Warn("woocommerce order sync upsert failed", zap.Error(result.err))
			continue
		}

		applyOutcome(summary, result.outcome)
	}

	if dispatchErr := <-dispatchErrChannel; dispatchErr != nil && canceledErr == nil {
		canceledErr = dispatchErr
	}
	if canceledErr != nil {
		return canceledErr
	}

	return nil
}

// applyOutcome applies upsert outcomes to sync summary counters.
func applyOutcome(summary *SyncSummary, outcome port.UpsertOutcome) {
	switch outcome {
	case port.UpsertOutcomeCreated:
		summary.Created++
	case port.UpsertOutcomeUnchanged:
		summary.Unchanged++
	default:
		summary.Updated++
	}
}
