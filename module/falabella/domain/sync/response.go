package sync

import (
	"encoding/xml"
	"fmt"
	"strings"
)

// ActionResponse defines parsed Falabella sync action response values.
type ActionResponse struct {
	// RequestID defines Falabella feed identifier values returned on async submission.
	RequestID string
	// RequestAction defines Falabella action type values (for example, ProductCreate or ProductUpdate).
	RequestAction string
	// Warnings defines WarningDetail values included in the response body.
	Warnings []Warning
}

// Warning defines Falabella sync action warning values.
type Warning struct {
	// Field defines the Falabella field that triggered the warning.
	Field string
	// Message defines the warning message description.
	Message string
	// Value defines the field value that triggered the warning.
	Value string
}

// HasWarnings reports whether the response contains warning values.
func (r *ActionResponse) HasWarnings() bool {
	return r != nil && len(r.Warnings) > 0
}

// IsCreate reports whether the response action was a product creation.
func (r *ActionResponse) IsCreate() bool {
	return r != nil && r.RequestAction == "ProductCreate"
}

// SyncAction resolves the domain sync action from the response action.
func (r *ActionResponse) SyncAction() SyncAction {
	if r != nil && r.RequestAction == "ProductUpdate" {
		return SyncActionUpdate
	}

	return SyncActionCreate
}

// HasRequiredFieldViolations reports whether warnings indicate missing required Falabella field values.
func (r *ActionResponse) HasRequiredFieldViolations() bool {
	if r == nil || len(r.Warnings) == 0 {
		return false
	}

	for _, w := range r.Warnings {
		msg := strings.ToLower(strings.TrimSpace(w.Message))
		if strings.Contains(msg, "cannot be empty") {
			return true
		}
	}

	return false
}

// WarningMessages resolves human-readable warning message values.
func (r *ActionResponse) WarningMessages() []string {
	if r == nil || len(r.Warnings) == 0 {
		return nil
	}

	messages := make([]string, 0, len(r.Warnings))
	for _, w := range r.Warnings {
		message := strings.TrimSpace(w.Message)
		field := strings.TrimSpace(w.Field)
		if message == "" {
			continue
		}
		if field != "" {
			messages = append(messages, fmt.Sprintf("[%s] %s", field, message))
		} else {
			messages = append(messages, message)
		}
	}

	return messages
}

// ParseActionResponse parses Falabella sync action XML response values.
func ParseActionResponse(data []byte) (*ActionResponse, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty response body")
	}

	var raw xmlActionResponse
	if err := xml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("unmarshal action response: %w", err)
	}

	warnings := make([]Warning, 0, len(raw.Body.WarningDetails))
	for _, w := range raw.Body.WarningDetails {
		warnings = append(warnings, Warning{
			Field:   strings.TrimSpace(w.Field),
			Message: strings.TrimSpace(w.Message),
			Value:   strings.TrimSpace(w.Value),
		})
	}

	return &ActionResponse{
		RequestID:     strings.TrimSpace(raw.Head.RequestID),
		RequestAction: strings.TrimSpace(raw.Head.RequestAction),
		Warnings:      warnings,
	}, nil
}

// xmlActionResponse defines XML response structure for Falabella sync actions.
type xmlActionResponse struct {
	XMLName xml.Name              `xml:"SuccessResponse"`
	Head    xmlActionResponseHead `xml:"Head"`
	Body    xmlActionResponseBody `xml:"Body"`
}

// xmlActionResponseHead defines XML head structure for Falabella sync actions.
type xmlActionResponseHead struct {
	// RequestID defines Falabella request identifier values.
	RequestID string `xml:"RequestId"`
	// RequestAction defines Falabella action type values.
	RequestAction string `xml:"RequestAction"`
	// ResponseType defines Falabella response type values.
	ResponseType string `xml:"ResponseType"`
	// Timestamp defines Falabella response timestamp values.
	Timestamp string `xml:"Timestamp"`
}

// xmlActionResponseBody defines XML body structure for Falabella sync actions.
type xmlActionResponseBody struct {
	// WarningDetails defines Falabella warning detail values.
	WarningDetails []xmlWarningDetail `xml:"WarningDetail"`
}

// xmlWarningDetail defines Falabella WarningDetail XML element values.
type xmlWarningDetail struct {
	// Field defines the field name that triggered the warning.
	Field string `xml:"Field"`
	// Message defines the warning message.
	Message string `xml:"Message"`
	// Value defines the field value that triggered the warning.
	Value string `xml:"Value"`
}
