package server

import (
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/kmuhub/kmuhub/internal/models"
	workv1 "github.com/kmuhub/kmuhub/proto/work/v1"
)

// loggedForProject returns the first captured record with this message for this project.
func (c *attrCapture) loggedForProject(msg string, projectID uuid.UUID) map[string]any {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, r := range c.records {
		if r["msg"] == msg && r["project_id"] == projectID {
			return r
		}
	}
	return nil
}

// withCapturedSlog swaps the default logger for a capturing one until the test ends.
func withCapturedSlog(t *testing.T) *attrCapture {
	t.Helper()
	capture := &attrCapture{}
	prev := slog.Default()
	slog.SetDefault(slog.New(capture))
	t.Cleanup(func() { slog.SetDefault(prev) })
	return capture
}

// The member mutations carry the calling user to the one place the project service
// records an actor: the member log line. uuid.Nil there made every entry anonymous.
func TestWorkGRPCServer_MemberMutationsLogTheCaller(t *testing.T) {
	tenantID, actor, target := uuid.New(), uuid.New(), uuid.New()
	srv, repo, _, _ := newWorkProjectTaskTestServer()
	p := seedProject(repo, tenantID, "ACT")
	repo.members[p.ID] = []models.ProjectMember{
		{ProjectID: p.ID, UserID: uuid.New(), Role: models.ProjectRoleOwner},
	}
	ctx := ctxWithTenantAndUser(tenantID, actor)

	capture := withCapturedSlog(t)

	_, err := srv.AddProjectMember(ctx, &workv1.AddProjectMemberRequest{
		ProjectId: p.ID.String(), UserId: target.String(), Role: models.ProjectRoleMember,
	})
	require.NoError(t, err)
	_, err = srv.UpdateProjectMemberRole(ctx, &workv1.UpdateProjectMemberRoleRequest{
		ProjectId: p.ID.String(), UserId: target.String(), Role: models.ProjectRoleViewer,
	})
	require.NoError(t, err)
	_, err = srv.RemoveProjectMember(ctx, &workv1.RemoveProjectMemberRequest{
		ProjectId: p.ID.String(), UserId: target.String(),
	})
	require.NoError(t, err)

	for _, msg := range []string{"project member added", "project member role updated", "project member removed"} {
		rec := capture.loggedForProject(msg, p.ID)
		require.NotNil(t, rec, msg)
		assert.Equal(t, actor, rec["actor_id"], msg)
		assert.Equal(t, target, rec["user_id"], msg)
	}
}

// Projects are open to everyone with project permissions (decision 2026-10-23): a
// caller who is a member of nothing still reads, lists and sees the members.
func TestWorkGRPCServer_NonMemberSeesTenantProjects(t *testing.T) {
	tenantID := uuid.New()
	srv, repo, _, _ := newWorkProjectTaskTestServer()
	p := seedProject(repo, tenantID, "OPN")
	repo.members[p.ID] = []models.ProjectMember{
		{ProjectID: p.ID, UserID: uuid.New(), Role: models.ProjectRoleOwner},
	}
	ctx := ctxWithTenantAndUser(tenantID, uuid.New()) // not in the member list

	got, err := srv.GetProject(ctx, &workv1.GetProjectRequest{Id: p.ID.String()})
	require.NoError(t, err)
	assert.Equal(t, p.ID.String(), got.Project.Id)

	list, err := srv.ListProjects(ctx, &workv1.ListProjectsRequest{})
	require.NoError(t, err)
	require.Len(t, list.Projects, 1)

	members, err := srv.ListProjectMembers(ctx, &workv1.ListProjectMembersRequest{ProjectId: p.ID.String()})
	require.NoError(t, err)
	assert.Len(t, members.Members, 1)
}

// Without a caller in the context no project RPC runs: the actor is never a guess.
func TestWorkGRPCServer_ProjectRPCsRefuseMissingCaller(t *testing.T) {
	tenantID := uuid.New()
	srv, repo, _, _ := newWorkProjectTaskTestServer()
	p := seedProject(repo, tenantID, "NOC")
	tenantOnly := ctxWithTenant(tenantID)
	pid, uid := p.ID.String(), uuid.NewString()

	calls := map[string]func() error{
		"GetProject":    func() error { _, err := srv.GetProject(tenantOnly, &workv1.GetProjectRequest{Id: pid}); return err },
		"ListProjects":  func() error { _, err := srv.ListProjects(tenantOnly, &workv1.ListProjectsRequest{}); return err },
		"UpdateProject": func() error { _, err := srv.UpdateProject(tenantOnly, &workv1.UpdateProjectRequest{Id: pid}); return err },
		"ArchiveProject": func() error {
			_, err := srv.ArchiveProject(tenantOnly, &workv1.ArchiveProjectRequest{Id: pid})
			return err
		},
		"DeleteProject": func() error { _, err := srv.DeleteProject(tenantOnly, &workv1.DeleteProjectRequest{Id: pid}); return err },
		"AddProjectMember": func() error {
			_, err := srv.AddProjectMember(tenantOnly, &workv1.AddProjectMemberRequest{ProjectId: pid, UserId: uid, Role: models.ProjectRoleMember})
			return err
		},
		"RemoveProjectMember": func() error {
			_, err := srv.RemoveProjectMember(tenantOnly, &workv1.RemoveProjectMemberRequest{ProjectId: pid, UserId: uid})
			return err
		},
		"ListProjectMembers": func() error {
			_, err := srv.ListProjectMembers(tenantOnly, &workv1.ListProjectMembersRequest{ProjectId: pid})
			return err
		},
		"UpdateProjectMemberRole": func() error {
			_, err := srv.UpdateProjectMemberRole(tenantOnly, &workv1.UpdateProjectMemberRoleRequest{ProjectId: pid, UserId: uid, Role: models.ProjectRoleMember})
			return err
		},
		"ListProjectTimeEntries": func() error {
			_, err := srv.ListProjectTimeEntries(tenantOnly, &workv1.ListProjectTimeEntriesRequest{ProjectId: pid})
			return err
		},
		"ListProjectTeamUtilization": func() error {
			_, err := srv.ListProjectTeamUtilization(tenantOnly, &workv1.ListProjectTeamUtilizationRequest{ProjectId: pid})
			return err
		},
	}
	for name, call := range calls {
		t.Run(name, func(t *testing.T) {
			err := call()
			require.Error(t, err)
			assert.Equal(t, codes.Unauthenticated, status.Code(err))
		})
	}
}
