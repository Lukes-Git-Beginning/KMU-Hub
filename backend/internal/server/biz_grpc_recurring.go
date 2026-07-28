package server

import (
	"context"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/kmuhub/kmuhub/internal/biz/recurring"
	"github.com/kmuhub/kmuhub/internal/models"
	bizv1 "github.com/kmuhub/kmuhub/proto/biz/v1"
)

// Recurring invoice schedules (Abo-Rechnungen) — Migration 000246.
//
// Thin handlers: parse, call, respond. All schedule semantics (interval
// arithmetic, idempotent generation, end-date enforcement) live in
// recurring.Service.

const recurringDateLayout = "2006-01-02"

func (s *BizGRPCServer) requireRecurring() error {
	if s.recurringSvc == nil {
		return status.Error(codes.Unimplemented, "recurring invoices not configured")
	}
	return nil
}

func (s *BizGRPCServer) CreateRecurringInvoice(ctx context.Context, req *bizv1.CreateRecurringInvoiceRequest) (*bizv1.CreateRecurringInvoiceResponse, error) {
	if err := s.requireRecurring(); err != nil {
		return nil, err
	}
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}
	startDate, err := parseRecurringDate(req.GetStartDate())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid start_date, expected YYYY-MM-DD")
	}
	endDate, err := parseOptionalRecurringDate(req.GetEndDate())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid end_date, expected YYYY-MM-DD")
	}
	// created_by is optional: a schedule created by an automation has no user.
	var userID uuid.UUID
	if raw := req.GetCreatedBy(); raw != "" {
		userID, err = uuid.Parse(raw)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid created_by")
		}
	}

	rec, err := s.recurringSvc.Create(ctx, recurring.CreateInput{
		TenantID:         tenantID,
		Title:            req.GetTitle(),
		Customer:         customerFromProto(req.GetCustomer()),
		LineItems:        protoLineItemsToModel(req.GetLineItems()),
		TaxMode:          taxModeFromProto(req.GetTaxMode()),
		Currency:         req.GetCurrency(),
		Interval:         req.GetInterval(),
		StartDate:        startDate,
		EndDate:          endDate,
		PaymentTermsDays: int(req.GetPaymentTermsDays()),
		Notes:            req.GetNotes(),
		UserID:           userID,
	})
	if err != nil {
		return nil, mapBizError(err)
	}
	return &bizv1.CreateRecurringInvoiceResponse{Recurring: toProtoRecurring(rec)}, nil
}

func (s *BizGRPCServer) GetRecurringInvoice(ctx context.Context, req *bizv1.GetRecurringInvoiceRequest) (*bizv1.GetRecurringInvoiceResponse, error) {
	if err := s.requireRecurring(); err != nil {
		return nil, err
	}
	tenantID, id, err := parseTenantAndID(req.GetTenantId(), req.GetId())
	if err != nil {
		return nil, err
	}
	rec, err := s.recurringSvc.Get(ctx, tenantID, id)
	if err != nil {
		return nil, mapBizError(err)
	}
	return &bizv1.GetRecurringInvoiceResponse{Recurring: toProtoRecurring(rec)}, nil
}

func (s *BizGRPCServer) ListRecurringInvoices(ctx context.Context, req *bizv1.ListRecurringInvoicesRequest) (*bizv1.ListRecurringInvoicesResponse, error) {
	if err := s.requireRecurring(); err != nil {
		return nil, err
	}
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}
	limit, offset := pageToLimitOffset(req.GetPage(), req.GetPerPage())

	items, total, err := s.recurringSvc.List(ctx, tenantID, recurring.ListFilter{
		Status: req.GetStatus(),
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, mapBizError(err)
	}

	out := make([]*bizv1.RecurringInvoice, 0, len(items))
	for _, rec := range items {
		out = append(out, toProtoRecurring(rec))
	}
	return &bizv1.ListRecurringInvoicesResponse{Recurring: out, Total: int32(total)}, nil
}

func (s *BizGRPCServer) UpdateRecurringInvoice(ctx context.Context, req *bizv1.UpdateRecurringInvoiceRequest) (*bizv1.UpdateRecurringInvoiceResponse, error) {
	if err := s.requireRecurring(); err != nil {
		return nil, err
	}
	tenantID, id, err := parseTenantAndID(req.GetTenantId(), req.GetId())
	if err != nil {
		return nil, err
	}

	in := recurring.UpdateInput{
		Title:        req.Title,
		Currency:     req.Currency,
		Interval:     req.Interval,
		Notes:        req.Notes,
		ClearEndDate: req.GetClearEndDate(),
	}
	if req.GetCustomer() != nil {
		customer := customerFromProto(req.GetCustomer())
		in.Customer = &customer
	}
	if len(req.GetLineItems()) > 0 {
		in.LineItems = protoLineItemsToModel(req.GetLineItems())
	}
	if req.TaxMode != nil {
		mode := taxModeFromProto(req.GetTaxMode())
		in.TaxMode = &mode
	}
	if req.PaymentTermsDays != nil {
		days := int(req.GetPaymentTermsDays())
		in.PaymentTermsDays = &days
	}
	if req.StartDate != nil {
		startDate, parseErr := parseRecurringDate(req.GetStartDate())
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid start_date, expected YYYY-MM-DD")
		}
		in.StartDate = &startDate
	}
	if req.EndDate != nil && req.GetEndDate() != "" {
		endDate, parseErr := parseRecurringDate(req.GetEndDate())
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid end_date, expected YYYY-MM-DD")
		}
		in.EndDate = &endDate
	}

	rec, err := s.recurringSvc.Update(ctx, tenantID, id, in)
	if err != nil {
		return nil, mapBizError(err)
	}
	return &bizv1.UpdateRecurringInvoiceResponse{Recurring: toProtoRecurring(rec)}, nil
}

func (s *BizGRPCServer) DeleteRecurringInvoice(ctx context.Context, req *bizv1.DeleteRecurringInvoiceRequest) (*bizv1.DeleteRecurringInvoiceResponse, error) {
	if err := s.requireRecurring(); err != nil {
		return nil, err
	}
	tenantID, id, err := parseTenantAndID(req.GetTenantId(), req.GetId())
	if err != nil {
		return nil, err
	}
	if err := s.recurringSvc.Delete(ctx, tenantID, id); err != nil {
		return nil, mapBizError(err)
	}
	return &bizv1.DeleteRecurringInvoiceResponse{Success: true}, nil
}

func (s *BizGRPCServer) SetRecurringInvoiceStatus(ctx context.Context, req *bizv1.SetRecurringInvoiceStatusRequest) (*bizv1.SetRecurringInvoiceStatusResponse, error) {
	if err := s.requireRecurring(); err != nil {
		return nil, err
	}
	tenantID, id, err := parseTenantAndID(req.GetTenantId(), req.GetId())
	if err != nil {
		return nil, err
	}
	rec, err := s.recurringSvc.SetStatus(ctx, tenantID, id, req.GetStatus())
	if err != nil {
		return nil, mapBizError(err)
	}
	return &bizv1.SetRecurringInvoiceStatusResponse{Recurring: toProtoRecurring(rec)}, nil
}

func (s *BizGRPCServer) GenerateRecurringInvoice(ctx context.Context, req *bizv1.GenerateRecurringInvoiceRequest) (*bizv1.GenerateRecurringInvoiceResponse, error) {
	if err := s.requireRecurring(); err != nil {
		return nil, err
	}
	tenantID, id, err := parseTenantAndID(req.GetTenantId(), req.GetId())
	if err != nil {
		return nil, err
	}
	var userID uuid.UUID
	if raw := req.GetUserId(); raw != "" {
		userID, err = uuid.Parse(raw)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid user_id")
		}
	}

	inv, rec, err := s.recurringSvc.Generate(ctx, tenantID, id, userID)
	if err != nil {
		return nil, mapBizError(err)
	}
	return &bizv1.GenerateRecurringInvoiceResponse{
		Invoice:   toProtoInvoice(inv),
		Recurring: toProtoRecurring(rec),
	}, nil
}

// ============================================================================
// Converters
// ============================================================================

func toProtoRecurring(rec *models.RecurringInvoice) *bizv1.RecurringInvoice {
	if rec == nil {
		return nil
	}
	pr := &bizv1.RecurringInvoice{
		Id:       rec.ID.String(),
		TenantId: rec.TenantID.String(),
		Title:    rec.Title,
		Customer: &bizv1.CustomerSnapshot{
			Name:    rec.CustomerName,
			Address: rec.CustomerAddress,
			Email:   rec.CustomerEmail,
			UstIdNr: rec.CustomerUStIDNr,
		},
		LineItems:        parseProtoLineItems(rec.LineItems),
		TaxMode:          taxModeToProto(rec.TaxMode),
		TaxBreakdown:     parseProtoTaxBreakdown(rec.TaxBreakdownRaw),
		Currency:         documentCurrency(rec.Currency),
		Interval:         rec.Interval,
		Status:           rec.Status,
		StartDate:        rec.StartDate.Format(recurringDateLayout),
		NextRun:          rec.NextRun.Format(recurringDateLayout),
		PaymentTermsDays: int32(rec.PaymentTermsDays),
		GeneratedCount:   int32(rec.GeneratedCount),
		Notes:            rec.Notes,
		CreatedAt:        timestamppb.New(rec.CreatedAt),
		UpdatedAt:        timestamppb.New(rec.UpdatedAt),
	}
	if rec.EndDate != nil {
		pr.EndDate = rec.EndDate.Format(recurringDateLayout)
	}
	if rec.LastGeneratedAt != nil {
		pr.LastGeneratedAt = rec.LastGeneratedAt.Format(recurringDateLayout)
	}
	return pr
}

func customerFromProto(c *bizv1.CustomerSnapshot) models.CustomerSnapshot {
	if c == nil {
		return models.CustomerSnapshot{}
	}
	return models.CustomerSnapshot{
		Name:    c.GetName(),
		Address: c.GetAddress(),
		Email:   c.GetEmail(),
		UStIDNr: c.GetUstIdNr(),
	}
}

func parseRecurringDate(value string) (time.Time, error) {
	return time.Parse(recurringDateLayout, value)
}

func parseOptionalRecurringDate(value string) (*time.Time, error) {
	if value == "" {
		return nil, nil
	}
	parsed, err := parseRecurringDate(value)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}
