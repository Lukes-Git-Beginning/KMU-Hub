package gateway

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server/response"
	crmv1 "github.com/kmuhub/kmuhub/proto/crm/v1"
)

// CRMExtRoutes handles extended CRM routes via gRPC proxy.
// These routes implement: duplicate detection, contact timeline, and GDPR consent management.
type CRMExtRoutes struct {
	registry *ServiceRegistry
}

// NewCRMExtRoutes creates CRMExtRoutes using the service registry for gRPC access.
func NewCRMExtRoutes(registry *ServiceRegistry) *CRMExtRoutes {
	return &CRMExtRoutes{registry: registry}
}

// getCRMClient lazily obtains a gRPC client for the CRM service.
func (c *CRMExtRoutes) getCRMClient() (crmv1.CRMServiceClient, error) {
	conn, err := c.registry.GetConnection("crm")
	if err != nil {
		return nil, err
	}
	return crmv1.NewCRMServiceClient(conn), nil
}

// ============================================================================
// Contact Duplicate Detection
// ============================================================================

func (c *CRMExtRoutes) HandleFindContactDuplicates(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, "crm")
		return
	}

	resp, err := client.FindContactDuplicates(r.Context(), &crmv1.FindContactDuplicatesRequest{
		ContactId: chi.URLParam(r, "id"),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"duplicates": resp.Duplicates,
		"total":      resp.Total,
	})
}

type mergeContactsRequest struct {
	PrimaryID   string `json:"primary_id" validate:"required,uuid"`
	DuplicateID string `json:"duplicate_id" validate:"required,uuid"`
}

func (c *CRMExtRoutes) HandleMergeContacts(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, "crm")
		return
	}

	req, ok := decodeAndValidate[mergeContactsRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.MergeContacts(r.Context(), &crmv1.MergeContactsRequest{
		PrimaryId:   req.PrimaryID,
		DuplicateId: req.DuplicateID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp.Contact)
}

// ============================================================================
// Company Duplicate Detection
// ============================================================================

func (c *CRMExtRoutes) HandleFindCompanyDuplicates(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, "crm")
		return
	}

	resp, err := client.FindCompanyDuplicates(r.Context(), &crmv1.FindCompanyDuplicatesRequest{
		CompanyId: chi.URLParam(r, "id"),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"duplicates": resp.Duplicates,
		"total":      resp.Total,
	})
}

type mergeCompaniesRequest struct {
	PrimaryID   string `json:"primary_id" validate:"required,uuid"`
	DuplicateID string `json:"duplicate_id" validate:"required,uuid"`
}

func (c *CRMExtRoutes) HandleMergeCompanies(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, "crm")
		return
	}

	req, ok := decodeAndValidate[mergeCompaniesRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.MergeCompanies(r.Context(), &crmv1.MergeCompaniesRequest{
		PrimaryId:   req.PrimaryID,
		DuplicateId: req.DuplicateID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp.Company)
}

// ============================================================================
// Contact Timeline
// ============================================================================

func (c *CRMExtRoutes) HandleContactTimeline(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, "crm")
		return
	}

	offsetStr := r.URL.Query().Get("offset")
	limitStr := r.URL.Query().Get("limit")

	page := int32(1)
	pageSize := int32(20)
	if offsetStr != "" {
		if v, err := strconv.Atoi(offsetStr); err == nil && v >= 0 {
			page = int32(v)
		}
	}
	if limitStr != "" {
		if v, err := strconv.Atoi(limitStr); err == nil && v > 0 && v <= 100 {
			pageSize = int32(v)
		}
	}

	resp, err := client.GetContactTimeline(r.Context(), &crmv1.GetContactTimelineRequest{
		ContactId: chi.URLParam(r, "id"),
		Page:      page,
		PageSize:  pageSize,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

// ============================================================================
// Consent Management
// ============================================================================

func (c *CRMExtRoutes) HandleGetConsents(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, "crm")
		return
	}

	resp, err := client.GetContactConsents(r.Context(), &crmv1.GetContactConsentsRequest{
		ContactId: chi.URLParam(r, "id"),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp.Summary)
}

type grantConsentRequest struct {
	ConsentType string  `json:"consent_type" validate:"required"`
	Source      string  `json:"source" validate:"required"`
	LegalBasis  string  `json:"legal_basis" validate:"required"`
	IPAddress   *string `json:"ip_address,omitempty"`
}

func (c *CRMExtRoutes) HandleGrantConsent(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, "crm")
		return
	}

	req, ok := decodeAndValidate[grantConsentRequest](w, r)
	if !ok {
		return
	}

	protoReq := &crmv1.GrantConsentRequest{
		ContactId:   chi.URLParam(r, "id"),
		ConsentType: req.ConsentType,
		Source:      req.Source,
		LegalBasis:  req.LegalBasis,
		IpAddress:   req.IPAddress,
	}

	userIDStr := middleware.GetUserID(r.Context())
	if userIDStr != "" {
		protoReq.CreatedBy = &userIDStr
	}

	resp, err := client.GrantConsent(r.Context(), protoReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, resp.Record)
}

type revokeConsentRequest struct {
	Notes string `json:"notes"`
}

func (c *CRMExtRoutes) HandleRevokeConsent(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, "crm")
		return
	}

	req, ok := decodeAndValidate[revokeConsentRequest](w, r)
	if !ok {
		return
	}

	protoReq := &crmv1.RevokeConsentRequest{
		ContactId:   chi.URLParam(r, "id"),
		ConsentType: chi.URLParam(r, "type"),
		Notes:       req.Notes,
	}

	userIDStr := middleware.GetUserID(r.Context())
	if userIDStr != "" {
		protoReq.RevokedBy = &userIDStr
	}

	resp, err := client.RevokeConsent(r.Context(), protoReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp.Record)
}

func (c *CRMExtRoutes) HandleGetConsentHistory(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, "crm")
		return
	}

	resp, err := client.GetConsentHistory(r.Context(), &crmv1.GetConsentHistoryRequest{
		ContactId:   chi.URLParam(r, "id"),
		ConsentType: chi.URLParam(r, "type"),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"history": resp.History,
		"total":   resp.Total,
	})
}

type requestDeletionBody struct {
	Reason string `json:"reason"`
}

func (c *CRMExtRoutes) HandleRequestDeletion(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, "crm")
		return
	}

	req, ok := decodeAndValidate[requestDeletionBody](w, r)
	if !ok {
		return
	}

	protoReq := &crmv1.RequestDeletionRequest{
		ContactId: chi.URLParam(r, "id"),
		Reason:    req.Reason,
	}

	userIDStr := middleware.GetUserID(r.Context())
	if userIDStr != "" {
		protoReq.RequestedBy = &userIDStr
	}

	resp, err := client.RequestDeletion(r.Context(), protoReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, resp.DeletionRequest)
}

func (c *CRMExtRoutes) HandleProcessDeletion(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, "crm")
		return
	}

	resp, err := client.ProcessDeletion(r.Context(), &crmv1.ProcessDeletionRequest{
		RequestId: chi.URLParam(r, "id"),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": resp.Status})
}

// ============================================================================
// Route registration helpers (called from route_crm.go within existing blocks)
// ============================================================================

// registerContactExtRoutes adds extension routes into an existing /contacts subrouter.
func (c *CRMExtRoutes) registerContactExtRoutes(r chi.Router) {
	// Duplicate detection
	r.With(middleware.RequirePermission("contacts", "read")).Get("/{id}/duplicates", c.HandleFindContactDuplicates)
	r.With(middleware.RequirePermission("contacts", "write")).Post("/merge", c.HandleMergeContacts)

	// Timeline
	r.With(middleware.RequirePermission("contacts", "read")).Get("/{id}/timeline", c.HandleContactTimeline)

	// Consent management
	r.With(middleware.RequirePermission("contacts", "read")).Get("/{id}/consents", c.HandleGetConsents)
	r.With(middleware.RequirePermission("contacts", "write")).Post("/{id}/consents", c.HandleGrantConsent)
	r.With(middleware.RequirePermission("contacts", "write")).Delete("/{id}/consents/{type}", c.HandleRevokeConsent)
	r.With(middleware.RequirePermission("contacts", "read")).Get("/{id}/consents/{type}/history", c.HandleGetConsentHistory)
	r.With(middleware.RequirePermission("contacts", "write")).Post("/{id}/gdpr/deletion-request", c.HandleRequestDeletion)
}

// registerCompanyExtRoutes adds extension routes into an existing /companies subrouter.
func (c *CRMExtRoutes) registerCompanyExtRoutes(r chi.Router) {
	r.With(middleware.RequirePermission("companies", "read")).Get("/{id}/duplicates", c.HandleFindCompanyDuplicates)
	r.With(middleware.RequirePermission("companies", "write")).Post("/merge", c.HandleMergeCompanies)
}

// registerGDPRRoutes adds top-level GDPR routes to the root router.
func (c *CRMExtRoutes) registerGDPRRoutes(r chi.Router, authMiddleware func(http.Handler) http.Handler) {
	r.Route("/api/v1/gdpr/deletion-requests", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequireRole("admin")).Post("/{id}/process", c.HandleProcessDeletion)
	})
}
