package http

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"net/http"
	neturl "net/url"
	"strings"

	corehttp "mannaiah/module/core/http"

	_ "golang.org/x/image/webp"
	_ "image/png"
)

// transcodeImage fetches source images and responds with image/jpeg payload values.
func (h *Handler) transcodeImage(ctx corehttp.Context) error {
	if h == nil || !h.imageTranscode.Enabled {
		return corehttp.NewAppError(503, "image_transcode_unavailable", fmt.Errorf("falabella image transcode is disabled"))
	}

	sourceURL := strings.TrimSpace(ctx.Query("src"))
	if sourceURL == "" {
		return corehttp.NewAppError(400, "invalid_image_source", fmt.Errorf("query param src is required"))
	}

	parsedSourceURL, err := neturl.ParseRequestURI(sourceURL)
	if err != nil {
		return corehttp.NewAppError(400, "invalid_image_source", err)
	}
	if !strings.EqualFold(parsedSourceURL.Scheme, "http") && !strings.EqualFold(parsedSourceURL.Scheme, "https") {
		return corehttp.NewAppError(400, "invalid_image_source", fmt.Errorf("unsupported image source scheme %q", parsedSourceURL.Scheme))
	}
	if !isTranscodeSourceAllowed(sourceURL, h.imageTranscode.AllowedSourcePrefixes) {
		return corehttp.NewAppError(403, "image_source_forbidden", fmt.Errorf("image source is not allowed"))
	}

	client := h.imageTranscode.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: h.imageTranscode.RequestTimeout}
	}

	request, err := http.NewRequestWithContext(ctx.Context(), http.MethodGet, sourceURL, nil)
	if err != nil {
		return corehttp.NewAppError(500, "image_fetch_failed", err)
	}
	response, err := client.Do(request)
	if err != nil {
		return corehttp.NewAppError(502, "image_fetch_failed", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return corehttp.NewAppError(502, "image_fetch_failed", fmt.Errorf("source image returned status %d", response.StatusCode))
	}

	inputReader := io.LimitReader(response.Body, h.imageTranscode.MaxInputBytes)
	decodedImage, _, err := image.Decode(inputReader)
	if err != nil {
		return corehttp.NewAppError(422, "image_decode_failed", err)
	}

	buffer := bytes.Buffer{}
	if err := jpeg.Encode(&buffer, decodedImage, &jpeg.Options{Quality: 90}); err != nil {
		return corehttp.NewAppError(500, "image_encode_failed", err)
	}

	ctx.SetHeader("Content-Type", "image/jpeg")
	ctx.SetHeader("Cache-Control", "public, max-age=3600")
	return ctx.Status(200).SendBytes(buffer.Bytes())
}

// isTranscodeSourceAllowed reports whether source URLs satisfy configured allowed-prefix constraints.
func isTranscodeSourceAllowed(sourceURL string, allowedPrefixes []string) bool {
	if len(allowedPrefixes) == 0 {
		return true
	}
	parsedSourceURL, err := neturl.ParseRequestURI(strings.TrimSpace(sourceURL))
	if err != nil {
		return false
	}
	sourcePath := strings.TrimRight(parsedSourceURL.EscapedPath(), "/")
	if sourcePath == "" {
		sourcePath = "/"
	}

	for _, prefix := range allowedPrefixes {
		trimmedPrefix := strings.TrimSpace(prefix)
		if trimmedPrefix == "" {
			continue
		}

		parsedPrefix, parseErr := neturl.ParseRequestURI(trimmedPrefix)
		if parseErr != nil || parsedPrefix.Scheme == "" || parsedPrefix.Host == "" {
			if strings.HasPrefix(sourceURL, trimmedPrefix) {
				return true
			}
			continue
		}
		if !strings.EqualFold(parsedSourceURL.Scheme, parsedPrefix.Scheme) || !strings.EqualFold(parsedSourceURL.Host, parsedPrefix.Host) {
			continue
		}
		prefixPath := strings.TrimRight(parsedPrefix.EscapedPath(), "/")
		if prefixPath == "" || prefixPath == "/" {
			return true
		}
		if sourcePath == prefixPath || strings.HasPrefix(sourcePath, prefixPath+"/") {
			return true
		}
	}

	return false
}
