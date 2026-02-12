package e2e_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	corehttp "mannaiah/module/core/http"
)

// DoJSONRequest executes HTTP requests against the in-memory server and decodes JSON responses.
func (h *contactsE2EHarness) DoJSONRequest(t *testing.T, method string, path string, token string, body []byte) (int, map[string]any) {
	t.Helper()

	status, payload, _, err := doJSONRequestRaw(h.server, method, path, token, body)
	if err != nil {
		t.Fatalf("DoJSONRequest() error = %v", err)
	}

	return status, payload
}

// doJSONRequestRaw executes HTTP requests and decodes JSON responses without testing-side failures.
func doJSONRequestRaw(server *corehttp.Server, method string, path string, token string, body []byte) (int, map[string]any, http.Header, error) {
	requestBody := bytes.NewReader(body)
	request, err := http.NewRequest(method, path, requestBody)
	if err != nil {
		return 0, nil, nil, err
	}
	if len(body) > 0 {
		request.Header.Set("Content-Type", "application/json")
	}
	if strings.TrimSpace(token) != "" {
		request.Header.Set("Authorization", "Bearer "+strings.TrimSpace(token))
	}

	response, err := server.App().Test(request)
	if err != nil {
		return 0, nil, nil, err
	}
	defer func() {
		_ = response.Body.Close()
	}()

	result := map[string]any{}
	if response.ContentLength != 0 {
		payload, readErr := io.ReadAll(response.Body)
		if readErr != nil {
			return 0, nil, nil, readErr
		}
		if len(payload) > 0 {
			if err := json.Unmarshal(payload, &result); err != nil {
				var listPayload []any
				if listErr := json.Unmarshal(payload, &listPayload); listErr != nil {
					return 0, nil, nil, err
				}
				result = map[string]any{"data": listPayload}
			}
		}
	}

	return response.StatusCode, result, response.Header, nil
}

// isClosedDBError reports whether a DB close failure is caused by an already-closed handle.
func isClosedDBError(err error) bool {
	if err == nil {
		return false
	}

	return strings.Contains(strings.ToLower(strings.TrimSpace(err.Error())), "closed")
}
