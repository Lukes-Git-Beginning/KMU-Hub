package vertraege

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRepository implements Repository using PostgreSQL.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgreSQL-backed repository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// SaveSignature updates a contract's inline EES signature fields and returns the updated record.
// Returns ErrContractNotFound when no contract with the given id+tenant_id exists.
func (r *PostgresRepository) SaveSignature(ctx context.Context, tenantID, contractID, signatureData, signedBy string) (*Contract, error) {
	c, err := r.scanContract(r.pool.QueryRow(ctx, `
		UPDATE contracts
		SET signature_data = $3, signed_at = NOW(), signed_by = $4, updated_at = NOW()
		WHERE id = $1 AND tenant_id = $2
		RETURNING id, tenant_id, contract_number, title, contract_type, status,
		          starts_on, ends_on, document_url, notes, created_by, signature_provider,
		          signature_data, signed_at, signed_by,
		          created_at, updated_at`,
		contractID, tenantID, signatureData, signedBy,
	))
	if err != nil {
		return nil, err // scanContract already wraps ErrContractNotFound / fmt.Errorf
	}
	return c, nil
}

// compile-time interface check
var _ Repository = (*PostgresRepository)(nil)

// ============================================================================
// Contracts
// ============================================================================

func (r *PostgresRepository) CreateContract(ctx context.Context, c *Contract) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO contracts
		    (id, tenant_id, contract_number, title, contract_type, status,
		     starts_on, ends_on, document_url, notes, created_by, signature_provider,
		     signature_data, signed_at, signed_by,
		     created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)`,
		c.ID, c.TenantID, c.ContractNumber, c.Title, c.ContractType, c.Status,
		c.StartsOn, c.EndsOn, c.DocumentURL, c.Notes, c.CreatedBy, c.SignatureProvider,
		c.SignatureData, c.SignedAt, c.SignedBy,
		c.CreatedAt, c.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) UpdateContract(ctx context.Context, c *Contract) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE contracts
		SET title=$1, contract_type=$2, status=$3, starts_on=$4, ends_on=$5,
		    document_url=$6, notes=$7, signature_provider=$8, updated_at=$9
		WHERE id=$10 AND tenant_id=$11`,
		c.Title, c.ContractType, c.Status, c.StartsOn, c.EndsOn,
		c.DocumentURL, c.Notes, c.SignatureProvider, c.UpdatedAt,
		c.ID, c.TenantID,
	)
	return err
}

func (r *PostgresRepository) GetContract(ctx context.Context, tenantID, contractID uuid.UUID) (*Contract, error) {
	c, err := r.scanContract(r.pool.QueryRow(ctx, `
		SELECT id, tenant_id, contract_number, title, contract_type, status,
		       starts_on, ends_on, document_url, notes, created_by, signature_provider,
		       signature_data, signed_at, signed_by,
		       created_at, updated_at
		FROM contracts
		WHERE id=$1 AND tenant_id=$2`,
		contractID, tenantID,
	))
	if err != nil {
		return nil, err
	}

	parties, err := r.ListParties(ctx, tenantID, contractID)
	if err != nil {
		return nil, fmt.Errorf("get parties: %w", err)
	}
	c.Parties = parties

	reminders, err := r.ListReminders(ctx, tenantID, contractID, false)
	if err != nil {
		return nil, fmt.Errorf("get reminders: %w", err)
	}
	c.Reminders = reminders

	return c, nil
}

func (r *PostgresRepository) ListContracts(ctx context.Context, tenantID uuid.UUID, filter ListContractsFilter, offset, limit int) ([]*Contract, int, error) {
	var conditions []string
	var args []any
	n := 1

	conditions = append(conditions, fmt.Sprintf("tenant_id = $%d", n))
	args = append(args, tenantID)
	n++

	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", n))
		args = append(args, *filter.Status)
		n++
	}
	if filter.Type != nil {
		conditions = append(conditions, fmt.Sprintf("contract_type = $%d", n))
		args = append(args, *filter.Type)
		n++
	}
	if filter.StartsAfter != nil {
		conditions = append(conditions, fmt.Sprintf("starts_on >= $%d", n))
		args = append(args, *filter.StartsAfter)
		n++
	}
	if filter.StartsBefore != nil {
		conditions = append(conditions, fmt.Sprintf("starts_on <= $%d", n))
		args = append(args, *filter.StartsBefore)
		n++
	}
	if filter.EndsAfter != nil {
		conditions = append(conditions, fmt.Sprintf("ends_on >= $%d", n))
		args = append(args, *filter.EndsAfter)
		n++
	}
	if filter.EndsBefore != nil {
		conditions = append(conditions, fmt.Sprintf("ends_on <= $%d", n))
		args = append(args, *filter.EndsBefore)
		n++
	}
	if filter.ContactID != nil {
		conditions = append(conditions, fmt.Sprintf(
			"EXISTS (SELECT 1 FROM contract_parties cp WHERE cp.contract_id = contracts.id AND cp.contact_id = $%d)", n))
		args = append(args, *filter.ContactID)
		n++
	}

	where := "WHERE " + strings.Join(conditions, " AND ")

	var total int
	if scanErr := r.pool.QueryRow(ctx,
		fmt.Sprintf("SELECT COUNT(*) FROM contracts %s", where), args...,
	).Scan(&total); scanErr != nil {
		return nil, 0, fmt.Errorf("count contracts: %w", scanErr)
	}

	query := fmt.Sprintf(`
		SELECT id, tenant_id, contract_number, title, contract_type, status,
		       starts_on, ends_on, document_url, notes, created_by, signature_provider,
		       signature_data, signed_at, signed_by,
		       created_at, updated_at
		FROM contracts %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d`, where, n, n+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list contracts: %w", err)
	}
	defer rows.Close()

	var contracts []*Contract
	for rows.Next() {
		c, scanErr := r.scanContractFromRows(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		contracts = append(contracts, c)
	}

	return contracts, total, rows.Err()
}

func (r *PostgresRepository) DeleteContract(ctx context.Context, tenantID, contractID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM contracts WHERE id=$1 AND tenant_id=$2`,
		contractID, tenantID,
	)
	return err
}

func (r *PostgresRepository) ContractNumberExists(ctx context.Context, tenantID uuid.UUID, number string, excludeID *uuid.UUID) (bool, error) {
	var exists bool
	if excludeID != nil {
		err := r.pool.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM contracts WHERE tenant_id=$1 AND contract_number=$2 AND id<>$3)`,
			tenantID, number, *excludeID,
		).Scan(&exists)
		return exists, err
	}
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM contracts WHERE tenant_id=$1 AND contract_number=$2)`,
		tenantID, number,
	).Scan(&exists)
	return exists, err
}

func (r *PostgresRepository) ExpireContracts(ctx context.Context) (int64, error) {
	tag, err := r.pool.Exec(ctx, `
		UPDATE contracts
		SET status='expired', updated_at=NOW()
		WHERE status='active'
		  AND ends_on IS NOT NULL
		  AND ends_on < CURRENT_DATE`)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// ============================================================================
// Parties
// ============================================================================

func (r *PostgresRepository) AddParty(ctx context.Context, p *ContractParty) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO contract_parties
		    (id, tenant_id, contract_id, party_type, contact_id, company_id,
		     external_name, role_in_contract, signed_on, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		p.ID, p.TenantID, p.ContractID, p.PartyType, p.ContactID, p.CompanyID,
		p.ExternalName, p.RoleInContract, p.SignedOn, p.CreatedAt,
	)
	return err
}

func (r *PostgresRepository) RemoveParty(ctx context.Context, tenantID, partyID uuid.UUID) (uuid.UUID, error) {
	var contractID uuid.UUID
	err := r.pool.QueryRow(ctx,
		`DELETE FROM contract_parties WHERE id=$1 AND tenant_id=$2 RETURNING contract_id`,
		partyID, tenantID,
	).Scan(&contractID)
	if errors.Is(err, pgx.ErrNoRows) {
		// Nothing matched — either the party is already gone or it belongs to
		// another tenant. Both were a silent no-op before RETURNING was added
		// and stay one; the caller just has no contract to file an entry against.
		return uuid.Nil, nil
	}
	if err != nil {
		return uuid.Nil, err
	}
	return contractID, nil
}

func (r *PostgresRepository) ListParties(ctx context.Context, tenantID, contractID uuid.UUID) ([]*ContractParty, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, tenant_id, contract_id, party_type, contact_id, company_id,
		       external_name, role_in_contract, signed_on, created_at
		FROM contract_parties
		WHERE tenant_id=$1 AND contract_id=$2
		ORDER BY created_at ASC`,
		tenantID, contractID,
	)
	if err != nil {
		return nil, fmt.Errorf("list parties: %w", err)
	}
	defer rows.Close()

	var parties []*ContractParty
	for rows.Next() {
		p, scanErr := r.scanParty(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		parties = append(parties, p)
	}
	return parties, rows.Err()
}

// ============================================================================
// Reminders
// ============================================================================

func (r *PostgresRepository) CreateReminder(ctx context.Context, rem *ContractReminder) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO contract_reminders
		    (id, tenant_id, contract_id, remind_at, reminder_type, subject, message, status, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		rem.ID, rem.TenantID, rem.ContractID, rem.RemindAt,
		rem.ReminderType, rem.Subject, rem.Message, rem.Status, rem.CreatedAt,
	)
	return err
}

func (r *PostgresRepository) UpdateReminder(ctx context.Context, rem *ContractReminder) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE contract_reminders
		SET remind_at=$1, reminder_type=$2, subject=$3, message=$4, status=$5
		WHERE id=$6 AND tenant_id=$7`,
		rem.RemindAt, rem.ReminderType, rem.Subject, rem.Message, rem.Status,
		rem.ID, rem.TenantID,
	)
	return err
}

func (r *PostgresRepository) GetReminder(ctx context.Context, tenantID, reminderID uuid.UUID) (*ContractReminder, error) {
	var rem ContractReminder
	err := r.pool.QueryRow(ctx, `
		SELECT id, tenant_id, contract_id, remind_at, reminder_type, subject, message, status, created_at, sent_at
		FROM contract_reminders
		WHERE id=$1 AND tenant_id=$2`,
		reminderID, tenantID,
	).Scan(
		&rem.ID, &rem.TenantID, &rem.ContractID, &rem.RemindAt,
		&rem.ReminderType, &rem.Subject, &rem.Message, &rem.Status,
		&rem.CreatedAt, &rem.SentAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrReminderNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get reminder: %w", err)
	}
	return &rem, nil
}

func (r *PostgresRepository) DeleteReminder(ctx context.Context, tenantID, reminderID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM contract_reminders WHERE id=$1 AND tenant_id=$2`,
		reminderID, tenantID,
	)
	return err
}

func (r *PostgresRepository) ListReminders(ctx context.Context, tenantID, contractID uuid.UUID, onlyPending bool) ([]*ContractReminder, error) {
	query := `
		SELECT id, tenant_id, contract_id, remind_at, reminder_type, subject, message, status, created_at, sent_at
		FROM contract_reminders
		WHERE tenant_id=$1 AND contract_id=$2`
	if onlyPending {
		query += ` AND status='pending'`
	}
	query += ` ORDER BY remind_at ASC`

	rows, err := r.pool.Query(ctx, query, tenantID, contractID)
	if err != nil {
		return nil, fmt.Errorf("list reminders: %w", err)
	}
	defer rows.Close()

	var reminders []*ContractReminder
	for rows.Next() {
		rem, scanErr := r.scanReminder(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		reminders = append(reminders, rem)
	}
	return reminders, rows.Err()
}

func (r *PostgresRepository) ClaimDueReminders(ctx context.Context) ([]*ContractReminder, error) {
	rows, err := r.pool.Query(ctx, `
		UPDATE contract_reminders
		SET status='sent', sent_at=NOW()
		WHERE status='pending' AND remind_at <= NOW()
		RETURNING id, tenant_id, contract_id, remind_at, reminder_type, subject, message, status, created_at, sent_at`)
	if err != nil {
		return nil, fmt.Errorf("claim due reminders: %w", err)
	}
	defer rows.Close()

	var reminders []*ContractReminder
	for rows.Next() {
		rem, scanErr := r.scanReminder(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		reminders = append(reminders, rem)
	}
	return reminders, rows.Err()
}

func (r *PostgresRepository) MarkReminderSent(ctx context.Context, reminderID uuid.UUID, sentAt time.Time) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE contract_reminders
		SET status='sent', sent_at=$1
		WHERE id=$2`,
		sentAt, reminderID,
	)
	return err
}

// ============================================================================
// Scan helpers
// ============================================================================

func (r *PostgresRepository) scanContract(row pgx.Row) (*Contract, error) {
	var c Contract
	err := row.Scan(
		&c.ID, &c.TenantID, &c.ContractNumber, &c.Title, &c.ContractType, &c.Status,
		&c.StartsOn, &c.EndsOn, &c.DocumentURL, &c.Notes, &c.CreatedBy, &c.SignatureProvider,
		&c.SignatureData, &c.SignedAt, &c.SignedBy,
		&c.CreatedAt, &c.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrContractNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan contract: %w", err)
	}
	return &c, nil
}

func (r *PostgresRepository) scanContractFromRows(rows pgx.Rows) (*Contract, error) {
	var c Contract
	err := rows.Scan(
		&c.ID, &c.TenantID, &c.ContractNumber, &c.Title, &c.ContractType, &c.Status,
		&c.StartsOn, &c.EndsOn, &c.DocumentURL, &c.Notes, &c.CreatedBy, &c.SignatureProvider,
		&c.SignatureData, &c.SignedAt, &c.SignedBy,
		&c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan contract row: %w", err)
	}
	return &c, nil
}

func (r *PostgresRepository) scanParty(rows pgx.Rows) (*ContractParty, error) {
	var p ContractParty
	err := rows.Scan(
		&p.ID, &p.TenantID, &p.ContractID, &p.PartyType, &p.ContactID, &p.CompanyID,
		&p.ExternalName, &p.RoleInContract, &p.SignedOn, &p.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan party row: %w", err)
	}
	return &p, nil
}

func (r *PostgresRepository) scanReminder(rows pgx.Rows) (*ContractReminder, error) {
	var rem ContractReminder
	err := rows.Scan(
		&rem.ID, &rem.TenantID, &rem.ContractID, &rem.RemindAt,
		&rem.ReminderType, &rem.Subject, &rem.Message, &rem.Status,
		&rem.CreatedAt, &rem.SentAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan reminder row: %w", err)
	}
	return &rem, nil
}

// ============================================================================
// Contract events (append-only)
// ============================================================================

func (r *PostgresRepository) CreateContractEvent(ctx context.Context, e *ContractEvent) error {
	payload := e.Payload
	if payload == nil {
		// The column is NOT NULL DEFAULT '{}' — pgx would encode a nil map as
		// SQL NULL and violate it, so an event without extra facts still
		// carries an empty object.
		payload = map[string]any{}
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO contract_events
		    (id, tenant_id, contract_id, action, user_id, payload, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		e.ID, e.TenantID, e.ContractID, string(e.Action), e.UserID, payload, e.CreatedAt,
	)
	return err
}

func (r *PostgresRepository) ListContractEvents(ctx context.Context, tenantID, contractID uuid.UUID, offset, limit int) ([]*ContractEvent, int, error) {
	var total int
	if err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM contract_events WHERE tenant_id=$1 AND contract_id=$2`,
		tenantID, contractID,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count contract events: %w", err)
	}

	rows, err := r.pool.Query(ctx, `
		SELECT id, tenant_id, contract_id, action, user_id, payload, created_at
		FROM contract_events
		WHERE tenant_id=$1 AND contract_id=$2
		ORDER BY created_at DESC, id DESC
		LIMIT $3 OFFSET $4`,
		tenantID, contractID, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list contract events: %w", err)
	}
	defer rows.Close()

	events := make([]*ContractEvent, 0, limit)
	for rows.Next() {
		var e ContractEvent
		var action string
		if scanErr := rows.Scan(
			&e.ID, &e.TenantID, &e.ContractID, &action, &e.UserID, &e.Payload, &e.CreatedAt,
		); scanErr != nil {
			return nil, 0, fmt.Errorf("scan contract event row: %w", scanErr)
		}
		e.Action = ContractEventAction(action)
		events = append(events, &e)
	}
	return events, total, rows.Err()
}
