package thread

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeRepo records calls and serves canned data; it is the "edge" of the
// service, the logic under test stays real.
type fakeRepo struct {
	seedCalls   int
	seedArgs    []any
	seedErr     error
	listed      []*ThreadMessage
	listErr     error
	appended    []*ThreadMessage
	appendErr   error
	canned      map[uuid.UUID]*CannedResponse
	createErr   error
	updateErr   error
	updatedWith *CannedResponse
	deletedID   uuid.UUID
	deleteErr   error
}

func (f *fakeRepo) AppendThreadMessage(_ context.Context, m *ThreadMessage) error {
	f.appended = append(f.appended, m)
	return f.appendErr
}

func (f *fakeRepo) SeedInboundIfEmpty(_ context.Context, tenantID, messageID uuid.UUID, senderName, body string, createdAt time.Time) error {
	f.seedCalls++
	f.seedArgs = []any{tenantID, messageID, senderName, body, createdAt}
	return f.seedErr
}

func (f *fakeRepo) ListThreadMessages(context.Context, uuid.UUID, uuid.UUID) ([]*ThreadMessage, error) {
	return f.listed, f.listErr
}

func (f *fakeRepo) CreateCannedResponse(_ context.Context, c *CannedResponse) error {
	if f.createErr != nil {
		return f.createErr
	}
	c.ID = uuid.New()
	return nil
}

func (f *fakeRepo) ListCannedResponses(context.Context, uuid.UUID) ([]*CannedResponse, error) {
	out := make([]*CannedResponse, 0, len(f.canned))
	for _, c := range f.canned {
		out = append(out, c)
	}
	return out, nil
}

func (f *fakeRepo) GetCannedResponse(_ context.Context, _ uuid.UUID, id uuid.UUID) (*CannedResponse, error) {
	c, ok := f.canned[id]
	if !ok {
		return nil, ErrCannedResponseNotFound
	}
	cp := *c
	return &cp, nil
}

func (f *fakeRepo) UpdateCannedResponse(_ context.Context, c *CannedResponse) error {
	f.updatedWith = c
	return f.updateErr
}

func (f *fakeRepo) DeleteCannedResponse(_ context.Context, _ uuid.UUID, id uuid.UUID) error {
	f.deletedID = id
	return f.deleteErr
}

func TestListThread_SeedsInboundThenLists(t *testing.T) {
	repo := &fakeRepo{listed: []*ThreadMessage{{Body: "first"}}}
	svc := NewService(repo)
	tenantID, messageID := uuid.New(), uuid.New()
	at := time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)

	got, err := svc.ListThread(context.Background(), tenantID, messageID, InboundSeed{SenderName: "Erika", Body: "Hallo", CreatedAt: at})

	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, 1, repo.seedCalls)
	assert.Equal(t, []any{tenantID, messageID, "Erika", "Hallo", at}, repo.seedArgs)
}

func TestListThread_EmptySeedBodySkipsSeeding(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo)

	_, err := svc.ListThread(context.Background(), uuid.New(), uuid.New(), InboundSeed{SenderName: "Erika"})

	require.NoError(t, err)
	assert.Zero(t, repo.seedCalls, "an empty preview must not create an empty inbound bubble")
}

func TestListThread_SeedErrorAbortsBeforeListing(t *testing.T) {
	boom := errors.New("seed failed")
	repo := &fakeRepo{seedErr: boom, listed: []*ThreadMessage{{Body: "must not be returned"}}}

	got, err := NewService(repo).ListThread(context.Background(), uuid.New(), uuid.New(), InboundSeed{Body: "x"})

	assert.ErrorIs(t, err, boom)
	assert.Nil(t, got)
}

func TestListThread_ListErrorPropagates(t *testing.T) {
	boom := errors.New("list failed")
	repo := &fakeRepo{listErr: boom}

	_, err := NewService(repo).ListThread(context.Background(), uuid.New(), uuid.New(), InboundSeed{})

	assert.ErrorIs(t, err, boom)
}

func TestAppendReply_WritesOutboundRowWithAuthor(t *testing.T) {
	repo := &fakeRepo{}
	tenantID, messageID, authorID := uuid.New(), uuid.New(), uuid.New()

	require.NoError(t, NewService(repo).AppendReply(context.Background(), tenantID, messageID, authorID, "Antwort"))

	require.Len(t, repo.appended, 1)
	m := repo.appended[0]
	assert.Equal(t, tenantID, m.TenantID)
	assert.Equal(t, messageID, m.MessageID)
	require.NotNil(t, m.AuthorID)
	assert.Equal(t, authorID, *m.AuthorID)
	assert.Equal(t, DirectionOutbound, m.Direction)
	assert.Equal(t, "Antwort", m.Body)
}

func TestAppendReply_RepoErrorPropagates(t *testing.T) {
	boom := errors.New("insert failed")

	err := NewService(&fakeRepo{appendErr: boom}).AppendReply(context.Background(), uuid.New(), uuid.New(), uuid.New(), "x")

	assert.ErrorIs(t, err, boom)
}

func TestCreateCanned_ReturnsStoredEntry(t *testing.T) {
	tenantID := uuid.New()

	c, err := NewService(&fakeRepo{}).CreateCanned(context.Background(), tenantID, "Gruss", "Hallo")

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, c.ID)
	assert.Equal(t, tenantID, c.TenantID)
	assert.Equal(t, "Gruss", c.Name)
	assert.Equal(t, "Hallo", c.Body)
}

func TestCreateCanned_RepoErrorYieldsNoEntry(t *testing.T) {
	boom := errors.New("insert failed")

	c, err := NewService(&fakeRepo{createErr: boom}).CreateCanned(context.Background(), uuid.New(), "a", "b")

	assert.ErrorIs(t, err, boom)
	assert.Nil(t, c)
}

func TestListCanned_DelegatesToRepo(t *testing.T) {
	id := uuid.New()
	repo := &fakeRepo{canned: map[uuid.UUID]*CannedResponse{id: {ID: id, Name: "a"}}}

	got, err := NewService(repo).ListCanned(context.Background(), uuid.New())

	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, id, got[0].ID)
}

func TestUpdateCanned_PartialUpdateKeepsOtherField(t *testing.T) {
	id := uuid.New()
	repo := &fakeRepo{canned: map[uuid.UUID]*CannedResponse{id: {ID: id, Name: "alt", Body: "body-alt"}}}
	svc := NewService(repo)
	newName, newBody := "neu", "body-neu"

	got, err := svc.UpdateCanned(context.Background(), uuid.New(), id, &newName, nil)
	require.NoError(t, err)
	assert.Equal(t, "neu", got.Name)
	assert.Equal(t, "body-alt", got.Body, "nil body keeps the stored body")

	got, err = svc.UpdateCanned(context.Background(), uuid.New(), id, nil, &newBody)
	require.NoError(t, err)
	assert.Equal(t, "alt", got.Name, "nil name keeps the stored name")
	assert.Equal(t, "body-neu", got.Body)
	assert.Same(t, got, repo.updatedWith)
}

func TestUpdateCanned_NotFoundSkipsWrite(t *testing.T) {
	repo := &fakeRepo{canned: map[uuid.UUID]*CannedResponse{}}
	name := "x"

	got, err := NewService(repo).UpdateCanned(context.Background(), uuid.New(), uuid.New(), &name, nil)

	assert.ErrorIs(t, err, ErrCannedResponseNotFound)
	assert.Nil(t, got)
	assert.Nil(t, repo.updatedWith, "a missing entry must not reach UpdateCannedResponse")
}

func TestUpdateCanned_WriteErrorYieldsNoEntry(t *testing.T) {
	id := uuid.New()
	boom := errors.New("update failed")
	repo := &fakeRepo{canned: map[uuid.UUID]*CannedResponse{id: {ID: id}}, updateErr: boom}
	name := "x"

	got, err := NewService(repo).UpdateCanned(context.Background(), uuid.New(), id, &name, nil)

	assert.ErrorIs(t, err, boom)
	assert.Nil(t, got)
}

func TestDeleteCanned_DelegatesAndPropagates(t *testing.T) {
	id := uuid.New()
	repo := &fakeRepo{deleteErr: ErrCannedResponseNotFound}

	err := NewService(repo).DeleteCanned(context.Background(), uuid.New(), id)

	assert.ErrorIs(t, err, ErrCannedResponseNotFound)
	assert.Equal(t, id, repo.deletedID)
}
