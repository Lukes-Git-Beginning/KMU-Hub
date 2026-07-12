package gateway

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server/response"
	produktionv1 "github.com/kmuhub/kmuhub/proto/produktion/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// RegisterExtRoutes mounts the BOM, WorkStep, Machine, and QualityCheck routes
// onto an existing chi.Router. Called from RegisterRoutes after the main produktion routes.
func (pr *ProduktionRoutes) RegisterExtRoutes(r chi.Router) {
	r.Route("/api/v1/produktion", func(r chi.Router) {
		// ----------------------------------------------------------------
		// BOMs
		// ----------------------------------------------------------------
		r.Route("/boms", func(r chi.Router) {
			r.With(middleware.RequirePermission("produktion:bom", "read")).Get("/", pr.HandleListBOMs)
			r.With(middleware.RequirePermission("produktion:bom", "write")).Post("/", pr.HandleCreateBOM)
			r.Route("/{bomId}", func(r chi.Router) {
				r.With(middleware.RequirePermission("produktion:bom", "read")).Get("/", pr.HandleGetBOM)
				r.With(middleware.RequirePermission("produktion:bom", "write")).Patch("/", pr.HandleUpdateBOM)
				r.With(middleware.RequirePermission("produktion:bom", "write")).Delete("/", pr.HandleDeleteBOM)
			})
		})

		// ----------------------------------------------------------------
		// Work Steps (per order)
		// ----------------------------------------------------------------
		r.Route("/orders/{orderId}/steps", func(r chi.Router) {
			r.With(middleware.RequirePermission("produktion:workstep", "read")).Get("/", pr.HandleListWorkSteps)
			r.With(middleware.RequirePermission("produktion:workstep", "write")).Post("/", pr.HandleCreateWorkStep)
			r.Route("/{stepId}", func(r chi.Router) {
				r.With(middleware.RequirePermission("produktion:workstep", "write")).Patch("/", pr.HandleUpdateWorkStep)
				r.With(middleware.RequirePermission("produktion:workstep", "write")).Delete("/", pr.HandleDeleteWorkStep)
			})
		})

		// ----------------------------------------------------------------
		// Machines
		// ----------------------------------------------------------------
		r.Route("/machines", func(r chi.Router) {
			r.With(middleware.RequirePermission("produktion:machine", "read")).Get("/", pr.HandleListMachines)
			r.With(middleware.RequirePermission("produktion:machine", "write")).Post("/", pr.HandleCreateMachine)
			r.Route("/{machineId}", func(r chi.Router) {
				r.With(middleware.RequirePermission("produktion:machine", "read")).Get("/", pr.HandleGetMachine)
				r.With(middleware.RequirePermission("produktion:machine", "write")).Patch("/", pr.HandleUpdateMachine)
				r.With(middleware.RequirePermission("produktion:machine", "write")).Delete("/", pr.HandleDeleteMachine)
			})
		})

		// ----------------------------------------------------------------
		// Quality Checks
		// ----------------------------------------------------------------
		r.Route("/quality", func(r chi.Router) {
			r.With(middleware.RequirePermission("produktion:quality", "read")).Get("/", pr.HandleListQualityChecks)
			r.With(middleware.RequirePermission("produktion:quality", "write")).Post("/", pr.HandleCreateQualityCheck)
			r.Route("/{checkId}", func(r chi.Router) {
				r.With(middleware.RequirePermission("produktion:quality", "read")).Get("/", pr.HandleGetQualityCheck)
			})
		})
	})
}

// ============================================================================
// BOM Handlers
// ============================================================================

func (pr *ProduktionRoutes) HandleListBOMs(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := pr.getClient()
	if err != nil {
		respondServiceUnavailable(w, pr.ServiceName())
		return
	}

	page, pageSize := parsePagination(r, 1, 20)

	resp, err := client.ListBOMs(r.Context(), &produktionv1.ListBOMsRequest{
		TenantId: tenantID.String(),
		Page:     int32(page),
		PageSize: int32(pageSize),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

func (pr *ProduktionRoutes) HandleCreateBOM(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := pr.getClient()
	if err != nil {
		respondServiceUnavailable(w, pr.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	type bomItemBody struct {
		MaterialName string  `json:"material_name"`
		Quantity     float64 `json:"quantity"`
		Unit         string  `json:"unit"`
		SortOrder    int32   `json:"sort_order"`
	}
	type body struct {
		ProductName string        `json:"product_name" validate:"required"`
		SKU         string        `json:"sku"          validate:"required"`
		Version     string        `json:"version"`
		Active      bool          `json:"active"`
		Notes       string        `json:"notes"`
		Items       []bomItemBody `json:"items"`
	}

	req, ok := decodeAndValidate[body](w, r)
	if !ok {
		return
	}

	grpcReq := &produktionv1.CreateBOMRequest{
		TenantId:    tenantID.String(),
		ProductName: req.ProductName,
		Sku:         req.SKU,
		Version:     req.Version,
		Active:      req.Active,
		Notes:       req.Notes,
		CreatedBy:   &userID,
	}
	for _, item := range req.Items {
		grpcReq.Items = append(grpcReq.Items, &produktionv1.CreateBomItemInput{
			MaterialName: item.MaterialName,
			Quantity:     item.Quantity,
			Unit:         item.Unit,
			SortOrder:    item.SortOrder,
		})
	}

	resp, err := client.CreateBOM(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusCreated, resp)
}

func (pr *ProduktionRoutes) HandleGetBOM(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := pr.getClient()
	if err != nil {
		respondServiceUnavailable(w, pr.ServiceName())
		return
	}

	bomID, ok := validateUUIDParam(w, r, "bomId")
	if !ok {
		return
	}

	resp, err := client.GetBOM(r.Context(), &produktionv1.GetBOMRequest{
		TenantId: tenantID.String(),
		BomId:    bomID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

func (pr *ProduktionRoutes) HandleUpdateBOM(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := pr.getClient()
	if err != nil {
		respondServiceUnavailable(w, pr.ServiceName())
		return
	}

	bomID, ok := validateUUIDParam(w, r, "bomId")
	if !ok {
		return
	}

	type body struct {
		ProductName *string `json:"product_name"`
		SKU         *string `json:"sku"`
		Version     *string `json:"version"`
		Active      *bool   `json:"active"`
		Notes       *string `json:"notes"`
	}
	req, ok := decodeAndValidate[body](w, r)
	if !ok {
		return
	}

	resp, err := client.UpdateBOM(r.Context(), &produktionv1.UpdateBOMRequest{
		TenantId:    tenantID.String(),
		BomId:       bomID,
		ProductName: req.ProductName,
		Sku:         req.SKU,
		Version:     req.Version,
		Active:      req.Active,
		Notes:       req.Notes,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

func (pr *ProduktionRoutes) HandleDeleteBOM(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := pr.getClient()
	if err != nil {
		respondServiceUnavailable(w, pr.ServiceName())
		return
	}

	bomID, ok := validateUUIDParam(w, r, "bomId")
	if !ok {
		return
	}

	_, err = client.DeleteBOM(r.Context(), &produktionv1.DeleteBOMRequest{
		TenantId: tenantID.String(),
		BomId:    bomID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ============================================================================
// Work Step Handlers
// ============================================================================

func (pr *ProduktionRoutes) HandleListWorkSteps(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := pr.getClient()
	if err != nil {
		respondServiceUnavailable(w, pr.ServiceName())
		return
	}

	orderID, ok := validateUUIDParam(w, r, "orderId")
	if !ok {
		return
	}

	resp, err := client.ListWorkSteps(r.Context(), &produktionv1.ListWorkStepsRequest{
		TenantId: tenantID.String(),
		OrderId:  orderID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

func (pr *ProduktionRoutes) HandleCreateWorkStep(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := pr.getClient()
	if err != nil {
		respondServiceUnavailable(w, pr.ServiceName())
		return
	}

	orderID, ok := validateUUIDParam(w, r, "orderId")
	if !ok {
		return
	}

	type body struct {
		StepNr          int    `json:"step_nr"`
		Name            string `json:"name"   validate:"required"`
		Description     string `json:"description"`
		DurationMinutes int    `json:"duration_minutes"`
		Assignee        string `json:"assignee"`
	}
	req, ok := decodeAndValidate[body](w, r)
	if !ok {
		return
	}

	resp, err := client.CreateWorkStep(r.Context(), &produktionv1.CreateWorkStepRequest{
		TenantId:        tenantID.String(),
		OrderId:         orderID,
		StepNr:          int32(req.StepNr),
		Name:            req.Name,
		Description:     req.Description,
		DurationMinutes: int32(req.DurationMinutes),
		Assignee:        req.Assignee,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusCreated, resp)
}

func (pr *ProduktionRoutes) HandleUpdateWorkStep(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := pr.getClient()
	if err != nil {
		respondServiceUnavailable(w, pr.ServiceName())
		return
	}

	stepID, ok := validateUUIDParam(w, r, "stepId")
	if !ok {
		return
	}

	type body struct {
		Name            *string `json:"name"`
		Description     *string `json:"description"`
		DurationMinutes *int32  `json:"duration_minutes"`
		Status          *string `json:"status"`
		Assignee        *string `json:"assignee"`
	}
	req, ok := decodeAndValidate[body](w, r)
	if !ok {
		return
	}

	resp, err := client.UpdateWorkStep(r.Context(), &produktionv1.UpdateWorkStepRequest{
		TenantId:        tenantID.String(),
		StepId:          stepID,
		Name:            req.Name,
		Description:     req.Description,
		DurationMinutes: req.DurationMinutes,
		Status:          req.Status,
		Assignee:        req.Assignee,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

func (pr *ProduktionRoutes) HandleDeleteWorkStep(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := pr.getClient()
	if err != nil {
		respondServiceUnavailable(w, pr.ServiceName())
		return
	}

	stepID, ok := validateUUIDParam(w, r, "stepId")
	if !ok {
		return
	}

	_, err = client.DeleteWorkStep(r.Context(), &produktionv1.DeleteWorkStepRequest{
		TenantId: tenantID.String(),
		StepId:   stepID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ============================================================================
// Machine Handlers
// ============================================================================

func (pr *ProduktionRoutes) HandleListMachines(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := pr.getClient()
	if err != nil {
		respondServiceUnavailable(w, pr.ServiceName())
		return
	}

	page, pageSize := parsePagination(r, 1, 20)
	statusFilter := r.URL.Query().Get("status")

	grpcReq := &produktionv1.ListMachinesRequest{
		TenantId: tenantID.String(),
		Page:     int32(page),
		PageSize: int32(pageSize),
	}
	if statusFilter != "" {
		grpcReq.Status = &statusFilter
	}

	resp, err := client.ListMachines(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

func (pr *ProduktionRoutes) HandleCreateMachine(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := pr.getClient()
	if err != nil {
		respondServiceUnavailable(w, pr.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	type body struct {
		Name  string `json:"name"  validate:"required"`
		Type  string `json:"type"`
		Notes string `json:"notes"`
	}
	req, ok := decodeAndValidate[body](w, r)
	if !ok {
		return
	}

	resp, err := client.CreateMachine(r.Context(), &produktionv1.CreateMachineRequest{
		TenantId:  tenantID.String(),
		Name:      req.Name,
		Type:      req.Type,
		Notes:     req.Notes,
		CreatedBy: &userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusCreated, resp)
}

func (pr *ProduktionRoutes) HandleGetMachine(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := pr.getClient()
	if err != nil {
		respondServiceUnavailable(w, pr.ServiceName())
		return
	}

	machineID, ok := validateUUIDParam(w, r, "machineId")
	if !ok {
		return
	}

	resp, err := client.GetMachine(r.Context(), &produktionv1.GetMachineRequest{
		TenantId:  tenantID.String(),
		MachineId: machineID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

func (pr *ProduktionRoutes) HandleUpdateMachine(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := pr.getClient()
	if err != nil {
		respondServiceUnavailable(w, pr.ServiceName())
		return
	}

	machineID, ok := validateUUIDParam(w, r, "machineId")
	if !ok {
		return
	}

	type body struct {
		Name   *string `json:"name"`
		Type   *string `json:"type"`
		Status *string `json:"status"`
		Notes  *string `json:"notes"`
	}
	req, ok := decodeAndValidate[body](w, r)
	if !ok {
		return
	}

	resp, err := client.UpdateMachine(r.Context(), &produktionv1.UpdateMachineRequest{
		TenantId:  tenantID.String(),
		MachineId: machineID,
		Name:      req.Name,
		Type:      req.Type,
		Status:    req.Status,
		Notes:     req.Notes,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

func (pr *ProduktionRoutes) HandleDeleteMachine(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := pr.getClient()
	if err != nil {
		respondServiceUnavailable(w, pr.ServiceName())
		return
	}

	machineID, ok := validateUUIDParam(w, r, "machineId")
	if !ok {
		return
	}

	_, err = client.DeleteMachine(r.Context(), &produktionv1.DeleteMachineRequest{
		TenantId:  tenantID.String(),
		MachineId: machineID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ============================================================================
// Quality Check Handlers
// ============================================================================

func (pr *ProduktionRoutes) HandleListQualityChecks(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := pr.getClient()
	if err != nil {
		respondServiceUnavailable(w, pr.ServiceName())
		return
	}

	page, pageSize := parsePagination(r, 1, 20)
	orderIDStr := r.URL.Query().Get("order_id")

	grpcReq := &produktionv1.ListQualityChecksRequest{
		TenantId: tenantID.String(),
		Page:     int32(page),
		PageSize: int32(pageSize),
	}
	if orderIDStr != "" {
		grpcReq.OrderId = &orderIDStr
	}

	resp, err := client.ListQualityChecks(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

func (pr *ProduktionRoutes) HandleCreateQualityCheck(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := pr.getClient()
	if err != nil {
		respondServiceUnavailable(w, pr.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	type body struct {
		OrderID      string `json:"order_id"   validate:"required,uuid"`
		Inspector    string `json:"inspector"  validate:"required"`
		CheckedAt    string `json:"checked_at"` // RFC3339; optional, defaults to now
		Passed       bool   `json:"passed"`
		DefectsFound int    `json:"defects_found"`
		Notes        string `json:"notes"`
	}
	req, ok := decodeAndValidate[body](w, r)
	if !ok {
		return
	}

	checkedAt := time.Now()
	if req.CheckedAt != "" {
		if t, parseErr := time.Parse(time.RFC3339, req.CheckedAt); parseErr == nil {
			checkedAt = t
		}
	}

	resp, err := client.CreateQualityCheck(r.Context(), &produktionv1.CreateQualityCheckRequest{
		TenantId:     tenantID.String(),
		OrderId:      req.OrderID,
		Inspector:    req.Inspector,
		CheckedAt:    timestamppb.New(checkedAt),
		Passed:       req.Passed,
		DefectsFound: int32(req.DefectsFound),
		Notes:        req.Notes,
		CreatedBy:    &userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusCreated, resp)
}

func (pr *ProduktionRoutes) HandleGetQualityCheck(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := pr.getClient()
	if err != nil {
		respondServiceUnavailable(w, pr.ServiceName())
		return
	}

	checkID, ok := validateUUIDParam(w, r, "checkId")
	if !ok {
		return
	}

	resp, err := client.GetQualityCheck(r.Context(), &produktionv1.GetQualityCheckRequest{
		TenantId: tenantID.String(),
		CheckId:  checkID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}
