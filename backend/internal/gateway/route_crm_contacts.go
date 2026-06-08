package gateway

import (
	"bytes"
	"net/http"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server/response"
	crmv1 "github.com/kmuhub/kmuhub/proto/crm/v1"
)

// ============================================================================
// Contacts Handlers
// ============================================================================

type customFieldValueRequest struct {
	FieldID string `json:"field_id"`
	Value   string `json:"value"`
}

type createContactRequest struct {
	FirstName    string                    `json:"first_name" validate:"required"`
	LastName     string                    `json:"last_name" validate:"required"`
	Email        string                    `json:"email" validate:"omitempty,email"`
	Phone        string                    `json:"phone" validate:"omitempty,phone_dach"`
	CompanyID    *string                   `json:"company_id,omitempty" validate:"omitempty,uuid"`
	Position     string                    `json:"position"`
	Notes        string                    `json:"notes"`
	TagIDs       []string                  `json:"tag_ids" validate:"omitempty,dive,uuid"`
	CustomFields []customFieldValueRequest `json:"custom_fields"`
}

func (c *CRMRoutes) HandleCreateContact(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, c.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	req, ok := decodeAndValidate[createContactRequest](w, r)
	if !ok {
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
	FirstName    *string                   `json:"first_name,omitempty" validate:"omitempty,min=1"`
	LastName     *string                   `json:"last_name,omitempty" validate:"omitempty,min=1"`
	Email        *string                   `json:"email,omitempty" validate:"omitempty,email"`
	Phone        *string                   `json:"phone,omitempty" validate:"omitempty,phone_dach"`
	CompanyID    *string                   `json:"company_id,omitempty" validate:"omitempty,uuid"`
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

	req, ok := decodeAndValidate[updateContactRequest](w, r)
	if !ok {
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
	TagIDs []string `json:"tag_ids" validate:"omitempty,dive,uuid"`
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

	req, ok := decodeAndValidate[modifyContactTagsRequest](w, r)
	if !ok {
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

	req, ok := decodeAndValidate[modifyContactTagsRequest](w, r)
	if !ok {
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
	ContactIDs []string `json:"contact_ids" validate:"omitempty,dive,uuid"`
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

	req, ok := decodeAndValidate[exportContactsCSVRequest](w, r)
	if !ok {
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
	ContactIDs []string `json:"contact_ids" validate:"omitempty,dive,uuid"`
}

func (c *CRMRoutes) HandleExportContactsVCard(w http.ResponseWriter, r *http.Request) {
	client, err := c.getCRMClient()
	if err != nil {
		respondServiceUnavailable(w, c.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())
	isAdmin := middleware.IsAdmin(r.Context())

	req, ok := decodeAndValidate[exportContactsVCardRequest](w, r)
	if !ok {
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
	Visibility string `json:"visibility" validate:"required,oneof=shared personal"`
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

	req, ok := decodeAndValidate[updateContactVisibilityRequest](w, r)
	if !ok {
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
