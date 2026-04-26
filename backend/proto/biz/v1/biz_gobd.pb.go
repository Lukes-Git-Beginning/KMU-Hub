// Hand-written proto message types for GoBD Journal, Payment Stats, Dunning Gaps,
// and GoBD Export RPCs. Follows the same simplified pattern as biz_time.pb.go and
// biz_quote_from_deal.pb.go (no protoimpl reflection machinery).
// Run `make proto-biz` after updating biz.proto to regenerate properly.

package bizv1

// ============================================================================
// GoBD Journal Summary
// ============================================================================

// GetJournalSummaryRequest requests the GoBD journal summary for a fiscal year.
type GetJournalSummaryRequest struct {
	TenantId string `protobuf:"bytes,1,opt,name=tenant_id,json=tenantId,proto3" json:"tenant_id,omitempty"`
	Year     int32  `protobuf:"varint,2,opt,name=year,proto3" json:"year,omitempty"`
}

func (x *GetJournalSummaryRequest) GetTenantId() string {
	if x != nil {
		return x.TenantId
	}
	return ""
}

func (x *GetJournalSummaryRequest) GetYear() int32 {
	if x != nil {
		return x.Year
	}
	return 0
}

// GetJournalSummaryResponse reports the journal numbering status for a fiscal year.
// GoBD compliance requires gaps_detected == 0.
type GetJournalSummaryResponse struct {
	Year               int32  `protobuf:"varint,1,opt,name=year,proto3" json:"year,omitempty"`
	TotalInvoicesIssued int32  `protobuf:"varint,2,opt,name=total_invoices_issued,json=totalInvoicesIssued,proto3" json:"total_invoices_issued,omitempty"`
	GapsDetected       int32  `protobuf:"varint,3,opt,name=gaps_detected,json=gapsDetected,proto3" json:"gaps_detected,omitempty"`
	FirstNumber        string `protobuf:"bytes,4,opt,name=first_number,json=firstNumber,proto3" json:"first_number,omitempty"`
	LastNumber         string `protobuf:"bytes,5,opt,name=last_number,json=lastNumber,proto3" json:"last_number,omitempty"`
	HighestSequence    int32  `protobuf:"varint,6,opt,name=highest_sequence,json=highestSequence,proto3" json:"highest_sequence,omitempty"`
}

func (x *GetJournalSummaryResponse) GetYear() int32 {
	if x != nil {
		return x.Year
	}
	return 0
}

func (x *GetJournalSummaryResponse) GetTotalInvoicesIssued() int32 {
	if x != nil {
		return x.TotalInvoicesIssued
	}
	return 0
}

func (x *GetJournalSummaryResponse) GetGapsDetected() int32 {
	if x != nil {
		return x.GapsDetected
	}
	return 0
}

func (x *GetJournalSummaryResponse) GetFirstNumber() string {
	if x != nil {
		return x.FirstNumber
	}
	return ""
}

func (x *GetJournalSummaryResponse) GetLastNumber() string {
	if x != nil {
		return x.LastNumber
	}
	return ""
}

func (x *GetJournalSummaryResponse) GetHighestSequence() int32 {
	if x != nil {
		return x.HighestSequence
	}
	return 0
}

// ============================================================================
// ValidateInvoiceNumber
// ============================================================================

// ValidateInvoiceNumberRequest validates a candidate invoice number.
type ValidateInvoiceNumberRequest struct {
	TenantId      string `protobuf:"bytes,1,opt,name=tenant_id,json=tenantId,proto3" json:"tenant_id,omitempty"`
	InvoiceNumber string `protobuf:"bytes,2,opt,name=invoice_number,json=invoiceNumber,proto3" json:"invoice_number,omitempty"`
}

func (x *ValidateInvoiceNumberRequest) GetTenantId() string {
	if x != nil {
		return x.TenantId
	}
	return ""
}

func (x *ValidateInvoiceNumberRequest) GetInvoiceNumber() string {
	if x != nil {
		return x.InvoiceNumber
	}
	return ""
}

// ValidateInvoiceNumberResponse reports whether the number is format-valid and unused.
type ValidateInvoiceNumberResponse struct {
	ValidFormat  bool   `protobuf:"varint,1,opt,name=valid_format,json=validFormat,proto3" json:"valid_format,omitempty"`
	AlreadyUsed  bool   `protobuf:"varint,2,opt,name=already_used,json=alreadyUsed,proto3" json:"already_used,omitempty"`
	Canonical    string `protobuf:"bytes,3,opt,name=canonical,proto3" json:"canonical,omitempty"`
}

func (x *ValidateInvoiceNumberResponse) GetValidFormat() bool {
	if x != nil {
		return x.ValidFormat
	}
	return false
}

func (x *ValidateInvoiceNumberResponse) GetAlreadyUsed() bool {
	if x != nil {
		return x.AlreadyUsed
	}
	return false
}

func (x *ValidateInvoiceNumberResponse) GetCanonical() string {
	if x != nil {
		return x.Canonical
	}
	return ""
}

// ============================================================================
// LockInvoice
// ============================================================================

// LockInvoiceRequest requests an administrative lock on an invoice.
// NOTE: locked_at is recorded in snapshot_data JSONB until Sprint 4 adds a
// dedicated locked_at TIMESTAMPTZ column (migration not yet created).
type LockInvoiceRequest struct {
	Id       string `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	TenantId string `protobuf:"bytes,2,opt,name=tenant_id,json=tenantId,proto3" json:"tenant_id,omitempty"`
	LockedBy string `protobuf:"bytes,3,opt,name=locked_by,json=lockedBy,proto3" json:"locked_by,omitempty"`
}

func (x *LockInvoiceRequest) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *LockInvoiceRequest) GetTenantId() string {
	if x != nil {
		return x.TenantId
	}
	return ""
}

func (x *LockInvoiceRequest) GetLockedBy() string {
	if x != nil {
		return x.LockedBy
	}
	return ""
}

// LockInvoiceResponse carries the locked invoice and the lock timestamp.
type LockInvoiceResponse struct {
	// InvoiceJson contains the JSON-encoded models.Invoice after locking.
	InvoiceJson []byte `protobuf:"bytes,1,opt,name=invoice_json,json=invoiceJson,proto3" json:"invoice_json,omitempty"`
	LockedAt    string `protobuf:"bytes,2,opt,name=locked_at,json=lockedAt,proto3" json:"locked_at,omitempty"` // RFC3339
}

func (x *LockInvoiceResponse) GetInvoiceJson() []byte {
	if x != nil {
		return x.InvoiceJson
	}
	return nil
}

func (x *LockInvoiceResponse) GetLockedAt() string {
	if x != nil {
		return x.LockedAt
	}
	return ""
}

// ============================================================================
// GetPaymentStats
// ============================================================================

// GetPaymentStatsRequest requests aggregated payment statistics.
type GetPaymentStatsRequest struct {
	TenantId string `protobuf:"bytes,1,opt,name=tenant_id,json=tenantId,proto3" json:"tenant_id,omitempty"`
	FromDate string `protobuf:"bytes,2,opt,name=from_date,json=fromDate,proto3" json:"from_date,omitempty"` // YYYY-MM-DD
	ToDate   string `protobuf:"bytes,3,opt,name=to_date,json=toDate,proto3" json:"to_date,omitempty"`         // YYYY-MM-DD
}

func (x *GetPaymentStatsRequest) GetTenantId() string {
	if x != nil {
		return x.TenantId
	}
	return ""
}

func (x *GetPaymentStatsRequest) GetFromDate() string {
	if x != nil {
		return x.FromDate
	}
	return ""
}

func (x *GetPaymentStatsRequest) GetToDate() string {
	if x != nil {
		return x.ToDate
	}
	return ""
}

// GetPaymentStatsResponse carries aggregated payment statistics.
type GetPaymentStatsResponse struct {
	TotalInvoices          int32  `protobuf:"varint,1,opt,name=total_invoices,json=totalInvoices,proto3" json:"total_invoices,omitempty"`
	TotalPaid              int32  `protobuf:"varint,2,opt,name=total_paid,json=totalPaid,proto3" json:"total_paid,omitempty"`
	TotalOutstanding       int32  `protobuf:"varint,3,opt,name=total_outstanding,json=totalOutstanding,proto3" json:"total_outstanding,omitempty"`
	TotalPaidAmount        string `protobuf:"bytes,4,opt,name=total_paid_amount,json=totalPaidAmount,proto3" json:"total_paid_amount,omitempty"`
	TotalOutstandingAmount string `protobuf:"bytes,5,opt,name=total_outstanding_amount,json=totalOutstandingAmount,proto3" json:"total_outstanding_amount,omitempty"`
	AverageDaysToPay       string `protobuf:"bytes,6,opt,name=average_days_to_pay,json=averageDaysToPay,proto3" json:"average_days_to_pay,omitempty"`
}

func (x *GetPaymentStatsResponse) GetTotalInvoices() int32 {
	if x != nil {
		return x.TotalInvoices
	}
	return 0
}

func (x *GetPaymentStatsResponse) GetTotalPaid() int32 {
	if x != nil {
		return x.TotalPaid
	}
	return 0
}

func (x *GetPaymentStatsResponse) GetTotalOutstanding() int32 {
	if x != nil {
		return x.TotalOutstanding
	}
	return 0
}

func (x *GetPaymentStatsResponse) GetTotalPaidAmount() string {
	if x != nil {
		return x.TotalPaidAmount
	}
	return ""
}

func (x *GetPaymentStatsResponse) GetTotalOutstandingAmount() string {
	if x != nil {
		return x.TotalOutstandingAmount
	}
	return ""
}

func (x *GetPaymentStatsResponse) GetAverageDaysToPay() string {
	if x != nil {
		return x.AverageDaysToPay
	}
	return ""
}

// ============================================================================
// UpdateDunningStatus
// ============================================================================

// UpdateDunningStatusRequest directly sets the status of a dunning record.
// Used for manual overrides outside the standard Send/Escalate flow.
type UpdateDunningStatusRequest struct {
	Id       string `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	TenantId string `protobuf:"bytes,2,opt,name=tenant_id,json=tenantId,proto3" json:"tenant_id,omitempty"`
	Status   string `protobuf:"bytes,3,opt,name=status,proto3" json:"status,omitempty"` // "draft" | "sent" | "paid"
}

func (x *UpdateDunningStatusRequest) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *UpdateDunningStatusRequest) GetTenantId() string {
	if x != nil {
		return x.TenantId
	}
	return ""
}

func (x *UpdateDunningStatusRequest) GetStatus() string {
	if x != nil {
		return x.Status
	}
	return ""
}

// UpdateDunningStatusResponse carries the updated dunning record.
type UpdateDunningStatusResponse struct {
	Dunning *DunningRecord `protobuf:"bytes,1,opt,name=dunning,proto3" json:"dunning,omitempty"`
}

func (x *UpdateDunningStatusResponse) GetDunning() *DunningRecord {
	if x != nil {
		return x.Dunning
	}
	return nil
}

// ============================================================================
// SendDunningNotice
// ============================================================================

// SendDunningNoticeRequest triggers the notice-send flow for a dunning record.
// Sets sent_at and increments dunning_level in the service layer.
// Actual email delivery is a Sprint 3 item -- see TODO in service implementation.
type SendDunningNoticeRequest struct {
	Id       string `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	TenantId string `protobuf:"bytes,2,opt,name=tenant_id,json=tenantId,proto3" json:"tenant_id,omitempty"`
}

func (x *SendDunningNoticeRequest) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *SendDunningNoticeRequest) GetTenantId() string {
	if x != nil {
		return x.TenantId
	}
	return ""
}

// SendDunningNoticeResponse carries the updated dunning record and email queue status.
type SendDunningNoticeResponse struct {
	Dunning      *DunningRecord `protobuf:"bytes,1,opt,name=dunning,proto3" json:"dunning,omitempty"`
	EmailQueued  bool           `protobuf:"varint,2,opt,name=email_queued,json=emailQueued,proto3" json:"email_queued,omitempty"`
}

func (x *SendDunningNoticeResponse) GetDunning() *DunningRecord {
	if x != nil {
		return x.Dunning
	}
	return nil
}

func (x *SendDunningNoticeResponse) GetEmailQueued() bool {
	if x != nil {
		return x.EmailQueued
	}
	return false
}

// ============================================================================
// GenerateGoBDExport
// ============================================================================

// GenerateGoBDExportRequest requests a GoBD-compliant CSV export.
// The export covers all invoices (sent/paid/overdue) and credit notes
// in the given date range, formatted per GoBD Anlage A requirements.
type GenerateGoBDExportRequest struct {
	TenantId string `protobuf:"bytes,1,opt,name=tenant_id,json=tenantId,proto3" json:"tenant_id,omitempty"`
	FromDate string `protobuf:"bytes,2,opt,name=from_date,json=fromDate,proto3" json:"from_date,omitempty"` // YYYY-MM-DD
	ToDate   string `protobuf:"bytes,3,opt,name=to_date,json=toDate,proto3" json:"to_date,omitempty"`         // YYYY-MM-DD
}

func (x *GenerateGoBDExportRequest) GetTenantId() string {
	if x != nil {
		return x.TenantId
	}
	return ""
}

func (x *GenerateGoBDExportRequest) GetFromDate() string {
	if x != nil {
		return x.FromDate
	}
	return ""
}

func (x *GenerateGoBDExportRequest) GetToDate() string {
	if x != nil {
		return x.ToDate
	}
	return ""
}

// GenerateGoBDExportResponse carries the generated GoBD CSV.
type GenerateGoBDExportResponse struct {
	CsvData       []byte `protobuf:"bytes,1,opt,name=csv_data,json=csvData,proto3" json:"csv_data,omitempty"`
	Filename      string `protobuf:"bytes,2,opt,name=filename,proto3" json:"filename,omitempty"`
	RecordCount   int32  `protobuf:"varint,3,opt,name=record_count,json=recordCount,proto3" json:"record_count,omitempty"`
	FormatVersion string `protobuf:"bytes,4,opt,name=format_version,json=formatVersion,proto3" json:"format_version,omitempty"`
}

func (x *GenerateGoBDExportResponse) GetCsvData() []byte {
	if x != nil {
		return x.CsvData
	}
	return nil
}

func (x *GenerateGoBDExportResponse) GetFilename() string {
	if x != nil {
		return x.Filename
	}
	return ""
}

func (x *GenerateGoBDExportResponse) GetRecordCount() int32 {
	if x != nil {
		return x.RecordCount
	}
	return 0
}

func (x *GenerateGoBDExportResponse) GetFormatVersion() string {
	if x != nil {
		return x.FormatVersion
	}
	return ""
}
