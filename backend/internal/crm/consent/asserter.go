package consent

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
)

// Channel identifies the communication channel for a consent check.
type Channel string

const (
	// ChannelEmail represents the email marketing consent type.
	ChannelEmail Channel = "marketing_email"
	// ChannelPhone represents the phone marketing consent type.
	ChannelPhone Channel = "marketing_phone"
)

// AssertRepo is the minimal read-only interface required by Asserter.
// It is a subset of Repository intentionally kept narrow for easy mocking.
type AssertRepo interface {
	// ActiveConsent returns true when the contact has a non-revoked, granted
	// consent record for the given channel (consent_type).
	ActiveConsent(ctx context.Context, contactID uuid.UUID, channel Channel) (bool, error)
	// ContactHasEmail returns true when the contact record has a non-empty
	// email address. Used to skip the email-channel check for email-less contacts.
	ContactHasEmail(ctx context.Context, contactID uuid.UUID) (bool, error)
}

// Asserter guards outbound communication by verifying active consent before
// any send operation reaches the transport layer.
type Asserter struct {
	repo AssertRepo
	log  *slog.Logger
}

// NewAsserter creates an Asserter backed by the given repo.
func NewAsserter(repo AssertRepo, log *slog.Logger) *Asserter {
	if log == nil {
		log = slog.Default()
	}
	return &Asserter{repo: repo, log: log}
}

// Assert blocks a send attempt when no active (granted, non-revoked) consent
// exists for the contact + channel pair.
//
// Special case: for ChannelEmail the check is skipped when the contact has no
// email address — there is nothing to send, so no consent is required.
func (a *Asserter) Assert(ctx context.Context, contactID uuid.UUID, channel Channel) error {
	if channel == ChannelEmail {
		hasEmail, err := a.repo.ContactHasEmail(ctx, contactID)
		if err != nil {
			return err
		}
		if !hasEmail {
			// No email address on file — consent check not applicable.
			return nil
		}
	}

	active, err := a.repo.ActiveConsent(ctx, contactID, channel)
	if err != nil {
		return err
	}
	if !active {
		a.log.WarnContext(ctx, "consent_block",
			"contact_id", contactID.String(),
			"channel", string(channel),
		)
		return ErrNoConsent
	}
	return nil
}
