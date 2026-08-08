package vertraege

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// ListContractsFilter holds optional filter criteria for listing contracts.
type ListContractsFilter struct {
	Status       *ContractStatus
	Type         *ContractType
	StartsAfter  *time.Time
	StartsBefore *time.Time
	EndsAfter    *time.Time
	EndsBefore   *time.Time
	// ContactID filters to contracts where at least one party has this contact_id.
	ContactID *uuid.UUID
}

// Repository defines the persistence interface for the vertraege module.
type Repository interface {
	// Contracts
	CreateContract(ctx context.Context, c *Contract) error
	UpdateContract(ctx context.Context, c *Contract) error
	GetContract(ctx context.Context, tenantID, contractID uuid.UUID) (*Contract, error)
	ListContracts(ctx context.Context, tenantID uuid.UUID, filter ListContractsFilter, offset, limit int) ([]*Contract, int, error)
	DeleteContract(ctx context.Context, tenantID, contractID uuid.UUID) error
	ContractNumberExists(ctx context.Context, tenantID uuid.UUID, number string, excludeID *uuid.UUID) (bool, error)

	// SaveSignature persists an EES inline signature for a contract and returns the updated record.
	SaveSignature(ctx context.Context, tenantID, contractID, signatureData, signedBy string) (*Contract, error)

	// Auto-expiry: marks active contracts as expired when ends_on < today
	ExpireContracts(ctx context.Context) (int64, error)

	// Parties
	AddParty(ctx context.Context, p *ContractParty) error
	// RemoveParty deletes a party and returns the contract it belonged to, so
	// the caller can file the audit entry against that contract. Removing a
	// party that does not exist stays a no-op and returns uuid.Nil.
	RemoveParty(ctx context.Context, tenantID, partyID uuid.UUID) (uuid.UUID, error)
	ListParties(ctx context.Context, tenantID, contractID uuid.UUID) ([]*ContractParty, error)

	// Contract events — append-only audit trail. There is deliberately no
	// update and no delete: an entry that can be rewritten is not a trail.
	CreateContractEvent(ctx context.Context, e *ContractEvent) error
	ListContractEvents(ctx context.Context, tenantID, contractID uuid.UUID, offset, limit int) ([]*ContractEvent, int, error)

	// Reminders
	CreateReminder(ctx context.Context, r *ContractReminder) error
	UpdateReminder(ctx context.Context, r *ContractReminder) error
	GetReminder(ctx context.Context, tenantID, reminderID uuid.UUID) (*ContractReminder, error)
	DeleteReminder(ctx context.Context, tenantID, reminderID uuid.UUID) error
	ListReminders(ctx context.Context, tenantID, contractID uuid.UUID, onlyPending bool) ([]*ContractReminder, error)

	// Worker: claim reminders that are due
	ClaimDueReminders(ctx context.Context) ([]*ContractReminder, error)
	MarkReminderSent(ctx context.Context, reminderID uuid.UUID, sentAt time.Time) error
}
