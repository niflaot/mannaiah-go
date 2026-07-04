// Package citycode resolves Colombian city names and municipality codes.
package citycode

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	citiesplatform "github.com/flockstore/lib-go-cities/platform"
)

//go:embed cities.json
var citiesJSON []byte

const (
	unknownCode      = "-1"
	matcherThreshold = 0.80
)

var cityMatcher *citiesplatform.Matcher
var cityNames map[string]string

func init() {
	matcher, err := citiesplatform.LoadBytes("module/core/citycode/cities.json", citiesJSON)
	if err != nil {
		panic("citycode: failed to parse embedded cities.json: " + err.Error())
	}

	cityMatcher = matcher
	cityNames = make(map[string]string, len(matcher.Cities()))
	for _, city := range matcher.Cities() {
		code := normalizeCode(string(city.Code))
		name := strings.TrimSpace(city.Name)
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

	match, found, err := cityMatcher.Match(context.Background(), citiesplatform.SearchRequest{
		City:      name,
		Threshold: matcherThreshold,
	})
	if err != nil || !found {
		return unknownCode
	}

	return normalizeCode(string(match.City.Code))
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

// ResolveResult defines a city resolution outcome with failure details.
type ResolveResult struct {
	// Code defines the resolved municipality code when Found is true.
	Code string
	// Name defines the resolved city display name when Found is true.
	Name string
	// Department defines the resolved department display name when Found is true.
	Department string
	// Found reports whether the input resolved safely.
	Found bool
	// Reason defines the city matcher rejection reason when Found is false.
	Reason string
	// Suggestions defines plausible city alternatives for operator repair.
	Suggestions []ResolveSuggestion
}

// ResolveSuggestion defines a suggested city candidate for failed resolution.
type ResolveSuggestion struct {
	// Code defines the suggested municipality code.
	Code string
	// Name defines the suggested city display name.
	Name string
	// Department defines the suggested department display name.
	Department string
	// Confidence defines the match confidence from 0 to 1.
	Confidence float64
}

// ResolveDetailed maps city and department text to a safe city-code result.
func ResolveDetailed(ctx context.Context, city string, department string) (ResolveResult, error) {
	match, found, err := cityMatcher.Match(ctx, citiesplatform.SearchRequest{
		City:       city,
		Department: department,
		Threshold:  matcherThreshold,
	})
	if err != nil {
		return ResolveResult{}, fmt.Errorf("resolve city: %w", err)
	}
	if found {
		return ResolveResult{
			Code:       normalizeCode(string(match.City.Code)),
			Name:       strings.TrimSpace(match.City.Name),
			Department: strings.TrimSpace(match.City.Department),
			Found:      true,
		}, nil
	}

	return ResolveResult{
		Found:       false,
		Reason:      string(match.Reason),
		Suggestions: mapSuggestions(match.Suggestions),
	}, nil
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

func mapSuggestions(candidates []citiesplatform.MatchCandidate) []ResolveSuggestion {
	suggestions := make([]ResolveSuggestion, 0, len(candidates))
	for _, candidate := range candidates {
		suggestions = append(suggestions, ResolveSuggestion{
			Code:       normalizeCode(string(candidate.City.Code)),
			Name:       strings.TrimSpace(candidate.City.Name),
			Department: strings.TrimSpace(candidate.City.Department),
			Confidence: candidate.Confidence,
		})
	}

	return suggestions
}

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
