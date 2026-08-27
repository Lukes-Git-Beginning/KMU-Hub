package sync

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/email/account"
	"github.com/kmuhub/kmuhub/internal/models"
)

// TestMain shrinks the worker reconnect backoff once, for the whole package,
// so the background goroutines Start/StartWorker spawn don't hold up test
// cleanup.
//
// It deliberately does NOT restore the values afterwards, and it is a TestMain
// rather than a per-test helper: those goroutines outlive the test that started
// them (StartWorker replaces a worker without waiting for the old one to
// return), so a t.Cleanup that reassigns these package-level vars races against
// a Worker.Run still reading initialBackoff -- exactly what the race detector
// caught in CI run 33071992258. Writing them once before any test starts has no
// concurrent reader.
func TestMain(m *testing.M) {
	initialBackoff, maxBackoff = 5*time.Millisecond, 20*time.Millisecond
	os.Exit(m.Run())
}

// failFastAccountRepo makes every worker's syncCycle fail before any network
// I/O -- GetByID errors immediately -- so Start/StartWorker's spawned
// goroutines never attempt a real IMAP connection.
func failFastAccountRepo(active []*models.EmailAccount) *fakeAccountRepo {
	return &fakeAccountRepo{
		listAllActive: active,
		getByIDFn: func(ctx context.Context, id, tenantID uuid.UUID) (*models.EmailAccount, error) {
			return nil, errors.New("no network in tests")
		},
	}
}

func TestEngine_Start_ListAllActiveErrorPropagatesWithoutStartingWorkers(t *testing.T) {
	repo := &fakeAccountRepo{listAllActiveErr: errors.New("db unavailable")}
	e := NewEngine(account.NewService(repo, fakeVaultEncryptor{}), &fakeMessageSyncer{}, newFakeFolderSyncer(), nil)

	err := e.Start(context.Background())

	assert.EqualError(t, err, "db unavailable")
	assert.Empty(t, e.workers)
}

func TestEngine_Start_SpawnsOneWorkerPerActiveAccount(t *testing.T) {
	acct1 := &models.EmailAccount{ID: uuid.New(), TenantID: uuid.New()}
	acct2 := &models.EmailAccount{ID: uuid.New(), TenantID: uuid.New()}
	repo := failFastAccountRepo([]*models.EmailAccount{acct1, acct2})
	e := NewEngine(account.NewService(repo, fakeVaultEncryptor{}), &fakeMessageSyncer{}, newFakeFolderSyncer(), nil)

	err := e.Start(context.Background())
	t.Cleanup(e.Stop)

	require.NoError(t, err)
	assert.Len(t, e.workers, 2)
	assert.Contains(t, e.workers, acct1.ID)
	assert.Contains(t, e.workers, acct2.ID)
	assert.Contains(t, e.statuses, acct1.ID)
	assert.Contains(t, e.statuses, acct2.ID)
}

func TestEngine_StartWorker_AccountNotFound_ReturnsErrSyncInProgress(t *testing.T) {
	repo := failFastAccountRepo(nil)
	e := NewEngine(account.NewService(repo, fakeVaultEncryptor{}), &fakeMessageSyncer{}, newFakeFolderSyncer(), nil)

	err := e.StartWorker(context.Background(), uuid.New())

	assert.ErrorIs(t, err, ErrSyncInProgress)
	assert.Empty(t, e.workers)
}

func TestEngine_StartWorker_ReplacesExistingWorkerForSameAccount(t *testing.T) {
	acctID := uuid.New()
	acct := &models.EmailAccount{ID: acctID, TenantID: uuid.New()}
	repo := failFastAccountRepo([]*models.EmailAccount{acct})
	e := NewEngine(account.NewService(repo, fakeVaultEncryptor{}), &fakeMessageSyncer{}, newFakeFolderSyncer(), nil)

	var oldStopped bool
	e.workers[acctID] = &Worker{cancelFn: func() { oldStopped = true }}

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	err := e.StartWorker(ctx, acctID)

	require.NoError(t, err)
	assert.True(t, oldStopped, "the pre-existing worker for this account must be stopped")
	assert.Contains(t, e.workers, acctID)
	assert.NotNil(t, e.statuses[acctID])
}

func TestEngine_StartWorker_UnknownAccountLeavesWorkersUntouched(t *testing.T) {
	knownID := uuid.New()
	repo := failFastAccountRepo([]*models.EmailAccount{{ID: knownID, TenantID: uuid.New()}})
	e := NewEngine(account.NewService(repo, fakeVaultEncryptor{}), &fakeMessageSyncer{}, newFakeFolderSyncer(), nil)

	err := e.StartWorker(context.Background(), uuid.New())

	assert.ErrorIs(t, err, ErrSyncInProgress)
	assert.Empty(t, e.workers)
}
