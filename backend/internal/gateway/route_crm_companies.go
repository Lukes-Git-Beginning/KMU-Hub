package gateway

import (
	"net/http"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server/response"
	crmv1 "github.com/kmuhub/kmuhub/proto/crm/v1"
)

// ============================================================================
// Companies Handlers
// ============================================================================

type createCompanyRequest struct {
	Name          string                    `json:"name" validate:"required"`
	Domain        string                    `json:"domain"`
	Industry      string                    `json:"industry"`
	EmployeeCount int32                     `json:"employee_count"`
	Address       string                    `json:"address"`
	City          string                    `json:"city"`
	Country       string                    `json:"country"`
	Notes         string                    `json:"notes"`
	TagIDs        []string                  `json:"tag_ids" validate:"omitempty,dive,uuid"`
	CustomFields  []customFieldValueRequest `json:"custom_fields"`
}

func (c *CRMRoutes) HandleCreateCompany(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, c.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	req, ok := decodeAndValidate[createCompanyRequest](w, r)
	if !ok {
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
	Name          *string                   `json:"name,omitempty" validate:"omitempty,min=1"`
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

	req, ok := decodeAndValidate[updateCompanyRequest](w, r)
	if !ok {
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
