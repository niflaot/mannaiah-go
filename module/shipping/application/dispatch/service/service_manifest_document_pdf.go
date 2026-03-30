package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"

	"github.com/jung-kurt/gofpdf"
	"golang.org/x/text/encoding/charmap"
)

const (
	// batchManifestPageWidthMM defines letter-size page width values in millimeters.
	batchManifestPageWidthMM = 215.9
	// batchManifestPageHeightMM defines letter-size page height values in millimeters.
	batchManifestPageHeightMM = 279.4
	// batchManifestMarginMM defines uniform page margin values in millimeters.
	batchManifestMarginMM = 10.0
	// batchManifestRowHeightMM defines table row-height values in millimeters.
	batchManifestRowHeightMM = 7.0
	// batchManifestItemsLineHeightMM defines item-list line-height values in millimeters.
	batchManifestItemsLineHeightMM = 4.0
)

var (
	// batchManifestTableColumnWidths defines rendered summary table column widths.
	// Columns: TrackingNumber, FreightCost, RecipientName, OrderNumber, City, Items.
	batchManifestTableColumnWidths = []float64{32, 22, 44, 24, 22, 51}
)

// buildBatchManifestCoverPDF creates one summary cover PDF page in letter format.
func (s *Service) buildBatchManifestCoverPDF(ctx context.Context, meta batchManifestCoverMeta, rows []batchManifestCoverRow) ([]byte, error) {
	pdf := gofpdf.NewCustom(&gofpdf.InitType{
		UnitStr: "mm",
		Size: gofpdf.SizeType{
			Wd: batchManifestPageWidthMM,
			Ht: batchManifestPageHeightMM,
		},
	})
	pdf.SetMargins(batchManifestMarginMM, batchManifestMarginMM, batchManifestMarginMM)
	pdf.SetAutoPageBreak(false, batchManifestMarginMM)
	pdf.AddPage()

	s.drawBatchManifestCoverHeader(ctx, pdf, meta)
	s.drawBatchManifestTableHeader(pdf)

	for _, row := range rows {
		rowHeight := s.batchManifestTableRowHeight(pdf, row)
		if pdf.GetY()+rowHeight > batchManifestPageHeightMM-batchManifestMarginMM {
			pdf.AddPage()
			s.drawBatchManifestTableHeader(pdf)
		}
		s.drawBatchManifestTableRow(pdf, row, rowHeight)
	}

	var output bytes.Buffer
	if err := pdf.Output(&output); err != nil {
		return nil, err
	}

	return append([]byte(nil), output.Bytes()...), nil
}

// drawBatchManifestCoverHeader draws cover header values and optional logo image.
func (s *Service) drawBatchManifestCoverHeader(ctx context.Context, pdf *gofpdf.Fpdf, meta batchManifestCoverMeta) {
	if pdf == nil {
		return
	}
	template := s.resolveBatchManifestCoverTemplate()
	logoY := batchManifestMarginMM
	logoWidth := 28.0
	logoX := batchManifestPageWidthMM - batchManifestMarginMM - logoWidth

	if logoBytes, imageType, err := s.downloadBatchManifestLogo(ctx); err == nil && len(logoBytes) > 0 {
		imageName := "batch_manifest_logo"
		opts := gofpdf.ImageOptions{ImageType: imageType, ReadDpi: true}
		pdf.RegisterImageOptionsReader(imageName, opts, bytes.NewReader(logoBytes))
		pdf.ImageOptions(imageName, logoX, logoY, logoWidth, 0, false, opts, 0, "")
	}

	headerX := batchManifestMarginMM
	headerWidth := logoX - headerX - 6
	pdf.SetXY(headerX, batchManifestMarginMM)
	pdf.SetFont("Arial", "B", 11)
	pdf.CellFormat(headerWidth, 6, encodeBatchManifestText(template.Title), "", 1, "L", false, 0, "")

	pdf.SetX(headerX)
	pdf.SetFont("Arial", "", 9)
	pdf.CellFormat(headerWidth, 5, encodeBatchManifestText(fmt.Sprintf("%s: %s", template.BatchIDLabel, sanitizeBatchManifestValue(meta.BatchID, template.EmptyValueFallback))), "", 1, "L", false, 0, "")
	pdf.SetX(headerX)
	pdf.CellFormat(headerWidth, 5, encodeBatchManifestText(fmt.Sprintf("%s: %s", template.GeneratedLabel, meta.GeneratedAt.UTC().Format("2006-01-02 15:04:05 UTC"))), "", 1, "L", false, 0, "")
	pdf.SetX(headerX)
	pdf.CellFormat(headerWidth, 5, encodeBatchManifestText(fmt.Sprintf("%s: %s", template.CarrierLabel, sanitizeBatchManifestValue(meta.CarrierID, template.EmptyValueFallback))), "", 1, "L", false, 0, "")
	pdf.SetX(headerX)
	pdf.CellFormat(headerWidth, 5, encodeBatchManifestText(fmt.Sprintf("%s: %d", template.QuantityLabel, maxBatchManifestQuantity(meta.Quantity))), "", 1, "L", false, 0, "")

	pdf.SetY(42)
}

// drawBatchManifestTableHeader draws one summary table header row.
func (s *Service) drawBatchManifestTableHeader(pdf *gofpdf.Fpdf) {
	if pdf == nil {
		return
	}
	template := s.resolveBatchManifestCoverTemplate()
	headers := []string{
		template.TrackingNumberHeader,
		template.FreightHeader,
		template.RecipientHeader,
		template.OrderNumberHeader,
		template.CityHeader,
		template.ItemsHeader,
	}
	pdf.SetFont("Arial", "B", 8)
	pdf.SetFillColor(240, 240, 240)
	for index, header := range headers {
		pdf.CellFormat(batchManifestTableColumnWidths[index], batchManifestRowHeightMM, encodeBatchManifestText(header), "1", 0, "L", true, 0, "")
	}
	pdf.Ln(-1)
}

// drawBatchManifestTableRow draws one summary table row.
func (s *Service) drawBatchManifestTableRow(pdf *gofpdf.Fpdf, row batchManifestCoverRow, rowHeight float64) {
	if pdf == nil {
		return
	}
	template := s.resolveBatchManifestCoverTemplate()
	itemCellText := formatBatchManifestItemsAsList(row.Items, template.ItemBulletPrefix, template.EmptyValueFallback)
	startX, startY := pdf.GetX(), pdf.GetY()

	freightText := fmt.Sprintf("$%.0f", row.FreightCost)
	if row.FreightCost <= 0 {
		freightText = template.EmptyValueFallback
	}
	pdf.SetFont("Arial", "", 8)
	pdf.CellFormat(batchManifestTableColumnWidths[0], rowHeight, encodeBatchManifestText(truncateBatchManifestValue(row.TrackingNumber, 24, template.EmptyValueFallback)), "1", 0, "L", false, 0, "")
	pdf.CellFormat(batchManifestTableColumnWidths[1], rowHeight, encodeBatchManifestText(freightText), "1", 0, "R", false, 0, "")
	pdf.CellFormat(batchManifestTableColumnWidths[2], rowHeight, encodeBatchManifestText(truncateBatchManifestValue(row.RecipientName, 30, template.EmptyValueFallback)), "1", 0, "L", false, 0, "")
	pdf.CellFormat(batchManifestTableColumnWidths[3], rowHeight, encodeBatchManifestText(truncateBatchManifestValue(row.OrderNumber, 16, template.EmptyValueFallback)), "1", 0, "L", false, 0, "")
	pdf.CellFormat(batchManifestTableColumnWidths[4], rowHeight, encodeBatchManifestText(truncateBatchManifestValue(row.City, 14, template.EmptyValueFallback)), "1", 0, "L", false, 0, "")

	itemCellX := startX + sumBatchManifestColumnWidths(5)
	itemCellWidth := batchManifestTableColumnWidths[5]
	pdf.Rect(itemCellX, startY, itemCellWidth, rowHeight, "D")
	pdf.SetXY(itemCellX, startY)
	pdf.MultiCell(itemCellWidth, batchManifestItemsLineHeightMM, encodeBatchManifestText(itemCellText), "", "L", false)
	pdf.SetXY(startX, startY+rowHeight)
}

// downloadBatchManifestLogo downloads one cover-logo image and resolves image type values for gofpdf.
func (s *Service) downloadBatchManifestLogo(ctx context.Context) ([]byte, string, error) {
	if s == nil || s.manifestDocuments == nil || s.manifestDocuments.httpClient == nil {
		return nil, "", fmt.Errorf("manifest logo client is unavailable")
	}
	logoURL := strings.TrimSpace(s.manifestDocuments.logoURL)
	if logoURL == "" {
		return nil, "", fmt.Errorf("manifest logo url is empty")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, logoURL, nil)
	if err != nil {
		return nil, "", err
	}
	response, err := s.manifestDocuments.httpClient.Do(request)
	if err != nil {
		return nil, "", err
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, "", fmt.Errorf("logo status code %d", response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, 3*1024*1024))
	if err != nil {
		return nil, "", err
	}
	if len(body) == 0 {
		return nil, "", fmt.Errorf("logo body is empty")
	}

	return body, resolveBatchManifestImageType(logoURL), nil
}

// resolveBatchManifestImageType resolves gofpdf image type values from URL extensions.
func resolveBatchManifestImageType(rawURL string) string {
	ext := strings.ToLower(strings.TrimPrefix(path.Ext(strings.TrimSpace(rawURL)), "."))
	switch ext {
	case "jpg":
		return "JPEG"
	case "jpeg":
		return "JPEG"
	case "gif":
		return "GIF"
	default:
		return "PNG"
	}
}

// truncateBatchManifestValue truncates long table values for compact cover rows.
func truncateBatchManifestValue(value string, maxLength int, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}
	if maxLength <= 0 || len(trimmed) <= maxLength {
		return trimmed
	}
	if maxLength <= 1 {
		return trimmed[:1]
	}
	if maxLength <= 3 {
		return trimmed[:maxLength]
	}
	return trimmed[:maxLength-3] + "..."
}

// sanitizeBatchManifestValue normalizes empty metadata values.
func sanitizeBatchManifestValue(value string, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}
	return trimmed
}

// maxBatchManifestQuantity normalizes negative row-count values.
func maxBatchManifestQuantity(value int) int {
	if value < 0 {
		return 0
	}
	return value
}

// resolveBatchManifestCoverTemplate resolves active cover template values with default fallback.
func (s *Service) resolveBatchManifestCoverTemplate() batchManifestCoverTemplate {
	if s == nil || s.manifestDocuments == nil {
		return loadDefaultBatchManifestCoverTemplate()
	}
	template := s.manifestDocuments.coverTemplate
	if err := template.validate(); err != nil {
		return loadDefaultBatchManifestCoverTemplate()
	}

	return template
}

// formatBatchManifestItemsAsList converts row items to a compact unordered-list text block.
func formatBatchManifestItemsAsList(items []string, bulletPrefix string, fallback string) string {
	normalizedItems := normalizeBatchManifestItems(items)
	if len(normalizedItems) == 0 {
		return fallback
	}
	trimmedPrefix := strings.TrimSpace(bulletPrefix)
	if trimmedPrefix == "" {
		trimmedPrefix = "-"
	}
	rows := make([]string, 0, len(normalizedItems))
	for _, item := range normalizedItems {
		rows = append(rows, trimmedPrefix+" "+item)
	}

	return strings.Join(rows, "\n")
}

// batchManifestTableRowHeight resolves dynamic row height values based on rendered items.
func (s *Service) batchManifestTableRowHeight(pdf *gofpdf.Fpdf, row batchManifestCoverRow) float64 {
	if pdf == nil {
		return batchManifestRowHeightMM
	}
	template := s.resolveBatchManifestCoverTemplate()
	itemText := encodeBatchManifestText(formatBatchManifestItemsAsList(row.Items, template.ItemBulletPrefix, template.EmptyValueFallback))
	pdf.SetFont("Arial", "", 8)
	lineCount := len(pdf.SplitLines([]byte(itemText), batchManifestTableColumnWidths[5]))
	if lineCount < 1 {
		lineCount = 1
	}
	height := float64(lineCount) * batchManifestItemsLineHeightMM
	if height < batchManifestRowHeightMM {
		return batchManifestRowHeightMM
	}

	return height
}

// sumBatchManifestColumnWidths sums column widths up to endColumn index.
func sumBatchManifestColumnWidths(endColumn int) float64 {
	if endColumn <= 0 {
		return 0
	}
	if endColumn > len(batchManifestTableColumnWidths) {
		endColumn = len(batchManifestTableColumnWidths)
	}
	total := 0.0
	for i := 0; i < endColumn; i++ {
		total += batchManifestTableColumnWidths[i]
	}

	return total
}

// encodeBatchManifestText encodes UTF-8 text to Windows-1252 for core PDF fonts.
func encodeBatchManifestText(value string) string {
	encoded, err := charmap.Windows1252.NewEncoder().String(value)
	if err != nil {
		return value
	}

	return encoded
}
