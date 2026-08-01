package banking

// Bank accounts (Migration 000258) — master data for the statements this
// package imports.
//
// Account numbers are personal financial data: no method here logs an IBAN, a
// BIC or a bank name, not even at debug level. Errors identify an account by
// its id, which is meaningless outside this database.

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/dachfmt"
	"github.com/kmuhub/kmuhub/internal/models"
)

// maxBankNameLen bounds the display name. Long enough for "Volksbank
// Raiffeisenbank Bayern Mitte eG", short enough that the column is not a
// free-text dump.
const maxBankNameLen = 160

// CreateAccountInput is a new bank account. The IBAN may arrive grouped as a
// human types it; it is normalised before it is stored.
type CreateAccountInput struct {
	TenantID uuid.UUID
	BankName string
	IBAN     string
	BIC      string
	// Currency is the ISO 4217 code; empty defaults to EUR.
	Currency string
}

// UpdateAccountInput edits an account. Every field is a pointer: an absent one
// stays untouched, so editing the BIC alone does not clear the bank name.
type UpdateAccountInput struct {
	TenantID uuid.UUID
	ID       uuid.UUID
	BankName *string
	IBAN     *string
	BIC      *string
	Currency *string
	// Connected toggles the link flag directly. ConnectAccount is the path the
	// desktop client uses; this one exists so a correction can undo it.
	Connected *bool
}

// CreateAccount records a new bank account.
func (s *Service) CreateAccount(ctx context.Context, in CreateAccountInput) (*models.BankAccount, error) {
	acc := &models.BankAccount{TenantID: in.TenantID}
	if err := applyAccountName(acc, in.BankName); err != nil {
		return nil, err
	}
	if err := applyAccountIBAN(acc, in.IBAN); err != nil {
		return nil, err
	}
	if err := applyAccountBIC(acc, in.BIC); err != nil {
		return nil, err
	}
	if err := applyAccountCurrency(acc, in.Currency); err != nil {
		return nil, err
	}

	if err := s.repo.CreateAccount(ctx, acc); err != nil {
		return nil, err
	}
	s.logger.InfoContext(ctx, "bank account created",
		"account_id", acc.ID, "tenant_id", acc.TenantID)
	return acc, nil
}

// GetAccount resolves one account of the tenant.
func (s *Service) GetAccount(ctx context.Context, tenantID, id uuid.UUID) (*models.BankAccount, error) {
	return s.repo.GetAccount(ctx, tenantID, id)
}

// ListAccounts returns every account the tenant holds.
func (s *Service) ListAccounts(ctx context.Context, tenantID uuid.UUID) ([]*models.BankAccount, error) {
	return s.repo.ListAccounts(ctx, tenantID)
}

// UpdateAccount edits the master data of an account.
//
// Changing the IBAN is allowed and rewrites which statements the account reads
// its balance from -- that is the point: an account entered with a typo would
// otherwise stay disconnected from its own imports forever.
func (s *Service) UpdateAccount(ctx context.Context, in UpdateAccountInput) (*models.BankAccount, error) {
	acc, err := s.repo.GetAccount(ctx, in.TenantID, in.ID)
	if err != nil {
		return nil, err
	}

	if in.BankName != nil {
		if err := applyAccountName(acc, *in.BankName); err != nil {
			return nil, err
		}
	}
	if in.IBAN != nil {
		if err := applyAccountIBAN(acc, *in.IBAN); err != nil {
			return nil, err
		}
	}
	if in.BIC != nil {
		if err := applyAccountBIC(acc, *in.BIC); err != nil {
			return nil, err
		}
	}
	if in.Currency != nil {
		if err := applyAccountCurrency(acc, *in.Currency); err != nil {
			return nil, err
		}
	}
	if in.Connected != nil {
		setAccountConnected(acc, *in.Connected)
	}

	if err := s.repo.UpdateAccount(ctx, acc); err != nil {
		return nil, err
	}
	// Re-read so the derived balance follows a changed IBAN instead of
	// reporting the previous account's statements.
	return s.repo.GetAccount(ctx, in.TenantID, in.ID)
}

// ConnectAccount marks an account as linked.
//
// There is no bank session behind this: the PSD2 handshake is simulated in the
// desktop client and the real integration is P5 (see the migration header).
// Connecting an already connected account is a no-op that keeps the original
// timestamp, so a retried request does not move the date.
//
// lean: a flag, not a connection. Replace this with the FinAPI/PSD2 token
// exchange once a customer's account is actually to be polled -- the signature
// and the stored fields stay, only the body grows a real handshake.
func (s *Service) ConnectAccount(ctx context.Context, tenantID, id uuid.UUID) (*models.BankAccount, error) {
	acc, err := s.repo.GetAccount(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if acc.Connected {
		return acc, nil
	}
	setAccountConnected(acc, true)
	if err := s.repo.UpdateAccount(ctx, acc); err != nil {
		return nil, err
	}
	s.logger.InfoContext(ctx, "bank account connected",
		"account_id", acc.ID, "tenant_id", acc.TenantID)
	return acc, nil
}

// DeleteAccount removes an account. Statements already imported for its IBAN
// stay: they are what a bank reported, not something this row owns.
func (s *Service) DeleteAccount(ctx context.Context, tenantID, id uuid.UUID) error {
	if err := s.repo.DeleteAccount(ctx, tenantID, id); err != nil {
		return err
	}
	s.logger.InfoContext(ctx, "bank account deleted",
		"account_id", id, "tenant_id", tenantID)
	return nil
}

// setAccountConnected flips the flag and keeps connected_at consistent with it:
// a disconnect clears the timestamp, so a later reconnect cannot be read as
// having been linked all along.
func setAccountConnected(acc *models.BankAccount, connected bool) {
	acc.Connected = connected
	if connected {
		now := time.Now().UTC()
		acc.ConnectedAt = &now
		return
	}
	acc.ConnectedAt = nil
}

func applyAccountName(acc *models.BankAccount, name string) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return fmt.Errorf("%w: bank name is required", ErrInvalidAccount)
	}
	if len([]rune(trimmed)) > maxBankNameLen {
		return fmt.Errorf("%w: bank name is too long", ErrInvalidAccount)
	}
	acc.BankName = trimmed
	return nil
}

// applyAccountIBAN validates and canonicalises. The check digits are verified
// here rather than trusted from the client: an IBAN that fails mod-97 is a typo,
// and stored uncorrected it would silently never match any imported statement.
func applyAccountIBAN(acc *models.BankAccount, raw string) error {
	if !dachfmt.ValidateIBAN(raw) {
		return ErrInvalidIBAN
	}
	acc.IBAN = dachfmt.NormalizeIBAN(raw)
	return nil
}

// applyAccountBIC accepts an empty BIC: SEPA transfers within the EU are
// IBAN-only, so demanding one would reject valid input.
func applyAccountBIC(acc *models.BankAccount, raw string) error {
	trimmed := strings.ToUpper(strings.Join(strings.Fields(raw), ""))
	if trimmed != "" && !dachfmt.ValidateBIC(trimmed) {
		return fmt.Errorf("%w: invalid BIC", ErrInvalidAccount)
	}
	acc.BIC = trimmed
	return nil
}

func applyAccountCurrency(acc *models.BankAccount, raw string) error {
	trimmed := strings.ToUpper(strings.TrimSpace(raw))
	if trimmed == "" {
		trimmed = "EUR"
	}
	if len(trimmed) != 3 {
		return fmt.Errorf("%w: currency must be a three-letter ISO 4217 code", ErrInvalidAccount)
	}
	for _, c := range trimmed {
		if c < 'A' || c > 'Z' {
			return fmt.Errorf("%w: currency must be a three-letter ISO 4217 code", ErrInvalidAccount)
		}
	}
	acc.Currency = trimmed
	return nil
}
