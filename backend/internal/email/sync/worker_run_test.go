package sync

import (
	"context"
	"errors"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/email/account"
	"github.com/kmuhub/kmuhub/internal/models"
)

// fakeAccountRepo is an account.Repository test double. Only GetByID and
// ListAllActive are exercised by the sync package; the rest panic if called
// so an unexpected code path fails loudly instead of silently returning a
// zero value.
type fakeAccountRepo struct {
	getByIDFn      func(ctx context.Context, id, tenantID uuid.UUID) (*models.EmailAccount, error)
	listAllActive  []*models.EmailAccount
	listAllActiveErr error
	updateFn       func(ctx context.Context, acct *models.EmailAccount) error
}

func (f *fakeAccountRepo) Create(ctx context.Context, acct *models.EmailAccount) error {
	panic("not used by sync package tests")
}
func (f *fakeAccountRepo) GetByID(ctx context.Context, id, tenantID uuid.UUID) (*models.EmailAccount, error) {
	return f.getByIDFn(ctx, id, tenantID)
}
func (f *fakeAccountRepo) GetByUserIDAndTenant(ctx context.Context, userID, tenantID uuid.UUID) (*models.EmailAccount, error) {
	panic("not used by sync package tests")
}
func (f *fakeAccountRepo) ListByUserAndTenant(ctx context.Context, userID, tenantID uuid.UUID) ([]*models.EmailAccount, error) {
	panic("not used by sync package tests")
}
func (f *fakeAccountRepo) Update(ctx context.Context, acct *models.EmailAccount) error {
	if f.updateFn != nil {
		return f.updateFn(ctx, acct)
	}
	return nil
}
func (f *fakeAccountRepo) Delete(ctx context.Context, id, tenantID uuid.UUID) error {
	panic("not used by sync package tests")
}
func (f *fakeAccountRepo) SetDefault(ctx context.Context, id, userID, tenantID uuid.UUID) error {
	panic("not used by sync package tests")
}
func (f *fakeAccountRepo) ListActive(ctx context.Context, tenantID uuid.UUID) ([]*models.EmailAccount, error) {
	panic("not used by sync package tests")
}
func (f *fakeAccountRepo) ListAllActive(ctx context.Context) ([]*models.EmailAccount, error) {
	return f.listAllActive, f.listAllActiveErr
}

// fakeVaultEncryptor is an account.VaultEncryptor test double.
type fakeVaultEncryptor struct{}

func (fakeVaultEncryptor) Encrypt(ctx context.Context, plaintext []byte) (string, error) {
	return string(plaintext), nil
}
func (fakeVaultEncryptor) Decrypt(ctx context.Context, encrypted string) ([]byte, error) {
	return []byte(encrypted), nil
}

func TestNewWorker_SetsAllFields(t *testing.T) {
	acct := &models.EmailAccount{ID: uuid.New(), TenantID: uuid.New(), EmailAddress: "a@example.com"}
	svc := account.NewService(&fakeAccountRepo{}, fakeVaultEncryptor{})
	ms := &fakeMessageSyncer{}
	fs := newFakeFolderSyncer()

	w := newWorker(acct, svc, ms, fs, nil, slog.Default())

	assert.Same(t, acct, w.account)
	assert.Same(t, svc, w.accountService)
	assert.Same(t, ms, w.msgSyncer)
	assert.NotNil(t, fs, "folderSyncer wired")
	assert.NotNil(t, w.triggerCh)
	assert.NotNil(t, w.logger)
}

// TestWorker_Run_RetriesAfterSyncCycleFailure proves the sync worker does not
// hang after a failed cycle: it backs off and comes back for another attempt,
// same failure class as the advisory-lock leak from run 10 (a worker that
// dies once and silently never runs again). GetDecryptedCredentials fails
// before any network I/O (repo.GetByID errors), so this needs no real IMAP
// server -- only the retry loop itself is under test.
func TestWorker_Run_RetriesAfterSyncCycleFailure(t *testing.T) {
	origInitial, origMax := initialBackoff, maxBackoff
	initialBackoff, maxBackoff = 20*time.Millisecond, 100*time.Millisecond
	t.Cleanup(func() { initialBackoff, maxBackoff = origInitial, origMax })

	var attempts int32
	repo := &fakeAccountRepo{
		getByIDFn: func(ctx context.Context, id, tenantID uuid.UUID) (*models.EmailAccount, error) {
			atomic.AddInt32(&attempts, 1)
			return nil, errors.New("account lookup failed")
		},
	}
	svc := account.NewService(repo, fakeVaultEncryptor{})
	acct := &models.EmailAccount{ID: uuid.New(), TenantID: uuid.New()}
	w := newWorker(acct, svc, &fakeMessageSyncer{}, newFakeFolderSyncer(), nil, slog.Default())

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		w.Run(ctx)
		close(done)
	}()

	require.Eventually(t, func() bool {
		return atomic.LoadInt32(&attempts) >= 2
	}, 2*time.Second, 5*time.Millisecond, "worker must retry syncCycle after a failure, not stop after the first")

	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return after context cancellation")
	}
}

// TestWorker_Run_StopsOnContextCancelDuringBackoff proves a canceled context
// interrupts the backoff wait itself rather than being checked only at loop
// top -- otherwise Stop() could block for up to maxBackoff.
func TestWorker_Run_StopsOnContextCancelDuringBackoff(t *testing.T) {
	origInitial, origMax := initialBackoff, maxBackoff
	initialBackoff, maxBackoff = 10*time.Second, 10*time.Second
	t.Cleanup(func() { initialBackoff, maxBackoff = origInitial, origMax })

	repo := &fakeAccountRepo{
		getByIDFn: func(ctx context.Context, id, tenantID uuid.UUID) (*models.EmailAccount, error) {
			return nil, errors.New("account lookup failed")
		},
	}
	svc := account.NewService(repo, fakeVaultEncryptor{})
	acct := &models.EmailAccount{ID: uuid.New(), TenantID: uuid.New()}
	w := newWorker(acct, svc, &fakeMessageSyncer{}, newFakeFolderSyncer(), nil, slog.Default())

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		w.Run(ctx)
		close(done)
	}()

	time.Sleep(30 * time.Millisecond) // let the first failed cycle enter its 10s backoff wait
	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Run kept blocking in backoff instead of returning on ctx cancellation")
	}
}

func TestWorker_Run_ReturnsImmediatelyOnAlreadyCanceledContext(t *testing.T) {
	svc := account.NewService(&fakeAccountRepo{}, fakeVaultEncryptor{})
	acct := &models.EmailAccount{ID: uuid.New(), TenantID: uuid.New()}
	w := newWorker(acct, svc, &fakeMessageSyncer{}, newFakeFolderSyncer(), nil, slog.Default())

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	done := make(chan struct{})
	go func() {
		w.Run(ctx)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("Run did not return immediately for an already-canceled context")
	}
}
