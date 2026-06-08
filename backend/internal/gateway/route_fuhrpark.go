package gateway

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/kmuhub/kmuhub/internal/featureflag"
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server/response"
	fuhrparkv1 "github.com/kmuhub/kmuhub/proto/fuhrpark/v1"
)

// FuhrparkRoutes handles HTTP routes for the Fuhrpark (fleet management) module.
type FuhrparkRoutes struct {
	registry *ServiceRegistry
	flags    *featureflag.Registry
}

// NewFuhrparkRoutes creates a new FuhrparkRoutes instance.
func NewFuhrparkRoutes(registry *ServiceRegistry, flags *featureflag.Registry) *FuhrparkRoutes {
	return &FuhrparkRoutes{registry: registry, flags: flags}
}

// ServiceName returns the service name for the route registrar.
func (fr *FuhrparkRoutes) ServiceName() string { return "fuhrpark" }

func (fr *FuhrparkRoutes) getClient() (fuhrparkv1.FuhrparkServiceClient, error) {
	conn, err := fr.registry.GetConnection("fuhrpark")
	if err != nil {
		return nil, err
	}
	return fuhrparkv1.NewFuhrparkServiceClient(conn), nil
}

// RegisterRoutes mounts all Fuhrpark HTTP routes behind the feature flag modules.fuhrpark.
func (fr *FuhrparkRoutes) RegisterRoutes(r chi.Router, authMiddleware func(http.Handler) http.Handler) {
	if !fr.flags.IsEnabled("modules.fuhrpark") {
		return
	}

	r.Route("/api/v1/fuhrpark", func(r chi.Router) {
		r.Use(authMiddleware)
		r.Use(RequireAuthenticated)

		// Vehicles
		r.Route("/vehicles", func(r chi.Router) {
			r.With(middleware.RequirePermission("fuhrpark:vehicle", "read")).Get("/", fr.HandleListVehicles)
			r.With(middleware.RequirePermission("fuhrpark:vehicle", "write")).Post("/", fr.HandleCreateVehicle)

			r.Route("/{id}", func(r chi.Router) {
				r.With(middleware.RequirePermission("fuhrpark:vehicle", "read")).Get("/", fr.HandleGetVehicle)
				r.With(middleware.RequirePermission("fuhrpark:vehicle", "write")).Patch("/", fr.HandleUpdateVehicle)
				r.With(middleware.RequirePermission("fuhrpark:vehicle", "write")).Delete("/", fr.HandleDeleteVehicle)

				r.With(middleware.RequirePermission("fuhrpark:vehicle", "read")).Get("/history", fr.HandleGetVehicleHistory)

				// Services sub-resource
				r.With(middleware.RequirePermission("fuhrpark:service", "read")).Get("/services", fr.HandleListVehicleServices)
				r.With(middleware.RequirePermission("fuhrpark:service", "write")).Post("/services", fr.HandleScheduleService)

				// Damages sub-resource
				r.With(middleware.RequirePermission("fuhrpark:damage", "read")).Get("/damages", fr.HandleListVehicleDamages)
				r.With(middleware.RequirePermission("fuhrpark:damage", "write")).Post("/damages", fr.HandleReportDamage)
			})
		})

		// Services top-level
		r.Route("/services", func(r chi.Router) {
			r.With(middleware.RequirePermission("fuhrpark:service", "read")).Get("/", fr.HandleListServices)
			r.With(middleware.RequirePermission("fuhrpark:service", "read")).Get("/upcoming", fr.HandleListUpcomingServices)

			r.Route("/{id}", func(r chi.Router) {
				r.With(middleware.RequirePermission("fuhrpark:service", "write")).Patch("/", fr.HandleUpdateService)
				r.With(middleware.RequirePermission("fuhrpark:service", "write")).Delete("/", fr.HandleDeleteService)
				r.With(middleware.RequirePermission("fuhrpark:service", "write")).Post("/complete", fr.HandleCompleteService)
			})
		})

		// Damages top-level
		r.Route("/damages", func(r chi.Router) {
			r.With(middleware.RequirePermission("fuhrpark:damage", "read")).Get("/", fr.HandleListDamages)

			r.Route("/{id}", func(r chi.Router) {
				r.With(middleware.RequirePermission("fuhrpark:damage", "write")).Patch("/", fr.HandleUpdateDamage)
				r.With(middleware.RequirePermission("fuhrpark:damage", "write")).Post("/resolve", fr.HandleResolveDamage)
			})
		})

		// TUEV check
		r.With(middleware.RequirePermission("fuhrpark:vehicle", "read")).Get("/tuev-due", fr.HandleCheckTuevDue)

		// Report export
		r.With(middleware.RequirePermission("fuhrpark:vehicle", "read")).Get("/export", fr.HandleExportVehicleReport)
	})
}

// ============================================================================
// Request types
// ============================================================================

type createVehicleRequest struct {
	LicensePlate     string  `json:"license_plate"              validate:"required"`
	Make             string  `json:"make"                       validate:"required"`
	Model            string  `json:"model"                      validate:"required"`
	Year             int32   `json:"year"                       validate:"omitempty,gte=1900,lte=2100"`
	VIN              *string `json:"vin,omitempty"`
	Color            *string `json:"color,omitempty"`
	FuelType         string  `json:"fuel_type"                  validate:"omitempty,oneof=petrol diesel electric hybrid other"`
	MileageKm        int64   `json:"mileage_km"                 validate:"gte=0"`
	TuevDueDate      *string `json:"tuev_due_date,omitempty"`
	AssignedDriverID *string `json:"assigned_driver_id,omitempty" validate:"omitempty,uuid"`
	Notes            *string `json:"notes,omitempty"`
}

type updateVehicleRequest struct {
	LicensePlate     *string `json:"license_plate,omitempty"`
	Make             *string `json:"make,omitempty"`
	Model            *string `json:"model,omitempty"`
	Year             *int32  `json:"year,omitempty"`
	VIN              *string `json:"vin,omitempty"`
	Color            *string `json:"color,omitempty"`
	FuelType         *string `json:"fuel_type,omitempty"           validate:"omitempty,oneof=petrol diesel electric hybrid other"`
	Status           *string `json:"status,omitempty"`
	MileageKm        *int64  `json:"mileage_km,omitempty"`
	TuevDueDate      *string `json:"tuev_due_date,omitempty"`
	AssignedDriverID *string `json:"assigned_driver_id,omitempty"  validate:"omitempty,uuid"`
	Notes            *string `json:"notes,omitempty"`
}

type scheduleServiceRequest struct {
	ServiceType string  `json:"service_type"          validate:"required"`
	Description *string `json:"description,omitempty"`
	ScheduledAt string  `json:"scheduled_at"          validate:"required"`
	CostCents   *int64  `json:"cost_cents,omitempty"  validate:"omitempty,gte=0"`
	Workshop    *string `json:"workshop,omitempty"`
	MileageKm   *int64  `json:"mileage_km,omitempty"`
	Notes       *string `json:"notes,omitempty"`
}

type updateServiceRequest struct {
	ServiceType *string `json:"service_type,omitempty"`
	Description *string `json:"description,omitempty"`
	ScheduledAt *string `json:"scheduled_at,omitempty"`
	CostCents   *int64  `json:"cost_cents,omitempty"`
	Workshop    *string `json:"workshop,omitempty"`
	MileageKm   *int64  `json:"mileage_km,omitempty"`
	Status      *string `json:"status,omitempty"`
	Notes       *string `json:"notes,omitempty"`
}

type completeServiceRequest struct {
	CostCents *int64  `json:"cost_cents,omitempty"`
	MileageKm *int64  `json:"mileage_km,omitempty"`
	Notes     *string `json:"notes,omitempty"`
}

type reportDamageRequest struct {
	Description string   `json:"description"           validate:"required"`
	Severity    string   `json:"severity"              validate:"omitempty,oneof=minor moderate major totalled"`
	ReportedBy  *string  `json:"reported_by,omitempty" validate:"omitempty,uuid"`
	PhotoKeys   []string `json:"photo_keys,omitempty"`
	CostCents   *int64   `json:"cost_cents,omitempty"  validate:"omitempty,gte=0"`
	Notes       *string  `json:"notes,omitempty"`
}

type updateDamageRequest struct {
	Description *string  `json:"description,omitempty"`
	Severity    *string  `json:"severity,omitempty"`
	Status      *string  `json:"status,omitempty"`
	PhotoKeys   []string `json:"photo_keys,omitempty"`
	CostCents   *int64   `json:"cost_cents,omitempty"`
	Notes       *string  `json:"notes,omitempty"`
}

type resolveDamageRequest struct {
	ResolvedBy *string `json:"resolved_by,omitempty"`
	CostCents  *int64  `json:"cost_cents,omitempty"`
	Notes      *string `json:"notes,omitempty"`
}

// ============================================================================
// Vehicle Handlers
// ============================================================================

func (fr *FuhrparkRoutes) HandleListVehicles(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}
	client, err := fr.getClient()
	if err != nil {
		respondServiceUnavailable(w, fr.ServiceName())
		return
	}

	page, pageSize := parsePagination(r, 1, 50)
	q := r.URL.Query()

	grpcReq := &fuhrparkv1.ListVehiclesRequest{
		TenantId: tenantID.String(),
		Search:   q.Get("search"),
		Page:     int32(page),
		PageSize: int32(pageSize),
	}
	if s := q.Get("status"); s != "" {
		grpcReq.Status = &s
	}
	if ft := q.Get("fuel_type"); ft != "" {
		grpcReq.FuelType = &ft
	}

	resp, err := client.ListVehicles(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

func (fr *FuhrparkRoutes) HandleCreateVehicle(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}
	client, err := fr.getClient()
	if err != nil {
		respondServiceUnavailable(w, fr.ServiceName())
		return
	}

	req, ok := decodeAndValidate[createVehicleRequest](w, r)
	if !ok {
		return
	}

	grpcReq := &fuhrparkv1.CreateVehicleRequest{
		TenantId:         tenantID.String(),
		LicensePlate:     req.LicensePlate,
		Make:             req.Make,
		Model:            req.Model,
		Year:             req.Year,
		Vin:              req.VIN,
		Color:            req.Color,
		FuelType:         req.FuelType,
		MileageKm:        req.MileageKm,
		TuevDueDate:      req.TuevDueDate,
		AssignedDriverId: req.AssignedDriverID,
		Notes:            req.Notes,
	}

	resp, err := client.CreateVehicle(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, resp)
}

func (fr *FuhrparkRoutes) HandleGetVehicle(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}
	client, err := fr.getClient()
	if err != nil {
		respondServiceUnavailable(w, fr.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.GetVehicle(r.Context(), &fuhrparkv1.GetVehicleRequest{
		TenantId:  tenantID.String(),
		VehicleId: id,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

func (fr *FuhrparkRoutes) HandleUpdateVehicle(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}
	client, err := fr.getClient()
	if err != nil {
		respondServiceUnavailable(w, fr.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	req, ok := decodeAndValidate[updateVehicleRequest](w, r)
	if !ok {
		return
	}

	grpcReq := &fuhrparkv1.UpdateVehicleRequest{
		TenantId:         tenantID.String(),
		VehicleId:        id,
		LicensePlate:     req.LicensePlate,
		Make:             req.Make,
		Model:            req.Model,
		Vin:              req.VIN,
		Color:            req.Color,
		FuelType:         req.FuelType,
		Status:           req.Status,
		MileageKm:        req.MileageKm,
		TuevDueDate:      req.TuevDueDate,
		AssignedDriverId: req.AssignedDriverID,
		Notes:            req.Notes,
	}
	if req.Year != nil {
		grpcReq.Year = req.Year
	}

	resp, err := client.UpdateVehicle(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

func (fr *FuhrparkRoutes) HandleDeleteVehicle(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}
	client, err := fr.getClient()
	if err != nil {
		respondServiceUnavailable(w, fr.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	_, err = client.DeleteVehicle(r.Context(), &fuhrparkv1.DeleteVehicleRequest{
		TenantId:  tenantID.String(),
		VehicleId: id,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (fr *FuhrparkRoutes) HandleGetVehicleHistory(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}
	client, err := fr.getClient()
	if err != nil {
		respondServiceUnavailable(w, fr.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	page, pageSize := parsePagination(r, 1, 50)

	resp, err := client.GetVehicleHistory(r.Context(), &fuhrparkv1.GetVehicleHistoryRequest{
		TenantId:  tenantID.String(),
		VehicleId: id,
		Page:      int32(page),
		PageSize:  int32(pageSize),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

// ============================================================================
// Service Handlers
// ============================================================================

func (fr *FuhrparkRoutes) HandleScheduleService(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}
	client, err := fr.getClient()
	if err != nil {
		respondServiceUnavailable(w, fr.ServiceName())
		return
	}

	vehicleID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	req, ok := decodeAndValidate[scheduleServiceRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.ScheduleService(r.Context(), &fuhrparkv1.ScheduleServiceRequest{
		TenantId:    tenantID.String(),
		VehicleId:   vehicleID,
		ServiceType: req.ServiceType,
		Description: req.Description,
		ScheduledAt: req.ScheduledAt,
		CostCents:   req.CostCents,
		Workshop:    req.Workshop,
		MileageKm:   req.MileageKm,
		Notes:       req.Notes,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, resp)
}

func (fr *FuhrparkRoutes) HandleListVehicleServices(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}
	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}
	client, err := fr.getClient()
	if err != nil {
		respondServiceUnavailable(w, fr.ServiceName())
		return
	}
	page, pageSize := parsePagination(r, 1, 50)
	resp, err := client.ListServices(r.Context(), &fuhrparkv1.ListServicesRequest{
		TenantId:  tenantID.String(),
		VehicleId: &id,
		Page:      int32(page),
		PageSize:  int32(pageSize),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

func (fr *FuhrparkRoutes) HandleListServices(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}
	client, err := fr.getClient()
	if err != nil {
		respondServiceUnavailable(w, fr.ServiceName())
		return
	}
	page, pageSize := parsePagination(r, 1, 50)
	q := r.URL.Query()
	grpcReq := &fuhrparkv1.ListServicesRequest{
		TenantId: tenantID.String(),
		Page:     int32(page),
		PageSize: int32(pageSize),
	}
	if s := q.Get("status"); s != "" {
		grpcReq.Status = &s
	}
	if vid := q.Get("vehicle_id"); vid != "" {
		grpcReq.VehicleId = &vid
	}
	resp, err := client.ListServices(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

func (fr *FuhrparkRoutes) HandleUpdateService(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}
	client, err := fr.getClient()
	if err != nil {
		respondServiceUnavailable(w, fr.ServiceName())
		return
	}
	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}
	req, ok := decodeAndValidate[updateServiceRequest](w, r)
	if !ok {
		return
	}
	resp, err := client.UpdateService(r.Context(), &fuhrparkv1.UpdateServiceRequest{
		TenantId:    tenantID.String(),
		ServiceId:   id,
		ServiceType: req.ServiceType,
		Description: req.Description,
		ScheduledAt: req.ScheduledAt,
		CostCents:   req.CostCents,
		Workshop:    req.Workshop,
		MileageKm:   req.MileageKm,
		Status:      req.Status,
		Notes:       req.Notes,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

func (fr *FuhrparkRoutes) HandleDeleteService(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}
	client, err := fr.getClient()
	if err != nil {
		respondServiceUnavailable(w, fr.ServiceName())
		return
	}
	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}
	_, err = client.DeleteService(r.Context(), &fuhrparkv1.DeleteServiceRequest{
		TenantId:  tenantID.String(),
		ServiceId: id,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (fr *FuhrparkRoutes) HandleCompleteService(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}
	client, err := fr.getClient()
	if err != nil {
		respondServiceUnavailable(w, fr.ServiceName())
		return
	}
	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}
	req, ok := decodeAndValidate[completeServiceRequest](w, r)
	if !ok {
		return
	}
	resp, err := client.CompleteService(r.Context(), &fuhrparkv1.CompleteServiceRequest{
		TenantId:  tenantID.String(),
		ServiceId: id,
		CostCents: req.CostCents,
		MileageKm: req.MileageKm,
		Notes:     req.Notes,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

func (fr *FuhrparkRoutes) HandleListUpcomingServices(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}
	client, err := fr.getClient()
	if err != nil {
		respondServiceUnavailable(w, fr.ServiceName())
		return
	}
	page, pageSize := parsePagination(r, 1, 50)
	resp, err := client.ListUpcomingServices(r.Context(), &fuhrparkv1.ListUpcomingServicesRequest{
		TenantId: tenantID.String(),
		Page:     int32(page),
		PageSize: int32(pageSize),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

// ============================================================================
// Damage Handlers
// ============================================================================

func (fr *FuhrparkRoutes) HandleReportDamage(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}
	client, err := fr.getClient()
	if err != nil {
		respondServiceUnavailable(w, fr.ServiceName())
		return
	}
	vehicleID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}
	req, ok := decodeAndValidate[reportDamageRequest](w, r)
	if !ok {
		return
	}
	resp, err := client.ReportDamage(r.Context(), &fuhrparkv1.ReportDamageRequest{
		TenantId:    tenantID.String(),
		VehicleId:   vehicleID,
		Description: req.Description,
		Severity:    req.Severity,
		ReportedBy:  req.ReportedBy,
		PhotoKeys:   req.PhotoKeys,
		CostCents:   req.CostCents,
		Notes:       req.Notes,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, resp)
}

func (fr *FuhrparkRoutes) HandleListVehicleDamages(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}
	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}
	client, err := fr.getClient()
	if err != nil {
		respondServiceUnavailable(w, fr.ServiceName())
		return
	}
	page, pageSize := parsePagination(r, 1, 50)
	resp, err := client.ListDamages(r.Context(), &fuhrparkv1.ListDamagesRequest{
		TenantId:  tenantID.String(),
		VehicleId: &id,
		Page:      int32(page),
		PageSize:  int32(pageSize),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

func (fr *FuhrparkRoutes) HandleListDamages(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}
	client, err := fr.getClient()
	if err != nil {
		respondServiceUnavailable(w, fr.ServiceName())
		return
	}
	page, pageSize := parsePagination(r, 1, 50)
	q := r.URL.Query()
	grpcReq := &fuhrparkv1.ListDamagesRequest{
		TenantId: tenantID.String(),
		Page:     int32(page),
		PageSize: int32(pageSize),
	}
	if s := q.Get("status"); s != "" {
		grpcReq.Status = &s
	}
	if vid := q.Get("vehicle_id"); vid != "" {
		grpcReq.VehicleId = &vid
	}
	resp, err := client.ListDamages(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

func (fr *FuhrparkRoutes) HandleUpdateDamage(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}
	client, err := fr.getClient()
	if err != nil {
		respondServiceUnavailable(w, fr.ServiceName())
		return
	}
	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}
	req, ok := decodeAndValidate[updateDamageRequest](w, r)
	if !ok {
		return
	}
	resp, err := client.UpdateDamage(r.Context(), &fuhrparkv1.UpdateDamageRequest{
		TenantId:    tenantID.String(),
		DamageId:    id,
		Description: req.Description,
		Severity:    req.Severity,
		Status:      req.Status,
		PhotoKeys:   req.PhotoKeys,
		CostCents:   req.CostCents,
		Notes:       req.Notes,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

func (fr *FuhrparkRoutes) HandleResolveDamage(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}
	client, err := fr.getClient()
	if err != nil {
		respondServiceUnavailable(w, fr.ServiceName())
		return
	}
	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}
	req, ok := decodeAndValidate[resolveDamageRequest](w, r)
	if !ok {
		return
	}
	resp, err := client.ResolveDamage(r.Context(), &fuhrparkv1.ResolveDamageRequest{
		TenantId:   tenantID.String(),
		DamageId:   id,
		ResolvedBy: req.ResolvedBy,
		CostCents:  req.CostCents,
		Notes:      req.Notes,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

// ============================================================================
// TUEV & Export Handlers
// ============================================================================

func (fr *FuhrparkRoutes) HandleCheckTuevDue(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}
	client, err := fr.getClient()
	if err != nil {
		respondServiceUnavailable(w, fr.ServiceName())
		return
	}
	resp, err := client.CheckTuevDue(r.Context(), &fuhrparkv1.CheckTuevDueRequest{
		TenantId: tenantID.String(),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

func (fr *FuhrparkRoutes) HandleExportVehicleReport(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}
	client, err := fr.getClient()
	if err != nil {
		respondServiceUnavailable(w, fr.ServiceName())
		return
	}
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "csv"
	}
	resp, err := client.ExportVehicleReport(r.Context(), &fuhrparkv1.ExportVehicleReportRequest{
		TenantId: tenantID.String(),
		Format:   format,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	ct := resp.GetContentType()
	if ct == "" {
		ct = "application/octet-stream"
	}
	filename := resp.GetFilename()
	if filename == "" {
		filename = "fuhrpark.csv"
	}

	w.Header().Set("Content-Type", ct)
	w.Header().Set("Content-Disposition", "attachment; filename="+formatFilename(filename))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(resp.GetPayload())
}
