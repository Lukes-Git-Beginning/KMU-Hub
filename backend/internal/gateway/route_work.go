package gateway

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server/response"
	workv1 "github.com/kmuhub/kmuhub/proto/work/v1"
)

// WorkRoutes handles HTTP routes for the Work backend service.
type WorkRoutes struct {
	registry *ServiceRegistry
}

// NewWorkRoutes creates a new WorkRoutes with the given service registry.
func NewWorkRoutes(registry *ServiceRegistry) *WorkRoutes {
	return &WorkRoutes{registry: registry}
}

// ServiceName returns the backend service name.
func (w *WorkRoutes) ServiceName() string { return "work" }

// getWorkClient lazily obtains a gRPC client for the Work service.
func (w *WorkRoutes) getWorkClient() (workv1.WorkServiceClient, error) {
	conn, err := w.registry.GetConnection("work")
	if err != nil {
		return nil, err
	}
	return workv1.NewWorkServiceClient(conn), nil
}

// RegisterRoutes registers all Work HTTP routes.
func (w *WorkRoutes) RegisterRoutes(r chi.Router, authMiddleware func(http.Handler) http.Handler) {
	// Projects
	r.Route("/api/v1/projects", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("projects", "read")).Get("/", w.HandleListProjects)
		r.With(middleware.RequirePermission("projects", "read")).Get("/{id}", w.HandleGetProject)
		r.With(middleware.RequirePermission("projects", "write")).Post("/", w.HandleCreateProject)
		r.With(middleware.RequirePermission("projects", "write")).Put("/{id}", w.HandleUpdateProject)
		r.With(middleware.RequirePermission("projects", "write")).Post("/{id}/archive", w.HandleArchiveProject)

		// Members
		r.With(middleware.RequirePermission("projects", "read")).Get("/{id}/members", w.HandleListProjectMembers)
		r.With(middleware.RequirePermission("projects", "write")).Post("/{id}/members", w.HandleAddProjectMember)
		r.With(middleware.RequirePermission("projects", "write")).Put("/{id}/members/{userId}", w.HandleUpdateProjectMemberRole)
		r.With(middleware.RequirePermission("projects", "write")).Delete("/{id}/members/{userId}", w.HandleRemoveProjectMember)

		// Templates
		r.With(middleware.RequirePermission("projects", "write")).Post("/{id}/template", w.HandleSaveProjectAsTemplate)
		r.With(middleware.RequirePermission("projects", "write")).Post("/from-template", w.HandleCreateProjectFromTemplate)

		// Statuses
		r.With(middleware.RequirePermission("projects", "read")).Get("/{id}/statuses", w.HandleListProjectStatuses)
		r.With(middleware.RequirePermission("projects", "write")).Post("/{id}/statuses", w.HandleCreateProjectStatus)
		r.With(middleware.RequirePermission("projects", "write")).Post("/{id}/statuses/reorder", w.HandleReorderProjectStatuses)

		// Preferences
		r.With(middleware.RequirePermission("projects", "read")).Get("/{id}/preferences", w.HandleGetUserProjectPreference)
		r.With(middleware.RequirePermission("projects", "write")).Put("/{id}/preferences", w.HandleSetUserProjectPreference)
	})

	// Project Statuses (top-level for update/delete by status ID)
	r.Route("/api/v1/project-statuses", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("projects", "write")).Put("/{id}", w.HandleUpdateProjectStatus)
		r.With(middleware.RequirePermission("projects", "delete")).Delete("/{id}", w.HandleDeleteProjectStatus)
	})

	// Tasks
	r.Route("/api/v1/tasks", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("tasks", "read")).Get("/", w.HandleListTasks)
		r.With(middleware.RequirePermission("tasks", "read")).Get("/{id}", w.HandleGetTask)
		r.With(middleware.RequirePermission("tasks", "write")).Post("/", w.HandleCreateTask)
		r.With(middleware.RequirePermission("tasks", "write")).Put("/{id}", w.HandleUpdateTask)
		r.With(middleware.RequirePermission("tasks", "delete")).Delete("/{id}", w.HandleDeleteTask)
		r.With(middleware.RequirePermission("tasks", "write")).Post("/{id}/move", w.HandleMoveTask)
		r.With(middleware.RequirePermission("tasks", "read")).Get("/{id}/subtasks", w.HandleListSubtasks)

		// Dependencies
		r.With(middleware.RequirePermission("tasks", "read")).Get("/{id}/dependencies", w.HandleListTaskDependencies)
		r.With(middleware.RequirePermission("tasks", "write")).Post("/{id}/dependencies", w.HandleCreateTaskDependency)

		// Comments
		r.With(middleware.RequirePermission("tasks", "read")).Get("/{id}/comments", w.HandleListTaskComments)
		r.With(middleware.RequirePermission("tasks", "write")).Post("/{id}/comments", w.HandleCreateTaskComment)

		// Entity links
		r.With(middleware.RequirePermission("tasks", "read")).Get("/{id}/links", w.HandleListTaskEntityLinks)
		r.With(middleware.RequirePermission("tasks", "write")).Post("/{id}/links", w.HandleLinkEntityToTask)

		// Activity log
		r.With(middleware.RequirePermission("tasks", "read")).Get("/{id}/activities", w.HandleListTaskActivities)

		// Files
		r.With(middleware.RequirePermission("tasks", "read")).Get("/{id}/files", w.HandleListTaskFiles)
		r.With(middleware.RequirePermission("tasks", "write")).Post("/{id}/files", w.HandleAttachFileToTask)

		// Custom fields
		r.With(middleware.RequirePermission("tasks", "read")).Get("/{id}/custom-fields", w.HandleGetTaskCustomFieldValues)
		r.With(middleware.RequirePermission("tasks", "write")).Put("/{id}/custom-fields", w.HandleSetTaskCustomFieldValues)
	})

	// Task dependencies (top-level delete)
	r.Route("/api/v1/task-dependencies", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("tasks", "delete")).Delete("/{id}", w.HandleDeleteTaskDependency)
	})

	// Task comments (top-level update/delete)
	r.Route("/api/v1/task-comments", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("tasks", "write")).Put("/{id}", w.HandleUpdateTaskComment)
		r.With(middleware.RequirePermission("tasks", "delete")).Delete("/{id}", w.HandleDeleteTaskComment)
	})

	// Entity links (top-level delete)
	r.Route("/api/v1/task-links", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("tasks", "delete")).Delete("/{id}", w.HandleUnlinkEntityFromTask)
	})

	// Task files (top-level delete)
	r.Route("/api/v1/task-files", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("tasks", "delete")).Delete("/{id}", w.HandleRemoveTaskFile)
	})

	// Entity tasks (list tasks linked to a CRM entity)
	r.Route("/api/v1/entity-tasks", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("tasks", "read")).Get("/", w.HandleListEntityTasks)
	})

	// Work search
	r.Route("/api/v1/work/search", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("tasks", "read")).Get("/", w.HandleSearchTasks)
	})
}

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
	if userID == "" {
		response.Error(wr, http.StatusUnauthorized, "user not authenticated")
		return
	}

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

	projectID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(projectID); err != nil {
		response.Error(wr, http.StatusBadRequest, "invalid project id")
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

	projectID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(projectID); err != nil {
		response.Error(wr, http.StatusBadRequest, "invalid project id")
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

	projectID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(projectID); err != nil {
		response.Error(wr, http.StatusBadRequest, "invalid project id")
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

	projectID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(projectID); err != nil {
		response.Error(wr, http.StatusBadRequest, "invalid project id")
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
// Task Handlers
// ============================================================================

type createTaskRequest struct {
	ProjectID    string                     `json:"project_id"`
	Title        string                     `json:"title"`
	Description  *string                    `json:"description,omitempty"`
	StatusID     *string                    `json:"status_id,omitempty"`
	Priority     string                     `json:"priority"`
	AssigneeID   *string                    `json:"assignee_id,omitempty"`
	ParentTaskID *string                    `json:"parent_task_id,omitempty"`
	DueDate      *string                    `json:"due_date,omitempty"`
	CustomFields []workCustomFieldValueReq  `json:"custom_fields,omitempty"`
}

type workCustomFieldValueReq struct {
	FieldID string `json:"field_id"`
	Value   string `json:"value"`
}

func (w *WorkRoutes) HandleCreateTask(wr http.ResponseWriter, r *http.Request) {
	client, err := w.getWorkClient()
	if err != nil {
		respondServiceUnavailable(wr, w.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		response.Error(wr, http.StatusUnauthorized, "user not authenticated")
		return
	}

	var req createTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(wr, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Title == "" {
		response.Error(wr, http.StatusBadRequest, "title is required")
		return
	}

	grpcReq := &workv1.CreateTaskRequest{
		ProjectId: req.ProjectID,
		Title:     req.Title,
		Priority:  req.Priority,
		CreatedBy: userID,
	}
	if req.Description != nil {
		grpcReq.Description = req.Description
	}
	if req.StatusID != nil {
		grpcReq.StatusId = req.StatusID
	}
	if req.AssigneeID != nil {
		grpcReq.AssigneeId = req.AssigneeID
	}
	if req.ParentTaskID != nil {
		grpcReq.ParentTaskId = req.ParentTaskID
	}
	if req.DueDate != nil {
		t, parseErr := parseTimestamp(*req.DueDate)
		if parseErr != nil {
			response.Error(wr, http.StatusBadRequest, "invalid due_date format")
			return
		}
		grpcReq.DueDate = t
	}
	for _, cf := range req.CustomFields {
		grpcReq.CustomFields = append(grpcReq.CustomFields, &workv1.CustomFieldValueInput{
			FieldId: cf.FieldID,
			Value:   cf.Value,
		})
	}

	resp, err := client.CreateTask(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.JSON(wr, http.StatusCreated, resp)
}

func (w *WorkRoutes) HandleGetTask(wr http.ResponseWriter, r *http.Request) {
	client, err := w.getWorkClient()
	if err != nil {
		respondServiceUnavailable(wr, w.ServiceName())
		return
	}

	taskID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(taskID); err != nil {
		response.Error(wr, http.StatusBadRequest, "invalid task id")
		return
	}

	resp, err := client.GetTask(r.Context(), &workv1.GetTaskRequest{Id: taskID})
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.JSON(wr, http.StatusOK, resp)
}

func (w *WorkRoutes) HandleListTasks(wr http.ResponseWriter, r *http.Request) {
	client, err := w.getWorkClient()
	if err != nil {
		respondServiceUnavailable(wr, w.ServiceName())
		return
	}

	q := r.URL.Query()
	page, pageSize := parsePagination(r, 1, 20)

	grpcReq := &workv1.ListTasksRequest{
		Search:           q.Get("search"),
		Page:             int32(page),
		PageSize:         int32(pageSize),
		SortBy:           q.Get("sort_by"),
		SortDesc:         q.Get("sort_desc") == "true",
		IncludeCompleted: q.Get("include_completed") == "true",
	}
	if pid := q.Get("project_id"); pid != "" {
		grpcReq.ProjectId = &pid
	}
	if aid := q.Get("assignee_id"); aid != "" {
		grpcReq.AssigneeId = &aid
	}
	if sid := q.Get("status_id"); sid != "" {
		grpcReq.StatusId = &sid
	}
	if pri := q.Get("priority"); pri != "" {
		grpcReq.Priority = &pri
	}
	if ptid := q.Get("parent_task_id"); ptid != "" {
		grpcReq.ParentTaskId = &ptid
	}

	resp, err := client.ListTasks(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.JSON(wr, http.StatusOK, resp)
}

type updateTaskRequest struct {
	Title        *string                    `json:"title,omitempty"`
	Description  *string                    `json:"description,omitempty"`
	StatusID     *string                    `json:"status_id,omitempty"`
	Priority     *string                    `json:"priority,omitempty"`
	AssigneeID   *string                    `json:"assignee_id,omitempty"`
	DueDate      *string                    `json:"due_date,omitempty"`
	CustomFields []workCustomFieldValueReq  `json:"custom_fields,omitempty"`
}

func (w *WorkRoutes) HandleUpdateTask(wr http.ResponseWriter, r *http.Request) {
	client, err := w.getWorkClient()
	if err != nil {
		respondServiceUnavailable(wr, w.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())
	taskID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(taskID); err != nil {
		response.Error(wr, http.StatusBadRequest, "invalid task id")
		return
	}

	var req updateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(wr, http.StatusBadRequest, "invalid request body")
		return
	}

	grpcReq := &workv1.UpdateTaskRequest{
		Id:        taskID,
		UpdatedBy: userID,
	}
	if req.Title != nil {
		grpcReq.Title = req.Title
	}
	if req.Description != nil {
		grpcReq.Description = req.Description
	}
	if req.StatusID != nil {
		grpcReq.StatusId = req.StatusID
	}
	if req.Priority != nil {
		grpcReq.Priority = req.Priority
	}
	if req.AssigneeID != nil {
		grpcReq.AssigneeId = req.AssigneeID
	}
	if req.DueDate != nil {
		t, parseErr := parseTimestamp(*req.DueDate)
		if parseErr != nil {
			response.Error(wr, http.StatusBadRequest, "invalid due_date format")
			return
		}
		grpcReq.DueDate = t
	}
	for _, cf := range req.CustomFields {
		grpcReq.CustomFields = append(grpcReq.CustomFields, &workv1.CustomFieldValueInput{
			FieldId: cf.FieldID,
			Value:   cf.Value,
		})
	}

	resp, err := client.UpdateTask(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.JSON(wr, http.StatusOK, resp)
}

func (w *WorkRoutes) HandleDeleteTask(wr http.ResponseWriter, r *http.Request) {
	client, err := w.getWorkClient()
	if err != nil {
		respondServiceUnavailable(wr, w.ServiceName())
		return
	}

	taskID := chi.URLParam(r, "id")
	_, err = client.DeleteTask(r.Context(), &workv1.DeleteTaskRequest{Id: taskID})
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.JSON(wr, http.StatusOK, map[string]string{"status": "task deleted"})
}

type moveTaskRequest struct {
	StatusID  string  `json:"status_id"`
	SortOrder float64 `json:"sort_order"`
}

func (w *WorkRoutes) HandleMoveTask(wr http.ResponseWriter, r *http.Request) {
	client, err := w.getWorkClient()
	if err != nil {
		respondServiceUnavailable(wr, w.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())
	taskID := chi.URLParam(r, "id")

	var req moveTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(wr, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := client.MoveTask(r.Context(), &workv1.MoveTaskRequest{
		Id:        taskID,
		StatusId:  req.StatusID,
		SortOrder: req.SortOrder,
		MovedBy:   userID,
	})
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.JSON(wr, http.StatusOK, resp)
}

func (w *WorkRoutes) HandleListSubtasks(wr http.ResponseWriter, r *http.Request) {
	client, err := w.getWorkClient()
	if err != nil {
		respondServiceUnavailable(wr, w.ServiceName())
		return
	}

	taskID := chi.URLParam(r, "id")
	recursive := r.URL.Query().Get("recursive") == "true"

	resp, err := client.ListSubtasks(r.Context(), &workv1.ListSubtasksRequest{
		ParentTaskId: taskID,
		Recursive:    recursive,
	})
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.JSON(wr, http.StatusOK, resp)
}

// ============================================================================
// Dependency Handlers
// ============================================================================

type createDependencyRequest struct {
	TargetTaskID   string `json:"target_task_id"`
	DependencyType string `json:"dependency_type"`
}

func (w *WorkRoutes) HandleCreateTaskDependency(wr http.ResponseWriter, r *http.Request) {
	client, err := w.getWorkClient()
	if err != nil {
		respondServiceUnavailable(wr, w.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())
	sourceTaskID := chi.URLParam(r, "id")

	var req createDependencyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(wr, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := client.CreateTaskDependency(r.Context(), &workv1.CreateTaskDependencyRequest{
		SourceTaskId:   sourceTaskID,
		TargetTaskId:   req.TargetTaskID,
		DependencyType: req.DependencyType,
		CreatedBy:      userID,
	})
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.JSON(wr, http.StatusCreated, resp)
}

func (w *WorkRoutes) HandleDeleteTaskDependency(wr http.ResponseWriter, r *http.Request) {
	client, err := w.getWorkClient()
	if err != nil {
		respondServiceUnavailable(wr, w.ServiceName())
		return
	}

	depID := chi.URLParam(r, "id")
	_, err = client.DeleteTaskDependency(r.Context(), &workv1.DeleteTaskDependencyRequest{Id: depID})
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.JSON(wr, http.StatusOK, map[string]string{"status": "dependency deleted"})
}

func (w *WorkRoutes) HandleListTaskDependencies(wr http.ResponseWriter, r *http.Request) {
	client, err := w.getWorkClient()
	if err != nil {
		respondServiceUnavailable(wr, w.ServiceName())
		return
	}

	taskID := chi.URLParam(r, "id")

	resp, err := client.ListTaskDependencies(r.Context(), &workv1.ListTaskDependenciesRequest{
		TaskId: taskID,
	})
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.JSON(wr, http.StatusOK, resp)
}

// ============================================================================
// Comment Handlers
// ============================================================================

type createCommentRequest struct {
	Content         string  `json:"content"`
	QuotedCommentID *string `json:"quoted_comment_id,omitempty"`
}

func (w *WorkRoutes) HandleCreateTaskComment(wr http.ResponseWriter, r *http.Request) {
	client, err := w.getWorkClient()
	if err != nil {
		respondServiceUnavailable(wr, w.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())
	taskID := chi.URLParam(r, "id")

	var req createCommentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(wr, http.StatusBadRequest, "invalid request body")
		return
	}

	grpcReq := &workv1.CreateTaskCommentRequest{
		TaskId:   taskID,
		AuthorId: userID,
		Content:  req.Content,
	}
	if req.QuotedCommentID != nil {
		grpcReq.QuotedCommentId = req.QuotedCommentID
	}

	resp, err := client.CreateTaskComment(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.JSON(wr, http.StatusCreated, resp)
}

type updateCommentRequest struct {
	Content string `json:"content"`
}

func (w *WorkRoutes) HandleUpdateTaskComment(wr http.ResponseWriter, r *http.Request) {
	client, err := w.getWorkClient()
	if err != nil {
		respondServiceUnavailable(wr, w.ServiceName())
		return
	}

	commentID := chi.URLParam(r, "id")

	var req updateCommentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(wr, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := client.UpdateTaskComment(r.Context(), &workv1.UpdateTaskCommentRequest{
		Id:      commentID,
		Content: req.Content,
	})
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.JSON(wr, http.StatusOK, resp)
}

func (w *WorkRoutes) HandleDeleteTaskComment(wr http.ResponseWriter, r *http.Request) {
	client, err := w.getWorkClient()
	if err != nil {
		respondServiceUnavailable(wr, w.ServiceName())
		return
	}

	commentID := chi.URLParam(r, "id")
	_, err = client.DeleteTaskComment(r.Context(), &workv1.DeleteTaskCommentRequest{Id: commentID})
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.JSON(wr, http.StatusOK, map[string]string{"status": "comment deleted"})
}

func (w *WorkRoutes) HandleListTaskComments(wr http.ResponseWriter, r *http.Request) {
	client, err := w.getWorkClient()
	if err != nil {
		respondServiceUnavailable(wr, w.ServiceName())
		return
	}

	taskID := chi.URLParam(r, "id")
	page, pageSize := parsePagination(r, 1, 20)

	resp, err := client.ListTaskComments(r.Context(), &workv1.ListTaskCommentsRequest{
		TaskId:   taskID,
		Page:     int32(page),
		PageSize: int32(pageSize),
	})
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.JSON(wr, http.StatusOK, resp)
}

// ============================================================================
// Entity Link Handlers
// ============================================================================

type linkEntityRequest struct {
	EntityType string `json:"entity_type"`
	EntityID   string `json:"entity_id"`
}

func (w *WorkRoutes) HandleLinkEntityToTask(wr http.ResponseWriter, r *http.Request) {
	client, err := w.getWorkClient()
	if err != nil {
		respondServiceUnavailable(wr, w.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())
	taskID := chi.URLParam(r, "id")

	var req linkEntityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(wr, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := client.LinkEntityToTask(r.Context(), &workv1.LinkEntityToTaskRequest{
		TaskId:     taskID,
		EntityType: req.EntityType,
		EntityId:   req.EntityID,
		CreatedBy:  userID,
	})
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.JSON(wr, http.StatusCreated, resp)
}

func (w *WorkRoutes) HandleUnlinkEntityFromTask(wr http.ResponseWriter, r *http.Request) {
	client, err := w.getWorkClient()
	if err != nil {
		respondServiceUnavailable(wr, w.ServiceName())
		return
	}

	linkID := chi.URLParam(r, "id")
	_, err = client.UnlinkEntityFromTask(r.Context(), &workv1.UnlinkEntityFromTaskRequest{Id: linkID})
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.JSON(wr, http.StatusOK, map[string]string{"status": "link removed"})
}

func (w *WorkRoutes) HandleListTaskEntityLinks(wr http.ResponseWriter, r *http.Request) {
	client, err := w.getWorkClient()
	if err != nil {
		respondServiceUnavailable(wr, w.ServiceName())
		return
	}

	taskID := chi.URLParam(r, "id")

	resp, err := client.ListTaskEntityLinks(r.Context(), &workv1.ListTaskEntityLinksRequest{
		TaskId: taskID,
	})
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.JSON(wr, http.StatusOK, resp)
}

func (w *WorkRoutes) HandleListEntityTasks(wr http.ResponseWriter, r *http.Request) {
	client, err := w.getWorkClient()
	if err != nil {
		respondServiceUnavailable(wr, w.ServiceName())
		return
	}

	q := r.URL.Query()
	entityType := q.Get("entity_type")
	entityID := q.Get("entity_id")
	page, pageSize := parsePagination(r, 1, 20)

	if entityType == "" || entityID == "" {
		response.Error(wr, http.StatusBadRequest, "entity_type and entity_id are required")
		return
	}

	resp, err := client.ListEntityTasks(r.Context(), &workv1.ListEntityTasksRequest{
		EntityType: entityType,
		EntityId:   entityID,
		Page:       int32(page),
		PageSize:   int32(pageSize),
	})
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.JSON(wr, http.StatusOK, resp)
}

// ============================================================================
// Activity Handler
// ============================================================================

func (w *WorkRoutes) HandleListTaskActivities(wr http.ResponseWriter, r *http.Request) {
	client, err := w.getWorkClient()
	if err != nil {
		respondServiceUnavailable(wr, w.ServiceName())
		return
	}

	taskID := chi.URLParam(r, "id")
	page, pageSize := parsePagination(r, 1, 20)

	resp, err := client.ListTaskActivities(r.Context(), &workv1.ListTaskActivitiesRequest{
		TaskId:   taskID,
		Page:     int32(page),
		PageSize: int32(pageSize),
	})
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.JSON(wr, http.StatusOK, resp)
}

// ============================================================================
// File Handlers
// ============================================================================

type attachFileRequest struct {
	Filename     string  `json:"filename"`
	MimeType     string  `json:"mime_type"`
	FileSize     int64   `json:"file_size"`
	StorageKey   string  `json:"storage_key"`
	ThumbnailKey *string `json:"thumbnail_key,omitempty"`
}

func (w *WorkRoutes) HandleAttachFileToTask(wr http.ResponseWriter, r *http.Request) {
	client, err := w.getWorkClient()
	if err != nil {
		respondServiceUnavailable(wr, w.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())
	taskID := chi.URLParam(r, "id")

	var req attachFileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(wr, http.StatusBadRequest, "invalid request body")
		return
	}

	grpcReq := &workv1.AttachFileToTaskRequest{
		TaskId:     taskID,
		Filename:   req.Filename,
		MimeType:   req.MimeType,
		FileSize:   req.FileSize,
		StorageKey: req.StorageKey,
		UploadedBy: userID,
	}
	if req.ThumbnailKey != nil {
		grpcReq.ThumbnailKey = req.ThumbnailKey
	}

	resp, err := client.AttachFileToTask(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.JSON(wr, http.StatusCreated, resp)
}

func (w *WorkRoutes) HandleRemoveTaskFile(wr http.ResponseWriter, r *http.Request) {
	client, err := w.getWorkClient()
	if err != nil {
		respondServiceUnavailable(wr, w.ServiceName())
		return
	}

	fileID := chi.URLParam(r, "id")
	_, err = client.RemoveTaskFile(r.Context(), &workv1.RemoveTaskFileRequest{Id: fileID})
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.JSON(wr, http.StatusOK, map[string]string{"status": "file removed"})
}

func (w *WorkRoutes) HandleListTaskFiles(wr http.ResponseWriter, r *http.Request) {
	client, err := w.getWorkClient()
	if err != nil {
		respondServiceUnavailable(wr, w.ServiceName())
		return
	}

	taskID := chi.URLParam(r, "id")

	resp, err := client.ListTaskFiles(r.Context(), &workv1.ListTaskFilesRequest{
		TaskId: taskID,
	})
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.JSON(wr, http.StatusOK, resp)
}

// ============================================================================
// Custom Fields Handlers
// ============================================================================

type setCustomFieldsRequest struct {
	Values []workCustomFieldValueReq `json:"values"`
}

func (w *WorkRoutes) HandleSetTaskCustomFieldValues(wr http.ResponseWriter, r *http.Request) {
	client, err := w.getWorkClient()
	if err != nil {
		respondServiceUnavailable(wr, w.ServiceName())
		return
	}

	taskID := chi.URLParam(r, "id")

	var req setCustomFieldsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(wr, http.StatusBadRequest, "invalid request body")
		return
	}

	var values []*workv1.CustomFieldValueInput
	for _, v := range req.Values {
		values = append(values, &workv1.CustomFieldValueInput{
			FieldId: v.FieldID,
			Value:   v.Value,
		})
	}

	resp, err := client.SetTaskCustomFieldValues(r.Context(), &workv1.SetTaskCustomFieldValuesRequest{
		TaskId: taskID,
		Values: values,
	})
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.JSON(wr, http.StatusOK, resp)
}

func (w *WorkRoutes) HandleGetTaskCustomFieldValues(wr http.ResponseWriter, r *http.Request) {
	client, err := w.getWorkClient()
	if err != nil {
		respondServiceUnavailable(wr, w.ServiceName())
		return
	}

	taskID := chi.URLParam(r, "id")

	resp, err := client.GetTaskCustomFieldValues(r.Context(), &workv1.GetTaskCustomFieldValuesRequest{
		TaskId: taskID,
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

// ============================================================================
// Search Handler
// ============================================================================

func (w *WorkRoutes) HandleSearchTasks(wr http.ResponseWriter, r *http.Request) {
	client, err := w.getWorkClient()
	if err != nil {
		respondServiceUnavailable(wr, w.ServiceName())
		return
	}

	q := r.URL.Query()
	query := q.Get("q")
	if query == "" {
		response.Error(wr, http.StatusBadRequest, "query parameter 'q' is required")
		return
	}

	page, pageSize := parsePagination(r, 1, 20)
	projectIDs := q["project_ids"]
	assigneeIDs := q["assignee_ids"]
	includeCompleted := q.Get("include_completed") == "true"

	resp, err := client.SearchTasks(r.Context(), &workv1.SearchTasksRequest{
		Query:            query,
		ProjectIds:       projectIDs,
		AssigneeIds:      assigneeIDs,
		IncludeCompleted: includeCompleted,
		Page:             int32(page),
		PageSize:         int32(pageSize),
	})
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.JSON(wr, http.StatusOK, resp)
}

// ============================================================================
// Helpers
// ============================================================================

// parseTimestamp parses a date string into a protobuf Timestamp.
// Supports RFC3339 and date-only formats.
func parseTimestamp(s string) (*timestamppb.Timestamp, error) {
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z",
		"2006-01-02",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return timestamppb.New(t), nil
		}
	}
	return nil, fmt.Errorf("invalid timestamp format: %s", s)
}
