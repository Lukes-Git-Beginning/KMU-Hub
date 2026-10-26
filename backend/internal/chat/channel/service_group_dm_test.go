package channel

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGroupDMKey_IgnoresOrder(t *testing.T) {
	a, b, c := uuid.New(), uuid.New(), uuid.New()
	assert.Equal(t, groupDMKey([]uuid.UUID{a, b, c}), groupDMKey([]uuid.UUID{c, a, b}))
	assert.NotEqual(t, groupDMKey([]uuid.UUID{a, b, c}), groupDMKey([]uuid.UUID{a, b}))
	assert.Len(t, groupDMKey([]uuid.UUID{a, b, c}), 64)
}

func TestService_GetOrCreateGroupDM(t *testing.T) {
	ctx := context.Background()
	newRepo := func() (*MockRepository, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID) {
		repo := NewMockRepository()
		a, b, c, d := uuid.New(), uuid.New(), uuid.New(), uuid.New()
		repo.AddUser(a, "A", "A", "a@example.com")
		repo.AddUser(b, "B", "B", "b@example.com")
		repo.AddUser(c, "C", "C", "c@example.com")
		repo.AddUser(d, "D", "D", "d@example.com")
		return repo, a, b, c, d
	}

	t.Run("same member set returns the same channel, in any order and from any member", func(t *testing.T) {
		repo, a, b, c, _ := newRepo()
		svc := NewService(repo)

		first, err := svc.GetOrCreateGroupDM(ctx, GetOrCreateGroupDMInput{UserID: a, ParticipantIDs: []uuid.UUID{b, c}})
		require.NoError(t, err)
		assert.True(t, first.Created)
		assert.True(t, first.Channel.IsDM)
		assert.True(t, first.Channel.IsPrivate)
		assert.Nil(t, first.Channel.DMUser1)
		assert.Nil(t, first.Channel.DMUser2)
		assert.Equal(t, 3, first.Channel.MemberCount)
		for _, u := range []uuid.UUID{a, b, c} {
			_, memErr := repo.GetMembership(ctx, first.Channel.ID, u)
			require.NoError(t, memErr, "member %s", u)
		}

		// Caller listed again, duplicates and a different order.
		second, err := svc.GetOrCreateGroupDM(ctx, GetOrCreateGroupDMInput{UserID: c, ParticipantIDs: []uuid.UUID{b, a, b, c}})
		require.NoError(t, err)
		assert.False(t, second.Created)
		assert.Equal(t, first.Channel.ID, second.Channel.ID)
	})

	t.Run("a different member set gets a new channel", func(t *testing.T) {
		repo, a, b, c, d := newRepo()
		svc := NewService(repo)

		first, err := svc.GetOrCreateGroupDM(ctx, GetOrCreateGroupDMInput{UserID: a, ParticipantIDs: []uuid.UUID{b, c}})
		require.NoError(t, err)
		other, err := svc.GetOrCreateGroupDM(ctx, GetOrCreateGroupDMInput{UserID: a, ParticipantIDs: []uuid.UUID{b, c, d}})
		require.NoError(t, err)
		assert.True(t, other.Created)
		assert.NotEqual(t, first.Channel.ID, other.Channel.ID)
	})

	t.Run("two people in total is the 1:1 DM", func(t *testing.T) {
		repo, a, b, _, _ := newRepo()
		svc := NewService(repo)

		dm, err := svc.GetOrCreateDM(ctx, GetOrCreateDMInput{UserID: a, OtherUserID: b})
		require.NoError(t, err)
		viaGroup, err := svc.GetOrCreateGroupDM(ctx, GetOrCreateGroupDMInput{UserID: b, ParticipantIDs: []uuid.UUID{a}})
		require.NoError(t, err)
		assert.False(t, viaGroup.Created)
		assert.Equal(t, dm.Channel.ID, viaGroup.Channel.ID)
		assert.NotNil(t, viaGroup.Channel.DMUser1)
	})

	t.Run("name is optional and trimmed", func(t *testing.T) {
		repo, a, b, c, d := newRepo()
		svc := NewService(repo)

		named := "  Projekt Nord  "
		res, err := svc.GetOrCreateGroupDM(ctx, GetOrCreateGroupDMInput{UserID: a, ParticipantIDs: []uuid.UUID{b, c}, Name: &named})
		require.NoError(t, err)
		assert.Equal(t, "Projekt Nord", res.Channel.Name)

		blank := "   "
		res, err = svc.GetOrCreateGroupDM(ctx, GetOrCreateGroupDMInput{UserID: a, ParticipantIDs: []uuid.UUID{b, d}, Name: &blank})
		require.NoError(t, err)
		assert.Equal(t, "DM", res.Channel.Name)
	})

	t.Run("rejected inputs", func(t *testing.T) {
		repo, a, b, c, _ := newRepo()
		svc := NewService(repo)

		_, err := svc.GetOrCreateGroupDM(ctx, GetOrCreateGroupDMInput{UserID: a})
		assert.ErrorIs(t, err, ErrCannotDMSelf)
		_, err = svc.GetOrCreateGroupDM(ctx, GetOrCreateGroupDMInput{UserID: a, ParticipantIDs: []uuid.UUID{a, a}})
		assert.ErrorIs(t, err, ErrCannotDMSelf)

		_, err = svc.GetOrCreateGroupDM(ctx, GetOrCreateGroupDMInput{UserID: a, ParticipantIDs: []uuid.UUID{b, uuid.New()}})
		assert.ErrorIs(t, err, ErrUserNotFound)

		long := strings.Repeat("ä", maxChannelNameLen+1)
		_, err = svc.GetOrCreateGroupDM(ctx, GetOrCreateGroupDMInput{UserID: a, ParticipantIDs: []uuid.UUID{b, c}, Name: &long})
		assert.ErrorIs(t, err, ErrDMNameTooLong)
		exact := strings.Repeat("ä", maxChannelNameLen)
		_, err = svc.GetOrCreateGroupDM(ctx, GetOrCreateGroupDMInput{UserID: a, ParticipantIDs: []uuid.UUID{b, c}, Name: &exact})
		assert.NoError(t, err)

		many := make([]uuid.UUID, maxGroupDMMembers)
		for i := range many {
			many[i] = uuid.New()
			repo.AddUser(many[i], "X", "X", "x@example.com")
		}
		_, err = svc.GetOrCreateGroupDM(ctx, GetOrCreateGroupDMInput{UserID: a, ParticipantIDs: many})
		assert.ErrorIs(t, err, ErrTooManyDMParticipants)
		_, err = svc.GetOrCreateGroupDM(ctx, GetOrCreateGroupDMInput{UserID: a, ParticipantIDs: many[:maxGroupDMMembers-1]})
		assert.NoError(t, err)
	})

	t.Run("losing the create race returns the winner's channel", func(t *testing.T) {
		repo, a, b, c, _ := newRepo()
		repo.groupRace = true
		svc := NewService(repo)

		res, err := svc.GetOrCreateGroupDM(ctx, GetOrCreateGroupDMInput{UserID: a, ParticipantIDs: []uuid.UUID{b, c}})
		require.NoError(t, err)
		assert.False(t, res.Created)
		assert.Equal(t, groupDMKey([]uuid.UUID{a, b, c}), keyOf(repo, res.Channel.ID))
	})
}

func keyOf(repo *MockRepository, channelID uuid.UUID) string {
	for k, id := range repo.groupKeys {
		if id == channelID {
			return k
		}
	}
	return ""
}
