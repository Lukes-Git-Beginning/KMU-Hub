package gateway

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server/response"
	einkaufv1 "github.com/kmuhub/kmuhub/proto/einkauf/v1"
)

// ============================================================================
// Route registration — called from EinkaufRoutes.RegisterRoutes
// ============================================================================

// registerExtendedRoutes mounts catalog, supplier-rating, and framework-contract
// sub-routes onto the provided router. It is called inside the authenticated
// /api/v1/einkauf route group.
func (er *EinkaufRoutes) registerExtendedRoutes(r chi.Router) {
	// Additive RBAC guards — see route_einkauf.go for the pattern rationale.
	ratingCreate := middleware.RequirePermissionAny([2]string{"einkauf:supplier", "write"}, [2]string{"einkauf:rating", "create"})
	contractCall := middleware.RequirePermissionAny([2]string{"einkauf:contract", "write"}, [2]string{"einkauf:contract", "call"})

	// Catalog
	r.Route("/catalog", func(r chi.Router) {
		r.With(middleware.RequirePermission("einkauf:catalog", "read")).Get("/", er.HandleListCatalogItems)
		r.With(middleware.RequirePermission("einkauf:catalog", "write")).Post("/", er.HandleCreateCatalogItem)

		r.Route("/{id}", func(r chi.Router) {
			r.With(middleware.RequirePermission("einkauf:catalog", "read")).Get("/", er.HandleGetCatalogItem)
			r.With(middleware.RequirePermission("einkauf:catalog", "write")).Patch("/", er.HandleUpdateCatalogItem)
			r.With(middleware.RequirePermission("einkauf:catalog", "write")).Delete("/", er.HandleDeleteCatalogItem)
		})
	})

	// Supplier ratings — scoped under /suppliers/:supplierId/ratings
	r.Route("/suppliers/{supplierId}/ratings", func(r chi.Router) {
		r.With(middleware.RequirePermission("einkauf:supplier", "read")).Get("/", er.HandleListSupplierRatings)
		r.With(ratingCreate).Post("/", er.HandleCreateSupplierRating)
		// No FE caller for rating deletion — legacy guard stays as-is.
		r.With(middleware.RequirePermission("einkauf:supplier", "write")).Delete("/{ratingId}", er.HandleDeleteSupplierRating)
	})

	// Framework contracts
	r.Route("/contracts", func(r chi.Router) {
		r.With(middleware.RequirePermission("einkauf:contract", "read")).Get("/", er.HandleListFrameworkContracts)
		r.With(middleware.RequirePermission("einkauf:contract", "write")).Post("/", er.HandleCreateFrameworkContract)

		r.Route("/{id}", func(r chi.Router) {
			r.With(middleware.RequirePermission("einkauf:contract", "read")).Get("/", er.HandleGetFrameworkContract)
			r.With(middleware.RequirePermission("einkauf:contract", "write")).Patch("/", er.HandleUpdateFrameworkContract)
			r.With(middleware.RequirePermission("einkauf:contract", "write")).Delete("/", er.HandleDeleteFrameworkContract)

			// Contract items
			r.Route("/items", func(r chi.Router) {
				r.With(middleware.RequirePermission("einkauf:contract", "write")).Post("/", er.HandleCreateContractItem)
				r.With(middleware.RequirePermission("einkauf:contract", "write")).Patch("/{itemId}", er.HandleUpdateContractItem)
				r.With(middleware.RequirePermission("einkauf:contract", "write")).Delete("/{itemId}", er.HandleDeleteContractItem)
			})

			// Contract call-offs
			r.Route("/calls", func(r chi.Router) {
				r.With(middleware.RequirePermission("einkauf:contract", "read")).Get("/", er.HandleListContractCalls)
				r.With(contractCall).Post("/", er.HandleCreateContractCall)
			})
		})
	})
}

// ============================================================================
// Request body types
// ============================================================================

type createCatalogItemRequest struct {
	SupplierID  string `json:"supplier_id"   validate:"required,uuid"`
	Name        string `json:"name"          validate:"required"`
	SKU         string `json:"sku,omitempty"`
	Category    string `json:"category,omitempty"`
	Price       string `json:"price"         validate:"required,decimal_gte0"`
	Currency    string `json:"currency,omitempty"`
	Unit        string `json:"unit,omitempty"`
	Available   bool   `json:"available"`
	MinOrderQty string `json:"min_order_qty,omitempty"`
}

type updateCatalogItemRequest struct {
	Name        *string `json:"name,omitempty"`
	SKU         *string `json:"sku,omitempty"`
	Category    *string `json:"category,omitempty"`
	Price       *string `json:"price,omitempty" validate:"omitempty,decimal_gte0"`
	Currency    *string `json:"currency,omitempty"`
	Unit        *string `json:"unit,omitempty"`
	Available   *bool   `json:"available,omitempty"`
	MinOrderQty *string `json:"min_order_qty,omitempty"`
}

type createSupplierRatingRequest struct {
	Category string `json:"category" validate:"required"`
	Rating   int32  `json:"rating"   validate:"required"`
	Comment  string `json:"comment,omitempty"`
}

type createFrameworkContractRequest struct {
	SupplierID string `json:"supplier_id"  validate:"required,uuid"`
	Title      string `json:"title"        validate:"required"`
	ContractNr string `json:"contract_nr,omitempty"`
	StartDate  string `json:"start_date,omitempty"`
	EndDate    string `json:"end_date,omitempty"`
	TotalValue string `json:"total_value,omitempty" validate:"omitempty,decimal_gte0"`
	Currency   string `json:"currency,omitempty"`
	Status     string `json:"status,omitempty"`
}

type updateFrameworkContractRequest struct {
	SupplierID *string `json:"supplier_id,omitempty"`
	Title      *string `json:"title,omitempty"`
	ContractNr *string `json:"contract_nr,omitempty"`
	StartDate  *string `json:"start_date,omitempty"`
	EndDate    *string `json:"end_date,omitempty"`
	TotalValue *string `json:"total_value,omitempty" validate:"omitempty,decimal_gte0"`
	Currency   *string `json:"currency,omitempty"`
	Status     *string `json:"status,omitempty"`
}

type createContractItemRequest struct {
	Name      string `json:"name"       validate:"required"`
	UnitPrice string `json:"unit_price,omitempty" validate:"omitempty,decimal_gte0"`
	Unit      string `json:"unit,omitempty"`
	AgreedQty string `json:"agreed_qty,omitempty"`
}

type updateContractItemRequest struct {
	Name      *string `json:"name,omitempty"`
	UnitPrice *string `json:"unit_price,omitempty" validate:"omitempty,decimal_gte0"`
	Unit      *string `json:"unit,omitempty"`
	AgreedQty *string `json:"agreed_qty,omitempty"`
}

type createContractCallRequest struct {
	POID     string `json:"po_id,omitempty"`
	Amount   string `json:"amount"    validate:"required,decimal_gte0"`
	Currency string `json:"currency,omitempty"`
	Notes    string `json:"notes,omitempty"`
}

// ============================================================================
// Catalog Handlers
// ============================================================================

func (er *EinkaufRoutes) HandleListCatalogItems(w http.ResponseWriter, r *http.Request) {
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

	grpcReq := &einkaufv1.ListCatalogItemsRequest{
		TenantId: tenantID.String(),
		Category: q.Get("category"),
		Search:   q.Get("search"),
		Page:     int32(page),
		PageSize: int32(pageSize),
	}
	if supplierID := q.Get("supplier_id"); supplierID != "" {
		grpcReq.SupplierId = &supplierID
	}
	if avail := q.Get("available"); avail == "true" || avail == "1" {
		t := true
		grpcReq.Available = &t
	} else if avail == "false" || avail == "0" {
		f := false
		grpcReq.Available = &f
	}

	resp, err := client.ListCatalogItems(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

func (er *EinkaufRoutes) HandleGetCatalogItem(w http.ResponseWriter, r *http.Request) {
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

	resp, err := client.GetCatalogItem(r.Context(), &einkaufv1.GetCatalogItemRequest{
		TenantId: tenantID.String(),
		ItemId:   id,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

func (er *EinkaufRoutes) HandleCreateCatalogItem(w http.ResponseWriter, r *http.Request) {
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

	req, ok := decodeAndValidate[createCatalogItemRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.CreateCatalogItem(r.Context(), &einkaufv1.CreateCatalogItemRequest{
		TenantId:    tenantID.String(),
		SupplierId:  req.SupplierID,
		Name:        req.Name,
		Sku:         req.SKU,
		Category:    req.Category,
		Price:       req.Price,
		Currency:    req.Currency,
		Unit:        req.Unit,
		Available:   req.Available,
		MinOrderQty: req.MinOrderQty,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusCreated, resp)
}

func (er *EinkaufRoutes) HandleUpdateCatalogItem(w http.ResponseWriter, r *http.Request) {
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

	req, ok := decodeAndValidate[updateCatalogItemRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.UpdateCatalogItem(r.Context(), &einkaufv1.UpdateCatalogItemRequest{
		TenantId:    tenantID.String(),
		ItemId:      id,
		Name:        req.Name,
		Sku:         req.SKU,
		Category:    req.Category,
		Price:       req.Price,
		Currency:    req.Currency,
		Unit:        req.Unit,
		Available:   req.Available,
		MinOrderQty: req.MinOrderQty,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

func (er *EinkaufRoutes) HandleDeleteCatalogItem(w http.ResponseWriter, r *http.Request) {
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

	_, err = client.DeleteCatalogItem(r.Context(), &einkaufv1.DeleteCatalogItemRequest{
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
// Supplier Rating Handlers
// ============================================================================

func (er *EinkaufRoutes) HandleListSupplierRatings(w http.ResponseWriter, r *http.Request) {
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

	supplierID, ok := validateUUIDParam(w, r, "supplierId")
	if !ok {
		return
	}

	resp, err := client.ListSupplierRatings(r.Context(), &einkaufv1.ListSupplierRatingsRequest{
		TenantId:   tenantID.String(),
		SupplierId: supplierID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

func (er *EinkaufRoutes) HandleCreateSupplierRating(w http.ResponseWriter, r *http.Request) {
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

	supplierID, ok := validateUUIDParam(w, r, "supplierId")
	if !ok {
		return
	}

	req, ok := decodeAndValidate[createSupplierRatingRequest](w, r)
	if !ok {
		return
	}

	userID := middleware.GetUserID(r.Context())

	resp, err := client.CreateSupplierRating(r.Context(), &einkaufv1.CreateSupplierRatingRequest{
		TenantId:   tenantID.String(),
		SupplierId: supplierID,
		Category:   req.Category,
		Rating:     req.Rating,
		Comment:    req.Comment,
		RatedBy:    &userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusCreated, resp)
}

func (er *EinkaufRoutes) HandleDeleteSupplierRating(w http.ResponseWriter, r *http.Request) {
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

	ratingID, ok := validateUUIDParam(w, r, "ratingId")
	if !ok {
		return
	}

	_, err = client.DeleteSupplierRating(r.Context(), &einkaufv1.DeleteSupplierRatingRequest{
		TenantId: tenantID.String(),
		RatingId: ratingID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ============================================================================
// Framework Contract Handlers
// ============================================================================

func (er *EinkaufRoutes) HandleListFrameworkContracts(w http.ResponseWriter, r *http.Request) {
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

	grpcReq := &einkaufv1.ListFrameworkContractsRequest{
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

	resp, err := client.ListFrameworkContracts(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

func (er *EinkaufRoutes) HandleGetFrameworkContract(w http.ResponseWriter, r *http.Request) {
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

	resp, err := client.GetFrameworkContract(r.Context(), &einkaufv1.GetFrameworkContractRequest{
		TenantId:   tenantID.String(),
		ContractId: id,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

func (er *EinkaufRoutes) HandleCreateFrameworkContract(w http.ResponseWriter, r *http.Request) {
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

	req, ok := decodeAndValidate[createFrameworkContractRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.CreateFrameworkContract(r.Context(), &einkaufv1.CreateFrameworkContractRequest{
		TenantId:   tenantID.String(),
		SupplierId: req.SupplierID,
		Title:      req.Title,
		ContractNr: req.ContractNr,
		TotalValue: req.TotalValue,
		Currency:   req.Currency,
		Status:     req.Status,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusCreated, resp)
}

func (er *EinkaufRoutes) HandleUpdateFrameworkContract(w http.ResponseWriter, r *http.Request) {
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

	req, ok := decodeAndValidate[updateFrameworkContractRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.UpdateFrameworkContract(r.Context(), &einkaufv1.UpdateFrameworkContractRequest{
		TenantId:   tenantID.String(),
		ContractId: id,
		SupplierId: req.SupplierID,
		Title:      req.Title,
		ContractNr: req.ContractNr,
		TotalValue: req.TotalValue,
		Currency:   req.Currency,
		Status:     req.Status,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

func (er *EinkaufRoutes) HandleDeleteFrameworkContract(w http.ResponseWriter, r *http.Request) {
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

	_, err = client.DeleteFrameworkContract(r.Context(), &einkaufv1.DeleteFrameworkContractRequest{
		TenantId:   tenantID.String(),
		ContractId: id,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ============================================================================
// Contract Item Handlers
// ============================================================================

func (er *EinkaufRoutes) HandleCreateContractItem(w http.ResponseWriter, r *http.Request) {
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

	contractID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	req, ok := decodeAndValidate[createContractItemRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.CreateContractItem(r.Context(), &einkaufv1.CreateContractItemRequest{
		TenantId:   tenantID.String(),
		ContractId: contractID,
		Name:       req.Name,
		UnitPrice:  req.UnitPrice,
		Unit:       req.Unit,
		AgreedQty:  req.AgreedQty,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusCreated, resp)
}

func (er *EinkaufRoutes) HandleUpdateContractItem(w http.ResponseWriter, r *http.Request) {
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

	_, ok := validateUUIDParam(w, r, "id") // contractID — not needed for gRPC call but validates URL
	if !ok {
		return
	}

	itemID, ok := validateUUIDParam(w, r, "itemId")
	if !ok {
		return
	}

	req, ok := decodeAndValidate[updateContractItemRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.UpdateContractItem(r.Context(), &einkaufv1.UpdateContractItemRequest{
		TenantId:  tenantID.String(),
		ItemId:    itemID,
		Name:      req.Name,
		UnitPrice: req.UnitPrice,
		Unit:      req.Unit,
		AgreedQty: req.AgreedQty,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

func (er *EinkaufRoutes) HandleDeleteContractItem(w http.ResponseWriter, r *http.Request) {
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

	itemID, ok := validateUUIDParam(w, r, "itemId")
	if !ok {
		return
	}

	_, err = client.DeleteContractItem(r.Context(), &einkaufv1.DeleteContractItemRequest{
		TenantId: tenantID.String(),
		ItemId:   itemID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ============================================================================
// Contract Call Handlers
// ============================================================================

func (er *EinkaufRoutes) HandleListContractCalls(w http.ResponseWriter, r *http.Request) {
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

	contractID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.ListContractCalls(r.Context(), &einkaufv1.ListContractCallsRequest{
		TenantId:   tenantID.String(),
		ContractId: contractID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

func (er *EinkaufRoutes) HandleCreateContractCall(w http.ResponseWriter, r *http.Request) {
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

	contractID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	req, ok := decodeAndValidate[createContractCallRequest](w, r)
	if !ok {
		return
	}

	grpcReq := &einkaufv1.CreateContractCallRequest{
		TenantId:   tenantID.String(),
		ContractId: contractID,
		Amount:     req.Amount,
		Currency:   req.Currency,
		Notes:      req.Notes,
	}
	if req.POID != "" {
		grpcReq.PoId = &req.POID
	}

	resp, err := client.CreateContractCall(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusCreated, resp)
}
