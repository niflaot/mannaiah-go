package service

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/jung-kurt/gofpdf"

	"mannaiah/module/shipping/domain"
)

// BatchAllRotulusDocument builds one PDF with all rotulus for marks in a batch, two per page.
func (s *Service) BatchAllRotulusDocument(ctx context.Context, batchID string) ([]byte, error) {
	if s == nil || s.repository == nil || s.rotulusDocuments == nil {
		return nil, domain.ErrInvalidID
	}
	trimmedBatchID := strings.TrimSpace(batchID)
	if trimmedBatchID == "" {
		return nil, domain.ErrInvalidID
	}

	marks, err := s.repository.ListByBatchID(ctx, trimmedBatchID)
	if err != nil {
		return nil, err
	}

	var included []domain.ShippingMark
	for _, mark := range marks {
		if mark.Status != domain.MarkStatusFailed {
			included = append(included, mark)
		}
	}
	if len(included) == 0 {
		return nil, domain.ErrNotFound
	}

	metas := make([]markRotulusMeta, 0, len(included))
	for i := range included {
		metas = append(metas, s.buildRotulusMetaForMark(ctx, &included[i]))
	}

	pdf := gofpdf.NewCustom(&gofpdf.InitType{
		UnitStr:        "mm",
		Size:           gofpdf.SizeType{Wd: rotulusPageWidthMM, Ht: rotulusPageHeightMM},
		OrientationStr: "P",
	})
	pdf.SetMargins(rotulusMarginMM, rotulusMarginMM, rotulusMarginMM)
	pdf.SetAutoPageBreak(false, 0)
	pdf.SetCompression(false)

	for i := 0; i < len(metas); i += 2 {
		pdf.AddPage()
		drawRotulusDivider(pdf, rotulusContentHeightMM)

		qrName0 := fmt.Sprintf("rotulus-qr-%d", i)
		if drawErr := s.drawRotulusOnPage(ctx, pdf, metas[i], 0, qrName0); drawErr != nil {
			return nil, drawErr
		}

		if i+1 < len(metas) {
			qrName1 := fmt.Sprintf("rotulus-qr-%d", i+1)
			if drawErr := s.drawRotulusOnPage(ctx, pdf, metas[i+1], rotulusContentHeightMM, qrName1); drawErr != nil {
				return nil, drawErr
			}
		}
	}

	var out bytes.Buffer
	if err := pdf.Output(&out); err != nil {
		return nil, err
	}

	return out.Bytes(), nil
}
