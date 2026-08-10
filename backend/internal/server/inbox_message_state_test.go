package server

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/kmuhub/kmuhub/internal/models"
	inboxv1 "github.com/kmuhub/kmuhub/proto/inbox/v1"
)

// ============================================================================
// MarkRead / MarkUnread
// ============================================================================

func TestInboxMarkRead_HappyPath(t *testing.T) {
	srv, deps := newTestInboxServer()
	tenantID := uuid.New()
	msg := seedInboxMessage(t, deps, tenantID, uuid.New(), func(m *models.InboxMessage) { m.IsRead = false })

	resp, err := srv.MarkRead(tenantCtx(tenantID), &inboxv1.MarkReadRequest{MessageId: msg.ID.String()})
	require.NoError(t, err)
	require.True(t, resp.Message.IsRead)
}

func TestInboxMarkRead_NotFound(t *testing.T) {
	srv, _ := newTestInboxServer()
	_, err := srv.MarkRead(tenantCtx(uuid.New()), &inboxv1.MarkReadRequest{MessageId: uuid.New().String()})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestInboxMarkRead_InvalidMessageID(t *testing.T) {
	srv, _ := newTestInboxServer()
	_, err := srv.MarkRead(tenantCtx(uuid.New()), &inboxv1.MarkReadRequest{MessageId: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestInboxMarkRead_MissingTenant(t *testing.T) {
	srv, _ := newTestInboxServer()
	_, err := srv.MarkRead(context.Background(), &inboxv1.MarkReadRequest{MessageId: uuid.New().String()})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestInboxMarkUnread_HappyPath(t *testing.T) {
	srv, deps := newTestInboxServer()
	tenantID := uuid.New()
	msg := seedInboxMessage(t, deps, tenantID, uuid.New(), func(m *models.InboxMessage) { m.IsRead = true })

	resp, err := srv.MarkUnread(tenantCtx(tenantID), &inboxv1.MarkUnreadRequest{MessageId: msg.ID.String()})
	require.NoError(t, err)
	require.False(t, resp.Message.IsRead)
}

func TestInboxMarkUnread_NotFound(t *testing.T) {
	srv, _ := newTestInboxServer()
	_, err := srv.MarkUnread(tenantCtx(uuid.New()), &inboxv1.MarkUnreadRequest{MessageId: uuid.New().String()})
	requireGRPCCode(t, err, codes.NotFound)
}

// ============================================================================
// ToggleStar
// ============================================================================

func TestInboxToggleStar_HappyPath(t *testing.T) {
	srv, deps := newTestInboxServer()
	tenantID := uuid.New()
	msg := seedInboxMessage(t, deps, tenantID, uuid.New(), func(m *models.InboxMessage) { m.IsStarred = false })

	resp, err := srv.ToggleStar(tenantCtx(tenantID), &inboxv1.ToggleStarRequest{MessageId: msg.ID.String()})
	require.NoError(t, err)
	require.True(t, resp.Message.IsStarred)

	resp, err = srv.ToggleStar(tenantCtx(tenantID), &inboxv1.ToggleStarRequest{MessageId: msg.ID.String()})
	require.NoError(t, err)
	require.False(t, resp.Message.IsStarred)
}

func TestInboxToggleStar_NotFound(t *testing.T) {
	srv, _ := newTestInboxServer()
	_, err := srv.ToggleStar(tenantCtx(uuid.New()), &inboxv1.ToggleStarRequest{MessageId: uuid.New().String()})
	requireGRPCCode(t, err, codes.NotFound)
}

// ============================================================================
// ArchiveMessage / UnarchiveMessage
// ============================================================================

func TestInboxArchiveMessage_HappyPath(t *testing.T) {
	srv, deps := newTestInboxServer()
	tenantID := uuid.New()
	msg := seedInboxMessage(t, deps, tenantID, uuid.New(), func(m *models.InboxMessage) { m.IsArchived = false })

	resp, err := srv.ArchiveMessage(tenantCtx(tenantID), &inboxv1.ArchiveMessageRequest{MessageId: msg.ID.String()})
	require.NoError(t, err)
	require.True(t, resp.Message.IsArchived)
}

func TestInboxArchiveMessage_NotFound(t *testing.T) {
	srv, _ := newTestInboxServer()
	_, err := srv.ArchiveMessage(tenantCtx(uuid.New()), &inboxv1.ArchiveMessageRequest{MessageId: uuid.New().String()})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestInboxUnarchiveMessage_HappyPath(t *testing.T) {
	srv, deps := newTestInboxServer()
	tenantID := uuid.New()
	msg := seedInboxMessage(t, deps, tenantID, uuid.New(), func(m *models.InboxMessage) { m.IsArchived = true })

	resp, err := srv.UnarchiveMessage(tenantCtx(tenantID), &inboxv1.UnarchiveMessageRequest{MessageId: msg.ID.String()})
	require.NoError(t, err)
	require.False(t, resp.Message.IsArchived)
}

func TestInboxUnarchiveMessage_NotFound(t *testing.T) {
	srv, _ := newTestInboxServer()
	_, err := srv.UnarchiveMessage(tenantCtx(uuid.New()), &inboxv1.UnarchiveMessageRequest{MessageId: uuid.New().String()})
	requireGRPCCode(t, err, codes.NotFound)
}

// ============================================================================
// SnoozeMessage / UnsnoozeMessage
// ============================================================================

func TestInboxSnoozeMessage_HappyPath(t *testing.T) {
	srv, deps := newTestInboxServer()
	tenantID := uuid.New()
	msg := seedInboxMessage(t, deps, tenantID, uuid.New(), nil)
	until := time.Now().Add(2 * time.Hour).UTC()

	resp, err := srv.SnoozeMessage(tenantCtx(tenantID), &inboxv1.SnoozeMessageRequest{
		MessageId:   msg.ID.String(),
		SnoozeUntil: timestamppb.New(until),
	})
	require.NoError(t, err)
	require.NotNil(t, resp.Message.SnoozedUntil)
}

func TestInboxSnoozeMessage_MissingSnoozeUntil(t *testing.T) {
	srv, deps := newTestInboxServer()
	tenantID := uuid.New()
	msg := seedInboxMessage(t, deps, tenantID, uuid.New(), nil)

	_, err := srv.SnoozeMessage(tenantCtx(tenantID), &inboxv1.SnoozeMessageRequest{MessageId: msg.ID.String()})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestInboxSnoozeMessage_PastTimeRejected(t *testing.T) {
	srv, deps := newTestInboxServer()
	tenantID := uuid.New()
	msg := seedInboxMessage(t, deps, tenantID, uuid.New(), nil)

	_, err := srv.SnoozeMessage(tenantCtx(tenantID), &inboxv1.SnoozeMessageRequest{
		MessageId:   msg.ID.String(),
		SnoozeUntil: timestamppb.New(time.Now().Add(-time.Hour)),
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestInboxSnoozeMessage_NotFound(t *testing.T) {
	srv, _ := newTestInboxServer()
	_, err := srv.SnoozeMessage(tenantCtx(uuid.New()), &inboxv1.SnoozeMessageRequest{
		MessageId:   uuid.New().String(),
		SnoozeUntil: timestamppb.New(time.Now().Add(time.Hour)),
	})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestInboxUnsnoozeMessage_HappyPath(t *testing.T) {
	srv, deps := newTestInboxServer()
	tenantID := uuid.New()
	until := time.Now().Add(time.Hour).UTC()
	msg := seedInboxMessage(t, deps, tenantID, uuid.New(), func(m *models.InboxMessage) {
		m.SnoozedUntil = &until
	})

	resp, err := srv.UnsnoozeMessage(tenantCtx(tenantID), &inboxv1.UnsnoozeMessageRequest{MessageId: msg.ID.String()})
	require.NoError(t, err)
	require.Nil(t, resp.Message.SnoozedUntil)
}

func TestInboxUnsnoozeMessage_NotFound(t *testing.T) {
	srv, _ := newTestInboxServer()
	_, err := srv.UnsnoozeMessage(tenantCtx(uuid.New()), &inboxv1.UnsnoozeMessageRequest{MessageId: uuid.New().String()})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestInboxUnsnoozeMessage_InvalidMessageID(t *testing.T) {
	srv, _ := newTestInboxServer()
	_, err := srv.UnsnoozeMessage(tenantCtx(uuid.New()), &inboxv1.UnsnoozeMessageRequest{MessageId: "bad-id"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

// ============================================================================
// AssignMessage
// ============================================================================

func TestInboxAssignMessage_HappyPath(t *testing.T) {
	srv, deps := newTestInboxServer()
	tenantID := uuid.New()
	assigneeID := uuid.New()
	msg := seedInboxMessage(t, deps, tenantID, uuid.New(), nil)

	resp, err := srv.AssignMessage(tenantCtx(tenantID), &inboxv1.AssignMessageRequest{
		MessageId:  msg.ID.String(),
		AssigneeId: assigneeID.String(),
	})
	require.NoError(t, err)
	require.Equal(t, assigneeID.String(), *resp.Message.AssignedTo)
}

func TestInboxAssignMessage_AlreadyAssigned(t *testing.T) {
	srv, deps := newTestInboxServer()
	tenantID := uuid.New()
	firstAssignee := uuid.New()
	msg := seedInboxMessage(t, deps, tenantID, uuid.New(), func(m *models.InboxMessage) {
		m.AssignedTo = &firstAssignee
	})

	_, err := srv.AssignMessage(tenantCtx(tenantID), &inboxv1.AssignMessageRequest{
		MessageId:  msg.ID.String(),
		AssigneeId: uuid.New().String(),
	})
	requireGRPCCode(t, err, codes.AlreadyExists)
}

func TestInboxAssignMessage_InvalidAssigneeID(t *testing.T) {
	srv, deps := newTestInboxServer()
	tenantID := uuid.New()
	msg := seedInboxMessage(t, deps, tenantID, uuid.New(), nil)

	_, err := srv.AssignMessage(tenantCtx(tenantID), &inboxv1.AssignMessageRequest{
		MessageId:  msg.ID.String(),
		AssigneeId: "not-a-uuid",
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestInboxAssignMessage_InvalidMessageID(t *testing.T) {
	srv, _ := newTestInboxServer()
	_, err := srv.AssignMessage(tenantCtx(uuid.New()), &inboxv1.AssignMessageRequest{
		MessageId:  "not-a-uuid",
		AssigneeId: uuid.New().String(),
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

// ============================================================================
// ClaimMessage
// ============================================================================

func seedTeamInboxWithMember(t *testing.T, deps *inboxTestDeps, tenantID, memberUserID uuid.UUID, assignmentMode string) uuid.UUID {
	t.Helper()
	teamInboxID := uuid.New()
	require.NoError(t, deps.teamRepo.CreateTeamInbox(context.Background(), &models.TeamInbox{
		ID:             teamInboxID,
		TenantID:       tenantID,
		Name:           "Support",
		AssignmentMode: assignmentMode,
	}))
	require.NoError(t, deps.teamRepo.AddMember(context.Background(), &models.TeamInboxMember{
		TeamInboxID: teamInboxID,
		UserID:      memberUserID,
		Role:        "member",
	}))
	return teamInboxID
}

func TestInboxClaimMessage_HappyPath(t *testing.T) {
	srv, deps := newTestInboxServer()
	tenantID := uuid.New()
	userID := uuid.New()
	teamInboxID := seedTeamInboxWithMember(t, deps, tenantID, userID, "manual")
	msg := seedInboxMessage(t, deps, tenantID, uuid.New(), func(m *models.InboxMessage) {
		m.TeamInboxID = &teamInboxID
	})

	resp, err := srv.ClaimMessage(tenantCtx(tenantID), &inboxv1.ClaimMessageRequest{
		MessageId: msg.ID.String(),
		UserId:    userID.String(),
	})
	require.NoError(t, err)
	require.Equal(t, userID.String(), *resp.Message.AssignedTo)
}

func TestInboxClaimMessage_NotInTeamInbox(t *testing.T) {
	srv, deps := newTestInboxServer()
	tenantID := uuid.New()
	msg := seedInboxMessage(t, deps, tenantID, uuid.New(), nil) // TeamInboxID stays nil

	_, err := srv.ClaimMessage(tenantCtx(tenantID), &inboxv1.ClaimMessageRequest{
		MessageId: msg.ID.String(),
		UserId:    uuid.New().String(),
	})
	requireGRPCCode(t, err, codes.FailedPrecondition)
}

func TestInboxClaimMessage_NotTeamMember(t *testing.T) {
	srv, deps := newTestInboxServer()
	tenantID := uuid.New()
	teamInboxID := seedTeamInboxWithMember(t, deps, tenantID, uuid.New(), "manual")
	msg := seedInboxMessage(t, deps, tenantID, uuid.New(), func(m *models.InboxMessage) {
		m.TeamInboxID = &teamInboxID
	})

	_, err := srv.ClaimMessage(tenantCtx(tenantID), &inboxv1.ClaimMessageRequest{
		MessageId: msg.ID.String(),
		UserId:    uuid.New().String(), // not added as a member
	})
	requireGRPCCode(t, err, codes.PermissionDenied)
}

func TestInboxClaimMessage_RoundRobinRejected(t *testing.T) {
	srv, deps := newTestInboxServer()
	tenantID := uuid.New()
	userID := uuid.New()
	teamInboxID := seedTeamInboxWithMember(t, deps, tenantID, userID, "round_robin")
	msg := seedInboxMessage(t, deps, tenantID, uuid.New(), func(m *models.InboxMessage) {
		m.TeamInboxID = &teamInboxID
	})

	_, err := srv.ClaimMessage(tenantCtx(tenantID), &inboxv1.ClaimMessageRequest{
		MessageId: msg.ID.String(),
		UserId:    userID.String(),
	})
	requireGRPCCode(t, err, codes.FailedPrecondition)
}

func TestInboxClaimMessage_AlreadyAssigned(t *testing.T) {
	srv, deps := newTestInboxServer()
	tenantID := uuid.New()
	userID := uuid.New()
	teamInboxID := seedTeamInboxWithMember(t, deps, tenantID, userID, "manual")
	otherAssignee := uuid.New()
	msg := seedInboxMessage(t, deps, tenantID, uuid.New(), func(m *models.InboxMessage) {
		m.TeamInboxID = &teamInboxID
		m.AssignedTo = &otherAssignee
	})

	_, err := srv.ClaimMessage(tenantCtx(tenantID), &inboxv1.ClaimMessageRequest{
		MessageId: msg.ID.String(),
		UserId:    userID.String(),
	})
	requireGRPCCode(t, err, codes.AlreadyExists)
}

func TestInboxClaimMessage_MessageNotFound(t *testing.T) {
	srv, _ := newTestInboxServer()
	_, err := srv.ClaimMessage(tenantCtx(uuid.New()), &inboxv1.ClaimMessageRequest{
		MessageId: uuid.New().String(),
		UserId:    uuid.New().String(),
	})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestInboxClaimMessage_InvalidUserID(t *testing.T) {
	srv, deps := newTestInboxServer()
	tenantID := uuid.New()
	msg := seedInboxMessage(t, deps, tenantID, uuid.New(), nil)

	_, err := srv.ClaimMessage(tenantCtx(tenantID), &inboxv1.ClaimMessageRequest{
		MessageId: msg.ID.String(),
		UserId:    "not-a-uuid",
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

// ============================================================================
// GetUnreadCount
// ============================================================================

func TestInboxGetUnreadCount_AggregatesByChannel(t *testing.T) {
	srv, deps := newTestInboxServer()
	tenantID := uuid.New()
	userID := uuid.New()
	futureSnooze := time.Now().Add(time.Hour).UTC()

	seedInboxMessage(t, deps, tenantID, userID, func(m *models.InboxMessage) { m.Channel = "email"; m.IsRead = false })
	seedInboxMessage(t, deps, tenantID, userID, func(m *models.InboxMessage) { m.Channel = "email"; m.IsRead = false })
	seedInboxMessage(t, deps, tenantID, userID, func(m *models.InboxMessage) { m.Channel = "chat"; m.IsRead = false })
	seedInboxMessage(t, deps, tenantID, userID, func(m *models.InboxMessage) { m.Channel = "email"; m.IsRead = true })
	seedInboxMessage(t, deps, tenantID, userID, func(m *models.InboxMessage) { m.Channel = "email"; m.IsRead = false; m.IsArchived = true })
	seedInboxMessage(t, deps, tenantID, userID, func(m *models.InboxMessage) {
		m.Channel = "email"
		m.IsRead = false
		m.SnoozedUntil = &futureSnooze
	})
	seedInboxMessage(t, deps, tenantID, uuid.New(), func(m *models.InboxMessage) { m.Channel = "email"; m.IsRead = false }) // other user

	resp, err := srv.GetUnreadCount(context.Background(), &inboxv1.GetUnreadCountRequest{UserId: userID.String()})
	require.NoError(t, err)
	require.EqualValues(t, 3, resp.Total)

	var emailCount, chatCount int32
	for _, c := range resp.ByChannel {
		switch c.Channel {
		case inboxv1.Channel_CHANNEL_EMAIL:
			emailCount = c.Count
		case inboxv1.Channel_CHANNEL_CHAT:
			chatCount = c.Count
		}
	}
	require.EqualValues(t, 2, emailCount)
	require.EqualValues(t, 1, chatCount)
}

func TestInboxGetUnreadCount_InvalidUserID(t *testing.T) {
	srv, _ := newTestInboxServer()
	_, err := srv.GetUnreadCount(context.Background(), &inboxv1.GetUnreadCountRequest{UserId: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

// ============================================================================
// BulkMarkRead / BulkArchive
// ============================================================================

func TestInboxBulkMarkRead_HappyPath(t *testing.T) {
	srv, deps := newTestInboxServer()
	tenantID := uuid.New()
	msg1 := seedInboxMessage(t, deps, tenantID, uuid.New(), func(m *models.InboxMessage) { m.IsRead = false })
	msg2 := seedInboxMessage(t, deps, tenantID, uuid.New(), func(m *models.InboxMessage) { m.IsRead = false })

	resp, err := srv.BulkMarkRead(tenantCtx(tenantID), &inboxv1.BulkMarkReadRequest{
		MessageIds: []string{msg1.ID.String(), msg2.ID.String()},
	})
	require.NoError(t, err)
	require.EqualValues(t, 2, resp.UpdatedCount)

	got1, err := deps.messageRepo.GetByID(context.Background(), msg1.ID, tenantID)
	require.NoError(t, err)
	require.True(t, got1.IsRead)
}

func TestInboxBulkMarkRead_InvalidIDInList(t *testing.T) {
	srv, deps := newTestInboxServer()
	tenantID := uuid.New()
	msg := seedInboxMessage(t, deps, tenantID, uuid.New(), nil)

	_, err := srv.BulkMarkRead(tenantCtx(tenantID), &inboxv1.BulkMarkReadRequest{
		MessageIds: []string{msg.ID.String(), "not-a-uuid"},
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestInboxBulkMarkRead_MissingTenant(t *testing.T) {
	srv, _ := newTestInboxServer()
	_, err := srv.BulkMarkRead(context.Background(), &inboxv1.BulkMarkReadRequest{
		MessageIds: []string{uuid.New().String()},
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestInboxBulkArchive_HappyPath(t *testing.T) {
	srv, deps := newTestInboxServer()
	tenantID := uuid.New()
	msg1 := seedInboxMessage(t, deps, tenantID, uuid.New(), func(m *models.InboxMessage) { m.IsArchived = false })
	msg2 := seedInboxMessage(t, deps, tenantID, uuid.New(), func(m *models.InboxMessage) { m.IsArchived = false })

	resp, err := srv.BulkArchive(tenantCtx(tenantID), &inboxv1.BulkArchiveRequest{
		MessageIds: []string{msg1.ID.String(), msg2.ID.String()},
	})
	require.NoError(t, err)
	require.EqualValues(t, 2, resp.UpdatedCount)

	got2, err := deps.messageRepo.GetByID(context.Background(), msg2.ID, tenantID)
	require.NoError(t, err)
	require.True(t, got2.IsArchived)
}

func TestInboxBulkArchive_InvalidIDInList(t *testing.T) {
	srv, deps := newTestInboxServer()
	tenantID := uuid.New()
	msg := seedInboxMessage(t, deps, tenantID, uuid.New(), nil)

	_, err := srv.BulkArchive(tenantCtx(tenantID), &inboxv1.BulkArchiveRequest{
		MessageIds: []string{msg.ID.String(), "not-a-uuid"},
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}
