package sync

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
)

// fakeFolderLister is a folderLister test double.
type fakeFolderLister struct {
	folders []*FolderInfo
	err     error
}

func (f *fakeFolderLister) ListFolders() ([]*FolderInfo, error) { return f.folders, f.err }

// fakeFolderFetcher is a folderFetcher test double.
type fakeFolderFetcher struct {
	status    *FolderStatus
	selectErr error
	envelopes []*MessageEnvelope
	fetchErr  error
	fetchedUIDSets []imap.UIDSet
}

func (f *fakeFolderFetcher) SelectFolder(name string) (*FolderStatus, error) {
	return f.status, f.selectErr
}

func (f *fakeFolderFetcher) FetchHeaders(uidSet imap.UIDSet) ([]*MessageEnvelope, error) {
	f.fetchedUIDSets = append(f.fetchedUIDSets, uidSet)
	return f.envelopes, f.fetchErr
}

// fakeFolderSyncer is a FolderSyncer test double.
type fakeFolderSyncer struct {
	byIMAPName          map[string]*models.EmailFolder
	createErr           error
	created             []*models.EmailFolder
	updateUIDValidityErr error
	updatedUIDValidity   []int64
	deleteMessagesErr    error
	deletedFolders       []uuid.UUID
}

func newFakeFolderSyncer() *fakeFolderSyncer {
	return &fakeFolderSyncer{byIMAPName: map[string]*models.EmailFolder{}}
}

func (f *fakeFolderSyncer) GetByIMAPName(ctx context.Context, accountID uuid.UUID, imapName string) (*models.EmailFolder, error) {
	return f.byIMAPName[imapName], nil
}

func (f *fakeFolderSyncer) CreateFolder(ctx context.Context, folder *models.EmailFolder) error {
	if f.createErr != nil {
		return f.createErr
	}
	f.created = append(f.created, folder)
	return nil
}

func (f *fakeFolderSyncer) UpdateUIDValidity(ctx context.Context, folderID uuid.UUID, uidValidity int64) error {
	f.updatedUIDValidity = append(f.updatedUIDValidity, uidValidity)
	return f.updateUIDValidityErr
}

func (f *fakeFolderSyncer) DeleteMessagesByFolder(ctx context.Context, folderID uuid.UUID) error {
	f.deletedFolders = append(f.deletedFolders, folderID)
	return f.deleteMessagesErr
}

func (f *fakeFolderSyncer) ListByAccount(ctx context.Context, accountID uuid.UUID) ([]*models.EmailFolder, error) {
	return nil, nil
}

func (f *fakeFolderSyncer) UpdateCounts(ctx context.Context, folderID uuid.UUID, messageCount, unreadCount int) error {
	return nil
}

// fakeMessageSyncer is a MessageSyncer test double.
type fakeMessageSyncer struct {
	highestUID     uint32
	highestUIDErr  error
	createErrByUID map[int64]error
	created        []*models.EmailMessage
}

func (f *fakeMessageSyncer) CreateSynced(ctx context.Context, msg *models.EmailMessage) error {
	if err, ok := f.createErrByUID[msg.UID]; ok {
		return err
	}
	f.created = append(f.created, msg)
	return nil
}

func (f *fakeMessageSyncer) GetHighestUID(ctx context.Context, folderID uuid.UUID) (uint32, error) {
	return f.highestUID, f.highestUIDErr
}

func workerWithSyncers(folderSyncer FolderSyncer, msgSyncer MessageSyncer) *Worker {
	return &Worker{
		account:      &models.EmailAccount{ID: uuid.New(), TenantID: uuid.New()},
		folderSyncer: folderSyncer,
		msgSyncer:    msgSyncer,
		logger:       slog.Default(),
	}
}

func envelope(uid uint32, subject string, date time.Time) *MessageEnvelope {
	return &MessageEnvelope{
		UID:     uid,
		Subject: subject,
		Date:    date.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// --- syncFolders ---

func TestSyncFolders_CreatesNewFoldersWithDetectedTypeAndDisplayName(t *testing.T) {
	fs := newFakeFolderSyncer()
	w := workerWithSyncers(fs, nil)
	client := &fakeFolderLister{folders: []*FolderInfo{
		{Name: "INBOX", Delimiter: '/'},
		{Name: "Projects/2026", Delimiter: '/'},
	}}

	err := w.syncFolders(context.Background(), client)

	require.NoError(t, err)
	require.Len(t, fs.created, 2)
	assert.Equal(t, "INBOX", fs.created[0].Name)
	assert.Equal(t, models.FolderTypeInbox, fs.created[0].FolderType)
	assert.Equal(t, "2026", fs.created[1].Name, "display name is the last delimiter segment")
	assert.Equal(t, models.FolderTypeCustom, fs.created[1].FolderType)
}

func TestSyncFolders_AlreadyTrackedFolderIsSkipped(t *testing.T) {
	fs := newFakeFolderSyncer()
	fs.byIMAPName["INBOX"] = &models.EmailFolder{IMAPName: "INBOX"}
	w := workerWithSyncers(fs, nil)
	client := &fakeFolderLister{folders: []*FolderInfo{{Name: "INBOX", Delimiter: '/'}}}

	err := w.syncFolders(context.Background(), client)

	require.NoError(t, err)
	assert.Empty(t, fs.created, "already-tracked folder must not be recreated")
}

func TestSyncFolders_CreateFolderErrorIsLoggedAndOthersStillProcessed(t *testing.T) {
	fs := newFakeFolderSyncer()
	fs.createErr = errors.New("insert failed")
	w := workerWithSyncers(fs, nil)
	client := &fakeFolderLister{folders: []*FolderInfo{
		{Name: "INBOX", Delimiter: '/'},
		{Name: "Sent", Delimiter: '/'},
	}}

	err := w.syncFolders(context.Background(), client)

	assert.NoError(t, err, "a single failed CreateFolder must not abort the whole sync")
	assert.Empty(t, fs.created)
}

func TestSyncFolders_ListFoldersErrorPropagates(t *testing.T) {
	w := workerWithSyncers(newFakeFolderSyncer(), nil)
	client := &fakeFolderLister{err: ErrIMAPConnectionLost}

	err := w.syncFolders(context.Background(), client)

	assert.ErrorIs(t, err, ErrIMAPConnectionLost)
}

// --- syncFolder ---

func TestSyncFolder_EmptyFolderReturnsWithoutFetching(t *testing.T) {
	ms := &fakeMessageSyncer{highestUID: 0}
	w := workerWithSyncers(newFakeFolderSyncer(), ms)
	client := &fakeFolderFetcher{status: &FolderStatus{NumMessages: 0, UIDValidity: 100}}
	folder := &models.EmailFolder{ID: uuid.New(), Name: "INBOX", IMAPName: "INBOX"}

	err := w.syncFolder(context.Background(), client, folder)

	require.NoError(t, err)
	assert.Empty(t, client.fetchedUIDSets, "an empty folder must not trigger a FetchHeaders call")
}

func TestSyncFolder_UIDValidityChanged_WipesLocalMessagesBeforeResync(t *testing.T) {
	fs := newFakeFolderSyncer()
	ms := &fakeMessageSyncer{highestUID: 5}
	w := workerWithSyncers(fs, ms)
	client := &fakeFolderFetcher{status: &FolderStatus{NumMessages: 1, UIDValidity: 200}}
	folder := &models.EmailFolder{ID: uuid.New(), Name: "INBOX", IMAPName: "INBOX", UIDValidity: 100}

	err := w.syncFolder(context.Background(), client, folder)

	require.NoError(t, err)
	require.Len(t, fs.deletedFolders, 1)
	assert.Equal(t, folder.ID, fs.deletedFolders[0])
	require.Len(t, fs.updatedUIDValidity, 1)
	assert.Equal(t, int64(200), fs.updatedUIDValidity[0])
}

func TestSyncFolder_UIDValidityUnchangedOnFirstSync_DoesNotWipe(t *testing.T) {
	fs := newFakeFolderSyncer()
	ms := &fakeMessageSyncer{highestUID: 0}
	w := workerWithSyncers(fs, ms)
	client := &fakeFolderFetcher{status: &FolderStatus{NumMessages: 1, UIDValidity: 200}}
	folder := &models.EmailFolder{ID: uuid.New(), Name: "INBOX", IMAPName: "INBOX", UIDValidity: 0}

	err := w.syncFolder(context.Background(), client, folder)

	require.NoError(t, err)
	assert.Empty(t, fs.deletedFolders, "UIDValidity 0 means never synced -- not a change")
}

func TestSyncFolder_InitialSync_FiltersMessagesOlderThanCutoff(t *testing.T) {
	ms := &fakeMessageSyncer{highestUID: 0}
	w := workerWithSyncers(newFakeFolderSyncer(), ms)
	old := envelope(1, "old", time.Now().AddDate(0, 0, -60))
	recent := envelope(2, "recent", time.Now().AddDate(0, 0, -1))
	client := &fakeFolderFetcher{
		status:    &FolderStatus{NumMessages: 2, UIDValidity: 1},
		envelopes: []*MessageEnvelope{old, recent},
	}
	folder := &models.EmailFolder{ID: uuid.New(), Name: "INBOX", IMAPName: "INBOX"}

	err := w.syncFolder(context.Background(), client, folder)

	require.NoError(t, err)
	require.Len(t, ms.created, 1, "only the message inside the initial sync depth must be stored")
	assert.Equal(t, "recent", ms.created[0].Subject)
}

func TestSyncFolder_DeltaSync_DoesNotFilterByCutoff(t *testing.T) {
	ms := &fakeMessageSyncer{highestUID: 5}
	w := workerWithSyncers(newFakeFolderSyncer(), ms)
	old := envelope(6, "old but new to us", time.Now().AddDate(0, 0, -60))
	client := &fakeFolderFetcher{
		status:    &FolderStatus{NumMessages: 6, UIDValidity: 1},
		envelopes: []*MessageEnvelope{old},
	}
	folder := &models.EmailFolder{ID: uuid.New(), Name: "INBOX", IMAPName: "INBOX"}

	err := w.syncFolder(context.Background(), client, folder)

	require.NoError(t, err)
	require.Len(t, ms.created, 1, "delta sync (highestUID > 0) must not apply the initial-sync cutoff")
}

func TestSyncFolder_PartialBatchFailure_ContinuesWithRemainingMessages(t *testing.T) {
	ms := &fakeMessageSyncer{
		highestUID:     0,
		createErrByUID: map[int64]error{2: errors.New("db write failed mid-batch")},
	}
	w := workerWithSyncers(newFakeFolderSyncer(), ms)
	now := time.Now()
	client := &fakeFolderFetcher{
		status: &FolderStatus{NumMessages: 3, UIDValidity: 1},
		envelopes: []*MessageEnvelope{
			envelope(1, "first", now),
			envelope(2, "fails to store", now),
			envelope(3, "third", now),
		},
	}
	folder := &models.EmailFolder{ID: uuid.New(), Name: "INBOX", IMAPName: "INBOX"}

	err := w.syncFolder(context.Background(), client, folder)

	require.NoError(t, err, "one message failing to store must not abort the whole batch")
	require.Len(t, ms.created, 2, "messages 1 and 3 must still be stored despite message 2 failing")
	assert.Equal(t, "first", ms.created[0].Subject)
	assert.Equal(t, "third", ms.created[1].Subject)
}

func TestSyncFolder_GetHighestUIDErrorPropagates(t *testing.T) {
	ms := &fakeMessageSyncer{highestUIDErr: errors.New("db unavailable")}
	w := workerWithSyncers(newFakeFolderSyncer(), ms)
	client := &fakeFolderFetcher{status: &FolderStatus{NumMessages: 1, UIDValidity: 1}}
	folder := &models.EmailFolder{ID: uuid.New(), Name: "INBOX", IMAPName: "INBOX"}

	err := w.syncFolder(context.Background(), client, folder)

	assert.EqualError(t, err, "db unavailable")
}

func TestSyncFolder_UpdateUIDValidityErrorPropagates(t *testing.T) {
	fs := newFakeFolderSyncer()
	fs.updateUIDValidityErr = errors.New("write failed")
	w := workerWithSyncers(fs, &fakeMessageSyncer{})
	client := &fakeFolderFetcher{status: &FolderStatus{NumMessages: 1, UIDValidity: 1}}
	folder := &models.EmailFolder{ID: uuid.New(), Name: "INBOX", IMAPName: "INBOX"}

	err := w.syncFolder(context.Background(), client, folder)

	assert.EqualError(t, err, "write failed")
}

func TestSyncFolder_FetchHeadersErrorPropagates(t *testing.T) {
	ms := &fakeMessageSyncer{highestUID: 0}
	w := workerWithSyncers(newFakeFolderSyncer(), ms)
	client := &fakeFolderFetcher{
		status:   &FolderStatus{NumMessages: 1, UIDValidity: 1},
		fetchErr: errors.New("fetch failed"),
	}
	folder := &models.EmailFolder{ID: uuid.New(), Name: "INBOX", IMAPName: "INBOX"}

	err := w.syncFolder(context.Background(), client, folder)

	assert.EqualError(t, err, "fetch failed")
}

func TestSyncFolder_SelectFolderErrorPropagates(t *testing.T) {
	w := workerWithSyncers(newFakeFolderSyncer(), &fakeMessageSyncer{})
	client := &fakeFolderFetcher{selectErr: ErrIMAPConnectionLost}
	folder := &models.EmailFolder{ID: uuid.New(), Name: "INBOX", IMAPName: "INBOX"}

	err := w.syncFolder(context.Background(), client, folder)

	assert.ErrorIs(t, err, ErrIMAPConnectionLost)
}

func TestSyncFolder_NoNewEnvelopes_ReturnsWithoutStoring(t *testing.T) {
	ms := &fakeMessageSyncer{highestUID: 5}
	w := workerWithSyncers(newFakeFolderSyncer(), ms)
	client := &fakeFolderFetcher{
		status:    &FolderStatus{NumMessages: 5, UIDValidity: 1},
		envelopes: nil,
	}
	folder := &models.EmailFolder{ID: uuid.New(), Name: "INBOX", IMAPName: "INBOX"}

	err := w.syncFolder(context.Background(), client, folder)

	require.NoError(t, err)
	assert.Empty(t, ms.created)
}
