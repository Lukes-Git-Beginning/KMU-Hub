package message

// Covers emitMessageEvents, the notification side-effect of Create that no
// existing service_test.go exercises: none of those tests ever call
// SetEventEmitter, so the mention/DM event payloads have never been
// asserted against. eventCapturingRepo overrides only the two repo reads
// emitMessageEvents needs (GetDMRecipient, GetChannelName) — the shared
// MockRepository has no test hooks for them, so overriding via embedding
// avoids touching the existing mock used by every other test in this file.

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
)

type eventCapturingRepo struct {
	*MockRepository
	dmRecipient *uuid.UUID
	channelName string
}

func (r *eventCapturingRepo) GetDMRecipient(_ context.Context, _, _ uuid.UUID) (*uuid.UUID, error) {
	return r.dmRecipient, nil
}

func (r *eventCapturingRepo) GetChannelName(_ context.Context, _ uuid.UUID) (string, error) {
	return r.channelName, nil
}

type capturingEmitter struct {
	events []models.EventPayload
}

func (e *capturingEmitter) EmitChatEvent(_ context.Context, payload models.EventPayload) error {
	e.events = append(e.events, payload)
	return nil
}

func newEventTestService() (*Service, *eventCapturingRepo, *capturingEmitter) {
	repo := &eventCapturingRepo{MockRepository: NewMockRepository(), channelName: "general"}
	svc := NewService(repo)
	emitter := &capturingEmitter{}
	svc.SetEventEmitter(emitter)
	return svc, repo, emitter
}

func TestEmitMessageEvents_MentionExcludesSender(t *testing.T) {
	svc, repo, emitter := newEventTestService()

	channelID := uuid.New()
	senderID := uuid.New()
	mentionedID := uuid.New()
	repo.AddChannel(channelID, false)
	repo.AddMember(channelID, senderID, models.ChannelRoleMember)
	repo.AddMember(channelID, mentionedID, models.ChannelRoleMember)
	repo.AddUser(senderID, "Alice", "Sender")
	repo.AddUser(mentionedID, "Bob", "Mentioned")

	_, err := svc.Create(context.Background(), CreateInput{
		ChannelID:        channelID,
		Content:          "Hey @Bob and @Alice",
		CreatedBy:        senderID,
		MentionedUserIDs: []uuid.UUID{senderID, mentionedID}, // sender mentions self AND another member
	})
	require.NoError(t, err)

	require.Len(t, emitter.events, 1, "expected exactly one chat.mention event, no DM event on a non-DM channel")
	ev := emitter.events[0]
	assert.Equal(t, "chat.mention", ev.Type)
	assert.Equal(t, models.PriorityNormal, ev.Priority)
	assert.Equal(t, senderID.String(), ev.ActorID)
	assert.Equal(t, channelID.String(), ev.ResourceID)
	require.Len(t, ev.TargetUserIDs, 1, "sender must be excluded from its own mention notification")
	assert.Equal(t, mentionedID.String(), ev.TargetUserIDs[0])
}

func TestEmitMessageEvents_SelfMentionOnlyEmitsNothing(t *testing.T) {
	svc, repo, emitter := newEventTestService()

	channelID := uuid.New()
	senderID := uuid.New()
	repo.AddChannel(channelID, false)
	repo.AddMember(channelID, senderID, models.ChannelRoleMember)
	repo.AddUser(senderID, "Alice", "Sender")

	_, err := svc.Create(context.Background(), CreateInput{
		ChannelID:        channelID,
		Content:          "Note to self @Alice",
		CreatedBy:        senderID,
		MentionedUserIDs: []uuid.UUID{senderID},
	})
	require.NoError(t, err)

	assert.Empty(t, emitter.events, "a mention that only targets the sender must not notify anyone")
}

func TestEmitMessageEvents_DMEmitsToRecipient(t *testing.T) {
	svc, repo, emitter := newEventTestService()

	channelID := uuid.New()
	senderID := uuid.New()
	recipientID := uuid.New()
	repo.dmRecipient = &recipientID
	repo.AddChannel(channelID, false)
	repo.AddMember(channelID, senderID, models.ChannelRoleMember)
	repo.AddUser(senderID, "Alice", "Sender")

	_, err := svc.Create(context.Background(), CreateInput{
		ChannelID: channelID,
		Content:   "Hi there",
		CreatedBy: senderID,
	})
	require.NoError(t, err)

	require.Len(t, emitter.events, 1)
	ev := emitter.events[0]
	assert.Equal(t, "chat.dm.new", ev.Type)
	assert.Equal(t, senderID.String(), ev.ActorID)
	require.Len(t, ev.TargetUserIDs, 1)
	assert.Equal(t, recipientID.String(), ev.TargetUserIDs[0])
}

func TestEmitMessageEvents_GuestMessageEmitsNothing(t *testing.T) {
	svc, repo, emitter := newEventTestService()

	channelID := uuid.New()
	guestSessionID := uuid.New()
	otherUserID := uuid.New()
	repo.dmRecipient = &otherUserID // even if a recipient exists, guest messages must not emit
	repo.AddChannel(channelID, false)

	_, err := svc.Create(context.Background(), CreateInput{
		ChannelID:      channelID,
		Content:        "guest hi",
		GuestSessionID: &guestSessionID,
	})
	require.NoError(t, err)

	assert.Empty(t, emitter.events, "guest messages have no CreatedBy and must not reach the event emitter")
}
