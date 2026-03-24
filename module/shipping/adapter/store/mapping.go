package store

import (
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
		Sender:                         domain.Address{Name: row.SenderName, ID: row.SenderID, IDType: row.SenderIDType, AddressLine: row.SenderAddress, CityCode: row.SenderCityCode, Phone: row.SenderPhone, Email: row.SenderEmail},
		Recipient:                      domain.Address{Name: row.RecipientName, ID: row.RecipientID, IDType: row.RecipientIDType, AddressLine: row.RecipientAddress, CityCode: row.RecipientCityCode, Phone: row.RecipientPhone, Email: row.RecipientEmail},
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
		ShipmentMode:                   domain.ShipmentMode(row.ShipmentMode),
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
		SenderName:                     normalized.Sender.Name,
		SenderID:                       normalized.Sender.ID,
		SenderIDType:                   normalized.Sender.IDType,
		SenderAddress:                  normalized.Sender.AddressLine,
		SenderCityCode:                 normalized.Sender.CityCode,
		SenderPhone:                    normalized.Sender.Phone,
		SenderEmail:                    normalized.Sender.Email,
		RecipientName:                  normalized.Recipient.Name,
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
		ShipmentMode:                   string(normalized.ShipmentMode),
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
	fullFreightCost := row.FullFreightCost
	if fullFreightCost <= 0 {
		fullFreightCost = row.FreightCost
	}
	discountedFreightCost := row.DiscountedCost
	if discountedFreightCost <= 0 {
		discountedFreightCost = row.FreightCost
	}
	freightCost := row.FreightCost
	if freightCost <= 0 {
		freightCost = discountedFreightCost
	}

	return port.QuotationRecord{
		ID:                    row.ID,
		OrderID:               row.OrderID,
		CarrierID:             row.CarrierID,
		OriginCityCode:        row.OriginCityCode,
		DestCityCode:          row.DestCityCode,
		FullFreightCost:       fullFreightCost,
		DiscountPercent:       row.DiscountPercent,
		DiscountedFreightCost: discountedFreightCost,
		FreightCost:           freightCost,
		EstimatedDays:         row.EstimatedDays,
		CurrencyCode:          row.CurrencyCode,
		ExpiresAt:             row.ExpiresAt.UTC(),
		RequestSnapshot:       row.RequestSnapshot,
		RawResponse:           row.RawResponse,
		CreatedAt:             row.CreatedAt.UTC(),
	}
}

func mapQuotationModel(record port.QuotationRecord) quotationModel {
	return quotationModel{
		ID:              record.ID,
		OrderID:         record.OrderID,
		CarrierID:       record.CarrierID,
		OriginCityCode:  record.OriginCityCode,
		DestCityCode:    record.DestCityCode,
		FullFreightCost: record.FullFreightCost,
		DiscountPercent: record.DiscountPercent,
		DiscountedCost:  record.DiscountedFreightCost,
		FreightCost:     record.FreightCost,
		EstimatedDays:   record.EstimatedDays,
		CurrencyCode:    record.CurrencyCode,
		ExpiresAt:       record.ExpiresAt.UTC(),
		RequestSnapshot: record.RequestSnapshot,
		RawResponse:     record.RawResponse,
		CreatedAt:       normalizeTime(record.CreatedAt),
	}
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
