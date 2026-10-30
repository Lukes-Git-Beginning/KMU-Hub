package main

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/kmuhub/kmuhub/internal/inbox/adapter"
	"github.com/kmuhub/kmuhub/internal/middleware"
	chatv1 "github.com/kmuhub/kmuhub/proto/chat/v1"
)

// chatGRPCClient implements adapter.ChatClient against the real chat service.
// channelID parameters carry the chat *channel* id -- the same id the chat
// service emits as ResourceID on chat.dm.new / chat.mention and the
// InboxConsumer stores as InboxMessage.SourceID.
type chatGRPCClient struct {
	client chatv1.ChatServiceClient
}

// ListDMsAndMentions returns nothing.
//
// lean: no chat RPC lists DMs+mentions since a timestamp, and nothing calls
// FetchNewMessages (the live inbox path is event-driven via
// InboxConsumer.HandleEvent). Build it on GetUserMentions/ListDMs once a
// polling worker starts calling this path.
func (c *chatGRPCClient) ListDMsAndMentions(_ context.Context, _ uuid.UUID, _ time.Time) ([]adapter.ChatMessageData, error) {
	return nil, nil
}

// CreateMessage posts the reply into the chat channel as userID. The chat
// service enforces channel membership of the sender.
func (c *chatGRPCClient) CreateMessage(ctx context.Context, channelID string, userID uuid.UUID, body string) error {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return fmt.Errorf("chat reply: %w", err)
	}
	_, err = c.client.SendMessage(ctx, &chatv1.SendMessageRequest{
		ChannelId: channelID,
		Content:   body,
		CreatedBy: userID.String(),
		TenantId:  tenantID.String(),
	})
	return chatClientError("send chat message", err)
}

// MarkRead marks the chat channel as read for userID up to its newest message.
func (c *chatGRPCClient) MarkRead(ctx context.Context, channelID string, userID uuid.UUID) error {
	resp, err := c.client.GetMessages(ctx, &chatv1.GetMessagesRequest{
		ChannelId: channelID,
		UserId:    userID.String(),
		Limit:     1,
	})
	if err != nil {
		return chatClientError("get newest chat message", err)
	}
	if len(resp.GetMessages()) == 0 {
		return nil
	}
	_, err = c.client.MarkChannelRead(ctx, &chatv1.MarkChannelReadRequest{
		ChannelId: channelID,
		UserId:    userID.String(),
		MessageId: resp.GetMessages()[0].GetId(),
	})
	return chatClientError("mark chat channel read", err)
}

// chatClientError translates the chat service's gRPC codes into the adapter
// sentinels the inbox maps to its own status codes.
func chatClientError(op string, err error) error {
	if err == nil {
		return nil
	}
	switch status.Code(err) {
	case codes.Unavailable, codes.DeadlineExceeded:
		return fmt.Errorf("%s: %w: %v", op, adapter.ErrChannelUnavailable, err)
	case codes.PermissionDenied:
		return fmt.Errorf("%s: %w: %v", op, adapter.ErrReplyNotPermitted, err)
	}
	return fmt.Errorf("%s: %w", op, err)
}
