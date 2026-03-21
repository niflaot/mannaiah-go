package template

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"text/template"
	"time"
)

const (
	// maxOutputBytes caps rendered template output at 2 MiB.
	maxOutputBytes = 2 * 1024 * 1024
)

// ErrOutputTooLarge is returned when rendered output exceeds maxOutputBytes.
var ErrOutputTooLarge = errors.New("rendered template output exceeds 2 MiB limit")

// Renderer executes campaign Go templates with a fixed function allowlist.
type Renderer struct {
	funcMap template.FuncMap
}

// NewRenderer creates campaign template renderers with the fixed function allowlist.
func NewRenderer() *Renderer {
	return &Renderer{
		funcMap: template.FuncMap{
			"formatDate":  formatDate,
			"formatPrice": formatPrice,
			"default":     defaultValue,
			"upper":       strings.ToUpper,
			"lower":       strings.ToLower,
		},
	}
}

// Render executes the given template source string with the provided data context.
// Returns ErrOutputTooLarge when the rendered output exceeds 2 MiB.
func (r *Renderer) Render(name string, src string, data any) (string, error) {
	tmpl, err := template.New(name).Funcs(r.funcMap).Parse(src)
	if err != nil {
		return "", fmt.Errorf("parse template %q: %w", name, err)
	}

	var buf bytes.Buffer
	buf.Grow(min(len(src)*2, maxOutputBytes))

	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute template %q: %w", name, err)
	}

	if buf.Len() > maxOutputBytes {
		return "", ErrOutputTooLarge
	}

	return buf.String(), nil
}

// formatDate formats a *time.Time as "2006-01-02". Returns empty string for nil.
func formatDate(t *time.Time) string {
	if t == nil {
		return ""
	}

	return t.Format("2006-01-02")
}

// formatPrice formats a float64 price value with two decimal places.
func formatPrice(price float64) string {
	return fmt.Sprintf("%.2f", price)
}

// defaultValue returns val if non-empty, otherwise fallback.
func defaultValue(fallback string, val string) string {
	if val == "" {
		return fallback
	}

	return val
}

// min returns the smaller of two ints.
func min(a, b int) int {
	if a < b {
		return a
	}

	return b
}
