package gateway

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/crm/activity"
	"github.com/kmuhub/kmuhub/internal/crm/company"
	"github.com/kmuhub/kmuhub/internal/crm/consent"
	"github.com/kmuhub/kmuhub/internal/crm/contact"
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server/response"
)

// CRMExtRoutes handles extended CRM routes with direct DB access.
// These routes implement features not available via the CRM gRPC service:
// duplicate detection, contact timeline, and GDPR consent management.
type CRMExtRoutes struct {
	contactSvc  *contact.Service
	companySvc  *company.Service
	activitySvc *activity.Service
	consentSvc  *consent.Service
}

// NewCRMExtRoutes creates CRMExtRoutes using direct DB connections.
func NewCRMExtRoutes(pool *pgxpool.Pool) *CRMExtRoutes {
	return &CRMExtRoutes{
		contactSvc:  contact.NewService(contact.NewPostgresRepository(pool)),
		companySvc:  company.NewService(company.NewPostgresRepository(pool)),
		activitySvc: activity.NewService(activity.NewPostgresRepository(pool)),
		consentSvc:  consent.NewService(consent.NewPostgresRepository(pool)),
	}
}

// ============================================================================
// Contact Duplicate Detection
// ============================================================================

func (c *CRMExtRoutes) HandleFindContactDuplicates(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid contact id")
		return
	}

	candidates, err := c.contactSvc.FindDuplicates(r.Context(), id)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"duplicates": candidates,
		"total":      len(candidates),
	})
}

type mergeContactsRequest struct {
	PrimaryID   string `json:"primary_id"`
	DuplicateID string `json:"duplicate_id"`
}

func (c *CRMExtRoutes) HandleMergeContacts(w http.ResponseWriter, r *http.Request) {
	var req mergeContactsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	primaryID, err := uuid.Parse(req.PrimaryID)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid primary_id")
		return
	}
	duplicateID, err := uuid.Parse(req.DuplicateID)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid duplicate_id")
		return
	}

	merged, err := c.contactSvc.MergeContacts(r.Context(), primaryID, duplicateID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	response.JSON(w, http.StatusOK, merged)
}

// ============================================================================
// Company Duplicate Detection
// ============================================================================

func (c *CRMExtRoutes) HandleFindCompanyDuplicates(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid company id")
		return
	}

	candidates, err := c.companySvc.FindDuplicates(r.Context(), id)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"duplicates": candidates,
		"total":      len(candidates),
	})
}

type mergeCompaniesRequest struct {
	PrimaryID   string `json:"primary_id"`
	DuplicateID string `json:"duplicate_id"`
}

func (c *CRMExtRoutes) HandleMergeCompanies(w http.ResponseWriter, r *http.Request) {
	var req mergeCompaniesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	primaryID, err := uuid.Parse(req.PrimaryID)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid primary_id")
		return
	}
	duplicateID, err := uuid.Parse(req.DuplicateID)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid duplicate_id")
		return
	}

	merged, err := c.companySvc.MergeCompanies(r.Context(), primaryID, duplicateID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	response.JSON(w, http.StatusOK, merged)
}

// ============================================================================
// Contact Timeline
// ============================================================================

func (c *CRMExtRoutes) HandleContactTimeline(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid contact id")
		return
	}

	offsetStr := r.URL.Query().Get("offset")
	limitStr := r.URL.Query().Get("limit")

	offset := 0
	limit := 20
	if offsetStr != "" {
		if v, err := strconv.Atoi(offsetStr); err == nil && v >= 0 {
			offset = v
		}
	}
	if limitStr != "" {
		if v, err := strconv.Atoi(limitStr); err == nil && v > 0 && v <= 100 {
			limit = v
		}
	}

	result, err := c.activitySvc.GetContactTimeline(r.Context(), id, offset, limit)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	response.JSON(w, http.StatusOK, result)
}

// ============================================================================
// Consent Management
// ============================================================================

func (c *CRMExtRoutes) HandleGetConsents(w http.ResponseWriter, r *http.Request) {
	contactID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid contact id")
		return
	}

	summary, err := c.consentSvc.GetContactConsents(r.Context(), contactID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	response.JSON(w, http.StatusOK, summary)
}

type grantConsentRequest struct {
	ConsentType string  `json:"consent_type"`
	Source      string  `json:"source"`
	LegalBasis  string  `json:"legal_basis"`
	IPAddress   *string `json:"ip_address,omitempty"`
}

func (c *CRMExtRoutes) HandleGrantConsent(w http.ResponseWriter, r *http.Request) {
	contactID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid contact id")
		return
	}

	var req grantConsentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	userIDStr := middleware.GetUserID(r.Context())
	var createdBy *uuid.UUID
	if uid, err := uuid.Parse(userIDStr); err == nil {
		createdBy = &uid
	}

	record, err := c.consentSvc.GrantConsent(r.Context(), contactID, req.ConsentType, req.Source, req.LegalBasis, req.IPAddress, createdBy)
	if err != nil {
		if errors.Is(err, consent.ErrInvalidConsentType) || errors.Is(err, consent.ErrInvalidLegalBasis) {
			response.Error(w, http.StatusBadRequest, err.Error())
			return
		}
		if errors.Is(err, consent.ErrContactNotFound) {
			response.Error(w, http.StatusNotFound, err.Error())
			return
		}
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	response.JSON(w, http.StatusCreated, record)
}

type revokeConsentRequest struct {
	Notes string `json:"notes"`
}

func (c *CRMExtRoutes) HandleRevokeConsent(w http.ResponseWriter, r *http.Request) {
	contactID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid contact id")
		return
	}

	consentType := chi.URLParam(r, "type")

	var req revokeConsentRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	userIDStr := middleware.GetUserID(r.Context())
	var revokedBy *uuid.UUID
	if uid, err := uuid.Parse(userIDStr); err == nil {
		revokedBy = &uid
	}

	record, err := c.consentSvc.RevokeConsent(r.Context(), contactID, consentType, req.Notes, revokedBy)
	if err != nil {
		if errors.Is(err, consent.ErrInvalidConsentType) {
			response.Error(w, http.StatusBadRequest, err.Error())
			return
		}
		if errors.Is(err, consent.ErrContactNotFound) {
			response.Error(w, http.StatusNotFound, err.Error())
			return
		}
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	response.JSON(w, http.StatusOK, record)
}

func (c *CRMExtRoutes) HandleGetConsentHistory(w http.ResponseWriter, r *http.Request) {
	contactID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid contact id")
		return
	}

	consentType := chi.URLParam(r, "type")

	records, err := c.consentSvc.GetConsentHistory(r.Context(), contactID, consentType)
	if err != nil {
		if errors.Is(err, consent.ErrInvalidConsentType) {
			response.Error(w, http.StatusBadRequest, err.Error())
			return
		}
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"history": records,
		"total":   len(records),
	})
}

type requestDeletionBody struct {
	Reason string `json:"reason"`
}

func (c *CRMExtRoutes) HandleRequestDeletion(w http.ResponseWriter, r *http.Request) {
	contactID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid contact id")
		return
	}

	var req requestDeletionBody
	_ = json.NewDecoder(r.Body).Decode(&req)

	userIDStr := middleware.GetUserID(r.Context())
	var requestedBy *uuid.UUID
	if uid, err := uuid.Parse(userIDStr); err == nil {
		requestedBy = &uid
	}

	deletionReq, err := c.consentSvc.RequestDeletion(r.Context(), contactID, req.Reason, requestedBy)
	if err != nil {
		if errors.Is(err, consent.ErrContactNotFound) {
			response.Error(w, http.StatusNotFound, err.Error())
			return
		}
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	response.JSON(w, http.StatusCreated, deletionReq)
}

func (c *CRMExtRoutes) HandleProcessDeletion(w http.ResponseWriter, r *http.Request) {
	requestID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request id")
		return
	}

	if err := c.consentSvc.ProcessDeletion(r.Context(), requestID); err != nil {
		if errors.Is(err, consent.ErrDeletionRequestNotFound) {
			response.Error(w, http.StatusNotFound, err.Error())
			return
		}
		if errors.Is(err, consent.ErrDeletionAlreadyComplete) {
			response.Error(w, http.StatusConflict, err.Error())
			return
		}
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "completed"})
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
