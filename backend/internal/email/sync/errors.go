package sync

import "errors"

var (
	// ErrSyncInProgress is returned when a sync is already running for the account.
	ErrSyncInProgress = errors.New("sync already in progress for this account")

	// ErrIMAPConnectionLost is returned when the IMAP connection drops unexpectedly.
	ErrIMAPConnectionLost = errors.New("IMAP connection lost")

	// ErrUIDValidityChanged is returned when the IMAP server reports a UIDVALIDITY change,
	// requiring a full re-sync of the affected folder.
	ErrUIDValidityChanged = errors.New("UIDVALIDITY changed, folder cache invalidated")
)
