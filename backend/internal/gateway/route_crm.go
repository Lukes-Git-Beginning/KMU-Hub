package gateway

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server/response"
	crmv1 "github.com/kmuhub/kmuhub/proto/crm/v1"
)

// CRMRoutes handles HTTP routes for the CRM backend service.
type CRMRoutes struct {
	registry *ServiceRegistry
	ext      *CRMExtRoutes // optional: direct-DB extension handlers
}

// NewCRMRoutes creates a new CRMRoutes with the given service registry.
// ext may be nil if extension features are not configured.
func NewCRMRoutes(registry *ServiceRegistry, ext *CRMExtRoutes) *CRMRoutes {
	return &CRMRoutes{registry: registry, ext: ext}
}

// ServiceName returns the backend service name.
func (c *CRMRoutes) ServiceName() string { return "crm" }

// getCRMClient lazily obtains a gRPC client for the CRM service.
func (c *CRMRoutes) getCRMClient() (crmv1.CRMServiceClient, error) {
	conn, err := c.registry.GetConnection("crm")
	if err != nil {
		return nil, err
	}
	return crmv1.NewCRMServiceClient(conn), nil
}

// RegisterRoutes registers all CRM HTTP routes.
func (c *CRMRoutes) RegisterRoutes(r chi.Router, authMiddleware func(http.Handler) http.Handler) {
	// Custom Fields
	r.Route("/api/v1/custom-fields", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("custom_fields", "read")).Get("/", c.HandleListCustomFields)
		r.With(middleware.RequirePermission("custom_fields", "read")).Get("/{id}", c.HandleGetCustomField)
		r.With(middleware.RequirePermission("custom_fields", "write")).Post("/", c.HandleCreateCustomField)
		r.With(middleware.RequirePermission("custom_fields", "write")).Put("/{id}", c.HandleUpdateCustomField)
		r.With(middleware.RequirePermission("custom_fields", "delete")).Delete("/{id}", c.HandleDeleteCustomField)
	})

	// Tags
	r.Route("/api/v1/tags", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("tags", "read")).Get("/", c.HandleListTags)
		r.With(middleware.RequirePermission("tags", "read")).Get("/{id}", c.HandleGetTag)
		r.With(middleware.RequirePermission("tags", "write")).Post("/", c.HandleCreateTag)
		r.With(middleware.RequirePermission("tags", "write")).Put("/{id}", c.HandleUpdateTag)
		r.With(middleware.RequirePermission("tags", "delete")).Delete("/{id}", c.HandleDeleteTag)
	})

	// Contacts
	r.Route("/api/v1/contacts", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("contacts", "read")).Get("/", c.HandleListContacts)
		r.With(middleware.RequirePermission("contacts", "read")).Get("/{id}", c.HandleGetContact)
		r.With(middleware.RequirePermission("contacts", "write")).Post("/", c.HandleCreateContact)
		r.With(middleware.RequirePermission("contacts", "write")).Put("/{id}", c.HandleUpdateContact)
		r.With(middleware.RequirePermission("contacts", "delete")).Delete("/{id}", c.HandleDeleteContact)
		r.With(middleware.RequirePermission("contacts", "write")).Post("/{id}/tags", c.HandleAddContactTags)
		r.With(middleware.RequirePermission("contacts", "write")).Delete("/{id}/tags", c.HandleRemoveContactTags)
		r.With(middleware.RequirePermission("contacts", "write")).Put("/{id}/visibility", c.HandleUpdateContactVisibility)

		// Import/Export
		r.With(middleware.RequirePermission("contacts", "write")).Post("/import/csv", c.HandleImportContactsCSV)
		r.With(middleware.RequirePermission("contacts", "write")).Post("/import/vcard", c.HandleImportContactsVCard)
		r.With(middleware.RequirePermission("contacts", "write")).Post("/import/preview", c.HandlePreviewImportCSV)
		r.With(middleware.RequirePermission("contacts", "read")).Post("/export/csv", c.HandleExportContactsCSV)
		r.With(middleware.RequirePermission("contacts", "read")).Post("/export/vcard", c.HandleExportContactsVCard)

		// Extension features: duplicate detection, timeline, consent management
		if c.ext != nil {
			c.ext.registerContactExtRoutes(r)
		}
	})

	// Companies
	r.Route("/api/v1/companies", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("companies", "read")).Get("/", c.HandleListCompanies)
		r.With(middleware.RequirePermission("companies", "read")).Get("/{id}", c.HandleGetCompany)
		r.With(middleware.RequirePermission("companies", "read")).Get("/{id}/contacts", c.HandleGetCompanyContacts)
		r.With(middleware.RequirePermission("companies", "write")).Post("/", c.HandleCreateCompany)
		r.With(middleware.RequirePermission("companies", "write")).Put("/{id}", c.HandleUpdateCompany)
		r.With(middleware.RequirePermission("companies", "delete")).Delete("/{id}", c.HandleDeleteCompany)

		// Extension features: duplicate detection
		if c.ext != nil {
			c.ext.registerCompanyExtRoutes(r)
		}
	})

	// Pipeline Stages
	r.Route("/api/v1/pipeline-stages", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("pipeline_stages", "read")).Get("/", c.HandleListPipelineStages)
		r.With(middleware.RequirePermission("pipeline_stages", "read")).Get("/{id}", c.HandleGetPipelineStage)
		r.With(middleware.RequirePermission("pipeline_stages", "write")).Post("/", c.HandleCreatePipelineStage)
		r.With(middleware.RequirePermission("pipeline_stages", "write")).Put("/{id}", c.HandleUpdatePipelineStage)
		r.With(middleware.RequirePermission("pipeline_stages", "write")).Post("/reorder", c.HandleReorderPipelineStages)
		r.With(middleware.RequirePermission("pipeline_stages", "delete")).Delete("/{id}", c.HandleDeletePipelineStage)
	})

	// Deals
	r.Route("/api/v1/deals", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("deals", "read")).Get("/", c.HandleListDeals)
		r.With(middleware.RequirePermission("deals", "read")).Get("/{id}", c.HandleGetDeal)
		r.With(middleware.RequirePermission("deals", "write")).Post("/", c.HandleCreateDeal)
		r.With(middleware.RequirePermission("deals", "write")).Put("/{id}", c.HandleUpdateDeal)
		r.With(middleware.RequirePermission("deals", "write")).Post("/{id}/stage", c.HandleMoveDealToStage)
		r.With(middleware.RequirePermission("deals", "write")).Post("/{id}/tags", c.HandleAddDealTags)
		r.With(middleware.RequirePermission("deals", "write")).Delete("/{id}/tags", c.HandleRemoveDealTags)
		r.With(middleware.RequirePermission("deals", "delete")).Delete("/{id}", c.HandleDeleteDeal)
	})

	// Activities
	r.Route("/api/v1/activities", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("activities", "read")).Get("/", c.HandleListActivities)
		r.With(middleware.RequirePermission("activities", "read")).Get("/{id}", c.HandleGetActivity)
		r.With(middleware.RequirePermission("activities", "write")).Post("/", c.HandleCreateActivity)
		r.With(middleware.RequirePermission("activities", "write")).Put("/{id}", c.HandleUpdateActivity)
		r.With(middleware.RequirePermission("activities", "write")).Post("/{id}/complete", c.HandleCompleteActivity)
		r.With(middleware.RequirePermission("activities", "write")).Post("/{id}/tags", c.HandleAddActivityTags)
		r.With(middleware.RequirePermission("activities", "write")).Delete("/{id}/tags", c.HandleRemoveActivityTags)
		r.With(middleware.RequirePermission("activities", "delete")).Delete("/{id}", c.HandleDeleteActivity)
	})

	// Search
	r.Route("/api/v1/search", func(r chi.Router) {
		r.Use(authMiddleware)
		r.Get("/", c.HandleSearch)
	})

	// Saved Filters
	r.Route("/api/v1/saved-filters", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("saved_filters", "read")).Get("/", c.HandleListSavedFilters)
		r.With(middleware.RequirePermission("saved_filters", "read")).Get("/{id}", c.HandleGetSavedFilter)
		r.With(middleware.RequirePermission("saved_filters", "write")).Post("/", c.HandleCreateSavedFilter)
		r.With(middleware.RequirePermission("saved_filters", "write")).Patch("/{id}", c.HandleUpdateSavedFilter)
		r.With(middleware.RequirePermission("saved_filters", "delete")).Delete("/{id}", c.HandleDeleteSavedFilter)
	})

	// Reports
	r.Route("/api/v1/reports", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("reports", "read")).Get("/pipeline", c.HandleGetPipelineReport)
		r.With(middleware.RequirePermission("reports", "read")).Get("/conversion", c.HandleGetConversionReport)
		r.With(middleware.RequirePermission("reports", "read")).Get("/activities", c.HandleGetActivityReport)
	})

	// GDPR deletion requests (top-level, admin only)
	if c.ext != nil {
		c.ext.registerGDPRRoutes(r, authMiddleware)
	}
}

// ============================================================================
// Custom Fields Handlers
// ============================================================================

type createCustomFieldRequest struct {
	EntityType   string   `json:"entity_type"`
	FieldName    string   `json:"field_name"`
	FieldLabel   string   `json:"field_label"`
	FieldType    string   `json:"field_type"`
	Options      []string `json:"options,omitempty"`
	DefaultValue *string  `json:"default_value,omitempty"`
	IsRequired   bool     `json:"is_required"`
	SortOrder    int      `json:"sort_order"`
}

func (c *CRMRoutes) HandleCreateCustomField(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, c.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	var req createCustomFieldRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.EntityType == "" || req.FieldName == "" || req.FieldLabel == "" || req.FieldType == "" {
		response.Error(w, http.StatusBadRequest, "entity_type, field_name, field_label, and field_type are required")
		return
	}

	grpcReq := &crmv1.CreateCustomFieldRequest{
		EntityType: req.EntityType,
		FieldName:  req.FieldName,
		FieldLabel: req.FieldLabel,
		FieldType:  req.FieldType,
		Options:    req.Options,
		IsRequired: req.IsRequired,
		SortOrder:  int32(req.SortOrder),
		CreatedBy:  userID,
	}
	if req.DefaultValue != nil {
		grpcReq.DefaultValue = req.DefaultValue
	}

	resp, err := client.CreateCustomField(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, resp)
}

func (c *CRMRoutes) HandleGetCustomField(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, c.ServiceName())
		return
	}

	fieldID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.GetCustomField(r.Context(), &crmv1.GetCustomFieldRequest{Id: fieldID})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (c *CRMRoutes) HandleListCustomFields(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, c.ServiceName())
		return
	}

	entityType := r.URL.Query().Get("entity_type")
	page, pageSize := parsePagination(r, 1, 20)

	resp, err := client.ListCustomFields(r.Context(), &crmv1.ListCustomFieldsRequest{
		EntityType: entityType,
		Page:       int32(page),
		PageSize:   int32(pageSize),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

type updateCustomFieldRequest struct {
	FieldLabel   *string  `json:"field_label,omitempty"`
	Options      []string `json:"options,omitempty"`
	DefaultValue *string  `json:"default_value,omitempty"`
	IsRequired   *bool    `json:"is_required,omitempty"`
	SortOrder    *int32   `json:"sort_order,omitempty"`
}

func (c *CRMRoutes) HandleUpdateCustomField(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, c.ServiceName())
		return
	}

	fieldID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	var req updateCustomFieldRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	grpcReq := &crmv1.UpdateCustomFieldRequest{
		Id:      fieldID,
		Options: req.Options,
	}
	if req.FieldLabel != nil {
		grpcReq.FieldLabel = req.FieldLabel
	}
	if req.DefaultValue != nil {
		grpcReq.DefaultValue = req.DefaultValue
	}
	if req.IsRequired != nil {
		grpcReq.IsRequired = req.IsRequired
	}
	if req.SortOrder != nil {
		grpcReq.SortOrder = req.SortOrder
	}

	resp, err := client.UpdateCustomField(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (c *CRMRoutes) HandleDeleteCustomField(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, c.ServiceName())
		return
	}

	fieldID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	_, err = client.DeleteCustomField(r.Context(), &crmv1.DeleteCustomFieldRequest{Id: fieldID})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "custom field deleted"})
}

// ============================================================================
// Tags Handlers
// ============================================================================

type createTagRequest struct {
	Name       string `json:"name"`
	Color      string `json:"color"`
	EntityType string `json:"entity_type"`
}

func (c *CRMRoutes) HandleCreateTag(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, c.ServiceName())
		return
	}

	var req createTagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" || req.EntityType == "" {
		response.Error(w, http.StatusBadRequest, "name and entity_type are required")
		return
	}

	resp, err := client.CreateTag(r.Context(), &crmv1.CreateTagRequest{
		Name:       req.Name,
		Color:      req.Color,
		EntityType: req.EntityType,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, resp)
}

func (c *CRMRoutes) HandleGetTag(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, c.ServiceName())
		return
	}

	tagID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.GetTag(r.Context(), &crmv1.GetTagRequest{Id: tagID})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (c *CRMRoutes) HandleListTags(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, c.ServiceName())
		return
	}

	entityType := r.URL.Query().Get("entity_type")
	page, pageSize := parsePagination(r, 1, 50)

	resp, err := client.ListTags(r.Context(), &crmv1.ListTagsRequest{
		EntityType: entityType,
		Page:       int32(page),
		PageSize:   int32(pageSize),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

type updateTagRequest struct {
	Name  *string `json:"name,omitempty"`
	Color *string `json:"color,omitempty"`
}

func (c *CRMRoutes) HandleUpdateTag(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, c.ServiceName())
		return
	}

	tagID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	var req updateTagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	grpcReq := &crmv1.UpdateTagRequest{Id: tagID}
	if req.Name != nil {
		grpcReq.Name = req.Name
	}
	if req.Color != nil {
		grpcReq.Color = req.Color
	}

	resp, err := client.UpdateTag(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (c *CRMRoutes) HandleDeleteTag(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, c.ServiceName())
		return
	}

	tagID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	_, err = client.DeleteTag(r.Context(), &crmv1.DeleteTagRequest{Id: tagID})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "tag deleted"})
}

// ============================================================================
// Contacts Handlers
// ============================================================================

type customFieldValueRequest struct {
	FieldID string `json:"field_id"`
	Value   string `json:"value"`
}

type createContactRequest struct {
	FirstName    string                    `json:"first_name"`
	LastName     string                    `json:"last_name"`
	Email        string                    `json:"email"`
	Phone        string                    `json:"phone"`
	CompanyID    *string                   `json:"company_id,omitempty"`
	Position     string                    `json:"position"`
	Notes        string                    `json:"notes"`
	TagIDs       []string                  `json:"tag_ids"`
	CustomFields []customFieldValueRequest `json:"custom_fields"`
}

func (c *CRMRoutes) HandleCreateContact(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, c.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	var req createContactRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.FirstName == "" || req.LastName == "" {
		response.Error(w, http.StatusBadRequest, "first_name and last_name are required")
		return
	}

	grpcReq := &crmv1.CreateContactRequest{
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Email:     req.Email,
		Phone:     req.Phone,
		Position:  req.Position,
		Notes:     req.Notes,
		TagIds:    req.TagIDs,
		CreatedBy: userID,
	}

	if req.CompanyID != nil {
		grpcReq.CompanyId = req.CompanyID
	}

	for _, cf := range req.CustomFields {
		grpcReq.CustomFields = append(grpcReq.CustomFields, &crmv1.CustomFieldValueInput{
			FieldId: cf.FieldID,
			Value:   cf.Value,
		})
	}

	resp, err := client.CreateContact(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, resp)
}

func (c *CRMRoutes) HandleGetContact(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, c.ServiceName())
		return
	}

	contactID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.GetContact(r.Context(), &crmv1.GetContactRequest{Id: contactID})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (c *CRMRoutes) HandleListContacts(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, c.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())
	isAdmin := middleware.IsAdmin(r.Context())
	companyID := r.URL.Query().Get("company_id")
	search := r.URL.Query().Get("search")
	sortBy := r.URL.Query().Get("sort_by")
	sortDesc := r.URL.Query().Get("sort_desc") == "true"
	visibilityFilter := r.URL.Query().Get("visibility")
	tagIDs := r.URL.Query()["tag_ids"]
	page, pageSize := parsePagination(r, 1, 20)

	grpcReq := &crmv1.ListContactsRequest{
		Search:           search,
		Page:             int32(page),
		PageSize:         int32(pageSize),
		SortBy:           sortBy,
		SortDesc:         sortDesc,
		TagIds:           tagIDs,
		UserId:           userID,
		IsAdmin:          isAdmin,
		VisibilityFilter: visibilityFilter,
	}

	if companyID != "" {
		grpcReq.CompanyId = &companyID
	}

	resp, err := client.ListContacts(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

type updateContactRequest struct {
	FirstName    *string                   `json:"first_name,omitempty"`
	LastName     *string                   `json:"last_name,omitempty"`
	Email        *string                   `json:"email,omitempty"`
	Phone        *string                   `json:"phone,omitempty"`
	CompanyID    *string                   `json:"company_id,omitempty"`
	Position     *string                   `json:"position,omitempty"`
	Notes        *string                   `json:"notes,omitempty"`
	CustomFields []customFieldValueRequest `json:"custom_fields"`
}

func (c *CRMRoutes) HandleUpdateContact(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, c.ServiceName())
		return
	}

	contactID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	var req updateContactRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	grpcReq := &crmv1.UpdateContactRequest{Id: contactID}
	if req.FirstName != nil {
		grpcReq.FirstName = req.FirstName
	}
	if req.LastName != nil {
		grpcReq.LastName = req.LastName
	}
	if req.Email != nil {
		grpcReq.Email = req.Email
	}
	if req.Phone != nil {
		grpcReq.Phone = req.Phone
	}
	if req.CompanyID != nil {
		grpcReq.CompanyId = req.CompanyID
	}
	if req.Position != nil {
		grpcReq.Position = req.Position
	}
	if req.Notes != nil {
		grpcReq.Notes = req.Notes
	}
	for _, cf := range req.CustomFields {
		grpcReq.CustomFields = append(grpcReq.CustomFields, &crmv1.CustomFieldValueInput{
			FieldId: cf.FieldID,
			Value:   cf.Value,
		})
	}

	resp, err := client.UpdateContact(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (c *CRMRoutes) HandleDeleteContact(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, c.ServiceName())
		return
	}

	contactID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	_, err = client.DeleteContact(r.Context(), &crmv1.DeleteContactRequest{Id: contactID})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "contact deleted"})
}

type modifyContactTagsRequest struct {
	TagIDs []string `json:"tag_ids"`
}

func (c *CRMRoutes) HandleAddContactTags(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, c.ServiceName())
		return
	}

	contactID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	var req modifyContactTagsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := client.AddContactTags(r.Context(), &crmv1.AddContactTagsRequest{
		ContactId: contactID,
		TagIds:    req.TagIDs,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (c *CRMRoutes) HandleRemoveContactTags(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, c.ServiceName())
		return
	}

	contactID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	var req modifyContactTagsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := client.RemoveContactTags(r.Context(), &crmv1.RemoveContactTagsRequest{
		ContactId: contactID,
		TagIds:    req.TagIDs,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

// ============================================================================
// Contact Import/Export Handlers
// ============================================================================

func (c *CRMRoutes) HandleImportContactsCSV(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, c.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	if parseErr := r.ParseMultipartForm(10 << 20); parseErr != nil { // 10MB max
		response.Error(w, http.StatusBadRequest, "invalid multipart form")
		return
	}

	file, _, fileErr := r.FormFile("file")
	if fileErr != nil {
		response.Error(w, http.StatusBadRequest, "file is required")
		return
	}
	defer file.Close()

	var buf bytes.Buffer
	if _, copyErr := buf.ReadFrom(file); copyErr != nil {
		response.Error(w, http.StatusBadRequest, "failed to read file")
		return
	}

	// Parse field mapping from form values
	fieldMapping := make(map[string]string)
	for key, values := range r.MultipartForm.Value {
		if len(key) > 4 && key[:4] == "map_" {
			fieldMapping[key[4:]] = values[0]
		}
	}

	visibility := r.FormValue("visibility")
	if visibility == "" {
		visibility = "shared"
	}
	mergeByEmail := r.FormValue("merge_by_email") == "true"

	resp, err := client.ImportContactsCSV(r.Context(), &crmv1.ImportContactsCSVRequest{
		FileContent:  buf.Bytes(),
		FieldMapping: fieldMapping,
		Visibility:   visibility,
		MergeByEmail: mergeByEmail,
		UserId:       userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (c *CRMRoutes) HandleImportContactsVCard(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, c.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	if parseErr := r.ParseMultipartForm(10 << 20); parseErr != nil {
		response.Error(w, http.StatusBadRequest, "invalid multipart form")
		return
	}

	file, _, fileErr := r.FormFile("file")
	if fileErr != nil {
		response.Error(w, http.StatusBadRequest, "file is required")
		return
	}
	defer file.Close()

	var buf bytes.Buffer
	if _, copyErr := buf.ReadFrom(file); copyErr != nil {
		response.Error(w, http.StatusBadRequest, "failed to read file")
		return
	}

	visibility := r.FormValue("visibility")
	if visibility == "" {
		visibility = "shared"
	}
	mergeByEmail := r.FormValue("merge_by_email") == "true"

	resp, err := client.ImportContactsVCard(r.Context(), &crmv1.ImportContactsVCardRequest{
		FileContent:  buf.Bytes(),
		Visibility:   visibility,
		MergeByEmail: mergeByEmail,
		UserId:       userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (c *CRMRoutes) HandlePreviewImportCSV(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, c.ServiceName())
		return
	}

	if parseErr := r.ParseMultipartForm(10 << 20); parseErr != nil {
		response.Error(w, http.StatusBadRequest, "invalid multipart form")
		return
	}

	file, _, fileErr := r.FormFile("file")
	if fileErr != nil {
		response.Error(w, http.StatusBadRequest, "file is required")
		return
	}
	defer file.Close()

	var buf bytes.Buffer
	if _, copyErr := buf.ReadFrom(file); copyErr != nil {
		response.Error(w, http.StatusBadRequest, "failed to read file")
		return
	}

	resp, err := client.PreviewImportCSV(r.Context(), &crmv1.PreviewImportCSVRequest{
		FileContent: buf.Bytes(),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

type exportContactsCSVRequest struct {
	ContactIDs []string `json:"contact_ids"`
	Fields     []string `json:"fields"`
}

func (c *CRMRoutes) HandleExportContactsCSV(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, c.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())
	isAdmin := middleware.IsAdmin(r.Context())

	var req exportContactsCSVRequest
	if decErr := json.NewDecoder(r.Body).Decode(&req); decErr != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := client.ExportContactsCSV(r.Context(), &crmv1.ExportContactsCSVRequest{
		ContactIds: req.ContactIDs,
		Fields:     req.Fields,
		UserId:     userID,
		IsAdmin:    isAdmin,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	w.Header().Set("Content-Type", resp.ContentType)
	w.Header().Set("Content-Disposition", "attachment; filename="+resp.Filename)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(resp.FileContent)
}

type exportContactsVCardRequest struct {
	ContactIDs []string `json:"contact_ids"`
}

func (c *CRMRoutes) HandleExportContactsVCard(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, c.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())
	isAdmin := middleware.IsAdmin(r.Context())

	var req exportContactsVCardRequest
	if decErr := json.NewDecoder(r.Body).Decode(&req); decErr != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := client.ExportContactsVCard(r.Context(), &crmv1.ExportContactsVCardRequest{
		ContactIds: req.ContactIDs,
		UserId:     userID,
		IsAdmin:    isAdmin,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	w.Header().Set("Content-Type", resp.ContentType)
	w.Header().Set("Content-Disposition", "attachment; filename="+resp.Filename)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(resp.FileContent)
}

type updateContactVisibilityRequest struct {
	Visibility string `json:"visibility"`
}

func (c *CRMRoutes) HandleUpdateContactVisibility(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, c.ServiceName())
		return
	}

	contactID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	userID := middleware.GetUserID(r.Context())
	isAdmin := middleware.IsAdmin(r.Context())

	var req updateContactVisibilityRequest
	if decErr := json.NewDecoder(r.Body).Decode(&req); decErr != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Visibility != "shared" && req.Visibility != "personal" {
		response.Error(w, http.StatusBadRequest, "visibility must be 'shared' or 'personal'")
		return
	}

	resp, err := client.UpdateContactVisibility(r.Context(), &crmv1.UpdateContactVisibilityRequest{
		ContactId:  contactID,
		Visibility: req.Visibility,
		UserId:     userID,
		IsAdmin:    isAdmin,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

// ============================================================================
// Companies Handlers
// ============================================================================

type createCompanyRequest struct {
	Name          string                    `json:"name"`
	Domain        string                    `json:"domain"`
	Industry      string                    `json:"industry"`
	EmployeeCount int32                     `json:"employee_count"`
	Address       string                    `json:"address"`
	City          string                    `json:"city"`
	Country       string                    `json:"country"`
	Notes         string                    `json:"notes"`
	TagIDs        []string                  `json:"tag_ids"`
	CustomFields  []customFieldValueRequest `json:"custom_fields"`
}

func (c *CRMRoutes) HandleCreateCompany(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, c.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	var req createCompanyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		response.Error(w, http.StatusBadRequest, "name is required")
		return
	}

	grpcReq := &crmv1.CreateCompanyRequest{
		Name:          req.Name,
		Domain:        req.Domain,
		Industry:      req.Industry,
		EmployeeCount: req.EmployeeCount,
		Address:       req.Address,
		City:          req.City,
		Country:       req.Country,
		Notes:         req.Notes,
		TagIds:        req.TagIDs,
		CreatedBy:     userID,
	}

	for _, cf := range req.CustomFields {
		grpcReq.CustomFields = append(grpcReq.CustomFields, &crmv1.CustomFieldValueInput{
			FieldId: cf.FieldID,
			Value:   cf.Value,
		})
	}

	resp, err := client.CreateCompany(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, resp)
}

func (c *CRMRoutes) HandleGetCompany(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, c.ServiceName())
		return
	}

	companyID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.GetCompany(r.Context(), &crmv1.GetCompanyRequest{Id: companyID})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (c *CRMRoutes) HandleListCompanies(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, c.ServiceName())
		return
	}

	industry := r.URL.Query().Get("industry")
	search := r.URL.Query().Get("search")
	sortBy := r.URL.Query().Get("sort_by")
	sortDesc := r.URL.Query().Get("sort_desc") == "true"
	tagIDs := r.URL.Query()["tag_ids"]
	page, pageSize := parsePagination(r, 1, 20)

	resp, err := client.ListCompanies(r.Context(), &crmv1.ListCompaniesRequest{
		Industry: industry,
		Search:   search,
		Page:     int32(page),
		PageSize: int32(pageSize),
		SortBy:   sortBy,
		SortDesc: sortDesc,
		TagIds:   tagIDs,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

type updateCompanyRequest struct {
	Name          *string                   `json:"name,omitempty"`
	Domain        *string                   `json:"domain,omitempty"`
	Industry      *string                   `json:"industry,omitempty"`
	EmployeeCount *int32                    `json:"employee_count,omitempty"`
	Address       *string                   `json:"address,omitempty"`
	City          *string                   `json:"city,omitempty"`
	Country       *string                   `json:"country,omitempty"`
	Notes         *string                   `json:"notes,omitempty"`
	CustomFields  []customFieldValueRequest `json:"custom_fields"`
}

func (c *CRMRoutes) HandleUpdateCompany(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, c.ServiceName())
		return
	}

	companyID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	var req updateCompanyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	grpcReq := &crmv1.UpdateCompanyRequest{Id: companyID}
	if req.Name != nil {
		grpcReq.Name = req.Name
	}
	if req.Domain != nil {
		grpcReq.Domain = req.Domain
	}
	if req.Industry != nil {
		grpcReq.Industry = req.Industry
	}
	if req.EmployeeCount != nil {
		grpcReq.EmployeeCount = req.EmployeeCount
	}
	if req.Address != nil {
		grpcReq.Address = req.Address
	}
	if req.City != nil {
		grpcReq.City = req.City
	}
	if req.Country != nil {
		grpcReq.Country = req.Country
	}
	if req.Notes != nil {
		grpcReq.Notes = req.Notes
	}
	for _, cf := range req.CustomFields {
		grpcReq.CustomFields = append(grpcReq.CustomFields, &crmv1.CustomFieldValueInput{
			FieldId: cf.FieldID,
			Value:   cf.Value,
		})
	}

	resp, err := client.UpdateCompany(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (c *CRMRoutes) HandleDeleteCompany(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, c.ServiceName())
		return
	}

	companyID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	_, err = client.DeleteCompany(r.Context(), &crmv1.DeleteCompanyRequest{Id: companyID})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "company deleted"})
}

func (c *CRMRoutes) HandleGetCompanyContacts(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, c.ServiceName())
		return
	}

	companyID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	page, pageSize := parsePagination(r, 1, 20)

	resp, err := client.GetCompanyContacts(r.Context(), &crmv1.GetCompanyContactsRequest{
		CompanyId: companyID,
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
// Pipeline Stages Handlers
// ============================================================================

type createPipelineStageRequest struct {
	Name        string  `json:"name"`
	Color       string  `json:"color"`
	IsWon       bool    `json:"is_won"`
	IsLost      bool    `json:"is_lost"`
	Probability float64 `json:"probability"`
}

func (c *CRMRoutes) HandleCreatePipelineStage(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, c.ServiceName())
		return
	}

	var req createPipelineStageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		response.Error(w, http.StatusBadRequest, "name is required")
		return
	}

	resp, err := client.CreatePipelineStage(r.Context(), &crmv1.CreatePipelineStageRequest{
		Name:        req.Name,
		Color:       req.Color,
		IsWon:       req.IsWon,
		IsLost:      req.IsLost,
		Probability: req.Probability,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, resp)
}

func (c *CRMRoutes) HandleGetPipelineStage(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, c.ServiceName())
		return
	}

	stageID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.GetPipelineStage(r.Context(), &crmv1.GetPipelineStageRequest{Id: stageID})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (c *CRMRoutes) HandleListPipelineStages(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, c.ServiceName())
		return
	}

	resp, err := client.ListPipelineStages(r.Context(), &crmv1.ListPipelineStagesRequest{})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

type updatePipelineStageRequest struct {
	Name        *string  `json:"name,omitempty"`
	Color       *string  `json:"color,omitempty"`
	IsWon       *bool    `json:"is_won,omitempty"`
	IsLost      *bool    `json:"is_lost,omitempty"`
	Probability *float64 `json:"probability,omitempty"`
}

func (c *CRMRoutes) HandleUpdatePipelineStage(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, c.ServiceName())
		return
	}

	stageID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	var req updatePipelineStageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	grpcReq := &crmv1.UpdatePipelineStageRequest{Id: stageID}
	if req.Name != nil {
		grpcReq.Name = req.Name
	}
	if req.Color != nil {
		grpcReq.Color = req.Color
	}
	if req.IsWon != nil {
		grpcReq.IsWon = req.IsWon
	}
	if req.IsLost != nil {
		grpcReq.IsLost = req.IsLost
	}
	if req.Probability != nil {
		grpcReq.Probability = req.Probability
	}

	resp, err := client.UpdatePipelineStage(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (c *CRMRoutes) HandleDeletePipelineStage(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, c.ServiceName())
		return
	}

	stageID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	_, err = client.DeletePipelineStage(r.Context(), &crmv1.DeletePipelineStageRequest{Id: stageID})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "pipeline stage deleted"})
}

type reorderPipelineStagesRequest struct {
	StageIDs []string `json:"stage_ids"`
}

func (c *CRMRoutes) HandleReorderPipelineStages(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, c.ServiceName())
		return
	}

	var req reorderPipelineStagesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.StageIDs) == 0 {
		response.Error(w, http.StatusBadRequest, "stage_ids is required")
		return
	}

	resp, err := client.ReorderPipelineStages(r.Context(), &crmv1.ReorderPipelineStagesRequest{
		StageIds: req.StageIDs,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

// ============================================================================
// Deals Handlers
// ============================================================================

type createDealRequest struct {
	Name              string                    `json:"name"`
	Value             float64                   `json:"value"`
	Currency          string                    `json:"currency"`
	StageID           string                    `json:"stage_id"`
	ContactID         *string                   `json:"contact_id,omitempty"`
	CompanyID         *string                   `json:"company_id,omitempty"`
	OwnerID           *string                   `json:"owner_id,omitempty"`
	ExpectedCloseDate string                    `json:"expected_close_date"`
	Notes             string                    `json:"notes"`
	TagIDs            []string                  `json:"tag_ids"`
	CustomFields      []customFieldValueRequest `json:"custom_fields"`
}

func (c *CRMRoutes) HandleCreateDeal(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, c.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	var req createDealRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" || req.StageID == "" {
		response.Error(w, http.StatusBadRequest, "name and stage_id are required")
		return
	}

	grpcReq := &crmv1.CreateDealRequest{
		Name:              req.Name,
		Value:             req.Value,
		Currency:          req.Currency,
		StageId:           req.StageID,
		ExpectedCloseDate: req.ExpectedCloseDate,
		Notes:             req.Notes,
		TagIds:            req.TagIDs,
		CreatedBy:         userID,
	}

	if req.ContactID != nil {
		grpcReq.ContactId = req.ContactID
	}
	if req.CompanyID != nil {
		grpcReq.CompanyId = req.CompanyID
	}
	if req.OwnerID != nil {
		grpcReq.OwnerId = req.OwnerID
	}

	for _, cf := range req.CustomFields {
		grpcReq.CustomFields = append(grpcReq.CustomFields, &crmv1.CustomFieldValueInput{
			FieldId: cf.FieldID,
			Value:   cf.Value,
		})
	}

	resp, err := client.CreateDeal(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, resp)
}

func (c *CRMRoutes) HandleGetDeal(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, c.ServiceName())
		return
	}

	dealID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.GetDeal(r.Context(), &crmv1.GetDealRequest{Id: dealID})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (c *CRMRoutes) HandleListDeals(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, c.ServiceName())
		return
	}

	stageID := r.URL.Query().Get("stage_id")
	contactID := r.URL.Query().Get("contact_id")
	companyID := r.URL.Query().Get("company_id")
	ownerID := r.URL.Query().Get("owner_id")
	search := r.URL.Query().Get("search")
	sortBy := r.URL.Query().Get("sort_by")
	sortDesc := r.URL.Query().Get("sort_desc") == "true"
	tagIDs := r.URL.Query()["tag_ids"]
	page, pageSize := parsePagination(r, 1, 20)

	grpcReq := &crmv1.ListDealsRequest{
		Search:   search,
		Page:     int32(page),
		PageSize: int32(pageSize),
		SortBy:   sortBy,
		SortDesc: sortDesc,
		TagIds:   tagIDs,
	}

	if stageID != "" {
		grpcReq.StageId = &stageID
	}
	if contactID != "" {
		grpcReq.ContactId = &contactID
	}
	if companyID != "" {
		grpcReq.CompanyId = &companyID
	}
	if ownerID != "" {
		grpcReq.OwnerId = &ownerID
	}

	resp, err := client.ListDeals(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

type updateDealRequest struct {
	Name              *string                   `json:"name,omitempty"`
	Value             *float64                  `json:"value,omitempty"`
	Currency          *string                   `json:"currency,omitempty"`
	ContactID         *string                   `json:"contact_id,omitempty"`
	CompanyID         *string                   `json:"company_id,omitempty"`
	OwnerID           *string                   `json:"owner_id,omitempty"`
	ExpectedCloseDate *string                   `json:"expected_close_date,omitempty"`
	Notes             *string                   `json:"notes,omitempty"`
	CustomFields      []customFieldValueRequest `json:"custom_fields"`
}

func (c *CRMRoutes) HandleUpdateDeal(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, c.ServiceName())
		return
	}

	dealID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	var req updateDealRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	grpcReq := &crmv1.UpdateDealRequest{Id: dealID}
	if req.Name != nil {
		grpcReq.Name = req.Name
	}
	if req.Value != nil {
		grpcReq.Value = req.Value
	}
	if req.Currency != nil {
		grpcReq.Currency = req.Currency
	}
	if req.ContactID != nil {
		grpcReq.ContactId = req.ContactID
	}
	if req.CompanyID != nil {
		grpcReq.CompanyId = req.CompanyID
	}
	if req.OwnerID != nil {
		grpcReq.OwnerId = req.OwnerID
	}
	if req.ExpectedCloseDate != nil {
		grpcReq.ExpectedCloseDate = req.ExpectedCloseDate
	}
	if req.Notes != nil {
		grpcReq.Notes = req.Notes
	}
	for _, cf := range req.CustomFields {
		grpcReq.CustomFields = append(grpcReq.CustomFields, &crmv1.CustomFieldValueInput{
			FieldId: cf.FieldID,
			Value:   cf.Value,
		})
	}

	resp, err := client.UpdateDeal(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (c *CRMRoutes) HandleDeleteDeal(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, c.ServiceName())
		return
	}

	dealID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	_, err = client.DeleteDeal(r.Context(), &crmv1.DeleteDealRequest{Id: dealID})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "deal deleted"})
}

type moveDealToStageRequest struct {
	StageID string `json:"stage_id"`
}

func (c *CRMRoutes) HandleMoveDealToStage(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, c.ServiceName())
		return
	}

	dealID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	var req moveDealToStageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.StageID == "" {
		response.Error(w, http.StatusBadRequest, "stage_id is required")
		return
	}

	resp, err := client.MoveDealToStage(r.Context(), &crmv1.MoveDealToStageRequest{
		DealId:  dealID,
		StageId: req.StageID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

type modifyDealTagsRequest struct {
	TagIDs []string `json:"tag_ids"`
}

func (c *CRMRoutes) HandleAddDealTags(w http.ResponseWriter, r *http.Request) {
	if _, ok := validateUUIDParam(w, r, "id"); !ok {
		return
	}

	var req modifyDealTagsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	response.Error(w, http.StatusNotImplemented, "add deal tags not implemented via HTTP, use gRPC")
}

func (c *CRMRoutes) HandleRemoveDealTags(w http.ResponseWriter, r *http.Request) {
	if _, ok := validateUUIDParam(w, r, "id"); !ok {
		return
	}

	var req modifyDealTagsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	response.Error(w, http.StatusNotImplemented, "remove deal tags not implemented via HTTP, use gRPC")
}

// ============================================================================
// Activities Handlers
// ============================================================================

type createActivityRequest struct {
	ActivityType string                    `json:"activity_type"`
	Subject      string                    `json:"subject"`
	Description  string                    `json:"description"`
	ContactID    *string                   `json:"contact_id,omitempty"`
	CompanyID    *string                   `json:"company_id,omitempty"`
	DealID       *string                   `json:"deal_id,omitempty"`
	AssignedTo   *string                   `json:"assigned_to,omitempty"`
	DueDate      string                    `json:"due_date"`
	TagIDs       []string                  `json:"tag_ids"`
	CustomFields []customFieldValueRequest `json:"custom_fields"`
}

func (c *CRMRoutes) HandleCreateActivity(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, c.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	var req createActivityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.ActivityType == "" || req.Subject == "" {
		response.Error(w, http.StatusBadRequest, "activity_type and subject are required")
		return
	}

	grpcReq := &crmv1.CreateActivityRequest{
		ActivityType: req.ActivityType,
		Subject:      req.Subject,
		Description:  req.Description,
		DueDate:      req.DueDate,
		CreatedBy:    userID,
	}

	if req.ContactID != nil {
		grpcReq.ContactId = req.ContactID
	}
	if req.CompanyID != nil {
		grpcReq.CompanyId = req.CompanyID
	}
	if req.DealID != nil {
		grpcReq.DealId = req.DealID
	}
	if req.AssignedTo != nil {
		grpcReq.AssignedTo = req.AssignedTo
	}

	resp, err := client.CreateActivity(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, resp)
}

func (c *CRMRoutes) HandleGetActivity(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, c.ServiceName())
		return
	}

	activityID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.GetActivity(r.Context(), &crmv1.GetActivityRequest{Id: activityID})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (c *CRMRoutes) HandleListActivities(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, c.ServiceName())
		return
	}

	activityType := r.URL.Query().Get("activity_type")
	contactID := r.URL.Query().Get("contact_id")
	companyID := r.URL.Query().Get("company_id")
	dealID := r.URL.Query().Get("deal_id")
	assignedTo := r.URL.Query().Get("assigned_to")
	isCompletedStr := r.URL.Query().Get("is_completed")
	sortBy := r.URL.Query().Get("sort_by")
	sortDesc := r.URL.Query().Get("sort_desc") == "true"
	page, pageSize := parsePagination(r, 1, 20)

	grpcReq := &crmv1.ListActivitiesRequest{
		Page:     int32(page),
		PageSize: int32(pageSize),
		SortBy:   sortBy,
		SortDesc: sortDesc,
	}

	if activityType != "" {
		grpcReq.ActivityType = &activityType
	}
	if contactID != "" {
		grpcReq.ContactId = &contactID
	}
	if companyID != "" {
		grpcReq.CompanyId = &companyID
	}
	if dealID != "" {
		grpcReq.DealId = &dealID
	}
	if assignedTo != "" {
		grpcReq.AssignedTo = &assignedTo
	}
	if isCompletedStr != "" {
		isCompleted := isCompletedStr == "true"
		grpcReq.IsCompleted = &isCompleted
	}

	resp, err := client.ListActivities(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

type updateActivityRequest struct {
	Subject      *string                   `json:"subject,omitempty"`
	Description  *string                   `json:"description,omitempty"`
	ContactID    *string                   `json:"contact_id,omitempty"`
	CompanyID    *string                   `json:"company_id,omitempty"`
	DealID       *string                   `json:"deal_id,omitempty"`
	AssignedTo   *string                   `json:"assigned_to,omitempty"`
	DueDate      *string                   `json:"due_date,omitempty"`
	CustomFields []customFieldValueRequest `json:"custom_fields"`
}

func (c *CRMRoutes) HandleUpdateActivity(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, c.ServiceName())
		return
	}

	activityID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	var req updateActivityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	grpcReq := &crmv1.UpdateActivityRequest{Id: activityID}
	if req.Subject != nil {
		grpcReq.Subject = req.Subject
	}
	if req.Description != nil {
		grpcReq.Description = req.Description
	}
	if req.ContactID != nil {
		grpcReq.ContactId = req.ContactID
	}
	if req.CompanyID != nil {
		grpcReq.CompanyId = req.CompanyID
	}
	if req.DealID != nil {
		grpcReq.DealId = req.DealID
	}
	if req.AssignedTo != nil {
		grpcReq.AssignedTo = req.AssignedTo
	}
	if req.DueDate != nil {
		grpcReq.DueDate = req.DueDate
	}

	resp, err := client.UpdateActivity(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (c *CRMRoutes) HandleDeleteActivity(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, c.ServiceName())
		return
	}

	activityID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	_, err = client.DeleteActivity(r.Context(), &crmv1.DeleteActivityRequest{Id: activityID})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "activity deleted"})
}

func (c *CRMRoutes) HandleCompleteActivity(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, c.ServiceName())
		return
	}

	activityID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.CompleteActivity(r.Context(), &crmv1.CompleteActivityRequest{Id: activityID})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

type modifyActivityTagsRequest struct {
	TagIDs []string `json:"tag_ids"`
}

func (c *CRMRoutes) HandleAddActivityTags(w http.ResponseWriter, r *http.Request) {
	if _, ok := validateUUIDParam(w, r, "id"); !ok {
		return
	}

	var req modifyActivityTagsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	response.Error(w, http.StatusNotImplemented, "add activity tags not implemented via HTTP, use gRPC")
}

func (c *CRMRoutes) HandleRemoveActivityTags(w http.ResponseWriter, r *http.Request) {
	if _, ok := validateUUIDParam(w, r, "id"); !ok {
		return
	}

	var req modifyActivityTagsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	response.Error(w, http.StatusNotImplemented, "remove activity tags not implemented via HTTP, use gRPC")
}

// ============================================================================
// Search Handler
// ============================================================================

func (c *CRMRoutes) HandleSearch(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, c.ServiceName())
		return
	}

	query := r.URL.Query().Get("q")
	if query == "" {
		response.Error(w, http.StatusBadRequest, "query parameter 'q' is required")
		return
	}

	entityTypes := r.URL.Query()["types"]
	limit := 20

	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	resp, err := client.Search(r.Context(), &crmv1.SearchRequest{
		Query:       query,
		EntityTypes: entityTypes,
		Limit:       int32(limit),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

// ============================================================================
// Saved Filters Handlers
// ============================================================================

type createSavedFilterRequest struct {
	Name       string `json:"name"`
	EntityType string `json:"entity_type"`
	FilterJSON string `json:"filter_json"`
	IsDefault  bool   `json:"is_default"`
}

func (c *CRMRoutes) HandleCreateSavedFilter(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, c.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	var req createSavedFilterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" || req.EntityType == "" || req.FilterJSON == "" {
		response.Error(w, http.StatusBadRequest, "name, entity_type, and filter_json are required")
		return
	}

	resp, err := client.CreateSavedFilter(r.Context(), &crmv1.CreateSavedFilterRequest{
		Name:       req.Name,
		EntityType: req.EntityType,
		FilterJson: req.FilterJSON,
		IsDefault:  req.IsDefault,
		CreatedBy:  userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, resp)
}

func (c *CRMRoutes) HandleGetSavedFilter(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, c.ServiceName())
		return
	}

	filterID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.GetSavedFilter(r.Context(), &crmv1.GetSavedFilterRequest{Id: filterID})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (c *CRMRoutes) HandleListSavedFilters(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, c.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())
	entityType := r.URL.Query().Get("entity_type")

	resp, err := client.ListSavedFilters(r.Context(), &crmv1.ListSavedFiltersRequest{
		EntityType: entityType,
		UserId:     userID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

type updateSavedFilterRequest struct {
	Name       *string `json:"name,omitempty"`
	FilterJSON *string `json:"filter_json,omitempty"`
	IsDefault  *bool   `json:"is_default,omitempty"`
}

func (c *CRMRoutes) HandleUpdateSavedFilter(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, c.ServiceName())
		return
	}

	filterID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	var req updateSavedFilterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	grpcReq := &crmv1.UpdateSavedFilterRequest{Id: filterID}
	if req.Name != nil {
		grpcReq.Name = req.Name
	}
	if req.FilterJSON != nil {
		grpcReq.FilterJson = req.FilterJSON
	}
	if req.IsDefault != nil {
		grpcReq.IsDefault = req.IsDefault
	}

	resp, err := client.UpdateSavedFilter(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (c *CRMRoutes) HandleDeleteSavedFilter(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, c.ServiceName())
		return
	}

	filterID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	_, err = client.DeleteSavedFilter(r.Context(), &crmv1.DeleteSavedFilterRequest{Id: filterID})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "saved filter deleted"})
}

// ============================================================================
// Reports Handlers
// ============================================================================

func (c *CRMRoutes) HandleGetPipelineReport(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, c.ServiceName())
		return
	}

	ownerID := r.URL.Query().Get("owner_id")
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")

	if startDate == "" || endDate == "" {
		response.Error(w, http.StatusBadRequest, "start_date and end_date are required")
		return
	}

	grpcReq := &crmv1.GetPipelineReportRequest{
		StartDate: startDate,
		EndDate:   endDate,
	}
	if ownerID != "" {
		grpcReq.OwnerId = &ownerID
	}

	resp, err := client.GetPipelineReport(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (c *CRMRoutes) HandleGetConversionReport(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, c.ServiceName())
		return
	}

	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")

	if startDate == "" || endDate == "" {
		response.Error(w, http.StatusBadRequest, "start_date and end_date are required")
		return
	}

	resp, err := client.GetConversionReport(r.Context(), &crmv1.GetConversionReportRequest{
		StartDate: startDate,
		EndDate:   endDate,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (c *CRMRoutes) HandleGetActivityReport(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, c.ServiceName())
		return
	}

	userID := r.URL.Query().Get("user_id")
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")

	if startDate == "" || endDate == "" {
		response.Error(w, http.StatusBadRequest, "start_date and end_date are required")
		return
	}

	grpcReq := &crmv1.GetActivityReportRequest{
		StartDate: startDate,
		EndDate:   endDate,
	}
	if userID != "" {
		grpcReq.UserId = &userID
	}

	resp, err := client.GetActivityReport(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}
