package http

import (
	"strings"

	assetsapplication "mannaiah/module/assets/application"
	corehttp "mannaiah/module/core/http"
)

// runJPGWorkerResponse defines manual JPG worker response payload values.
type runJPGWorkerResponse struct {
	// Scanned defines selected assets for the execution.
	Scanned int `json:"scanned"`
	// Converted defines successfully converted assets.
	Converted int `json:"converted"`
	// Skipped defines already-jpg assets that were skipped.
	Skipped int `json:"skipped"`
	// Failed defines assets that failed conversion/replacement.
	Failed int `json:"failed"`
	// Tags defines the effective tag filter values used by the execution.
	Tags []string `json:"tags"`
	// BatchSize defines the effective batch size used by the execution.
	BatchSize int `json:"batchSize"`
	// JPEGQuality defines the effective jpeg quality used by the execution.
	JPEGQuality int `json:"jpegQuality"`
}

// runJPGWorker triggers one manual JPG worker execution.
func (h *Handler) runJPGWorker(ctx corehttp.Context) error {
	command, err := h.resolveJPGWorkerCommand(ctx)
	if err != nil {
		return corehttp.NewAppError(400, "invalid_payload", err)
	}

	result, runErr := h.service.RunJPGWorker(ctx.Context(), command)
	if runErr != nil {
		return h.mapError(runErr)
	}

	return ctx.Status(200).JSON(runJPGWorkerResponse{
		Scanned:     result.Scanned,
		Converted:   result.Converted,
		Skipped:     result.Skipped,
		Failed:      result.Failed,
		Tags:        command.Tags,
		BatchSize:   command.BatchSize,
		JPEGQuality: command.JPEGQuality,
	})
}

// resolveJPGWorkerCommand resolves manual JPG worker command values from defaults and query overrides.
func (h *Handler) resolveJPGWorkerCommand(ctx corehttp.Context) (assetsapplication.JPGWorkerCommand, error) {
	command := h.jpgWorkerDefaults
	if command.BatchSize <= 0 {
		command.BatchSize = 100
	}
	if command.JPEGQuality <= 0 {
		command.JPEGQuality = 90
	}

	tagsQuery := strings.TrimSpace(ctx.Query("tags"))
	if tagsQuery != "" {
		command.Tags = splitCSV(tagsQuery)
	}

	batchSizeQuery := strings.TrimSpace(ctx.Query("batchSize"))
	if batchSizeQuery != "" {
		batchSize, err := parseIntQuery(ctx, "batchSize", command.BatchSize)
		if err != nil {
			return assetsapplication.JPGWorkerCommand{}, err
		}
		command.BatchSize = batchSize
	}

	jpegQualityQuery := strings.TrimSpace(ctx.Query("jpegQuality"))
	if jpegQualityQuery != "" {
		jpegQuality, err := parseIntQuery(ctx, "jpegQuality", command.JPEGQuality)
		if err != nil {
			return assetsapplication.JPGWorkerCommand{}, err
		}
		command.JPEGQuality = jpegQuality
	}

	return command, nil
}

// splitCSV splits comma-separated values trimming empty entries.
func splitCSV(raw string) []string {
	segments := strings.Split(strings.TrimSpace(raw), ",")
	result := make([]string, 0, len(segments))
	for _, segment := range segments {
		trimmed := strings.TrimSpace(segment)
		if trimmed == "" {
			continue
		}
		result = append(result, trimmed)
	}

	return result
}
