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
)

var (
	// batchManifestTableHeaders defines rendered summary table headers.
	batchManifestTableHeaders = []string{"TRACKING NUMBER", "RECIPIENT", "ORDER #", "CITY", "ITEMS"}
	// batchManifestTableColumnWidths defines rendered summary table column widths.
	batchManifestTableColumnWidths = []float64{32, 44, 24, 22, 73}
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
		if pdf.GetY()+batchManifestRowHeightMM > batchManifestPageHeightMM-batchManifestMarginMM {
			pdf.AddPage()
			s.drawBatchManifestTableHeader(pdf)
		}
		s.drawBatchManifestTableRow(pdf, row)
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
	logoY := batchManifestMarginMM
	logoX := batchManifestMarginMM
	logoWidth := 28.0

	if logoBytes, imageType, err := s.downloadBatchManifestLogo(ctx); err == nil && len(logoBytes) > 0 {
		imageName := "batch_manifest_logo"
		opts := gofpdf.ImageOptions{ImageType: imageType, ReadDpi: true}
		pdf.RegisterImageOptionsReader(imageName, opts, bytes.NewReader(logoBytes))
		pdf.ImageOptions(imageName, logoX, logoY, logoWidth, 0, false, opts, 0, "")
	}

	headerX := logoX + logoWidth + 6
	pdf.SetXY(headerX, batchManifestMarginMM)
	pdf.SetFont("Arial", "B", 11)
	pdf.CellFormat(130, 6, "MANIFEST BATCH SUMMARY", "", 1, "L", false, 0, "")

	pdf.SetX(headerX)
	pdf.SetFont("Arial", "", 9)
	pdf.CellFormat(130, 5, fmt.Sprintf("Batch ID: %s", sanitizeBatchManifestValue(meta.BatchID, "-")), "", 1, "L", false, 0, "")
	pdf.SetX(headerX)
	pdf.CellFormat(130, 5, fmt.Sprintf("Generated: %s", meta.GeneratedAt.UTC().Format("2006-01-02 15:04:05 UTC")), "", 1, "L", false, 0, "")
	pdf.SetX(headerX)
	pdf.CellFormat(130, 5, fmt.Sprintf("Carrier: %s", sanitizeBatchManifestValue(meta.CarrierID, "-")), "", 1, "L", false, 0, "")
	pdf.SetX(headerX)
	pdf.CellFormat(130, 5, fmt.Sprintf("Quantity: %d", maxBatchManifestQuantity(meta.Quantity)), "", 1, "L", false, 0, "")

	pdf.SetY(42)
}

// drawBatchManifestTableHeader draws one summary table header row.
func (s *Service) drawBatchManifestTableHeader(pdf *gofpdf.Fpdf) {
	if pdf == nil {
		return
	}
	pdf.SetFont("Arial", "B", 8)
	pdf.SetFillColor(240, 240, 240)
	for index, header := range batchManifestTableHeaders {
		pdf.CellFormat(batchManifestTableColumnWidths[index], batchManifestRowHeightMM, header, "1", 0, "L", true, 0, "")
	}
	pdf.Ln(-1)
}

// drawBatchManifestTableRow draws one summary table row.
func (s *Service) drawBatchManifestTableRow(pdf *gofpdf.Fpdf, row batchManifestCoverRow) {
	if pdf == nil {
		return
	}
	pdf.SetFont("Arial", "", 8)
	pdf.CellFormat(batchManifestTableColumnWidths[0], batchManifestRowHeightMM, truncateBatchManifestValue(row.TrackingNumber, 24), "1", 0, "L", false, 0, "")
	pdf.CellFormat(batchManifestTableColumnWidths[1], batchManifestRowHeightMM, truncateBatchManifestValue(row.RecipientName, 30), "1", 0, "L", false, 0, "")
	pdf.CellFormat(batchManifestTableColumnWidths[2], batchManifestRowHeightMM, truncateBatchManifestValue(row.OrderNumber, 16), "1", 0, "L", false, 0, "")
	pdf.CellFormat(batchManifestTableColumnWidths[3], batchManifestRowHeightMM, truncateBatchManifestValue(row.City, 14), "1", 0, "L", false, 0, "")
	pdf.CellFormat(batchManifestTableColumnWidths[4], batchManifestRowHeightMM, truncateBatchManifestValue(row.Items, 58), "1", 0, "L", false, 0, "")
	pdf.Ln(-1)
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
func truncateBatchManifestValue(value string, maxLength int) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "-"
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
