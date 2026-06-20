package service

import (
	"bytes"
	"context"
	"fmt"

	"github.com/jung-kurt/gofpdf"

	"mannaiah/module/shipping/domain"
)

const (
	// batchChecklistPageWidthMM defines letter-size page width values in millimeters.
	batchChecklistPageWidthMM = 215.9
	// batchChecklistPageHeightMM defines letter-size page height values in millimeters.
	batchChecklistPageHeightMM = 279.4
	// batchChecklistMarginMM defines uniform page margin values in millimeters.
	batchChecklistMarginMM = 10.0
	// batchChecklistRowHeightMM defines table row-height values in millimeters.
	batchChecklistRowHeightMM = 7.0
	// batchChecklistItemsLineHeightMM defines items line-height values in millimeters.
	batchChecklistItemsLineHeightMM = 4.0
)

var (
	// batchChecklistTableColumnWidths defines rendered checklist table column widths.
	// Columns: OrderNumber, RecipientName, City, Items, CheckOne, CheckTwo, CheckThree.
	batchChecklistTableColumnWidths = []float64{24, 48, 28, 55, 13, 13, 13}
)

// buildBatchChecklistPDF creates one checklist PDF page in letter format.
func (s *Service) buildBatchChecklistPDF(ctx context.Context, meta batchChecklistMeta, rows []batchChecklistRow) ([]byte, error) {
	pdf := gofpdf.NewCustom(&gofpdf.InitType{
		UnitStr: "mm",
		Size: gofpdf.SizeType{
			Wd: batchChecklistPageWidthMM,
			Ht: batchChecklistPageHeightMM,
		},
	})
	pdf.SetMargins(batchChecklistMarginMM, batchChecklistMarginMM, batchChecklistMarginMM)
	pdf.SetAutoPageBreak(false, batchChecklistMarginMM)
	pdf.AddPage()

	s.drawBatchChecklistHeader(ctx, pdf, meta)
	s.drawBatchChecklistTableHeader(pdf)

	for _, row := range rows {
		rowHeight := batchChecklistTableRowHeight(pdf, row)
		if pdf.GetY()+rowHeight > batchChecklistPageHeightMM-batchChecklistMarginMM {
			pdf.AddPage()
			s.drawBatchChecklistTableHeader(pdf)
		}
		s.drawBatchChecklistTableRow(pdf, row, rowHeight)
	}

	var output bytes.Buffer
	if err := pdf.Output(&output); err != nil {
		return nil, err
	}

	return append([]byte(nil), output.Bytes()...), nil
}

// drawBatchChecklistHeader draws checklist header values and optional logo image.
func (s *Service) drawBatchChecklistHeader(ctx context.Context, pdf *gofpdf.Fpdf, meta batchChecklistMeta) {
	if pdf == nil {
		return
	}
	logoY := batchChecklistMarginMM
	logoWidth := 28.0
	logoX := batchChecklistPageWidthMM - batchChecklistMarginMM - logoWidth

	if logoBytes, imageType, err := s.downloadBatchManifestLogo(ctx); err == nil && len(logoBytes) > 0 {
		imageName := "batch_checklist_logo"
		opts := gofpdf.ImageOptions{ImageType: imageType, ReadDpi: true}
		pdf.RegisterImageOptionsReader(imageName, opts, bytes.NewReader(logoBytes))
		pdf.ImageOptions(imageName, logoX, logoY, logoWidth, 0, false, opts, 0, "")
	}

	headerX := batchChecklistMarginMM
	headerWidth := logoX - headerX - 6
	pdf.SetXY(headerX, batchChecklistMarginMM)
	pdf.SetFont("Arial", "B", 11)
	pdf.CellFormat(headerWidth, 6, encodeBatchManifestText("LISTA DE CHEQUEO DE LOTE"), "", 1, "L", false, 0, "")

	pdf.SetX(headerX)
	pdf.SetFont("Arial", "", 9)
	pdf.CellFormat(headerWidth, 5, encodeBatchManifestText(fmt.Sprintf("Lote: %s", sanitizeBatchManifestValue(meta.BatchID, "-"))), "", 1, "L", false, 0, "")
	pdf.SetX(headerX)
	pdf.CellFormat(headerWidth, 5, encodeBatchManifestText(fmt.Sprintf("Generado: %s", domain.FormatUTCMinusFiveTimestamp(meta.GeneratedAt))), "", 1, "L", false, 0, "")
	pdf.SetX(headerX)
	pdf.CellFormat(headerWidth, 5, encodeBatchManifestText(fmt.Sprintf("Transportadora: %s", sanitizeBatchManifestValue(meta.CarrierID, "-"))), "", 1, "L", false, 0, "")
	pdf.SetX(headerX)
	pdf.CellFormat(headerWidth, 5, encodeBatchManifestText(fmt.Sprintf("Cantidad: %d", maxBatchManifestQuantity(meta.Quantity))), "", 1, "L", false, 0, "")

	pdf.SetY(42)
}

// drawBatchChecklistTableHeader draws one checklist table header row.
func (s *Service) drawBatchChecklistTableHeader(pdf *gofpdf.Fpdf) {
	if pdf == nil {
		return
	}
	headers := []string{"PEDIDO", "DESTINATARIO", "CIUDAD", "ARTÍCULOS", "1", "2", "3"}
	pdf.SetFont("Arial", "B", 8)
	pdf.SetFillColor(240, 240, 240)
	for index, header := range headers {
		align := "L"
		if index >= 4 {
			align = "C"
		}
		pdf.CellFormat(batchChecklistTableColumnWidths[index], batchChecklistRowHeightMM, encodeBatchManifestText(header), "1", 0, align, true, 0, "")
	}
	pdf.Ln(-1)
}

// drawBatchChecklistTableRow draws one checklist table row.
func (s *Service) drawBatchChecklistTableRow(pdf *gofpdf.Fpdf, row batchChecklistRow, rowHeight float64) {
	if pdf == nil {
		return
	}
	startX, startY := pdf.GetX(), pdf.GetY()

	pdf.SetFont("Arial", "", 8)
	pdf.CellFormat(batchChecklistTableColumnWidths[0], rowHeight, encodeBatchManifestText(truncateBatchManifestValue(row.OrderNumber, 18, "-")), "1", 0, "L", false, 0, "")
	pdf.CellFormat(batchChecklistTableColumnWidths[1], rowHeight, encodeBatchManifestText(truncateBatchManifestValue(row.RecipientName, 34, "-")), "1", 0, "L", false, 0, "")
	pdf.CellFormat(batchChecklistTableColumnWidths[2], rowHeight, encodeBatchManifestText(truncateBatchManifestValue(row.City, 18, "-")), "1", 0, "L", false, 0, "")

	itemsX := startX + sumBatchManifestColumnWidths(batchChecklistTableColumnWidths, 3)
	itemsWidth := batchChecklistTableColumnWidths[3]
	pdf.Rect(itemsX, startY, itemsWidth, rowHeight, "D")
	drawBatchManifestItemsCell(pdf, itemsX, startY, itemsWidth, batchChecklistItemsLineHeightMM, row.Items, "-", "-")

	checkStartX := startX + sumBatchManifestColumnWidths(batchChecklistTableColumnWidths, 4)
	pdf.SetXY(checkStartX, startY)
	for index := 4; index < len(batchChecklistTableColumnWidths); index++ {
		pdf.CellFormat(batchChecklistTableColumnWidths[index], rowHeight, "", "1", 0, "C", false, 0, "")
	}

	pdf.SetXY(startX, startY+rowHeight)
}

// batchChecklistTableRowHeight resolves dynamic row height values based on rendered items.
func batchChecklistTableRowHeight(pdf *gofpdf.Fpdf, row batchChecklistRow) float64 {
	if pdf == nil {
		return batchChecklistRowHeightMM
	}
	itemsText := encodeBatchManifestText(formatBatchManifestItemsAsList(row.Items, "-", "-"))
	pdf.SetFont("Arial", "", 8)
	lineCount := len(pdf.SplitLines([]byte(itemsText), batchChecklistTableColumnWidths[3]))
	if lineCount < 1 {
		lineCount = 1
	}
	height := float64(lineCount) * batchChecklistItemsLineHeightMM
	if height < batchChecklistRowHeightMM {
		return batchChecklistRowHeightMM
	}

	return height
}
