package gateway

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/kmuhub/kmuhub/internal/featureflag"
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server/response"
	vertraegev1 "github.com/kmuhub/kmuhub/proto/vertraege/v1"
)

// VertraegeRoutes handles HTTP routes for the Vertraege (contracts) module.
type VertraegeRoutes struct {
	registry *ServiceRegistry
	flags    *featureflag.Registry
}

// NewVertraegeRoutes creates a new VertraegeRoutes.
func NewVertraegeRoutes(registry *ServiceRegistry, flags *featureflag.Registry) *VertraegeRoutes {
	return &VertraegeRoutes{registry: registry, flags: flags}
}

// ServiceName returns the service name for the route registrar.
func (vr *VertraegeRoutes) ServiceName() string { return "vertraege" }

// getClient lazily obtains a gRPC client for the vertraege service.
func (vr *VertraegeRoutes) getClient() (vertraegev1.VertraegeServiceClient, error) {
	conn, err := vr.registry.GetConnection("vertraege")
	if err != nil {
		return nil, err
	}
	return vertraegev1.NewVertraegeServiceClient(conn), nil
}


// RegisterRoutes mounts all Vertraege HTTP routes behind the feature flag.
func (vr *VertraegeRoutes) RegisterRoutes(r chi.Router, authMiddleware func(http.Handler) http.Handler) {
	if !vr.flags.IsEnabled("modules.vertraege") {
		return
	}

	r.Route("/api/v1/vertraege", func(r chi.Router) {
		r.Use(authMiddleware)
		r.Use(RequireAuthenticated)

		// Contracts
		r.Route("/contracts", func(r chi.Router) {
			r.With(middleware.RequirePermission("vertraege:contract", "read")).Get("/", vr.HandleListContracts)
			r.With(middleware.RequirePermission("vertraege:contract", "write")).Post("/", vr.HandleCreateContract)

			r.Route("/{id}", func(r chi.Router) {
				r.With(middleware.RequirePermission("vertraege:contract", "read")).Get("/", vr.HandleGetContract)
				r.With(middleware.RequirePermission("vertraege:contract", "write")).Patch("/", vr.HandleUpdateContract)
				r.With(middleware.RequirePermission("vertraege:contract", "write")).Delete("/", vr.HandleDeleteContract)

				r.With(middleware.RequirePermission("vertraege:contract", "write")).Post("/document", vr.HandleUploadDocument)
				r.With(middleware.RequirePermission("vertraege:contract", "read")).Get("/export", vr.HandleExportContract)

				// Parties
				r.Route("/parties", func(r chi.Router) {
					r.With(middleware.RequirePermission("vertraege:party", "read")).Get("/", vr.HandleListParties)
					r.With(middleware.RequirePermission("vertraege:party", "write")).Post("/", vr.HandleAddParty)
					r.With(middleware.RequirePermission("vertraege:party", "write")).Delete("/{partyId}", vr.HandleRemoveParty)
				})

				// Reminders
				r.Route("/reminders", func(r chi.Router) {
					r.With(middleware.RequirePermission("vertraege:reminder", "read")).Get("/", vr.HandleListReminders)
					r.With(middleware.RequirePermission("vertraege:reminder", "write")).Post("/", vr.HandleCreateReminder)

					r.Route("/{reminderId}", func(r chi.Router) {
						r.With(middleware.RequirePermission("vertraege:reminder", "write")).Patch("/", vr.HandleUpdateReminder)
						r.With(middleware.RequirePermission("vertraege:reminder", "write")).Delete("/", vr.HandleDeleteReminder)
					})
				})
			})
		})
	})
}

// ============================================================================
// Request types
// ============================================================================

type createContractRequest struct {
	ContractNumber string `json:"contract_number"`
	Title          string `json:"title"`
	ContractType   string `json:"contract_type"`
	StartsOn       string `json:"starts_on"` // RFC3339 date
	EndsOn         string `json:"ends_on,omitempty"`
	DocumentURL    string `json:"document_url,omitempty"`
	Notes          string `json:"notes,omitempty"`
}

type updateContractRequest struct {
	Title        *string `json:"title,omitempty"`
	ContractType *string `json:"contract_type,omitempty"`
	Status       *string `json:"status,omitempty"`
	StartsOn     *string `json:"starts_on,omitempty"`
	EndsOn       *string `json:"ends_on,omitempty"`
	ClearEndsOn  bool    `json:"clear_ends_on,omitempty"`
	DocumentURL  *string `json:"document_url,omitempty"`
	Notes        *string `json:"notes,omitempty"`
}

type addPartyRequest struct {
	PartyType      string `json:"party_type"`
	ContactID      string `json:"contact_id,omitempty"`
	CompanyID      string `json:"company_id,omitempty"`
	ExternalName   string `json:"external_name,omitempty"`
	RoleInContract string `json:"role_in_contract"`
	SignedOn       string `json:"signed_on,omitempty"`
}

type createReminderRequest struct {
	RemindAt     string `json:"remind_at"`
	ReminderType string `json:"reminder_type"`
	Subject      string `json:"subject"`
	Message      string `json:"message,omitempty"`
}

type updateReminderRequest struct {
	RemindAt     *string `json:"remind_at,omitempty"`
	ReminderType *string `json:"reminder_type,omitempty"`
	Subject      *string `json:"subject,omitempty"`
	Message      *string `json:"message,omitempty"`
	Status       *string `json:"status,omitempty"`
}

// ============================================================================
// Contract Handlers
// ============================================================================

func (vr *VertraegeRoutes) HandleListContracts(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}
	client, err := vr.getClient()
	if err != nil {
		respondServiceUnavailable(w, vr.ServiceName())
		return
	}

	page, pageSize := parsePagination(r, 1, 20)
	q := r.URL.Query()

	grpcReq := &vertraegev1.ListContractsRequest{
		TenantId: tenantID.String(),
		Page:     int32(page),
		PageSize: int32(pageSize),
	}
	if s := q.Get("status"); s != "" {
		grpcReq.Status = &s
	}
	if ct := q.Get("contract_type"); ct != "" {
		grpcReq.ContractType = &ct
	}

	resp, err := client.ListContracts(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

func (vr *VertraegeRoutes) HandleCreateContract(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}
	client, err := vr.getClient()
	if err != nil {
		respondServiceUnavailable(w, vr.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	var req createContractRequest
	if decErr := json.NewDecoder(r.Body).Decode(&req); decErr != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.ContractNumber == "" {
		response.Error(w, http.StatusBadRequest, "contract_number is required")
		return
	}
	if req.Title == "" {
		response.Error(w, http.StatusBadRequest, "title is required")
		return
	}
	if req.ContractType == "" {
		response.Error(w, http.StatusBadRequest, "contract_type is required")
		return
	}

	grpcReq := &vertraegev1.CreateContractRequest{
		TenantId:       tenantID.String(),
		ContractNumber: req.ContractNumber,
		Title:          req.Title,
		ContractType:   req.ContractType,
		Notes:          req.Notes,
		CreatedBy:      &userID,
	}
	if req.DocumentURL != "" {
		grpcReq.DocumentUrl = &req.DocumentURL
	}

	resp, err := client.CreateContract(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, resp)
}

func (vr *VertraegeRoutes) HandleGetContract(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}
	client, err := vr.getClient()
	if err != nil {
		respondServiceUnavailable(w, vr.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.GetContract(r.Context(), &vertraegev1.GetContractRequest{
		TenantId:   tenantID.String(),
		ContractId: id,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

func (vr *VertraegeRoutes) HandleUpdateContract(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}
	client, err := vr.getClient()
	if err != nil {
		respondServiceUnavailable(w, vr.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	var req updateContractRequest
	if decErr := json.NewDecoder(r.Body).Decode(&req); decErr != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	grpcReq := &vertraegev1.UpdateContractRequest{
		TenantId:    tenantID.String(),
		ContractId:  id,
		Title:       req.Title,
		Status:      req.Status,
		ClearEndsOn: req.ClearEndsOn,
		Notes:       req.Notes,
	}
	if req.ContractType != nil {
		grpcReq.ContractType = req.ContractType
	}
	if req.DocumentURL != nil {
		grpcReq.DocumentUrl = req.DocumentURL
	}

	resp, err := client.UpdateContract(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

func (vr *VertraegeRoutes) HandleDeleteContract(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}
	client, err := vr.getClient()
	if err != nil {
		respondServiceUnavailable(w, vr.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	_, err = client.DeleteContract(r.Context(), &vertraegev1.DeleteContractRequest{
		TenantId:   tenantID.String(),
		ContractId: id,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (vr *VertraegeRoutes) HandleUploadDocument(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}
	client, err := vr.getClient()
	if err != nil {
		respondServiceUnavailable(w, vr.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.UploadDocument(r.Context(), &vertraegev1.UploadDocumentRequest{
		TenantId:   tenantID.String(),
		ContractId: id,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

func (vr *VertraegeRoutes) HandleExportContract(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}
	client, err := vr.getClient()
	if err != nil {
		respondServiceUnavailable(w, vr.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.ExportContract(r.Context(), &vertraegev1.ExportContractRequest{
		TenantId:   tenantID.String(),
		ContractId: id,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	contentType := resp.GetContentType()
	if contentType == "" {
		contentType = "text/plain; charset=utf-8"
	}
	filename := resp.GetFilename()
	if filename == "" {
		filename = "contract.txt"
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", "attachment; filename="+formatFilename(filename))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(resp.GetPayload())
}

// ============================================================================
// Party Handlers
// ============================================================================

func (vr *VertraegeRoutes) HandleListParties(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}
	client, err := vr.getClient()
	if err != nil {
		respondServiceUnavailable(w, vr.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.ListParties(r.Context(), &vertraegev1.ListPartiesRequest{
		TenantId:   tenantID.String(),
		ContractId: id,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

func (vr *VertraegeRoutes) HandleAddParty(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}
	client, err := vr.getClient()
	if err != nil {
		respondServiceUnavailable(w, vr.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	var req addPartyRequest
	if decErr := json.NewDecoder(r.Body).Decode(&req); decErr != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.PartyType == "" {
		response.Error(w, http.StatusBadRequest, "party_type is required")
		return
	}
	if req.RoleInContract == "" {
		response.Error(w, http.StatusBadRequest, "role_in_contract is required")
		return
	}

	grpcReq := &vertraegev1.AddPartyRequest{
		TenantId:       tenantID.String(),
		ContractId:     id,
		PartyType:      req.PartyType,
		RoleInContract: req.RoleInContract,
	}
	if req.ContactID != "" {
		grpcReq.ContactId = &req.ContactID
	}
	if req.CompanyID != "" {
		grpcReq.CompanyId = &req.CompanyID
	}
	if req.ExternalName != "" {
		grpcReq.ExternalName = &req.ExternalName
	}

	resp, err := client.AddParty(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, resp)
}

func (vr *VertraegeRoutes) HandleRemoveParty(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}
	client, err := vr.getClient()
	if err != nil {
		respondServiceUnavailable(w, vr.ServiceName())
		return
	}

	partyID, ok := validateUUIDParam(w, r, "partyId")
	if !ok {
		return
	}

	_, err = client.RemoveParty(r.Context(), &vertraegev1.RemovePartyRequest{
		TenantId: tenantID.String(),
		PartyId:  partyID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ============================================================================
// Reminder Handlers
// ============================================================================

func (vr *VertraegeRoutes) HandleListReminders(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}
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
	onlyPending := q.Get("only_pending") == "true" || q.Get("only_pending") == "1"

	resp, err := client.ListReminders(r.Context(), &vertraegev1.ListRemindersRequest{
		TenantId:   tenantID.String(),
		ContractId: id,
		OnlyPending: onlyPending,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

func (vr *VertraegeRoutes) HandleCreateReminder(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}
	client, err := vr.getClient()
	if err != nil {
		respondServiceUnavailable(w, vr.ServiceName())
		return
	}

	id, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	var req createReminderRequest
	if decErr := json.NewDecoder(r.Body).Decode(&req); decErr != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Subject == "" {
		response.Error(w, http.StatusBadRequest, "subject is required")
		return
	}

	grpcReq := &vertraegev1.CreateReminderRequest{
		TenantId:     tenantID.String(),
		ContractId:   id,
		ReminderType: req.ReminderType,
		Subject:      req.Subject,
		Message:      req.Message,
	}

	resp, err := client.CreateReminder(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, resp)
}

func (vr *VertraegeRoutes) HandleUpdateReminder(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}
	client, err := vr.getClient()
	if err != nil {
		respondServiceUnavailable(w, vr.ServiceName())
		return
	}

	reminderID, ok := validateUUIDParam(w, r, "reminderId")
	if !ok {
		return
	}

	var req updateReminderRequest
	if decErr := json.NewDecoder(r.Body).Decode(&req); decErr != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	grpcReq := &vertraegev1.UpdateReminderRequest{
		TenantId:     tenantID.String(),
		ReminderId:   reminderID,
		ReminderType: req.ReminderType,
		Subject:      req.Subject,
		Message:      req.Message,
		Status:       req.Status,
	}

	resp, err := client.UpdateReminder(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

func (vr *VertraegeRoutes) HandleDeleteReminder(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		http.Error(w, "missing or invalid tenant", http.StatusUnauthorized)
		return
	}
	client, err := vr.getClient()
	if err != nil {
		respondServiceUnavailable(w, vr.ServiceName())
		return
	}

	reminderID, ok := validateUUIDParam(w, r, "reminderId")
	if !ok {
		return
	}

	_, err = client.DeleteReminder(r.Context(), &vertraegev1.DeleteReminderRequest{
		TenantId:   tenantID.String(),
		ReminderId: reminderID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
