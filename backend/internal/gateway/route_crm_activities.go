package gateway

import (
	"net/http"
	"strconv"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server/response"
	crmv1 "github.com/kmuhub/kmuhub/proto/crm/v1"
)

// ============================================================================
// Activities Handlers
// ============================================================================

type createActivityRequest struct {
	ActivityType string                    `json:"activity_type" validate:"required"`
	Subject      string                    `json:"subject" validate:"required"`
	Description  string                    `json:"description"`
	ContactID    *string                   `json:"contact_id,omitempty" validate:"omitempty,uuid"`
	CompanyID    *string                   `json:"company_id,omitempty" validate:"omitempty,uuid"`
	DealID       *string                   `json:"deal_id,omitempty" validate:"omitempty,uuid"`
	AssignedTo   *string                   `json:"assigned_to,omitempty"`
	DueDate      string                    `json:"due_date"`
	TagIDs       []string                  `json:"tag_ids" validate:"omitempty,dive,uuid"`
	CustomFields []customFieldValueRequest `json:"custom_fields"`
}

func (c *CRMRoutes) HandleCreateActivity(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, c.ServiceName())
		return
	}

	actTID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}

	userID := middleware.GetUserID(r.Context())

	req, ok := decodeAndValidate[createActivityRequest](w, r)
	if !ok {
		return
	}

	grpcReq := &crmv1.CreateActivityRequest{
		TenantId:     actTID.String(),
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

	response.Proto(w, http.StatusCreated, resp)
}

func (c *CRMRoutes) HandleGetActivity(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, c.ServiceName())
		return
	}

	actGetTID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}

	activityID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.GetActivity(r.Context(), &crmv1.GetActivityRequest{Id: activityID, TenantId: actGetTID.String()})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

func (c *CRMRoutes) HandleListActivities(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, c.ServiceName())
		return
	}

	actListTID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
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
		TenantId: actListTID.String(),
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

	response.Proto(w, http.StatusOK, resp)
}

type updateActivityRequest struct {
	Subject      *string                   `json:"subject,omitempty" validate:"omitempty,min=1"`
	Description  *string                   `json:"description,omitempty"`
	ContactID    *string                   `json:"contact_id,omitempty" validate:"omitempty,uuid"`
	CompanyID    *string                   `json:"company_id,omitempty" validate:"omitempty,uuid"`
	DealID       *string                   `json:"deal_id,omitempty" validate:"omitempty,uuid"`
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

	req, ok := decodeAndValidate[updateActivityRequest](w, r)
	if !ok {
		return
	}

	actUpdateTID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}

	grpcReq := &crmv1.UpdateActivityRequest{Id: activityID, TenantId: actUpdateTID.String()}
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

	response.Proto(w, http.StatusOK, resp)
}

func (c *CRMRoutes) HandleDeleteActivity(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, c.ServiceName())
		return
	}

	actDelTID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}

	activityID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	_, err = client.DeleteActivity(r.Context(), &crmv1.DeleteActivityRequest{Id: activityID, TenantId: actDelTID.String()})
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

	actCompleteTID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}

	activityID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.CompleteActivity(r.Context(), &crmv1.CompleteActivityRequest{Id: activityID, TenantId: actCompleteTID.String()})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

type modifyActivityTagsRequest struct {
	TagIDs []string `json:"tag_ids" validate:"omitempty,dive,uuid"`
}

func (c *CRMRoutes) HandleAddActivityTags(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, c.ServiceName())
		return
	}

	activityID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	req, ok := decodeAndValidate[modifyActivityTagsRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.AddActivityTags(r.Context(), &crmv1.AddActivityTagsRequest{
		ActivityId: activityID,
		TagIds:     req.TagIDs,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

func (c *CRMRoutes) HandleRemoveActivityTags(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, c.ServiceName())
		return
	}

	activityID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	req, ok := decodeAndValidate[modifyActivityTagsRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.RemoveActivityTags(r.Context(), &crmv1.RemoveActivityTagsRequest{
		ActivityId: activityID,
		TagIds:     req.TagIDs,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
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

	response.Proto(w, http.StatusOK, resp)
}

// ============================================================================
// Saved Filters Handlers
// ============================================================================

type createSavedFilterRequest struct {
	Name       string `json:"name" validate:"required"`
	EntityType string `json:"entity_type" validate:"required"`
	FilterJSON string `json:"filter_json" validate:"required"`
	IsDefault  bool   `json:"is_default"`
}

func (c *CRMRoutes) HandleCreateSavedFilter(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, c.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	req, ok := decodeAndValidate[createSavedFilterRequest](w, r)
	if !ok {
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

	response.Proto(w, http.StatusCreated, resp)
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

	response.Proto(w, http.StatusOK, resp)
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

	response.Proto(w, http.StatusOK, resp)
}

type updateSavedFilterRequest struct {
	Name       *string `json:"name,omitempty" validate:"omitempty,min=1"`
	FilterJSON *string `json:"filter_json,omitempty" validate:"omitempty,min=1"`
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

	req, ok := decodeAndValidate[updateSavedFilterRequest](w, r)
	if !ok {
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

	response.Proto(w, http.StatusOK, resp)
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
	if !validateDateParam(w, "start_date", startDate) || !validateDateParam(w, "end_date", endDate) {
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

	response.Proto(w, http.StatusOK, resp)
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
	if !validateDateParam(w, "start_date", startDate) || !validateDateParam(w, "end_date", endDate) {
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

	response.Proto(w, http.StatusOK, resp)
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
	if !validateDateParam(w, "start_date", startDate) || !validateDateParam(w, "end_date", endDate) {
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

	response.Proto(w, http.StatusOK, resp)
}

// ============================================================================
// Custom Fields Handlers
// ============================================================================

type createCustomFieldRequest struct {
	EntityType   string   `json:"entity_type" validate:"required"`
	FieldName    string   `json:"field_name" validate:"required"`
	FieldLabel   string   `json:"field_label" validate:"required"`
	FieldType    string   `json:"field_type" validate:"required"`
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

	req, ok := decodeAndValidate[createCustomFieldRequest](w, r)
	if !ok {
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

	response.Proto(w, http.StatusCreated, resp)
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

	response.Proto(w, http.StatusOK, resp)
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

	response.Proto(w, http.StatusOK, resp)
}

type updateCustomFieldRequest struct {
	FieldLabel   *string  `json:"field_label,omitempty" validate:"omitempty,min=1"`
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

	req, ok := decodeAndValidate[updateCustomFieldRequest](w, r)
	if !ok {
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

	response.Proto(w, http.StatusOK, resp)
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
	Name       string `json:"name" validate:"required"`
	Color      string `json:"color"`
	EntityType string `json:"entity_type" validate:"required"`
}

func (c *CRMRoutes) HandleCreateTag(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, c.ServiceName())
		return
	}

	req, ok := decodeAndValidate[createTagRequest](w, r)
	if !ok {
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

	response.Proto(w, http.StatusCreated, resp)
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

	response.Proto(w, http.StatusOK, resp)
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

	response.Proto(w, http.StatusOK, resp)
}

type updateTagRequest struct {
	Name  *string `json:"name,omitempty" validate:"omitempty,min=1"`
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

	req, ok := decodeAndValidate[updateTagRequest](w, r)
	if !ok {
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

	response.Proto(w, http.StatusOK, resp)
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
