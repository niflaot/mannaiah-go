// Package citycode resolves WooCommerce billing city name strings to Colombian
// municipality numeric codes using a compile-time embedded lookup table.
package citycode

import (
	_ "embed"
	"encoding/json"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

//go:embed cities.json
var citiesJSON []byte

// unknownCode is returned when no matching city can be resolved.
const unknownCode = "-1"

// fuzzyThreshold is the minimum similarity ratio required to accept a fuzzy match.
const fuzzyThreshold = 0.80

// cityCode holds a municipality code value that may be encoded as a JSON integer or string.
type cityCode struct {
	// value is the resolved string representation of the code.
	value string
}

// UnmarshalJSON decodes city codes from both JSON number and JSON string representations.
func (c *cityCode) UnmarshalJSON(b []byte) error {
	if len(b) > 0 && b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		c.value = s
		return nil
	}
	var n int
	if err := json.Unmarshal(b, &n); err != nil {
		return err
	}
	c.value = strconv.Itoa(n)
	return nil
}

// cityEntry holds a single city record from the embedded dataset.
type cityEntry struct {
	// Code is the Colombian municipality numeric code.
	Code cityCode `json:"code"`
	// Normalized is the accent-stripped city name used as lookup key.
	Normalized string `json:"normalized"`
}

// cityMap maps lowercase-normalized city names to their string municipality codes.
var cityMap map[string]string

// cityKeys holds all normalized keys for fuzzy fallback iteration.
var cityKeys []string

func init() {
	var entries []cityEntry
	if err := json.Unmarshal(citiesJSON, &entries); err != nil {
		panic("citycode: failed to parse embedded cities.json: " + err.Error())
	}

	cityMap = make(map[string]string, len(entries))
	cityKeys = make([]string, 0, len(entries))

	for _, e := range entries {
		key := strings.ToLower(strings.TrimSpace(e.Normalized))
		if key == "" {
			continue
		}
		code := e.Code.value
		if _, exists := cityMap[key]; !exists {
			cityMap[key] = code
			cityKeys = append(cityKeys, key)
		}
	}
}

// Resolve maps a city name string to its Colombian municipality code string.
// If the input is already a numeric municipality code it is returned unchanged.
// Returns "-1" when no sufficiently similar match is found.
func Resolve(name string) string {
	if IsNumericCode(name) {
		return strings.TrimSpace(name)
	}

	normalized := normalizeName(name)
	if normalized == "" {
		return unknownCode
	}

	if code, ok := cityMap[normalized]; ok {
		return code
	}

	if code, ok := prefixResolve(normalized); ok {
		return code
	}

	return fuzzyResolve(normalized)
}

// IsNumericCode reports whether a city code value is already a resolved numeric municipality code.
// Negative sentinel values such as "-1" are not considered valid numeric codes.
func IsNumericCode(value string) bool {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return false
	}
	n, err := strconv.Atoi(trimmed)
	return err == nil && n > 0
}

// prefixResolve returns the city code when the normalized input is an unambiguous
// prefix of exactly one city key (minimum 4 runes). This handles common inputs like
// "Bogota" that match longer canonical names such as "Bogota D.C.".
func prefixResolve(normalized string) (string, bool) {
	if len([]rune(normalized)) < 4 {
		return "", false
	}

	matched := ""
	count := 0
	for _, key := range cityKeys {
		if strings.HasPrefix(key, normalized) {
			matched = key
			count++
			if count > 1 {
				return "", false
			}
		}
	}

	if count == 1 {
		return cityMap[matched], true
	}

	return "", false
}

// normalizeName lowercases and strips diacritical marks from name values.
func normalizeName(s string) string {
	lower := strings.ToLower(strings.TrimSpace(s))
	if lower == "" {
		return ""
	}

	decomposed := norm.NFD.String(lower)
	result := make([]rune, 0, len(decomposed))
	for _, r := range decomposed {
		if unicode.Is(unicode.Mn, r) {
			continue
		}
		result = append(result, r)
	}

	return string(result)
}

// fuzzyResolve finds the best-matching city key using Levenshtein similarity.
// Returns unknownCode when no candidate meets the similarity threshold.
func fuzzyResolve(normalized string) string {
	best := ""
	bestSim := 0.0

	for _, key := range cityKeys {
		sim := similarity(normalized, key)
		if sim > bestSim {
			bestSim = sim
			best = key
		}
	}

	if bestSim >= fuzzyThreshold {
		return cityMap[best]
	}

	return unknownCode
}

// similarity returns a value in [0,1] representing how closely two strings match,
// based on normalised Levenshtein distance.
func similarity(a, b string) float64 {
	ra, rb := []rune(a), []rune(b)
	la, lb := len(ra), len(rb)
	if la == 0 && lb == 0 {
		return 1.0
	}
	maxLen := la
	if lb > maxLen {
		maxLen = lb
	}
	dist := levenshtein(ra, rb, la, lb)
	return 1.0 - float64(dist)/float64(maxLen)
}

// levenshtein computes the edit distance between two rune slices.
func levenshtein(ra, rb []rune, la, lb int) int {
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}

	prev := make([]int, lb+1)
	curr := make([]int, lb+1)

	for j := 0; j <= lb; j++ {
		prev[j] = j
	}

	for i := 1; i <= la; i++ {
		curr[0] = i
		for j := 1; j <= lb; j++ {
			if ra[i-1] == rb[j-1] {
				curr[j] = prev[j-1]
			} else {
				curr[j] = 1 + min3(prev[j], curr[j-1], prev[j-1])
			}
		}
		prev, curr = curr, prev
	}

	return prev[lb]
}

// min3 returns the minimum of three integers.
func min3(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}
