package gateway

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/kmuhub/kmuhub/internal/featureflag"
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server/response"
	vermietungv1 "github.com/kmuhub/kmuhub/proto/vermietung/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// VermietungRoutes handles HTTP routes for the Vermietung (rental) module.
type VermietungRoutes struct {
	registry *ServiceRegistry
	flags    *featureflag.Registry
}

// NewVermietungRoutes creates a new VermietungRoutes.
func NewVermietungRoutes(registry *ServiceRegistry, flags *featureflag.Registry) *VermietungRoutes {
	return &VermietungRoutes{registry: registry, flags: flags}
}

// ServiceName returns the service name for the route registrar.
func (vr *VermietungRoutes) ServiceName() string { return "vermietung" }

// getClient lazily obtains a gRPC client for the vermietung service.
func (vr *VermietungRoutes) getClient() (vermietungv1.VermietungServiceClient, error) {
	conn, err := vr.registry.GetConnection("vermietung")
	if err != nil {
		return nil, err
	}
	return vermietungv1.NewVermietungServiceClient(conn), nil
}

// tenantID placeholder until JWT claim extraction is in place.
const vermietungPlaceholderTenantID = "00000000-0000-0000-0000-000000000001"

// RegisterRoutes mounts all Vermietung HTTP routes behind the feature flag modules.vermietung.
func (vr *VermietungRoutes) RegisterRoutes(r chi.Router, authMiddleware func(http.Handler) http.Handler) {
	if !vr.flags.IsEnabled("modules.vermietung") {
		return
	}

	r.Route("/api/v1/vermietung", func(r chi.Router) {
		r.Use(authMiddleware)
		r.Use(RequireAuthenticated)

		// Objects
		r.Route("/objects", func(r chi.Router) {
			r.With(middleware.RequirePermission("vermietung:object", "read")).Get("/", vr.HandleListObjects)
			r.With(middleware.RequirePermission("vermietung:object", "write")).Post("/", vr.HandleCreateObject)

			r.Route("/{id}", func(r chi.Router) {
				r.With(middleware.RequirePermission("vermietung:object", "read")).Get("/", vr.HandleGetObject)
				r.With(middleware.RequirePermission("vermietung:object", "write")).Patch("/", vr.HandleUpdateObject)
				r.With(middleware.RequirePermission("vermietung:object", "write")).Delete("/", vr.HandleDeleteObject)

				r.With(middleware.RequirePermission("vermietung:object", "read")).Get("/availability", vr.HandleCheckAvailability)
			})
		})

		// Rentals
		r.Route("/rentals", func(r chi.Router) {
			r.With(middleware.RequirePermission("vermietung:rental", "read")).Get("/", vr.HandleListRentals)
			r.With(middleware.RequirePermission("vermietung:rental", "write")).Post("/", vr.HandleCreateRental)

			r.Route("/{id}", func(r chi.Router) {
				r.With(middleware.RequirePermission("vermietung:rental", "read")).Get("/", vr.HandleGetRental)
				r.With(middleware.RequirePermission("vermietung:rental", "write")).Patch("/", vr.HandleUpdateRental)
				r.With(middleware.RequirePermission("vermietung:rental", "write")).Delete("/", vr.HandleDeleteRental)

				r.With(middleware.RequirePermission("vermietung:rental", "write")).Post("/start", vr.HandleStartRental)
				r.With(middleware.RequirePermission("vermietung:rental", "write")).Post("/end", vr.HandleEndRental)

				// Inspections scoped to a rental
				r.Route("/inspections", func(r chi.Router) {
					r.With(middleware.RequirePermission("vermietung:inspection", "read")).Get("/", vr.HandleListInspections)
					r.With(middleware.RequirePermission("vermietung:inspection", "write")).Post("/", vr.HandleCreateInspection)
				})
			})
		})

		// Inspections (direct access)
		r.Route("/inspections/{id}", func(r chi.Router) {
			r.With(middleware.RequirePermission("vermietung:inspection", "read")).Get("/", vr.HandleGetInspection)
			r.With(middleware.RequirePermission("vermietung:inspection", "write")).Patch("/", vr.HandleUpdateInspection)
		})

		// Calendar
		r.With(middleware.RequirePermission("vermietung:rental", "read")).Get("/calendar", vr.HandleGetRentalCalendar)

		// Export
		r.With(middleware.RequirePermission("vermietung:rental", "read")).Get("/export", vr.HandleExportRentalReport)
	})
}

// ============================================================================
// Request types
// ============================================================================

type createObjectRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	Category    string  `json:"category"`
	DailyRate   float64 `json:"daily_rate"`
	Deposit     float64 `json:"deposit"`
	Location    *string `json:"location,omitempty"`
	Notes       *string `json:"notes,omitempty"`
}

type updateObjectRequest struct {
	Name        *string  `json:"name,omitempty"`
	Description *string  `json:"description,omitempty"`
	Category    *string  `json:"category,omitempty"`
	DailyRate   *float64 `json:"daily_rate,omitempty"`
	Deposit     *float64 `json:"deposit,omitempty"`
	Location    *string  `json:"location,omitempty"`
	Notes       *string  `json:"notes,omitempty"`
	Active      *bool    `json:"active,omitempty"`
}

type createRentalRequest struct {
	ObjectID    string  `json:"object_id"`
	ContactID   *string `json:"contact_id,omitempty"`
	RenterName  string  `json:"renter_name"`
	StartDate   string  `json:"start_date"` // RFC3339
	EndDate     string  `json:"end_date"`   // RFC3339
	TotalPrice  float64 `json:"total_price"`
	DepositPaid bool    `json:"deposit_paid"`
	Notes       *string `json:"notes,omitempty"`
}

type updateRentalRequest struct {
	RenterName  *string  `json:"renter_name,omitempty"`
	StartDate   *string  `json:"start_date,omitempty"`
	EndDate     *string  `json:"end_date,omitempty"`
	TotalPrice  *float64 `json:"total_price,omitempty"`
	DepositPaid *bool    `json:"deposit_paid,omitempty"`
	Notes       *string  `json:"notes,omitempty"`
}

type createInspectionRequest struct {
	Kind        string   `json:"kind"` // handover|return
	Notes       string   `json:"notes"`
	PhotoURLs   []string `json:"photo_urls,omitempty"`
	PerformedBy *string  `json:"performed_by,omitempty"`
}

type updateInspectionRequest struct {
	Notes     *string  `json:"notes,omitempty"`
	PhotoURLs []string `json:"photo_urls,omitempty"`
}

// ============================================================================
// Object Handlers
// ============================================================================

func (vr *VermietungRoutes) HandleListObjects(w http.ResponseWriter, r *http.Request) {
	client, err := vr.getClient()
	if err != nil {
		respondServiceUnavailable(w, vr.ServiceName())
		return
	}

	page, pageSize := parsePagination(r, 1, 50)
	q := r.URL.Query()

	grpcReq := &vermietungv1.ListObjectsRequest{
		TenantId: vermietungPlaceholderTenantID,
		Search:   q.Get("search"),
		Page:     int32(page),
		PageSize: int32(pageSize),
	}
	if cat := q.Get("category"); cat != "" {
		grpcReq.Category = &cat
	}
	if q.Get("active_only") == "true" {
		t := true
		grpcReq.ActiveOnly = &t
	}

	resp, err := client.ListObjects(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

func (vr *VermietungRoutes) HandleCreateObject(w http.ResponseWriter, r *http.Request) {
	client, err := vr.getClient()
	if err != nil {
		respondServiceUnavailable(w, vr.ServiceName())
		return
	}

	var req createObjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		response.Error(w, http.StatusBadRequest, "name is required")
		return
	}

	resp, err := client.CreateObject(r.Context(), &vermietungv1.CreateObjectRequest{
		TenantId:    vermietungPlaceholderTenantID,
		Name:        req.Name,
		Description: req.Description,
		Category:    req.Category,
		DailyRate:   req.DailyRate,
		Deposit:     req.Deposit,
		Location:    req.Location,
		Notes:       req.Notes,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, resp)
}

func (vr *VermietungRoutes) HandleGetObject(w http.ResponseWriter, r *http.Request) {
	client, err := vr.getClient()
	if err != nil {
		respondServiceUnavailable(w, vr.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.GetObject(r.Context(), &vermietungv1.GetObjectRequest{
		TenantId: vermietungPlaceholderTenantID,
		ObjectId: id,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

func (vr *VermietungRoutes) HandleUpdateObject(w http.ResponseWriter, r *http.Request) {
	client, err := vr.getClient()
	if err != nil {
		respondServiceUnavailable(w, vr.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	var req updateObjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	grpcReq := &vermietungv1.UpdateObjectRequest{
		TenantId:    vermietungPlaceholderTenantID,
		ObjectId:    id,
		Name:        req.Name,
		Description: req.Description,
		Category:    req.Category,
		Location:    req.Location,
		Notes:       req.Notes,
		Active:      req.Active,
	}
	if req.DailyRate != nil {
		grpcReq.DailyRate = req.DailyRate
	}
	if req.Deposit != nil {
		grpcReq.Deposit = req.Deposit
	}

	resp, err := client.UpdateObject(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

func (vr *VermietungRoutes) HandleDeleteObject(w http.ResponseWriter, r *http.Request) {
	client, err := vr.getClient()
	if err != nil {
		respondServiceUnavailable(w, vr.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	_, err = client.DeleteObject(r.Context(), &vermietungv1.DeleteObjectRequest{
		TenantId: vermietungPlaceholderTenantID,
		ObjectId: id,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (vr *VermietungRoutes) HandleCheckAvailability(w http.ResponseWriter, r *http.Request) {
	client, err := vr.getClient()
	if err != nil {
		respondServiceUnavailable(w, vr.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	q := r.URL.Query()
	startStr := q.Get("start_date")
	endStr := q.Get("end_date")
	if startStr == "" || endStr == "" {
		response.Error(w, http.StatusBadRequest, "start_date and end_date are required")
		return
	}

	startTime, parseErr := parseRFC3339(startStr)
	if parseErr != nil {
		response.Error(w, http.StatusBadRequest, "invalid start_date format (RFC3339 required)")
		return
	}
	endTime, parseErr := parseRFC3339(endStr)
	if parseErr != nil {
		response.Error(w, http.StatusBadRequest, "invalid end_date format (RFC3339 required)")
		return
	}

	grpcReq := &vermietungv1.CheckAvailabilityRequest{
		TenantId:  vermietungPlaceholderTenantID,
		ObjectId:  id,
		StartDate: timestamppb.New(startTime),
		EndDate:   timestamppb.New(endTime),
	}
	if excl := q.Get("exclude_rental_id"); excl != "" {
		grpcReq.ExcludeRentalId = &excl
	}

	resp, err := client.CheckAvailability(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

// ============================================================================
// Rental Handlers
// ============================================================================

func (vr *VermietungRoutes) HandleListRentals(w http.ResponseWriter, r *http.Request) {
	client, err := vr.getClient()
	if err != nil {
		respondServiceUnavailable(w, vr.ServiceName())
		return
	}

	page, pageSize := parsePagination(r, 1, 50)
	q := r.URL.Query()

	grpcReq := &vermietungv1.ListRentalsRequest{
		TenantId: vermietungPlaceholderTenantID,
		Page:     int32(page),
		PageSize: int32(pageSize),
	}
	if obj := q.Get("object_id"); obj != "" {
		grpcReq.ObjectId = &obj
	}
	if st := q.Get("status"); st != "" {
		grpcReq.Status = &st
	}
	if from := q.Get("from"); from != "" {
		t, parseErr := parseRFC3339(from)
		if parseErr == nil {
			grpcReq.From = timestamppb.New(t)
		}
	}
	if to := q.Get("to"); to != "" {
		t, parseErr := parseRFC3339(to)
		if parseErr == nil {
			grpcReq.To = timestamppb.New(t)
		}
	}

	resp, err := client.ListRentals(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

func (vr *VermietungRoutes) HandleCreateRental(w http.ResponseWriter, r *http.Request) {
	client, err := vr.getClient()
	if err != nil {
		respondServiceUnavailable(w, vr.ServiceName())
		return
	}

	var req createRentalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.ObjectID == "" {
		response.Error(w, http.StatusBadRequest, "object_id is required")
		return
	}
	if req.RenterName == "" {
		response.Error(w, http.StatusBadRequest, "renter_name is required")
		return
	}

	startTime, parseErr := parseRFC3339(req.StartDate)
	if parseErr != nil {
		response.Error(w, http.StatusBadRequest, "invalid start_date format (RFC3339 required)")
		return
	}
	endTime, parseErr := parseRFC3339(req.EndDate)
	if parseErr != nil {
		response.Error(w, http.StatusBadRequest, "invalid end_date format (RFC3339 required)")
		return
	}

	grpcReq := &vermietungv1.CreateRentalRequest{
		TenantId:    vermietungPlaceholderTenantID,
		ObjectId:    req.ObjectID,
		ContactId:   req.ContactID,
		RenterName:  req.RenterName,
		StartDate:   timestamppb.New(startTime),
		EndDate:     timestamppb.New(endTime),
		TotalPrice:  req.TotalPrice,
		DepositPaid: req.DepositPaid,
		Notes:       req.Notes,
	}

	resp, err := client.CreateRental(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, resp)
}

func (vr *VermietungRoutes) HandleGetRental(w http.ResponseWriter, r *http.Request) {
	client, err := vr.getClient()
	if err != nil {
		respondServiceUnavailable(w, vr.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.GetRental(r.Context(), &vermietungv1.GetRentalRequest{
		TenantId: vermietungPlaceholderTenantID,
		RentalId: id,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

func (vr *VermietungRoutes) HandleUpdateRental(w http.ResponseWriter, r *http.Request) {
	client, err := vr.getClient()
	if err != nil {
		respondServiceUnavailable(w, vr.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	var req updateRentalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	grpcReq := &vermietungv1.UpdateRentalRequest{
		TenantId:   vermietungPlaceholderTenantID,
		RentalId:   id,
		RenterName: req.RenterName,
		Notes:      req.Notes,
	}
	if req.StartDate != nil {
		t, parseErr := parseRFC3339(*req.StartDate)
		if parseErr != nil {
			response.Error(w, http.StatusBadRequest, "invalid start_date format")
			return
		}
		grpcReq.StartDate = timestamppb.New(t)
	}
	if req.EndDate != nil {
		t, parseErr := parseRFC3339(*req.EndDate)
		if parseErr != nil {
			response.Error(w, http.StatusBadRequest, "invalid end_date format")
			return
		}
		grpcReq.EndDate = timestamppb.New(t)
	}
	if req.TotalPrice != nil {
		grpcReq.TotalPrice = req.TotalPrice
	}
	if req.DepositPaid != nil {
		grpcReq.DepositPaid = req.DepositPaid
	}

	resp, err := client.UpdateRental(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

func (vr *VermietungRoutes) HandleDeleteRental(w http.ResponseWriter, r *http.Request) {
	client, err := vr.getClient()
	if err != nil {
		respondServiceUnavailable(w, vr.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	_, err = client.DeleteRental(r.Context(), &vermietungv1.DeleteRentalRequest{
		TenantId: vermietungPlaceholderTenantID,
		RentalId: id,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (vr *VermietungRoutes) HandleStartRental(w http.ResponseWriter, r *http.Request) {
	client, err := vr.getClient()
	if err != nil {
		respondServiceUnavailable(w, vr.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.StartRental(r.Context(), &vermietungv1.StartRentalRequest{
		TenantId: vermietungPlaceholderTenantID,
		RentalId: id,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

func (vr *VermietungRoutes) HandleEndRental(w http.ResponseWriter, r *http.Request) {
	client, err := vr.getClient()
	if err != nil {
		respondServiceUnavailable(w, vr.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.EndRental(r.Context(), &vermietungv1.EndRentalRequest{
		TenantId: vermietungPlaceholderTenantID,
		RentalId: id,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

// ============================================================================
// Inspection Handlers
// ============================================================================

func (vr *VermietungRoutes) HandleListInspections(w http.ResponseWriter, r *http.Request) {
	client, err := vr.getClient()
	if err != nil {
		respondServiceUnavailable(w, vr.ServiceName())
		return
	}

	rentalID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	page, pageSize := parsePagination(r, 1, 50)

	resp, err := client.ListInspections(r.Context(), &vermietungv1.ListInspectionsRequest{
		TenantId: vermietungPlaceholderTenantID,
		RentalId: rentalID,
		Page:     int32(page),
		PageSize: int32(pageSize),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

func (vr *VermietungRoutes) HandleCreateInspection(w http.ResponseWriter, r *http.Request) {
	client, err := vr.getClient()
	if err != nil {
		respondServiceUnavailable(w, vr.ServiceName())
		return
	}

	rentalID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	var req createInspectionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Kind == "" {
		response.Error(w, http.StatusBadRequest, "kind is required (handover or return)")
		return
	}

	resp, err := client.CreateInspection(r.Context(), &vermietungv1.CreateInspectionRequest{
		TenantId:    vermietungPlaceholderTenantID,
		RentalId:    rentalID,
		Kind:        req.Kind,
		Notes:       req.Notes,
		PhotoUrls:   req.PhotoURLs,
		PerformedBy: req.PerformedBy,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, resp)
}

func (vr *VermietungRoutes) HandleGetInspection(w http.ResponseWriter, r *http.Request) {
	client, err := vr.getClient()
	if err != nil {
		respondServiceUnavailable(w, vr.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.GetInspection(r.Context(), &vermietungv1.GetInspectionRequest{
		TenantId:     vermietungPlaceholderTenantID,
		InspectionId: id,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

func (vr *VermietungRoutes) HandleUpdateInspection(w http.ResponseWriter, r *http.Request) {
	client, err := vr.getClient()
	if err != nil {
		respondServiceUnavailable(w, vr.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	var req updateInspectionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := client.UpdateInspection(r.Context(), &vermietungv1.UpdateInspectionRequest{
		TenantId:     vermietungPlaceholderTenantID,
		InspectionId: id,
		Notes:        req.Notes,
		PhotoUrls:    req.PhotoURLs,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

// ============================================================================
// Calendar & Export Handlers
// ============================================================================

func (vr *VermietungRoutes) HandleGetRentalCalendar(w http.ResponseWriter, r *http.Request) {
	client, err := vr.getClient()
	if err != nil {
		respondServiceUnavailable(w, vr.ServiceName())
		return
	}

	q := r.URL.Query()
	year := 0
	month := 0
	if y := q.Get("year"); y != "" {
		if parsed, parseErr := strconv.Atoi(y); parseErr == nil {
			year = parsed
		}
	}
	if m := q.Get("month"); m != "" {
		if parsed, parseErr := strconv.Atoi(m); parseErr == nil {
			month = parsed
		}
	}

	resp, err := client.GetRentalCalendar(r.Context(), &vermietungv1.GetRentalCalendarRequest{
		TenantId: vermietungPlaceholderTenantID,
		Year:     int32(year),
		Month:    int32(month),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

func (vr *VermietungRoutes) HandleExportRentalReport(w http.ResponseWriter, r *http.Request) {
	client, err := vr.getClient()
	if err != nil {
		respondServiceUnavailable(w, vr.ServiceName())
		return
	}

	format := r.URL.Query().Get("format")
	if format == "" {
		format = "csv"
	}

	grpcReq := &vermietungv1.ExportRentalReportRequest{
		TenantId: vermietungPlaceholderTenantID,
		Format:   format,
	}

	resp, err := client.ExportRentalReport(r.Context(), grpcReq)
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
		filename = "vermietung.csv"
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", "attachment; filename="+formatFilename(filename))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(resp.GetPayload())
}

// parseRFC3339 parses an RFC3339 timestamp string into a time.Time value.
func parseRFC3339(s string) (time.Time, error) {
	return time.Parse(time.RFC3339, s)
}
