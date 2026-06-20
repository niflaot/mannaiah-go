package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/jung-kurt/gofpdf"
	"github.com/skip2/go-qrcode"
	"golang.org/x/text/encoding/charmap"
	"mannaiah/module/shipping/domain"
)

const (
	// rotulusPageWidthMM defines letter portrait width values.
	rotulusPageWidthMM = 215.9
	// rotulusPageHeightMM defines letter portrait height values.
	rotulusPageHeightMM = 279.4
	// rotulusContentHeightMM defines the top-half printable rotulus height.
	rotulusContentHeightMM = 139.7
	// rotulusMarginMM defines page margin values.
	rotulusMarginMM = 10.0
	// rotulusColumnGapMM defines gap values between left/right columns.
	rotulusColumnGapMM = 8.0
	// rotulusLogoWidthMM defines rendered logo width values.
	rotulusLogoWidthMM = 46.0
	// rotulusQRSizeMM defines signed QR code size values.
	rotulusQRSizeMM = 42.0
	// rotulusMaxLogoBytes defines maximum logo download size.
	rotulusMaxLogoBytes = 5 * 1024 * 1024
)

// buildRotulusPDF renders one PDF rotulus payload for the provided mark meta.
func (s *Service) buildRotulusPDF(ctx context.Context, meta markRotulusMeta) ([]byte, error) {
	if s == nil || s.rotulusDocuments == nil {
		return nil, domain.ErrInvalidID
	}
	pdf := gofpdf.NewCustom(&gofpdf.InitType{
		UnitStr: "mm",
		Size: gofpdf.SizeType{
			Wd: rotulusPageWidthMM,
			Ht: rotulusPageHeightMM,
		},
		OrientationStr: "P",
	})
	pdf.SetMargins(rotulusMarginMM, rotulusMarginMM, rotulusMarginMM)
	pdf.SetAutoPageBreak(false, rotulusMarginMM)
	pdf.SetCompression(false)
	pdf.AddPage()

	if err := s.drawRotulusOnPage(ctx, pdf, meta, 0, "rotulus-qr"); err != nil {
		return nil, err
	}

	var output bytes.Buffer
	if err := pdf.Output(&output); err != nil {
		return nil, err
	}

	return output.Bytes(), nil
}

// drawRotulusOnPage renders one rotulus at the given vertical offset within an existing PDF page.
func (s *Service) drawRotulusOnPage(ctx context.Context, pdf *gofpdf.Fpdf, meta markRotulusMeta, yOffset float64, qrImageName string) error {
	if s == nil || s.rotulusDocuments == nil || pdf == nil {
		return domain.ErrInvalidID
	}
	s.drawRotulusLeftColumn(ctx, pdf, meta, yOffset)
	if err := s.drawRotulusRightColumn(pdf, meta, yOffset, qrImageName); err != nil {
		return err
	}
	s.drawRotulusFooter(pdf, meta, yOffset)
	return nil
}

// drawRotulusDivider renders a dashed separator line at the given vertical position.
func drawRotulusDivider(pdf *gofpdf.Fpdf, y float64) {
	if pdf == nil {
		return
	}
	pdf.SetDrawColor(180, 180, 180)
	pdf.SetLineWidth(0.3)
	pdf.SetDashPattern([]float64{3, 2}, 0)
	pdf.Line(rotulusMarginMM, y, rotulusPageWidthMM-rotulusMarginMM, y)
	pdf.SetDashPattern([]float64{}, 0)
	pdf.SetLineWidth(0.2)
	pdf.SetDrawColor(0, 0, 0)
}

// drawRotulusLeftColumn renders logo + mark information at the given vertical offset.
func (s *Service) drawRotulusLeftColumn(ctx context.Context, pdf *gofpdf.Fpdf, meta markRotulusMeta, yOffset float64) {
	if pdf == nil || s == nil || s.rotulusDocuments == nil {
		return
	}
	template := s.rotulusDocuments.template
	leftWidth := (rotulusPageWidthMM - (2 * rotulusMarginMM) - rotulusColumnGapMM) / 2
	logoY := rotulusMarginMM + yOffset
	if logoBytes, imageType, err := s.downloadRotulusLogo(ctx); err == nil && len(logoBytes) > 0 {
		imageName := "rotulus-logo"
		opts := gofpdf.ImageOptions{ImageType: imageType, ReadDpi: true}
		pdf.RegisterImageOptionsReader(imageName, opts, bytes.NewReader(logoBytes))
		pdf.ImageOptions(imageName, rotulusMarginMM, logoY, rotulusLogoWidthMM, 0, false, opts, 0, "")
	}

	textX := rotulusMarginMM
	textY := rotulusMarginMM + 32 + yOffset
	pdf.SetXY(textX, textY)
	pdf.SetFont("Arial", "B", 18)
	pdf.CellFormat(leftWidth, 8, encodeRotulusText(strings.ToUpper(resolveRotulusTitle(template, meta))), "", 1, "L", false, 0, "")

	rows := []struct {
		label string
		value string
	}{
		{label: template.CarrierLabel, value: sanitizeRotulusValue(meta.CarrierLabel, template.EmptyValueFallback)},
		{label: template.RecipientLabel, value: sanitizeRotulusValue(meta.RecipientName, template.EmptyValueFallback)},
		{label: template.AddressLabel, value: sanitizeRotulusValue(meta.RecipientAddressLine, template.EmptyValueFallback)},
		{label: template.Address2Label, value: sanitizeRotulusValue(meta.RecipientAddressLine2, template.EmptyValueFallback)},
		{label: template.PhoneLabel, value: sanitizeRotulusValue(meta.RecipientPhone, template.EmptyValueFallback)},
		{label: template.CityLabel, value: sanitizeRotulusValue(meta.RecipientCity, template.EmptyValueFallback)},
	}

	pdf.Ln(2)
	for _, row := range rows {
		drawRotulusLabelValueRow(pdf, textX, leftWidth, formatRotulusLabel(row.label), row.value)
	}
	drawRotulusCollectOnDelivery(pdf, textX, leftWidth, meta.CollectOnDeliveryAmount)
	drawRotulusContent(pdf, textX, leftWidth, meta.Content, template.EmptyValueFallback)
}

// drawRotulusRightColumn renders the centered signed QR code payload at the given vertical offset.
func (s *Service) drawRotulusRightColumn(pdf *gofpdf.Fpdf, meta markRotulusMeta, yOffset float64, qrImageName string) error {
	if pdf == nil || s == nil || s.rotulusDocuments == nil {
		return nil
	}
	token, err := s.buildSignedRotulusQRToken(meta)
	if err != nil {
		return err
	}
	qrBytes, err := qrcode.Encode(token, qrcode.Medium, 320)
	if err != nil {
		return err
	}

	rightX := rotulusMarginMM + ((rotulusPageWidthMM - (2 * rotulusMarginMM) - rotulusColumnGapMM) / 2) + rotulusColumnGapMM
	rightWidth := (rotulusPageWidthMM - (2 * rotulusMarginMM) - rotulusColumnGapMM) / 2
	qrSize := rotulusQRSizeMM
	qrX := rightX + (rightWidth-qrSize)/2
	qrY := rotulusMarginMM + ((rotulusContentHeightMM - (2 * rotulusMarginMM) - qrSize) / 2) + yOffset
	opts := gofpdf.ImageOptions{ImageType: "PNG", ReadDpi: true}
	pdf.RegisterImageOptionsReader(qrImageName, opts, bytes.NewReader(qrBytes))
	pdf.ImageOptions(qrImageName, qrX, qrY, qrSize, qrSize, false, opts, 0, "")

	return nil
}

// drawRotulusFooter renders the generation timestamp footer at the given vertical offset.
func (s *Service) drawRotulusFooter(pdf *gofpdf.Fpdf, meta markRotulusMeta, yOffset float64) {
	if pdf == nil || s == nil || s.rotulusDocuments == nil {
		return
	}
	template := s.rotulusDocuments.template
	footerY := rotulusContentHeightMM - rotulusMarginMM - 4 + yOffset
	pdf.SetXY(rotulusMarginMM, footerY)
	drawRotulusInlineLabelValue(
		pdf,
		rotulusMarginMM,
		footerY,
		rotulusPageWidthMM-(2*rotulusMarginMM),
		4,
		8,
		formatRotulusLabel(template.FooterLabel),
		domain.FormatUTCMinusFiveTimestamp(meta.GeneratedAt),
	)
}

// resolveRotulusTitle resolves one dynamic order title line.
func resolveRotulusTitle(template markRotulusTemplate, meta markRotulusMeta) string {
	return strings.TrimSpace(firstNonEmptyString(template.OrderTitlePrefix, "Pedido #")) + sanitizeRotulusValue(meta.OrderNumber, template.EmptyValueFallback)
}

// formatRotulusLabel normalizes one label prefix for uppercase bold rendering.
func formatRotulusLabel(label string) string {
	trimmed := strings.TrimSpace(label)
	if trimmed == "" {
		return ""
	}

	return strings.ToUpper(trimmed) + ":"
}

// drawRotulusLabelValueRow renders one two-style label/value row with a bold uppercase prefix.
func drawRotulusLabelValueRow(pdf *gofpdf.Fpdf, x float64, width float64, label string, value string) {
	if pdf == nil {
		return
	}
	drawRotulusInlineLabelValue(pdf, x, pdf.GetY(), width, 5.5, 10, label, value)
	pdf.Ln(0.7)
}

// drawRotulusInlineLabelValue renders a bold prefix and normal value on the same baseline.
func drawRotulusInlineLabelValue(pdf *gofpdf.Fpdf, x float64, y float64, width float64, lineHeight float64, fontSize float64, label string, value string) {
	if pdf == nil {
		return
	}
	labelText := encodeRotulusText(strings.TrimSpace(label))
	valueText := encodeRotulusText(strings.TrimSpace(value))

	pdf.SetXY(x, y)
	pdf.SetFont("Arial", "B", fontSize)
	pdf.CellFormat(0, lineHeight, labelText, "", 0, "L", false, 0, "")

	labelWidth := pdf.GetStringWidth(labelText + " ")
	pdf.SetXY(x+labelWidth, y)
	pdf.SetFont("Arial", "", fontSize)
	pdf.MultiCell(width-labelWidth, lineHeight, valueText, "", "L", false)
}

// drawRotulusCollectOnDelivery renders highlighted COD collection amount when present.
func drawRotulusCollectOnDelivery(pdf *gofpdf.Fpdf, x float64, width float64, value float64) {
	if pdf == nil || value <= 0 {
		return
	}
	pdf.Ln(1.5)
	pdf.SetX(x)
	pdf.SetFont("Arial", "B", 16)
	pdf.CellFormat(width, 9, encodeRotulusText("RECAUDO: "+formatRotulusCOP(value)), "", 1, "L", false, 0, "")
}

// drawRotulusContent renders the content line directly in the detail stack (after city and optional COD).
func drawRotulusContent(pdf *gofpdf.Fpdf, x float64, width float64, content string, fallback string) {
	if pdf == nil {
		return
	}
	label := formatRotulusLabel("Contenido")
	lines := splitRotulusContentLines(sanitizeRotulusValue(content, fallback))
	if !rotulusContentLinesHaveQuantity(lines) {
		drawRotulusLabelValueRow(pdf, x, width, label, strings.Join(lines, "\n"))
		return
	}
	y := pdf.GetY()
	labelText := encodeRotulusText(label + " ")
	pdf.SetXY(x, y)
	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(0, 5.5, labelText, "", 0, "L", false, 0, "")
	labelWidth := pdf.GetStringWidth(labelText)
	nextY := drawRotulusContentItemLines(pdf, x+labelWidth, y, width-labelWidth, lines, true)
	if nextY <= y {
		nextY = y + 5.5
	}
	pdf.SetY(nextY + 0.7)
}

// rotulusContentLinesHaveQuantity reports whether any content row has an Xn quantity prefix.
func rotulusContentLinesHaveQuantity(lines []string) bool {
	for _, line := range lines {
		_, item := splitRotulusBullet(line)
		if _, _, ok := splitRotulusQuantityPrefix(item); ok {
			return true
		}
	}

	return false
}

// splitRotulusContentLines normalizes rendered rotulus content rows.
func splitRotulusContentLines(content string) []string {
	rows := strings.Split(strings.TrimSpace(content), "\n")
	normalized := make([]string, 0, len(rows))
	for _, row := range rows {
		trimmed := strings.TrimSpace(row)
		if trimmed == "" {
			continue
		}
		normalized = append(normalized, trimmed)
	}
	if len(normalized) == 0 {
		return []string{"-"}
	}

	return normalized
}

// drawRotulusContentItemLines draws content rows with bold quantity prefixes.
func drawRotulusContentItemLines(pdf *gofpdf.Fpdf, x float64, y float64, width float64, lines []string, firstLineInline bool) float64 {
	currentY := y
	for index, line := range lines {
		lineX := x
		if index > 0 || !firstLineInline {
			lineX = x
		}
		nextY := drawRotulusContentItemLine(pdf, lineX, currentY, width, line)
		if nextY <= currentY {
			nextY = currentY + 5.5
		}
		currentY = nextY
	}

	return currentY
}

// drawRotulusContentItemLine draws one rotulus content row with an optional bold quantity prefix.
func drawRotulusContentItemLine(pdf *gofpdf.Fpdf, x float64, y float64, width float64, line string) float64 {
	if pdf == nil {
		return y
	}
	bullet, item := splitRotulusBullet(strings.TrimSpace(line))
	quantity, rest, hasQuantity := splitRotulusQuantityPrefix(item)
	pdf.SetXY(x, y)
	pdf.SetFont("Arial", "", 10)
	if bullet != "" {
		bulletText := encodeRotulusText(bullet + " ")
		pdf.CellFormat(pdf.GetStringWidth(bulletText), 5.5, bulletText, "", 0, "L", false, 0, "")
	}
	remainingX := pdf.GetX()
	if hasQuantity {
		pdf.SetFont("Arial", "B", 10)
		quantityText := encodeRotulusText(quantity + " ")
		pdf.CellFormat(pdf.GetStringWidth(quantityText), 5.5, quantityText, "", 0, "L", false, 0, "")
		remainingX = pdf.GetX()
	}
	value := strings.TrimSpace(item)
	if hasQuantity {
		value = rest
	}
	pdf.SetXY(remainingX, y)
	pdf.SetFont("Arial", "", 10)
	pdf.MultiCell(math.Max(width-(remainingX-x), 4), 5.5, encodeRotulusText(value), "", "L", false)

	return pdf.GetY()
}

// splitRotulusBullet separates list bullet prefixes from content labels.
func splitRotulusBullet(line string) (string, string) {
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "- ") {
		return "-", strings.TrimSpace(strings.TrimPrefix(trimmed, "-"))
	}

	return "", trimmed
}

// splitRotulusQuantityPrefix splits labels like "X2 Product" into quantity and item text.
func splitRotulusQuantityPrefix(item string) (string, string, bool) {
	fields := strings.Fields(strings.TrimSpace(item))
	if len(fields) == 0 {
		return "", "", false
	}
	token := strings.ToUpper(fields[0])
	if len(token) < 2 || token[0] != 'X' {
		return "", strings.TrimSpace(item), false
	}
	for _, digit := range token[1:] {
		if digit < '0' || digit > '9' {
			return "", strings.TrimSpace(item), false
		}
	}

	return token, strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(item), fields[0])), true
}

// downloadRotulusLogo downloads one logo image and resolves image type values for gofpdf.
func (s *Service) downloadRotulusLogo(ctx context.Context) ([]byte, string, error) {
	if s == nil || s.rotulusDocuments == nil {
		return nil, "", domain.ErrInvalidID
	}
	logoURL := strings.TrimSpace(s.rotulusDocuments.logoURL)
	if logoURL == "" {
		return nil, "", domain.ErrInvalidID
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, logoURL, nil)
	if err != nil {
		return nil, "", err
	}
	response, err := s.rotulusDocuments.httpClient.Do(request)
	if err != nil {
		return nil, "", err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, "", fmt.Errorf("download rotulus logo: status=%d", response.StatusCode)
	}
	payload, err := io.ReadAll(io.LimitReader(response.Body, rotulusMaxLogoBytes))
	if err != nil {
		return nil, "", err
	}
	if len(payload) == 0 {
		return nil, "", fmt.Errorf("download rotulus logo: empty body")
	}

	return payload, resolveRotulusImageType(logoURL), nil
}

// resolveRotulusImageType resolves gofpdf image types from URL extensions.
func resolveRotulusImageType(rawURL string) string {
	parsedURL, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return "PNG"
	}
	path := strings.ToLower(strings.TrimSpace(parsedURL.Path))
	switch {
	case strings.HasSuffix(path, ".jpg"), strings.HasSuffix(path, ".jpeg"):
		return "JPG"
	case strings.HasSuffix(path, ".gif"):
		return "GIF"
	default:
		return "PNG"
	}
}

// sanitizeRotulusValue resolves fallback values for empty rotulus fields.
func sanitizeRotulusValue(value string, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return strings.TrimSpace(fallback)
	}

	return trimmed
}

// formatRotulusCOP formats whole monetary values using Colombian thousands separators.
func formatRotulusCOP(value float64) string {
	if value <= 0 {
		return "$0"
	}
	rounded := int64(math.Round(value))
	raw := strconv.FormatInt(rounded, 10)
	if len(raw) <= 3 {
		return "$" + raw
	}

	var builder strings.Builder
	remainder := len(raw) % 3
	if remainder > 0 {
		builder.WriteString(raw[:remainder])
		if len(raw) > remainder {
			builder.WriteByte('.')
		}
	}

	for index := remainder; index < len(raw); index += 3 {
		builder.WriteString(raw[index : index+3])
		if index+3 < len(raw) {
			builder.WriteByte('.')
		}
	}

	return "$" + builder.String()
}

// encodeRotulusText encodes UTF-8 strings into ISO-8859-1 compatible PDF text.
func encodeRotulusText(value string) string {
	encoded, err := charmap.Windows1252.NewEncoder().String(value)
	if err != nil {
		return value
	}

	return encoded
}
