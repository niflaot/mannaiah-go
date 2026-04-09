package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/jung-kurt/gofpdf"
	"github.com/skip2/go-qrcode"
	"golang.org/x/text/encoding/charmap"
	"mannaiah/module/shipping/domain"
)

const (
	// rotulusPageWidthMM defines half-letter landscape width values.
	rotulusPageWidthMM = 215.9
	// rotulusPageHeightMM defines half-letter landscape height values.
	rotulusPageHeightMM = 139.7
	// rotulusMarginMM defines page margin values.
	rotulusMarginMM = 10.0
	// rotulusColumnGapMM defines gap values between left/right columns.
	rotulusColumnGapMM = 8.0
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
		OrientationStr: "L",
	})
	pdf.SetMargins(rotulusMarginMM, rotulusMarginMM, rotulusMarginMM)
	pdf.SetAutoPageBreak(false, rotulusMarginMM)
	pdf.AddPage()

	s.drawRotulusLeftColumn(ctx, pdf, meta)
	if err := s.drawRotulusRightColumn(pdf, meta); err != nil {
		return nil, err
	}
	s.drawRotulusFooter(pdf, meta)

	var output bytes.Buffer
	if err := pdf.Output(&output); err != nil {
		return nil, err
	}

	return output.Bytes(), nil
}

// drawRotulusLeftColumn renders logo + mark information.
func (s *Service) drawRotulusLeftColumn(ctx context.Context, pdf *gofpdf.Fpdf, meta markRotulusMeta) {
	if pdf == nil || s == nil || s.rotulusDocuments == nil {
		return
	}
	template := s.rotulusDocuments.template
	leftWidth := (rotulusPageWidthMM - (2 * rotulusMarginMM) - rotulusColumnGapMM) / 2
	logoY := rotulusMarginMM
	if logoBytes, imageType, err := s.downloadRotulusLogo(ctx); err == nil && len(logoBytes) > 0 {
		imageName := "rotulus-logo"
		opts := gofpdf.ImageOptions{ImageType: imageType, ReadDpi: true}
		pdf.RegisterImageOptionsReader(imageName, opts, bytes.NewReader(logoBytes))
		pdf.ImageOptions(imageName, rotulusMarginMM, logoY, 34, 0, false, opts, 0, "")
	}

	textX := rotulusMarginMM
	textY := rotulusMarginMM + 30
	pdf.SetXY(textX, textY)
	pdf.SetFont("Arial", "B", 16)
	pdf.CellFormat(leftWidth, 7, encodeRotulusText(template.Title), "", 1, "L", false, 0, "")

	rows := []struct {
		label string
		value string
	}{
		{label: template.OrderLabel, value: sanitizeRotulusValue(meta.OrderNumber, template.EmptyValueFallback)},
		{label: template.TrackingLabel, value: sanitizeRotulusValue(meta.TrackingNumber, template.EmptyValueFallback)},
		{label: template.CarrierLabel, value: sanitizeRotulusValue(meta.CarrierLabel, template.EmptyValueFallback)},
		{label: template.RecipientLabel, value: sanitizeRotulusValue(meta.RecipientName, template.EmptyValueFallback)},
		{label: template.GeneratedLabel, value: meta.GeneratedAt.UTC().Format("2006-01-02 15:04:05 UTC")},
	}

	pdf.SetFont("Arial", "", 11)
	for _, row := range rows {
		pdf.SetX(textX)
		pdf.CellFormat(leftWidth, 7, encodeRotulusText(fmt.Sprintf("%s: %s", row.label, row.value)), "", 1, "L", false, 0, "")
	}
}

// drawRotulusRightColumn renders the centered signed QR code payload.
func (s *Service) drawRotulusRightColumn(pdf *gofpdf.Fpdf, meta markRotulusMeta) error {
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
	qrSize := 58.0
	qrX := rightX + (rightWidth-qrSize)/2
	qrY := (rotulusPageHeightMM - qrSize) / 2
	opts := gofpdf.ImageOptions{ImageType: "PNG", ReadDpi: true}
	pdf.RegisterImageOptionsReader("rotulus-qr", opts, bytes.NewReader(qrBytes))
	pdf.ImageOptions("rotulus-qr", qrX, qrY, qrSize, qrSize, false, opts, 0, "")

	return nil
}

// drawRotulusFooter renders the generation timestamp footer.
func (s *Service) drawRotulusFooter(pdf *gofpdf.Fpdf, meta markRotulusMeta) {
	if pdf == nil || s == nil || s.rotulusDocuments == nil {
		return
	}
	template := s.rotulusDocuments.template
	pdf.SetFont("Arial", "", 8)
	pdf.SetXY(rotulusMarginMM, rotulusPageHeightMM-rotulusMarginMM-4)
	pdf.CellFormat(rotulusPageWidthMM-(2*rotulusMarginMM), 4, encodeRotulusText(fmt.Sprintf("%s: %s", template.FooterLabel, meta.GeneratedAt.UTC().Format("2006-01-02 15:04:05 UTC"))), "", 0, "L", false, 0, "")
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

// encodeRotulusText encodes UTF-8 strings into ISO-8859-1 compatible PDF text.
func encodeRotulusText(value string) string {
	encoded, err := charmap.Windows1252.NewEncoder().String(value)
	if err != nil {
		return value
	}

	return encoded
}
