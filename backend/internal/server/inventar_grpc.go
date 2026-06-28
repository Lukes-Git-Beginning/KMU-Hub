package server

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/kmuhub/kmuhub/internal/inventar"
	inventarv1 "github.com/kmuhub/kmuhub/proto/inventar/v1"
)

// InventarGRPCServer implements the InventarService gRPC server.
type InventarGRPCServer struct {
	inventarv1.UnimplementedInventarServiceServer
	svc *inventar.Service
}

// NewInventarGRPCServer creates a new Inventar gRPC server.
func NewInventarGRPCServer(svc *inventar.Service) *InventarGRPCServer {
	return &InventarGRPCServer{svc: svc}
}

// ============================================================================
// Item RPCs
// ============================================================================

func (s *InventarGRPCServer) CreateItem(ctx context.Context, req *inventarv1.CreateItemRequest) (*inventarv1.ItemResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}

	input := inventar.CreateItemInput{
		TenantID:            tenantID,
		Name:                req.GetName(),
		SKU:                 req.GetSku(),
		Barcode:             req.Barcode,
		Quantity:            req.GetQuantity(),
		MinQuantity:         req.GetMinQuantity(),
		Unit:                req.GetUnit(),
		Location:            req.Location,
		Category:            req.Category,
		Price:               req.Price,
		Currency:            req.Currency,
		BatchNumber:         req.BatchNumber,
		SerialNumbers:       req.GetSerialNumbers(),
		LinkedPurchaseOrder: req.LinkedPurchaseOrder,
	}
	if req.LocationId != nil {
		locID, parseErr := uuid.Parse(*req.LocationId)
		if parseErr != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid location_id: %v", parseErr)
		}
		input.LocationID = &locID
	}

	item, err := s.svc.CreateItem(ctx, input)
	if err != nil {
		return nil, mapInventarError(err)
	}
	return &inventarv1.ItemResponse{Item: itemToProto(item)}, nil
}

func (s *InventarGRPCServer) GetItem(ctx context.Context, req *inventarv1.GetItemRequest) (*inventarv1.ItemResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	itemID, err := uuid.Parse(req.GetItemId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid item_id: %v", err)
	}

	item, err := s.svc.GetItem(ctx, tenantID, itemID)
	if err != nil {
		return nil, mapInventarError(err)
	}
	return &inventarv1.ItemResponse{Item: itemToProto(item)}, nil
}

func (s *InventarGRPCServer) UpdateItem(ctx context.Context, req *inventarv1.UpdateItemRequest) (*inventarv1.ItemResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	itemID, err := uuid.Parse(req.GetItemId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid item_id: %v", err)
	}

	input := inventar.UpdateItemInput{
		TenantID:            tenantID,
		ItemID:              itemID,
		Name:                req.Name,
		SKU:                 req.Sku,
		Barcode:             req.Barcode,
		Unit:                req.Unit,
		Location:            req.Location,
		Category:            req.Category,
		Price:               req.Price,
		Currency:            req.Currency,
		BatchNumber:         req.BatchNumber,
		LinkedPurchaseOrder: req.LinkedPurchaseOrder,
	}
	if req.MinQuantity != nil {
		input.MinQuantity = req.MinQuantity
	}
	if req.LocationId != nil {
		locID, parseErr := uuid.Parse(*req.LocationId)
		if parseErr != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid location_id: %v", parseErr)
		}
		input.LocationID = &locID
	}

	item, err := s.svc.UpdateItem(ctx, input)
	if err != nil {
		return nil, mapInventarError(err)
	}
	return &inventarv1.ItemResponse{Item: itemToProto(item)}, nil
}

func (s *InventarGRPCServer) DeleteItem(ctx context.Context, req *inventarv1.DeleteItemRequest) (*inventarv1.DeleteItemResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	itemID, err := uuid.Parse(req.GetItemId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid item_id: %v", err)
	}

	if err := s.svc.DeleteItem(ctx, tenantID, itemID); err != nil {
		return nil, mapInventarError(err)
	}
	return &inventarv1.DeleteItemResponse{}, nil
}

func (s *InventarGRPCServer) ListItems(ctx context.Context, req *inventarv1.ListItemsRequest) (*inventarv1.ListItemsResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}

	input := inventar.ListItemsInput{
		TenantID: tenantID,
		Search:   req.GetSearch(),
		Location: req.Location,
		LowStock: req.GetLowStock(),
		Page:     int(req.GetPage()),
		PageSize: int(req.GetPageSize()),
	}

	items, total, err := s.svc.ListItems(ctx, input)
	if err != nil {
		return nil, mapInventarError(err)
	}

	protoItems := make([]*inventarv1.Item, len(items))
	for i, item := range items {
		protoItems[i] = itemToProto(item)
	}
	return &inventarv1.ListItemsResponse{
		Items: protoItems,
		Total: int32(total),
	}, nil
}

// ============================================================================
// Stock Operation RPCs
// ============================================================================

func (s *InventarGRPCServer) AdjustStock(ctx context.Context, req *inventarv1.AdjustStockRequest) (*inventarv1.ItemResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	itemID, err := uuid.Parse(req.GetItemId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid item_id: %v", err)
	}

	input := inventar.AdjustStockInput{
		TenantID: tenantID,
		ItemID:   itemID,
		Delta:    req.GetDelta(),
		Reason:   req.GetReason(),
	}

	if req.PerformedBy != nil {
		id, parseErr := uuid.Parse(*req.PerformedBy)
		if parseErr != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid performed_by: %v", parseErr)
		}
		input.PerformedBy = &id
	}

	item, err := s.svc.AdjustStock(ctx, input)
	if err != nil {
		return nil, mapInventarError(err)
	}
	return &inventarv1.ItemResponse{Item: itemToProto(item)}, nil
}

func (s *InventarGRPCServer) TransferStock(ctx context.Context, req *inventarv1.TransferStockRequest) (*inventarv1.TransferStockResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	fromID, err := uuid.Parse(req.GetFromItemId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid from_item_id: %v", err)
	}
	toID, err := uuid.Parse(req.GetToItemId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid to_item_id: %v", err)
	}

	input := inventar.TransferStockInput{
		TenantID:   tenantID,
		FromItemID: fromID,
		ToItemID:   toID,
		Quantity:   req.GetQuantity(),
		Reason:     req.GetReason(),
	}

	if req.PerformedBy != nil {
		id, parseErr := uuid.Parse(*req.PerformedBy)
		if parseErr != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid performed_by: %v", parseErr)
		}
		input.PerformedBy = &id
	}

	if err := s.svc.TransferStock(ctx, input); err != nil {
		return nil, mapInventarError(err)
	}
	return &inventarv1.TransferStockResponse{}, nil
}

func (s *InventarGRPCServer) RecordMovement(ctx context.Context, req *inventarv1.RecordMovementRequest) (*inventarv1.MovementResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	itemID, err := uuid.Parse(req.GetItemId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid item_id: %v", err)
	}

	input := inventar.RecordMovementInput{
		TenantID:     tenantID,
		ItemID:       itemID,
		MovementType: inventar.MovementType(req.GetMovementType()),
		Quantity:     req.GetQuantity(),
		Reason:       req.GetReason(),
	}

	if req.PerformedBy != nil {
		id, parseErr := uuid.Parse(*req.PerformedBy)
		if parseErr != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid performed_by: %v", parseErr)
		}
		input.PerformedBy = &id
	}

	movement, err := s.svc.RecordMovement(ctx, input)
	if err != nil {
		return nil, mapInventarError(err)
	}
	return &inventarv1.MovementResponse{Movement: movementToProto(movement)}, nil
}

func (s *InventarGRPCServer) ListMovements(ctx context.Context, req *inventarv1.ListMovementsRequest) (*inventarv1.ListMovementsResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	itemID, err := uuid.Parse(req.GetItemId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid item_id: %v", err)
	}

	movements, total, err := s.svc.ListMovements(ctx, inventar.ListMovementsInput{
		TenantID: tenantID,
		ItemID:   itemID,
		Page:     int(req.GetPage()),
		PageSize: int(req.GetPageSize()),
	})
	if err != nil {
		return nil, mapInventarError(err)
	}

	protoMovements := make([]*inventarv1.Movement, len(movements))
	for i, m := range movements {
		protoMovements[i] = movementToProto(m)
	}
	return &inventarv1.ListMovementsResponse{
		Movements: protoMovements,
		Total:     int32(total),
	}, nil
}

// GetStockHistory is an alias for ListMovements with pagination.
func (s *InventarGRPCServer) GetStockHistory(ctx context.Context, req *inventarv1.GetStockHistoryRequest) (*inventarv1.ListMovementsResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	itemID, err := uuid.Parse(req.GetItemId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid item_id: %v", err)
	}

	movements, total, err := s.svc.ListMovements(ctx, inventar.ListMovementsInput{
		TenantID: tenantID,
		ItemID:   itemID,
		Page:     int(req.GetPage()),
		PageSize: int(req.GetPageSize()),
	})
	if err != nil {
		return nil, mapInventarError(err)
	}

	protoMovements := make([]*inventarv1.Movement, len(movements))
	for i, m := range movements {
		protoMovements[i] = movementToProto(m)
	}
	return &inventarv1.ListMovementsResponse{
		Movements: protoMovements,
		Total:     int32(total),
	}, nil
}

// ============================================================================
// Warning RPCs
// ============================================================================

func (s *InventarGRPCServer) CreateWarning(ctx context.Context, req *inventarv1.CreateWarningRequest) (*inventarv1.WarningResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	itemID, err := uuid.Parse(req.GetItemId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid item_id: %v", err)
	}

	warning, err := s.svc.CreateWarning(ctx, inventar.CreateWarningInput{
		TenantID:        tenantID,
		ItemID:          itemID,
		Threshold:       req.GetThreshold(),
		CurrentQuantity: req.GetCurrentQuantity(),
	})
	if err != nil {
		return nil, mapInventarError(err)
	}
	return &inventarv1.WarningResponse{Warning: warningToProto(warning)}, nil
}

func (s *InventarGRPCServer) UpdateWarning(ctx context.Context, req *inventarv1.UpdateWarningRequest) (*inventarv1.WarningResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	warningID, err := uuid.Parse(req.GetWarningId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid warning_id: %v", err)
	}

	warning, err := s.svc.UpdateWarning(ctx, inventar.UpdateWarningInput{
		TenantID:  tenantID,
		WarningID: warningID,
		Status:    inventar.WarningStatus(req.GetStatus()),
	})
	if err != nil {
		return nil, mapInventarError(err)
	}
	return &inventarv1.WarningResponse{Warning: warningToProto(warning)}, nil
}

func (s *InventarGRPCServer) AcknowledgeWarning(ctx context.Context, req *inventarv1.AcknowledgeWarningRequest) (*inventarv1.WarningResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	warningID, err := uuid.Parse(req.GetWarningId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid warning_id: %v", err)
	}
	acknowledgedBy, err := uuid.Parse(req.GetAcknowledgedBy())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid acknowledged_by: %v", err)
	}

	warning, err := s.svc.AcknowledgeWarning(ctx, inventar.AcknowledgeWarningInput{
		TenantID:       tenantID,
		WarningID:      warningID,
		AcknowledgedBy: acknowledgedBy,
	})
	if err != nil {
		return nil, mapInventarError(err)
	}
	return &inventarv1.WarningResponse{Warning: warningToProto(warning)}, nil
}

func (s *InventarGRPCServer) ListWarnings(ctx context.Context, req *inventarv1.ListWarningsRequest) (*inventarv1.ListWarningsResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}

	input := inventar.ListWarningsInput{
		TenantID: tenantID,
		Page:     int(req.GetPage()),
		PageSize: int(req.GetPageSize()),
	}
	if req.Status != nil {
		ws := inventar.WarningStatus(*req.Status)
		input.Status = &ws
	}

	warnings, total, err := s.svc.ListWarnings(ctx, input)
	if err != nil {
		return nil, mapInventarError(err)
	}

	protoWarnings := make([]*inventarv1.Warning, len(warnings))
	for i, w := range warnings {
		protoWarnings[i] = warningToProto(w)
	}
	return &inventarv1.ListWarningsResponse{
		Warnings: protoWarnings,
		Total:    int32(total),
	}, nil
}

// ============================================================================
// Report RPCs
// ============================================================================

func (s *InventarGRPCServer) GetStockReport(ctx context.Context, req *inventarv1.GetStockReportRequest) (*inventarv1.StockReportResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}

	report, err := s.svc.GetStockReport(ctx, tenantID)
	if err != nil {
		return nil, mapInventarError(err)
	}
	return &inventarv1.StockReportResponse{
		TotalItems:     int32(report.TotalItems),
		TotalQuantity:  report.TotalQuantity,
		LowStockCount:  int32(report.LowStockCount),
		ActiveWarnings: int32(report.ActiveWarnings),
	}, nil
}

func (s *InventarGRPCServer) ExportInventory(ctx context.Context, req *inventarv1.ExportInventoryRequest) (*inventarv1.ExportInventoryResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}

	items, _, err := s.svc.ListItems(ctx, inventar.ListItemsInput{
		TenantID: tenantID,
		Page:     1,
		PageSize: 10000,
	})
	if err != nil {
		return nil, mapInventarError(err)
	}

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)

	if writeErr := w.Write([]string{"id", "name", "sku", "barcode", "quantity", "min_quantity", "unit", "location"}); writeErr != nil {
		return nil, status.Errorf(codes.Internal, "write csv header: %v", writeErr)
	}

	for _, item := range items {
		barcode := ""
		if item.Barcode != nil {
			barcode = *item.Barcode
		}
		location := ""
		if item.Location != nil {
			location = *item.Location
		}
		row := []string{
			item.ID.String(),
			item.Name,
			item.SKU,
			barcode,
			strconv.FormatInt(item.Quantity, 10),
			strconv.FormatInt(item.MinQuantity, 10),
			item.Unit,
			location,
		}
		if writeErr := w.Write(row); writeErr != nil {
			return nil, status.Errorf(codes.Internal, "write csv row: %v", writeErr)
		}
	}
	w.Flush()
	if flushErr := w.Error(); flushErr != nil {
		return nil, status.Errorf(codes.Internal, "flush csv: %v", flushErr)
	}

	filename := fmt.Sprintf("inventar_%s.csv", tenantID.String()[:8])

	return &inventarv1.ExportInventoryResponse{
		Payload:     buf.Bytes(),
		ContentType: "text/csv",
		Filename:    filename,
	}, nil
}

// ============================================================================
// Location RPCs
// ============================================================================

func (s *InventarGRPCServer) CreateLocation(ctx context.Context, req *inventarv1.CreateLocationRequest) (*inventarv1.LocationResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	loc, err := s.svc.CreateLocation(ctx, inventar.CreateLocationInput{
		TenantID: tenantID,
		Name:     req.GetName(),
		Address:  req.GetAddress(),
		Type:     inventar.LocationType(req.GetType()),
	})
	if err != nil {
		return nil, mapInventarError(err)
	}
	return &inventarv1.LocationResponse{Location: locationToProto(loc)}, nil
}

func (s *InventarGRPCServer) GetLocation(ctx context.Context, req *inventarv1.GetLocationRequest) (*inventarv1.LocationResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	locID, err := uuid.Parse(req.GetLocationId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid location_id: %v", err)
	}
	loc, err := s.svc.GetLocation(ctx, tenantID, locID)
	if err != nil {
		return nil, mapInventarError(err)
	}
	return &inventarv1.LocationResponse{Location: locationToProto(loc)}, nil
}

func (s *InventarGRPCServer) UpdateLocation(ctx context.Context, req *inventarv1.UpdateLocationRequest) (*inventarv1.LocationResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	locID, err := uuid.Parse(req.GetLocationId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid location_id: %v", err)
	}
	input := inventar.UpdateLocationInput{
		TenantID:   tenantID,
		LocationID: locID,
		Name:       req.Name,
		Address:    req.Address,
	}
	if req.Type != nil {
		lt := inventar.LocationType(*req.Type)
		input.Type = &lt
	}
	loc, err := s.svc.UpdateLocation(ctx, input)
	if err != nil {
		return nil, mapInventarError(err)
	}
	return &inventarv1.LocationResponse{Location: locationToProto(loc)}, nil
}

func (s *InventarGRPCServer) DeleteLocation(ctx context.Context, req *inventarv1.DeleteLocationRequest) (*inventarv1.DeleteLocationResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	locID, err := uuid.Parse(req.GetLocationId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid location_id: %v", err)
	}
	if err := s.svc.DeleteLocation(ctx, tenantID, locID); err != nil {
		return nil, mapInventarError(err)
	}
	return &inventarv1.DeleteLocationResponse{}, nil
}

func (s *InventarGRPCServer) ListLocations(ctx context.Context, req *inventarv1.ListLocationsRequest) (*inventarv1.ListLocationsResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	locs, total, err := s.svc.ListLocations(ctx, inventar.ListLocationsInput{
		TenantID: tenantID,
		Page:     int(req.GetPage()),
		PageSize: int(req.GetPageSize()),
	})
	if err != nil {
		return nil, mapInventarError(err)
	}
	protoLocs := make([]*inventarv1.Location, len(locs))
	for i, loc := range locs {
		protoLocs[i] = locationToProto(loc)
	}
	return &inventarv1.ListLocationsResponse{Locations: protoLocs, Total: int32(total)}, nil
}

// ============================================================================
// Inventur Session RPCs
// ============================================================================

func (s *InventarGRPCServer) CreateInventurSession(ctx context.Context, req *inventarv1.CreateInventurSessionRequest) (*inventarv1.InventurSessionResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	input := inventar.CreateInventurSessionInput{
		TenantID: tenantID,
		Name:     req.GetName(),
	}
	if req.GetDate() != nil {
		input.Date = req.GetDate().AsTime()
	}
	if req.LocationId != nil {
		id, parseErr := uuid.Parse(*req.LocationId)
		if parseErr != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid location_id: %v", parseErr)
		}
		input.LocationID = &id
	}
	if req.CreatedBy != nil {
		id, parseErr := uuid.Parse(*req.CreatedBy)
		if parseErr != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid created_by: %v", parseErr)
		}
		input.CreatedBy = &id
	}
	for _, raw := range req.GetItemIds() {
		id, parseErr := uuid.Parse(raw)
		if parseErr != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid item_id %q: %v", raw, parseErr)
		}
		input.ItemIDs = append(input.ItemIDs, id)
	}
	session, err := s.svc.CreateInventurSession(ctx, input)
	if err != nil {
		return nil, mapInventarError(err)
	}
	return &inventarv1.InventurSessionResponse{Session: inventurSessionToProto(session)}, nil
}

func (s *InventarGRPCServer) GetInventurSession(ctx context.Context, req *inventarv1.GetInventurSessionRequest) (*inventarv1.InventurSessionResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	sessionID, err := uuid.Parse(req.GetSessionId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid session_id: %v", err)
	}
	session, err := s.svc.GetInventurSession(ctx, tenantID, sessionID)
	if err != nil {
		return nil, mapInventarError(err)
	}
	return &inventarv1.InventurSessionResponse{Session: inventurSessionToProto(session)}, nil
}

func (s *InventarGRPCServer) UpdateInventurSessionStatus(ctx context.Context, req *inventarv1.UpdateInventurSessionStatusRequest) (*inventarv1.InventurSessionResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	sessionID, err := uuid.Parse(req.GetSessionId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid session_id: %v", err)
	}
	session, err := s.svc.UpdateInventurSessionStatus(ctx, inventar.UpdateInventurSessionStatusInput{
		TenantID:  tenantID,
		SessionID: sessionID,
		Status:    inventar.InventurStatus(req.GetStatus()),
	})
	if err != nil {
		return nil, mapInventarError(err)
	}
	return &inventarv1.InventurSessionResponse{Session: inventurSessionToProto(session)}, nil
}

func (s *InventarGRPCServer) DeleteInventurSession(ctx context.Context, req *inventarv1.DeleteInventurSessionRequest) (*inventarv1.DeleteInventurSessionResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	sessionID, err := uuid.Parse(req.GetSessionId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid session_id: %v", err)
	}
	if err := s.svc.DeleteInventurSession(ctx, tenantID, sessionID); err != nil {
		return nil, mapInventarError(err)
	}
	return &inventarv1.DeleteInventurSessionResponse{}, nil
}

func (s *InventarGRPCServer) ListInventurSessions(ctx context.Context, req *inventarv1.ListInventurSessionsRequest) (*inventarv1.ListInventurSessionsResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	sessions, total, err := s.svc.ListInventurSessions(ctx, inventar.ListInventurSessionsInput{
		TenantID: tenantID,
		Page:     int(req.GetPage()),
		PageSize: int(req.GetPageSize()),
	})
	if err != nil {
		return nil, mapInventarError(err)
	}
	protoSessions := make([]*inventarv1.InventurSession, len(sessions))
	for i, sess := range sessions {
		protoSessions[i] = inventurSessionToProto(sess)
	}
	return &inventarv1.ListInventurSessionsResponse{Sessions: protoSessions, Total: int32(total)}, nil
}

func (s *InventarGRPCServer) UpsertInventurCount(ctx context.Context, req *inventarv1.UpsertInventurCountRequest) (*inventarv1.InventurCountResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	sessionID, err := uuid.Parse(req.GetSessionId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid session_id: %v", err)
	}
	itemID, err := uuid.Parse(req.GetItemId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid item_id: %v", err)
	}
	count, err := s.svc.UpsertInventurCount(ctx, inventar.UpsertInventurCountInput{
		TenantID:  tenantID,
		SessionID: sessionID,
		ItemID:    itemID,
		Counted:   req.GetCounted(),
	})
	if err != nil {
		return nil, mapInventarError(err)
	}
	return &inventarv1.InventurCountResponse{Count: inventurCountToProto(count)}, nil
}

func (s *InventarGRPCServer) BookInventurDifferences(ctx context.Context, req *inventarv1.BookInventurDifferencesRequest) (*inventarv1.InventurSessionResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	sessionID, err := uuid.Parse(req.GetSessionId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid session_id: %v", err)
	}
	input := inventar.BookInventurDifferencesInput{
		TenantID:  tenantID,
		SessionID: sessionID,
	}
	if req.BookedBy != nil {
		id, parseErr := uuid.Parse(*req.BookedBy)
		if parseErr != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid booked_by: %v", parseErr)
		}
		input.BookedBy = &id
	}
	session, err := s.svc.BookInventurDifferences(ctx, input)
	if err != nil {
		return nil, mapInventarError(err)
	}
	return &inventarv1.InventurSessionResponse{Session: inventurSessionToProto(session)}, nil
}

// ============================================================================
// Item Attachment RPCs
// ============================================================================

func (s *InventarGRPCServer) CreateItemAttachment(ctx context.Context, req *inventarv1.CreateItemAttachmentRequest) (*inventarv1.ItemAttachmentResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	itemID, err := uuid.Parse(req.GetItemId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid item_id: %v", err)
	}

	att, err := s.svc.CreateItemAttachment(ctx, inventar.CreateItemAttachmentInput{
		TenantID:  tenantID,
		ItemID:    itemID,
		Name:      req.GetName(),
		ObjectKey: req.GetObjectKey(),
		FileType:  req.GetFileType(),
	})
	if err != nil {
		return nil, mapInventarError(err)
	}
	return &inventarv1.ItemAttachmentResponse{Attachment: itemAttachmentToProto(att)}, nil
}

func (s *InventarGRPCServer) ListItemAttachments(ctx context.Context, req *inventarv1.ListItemAttachmentsRequest) (*inventarv1.ListItemAttachmentsResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	itemID, err := uuid.Parse(req.GetItemId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid item_id: %v", err)
	}

	atts, err := s.svc.ListItemAttachments(ctx, tenantID, itemID)
	if err != nil {
		return nil, mapInventarError(err)
	}
	resp := &inventarv1.ListItemAttachmentsResponse{}
	for _, a := range atts {
		resp.Attachments = append(resp.Attachments, itemAttachmentToProto(a))
	}
	return resp, nil
}

func (s *InventarGRPCServer) DeleteItemAttachment(ctx context.Context, req *inventarv1.DeleteItemAttachmentRequest) (*inventarv1.DeleteItemAttachmentResponse, error) {
	tenantID, err := uuid.Parse(req.GetTenantId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id: %v", err)
	}
	attachmentID, err := uuid.Parse(req.GetAttachmentId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid attachment_id: %v", err)
	}

	if err := s.svc.DeleteItemAttachment(ctx, tenantID, attachmentID); err != nil {
		return nil, mapInventarError(err)
	}
	return &inventarv1.DeleteItemAttachmentResponse{}, nil
}

// ============================================================================
// Conversion helpers
// ============================================================================

func itemAttachmentToProto(a *inventar.ItemAttachment) *inventarv1.ItemAttachment {
	if a == nil {
		return nil
	}
	return &inventarv1.ItemAttachment{
		Id:        a.ID.String(),
		TenantId:  a.TenantID.String(),
		ItemId:    a.ItemID.String(),
		Name:      a.Name,
		ObjectKey: a.ObjectKey,
		FileType:  a.FileType,
		CreatedAt: timestamppb.New(a.CreatedAt),
		UpdatedAt: timestamppb.New(a.UpdatedAt),
	}
}

func itemToProto(item *inventar.Item) *inventarv1.Item {
	if item == nil {
		return nil
	}
	proto := &inventarv1.Item{
		Id:                  item.ID.String(),
		TenantId:            item.TenantID.String(),
		Name:                item.Name,
		Sku:                 item.SKU,
		Barcode:             item.Barcode,
		Quantity:            item.Quantity,
		MinQuantity:         item.MinQuantity,
		Unit:                item.Unit,
		Location:            item.Location,
		Category:            item.Category,
		Price:               item.Price,
		Currency:            item.Currency,
		BatchNumber:         item.BatchNumber,
		SerialNumbers:       item.SerialNumbers,
		LinkedPurchaseOrder: item.LinkedPurchaseOrder,
		CreatedAt:           timestamppb.New(item.CreatedAt),
		UpdatedAt:           timestamppb.New(item.UpdatedAt),
	}
	if item.LocationID != nil {
		s := item.LocationID.String()
		proto.LocationId = &s
	}
	return proto
}

func movementToProto(m *inventar.Movement) *inventarv1.Movement {
	if m == nil {
		return nil
	}
	proto := &inventarv1.Movement{
		Id:           m.ID.String(),
		TenantId:     m.TenantID.String(),
		ItemId:       m.ItemID.String(),
		MovementType: string(m.MovementType),
		Quantity:     m.Quantity,
		Reason:       m.Reason,
		Reference:    m.Reference,
		LocationFrom: m.LocationFrom,
		LocationTo:   m.LocationTo,
		Notes:        m.Notes,
		CreatedAt:    timestamppb.New(m.CreatedAt),
	}
	if m.PerformedBy != nil {
		s := m.PerformedBy.String()
		proto.PerformedBy = &s
	}
	return proto
}

func warningToProto(w *inventar.Warning) *inventarv1.Warning {
	if w == nil {
		return nil
	}
	proto := &inventarv1.Warning{
		Id:              w.ID.String(),
		TenantId:        w.TenantID.String(),
		ItemId:          w.ItemID.String(),
		Threshold:       w.Threshold,
		CurrentQuantity: w.CurrentQuantity,
		Status:          string(w.Status),
		CreatedAt:       timestamppb.New(w.CreatedAt),
	}
	if w.AcknowledgedAt != nil {
		proto.AcknowledgedAt = timestamppb.New(*w.AcknowledgedAt)
	}
	if w.AcknowledgedBy != nil {
		s := w.AcknowledgedBy.String()
		proto.AcknowledgedBy = &s
	}
	return proto
}

func locationToProto(loc *inventar.InventoryLocation) *inventarv1.Location {
	if loc == nil {
		return nil
	}
	return &inventarv1.Location{
		Id:        loc.ID.String(),
		TenantId:  loc.TenantID.String(),
		Name:      loc.Name,
		Address:   loc.Address,
		Type:      string(loc.Type),
		CreatedAt: timestamppb.New(loc.CreatedAt),
		UpdatedAt: timestamppb.New(loc.UpdatedAt),
	}
}

func inventurCountToProto(c *inventar.InventurCount) *inventarv1.InventurCount {
	if c == nil {
		return nil
	}
	proto := &inventarv1.InventurCount{
		Id:        c.ID.String(),
		TenantId:  c.TenantID.String(),
		SessionId: c.SessionID.String(),
		ItemId:    c.ItemID.String(),
		Expected:  c.Expected,
		Counted:   c.Counted,
	}
	if c.CountedAt != nil {
		proto.CountedAt = timestamppb.New(*c.CountedAt)
	}
	return proto
}

func inventurSessionToProto(sess *inventar.InventurSession) *inventarv1.InventurSession {
	if sess == nil {
		return nil
	}
	proto := &inventarv1.InventurSession{
		Id:        sess.ID.String(),
		TenantId:  sess.TenantID.String(),
		Name:      sess.Name,
		Date:      timestamppb.New(sess.Date),
		Status:    string(sess.Status),
		CreatedAt: timestamppb.New(sess.CreatedAt),
		UpdatedAt: timestamppb.New(sess.UpdatedAt),
	}
	if sess.LocationID != nil {
		s := sess.LocationID.String()
		proto.LocationId = &s
	}
	if sess.CreatedBy != nil {
		s := sess.CreatedBy.String()
		proto.CreatedBy = &s
	}
	for i := range sess.Counts {
		proto.Counts = append(proto.Counts, inventurCountToProto(&sess.Counts[i]))
	}
	return proto
}

// ============================================================================
// Error mapping
// ============================================================================

func mapInventarError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, inventar.ErrItemNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, inventar.ErrMovementNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, inventar.ErrWarningNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, inventar.ErrLocationNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, inventar.ErrInventurSessionNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, inventar.ErrInventurCountNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, inventar.ErrAttachmentNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, inventar.ErrInventurAlreadyCompleted):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, inventar.ErrSKUTaken):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, inventar.ErrInvalidInput):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		slog.Error("unhandled inventar service error", "error", err)
		return status.Error(codes.Internal, "internal error")
	}
}
