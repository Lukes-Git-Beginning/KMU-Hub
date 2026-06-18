package server

import (
	"context"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/kmuhub/kmuhub/internal/einkauf"
	einkaufv1 "github.com/kmuhub/kmuhub/proto/einkauf/v1"
)

// ============================================================================
// Catalog RPCs
// ============================================================================

func (s *EinkaufGRPCServer) ListCatalogItems(ctx context.Context, req *einkaufv1.ListCatalogItemsRequest) (*einkaufv1.ListCatalogItemsResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}

	input := einkauf.ListCatalogItemsInput{
		TenantID: tenantID,
		Category: req.GetCategory(),
		Search:   req.GetSearch(),
		Page:     int(req.GetPage()),
		PageSize: int(req.GetPageSize()),
	}

	if req.SupplierId != nil && *req.SupplierId != "" {
		id, err := uuid.Parse(*req.SupplierId)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid supplier_id: %v", err)
		}
		input.SupplierID = &id
	}

	if req.Available != nil {
		input.Available = req.Available
	}

	items, total, err := s.svc.ListCatalogItems(ctx, input)
	if err != nil {
		return nil, mapEinkaufError(err)
	}

	protoItems := make([]*einkaufv1.CatalogItem, len(items))
	for i, item := range items {
		protoItems[i] = catalogItemToProto(item)
	}
	return &einkaufv1.ListCatalogItemsResponse{
		Items: protoItems,
		Total: int32(total),
	}, nil
}

func (s *EinkaufGRPCServer) GetCatalogItem(ctx context.Context, req *einkaufv1.GetCatalogItemRequest) (*einkaufv1.CatalogItemResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	itemID, err := uuid.Parse(req.GetItemId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid item_id: %v", err)
	}

	item, err := s.svc.GetCatalogItem(ctx, tenantID, itemID)
	if err != nil {
		return nil, mapEinkaufError(err)
	}
	return &einkaufv1.CatalogItemResponse{Item: catalogItemToProto(item)}, nil
}

func (s *EinkaufGRPCServer) CreateCatalogItem(ctx context.Context, req *einkaufv1.CreateCatalogItemRequest) (*einkaufv1.CatalogItemResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	supplierID, err := uuid.Parse(req.GetSupplierId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid supplier_id: %v", err)
	}

	item, err := s.svc.CreateCatalogItem(ctx, einkauf.CreateCatalogItemInput{
		TenantID:    tenantID,
		SupplierID:  supplierID,
		Name:        req.GetName(),
		SKU:         req.GetSku(),
		Category:    req.GetCategory(),
		Price:       req.GetPrice(),
		Currency:    req.GetCurrency(),
		Unit:        req.GetUnit(),
		Available:   req.GetAvailable(),
		MinOrderQty: req.GetMinOrderQty(),
	})
	if err != nil {
		return nil, mapEinkaufError(err)
	}
	return &einkaufv1.CatalogItemResponse{Item: catalogItemToProto(item)}, nil
}

func (s *EinkaufGRPCServer) UpdateCatalogItem(ctx context.Context, req *einkaufv1.UpdateCatalogItemRequest) (*einkaufv1.CatalogItemResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	itemID, err := uuid.Parse(req.GetItemId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid item_id: %v", err)
	}

	item, err := s.svc.UpdateCatalogItem(ctx, einkauf.UpdateCatalogItemInput{
		TenantID:    tenantID,
		ItemID:      itemID,
		Name:        req.Name,
		SKU:         req.Sku,
		Category:    req.Category,
		Price:       req.Price,
		Currency:    req.Currency,
		Unit:        req.Unit,
		Available:   req.Available,
		MinOrderQty: req.MinOrderQty,
	})
	if err != nil {
		return nil, mapEinkaufError(err)
	}
	return &einkaufv1.CatalogItemResponse{Item: catalogItemToProto(item)}, nil
}

func (s *EinkaufGRPCServer) DeleteCatalogItem(ctx context.Context, req *einkaufv1.DeleteCatalogItemRequest) (*einkaufv1.DeleteCatalogItemResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	itemID, err := uuid.Parse(req.GetItemId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid item_id: %v", err)
	}

	if err := s.svc.DeleteCatalogItem(ctx, tenantID, itemID); err != nil {
		return nil, mapEinkaufError(err)
	}
	return &einkaufv1.DeleteCatalogItemResponse{}, nil
}

// ============================================================================
// Supplier Rating RPCs
// ============================================================================

func (s *EinkaufGRPCServer) ListSupplierRatings(ctx context.Context, req *einkaufv1.ListSupplierRatingsRequest) (*einkaufv1.ListSupplierRatingsResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	supplierID, err := uuid.Parse(req.GetSupplierId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid supplier_id: %v", err)
	}

	ratings, err := s.svc.ListSupplierRatings(ctx, tenantID, supplierID)
	if err != nil {
		return nil, mapEinkaufError(err)
	}

	protoRatings := make([]*einkaufv1.SupplierRating, len(ratings))
	for i, r := range ratings {
		protoRatings[i] = supplierRatingToProto(r)
	}
	return &einkaufv1.ListSupplierRatingsResponse{Ratings: protoRatings}, nil
}

func (s *EinkaufGRPCServer) CreateSupplierRating(ctx context.Context, req *einkaufv1.CreateSupplierRatingRequest) (*einkaufv1.SupplierRatingResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	supplierID, err := uuid.Parse(req.GetSupplierId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid supplier_id: %v", err)
	}

	input := einkauf.CreateSupplierRatingInput{
		TenantID:   tenantID,
		SupplierID: supplierID,
		Category:   einkauf.RatingCategory(req.GetCategory()),
		Rating:     req.GetRating(),
		Comment:    req.GetComment(),
	}

	if req.RatedBy != nil && *req.RatedBy != "" {
		id, err := uuid.Parse(*req.RatedBy)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid rated_by: %v", err)
		}
		input.RatedBy = &id
	}

	rating, err := s.svc.CreateSupplierRating(ctx, input)
	if err != nil {
		return nil, mapEinkaufError(err)
	}
	return &einkaufv1.SupplierRatingResponse{Rating: supplierRatingToProto(rating)}, nil
}

func (s *EinkaufGRPCServer) DeleteSupplierRating(ctx context.Context, req *einkaufv1.DeleteSupplierRatingRequest) (*einkaufv1.DeleteSupplierRatingResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	ratingID, err := uuid.Parse(req.GetRatingId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid rating_id: %v", err)
	}

	if err := s.svc.DeleteSupplierRating(ctx, tenantID, ratingID); err != nil {
		return nil, mapEinkaufError(err)
	}
	return &einkaufv1.DeleteSupplierRatingResponse{}, nil
}

// ============================================================================
// Framework Contract RPCs
// ============================================================================

func (s *EinkaufGRPCServer) ListFrameworkContracts(ctx context.Context, req *einkaufv1.ListFrameworkContractsRequest) (*einkaufv1.ListFrameworkContractsResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}

	input := einkauf.ListFrameworkContractsInput{
		TenantID: tenantID,
		Page:     int(req.GetPage()),
		PageSize: int(req.GetPageSize()),
	}

	if req.SupplierId != nil && *req.SupplierId != "" {
		id, err := uuid.Parse(*req.SupplierId)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid supplier_id: %v", err)
		}
		input.SupplierID = &id
	}

	if req.Status != nil {
		st := einkauf.ContractStatus(*req.Status)
		input.Status = &st
	}

	contracts, total, err := s.svc.ListFrameworkContracts(ctx, input)
	if err != nil {
		return nil, mapEinkaufError(err)
	}

	protoContracts := make([]*einkaufv1.FrameworkContract, len(contracts))
	for i, c := range contracts {
		protoContracts[i] = frameworkContractToProto(c)
	}
	return &einkaufv1.ListFrameworkContractsResponse{
		Contracts: protoContracts,
		Total:     int32(total),
	}, nil
}

func (s *EinkaufGRPCServer) GetFrameworkContract(ctx context.Context, req *einkaufv1.GetFrameworkContractRequest) (*einkaufv1.FrameworkContractResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	contractID, err := uuid.Parse(req.GetContractId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid contract_id: %v", err)
	}

	fc, err := s.svc.GetFrameworkContract(ctx, tenantID, contractID)
	if err != nil {
		return nil, mapEinkaufError(err)
	}
	return &einkaufv1.FrameworkContractResponse{Contract: frameworkContractToProto(fc)}, nil
}

func (s *EinkaufGRPCServer) CreateFrameworkContract(ctx context.Context, req *einkaufv1.CreateFrameworkContractRequest) (*einkaufv1.FrameworkContractResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	supplierID, err := uuid.Parse(req.GetSupplierId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid supplier_id: %v", err)
	}

	input := einkauf.CreateFrameworkContractInput{
		TenantID:   tenantID,
		SupplierID: supplierID,
		Title:      req.GetTitle(),
		ContractNr: req.GetContractNr(),
		TotalValue: req.GetTotalValue(),
		Currency:   req.GetCurrency(),
		Status:     einkauf.ContractStatus(req.GetStatus()),
	}

	if req.StartDate != nil {
		t := req.StartDate.AsTime()
		input.StartDate = &t
	}
	if req.EndDate != nil {
		t := req.EndDate.AsTime()
		input.EndDate = &t
	}

	fc, err := s.svc.CreateFrameworkContract(ctx, input)
	if err != nil {
		return nil, mapEinkaufError(err)
	}
	return &einkaufv1.FrameworkContractResponse{Contract: frameworkContractToProto(fc)}, nil
}

func (s *EinkaufGRPCServer) UpdateFrameworkContract(ctx context.Context, req *einkaufv1.UpdateFrameworkContractRequest) (*einkaufv1.FrameworkContractResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	contractID, err := uuid.Parse(req.GetContractId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid contract_id: %v", err)
	}

	input := einkauf.UpdateFrameworkContractInput{
		TenantID:   tenantID,
		ContractID: contractID,
		Title:      req.Title,
		ContractNr: req.ContractNr,
		TotalValue: req.TotalValue,
		Currency:   req.Currency,
	}

	if req.SupplierId != nil && *req.SupplierId != "" {
		id, err := uuid.Parse(*req.SupplierId)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid supplier_id: %v", err)
		}
		input.SupplierID = &id
	}

	if req.Status != nil {
		st := einkauf.ContractStatus(*req.Status)
		input.Status = &st
	}

	if req.StartDate != nil {
		t := req.StartDate.AsTime()
		if t.Unix() == 0 {
			zeroT := time.Time{}
			input.StartDate = &zeroT
		} else {
			input.StartDate = &t
		}
	}

	if req.EndDate != nil {
		t := req.EndDate.AsTime()
		if t.Unix() == 0 {
			zeroT := time.Time{}
			input.EndDate = &zeroT
		} else {
			input.EndDate = &t
		}
	}

	fc, err := s.svc.UpdateFrameworkContract(ctx, input)
	if err != nil {
		return nil, mapEinkaufError(err)
	}
	return &einkaufv1.FrameworkContractResponse{Contract: frameworkContractToProto(fc)}, nil
}

func (s *EinkaufGRPCServer) DeleteFrameworkContract(ctx context.Context, req *einkaufv1.DeleteFrameworkContractRequest) (*einkaufv1.DeleteFrameworkContractResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	contractID, err := uuid.Parse(req.GetContractId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid contract_id: %v", err)
	}

	if err := s.svc.DeleteFrameworkContract(ctx, tenantID, contractID); err != nil {
		return nil, mapEinkaufError(err)
	}
	return &einkaufv1.DeleteFrameworkContractResponse{}, nil
}

func (s *EinkaufGRPCServer) CreateContractItem(ctx context.Context, req *einkaufv1.CreateContractItemRequest) (*einkaufv1.ContractItemResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	contractID, err := uuid.Parse(req.GetContractId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid contract_id: %v", err)
	}

	item, err := s.svc.CreateContractItem(ctx, einkauf.CreateContractItemInput{
		TenantID:   tenantID,
		ContractID: contractID,
		Name:       req.GetName(),
		UnitPrice:  req.GetUnitPrice(),
		Unit:       req.GetUnit(),
		AgreedQty:  req.GetAgreedQty(),
	})
	if err != nil {
		return nil, mapEinkaufError(err)
	}
	return &einkaufv1.ContractItemResponse{Item: contractItemToProto(item)}, nil
}

func (s *EinkaufGRPCServer) UpdateContractItem(ctx context.Context, req *einkaufv1.UpdateContractItemRequest) (*einkaufv1.ContractItemResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	itemID, err := uuid.Parse(req.GetItemId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid item_id: %v", err)
	}

	item, err := s.svc.UpdateContractItem(ctx, einkauf.UpdateContractItemInput{
		TenantID:  tenantID,
		ItemID:    itemID,
		Name:      req.Name,
		UnitPrice: req.UnitPrice,
		Unit:      req.Unit,
		AgreedQty: req.AgreedQty,
	})
	if err != nil {
		return nil, mapEinkaufError(err)
	}
	return &einkaufv1.ContractItemResponse{Item: contractItemToProto(item)}, nil
}

func (s *EinkaufGRPCServer) DeleteContractItem(ctx context.Context, req *einkaufv1.DeleteContractItemRequest) (*einkaufv1.DeleteContractItemResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	itemID, err := uuid.Parse(req.GetItemId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid item_id: %v", err)
	}

	if err := s.svc.DeleteContractItem(ctx, tenantID, itemID); err != nil {
		return nil, mapEinkaufError(err)
	}
	return &einkaufv1.DeleteContractItemResponse{}, nil
}

func (s *EinkaufGRPCServer) CreateContractCall(ctx context.Context, req *einkaufv1.CreateContractCallRequest) (*einkaufv1.ContractCallResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	contractID, err := uuid.Parse(req.GetContractId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid contract_id: %v", err)
	}

	input := einkauf.CreateContractCallInput{
		TenantID:   tenantID,
		ContractID: contractID,
		Amount:     req.GetAmount(),
		Currency:   req.GetCurrency(),
		Notes:      req.GetNotes(),
	}

	if req.PoId != nil && *req.PoId != "" {
		id, err := uuid.Parse(*req.PoId)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid po_id: %v", err)
		}
		input.POID = &id
	}

	call, err := s.svc.CreateContractCall(ctx, input)
	if err != nil {
		return nil, mapEinkaufError(err)
	}
	return &einkaufv1.ContractCallResponse{Call: contractCallToProto(call)}, nil
}

func (s *EinkaufGRPCServer) ListContractCalls(ctx context.Context, req *einkaufv1.ListContractCallsRequest) (*einkaufv1.ListContractCallsResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	contractID, err := uuid.Parse(req.GetContractId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid contract_id: %v", err)
	}

	calls, err := s.svc.ListContractCalls(ctx, tenantID, contractID)
	if err != nil {
		return nil, mapEinkaufError(err)
	}

	protoCalls := make([]*einkaufv1.FrameworkContractCall, len(calls))
	for i, c := range calls {
		protoCalls[i] = contractCallToProto(c)
	}
	return &einkaufv1.ListContractCallsResponse{Calls: protoCalls}, nil
}

// ============================================================================
// Conversion helpers
// ============================================================================

func catalogItemToProto(item *einkauf.CatalogItem) *einkaufv1.CatalogItem {
	if item == nil {
		return nil
	}
	return &einkaufv1.CatalogItem{
		Id:          item.ID.String(),
		TenantId:    item.TenantID.String(),
		SupplierId:  item.SupplierID.String(),
		Name:        item.Name,
		Sku:         item.SKU,
		Category:    item.Category,
		Price:       item.Price,
		Currency:    item.Currency,
		Unit:        item.Unit,
		Available:   item.Available,
		MinOrderQty: item.MinOrderQty,
		CreatedAt:   timestamppb.New(item.CreatedAt),
		UpdatedAt:   timestamppb.New(item.UpdatedAt),
	}
}

func supplierRatingToProto(rt *einkauf.SupplierRating) *einkaufv1.SupplierRating {
	if rt == nil {
		return nil
	}
	p := &einkaufv1.SupplierRating{
		Id:         rt.ID.String(),
		TenantId:   rt.TenantID.String(),
		SupplierId: rt.SupplierID.String(),
		Category:   string(rt.Category),
		Rating:     rt.Rating,
		Comment:    rt.Comment,
		RatedAt:    timestamppb.New(rt.RatedAt),
		CreatedAt:  timestamppb.New(rt.CreatedAt),
		UpdatedAt:  timestamppb.New(rt.UpdatedAt),
	}
	if rt.RatedBy != nil {
		str := rt.RatedBy.String()
		p.RatedBy = &str
	}
	return p
}

func frameworkContractToProto(fc *einkauf.FrameworkContract) *einkaufv1.FrameworkContract {
	if fc == nil {
		return nil
	}
	p := &einkaufv1.FrameworkContract{
		Id:         fc.ID.String(),
		TenantId:   fc.TenantID.String(),
		SupplierId: fc.SupplierID.String(),
		Title:      fc.Title,
		ContractNr: fc.ContractNr,
		TotalValue: fc.TotalValue,
		UsedValue:  fc.UsedValue,
		Currency:   fc.Currency,
		Status:     string(fc.Status),
		CreatedAt:  timestamppb.New(fc.CreatedAt),
		UpdatedAt:  timestamppb.New(fc.UpdatedAt),
	}
	if fc.StartDate != nil {
		p.StartDate = timestamppb.New(*fc.StartDate)
	}
	if fc.EndDate != nil {
		p.EndDate = timestamppb.New(*fc.EndDate)
	}
	for _, item := range fc.Items {
		p.Items = append(p.Items, contractItemToProto(item))
	}
	return p
}

func contractItemToProto(item *einkauf.FrameworkContractItem) *einkaufv1.FrameworkContractItem {
	if item == nil {
		return nil
	}
	return &einkaufv1.FrameworkContractItem{
		Id:         item.ID.String(),
		TenantId:   item.TenantID.String(),
		ContractId: item.ContractID.String(),
		Name:       item.Name,
		UnitPrice:  item.UnitPrice,
		Unit:       item.Unit,
		AgreedQty:  item.AgreedQty,
		CalledQty:  item.CalledQty,
		CreatedAt:  timestamppb.New(item.CreatedAt),
		UpdatedAt:  timestamppb.New(item.UpdatedAt),
	}
}

func contractCallToProto(call *einkauf.FrameworkContractCall) *einkaufv1.FrameworkContractCall {
	if call == nil {
		return nil
	}
	p := &einkaufv1.FrameworkContractCall{
		Id:         call.ID.String(),
		TenantId:   call.TenantID.String(),
		ContractId: call.ContractID.String(),
		Amount:     call.Amount,
		Currency:   call.Currency,
		CalledAt:   timestamppb.New(call.CalledAt),
		Notes:      call.Notes,
		CreatedAt:  timestamppb.New(call.CreatedAt),
		UpdatedAt:  timestamppb.New(call.UpdatedAt),
	}
	if call.POID != nil {
		str := call.POID.String()
		p.PoId = &str
	}
	return p
}

