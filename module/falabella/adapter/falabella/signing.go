package falabella

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"sort"
	"strings"
)

// signParams computes Falabella HMAC SHA256 signatures from request params.
func signParams(apiKey string, params map[string]string) string {
	canonical := canonicalQuery(params)
	if canonical == "" {
		return ""
	}

	return signCanonical(apiKey, canonical, false)
}

// signCanonical computes Falabella HMAC SHA256 signatures from canonical query values.
func signCanonical(apiKey string, canonical string, uppercase bool) string {
	h := hmac.New(sha256.New, []byte(apiKey))
	_, _ = h.Write([]byte(strings.TrimSpace(canonical)))
	signature := hex.EncodeToString(h.Sum(nil))
	if uppercase {
		return strings.ToUpper(signature)
	}

	return signature
}

// canonicalQuery builds the Falabella/Lazada canonical query string for signature generation.
func canonicalQuery(params map[string]string) string {
	return canonicalQueryWithEncoder(params, encodeRFC3986)
}

// canonicalQueryRaw builds sorted canonical query strings without URL encoding.
func canonicalQueryRaw(params map[string]string) string {
	return canonicalQueryWithEncoder(params, identityEncode)
}

// canonicalQueryWithEncoder builds canonical query strings using the provided encoder.
func canonicalQueryWithEncoder(params map[string]string, encoder func(string) string) string {
	keys := make([]string, 0, len(params))
	for key := range params {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	encoded := make([]string, 0, len(keys))
	for _, key := range keys {
		encodedKey := encoder(key)
		encodedValue := encoder(params[key])
		encoded = append(encoded, encodedKey+"="+encodedValue)
	}

	return strings.Join(encoded, "&")
}

// encodeRFC3986 applies URL encoding rules required by Seller Center signature generation.
func encodeRFC3986(value string) string {
	return strings.ReplaceAll(url.QueryEscape(value), "+", "%20")
}

// identityEncode keeps values unchanged for raw canonical string variants.
func identityEncode(value string) string {
	return value
}
