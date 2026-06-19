package gateway

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

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

				r.With(middleware.RequirePermission("vertraege:contract", "read")).Get("/export", vr.HandleExportContract)
				r.With(middleware.RequirePermission("vertraege:contract", "write")).Put("/signature", vr.HandleSaveContractSignature)

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
	ContractNumber string `json:"contract_number"    validate:"required"`
	Title          string `json:"title"              validate:"required"`
	ContractType   string `json:"contract_type"      validate:"required"`
	StartsOn       string `json:"starts_on"` // RFC3339 date — optional, no parse in original
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
	PartyType      string `json:"party_type"          validate:"required"`
	ContactID      string `json:"contact_id,omitempty"  validate:"omitempty,uuid"`
	CompanyID      string `json:"company_id,omitempty"  validate:"omitempty,uuid"`
	ExternalName   string `json:"external_name,omitempty"`
	RoleInContract string `json:"role_in_contract"    validate:"required"`
	SignedOn       string `json:"signed_on,omitempty"`
}

type saveContractSignatureRequest struct {
	SignatureData string `json:"signature_data" validate:"required"`
	SignedBy      string `json:"signed_by"      validate:"required"`
}

type createReminderRequest struct {
	RemindAt     string `json:"remind_at"`
	ReminderType string `json:"reminder_type"`
	Subject      string `json:"subject" validate:"required"`
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
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
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
	if cid := q.Get("contact_id"); cid != "" {
		if _, parseErr := uuid.Parse(cid); parseErr != nil {
			response.JSON(w, http.StatusBadRequest, struct {
				Error string `json:"error"`
				Code  string `json:"code"`
			}{
				Error: "invalid contact_id: must be a valid UUID",
				Code:  "INVALID_CONTACT_ID",
			})
			return
		}
		grpcReq.ContactId = &cid
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
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := vr.getClient()
	if err != nil {
		respondServiceUnavailable(w, vr.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	req, ok := decodeAndValidate[createContractRequest](w, r)
	if !ok {
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
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
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
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
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

	req, ok := decodeAndValidate[updateContractRequest](w, r)
	if !ok {
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
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
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

func (vr *VertraegeRoutes) HandleExportContract(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
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
// Signature Handler
// ============================================================================

func (vr *VertraegeRoutes) HandleSaveContractSignature(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}
	client, err := vr.getClient()
	if err != nil {
		respondServiceUnavailable(w, vr.ServiceName())
		return
	}

	contractID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	req, ok := decodeAndValidate[saveContractSignatureRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.SaveSignature(r.Context(), &vertraegev1.SaveContractSignatureRequest{
		TenantId:      tenantID.String(),
		ContractId:    contractID,
		SignatureData: req.SignatureData,
		SignedBy:      req.SignedBy,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, resp)
}

// ============================================================================
// Party Handlers
// ============================================================================

func (vr *VertraegeRoutes) HandleListParties(w http.ResponseWriter, r *http.Request) {
	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
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
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
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

	req, ok := decodeAndValidate[addPartyRequest](w, r)
	if !ok {
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
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
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
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
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
		TenantId:    tenantID.String(),
		ContractId:  id,
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
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
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

	req, ok := decodeAndValidate[createReminderRequest](w, r)
	if !ok {
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
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
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

	req, ok := decodeAndValidate[updateReminderRequest](w, r)
	if !ok {
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
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
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
