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

type fakeEmailClient struct {
	messages []EmailMessageData
	err      error

	replies  []string
	forwards []string
	reads    []string
}

func (f *fakeEmailClient) ListMessages(_ context.Context, _ uuid.UUID, _ time.Time) ([]EmailMessageData, error) {
	return f.messages, f.err
}

func (f *fakeEmailClient) SendReply(_ context.Context, threadID string, _ uuid.UUID, body string) error {
	if f.err != nil {
		return f.err
	}
	f.replies = append(f.replies, threadID+"|"+body)
	return nil
}

func (f *fakeEmailClient) ForwardEmail(_ context.Context, messageID string, _ uuid.UUID, to, note string) error {
	if f.err != nil {
		return f.err
	}
	f.forwards = append(f.forwards, messageID+"|"+to+"|"+note)
	return nil
}

func (f *fakeEmailClient) MarkRead(_ context.Context, threadID string, _ uuid.UUID) error {
	if f.err != nil {
		return f.err
	}
	f.reads = append(f.reads, threadID)
	return nil
}

func TestEmailAdapter_Channel(t *testing.T) {
	assert.Equal(t, "email", NewEmailAdapter(nil).Channel())
}

func TestEmailAdapter_FetchNewMessages_MapsFields(t *testing.T) {
	userID := uuid.New()
	receivedAt := time.Now().UTC().Truncate(time.Second)
	client := &fakeEmailClient{messages: []EmailMessageData{
		{
			ThreadID: "thread-1", MessageID: "msg-1", FromName: "Erika Muster",
			FromEmail: "erika@kunde.de", Subject: "Angebot", Body: "kurz", ReceivedAt: receivedAt,
		},
		{ThreadID: "thread-2", MessageID: "msg-2", Body: strings.Repeat("x", 199) + "ü" + strings.Repeat("x", 50)},
	}}

	got, err := NewEmailAdapter(client).FetchNewMessages(context.Background(), userID, time.Time{})
	require.NoError(t, err)
	require.Len(t, got, 2)

	first := got[0]
	assert.NotEqual(t, uuid.Nil, first.ID)
	assert.Equal(t, userID, first.UserID)
	assert.Equal(t, "email", first.Channel)
	assert.Equal(t, "thread-1", first.SourceID, "one inbox item per thread, not per message")
	assert.Equal(t, "Erika Muster", first.SenderName)
	require.NotNil(t, first.SenderEmail)
	assert.Equal(t, "erika@kunde.de", *first.SenderEmail)
	assert.Equal(t, "Angebot", first.Subject)
	assert.Equal(t, "kurz", first.Preview)
	assert.Equal(t, "/mails/msg-1", first.DeepLink)
	assert.Equal(t, receivedAt, first.ReceivedAt)

	assert.Equal(t, strings.Repeat("x", 199), got[1].Preview, "preview is cut before the umlaut that straddles byte 200")
	assert.NotEqual(t, first.ID, got[1].ID)
}

func TestEmailAdapter_FetchNewMessages_EmptyResultIsNonNilSlice(t *testing.T) {
	got, err := NewEmailAdapter(&fakeEmailClient{}).FetchNewMessages(context.Background(), uuid.New(), time.Time{})
	require.NoError(t, err)
	assert.NotNil(t, got)
	assert.Empty(t, got)
}

func TestEmailAdapter_FetchNewMessages_NilClientDegradesGracefully(t *testing.T) {
	got, err := NewEmailAdapter(nil).FetchNewMessages(context.Background(), uuid.New(), time.Time{})
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestEmailAdapter_FetchNewMessages_ClientErrorIsWrapped(t *testing.T) {
	boom := errors.New("imap down")
	got, err := NewEmailAdapter(&fakeEmailClient{err: boom}).FetchNewMessages(context.Background(), uuid.New(), time.Time{})
	assert.Nil(t, got)
	require.ErrorIs(t, err, boom)
	assert.Contains(t, err.Error(), "email adapter fetch")
}

func TestEmailAdapter_HandleReplyAndForward_DelegateToClient(t *testing.T) {
	client := &fakeEmailClient{}
	a := NewEmailAdapter(client)
	userID := uuid.New()

	require.NoError(t, a.HandleReply(context.Background(), "thread-1", userID, "Danke"))
	require.NoError(t, a.HandleForward(context.Background(), "msg-1", userID, "chef@firma.de", "zur Info"))

	assert.Equal(t, []string{"thread-1|Danke"}, client.replies)
	assert.Equal(t, []string{"msg-1|chef@firma.de|zur Info"}, client.forwards)
}

func TestEmailAdapter_HandleReplyAndForward_NilClientReturnsError(t *testing.T) {
	a := NewEmailAdapter(nil)

	assert.Error(t, a.HandleReply(context.Background(), "thread-1", uuid.New(), "x"))
	assert.Error(t, a.HandleForward(context.Background(), "msg-1", uuid.New(), "a@b.de", ""))
}

func TestEmailAdapter_HandleReplyAndForward_ClientErrorPassesThrough(t *testing.T) {
	boom := errors.New("smtp down")
	a := NewEmailAdapter(&fakeEmailClient{err: boom})

	assert.ErrorIs(t, a.HandleReply(context.Background(), "t", uuid.New(), "x"), boom)
	assert.ErrorIs(t, a.HandleForward(context.Background(), "m", uuid.New(), "a@b.de", ""), boom)
}

func TestEmailAdapter_MarkReadOnSource(t *testing.T) {
	client := &fakeEmailClient{}
	require.NoError(t, NewEmailAdapter(client).MarkReadOnSource(context.Background(), "thread-1", uuid.New()))
	assert.Equal(t, []string{"thread-1"}, client.reads)

	assert.NoError(t, NewEmailAdapter(nil).MarkReadOnSource(context.Background(), "thread-1", uuid.New()), "nil client is a no-op")

	boom := errors.New("imap down")
	assert.ErrorIs(t, NewEmailAdapter(&fakeEmailClient{err: boom}).MarkReadOnSource(context.Background(), "t", uuid.New()), boom)
}
