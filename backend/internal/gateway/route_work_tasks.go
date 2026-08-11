package gateway

import (
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
	ProjectID    string                    `json:"project_id" validate:"required,uuid"`
	Title        string                    `json:"title" validate:"required"`
	Description  *string                   `json:"description,omitempty"`
	StatusID     *string                   `json:"status_id,omitempty" validate:"omitempty,uuid"`
	Priority     string                    `json:"priority" validate:"omitempty,oneof=low normal high urgent"`
	AssigneeID   *string                   `json:"assignee_id,omitempty" validate:"omitempty,uuid"`
	ParentTaskID *string                   `json:"parent_task_id,omitempty" validate:"omitempty,uuid"`
	StartDate    *string                   `json:"start_date,omitempty"`
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

	req, ok := decodeAndValidate[createTaskRequest](wr, r)
	if !ok {
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
	if req.StartDate != nil {
		t, parseErr := parseTimestamp(*req.StartDate)
		if parseErr != nil {
			response.Error(wr, http.StatusBadRequest, "invalid start_date format")
			return
		}
		grpcReq.StartDate = t
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

	response.Proto(wr, http.StatusCreated, resp)
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

	response.Proto(wr, http.StatusOK, resp)
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

	ownerID, ok := ownerFilterForScopeAny(wr, r, [2]string{"tasks", "read"}, [2]string{"work:task", "read"})
	if !ok {
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
	if ownerID != nil {
		// Overrides any client-supplied assignee_id — a scope of "own" is a
		// boundary, not a suggestion the caller can widen with a query param.
		grpcReq.AssigneeId = ownerID
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
	if dueFrom := q.Get("due_from"); dueFrom != "" {
		t, parseErr := parseTimestamp(dueFrom)
		if parseErr != nil {
			response.Error(wr, http.StatusBadRequest, "invalid due_from format")
			return
		}
		grpcReq.DueDateFrom = t
	}
	if dueTo := q.Get("due_to"); dueTo != "" {
		t, parseErr := parseTimestamp(dueTo)
		if parseErr != nil {
			response.Error(wr, http.StatusBadRequest, "invalid due_to format")
			return
		}
		grpcReq.DueDateTo = t
	}
	if labelIDs := labelIDsFromQuery(r, "label_ids"); len(labelIDs) > 0 {
		grpcReq.FilterLabelIds = labelIDs
	}

	resp, err := client.ListTasks(r.Context(), grpcReq)
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.Proto(wr, http.StatusOK, resp)
}

type updateTaskRequest struct {
	Title        *string                   `json:"title,omitempty" validate:"omitempty,min=1"`
	Description  *string                   `json:"description,omitempty"`
	StatusID     *string                   `json:"status_id,omitempty" validate:"omitempty,uuid"`
	Priority     *string                   `json:"priority,omitempty" validate:"omitempty,oneof=low normal high urgent"`
	AssigneeID   *string                   `json:"assignee_id,omitempty" validate:"omitempty,uuid"`
	StartDate    *string                   `json:"start_date,omitempty"`
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

	req, ok := decodeAndValidate[updateTaskRequest](wr, r)
	if !ok {
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
	if req.StartDate != nil {
		t, parseErr := parseTimestamp(*req.StartDate)
		if parseErr != nil {
			response.Error(wr, http.StatusBadRequest, "invalid start_date format")
			return
		}
		grpcReq.StartDate = t
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

	response.Proto(wr, http.StatusOK, resp)
}

func (w *WorkRoutes) HandleDeleteTask(wr http.ResponseWriter, r *http.Request) {
	client, err := w.getWorkClient()
	if err != nil {
		respondServiceUnavailable(wr, w.ServiceName())
		return
	}

	taskID, ok := validateUUIDParam(wr, r, "id")
	if !ok {
		return
	}
	_, err = client.DeleteTask(r.Context(), &workv1.DeleteTaskRequest{Id: taskID})
	if err != nil {
		respondGRPCError(wr, err)
		return
	}

	response.JSON(wr, http.StatusOK, map[string]string{"status": "task deleted"})
}

type moveTaskRequest struct {
	StatusID  string  `json:"status_id" validate:"required,uuid"`
	SortOrder float64 `json:"sort_order"`
}

func (w *WorkRoutes) HandleMoveTask(wr http.ResponseWriter, r *http.Request) {
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

	req, ok := decodeAndValidate[moveTaskRequest](wr, r)
	if !ok {
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

	response.Proto(wr, http.StatusOK, resp)
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

	response.Proto(wr, http.StatusOK, resp)
}

// ============================================================================
// Dependency Handlers
// ============================================================================

type createDependencyRequest struct {
	TargetTaskID   string `json:"target_task_id" validate:"required,uuid"`
	DependencyType string `json:"dependency_type" validate:"omitempty,oneof=blocks blocked_by relates_to duplicates"`
}

func (w *WorkRoutes) HandleCreateTaskDependency(wr http.ResponseWriter, r *http.Request) {
	client, err := w.getWorkClient()
	if err != nil {
		respondServiceUnavailable(wr, w.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())
	sourceTaskID := chi.URLParam(r, "id")

	req, ok := decodeAndValidate[createDependencyRequest](wr, r)
	if !ok {
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

	response.Proto(wr, http.StatusCreated, resp)
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

	response.Proto(wr, http.StatusOK, resp)
}

// ============================================================================
// Comment Handlers
// ============================================================================

type createCommentRequest struct {
	Content         string  `json:"content" validate:"required"`
	QuotedCommentID *string `json:"quoted_comment_id,omitempty" validate:"omitempty,uuid"`
}

func (w *WorkRoutes) HandleCreateTaskComment(wr http.ResponseWriter, r *http.Request) {
	client, err := w.getWorkClient()
	if err != nil {
		respondServiceUnavailable(wr, w.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())
	taskID := chi.URLParam(r, "id")

	req, ok := decodeAndValidate[createCommentRequest](wr, r)
	if !ok {
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

	response.Proto(wr, http.StatusCreated, resp)
}

type updateCommentRequest struct {
	Content string `json:"content" validate:"required"`
}

func (w *WorkRoutes) HandleUpdateTaskComment(wr http.ResponseWriter, r *http.Request) {
	client, err := w.getWorkClient()
	if err != nil {
		respondServiceUnavailable(wr, w.ServiceName())
		return
	}

	commentID := chi.URLParam(r, "id")

	req, ok := decodeAndValidate[updateCommentRequest](wr, r)
	if !ok {
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

	response.Proto(wr, http.StatusOK, resp)
}

func (w *WorkRoutes) HandleDeleteTaskComment(wr http.ResponseWriter, r *http.Request) {
	client, err := w.getWorkClient()
	if err != nil {
		respondServiceUnavailable(wr, w.ServiceName())
		return
	}

	commentID := chi.URLParam(r, "id")
	_, err = client.DeleteTaskComment(r.Context(), &workv1.DeleteTaskCommentRequest{
		Id:      commentID,
		IsAdmin: middleware.IsAdmin(r.Context()),
	})
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

	response.Proto(wr, http.StatusOK, resp)
}

// ============================================================================
// Entity Link Handlers
// ============================================================================

type linkEntityRequest struct {
	EntityType string `json:"entity_type" validate:"required"`
	EntityID   string `json:"entity_id" validate:"required,uuid"`
}

func (w *WorkRoutes) HandleLinkEntityToTask(wr http.ResponseWriter, r *http.Request) {
	client, err := w.getWorkClient()
	if err != nil {
		respondServiceUnavailable(wr, w.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())
	taskID := chi.URLParam(r, "id")

	req, ok := decodeAndValidate[linkEntityRequest](wr, r)
	if !ok {
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

	response.Proto(wr, http.StatusCreated, resp)
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

	response.Proto(wr, http.StatusOK, resp)
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

	response.Proto(wr, http.StatusOK, resp)
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

	response.Proto(wr, http.StatusOK, resp)
}

// ============================================================================
// File Handlers
// ============================================================================

type attachFileRequest struct {
	Filename     string  `json:"filename" validate:"required"`
	MimeType     string  `json:"mime_type" validate:"required"`
	FileSize     int64   `json:"file_size" validate:"gt=0"`
	StorageKey   string  `json:"storage_key" validate:"required"`
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

	req, ok := decodeAndValidate[attachFileRequest](wr, r)
	if !ok {
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

	response.Proto(wr, http.StatusCreated, resp)
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

	response.Proto(wr, http.StatusOK, resp)
}

// ============================================================================
// Custom Fields Handlers
// ============================================================================

type setCustomFieldsRequest struct {
	Values []workCustomFieldValueReq `json:"values" validate:"required,min=1"`
}

func (w *WorkRoutes) HandleSetTaskCustomFieldValues(wr http.ResponseWriter, r *http.Request) {
	client, err := w.getWorkClient()
	if err != nil {
		respondServiceUnavailable(wr, w.ServiceName())
		return
	}

	taskID := chi.URLParam(r, "id")

	req, ok := decodeAndValidate[setCustomFieldsRequest](wr, r)
	if !ok {
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

	response.Proto(wr, http.StatusOK, resp)
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

	response.Proto(wr, http.StatusOK, resp)
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

	response.Proto(wr, http.StatusOK, resp)
}
