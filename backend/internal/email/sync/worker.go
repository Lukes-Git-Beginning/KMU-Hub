package sync

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/database"
	"github.com/kmuhub/kmuhub/internal/email/account"
	"github.com/kmuhub/kmuhub/internal/models"
)

const (
	// IDLE restart interval per RFC 2177 recommendation.
	// go-imap v2 handles IDLE auto-restart internally, but we use this
	// as a context timeout to periodically break out and sync.
	idleRestartInterval = 25 * time.Minute

	// Polling intervals
	inboxPollInterval       = 60 * time.Second
	otherFolderPollInterval = 5 * time.Minute

	// Initial sync depth: fetch last 30 days of headers
	initialSyncDays = 30

	// Reconnection backoff
	initialBackoff = 1 * time.Second
	maxBackoff     = 60 * time.Second
)

// folderTypeMap maps well-known IMAP folder names to folder types.
var folderTypeMap = map[string]string{
	"inbox":               models.FolderTypeInbox,
	"sent":                models.FolderTypeSent,
	"sent items":          models.FolderTypeSent,
	"sent messages":       models.FolderTypeSent,
	"gesendet":            models.FolderTypeSent,
	"gesendete objekte":   models.FolderTypeSent,
	"drafts":              models.FolderTypeDrafts,
	"entwürfe":            models.FolderTypeDrafts,
	"entwuerfe":           models.FolderTypeDrafts,
	"trash":               models.FolderTypeTrash,
	"deleted":             models.FolderTypeTrash,
	"deleted items":       models.FolderTypeTrash,
	"papierkorb":          models.FolderTypeTrash,
	"spam":                models.FolderTypeSpam,
	"junk":                models.FolderTypeSpam,
	"junk-e-mail":         models.FolderTypeSpam,
	"archive":             models.FolderTypeArchive,
	"archiv":              models.FolderTypeArchive,
}

// DetectFolderType determines the folder type from an IMAP folder name.
func DetectFolderType(imapName string) string {
	normalized := strings.ToLower(imapName)
	if ft, ok := folderTypeMap[normalized]; ok {
		return ft
	}
	return models.FolderTypeCustom
}

// Worker runs per-user IMAP sync.
type Worker struct {
	account        *models.EmailAccount
	accountService *account.Service
	msgSyncer      MessageSyncer
	folderSyncer   FolderSyncer
	attachStore    AttachmentStorer
	triggerCh      chan struct{}
	cancelFn       context.CancelFunc
	logger         *slog.Logger
}

// newWorker creates a new sync worker for an account.
func newWorker(
	acct *models.EmailAccount,
	accountService *account.Service,
	msgSyncer MessageSyncer,
	folderSyncer FolderSyncer,
	attachStore AttachmentStorer,
	logger *slog.Logger,
) *Worker {
	return &Worker{
		account:        acct,
		accountService: accountService,
		msgSyncer:      msgSyncer,
		folderSyncer:   folderSyncer,
		attachStore:    attachStore,
		triggerCh:      make(chan struct{}, 1),
		logger:         logger.With("account_id", acct.ID, "email", acct.EmailAddress),
	}
}

// Run executes the sync loop with connection management and reconnection.
func (w *Worker) Run(ctx context.Context) {
	ctx = database.WithSystemContext(ctx)
	backoff := initialBackoff

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("sync worker stopping")
			return
		default:
		}

		err := w.syncCycle(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			w.logger.Error("sync cycle failed, reconnecting",
				"error", err,
				"backoff", backoff,
			)
			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
			}
			// Exponential backoff
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
			continue
		}

		// Reset backoff on success
		backoff = initialBackoff
	}
}

// syncCycle performs one complete sync: connect, sync folders, sync messages, enter IDLE/poll.
func (w *Worker) syncCycle(ctx context.Context) error {
	// Get decrypted credentials — scoped to tenant for isolation
	creds, err := w.accountService.GetDecryptedCredentials(ctx, w.account.ID, w.account.TenantID)
	if err != nil {
		return err
	}

	// Connect
	client := NewIMAPClient(w.account.ID, w.logger)
	if err := client.Connect(creds.IMAPHost, creds.IMAPPort, creds.UseSSL); err != nil {
		return err
	}
	defer client.Close()

	// Login
	if err := client.Login(creds.Username, creds.Password); err != nil {
		return err
	}

	// Sync folder list
	if err := w.syncFolders(ctx, client); err != nil {
		return err
	}

	// Delta sync each folder
	folders, err := w.folderSyncer.ListByAccount(ctx, w.account.ID)
	if err != nil {
		return err
	}

	for _, folder := range folders {
		if err := w.syncFolder(ctx, client, folder); err != nil {
			w.logger.Error("folder sync failed",
				"folder", folder.Name,
				"error", err,
			)
			// Continue with other folders
		}
	}

	// Update last sync timestamp — scoped to tenant for isolation
	if err := w.accountService.UpdateLastSync(ctx, w.account.ID, w.account.TenantID); err != nil {
		w.logger.Warn("failed to update last_sync_at", "error", err)
	}

	// Enter IDLE/poll loop for INBOX
	return w.idleOrPoll(ctx, client, folders)
}

// syncFolders synchronizes the IMAP folder list to the database.
func (w *Worker) syncFolders(ctx context.Context, client *IMAPClient) error {
	imapFolders, err := client.ListFolders()
	if err != nil {
		return err
	}

	sortOrder := 0
	for _, f := range imapFolders {
		existing, _ := w.folderSyncer.GetByIMAPName(ctx, w.account.ID, f.Name)
		if existing != nil {
			continue // folder already tracked
		}

		folderType := DetectFolderType(f.Name)
		// Determine display name: use the last segment of the IMAP name
		displayName := f.Name
		if idx := strings.LastIndexByte(f.Name, byte(f.Delimiter)); idx >= 0 {
			displayName = f.Name[idx+1:]
		}

		folder := &models.EmailFolder{
			ID:         uuid.New(),
			AccountID:  w.account.ID,
			Name:       displayName,
			IMAPName:   f.Name,
			FolderType: folderType,
			SortOrder:  sortOrder,
			CreatedAt:  time.Now().UTC(),
			UpdatedAt:  time.Now().UTC(),
		}

		if err := w.folderSyncer.CreateFolder(ctx, folder); err != nil {
			w.logger.Error("failed to create folder",
				"folder", f.Name,
				"error", err,
			)
		}
		sortOrder++
	}

	w.logger.Info("folder sync complete", "imap_folders", len(imapFolders))
	return nil
}

// syncFolder performs delta sync for a single folder.
func (w *Worker) syncFolder(ctx context.Context, client *IMAPClient, folder *models.EmailFolder) error {
	status, err := client.SelectFolder(folder.IMAPName)
	if err != nil {
		return err
	}

	// Check UIDVALIDITY
	if folder.UIDValidity != 0 && folder.UIDValidity != int64(status.UIDValidity) {
		w.logger.Warn("UIDVALIDITY changed, resetting folder",
			"folder", folder.Name,
			"old_validity", folder.UIDValidity,
			"new_validity", status.UIDValidity,
		)
		// Wipe local messages for this folder
		if err := w.folderSyncer.DeleteMessagesByFolder(ctx, folder.ID); err != nil {
			return err
		}
	}

	// Update stored UIDVALIDITY
	if err := w.folderSyncer.UpdateUIDValidity(ctx, folder.ID, int64(status.UIDValidity)); err != nil {
		return err
	}

	// Get highest UID we already have
	highestUID, err := w.msgSyncer.GetHighestUID(ctx, folder.ID)
	if err != nil {
		return err
	}

	// If empty folder, nothing to sync
	if highestUID == 0 && status.NumMessages == 0 {
		return nil
	}

	// Delta sync: fetch UIDs newer than highest_uid
	// UID 0 means "*" (wildcard/max) in IMAP UID sets
	var uidSet imap.UIDSet
	if highestUID > 0 {
		uidSet.AddRange(imap.UID(highestUID+1), 0)
	} else {
		// First sync: fetch all available
		uidSet.AddRange(1, 0)
	}

	envelopes, err := client.FetchHeaders(uidSet)
	if err != nil {
		return err
	}

	if len(envelopes) == 0 {
		return nil
	}

	// Filter by initial sync depth if first sync
	cutoff := time.Now().AddDate(0, 0, -initialSyncDays)
	synced := 0

	for _, env := range envelopes {
		envDate := parseEnvelopeDate(env.Date)
		if highestUID == 0 && envDate.Before(cutoff) {
			continue // Skip old messages on initial sync
		}

		msg := w.envelopeToMessage(env, folder, envDate)
		if err := w.msgSyncer.CreateSynced(ctx, msg); err != nil {
			w.logger.Error("failed to store synced message",
				"uid", env.UID,
				"error", err,
			)
			continue
		}
		synced++
	}

	w.logger.Info("folder delta sync complete",
		"folder", folder.Name,
		"new_messages", synced,
	)

	return nil
}

// idleOrPoll enters IDLE mode for INBOX or polls if IDLE not supported.
func (w *Worker) idleOrPoll(ctx context.Context, client *IMAPClient, folders []*models.EmailFolder) error {
	// Find INBOX folder
	var inbox *models.EmailFolder
	for _, f := range folders {
		if f.FolderType == models.FolderTypeInbox {
			inbox = f
			break
		}
	}

	if inbox == nil {
		w.logger.Warn("no INBOX folder found, skipping IDLE/poll")
		return nil
	}

	// Select INBOX
	if _, err := client.SelectFolder(inbox.IMAPName); err != nil {
		return err
	}

	if client.HasIDLE() {
		return w.idleLoop(ctx, client, inbox)
	}
	return w.pollLoop(ctx, client, inbox)
}

// idleLoop enters IDLE mode with periodic restart per RFC 2177.
func (w *Worker) idleLoop(ctx context.Context, client *IMAPClient, inbox *models.EmailFolder) error {
	w.logger.Info("entering IDLE mode for INBOX")

	for {
		idleCtx, idleCancel := context.WithTimeout(ctx, idleRestartInterval)

		err := client.Idle(idleCtx)
		idleCancel()

		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Check for trigger
		select {
		case <-w.triggerCh:
			w.logger.Info("manual sync triggered during IDLE")
		default:
		}

		// IDLE returned (timeout or new data) -- sync INBOX
		if err != nil && ctx.Err() == nil {
			return err
		}

		if err := w.syncFolder(ctx, client, inbox); err != nil {
			return err
		}
	}
}

// pollLoop polls for new messages at regular intervals.
func (w *Worker) pollLoop(ctx context.Context, client *IMAPClient, inbox *models.EmailFolder) error {
	w.logger.Info("entering poll mode for INBOX", "interval", inboxPollInterval)

	ticker := time.NewTicker(inboxPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			// NOOP to check for changes
			if err := client.Noop(); err != nil {
				return err
			}
			if err := w.syncFolder(ctx, client, inbox); err != nil {
				return err
			}
		case <-w.triggerCh:
			w.logger.Info("manual sync triggered during poll")
			if err := w.syncFolder(ctx, client, inbox); err != nil {
				return err
			}
		}
	}
}

// Stop signals the worker to stop.
func (w *Worker) Stop() {
	if w.cancelFn != nil {
		w.cancelFn()
	}
}

// Trigger forces an immediate sync cycle.
func (w *Worker) Trigger() {
	select {
	case w.triggerCh <- struct{}{}:
	default:
		// Already triggered
	}
}

// envelopeToMessage converts an IMAP envelope to an EmailMessage model.
func (w *Worker) envelopeToMessage(env *MessageEnvelope, folder *models.EmailFolder, date time.Time) *models.EmailMessage {
	msg := &models.EmailMessage{
		ID:        uuid.New(),
		TenantID:  w.account.TenantID,
		AccountID: w.account.ID,
		FolderID:  folder.ID,
		UID:       int64(env.UID),
		MessageID: env.MessageID,
		InReplyTo: firstInReplyTo(env.InReplyTo),
		Subject:   env.Subject,
		Date:      date,
		SizeBytes: env.Size,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	// Parse from address
	if len(env.From) > 0 {
		msg.FromName = env.From[0].Name
		msg.FromEmail = env.From[0].Addr()
	}

	// Parse to, cc, bcc addresses
	msg.ToAddresses = imapAddressesToModel(env.To)
	msg.CcAddresses = imapAddressesToModel(env.CC)
	msg.BccAddresses = imapAddressesToModel(env.BCC)

	// Store all InReplyTo values as references
	if len(env.InReplyTo) > 0 {
		msg.References = env.InReplyTo
	}

	// Check flags
	for _, flag := range env.Flags {
		switch flag {
		case imap.FlagSeen:
			msg.IsRead = true
		case imap.FlagFlagged:
			msg.IsStarred = true
		case imap.FlagDraft:
			msg.IsDraft = true
		}
	}

	// Generate preview from subject (body fetched on-demand)
	if len(msg.Subject) > 200 {
		msg.Preview = msg.Subject[:200]
	} else {
		msg.Preview = msg.Subject
	}

	return msg
}

// imapAddressesToModel converts IMAP addresses to the model format.
func imapAddressesToModel(addrs []imap.Address) []models.EmailAddress {
	if len(addrs) == 0 {
		return nil
	}
	result := make([]models.EmailAddress, 0, len(addrs))
	for _, a := range addrs {
		result = append(result, models.EmailAddress{
			Name:  a.Name,
			Email: a.Addr(),
		})
	}
	return result
}

// parseEnvelopeDate parses a date string from an IMAP envelope.
func parseEnvelopeDate(dateStr string) time.Time {
	if dateStr == "" {
		return time.Now().UTC()
	}
	t, err := time.Parse("2006-01-02T15:04:05Z07:00", dateStr)
	if err != nil {
		return time.Now().UTC()
	}
	return t
}
