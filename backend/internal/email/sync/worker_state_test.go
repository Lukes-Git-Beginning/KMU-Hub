package sync

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/kmuhub/kmuhub/internal/models"
)

// newUnconnectedTestWorker returns a Worker with no accountService/msgSyncer/
// folderSyncer wiring, alongside a fresh IMAPClient that was never Connect()ed
// (client.client == nil). Every function exercised in this file hits its
// IMAP-error early-return before ever touching those nil fields -- see the
// nil-client guards proven by TestIMAPClient_NilClientGuards.
func newUnconnectedTestWorker() (*Worker, *IMAPClient) {
	w := &Worker{
		account:   &models.EmailAccount{ID: uuid.New(), TenantID: uuid.New()},
		triggerCh: make(chan struct{}, 1),
		logger:    slog.Default(),
	}
	client := NewIMAPClient(uuid.New(), slog.Default())
	return w, client
}

func TestIdleOrPoll_NoInboxFolder_ReturnsWithoutTouchingClient(t *testing.T) {
	w, client := newUnconnectedTestWorker()
	folders := []*models.EmailFolder{
		{ID: uuid.New(), FolderType: models.FolderTypeSent, IMAPName: "Sent"},
		{ID: uuid.New(), FolderType: models.FolderTypeCustom, IMAPName: "Projects"},
	}

	err := w.idleOrPoll(context.Background(), client, folders)

	// If idleOrPoll had reached client.SelectFolder, the unconnected client
	// would return ErrIMAPConnectionLost -- nil proves it returned before that.
	assert.NoError(t, err)
}

func TestIdleOrPoll_InboxFolder_UnconnectedClient_PropagatesSelectError(t *testing.T) {
	w, client := newUnconnectedTestWorker()
	folders := []*models.EmailFolder{
		{ID: uuid.New(), FolderType: models.FolderTypeInbox, IMAPName: "INBOX"},
	}

	err := w.idleOrPoll(context.Background(), client, folders)

	assert.ErrorIs(t, err, ErrIMAPConnectionLost)
}

func TestSyncFolders_UnconnectedClient_ReturnsListFoldersError(t *testing.T) {
	w, client := newUnconnectedTestWorker()

	err := w.syncFolders(context.Background(), client)

	assert.ErrorIs(t, err, ErrIMAPConnectionLost)
}

func TestSyncFolder_UnconnectedClient_ReturnsSelectFolderError(t *testing.T) {
	w, client := newUnconnectedTestWorker()
	folder := &models.EmailFolder{ID: uuid.New(), IMAPName: "INBOX"}

	err := w.syncFolder(context.Background(), client, folder)

	assert.ErrorIs(t, err, ErrIMAPConnectionLost)
}

func TestWorker_Stop_WithoutCancelFn_DoesNotPanic(t *testing.T) {
	w := &Worker{}
	assert.NotPanics(t, func() { w.Stop() })
}

func TestWorker_Stop_CallsCancelFn(t *testing.T) {
	called := false
	w := &Worker{cancelFn: func() { called = true }}

	w.Stop()

	assert.True(t, called)
}

func TestWorker_Trigger_TwiceDoesNotBlock(t *testing.T) {
	w := &Worker{triggerCh: make(chan struct{}, 1)}

	done := make(chan struct{})
	go func() {
		w.Trigger()
		w.Trigger() // channel already holds a pending signal -- must hit the default case
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("second Trigger() call blocked instead of hitting the default case")
	}

	select {
	case <-w.triggerCh:
	default:
		t.Fatal("expected a pending trigger signal in triggerCh")
	}
}
