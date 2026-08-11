package adapter

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeChatClient implements ChatClient for testing.
type fakeChatClient struct {
	messages    []ChatMessageData
	fetchErr    error
	createErr   error
	markReadErr error

	createCalls   []createMessageCall
	markReadCalls []markReadCall
}

type createMessageCall struct {
	channelID string
	userID    uuid.UUID
	body      string
}

type markReadCall struct {
	channelID string
	userID    uuid.UUID
}

func (f *fakeChatClient) ListDMsAndMentions(_ context.Context, _ uuid.UUID, _ time.Time) ([]ChatMessageData, error) {
	if f.fetchErr != nil {
		return nil, f.fetchErr
	}
	return f.messages, nil
}

func (f *fakeChatClient) CreateMessage(_ context.Context, channelID string, userID uuid.UUID, body string) error {
	if f.createErr != nil {
		return f.createErr
	}
	f.createCalls = append(f.createCalls, createMessageCall{channelID: channelID, userID: userID, body: body})
	return nil
}

func (f *fakeChatClient) MarkRead(_ context.Context, channelID string, userID uuid.UUID) error {
	if f.markReadErr != nil {
		return f.markReadErr
	}
	f.markReadCalls = append(f.markReadCalls, markReadCall{channelID: channelID, userID: userID})
	return nil
}

func TestChatAdapter_Channel(t *testing.T) {
	a := NewChatAdapter(nil)
	assert.Equal(t, "chat", a.Channel())
}

func TestChatAdapter_FetchNewMessages_MapsDMsAndMentions(t *testing.T) {
	userID := uuid.New()
	authorID := uuid.New()
	receivedAt := time.Now().UTC()
	client := &fakeChatClient{
		messages: []ChatMessageData{
			{
				MessageID: "msg-1", ChannelID: "chan-1", ChannelName: "general",
				IsDM: false, AuthorID: authorID, AuthorName: "Alice", Body: "hello",
				ReceivedAt: receivedAt,
			},
			{
				MessageID: "msg-2", ChannelID: "dm-1", ChannelName: "",
				IsDM: true, AuthorID: authorID, AuthorName: "Alice", Body: strings.Repeat("x", 250),
				ReceivedAt: receivedAt,
			},
		},
	}
	a := NewChatAdapter(client)

	got, err := a.FetchNewMessages(context.Background(), userID, receivedAt)

	require.NoError(t, err)
	require.Len(t, got, 2)

	mention := got[0]
	assert.Equal(t, "chat", mention.Channel)
	assert.Equal(t, "msg-1", mention.SourceID)
	assert.Equal(t, "general", mention.Subject)
	assert.Equal(t, "hello", mention.Preview)
	assert.Equal(t, "/chat/chan-1", mention.DeepLink)
	require.NotNil(t, mention.SenderID)
	assert.Equal(t, authorID, *mention.SenderID)

	dm := got[1]
	assert.Equal(t, "Direktnachricht", dm.Subject)
	assert.Len(t, dm.Preview, 200, "preview must be truncated to 200 chars")
}

func TestChatAdapter_FetchNewMessages_NilClientDegradesGracefully(t *testing.T) {
	a := NewChatAdapter(nil)

	got, err := a.FetchNewMessages(context.Background(), uuid.New(), time.Now())

	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestChatAdapter_FetchNewMessages_ClientErrorIsWrapped(t *testing.T) {
	client := &fakeChatClient{fetchErr: errors.New("chat service unavailable")}
	a := NewChatAdapter(client)

	_, err := a.FetchNewMessages(context.Background(), uuid.New(), time.Now())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "chat service unavailable")
}

func TestChatAdapter_HandleReply_Success(t *testing.T) {
	client := &fakeChatClient{}
	a := NewChatAdapter(client)
	userID := uuid.New()

	err := a.HandleReply(context.Background(), "chan-1", userID, "reply body")

	require.NoError(t, err)
	require.Len(t, client.createCalls, 1)
	assert.Equal(t, "chan-1", client.createCalls[0].channelID)
	assert.Equal(t, userID, client.createCalls[0].userID)
	assert.Equal(t, "reply body", client.createCalls[0].body)
}

func TestChatAdapter_HandleReply_NilClientReturnsError(t *testing.T) {
	a := NewChatAdapter(nil)

	err := a.HandleReply(context.Background(), "chan-1", uuid.New(), "reply body")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not configured")
}

func TestChatAdapter_HandleForward_ReturnsNotSupported(t *testing.T) {
	client := &fakeChatClient{}
	a := NewChatAdapter(client)

	err := a.HandleForward(context.Background(), "chan-1", uuid.New(), "someone@example.test", "note")

	assert.ErrorIs(t, err, ErrForwardNotSupported)
}

func TestChatAdapter_MarkReadOnSource_Success(t *testing.T) {
	client := &fakeChatClient{}
	a := NewChatAdapter(client)
	userID := uuid.New()

	err := a.MarkReadOnSource(context.Background(), "chan-1", userID)

	require.NoError(t, err)
	require.Len(t, client.markReadCalls, 1)
	assert.Equal(t, "chan-1", client.markReadCalls[0].channelID)
	assert.Equal(t, userID, client.markReadCalls[0].userID)
}

func TestChatAdapter_MarkReadOnSource_NilClientIsNoOp(t *testing.T) {
	a := NewChatAdapter(nil)

	err := a.MarkReadOnSource(context.Background(), "chan-1", uuid.New())

	require.NoError(t, err)
}

func TestChatAdapter_MarkReadOnSource_ClientError(t *testing.T) {
	client := &fakeChatClient{markReadErr: errors.New("mark read failed")}
	a := NewChatAdapter(client)

	err := a.MarkReadOnSource(context.Background(), "chan-1", uuid.New())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "mark read failed")
}
