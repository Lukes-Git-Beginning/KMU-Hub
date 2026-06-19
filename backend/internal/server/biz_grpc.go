package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/kmuhub/kmuhub/internal/biz/creditnote"
	"github.com/kmuhub/kmuhub/internal/biz/dashboard"
	"github.com/kmuhub/kmuhub/internal/biz/datev"
	"github.com/kmuhub/kmuhub/internal/biz/dunning"
	"github.com/kmuhub/kmuhub/internal/biz/einvoice"
	"github.com/kmuhub/kmuhub/internal/biz/gobdarchive"
	"github.com/kmuhub/kmuhub/internal/biz/hr/timetracking"
	"github.com/kmuhub/kmuhub/internal/biz/invoice"
	"github.com/kmuhub/kmuhub/internal/biz/payment"
	"github.com/kmuhub/kmuhub/internal/biz/pdf"
	"github.com/kmuhub/kmuhub/internal/biz/quote"
	"github.com/kmuhub/kmuhub/internal/biz/tax"
	"github.com/kmuhub/kmuhub/internal/models"
	bizv1 "github.com/kmuhub/kmuhub/proto/biz/v1"
	crmv1 "github.com/kmuhub/kmuhub/proto/crm/v1"
)

// CompanySettingsRepository provides CRUD for company settings in the gRPC server.
type CompanySettingsRepository interface {
	GetByTenantID(ctx context.Context, tenantID uuid.UUID) (*models.CompanySettings, error)
	Upsert(ctx context.Context, settings *models.CompanySettings) error
}

// BizGRPCServer implements the FinanceService gRPC service.
type BizGRPCServer struct {
	bizv1.UnimplementedFinanceServiceServer
	quoteService      *quote.Service
	invoiceService    *invoice.Service
	creditNoteService *creditnote.Service
	paymentService    *payment.Service
	dunningService    *dunning.Service
	dashboardService  *dashboard.Service
	pdfGenerator      *pdf.Generator
	datevExporter     *datev.Exporter
	companySettings   CompanySettingsRepository
	timetrackingRepo  timetracking.WorkTimeRepository
	crmClient         crmv1.CRMServiceClient // optional; nil if CRM service unreachable
	gobdArchiveSvc    *gobdarchive.Service   // GoBD Belegarchiv (§147 AO)
	einvoiceSvc       *einvoice.Service      // incoming e-invoice processing (E-Rechnung Eingang)
}

// NewBizGRPCServer creates a new BizGRPCServer with all finance services.
func NewBizGRPCServer(
	quoteService *quote.Service,
	invoiceService *invoice.Service,
	creditNoteService *creditnote.Service,
	paymentService *payment.Service,
	dunningService *dunning.Service,
	dashboardService *dashboard.Service,
	pdfGenerator *pdf.Generator,
	datevExporter *datev.Exporter,
	companySettings CompanySettingsRepository,
	timetrackingRepo timetracking.WorkTimeRepository,
	crmClient crmv1.CRMServiceClient,
	gobdArchiveSvc *gobdarchive.Service,
	einvoiceSvc *einvoice.Service,
) *BizGRPCServer {
	return &BizGRPCServer{
		quoteService:      quoteService,
		invoiceService:    invoiceService,
		creditNoteService: creditNoteService,
		paymentService:    paymentService,
		dunningService:    dunningService,
		dashboardService:  dashboardService,
		pdfGenerator:      pdfGenerator,
		datevExporter:     datevExporter,
		companySettings:   companySettings,
		timetrackingRepo:  timetrackingRepo,
		crmClient:         crmClient,
		gobdArchiveSvc:    gobdArchiveSvc,
		einvoiceSvc:       einvoiceSvc,
	}
}

// pageToLimitOffset converts page/per_page to limit/offset.
func pageToLimitOffset(page, perPage int32) (int, int) {
	limit := int(perPage)
	if limit <= 0 {
		limit = 50
	}
	p := int(page)
	if p <= 0 {
		p = 1
	}
	offset := (p - 1) * limit
	return limit, offset
}

// ============================================================================
// Company Settings
// ============================================================================

// requireCompanySettings loads the tenant's company settings for document
// generation. GetByTenantID returns (nil, nil) when no settings row exists
// yet — callers dereference the result, so that case must be a clean
// FailedPrecondition instead of a nil-pointer panic.
func (s *BizGRPCServer) requireCompanySettings(ctx context.Context, tenantID uuid.UUID) (*models.CompanySettings, error) {
	settings, err := s.companySettings.GetByTenantID(ctx, tenantID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to load company settings")
	}
	if settings == nil {
		return nil, status.Error(codes.FailedPrecondition, "company settings not configured")
	}
	return settings, nil
}

// taxRateForMode returns the line-item tax rate for the given tax mode.
// Kleinunternehmer (section 19 UStG) and reverse-charge are tax-exempt (0%);
// the standard rate (19%) applies otherwise.
func taxRateForMode(mode string) decimal.Decimal {
	switch mode {
	case models.TaxModeKleinunternehmer, models.TaxModeReverseCharge:
		return decimal.Zero
	default:
		return tax.StandardRate()
	}
}

// resolveTaxMode picks the effective tax mode for a generated invoice. An
// explicit request mode always wins; otherwise a Kleinunternehmer company
// (section 19 UStG) defaults to kleinunternehmer, else standard. Reverse-charge
// is customer-/document-driven and only ever comes from an explicit request.
func resolveTaxMode(requestMode string, settings *models.CompanySettings) string {
	if requestMode != "" {
		return requestMode
	}
	if settings != nil && settings.IsKleinunternehmer {
		return models.TaxModeKleinunternehmer
	}
	return models.TaxModeStandard
}

func (s *BizGRPCServer) GetCompanySettings(ctx context.Context, req *bizv1.GetCompanySettingsRequest) (*bizv1.GetCompanySettingsResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}

	settings, err := s.companySettings.GetByTenantID(ctx, tenantID)
	if err != nil {
		return nil, mapBizError(err)
	}

	return &bizv1.GetCompanySettingsResponse{
		Settings: toProtoCompanySettings(settings),
	}, nil
}

func (s *BizGRPCServer) UpdateCompanySettings(ctx context.Context, req *bizv1.UpdateCompanySettingsRequest) (*bizv1.UpdateCompanySettingsResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}

	ps := req.GetSettings()
	if ps == nil {
		return nil, status.Error(codes.InvalidArgument, "settings is required")
	}

	settings := &models.CompanySettings{
		TenantID:                 tenantID,
		Name:                     ps.GetName(),
		Street:                   ps.GetStreet(),
		PLZ:                      ps.GetPlz(),
		City:                     ps.GetCity(),
		Country:                  ps.GetCountry(),
		Steuernummer:             ps.GetSteuernummer(),
		UStIDNr:                  ps.GetUstIdNr(),
		Handelsregister:          ps.GetHandelsregister(),
		BankName:                 ps.GetBankName(),
		IBAN:                     ps.GetIban(),
		BIC:                      ps.GetBic(),
		LogoURL:                  ps.GetLogoUrl(),
		AccentColor:              ps.GetAccentColor(),
		IsKleinunternehmer:       ps.GetIsKleinunternehmer(),
		DefaultPaymentTermsDays:  int(ps.GetDefaultPaymentTermsDays()),
		DefaultQuoteValidityDays: int(ps.GetDefaultQuoteValidityDays()),
		UpdatedAt:                time.Now(),
	}

	if ps.GetBasiszinssatz() != "" {
		bz, bzErr := decimal.NewFromString(ps.GetBasiszinssatz())
		if bzErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid basiszinssatz")
		}
		settings.Basiszinssatz = bz
	}

	if err := s.companySettings.Upsert(ctx, settings); err != nil {
		return nil, mapBizError(err)
	}

	updated, err := s.companySettings.GetByTenantID(ctx, tenantID)
	if err != nil {
		return nil, mapBizError(err)
	}

	return &bizv1.UpdateCompanySettingsResponse{
		Settings: toProtoCompanySettings(updated),
	}, nil
}

// ============================================================================
// Quotes
// ============================================================================

func (s *BizGRPCServer) CreateQuote(ctx context.Context, req *bizv1.CreateQuoteRequest) (*bizv1.CreateQuoteResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}

	userID, err := uuid.Parse(req.GetCreatedBy())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid created_by")
	}

	lineItems := protoLineItemsToModel(req.GetLineItems())

	input := quote.CreateInput{
		TenantID:        tenantID,
		CustomerName:    req.GetCustomer().GetName(),
		CustomerAddress: req.GetCustomer().GetAddress(),
		CustomerEmail:   req.GetCustomer().GetEmail(),
		CustomerUStIDNr: req.GetCustomer().GetUstIdNr(),
		TaxMode:         taxModeFromProto(req.GetTaxMode()),
		LineItems:       lineItems,
		Notes:           req.GetNotes(),
		UserID:          userID,
	}

	if req.GetValidUntil() != "" {
		t, parseErr := time.Parse("2006-01-02", req.GetValidUntil())
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid valid_until date")
		}
		input.ValidUntil = &t
	}

	if req.GetDealId() != "" {
		dealID, dealErr := uuid.Parse(req.GetDealId())
		if dealErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid deal_id")
		}
		input.DealID = &dealID
	}

	q, err := s.quoteService.Create(ctx, input)
	if err != nil {
		return nil, mapBizError(err)
	}

	return &bizv1.CreateQuoteResponse{Quote: toProtoQuote(q)}, nil
}

func (s *BizGRPCServer) GetQuote(ctx context.Context, req *bizv1.GetQuoteRequest) (*bizv1.GetQuoteResponse, error) {
	tenantID, id, err := parseTenantAndID(req.GetTenantId(), req.GetId())
	if err != nil {
		return nil, err
	}

	q, err := s.quoteService.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, mapBizError(err)
	}

	return &bizv1.GetQuoteResponse{Quote: toProtoQuote(q)}, nil
}

func (s *BizGRPCServer) ListQuotes(ctx context.Context, req *bizv1.ListQuotesRequest) (*bizv1.ListQuotesResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}

	limit, offset := pageToLimitOffset(req.GetPage(), req.GetPerPage())

	filter := quote.ListFilter{
		Status: quoteStatusFromProto(req.GetStatus()),
		Limit:  limit,
		Offset: offset,
	}

	if req.GetDealId() != "" {
		dealID, dealErr := uuid.Parse(req.GetDealId())
		if dealErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid deal_id")
		}
		filter.DealID = &dealID
	}

	quotes, total, err := s.quoteService.List(ctx, tenantID, filter)
	if err != nil {
		return nil, mapBizError(err)
	}

	protoQuotes := make([]*bizv1.Quote, 0, len(quotes))
	for _, q := range quotes {
		protoQuotes = append(protoQuotes, toProtoQuote(q))
	}

	return &bizv1.ListQuotesResponse{Quotes: protoQuotes, Total: int32(total)}, nil
}

func (s *BizGRPCServer) UpdateQuote(ctx context.Context, req *bizv1.UpdateQuoteRequest) (*bizv1.UpdateQuoteResponse, error) {
	tenantID, id, err := parseTenantAndID(req.GetTenantId(), req.GetId())
	if err != nil {
		return nil, err
	}

	input := quote.UpdateInput{}

	if req.Customer != nil {
		name := req.Customer.GetName()
		input.CustomerName = &name
		addr := req.Customer.GetAddress()
		input.CustomerAddress = &addr
		email := req.Customer.GetEmail()
		input.CustomerEmail = &email
		ustid := req.Customer.GetUstIdNr()
		input.CustomerUStIDNr = &ustid
	}

	if req.GetTaxMode() != bizv1.TaxMode_TAX_MODE_UNSPECIFIED {
		mode := taxModeFromProto(req.GetTaxMode())
		input.TaxMode = &mode
	}

	if len(req.GetLineItems()) > 0 {
		input.LineItems = protoLineItemsToModel(req.GetLineItems())
	}

	if req.GetValidUntil() != "" {
		t, parseErr := time.Parse("2006-01-02", req.GetValidUntil())
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid valid_until date")
		}
		input.ValidUntil = &t
	}

	if req.GetNotes() != "" {
		notes := req.GetNotes()
		input.Notes = &notes
	}

	q, err := s.quoteService.Update(ctx, tenantID, id, input)
	if err != nil {
		return nil, mapBizError(err)
	}

	return &bizv1.UpdateQuoteResponse{Quote: toProtoQuote(q)}, nil
}

func (s *BizGRPCServer) DeleteQuote(ctx context.Context, req *bizv1.DeleteQuoteRequest) (*bizv1.DeleteQuoteResponse, error) {
	tenantID, id, err := parseTenantAndID(req.GetTenantId(), req.GetId())
	if err != nil {
		return nil, err
	}

	if err := s.quoteService.Delete(ctx, tenantID, id); err != nil {
		return nil, mapBizError(err)
	}

	return &bizv1.DeleteQuoteResponse{}, nil
}

func (s *BizGRPCServer) SendQuote(ctx context.Context, req *bizv1.SendQuoteRequest) (*bizv1.SendQuoteResponse, error) {
	tenantID, id, err := parseTenantAndID(req.GetTenantId(), req.GetId())
	if err != nil {
		return nil, err
	}

	// Proto doesn't include user_id; use uuid.Nil (gateway handles auth)
	if err := s.quoteService.Send(ctx, tenantID, id, uuid.Nil); err != nil {
		return nil, mapBizError(err)
	}

	q, err := s.quoteService.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, mapBizError(err)
	}

	return &bizv1.SendQuoteResponse{Quote: toProtoQuote(q)}, nil
}

func (s *BizGRPCServer) AcceptQuote(ctx context.Context, req *bizv1.AcceptQuoteRequest) (*bizv1.AcceptQuoteResponse, error) {
	tenantID, id, err := parseTenantAndID(req.GetTenantId(), req.GetId())
	if err != nil {
		return nil, err
	}

	if err := s.quoteService.Accept(ctx, tenantID, id); err != nil {
		return nil, mapBizError(err)
	}

	q, err := s.quoteService.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, mapBizError(err)
	}

	return &bizv1.AcceptQuoteResponse{Quote: toProtoQuote(q)}, nil
}

func (s *BizGRPCServer) RejectQuote(ctx context.Context, req *bizv1.RejectQuoteRequest) (*bizv1.RejectQuoteResponse, error) {
	tenantID, id, err := parseTenantAndID(req.GetTenantId(), req.GetId())
	if err != nil {
		return nil, err
	}

	if err := s.quoteService.Reject(ctx, tenantID, id); err != nil {
		return nil, mapBizError(err)
	}

	q, err := s.quoteService.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, mapBizError(err)
	}

	return &bizv1.RejectQuoteResponse{Quote: toProtoQuote(q)}, nil
}

func (s *BizGRPCServer) ExpireQuote(ctx context.Context, req *bizv1.ExpireQuoteRequest) (*bizv1.ExpireQuoteResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}

	// ExpireQuotes is a bulk operation; proto ExpireQuote returns a single quote
	count, expErr := s.quoteService.ExpireQuotes(ctx, tenantID)
	if expErr != nil {
		return nil, mapBizError(expErr)
	}

	if count == 0 {
		return &bizv1.ExpireQuoteResponse{}, nil
	}

	// Return the first expired quote if a specific one was requested
	if req.GetId() != "" {
		id, idErr := uuid.Parse(req.GetId())
		if idErr == nil {
			q, qErr := s.quoteService.GetByID(ctx, tenantID, id)
			if qErr == nil {
				return &bizv1.ExpireQuoteResponse{Quote: toProtoQuote(q)}, nil
			}
		}
	}

	return &bizv1.ExpireQuoteResponse{}, nil
}

func (s *BizGRPCServer) ConvertQuoteToInvoice(ctx context.Context, req *bizv1.ConvertQuoteToInvoiceRequest) (*bizv1.ConvertQuoteToInvoiceResponse, error) {
	tenantID, quoteID, err := parseTenantAndID(req.GetTenantId(), req.GetId())
	if err != nil {
		return nil, err
	}

	inv, err := s.invoiceService.CreateFromQuote(ctx, tenantID, quoteID, uuid.Nil)
	if err != nil {
		return nil, mapBizError(err)
	}

	return &bizv1.ConvertQuoteToInvoiceResponse{Invoice: toProtoInvoice(inv)}, nil
}

// ============================================================================
// Invoices
// ============================================================================

func (s *BizGRPCServer) CreateInvoice(ctx context.Context, req *bizv1.CreateInvoiceRequest) (*bizv1.CreateInvoiceResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}

	userID, err := uuid.Parse(req.GetCreatedBy())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid created_by")
	}

	lineItems := protoLineItemsToModel(req.GetLineItems())

	input := invoice.CreateInput{
		TenantID:        tenantID,
		CustomerName:    req.GetCustomer().GetName(),
		CustomerAddress: req.GetCustomer().GetAddress(),
		CustomerEmail:   req.GetCustomer().GetEmail(),
		CustomerUStIDNr: req.GetCustomer().GetUstIdNr(),
		TaxMode:         taxModeFromProto(req.GetTaxMode()),
		LineItems:       lineItems,
		Notes:           req.GetNotes(),
		UserID:          userID,
	}

	if req.GetInvoiceDate() != "" {
		t, parseErr := time.Parse("2006-01-02", req.GetInvoiceDate())
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid invoice_date")
		}
		input.InvoiceDate = t
	} else {
		input.InvoiceDate = time.Now()
	}

	if req.GetDeliveryDate() != "" {
		t, parseErr := time.Parse("2006-01-02", req.GetDeliveryDate())
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid delivery_date")
		}
		input.DeliveryDate = &t
	}

	// PaymentTerms is a string like "30 Tage netto" in the proto
	// Extract days if possible, otherwise use the default
	if req.GetPaymentTerms() != "" {
		// Try to extract number from payment terms string
		var days int
		if _, scanErr := fmt.Sscanf(req.GetPaymentTerms(), "%d", &days); scanErr == nil && days > 0 {
			input.PaymentTermsDays = days
		}
	}

	if req.GetSourceQuoteId() != "" {
		sqid, sqErr := uuid.Parse(req.GetSourceQuoteId())
		if sqErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid source_quote_id")
		}
		input.SourceQuoteID = &sqid
	}

	if cid := req.GetContactId(); cid != "" {
		parsed, parseErr := uuid.Parse(cid)
		if parseErr != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid contact_id: %v", parseErr)
		}
		input.ContactID = &parsed
	}

	inv, err := s.invoiceService.Create(ctx, input)
	if err != nil {
		return nil, mapBizError(err)
	}

	return &bizv1.CreateInvoiceResponse{Invoice: toProtoInvoice(inv)}, nil
}

func (s *BizGRPCServer) GetInvoice(ctx context.Context, req *bizv1.GetInvoiceRequest) (*bizv1.GetInvoiceResponse, error) {
	tenantID, id, err := parseTenantAndID(req.GetTenantId(), req.GetId())
	if err != nil {
		return nil, err
	}

	inv, err := s.invoiceService.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, mapBizError(err)
	}

	return &bizv1.GetInvoiceResponse{Invoice: toProtoInvoice(inv)}, nil
}

func (s *BizGRPCServer) ListInvoices(ctx context.Context, req *bizv1.ListInvoicesRequest) (*bizv1.ListInvoicesResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}

	limit, offset := pageToLimitOffset(req.GetPage(), req.GetPerPage())

	filter := invoice.ListFilter{
		Status: invoiceStatusFromProto(req.GetStatus()),
		Limit:  limit,
		Offset: offset,
	}
	if cid := req.GetContactId(); cid != "" {
		parsed, parseErr := uuid.Parse(cid)
		if parseErr != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid contact_id: %v", parseErr)
		}
		filter.ContactID = &parsed
	}

	invoices, total, err := s.invoiceService.List(ctx, tenantID, filter)
	if err != nil {
		return nil, mapBizError(err)
	}

	protoInvoices := make([]*bizv1.Invoice, 0, len(invoices))
	for _, inv := range invoices {
		protoInvoices = append(protoInvoices, toProtoInvoice(inv))
	}

	return &bizv1.ListInvoicesResponse{Invoices: protoInvoices, Total: int32(total)}, nil
}

func (s *BizGRPCServer) UpdateInvoice(ctx context.Context, req *bizv1.UpdateInvoiceRequest) (*bizv1.UpdateInvoiceResponse, error) {
	tenantID, id, err := parseTenantAndID(req.GetTenantId(), req.GetId())
	if err != nil {
		return nil, err
	}

	input := invoice.UpdateInput{}

	if req.Customer != nil {
		name := req.Customer.GetName()
		input.CustomerName = &name
		addr := req.Customer.GetAddress()
		input.CustomerAddress = &addr
		email := req.Customer.GetEmail()
		input.CustomerEmail = &email
		ustid := req.Customer.GetUstIdNr()
		input.CustomerUStIDNr = &ustid
	}

	if req.GetTaxMode() != bizv1.TaxMode_TAX_MODE_UNSPECIFIED {
		mode := taxModeFromProto(req.GetTaxMode())
		input.TaxMode = &mode
	}

	if len(req.GetLineItems()) > 0 {
		input.LineItems = protoLineItemsToModel(req.GetLineItems())
	}

	if req.GetNotes() != "" {
		notes := req.GetNotes()
		input.Notes = &notes
	}

	inv, err := s.invoiceService.Update(ctx, tenantID, id, input)
	if err != nil {
		return nil, mapBizError(err)
	}

	return &bizv1.UpdateInvoiceResponse{Invoice: toProtoInvoice(inv)}, nil
}

func (s *BizGRPCServer) SendInvoice(ctx context.Context, req *bizv1.SendInvoiceRequest) (*bizv1.SendInvoiceResponse, error) {
	tenantID, id, err := parseTenantAndID(req.GetTenantId(), req.GetId())
	if err != nil {
		return nil, err
	}

	// Proto doesn't include user_id; use uuid.Nil (gateway handles auth)
	if err := s.invoiceService.Send(ctx, tenantID, id, uuid.Nil); err != nil {
		return nil, mapBizError(err)
	}

	inv, err := s.invoiceService.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, mapBizError(err)
	}

	return &bizv1.SendInvoiceResponse{Invoice: toProtoInvoice(inv)}, nil
}

func (s *BizGRPCServer) MarkInvoicePaid(ctx context.Context, req *bizv1.MarkInvoicePaidRequest) (*bizv1.MarkInvoicePaidResponse, error) {
	tenantID, id, err := parseTenantAndID(req.GetTenantId(), req.GetId())
	if err != nil {
		return nil, err
	}

	if err := s.invoiceService.MarkPaid(ctx, tenantID, id); err != nil {
		return nil, mapBizError(err)
	}

	inv, err := s.invoiceService.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, mapBizError(err)
	}

	return &bizv1.MarkInvoicePaidResponse{Invoice: toProtoInvoice(inv)}, nil
}

func (s *BizGRPCServer) CancelInvoice(ctx context.Context, req *bizv1.CancelInvoiceRequest) (*bizv1.CancelInvoiceResponse, error) {
	tenantID, id, err := parseTenantAndID(req.GetTenantId(), req.GetId())
	if err != nil {
		return nil, err
	}

	// Proto doesn't include user_id; use uuid.Nil (gateway handles auth)
	if err := s.invoiceService.Cancel(ctx, tenantID, id, uuid.Nil); err != nil {
		return nil, mapBizError(err)
	}

	inv, err := s.invoiceService.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, mapBizError(err)
	}

	return &bizv1.CancelInvoiceResponse{Invoice: toProtoInvoice(inv)}, nil
}

// ============================================================================
// Credit Notes
// ============================================================================

func (s *BizGRPCServer) CreateCreditNote(ctx context.Context, req *bizv1.CreateCreditNoteRequest) (*bizv1.CreateCreditNoteResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}

	userID, err := uuid.Parse(req.GetCreatedBy())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid created_by")
	}

	invoiceID, err := uuid.Parse(req.GetOriginalInvoiceId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid original_invoice_id")
	}

	lineItems := protoLineItemsToModel(req.GetLineItems())

	input := creditnote.CreateInput{
		TenantID:          tenantID,
		OriginalInvoiceID: invoiceID,
		LineItems:         lineItems,
		TaxMode:           taxModeFromProto(req.GetTaxMode()),
		Reason:            req.GetReason(),
		UserID:            userID,
	}

	cn, err := s.creditNoteService.Create(ctx, input)
	if err != nil {
		return nil, mapBizError(err)
	}

	return &bizv1.CreateCreditNoteResponse{CreditNote: toProtoCreditNote(cn)}, nil
}

func (s *BizGRPCServer) GetCreditNote(ctx context.Context, req *bizv1.GetCreditNoteRequest) (*bizv1.GetCreditNoteResponse, error) {
	tenantID, id, err := parseTenantAndID(req.GetTenantId(), req.GetId())
	if err != nil {
		return nil, err
	}

	cn, err := s.creditNoteService.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, mapBizError(err)
	}

	return &bizv1.GetCreditNoteResponse{CreditNote: toProtoCreditNote(cn)}, nil
}

func (s *BizGRPCServer) ListCreditNotes(ctx context.Context, req *bizv1.ListCreditNotesRequest) (*bizv1.ListCreditNotesResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}

	limit, offset := pageToLimitOffset(req.GetPage(), req.GetPerPage())

	filter := creditnote.ListFilter{
		Limit:  limit,
		Offset: offset,
	}

	if req.GetInvoiceId() != "" {
		invID, invErr := uuid.Parse(req.GetInvoiceId())
		if invErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid invoice_id")
		}
		filter.OriginalInvID = &invID
	}

	notes, total, err := s.creditNoteService.List(ctx, tenantID, filter)
	if err != nil {
		return nil, mapBizError(err)
	}

	protoNotes := make([]*bizv1.CreditNote, 0, len(notes))
	for _, cn := range notes {
		protoNotes = append(protoNotes, toProtoCreditNote(cn))
	}

	return &bizv1.ListCreditNotesResponse{CreditNotes: protoNotes, Total: int32(total)}, nil
}

func (s *BizGRPCServer) SendCreditNote(ctx context.Context, req *bizv1.SendCreditNoteRequest) (*bizv1.SendCreditNoteResponse, error) {
	tenantID, id, err := parseTenantAndID(req.GetTenantId(), req.GetId())
	if err != nil {
		return nil, err
	}

	// Proto doesn't include user_id; use uuid.Nil (gateway handles auth)
	if err := s.creditNoteService.Send(ctx, tenantID, id, uuid.Nil); err != nil {
		return nil, mapBizError(err)
	}

	cn, err := s.creditNoteService.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, mapBizError(err)
	}

	return &bizv1.SendCreditNoteResponse{CreditNote: toProtoCreditNote(cn)}, nil
}

// ============================================================================
// Payments
// ============================================================================

func (s *BizGRPCServer) RecordPayment(ctx context.Context, req *bizv1.RecordPaymentRequest) (*bizv1.RecordPaymentResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}

	invoiceID, err := uuid.Parse(req.GetInvoiceId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid invoice_id")
	}

	userID, err := uuid.Parse(req.GetCreatedBy())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid created_by")
	}

	amount, err := decimal.NewFromString(req.GetAmount())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid amount")
	}

	paymentDate := time.Now()
	if req.GetPaymentDate() != "" {
		pd, parseErr := time.Parse("2006-01-02", req.GetPaymentDate())
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid payment_date")
		}
		paymentDate = pd
	}

	input := payment.RecordInput{
		TenantID:       tenantID,
		InvoiceID:      invoiceID,
		Amount:         amount,
		Date:           paymentDate,
		Method:         paymentMethodFromProto(req.GetMethod()),
		Reference:      req.GetReference(),
		Notes:          req.GetNotes(),
		UserID:         userID,
		IdempotencyKey: req.GetIdempotencyKey(),
	}

	p, err := s.paymentService.Record(ctx, input)
	if err != nil {
		return nil, mapBizError(err)
	}

	return &bizv1.RecordPaymentResponse{Payment: toProtoPayment(p)}, nil
}

func (s *BizGRPCServer) ListPayments(ctx context.Context, req *bizv1.ListPaymentsRequest) (*bizv1.ListPaymentsResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}

	invoiceID, err := uuid.Parse(req.GetInvoiceId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid invoice_id")
	}

	payments, err := s.paymentService.List(ctx, tenantID, invoiceID)
	if err != nil {
		return nil, mapBizError(err)
	}

	protoPayments := make([]*bizv1.Payment, 0, len(payments))
	for _, p := range payments {
		protoPayments = append(protoPayments, toProtoPayment(p))
	}

	return &bizv1.ListPaymentsResponse{Payments: protoPayments, Total: int32(len(payments))}, nil
}

func (s *BizGRPCServer) DeletePayment(ctx context.Context, req *bizv1.DeletePaymentRequest) (*bizv1.DeletePaymentResponse, error) {
	tenantID, id, err := parseTenantAndID(req.GetTenantId(), req.GetId())
	if err != nil {
		return nil, err
	}

	if err := s.paymentService.Delete(ctx, tenantID, id); err != nil {
		return nil, mapBizError(err)
	}

	return &bizv1.DeletePaymentResponse{}, nil
}

// ============================================================================
// Dunning
// ============================================================================

func (s *BizGRPCServer) ListDunnings(ctx context.Context, req *bizv1.ListDunningsRequest) (*bizv1.ListDunningsResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}

	limit, offset := pageToLimitOffset(req.GetPage(), req.GetPerPage())

	filter := dunning.ListFilter{
		Status: dunningStatusFromProto(req.GetStatus()),
		Limit:  limit,
		Offset: offset,
	}

	if req.GetInvoiceId() != "" {
		invID, invErr := uuid.Parse(req.GetInvoiceId())
		if invErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid invoice_id")
		}
		filter.InvoiceID = &invID
	}

	records, total, err := s.dunningService.List(ctx, tenantID, filter)
	if err != nil {
		return nil, mapBizError(err)
	}

	protoRecords := make([]*bizv1.DunningRecord, 0, len(records))
	for _, r := range records {
		protoRecords = append(protoRecords, toProtoDunningRecord(r))
	}

	return &bizv1.ListDunningsResponse{Dunnings: protoRecords, Total: int32(total)}, nil
}

func (s *BizGRPCServer) CreateDunning(ctx context.Context, req *bizv1.CreateDunningRequest) (*bizv1.CreateDunningResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}

	// DetectAndCreateDunnings auto-detects overdue invoices and creates appropriate dunning drafts.
	records, err := s.dunningService.DetectAndCreateDunnings(ctx, tenantID)
	if err != nil {
		return nil, mapBizError(err)
	}

	// If a specific invoice was requested, find the dunning for that invoice
	if req.GetInvoiceId() != "" {
		invoiceID, invErr := uuid.Parse(req.GetInvoiceId())
		if invErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid invoice_id")
		}
		for _, r := range records {
			if r.InvoiceID == invoiceID {
				return &bizv1.CreateDunningResponse{Dunning: toProtoDunningRecord(r)}, nil
			}
		}
		return nil, status.Error(codes.NotFound, "no dunning created for specified invoice")
	}

	if len(records) > 0 {
		return &bizv1.CreateDunningResponse{Dunning: toProtoDunningRecord(records[0])}, nil
	}

	return nil, status.Error(codes.NotFound, "no overdue invoices found for dunning creation")
}

func (s *BizGRPCServer) SendDunning(ctx context.Context, req *bizv1.SendDunningRequest) (*bizv1.SendDunningResponse, error) {
	tenantID, id, err := parseTenantAndID(req.GetTenantId(), req.GetId())
	if err != nil {
		return nil, err
	}

	// Proto doesn't include user_id; use uuid.Nil (gateway handles auth)
	if err := s.dunningService.Send(ctx, tenantID, id, uuid.Nil); err != nil {
		return nil, mapBizError(err)
	}

	// Re-fetch by ID since Send() only returns an error. Fetching by ID (not a
	// List(Limit:1), which never reliably hits the sent record) guarantees the
	// response carries the dunning that was just sent (status, sent_at populated).
	record, fetchErr := s.dunningService.GetByID(ctx, tenantID, id)
	if fetchErr != nil {
		return nil, mapBizError(fetchErr)
	}

	return &bizv1.SendDunningResponse{Dunning: toProtoDunningRecord(record)}, nil
}

func (s *BizGRPCServer) EscalateDunning(ctx context.Context, req *bizv1.EscalateDunningRequest) (*bizv1.EscalateDunningResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}

	invoiceID, err := uuid.Parse(req.GetInvoiceId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid invoice_id")
	}

	// Escalation is handled by DetectAndCreateDunnings which checks existing levels
	records, err := s.dunningService.DetectAndCreateDunnings(ctx, tenantID)
	if err != nil {
		return nil, mapBizError(err)
	}

	for _, r := range records {
		if r.InvoiceID == invoiceID {
			return &bizv1.EscalateDunningResponse{Dunning: toProtoDunningRecord(r)}, nil
		}
	}

	return nil, status.Error(codes.FailedPrecondition, "no escalation available for this invoice")
}

func (s *BizGRPCServer) GetDunningConfig(ctx context.Context, req *bizv1.GetDunningConfigRequest) (*bizv1.GetDunningConfigResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}

	config, err := s.dunningService.GetConfig(ctx, tenantID)
	if err != nil {
		return nil, mapBizError(err)
	}

	return &bizv1.GetDunningConfigResponse{Config: toProtoDunningConfig(config)}, nil
}

func (s *BizGRPCServer) UpdateDunningConfig(ctx context.Context, req *bizv1.UpdateDunningConfigRequest) (*bizv1.UpdateDunningConfigResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}

	pc := req.GetConfig()
	if pc == nil {
		return nil, status.Error(codes.InvalidArgument, "config is required")
	}

	input := dunning.UpdateConfigInput{}

	if pc.GetLevel1DaysAfterDue() > 0 {
		days := int(pc.GetLevel1DaysAfterDue())
		input.Level1DaysAfterDue = &days
	}
	if pc.GetLevel2DaysAfterLevel1() > 0 {
		days := int(pc.GetLevel2DaysAfterLevel1())
		input.Level2DaysAfterLevel1 = &days
	}
	if pc.GetLevel3DaysAfterLevel2() > 0 {
		days := int(pc.GetLevel3DaysAfterLevel2())
		input.Level3DaysAfterLevel2 = &days
	}

	if pc.GetLevel1Fee() != "" {
		fee, fErr := decimal.NewFromString(pc.GetLevel1Fee())
		if fErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid level1_fee")
		}
		input.Level1Fee = &fee
	}
	if pc.GetLevel2Fee() != "" {
		fee, fErr := decimal.NewFromString(pc.GetLevel2Fee())
		if fErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid level2_fee")
		}
		input.Level2Fee = &fee
	}
	if pc.GetLevel3Fee() != "" {
		fee, fErr := decimal.NewFromString(pc.GetLevel3Fee())
		if fErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid level3_fee")
		}
		input.Level3Fee = &fee
	}

	updated, err := s.dunningService.UpdateConfig(ctx, tenantID, input)
	if err != nil {
		return nil, mapBizError(err)
	}

	return &bizv1.UpdateDunningConfigResponse{Config: toProtoDunningConfig(updated)}, nil
}

// ============================================================================
// Dashboard
// ============================================================================

func (s *BizGRPCServer) GetFinanceDashboard(ctx context.Context, req *bizv1.GetFinanceDashboardRequest) (*bizv1.GetFinanceDashboardResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}

	now := time.Now()
	dateFrom := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
	dateTo := now

	dash, err := s.dashboardService.GetDashboard(ctx, tenantID, dateFrom, dateTo)
	if err != nil {
		return nil, mapBizError(err)
	}

	return &bizv1.GetFinanceDashboardResponse{Dashboard: toProtoDashboard(dash)}, nil
}

// ============================================================================
// DATEV Export
// ============================================================================

func (s *BizGRPCServer) ExportDATEV(ctx context.Context, req *bizv1.ExportDATEVRequest) (*bizv1.ExportDATEVResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}

	startDate, err := time.Parse("2006-01-02", req.GetStartDate())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid start_date")
	}

	endDate, err := time.Parse("2006-01-02", req.GetEndDate())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid end_date")
	}

	fiscalYearStart := startDate
	if req.GetFiscalYearStart() != "" {
		fys, fysErr := time.Parse("2006-01-02", req.GetFiscalYearStart())
		if fysErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid fiscal_year_start")
		}
		fiscalYearStart = fys
	}

	// Fetch invoices and credit notes in date range
	invoices, _, invErr := s.invoiceService.List(ctx, tenantID, invoice.ListFilter{
		DateFrom: &startDate,
		DateTo:   &endDate,
		Limit:    10000,
	})
	if invErr != nil {
		return nil, mapBizError(invErr)
	}

	creditNotes, _, cnErr := s.creditNoteService.List(ctx, tenantID, creditnote.ListFilter{
		Limit: 10000,
	})
	if cnErr != nil {
		return nil, mapBizError(cnErr)
	}

	csvData, err := s.datevExporter.Export(invoices, creditNotes, fiscalYearStart, time.Now())
	if err != nil {
		return nil, status.Error(codes.Internal, "datev export failed: "+err.Error())
	}

	// Count booking records
	recordCount := int32(0)
	for _, inv := range invoices {
		if inv.Status == models.InvoiceStatusSent || inv.Status == models.InvoiceStatusPaid || inv.Status == models.InvoiceStatusOverdue {
			var lineItems []models.LineItem
			_ = json.Unmarshal(inv.LineItems, &lineItems)
			recordCount += int32(len(lineItems))
		}
	}
	for _, cn := range creditNotes {
		if cn.Status == models.CreditNoteStatusSent {
			var lineItems []models.LineItem
			_ = json.Unmarshal(cn.LineItems, &lineItems)
			recordCount += int32(len(lineItems))
		}
	}

	filename := fmt.Sprintf("DATEV_Buchungsstapel_%s_%s.csv", req.GetStartDate(), req.GetEndDate())

	return &bizv1.ExportDATEVResponse{
		CsvData:     csvData,
		Filename:    filename,
		RecordCount: recordCount,
	}, nil
}

// ============================================================================
// PDF Generation
// ============================================================================

func (s *BizGRPCServer) GenerateQuotePDF(ctx context.Context, req *bizv1.GenerateQuotePDFRequest) (*bizv1.GenerateQuotePDFResponse, error) {
	tenantID, id, err := parseTenantAndID(req.GetTenantId(), req.GetId())
	if err != nil {
		return nil, err
	}

	q, err := s.quoteService.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, mapBizError(err)
	}

	settings, err := s.requireCompanySettings(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	gen := pdf.NewGenerator(*settings)
	pdfBytes, err := gen.GenerateQuotePDF(*q)
	if err != nil {
		slog.Error("failed to generate quote PDF", "quote_id", id, "error", err)
		return nil, status.Error(codes.Internal, "PDF generation failed")
	}

	filename := fmt.Sprintf("Angebot_%s.pdf", q.QuoteNumber)
	return &bizv1.GenerateQuotePDFResponse{
		PdfData:  pdfBytes,
		Filename: filename,
	}, nil
}

func (s *BizGRPCServer) GenerateInvoicePDF(ctx context.Context, req *bizv1.GenerateInvoicePDFRequest) (*bizv1.GenerateInvoicePDFResponse, error) {
	tenantID, id, err := parseTenantAndID(req.GetTenantId(), req.GetId())
	if err != nil {
		return nil, err
	}

	inv, err := s.invoiceService.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, mapBizError(err)
	}

	settings, err := s.requireCompanySettings(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	gen := pdf.NewGenerator(*settings)
	pdfBytes, err := gen.GenerateInvoicePDF(*inv)
	if err != nil {
		slog.Error("failed to generate invoice PDF", "invoice_id", id, "error", err)
		return nil, status.Error(codes.Internal, "PDF generation failed")
	}

	filename := fmt.Sprintf("Rechnung_%s.pdf", inv.InvoiceNumber)
	return &bizv1.GenerateInvoicePDFResponse{
		PdfData:  pdfBytes,
		Filename: filename,
	}, nil
}

func (s *BizGRPCServer) GenerateCreditNotePDF(ctx context.Context, req *bizv1.GenerateCreditNotePDFRequest) (*bizv1.GenerateCreditNotePDFResponse, error) {
	tenantID, id, err := parseTenantAndID(req.GetTenantId(), req.GetId())
	if err != nil {
		return nil, err
	}

	cn, err := s.creditNoteService.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, mapBizError(err)
	}

	settings, err := s.requireCompanySettings(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	gen := pdf.NewGenerator(*settings)
	pdfBytes, err := gen.GenerateCreditNotePDF(*cn)
	if err != nil {
		slog.Error("failed to generate credit note PDF", "credit_note_id", id, "error", err)
		return nil, status.Error(codes.Internal, "PDF generation failed")
	}

	filename := fmt.Sprintf("Gutschrift_%s.pdf", cn.CreditNoteNumber)
	return &bizv1.GenerateCreditNotePDFResponse{
		PdfData:  pdfBytes,
		Filename: filename,
	}, nil
}

func (s *BizGRPCServer) GenerateDunningPDF(ctx context.Context, req *bizv1.GenerateDunningPDFRequest) (*bizv1.GenerateDunningPDFResponse, error) {
	tenantID, id, err := parseTenantAndID(req.GetTenantId(), req.GetId())
	if err != nil {
		return nil, err
	}

	dr, err := s.dunningService.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, mapBizError(err)
	}

	// Dunning PDF requires the linked invoice for customer data and amounts
	inv, err := s.invoiceService.GetByID(ctx, tenantID, dr.InvoiceID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to load linked invoice for dunning PDF")
	}

	settings, err := s.requireCompanySettings(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	gen := pdf.NewGenerator(*settings)
	pdfBytes, err := gen.GenerateDunningPDF(*dr, *inv, dr.Level)
	if err != nil {
		slog.Error("failed to generate dunning PDF", "dunning_id", id, "error", err)
		return nil, status.Error(codes.Internal, "PDF generation failed")
	}

	// Filename based on level: Zahlungserinnerung for level 1, Mahnung for 2+
	var docPrefix string
	switch dr.Level {
	case 1:
		docPrefix = "Zahlungserinnerung"
	case 2:
		docPrefix = "1_Mahnung"
	default:
		docPrefix = "2_Mahnung"
	}
	filename := fmt.Sprintf("%s_%s.pdf", docPrefix, inv.InvoiceNumber)
	return &bizv1.GenerateDunningPDFResponse{
		PdfData:  pdfBytes,
		Filename: filename,
	}, nil
}

func (s *BizGRPCServer) GenerateZUGFeRDInvoicePDF(ctx context.Context, req *bizv1.GenerateZUGFeRDInvoicePDFRequest) (*bizv1.GenerateZUGFeRDInvoicePDFResponse, error) {
	tenantID, id, err := parseTenantAndID(req.GetTenantId(), req.GetId())
	if err != nil {
		return nil, err
	}

	inv, err := s.invoiceService.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, mapBizError(err)
	}

	settings, err := s.requireCompanySettings(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	// Generate plain PDF first — used as fallback on ZUGFeRD failure.
	gen := pdf.NewGenerator(*settings)
	plainPDF, err := gen.GenerateInvoicePDF(*inv)
	if err != nil {
		slog.Error("failed to generate invoice PDF for zugferd", "invoice_id", id, "error", err)
		return nil, status.Error(codes.Internal, "PDF generation failed")
	}

	plainFilename := fmt.Sprintf("Rechnung_%s.pdf", inv.InvoiceNumber)

	// Generate ZUGFeRD XML (Factur-X EN16931 profile).
	xmlBytes, err := pdf.GenerateZUGFeRDXML(*inv, *settings)
	if err != nil {
		slog.Error("zugferd xml generation failed, returning plain PDF", "invoice_id", id, "error", err)
		return &bizv1.GenerateZUGFeRDInvoicePDFResponse{
			PdfData:  plainPDF,
			Filename: plainFilename,
		}, nil
	}

	// Embed the XML attachment into the PDF.
	zugferdPDF, err := pdf.EmbedZUGFeRDXML(plainPDF, xmlBytes, inv.InvoiceNumber)
	if err != nil {
		slog.Error("zugferd embed failed, returning plain PDF", "invoice_id", id, "error", err)
		return &bizv1.GenerateZUGFeRDInvoicePDFResponse{
			PdfData:  plainPDF,
			Filename: plainFilename,
		}, nil
	}

	filename := fmt.Sprintf("factur-x_%s.pdf", inv.InvoiceNumber)
	return &bizv1.GenerateZUGFeRDInvoicePDFResponse{
		PdfData:  zugferdPDF,
		Filename: filename,
	}, nil
}

// ============================================================================
// Proto Conversion Helpers
// ============================================================================

func toProtoCompanySettings(s *models.CompanySettings) *bizv1.CompanySettings {
	if s == nil {
		return nil
	}
	return &bizv1.CompanySettings{
		Id:                       s.ID.String(),
		TenantId:                 s.TenantID.String(),
		Name:                     s.Name,
		Street:                   s.Street,
		Plz:                      s.PLZ,
		City:                     s.City,
		Country:                  s.Country,
		Steuernummer:             s.Steuernummer,
		UstIdNr:                  s.UStIDNr,
		Handelsregister:          s.Handelsregister,
		BankName:                 s.BankName,
		Iban:                     s.IBAN,
		Bic:                      s.BIC,
		LogoUrl:                  s.LogoURL,
		AccentColor:              s.AccentColor,
		IsKleinunternehmer:       s.IsKleinunternehmer,
		DefaultPaymentTermsDays:  int32(s.DefaultPaymentTermsDays),
		DefaultQuoteValidityDays: int32(s.DefaultQuoteValidityDays),
		Basiszinssatz:            s.Basiszinssatz.String(),
	}
}

func toProtoQuote(q *models.Quote) *bizv1.Quote {
	if q == nil {
		return nil
	}

	lineItems := parseProtoLineItems(q.LineItems)
	taxBreakdown := parseProtoTaxBreakdown(q.TaxBreakdownRaw)

	pq := &bizv1.Quote{
		Id:           q.ID.String(),
		QuoteNumber:  q.QuoteNumber,
		Status:       quoteStatusToProto(q.Status),
		Customer:     &bizv1.CustomerSnapshot{Name: q.CustomerName, Address: q.CustomerAddress, Email: q.CustomerEmail, UstIdNr: q.CustomerUStIDNr},
		LineItems:    lineItems,
		TaxMode:      taxModeToProto(q.TaxMode),
		TaxBreakdown: taxBreakdown,
		Notes:        q.Notes,
		TenantId:     q.TenantID.String(),
		CreatedBy:    q.CreatedBy.String(),
		CreatedAt:    timestamppb.New(q.CreatedAt),
		UpdatedAt:    timestamppb.New(q.UpdatedAt),
	}

	if q.ValidUntil != nil {
		pq.ValidUntil = q.ValidUntil.Format("2006-01-02")
	}
	if q.DealID != nil {
		pq.DealId = q.DealID.String()
	}
	if q.SourceQuoteID != nil {
		pq.SourceQuoteId = q.SourceQuoteID.String()
	}

	return pq
}

func toProtoInvoice(inv *models.Invoice) *bizv1.Invoice {
	if inv == nil {
		return nil
	}

	lineItems := parseProtoLineItems(inv.LineItems)
	taxBreakdown := parseProtoTaxBreakdown(inv.TaxBreakdownRaw)

	pi := &bizv1.Invoice{
		Id:            inv.ID.String(),
		InvoiceNumber: inv.InvoiceNumber,
		Status:        invoiceStatusToProto(inv.Status),
		Customer:      &bizv1.CustomerSnapshot{Name: inv.CustomerName, Address: inv.CustomerAddress, Email: inv.CustomerEmail, UstIdNr: inv.CustomerUStIDNr},
		LineItems:     lineItems,
		TaxMode:       taxModeToProto(inv.TaxMode),
		TaxBreakdown:  taxBreakdown,
		InvoiceDate:   inv.InvoiceDate.Format("2006-01-02"),
		DueDate:       inv.DueDate.Format("2006-01-02"),
		PaymentTerms:  inv.PaymentTerms,
		SnapshotData:  inv.SnapshotData,
		Notes:         inv.Notes,
		TenantId:      inv.TenantID.String(),
		CreatedBy:     inv.CreatedBy.String(),
		CreatedAt:     timestamppb.New(inv.CreatedAt),
		UpdatedAt:     timestamppb.New(inv.UpdatedAt),
	}

	if inv.DeliveryDate != nil {
		pi.DeliveryDate = inv.DeliveryDate.Format("2006-01-02")
	}
	if inv.SourceQuoteID != nil {
		pi.SourceQuoteId = inv.SourceQuoteID.String()
	}

	if len(inv.CompanySnapshotRaw) > 0 {
		var cs models.CompanySnapshot
		if err := json.Unmarshal(inv.CompanySnapshotRaw, &cs); err == nil {
			pi.Company = &bizv1.CompanySnapshot{
				Name:            cs.Name,
				Street:          cs.Street,
				Plz:             cs.PLZ,
				City:            cs.City,
				Country:         cs.Country,
				Steuernummer:    cs.Steuernummer,
				UstIdNr:         cs.UStIDNr,
				Handelsregister: cs.Handelsregister,
				BankName:        cs.BankName,
				Iban:            cs.IBAN,
				Bic:             cs.BIC,
				LogoUrl:         cs.LogoURL,
				AccentColor:     cs.AccentColor,
			}
		}
	}

	return pi
}

func toProtoCreditNote(cn *models.CreditNote) *bizv1.CreditNote {
	if cn == nil {
		return nil
	}

	lineItems := parseProtoLineItems(cn.LineItems)
	taxBreakdown := parseProtoTaxBreakdown(cn.TaxBreakdownRaw)

	return &bizv1.CreditNote{
		Id:                cn.ID.String(),
		CreditNoteNumber:  cn.CreditNoteNumber,
		Status:            creditNoteStatusToProto(cn.Status),
		OriginalInvoiceId: cn.OriginalInvoiceID.String(),
		Customer:          &bizv1.CustomerSnapshot{Name: cn.CustomerName, Address: cn.CustomerAddress, Email: cn.CustomerEmail, UstIdNr: cn.CustomerUStIDNr},
		LineItems:         lineItems,
		TaxMode:           taxModeToProto(cn.TaxMode),
		TaxBreakdown:      taxBreakdown,
		Reason:            cn.Reason,
		TenantId:          cn.TenantID.String(),
		CreatedBy:         cn.CreatedBy.String(),
		CreatedAt:         timestamppb.New(cn.CreatedAt),
		UpdatedAt:         timestamppb.New(cn.UpdatedAt),
	}
}

func toProtoPayment(p *models.Payment) *bizv1.Payment {
	if p == nil {
		return nil
	}

	return &bizv1.Payment{
		Id:          p.ID.String(),
		InvoiceId:   p.InvoiceID.String(),
		Amount:      p.Amount.String(),
		PaymentDate: p.PaymentDate.Format("2006-01-02"),
		Method:      paymentMethodToProto(p.Method),
		Reference:   p.Reference,
		Notes:       p.Notes,
		TenantId:    p.TenantID.String(),
		CreatedBy:   p.CreatedBy.String(),
		CreatedAt:   timestamppb.New(p.CreatedAt),
	}
}

func toProtoDunningRecord(r *models.DunningRecord) *bizv1.DunningRecord {
	if r == nil {
		return nil
	}

	dr := &bizv1.DunningRecord{
		Id:        r.ID.String(),
		InvoiceId: r.InvoiceID.String(),
		Level:     int32(r.Level),
		Status:    dunningStatusToProto(r.Status),
		Fee:       r.Fee.String(),
		Interest:  r.Interest.String(),
		TenantId:  r.TenantID.String(),
		CreatedBy: r.CreatedBy.String(),
		CreatedAt: timestamppb.New(r.CreatedAt),
	}

	if r.SentAt != nil {
		dr.SentAt = timestamppb.New(*r.SentAt)
	}

	return dr
}

func toProtoDunningConfig(c *models.DunningConfig) *bizv1.DunningConfig {
	if c == nil {
		return nil
	}

	return &bizv1.DunningConfig{
		Id:                    c.ID.String(),
		TenantId:              c.TenantID.String(),
		Level1DaysAfterDue:    int32(c.Level1DaysAfterDue),
		Level2DaysAfterLevel1: int32(c.Level2DaysAfterLevel1),
		Level3DaysAfterLevel2: int32(c.Level3DaysAfterLevel2),
		Level1Fee:             c.Level1Fee.String(),
		Level2Fee:             c.Level2Fee.String(),
		Level3Fee:             c.Level3Fee.String(),
		UpdatedAt:             timestamppb.New(c.UpdatedAt),
	}
}

func toProtoDashboard(d *models.FinanceDashboard) *bizv1.FinanceDashboard {
	if d == nil {
		return nil
	}

	statusBreakdown := map[string]int32{
		"draft":     int32(d.StatusBreakdown.Draft),
		"sent":      int32(d.StatusBreakdown.Sent),
		"overdue":   int32(d.StatusBreakdown.Overdue),
		"paid":      int32(d.StatusBreakdown.Paid),
		"cancelled": int32(d.StatusBreakdown.Cancelled),
	}

	fd := &bizv1.FinanceDashboard{
		TotalInvoiced:    d.Revenue.TotalInvoiced.String(),
		TotalPaid:        d.Revenue.TotalPaid.String(),
		TotalOutstanding: d.Revenue.TotalOutstanding.String(),
		OverdueAmount:    d.Revenue.OverdueAmount.String(),
		QuotesPending:    int32(d.Pipeline.QuotesPending),
		ConversionRate:   d.Pipeline.ConversionRate.String(),
		AverageDealSize:  d.Pipeline.AverageDealSize.String(),
		RevenueForecast:  d.RevenueForecast.String(),
		StatusBreakdown:  statusBreakdown,
	}

	for _, inv := range d.RecentInvoices {
		fd.RecentInvoices = append(fd.RecentInvoices, toProtoInvoice(inv))
	}
	for _, q := range d.ExpiringQuotes {
		fd.ExpiringQuotes = append(fd.ExpiringQuotes, toProtoQuote(q))
	}
	for _, dr := range d.PendingDunnings {
		fd.PendingDunnings = append(fd.PendingDunnings, toProtoDunningRecord(dr))
	}

	return fd
}

// ============================================================================
// Proto Line Item / Tax Breakdown Helpers
// ============================================================================

func protoLineItemsToModel(items []*bizv1.LineItem) []models.LineItem {
	result := make([]models.LineItem, 0, len(items))
	for _, item := range items {
		qty, _ := decimal.NewFromString(item.GetQuantity())
		price, _ := decimal.NewFromString(item.GetUnitPrice())
		rate, _ := decimal.NewFromString(item.GetTaxRate())
		lineTotal, _ := decimal.NewFromString(item.GetLineTotal())

		result = append(result, models.LineItem{
			ID:          item.GetId(),
			Position:    int(item.GetPosition()),
			Description: item.GetDescription(),
			Quantity:    qty,
			UnitPrice:   price,
			TaxRate:     rate,
			LineTotal:   lineTotal,
		})
	}
	return result
}

func parseProtoLineItems(raw json.RawMessage) []*bizv1.LineItem {
	if len(raw) == 0 {
		return nil
	}

	var items []models.LineItem
	if err := json.Unmarshal(raw, &items); err != nil {
		slog.Warn("failed to parse line items for proto conversion", "error", err)
		return nil
	}

	result := make([]*bizv1.LineItem, 0, len(items))
	for _, item := range items {
		result = append(result, &bizv1.LineItem{
			Id:          item.ID,
			Position:    int32(item.Position),
			Description: item.Description,
			Quantity:    item.Quantity.String(),
			UnitPrice:   item.UnitPrice.String(),
			TaxRate:     item.TaxRate.String(),
			LineTotal:   item.LineTotal.String(),
		})
	}
	return result
}

func parseProtoTaxBreakdown(raw json.RawMessage) *bizv1.TaxBreakdown {
	if len(raw) == 0 {
		return nil
	}

	var tb models.TaxBreakdown
	if err := json.Unmarshal(raw, &tb); err != nil {
		slog.Warn("failed to parse tax breakdown for proto conversion", "error", err)
		return nil
	}

	taxByRate := make(map[string]string, len(tb.TaxByRate))
	for k, v := range tb.TaxByRate {
		taxByRate[k] = v.String()
	}

	return &bizv1.TaxBreakdown{
		Subtotal:   tb.Subtotal.String(),
		TaxByRate:  taxByRate,
		TotalTax:   tb.TotalTax.String(),
		GrossTotal: tb.GrossTotal.String(),
	}
}

// ============================================================================
// Enum Conversion Helpers
// ============================================================================

func taxModeFromProto(m bizv1.TaxMode) string {
	switch m {
	case bizv1.TaxMode_TAX_MODE_REVERSE_CHARGE:
		return models.TaxModeReverseCharge
	case bizv1.TaxMode_TAX_MODE_KLEINUNTERNEHMER:
		return models.TaxModeKleinunternehmer
	default:
		return models.TaxModeStandard
	}
}

func taxModeToProto(m string) bizv1.TaxMode {
	switch m {
	case models.TaxModeReverseCharge:
		return bizv1.TaxMode_TAX_MODE_REVERSE_CHARGE
	case models.TaxModeKleinunternehmer:
		return bizv1.TaxMode_TAX_MODE_KLEINUNTERNEHMER
	default:
		return bizv1.TaxMode_TAX_MODE_STANDARD
	}
}

func quoteStatusFromProto(s bizv1.QuoteStatus) string {
	switch s {
	case bizv1.QuoteStatus_QUOTE_DRAFT:
		return models.QuoteStatusDraft
	case bizv1.QuoteStatus_QUOTE_SENT:
		return models.QuoteStatusSent
	case bizv1.QuoteStatus_QUOTE_ACCEPTED:
		return models.QuoteStatusAccepted
	case bizv1.QuoteStatus_QUOTE_REJECTED:
		return models.QuoteStatusRejected
	case bizv1.QuoteStatus_QUOTE_EXPIRED:
		return models.QuoteStatusExpired
	default:
		return ""
	}
}

func quoteStatusToProto(s string) bizv1.QuoteStatus {
	switch s {
	case models.QuoteStatusDraft:
		return bizv1.QuoteStatus_QUOTE_DRAFT
	case models.QuoteStatusSent:
		return bizv1.QuoteStatus_QUOTE_SENT
	case models.QuoteStatusAccepted:
		return bizv1.QuoteStatus_QUOTE_ACCEPTED
	case models.QuoteStatusRejected:
		return bizv1.QuoteStatus_QUOTE_REJECTED
	case models.QuoteStatusExpired:
		return bizv1.QuoteStatus_QUOTE_EXPIRED
	default:
		return bizv1.QuoteStatus_QUOTE_STATUS_UNSPECIFIED
	}
}

func invoiceStatusFromProto(s bizv1.InvoiceStatus) string {
	switch s {
	case bizv1.InvoiceStatus_INVOICE_DRAFT:
		return models.InvoiceStatusDraft
	case bizv1.InvoiceStatus_INVOICE_SENT:
		return models.InvoiceStatusSent
	case bizv1.InvoiceStatus_INVOICE_PAID:
		return models.InvoiceStatusPaid
	case bizv1.InvoiceStatus_INVOICE_OVERDUE:
		return models.InvoiceStatusOverdue
	case bizv1.InvoiceStatus_INVOICE_CANCELLED:
		return models.InvoiceStatusCancelled
	default:
		return ""
	}
}

func invoiceStatusToProto(s string) bizv1.InvoiceStatus {
	switch s {
	case models.InvoiceStatusDraft:
		return bizv1.InvoiceStatus_INVOICE_DRAFT
	case models.InvoiceStatusSent:
		return bizv1.InvoiceStatus_INVOICE_SENT
	case models.InvoiceStatusPaid:
		return bizv1.InvoiceStatus_INVOICE_PAID
	case models.InvoiceStatusOverdue:
		return bizv1.InvoiceStatus_INVOICE_OVERDUE
	case models.InvoiceStatusCancelled:
		return bizv1.InvoiceStatus_INVOICE_CANCELLED
	default:
		return bizv1.InvoiceStatus_INVOICE_STATUS_UNSPECIFIED
	}
}

func creditNoteStatusToProto(s string) bizv1.CreditNoteStatus {
	switch s {
	case models.CreditNoteStatusDraft:
		return bizv1.CreditNoteStatus_CREDIT_DRAFT
	case models.CreditNoteStatusSent:
		return bizv1.CreditNoteStatus_CREDIT_SENT
	default:
		return bizv1.CreditNoteStatus_CREDIT_NOTE_STATUS_UNSPECIFIED
	}
}

func dunningStatusFromProto(s bizv1.DunningStatus) string {
	switch s {
	case bizv1.DunningStatus_DUNNING_DRAFT:
		return models.DunningStatusDraft
	case bizv1.DunningStatus_DUNNING_SENT:
		return models.DunningStatusSent
	case bizv1.DunningStatus_DUNNING_PAID:
		return models.DunningStatusPaid
	default:
		return ""
	}
}

func dunningStatusToProto(s string) bizv1.DunningStatus {
	switch s {
	case models.DunningStatusDraft:
		return bizv1.DunningStatus_DUNNING_DRAFT
	case models.DunningStatusSent:
		return bizv1.DunningStatus_DUNNING_SENT
	case models.DunningStatusPaid:
		return bizv1.DunningStatus_DUNNING_PAID
	default:
		return bizv1.DunningStatus_DUNNING_STATUS_UNSPECIFIED
	}
}

func paymentMethodFromProto(m bizv1.PaymentMethod) string {
	switch m {
	case bizv1.PaymentMethod_PAYMENT_METHOD_BANK_TRANSFER:
		return models.PaymentMethodBankTransfer
	case bizv1.PaymentMethod_PAYMENT_METHOD_CASH:
		return models.PaymentMethodCash
	case bizv1.PaymentMethod_PAYMENT_METHOD_CREDIT_CARD:
		return models.PaymentMethodCreditCard
	case bizv1.PaymentMethod_PAYMENT_METHOD_OTHER:
		return models.PaymentMethodOther
	default:
		return models.PaymentMethodBankTransfer
	}
}

func paymentMethodToProto(m string) bizv1.PaymentMethod {
	switch m {
	case models.PaymentMethodBankTransfer:
		return bizv1.PaymentMethod_PAYMENT_METHOD_BANK_TRANSFER
	case models.PaymentMethodCash:
		return bizv1.PaymentMethod_PAYMENT_METHOD_CASH
	case models.PaymentMethodCreditCard:
		return bizv1.PaymentMethod_PAYMENT_METHOD_CREDIT_CARD
	case models.PaymentMethodOther:
		return bizv1.PaymentMethod_PAYMENT_METHOD_OTHER
	default:
		return bizv1.PaymentMethod_PAYMENT_METHOD_UNSPECIFIED
	}
}

// ============================================================================
// Time-Tracking → Invoice
// ============================================================================

// CreateInvoiceFromTimeEntries aggregates completed work time entries for an employee
// in the given date range and creates a draft invoice. The invoice is returned as
// JSON bytes in the response so the gateway proxy can forward it without needing
// to deserialize the full Invoice proto message.
func (s *BizGRPCServer) CreateInvoiceFromTimeEntries(ctx context.Context, req *bizv1.CreateInvoiceFromTimeEntriesRequest) (*bizv1.CreateInvoiceFromTimeEntriesResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}

	employeeID, err := uuid.Parse(req.GetEmployeeId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid employee_id")
	}

	userID, err := uuid.Parse(req.GetCreatedBy())
	if err != nil {
		// created_by is best-effort; fall back to Nil so invoice.Create still works.
		userID = uuid.Nil
	}

	if req.GetCustomerName() == "" {
		return nil, status.Error(codes.InvalidArgument, "customer_name is required")
	}

	dateFrom, err := time.Parse("2006-01-02", req.GetDateFrom())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid date_from, expected YYYY-MM-DD")
	}
	dateTo, err := time.Parse("2006-01-02", req.GetDateTo())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid date_to, expected YYYY-MM-DD")
	}
	// Include the full last day
	dateTo = dateTo.Add(24*time.Hour - time.Second)

	hourlyRate, parseErr := decimal.NewFromString(req.GetHourlyRate())
	if parseErr != nil || hourlyRate.IsNegative() {
		return nil, status.Error(codes.InvalidArgument, "invalid hourly_rate")
	}

	// Resolve the effective tax mode. An explicit request mode wins; otherwise a
	// Kleinunternehmer company (section 19 UStG) must not charge VAT, so derive
	// the mode from company settings (soft fallback to standard on load error).
	taxMode := req.GetTaxMode()
	if taxMode == "" {
		settings, sErr := s.companySettings.GetByTenantID(ctx, tenantID)
		if sErr != nil {
			slog.Warn("failed to load company settings for time-entry invoice tax mode",
				"tenant_id", tenantID, "error", sErr)
		}
		taxMode = resolveTaxMode("", settings)
	}

	if s.timetrackingRepo == nil {
		return nil, status.Error(codes.Unavailable, "time tracking repository not configured")
	}

	// Aggregate completed work time entries via HR repository
	totalMinutes, entryIDs, err := s.timetrackingRepo.AggregateWorkTimeForInvoice(ctx, tenantID, employeeID, dateFrom, dateTo)
	if err != nil {
		slog.Error("aggregate work time failed", "tenant_id", tenantID, "employee_id", employeeID, "error", err)
		return nil, status.Error(codes.Internal, fmt.Sprintf("aggregate work time: %s", err.Error()))
	}

	if totalMinutes == 0 {
		return nil, status.Error(codes.FailedPrecondition, "no completed work time entries found for the given period")
	}

	// Convert minutes to hours (2 decimal places)
	hours := decimal.NewFromInt(int64(totalMinutes)).Div(decimal.NewFromInt(60)).Round(2)
	lineTotal := hours.Mul(hourlyRate)

	description := req.GetDescription()
	if description == "" {
		description = fmt.Sprintf("Arbeitszeit %s–%s", req.GetDateFrom(), req.GetDateTo())
	}

	lineItem := models.LineItem{
		ID:          uuid.New().String(),
		Position:    1,
		Description: description,
		Quantity:    hours,
		UnitPrice:   hourlyRate,
		TaxRate:     taxRateForMode(taxMode),
		LineTotal:   lineTotal,
	}

	// Create draft invoice
	inv, err := s.invoiceService.Create(ctx, invoice.CreateInput{
		TenantID:        tenantID,
		CustomerName:    req.GetCustomerName(),
		CustomerAddress: req.GetCustomerAddress(),
		CustomerEmail:   req.GetCustomerEmail(),
		TaxMode:         taxMode,
		LineItems:       []models.LineItem{lineItem},
		InvoiceDate:     time.Now(),
		UserID:          userID,
	})
	if err != nil {
		slog.Error("create invoice from time entries failed", "tenant_id", tenantID, "error", err)
		return nil, mapBizError(err)
	}

	// Build time_tracking_source audit trail
	type timeTrackingSource struct {
		EmployeeID   string    `json:"employee_id"`
		DateFrom     string    `json:"date_from"`
		DateTo       string    `json:"date_to"`
		TotalMinutes int       `json:"total_minutes"`
		EntryIDs     []string  `json:"entry_ids"`
		CreatedAt    time.Time `json:"created_at"`
	}
	ttSource := timeTrackingSource{
		EmployeeID:   employeeID.String(),
		DateFrom:     req.GetDateFrom(),
		DateTo:       req.GetDateTo(),
		TotalMinutes: totalMinutes,
		EntryIDs:     entryIDs,
		CreatedAt:    time.Now(),
	}
	ttSourceJSON, _ := json.Marshal(ttSource)

	// Attach time_tracking_source to the invoice record (best-effort, non-fatal)
	if linkErr := s.invoiceService.LinkTimeTracking(ctx, inv.TenantID, inv.ID, json.RawMessage(ttSourceJSON)); linkErr != nil {
		slog.Warn("failed to link time tracking source to invoice",
			"invoice_id", inv.ID,
			"error", linkErr,
		)
	}
	inv.TimeTrackingSource = json.RawMessage(ttSourceJSON)

	// Serialize the invoice as JSON for the response
	invJSON, err := json.Marshal(inv)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to serialize invoice")
	}

	return &bizv1.CreateInvoiceFromTimeEntriesResponse{
		InvoiceJson: invJSON,
	}, nil
}

// ============================================================================
// Deal-to-Quote
// ============================================================================

// CreateQuoteFromDeal fetches a CRM deal (and its linked contact/company) and
// creates a pre-populated draft quote. The orchestration lives here rather than
// in the gateway so that the business logic is encapsulated in the biz service.
func (s *BizGRPCServer) CreateQuoteFromDeal(ctx context.Context, req *bizv1.CreateQuoteFromDealRequest) (*bizv1.CreateQuoteFromDealResponse, error) {
	if req.GetDealId() == "" {
		return nil, status.Error(codes.InvalidArgument, "deal_id is required")
	}

	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}

	userID, err := uuid.Parse(req.GetCreatedBy())
	if err != nil {
		userID = uuid.Nil
	}

	if s.crmClient == nil {
		return nil, status.Error(codes.Unavailable, "CRM service not available")
	}

	// 1. Fetch the CRM deal
	dealResp, err := s.crmClient.GetDeal(ctx, &crmv1.GetDealRequest{Id: req.GetDealId()})
	if err != nil {
		slog.Error("CreateQuoteFromDeal: fetch deal failed", "deal_id", req.GetDealId(), "error", err)
		return nil, err
	}
	deal := dealResp.GetDeal()

	// 2. Build customer snapshot — prefer company name for B2B (DACH norm)
	customerName := ""
	customerAddress := ""
	customerEmail := ""

	if deal.GetCompanyName() != "" {
		customerName = deal.GetCompanyName()
	} else if deal.GetContactName() != "" {
		customerName = deal.GetContactName()
	}

	// Enrich from contact if available
	if deal.GetContactId() != "" {
		contactResp, contactErr := s.crmClient.GetContact(ctx, &crmv1.GetContactRequest{Id: deal.GetContactId()})
		if contactErr == nil && contactResp.GetContact() != nil {
			contact := contactResp.GetContact()
			if customerEmail == "" {
				customerEmail = contact.GetEmail()
			}
			if customerName == "" {
				first := contact.GetFirstName()
				last := contact.GetLastName()
				if first != "" && last != "" {
					customerName = first + " " + last
				} else {
					customerName = first + last
				}
			}
		}
	}

	// Enrich address from company if available
	if deal.GetCompanyId() != "" {
		companyResp, companyErr := s.crmClient.GetCompany(ctx, &crmv1.GetCompanyRequest{Id: deal.GetCompanyId()})
		if companyErr == nil && companyResp.GetCompany() != nil {
			company := companyResp.GetCompany()
			if customerAddress == "" {
				parts := []string{}
				if company.GetAddress() != "" {
					parts = append(parts, company.GetAddress())
				}
				if company.GetCity() != "" {
					parts = append(parts, company.GetCity())
				}
				if company.GetCountry() != "" {
					parts = append(parts, company.GetCountry())
				}
				if len(parts) > 0 {
					customerAddress = parts[0]
					for i := 1; i < len(parts); i++ {
						customerAddress += ", " + parts[i]
					}
				}
			}
			if customerName == "" {
				customerName = company.GetName()
			}
		}
	}

	// 3. Create draft quote via quote service.
	dealIDStr := req.GetDealId()
	dealUUID, dealUUIDErr := uuid.Parse(dealIDStr)
	if dealUUIDErr != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid deal_id: must be a UUID")
	}

	// Resolve the effective tax mode. An explicit request mode wins; otherwise a
	// Kleinunternehmer company (section 19 UStG) must not charge VAT, so derive
	// the mode from company settings (soft fallback to standard on load error).
	taxMode := req.GetTaxMode()
	if taxMode == "" {
		settings, sErr := s.companySettings.GetByTenantID(ctx, tenantID)
		if sErr != nil {
			slog.Warn("failed to load company settings for deal-to-quote tax mode",
				"tenant_id", tenantID, "error", sErr)
		}
		taxMode = resolveTaxMode("", settings)
	}

	// A single placeholder line item is required by the service; the user fills
	// in the actual items after opening the quote in the UI.
	placeholderItem := models.LineItem{
		ID:          uuid.New().String(),
		Position:    1,
		Description: "Leistungsbeschreibung (bitte ausfüllen)",
		Quantity:    decimal.NewFromInt(1),
		UnitPrice:   decimal.Zero,
		TaxRate:     taxRateForMode(taxMode),
		LineTotal:   decimal.Zero,
	}

	q, err := s.quoteService.Create(ctx, quote.CreateInput{
		TenantID:        tenantID,
		CustomerName:    customerName,
		CustomerAddress: customerAddress,
		CustomerEmail:   customerEmail,
		TaxMode:         taxMode,
		LineItems:       []models.LineItem{placeholderItem},
		DealID:          &dealUUID,
		UserID:          userID,
	})
	if err != nil {
		slog.Error("CreateQuoteFromDeal: create quote failed",
			"deal_id", req.GetDealId(),
			"tenant_id", tenantID,
			"error", err,
		)
		return nil, mapBizError(err)
	}

	return &bizv1.CreateQuoteFromDealResponse{
		Quote: toProtoQuote(q),
	}, nil
}

// ============================================================================
// GoBD Journal & Compliance (Sprint 2 / Wave 1.B)
// ============================================================================

func (s *BizGRPCServer) GetJournalSummary(ctx context.Context, req *bizv1.GetJournalSummaryRequest) (*bizv1.GetJournalSummaryResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}

	year := int(req.GetYear())
	if year < 2000 || year > 2100 {
		return nil, status.Error(codes.InvalidArgument, "year must be between 2000 and 2100")
	}

	summary, err := s.invoiceService.GetJournalSummary(ctx, tenantID, year)
	if err != nil {
		slog.Error("GetJournalSummary failed", "tenant_id", tenantID, "year", year, "error", err)
		return nil, mapBizError(err)
	}

	return &bizv1.GetJournalSummaryResponse{
		Year:                int32(summary.Year),
		TotalInvoicesIssued: int32(summary.TotalInvoicesIssued),
		GapsDetected:        int32(summary.GapsDetected),
		FirstNumber:         summary.FirstNumber,
		LastNumber:          summary.LastNumber,
		HighestSequence:     int32(summary.HighestSequence),
	}, nil
}

func (s *BizGRPCServer) ValidateInvoiceNumber(ctx context.Context, req *bizv1.ValidateInvoiceNumberRequest) (*bizv1.ValidateInvoiceNumberResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}

	result, err := s.invoiceService.ValidateInvoiceNumber(ctx, tenantID, req.GetInvoiceNumber())
	if err != nil {
		slog.Error("ValidateInvoiceNumber failed", "tenant_id", tenantID, "error", err)
		return nil, mapBizError(err)
	}

	return &bizv1.ValidateInvoiceNumberResponse{
		ValidFormat: result.ValidFormat,
		AlreadyUsed: result.AlreadyUsed,
		Canonical:   result.Canonical,
	}, nil
}

func (s *BizGRPCServer) LockInvoice(ctx context.Context, req *bizv1.LockInvoiceRequest) (*bizv1.LockInvoiceResponse, error) {
	// LockInvoiceRequest uses Id (not InvoiceId) per biz_gobd.pb.go
	tenantID, id, err := parseTenantAndID(req.GetTenantId(), req.GetId())
	if err != nil {
		return nil, err
	}

	lockedBy, parseErr := uuid.Parse(req.GetLockedBy())
	if parseErr != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid locked_by user id")
	}

	result, lockErr := s.invoiceService.LockInvoice(ctx, tenantID, id, lockedBy)
	if lockErr != nil {
		slog.Error("LockInvoice failed", "invoice_id", id, "tenant_id", tenantID, "error", lockErr)
		return nil, mapBizError(lockErr)
	}

	// Serialize invoice to JSON for transport (Sprint 4: replace with typed proto fields)
	var invJSON []byte
	if result.Invoice != nil {
		invJSON, _ = json.Marshal(result.Invoice)
	}

	// LockInvoiceResponse.LockedAt is RFC3339 string per biz_gobd.pb.go
	return &bizv1.LockInvoiceResponse{
		InvoiceJson: invJSON,
		LockedAt:    result.LockedAt.Format(time.RFC3339),
	}, nil
}

func (s *BizGRPCServer) GetPaymentStats(ctx context.Context, req *bizv1.GetPaymentStatsRequest) (*bizv1.GetPaymentStatsResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}

	// FromDate/ToDate are YYYY-MM-DD strings per biz_gobd.pb.go
	fromDate, fromErr := time.Parse("2006-01-02", req.GetFromDate())
	if fromErr != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid from_date: expected YYYY-MM-DD")
	}
	toDate, toErr := time.Parse("2006-01-02", req.GetToDate())
	if toErr != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid to_date: expected YYYY-MM-DD")
	}
	if toDate.Before(fromDate) {
		return nil, status.Error(codes.InvalidArgument, "to_date must be after from_date")
	}

	stats, statsErr := s.invoiceService.GetPaymentStats(ctx, tenantID, fromDate, toDate)
	if statsErr != nil {
		slog.Error("GetPaymentStats failed", "tenant_id", tenantID, "error", statsErr)
		return nil, mapBizError(statsErr)
	}

	// Response amounts are decimal strings per biz_gobd.pb.go
	return &bizv1.GetPaymentStatsResponse{
		TotalInvoices:          int32(stats.TotalInvoices),
		TotalPaid:              int32(stats.TotalPaid),
		TotalOutstanding:       int32(stats.TotalOutstanding),
		TotalPaidAmount:        stats.TotalPaidAmount.StringFixed(2),
		TotalOutstandingAmount: stats.TotalOutstandingAmount.StringFixed(2),
		AverageDaysToPay:       stats.AverageDaysToPay.StringFixed(1),
	}, nil
}

// ============================================================================
// Dunning Gaps (Sprint 2 / Wave 1.B)
// ============================================================================

func (s *BizGRPCServer) UpdateDunningStatus(ctx context.Context, req *bizv1.UpdateDunningStatusRequest) (*bizv1.UpdateDunningStatusResponse, error) {
	// UpdateDunningStatusRequest uses Id (not DunningId) per biz_gobd.pb.go
	tenantID, id, err := parseTenantAndID(req.GetTenantId(), req.GetId())
	if err != nil {
		return nil, err
	}

	record, updateErr := s.dunningService.UpdateDunningStatus(ctx, tenantID, id, req.GetStatus())
	if updateErr != nil {
		slog.Error("UpdateDunningStatus failed", "dunning_id", id, "tenant_id", tenantID, "error", updateErr)
		return nil, mapBizError(updateErr)
	}

	// Response wraps a full DunningRecord proto per biz_gobd.pb.go
	return &bizv1.UpdateDunningStatusResponse{
		Dunning: toProtoDunningRecord(record),
	}, nil
}

func (s *BizGRPCServer) SendDunningNotice(ctx context.Context, req *bizv1.SendDunningNoticeRequest) (*bizv1.SendDunningNoticeResponse, error) {
	// SendDunningNoticeRequest uses Id (not DunningId), no UserId field per biz_gobd.pb.go
	tenantID, id, err := parseTenantAndID(req.GetTenantId(), req.GetId())
	if err != nil {
		return nil, err
	}

	record, sendErr := s.dunningService.SendDunningNotice(ctx, tenantID, id, uuid.Nil)
	if sendErr != nil {
		slog.Error("SendDunningNotice failed", "dunning_id", id, "tenant_id", tenantID, "error", sendErr)
		return nil, mapBizError(sendErr)
	}

	// EmailQueued = false until Sprint 3 email integration
	return &bizv1.SendDunningNoticeResponse{
		Dunning:     toProtoDunningRecord(record),
		EmailQueued: false,
	}, nil
}

// ============================================================================
// GoBD Export (Sprint 2 / Wave 1.B)
// ============================================================================

func (s *BizGRPCServer) GenerateGoBDExport(_ context.Context, _ *bizv1.GenerateGoBDExportRequest) (*bizv1.GenerateGoBDExportResponse, error) {
	// F3: the GoBD CSV export is not yet IDEA/BMF-compliant — it omits the mandatory
	// posting columns (Buchungsbetrag netto, MwSt-Betrag, Steuerschlüssel,
	// Soll/Haben-Konto, Buchungstext, interne Belegnummer). Returning a "preview"
	// file that looks like a GoBD export risks an operator filing a non-compliant
	// document with the Finanzamt. Disable the endpoint (HTTP 501 at the gateway)
	// until the format is completed. The CSV builder in dunning.GenerateGoBDExport
	// and its tests remain in place for the rebuild.
	return nil, status.Error(codes.Unimplemented,
		"GoBD export is not yet IDEA/BMF-compliant and is temporarily disabled")
}

// ============================================================================
// Shared Helpers
// ============================================================================

func parseTenantAndID(tenantIDStr, idStr string) (uuid.UUID, uuid.UUID, error) {
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		return uuid.Nil, uuid.Nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		return uuid.Nil, uuid.Nil, status.Error(codes.InvalidArgument, "invalid id")
	}

	return tenantID, id, nil
}

// ============================================================================
// Error Mapping
// ============================================================================

func mapBizError(err error) error {
	if err == nil {
		return nil
	}

	switch {
	// Quote errors
	case errors.Is(err, quote.ErrQuoteNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, quote.ErrQuoteNotDraft):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, quote.ErrQuoteNotSent):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, quote.ErrNoLineItems):
		return status.Error(codes.InvalidArgument, err.Error())

	// Invoice errors
	case errors.Is(err, invoice.ErrInvoiceNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, invoice.ErrInvoiceImmutable):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, invoice.ErrInvoiceNotDraft):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, invoice.ErrInvoiceAlreadyPaid):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, invoice.ErrInvoiceAlreadyCancelled):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, invoice.ErrNoLineItems):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, invoice.ErrQuoteNotAccepted):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, invoice.ErrInvoiceLocked):
		return status.Error(codes.FailedPrecondition, err.Error())

	// Credit note errors
	case errors.Is(err, creditnote.ErrCreditNoteNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, creditnote.ErrCreditNoteNotDraft):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, creditnote.ErrInvalidInvoiceForCredit):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, creditnote.ErrNoLineItems):
		return status.Error(codes.InvalidArgument, err.Error())

	// Dunning errors
	case errors.Is(err, dunning.ErrDunningNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, dunning.ErrDunningNotDraft):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, dunning.ErrDunningMaxLevel):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, dunning.ErrConfigNotFound):
		return status.Error(codes.NotFound, err.Error())

	default:
		slog.Error("unhandled biz service error", "error", err)
		return status.Error(codes.Internal, "internal server error")
	}
}
