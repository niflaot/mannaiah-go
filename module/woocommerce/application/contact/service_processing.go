package contact

import (
	"context"
	"errors"
	"strings"
	"sync"

	"go.uber.org/zap"
	"mannaiah/module/woocommerce/port"
)

// collectCommandsFromOrders maps order payload values into deduplicated contact commands.
func collectCommandsFromOrders(orders []port.WooOrder, seenEmails map[string]struct{}, summary *SyncSummary) []port.ContactSyncCommand {
	commands := make([]port.ContactSyncCommand, 0, len(orders))
	for _, order := range orders {
		command, shouldProcess := mapOrderToCommand(order)
		if !shouldProcess {
			summary.Skipped++
			continue
		}

		emailKey := strings.ToLower(strings.TrimSpace(command.Email))
		if _, seen := seenEmails[emailKey]; seen {
			summary.Skipped++
			continue
		}
		seenEmails[emailKey] = struct{}{}
		commands = append(commands, command)
	}

	return commands
}

// processCommands applies concurrent upsert behavior for prepared sync command values.
func (s *ContactSyncService) processCommands(ctx context.Context, commands []port.ContactSyncCommand, summary *SyncSummary) error {
	if len(commands) == 0 {
		return nil
	}

	workerCount := s.cfg.WorkerCount
	if workerCount > len(commands) {
		workerCount = len(commands)
	}

	workChannel := make(chan port.ContactSyncCommand, len(commands))
	resultChannel := make(chan upsertResult, len(commands))
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
					outcome, operationErr = s.target.UpsertByEmail(ctx, command)
					return operationErr
				})
				resultChannel <- upsertResult{outcome: outcome, err: upsertErr}
			}
		}()
	}

	for _, command := range commands {
		if err := ctx.Err(); err != nil {
			close(workChannel)
			workerWait.Wait()
			close(resultChannel)
			return err
		}
		workChannel <- command
	}
	close(workChannel)

	workerWait.Wait()
	close(resultChannel)

	for result := range resultChannel {
		if errors.Is(result.err, context.Canceled) || errors.Is(result.err, context.DeadlineExceeded) {
			return result.err
		}

		summary.Processed++
		if result.err != nil {
			summary.Failed++
			s.logger.Warn("woocommerce contact sync upsert failed", zap.Error(result.err))
			continue
		}

		applyOutcome(summary, result.outcome)
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
