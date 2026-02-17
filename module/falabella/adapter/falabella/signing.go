package falabella

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"
)

// signParams computes Falabella HMAC SHA256 signatures from request params.
func signParams(apiKey string, params map[string]string) string {
	keys := make([]string, 0, len(params))
	for key := range params {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var payload strings.Builder
	for _, key := range keys {
		payload.WriteString(key)
		payload.WriteString(params[key])
	}

	h := hmac.New(sha256.New, []byte(apiKey))
	_, _ = h.Write([]byte(payload.String()))
	return strings.ToUpper(hex.EncodeToString(h.Sum(nil)))
}
