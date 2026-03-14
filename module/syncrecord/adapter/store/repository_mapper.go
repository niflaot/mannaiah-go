package store

import (
	"encoding/json"
	"sort"
	"strings"

	"mannaiah/module/syncrecord/domain"
)

// mapRunToModel maps domain run values into persistence row values.
func mapRunToModel(run *domain.SyncRun) runModel {
	model := runModel{}
	if run == nil {
		return model
	}

	model.ID = strings.TrimSpace(run.ID)
	model.Kind = strings.TrimSpace(string(run.Kind))
	model.Trigger = strings.TrimSpace(string(run.Trigger))
	model.Status = strings.TrimSpace(string(run.Status))
	model.StartedAt = run.StartedAt.UTC()
	model.EndedAt = run.EndedAt
	model.DurationMS = run.DurationMS
	model.Processed = run.Processed
	model.Succeeded = run.Succeeded
	model.Failed = run.Failed
	model.Skipped = run.Skipped
	model.ErrorCount = run.ErrorCount
	model.MetadataJSON = marshalMetadata(run.Metadata)
	model.CreatedAt = run.CreatedAt
	model.UpdatedAt = run.UpdatedAt

	return model
}

// mapErrorToModel maps domain error values into persistence row values.
func mapErrorToModel(entry domain.SyncRunError) runErrorModel {
	return runErrorModel{
		ID:        strings.TrimSpace(entry.ID),
		RunID:     strings.TrimSpace(entry.RunID),
		ErrorType: strings.TrimSpace(entry.ErrorType),
		ErrorCode: strings.TrimSpace(entry.ErrorCode),
		Message:   strings.TrimSpace(entry.Message),
		CreatedAt: entry.CreatedAt.UTC(),
	}
}

// mapModelToRun maps persistence run rows into domain values.
func mapModelToRun(model runModel) domain.SyncRun {
	run := domain.SyncRun{
		ID:         strings.TrimSpace(model.ID),
		Kind:       domain.SyncKind(strings.TrimSpace(model.Kind)),
		Trigger:    domain.SyncTrigger(strings.TrimSpace(model.Trigger)),
		Status:     domain.RunStatus(strings.TrimSpace(model.Status)),
		StartedAt:  model.StartedAt.UTC(),
		EndedAt:    model.EndedAt,
		DurationMS: model.DurationMS,
		Processed:  model.Processed,
		Succeeded:  model.Succeeded,
		Failed:     model.Failed,
		Skipped:    model.Skipped,
		ErrorCount: model.ErrorCount,
		Metadata:   unmarshalMetadata(model.MetadataJSON),
		CreatedAt:  model.CreatedAt.UTC(),
		UpdatedAt:  model.UpdatedAt.UTC(),
	}
	if len(model.Errors) > 0 {
		run.Errors = make([]domain.SyncRunError, 0, len(model.Errors))
		for _, errorModel := range model.Errors {
			run.Errors = append(run.Errors, mapModelToError(errorModel))
		}
		sort.Slice(run.Errors, func(i, j int) bool {
			return run.Errors[i].CreatedAt.Before(run.Errors[j].CreatedAt)
		})
	}

	return run
}

// mapModelToError maps persistence error rows into domain values.
func mapModelToError(model runErrorModel) domain.SyncRunError {
	return domain.SyncRunError{
		ID:        strings.TrimSpace(model.ID),
		RunID:     strings.TrimSpace(model.RunID),
		ErrorType: strings.TrimSpace(model.ErrorType),
		ErrorCode: strings.TrimSpace(model.ErrorCode),
		Message:   strings.TrimSpace(model.Message),
		CreatedAt: model.CreatedAt.UTC(),
	}
}

// marshalMetadata serializes metadata maps to JSON values.
func marshalMetadata(metadata map[string]string) string {
	if len(metadata) == 0 {
		return ""
	}

	payload, err := json.Marshal(metadata)
	if err != nil {
		return ""
	}

	return string(payload)
}

// unmarshalMetadata deserializes metadata json into maps.
func unmarshalMetadata(value string) map[string]string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}

	metadata := map[string]string{}
	if err := json.Unmarshal([]byte(trimmed), &metadata); err != nil {
		return nil
	}
	if len(metadata) == 0 {
		return nil
	}

	return metadata
}
