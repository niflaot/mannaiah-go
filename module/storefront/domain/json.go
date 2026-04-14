package domain

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

// NormalizeJSONObject compacts JSON objects and resolves an empty object fallback.
func NormalizeJSONObject(raw json.RawMessage) (json.RawMessage, error) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return json.RawMessage(`{}`), nil
	}

	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, err
	}
	if _, ok := value.(map[string]any); !ok {
		return nil, fmt.Errorf("expected json object")
	}

	return compactJSON(raw)
}

// NormalizeJSONDocument compacts any valid JSON payload and resolves an empty object fallback.
func NormalizeJSONDocument(raw json.RawMessage) (json.RawMessage, error) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return json.RawMessage(`{}`), nil
	}

	if !json.Valid(raw) {
		return nil, fmt.Errorf("invalid json document")
	}

	return compactJSON(raw)
}

// SnapshotHash returns a stable checksum for one renderable snapshot.
func SnapshotHash(metadata json.RawMessage, content json.RawMessage) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(string(metadata)) + "\n" + strings.TrimSpace(string(content))))
	return hex.EncodeToString(sum[:])
}

// CloneJSON copies raw JSON values into a detached buffer.
func CloneJSON(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return json.RawMessage(`{}`)
	}

	cloned := make(json.RawMessage, len(raw))
	copy(cloned, raw)
	return cloned
}

// compactJSON returns a storage-safe compact JSON value.
func compactJSON(raw json.RawMessage) (json.RawMessage, error) {
	var buffer bytes.Buffer
	if err := json.Compact(&buffer, raw); err != nil {
		return nil, err
	}

	return json.RawMessage(buffer.Bytes()), nil
}
