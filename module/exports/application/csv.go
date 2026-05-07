package application

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"mannaiah/module/exports/port"
)

// buildContactsCSV serializes contact export rows to CSV bytes.
func buildContactsCSV(rows []port.ContactRow) ([]byte, error) {
	records := [][]string{{
		"id", "documentType", "documentNumber", "legalName", "firstName", "lastName", "email", "phone",
		"address", "addressExtra", "cityCode", "membershipOptIn", "membershipOptInAt", "privacyAccepted",
		"privacyAcceptedAt", "metadata", "createdAt", "updatedAt",
	}}
	for _, row := range rows {
		records = append(records, []string{
			row.ID,
			row.DocumentType,
			row.DocumentNumber,
			row.LegalName,
			row.FirstName,
			row.LastName,
			row.Email,
			row.Phone,
			row.Address,
			row.AddressExtra,
			row.CityCode,
			strconv.FormatBool(row.MembershipOptIn),
			formatTime(row.MembershipOptInAt),
			strconv.FormatBool(row.PrivacyAccepted),
			formatTime(row.PrivacyAcceptedAt),
			metadataString(row.Metadata),
			formatTime(row.CreatedAt),
			formatTime(row.UpdatedAt),
		})
	}

	return writeCSV(records)
}

// buildOrdersCSV serializes order export rows to CSV bytes.
func buildOrdersCSV(rows []port.OrderRow) ([]byte, error) {
	records := [][]string{{
		"id", "identifier", "realm", "contactId", "contactEmail", "address", "address2", "phone",
		"cityName", "cityCode", "status", "itemsOrdered", "paymentMethod", "metadata", "createdAt", "updatedAt",
	}}
	for _, row := range rows {
		records = append(records, []string{
			row.ID,
			row.Identifier,
			row.Realm,
			row.ContactID,
			row.ContactEmail,
			row.Address,
			row.Address2,
			row.Phone,
			row.CityName,
			row.CityCode,
			row.Status,
			itemsString(row.Items),
			row.PaymentMethod,
			metadataString(row.Metadata),
			formatTime(row.CreatedAt),
			formatTime(row.UpdatedAt),
		})
	}

	return writeCSV(records)
}

// writeCSV writes records using RFC 4180-compatible CSV encoding.
func writeCSV(records [][]string) ([]byte, error) {
	buffer := bytes.Buffer{}
	writer := csv.NewWriter(&buffer)
	if err := writer.WriteAll(records); err != nil {
		return nil, err
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

// metadataString serializes metadata values deterministically enough for CSV exports.
func metadataString(metadata map[string]string) string {
	if len(metadata) == 0 {
		return ""
	}
	body, err := json.Marshal(metadata)
	if err != nil {
		return ""
	}
	return string(body)
}

// itemsString serializes ordered items as compact semicolon-separated values.
func itemsString(items []port.OrderItemRow) string {
	if len(items) == 0 {
		return ""
	}
	parts := make([]string, 0, len(items))
	for _, item := range items {
		segments := []string{
			"sku=" + item.SKU,
			"name=" + item.AlternateName,
			"quantity=" + strconv.Itoa(item.Quantity),
			"value=" + fmt.Sprintf("%.2f", item.Value),
		}
		if item.ProductID != "" {
			segments = append(segments, "productId="+item.ProductID)
		}
		parts = append(parts, strings.Join(segments, "|"))
	}

	return strings.Join(parts, "; ")
}

// formatTime formats timestamps for CSV exports.
func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}
