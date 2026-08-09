package gateway

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/kmuhub/kmuhub/internal/featureflag"
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server/response"
	produktionv1 "github.com/kmuhub/kmuhub/proto/produktion/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ProduktionRoutes handles HTTP routes for the Produktion module.
type ProduktionRoutes struct {
	registry *ServiceRegistry
	flags    *featureflag.Registry
}

// NewProduktionRoutes creates a new ProduktionRoutes with the given service registry and feature flags.
func NewProduktionRoutes(registry *ServiceRegistry, flags *featureflag.Registry) *ProduktionRoutes {
	return &ProduktionRoutes{registry: registry, flags: flags}
}

// ServiceName returns the service name for the route registrar.
func (pr *ProduktionRoutes) ServiceName() string { return "produktion" }

// getClient lazily obtains a gRPC client for the produktion service.
func (pr *ProduktionRoutes) getClient() (produktionv1.ProduktionServiceClient, error) {
	conn, err := pr.registry.GetConnection("produktion")
	if err != nil {
		return nil, err
	}
	return produktionv1.NewProduktionServiceClient(conn), nil
}

// RegisterRoutes mounts all Produktion HTTP routes behind the feature flag modules.produktion.
// Routes are only registered if the flag is enabled.
func (pr *ProduktionRoutes) RegisterRoutes(r chi.Router, authMiddleware func(http.Handler) http.Handler) {
	if !pr.flags.IsEnabled("modules.produktion") {
		return
	}

	r.Route("/api/v1/produktion", func(r chi.Router) {
		r.Use(authMiddleware)
		r.Use(RequireAuthenticated)

		// Additive RBAC guards (RequirePermissionAny keeps the legacy "write"
		// token valid while granting the capability-catalog.ts fine keys that
		// ProduktionPage.tsx/ProduktionDetailModals.tsx actually gate on). Reads
		// already match the fine key 1:1 and stay plain RequirePermission.
		orderCreate := middleware.RequirePermissionAny([2]string{"produktion:order", "write"}, [2]string{"produktion:order", "create"})
		orderEdit := middleware.RequirePermissionAny([2]string{"produktion:order", "write"}, [2]string{"produktion:order", "edit"})
		orderDelete := middleware.RequirePermissionAny([2]string{"produktion:order", "write"}, [2]string{"produktion:order", "delete"})
		orderStart := middleware.RequirePermissionAny([2]string{"produktion:order", "write"}, [2]string{"produktion:order", "start"})
		orderComplete := middleware.RequirePermissionAny([2]string{"produktion:order", "write"}, [2]string{"produktion:order", "complete"})
		orderCancel := middleware.RequirePermissionAny([2]string{"produktion:order", "write"}, [2]string{"produktion:order", "cancel"})

		// Production Orders
		r.Route("/orders", func(r chi.Router) {
			r.With(middleware.RequirePermission("produktion:order", "read")).Get("/", pr.HandleListOrders)
			r.With(orderCreate).Post("/", pr.HandleCreateOrder)

			r.Route("/{id}", func(r chi.Router) {
				r.With(middleware.RequirePermission("produktion:order", "read")).Get("/", pr.HandleGetOrder)
				r.With(orderEdit).Patch("/", pr.HandleUpdateOrder)
				r.With(orderDelete).Delete("/", pr.HandleDeleteOrder)

				r.With(orderStart).Post("/start", pr.HandleStartOrder)
				r.With(orderComplete).Post("/complete", pr.HandleCompleteOrder)
				r.With(orderCancel).Post("/cancel", pr.HandleCancelOrder)

				r.With(middleware.RequirePermission("produktion:order", "read")).Get("/material-availability", pr.HandleGetMaterialAvailability)
			})

			// Work Steps (per order) — see route_produktion_ext.go
			pr.registerWorkStepRoutes(r)
		})

		// Machine Bookings
		r.Route("/bookings", func(r chi.Router) {
			r.With(middleware.RequirePermission("produktion:booking", "read")).Get("/", pr.HandleListMachineBookings)
			r.With(middleware.RequirePermission("produktion:booking", "write")).Post("/", pr.HandleCreateMachineBooking)

			r.Route("/{id}", func(r chi.Router) {
				r.With(middleware.RequirePermission("produktion:booking", "write")).Patch("/", pr.HandleUpdateMachineBooking)
				r.With(middleware.RequirePermission("produktion:booking", "write")).Delete("/", pr.HandleDeleteMachineBooking)
			})
		})

		// Production Plans
		r.Route("/plans", func(r chi.Router) {
			r.With(middleware.RequirePermission("produktion:plan", "write")).Post("/", pr.HandleCreatePlan)

			r.Route("/{id}", func(r chi.Router) {
				r.With(middleware.RequirePermission("produktion:plan", "read")).Get("/", pr.HandleGetPlan)
				r.With(middleware.RequirePermission("produktion:plan", "write")).Patch("/", pr.HandleUpdatePlan)

				r.With(middleware.RequirePermission("produktion:plan", "read")).Get("/capacity", pr.HandleGetCapacityOverview)
			})
		})

		// BOMs, machines and quality checks — see route_produktion_ext.go
		pr.registerExtRoutes(r)
	})
}

// ============================================================================
// Request types
// ============================================================================

type createOrderRequest struct {
	OrderNumber  string  `json:"order_number"   validate:"required"`
	ProductName  string  `json:"product_name"   validate:"required"`
	Quantity     int     `json:"quantity"       validate:"gt=0"`
	PlannedStart string  `json:"planned_start"  validate:"required"` // RFC3339
	PlannedEnd   string  `json:"planned_end"    validate:"required"` // RFC3339
	Priority     int     `json:"priority"`
	Notes        string  `json:"notes,omitempty"`
	BomID        *string `json:"bom_id,omitempty" validate:"omitempty,uuid"`
}

type updateOrderRequest struct {
	ProductName  *string `json:"product_name,omitempty"`
	Quantity     *int    `json:"quantity,omitempty"`
	PlannedStart *string `json:"planned_start,omitempty"`
	PlannedEnd   *string `json:"planned_end,omitempty"`
	Priority     *int    `json:"priority,omitempty"`
	Notes        *string `json:"notes,omitempty"`
	BomID        *string `json:"bom_id,omitempty" validate:"omitempty,uuid"`
}

type createBookingRequest struct {
	MachineID         string `json:"machine_id"          validate:"required,uuid"`
	ProductionOrderID string `json:"production_order_id" validate:"required,uuid"`
	StartsAt          string `json:"starts_at"           validate:"required"` // RFC3339
	EndsAt            string `json:"ends_at"             validate:"required"` // RFC3339
	Notes             string `json:"notes,omitempty"`
}

type updateBookingRequest struct {
	MachineID         *string `json:"machine_id,omitempty"`
	ProductionOrderID *string `json:"production_order_id,omitempty"`
	StartsAt          *string `json:"starts_at,omitempty"`
	EndsAt            *string `json:"ends_at,omitempty"`
	Notes             *string `json:"notes,omitempty"`
}

type createPlanRequest struct {
	Name                 string  `json:"name"                   validate:"required"`
	WeekNumber           int     `json:"week_number"`
	Year                 int     `json:"year"`
	TotalCapacityHours   float64 `json:"total_capacity_hours"   validate:"gt=0"`
	PlannedCapacityHours float64 `json:"planned_capacity_hours,omitempty"`
	Notes                string  `json:"notes,omitempty"`
}

type updatePlanRequest struct {
	Name                 *string  `json:"name,omitempty"`
	WeekNumber           *int     `json:"week_number,omitempty"`
	Year                 *int     `json:"year,omitempty"`
	TotalCapacityHours   *float64 `json:"total_capacity_hours,omitempty"`
	PlannedCapacityHours *float64 `json:"planned_capacity_hours,omitempty"`
	Status               *string  `json:"status,omitempty"`
	Notes                *string  `json:"notes,omitempty"`
}

// ============================================================================
// Order Handlers
// ============================================================================

func (pr *ProduktionRoutes) HandleListOrders(w http.ResponseWriter, r *http.Request) {
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
	q := r.URL.Query()

	grpcReq := &produktionv1.ListOrdersRequest{
		TenantId: tenantID.String(),
		Page:     int32(page),
		PageSize: int32(pageSize),
	}
	if s := q.Get("status"); s != "" {
		grpcReq.Status = &s
	}
	if p := q.Get("priority"); p != "" {
		if pv, parseErr := parseIntParam(p); parseErr == nil {
			prio := int32(pv)
			grpcReq.Priority = &prio
		}
	}
	if from := q.Get("date_from"); from != "" {
		if t, parseErr := time.Parse(time.RFC3339, from); parseErr == nil {
			grpcReq.DateFrom = timestamppb.New(t)
		}
	}
	if to := q.Get("date_to"); to != "" {
		if t, parseErr := time.Parse(time.RFC3339, to); parseErr == nil {
			grpcReq.DateTo = timestamppb.New(t)
		}
	}

	resp, err := client.ListOrders(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

func (pr *ProduktionRoutes) HandleCreateOrder(w http.ResponseWriter, r *http.Request) {
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

	req, ok := decodeAndValidate[createOrderRequest](w, r)
	if !ok {
		return
	}

	plannedStart, err := time.Parse(time.RFC3339, req.PlannedStart)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid planned_start (use RFC3339)")
		return
	}
	plannedEnd, err := time.Parse(time.RFC3339, req.PlannedEnd)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid planned_end (use RFC3339)")
		return
	}

	grpcReq := &produktionv1.CreateOrderRequest{
		TenantId:     tenantID.String(),
		OrderNumber:  req.OrderNumber,
		ProductName:  req.ProductName,
		Quantity:     int32(req.Quantity),
		PlannedStart: timestamppb.New(plannedStart),
		PlannedEnd:   timestamppb.New(plannedEnd),
		Priority:     int32(req.Priority),
		Notes:        req.Notes,
		CreatedBy:    &userID,
		BomId:        req.BomID,
	}

	resp, err := client.CreateOrder(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusCreated, resp)
}

func (pr *ProduktionRoutes) HandleGetOrder(w http.ResponseWriter, r *http.Request) {
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

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.GetOrder(r.Context(), &produktionv1.GetOrderRequest{
		TenantId: tenantID.String(),
		OrderId:  id,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

func (pr *ProduktionRoutes) HandleUpdateOrder(w http.ResponseWriter, r *http.Request) {
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

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	req, ok := decodeAndValidate[updateOrderRequest](w, r)
	if !ok {
		return
	}

	grpcReq := &produktionv1.UpdateOrderRequest{
		TenantId: tenantID.String(),
		OrderId:  id,
		Notes:    req.Notes,
	}
	if req.ProductName != nil {
		grpcReq.ProductName = req.ProductName
	}
	if req.Quantity != nil {
		q := int32(*req.Quantity)
		grpcReq.Quantity = &q
	}
	if req.Priority != nil {
		p := int32(*req.Priority)
		grpcReq.Priority = &p
	}
	if req.PlannedStart != nil {
		if t, parseErr := time.Parse(time.RFC3339, *req.PlannedStart); parseErr == nil {
			grpcReq.PlannedStart = timestamppb.New(t)
		}
	}
	if req.PlannedEnd != nil {
		if t, parseErr := time.Parse(time.RFC3339, *req.PlannedEnd); parseErr == nil {
			grpcReq.PlannedEnd = timestamppb.New(t)
		}
	}
	if req.BomID != nil {
		grpcReq.BomId = req.BomID
	}

	resp, err := client.UpdateOrder(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

func (pr *ProduktionRoutes) HandleDeleteOrder(w http.ResponseWriter, r *http.Request) {
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

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	_, err = client.DeleteOrder(r.Context(), &produktionv1.DeleteOrderRequest{
		TenantId: tenantID.String(),
		OrderId:  id,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (pr *ProduktionRoutes) HandleStartOrder(w http.ResponseWriter, r *http.Request) {
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
	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}
	resp, err := client.StartOrder(r.Context(), &produktionv1.OrderActionRequest{
		TenantId: tenantID.String(),
		OrderId:  id,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

func (pr *ProduktionRoutes) HandleCompleteOrder(w http.ResponseWriter, r *http.Request) {
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
	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}
	resp, err := client.CompleteOrder(r.Context(), &produktionv1.OrderActionRequest{
		TenantId: tenantID.String(),
		OrderId:  id,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

func (pr *ProduktionRoutes) HandleCancelOrder(w http.ResponseWriter, r *http.Request) {
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
	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}
	resp, err := client.CancelOrder(r.Context(), &produktionv1.OrderActionRequest{
		TenantId: tenantID.String(),
		OrderId:  id,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

func (pr *ProduktionRoutes) HandleGetMaterialAvailability(w http.ResponseWriter, r *http.Request) {
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
	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}
	resp, err := client.GetMaterialAvailability(r.Context(), &produktionv1.GetMaterialAvailabilityRequest{
		TenantId: tenantID.String(),
		OrderId:  id,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

// ============================================================================
// Booking Handlers
// ============================================================================

func (pr *ProduktionRoutes) HandleListMachineBookings(w http.ResponseWriter, r *http.Request) {
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
	q := r.URL.Query()

	grpcReq := &produktionv1.ListMachineBookingsRequest{
		TenantId: tenantID.String(),
		Page:     int32(page),
		PageSize: int32(pageSize),
	}
	if mid := q.Get("machine_id"); mid != "" {
		grpcReq.MachineId = &mid
	}
	if oid := q.Get("production_order_id"); oid != "" {
		grpcReq.ProductionOrderId = &oid
	}
	if from := q.Get("date_from"); from != "" {
		if t, parseErr := time.Parse(time.RFC3339, from); parseErr == nil {
			grpcReq.DateFrom = timestamppb.New(t)
		}
	}
	if to := q.Get("date_to"); to != "" {
		if t, parseErr := time.Parse(time.RFC3339, to); parseErr == nil {
			grpcReq.DateTo = timestamppb.New(t)
		}
	}

	resp, err := client.ListMachineBookings(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

func (pr *ProduktionRoutes) HandleCreateMachineBooking(w http.ResponseWriter, r *http.Request) {
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

	req, ok := decodeAndValidate[createBookingRequest](w, r)
	if !ok {
		return
	}

	startsAt, err := time.Parse(time.RFC3339, req.StartsAt)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid starts_at (use RFC3339)")
		return
	}
	endsAt, err := time.Parse(time.RFC3339, req.EndsAt)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid ends_at (use RFC3339)")
		return
	}

	grpcReq := &produktionv1.CreateMachineBookingRequest{
		TenantId:          tenantID.String(),
		MachineId:         req.MachineID,
		ProductionOrderId: req.ProductionOrderID,
		StartsAt:          timestamppb.New(startsAt),
		EndsAt:            timestamppb.New(endsAt),
		Notes:             req.Notes,
		CreatedBy:         &userID,
	}

	resp, err := client.CreateMachineBooking(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusCreated, resp)
}

func (pr *ProduktionRoutes) HandleUpdateMachineBooking(w http.ResponseWriter, r *http.Request) {
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

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	req, ok := decodeAndValidate[updateBookingRequest](w, r)
	if !ok {
		return
	}

	grpcReq := &produktionv1.UpdateMachineBookingRequest{
		TenantId:  tenantID.String(),
		BookingId: id,
		MachineId: req.MachineID,
		Notes:     req.Notes,
	}
	if req.ProductionOrderID != nil {
		grpcReq.ProductionOrderId = req.ProductionOrderID
	}
	if req.StartsAt != nil {
		if t, parseErr := time.Parse(time.RFC3339, *req.StartsAt); parseErr == nil {
			grpcReq.StartsAt = timestamppb.New(t)
		}
	}
	if req.EndsAt != nil {
		if t, parseErr := time.Parse(time.RFC3339, *req.EndsAt); parseErr == nil {
			grpcReq.EndsAt = timestamppb.New(t)
		}
	}

	resp, err := client.UpdateMachineBooking(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

func (pr *ProduktionRoutes) HandleDeleteMachineBooking(w http.ResponseWriter, r *http.Request) {
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

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	_, err = client.DeleteMachineBooking(r.Context(), &produktionv1.DeleteMachineBookingRequest{
		TenantId:  tenantID.String(),
		BookingId: id,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ============================================================================
// Plan Handlers
// ============================================================================

func (pr *ProduktionRoutes) HandleCreatePlan(w http.ResponseWriter, r *http.Request) {
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

	req, ok := decodeAndValidate[createPlanRequest](w, r)
	if !ok {
		return
	}

	grpcReq := &produktionv1.CreatePlanRequest{
		TenantId:             tenantID.String(),
		Name:                 req.Name,
		WeekNumber:           int32(req.WeekNumber),
		Year:                 int32(req.Year),
		TotalCapacityHours:   req.TotalCapacityHours,
		PlannedCapacityHours: req.PlannedCapacityHours,
		Notes:                req.Notes,
		CreatedBy:            &userID,
	}

	resp, err := client.CreatePlan(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusCreated, resp)
}

func (pr *ProduktionRoutes) HandleGetPlan(w http.ResponseWriter, r *http.Request) {
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

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.GetPlan(r.Context(), &produktionv1.GetPlanRequest{
		TenantId: tenantID.String(),
		PlanId:   id,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

func (pr *ProduktionRoutes) HandleUpdatePlan(w http.ResponseWriter, r *http.Request) {
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

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	req, ok := decodeAndValidate[updatePlanRequest](w, r)
	if !ok {
		return
	}

	grpcReq := &produktionv1.UpdatePlanRequest{
		TenantId: tenantID.String(),
		PlanId:   id,
		Name:     req.Name,
		Status:   req.Status,
		Notes:    req.Notes,
	}
	if req.WeekNumber != nil {
		wn := int32(*req.WeekNumber)
		grpcReq.WeekNumber = &wn
	}
	if req.Year != nil {
		y := int32(*req.Year)
		grpcReq.Year = &y
	}
	if req.TotalCapacityHours != nil {
		grpcReq.TotalCapacityHours = req.TotalCapacityHours
	}
	if req.PlannedCapacityHours != nil {
		grpcReq.PlannedCapacityHours = req.PlannedCapacityHours
	}

	resp, err := client.UpdatePlan(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}

func (pr *ProduktionRoutes) HandleGetCapacityOverview(w http.ResponseWriter, r *http.Request) {
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

	planID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	machineID := r.URL.Query().Get("machine_id")
	if machineID == "" {
		response.Error(w, http.StatusBadRequest, "machine_id query param is required")
		return
	}

	resp, err := client.GetCapacityOverview(r.Context(), &produktionv1.GetCapacityOverviewRequest{
		TenantId:  tenantID.String(),
		MachineId: machineID,
		PlanId:    planID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.Proto(w, http.StatusOK, resp)
}
