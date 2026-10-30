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

type fakeNotificationClient struct {
	items []NotificationData
	err   error

	reads []string
}

func (f *fakeNotificationClient) ListNotifications(_ context.Context, _ uuid.UUID, _ time.Time) ([]NotificationData, error) {
	return f.items, f.err
}

func (f *fakeNotificationClient) MarkRead(_ context.Context, id string, _ uuid.UUID) error {
	if f.err != nil {
		return f.err
	}
	f.reads = append(f.reads, id)
	return nil
}

func TestNotificationAdapter_Channel(t *testing.T) {
	assert.Equal(t, "notification", NewNotificationAdapter(nil).Channel())
}

func TestNotificationAdapter_FetchNewMessages_MapsFields(t *testing.T) {
	userID := uuid.New()
	receivedAt := time.Now().UTC().Truncate(time.Second)
	client := &fakeNotificationClient{items: []NotificationData{
		{
			NotificationID: "n-1", Title: "Urlaub genehmigt", Body: "Dein Antrag wurde angenommen",
			ModuleName: "HR", DeepLink: "/hr/leave/42", ReceivedAt: receivedAt,
		},
		{NotificationID: "n-2", Title: "Hinweis", Body: strings.Repeat("y", 199) + "ü" + strings.Repeat("y", 50)},
	}}

	got, err := NewNotificationAdapter(client).FetchNewMessages(context.Background(), userID, time.Time{})
	require.NoError(t, err)
	require.Len(t, got, 2)

	first := got[0]
	assert.NotEqual(t, uuid.Nil, first.ID)
	assert.Equal(t, userID, first.UserID)
	assert.Equal(t, "notification", first.Channel)
	assert.Equal(t, "n-1", first.SourceID)
	assert.Equal(t, "HR", first.SenderName)
	assert.Equal(t, "Urlaub genehmigt", first.Subject)
	assert.Equal(t, "Dein Antrag wurde angenommen", first.Preview)
	assert.Equal(t, "/hr/leave/42", first.DeepLink)
	assert.Equal(t, receivedAt, first.ReceivedAt)
	assert.Nil(t, first.SenderEmail)
	assert.Nil(t, first.SenderID)

	assert.Equal(t, "System", got[1].SenderName, "notifications without a module come from System")
	assert.Equal(t, strings.Repeat("y", 199), got[1].Preview, "preview is cut before the umlaut that straddles byte 200")
}

func TestNotificationAdapter_FetchNewMessages_NilClientDegradesGracefully(t *testing.T) {
	got, err := NewNotificationAdapter(nil).FetchNewMessages(context.Background(), uuid.New(), time.Time{})
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestNotificationAdapter_FetchNewMessages_ClientErrorIsWrapped(t *testing.T) {
	boom := errors.New("notification down")
	got, err := NewNotificationAdapter(&fakeNotificationClient{err: boom}).FetchNewMessages(context.Background(), uuid.New(), time.Time{})
	assert.Nil(t, got)
	require.ErrorIs(t, err, boom)
	assert.Contains(t, err.Error(), "notification adapter fetch")
}

func TestNotificationAdapter_ReplyAndForwardUnsupported(t *testing.T) {
	a := NewNotificationAdapter(nil)

	assert.ErrorIs(t, a.HandleReply(context.Background(), "n-1", uuid.New(), "x"), ErrReplyNotSupported)
	assert.ErrorIs(t, a.HandleForward(context.Background(), "n-1", uuid.New(), "a@b.de", ""), ErrForwardNotSupported)
}

func TestNotificationAdapter_MarkReadOnSource(t *testing.T) {
	client := &fakeNotificationClient{}
	require.NoError(t, NewNotificationAdapter(client).MarkReadOnSource(context.Background(), "n-1", uuid.New()))
	assert.Equal(t, []string{"n-1"}, client.reads)

	assert.NoError(t, NewNotificationAdapter(nil).MarkReadOnSource(context.Background(), "n-1", uuid.New()), "nil client is a no-op")

	boom := errors.New("notification down")
	assert.ErrorIs(t, NewNotificationAdapter(&fakeNotificationClient{err: boom}).MarkReadOnSource(context.Background(), "n", uuid.New()), boom)
}
