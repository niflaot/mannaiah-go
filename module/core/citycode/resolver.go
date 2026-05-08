// Package citycode resolves Colombian city names and municipality codes.
package citycode

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

//go:embed cities.json
var citiesJSON []byte

const (
	unknownCode    = "-1"
	fuzzyThreshold = 0.80
)

// CityCode holds a municipality code value that may be encoded as a JSON integer or string.
type CityCode struct {
	// value is the resolved string representation of the code.
	value string
}

// UnmarshalJSON decodes city codes from both JSON number and JSON string representations.
func (c *CityCode) UnmarshalJSON(b []byte) error {
	if len(b) > 0 && b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		c.value = normalizeCode(s)
		return nil
	}
	var n int
	if err := json.Unmarshal(b, &n); err != nil {
		return err
	}
	c.value = normalizeCode(strconv.Itoa(n))
	return nil
}

// CityEntry holds a single city record from the embedded dataset.
type CityEntry struct {
	// Code is the Colombian municipality numeric code.
	Code CityCode `json:"code"`
	// Name is the human-readable city name.
	Name string `json:"name"`
	// Normalized is the accent-stripped city name used as lookup key.
	Normalized string `json:"normalized"`
}

var cityMap map[string]string
var cityNames map[string]string
var cityKeys []string

func init() {
	var entries []CityEntry
	if err := json.Unmarshal(citiesJSON, &entries); err != nil {
		panic("citycode: failed to parse embedded cities.json: " + err.Error())
	}

	cityMap = make(map[string]string, len(entries))
	cityNames = make(map[string]string, len(entries))
	cityKeys = make([]string, 0, len(entries))

	for _, entry := range entries {
		key := strings.ToLower(strings.TrimSpace(entry.Normalized))
		code := normalizeCode(entry.Code.value)
		name := strings.TrimSpace(entry.Name)
		if key != "" && code != "" {
			if _, exists := cityMap[key]; !exists {
				cityMap[key] = code
				cityKeys = append(cityKeys, key)
			}
		}
		if code != "" && name != "" {
			cityNames[code] = name
		}
	}
}

// Resolve maps a city name string to its Colombian municipality code string.
func Resolve(name string) string {
	if IsNumericCode(name) {
		return normalizeCode(name)
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

// Name maps a municipality code to its human-readable city name.
func Name(code string) string {
	trimmed := normalizeCode(code)
	if trimmed == "" {
		return ""
	}
	if name := strings.TrimSpace(cityNames[trimmed]); name != "" {
		return name
	}
	return strings.TrimSpace(code)
}

// IsNumericCode reports whether a city code value is already a resolved numeric municipality code.
func IsNumericCode(value string) bool {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return false
	}
	n, err := strconv.Atoi(trimmed)
	return err == nil && n > 0
}

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

func similarity(a string, b string) float64 {
	if a == "" && b == "" {
		return 1
	}
	maxLen := len([]rune(a))
	if l := len([]rune(b)); l > maxLen {
		maxLen = l
	}
	if maxLen == 0 {
		return 1
	}
	distance := levenshtein(a, b)
	return 1 - float64(distance)/float64(maxLen)
}

func levenshtein(a string, b string) int {
	ar := []rune(a)
	br := []rune(b)
	if len(ar) == 0 {
		return len(br)
	}
	if len(br) == 0 {
		return len(ar)
	}

	prev := make([]int, len(br)+1)
	curr := make([]int, len(br)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ar); i++ {
		curr[0] = i
		for j := 1; j <= len(br); j++ {
			cost := 0
			if ar[i-1] != br[j-1] {
				cost = 1
			}
			curr[j] = minInt(curr[j-1]+1, prev[j]+1, prev[j-1]+cost)
		}
		prev, curr = curr, prev
	}
	return prev[len(br)]
}

func minInt(values ...int) int {
	if len(values) == 0 {
		return 0
	}
	result := values[0]
	for _, value := range values[1:] {
		if value < result {
			result = value
		}
	}
	return result
}

func normalizeCode(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	n, err := strconv.Atoi(trimmed)
	if err != nil {
		return trimmed
	}
	if n > 0 && n < 10000 {
		return fmt.Sprintf("%05d", n)
	}
	return strconv.Itoa(n)
}
