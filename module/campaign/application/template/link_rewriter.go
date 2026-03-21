package template

import (
	"regexp"
	"strings"
)

// hrefRegex matches href attribute values in HTML anchor tags.
var hrefRegex = regexp.MustCompile(`(?i)(href=")([^"]+)(")`)

// RewriteLinks appends UTM tracking parameters to all http/https links in html.
// Parameters are built from campaignID and campaignSlug. Non-HTTP links (mailto:, tel:, #)
// are left unchanged.
func RewriteLinks(html string, campaignID string, campaignSlug string) string {
	if html == "" {
		return html
	}

	utmSuffix := buildUTMSuffix(campaignID, campaignSlug)
	if utmSuffix == "" {
		return html
	}

	return hrefRegex.ReplaceAllStringFunc(html, func(match string) string {
		sub := hrefRegex.FindStringSubmatch(match)
		if len(sub) != 4 {
			return match
		}
		rawURL := sub[2]
		lower := strings.ToLower(rawURL)
		if !strings.HasPrefix(lower, "http://") && !strings.HasPrefix(lower, "https://") {
			return match
		}
		sep := "?"
		if strings.Contains(rawURL, "?") {
			sep = "&"
		}

		return sub[1] + rawURL + sep + utmSuffix + sub[3]
	})
}

// buildUTMSuffix builds a URL-encoded UTM query string fragment.
func buildUTMSuffix(campaignID string, campaignSlug string) string {
	var b strings.Builder
	b.WriteString("utm_source=email&utm_medium=campaign")
	if slug := strings.TrimSpace(campaignSlug); slug != "" {
		b.WriteString("&utm_campaign=")
		b.WriteString(slug)
	}
	if id := strings.TrimSpace(campaignID); id != "" {
		b.WriteString("&utm_id=")
		b.WriteString(id)
	}

	return b.String()
}
