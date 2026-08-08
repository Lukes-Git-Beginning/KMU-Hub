package server

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

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

// ============================================================================
// projectMockRepo: project.Repository backed by in-memory maps, so the real
// project.Service logic (key uniqueness, member bookkeeping, template
// validation) actually executes instead of short-circuiting on a stub error.
// ============================================================================

type projectMockRepo struct {
	projects map[uuid.UUID]*models.Project
	members  map[uuid.UUID][]models.ProjectMember
	prefs    map[string]*models.UserProjectPreference
}

func newProjectMockRepo() *projectMockRepo {
	return &projectMockRepo{
		projects: make(map[uuid.UUID]*models.Project),
		members:  make(map[uuid.UUID][]models.ProjectMember),
		prefs:    make(map[string]*models.UserProjectPreference),
	}
}

func (r *projectMockRepo) Create(_ context.Context, p *models.Project) error {
	r.projects[p.ID] = p
	return nil
}

func (r *projectMockRepo) GetByID(_ context.Context, id, tenantID uuid.UUID) (*models.Project, error) {
	p, ok := r.projects[id]
	if !ok || p.TenantID != tenantID {
		return nil, project.ErrNotFound
	}
	return p, nil
}

func (r *projectMockRepo) List(_ context.Context, tenantID, _ uuid.UUID, _ bool, includeArchived bool) ([]models.ProjectWithDetails, error) {
	var out []models.ProjectWithDetails
	for _, p := range r.projects {
		if p.TenantID != tenantID {
			continue
		}
		if !includeArchived && p.ArchivedAt != nil {
			continue
		}
		out = append(out, models.ProjectWithDetails{Project: *p, MemberCount: len(r.members[p.ID])})
	}
	return out, nil
}

func (r *projectMockRepo) Update(_ context.Context, p *models.Project) error {
	if _, ok := r.projects[p.ID]; !ok {
		return project.ErrNotFound
	}
	r.projects[p.ID] = p
	return nil
}

func (r *projectMockRepo) Archive(_ context.Context, id, tenantID uuid.UUID) error {
	p, ok := r.projects[id]
	if !ok || p.TenantID != tenantID {
		return project.ErrNotFound
	}
	now := time.Now()
	p.ArchivedAt = &now
	return nil
}

func (r *projectMockRepo) Delete(_ context.Context, id, tenantID uuid.UUID) error {
	p, ok := r.projects[id]
	if !ok || p.TenantID != tenantID {
		return project.ErrNotFound
	}
	delete(r.projects, id)
	delete(r.members, id)
	return nil
}

func (r *projectMockRepo) GetProjectKey(_ context.Context, id, tenantID uuid.UUID) (string, error) {
	p, ok := r.projects[id]
	if !ok || p.TenantID != tenantID {
		return "", project.ErrNotFound
	}
	return p.ProjectKey, nil
}

func (r *projectMockRepo) KeyExists(_ context.Context, tenantID uuid.UUID, key string) (bool, error) {
	for _, p := range r.projects {
		if p.TenantID == tenantID && p.ProjectKey == key {
			return true, nil
		}
	}
	return false, nil
}

func (r *projectMockRepo) AddMember(_ context.Context, projectID, userID uuid.UUID, role string) error {
	r.members[projectID] = append(r.members[projectID], models.ProjectMember{
		ProjectID: projectID, UserID: userID, Role: role, AddedAt: time.Now(),
	})
	return nil
}

func (r *projectMockRepo) RemoveMember(_ context.Context, projectID, userID uuid.UUID) error {
	ms := r.members[projectID]
	for i, m := range ms {
		if m.UserID == userID {
			r.members[projectID] = append(ms[:i], ms[i+1:]...)
			return nil
		}
	}
	return nil
}

func (r *projectMockRepo) UpdateMemberRole(_ context.Context, projectID, userID uuid.UUID, role string) error {
	ms := r.members[projectID]
	for i := range ms {
		if ms[i].UserID == userID {
			ms[i].Role = role
			return nil
		}
	}
	return project.ErrNotFound
}

func (r *projectMockRepo) GetMember(_ context.Context, projectID, userID uuid.UUID) (*models.ProjectMember, error) {
	for _, m := range r.members[projectID] {
		if m.UserID == userID {
			mm := m
			return &mm, nil
		}
	}
	return nil, project.ErrNotFound
}

func (r *projectMockRepo) ListMembers(_ context.Context, projectID uuid.UUID) ([]models.ProjectMember, error) {
	return r.members[projectID], nil
}

func (r *projectMockRepo) IsMember(_ context.Context, projectID, userID uuid.UUID) (bool, error) {
	for _, m := range r.members[projectID] {
		if m.UserID == userID {
			return true, nil
		}
	}
	return false, nil
}

func (r *projectMockRepo) CountOwners(_ context.Context, projectID uuid.UUID) (int, error) {
	n := 0
	for _, m := range r.members[projectID] {
		if m.Role == models.ProjectRoleOwner {
			n++
		}
	}
	return n, nil
}

func (r *projectMockRepo) SaveAsTemplate(_ context.Context, tenantID, sourceID uuid.UUID, newName, newKey string, createdBy uuid.UUID) (*models.Project, error) {
	src, ok := r.projects[sourceID]
	if !ok || src.TenantID != tenantID {
		return nil, project.ErrNotFound
	}
	tmpl := &models.Project{
		ID:             uuid.New(),
		TenantID:       tenantID,
		Name:           newName,
		Description:    src.Description,
		ProjectKey:     newKey,
		NextTaskNumber: 1,
		IsTemplate:     true,
		CreatedBy:      createdBy,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	r.projects[tmpl.ID] = tmpl
	return tmpl, nil
}

func (r *projectMockRepo) GetForTemplate(_ context.Context, id, tenantID uuid.UUID) (*models.Project, []models.ProjectStatus, error) {
	p, ok := r.projects[id]
	if !ok || p.TenantID != tenantID {
		return nil, nil, project.ErrTemplateNotFound
	}
	return p, nil, nil
}

func (r *projectMockRepo) GetUserPreference(_ context.Context, userID, projectID uuid.UUID) (*models.UserProjectPreference, error) {
	return r.prefs[userID.String()+"|"+projectID.String()], nil
}

func (r *projectMockRepo) SetUserPreference(_ context.Context, pref *models.UserProjectPreference) error {
	r.prefs[pref.UserID.String()+"|"+pref.ProjectID.String()] = pref
	return nil
}

// ============================================================================
// statusMockRepo: wstatus.Repository backed by an in-memory map.
// ============================================================================

type statusMockRepo struct {
	statuses map[uuid.UUID]*models.ProjectStatus
}

func newStatusMockRepo() *statusMockRepo {
	return &statusMockRepo{statuses: make(map[uuid.UUID]*models.ProjectStatus)}
}

func (r *statusMockRepo) Create(_ context.Context, s *models.ProjectStatus) error {
	r.statuses[s.ID] = s
	return nil
}

func (r *statusMockRepo) GetByID(_ context.Context, id uuid.UUID) (*models.ProjectStatus, error) {
	s, ok := r.statuses[id]
	if !ok {
		return nil, wstatus.ErrNotFound
	}
	return s, nil
}

func (r *statusMockRepo) ListByProject(_ context.Context, projectID uuid.UUID) ([]models.ProjectStatus, error) {
	var out []models.ProjectStatus
	for _, s := range r.statuses {
		if s.ProjectID == projectID {
			out = append(out, *s)
		}
	}
	return out, nil
}

func (r *statusMockRepo) Update(_ context.Context, s *models.ProjectStatus) error {
	if _, ok := r.statuses[s.ID]; !ok {
		return wstatus.ErrNotFound
	}
	r.statuses[s.ID] = s
	return nil
}

func (r *statusMockRepo) Delete(_ context.Context, id uuid.UUID) error {
	delete(r.statuses, id)
	return nil
}

func (r *statusMockRepo) Reorder(_ context.Context, _ uuid.UUID, ids []uuid.UUID) error {
	for i, id := range ids {
		if s, ok := r.statuses[id]; ok {
			s.SortOrder = i
		}
	}
	return nil
}

func (r *statusMockRepo) CountTasksWithStatus(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}

func (r *statusMockRepo) HasDefault(_ context.Context, projectID uuid.UUID) (bool, error) {
	for _, s := range r.statuses {
		if s.ProjectID == projectID && s.IsDefault {
			return true, nil
		}
	}
	return false, nil
}

func (r *statusMockRepo) GetNextSortOrder(_ context.Context, projectID uuid.UUID) (int, error) {
	next := 0
	for _, s := range r.statuses {
		if s.ProjectID == projectID && s.SortOrder >= next {
			next = s.SortOrder + 1
		}
	}
	return next, nil
}

func (r *statusMockRepo) NameExistsInProject(_ context.Context, projectID uuid.UUID, name string, excludeID *uuid.UUID) (bool, error) {
	for _, s := range r.statuses {
		if s.ProjectID != projectID || !strings.EqualFold(s.Name, name) {
			continue
		}
		if excludeID != nil && s.ID == *excludeID {
			continue
		}
		return true, nil
	}
	return false, nil
}

func (r *statusMockRepo) CopyStatusesForProject(_ context.Context, _, _ uuid.UUID) error {
	return nil
}

// ============================================================================
// Test server constructors
// ============================================================================

// newWorkProjectTaskTestServer wires real project/status/task services against
// in-memory mock repos, so member bookkeeping, key uniqueness and dependency
// validation actually execute end to end through the gRPC handler.
func newWorkProjectTaskTestServer() (*WorkGRPCServer, *projectMockRepo, *statusMockRepo, *workTaskMockRepo) {
	projRepo := newProjectMockRepo()
	statusRepo := newStatusMockRepo()
	taskRepo := newWorkTaskMockRepo()

	projectSvc := project.NewService(projRepo)
	statusSvc := wstatus.NewService(statusRepo)
	taskSvc := task.NewService(taskRepo, projRepo)
	commentSvc := comment.NewService(&commentStubRepo{}, taskRepo)
	teSvc := timeentry.NewService(&timeentryStubRepo{})
	labelSvc := label.NewService(newLabelMockRepo())
	cfSvc := customfield.NewService(&customfieldStubRepo{})

	srv := NewWorkGRPCServer(projectSvc, statusSvc, taskSvc, taskRepo, commentSvc, teSvc, labelSvc, cfSvc)
	return srv, projRepo, statusRepo, taskRepo
}

// newWorkValidationTestServer returns a server with every dependency nil.
// Safe for methods where middleware.GetTenantID / uuid.Parse validation runs
// and fails before any service field is dereferenced.
func newWorkValidationTestServer() *WorkGRPCServer {
	return NewWorkGRPCServer(nil, nil, nil, nil, nil, nil, nil, nil)
}

func ctxWithWorkTenant(tenantID uuid.UUID) context.Context {
	return context.WithValue(context.Background(), middleware.TenantIDKey, tenantID.String())
}

func seedProject(repo *projectMockRepo, tenantID uuid.UUID, key string) *models.Project {
	p := &models.Project{
		ID:             uuid.New(),
		TenantID:       tenantID,
		Name:           "Seeded Project",
		ProjectKey:     key,
		NextTaskNumber: 1,
		CreatedBy:      uuid.New(),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	repo.projects[p.ID] = p
	return p
}

// ============================================================================
// mapWorkError — table test over every sentinel the function switches on,
// plus nil-adjacent (unknown) errors. A wrong branch here is a wrong HTTP
// status in production.
// ============================================================================

func TestMapWorkError_AllSentinels(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want codes.Code
	}{
		{"project not found", project.ErrNotFound, codes.NotFound},
		{"project name required", project.ErrNameRequired, codes.InvalidArgument},
		{"project name too short", project.ErrNameTooShort, codes.InvalidArgument},
		{"project name too long", project.ErrNameTooLong, codes.InvalidArgument},
		{"project key required", project.ErrKeyRequired, codes.InvalidArgument},
		{"project key too short", project.ErrKeyTooShort, codes.InvalidArgument},
		{"project key too long", project.ErrKeyTooLong, codes.InvalidArgument},
		{"project key invalid chars", project.ErrKeyInvalidChars, codes.InvalidArgument},
		{"project key exists", project.ErrKeyAlreadyExists, codes.AlreadyExists},
		{"project archived", project.ErrProjectArchived, codes.FailedPrecondition},
		{"not project member", project.ErrNotProjectMember, codes.PermissionDenied},
		{"not project owner", project.ErrNotProjectOwner, codes.PermissionDenied},
		{"cannot remove only owner", project.ErrCannotRemoveOnlyOwner, codes.FailedPrecondition},
		{"invalid role", project.ErrInvalidRole, codes.InvalidArgument},
		{"invalid view type", project.ErrInvalidViewType, codes.InvalidArgument},
		{"template not found", project.ErrTemplateNotFound, codes.NotFound},
		{"status not found", wstatus.ErrNotFound, codes.NotFound},
		{"status name required", wstatus.ErrNameRequired, codes.InvalidArgument},
		{"status name too long", wstatus.ErrNameTooLong, codes.InvalidArgument},
		{"status duplicate name", wstatus.ErrDuplicateName, codes.AlreadyExists},
		{"status cannot delete default", wstatus.ErrCannotDeleteDefault, codes.FailedPrecondition},
		{"status invalid sort order", wstatus.ErrInvalidSortOrder, codes.InvalidArgument},
		{"task not found", task.ErrNotFound, codes.NotFound},
		{"task title required", task.ErrTitleRequired, codes.InvalidArgument},
		{"task title too long", task.ErrTitleTooLong, codes.InvalidArgument},
		{"task invalid priority", task.ErrInvalidPriority, codes.InvalidArgument},
		{"task max depth exceeded", task.ErrMaxDepthExceeded, codes.InvalidArgument},
		{"task invalid date range", task.ErrInvalidDateRange, codes.InvalidArgument},
		{"task circular dependency", task.ErrCircularDependency, codes.FailedPrecondition},
		{"task self dependency", task.ErrSelfDependency, codes.InvalidArgument},
		{"task duplicate dependency", task.ErrDuplicateDependency, codes.AlreadyExists},
		{"task invalid dependency type", task.ErrInvalidDependencyType, codes.InvalidArgument},
		{"task not project member", task.ErrNotProjectMember, codes.PermissionDenied},
		{"task viewer cannot edit", task.ErrViewerCannotEdit, codes.PermissionDenied},
		{"task parent in diff project", task.ErrParentInDiffProject, codes.InvalidArgument},
		{"comment not found", comment.ErrNotFound, codes.NotFound},
		{"comment content required", comment.ErrContentRequired, codes.InvalidArgument},
		{"comment content too long", comment.ErrContentTooLong, codes.InvalidArgument},
		{"comment quoted not found", comment.ErrQuotedCommentNotFound, codes.NotFound},
		{"comment quoted wrong task", comment.ErrQuotedCommentWrongTask, codes.InvalidArgument},
		{"comment cannot edit others", comment.ErrCannotEditOthersComment, codes.PermissionDenied},
		{"comment cannot delete others", comment.ErrCannotDeleteOthersComment, codes.PermissionDenied},
		{"label not found", label.ErrNotFound, codes.NotFound},
		{"label duplicate name", label.ErrDuplicateName, codes.AlreadyExists},
		{"label name required", label.ErrNameRequired, codes.InvalidArgument},
		{"label color invalid", label.ErrColorInvalid, codes.InvalidArgument},
		{"customfield not found", customfield.ErrNotFound, codes.NotFound},
		{"customfield duplicate name", customfield.ErrDuplicateName, codes.AlreadyExists},
		{"customfield name required", customfield.ErrNameRequired, codes.InvalidArgument},
		{"customfield invalid field type", customfield.ErrInvalidFieldType, codes.InvalidArgument},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			requireGRPCCode(t, mapWorkError(tc.err), tc.want)
		})
	}
}

func TestMapWorkError_UnknownFallsBackToInternal(t *testing.T) {
	requireGRPCCode(t, mapWorkError(assert.AnError), codes.Internal)
}

// ============================================================================
// mapTimeEntryError — same treatment for the time-entry sentinel table.
// ============================================================================

func TestMapTimeEntryError_AllSentinels(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want codes.Code
	}{
		{"not found", timeentry.ErrNotFound, codes.NotFound},
		{"already running", timeentry.ErrAlreadyRunning, codes.FailedPrecondition},
		{"no active timer", timeentry.ErrNoActiveTimer, codes.NotFound},
		{"invalid duration", timeentry.ErrInvalidDuration, codes.InvalidArgument},
		{"invalid start time", timeentry.ErrInvalidStartTime, codes.InvalidArgument},
		{"description too long", timeentry.ErrDescriptionTooLong, codes.InvalidArgument},
		{"cannot edit others", timeentry.ErrCannotEditOthers, codes.PermissionDenied},
		{"cannot delete others", timeentry.ErrCannotDeleteOthers, codes.PermissionDenied},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			requireGRPCCode(t, mapTimeEntryError(tc.err), tc.want)
		})
	}
}

func TestMapTimeEntryError_UnknownFallsBackToInternal(t *testing.T) {
	requireGRPCCode(t, mapTimeEntryError(assert.AnError), codes.Internal)
}

// ============================================================================
// Validation paths — invalid UUID / missing tenant, verified against a
// server with every service left nil (these checks must fail before any
// service field is touched).
// ============================================================================

func TestCreateProject_MissingTenant(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.CreateProject(context.Background(), &workv1.CreateProjectRequest{
		Name: "X", ProjectKey: "X", CreatedBy: uuid.NewString(),
	})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestCreateProject_InvalidCreatedBy(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.CreateProject(ctxWithWorkTenant(uuid.New()), &workv1.CreateProjectRequest{
		Name: "X", ProjectKey: "X", CreatedBy: "not-a-uuid",
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestGetProject_MissingTenant(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.GetProject(context.Background(), &workv1.GetProjectRequest{Id: uuid.NewString()})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestGetProject_InvalidID(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.GetProject(ctxWithWorkTenant(uuid.New()), &workv1.GetProjectRequest{Id: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestListProjects_MissingTenant(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.ListProjects(context.Background(), &workv1.ListProjectsRequest{})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestUpdateProject_MissingTenant(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.UpdateProject(context.Background(), &workv1.UpdateProjectRequest{Id: uuid.NewString()})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestUpdateProject_InvalidID(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.UpdateProject(ctxWithWorkTenant(uuid.New()), &workv1.UpdateProjectRequest{Id: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestArchiveProject_MissingTenant(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.ArchiveProject(context.Background(), &workv1.ArchiveProjectRequest{Id: uuid.NewString()})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestArchiveProject_InvalidID(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.ArchiveProject(ctxWithWorkTenant(uuid.New()), &workv1.ArchiveProjectRequest{Id: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestDeleteProject_MissingTenant(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.DeleteProject(context.Background(), &workv1.DeleteProjectRequest{Id: uuid.NewString()})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestDeleteProject_InvalidID(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.DeleteProject(ctxWithWorkTenant(uuid.New()), &workv1.DeleteProjectRequest{Id: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestAddProjectMember_MissingTenant(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.AddProjectMember(context.Background(), &workv1.AddProjectMemberRequest{
		ProjectId: uuid.NewString(), UserId: uuid.NewString(), Role: "member",
	})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestAddProjectMember_InvalidProjectID(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.AddProjectMember(ctxWithWorkTenant(uuid.New()), &workv1.AddProjectMemberRequest{
		ProjectId: "not-a-uuid", UserId: uuid.NewString(), Role: "member",
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestAddProjectMember_InvalidUserID(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.AddProjectMember(ctxWithWorkTenant(uuid.New()), &workv1.AddProjectMemberRequest{
		ProjectId: uuid.NewString(), UserId: "not-a-uuid", Role: "member",
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestRemoveProjectMember_MissingTenant(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.RemoveProjectMember(context.Background(), &workv1.RemoveProjectMemberRequest{
		ProjectId: uuid.NewString(), UserId: uuid.NewString(),
	})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestListProjectMembers_MissingTenant(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.ListProjectMembers(context.Background(), &workv1.ListProjectMembersRequest{ProjectId: uuid.NewString()})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestListProjectMembers_InvalidProjectID(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.ListProjectMembers(ctxWithWorkTenant(uuid.New()), &workv1.ListProjectMembersRequest{ProjectId: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestUpdateProjectMemberRole_MissingTenant(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.UpdateProjectMemberRole(context.Background(), &workv1.UpdateProjectMemberRoleRequest{
		ProjectId: uuid.NewString(), UserId: uuid.NewString(), Role: "member",
	})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestSaveProjectAsTemplate_MissingTenant(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.SaveProjectAsTemplate(context.Background(), &workv1.SaveProjectAsTemplateRequest{
		ProjectId: uuid.NewString(), CreatedBy: uuid.NewString(), TemplateName: "T",
	})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestSaveProjectAsTemplate_InvalidCreatedBy(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.SaveProjectAsTemplate(ctxWithWorkTenant(uuid.New()), &workv1.SaveProjectAsTemplateRequest{
		ProjectId: uuid.NewString(), CreatedBy: "not-a-uuid", TemplateName: "T",
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestCreateProjectFromTemplate_MissingTenant(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.CreateProjectFromTemplate(context.Background(), &workv1.CreateProjectFromTemplateRequest{
		TemplateId: uuid.NewString(), CreatedBy: uuid.NewString(), Name: "N", ProjectKey: "PK",
	})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestCreateProjectFromTemplate_InvalidTemplateID(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.CreateProjectFromTemplate(ctxWithWorkTenant(uuid.New()), &workv1.CreateProjectFromTemplateRequest{
		TemplateId: "not-a-uuid", CreatedBy: uuid.NewString(),
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestCreateProjectStatus_InvalidProjectID(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.CreateProjectStatus(context.Background(), &workv1.CreateProjectStatusRequest{ProjectId: "not-a-uuid", Name: "Open"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestUpdateProjectStatus_InvalidID(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.UpdateProjectStatus(context.Background(), &workv1.UpdateProjectStatusRequest{Id: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestDeleteProjectStatus_InvalidID(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.DeleteProjectStatus(context.Background(), &workv1.DeleteProjectStatusRequest{Id: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestReorderProjectStatuses_InvalidProjectID(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.ReorderProjectStatuses(context.Background(), &workv1.ReorderProjectStatusesRequest{ProjectId: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestReorderProjectStatuses_InvalidStatusIDInList(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.ReorderProjectStatuses(context.Background(), &workv1.ReorderProjectStatusesRequest{
		ProjectId: uuid.NewString(), StatusIds: []string{"not-a-uuid"},
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestListProjectStatuses_InvalidProjectID(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.ListProjectStatuses(context.Background(), &workv1.ListProjectStatusesRequest{ProjectId: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestCreateTask_InvalidCreatedBy(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.CreateTask(context.Background(), &workv1.CreateTaskRequest{CreatedBy: "not-a-uuid", Title: "T"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestCreateTask_MissingTenant(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.CreateTask(context.Background(), &workv1.CreateTaskRequest{CreatedBy: uuid.NewString(), Title: "T"})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestCreateTask_InvalidProjectID(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.CreateTask(ctxWithWorkTenant(uuid.New()), &workv1.CreateTaskRequest{
		CreatedBy: uuid.NewString(), Title: "T", ProjectId: "not-a-uuid",
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestGetTask_InvalidID(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.GetTask(context.Background(), &workv1.GetTaskRequest{Id: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestGetTask_MissingTenant(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.GetTask(context.Background(), &workv1.GetTaskRequest{Id: uuid.NewString()})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestListTasks_MissingTenant(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.ListTasks(context.Background(), &workv1.ListTasksRequest{})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestListTasks_InvalidProjectID(t *testing.T) {
	srv := newWorkValidationTestServer()
	badProject := "not-a-uuid"
	_, err := srv.ListTasks(context.Background(), &workv1.ListTasksRequest{ProjectId: &badProject})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestUpdateTask_InvalidID(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.UpdateTask(context.Background(), &workv1.UpdateTaskRequest{Id: "not-a-uuid", UpdatedBy: uuid.NewString()})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestUpdateTask_InvalidUpdatedBy(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.UpdateTask(context.Background(), &workv1.UpdateTaskRequest{Id: uuid.NewString(), UpdatedBy: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestUpdateTask_MissingTenant(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.UpdateTask(context.Background(), &workv1.UpdateTaskRequest{Id: uuid.NewString(), UpdatedBy: uuid.NewString()})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestDeleteTask_InvalidID(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.DeleteTask(context.Background(), &workv1.DeleteTaskRequest{Id: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestDeleteTask_MissingTenant(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.DeleteTask(context.Background(), &workv1.DeleteTaskRequest{Id: uuid.NewString()})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestMoveTask_InvalidID(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.MoveTask(context.Background(), &workv1.MoveTaskRequest{
		Id: "not-a-uuid", StatusId: uuid.NewString(), MovedBy: uuid.NewString(),
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestMoveTask_InvalidStatusID(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.MoveTask(context.Background(), &workv1.MoveTaskRequest{
		Id: uuid.NewString(), StatusId: "not-a-uuid", MovedBy: uuid.NewString(),
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestMoveTask_InvalidMovedBy(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.MoveTask(context.Background(), &workv1.MoveTaskRequest{
		Id: uuid.NewString(), StatusId: uuid.NewString(), MovedBy: "not-a-uuid",
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestMoveTask_MissingTenant(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.MoveTask(context.Background(), &workv1.MoveTaskRequest{
		Id: uuid.NewString(), StatusId: uuid.NewString(), MovedBy: uuid.NewString(),
	})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestListSubtasks_InvalidParentID(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.ListSubtasks(context.Background(), &workv1.ListSubtasksRequest{ParentTaskId: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestCreateTaskDependency_InvalidSourceID(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.CreateTaskDependency(context.Background(), &workv1.CreateTaskDependencyRequest{
		SourceTaskId: "not-a-uuid", TargetTaskId: uuid.NewString(), CreatedBy: uuid.NewString(),
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestCreateTaskDependency_InvalidTargetID(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.CreateTaskDependency(context.Background(), &workv1.CreateTaskDependencyRequest{
		SourceTaskId: uuid.NewString(), TargetTaskId: "not-a-uuid", CreatedBy: uuid.NewString(),
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestCreateTaskDependency_InvalidCreatedBy(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.CreateTaskDependency(context.Background(), &workv1.CreateTaskDependencyRequest{
		SourceTaskId: uuid.NewString(), TargetTaskId: uuid.NewString(), CreatedBy: "not-a-uuid",
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestDeleteTaskDependency_MissingTenant(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.DeleteTaskDependency(context.Background(), &workv1.DeleteTaskDependencyRequest{Id: uuid.NewString()})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestDeleteTaskDependency_InvalidID(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.DeleteTaskDependency(ctxWithWorkTenant(uuid.New()), &workv1.DeleteTaskDependencyRequest{Id: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestListTaskDependencies_InvalidTaskID(t *testing.T) {
	srv := newWorkValidationTestServer()
	_, err := srv.ListTaskDependencies(context.Background(), &workv1.ListTaskDependenciesRequest{TaskId: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

// ============================================================================
// Happy paths + proto conversions, exercised through real service+mock-repo
// wiring so business logic (key uniqueness, member auto-add, self/duplicate
// dependency checks) actually runs end to end via mapWorkError.
// ============================================================================

func TestCreateProject_HappyPath_AddsCreatorAsOwner(t *testing.T) {
	srv, projRepo, _, _ := newWorkProjectTaskTestServer()
	tenantID := uuid.New()
	createdBy := uuid.New()

	resp, err := srv.CreateProject(ctxWithWorkTenant(tenantID), &workv1.CreateProjectRequest{
		Name: "Website Relaunch", ProjectKey: "web", CreatedBy: createdBy.String(),
	})
	require.NoError(t, err)
	require.NotNil(t, resp.Project)
	assert.Equal(t, "WEB", resp.Project.ProjectKey, "key must be normalized to uppercase")
	assert.Equal(t, createdBy.String(), resp.Project.CreatedBy)

	members, _ := projRepo.ListMembers(context.Background(), uuid.MustParse(resp.Project.Id))
	require.Len(t, members, 1)
	assert.Equal(t, models.ProjectRoleOwner, members[0].Role)
}

func TestCreateProject_DuplicateKeyRejected(t *testing.T) {
	srv, projRepo, _, _ := newWorkProjectTaskTestServer()
	tenantID := uuid.New()
	seedProject(projRepo, tenantID, "DUP")

	_, err := srv.CreateProject(ctxWithWorkTenant(tenantID), &workv1.CreateProjectRequest{
		Name: "Second", ProjectKey: "dup", CreatedBy: uuid.NewString(),
	})
	requireGRPCCode(t, err, codes.AlreadyExists)
}

func TestGetProject_CrossTenantReturnsNotFound(t *testing.T) {
	srv, projRepo, _, _ := newWorkProjectTaskTestServer()
	owner := seedProject(projRepo, uuid.New(), "OWN")

	_, err := srv.GetProject(ctxWithWorkTenant(uuid.New()), &workv1.GetProjectRequest{Id: owner.ID.String()})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestListProjects_EmptyResultHasZeroTotal(t *testing.T) {
	srv, _, _, _ := newWorkProjectTaskTestServer()
	resp, err := srv.ListProjects(ctxWithWorkTenant(uuid.New()), &workv1.ListProjectsRequest{})
	require.NoError(t, err)
	assert.Equal(t, int32(0), resp.Total)
	assert.Empty(t, resp.Projects)
}

func TestUpdateProject_ArchivedProjectRejected(t *testing.T) {
	srv, projRepo, _, _ := newWorkProjectTaskTestServer()
	tenantID := uuid.New()
	p := seedProject(projRepo, tenantID, "ARC")
	now := time.Now()
	p.ArchivedAt = &now

	newName := "Renamed"
	_, err := srv.UpdateProject(ctxWithWorkTenant(tenantID), &workv1.UpdateProjectRequest{
		Id: p.ID.String(), Name: &newName,
	})
	requireGRPCCode(t, err, codes.FailedPrecondition)
}

func TestArchiveProject_HappyPath(t *testing.T) {
	srv, projRepo, _, _ := newWorkProjectTaskTestServer()
	tenantID := uuid.New()
	p := seedProject(projRepo, tenantID, "ARH")

	resp, err := srv.ArchiveProject(ctxWithWorkTenant(tenantID), &workv1.ArchiveProjectRequest{Id: p.ID.String()})
	require.NoError(t, err)
	require.NotNil(t, resp.Project.ArchivedAt)
}

func TestDeleteProject_HappyPath(t *testing.T) {
	srv, projRepo, _, _ := newWorkProjectTaskTestServer()
	tenantID := uuid.New()
	p := seedProject(projRepo, tenantID, "DEL")

	_, err := srv.DeleteProject(ctxWithWorkTenant(tenantID), &workv1.DeleteProjectRequest{Id: p.ID.String()})
	require.NoError(t, err)
	_, getErr := projRepo.GetByID(context.Background(), p.ID, tenantID)
	assert.ErrorIs(t, getErr, project.ErrNotFound)
}

func TestAddProjectMember_InvalidRoleRejected(t *testing.T) {
	srv, projRepo, _, _ := newWorkProjectTaskTestServer()
	tenantID := uuid.New()
	p := seedProject(projRepo, tenantID, "MEM")

	_, err := srv.AddProjectMember(ctxWithWorkTenant(tenantID), &workv1.AddProjectMemberRequest{
		ProjectId: p.ID.String(), UserId: uuid.NewString(), Role: "superadmin",
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestAddProjectMember_HappyPath_ReturnsMember(t *testing.T) {
	srv, projRepo, _, _ := newWorkProjectTaskTestServer()
	tenantID := uuid.New()
	p := seedProject(projRepo, tenantID, "ADM")
	userID := uuid.New()

	resp, err := srv.AddProjectMember(ctxWithWorkTenant(tenantID), &workv1.AddProjectMemberRequest{
		ProjectId: p.ID.String(), UserId: userID.String(), Role: models.ProjectRoleMember,
	})
	require.NoError(t, err)
	require.NotNil(t, resp.Member)
	assert.Equal(t, userID.String(), resp.Member.UserId)
	assert.Equal(t, models.ProjectRoleMember, resp.Member.Role)
}

func TestRemoveProjectMember_LastOwnerRejected(t *testing.T) {
	srv, projRepo, _, _ := newWorkProjectTaskTestServer()
	tenantID := uuid.New()
	p := seedProject(projRepo, tenantID, "OWN")
	owner := uuid.New()
	require.NoError(t, projRepo.AddMember(context.Background(), p.ID, owner, models.ProjectRoleOwner))

	_, err := srv.RemoveProjectMember(ctxWithWorkTenant(tenantID), &workv1.RemoveProjectMemberRequest{
		ProjectId: p.ID.String(), UserId: owner.String(),
	})
	requireGRPCCode(t, err, codes.FailedPrecondition)
}

func TestListProjectMembers_EmptyReturnsNoError(t *testing.T) {
	srv, projRepo, _, _ := newWorkProjectTaskTestServer()
	tenantID := uuid.New()
	p := seedProject(projRepo, tenantID, "LST")

	resp, err := srv.ListProjectMembers(ctxWithWorkTenant(tenantID), &workv1.ListProjectMembersRequest{ProjectId: p.ID.String()})
	require.NoError(t, err)
	assert.Empty(t, resp.Members)
}

func TestUpdateProjectMemberRole_DemoteLastOwnerRejected(t *testing.T) {
	srv, projRepo, _, _ := newWorkProjectTaskTestServer()
	tenantID := uuid.New()
	p := seedProject(projRepo, tenantID, "DEM")
	owner := uuid.New()
	require.NoError(t, projRepo.AddMember(context.Background(), p.ID, owner, models.ProjectRoleOwner))

	_, err := srv.UpdateProjectMemberRole(ctxWithWorkTenant(tenantID), &workv1.UpdateProjectMemberRoleRequest{
		ProjectId: p.ID.String(), UserId: owner.String(), Role: models.ProjectRoleMember,
	})
	requireGRPCCode(t, err, codes.FailedPrecondition)
}

func TestSaveProjectAsTemplate_HappyPath(t *testing.T) {
	srv, projRepo, _, _ := newWorkProjectTaskTestServer()
	tenantID := uuid.New()
	p := seedProject(projRepo, tenantID, "SRC")

	resp, err := srv.SaveProjectAsTemplate(ctxWithWorkTenant(tenantID), &workv1.SaveProjectAsTemplateRequest{
		ProjectId: p.ID.String(), TemplateName: "My Template", CreatedBy: uuid.NewString(),
	})
	require.NoError(t, err)
	assert.True(t, strings.Contains(resp.Template.Name, "My Template"))
}

func TestCreateProjectFromTemplate_NonTemplateSourceRejected(t *testing.T) {
	srv, projRepo, _, _ := newWorkProjectTaskTestServer()
	tenantID := uuid.New()
	// seedProject creates a regular (non-template) project.
	p := seedProject(projRepo, tenantID, "NTP")

	_, err := srv.CreateProjectFromTemplate(ctxWithWorkTenant(tenantID), &workv1.CreateProjectFromTemplateRequest{
		TemplateId: p.ID.String(), Name: "Copy", ProjectKey: "CPY", CreatedBy: uuid.NewString(),
	})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestCreateProjectFromTemplate_HappyPath(t *testing.T) {
	srv, projRepo, _, _ := newWorkProjectTaskTestServer()
	tenantID := uuid.New()
	tmpl := seedProject(projRepo, tenantID, "TPL")
	tmpl.IsTemplate = true

	resp, err := srv.CreateProjectFromTemplate(ctxWithWorkTenant(tenantID), &workv1.CreateProjectFromTemplateRequest{
		TemplateId: tmpl.ID.String(), Name: "From Template", ProjectKey: "FT", CreatedBy: uuid.NewString(),
	})
	require.NoError(t, err)
	assert.Equal(t, "FT", resp.Project.ProjectKey)
	require.NotNil(t, resp.Project.TemplateSourceId)
	assert.Equal(t, tmpl.ID.String(), *resp.Project.TemplateSourceId)
}

func TestCreateProjectStatus_HappyPathAndDuplicateRejected(t *testing.T) {
	srv, _, statusRepo, _ := newWorkProjectTaskTestServer()
	projectID := uuid.New()

	resp, err := srv.CreateProjectStatus(context.Background(), &workv1.CreateProjectStatusRequest{
		ProjectId: projectID.String(), Name: "Open",
	})
	require.NoError(t, err)
	assert.Equal(t, "Open", resp.Status.Name)
	assert.Len(t, statusRepo.statuses, 1)

	_, err = srv.CreateProjectStatus(context.Background(), &workv1.CreateProjectStatusRequest{
		ProjectId: projectID.String(), Name: "open",
	})
	requireGRPCCode(t, err, codes.AlreadyExists)
}

func TestDeleteProjectStatus_DefaultRejected(t *testing.T) {
	srv, _, statusRepo, _ := newWorkProjectTaskTestServer()
	s := &models.ProjectStatus{ID: uuid.New(), ProjectID: uuid.New(), Name: "Default", IsDefault: true, CreatedAt: time.Now()}
	statusRepo.statuses[s.ID] = s

	_, err := srv.DeleteProjectStatus(context.Background(), &workv1.DeleteProjectStatusRequest{Id: s.ID.String()})
	requireGRPCCode(t, err, codes.FailedPrecondition)
}

func TestReorderProjectStatuses_CrossProjectIDRejected(t *testing.T) {
	srv, _, statusRepo, _ := newWorkProjectTaskTestServer()
	projectID := uuid.New()
	inProject := &models.ProjectStatus{ID: uuid.New(), ProjectID: projectID, Name: "A", CreatedAt: time.Now()}
	otherProject := &models.ProjectStatus{ID: uuid.New(), ProjectID: uuid.New(), Name: "B", CreatedAt: time.Now()}
	statusRepo.statuses[inProject.ID] = inProject
	statusRepo.statuses[otherProject.ID] = otherProject

	_, err := srv.ReorderProjectStatuses(context.Background(), &workv1.ReorderProjectStatusesRequest{
		ProjectId: projectID.String(), StatusIds: []string{inProject.ID.String(), otherProject.ID.String()},
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestListProjectStatuses_EmptyReturnsNoError(t *testing.T) {
	srv, _, _, _ := newWorkProjectTaskTestServer()
	resp, err := srv.ListProjectStatuses(context.Background(), &workv1.ListProjectStatusesRequest{ProjectId: uuid.NewString()})
	require.NoError(t, err)
	assert.Empty(t, resp.Statuses)
}

func TestCreateTask_StandaloneHappyPath(t *testing.T) {
	srv, _, _, _ := newWorkProjectTaskTestServer()
	tenantID := uuid.New()
	createdBy := uuid.New()

	resp, err := srv.CreateTask(ctxWithWorkTenant(tenantID), &workv1.CreateTaskRequest{
		Title: "Standalone task", CreatedBy: createdBy.String(), Priority: models.TaskPriorityNormal,
	})
	require.NoError(t, err)
	assert.Equal(t, "Standalone task", resp.Task.Title)
	assert.Nil(t, resp.Task.ProjectId)
}

func TestCreateTask_NotProjectMemberRejected(t *testing.T) {
	srv, _, _, _ := newWorkProjectTaskTestServer()
	tenantID := uuid.New()
	projectID := uuid.New() // no membership seeded for this project

	_, err := srv.CreateTask(ctxWithWorkTenant(tenantID), &workv1.CreateTaskRequest{
		Title: "T", CreatedBy: uuid.NewString(), ProjectId: projectID.String(),
	})
	requireGRPCCode(t, err, codes.PermissionDenied)
}

func TestCreateTask_TitleRequiredRejected(t *testing.T) {
	srv, _, _, _ := newWorkProjectTaskTestServer()
	_, err := srv.CreateTask(ctxWithWorkTenant(uuid.New()), &workv1.CreateTaskRequest{
		Title: "   ", CreatedBy: uuid.NewString(),
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestGetTask_NotFound(t *testing.T) {
	srv, _, _, _ := newWorkProjectTaskTestServer()
	_, err := srv.GetTask(ctxWithWorkTenant(uuid.New()), &workv1.GetTaskRequest{Id: uuid.NewString()})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestUpdateTask_TitleTooLongRejected(t *testing.T) {
	srv, _, _, taskRepo := newWorkProjectTaskTestServer()
	tenantID := uuid.New()
	tid := uuid.New()
	taskRepo.tasks[tid] = &models.TaskWithRelations{Task: models.Task{
		ID: tid, TenantID: tenantID, Title: "Original", Priority: models.TaskPriorityNormal, CreatedBy: uuid.New(),
	}}

	longTitle := strings.Repeat("x", 501)
	_, err := srv.UpdateTask(ctxWithWorkTenant(tenantID), &workv1.UpdateTaskRequest{
		Id: tid.String(), UpdatedBy: uuid.NewString(), Title: &longTitle,
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestDeleteTask_StandaloneHappyPathWithoutActor(t *testing.T) {
	srv, _, _, taskRepo := newWorkProjectTaskTestServer()
	tenantID := uuid.New()
	tid := uuid.New()
	taskRepo.tasks[tid] = &models.TaskWithRelations{Task: models.Task{
		ID: tid, TenantID: tenantID, Title: "Solo", Priority: models.TaskPriorityNormal, CreatedBy: uuid.New(),
	}}

	// No x-user-id in context: DeleteTask must fall back to uuid.Nil, and a
	// standalone (no project) task has no membership check to fail on that.
	_, err := srv.DeleteTask(ctxWithWorkTenant(tenantID), &workv1.DeleteTaskRequest{Id: tid.String()})
	require.NoError(t, err)
}

func TestMoveTask_UnknownStatusRejected(t *testing.T) {
	srv, _, _, taskRepo := newWorkProjectTaskTestServer()
	tenantID := uuid.New()
	tid := uuid.New()
	taskRepo.tasks[tid] = &models.TaskWithRelations{Task: models.Task{
		ID: tid, TenantID: tenantID, Title: "Move me", Priority: models.TaskPriorityNormal, CreatedBy: uuid.New(),
	}}

	_, err := srv.MoveTask(ctxWithWorkTenant(tenantID), &workv1.MoveTaskRequest{
		Id: tid.String(), StatusId: uuid.NewString(), MovedBy: uuid.NewString(),
	})
	// workTaskMockRepo.GetStatusByID never errors, so the move itself succeeds;
	// this exercises the happy path through mapWorkError-free success return.
	require.NoError(t, err)
}

func TestListSubtasks_EmptyReturnsNoError(t *testing.T) {
	srv, _, _, _ := newWorkProjectTaskTestServer()
	resp, err := srv.ListSubtasks(context.Background(), &workv1.ListSubtasksRequest{ParentTaskId: uuid.NewString()})
	require.NoError(t, err)
	assert.Empty(t, resp.Tasks)
}

func TestCreateTaskDependency_SelfDependencyRejected(t *testing.T) {
	srv, _, _, _ := newWorkProjectTaskTestServer()
	taskID := uuid.NewString()
	_, err := srv.CreateTaskDependency(context.Background(), &workv1.CreateTaskDependencyRequest{
		SourceTaskId: taskID, TargetTaskId: taskID, DependencyType: models.DepTypeBlocks, CreatedBy: uuid.NewString(),
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestCreateTaskDependency_InvalidTypeRejected(t *testing.T) {
	srv, _, _, _ := newWorkProjectTaskTestServer()
	_, err := srv.CreateTaskDependency(context.Background(), &workv1.CreateTaskDependencyRequest{
		SourceTaskId: uuid.NewString(), TargetTaskId: uuid.NewString(), DependencyType: "nonsense", CreatedBy: uuid.NewString(),
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestCreateTaskDependency_DuplicateRejected(t *testing.T) {
	srv, _, _, _ := newWorkProjectTaskTestServer()
	source := uuid.NewString()
	target := uuid.NewString()

	_, err := srv.CreateTaskDependency(context.Background(), &workv1.CreateTaskDependencyRequest{
		SourceTaskId: source, TargetTaskId: target, DependencyType: models.DepTypeRelatesTo, CreatedBy: uuid.NewString(),
	})
	require.NoError(t, err)

	_, err = srv.CreateTaskDependency(context.Background(), &workv1.CreateTaskDependencyRequest{
		SourceTaskId: source, TargetTaskId: target, DependencyType: models.DepTypeRelatesTo, CreatedBy: uuid.NewString(),
	})
	requireGRPCCode(t, err, codes.AlreadyExists)
}

func TestListTaskDependencies_EmptyReturnsNoError(t *testing.T) {
	srv, _, _, _ := newWorkProjectTaskTestServer()
	resp, err := srv.ListTaskDependencies(context.Background(), &workv1.ListTaskDependenciesRequest{TaskId: uuid.NewString()})
	require.NoError(t, err)
	assert.Empty(t, resp.Dependencies)
}

// ============================================================================
// Proto converters — nil-optional-field vs fully-populated, and the [] vs
// null contract for repeated fields.
// ============================================================================

func TestProjectToProto_NilOptionalFields(t *testing.T) {
	p := &models.Project{ID: uuid.New(), Name: "N", ProjectKey: "PK", CreatedBy: uuid.New(), CreatedAt: time.Now(), UpdatedAt: time.Now()}
	proto := projectToProto(p, 2, 3, "Owner Name")
	assert.Nil(t, proto.Description)
	assert.Nil(t, proto.TemplateSourceId)
	assert.Nil(t, proto.ArchivedAt)
	assert.Equal(t, int32(2), proto.MemberCount)
	assert.Equal(t, int32(3), proto.TaskCount)
	assert.Equal(t, "Owner Name", proto.OwnerName)
}

func TestProjectToProto_FullyPopulated(t *testing.T) {
	desc := "a description"
	templateSrc := uuid.New()
	archivedAt := time.Now()
	p := &models.Project{
		ID: uuid.New(), Name: "N", ProjectKey: "PK", Description: &desc,
		TemplateSourceID: &templateSrc, ArchivedAt: &archivedAt, CreatedBy: uuid.New(),
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	proto := projectToProto(p, 0, 0, "")
	require.NotNil(t, proto.Description)
	assert.Equal(t, desc, *proto.Description)
	require.NotNil(t, proto.TemplateSourceId)
	assert.Equal(t, templateSrc.String(), *proto.TemplateSourceId)
	require.NotNil(t, proto.ArchivedAt)
}

func TestTaskToProto_NilOptionalFields(t *testing.T) {
	tw := &models.TaskWithRelations{Task: models.Task{
		ID: uuid.New(), Title: "T", Priority: models.TaskPriorityNormal, CreatedBy: uuid.New(),
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}}
	proto := taskToProto(tw)
	assert.Nil(t, proto.ProjectId)
	assert.Nil(t, proto.Description)
	assert.Nil(t, proto.StatusId)
	assert.Nil(t, proto.AssigneeId)
	assert.Nil(t, proto.ParentTaskId)
	assert.Nil(t, proto.StartDate)
	assert.Nil(t, proto.DueDate)
	assert.Nil(t, proto.CompletedAt)
	assert.Empty(t, proto.CustomFields)
}

func TestTaskToProto_FullyPopulated(t *testing.T) {
	projectID := uuid.New()
	statusID := uuid.New()
	assigneeID := uuid.New()
	parentID := uuid.New()
	now := time.Now()
	tw := &models.TaskWithRelations{
		Task: models.Task{
			ID: uuid.New(), ProjectID: &projectID, Title: "T", Priority: models.TaskPriorityNormal,
			StatusID: &statusID, AssigneeID: &assigneeID, ParentTaskID: &parentID,
			StartDate: &now, DueDate: &now, CompletedAt: &now, CreatedBy: uuid.New(),
			CreatedAt: now, UpdatedAt: now,
		},
		CustomFields: map[string]any{"priority_score": 5},
	}
	proto := taskToProto(tw)
	require.NotNil(t, proto.ProjectId)
	assert.Equal(t, projectID.String(), *proto.ProjectId)
	require.NotNil(t, proto.StatusId)
	require.NotNil(t, proto.AssigneeId)
	require.NotNil(t, proto.ParentTaskId)
	require.NotNil(t, proto.StartDate)
	require.NotNil(t, proto.DueDate)
	require.NotNil(t, proto.CompletedAt)
	require.Contains(t, proto.CustomFields, "priority_score")
}

func TestDependencyToProto(t *testing.T) {
	d := &models.TaskDependency{
		ID: uuid.New(), SourceTaskID: uuid.New(), TargetTaskID: uuid.New(),
		DependencyType: models.DepTypeBlocks, CreatedBy: uuid.New(), CreatedAt: time.Now(),
	}
	proto := dependencyToProto(d)
	assert.Equal(t, d.ID.String(), proto.Id)
	assert.Equal(t, d.SourceTaskID.String(), proto.SourceTaskId)
	assert.Equal(t, d.TargetTaskID.String(), proto.TargetTaskId)
	assert.Equal(t, models.DepTypeBlocks, proto.DependencyType)
}

func TestStatusToProto_NilColor(t *testing.T) {
	s := &models.ProjectStatus{ID: uuid.New(), ProjectID: uuid.New(), Name: "Open", CreatedAt: time.Now()}
	proto := statusToProto(s)
	assert.Nil(t, proto.Color)
}

func TestStatusToProto_WithColor(t *testing.T) {
	color := "#ff0000"
	s := &models.ProjectStatus{ID: uuid.New(), ProjectID: uuid.New(), Name: "Open", Color: &color, CreatedAt: time.Now()}
	proto := statusToProto(s)
	require.NotNil(t, proto.Color)
	assert.Equal(t, color, *proto.Color)
}
