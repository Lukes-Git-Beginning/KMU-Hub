package gateway

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/kmuhub/kmuhub/internal/featureflag"
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server/response"
	inventarv1 "github.com/kmuhub/kmuhub/proto/inventar/v1"
)

// InventarRoutes handles HTTP routes for the Inventar (inventory) module.
type InventarRoutes struct {
	registry *ServiceRegistry
	flags    *featureflag.Registry
}

// NewInventarRoutes creates a new InventarRoutes with the given service registry and feature flags.
func NewInventarRoutes(registry *ServiceRegistry, flags *featureflag.Registry) *InventarRoutes {
	return &InventarRoutes{registry: registry, flags: flags}
}

// ServiceName returns the service name for the route registrar.
func (ir *InventarRoutes) ServiceName() string { return "inventar" }

// getClient lazily obtains a gRPC client for the inventar service.
func (ir *InventarRoutes) getClient() (inventarv1.InventarServiceClient, error) {
	conn, err := ir.registry.GetConnection("inventar")
	if err != nil {
		return nil, err
	}
	return inventarv1.NewInventarServiceClient(conn), nil
}


// RegisterRoutes mounts all Inventar HTTP routes behind the feature flag modules.inventar.
// Routes are only registered if the flag is enabled.
func (ir *InventarRoutes) RegisterRoutes(r chi.Router, authMiddleware func(http.Handler) http.Handler) {
	if !ir.flags.IsEnabled("modules.inventar") {
		return
	}

	r.Route("/api/v1/inventar", func(r chi.Router) {
		r.Use(authMiddleware)
		r.Use(RequireAuthenticated)

		// Items
		r.Route("/items", func(r chi.Router) {
			r.With(middleware.RequirePermission("inventar:item", "read")).Get("/", ir.HandleListItems)
			r.With(middleware.RequirePermission("inventar:item", "write")).Post("/", ir.HandleCreateItem)

			r.Route("/{id}", func(r chi.Router) {
				r.With(middleware.RequirePermission("inventar:item", "read")).Get("/", ir.HandleGetItem)
				r.With(middleware.RequirePermission("inventar:item", "write")).Patch("/", ir.HandleUpdateItem)
				r.With(middleware.RequirePermission("inventar:item", "write")).Delete("/", ir.HandleDeleteItem)

				r.With(middleware.RequirePermission("inventar:item", "write")).Post("/adjust", ir.HandleAdjustStock)

				// Movements
				r.With(middleware.RequirePermission("inventar:movement", "read")).Get("/movements", ir.HandleListMovements)
				r.With(middleware.RequirePermission("inventar:movement", "read")).Get("/history", ir.HandleGetStockHistory)
				r.With(middleware.RequirePermission("inventar:movement", "write")).Post("/movements", ir.HandleRecordMovement)
			})
		})

		// Transfer between items
		r.With(middleware.RequirePermission("inventar:item", "write")).Post("/transfer", ir.HandleTransferStock)

		// Warnings
		r.Route("/warnings", func(r chi.Router) {
			r.With(middleware.RequirePermission("inventar:warning", "read")).Get("/", ir.HandleListWarnings)
			r.With(middleware.RequirePermission("inventar:warning", "write")).Post("/", ir.HandleCreateWarning)

			r.Route("/{id}", func(r chi.Router) {
				r.With(middleware.RequirePermission("inventar:warning", "write")).Patch("/", ir.HandleUpdateWarning)
				r.With(middleware.RequirePermission("inventar:warning", "write")).Post("/acknowledge", ir.HandleAcknowledgeWarning)
			})
		})

		// Reports
		r.With(middleware.RequirePermission("inventar:item", "read")).Get("/report", ir.HandleGetStockReport)
		r.With(middleware.RequirePermission("inventar:item", "read")).Get("/export", ir.HandleExportInventory)
	})
}

// ============================================================================
// Request types
// ============================================================================

type createItemRequest struct {
	Name        string  `json:"name"`
	SKU         string  `json:"sku"`
	Barcode     *string `json:"barcode,omitempty"`
	Quantity    int64   `json:"quantity"`
	MinQuantity int64   `json:"min_quantity"`
	Unit        string  `json:"unit"`
	Location    *string `json:"location,omitempty"`
}

type updateItemRequest struct {
	Name        *string `json:"name,omitempty"`
	SKU         *string `json:"sku,omitempty"`
	Barcode     *string `json:"barcode,omitempty"`
	MinQuantity *int64  `json:"min_quantity,omitempty"`
	Unit        *string `json:"unit,omitempty"`
	Location    *string `json:"location,omitempty"`
}

type adjustStockRequest struct {
	Delta       int64   `json:"delta"`
	PerformedBy *string `json:"performed_by,omitempty"`
	Reason      string  `json:"reason"`
}

type transferStockRequest struct {
	FromItemID  string  `json:"from_item_id"`
	ToItemID    string  `json:"to_item_id"`
	Quantity    int64   `json:"quantity"`
	PerformedBy *string `json:"performed_by,omitempty"`
	Reason      string  `json:"reason"`
}

type recordMovementRequest struct {
	MovementType string  `json:"movement_type"`
	Quantity     int64   `json:"quantity"`
	PerformedBy  *string `json:"performed_by,omitempty"`
	Reason       string  `json:"reason"`
}

type createWarningRequest struct {
	ItemID          string `json:"item_id"`
	Threshold       int64  `json:"threshold"`
	CurrentQuantity int64  `json:"current_quantity"`
}

type updateWarningRequest struct {
	Status string `json:"status"`
}

type acknowledgeWarningRequest struct {
	AcknowledgedBy string `json:"acknowledged_by"`
}

// ============================================================================
// Item Handlers
// ============================================================================

func (ir *InventarRoutes) HandleListItems(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}
	client, err := ir.getClient()
	if err != nil {
		respondServiceUnavailable(w, ir.ServiceName())
		return
	}

	page, pageSize := parsePagination(r, 1, 50)
	q := r.URL.Query()

	grpcReq := &inventarv1.ListItemsRequest{
		TenantId: tenantID.String(),
		Search:   q.Get("search"),
		Page:     int32(page),
		PageSize: int32(pageSize),
	}
	if loc := q.Get("location"); loc != "" {
		grpcReq.Location = &loc
	}
	if ls := q.Get("low_stock"); ls == "true" || ls == "1" {
		grpcReq.LowStock = true
	}

	resp, err := client.ListItems(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

func (ir *InventarRoutes) HandleCreateItem(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}
	client, err := ir.getClient()
	if err != nil {
		respondServiceUnavailable(w, ir.ServiceName())
		return
	}

	var req createItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		response.Error(w, http.StatusBadRequest, "name is required")
		return
	}
	if req.SKU == "" {
		response.Error(w, http.StatusBadRequest, "sku is required")
		return
	}

	grpcReq := &inventarv1.CreateItemRequest{
		TenantId:    tenantID.String(),
		Name:        req.Name,
		Sku:         req.SKU,
		Barcode:     req.Barcode,
		Quantity:    req.Quantity,
		MinQuantity: req.MinQuantity,
		Unit:        req.Unit,
		Location:    req.Location,
	}

	resp, err := client.CreateItem(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, resp)
}

func (ir *InventarRoutes) HandleGetItem(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}
	client, err := ir.getClient()
	if err != nil {
		respondServiceUnavailable(w, ir.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.GetItem(r.Context(), &inventarv1.GetItemRequest{
		TenantId: tenantID.String(),
		ItemId:   id,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

func (ir *InventarRoutes) HandleUpdateItem(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}
	client, err := ir.getClient()
	if err != nil {
		respondServiceUnavailable(w, ir.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	var req updateItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	grpcReq := &inventarv1.UpdateItemRequest{
		TenantId: tenantID.String(),
		ItemId:   id,
		Name:     req.Name,
		Sku:      req.SKU,
		Barcode:  req.Barcode,
		Unit:     req.Unit,
		Location: req.Location,
	}
	if req.MinQuantity != nil {
		grpcReq.MinQuantity = req.MinQuantity
	}

	resp, err := client.UpdateItem(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

func (ir *InventarRoutes) HandleDeleteItem(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}
	client, err := ir.getClient()
	if err != nil {
		respondServiceUnavailable(w, ir.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	_, err = client.DeleteItem(r.Context(), &inventarv1.DeleteItemRequest{
		TenantId: tenantID.String(),
		ItemId:   id,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ============================================================================
// Stock Operation Handlers
// ============================================================================

func (ir *InventarRoutes) HandleAdjustStock(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}
	client, err := ir.getClient()
	if err != nil {
		respondServiceUnavailable(w, ir.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	var req adjustStockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	grpcReq := &inventarv1.AdjustStockRequest{
		TenantId:    tenantID.String(),
		ItemId:      id,
		Delta:       req.Delta,
		PerformedBy: req.PerformedBy,
		Reason:      req.Reason,
	}

	resp, err := client.AdjustStock(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

func (ir *InventarRoutes) HandleTransferStock(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}
	client, err := ir.getClient()
	if err != nil {
		respondServiceUnavailable(w, ir.ServiceName())
		return
	}

	var req transferStockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.FromItemID == "" || req.ToItemID == "" {
		response.Error(w, http.StatusBadRequest, "from_item_id and to_item_id are required")
		return
	}

	grpcReq := &inventarv1.TransferStockRequest{
		TenantId:    tenantID.String(),
		FromItemId:  req.FromItemID,
		ToItemId:    req.ToItemID,
		Quantity:    req.Quantity,
		PerformedBy: req.PerformedBy,
		Reason:      req.Reason,
	}

	_, err = client.TransferStock(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (ir *InventarRoutes) HandleListMovements(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}
	client, err := ir.getClient()
	if err != nil {
		respondServiceUnavailable(w, ir.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	page, pageSize := parsePagination(r, 1, 50)

	resp, err := client.ListMovements(r.Context(), &inventarv1.ListMovementsRequest{
		TenantId: tenantID.String(),
		ItemId:   id,
		Page:     int32(page),
		PageSize: int32(pageSize),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

func (ir *InventarRoutes) HandleGetStockHistory(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}
	client, err := ir.getClient()
	if err != nil {
		respondServiceUnavailable(w, ir.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	page, pageSize := parsePagination(r, 1, 50)

	resp, err := client.GetStockHistory(r.Context(), &inventarv1.GetStockHistoryRequest{
		TenantId: tenantID.String(),
		ItemId:   id,
		Page:     int32(page),
		PageSize: int32(pageSize),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

func (ir *InventarRoutes) HandleRecordMovement(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}
	client, err := ir.getClient()
	if err != nil {
		respondServiceUnavailable(w, ir.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	var req recordMovementRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	grpcReq := &inventarv1.RecordMovementRequest{
		TenantId:     tenantID.String(),
		ItemId:       id,
		MovementType: req.MovementType,
		Quantity:     req.Quantity,
		PerformedBy:  req.PerformedBy,
		Reason:       req.Reason,
	}

	resp, err := client.RecordMovement(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, resp)
}

// ============================================================================
// Warning Handlers
// ============================================================================

func (ir *InventarRoutes) HandleListWarnings(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}
	client, err := ir.getClient()
	if err != nil {
		respondServiceUnavailable(w, ir.ServiceName())
		return
	}

	page, pageSize := parsePagination(r, 1, 50)
	q := r.URL.Query()

	grpcReq := &inventarv1.ListWarningsRequest{
		TenantId: tenantID.String(),
		Page:     int32(page),
		PageSize: int32(pageSize),
	}
	if s := q.Get("status"); s != "" {
		grpcReq.Status = &s
	}

	resp, err := client.ListWarnings(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

func (ir *InventarRoutes) HandleCreateWarning(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}
	client, err := ir.getClient()
	if err != nil {
		respondServiceUnavailable(w, ir.ServiceName())
		return
	}

	var req createWarningRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.ItemID == "" {
		response.Error(w, http.StatusBadRequest, "item_id is required")
		return
	}

	resp, err := client.CreateWarning(r.Context(), &inventarv1.CreateWarningRequest{
		TenantId:        tenantID.String(),
		ItemId:          req.ItemID,
		Threshold:       req.Threshold,
		CurrentQuantity: req.CurrentQuantity,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, resp)
}

func (ir *InventarRoutes) HandleUpdateWarning(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}
	client, err := ir.getClient()
	if err != nil {
		respondServiceUnavailable(w, ir.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	var req updateWarningRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := client.UpdateWarning(r.Context(), &inventarv1.UpdateWarningRequest{
		TenantId:  tenantID.String(),
		WarningId: id,
		Status:    req.Status,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

func (ir *InventarRoutes) HandleAcknowledgeWarning(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}
	client, err := ir.getClient()
	if err != nil {
		respondServiceUnavailable(w, ir.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	var req acknowledgeWarningRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.AcknowledgedBy == "" {
		// Fall back to the authenticated user
		req.AcknowledgedBy = middleware.GetUserID(r.Context())
	}

	resp, err := client.AcknowledgeWarning(r.Context(), &inventarv1.AcknowledgeWarningRequest{
		TenantId:       tenantID.String(),
		WarningId:      id,
		AcknowledgedBy: req.AcknowledgedBy,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

// ============================================================================
// Report Handlers
// ============================================================================

func (ir *InventarRoutes) HandleGetStockReport(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}
	client, err := ir.getClient()
	if err != nil {
		respondServiceUnavailable(w, ir.ServiceName())
		return
	}

	resp, err := client.GetStockReport(r.Context(), &inventarv1.GetStockReportRequest{
		TenantId: tenantID.String(),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

func (ir *InventarRoutes) HandleExportInventory(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}
	client, err := ir.getClient()
	if err != nil {
		respondServiceUnavailable(w, ir.ServiceName())
		return
	}

	format := r.URL.Query().Get("format")
	if format == "" {
		format = "csv"
	}

	resp, err := client.ExportInventory(r.Context(), &inventarv1.ExportInventoryRequest{
		TenantId: tenantID.String(),
		Format:   format,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	contentType := resp.GetContentType()
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	filename := resp.GetFilename()
	if filename == "" {
		filename = "inventar.csv"
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", "attachment; filename="+formatFilename(filename))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(resp.GetPayload())
}
