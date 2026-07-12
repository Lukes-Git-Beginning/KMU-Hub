package gateway

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/kmuhub/kmuhub/internal/featureflag"
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server/response"
	einkaufv1 "github.com/kmuhub/kmuhub/proto/einkauf/v1"
)

// EinkaufRoutes handles HTTP routes for the Einkauf (purchasing) module.
type EinkaufRoutes struct {
	registry *ServiceRegistry
	flags    *featureflag.Registry
}

// NewEinkaufRoutes creates a new EinkaufRoutes with the given service registry and feature flags.
func NewEinkaufRoutes(registry *ServiceRegistry, flags *featureflag.Registry) *EinkaufRoutes {
	return &EinkaufRoutes{registry: registry, flags: flags}
}

// ServiceName returns the service name for the route registrar.
func (er *EinkaufRoutes) ServiceName() string { return "einkauf" }

// getClient lazily obtains a gRPC client for the einkauf service.
func (er *EinkaufRoutes) getClient() (einkaufv1.EinkaufServiceClient, error) {
	conn, err := er.registry.GetConnection("einkauf")
	if err != nil {
		return nil, err
	}
	return einkaufv1.NewEinkaufServiceClient(conn), nil
}

// RegisterRoutes mounts all Einkauf HTTP routes behind the feature flag modules.einkauf.
// Routes are only registered if the flag is enabled.
func (er *EinkaufRoutes) RegisterRoutes(r chi.Router, authMiddleware func(http.Handler) http.Handler) {
	if !er.flags.IsEnabled("modules.einkauf") {
		return
	}

	r.Route("/api/v1/einkauf", func(r chi.Router) {
		r.Use(authMiddleware)
		r.Use(RequireAuthenticated)

		// Suppliers
		r.Route("/suppliers", func(r chi.Router) {
			r.With(middleware.RequirePermission("einkauf:supplier", "read")).Get("/", er.HandleListSuppliers)
			r.With(middleware.RequirePermission("einkauf:supplier", "write")).Post("/", er.HandleCreateSupplier)

			r.Route("/{id}", func(r chi.Router) {
				r.With(middleware.RequirePermission("einkauf:supplier", "read")).Get("/", er.HandleGetSupplier)
				r.With(middleware.RequirePermission("einkauf:supplier", "write")).Patch("/", er.HandleUpdateSupplier)
				r.With(middleware.RequirePermission("einkauf:supplier", "write")).Delete("/", er.HandleDeleteSupplier)
			})
		})

		// Purchase Orders
		r.Route("/pos", func(r chi.Router) {
			r.With(middleware.RequirePermission("einkauf:po", "read")).Get("/", er.HandleListPOs)
			r.With(middleware.RequirePermission("einkauf:po", "write")).Post("/", er.HandleCreatePO)

			r.Route("/{id}", func(r chi.Router) {
				r.With(middleware.RequirePermission("einkauf:po", "read")).Get("/", er.HandleGetPO)
				r.With(middleware.RequirePermission("einkauf:po", "write")).Patch("/", er.HandleUpdatePO)
				r.With(middleware.RequirePermission("einkauf:po", "write")).Delete("/", er.HandleDeletePO)

				// Workflow
				r.With(middleware.RequirePermission("einkauf:po", "write")).Post("/submit", er.HandleSubmitPO)
				r.With(middleware.RequirePermission("einkauf:po", "write")).Post("/receive", er.HandleReceiveGoods)
				r.With(middleware.RequirePermission("einkauf:po", "write")).Post("/partial-receive", er.HandlePartialReceive)
				r.With(middleware.RequirePermission("einkauf:po", "read")).Post("/export", er.HandleExportPO)

				// Lines
				r.Route("/lines", func(r chi.Router) {
					r.With(middleware.RequirePermission("einkauf:line", "read")).Get("/", er.HandleListPOLines)
					r.With(middleware.RequirePermission("einkauf:line", "write")).Post("/", er.HandleAddPOLine)

					r.Route("/{lineId}", func(r chi.Router) {
						r.With(middleware.RequirePermission("einkauf:line", "write")).Patch("/", er.HandleUpdatePOLine)
						r.With(middleware.RequirePermission("einkauf:line", "write")).Delete("/", er.HandleDeletePOLine)
					})
				})
			})
		})

		// Extended routes: Catalog, Supplier Ratings, Framework Contracts
		er.registerExtendedRoutes(r)
	})
}

// ============================================================================
// Request types
// ============================================================================

type createSupplierRequest struct {
	Name         string `json:"name"               validate:"required"`
	ContactID    string `json:"contact_id,omitempty" validate:"omitempty,uuid"`
	Email        string `json:"email,omitempty"    validate:"omitempty,email"`
	Phone        string `json:"phone,omitempty"`
	Address      string `json:"address,omitempty"`
	PaymentTerms string `json:"payment_terms,omitempty"`
	Notes        string `json:"notes,omitempty"`
}

type updateSupplierRequest struct {
	Name         *string `json:"name,omitempty"`
	ContactID    *string `json:"contact_id,omitempty"` // empty string to clear
	Email        *string `json:"email,omitempty"    validate:"omitempty,email"`
	Phone        *string `json:"phone,omitempty"`
	Address      *string `json:"address,omitempty"`
	PaymentTerms *string `json:"payment_terms,omitempty"`
	Notes        *string `json:"notes,omitempty"`
}

type createPORequest struct {
	SupplierID           string `json:"supplier_id"             validate:"required,uuid"`
	PONumber             string `json:"po_number"               validate:"required"`
	OrderDate            string `json:"order_date,omitempty"`
	ExpectedDeliveryDate string `json:"expected_delivery_date,omitempty"`
	Currency             string `json:"currency,omitempty"`
	Notes                string `json:"notes,omitempty"`
}

type updatePORequest struct {
	SupplierID           *string `json:"supplier_id,omitempty"`
	PONumber             *string `json:"po_number,omitempty"`
	OrderDate            *string `json:"order_date,omitempty"`
	ExpectedDeliveryDate *string `json:"expected_delivery_date,omitempty"`
	Currency             *string `json:"currency,omitempty"`
	Notes                *string `json:"notes,omitempty"`
}

type addPOLineRequest struct {
	ProductName  string `json:"product_name"          validate:"required"`
	SKU          string `json:"sku,omitempty"`
	Quantity     string `json:"quantity"              validate:"required,decimal_gt0"`
	UnitPrice    string `json:"unit_price"            validate:"required,decimal_gte0"`
	TaxRate      string `json:"tax_rate,omitempty"`
	LinePosition int32  `json:"line_position,omitempty"`
}

type updatePOLineRequest struct {
	ProductName  *string `json:"product_name,omitempty"`
	SKU          *string `json:"sku,omitempty"`
	Quantity     *string `json:"quantity,omitempty"     validate:"omitempty,decimal_gt0"`
	UnitPrice    *string `json:"unit_price,omitempty"   validate:"omitempty,decimal_gte0"`
	TaxRate      *string `json:"tax_rate,omitempty"`
	LinePosition *int32  `json:"line_position,omitempty"`
}

type partialReceiveItem struct {
	LineID           string `json:"line_id"`
	ReceivedQuantity string `json:"received_quantity"`
}

type partialReceiveRequest struct {
	Items []partialReceiveItem `json:"items" validate:"required,min=1"`
}

// ============================================================================
// Supplier Handlers
// ============================================================================

func (er *EinkaufRoutes) HandleListSuppliers(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := er.getClient()
	if err != nil {
		respondServiceUnavailable(w, er.ServiceName())
		return
	}

	page, pageSize := parsePagination(r, 1, 20)
	q := r.URL.Query()

	grpcReq := &einkaufv1.ListSuppliersRequest{
		TenantId: tenantID.String(),
		Search:   q.Get("search"),
		Page:     int32(page),
		PageSize: int32(pageSize),
	}
	if active := q.Get("active_only"); active == "true" || active == "1" {
		grpcReq.ActiveOnly = true
	}

	resp, err := client.ListSuppliers(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

func (er *EinkaufRoutes) HandleCreateSupplier(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := er.getClient()
	if err != nil {
		respondServiceUnavailable(w, er.ServiceName())
		return
	}

	req, ok := decodeAndValidate[createSupplierRequest](w, r)
	if !ok {
		return
	}

	grpcReq := &einkaufv1.CreateSupplierRequest{
		TenantId:     tenantID.String(),
		Name:         req.Name,
		Email:        req.Email,
		Phone:        req.Phone,
		Address:      req.Address,
		PaymentTerms: req.PaymentTerms,
		Notes:        req.Notes,
	}
	if req.ContactID != "" {
		grpcReq.ContactId = &req.ContactID
	}

	resp, err := client.CreateSupplier(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusCreated, resp)
}

func (er *EinkaufRoutes) HandleGetSupplier(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := er.getClient()
	if err != nil {
		respondServiceUnavailable(w, er.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.GetSupplier(r.Context(), &einkaufv1.GetSupplierRequest{
		TenantId:   tenantID.String(),
		SupplierId: id,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

func (er *EinkaufRoutes) HandleUpdateSupplier(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := er.getClient()
	if err != nil {
		respondServiceUnavailable(w, er.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	req, ok := decodeAndValidate[updateSupplierRequest](w, r)
	if !ok {
		return
	}

	grpcReq := &einkaufv1.UpdateSupplierRequest{
		TenantId:     tenantID.String(),
		SupplierId:   id,
		Name:         req.Name,
		Email:        req.Email,
		Phone:        req.Phone,
		Address:      req.Address,
		PaymentTerms: req.PaymentTerms,
		Notes:        req.Notes,
		ContactId:    req.ContactID,
	}

	resp, err := client.UpdateSupplier(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

func (er *EinkaufRoutes) HandleDeleteSupplier(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := er.getClient()
	if err != nil {
		respondServiceUnavailable(w, er.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	_, err = client.DeleteSupplier(r.Context(), &einkaufv1.DeleteSupplierRequest{
		TenantId:   tenantID.String(),
		SupplierId: id,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ============================================================================
// Purchase Order Handlers
// ============================================================================

func (er *EinkaufRoutes) HandleListPOs(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := er.getClient()
	if err != nil {
		respondServiceUnavailable(w, er.ServiceName())
		return
	}

	page, pageSize := parsePagination(r, 1, 20)
	q := r.URL.Query()

	grpcReq := &einkaufv1.ListPOsRequest{
		TenantId: tenantID.String(),
		Page:     int32(page),
		PageSize: int32(pageSize),
	}
	if supplierID := q.Get("supplier_id"); supplierID != "" {
		grpcReq.SupplierId = &supplierID
	}
	if st := q.Get("status"); st != "" {
		grpcReq.Status = &st
	}

	resp, err := client.ListPOs(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

func (er *EinkaufRoutes) HandleCreatePO(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := er.getClient()
	if err != nil {
		respondServiceUnavailable(w, er.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	req, ok := decodeAndValidate[createPORequest](w, r)
	if !ok {
		return
	}

	grpcReq := &einkaufv1.CreatePORequest{
		TenantId:   tenantID.String(),
		SupplierId: req.SupplierID,
		PoNumber:   req.PONumber,
		Currency:   req.Currency,
		Notes:      req.Notes,
		CreatedBy:  &userID,
	}

	resp, err := client.CreatePO(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusCreated, resp)
}

func (er *EinkaufRoutes) HandleGetPO(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := er.getClient()
	if err != nil {
		respondServiceUnavailable(w, er.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.GetPO(r.Context(), &einkaufv1.GetPORequest{
		TenantId: tenantID.String(),
		PoId:     id,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

func (er *EinkaufRoutes) HandleUpdatePO(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := er.getClient()
	if err != nil {
		respondServiceUnavailable(w, er.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	req, ok := decodeAndValidate[updatePORequest](w, r)
	if !ok {
		return
	}

	grpcReq := &einkaufv1.UpdatePORequest{
		TenantId:   tenantID.String(),
		PoId:       id,
		SupplierId: req.SupplierID,
		PoNumber:   req.PONumber,
		Currency:   req.Currency,
		Notes:      req.Notes,
	}

	resp, err := client.UpdatePO(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

func (er *EinkaufRoutes) HandleDeletePO(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := er.getClient()
	if err != nil {
		respondServiceUnavailable(w, er.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	_, err = client.DeletePO(r.Context(), &einkaufv1.DeletePORequest{
		TenantId: tenantID.String(),
		PoId:     id,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ============================================================================
// Workflow Handlers
// ============================================================================

func (er *EinkaufRoutes) HandleSubmitPO(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := er.getClient()
	if err != nil {
		respondServiceUnavailable(w, er.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.SubmitPO(r.Context(), &einkaufv1.SubmitPORequest{
		TenantId: tenantID.String(),
		PoId:     id,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

func (er *EinkaufRoutes) HandleReceiveGoods(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := er.getClient()
	if err != nil {
		respondServiceUnavailable(w, er.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.ReceiveGoods(r.Context(), &einkaufv1.ReceiveGoodsRequest{
		TenantId: tenantID.String(),
		PoId:     id,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

func (er *EinkaufRoutes) HandlePartialReceive(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := er.getClient()
	if err != nil {
		respondServiceUnavailable(w, er.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	req, ok := decodeAndValidate[partialReceiveRequest](w, r)
	if !ok {
		return
	}

	protoItems := make([]*einkaufv1.PartialReceiveItem, 0, len(req.Items))
	for _, item := range req.Items {
		protoItems = append(protoItems, &einkaufv1.PartialReceiveItem{
			LineId:           item.LineID,
			ReceivedQuantity: item.ReceivedQuantity,
		})
	}

	resp, err := client.PartialReceive(r.Context(), &einkaufv1.PartialReceiveRequest{
		TenantId: tenantID.String(),
		PoId:     id,
		Items:    protoItems,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

func (er *EinkaufRoutes) HandleExportPO(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := er.getClient()
	if err != nil {
		respondServiceUnavailable(w, er.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	format := r.URL.Query().Get("format")
	if format == "" {
		format = "pdf"
	}

	resp, err := client.ExportPO(r.Context(), &einkaufv1.ExportPORequest{
		TenantId: tenantID.String(),
		PoId:     id,
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
		filename = "po." + format
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", "attachment; filename="+formatFilename(filename))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(resp.GetPayload())
}

// ============================================================================
// PO Line Handlers
// ============================================================================

func (er *EinkaufRoutes) HandleListPOLines(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := er.getClient()
	if err != nil {
		respondServiceUnavailable(w, er.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.ListPOLines(r.Context(), &einkaufv1.ListPOLinesRequest{
		TenantId: tenantID.String(),
		PoId:     id,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

func (er *EinkaufRoutes) HandleAddPOLine(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := er.getClient()
	if err != nil {
		respondServiceUnavailable(w, er.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	req, ok := decodeAndValidate[addPOLineRequest](w, r)
	if !ok {
		return
	}

	grpcReq := &einkaufv1.AddPOLineRequest{
		TenantId:     tenantID.String(),
		PoId:         id,
		ProductName:  req.ProductName,
		Sku:          req.SKU,
		Quantity:     req.Quantity,
		UnitPrice:    req.UnitPrice,
		TaxRate:      req.TaxRate,
		LinePosition: req.LinePosition,
	}

	resp, err := client.AddPOLine(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusCreated, resp)
}

func (er *EinkaufRoutes) HandleUpdatePOLine(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := er.getClient()
	if err != nil {
		respondServiceUnavailable(w, er.ServiceName())
		return
	}

	_, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	lineID, ok := validateUUIDParam(w, r, "lineId")
	if !ok {
		return
	}

	req, ok := decodeAndValidate[updatePOLineRequest](w, r)
	if !ok {
		return
	}

	grpcReq := &einkaufv1.UpdatePOLineRequest{
		TenantId:     tenantID.String(),
		LineId:       lineID,
		ProductName:  req.ProductName,
		Sku:          req.SKU,
		Quantity:     req.Quantity,
		UnitPrice:    req.UnitPrice,
		TaxRate:      req.TaxRate,
		LinePosition: req.LinePosition,
	}

	resp, err := client.UpdatePOLine(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

func (er *EinkaufRoutes) HandleDeletePOLine(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := er.getClient()
	if err != nil {
		respondServiceUnavailable(w, er.ServiceName())
		return
	}

	_, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	lineID, ok := validateUUIDParam(w, r, "lineId")
	if !ok {
		return
	}

	_, err = client.DeletePOLine(r.Context(), &einkaufv1.DeletePOLineRequest{
		TenantId: tenantID.String(),
		LineId:   lineID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
