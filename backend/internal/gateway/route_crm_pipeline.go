package gateway

import (
	"net/http"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server/response"
	crmv1 "github.com/kmuhub/kmuhub/proto/crm/v1"
)

// ============================================================================
// Pipeline Stages Handlers
// ============================================================================

type createPipelineStageRequest struct {
	Name        string  `json:"name" validate:"required"`
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

	req, ok := decodeAndValidate[createPipelineStageRequest](w, r)
	if !ok {
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

	response.Proto(w, http.StatusCreated, resp)
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

	response.Proto(w, http.StatusOK, resp)
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

	response.Proto(w, http.StatusOK, resp)
}

type updatePipelineStageRequest struct {
	Name        *string  `json:"name,omitempty" validate:"omitempty,min=1"`
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

	req, ok := decodeAndValidate[updatePipelineStageRequest](w, r)
	if !ok {
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

	response.Proto(w, http.StatusOK, resp)
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
	StageIDs []string `json:"stage_ids" validate:"required,min=1,dive,uuid"`
}

func (c *CRMRoutes) HandleReorderPipelineStages(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, c.ServiceName())
		return
	}

	req, ok := decodeAndValidate[reorderPipelineStagesRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.ReorderPipelineStages(r.Context(), &crmv1.ReorderPipelineStagesRequest{
		StageIds: req.StageIDs,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

// ============================================================================
// Deals Handlers
// ============================================================================

type createDealRequest struct {
	Name              string                    `json:"name" validate:"required"`
	Value             float64                   `json:"value"`
	Currency          string                    `json:"currency"`
	StageID           string                    `json:"stage_id" validate:"required,uuid"`
	ContactID         *string                   `json:"contact_id,omitempty" validate:"omitempty,uuid"`
	CompanyID         *string                   `json:"company_id,omitempty" validate:"omitempty,uuid"`
	OwnerID           *string                   `json:"owner_id,omitempty" validate:"omitempty,uuid"`
	ExpectedCloseDate string                    `json:"expected_close_date"`
	Notes             string                    `json:"notes"`
	TagIDs            []string                  `json:"tag_ids" validate:"omitempty,dive,uuid"`
	CustomFields      []customFieldValueRequest `json:"custom_fields"`
}

func (c *CRMRoutes) HandleCreateDeal(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, c.ServiceName())
		return
	}

	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}

	userID := middleware.GetUserID(r.Context())

	req, ok := decodeAndValidate[createDealRequest](w, r)
	if !ok {
		return
	}

	grpcReq := &crmv1.CreateDealRequest{
		TenantId:          tenantID.String(),
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

	response.Proto(w, http.StatusCreated, resp)
}

func (c *CRMRoutes) HandleGetDeal(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, c.ServiceName())
		return
	}

	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}

	dealID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	resp, err := client.GetDeal(r.Context(), &crmv1.GetDealRequest{Id: dealID, TenantId: tenantID.String()})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

func (c *CRMRoutes) HandleListDeals(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, c.ServiceName())
		return
	}

	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
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
		TenantId: tenantID.String(),
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

	response.Proto(w, http.StatusOK, resp)
}

type updateDealRequest struct {
	Name              *string                   `json:"name,omitempty" validate:"omitempty,min=1"`
	Value             *float64                  `json:"value,omitempty"`
	Currency          *string                   `json:"currency,omitempty"`
	ContactID         *string                   `json:"contact_id,omitempty" validate:"omitempty,uuid"`
	CompanyID         *string                   `json:"company_id,omitempty" validate:"omitempty,uuid"`
	OwnerID           *string                   `json:"owner_id,omitempty" validate:"omitempty,uuid"`
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

	req, ok := decodeAndValidate[updateDealRequest](w, r)
	if !ok {
		return
	}

	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}

	grpcReq := &crmv1.UpdateDealRequest{Id: dealID, TenantId: tenantID.String()}
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

	response.Proto(w, http.StatusOK, resp)
}

func (c *CRMRoutes) HandleDeleteDeal(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, c.ServiceName())
		return
	}

	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}

	dealID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	_, err = client.DeleteDeal(r.Context(), &crmv1.DeleteDealRequest{Id: dealID, TenantId: tenantID.String()})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"status": "deal deleted"})
}

type moveDealToStageRequest struct {
	StageID string `json:"stage_id" validate:"required,uuid"`
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

	req, ok := decodeAndValidate[moveDealToStageRequest](w, r)
	if !ok {
		return
	}

	tenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}

	resp, err := client.MoveDealToStage(r.Context(), &crmv1.MoveDealToStageRequest{
		DealId:   dealID,
		StageId:  req.StageID,
		TenantId: tenantID.String(),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

type modifyDealTagsRequest struct {
	TagIDs []string `json:"tag_ids" validate:"omitempty,dive,uuid"`
}

func (c *CRMRoutes) HandleAddDealTags(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, c.ServiceName())
		return
	}

	dealID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	req, ok := decodeAndValidate[modifyDealTagsRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.AddDealTags(r.Context(), &crmv1.AddDealTagsRequest{
		DealId: dealID,
		TagIds: req.TagIDs,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

func (c *CRMRoutes) HandleRemoveDealTags(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, c.ServiceName())
		return
	}

	dealID, ok := validateUUIDParam(w, r, "id")
	if !ok {
		return
	}

	req, ok := decodeAndValidate[modifyDealTagsRequest](w, r)
	if !ok {
		return
	}

	resp, err := client.RemoveDealTags(r.Context(), &crmv1.RemoveDealTagsRequest{
		DealId: dealID,
		TagIds: req.TagIDs,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}
