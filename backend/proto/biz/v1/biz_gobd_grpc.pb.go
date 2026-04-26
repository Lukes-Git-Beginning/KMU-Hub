// Hand-written gRPC method name constants for GoBD-completion RPCs.
// The actual client/server interface methods and ServiceDesc entries are
// appended directly to biz_grpc.pb.go to keep the single registration point.
// See biz_gobd.pb.go for the request/response message types.

package bizv1

// Full-method name constants for GoBD-completion RPCs.
const (
	FinanceService_GetJournalSummary_FullMethodName    = "/biz.v1.FinanceService/GetJournalSummary"
	FinanceService_ValidateInvoiceNumber_FullMethodName = "/biz.v1.FinanceService/ValidateInvoiceNumber"
	FinanceService_LockInvoice_FullMethodName          = "/biz.v1.FinanceService/LockInvoice"
	FinanceService_GetPaymentStats_FullMethodName      = "/biz.v1.FinanceService/GetPaymentStats"
	FinanceService_UpdateDunningStatus_FullMethodName  = "/biz.v1.FinanceService/UpdateDunningStatus"
	FinanceService_SendDunningNotice_FullMethodName    = "/biz.v1.FinanceService/SendDunningNotice"
	FinanceService_GenerateGoBDExport_FullMethodName   = "/biz.v1.FinanceService/GenerateGoBDExport"
)
