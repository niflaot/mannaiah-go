package service

import (
	_ "embed"
	"encoding/json"
	"strconv"
	"strings"
	"sync"
)

//go:embed templates/cities.co.json
var rotulusCitiesJSON []byte

type rotulusCityCode struct {
	value string
}

// UnmarshalJSON decodes city-code values from either JSON numbers or strings.
func (c *rotulusCityCode) UnmarshalJSON(payload []byte) error {
	if len(payload) > 0 && payload[0] == '"' {
		var value string
		if err := json.Unmarshal(payload, &value); err != nil {
			return err
		}
		c.value = strings.TrimSpace(value)
		return nil
	}

	var value int
	if err := json.Unmarshal(payload, &value); err != nil {
		return err
	}
	c.value = strconv.Itoa(value)
	return nil
}

type rotulusCityEntry struct {
	Code       rotulusCityCode `json:"code"`
	Name       string          `json:"name"`
	Department string          `json:"department"`
}

type rotulusCityDisplay struct {
	Name       string
	Department string
}

var (
	rotulusCityNamesOnce sync.Once
	rotulusCityNames     map[string]rotulusCityDisplay
)

// resolveRotulusCityDisplayName resolves municipality codes into human-readable city labels.
func resolveRotulusCityDisplayName(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}

	rotulusCityNamesOnce.Do(loadRotulusCityNames)

	for _, candidate := range resolveRotulusCityLookupCandidates(trimmed) {
		if city, ok := rotulusCityNames[candidate]; ok && strings.TrimSpace(city.Name) != "" {
			return formatRotulusCityDisplay(city)
		}
	}

	return trimmed
}

// ResolveShippingCityDisplayName resolves municipality codes into city labels for shipping documents.
func ResolveShippingCityDisplayName(value string) string {
	return resolveRotulusCityDisplayName(value)
}

// resolveRotulusCityLookupCandidates resolves lookup variants for 5-digit and TCC-style 8-digit codes.
func resolveRotulusCityLookupCandidates(value string) []string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}

	candidates := []string{trimmed}
	if len(trimmed) == 8 && strings.HasSuffix(trimmed, "000") {
		candidates = append(candidates, strings.TrimLeft(strings.TrimSuffix(trimmed, "000"), "0"))
		candidates = append(candidates, trimmed[:5])
	}

	return candidates
}

// loadRotulusCityNames parses the embedded municipality dataset into a code-to-name map.
func loadRotulusCityNames() {
	rotulusCityNames = map[string]rotulusCityDisplay{}
	if len(rotulusCitiesJSON) == 0 {
		return
	}

	var entries []rotulusCityEntry
	if err := json.Unmarshal(rotulusCitiesJSON, &entries); err != nil {
		return
	}

	for _, entry := range entries {
		code := strings.TrimSpace(entry.Code.value)
		name := strings.TrimSpace(entry.Name)
		if code == "" || name == "" {
			continue
		}
		rotulusCityNames[code] = rotulusCityDisplay{
			Name:       name,
			Department: strings.TrimSpace(entry.Department),
		}
	}
}

func formatRotulusCityDisplay(city rotulusCityDisplay) string {
	name := strings.TrimSpace(city.Name)
	department := strings.TrimSpace(city.Department)
	if name == "" || department == "" {
		return name
	}
	if strings.EqualFold(name, department) {
		return name
	}
	return name + " (" + department + ")"
}
