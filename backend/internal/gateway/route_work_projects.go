package gateway

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server/response"
	workv1 "github.com/kmuhub/kmuhub/proto/work/v1"
)

// ============================================================================
// Project Handlers
// ============================================================================

type createProjectRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	ProjectKey  string  `json:"project_key"`
}

func (w *WorkRoutes) HandleCreateProject(wr http.ResponseWriter, r *http.Request) {
	client, err := w.getWorkClient()
	if err != nil {
		respondServiceUnavailable(wr, w.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	var req createProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(wr, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" || req.ProjectKey == "" {
		response.Error(wr, http.StatusBadRequest, "name and project_key are required")
		return
	}

	resp, err := client.CreateProject(r.Context(), &workv1.CreateProjectRequest{
		Name:        req.Name,
		Description: req.Description,
		ProjectKey:  req.ProjectKey,
		CreatedBy:   userID,
	})
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.JSON(wr, http.StatusCreated, resp)
}

func (w *WorkRoutes) HandleGetProject(wr http.ResponseWriter, r *http.Request) {
	client, err := w.getWorkClient()
	if err != nil {
		respondServiceUnavailable(wr, w.ServiceName())
		return
	}

	projectID, ok := validateUUIDParam(wr, r, "id")
	if !ok {
		return
	}

	resp, err := client.GetProject(r.Context(), &workv1.GetProjectRequest{Id: projectID})
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.JSON(wr, http.StatusOK, resp)
}

func (w *WorkRoutes) HandleListProjects(wr http.ResponseWriter, r *http.Request) {
	client, err := w.getWorkClient()
	if err != nil {
		respondServiceUnavailable(wr, w.ServiceName())
		return
	}

	includeArchived := r.URL.Query().Get("include_archived") == "true"
	templatesOnly := r.URL.Query().Get("templates_only") == "true"
	search := r.URL.Query().Get("search")
	page, pageSize := parsePagination(r, 1, 20)

	resp, err := client.ListProjects(r.Context(), &workv1.ListProjectsRequest{
		IncludeArchived: includeArchived,
		TemplatesOnly:   templatesOnly,
		Search:          search,
		Page:            int32(page),
		PageSize:        int32(pageSize),
	})
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.JSON(wr, http.StatusOK, resp)
}

type updateProjectRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
}

func (w *WorkRoutes) HandleUpdateProject(wr http.ResponseWriter, r *http.Request) {
	client, err := w.getWorkClient()
	if err != nil {
		respondServiceUnavailable(wr, w.ServiceName())
		return
	}

	projectID, ok := validateUUIDParam(wr, r, "id")
	if !ok {
		return
	}

	var req updateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(wr, http.StatusBadRequest, "invalid request body")
		return
	}

	grpcReq := &workv1.UpdateProjectRequest{Id: projectID}
	if req.Name != nil {
		grpcReq.Name = req.Name
	}
	if req.Description != nil {
		grpcReq.Description = req.Description
	}

	resp, err := client.UpdateProject(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.JSON(wr, http.StatusOK, resp)
}

func (w *WorkRoutes) HandleArchiveProject(wr http.ResponseWriter, r *http.Request) {
	client, err := w.getWorkClient()
	if err != nil {
		respondServiceUnavailable(wr, w.ServiceName())
		return
	}

	projectID, ok := validateUUIDParam(wr, r, "id")
	if !ok {
		return
	}

	resp, err := client.ArchiveProject(r.Context(), &workv1.ArchiveProjectRequest{Id: projectID})
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.JSON(wr, http.StatusOK, resp)
}

// ============================================================================
// Project Member Handlers
// ============================================================================

type addProjectMemberRequest struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
}

func (w *WorkRoutes) HandleAddProjectMember(wr http.ResponseWriter, r *http.Request) {
	client, err := w.getWorkClient()
	if err != nil {
		respondServiceUnavailable(wr, w.ServiceName())
		return
	}

	projectID, ok := validateUUIDParam(wr, r, "id")
	if !ok {
		return
	}

	var req addProjectMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(wr, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := client.AddProjectMember(r.Context(), &workv1.AddProjectMemberRequest{
		ProjectId: projectID,
		UserId:    req.UserID,
		Role:      req.Role,
	})
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.JSON(wr, http.StatusCreated, resp)
}

func (w *WorkRoutes) HandleRemoveProjectMember(wr http.ResponseWriter, r *http.Request) {
	client, err := w.getWorkClient()
	if err != nil {
		respondServiceUnavailable(wr, w.ServiceName())
		return
	}

	projectID := chi.URLParam(r, "id")
	memberUserID := chi.URLParam(r, "userId")

	_, err = client.RemoveProjectMember(r.Context(), &workv1.RemoveProjectMemberRequest{
		ProjectId: projectID,
		UserId:    memberUserID,
	})
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.JSON(wr, http.StatusOK, map[string]string{"status": "member removed"})
}

func (w *WorkRoutes) HandleListProjectMembers(wr http.ResponseWriter, r *http.Request) {
	client, err := w.getWorkClient()
	if err != nil {
		respondServiceUnavailable(wr, w.ServiceName())
		return
	}

	projectID := chi.URLParam(r, "id")

	resp, err := client.ListProjectMembers(r.Context(), &workv1.ListProjectMembersRequest{
		ProjectId: projectID,
	})
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.JSON(wr, http.StatusOK, resp)
}

type updateProjectMemberRoleRequest struct {
	Role string `json:"role"`
}

func (w *WorkRoutes) HandleUpdateProjectMemberRole(wr http.ResponseWriter, r *http.Request) {
	client, err := w.getWorkClient()
	if err != nil {
		respondServiceUnavailable(wr, w.ServiceName())
		return
	}

	projectID := chi.URLParam(r, "id")
	memberUserID := chi.URLParam(r, "userId")

	var req updateProjectMemberRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(wr, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := client.UpdateProjectMemberRole(r.Context(), &workv1.UpdateProjectMemberRoleRequest{
		ProjectId: projectID,
		UserId:    memberUserID,
		Role:      req.Role,
	})
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.JSON(wr, http.StatusOK, resp)
}

// ============================================================================
// Template Handlers
// ============================================================================

type saveAsTemplateRequest struct {
	TemplateName string `json:"template_name"`
}

func (w *WorkRoutes) HandleSaveProjectAsTemplate(wr http.ResponseWriter, r *http.Request) {
	client, err := w.getWorkClient()
	if err != nil {
		respondServiceUnavailable(wr, w.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())
	projectID := chi.URLParam(r, "id")

	var req saveAsTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(wr, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := client.SaveProjectAsTemplate(r.Context(), &workv1.SaveProjectAsTemplateRequest{
		ProjectId:    projectID,
		TemplateName: req.TemplateName,
		CreatedBy:    userID,
	})
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.JSON(wr, http.StatusCreated, resp)
}

type createFromTemplateRequest struct {
	TemplateID string `json:"template_id"`
	Name       string `json:"name"`
	ProjectKey string `json:"project_key"`
}

func (w *WorkRoutes) HandleCreateProjectFromTemplate(wr http.ResponseWriter, r *http.Request) {
	client, err := w.getWorkClient()
	if err != nil {
		respondServiceUnavailable(wr, w.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())

	var req createFromTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(wr, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := client.CreateProjectFromTemplate(r.Context(), &workv1.CreateProjectFromTemplateRequest{
		TemplateId: req.TemplateID,
		Name:       req.Name,
		ProjectKey: req.ProjectKey,
		CreatedBy:  userID,
	})
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.JSON(wr, http.StatusCreated, resp)
}

// ============================================================================
// Status Handlers
// ============================================================================

type createStatusRequest struct {
	Name      string  `json:"name"`
	Color     *string `json:"color,omitempty"`
	IsDefault bool    `json:"is_default"`
	IsClosed  bool    `json:"is_closed"`
}

func (w *WorkRoutes) HandleCreateProjectStatus(wr http.ResponseWriter, r *http.Request) {
	client, err := w.getWorkClient()
	if err != nil {
		respondServiceUnavailable(wr, w.ServiceName())
		return
	}

	projectID := chi.URLParam(r, "id")

	var req createStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(wr, http.StatusBadRequest, "invalid request body")
		return
	}

	grpcReq := &workv1.CreateProjectStatusRequest{
		ProjectId: projectID,
		Name:      req.Name,
		IsDefault: req.IsDefault,
		IsClosed:  req.IsClosed,
	}
	if req.Color != nil {
		grpcReq.Color = req.Color
	}

	resp, err := client.CreateProjectStatus(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.JSON(wr, http.StatusCreated, resp)
}

type updateStatusRequest struct {
	Name      *string `json:"name,omitempty"`
	Color     *string `json:"color,omitempty"`
	IsDefault *bool   `json:"is_default,omitempty"`
	IsClosed  *bool   `json:"is_closed,omitempty"`
}

func (w *WorkRoutes) HandleUpdateProjectStatus(wr http.ResponseWriter, r *http.Request) {
	client, err := w.getWorkClient()
	if err != nil {
		respondServiceUnavailable(wr, w.ServiceName())
		return
	}

	statusID := chi.URLParam(r, "id")

	var req updateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(wr, http.StatusBadRequest, "invalid request body")
		return
	}

	grpcReq := &workv1.UpdateProjectStatusRequest{Id: statusID}
	if req.Name != nil {
		grpcReq.Name = req.Name
	}
	if req.Color != nil {
		grpcReq.Color = req.Color
	}
	if req.IsDefault != nil {
		grpcReq.IsDefault = req.IsDefault
	}
	if req.IsClosed != nil {
		grpcReq.IsClosed = req.IsClosed
	}

	resp, err := client.UpdateProjectStatus(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.JSON(wr, http.StatusOK, resp)
}

func (w *WorkRoutes) HandleDeleteProjectStatus(wr http.ResponseWriter, r *http.Request) {
	client, err := w.getWorkClient()
	if err != nil {
		respondServiceUnavailable(wr, w.ServiceName())
		return
	}

	statusID := chi.URLParam(r, "id")

	_, err = client.DeleteProjectStatus(r.Context(), &workv1.DeleteProjectStatusRequest{Id: statusID})
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.JSON(wr, http.StatusOK, map[string]string{"status": "status deleted"})
}

type reorderStatusesRequest struct {
	StatusIDs []string `json:"status_ids"`
}

func (w *WorkRoutes) HandleReorderProjectStatuses(wr http.ResponseWriter, r *http.Request) {
	client, err := w.getWorkClient()
	if err != nil {
		respondServiceUnavailable(wr, w.ServiceName())
		return
	}

	projectID := chi.URLParam(r, "id")

	var req reorderStatusesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(wr, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := client.ReorderProjectStatuses(r.Context(), &workv1.ReorderProjectStatusesRequest{
		ProjectId: projectID,
		StatusIds: req.StatusIDs,
	})
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.JSON(wr, http.StatusOK, resp)
}

func (w *WorkRoutes) HandleListProjectStatuses(wr http.ResponseWriter, r *http.Request) {
	client, err := w.getWorkClient()
	if err != nil {
		respondServiceUnavailable(wr, w.ServiceName())
		return
	}

	projectID := chi.URLParam(r, "id")

	resp, err := client.ListProjectStatuses(r.Context(), &workv1.ListProjectStatusesRequest{
		ProjectId: projectID,
	})
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.JSON(wr, http.StatusOK, resp)
}

// ============================================================================
// Preference Handlers
// ============================================================================

func (w *WorkRoutes) HandleGetUserProjectPreference(wr http.ResponseWriter, r *http.Request) {
	client, err := w.getWorkClient()
	if err != nil {
		respondServiceUnavailable(wr, w.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())
	projectID := chi.URLParam(r, "id")

	resp, err := client.GetUserProjectPreference(r.Context(), &workv1.GetUserProjectPreferenceRequest{
		UserId:    userID,
		ProjectId: projectID,
	})
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.JSON(wr, http.StatusOK, resp)
}

type setPreferenceRequest struct {
	ViewType     *string `json:"view_type,omitempty"`
	ListGroupBy  *string `json:"list_group_by,omitempty"`
	ListSortBy   *string `json:"list_sort_by,omitempty"`
	ListSortDesc *bool   `json:"list_sort_desc,omitempty"`
}

func (w *WorkRoutes) HandleSetUserProjectPreference(wr http.ResponseWriter, r *http.Request) {
	client, err := w.getWorkClient()
	if err != nil {
		respondServiceUnavailable(wr, w.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())
	projectID := chi.URLParam(r, "id")

	var req setPreferenceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(wr, http.StatusBadRequest, "invalid request body")
		return
	}

	grpcReq := &workv1.SetUserProjectPreferenceRequest{
		UserId:    userID,
		ProjectId: projectID,
	}
	if req.ViewType != nil {
		grpcReq.ViewType = req.ViewType
	}
	if req.ListGroupBy != nil {
		grpcReq.ListGroupBy = req.ListGroupBy
	}
	if req.ListSortBy != nil {
		grpcReq.ListSortBy = req.ListSortBy
	}
	if req.ListSortDesc != nil {
		grpcReq.ListSortDesc = req.ListSortDesc
	}

	resp, err := client.SetUserProjectPreference(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.JSON(wr, http.StatusOK, resp)
}
