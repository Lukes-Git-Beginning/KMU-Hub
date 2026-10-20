package server

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/kmuhub/kmuhub/internal/models"
	inboxv1 "github.com/kmuhub/kmuhub/proto/inbox/v1"
)

// ============================================================================
// Single-message RPCs: the shared guard ladder (tenant, id, not found)
// ============================================================================

func TestInboxMessageRPCs_GuardLadder(t *testing.T) {
	srv, _ := newTestInboxServer()

	calls := map[string]func(ctx context.Context, id string) error{
		"GetMessage": func(ctx context.Context, id string) error {
			_, err := srv.GetMessage(ctx, &inboxv1.GetMessageRequest{MessageId: id})
			return err
		},
		"MarkRead": func(ctx context.Context, id string) error {
			_, err := srv.MarkRead(ctx, &inboxv1.MarkReadRequest{MessageId: id})
			return err
		},
		"MarkUnread": func(ctx context.Context, id string) error {
			_, err := srv.MarkUnread(ctx, &inboxv1.MarkUnreadRequest{MessageId: id})
			return err
		},
		"ToggleStar": func(ctx context.Context, id string) error {
			_, err := srv.ToggleStar(ctx, &inboxv1.ToggleStarRequest{MessageId: id})
			return err
		},
		"ArchiveMessage": func(ctx context.Context, id string) error {
			_, err := srv.ArchiveMessage(ctx, &inboxv1.ArchiveMessageRequest{MessageId: id})
			return err
		},
		"UnarchiveMessage": func(ctx context.Context, id string) error {
			_, err := srv.UnarchiveMessage(ctx, &inboxv1.UnarchiveMessageRequest{MessageId: id})
			return err
		},
		"SnoozeMessage": func(ctx context.Context, id string) error {
			_, err := srv.SnoozeMessage(ctx, &inboxv1.SnoozeMessageRequest{
				MessageId: id, SnoozeUntil: timestamppb.New(time.Now().Add(time.Hour)),
			})
			return err
		},
		"UnsnoozeMessage": func(ctx context.Context, id string) error {
			_, err := srv.UnsnoozeMessage(ctx, &inboxv1.UnsnoozeMessageRequest{MessageId: id})
			return err
		},
		"AssignMessage": func(ctx context.Context, id string) error {
			_, err := srv.AssignMessage(ctx, &inboxv1.AssignMessageRequest{
				MessageId: id, AssigneeId: uuid.NewString(),
			})
			return err
		},
		"ReplyToMessage": func(ctx context.Context, id string) error {
			_, err := srv.ReplyToMessage(ctx, &inboxv1.ReplyToMessageRequest{
				MessageId: id, UserId: uuid.NewString(), Body: "hi",
			})
			return err
		},
		"ForwardMessage": func(ctx context.Context, id string) error {
			_, err := srv.ForwardMessage(ctx, &inboxv1.ForwardMessageRequest{
				MessageId: id, UserId: uuid.NewString(), To: "a@example.com",
			})
			return err
		},
		"ClaimMessage": func(ctx context.Context, id string) error {
			_, err := srv.ClaimMessage(ctx, &inboxv1.ClaimMessageRequest{
				MessageId: id, UserId: uuid.NewString(),
			})
			return err
		},
	}

	for name, call := range calls {
		t.Run(name+"/missing tenant", func(t *testing.T) {
			requireGRPCCode(t, call(context.Background(), uuid.NewString()), codes.InvalidArgument)
		})
		t.Run(name+"/invalid id", func(t *testing.T) {
			requireGRPCCode(t, call(tenantCtx(uuid.New()), "nope"), codes.InvalidArgument)
		})
		t.Run(name+"/unknown id", func(t *testing.T) {
			requireGRPCCode(t, call(tenantCtx(uuid.New()), uuid.NewString()), codes.NotFound)
		})
	}
}

func TestInboxMessageRPCs_ForeignTenantSeesNotFound(t *testing.T) {
	srv, deps := newTestInboxServer()
	owner, intruder := uuid.New(), uuid.New()
	msg := seedInboxMessage(t, deps, owner, uuid.New(), nil)

	_, err := srv.GetMessage(tenantCtx(intruder), &inboxv1.GetMessageRequest{MessageId: msg.ID.String()})
	requireGRPCCode(t, err, codes.NotFound)

	_, err = srv.ArchiveMessage(tenantCtx(intruder), &inboxv1.ArchiveMessageRequest{MessageId: msg.ID.String()})
	requireGRPCCode(t, err, codes.NotFound)

	// The owner's copy is untouched by the intruder's attempt.
	got, err := srv.GetMessage(tenantCtx(owner), &inboxv1.GetMessageRequest{MessageId: msg.ID.String()})
	require.NoError(t, err)
	require.False(t, got.Message.IsArchived)
}

func TestInboxMessageRPCs_HappyPaths(t *testing.T) {
	srv, deps := newTestInboxServer()
	tenantID := uuid.New()
	ctx := tenantCtx(tenantID)
	msg := seedInboxMessage(t, deps, tenantID, uuid.New(), nil)
	id := msg.ID.String()

	got, err := srv.GetMessage(ctx, &inboxv1.GetMessageRequest{MessageId: id})
	require.NoError(t, err)
	require.Equal(t, id, got.Message.Id)

	read, err := srv.MarkRead(ctx, &inboxv1.MarkReadRequest{MessageId: id})
	require.NoError(t, err)
	require.True(t, read.Message.IsRead)

	unread, err := srv.MarkUnread(ctx, &inboxv1.MarkUnreadRequest{MessageId: id})
	require.NoError(t, err)
	require.False(t, unread.Message.IsRead)

	star, err := srv.ToggleStar(ctx, &inboxv1.ToggleStarRequest{MessageId: id})
	require.NoError(t, err)
	require.True(t, star.Message.IsStarred)

	arch, err := srv.ArchiveMessage(ctx, &inboxv1.ArchiveMessageRequest{MessageId: id})
	require.NoError(t, err)
	require.True(t, arch.Message.IsArchived)

	unarch, err := srv.UnarchiveMessage(ctx, &inboxv1.UnarchiveMessageRequest{MessageId: id})
	require.NoError(t, err)
	require.False(t, unarch.Message.IsArchived)

	snoozed, err := srv.SnoozeMessage(ctx, &inboxv1.SnoozeMessageRequest{
		MessageId: id, SnoozeUntil: timestamppb.New(time.Now().Add(time.Hour)),
	})
	require.NoError(t, err)
	require.NotNil(t, snoozed.Message.SnoozedUntil)

	unsnoozed, err := srv.UnsnoozeMessage(ctx, &inboxv1.UnsnoozeMessageRequest{MessageId: id})
	require.NoError(t, err)
	require.Nil(t, unsnoozed.Message.SnoozedUntil)

	assignee := uuid.New()
	assigned, err := srv.AssignMessage(ctx, &inboxv1.AssignMessageRequest{MessageId: id, AssigneeId: assignee.String()})
	require.NoError(t, err)
	require.Equal(t, assignee.String(), *assigned.Message.AssignedTo)

	// A second assignment loses the race and surfaces as AlreadyExists.
	_, err = srv.AssignMessage(ctx, &inboxv1.AssignMessageRequest{MessageId: id, AssigneeId: uuid.NewString()})
	requireGRPCCode(t, err, codes.AlreadyExists)
}

func TestSnoozeMessage_RequiresTimestampAndFutureTime(t *testing.T) {
	srv, deps := newTestInboxServer()
	tenantID := uuid.New()
	msg := seedInboxMessage(t, deps, tenantID, uuid.New(), nil)

	_, err := srv.SnoozeMessage(tenantCtx(tenantID), &inboxv1.SnoozeMessageRequest{MessageId: msg.ID.String()})
	requireGRPCCode(t, err, codes.InvalidArgument)

	_, err = srv.SnoozeMessage(tenantCtx(tenantID), &inboxv1.SnoozeMessageRequest{
		MessageId: msg.ID.String(), SnoozeUntil: timestamppb.New(time.Now().Add(-time.Hour)),
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestAssignMessage_InvalidAssignee(t *testing.T) {
	srv, _ := newTestInboxServer()
	_, err := srv.AssignMessage(tenantCtx(uuid.New()), &inboxv1.AssignMessageRequest{
		MessageId: uuid.NewString(), AssigneeId: "nope",
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestClaimMessage_Paths(t *testing.T) {
	srv, deps := newTestInboxServer()
	tenantID := uuid.New()
	ctx := tenantCtx(tenantID)
	userID := uuid.New()

	t.Run("invalid user", func(t *testing.T) {
		_, err := srv.ClaimMessage(ctx, &inboxv1.ClaimMessageRequest{MessageId: uuid.NewString(), UserId: "nope"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("message outside any team inbox", func(t *testing.T) {
		msg := seedInboxMessage(t, deps, tenantID, uuid.New(), nil)
		_, err := srv.ClaimMessage(ctx, &inboxv1.ClaimMessageRequest{MessageId: msg.ID.String(), UserId: userID.String()})
		requireGRPCCode(t, err, codes.FailedPrecondition)
	})

	t.Run("non-member is refused", func(t *testing.T) {
		created, err := srv.CreateTeamInbox(ctx, &inboxv1.CreateTeamInboxRequest{
			UserId: uuid.NewString(), Name: "Support",
		})
		require.NoError(t, err)
		teamID, err := uuid.Parse(created.TeamInbox.Id)
		require.NoError(t, err)
		msg := seedInboxMessage(t, deps, tenantID, uuid.New(), func(m *models.InboxMessage) { m.TeamInboxID = &teamID })

		_, err = srv.ClaimMessage(ctx, &inboxv1.ClaimMessageRequest{MessageId: msg.ID.String(), UserId: userID.String()})
		requireGRPCCode(t, err, codes.PermissionDenied)
	})

	t.Run("member claims once", func(t *testing.T) {
		created, err := srv.CreateTeamInbox(ctx, &inboxv1.CreateTeamInboxRequest{
			UserId: userID.String(), Name: "Sales",
		})
		require.NoError(t, err)
		teamID, err := uuid.Parse(created.TeamInbox.Id)
		require.NoError(t, err)
		msg := seedInboxMessage(t, deps, tenantID, uuid.New(), func(m *models.InboxMessage) { m.TeamInboxID = &teamID })

		resp, err := srv.ClaimMessage(ctx, &inboxv1.ClaimMessageRequest{MessageId: msg.ID.String(), UserId: userID.String()})
		require.NoError(t, err)
		require.Equal(t, userID.String(), *resp.Message.AssignedTo)

		_, err = srv.ClaimMessage(ctx, &inboxv1.ClaimMessageRequest{MessageId: msg.ID.String(), UserId: userID.String()})
		requireGRPCCode(t, err, codes.AlreadyExists)
	})
}

func TestReplyAndForward_InvalidUserID(t *testing.T) {
	srv, deps := newTestInboxServer()
	tenantID := uuid.New()
	msg := seedInboxMessage(t, deps, tenantID, uuid.New(), nil)

	_, err := srv.ReplyToMessage(tenantCtx(tenantID), &inboxv1.ReplyToMessageRequest{
		MessageId: msg.ID.String(), UserId: "nope", Body: "x",
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

// ============================================================================
// ListMessages: filter plumbing and cursor
// ============================================================================

func TestListMessages_Guards(t *testing.T) {
	srv, _ := newTestInboxServer()

	_, err := srv.ListMessages(context.Background(), &inboxv1.ListMessagesRequest{UserId: uuid.NewString()})
	requireGRPCCode(t, err, codes.InvalidArgument)

	_, err = srv.ListMessages(tenantCtx(uuid.New()), &inboxv1.ListMessagesRequest{UserId: "nope"})
	requireGRPCCode(t, err, codes.InvalidArgument)

	bad := "nope"
	_, err = srv.ListMessages(tenantCtx(uuid.New()), &inboxv1.ListMessagesRequest{
		UserId: uuid.NewString(), TeamInboxId: &bad,
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestListMessages_AllFiltersAndNextPageToken(t *testing.T) {
	srv, deps := newTestInboxServer()
	tenantID := uuid.New()
	userID := uuid.New()
	msg := seedInboxMessage(t, deps, tenantID, userID, nil)

	ch := inboxv1.Channel_CHANNEL_EMAIL
	yes, search, st := true, "x", "open"
	team := uuid.NewString()
	cursor := buildPageToken(time.Now().UTC(), uuid.New())

	resp, err := srv.ListMessages(tenantCtx(tenantID), &inboxv1.ListMessagesRequest{
		UserId:      userID.String(),
		PageSize:    1,
		PageToken:   cursor,
		Channel:     &ch,
		IsRead:      &yes,
		IsStarred:   &yes,
		IsArchived:  &yes,
		TeamInboxId: &team,
		SearchQuery: &search,
		Status:      &st,
	})
	require.NoError(t, err)
	require.Len(t, resp.Messages, 1)
	require.Equal(t, int32(1), resp.TotalCount)
	// A full page hands back a cursor built from the last row.
	require.Equal(t, buildPageToken(msg.ReceivedAt, msg.ID), resp.NextPageToken)

	// A malformed cursor is ignored rather than failing the list.
	resp, err = srv.ListMessages(tenantCtx(tenantID), &inboxv1.ListMessagesRequest{
		UserId: userID.String(), PageToken: "garbage",
	})
	require.NoError(t, err)
	require.Empty(t, resp.NextPageToken)
}

func TestListMessages_ForeignTenantGetsNothing(t *testing.T) {
	srv, deps := newTestInboxServer()
	owner := uuid.New()
	userID := uuid.New()
	seedInboxMessage(t, deps, owner, userID, nil)

	resp, err := srv.ListMessages(tenantCtx(uuid.New()), &inboxv1.ListMessagesRequest{UserId: userID.String()})
	require.NoError(t, err)
	require.Empty(t, resp.Messages)
	require.NotNil(t, resp.Messages)
}

func TestBulkRPCs(t *testing.T) {
	srv, deps := newTestInboxServer()
	tenantID := uuid.New()
	ctx := tenantCtx(tenantID)
	a := seedInboxMessage(t, deps, tenantID, uuid.New(), nil)
	b := seedInboxMessage(t, deps, tenantID, uuid.New(), nil)
	ids := []string{a.ID.String(), b.ID.String()}

	read, err := srv.BulkMarkRead(ctx, &inboxv1.BulkMarkReadRequest{MessageIds: ids})
	require.NoError(t, err)
	require.Equal(t, int32(2), read.UpdatedCount)

	arch, err := srv.BulkArchive(ctx, &inboxv1.BulkArchiveRequest{MessageIds: ids})
	require.NoError(t, err)
	require.Equal(t, int32(2), arch.UpdatedCount)

	got, err := srv.GetMessage(ctx, &inboxv1.GetMessageRequest{MessageId: ids[0]})
	require.NoError(t, err)
	require.True(t, got.Message.IsRead)
	require.True(t, got.Message.IsArchived)

	_, err = srv.BulkMarkRead(ctx, &inboxv1.BulkMarkReadRequest{MessageIds: []string{"nope"}})
	requireGRPCCode(t, err, codes.InvalidArgument)
	_, err = srv.BulkArchive(ctx, &inboxv1.BulkArchiveRequest{MessageIds: []string{"nope"}})
	requireGRPCCode(t, err, codes.InvalidArgument)
	_, err = srv.BulkMarkRead(context.Background(), &inboxv1.BulkMarkReadRequest{})
	requireGRPCCode(t, err, codes.InvalidArgument)
	_, err = srv.BulkArchive(context.Background(), &inboxv1.BulkArchiveRequest{})
	requireGRPCCode(t, err, codes.InvalidArgument)

	// A foreign tenant's bulk call cannot touch the owner's rows.
	_, err = srv.BulkArchive(tenantCtx(uuid.New()), &inboxv1.BulkArchiveRequest{MessageIds: ids})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestGetUnreadCount_InvalidUser(t *testing.T) {
	srv, _ := newTestInboxServer()
	_, err := srv.GetUnreadCount(context.Background(), &inboxv1.GetUnreadCountRequest{UserId: "nope"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

// ============================================================================
// Canned responses
// ============================================================================

func TestCannedResponses_CRUD(t *testing.T) {
	srv, _ := newTestInboxServer()
	tenantID := uuid.New()
	ctx := tenantCtx(tenantID)

	created, err := srv.CreateCannedResponse(ctx, &inboxv1.CreateCannedResponseRequest{Name: "Danke", Body: "Vielen Dank"})
	require.NoError(t, err)
	require.Equal(t, "Danke", created.Name)

	list, err := srv.ListCannedResponses(ctx, &inboxv1.ListCannedResponsesRequest{})
	require.NoError(t, err)
	require.Len(t, list.CannedResponses, 1)

	updated, err := srv.UpdateCannedResponse(ctx, &inboxv1.UpdateCannedResponseRequest{
		Id: created.Id, Name: ptrString("Danke!"), Body: ptrString("Besten Dank"),
	})
	require.NoError(t, err)
	require.Equal(t, "Besten Dank", updated.Body)

	// A foreign tenant sees none of it and cannot change or delete it.
	other := tenantCtx(uuid.New())
	foreign, err := srv.ListCannedResponses(other, &inboxv1.ListCannedResponsesRequest{})
	require.NoError(t, err)
	require.Empty(t, foreign.CannedResponses)
	_, err = srv.UpdateCannedResponse(other, &inboxv1.UpdateCannedResponseRequest{Id: created.Id, Name: ptrString("x"), Body: ptrString("y")})
	requireGRPCCode(t, err, codes.NotFound)
	_, err = srv.DeleteCannedResponse(other, &inboxv1.DeleteCannedResponseRequest{Id: created.Id})
	requireGRPCCode(t, err, codes.NotFound)

	_, err = srv.DeleteCannedResponse(ctx, &inboxv1.DeleteCannedResponseRequest{Id: created.Id})
	require.NoError(t, err)
	_, err = srv.DeleteCannedResponse(ctx, &inboxv1.DeleteCannedResponseRequest{Id: created.Id})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestCannedResponses_Guards(t *testing.T) {
	srv, _ := newTestInboxServer()
	ctx := tenantCtx(uuid.New())

	_, err := srv.CreateCannedResponse(context.Background(), &inboxv1.CreateCannedResponseRequest{Name: "a", Body: "b"})
	requireGRPCCode(t, err, codes.InvalidArgument)
	_, err = srv.CreateCannedResponse(ctx, &inboxv1.CreateCannedResponseRequest{Name: "", Body: "b"})
	requireGRPCCode(t, err, codes.InvalidArgument)
	_, err = srv.CreateCannedResponse(ctx, &inboxv1.CreateCannedResponseRequest{Name: "a", Body: ""})
	requireGRPCCode(t, err, codes.InvalidArgument)

	_, err = srv.ListCannedResponses(context.Background(), &inboxv1.ListCannedResponsesRequest{})
	requireGRPCCode(t, err, codes.InvalidArgument)

	_, err = srv.UpdateCannedResponse(context.Background(), &inboxv1.UpdateCannedResponseRequest{Id: uuid.NewString()})
	requireGRPCCode(t, err, codes.InvalidArgument)
	_, err = srv.UpdateCannedResponse(ctx, &inboxv1.UpdateCannedResponseRequest{Id: "nope"})
	requireGRPCCode(t, err, codes.InvalidArgument)

	_, err = srv.DeleteCannedResponse(context.Background(), &inboxv1.DeleteCannedResponseRequest{Id: uuid.NewString()})
	requireGRPCCode(t, err, codes.InvalidArgument)
	_, err = srv.DeleteCannedResponse(ctx, &inboxv1.DeleteCannedResponseRequest{Id: "nope"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

// ============================================================================
// Team inboxes
// ============================================================================

func TestTeamInbox_LifecycleAndAdminGate(t *testing.T) {
	srv, _ := newTestInboxServer()
	tenantID := uuid.New()
	ctx := tenantCtx(tenantID)
	admin, member := uuid.New(), uuid.New()

	created, err := srv.CreateTeamInbox(ctx, &inboxv1.CreateTeamInboxRequest{
		UserId:         admin.String(),
		Name:           "Support",
		Description:    "Erste Linie",
		AssignmentMode: inboxv1.AssignmentMode_ASSIGNMENT_MODE_ROUND_ROBIN,
		Visibility:     inboxv1.TeamInboxVisibility_VISIBILITY_PRIVATE,
	})
	require.NoError(t, err)
	require.Equal(t, "Support", created.TeamInbox.Name)
	teamID := created.TeamInbox.Id

	// The creator is auto-added as admin.
	members, err := srv.ListTeamMembers(ctx, &inboxv1.ListTeamMembersRequest{TeamInboxId: teamID})
	require.NoError(t, err)
	require.Len(t, members.Members, 1)
	require.Equal(t, admin.String(), members.Members[0].UserId)

	// A member (non-admin) may not edit or delete.
	_, err = srv.AddTeamMember(ctx, &inboxv1.AddTeamMemberRequest{
		TeamInboxId: teamID, MemberUserId: member.String(),
	})
	require.NoError(t, err)
	newName := "Hijacked"
	_, err = srv.UpdateTeamInbox(ctx, &inboxv1.UpdateTeamInboxRequest{
		TeamInboxId: teamID, UserId: member.String(), Name: &newName,
	})
	requireGRPCCode(t, err, codes.PermissionDenied)
	_, err = srv.DeleteTeamInbox(ctx, &inboxv1.DeleteTeamInboxRequest{TeamInboxId: teamID, UserId: member.String()})
	requireGRPCCode(t, err, codes.PermissionDenied)

	// The admin edits every field.
	desc, mode, vis := "neu", inboxv1.AssignmentMode_ASSIGNMENT_MODE_MANUAL, inboxv1.TeamInboxVisibility_VISIBILITY_OPEN
	upd, err := srv.UpdateTeamInbox(ctx, &inboxv1.UpdateTeamInboxRequest{
		TeamInboxId: teamID, UserId: admin.String(),
		Name: &newName, Description: &desc, AssignmentMode: &mode, Visibility: &vis,
	})
	require.NoError(t, err)
	require.Equal(t, newName, upd.TeamInbox.Name)

	list, err := srv.ListTeamInboxes(ctx, &inboxv1.ListTeamInboxesRequest{UserId: member.String()})
	require.NoError(t, err)
	require.Len(t, list.TeamInboxes, 1)

	// Another tenant cannot see or change it.
	other := tenantCtx(uuid.New())
	_, err = srv.UpdateTeamInbox(other, &inboxv1.UpdateTeamInboxRequest{TeamInboxId: teamID, UserId: admin.String()})
	requireGRPCCode(t, err, codes.NotFound)
	foreign, err := srv.ListTeamInboxes(other, &inboxv1.ListTeamInboxesRequest{UserId: member.String()})
	require.NoError(t, err)
	require.Empty(t, foreign.TeamInboxes)

	_, err = srv.DeleteTeamInbox(ctx, &inboxv1.DeleteTeamInboxRequest{TeamInboxId: teamID, UserId: admin.String()})
	require.NoError(t, err)
	_, err = srv.UpdateTeamInbox(ctx, &inboxv1.UpdateTeamInboxRequest{TeamInboxId: teamID, UserId: admin.String()})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestTeamInbox_MemberManagement(t *testing.T) {
	srv, _ := newTestInboxServer()
	ctx := tenantCtx(uuid.New())
	admin, member := uuid.New(), uuid.New()

	created, err := srv.CreateTeamInbox(ctx, &inboxv1.CreateTeamInboxRequest{UserId: admin.String(), Name: "Team"})
	require.NoError(t, err)
	teamID := created.TeamInbox.Id

	added, err := srv.AddTeamMember(ctx, &inboxv1.AddTeamMemberRequest{
		TeamInboxId: teamID, MemberUserId: member.String(), Role: inboxv1.TeamMemberRole_TEAM_ROLE_ADMIN,
	})
	require.NoError(t, err)
	require.Equal(t, inboxv1.TeamMemberRole_TEAM_ROLE_ADMIN, added.Member.Role)

	_, err = srv.AddTeamMember(ctx, &inboxv1.AddTeamMemberRequest{TeamInboxId: teamID, MemberUserId: member.String()})
	requireGRPCCode(t, err, codes.AlreadyExists)

	_, err = srv.RemoveTeamMember(ctx, &inboxv1.RemoveTeamMemberRequest{TeamInboxId: teamID, MemberUserId: member.String()})
	require.NoError(t, err)

	// Removing a non-member and the last admin are both refused.
	_, err = srv.RemoveTeamMember(ctx, &inboxv1.RemoveTeamMemberRequest{TeamInboxId: teamID, MemberUserId: member.String()})
	requireGRPCCode(t, err, codes.PermissionDenied)
	_, err = srv.RemoveTeamMember(ctx, &inboxv1.RemoveTeamMemberRequest{TeamInboxId: teamID, MemberUserId: admin.String()})
	requireGRPCCode(t, err, codes.FailedPrecondition)
}

func TestTeamInbox_Guards(t *testing.T) {
	srv, _ := newTestInboxServer()
	ctx := tenantCtx(uuid.New())
	no := context.Background()
	id, bad := uuid.NewString(), "nope"

	_, err := srv.CreateTeamInbox(no, &inboxv1.CreateTeamInboxRequest{UserId: id, Name: "x"})
	requireGRPCCode(t, err, codes.InvalidArgument)
	_, err = srv.CreateTeamInbox(ctx, &inboxv1.CreateTeamInboxRequest{UserId: bad, Name: "x"})
	requireGRPCCode(t, err, codes.InvalidArgument)
	_, err = srv.CreateTeamInbox(ctx, &inboxv1.CreateTeamInboxRequest{UserId: id})
	require.Error(t, err, "empty name is rejected by the service")

	_, err = srv.UpdateTeamInbox(no, &inboxv1.UpdateTeamInboxRequest{TeamInboxId: id, UserId: id})
	requireGRPCCode(t, err, codes.InvalidArgument)
	_, err = srv.UpdateTeamInbox(ctx, &inboxv1.UpdateTeamInboxRequest{TeamInboxId: bad, UserId: id})
	requireGRPCCode(t, err, codes.InvalidArgument)
	_, err = srv.UpdateTeamInbox(ctx, &inboxv1.UpdateTeamInboxRequest{TeamInboxId: id, UserId: bad})
	requireGRPCCode(t, err, codes.InvalidArgument)

	_, err = srv.DeleteTeamInbox(no, &inboxv1.DeleteTeamInboxRequest{TeamInboxId: id, UserId: id})
	requireGRPCCode(t, err, codes.InvalidArgument)
	_, err = srv.DeleteTeamInbox(ctx, &inboxv1.DeleteTeamInboxRequest{TeamInboxId: bad, UserId: id})
	requireGRPCCode(t, err, codes.InvalidArgument)
	_, err = srv.DeleteTeamInbox(ctx, &inboxv1.DeleteTeamInboxRequest{TeamInboxId: id, UserId: bad})
	requireGRPCCode(t, err, codes.InvalidArgument)

	_, err = srv.ListTeamInboxes(no, &inboxv1.ListTeamInboxesRequest{UserId: id})
	requireGRPCCode(t, err, codes.InvalidArgument)
	_, err = srv.ListTeamInboxes(ctx, &inboxv1.ListTeamInboxesRequest{UserId: bad})
	requireGRPCCode(t, err, codes.InvalidArgument)

	_, err = srv.AddTeamMember(ctx, &inboxv1.AddTeamMemberRequest{TeamInboxId: bad, MemberUserId: id})
	requireGRPCCode(t, err, codes.InvalidArgument)
	_, err = srv.AddTeamMember(ctx, &inboxv1.AddTeamMemberRequest{TeamInboxId: id, MemberUserId: bad})
	requireGRPCCode(t, err, codes.InvalidArgument)
	_, err = srv.RemoveTeamMember(ctx, &inboxv1.RemoveTeamMemberRequest{TeamInboxId: bad, MemberUserId: id})
	requireGRPCCode(t, err, codes.InvalidArgument)
	_, err = srv.RemoveTeamMember(ctx, &inboxv1.RemoveTeamMemberRequest{TeamInboxId: id, MemberUserId: bad})
	requireGRPCCode(t, err, codes.InvalidArgument)
	_, err = srv.ListTeamMembers(ctx, &inboxv1.ListTeamMembersRequest{TeamInboxId: bad})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

// ============================================================================
// Routing rules
// ============================================================================

func mustStruct(t *testing.T, m map[string]any) *structpb.Struct {
	t.Helper()
	s, err := structpb.NewStruct(m)
	require.NoError(t, err)
	return s
}

func TestRoutingRule_Guards(t *testing.T) {
	srv, _ := newTestInboxServer()
	ctx := tenantCtx(uuid.New())
	no := context.Background()
	id, bad := uuid.NewString(), "nope"

	_, err := srv.CreateRoutingRule(no, &inboxv1.CreateRoutingRuleRequest{UserId: id})
	requireGRPCCode(t, err, codes.InvalidArgument)
	_, err = srv.CreateRoutingRule(ctx, &inboxv1.CreateRoutingRuleRequest{UserId: bad})
	requireGRPCCode(t, err, codes.InvalidArgument)

	_, err = srv.UpdateRoutingRule(no, &inboxv1.UpdateRoutingRuleRequest{RuleId: id})
	requireGRPCCode(t, err, codes.InvalidArgument)
	_, err = srv.UpdateRoutingRule(ctx, &inboxv1.UpdateRoutingRuleRequest{RuleId: bad})
	requireGRPCCode(t, err, codes.InvalidArgument)
	_, err = srv.UpdateRoutingRule(ctx, &inboxv1.UpdateRoutingRuleRequest{RuleId: id})
	requireGRPCCode(t, err, codes.NotFound)

	_, err = srv.DeleteRoutingRule(no, &inboxv1.DeleteRoutingRuleRequest{RuleId: id})
	requireGRPCCode(t, err, codes.InvalidArgument)
	_, err = srv.DeleteRoutingRule(ctx, &inboxv1.DeleteRoutingRuleRequest{RuleId: bad})
	requireGRPCCode(t, err, codes.InvalidArgument)
	_, err = srv.DeleteRoutingRule(ctx, &inboxv1.DeleteRoutingRuleRequest{RuleId: id})
	requireGRPCCode(t, err, codes.NotFound)

	_, err = srv.ListRoutingRules(no, &inboxv1.ListRoutingRulesRequest{})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestRoutingRule_ForeignTenantCannotTouch(t *testing.T) {
	srv, deps := newTestInboxServer()
	owner := uuid.New()
	rule := &models.RoutingRule{ID: uuid.New(), TenantID: owner, Name: "r", IsActive: true}
	require.NoError(t, deps.routingRepo.Create(context.Background(), rule))

	other := tenantCtx(uuid.New())
	_, err := srv.UpdateRoutingRule(other, &inboxv1.UpdateRoutingRuleRequest{RuleId: rule.ID.String()})
	requireGRPCCode(t, err, codes.NotFound)
	_, err = srv.DeleteRoutingRule(other, &inboxv1.DeleteRoutingRuleRequest{RuleId: rule.ID.String()})
	requireGRPCCode(t, err, codes.NotFound)

	mine, err := srv.ListRoutingRules(tenantCtx(owner), &inboxv1.ListRoutingRulesRequest{})
	require.NoError(t, err)
	require.Len(t, mine.Rules, 1)
}

func TestRoutingRule_UpdateAndDelete(t *testing.T) {
	srv, deps := newTestInboxServer()
	tenantID := uuid.New()
	ctx := tenantCtx(tenantID)
	rule := &models.RoutingRule{
		ID: uuid.New(), TenantID: tenantID, Name: "r", IsActive: true,
		Conditions: []byte(`{"field":"sender_email","operator":"contains","value":"x"}`),
		Actions:    []byte(`[{"type":"add_tags","config":{"tags":["a"]}}]`),
	}
	require.NoError(t, deps.routingRepo.Create(context.Background(), rule))
	id := rule.ID.String()

	name, prio, off := "renamed", int32(7), false
	ch := inboxv1.Channel_CHANNEL_EMAIL
	_, err := srv.UpdateRoutingRule(ctx, &inboxv1.UpdateRoutingRuleRequest{
		RuleId: id, Name: &name, Channel: &ch, Priority: &prio, IsActive: &off,
	})
	// The stored actions are a JSON array, which the service accepts.
	require.NoError(t, err)

	got, err := deps.routingRepo.GetByID(context.Background(), tenantID, rule.ID)
	require.NoError(t, err)
	require.Equal(t, "renamed", got.Name)
	require.Equal(t, 7, got.Priority)
	require.False(t, got.IsActive)
	require.Equal(t, "email", *got.Channel)

	// Overwriting conditions/actions with a Struct (always a JSON object on the
	// wire) collides with the service's array contract; see
	// TestCreateRoutingRule_StructActionsAreAlwaysRejected.
	_, err = srv.UpdateRoutingRule(ctx, &inboxv1.UpdateRoutingRuleRequest{
		RuleId: id, Conditions: mustStruct(t, map[string]any{"field": "subject"}),
	})
	require.NoError(t, err)
	_, err = srv.UpdateRoutingRule(ctx, &inboxv1.UpdateRoutingRuleRequest{
		RuleId: id, Actions: mustStruct(t, map[string]any{"type": "add_tags"}),
	})
	requireGRPCCode(t, err, codes.InvalidArgument)

	_, err = srv.DeleteRoutingRule(ctx, &inboxv1.DeleteRoutingRuleRequest{RuleId: id})
	require.NoError(t, err)
}

// Pins the CURRENT behaviour of a defect, see BACKLOG unit
// fix-inbox-routing-rule-actions-struct-vs-array: proto `actions` is a
// google.protobuf.Struct (a JSON object), while routing.validateActions only
// accepts a JSON array of {type, config}. No request that reaches this RPC can
// therefore pass validation, so no routing rule can be created through the API.
// When the contract is fixed this test must flip to NoError.
func TestCreateRoutingRule_StructActionsAreAlwaysRejected(t *testing.T) {
	srv, _ := newTestInboxServer()
	ctx := tenantCtx(uuid.New())
	conditions := mustStruct(t, map[string]any{"field": "sender_email", "operator": "contains", "value": "x"})

	for name, actions := range map[string]*structpb.Struct{
		"nil actions":           nil,
		"empty object":          mustStruct(t, map[string]any{}),
		"object with type":      mustStruct(t, map[string]any{"type": "add_tags", "config": map[string]any{}}),
		"object with unknown":   mustStruct(t, map[string]any{"type": "nope"}),
		"object wrapping array": mustStruct(t, map[string]any{"actions": []any{map[string]any{"type": "add_tags"}}}),
	} {
		t.Run(name, func(t *testing.T) {
			_, err := srv.CreateRoutingRule(ctx, &inboxv1.CreateRoutingRuleRequest{
				UserId: uuid.NewString(), Name: "r", Conditions: conditions, Actions: actions,
			})
			requireGRPCCode(t, err, codes.InvalidArgument)
		})
	}
}

func TestTestRoutingRule(t *testing.T) {
	srv, _ := newTestInboxServer()
	ctx := tenantCtx(uuid.New())
	email := "kunde@example.com"
	msg := &inboxv1.InboxMessageInfo{
		Channel:     inboxv1.Channel_CHANNEL_EMAIL,
		SenderEmail: &email,
		Subject:     "Rechnung",
	}

	hit, err := srv.TestRoutingRule(ctx, &inboxv1.TestRoutingRuleRequest{
		Conditions:  mustStruct(t, map[string]any{"field": "sender_email", "operator": "contains", "value": "kunde"}),
		TestMessage: msg,
	})
	require.NoError(t, err)
	require.True(t, hit.Matches)

	miss, err := srv.TestRoutingRule(ctx, &inboxv1.TestRoutingRuleRequest{
		Conditions:  mustStruct(t, map[string]any{"field": "sender_email", "operator": "contains", "value": "lieferant"}),
		TestMessage: msg,
	})
	require.NoError(t, err)
	require.False(t, miss.Matches)

	// A condition whose value has the wrong JSON type for models.Condition.And.
	_, err = srv.TestRoutingRule(ctx, &inboxv1.TestRoutingRuleRequest{
		Conditions:  mustStruct(t, map[string]any{"and": "not-a-list"}),
		TestMessage: msg,
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}
