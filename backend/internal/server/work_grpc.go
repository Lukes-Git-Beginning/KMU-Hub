package server

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/work/comment"
	"github.com/kmuhub/kmuhub/internal/work/customfield"
	"github.com/kmuhub/kmuhub/internal/work/label"
	"github.com/kmuhub/kmuhub/internal/work/project"
	wstatus "github.com/kmuhub/kmuhub/internal/work/status"
	"github.com/kmuhub/kmuhub/internal/work/task"
	"github.com/kmuhub/kmuhub/internal/work/timeentry"
	workv1 "github.com/kmuhub/kmuhub/proto/work/v1"
)

// WorkGRPCServer implements the Work gRPC service
type WorkGRPCServer struct {
	workv1.UnimplementedWorkServiceServer
	projectService      *project.Service
	statusService       *wstatus.Service
	taskService         *task.Service
	taskRepo            task.Repository
	commentService      *comment.Service
	timeEntryService    *timeentry.Service
	labelService        *label.Service
	customFieldService  *customfield.Service
}

// NewWorkGRPCServer creates a new Work gRPC server
func NewWorkGRPCServer(
	projectService *project.Service,
	statusService *wstatus.Service,
	taskService *task.Service,
	taskRepo task.Repository,
	commentService *comment.Service,
	timeEntryService *timeentry.Service,
	labelService *label.Service,
	customFieldService *customfield.Service,
) *WorkGRPCServer {
	return &WorkGRPCServer{
		projectService:     projectService,
		statusService:      statusService,
		taskService:        taskService,
		taskRepo:           taskRepo,
		commentService:     commentService,
		timeEntryService:   timeEntryService,
		labelService:       labelService,
		customFieldService: customFieldService,
	}
}

// ============================================================================
// Projects
// ============================================================================

func (s *WorkGRPCServer) CreateProject(ctx context.Context, req *workv1.CreateProjectRequest) (*workv1.CreateProjectResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	createdBy, err := uuid.Parse(req.CreatedBy)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid created_by user id")
	}

	input := project.CreateInput{
		TenantID:   tenantID,
		Name:       req.Name,
		ProjectKey: req.ProjectKey,
		CreatedBy:  createdBy,
	}
	if req.Description != nil {
		input.Description = req.Description
	}

	p, err := s.projectService.Create(ctx, input)
	if err != nil {
		return nil, mapWorkError(err)
	}

	return &workv1.CreateProjectResponse{
		Project: projectToProto(p, 0, 0, ""),
	}, nil
}

func (s *WorkGRPCServer) GetProject(ctx context.Context, req *workv1.GetProjectRequest) (*workv1.GetProjectResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid project id")
	}

	p, err := s.projectService.Get(ctx, id, tenantID, uuid.Nil, true)
	if err != nil {
		return nil, mapWorkError(err)
	}

	return &workv1.GetProjectResponse{
		Project: projectToProto(p, 0, 0, ""),
	}, nil
}

func (s *WorkGRPCServer) ListProjects(ctx context.Context, req *workv1.ListProjectsRequest) (*workv1.ListProjectsResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	projects, err := s.projectService.List(ctx, tenantID, uuid.Nil, true, req.IncludeArchived)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list projects")
	}

	var protos []*workv1.ProjectProto
	for _, p := range projects {
		protos = append(protos, projectWithDetailsToProto(&p))
	}

	return &workv1.ListProjectsResponse{
		Projects: protos,
		Total:    int32(len(projects)),
	}, nil
}

func (s *WorkGRPCServer) UpdateProject(ctx context.Context, req *workv1.UpdateProjectRequest) (*workv1.UpdateProjectResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid project id")
	}

	input := project.UpdateInput{}
	if req.Name != nil {
		input.Name = req.Name
	}
	if req.Description != nil {
		input.Description = req.Description
	}

	p, err := s.projectService.Update(ctx, id, tenantID, uuid.Nil, true, input)
	if err != nil {
		return nil, mapWorkError(err)
	}

	return &workv1.UpdateProjectResponse{
		Project: projectToProto(p, 0, 0, ""),
	}, nil
}

func (s *WorkGRPCServer) ArchiveProject(ctx context.Context, req *workv1.ArchiveProjectRequest) (*workv1.ArchiveProjectResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid project id")
	}

	if err := s.projectService.Archive(ctx, id, tenantID, uuid.Nil, true); err != nil {
		return nil, mapWorkError(err)
	}

	p, err := s.projectService.Get(ctx, id, tenantID, uuid.Nil, true)
	if err != nil {
		return nil, mapWorkError(err)
	}

	return &workv1.ArchiveProjectResponse{
		Project: projectToProto(p, 0, 0, ""),
	}, nil
}

func (s *WorkGRPCServer) DeleteProject(ctx context.Context, req *workv1.DeleteProjectRequest) (*workv1.DeleteProjectResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid project id")
	}

	if err := s.projectService.Delete(ctx, id, tenantID, uuid.Nil, true); err != nil {
		return nil, mapWorkError(err)
	}

	return &workv1.DeleteProjectResponse{}, nil
}

// ============================================================================
// Project Members
// ============================================================================

func (s *WorkGRPCServer) AddProjectMember(ctx context.Context, req *workv1.AddProjectMemberRequest) (*workv1.AddProjectMemberResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	projectID, err := uuid.Parse(req.ProjectId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid project_id")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	if err := s.projectService.AddMember(ctx, projectID, tenantID, userID, uuid.Nil, true, req.Role); err != nil {
		return nil, mapWorkError(err)
	}

	// Return the member info
	members, err := s.projectService.ListMembers(ctx, projectID, tenantID, uuid.Nil, true)
	if err != nil {
		return nil, mapWorkError(err)
	}

	for _, m := range members {
		if m.UserID == userID {
			return &workv1.AddProjectMemberResponse{
				Member: memberToProto(&m),
			}, nil
		}
	}

	return &workv1.AddProjectMemberResponse{}, nil
}

func (s *WorkGRPCServer) RemoveProjectMember(ctx context.Context, req *workv1.RemoveProjectMemberRequest) (*workv1.RemoveProjectMemberResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	projectID, err := uuid.Parse(req.ProjectId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid project_id")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	if err := s.projectService.RemoveMember(ctx, projectID, tenantID, userID, uuid.Nil, true); err != nil {
		return nil, mapWorkError(err)
	}

	return &workv1.RemoveProjectMemberResponse{}, nil
}

func (s *WorkGRPCServer) ListProjectMembers(ctx context.Context, req *workv1.ListProjectMembersRequest) (*workv1.ListProjectMembersResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	projectID, err := uuid.Parse(req.ProjectId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid project_id")
	}

	members, err := s.projectService.ListMembers(ctx, projectID, tenantID, uuid.Nil, true)
	if err != nil {
		return nil, mapWorkError(err)
	}

	var protos []*workv1.ProjectMemberProto
	for _, m := range members {
		protos = append(protos, memberToProto(&m))
	}

	return &workv1.ListProjectMembersResponse{
		Members: protos,
	}, nil
}

func (s *WorkGRPCServer) UpdateProjectMemberRole(ctx context.Context, req *workv1.UpdateProjectMemberRoleRequest) (*workv1.UpdateProjectMemberRoleResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	projectID, err := uuid.Parse(req.ProjectId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid project_id")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	if err := s.projectService.UpdateMemberRole(ctx, projectID, tenantID, userID, uuid.Nil, true, req.Role); err != nil {
		return nil, mapWorkError(err)
	}

	members, err := s.projectService.ListMembers(ctx, projectID, tenantID, uuid.Nil, true)
	if err != nil {
		return nil, mapWorkError(err)
	}

	for _, m := range members {
		if m.UserID == userID {
			return &workv1.UpdateProjectMemberRoleResponse{
				Member: memberToProto(&m),
			}, nil
		}
	}

	return &workv1.UpdateProjectMemberRoleResponse{}, nil
}

// ============================================================================
// Project Templates
// ============================================================================

func (s *WorkGRPCServer) SaveProjectAsTemplate(ctx context.Context, req *workv1.SaveProjectAsTemplateRequest) (*workv1.SaveProjectAsTemplateResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	projectID, err := uuid.Parse(req.ProjectId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid project_id")
	}

	createdBy, err := uuid.Parse(req.CreatedBy)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid created_by")
	}

	// Generate a template key from the name
	tmpl, err := s.projectService.SaveAsTemplate(ctx, tenantID, projectID, req.TemplateName, generateTemplateKey(req.TemplateName), createdBy)
	if err != nil {
		return nil, mapWorkError(err)
	}

	return &workv1.SaveProjectAsTemplateResponse{
		Template: projectToProto(tmpl, 0, 0, ""),
	}, nil
}

func (s *WorkGRPCServer) CreateProjectFromTemplate(ctx context.Context, req *workv1.CreateProjectFromTemplateRequest) (*workv1.CreateProjectFromTemplateResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	templateID, err := uuid.Parse(req.TemplateId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid template_id")
	}

	createdBy, err := uuid.Parse(req.CreatedBy)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid created_by")
	}

	p, err := s.projectService.CreateFromTemplate(ctx, tenantID, templateID, req.Name, req.ProjectKey, createdBy)
	if err != nil {
		return nil, mapWorkError(err)
	}

	return &workv1.CreateProjectFromTemplateResponse{
		Project: projectToProto(p, 0, 0, ""),
	}, nil
}

// ============================================================================
// Project Statuses
// ============================================================================

func (s *WorkGRPCServer) CreateProjectStatus(ctx context.Context, req *workv1.CreateProjectStatusRequest) (*workv1.CreateProjectStatusResponse, error) {
	projectID, err := uuid.Parse(req.ProjectId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid project_id")
	}

	input := wstatus.CreateInput{
		ProjectID: projectID,
		Name:      req.Name,
		IsDefault: req.IsDefault,
		IsClosed:  req.IsClosed,
	}
	if req.Color != nil {
		input.Color = req.Color
	}

	st, err := s.statusService.Create(ctx, input)
	if err != nil {
		return nil, mapWorkError(err)
	}

	return &workv1.CreateProjectStatusResponse{
		Status: statusToProto(st),
	}, nil
}

func (s *WorkGRPCServer) UpdateProjectStatus(ctx context.Context, req *workv1.UpdateProjectStatusRequest) (*workv1.UpdateProjectStatusResponse, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid status id")
	}

	input := wstatus.UpdateInput{}
	if req.Name != nil {
		input.Name = req.Name
	}
	if req.Color != nil {
		input.Color = req.Color
	}
	if req.IsDefault != nil {
		input.IsDefault = req.IsDefault
	}
	if req.IsClosed != nil {
		input.IsClosed = req.IsClosed
	}

	st, err := s.statusService.Update(ctx, id, input)
	if err != nil {
		return nil, mapWorkError(err)
	}

	return &workv1.UpdateProjectStatusResponse{
		Status: statusToProto(st),
	}, nil
}

func (s *WorkGRPCServer) DeleteProjectStatus(ctx context.Context, req *workv1.DeleteProjectStatusRequest) (*workv1.DeleteProjectStatusResponse, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid status id")
	}

	if err := s.statusService.Delete(ctx, id); err != nil {
		return nil, mapWorkError(err)
	}

	return &workv1.DeleteProjectStatusResponse{}, nil
}

func (s *WorkGRPCServer) ReorderProjectStatuses(ctx context.Context, req *workv1.ReorderProjectStatusesRequest) (*workv1.ReorderProjectStatusesResponse, error) {
	projectID, err := uuid.Parse(req.ProjectId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid project_id")
	}

	var statusIDs []uuid.UUID
	for _, idStr := range req.StatusIds {
		id, parseErr := uuid.Parse(idStr)
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid status_id in list")
		}
		statusIDs = append(statusIDs, id)
	}

	if err := s.statusService.Reorder(ctx, projectID, statusIDs); err != nil {
		return nil, mapWorkError(err)
	}

	statuses, err := s.statusService.ListByProject(ctx, projectID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list statuses")
	}

	var protos []*workv1.ProjectStatusProto
	for _, st := range statuses {
		protos = append(protos, statusToProto(&st))
	}

	return &workv1.ReorderProjectStatusesResponse{
		Statuses: protos,
	}, nil
}

func (s *WorkGRPCServer) ListProjectStatuses(ctx context.Context, req *workv1.ListProjectStatusesRequest) (*workv1.ListProjectStatusesResponse, error) {
	projectID, err := uuid.Parse(req.ProjectId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid project_id")
	}

	statuses, err := s.statusService.ListByProject(ctx, projectID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list statuses")
	}

	var protos []*workv1.ProjectStatusProto
	for _, st := range statuses {
		protos = append(protos, statusToProto(&st))
	}

	return &workv1.ListProjectStatusesResponse{
		Statuses: protos,
	}, nil
}

// ============================================================================
// Tasks
// ============================================================================

func (s *WorkGRPCServer) CreateTask(ctx context.Context, req *workv1.CreateTaskRequest) (*workv1.CreateTaskResponse, error) {
	createdBy, err := uuid.Parse(req.CreatedBy)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid created_by")
	}

	taskTenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	input := task.CreateInput{
		TenantID:  taskTenantID,
		Title:     req.Title,
		Priority:  req.Priority,
		CreatedBy: createdBy,
	}

	if req.ProjectId != "" {
		projectID, parseErr := uuid.Parse(req.ProjectId)
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid project_id")
		}
		input.ProjectID = &projectID
	}
	if req.Description != nil {
		input.Description = req.Description
	}
	if req.StatusId != nil {
		statusID, parseErr := uuid.Parse(*req.StatusId)
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid status_id")
		}
		input.StatusID = &statusID
	}
	if req.AssigneeId != nil {
		assigneeID, parseErr := uuid.Parse(*req.AssigneeId)
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid assignee_id")
		}
		input.AssigneeID = &assigneeID
	}
	if req.ParentTaskId != nil {
		parentTaskID, parseErr := uuid.Parse(*req.ParentTaskId)
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid parent_task_id")
		}
		input.ParentTaskID = &parentTaskID
	}
	if req.DueDate != nil {
		t := req.DueDate.AsTime()
		input.DueDate = &t
	}

	result, err := s.taskService.Create(ctx, input)
	if err != nil {
		return nil, mapWorkError(err)
	}

	// Set custom field values if provided
	if len(req.CustomFields) > 0 {
		cfMap := make(map[uuid.UUID]any)
		for _, cf := range req.CustomFields {
			fieldID, parseErr := uuid.Parse(cf.FieldId)
			if parseErr != nil {
				return nil, status.Error(codes.InvalidArgument, "invalid custom field_id")
			}
			var value any
			if unmarshalErr := json.Unmarshal([]byte(cf.Value), &value); unmarshalErr != nil {
				value = cf.Value
			}
			cfMap[fieldID] = value
		}
		_ = s.taskRepo.SetCustomFieldValues(ctx, result.ID, taskTenantID, cfMap)
		// Refresh result
		if result, err = s.taskService.GetByID(ctx, taskTenantID, result.ID); err != nil {
			return nil, mapWorkError(err)
		}
	}

	return &workv1.CreateTaskResponse{
		Task: taskToProto(result),
	}, nil
}

func (s *WorkGRPCServer) GetTask(ctx context.Context, req *workv1.GetTaskRequest) (*workv1.GetTaskResponse, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid task id")
	}

	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	t, err := s.taskService.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, mapWorkError(err)
	}

	proto := taskToProto(t)

	// Batch-load labels for this task (single query)
	labelMap, labelErr := s.labelService.GetLabelsByTaskIDs(ctx, tenantID.String(), []string{t.ID.String()})
	if labelErr == nil {
		for _, l := range labelMap[t.ID.String()] {
			proto.LabelIds = append(proto.LabelIds, l.ID)
		}
	}

	return &workv1.GetTaskResponse{
		Task: proto,
	}, nil
}

func (s *WorkGRPCServer) ListTasks(ctx context.Context, req *workv1.ListTasksRequest) (*workv1.ListTasksResponse, error) {
	filters := task.TaskFilters{
		Search:   req.Search,
		Page:     int(req.Page),
		PageSize: int(req.PageSize),
		SortBy:   req.SortBy,
		SortDesc: req.SortDesc,
	}

	if req.ProjectId != nil {
		projectID, err := uuid.Parse(*req.ProjectId)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid project_id")
		}
		filters.ProjectID = &projectID
	}
	if req.AssigneeId != nil {
		assigneeID, err := uuid.Parse(*req.AssigneeId)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid assignee_id")
		}
		filters.AssigneeID = &assigneeID
	}
	if req.StatusId != nil {
		statusID, err := uuid.Parse(*req.StatusId)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid status_id")
		}
		filters.StatusID = &statusID
	}
	if req.Priority != nil {
		filters.Priority = req.Priority
	}
	if req.ParentTaskId != nil {
		parentID, err := uuid.Parse(*req.ParentTaskId)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid parent_task_id")
		}
		filters.ParentTaskID = &parentID
	}
	if req.DueDateFrom != nil {
		t := req.DueDateFrom.AsTime()
		filters.DueDateFrom = &t
	}
	if req.DueDateTo != nil {
		t := req.DueDateTo.AsTime()
		filters.DueDateTo = &t
	}
	for _, lidStr := range req.FilterLabelIds {
		lid, err := uuid.Parse(lidStr)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid label_id in filter_label_ids")
		}
		filters.LabelIDs = append(filters.LabelIDs, lid)
	}

	taskListTenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	tasks, total, err := s.taskService.List(ctx, taskListTenantID, filters)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list tasks")
	}

	// Batch-load labels for all returned tasks (single query)
	var taskIDs []string
	for _, t := range tasks {
		taskIDs = append(taskIDs, t.ID.String())
	}
	labelMap := make(map[string][]string)
	if len(taskIDs) > 0 {
		loaded, labelErr := s.labelService.GetLabelsByTaskIDs(ctx, taskListTenantID.String(), taskIDs)
		if labelErr == nil {
			for tid, lbls := range loaded {
				for _, l := range lbls {
					labelMap[tid] = append(labelMap[tid], l.ID)
				}
			}
		}
	}

	var protos []*workv1.TaskProto
	for _, t := range tasks {
		p := taskToProto(&t)
		p.LabelIds = labelMap[t.ID.String()]
		protos = append(protos, p)
	}

	return &workv1.ListTasksResponse{
		Tasks: protos,
		Total: int32(total),
	}, nil
}

func (s *WorkGRPCServer) UpdateTask(ctx context.Context, req *workv1.UpdateTaskRequest) (*workv1.UpdateTaskResponse, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid task id")
	}

	updatedBy, err := uuid.Parse(req.UpdatedBy)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid updated_by")
	}

	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	input := task.UpdateInput{}
	if req.Title != nil {
		input.Title = req.Title
	}
	if req.Description != nil {
		input.Description = req.Description
	}
	if req.StatusId != nil {
		statusID, parseErr := uuid.Parse(*req.StatusId)
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid status_id")
		}
		input.StatusID = &statusID
	}
	if req.Priority != nil {
		input.Priority = req.Priority
	}
	if req.AssigneeId != nil {
		if *req.AssigneeId == "" {
			nilUUID := uuid.Nil
			input.AssigneeID = &nilUUID
		} else {
			assigneeID, parseErr := uuid.Parse(*req.AssigneeId)
			if parseErr != nil {
				return nil, status.Error(codes.InvalidArgument, "invalid assignee_id")
			}
			input.AssigneeID = &assigneeID
		}
	}
	if req.DueDate != nil {
		t := req.DueDate.AsTime()
		input.DueDate = &t
	}

	result, err := s.taskService.Update(ctx, tenantID, id, input, updatedBy)
	if err != nil {
		return nil, mapWorkError(err)
	}

	// Set custom field values if provided
	if len(req.CustomFields) > 0 {
		cfMap := make(map[uuid.UUID]any)
		for _, cf := range req.CustomFields {
			fieldID, parseErr := uuid.Parse(cf.FieldId)
			if parseErr != nil {
				return nil, status.Error(codes.InvalidArgument, "invalid custom field_id")
			}
			var value any
			if unmarshalErr := json.Unmarshal([]byte(cf.Value), &value); unmarshalErr != nil {
				value = cf.Value
			}
			cfMap[fieldID] = value
		}
		_ = s.taskRepo.SetCustomFieldValues(ctx, id, tenantID, cfMap)
		if result, err = s.taskService.GetByID(ctx, tenantID, id); err != nil {
			return nil, mapWorkError(err)
		}
	}

	return &workv1.UpdateTaskResponse{
		Task: taskToProto(result),
	}, nil
}

func (s *WorkGRPCServer) DeleteTask(ctx context.Context, req *workv1.DeleteTaskRequest) (*workv1.DeleteTaskResponse, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid task id")
	}

	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	// DeleteTaskRequest carries no actor field — read the caller from the
	// x-user-id metadata the gateway propagates. uuid.Nil would always fail
	// the project-membership check in the service.
	actorID := uuid.Nil
	if uid := middleware.GetUserID(ctx); uid != "" {
		if parsed, parseErr := uuid.Parse(uid); parseErr == nil {
			actorID = parsed
		}
	}

	if err := s.taskService.Delete(ctx, tenantID, id, actorID); err != nil {
		return nil, mapWorkError(err)
	}

	return &workv1.DeleteTaskResponse{}, nil
}

func (s *WorkGRPCServer) MoveTask(ctx context.Context, req *workv1.MoveTaskRequest) (*workv1.MoveTaskResponse, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid task id")
	}

	statusID, err := uuid.Parse(req.StatusId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid status_id")
	}

	movedBy, err := uuid.Parse(req.MovedBy)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid moved_by")
	}

	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	if err := s.taskService.MoveTask(ctx, tenantID, id, statusID, req.SortOrder, movedBy); err != nil {
		return nil, mapWorkError(err)
	}

	result, err := s.taskService.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, mapWorkError(err)
	}

	return &workv1.MoveTaskResponse{
		Task: taskToProto(result),
	}, nil
}

func (s *WorkGRPCServer) ListSubtasks(ctx context.Context, req *workv1.ListSubtasksRequest) (*workv1.ListSubtasksResponse, error) {
	parentID, err := uuid.Parse(req.ParentTaskId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid parent_task_id")
	}

	maxDepth := 1
	if req.Recursive {
		maxDepth = task.MaxNestingDepth
	}

	tasks, err := s.taskRepo.GetSubtasks(ctx, parentID, maxDepth)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list subtasks")
	}

	var protos []*workv1.TaskProto
	for _, t := range tasks {
		protos = append(protos, taskToProto(&t))
	}

	return &workv1.ListSubtasksResponse{
		Tasks: protos,
	}, nil
}

// ============================================================================
// Task Dependencies
// ============================================================================

func (s *WorkGRPCServer) CreateTaskDependency(ctx context.Context, req *workv1.CreateTaskDependencyRequest) (*workv1.CreateTaskDependencyResponse, error) {
	sourceID, err := uuid.Parse(req.SourceTaskId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid source_task_id")
	}

	targetID, err := uuid.Parse(req.TargetTaskId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid target_task_id")
	}

	createdBy, err := uuid.Parse(req.CreatedBy)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid created_by")
	}

	dep, err := s.taskService.CreateDependency(ctx, task.DepInput{
		SourceTaskID:   sourceID,
		TargetTaskID:   targetID,
		DependencyType: req.DependencyType,
		CreatedBy:      createdBy,
	})
	if err != nil {
		return nil, mapWorkError(err)
	}

	return &workv1.CreateTaskDependencyResponse{
		Dependency: dependencyToProto(dep),
	}, nil
}

func (s *WorkGRPCServer) DeleteTaskDependency(ctx context.Context, req *workv1.DeleteTaskDependencyRequest) (*workv1.DeleteTaskDependencyResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid dependency id")
	}

	if err := s.taskService.DeleteDependency(ctx, tenantID, id); err != nil {
		return nil, mapWorkError(err)
	}

	return &workv1.DeleteTaskDependencyResponse{}, nil
}

func (s *WorkGRPCServer) ListTaskDependencies(ctx context.Context, req *workv1.ListTaskDependenciesRequest) (*workv1.ListTaskDependenciesResponse, error) {
	taskID, err := uuid.Parse(req.TaskId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid task_id")
	}

	deps, err := s.taskRepo.ListDependencies(ctx, taskID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list dependencies")
	}

	var protos []*workv1.TaskDependencyProto
	for _, d := range deps {
		protos = append(protos, dependencyToProto(&d))
	}

	return &workv1.ListTaskDependenciesResponse{
		Dependencies: protos,
	}, nil
}

// ============================================================================
// Task Comments
// ============================================================================

func (s *WorkGRPCServer) CreateTaskComment(ctx context.Context, req *workv1.CreateTaskCommentRequest) (*workv1.CreateTaskCommentResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing or invalid tenant")
	}

	taskID, err := uuid.Parse(req.TaskId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid task_id")
	}

	authorID, err := uuid.Parse(req.AuthorId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid author_id")
	}

	input := comment.CreateInput{
		TenantID: tenantID,
		TaskID:   taskID,
		Content:  req.Content,
		AuthorID: authorID,
	}
	if req.QuotedCommentId != nil {
		quotedID, parseErr := uuid.Parse(*req.QuotedCommentId)
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid quoted_comment_id")
		}
		input.QuotedCommentID = &quotedID
	}

	c, err := s.commentService.Create(ctx, input)
	if err != nil {
		return nil, mapWorkError(err)
	}

	return &workv1.CreateTaskCommentResponse{
		Comment: commentToWorkProto(&c.TaskComment),
	}, nil
}

func (s *WorkGRPCServer) UpdateTaskComment(ctx context.Context, req *workv1.UpdateTaskCommentRequest) (*workv1.UpdateTaskCommentResponse, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid comment id")
	}

	// Use uuid.Nil as actorID since gateway handles auth
	if err := s.commentService.Update(ctx, id, req.Content, uuid.Nil); err != nil {
		return nil, mapWorkError(err)
	}

	return &workv1.UpdateTaskCommentResponse{
		Comment: &workv1.TaskCommentProto{Id: id.String()},
	}, nil
}

func (s *WorkGRPCServer) DeleteTaskComment(ctx context.Context, req *workv1.DeleteTaskCommentRequest) (*workv1.DeleteTaskCommentResponse, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid comment id")
	}

	if err := s.commentService.Delete(ctx, id, uuid.Nil, true); err != nil {
		return nil, mapWorkError(err)
	}

	return &workv1.DeleteTaskCommentResponse{}, nil
}

func (s *WorkGRPCServer) ListTaskComments(ctx context.Context, req *workv1.ListTaskCommentsRequest) (*workv1.ListTaskCommentsResponse, error) {
	taskID, err := uuid.Parse(req.TaskId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid task_id")
	}

	page := int(req.Page)
	pageSize := int(req.PageSize)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	comments, total, err := s.commentService.List(ctx, taskID, page, pageSize)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list comments")
	}

	var protos []*workv1.TaskCommentProto
	for _, c := range comments {
		protos = append(protos, commentToWorkProto(&c.TaskComment))
	}

	return &workv1.ListTaskCommentsResponse{
		Comments: protos,
		Total:    int32(total),
	}, nil
}

// ============================================================================
// Entity Links
// ============================================================================

func (s *WorkGRPCServer) LinkEntityToTask(ctx context.Context, req *workv1.LinkEntityToTaskRequest) (*workv1.LinkEntityToTaskResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing tenant")
	}

	taskID, err := uuid.Parse(req.TaskId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid task_id")
	}

	entityID, err := uuid.Parse(req.EntityId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid entity_id")
	}

	createdBy, err := uuid.Parse(req.CreatedBy)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid created_by")
	}

	link := &models.TaskEntityLink{
		ID:         uuid.New(),
		TenantID:   tenantID,
		TaskID:     taskID,
		EntityType: req.EntityType,
		EntityID:   entityID,
		CreatedBy:  createdBy,
		CreatedAt:  time.Now(),
	}

	if err := s.taskRepo.LinkEntity(ctx, link); err != nil {
		return nil, mapWorkError(err)
	}

	return &workv1.LinkEntityToTaskResponse{
		Link: entityLinkToProto(link),
	}, nil
}

func (s *WorkGRPCServer) UnlinkEntityFromTask(ctx context.Context, req *workv1.UnlinkEntityFromTaskRequest) (*workv1.UnlinkEntityFromTaskResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid link id")
	}

	if err := s.taskRepo.UnlinkEntity(ctx, tenantID, id); err != nil {
		return nil, mapWorkError(err)
	}

	return &workv1.UnlinkEntityFromTaskResponse{}, nil
}

func (s *WorkGRPCServer) ListTaskEntityLinks(ctx context.Context, req *workv1.ListTaskEntityLinksRequest) (*workv1.ListTaskEntityLinksResponse, error) {
	taskID, err := uuid.Parse(req.TaskId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid task_id")
	}

	links, err := s.taskRepo.ListEntityLinks(ctx, taskID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list entity links")
	}

	var protos []*workv1.TaskEntityLinkProto
	for _, l := range links {
		protos = append(protos, entityLinkToProto(&l))
	}

	return &workv1.ListTaskEntityLinksResponse{
		Links: protos,
	}, nil
}

func (s *WorkGRPCServer) ListEntityTasks(ctx context.Context, req *workv1.ListEntityTasksRequest) (*workv1.ListEntityTasksResponse, error) {
	entityID, err := uuid.Parse(req.EntityId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid entity_id")
	}

	tasks, err := s.taskRepo.ListTasksForEntity(ctx, req.EntityType, entityID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list entity tasks")
	}

	var protos []*workv1.TaskProto
	for _, t := range tasks {
		protos = append(protos, taskToProto(&t))
	}

	return &workv1.ListEntityTasksResponse{
		Tasks: protos,
		Total: int32(len(tasks)),
	}, nil
}

// ============================================================================
// Activity Log
// ============================================================================

func (s *WorkGRPCServer) ListTaskActivities(ctx context.Context, req *workv1.ListTaskActivitiesRequest) (*workv1.ListTaskActivitiesResponse, error) {
	taskID, err := uuid.Parse(req.TaskId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid task_id")
	}

	page := int(req.Page)
	pageSize := int(req.PageSize)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	activities, total, err := s.taskRepo.ListActivities(ctx, taskID, page, pageSize)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list activities")
	}

	var protos []*workv1.TaskActivityProto
	for _, a := range activities {
		protos = append(protos, activityToWorkProto(&a))
	}

	return &workv1.ListTaskActivitiesResponse{
		Activities: protos,
		Total:      int32(total),
	}, nil
}

// ============================================================================
// Files
// ============================================================================

func (s *WorkGRPCServer) AttachFileToTask(ctx context.Context, req *workv1.AttachFileToTaskRequest) (*workv1.AttachFileToTaskResponse, error) {
	taskID, err := uuid.Parse(req.TaskId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid task_id")
	}

	uploadedBy, err := uuid.Parse(req.UploadedBy)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid uploaded_by")
	}

	f := &models.TaskFile{
		ID:         uuid.New(),
		TaskID:     taskID,
		Filename:   req.Filename,
		MimeType:   req.MimeType,
		FileSize:   req.FileSize,
		StorageKey: req.StorageKey,
		UploadedBy: uploadedBy,
		CreatedAt:  time.Now(),
	}
	if req.ThumbnailKey != nil {
		f.ThumbnailKey = req.ThumbnailKey
	}

	if err := s.taskRepo.AttachFile(ctx, f); err != nil {
		return nil, mapWorkError(err)
	}

	return &workv1.AttachFileToTaskResponse{
		File: fileToProto(f),
	}, nil
}

func (s *WorkGRPCServer) RemoveTaskFile(ctx context.Context, req *workv1.RemoveTaskFileRequest) (*workv1.RemoveTaskFileResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid file id")
	}

	if err := s.taskRepo.RemoveFile(ctx, tenantID, id); err != nil {
		return nil, mapWorkError(err)
	}

	return &workv1.RemoveTaskFileResponse{}, nil
}

func (s *WorkGRPCServer) ListTaskFiles(ctx context.Context, req *workv1.ListTaskFilesRequest) (*workv1.ListTaskFilesResponse, error) {
	taskID, err := uuid.Parse(req.TaskId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid task_id")
	}

	files, err := s.taskRepo.ListFiles(ctx, taskID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list files")
	}

	var protos []*workv1.TaskFileProto
	for _, f := range files {
		protos = append(protos, fileToProto(&f))
	}

	return &workv1.ListTaskFilesResponse{
		Files: protos,
	}, nil
}

// ============================================================================
// Custom Fields
// ============================================================================

func (s *WorkGRPCServer) SetTaskCustomFieldValues(ctx context.Context, req *workv1.SetTaskCustomFieldValuesRequest) (*workv1.SetTaskCustomFieldValuesResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing tenant")
	}

	taskID, parseErr := uuid.Parse(req.TaskId)
	if parseErr != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid task_id")
	}

	values := make(map[uuid.UUID]any)
	for _, v := range req.Values {
		fieldID, fieldErr := uuid.Parse(v.FieldId)
		if fieldErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid field_id")
		}
		var val any
		if unmarshalErr := json.Unmarshal([]byte(v.Value), &val); unmarshalErr != nil {
			val = v.Value
		}
		values[fieldID] = val
	}

	if err := s.taskRepo.SetCustomFieldValues(ctx, taskID, tenantID, values); err != nil {
		return nil, mapWorkError(err)
	}

	cfValues, err := s.taskRepo.GetCustomFieldValues(ctx, taskID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get custom field values")
	}

	result := make(map[string]string)
	for k, v := range cfValues {
		jsonBytes, _ := json.Marshal(v)
		result[k] = string(jsonBytes)
	}

	return &workv1.SetTaskCustomFieldValuesResponse{
		CustomFields: result,
	}, nil
}

func (s *WorkGRPCServer) GetTaskCustomFieldValues(ctx context.Context, req *workv1.GetTaskCustomFieldValuesRequest) (*workv1.GetTaskCustomFieldValuesResponse, error) {
	taskID, err := uuid.Parse(req.TaskId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid task_id")
	}

	cfValues, err := s.taskRepo.GetCustomFieldValues(ctx, taskID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get custom field values")
	}

	result := make(map[string]string)
	for k, v := range cfValues {
		jsonBytes, _ := json.Marshal(v)
		result[k] = string(jsonBytes)
	}

	return &workv1.GetTaskCustomFieldValuesResponse{
		CustomFields: result,
	}, nil
}

// ============================================================================
// Preferences
// ============================================================================

func (s *WorkGRPCServer) GetUserProjectPreference(ctx context.Context, req *workv1.GetUserProjectPreferenceRequest) (*workv1.GetUserProjectPreferenceResponse, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	projectID, err := uuid.Parse(req.ProjectId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid project_id")
	}

	pref, err := s.projectService.GetUserPreference(ctx, userID, projectID)
	if err != nil {
		return nil, mapWorkError(err)
	}

	if pref == nil {
		// Return defaults
		return &workv1.GetUserProjectPreferenceResponse{
			Preference: &workv1.UserProjectPreferenceProto{
				UserId:       req.UserId,
				ProjectId:    req.ProjectId,
				ViewType:     "list",
				ListGroupBy:  "status",
				ListSortBy:   "sort_order",
				ListSortDesc: false,
			},
		}, nil
	}

	return &workv1.GetUserProjectPreferenceResponse{
		Preference: preferenceToProto(pref),
	}, nil
}

func (s *WorkGRPCServer) SetUserProjectPreference(ctx context.Context, req *workv1.SetUserProjectPreferenceRequest) (*workv1.SetUserProjectPreferenceResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing tenant")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	projectID, err := uuid.Parse(req.ProjectId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid project_id")
	}

	// Get existing or create new
	pref, _ := s.projectService.GetUserPreference(ctx, userID, projectID)
	if pref == nil {
		pref = &models.UserProjectPreference{
			TenantID:     tenantID,
			UserID:       userID,
			ProjectID:    projectID,
			ViewType:     "list",
			ListGroupBy:  "status",
			ListSortBy:   "sort_order",
			ListSortDesc: false,
		}
	} else {
		// Ensure TenantID is set on existing preferences (backfill path)
		if pref.TenantID == uuid.Nil {
			pref.TenantID = tenantID
		}
	}

	if req.ViewType != nil {
		pref.ViewType = *req.ViewType
	}
	if req.ListGroupBy != nil {
		pref.ListGroupBy = *req.ListGroupBy
	}
	if req.ListSortBy != nil {
		pref.ListSortBy = *req.ListSortBy
	}
	if req.ListSortDesc != nil {
		pref.ListSortDesc = *req.ListSortDesc
	}

	if err := s.projectService.SetUserPreference(ctx, pref); err != nil {
		return nil, mapWorkError(err)
	}

	return &workv1.SetUserProjectPreferenceResponse{
		Preference: preferenceToProto(pref),
	}, nil
}

// ============================================================================
// Search
// ============================================================================

func (s *WorkGRPCServer) SearchTasks(ctx context.Context, req *workv1.SearchTasksRequest) (*workv1.SearchTasksResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	filters := task.TaskSearchFilters{
		Page:     int(req.Page),
		PageSize: int(req.PageSize),
	}

	for _, pid := range req.ProjectIds {
		id, parseErr := uuid.Parse(pid)
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid project_id in list")
		}
		filters.ProjectIDs = append(filters.ProjectIDs, id)
	}
	for _, aid := range req.AssigneeIds {
		id, parseErr := uuid.Parse(aid)
		if parseErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid assignee_id in list")
		}
		filters.AssigneeIDs = append(filters.AssigneeIDs, id)
	}

	tasks, total, err := s.taskRepo.Search(ctx, tenantID, req.Query, filters)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to search tasks")
	}

	var protos []*workv1.TaskProto
	for _, t := range tasks {
		protos = append(protos, taskToProto(&t))
	}

	return &workv1.SearchTasksResponse{
		Tasks: protos,
		Total: int32(total),
	}, nil
}

// ============================================================================
// Proto Converters
// ============================================================================

func projectToProto(p *models.Project, memberCount, taskCount int, ownerName string) *workv1.ProjectProto {
	proto := &workv1.ProjectProto{
		Id:             p.ID.String(),
		Name:           p.Name,
		ProjectKey:     p.ProjectKey,
		NextTaskNumber: int32(p.NextTaskNumber),
		IsTemplate:     p.IsTemplate,
		CreatedBy:      p.CreatedBy.String(),
		CreatedAt:      timestamppb.New(p.CreatedAt),
		UpdatedAt:      timestamppb.New(p.UpdatedAt),
		MemberCount:    int32(memberCount),
		TaskCount:      int32(taskCount),
		OwnerName:      ownerName,
	}
	if p.Description != nil {
		proto.Description = p.Description
	}
	if p.TemplateSourceID != nil {
		s := p.TemplateSourceID.String()
		proto.TemplateSourceId = &s
	}
	if p.ArchivedAt != nil {
		proto.ArchivedAt = timestamppb.New(*p.ArchivedAt)
	}
	return proto
}

func projectWithDetailsToProto(p *models.ProjectWithDetails) *workv1.ProjectProto {
	proto := projectToProto(&p.Project, p.MemberCount, p.TaskCount, p.OwnerName)
	return proto
}

func memberToProto(m *models.ProjectMember) *workv1.ProjectMemberProto {
	return &workv1.ProjectMemberProto{
		ProjectId: m.ProjectID.String(),
		UserId:    m.UserID.String(),
		Role:      m.Role,
		AddedAt:   timestamppb.New(m.AddedAt),
		FirstName: m.FirstName,
		LastName:  m.LastName,
	}
}

func statusToProto(s *models.ProjectStatus) *workv1.ProjectStatusProto {
	proto := &workv1.ProjectStatusProto{
		Id:        s.ID.String(),
		ProjectId: s.ProjectID.String(),
		Name:      s.Name,
		SortOrder: int32(s.SortOrder),
		IsDefault: s.IsDefault,
		IsClosed:  s.IsClosed,
		CreatedAt: timestamppb.New(s.CreatedAt),
	}
	if s.Color != nil {
		proto.Color = s.Color
	}
	return proto
}

func taskToProto(t *models.TaskWithRelations) *workv1.TaskProto {
	proto := &workv1.TaskProto{
		Id:             t.ID.String(),
		TaskNumber:     int32(t.TaskNumber),
		Title:          t.Title,
		Priority:       t.Priority,
		Depth:          int32(t.Depth),
		SortOrder:      t.SortOrder,
		CreatedBy:      t.CreatedBy.String(),
		CreatedAt:      timestamppb.New(t.CreatedAt),
		UpdatedAt:      timestamppb.New(t.UpdatedAt),
		SubtaskCount:   int32(t.SubtaskCount),
		HasBlockedDeps: t.HasBlockedDeps,
	}
	if t.ProjectID != nil {
		s := t.ProjectID.String()
		proto.ProjectId = &s
	}
	if t.Description != nil {
		proto.Description = t.Description
	}
	if t.StatusID != nil {
		s := t.StatusID.String()
		proto.StatusId = &s
	}
	if t.StatusName != nil {
		proto.StatusName = t.StatusName
	}
	if t.StatusColor != nil {
		proto.StatusColor = t.StatusColor
	}
	if t.AssigneeID != nil {
		s := t.AssigneeID.String()
		proto.AssigneeId = &s
	}
	if t.AssigneeName != nil {
		proto.AssigneeName = t.AssigneeName
	}
	if t.ParentTaskID != nil {
		s := t.ParentTaskID.String()
		proto.ParentTaskId = &s
	}
	if t.DueDate != nil {
		proto.DueDate = timestamppb.New(*t.DueDate)
	}
	if t.CompletedAt != nil {
		proto.CompletedAt = timestamppb.New(*t.CompletedAt)
	}
	if t.ProjectKey != nil {
		proto.ProjectKey = t.ProjectKey
	}
	if len(t.CustomFields) > 0 {
		proto.CustomFields = make(map[string]string)
		for k, v := range t.CustomFields {
			jsonBytes, _ := json.Marshal(v)
			proto.CustomFields[k] = string(jsonBytes)
		}
	}
	return proto
}

func dependencyToProto(d *models.TaskDependency) *workv1.TaskDependencyProto {
	return &workv1.TaskDependencyProto{
		Id:             d.ID.String(),
		SourceTaskId:   d.SourceTaskID.String(),
		TargetTaskId:   d.TargetTaskID.String(),
		DependencyType: d.DependencyType,
		CreatedBy:      d.CreatedBy.String(),
		CreatedAt:      timestamppb.New(d.CreatedAt),
	}
}

func commentToWorkProto(c *models.TaskComment) *workv1.TaskCommentProto {
	proto := &workv1.TaskCommentProto{
		Id:         c.ID.String(),
		TaskId:     c.TaskID.String(),
		AuthorId:   c.AuthorID.String(),
		AuthorName: c.AuthorName,
		Content:    c.Content,
		CreatedAt:  timestamppb.New(c.CreatedAt),
		UpdatedAt:  timestamppb.New(c.UpdatedAt),
	}
	if c.QuotedCommentID != nil {
		s := c.QuotedCommentID.String()
		proto.QuotedCommentId = &s
	}
	if c.QuotedPreview != nil {
		proto.QuotedContentPreview = c.QuotedPreview
	}
	return proto
}

func activityToWorkProto(a *models.TaskActivity) *workv1.TaskActivityProto {
	proto := &workv1.TaskActivityProto{
		Id:        a.ID.String(),
		TaskId:    a.TaskID.String(),
		ActorId:   a.ActorID.String(),
		ActorName: a.ActorName,
		Action:    a.Action,
		CreatedAt: timestamppb.New(a.CreatedAt),
	}
	if a.FieldName != nil {
		proto.FieldName = a.FieldName
	}
	if a.OldValue != nil {
		s := string(*a.OldValue)
		proto.OldValue = &s
	}
	if a.NewValue != nil {
		s := string(*a.NewValue)
		proto.NewValue = &s
	}
	return proto
}

func entityLinkToProto(l *models.TaskEntityLink) *workv1.TaskEntityLinkProto {
	return &workv1.TaskEntityLinkProto{
		Id:                l.ID.String(),
		TaskId:            l.TaskID.String(),
		EntityType:        l.EntityType,
		EntityId:          l.EntityID.String(),
		EntityDisplayName: l.EntityDisplayName,
		CreatedBy:         l.CreatedBy.String(),
		CreatedAt:         timestamppb.New(l.CreatedAt),
	}
}

func fileToProto(f *models.TaskFile) *workv1.TaskFileProto {
	proto := &workv1.TaskFileProto{
		Id:             f.ID.String(),
		TaskId:         f.TaskID.String(),
		Filename:       f.Filename,
		MimeType:       f.MimeType,
		FileSize:       f.FileSize,
		StorageKey:     f.StorageKey,
		UploadedBy:     f.UploadedBy.String(),
		UploadedByName: f.UploadedByName,
		CreatedAt:      timestamppb.New(f.CreatedAt),
	}
	if f.ThumbnailKey != nil {
		proto.ThumbnailKey = f.ThumbnailKey
	}
	return proto
}

func preferenceToProto(p *models.UserProjectPreference) *workv1.UserProjectPreferenceProto {
	return &workv1.UserProjectPreferenceProto{
		UserId:       p.UserID.String(),
		ProjectId:    p.ProjectID.String(),
		ViewType:     p.ViewType,
		ListGroupBy:  p.ListGroupBy,
		ListSortBy:   p.ListSortBy,
		ListSortDesc: p.ListSortDesc,
	}
}

// ============================================================================
// Time Tracking
// ============================================================================

func (s *WorkGRPCServer) StartTimer(ctx context.Context, req *workv1.StartTimerRequest) (*workv1.StartTimerResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing or invalid tenant")
	}

	taskID, err := uuid.Parse(req.TaskId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid task_id")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	entry, stoppedEntry, err := s.timeEntryService.StartTimer(ctx, taskID, userID, tenantID)
	if err != nil {
		return nil, mapTimeEntryError(err)
	}

	resp := &workv1.StartTimerResponse{
		Entry: timeEntryToProto(entry, ""),
	}
	if stoppedEntry != nil {
		resp.StoppedEntry = timeEntryToProto(stoppedEntry, "")
	}

	return resp, nil
}

func (s *WorkGRPCServer) StopTimer(ctx context.Context, req *workv1.StopTimerRequest) (*workv1.StopTimerResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing or invalid tenant")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	entry, err := s.timeEntryService.StopTimer(ctx, userID, tenantID)
	if err != nil {
		return nil, mapTimeEntryError(err)
	}

	return &workv1.StopTimerResponse{
		Entry: timeEntryToProto(entry, ""),
	}, nil
}

func (s *WorkGRPCServer) GetActiveTimer(ctx context.Context, req *workv1.GetActiveTimerRequest) (*workv1.GetActiveTimerResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing or invalid tenant")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	timer, err := s.timeEntryService.GetActiveTimer(ctx, userID, tenantID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get active timer")
	}

	resp := &workv1.GetActiveTimerResponse{}
	if timer != nil {
		resp.Timer = &workv1.ActiveTimerProto{
			Id:        timer.ID.String(),
			TaskId:    timer.TaskID.String(),
			TaskTitle: timer.TaskTitle,
			UserId:    timer.UserID.String(),
			StartedAt: timestamppb.New(timer.StartedAt),
		}
	}

	return resp, nil
}

func (s *WorkGRPCServer) AddManualTimeEntry(ctx context.Context, req *workv1.AddManualTimeEntryRequest) (*workv1.AddManualTimeEntryResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing or invalid tenant")
	}

	taskID, err := uuid.Parse(req.TaskId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid task_id")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	input := timeentry.ManualEntryInput{
		TenantID:        tenantID,
		TaskID:          taskID,
		UserID:          userID,
		StartedAt:       req.StartedAt.AsTime(),
		DurationSeconds: int(req.DurationSeconds),
	}
	if req.Description != nil {
		input.Description = req.Description
	}

	entry, err := s.timeEntryService.AddManualEntry(ctx, input)
	if err != nil {
		return nil, mapTimeEntryError(err)
	}

	return &workv1.AddManualTimeEntryResponse{
		Entry: timeEntryToProto(entry, ""),
	}, nil
}

func (s *WorkGRPCServer) UpdateTimeEntry(ctx context.Context, req *workv1.UpdateTimeEntryRequest) (*workv1.UpdateTimeEntryResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing or invalid tenant")
	}

	entryID, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid time entry id")
	}

	actorID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	input := timeentry.UpdateEntryInput{}
	if req.StartedAt != nil {
		t := req.StartedAt.AsTime()
		input.StartedAt = &t
	}
	if req.DurationSeconds != nil {
		d := int(*req.DurationSeconds)
		input.DurationSeconds = &d
	}
	if req.Description != nil {
		input.Description = req.Description
	}

	result, err := s.timeEntryService.UpdateEntry(ctx, entryID, actorID, tenantID, input)
	if err != nil {
		return nil, mapTimeEntryError(err)
	}

	return &workv1.UpdateTimeEntryResponse{
		Entry: timeEntryWithUserToProto(result),
	}, nil
}

func (s *WorkGRPCServer) DeleteTimeEntry(ctx context.Context, req *workv1.DeleteTimeEntryRequest) (*workv1.DeleteTimeEntryResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing or invalid tenant")
	}

	entryID, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid time entry id")
	}

	actorID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	if err := s.timeEntryService.DeleteEntry(ctx, entryID, actorID, tenantID, false); err != nil {
		return nil, mapTimeEntryError(err)
	}

	return &workv1.DeleteTimeEntryResponse{}, nil
}

func (s *WorkGRPCServer) ListTimeEntries(ctx context.Context, req *workv1.ListTimeEntriesRequest) (*workv1.ListTimeEntriesResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing or invalid tenant")
	}

	taskID, err := uuid.Parse(req.TaskId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid task_id")
	}

	page := int(req.Page)
	pageSize := int(req.PageSize)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	entries, total, err := s.timeEntryService.ListByTask(ctx, taskID, tenantID, page, pageSize)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list time entries")
	}

	var protos []*workv1.TimeEntryProto
	for _, e := range entries {
		protos = append(protos, timeEntryWithUserToProto(&e))
	}

	return &workv1.ListTimeEntriesResponse{
		Entries: protos,
		Total:   int32(total),
	}, nil
}

func (s *WorkGRPCServer) GetTaskTimeSummary(ctx context.Context, req *workv1.GetTaskTimeSummaryRequest) (*workv1.GetTaskTimeSummaryResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "missing or invalid tenant")
	}

	taskID, err := uuid.Parse(req.TaskId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid task_id")
	}

	summary, err := s.timeEntryService.GetTaskTimeSummary(ctx, taskID, tenantID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get time summary")
	}

	return &workv1.GetTaskTimeSummaryResponse{
		Summary: &workv1.TimeSummaryProto{
			TaskId:               summary.TaskID.String(),
			TotalDurationSeconds: int32(summary.TotalDurationSeconds),
			EntryCount:           int32(summary.EntryCount),
		},
	}, nil
}

// ============================================================================
// Time Entry Proto Converters
// ============================================================================

func timeEntryToProto(e *models.TimeEntry, userName string) *workv1.TimeEntryProto {
	proto := &workv1.TimeEntryProto{
		Id:        e.ID.String(),
		TaskId:    e.TaskID.String(),
		UserId:    e.UserID.String(),
		UserName:  userName,
		StartedAt: timestamppb.New(e.StartedAt),
		IsManual:  e.IsManual,
		CreatedAt: timestamppb.New(e.CreatedAt),
		UpdatedAt: timestamppb.New(e.UpdatedAt),
	}
	if e.EndedAt != nil {
		proto.EndedAt = timestamppb.New(*e.EndedAt)
	}
	if e.DurationSeconds != nil {
		d := int32(*e.DurationSeconds)
		proto.DurationSeconds = &d
	}
	if e.Description != nil {
		proto.Description = e.Description
	}
	return proto
}

func timeEntryWithUserToProto(e *models.TimeEntryWithUser) *workv1.TimeEntryProto {
	proto := timeEntryToProto(&e.TimeEntry, e.UserName)
	return proto
}

// mapTimeEntryError maps time entry service errors to gRPC status errors
func mapTimeEntryError(err error) error {
	switch {
	case errors.Is(err, timeentry.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, timeentry.ErrAlreadyRunning):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, timeentry.ErrNoActiveTimer):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, timeentry.ErrInvalidDuration):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, timeentry.ErrInvalidStartTime):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, timeentry.ErrDescriptionTooLong):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, timeentry.ErrCannotEditOthers):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, timeentry.ErrCannotDeleteOthers):
		return status.Error(codes.PermissionDenied, err.Error())
	default:
		slog.Error("unhandled timeentry service error", "error", err)
		return status.Error(codes.Internal, "internal error")
	}
}

// ============================================================================
// Helpers
// ============================================================================

// generateTemplateKey creates a project key from a template name
func generateTemplateKey(name string) string {
	// Take first 6 uppercase alphanumeric chars
	key := ""
	for _, ch := range name {
		if (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') {
			if ch >= 'a' && ch <= 'z' {
				ch -= 32 // to uppercase
			}
			key += string(ch)
			if len(key) >= 6 {
				break
			}
		}
	}
	if len(key) < 2 {
		key = "TMPL"
	}
	// Add a random suffix to ensure uniqueness
	return key + uuid.New().String()[:4]
}

// ============================================================================
// Labels
// ============================================================================

func (s *WorkGRPCServer) CreateLabel(ctx context.Context, req *workv1.CreateLabelRequest) (*workv1.CreateLabelResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	color := req.Color
	if color == "" {
		color = "#6b7280"
	}

	l, err := s.labelService.Create(ctx, tenantID.String(), req.Name, color)
	if err != nil {
		return nil, mapWorkError(err)
	}
	return &workv1.CreateLabelResponse{Label: labelToProto(l)}, nil
}

func (s *WorkGRPCServer) GetLabel(ctx context.Context, req *workv1.GetLabelRequest) (*workv1.GetLabelResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	l, err := s.labelService.GetByID(ctx, tenantID.String(), req.Id)
	if err != nil {
		return nil, mapWorkError(err)
	}
	return &workv1.GetLabelResponse{Label: labelToProto(l)}, nil
}

func (s *WorkGRPCServer) ListLabels(ctx context.Context, _ *workv1.ListLabelsRequest) (*workv1.ListLabelsResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	labels, err := s.labelService.List(ctx, tenantID.String())
	if err != nil {
		return nil, mapWorkError(err)
	}

	protos := make([]*workv1.LabelProto, 0, len(labels))
	for _, l := range labels {
		protos = append(protos, labelToProto(l))
	}
	return &workv1.ListLabelsResponse{Labels: protos}, nil
}

func (s *WorkGRPCServer) UpdateLabel(ctx context.Context, req *workv1.UpdateLabelRequest) (*workv1.UpdateLabelResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	l, err := s.labelService.Update(ctx, tenantID.String(), req.Id, req.Name, req.Color)
	if err != nil {
		return nil, mapWorkError(err)
	}
	return &workv1.UpdateLabelResponse{Label: labelToProto(l)}, nil
}

func (s *WorkGRPCServer) DeleteLabel(ctx context.Context, req *workv1.DeleteLabelRequest) (*workv1.DeleteLabelResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	if err := s.labelService.Delete(ctx, tenantID.String(), req.Id); err != nil {
		return nil, mapWorkError(err)
	}
	return &workv1.DeleteLabelResponse{}, nil
}

func (s *WorkGRPCServer) SetTaskLabels(ctx context.Context, req *workv1.SetTaskLabelsRequest) (*workv1.SetTaskLabelsResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	if err := s.labelService.SetTaskLabels(ctx, tenantID.String(), req.TaskId, req.LabelIds); err != nil {
		return nil, mapWorkError(err)
	}
	return &workv1.SetTaskLabelsResponse{}, nil
}

// ============================================================================
// Custom Field Definitions
// ============================================================================

func (s *WorkGRPCServer) CreateCustomFieldDefinition(ctx context.Context, req *workv1.CreateCustomFieldDefinitionRequest) (*workv1.CreateCustomFieldDefinitionResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	d, err := s.customFieldService.Create(ctx, tenantID.String(), req.Name, req.FieldType, req.Options, int(req.Position))
	if err != nil {
		return nil, mapWorkError(err)
	}
	return &workv1.CreateCustomFieldDefinitionResponse{Definition: customFieldToProto(d)}, nil
}

func (s *WorkGRPCServer) GetCustomFieldDefinition(ctx context.Context, req *workv1.GetCustomFieldDefinitionRequest) (*workv1.GetCustomFieldDefinitionResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	d, err := s.customFieldService.GetByID(ctx, tenantID.String(), req.Id)
	if err != nil {
		return nil, mapWorkError(err)
	}
	return &workv1.GetCustomFieldDefinitionResponse{Definition: customFieldToProto(d)}, nil
}

func (s *WorkGRPCServer) ListCustomFieldDefinitions(ctx context.Context, _ *workv1.ListCustomFieldDefinitionsRequest) (*workv1.ListCustomFieldDefinitionsResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	defs, err := s.customFieldService.List(ctx, tenantID.String())
	if err != nil {
		return nil, mapWorkError(err)
	}

	protos := make([]*workv1.CustomFieldDefinitionProto, 0, len(defs))
	for _, d := range defs {
		protos = append(protos, customFieldToProto(d))
	}
	return &workv1.ListCustomFieldDefinitionsResponse{Definitions: protos}, nil
}

func (s *WorkGRPCServer) UpdateCustomFieldDefinition(ctx context.Context, req *workv1.UpdateCustomFieldDefinitionRequest) (*workv1.UpdateCustomFieldDefinitionResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	d, err := s.customFieldService.Update(ctx, tenantID.String(), req.Id, req.Name, req.FieldType, req.Options, int(req.Position))
	if err != nil {
		return nil, mapWorkError(err)
	}
	return &workv1.UpdateCustomFieldDefinitionResponse{Definition: customFieldToProto(d)}, nil
}

func (s *WorkGRPCServer) DeleteCustomFieldDefinition(ctx context.Context, req *workv1.DeleteCustomFieldDefinitionRequest) (*workv1.DeleteCustomFieldDefinitionResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	if err := s.customFieldService.Delete(ctx, tenantID.String(), req.Id); err != nil {
		return nil, mapWorkError(err)
	}
	return &workv1.DeleteCustomFieldDefinitionResponse{}, nil
}

// ============================================================================
// Proto converters
// ============================================================================

func labelToProto(l *label.Label) *workv1.LabelProto {
	return &workv1.LabelProto{
		Id:    l.ID,
		Name:  l.Name,
		Color: l.Color,
	}
}

func customFieldToProto(d *customfield.Definition) *workv1.CustomFieldDefinitionProto {
	return &workv1.CustomFieldDefinitionProto{
		Id:        d.ID,
		Name:      d.Name,
		FieldType: d.FieldType,
		Options:   d.Options,
		Position:  int32(d.Position),
	}
}

// mapWorkError maps work service errors to gRPC status errors
func mapWorkError(err error) error {
	switch {
	// Project errors
	case errors.Is(err, project.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, project.ErrNameRequired):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, project.ErrNameTooShort):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, project.ErrNameTooLong):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, project.ErrKeyRequired):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, project.ErrKeyTooShort):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, project.ErrKeyTooLong):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, project.ErrKeyInvalidChars):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, project.ErrKeyAlreadyExists):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, project.ErrProjectArchived):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, project.ErrNotProjectMember):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, project.ErrNotProjectOwner):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, project.ErrCannotRemoveOnlyOwner):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, project.ErrInvalidRole):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, project.ErrInvalidViewType):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, project.ErrTemplateNotFound):
		return status.Error(codes.NotFound, err.Error())
	// Status errors
	case errors.Is(err, wstatus.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, wstatus.ErrNameRequired):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, wstatus.ErrNameTooLong):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, wstatus.ErrDuplicateName):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, wstatus.ErrCannotDeleteDefault):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, wstatus.ErrInvalidSortOrder):
		return status.Error(codes.InvalidArgument, err.Error())
	// Task errors
	case errors.Is(err, task.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, task.ErrTitleRequired):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, task.ErrTitleTooLong):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, task.ErrInvalidPriority):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, task.ErrMaxDepthExceeded):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, task.ErrCircularDependency):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, task.ErrSelfDependency):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, task.ErrDuplicateDependency):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, task.ErrInvalidDependencyType):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, task.ErrNotProjectMember):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, task.ErrViewerCannotEdit):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, task.ErrParentInDiffProject):
		return status.Error(codes.InvalidArgument, err.Error())
	// Comment errors
	case errors.Is(err, comment.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, comment.ErrContentRequired):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, comment.ErrContentTooLong):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, comment.ErrQuotedCommentNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, comment.ErrQuotedCommentWrongTask):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, comment.ErrCannotEditOthersComment):
		return status.Error(codes.PermissionDenied, err.Error())
	case errors.Is(err, comment.ErrCannotDeleteOthersComment):
		return status.Error(codes.PermissionDenied, err.Error())
	// Label errors
	case errors.Is(err, label.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, label.ErrDuplicateName):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, label.ErrNameRequired):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, label.ErrColorInvalid):
		return status.Error(codes.InvalidArgument, err.Error())
	// Custom field errors
	case errors.Is(err, customfield.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, customfield.ErrDuplicateName):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, customfield.ErrNameRequired):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, customfield.ErrInvalidFieldType):
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		slog.Error("unhandled work service error", "error", err)
		return status.Error(codes.Internal, "internal error")
	}
}
