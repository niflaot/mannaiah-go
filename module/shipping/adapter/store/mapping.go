package store

import (
	"encoding/base64"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"mannaiah/module/shipping/domain"
	"mannaiah/module/shipping/port"
)

func mapMarkModel(row shippingMarkModel) domain.ShippingMark {
	units := make([]domain.PackageUnit, 0, len(row.Units))
	for _, unit := range row.Units {
		units = append(units, domain.PackageUnit{
			Description: unit.Description,
			PackageType: unit.PackageType,
			Dimensions: domain.Dimensions{
				HeightCM:           unit.HeightCM,
				WidthCM:            unit.WidthCM,
				DepthCM:            unit.DepthCM,
				RealWeightKG:       unit.RealWeightKG,
				VolumetricWeightKG: unit.VolumetricWeightKG,
				DeclaredValueCOP:   unit.DeclaredValue,
			},
		})
	}
	return domain.ShippingMark{
		ID:                             row.ID,
		OrderID:                        row.OrderID,
		CarrierID:                      row.CarrierID,
		TrackingNumber:                 derefString(row.TrackingNumber),
		Status:                         domain.MarkStatus(row.Status),
		DocumentType:                   domain.MarkDocumentType(derefString(row.DocumentType)),
		DocumentRef:                    derefString(row.DocumentRef),
		ManifestType:                   domain.MarkDocumentType(derefString(row.ManifestType)),
		ManifestRef:                    derefString(row.ManifestRef),
		Sender:                         domain.Address{Name: row.SenderName, LegalName: row.SenderLegalName, ID: row.SenderID, IDType: row.SenderIDType, AddressLine: row.SenderAddress, CityCode: row.SenderCityCode, Phone: row.SenderPhone, Email: row.SenderEmail},
		Recipient:                      domain.Address{Name: row.RecipientName, LegalName: row.RecipientLegalName, ID: row.RecipientID, IDType: row.RecipientIDType, AddressLine: row.RecipientAddress, CityCode: row.RecipientCityCode, Phone: row.RecipientPhone, Email: row.RecipientEmail},
		Units:                          units,
		TotalWeight:                    row.TotalWeight,
		TotalVolumetricWeight:          row.TotalVolumetricWeight,
		DeclaredValue:                  row.DeclaredValue,
		PaymentForm:                    row.PaymentForm,
		CollectOnDeliveryAmount:        row.CollectOnDeliveryAmount,
		CollectOnDeliveryFeePercent:    row.CollectOnDeliveryFeePercent,
		CollectOnDeliveryChargedAmount: row.CollectOnDeliveryChargedAmount,
		Observations:                   row.Observations,
		DispatchBatchID:                row.DispatchBatchID,
		QuotationID:                    row.QuotationID,
		QuotedFreightCost:              row.QuotedFreightCost,
		DraftSnapshot:                  row.DraftSnapshot,
		ResponseSnapshot:               row.ResponseSnapshot,
		ShipmentMode:                   domain.ShipmentMode(row.ShipmentMode),
		FailureReason:                  row.FailureReason,
		CustomTrackingURL:              row.CustomTrackingURL,
		CreatedAt:                      row.CreatedAt.UTC(),
		UpdatedAt:                      row.UpdatedAt.UTC(),
	}
}

func mapMarkDomain(mark domain.ShippingMark) shippingMarkModel {
	normalized := mark.Normalize()
	row := shippingMarkModel{
		ID:                             normalized.ID,
		OrderID:                        normalized.OrderID,
		CarrierID:                      normalized.CarrierID,
		TrackingNumber:                 nullableString(normalized.TrackingNumber),
		Status:                         string(normalized.Status),
		DocumentType:                   nullableString(string(normalized.DocumentType)),
		DocumentRef:                    nullableString(normalized.DocumentRef),
		ManifestType:                   nullableString(string(normalized.ManifestType)),
		ManifestRef:                    nullableString(normalized.ManifestRef),
		SenderName:                     normalized.Sender.Name,
		SenderLegalName:                normalized.Sender.LegalName,
		SenderID:                       normalized.Sender.ID,
		SenderIDType:                   normalized.Sender.IDType,
		SenderAddress:                  normalized.Sender.AddressLine,
		SenderCityCode:                 normalized.Sender.CityCode,
		SenderPhone:                    normalized.Sender.Phone,
		SenderEmail:                    normalized.Sender.Email,
		RecipientName:                  normalized.Recipient.Name,
		RecipientLegalName:             normalized.Recipient.LegalName,
		RecipientID:                    normalized.Recipient.ID,
		RecipientIDType:                normalized.Recipient.IDType,
		RecipientAddress:               normalized.Recipient.AddressLine,
		RecipientCityCode:              normalized.Recipient.CityCode,
		RecipientPhone:                 normalized.Recipient.Phone,
		RecipientEmail:                 normalized.Recipient.Email,
		TotalWeight:                    normalized.TotalWeight,
		TotalVolumetricWeight:          normalized.TotalVolumetricWeight,
		DeclaredValue:                  normalized.DeclaredValue,
		PaymentForm:                    normalized.PaymentForm,
		CollectOnDeliveryAmount:        normalized.CollectOnDeliveryAmount,
		CollectOnDeliveryFeePercent:    normalized.CollectOnDeliveryFeePercent,
		CollectOnDeliveryChargedAmount: normalized.CollectOnDeliveryChargedAmount,
		Observations:                   normalized.Observations,
		DispatchBatchID:                normalized.DispatchBatchID,
		QuotationID:                    normalized.QuotationID,
		QuotedFreightCost:              normalized.QuotedFreightCost,
		DraftSnapshot:                  normalized.DraftSnapshot,
		ResponseSnapshot:               normalized.ResponseSnapshot,
		ShipmentMode:                   string(normalized.ShipmentMode),
		FailureReason:                  normalized.FailureReason,
		CustomTrackingURL:              normalized.CustomTrackingURL,
		CreatedAt:                      normalized.CreatedAt.UTC(),
		UpdatedAt:                      normalized.UpdatedAt.UTC(),
	}
	units := make([]shippingMarkUnitModel, 0, len(normalized.Units))
	for index, unit := range normalized.Units {
		units = append(units, shippingMarkUnitModel{
			ID:                 buildDeterministicID(normalized.ID, index),
			ShippingMarkID:     normalized.ID,
			Description:        unit.Description,
			PackageType:        unit.PackageType,
			HeightCM:           unit.Dimensions.HeightCM,
			WidthCM:            unit.Dimensions.WidthCM,
			DepthCM:            unit.Dimensions.DepthCM,
			RealWeightKG:       unit.Dimensions.RealWeightKG,
			VolumetricWeightKG: unit.Dimensions.VolumetricWeightKG,
			DeclaredValue:      unit.Dimensions.DeclaredValueCOP,
		})
	}
	row.Units = units

	return row
}

func mapBatchModel(row dispatchBatchModel, markIDs []string) domain.DispatchBatch {
	return domain.DispatchBatch{
		ID:        row.ID,
		CarrierID: row.CarrierID,
		Status:    domain.BatchStatus(row.Status),
		CreatedBy: row.CreatedBy,
		MarkIDs:   markIDs,
		CreatedAt: row.CreatedAt.UTC(),
		ClosedAt:  row.ClosedAt,
	}
}

func mapQuotationRecord(row quotationModel) port.QuotationRecord {
	record := port.QuotationRecord{
		ID:                         row.ID,
		OrderID:                    row.OrderID,
		OrderIdentifier:            row.OrderIdentifier,
		CarrierID:                  row.CarrierID,
		OriginCityCode:             row.OriginCityCode,
		DestCityCode:               row.DestCityCode,
		FreightCost:                row.FreightCost,
		CollectOnDeliveryFeeAmount: row.CollectOnDeliveryFeeAmount,
		EstimatedDays:              row.EstimatedDays,
		CurrencyCode:               row.CurrencyCode,
		ExpiresAt:                  row.ExpiresAt.UTC(),
		RequestSnapshot:            row.RequestSnapshot,
		RawResponse:                row.RawResponse,
		CreatedAt:                  row.CreatedAt.UTC(),
	}
	record.Units = mapQuotationUnitModels(row.Units)
	if len(record.Units) == 0 {
		record.Units = extractQuotationUnitsFromSnapshot(row.RequestSnapshot)
	}

	return record
}

func mapQuotationModel(record port.QuotationRecord) quotationModel {
	row := quotationModel{
		ID:                         record.ID,
		OrderID:                    record.OrderID,
		OrderIdentifier:            record.OrderIdentifier,
		CarrierID:                  record.CarrierID,
		OriginCityCode:             record.OriginCityCode,
		DestCityCode:               record.DestCityCode,
		FreightCost:                record.FreightCost,
		CollectOnDeliveryFeeAmount: record.CollectOnDeliveryFeeAmount,
		EstimatedDays:              record.EstimatedDays,
		CurrencyCode:               record.CurrencyCode,
		ExpiresAt:                  record.ExpiresAt.UTC(),
		RequestSnapshot:            record.RequestSnapshot,
		RawResponse:                record.RawResponse,
		CreatedAt:                  normalizeTime(record.CreatedAt),
	}
	row.Units = mapQuotationUnitDomain(record.ID, record.Units)

	return row
}

func normalizeTime(value time.Time) time.Time {
	if value.IsZero() {
		return time.Now().UTC()
	}

	return value.UTC()
}

func nullableString(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}

	return &trimmed
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}

	return *value
}

func buildDeterministicID(base string, index int) string {
	return base + "-unit-" + strconv.Itoa(index+1)
}

func mapQuotationUnitModels(rows []quotationUnitModel) []domain.PackageUnit {
	result := make([]domain.PackageUnit, 0, len(rows))
	for _, row := range rows {
		result = append(result, domain.PackageUnit{
			Description: row.Description,
			PackageType: row.PackageType,
			Dimensions: domain.Dimensions{
				HeightCM:           row.HeightCM,
				WidthCM:            row.WidthCM,
				DepthCM:            row.DepthCM,
				RealWeightKG:       row.RealWeightKG,
				VolumetricWeightKG: row.VolumetricWeightKG,
				DeclaredValueCOP:   row.DeclaredValue,
			},
		})
	}

	return result
}

func mapQuotationUnitDomain(quotationID string, units []domain.PackageUnit) []quotationUnitModel {
	result := make([]quotationUnitModel, 0, len(units))
	for index, unit := range units {
		result = append(result, quotationUnitModel{
			ID:                  buildDeterministicID(quotationID, index),
			ShippingQuotationID: quotationID,
			UnitIndex:           index + 1,
			Description:         strings.TrimSpace(unit.Description),
			PackageType:         strings.TrimSpace(unit.PackageType),
			HeightCM:            unit.Dimensions.HeightCM,
			WidthCM:             unit.Dimensions.WidthCM,
			DepthCM:             unit.Dimensions.DepthCM,
			RealWeightKG:        unit.Dimensions.RealWeightKG,
			VolumetricWeightKG:  unit.Dimensions.VolumetricWeightKG,
			DeclaredValue:       unit.Dimensions.DeclaredValueCOP,
			CreatedAt:           time.Now().UTC(),
		})
	}

	return result
}

func extractQuotationUnitsFromSnapshot(snapshotValue string) []domain.PackageUnit {
	trimmedSnapshot := strings.TrimSpace(snapshotValue)
	if trimmedSnapshot == "" {
		return nil
	}
	decodedSnapshot, decodeErr := base64.StdEncoding.DecodeString(trimmedSnapshot)
	if decodeErr != nil {
		decodedSnapshot = []byte(trimmedSnapshot)
	}
	var snapshot struct {
		Units []domain.PackageUnit `json:"units"`
	}
	if jsonErr := json.Unmarshal(decodedSnapshot, &snapshot); jsonErr != nil {
		return nil
	}

	return snapshot.Units
}
