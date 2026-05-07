package store

import "mannaiah/module/exports/domain"

// mapReportToModel maps domain reports to persistence rows.
func mapReportToModel(report *domain.Report) reportModel {
	if report == nil {
		return reportModel{}
	}
	return reportModel{
		ID:          report.ID,
		ReportType:  string(report.Type),
		Status:      string(report.Status),
		Stamp:       report.Stamp,
		FileName:    report.FileName,
		StorageKey:  report.StorageKey,
		SHA256Hash:  report.SHA256,
		ContentType: report.ContentType,
		RowCount:    report.RowCount,
		ByteSize:    report.ByteSize,
		GeneratedAt: report.GeneratedAt,
		CreatedAt:   report.CreatedAt,
		UpdatedAt:   report.UpdatedAt,
	}
}

// mapModelToReport maps persistence rows to domain reports.
func mapModelToReport(model reportModel) domain.Report {
	return domain.Report{
		ID:          model.ID,
		Type:        domain.ReportType(model.ReportType),
		Status:      domain.ReportStatus(model.Status),
		Stamp:       model.Stamp,
		FileName:    model.FileName,
		StorageKey:  model.StorageKey,
		SHA256:      model.SHA256Hash,
		ContentType: model.ContentType,
		RowCount:    model.RowCount,
		ByteSize:    model.ByteSize,
		GeneratedAt: model.GeneratedAt,
		CreatedAt:   model.CreatedAt,
		UpdatedAt:   model.UpdatedAt,
	}
}
