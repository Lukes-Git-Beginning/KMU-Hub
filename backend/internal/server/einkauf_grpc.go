package server

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/kmuhub/kmuhub/internal/einkauf"
	einkaufv1 "github.com/kmuhub/kmuhub/proto/einkauf/v1"
)

// EinkaufGRPCServer implements the EinkaufService gRPC server.
type EinkaufGRPCServer struct {
	einkaufv1.UnimplementedEinkaufServiceServer
	svc *einkauf.Service
}

// NewEinkaufGRPCServer creates a new Einkauf gRPC server.
func NewEinkaufGRPCServer(svc *einkauf.Service) *EinkaufGRPCServer {
	return &EinkaufGRPCServer{svc: svc}
}

// ============================================================================
// Supplier RPCs
// ============================================================================

func (s *EinkaufGRPCServer) CreateSupplier(ctx context.Context, req *einkaufv1.CreateSupplierRequest) (*einkaufv1.SupplierResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}

	input := einkauf.CreateSupplierInput{
		TenantID:     tenantID,
		Name:         req.GetName(),
		Email:        req.GetEmail(),
		Phone:        req.GetPhone(),
		Address:      req.GetAddress(),
		PaymentTerms: req.GetPaymentTerms(),
		Notes:        req.GetNotes(),
	}

	if req.ContactId != nil && *req.ContactId != "" {
		id, err := uuid.Parse(*req.ContactId)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid contact_id: %v", err)
		}
		input.ContactID = &id
	}

	supplier, err := s.svc.CreateSupplier(ctx, input)
	if err != nil {
		return nil, mapEinkaufError(err)
	}
	return &einkaufv1.SupplierResponse{Supplier: supplierToProto(supplier)}, nil
}

func (s *EinkaufGRPCServer) UpdateSupplier(ctx context.Context, req *einkaufv1.UpdateSupplierRequest) (*einkaufv1.SupplierResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	supplierID, err := uuid.Parse(req.GetSupplierId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid supplier_id: %v", err)
	}

	input := einkauf.UpdateSupplierInput{
		TenantID:   tenantID,
		SupplierID: supplierID,
		Name:       req.Name,
		Email:      req.Email,
		Phone:      req.Phone,
		Address:    req.Address,
		Notes:      req.Notes,
	}

	if req.PaymentTerms != nil {
		input.PaymentTerms = req.PaymentTerms
	}

	if req.ContactId != nil {
		if *req.ContactId == "" {
			empty := uuid.Nil
			input.ContactID = &empty
		} else {
			id, parseErr := uuid.Parse(*req.ContactId)
			if parseErr != nil {
				return nil, status.Errorf(codes.InvalidArgument, "invalid contact_id: %v", parseErr)
			}
			input.ContactID = &id
		}
	}

	supplier, err := s.svc.UpdateSupplier(ctx, input)
	if err != nil {
		return nil, mapEinkaufError(err)
	}
	return &einkaufv1.SupplierResponse{Supplier: supplierToProto(supplier)}, nil
}

func (s *EinkaufGRPCServer) DeleteSupplier(ctx context.Context, req *einkaufv1.DeleteSupplierRequest) (*einkaufv1.DeleteSupplierResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	supplierID, err := uuid.Parse(req.GetSupplierId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid supplier_id: %v", err)
	}

	if err := s.svc.DeleteSupplier(ctx, tenantID, supplierID); err != nil {
		return nil, mapEinkaufError(err)
	}
	return &einkaufv1.DeleteSupplierResponse{}, nil
}

func (s *EinkaufGRPCServer) GetSupplier(ctx context.Context, req *einkaufv1.GetSupplierRequest) (*einkaufv1.SupplierResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	supplierID, err := uuid.Parse(req.GetSupplierId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid supplier_id: %v", err)
	}

	supplier, err := s.svc.GetSupplier(ctx, tenantID, supplierID)
	if err != nil {
		return nil, mapEinkaufError(err)
	}
	return &einkaufv1.SupplierResponse{Supplier: supplierToProto(supplier)}, nil
}

func (s *EinkaufGRPCServer) ListSuppliers(ctx context.Context, req *einkaufv1.ListSuppliersRequest) (*einkaufv1.ListSuppliersResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}

	suppliers, total, err := s.svc.ListSuppliers(ctx, einkauf.ListSuppliersInput{
		TenantID:   tenantID,
		Search:     req.GetSearch(),
		ActiveOnly: req.GetActiveOnly(),
		Page:       int(req.GetPage()),
		PageSize:   int(req.GetPageSize()),
	})
	if err != nil {
		return nil, mapEinkaufError(err)
	}

	protoSuppliers := make([]*einkaufv1.Supplier, len(suppliers))
	for i, s := range suppliers {
		protoSuppliers[i] = supplierToProto(s)
	}
	return &einkaufv1.ListSuppliersResponse{
		Suppliers: protoSuppliers,
		Total:     int32(total),
	}, nil
}

// ============================================================================
// Purchase Order RPCs
// ============================================================================

func (s *EinkaufGRPCServer) CreatePO(ctx context.Context, req *einkaufv1.CreatePORequest) (*einkaufv1.POResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	supplierID, err := uuid.Parse(req.GetSupplierId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid supplier_id: %v", err)
	}

	input := einkauf.CreatePOInput{
		TenantID:   tenantID,
		SupplierID: supplierID,
		PONumber:   req.GetPoNumber(),
		Currency:   req.GetCurrency(),
		Notes:      req.GetNotes(),
	}

	if req.OrderDate != nil {
		input.OrderDate = req.OrderDate.AsTime()
	} else {
		input.OrderDate = time.Now()
	}

	if req.ExpectedDeliveryDate != nil {
		t := req.ExpectedDeliveryDate.AsTime()
		input.ExpectedDeliveryDate = &t
	}

	if req.CreatedBy != nil && *req.CreatedBy != "" {
		id, err := uuid.Parse(*req.CreatedBy)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid created_by: %v", err)
		}
		input.CreatedBy = &id
	}

	po, err := s.svc.CreatePO(ctx, input)
	if err != nil {
		return nil, mapEinkaufError(err)
	}
	return &einkaufv1.POResponse{Po: poToProto(po)}, nil
}

func (s *EinkaufGRPCServer) UpdatePO(ctx context.Context, req *einkaufv1.UpdatePORequest) (*einkaufv1.POResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	poID, err := uuid.Parse(req.GetPoId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid po_id: %v", err)
	}

	input := einkauf.UpdatePOInput{
		TenantID: tenantID,
		POID:     poID,
		PONumber: req.PoNumber,
		Currency: req.Currency,
		Notes:    req.Notes,
	}

	if req.SupplierId != nil {
		id, err := uuid.Parse(*req.SupplierId)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid supplier_id: %v", err)
		}
		input.SupplierID = &id
	}

	if req.OrderDate != nil {
		t := req.OrderDate.AsTime()
		input.OrderDate = &t
	}

	if req.ExpectedDeliveryDate != nil {
		t := req.ExpectedDeliveryDate.AsTime()
		// Epoch means "clear"
		if t.Unix() == 0 {
			zeroT := time.Time{}
			input.ExpectedDeliveryDate = &zeroT
		} else {
			input.ExpectedDeliveryDate = &t
		}
	}

	po, err := s.svc.UpdatePO(ctx, input)
	if err != nil {
		return nil, mapEinkaufError(err)
	}
	return &einkaufv1.POResponse{Po: poToProto(po)}, nil
}

func (s *EinkaufGRPCServer) DeletePO(ctx context.Context, req *einkaufv1.DeletePORequest) (*einkaufv1.DeletePOResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	poID, err := uuid.Parse(req.GetPoId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid po_id: %v", err)
	}

	if err := s.svc.DeletePO(ctx, tenantID, poID); err != nil {
		return nil, mapEinkaufError(err)
	}
	return &einkaufv1.DeletePOResponse{}, nil
}

func (s *EinkaufGRPCServer) GetPO(ctx context.Context, req *einkaufv1.GetPORequest) (*einkaufv1.POResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	poID, err := uuid.Parse(req.GetPoId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid po_id: %v", err)
	}

	po, err := s.svc.GetPO(ctx, tenantID, poID)
	if err != nil {
		return nil, mapEinkaufError(err)
	}
	return &einkaufv1.POResponse{Po: poToProto(po)}, nil
}

func (s *EinkaufGRPCServer) ListPOs(ctx context.Context, req *einkaufv1.ListPOsRequest) (*einkaufv1.ListPOsResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}

	input := einkauf.ListPOsInput{
		TenantID: tenantID,
		Page:     int(req.GetPage()),
		PageSize: int(req.GetPageSize()),
	}

	if req.SupplierId != nil {
		id, err := uuid.Parse(*req.SupplierId)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid supplier_id: %v", err)
		}
		input.SupplierID = &id
	}

	if req.Status != nil {
		st := einkauf.POStatus(*req.Status)
		input.Status = &st
	}

	if req.DateFrom != nil {
		t := req.DateFrom.AsTime()
		input.DateFrom = &t
	}
	if req.DateTo != nil {
		t := req.DateTo.AsTime()
		input.DateTo = &t
	}

	pos, total, err := s.svc.ListPOs(ctx, input)
	if err != nil {
		return nil, mapEinkaufError(err)
	}

	protoPOs := make([]*einkaufv1.PurchaseOrder, len(pos))
	for i, po := range pos {
		protoPOs[i] = poToProto(po)
	}
	return &einkaufv1.ListPOsResponse{
		Pos:   protoPOs,
		Total: int32(total),
	}, nil
}

// ============================================================================
// PO Line RPCs
// ============================================================================

func (s *EinkaufGRPCServer) AddPOLine(ctx context.Context, req *einkaufv1.AddPOLineRequest) (*einkaufv1.POLineResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	poID, err := uuid.Parse(req.GetPoId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid po_id: %v", err)
	}

	line, err := s.svc.AddPOLine(ctx, einkauf.AddPOLineInput{
		TenantID:     tenantID,
		POID:         poID,
		ProductName:  req.GetProductName(),
		SKU:          req.GetSku(),
		Quantity:     req.GetQuantity(),
		UnitPrice:    req.GetUnitPrice(),
		TaxRate:      req.GetTaxRate(),
		LinePosition: int(req.GetLinePosition()),
	})
	if err != nil {
		return nil, mapEinkaufError(err)
	}
	return &einkaufv1.POLineResponse{Line: lineToProto(line)}, nil
}

func (s *EinkaufGRPCServer) UpdatePOLine(ctx context.Context, req *einkaufv1.UpdatePOLineRequest) (*einkaufv1.POLineResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	lineID, err := uuid.Parse(req.GetLineId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid line_id: %v", err)
	}

	input := einkauf.UpdatePOLineInput{
		TenantID:    tenantID,
		LineID:      lineID,
		ProductName: req.ProductName,
		SKU:         req.Sku,
		Quantity:    req.Quantity,
		UnitPrice:   req.UnitPrice,
		TaxRate:     req.TaxRate,
	}

	if req.LinePosition != nil {
		pos := int(*req.LinePosition)
		input.LinePosition = &pos
	}

	line, err := s.svc.UpdatePOLine(ctx, input)
	if err != nil {
		return nil, mapEinkaufError(err)
	}
	return &einkaufv1.POLineResponse{Line: lineToProto(line)}, nil
}

func (s *EinkaufGRPCServer) DeletePOLine(ctx context.Context, req *einkaufv1.DeletePOLineRequest) (*einkaufv1.DeletePOLineResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	lineID, err := uuid.Parse(req.GetLineId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid line_id: %v", err)
	}

	if err := s.svc.DeletePOLine(ctx, tenantID, lineID); err != nil {
		return nil, mapEinkaufError(err)
	}
	return &einkaufv1.DeletePOLineResponse{}, nil
}

func (s *EinkaufGRPCServer) ListPOLines(ctx context.Context, req *einkaufv1.ListPOLinesRequest) (*einkaufv1.ListPOLinesResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	poID, err := uuid.Parse(req.GetPoId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid po_id: %v", err)
	}

	lines, err := s.svc.ListPOLines(ctx, tenantID, poID)
	if err != nil {
		return nil, mapEinkaufError(err)
	}

	protoLines := make([]*einkaufv1.POLine, len(lines))
	for i, l := range lines {
		protoLines[i] = lineToProto(l)
	}
	return &einkaufv1.ListPOLinesResponse{Lines: protoLines}, nil
}

// ============================================================================
// Workflow RPCs
// ============================================================================

func (s *EinkaufGRPCServer) SubmitPO(ctx context.Context, req *einkaufv1.SubmitPORequest) (*einkaufv1.POResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	poID, err := uuid.Parse(req.GetPoId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid po_id: %v", err)
	}

	po, err := s.svc.SubmitPO(ctx, tenantID, poID)
	if err != nil {
		return nil, mapEinkaufError(err)
	}
	return &einkaufv1.POResponse{Po: poToProto(po)}, nil
}

func (s *EinkaufGRPCServer) ReceiveGoods(ctx context.Context, req *einkaufv1.ReceiveGoodsRequest) (*einkaufv1.POResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	poID, err := uuid.Parse(req.GetPoId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid po_id: %v", err)
	}

	po, err := s.svc.ReceiveGoods(ctx, tenantID, poID)
	if err != nil {
		return nil, mapEinkaufError(err)
	}
	return &einkaufv1.POResponse{Po: poToProto(po)}, nil
}

func (s *EinkaufGRPCServer) PartialReceive(ctx context.Context, req *einkaufv1.PartialReceiveRequest) (*einkaufv1.POResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	poID, err := uuid.Parse(req.GetPoId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid po_id: %v", err)
	}

	items := make([]einkauf.PartialReceiveItem, 0, len(req.GetItems()))
	for _, item := range req.GetItems() {
		lineID, err := uuid.Parse(item.GetLineId())
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid line_id in items: %v", err)
		}
		items = append(items, einkauf.PartialReceiveItem{
			LineID:           lineID,
			ReceivedQuantity: item.GetReceivedQuantity(),
		})
	}

	po, err := s.svc.PartialReceive(ctx, tenantID, poID, items)
	if err != nil {
		return nil, mapEinkaufError(err)
	}
	return &einkaufv1.POResponse{Po: poToProto(po)}, nil
}

func (s *EinkaufGRPCServer) ExportPO(ctx context.Context, req *einkaufv1.ExportPORequest) (*einkaufv1.ExportPOResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	poID, err := uuid.Parse(req.GetPoId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid po_id: %v", err)
	}

	format := req.GetFormat()
	if format == "" {
		format = "pdf"
	}

	// ExportPO currently returns the PO data only; rendering to PDF/CSV is a
	// follow-up task. The gateway will receive an empty payload with correct headers.
	po, err := s.svc.ExportPO(ctx, tenantID, poID, format)
	if err != nil {
		return nil, mapEinkaufError(err)
	}

	// Stub: return PO number as plain filename, no binary payload yet.
	_ = po
	return &einkaufv1.ExportPOResponse{
		Payload:     []byte{},
		ContentType: "application/octet-stream",
		Filename:    "po." + format,
	}, nil
}

// ============================================================================
// Conversion helpers
// ============================================================================

func supplierToProto(s *einkauf.Supplier) *einkaufv1.Supplier {
	if s == nil {
		return nil
	}
	p := &einkaufv1.Supplier{
		Id:           s.ID.String(),
		TenantId:     s.TenantID.String(),
		Name:         s.Name,
		Email:        s.Email,
		Phone:        s.Phone,
		Address:      s.Address,
		PaymentTerms: s.PaymentTerms,
		Notes:        s.Notes,
		CreatedAt:    timestamppb.New(s.CreatedAt),
		UpdatedAt:    timestamppb.New(s.UpdatedAt),
	}
	if s.ContactID != nil {
		str := s.ContactID.String()
		p.ContactId = &str
	}
	return p
}

func poToProto(po *einkauf.PurchaseOrder) *einkaufv1.PurchaseOrder {
	if po == nil {
		return nil
	}
	p := &einkaufv1.PurchaseOrder{
		Id:          po.ID.String(),
		TenantId:    po.TenantID.String(),
		SupplierId:  po.SupplierID.String(),
		PoNumber:    po.PONumber,
		Status:      string(po.Status),
		OrderDate:   timestamppb.New(po.OrderDate),
		TotalAmount: po.TotalAmount,
		Currency:    po.Currency,
		Notes:       po.Notes,
		CreatedAt:   timestamppb.New(po.CreatedAt),
		UpdatedAt:   timestamppb.New(po.UpdatedAt),
	}
	if po.ExpectedDeliveryDate != nil {
		p.ExpectedDeliveryDate = timestamppb.New(*po.ExpectedDeliveryDate)
	}
	if po.CreatedBy != nil {
		str := po.CreatedBy.String()
		p.CreatedBy = &str
	}
	for _, l := range po.Lines {
		p.Lines = append(p.Lines, lineToProto(l))
	}
	return p
}

func lineToProto(l *einkauf.POLine) *einkaufv1.POLine {
	if l == nil {
		return nil
	}
	return &einkaufv1.POLine{
		Id:               l.ID.String(),
		TenantId:         l.TenantID.String(),
		PoId:             l.POID.String(),
		ProductName:      l.ProductName,
		Sku:              l.SKU,
		Quantity:         l.Quantity,
		UnitPrice:        l.UnitPrice,
		TaxRate:          l.TaxRate,
		ReceivedQuantity: l.ReceivedQuantity,
		LinePosition:     int32(l.LinePosition),
		CreatedAt:        timestamppb.New(l.CreatedAt),
		UpdatedAt:        timestamppb.New(l.UpdatedAt),
	}
}

// ============================================================================
// Error mapping
// ============================================================================

func mapEinkaufError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, einkauf.ErrSupplierNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, einkauf.ErrPONotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, einkauf.ErrPOLineNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, einkauf.ErrPONumberTaken):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, einkauf.ErrInvalidInput):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, einkauf.ErrPONotDraft):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, einkauf.ErrPONotSubmittable):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, einkauf.ErrPONotReceivable):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, einkauf.ErrInvalidQuantity):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, einkauf.ErrExceedsOrderedQty):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		slog.Error("unhandled einkauf service error", "error", err)
		return status.Error(codes.Internal, "internal error")
	}
}
