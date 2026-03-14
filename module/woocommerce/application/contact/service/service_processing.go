package service

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"mannaiah/module/woocommerce/port"
)

// collectCommandsFromOrders maps order payload values into deduplicated contact commands.
func collectCommandsFromOrders(orders []port.WooOrder, commandIndexByEmail map[string]int, commands []port.ContactSyncCommand, summary *SyncSummary) []port.ContactSyncCommand {
	if commandIndexByEmail == nil {
		commandIndexByEmail = map[string]int{}
	}

	for _, order := range orders {
		command, shouldProcess := mapOrderToCommand(order)
		if !shouldProcess {
			summary.Skipped++
			continue
		}

		emailKey := strings.ToLower(strings.TrimSpace(command.Email))
		if index, seen := commandIndexByEmail[emailKey]; seen {
			mergeContactSyncCommand(&commands[index], command)
			summary.Skipped++
			continue
		}
		commandIndexByEmail[emailKey] = len(commands)
		commands = append(commands, command)
	}

	return commands
}

// mergeContactSyncCommand merges duplicate-email sync command values while preserving oldest order timestamps.
func mergeContactSyncCommand(existing *port.ContactSyncCommand, candidate port.ContactSyncCommand) {
	if existing == nil {
		return
	}

	if strings.TrimSpace(existing.FirstName) == "" && strings.TrimSpace(existing.LegalName) == "" {
		existing.FirstName = strings.TrimSpace(candidate.FirstName)
	}
	if strings.TrimSpace(existing.LastName) == "" && strings.TrimSpace(existing.LegalName) == "" {
		existing.LastName = strings.TrimSpace(candidate.LastName)
	}
	if strings.TrimSpace(existing.LegalName) == "" && strings.TrimSpace(existing.FirstName) == "" && strings.TrimSpace(existing.LastName) == "" {
		existing.LegalName = strings.TrimSpace(candidate.LegalName)
	}
	if strings.TrimSpace(existing.Phone) == "" {
		existing.Phone = strings.TrimSpace(candidate.Phone)
	}
	if strings.TrimSpace(existing.Address) == "" {
		existing.Address = strings.TrimSpace(candidate.Address)
	}
	if strings.TrimSpace(existing.AddressExtra) == "" {
		existing.AddressExtra = strings.TrimSpace(candidate.AddressExtra)
	}
	if strings.TrimSpace(existing.CityCode) == "" {
		existing.CityCode = strings.TrimSpace(candidate.CityCode)
	}
	if strings.TrimSpace(existing.DocumentType) == "" {
		existing.DocumentType = strings.TrimSpace(candidate.DocumentType)
	}
	if strings.TrimSpace(existing.DocumentNumber) == "" {
		existing.DocumentNumber = strings.TrimSpace(candidate.DocumentNumber)
	}

	usesCandidateOldest := shouldUseCandidateCreatedAt(existing.CreatedAt, candidate.CreatedAt)
	if usesCandidateOldest {
		existing.CreatedAt = cloneTimePointer(candidate.CreatedAt)
	}
	existing.Metadata = mergeSyncMetadata(existing.Metadata, candidate.Metadata, usesCandidateOldest)
}

// shouldUseCandidateCreatedAt reports whether candidate timestamps should replace existing oldest timestamps.
func shouldUseCandidateCreatedAt(existing *time.Time, candidate *time.Time) bool {
	if candidate == nil || candidate.IsZero() {
		return false
	}
	if existing == nil || existing.IsZero() {
		return true
	}

	return candidate.UTC().Before(existing.UTC())
}

// cloneTimePointer clones optional time pointer values.
func cloneTimePointer(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}

	copied := value.UTC()
	return &copied
}

// mergeSyncMetadata merges sync metadata while preserving oldest-order metadata keys.
func mergeSyncMetadata(existing map[string]string, candidate map[string]string, useCandidateOldest bool) map[string]string {
	merged := cloneStringMap(existing)
	if merged == nil {
		merged = map[string]string{}
	}

	for key, value := range candidate {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			continue
		}
		trimmedValue := strings.TrimSpace(value)
		if _, exists := merged[trimmedKey]; !exists {
			merged[trimmedKey] = trimmedValue
			continue
		}
		if isCheckerMetadataKey(trimmedKey) {
			merged[trimmedKey] = trimmedValue
			continue
		}
		if !isOldestOrderMetadataKey(trimmedKey) {
			continue
		}
		if useCandidateOldest {
			merged[trimmedKey] = trimmedValue
		}
	}
	normalizeCircleOptInMetadata(merged)

	if len(merged) == 0 {
		return nil
	}

	return merged
}

// isCheckerMetadataKey reports whether metadata keys belong to checker decision metadata groups.
func isCheckerMetadataKey(value string) bool {
	return strings.HasPrefix(strings.TrimSpace(value), checkerMetadataPrefix)
}

// isOldestOrderMetadataKey reports whether metadata keys depend on oldest-order selection behavior.
func isOldestOrderMetadataKey(value string) bool {
	switch value {
	case syncMetadataOldestOrderIDKey, syncMetadataOldestOrderAtKey:
		return true
	default:
		return false
	}
}

// cloneStringMap clones map values.
func cloneStringMap(values map[string]string) map[string]string {
	if values == nil {
		return nil
	}

	cloned := make(map[string]string, len(values))
	for key, value := range values {
		cloned[key] = value
	}

	return cloned
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

	channelSize := workerCount * 2
	if channelSize < 1 {
		channelSize = 1
	}

	workChannel := make(chan port.ContactSyncCommand, channelSize)
	resultChannel := make(chan upsertResult, channelSize)
	dispatchErrChannel := make(chan error, 1)
	var workerWait sync.WaitGroup

	for workerIndex := 0; workerIndex < workerCount; workerIndex++ {
		workerWait.Add(1)
		go func() {
			defer workerWait.Done()
			for command := range workChannel {
				if err := ctx.Err(); err != nil {
					resultChannel <- upsertResult{command: command, err: err}
					continue
				}

				var outcome port.UpsertOutcome
				upsertErr := s.executeWithBreaker(s.upsertBreaker, ErrUpsertUnavailable, func() error {
					var operationErr error
					outcome, operationErr = s.target.UpsertByEmail(ctx, command)
					return operationErr
				})
				resultChannel <- upsertResult{command: command, outcome: outcome, err: upsertErr}
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
			s.logger.Warn("woocommerce contact sync upsert failed", zap.Error(result.err))
			continue
		}

		applyOutcome(summary, result.outcome)
		s.stampMembership(ctx, result.command)
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
