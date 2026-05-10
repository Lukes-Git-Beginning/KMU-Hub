package sync

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/database"
	"github.com/kmuhub/kmuhub/internal/email/account"
	"github.com/kmuhub/kmuhub/internal/models"
)

// MessageSyncer defines the interface for storing synced messages.
// Separated from message.Service to avoid circular imports.
type MessageSyncer interface {
	CreateSynced(ctx context.Context, msg *models.EmailMessage) error
	GetHighestUID(ctx context.Context, folderID uuid.UUID) (uint32, error)
}

// FolderSyncer defines the interface for managing email folders during sync.
type FolderSyncer interface {
	GetByIMAPName(ctx context.Context, accountID uuid.UUID, imapName string) (*models.EmailFolder, error)
	CreateFolder(ctx context.Context, folder *models.EmailFolder) error
	UpdateUIDValidity(ctx context.Context, folderID uuid.UUID, uidValidity int64) error
	DeleteMessagesByFolder(ctx context.Context, folderID uuid.UUID) error
	ListByAccount(ctx context.Context, accountID uuid.UUID) ([]*models.EmailFolder, error)
	UpdateCounts(ctx context.Context, folderID uuid.UUID, messageCount, unreadCount int) error
}

// AttachmentStorer defines the interface for storing email attachments during sync.
type AttachmentStorer interface {
	CreateFromStream(ctx context.Context, messageID uuid.UUID, accountID uuid.UUID, messageUID uint32, filename, contentType string, size int64, reader interface{}, contentID string, isInline bool, tenantID uuid.UUID) (*models.EmailAttachment, error)
}

// SyncStatus holds the current sync status for an account.
type SyncStatus struct {
	AccountID uuid.UUID  `json:"account_id"`
	IsSyncing bool       `json:"is_syncing"`
	LastSync  *time.Time `json:"last_sync,omitempty"`
	Error     string     `json:"error,omitempty"`
}

// Engine manages per-user IMAP sync workers.
type Engine struct {
	accountService *account.Service
	messageSyncer  MessageSyncer
	folderSyncer   FolderSyncer
	attachStore    AttachmentStorer
	workers        map[uuid.UUID]*Worker
	statuses       map[uuid.UUID]*SyncStatus
	mu             sync.RWMutex
	logger         *slog.Logger
	cancelFn       context.CancelFunc
}

// NewEngine creates a new sync engine.
func NewEngine(
	accountService *account.Service,
	messageSyncer MessageSyncer,
	folderSyncer FolderSyncer,
	attachStore AttachmentStorer,
) *Engine {
	return &Engine{
		accountService: accountService,
		messageSyncer:  messageSyncer,
		folderSyncer:   folderSyncer,
		attachStore:    attachStore,
		workers:        make(map[uuid.UUID]*Worker),
		statuses:       make(map[uuid.UUID]*SyncStatus),
		logger:         slog.Default().With("component", "sync-engine"),
	}
}

// Start loads all active accounts and starts a worker goroutine per account.
func (e *Engine) Start(ctx context.Context) error {
	ctx = database.WithSystemContext(ctx)
	ctx, cancel := context.WithCancel(ctx)
	e.cancelFn = cancel

	accounts, err := e.accountService.ListAllActive(ctx)
	if err != nil {
		cancel()
		return err
	}

	e.logger.Info("starting sync engine", "active_accounts", len(accounts))

	for _, acct := range accounts {
		e.startWorkerInternal(ctx, acct)
	}

	return nil
}

// Stop gracefully stops all workers.
func (e *Engine) Stop() {
	if e.cancelFn != nil {
		e.cancelFn()
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	for id, w := range e.workers {
		w.Stop()
		delete(e.workers, id)
		e.logger.Info("stopped sync worker", "account_id", id)
	}

	e.logger.Info("sync engine stopped")
}

// StartWorker starts sync for a specific account by ID.
func (e *Engine) StartWorker(ctx context.Context, accountID uuid.UUID) error {
	// Get full account data from service (includes encrypted credentials and tenant_id).
	accounts, err := e.accountService.ListAllActive(ctx)
	if err != nil {
		return err
	}

	var targetAccount *models.EmailAccount
	for _, a := range accounts {
		if a.ID == accountID {
			targetAccount = a
			break
		}
	}

	if targetAccount == nil {
		return ErrSyncInProgress // account not found or not active
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	// Stop existing worker if any
	if existing, ok := e.workers[accountID]; ok {
		existing.Stop()
	}

	workerCtx, cancel := context.WithCancel(ctx)
	w := newWorker(targetAccount, e.accountService, e.messageSyncer, e.folderSyncer, e.attachStore, e.logger)
	w.cancelFn = cancel
	e.workers[accountID] = w
	e.statuses[accountID] = &SyncStatus{AccountID: accountID}

	go w.Run(workerCtx)

	e.logger.Info("started sync worker", "account_id", accountID)
	return nil
}

// StopWorker stops sync for a specific account.
func (e *Engine) StopWorker(accountID uuid.UUID) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if w, ok := e.workers[accountID]; ok {
		w.Stop()
		delete(e.workers, accountID)
		e.logger.Info("stopped sync worker", "account_id", accountID)
	}
}

// TriggerSync forces an immediate sync cycle for an account.
func (e *Engine) TriggerSync(accountID uuid.UUID) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if w, ok := e.workers[accountID]; ok {
		w.Trigger()
	}
}

// GetStatus returns the current sync status for an account.
func (e *Engine) GetStatus(accountID uuid.UUID) *SyncStatus {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if status, ok := e.statuses[accountID]; ok {
		return status
	}
	return &SyncStatus{AccountID: accountID}
}

// startWorkerInternal starts a worker without acquiring the lock (called from Start).
func (e *Engine) startWorkerInternal(ctx context.Context, acct *models.EmailAccount) {
	e.mu.Lock()
	defer e.mu.Unlock()

	workerCtx, cancel := context.WithCancel(ctx)
	w := newWorker(acct, e.accountService, e.messageSyncer, e.folderSyncer, e.attachStore, e.logger)
	w.cancelFn = cancel
	e.workers[acct.ID] = w
	e.statuses[acct.ID] = &SyncStatus{AccountID: acct.ID}

	go w.Run(workerCtx)
}
