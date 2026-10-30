package main

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/kmuhub/kmuhub/internal/inbox/adapter"
	"github.com/kmuhub/kmuhub/internal/inbox/message"
	"github.com/kmuhub/kmuhub/internal/inbox/routing"
	"github.com/kmuhub/kmuhub/internal/inbox/team"
	"github.com/kmuhub/kmuhub/internal/inbox/thread"
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/notification/event"
	"github.com/kmuhub/kmuhub/internal/server"
	"github.com/kmuhub/kmuhub/internal/testutil"
	chatv1 "github.com/kmuhub/kmuhub/proto/chat/v1"
	inboxv1 "github.com/kmuhub/kmuhub/proto/inbox/v1"
)

// fakeChatService records the chat RPCs the inbox chat client issues and
// answers SendMessage with sendErr.
type fakeChatService struct {
	chatv1.ChatServiceClient
	sendErr  error
	sent     []*chatv1.SendMessageRequest
	markRead []*chatv1.MarkChannelReadRequest
	newestID string
}

func (f *fakeChatService) SendMessage(_ context.Context, req *chatv1.SendMessageRequest, _ ...grpc.CallOption) (*chatv1.SendMessageResponse, error) {
	f.sent = append(f.sent, req)
	if f.sendErr != nil {
		return nil, f.sendErr
	}
	return &chatv1.SendMessageResponse{}, nil
}

func (f *fakeChatService) GetMessages(_ context.Context, _ *chatv1.GetMessagesRequest, _ ...grpc.CallOption) (*chatv1.GetMessagesResponse, error) {
	if f.newestID == "" {
		return &chatv1.GetMessagesResponse{}, nil
	}
	return &chatv1.GetMessagesResponse{Messages: []*chatv1.MessageInfo{{Id: f.newestID}}}, nil
}

func (f *fakeChatService) MarkChannelRead(_ context.Context, req *chatv1.MarkChannelReadRequest, _ ...grpc.CallOption) (*chatv1.MarkChannelReadResponse, error) {
	f.markRead = append(f.markRead, req)
	return &chatv1.MarkChannelReadResponse{}, nil
}

// TestInboxReply_ChatDMReachesTheChatChannel runs a chat.dm.new event through
// the consumer into the recipient's inbox against the real database as
// kmuhub_app, then answers it through the inbox gRPC server wired the way
// main wires it. The reply must reach the chat service as a message in the
// DM channel (the inbox message's SourceID), sent by the replier in the
// replier's tenant. A chat service that is down or refuses the sender ends
// as Unavailable / PermissionDenied, and no reply lands in the thread.
func TestInboxReply_ChatDMReachesTheChatChannel(t *testing.T) {
	testutil.SkipIfNoDB(t)

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Inbox Chat Reply Tenant")
	seedUser := func() uuid.UUID {
		id := testutil.SeedRow(t, pool, "users", map[string]any{
			"tenant_id":     tenantID,
			"email":         fmt.Sprintf("inbox-chat-reply-%s@tenant.local", uuid.NewString()[:8]),
			"password_hash": "$argon2id$v=19$test",
		})
		t.Cleanup(func() { testutil.CleanupRow(t, pool, "users", id) })
		return id
	}
	sender, recipient := seedUser(), seedUser()

	chat := &fakeChatService{}
	registry := adapter.NewAdapterRegistry()
	registry.Register(adapter.NewChatAdapter(&chatGRPCClient{client: chat}))

	msgRepo := message.NewPostgresRepository(pool)
	msgSvc := message.NewService(msgRepo, registry)
	routingSvc := routing.NewService(routing.NewPostgresRepository(pool), msgRepo, nil)
	consumer := &InboxConsumer{
		messageService: msgSvc,
		messageRepo:    msgRepo,
		routingService: routingSvc,
		adapters:       registry,
	}
	inbox := server.NewInboxGRPCServer(msgSvc, team.NewService(team.NewPostgresRepository(pool), msgRepo),
		routingSvc, thread.NewService(thread.NewPostgresRepository(pool)))

	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	channelID := uuid.NewString()
	if err := consumer.HandleEvent(ctx, models.EventPayload{
		Type:          "chat.dm.new",
		TenantID:      tenantID,
		Priority:      models.PriorityNormal,
		ActorID:       sender.String(),
		ResourceID:    channelID,
		ModuleID:      event.ModuleChat,
		Title:         "Anna sent you a message",
		Body:          "Hast du kurz Zeit?",
		DeepLink:      "/chat/dm/" + channelID,
		TargetUserIDs: []string{recipient.String()},
		Timestamp:     time.Now().UTC(),
	}); err != nil {
		t.Fatalf("HandleEvent: %v", err)
	}

	var inboxID uuid.UUID
	if err := pool.QueryRow(testutil.WithSystemCtx(context.Background()),
		`SELECT id FROM inbox_messages WHERE user_id = $1 AND channel = 'chat' AND source_id = $2`,
		recipient, channelID).Scan(&inboxID); err != nil {
		t.Fatalf("inbox message for the DM recipient: %v", err)
	}
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "inbox_messages", inboxID) })

	callerCtx := context.WithValue(ctx, middleware.UserIDKey, recipient.String())
	reply := func(body string) error {
		_, err := inbox.ReplyToMessage(callerCtx, &inboxv1.ReplyToMessageRequest{
			MessageId: inboxID.String(),
			UserId:    recipient.String(),
			Body:      body,
		})
		return err
	}
	threadReplies := func() int {
		var n int
		if err := pool.QueryRow(testutil.WithSystemCtx(context.Background()),
			`SELECT count(*) FROM inbox_thread_messages WHERE message_id = $1`, inboxID).Scan(&n); err != nil {
			t.Fatalf("count thread replies: %v", err)
		}
		return n
	}

	t.Run("reply is sent into the DM channel as the replier", func(t *testing.T) {
		if err := reply("Ja, gleich."); err != nil {
			t.Fatalf("ReplyToMessage: %v", err)
		}
		if len(chat.sent) != 1 {
			t.Fatalf("SendMessage calls = %d, want 1", len(chat.sent))
		}
		got := chat.sent[0]
		if got.GetChannelId() != channelID || got.GetCreatedBy() != recipient.String() ||
			got.GetTenantId() != tenantID.String() || got.GetContent() != "Ja, gleich." {
			t.Fatalf("SendMessage(channel=%s, created_by=%s, tenant=%s, content=%q), want (%s, %s, %s, %q)",
				got.GetChannelId(), got.GetCreatedBy(), got.GetTenantId(), got.GetContent(),
				channelID, recipient, tenantID, "Ja, gleich.")
		}
	})

	before := threadReplies()

	t.Run("chat service down -> Unavailable, no thread entry", func(t *testing.T) {
		chat.sendErr = status.Error(codes.Unavailable, "connection refused")
		err := reply("geht nicht raus")
		if status.Code(err) != codes.Unavailable {
			t.Fatalf("code = %v (%v), want Unavailable", status.Code(err), err)
		}
		if n := threadReplies(); n != before {
			t.Fatalf("thread replies = %d, want %d (a failed reply must not land in the thread)", n, before)
		}
	})

	t.Run("chat service refuses the sender -> PermissionDenied", func(t *testing.T) {
		chat.sendErr = status.Error(codes.PermissionDenied, "user is not a member of this channel")
		if err := reply("fremder Kanal"); status.Code(err) != codes.PermissionDenied {
			t.Fatalf("code = %v (%v), want PermissionDenied", status.Code(err), err)
		}
	})

	t.Run("marking read reaches the chat channel up to its newest message", func(t *testing.T) {
		chat.newestID = uuid.NewString()
		if _, err := inbox.MarkRead(callerCtx, &inboxv1.MarkReadRequest{
			MessageId: inboxID.String(),
			UserId:    recipient.String(),
		}); err != nil {
			t.Fatalf("MarkRead: %v", err)
		}
		if len(chat.markRead) != 1 {
			t.Fatalf("MarkChannelRead calls = %d, want 1", len(chat.markRead))
		}
		got := chat.markRead[0]
		if got.GetChannelId() != channelID || got.GetUserId() != recipient.String() || got.GetMessageId() != chat.newestID {
			t.Fatalf("MarkChannelRead(%s, %s, %s), want (%s, %s, %s)",
				got.GetChannelId(), got.GetUserId(), got.GetMessageId(), channelID, recipient, chat.newestID)
		}
	})
}

// TestChatAdapter_NoClientIsUnavailable pins the degraded path: without a chat
// client a reply fails with the Unavailable sentinel the inbox maps to
// codes.Unavailable, not with a bare error that ends as Internal.
func TestChatAdapter_NoClientIsUnavailable(t *testing.T) {
	err := adapter.NewChatAdapter(nil).HandleReply(context.Background(), uuid.NewString(), uuid.New(), "x")
	if !errors.Is(err, adapter.ErrChannelUnavailable) {
		t.Fatalf("err = %v, want adapter.ErrChannelUnavailable", err)
	}
}
