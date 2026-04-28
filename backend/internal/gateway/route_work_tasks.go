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
// Task Handlers
// ============================================================================

type createTaskRequest struct {
	ProjectID    string                    `json:"project_id"`
	Title        string                    `json:"title"`
	Description  *string                   `json:"description,omitempty"`
	StatusID     *string                   `json:"status_id,omitempty"`
	Priority     string                    `json:"priority"`
	AssigneeID   *string                   `json:"assignee_id,omitempty"`
	ParentTaskID *string                   `json:"parent_task_id,omitempty"`
	DueDate      *string                   `json:"due_date,omitempty"`
	CustomFields []workCustomFieldValueReq `json:"custom_fields,omitempty"`
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

	var req createTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(wr, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Title == "" {
		response.Error(wr, http.StatusBadRequest, "title is required")
		return
	}

	createTenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(wr, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}

	grpcReq := &workv1.CreateTaskRequest{
		TenantId:  createTenantID.String(),
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

	taskID, ok := validateUUIDParam(wr, r, "id")
	if !ok {
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

	listTenantID, err := middleware.GetTenantID(r.Context())
	if err != nil {
		response.Error(wr, http.StatusUnauthorized, "missing or invalid tenant")
		return
	}

	q := r.URL.Query()
	page, pageSize := parsePagination(r, 1, 20)

	grpcReq := &workv1.ListTasksRequest{
		TenantId:         listTenantID.String(),
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
	Title        *string                   `json:"title,omitempty"`
	Description  *string                   `json:"description,omitempty"`
	StatusID     *string                   `json:"status_id,omitempty"`
	Priority     *string                   `json:"priority,omitempty"`
	AssigneeID   *string                   `json:"assignee_id,omitempty"`
	DueDate      *string                   `json:"due_date,omitempty"`
	CustomFields []workCustomFieldValueReq `json:"custom_fields,omitempty"`
}

func (w *WorkRoutes) HandleUpdateTask(wr http.ResponseWriter, r *http.Request) {
	client, err := w.getWorkClient()
	if err != nil {
		respondServiceUnavailable(wr, w.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())
	taskID, ok := validateUUIDParam(wr, r, "id")
	if !ok {
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
