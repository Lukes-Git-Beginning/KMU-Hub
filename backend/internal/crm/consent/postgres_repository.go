package consent

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRepository implements Repository using PostgreSQL.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgreSQL consent repository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) CreateConsentRecord(ctx context.Context, record *ConsentRecord) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO consent_records
			(id, tenant_id, contact_id, consent_type, granted, legal_basis, source, ip_address, notes, granted_at, revoked_at, created_by, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
		record.ID, record.TenantID, record.ContactID, record.ConsentType, record.Granted, record.LegalBasis,
		record.Source, record.IPAddress, record.Notes,
		record.GrantedAt, record.RevokedAt, record.CreatedBy, record.CreatedAt,
	)
	return err
}

func (r *PostgresRepository) GetConsentHistory(ctx context.Context, tenantID, contactID uuid.UUID, consentType string) ([]*ConsentRecord, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, contact_id, consent_type, granted, legal_basis, source, ip_address::text, notes,
		        granted_at, revoked_at, created_by, created_at
		 FROM consent_records
		 WHERE tenant_id = $1 AND contact_id = $2 AND consent_type = $3
		 ORDER BY created_at DESC`,
		tenantID, contactID, consentType,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []*ConsentRecord
	for rows.Next() {
		rec, scanErr := scanConsentRecord(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		records = append(records, rec)
	}
	return records, rows.Err()
}

func (r *PostgresRepository) GetLatestConsents(ctx context.Context, tenantID, contactID uuid.UUID) ([]*ConsentRecord, error) {
	// Get the latest record per consent type (DISTINCT ON), scoped to tenant
	rows, err := r.pool.Query(ctx,
		`SELECT DISTINCT ON (consent_type)
		        id, contact_id, consent_type, granted, legal_basis, source, ip_address::text, notes,
		        granted_at, revoked_at, created_by, created_at
		 FROM consent_records
		 WHERE tenant_id = $1 AND contact_id = $2
		 ORDER BY consent_type, created_at DESC`,
		tenantID, contactID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []*ConsentRecord
	for rows.Next() {
		rec, scanErr := scanConsentRecord(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		records = append(records, rec)
	}
	return records, rows.Err()
}

func (r *PostgresRepository) CreateDeletionRequest(ctx context.Context, req *GDPRDeletionRequest) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO gdpr_deletion_requests (id, contact_id, requested_by, reason, status, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		req.ID, req.ContactID, req.RequestedBy, req.Reason, req.Status, req.CreatedAt,
	)
	return err
}

func (r *PostgresRepository) GetDeletionRequest(ctx context.Context, id uuid.UUID) (*GDPRDeletionRequest, error) {
	var req GDPRDeletionRequest
	err := r.pool.QueryRow(ctx,
		`SELECT id, contact_id, requested_by, reason, status, completed_at, created_at
		 FROM gdpr_deletion_requests WHERE id = $1`,
		id,
	).Scan(&req.ID, &req.ContactID, &req.RequestedBy, &req.Reason, &req.Status, &req.CompletedAt, &req.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrDeletionRequestNotFound
	}
	return &req, err
}

func (r *PostgresRepository) UpdateDeletionRequest(ctx context.Context, req *GDPRDeletionRequest) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE gdpr_deletion_requests SET status = $1, completed_at = $2 WHERE id = $3`,
		req.Status, req.CompletedAt, req.ID,
	)
	return err
}

func (r *PostgresRepository) AnonymizeContact(ctx context.Context, contactID uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Anonymize contact data (GDPR Art. 17 - right to erasure)
	if _, err := tx.Exec(ctx,
		`UPDATE contacts SET
			first_name = 'Gelöschte',
			last_name = 'Person',
			email = NULL,
			phone = NULL,
			position = NULL,
			notes = NULL,
			updated_at = NOW()
		 WHERE id = $1`,
		contactID,
	); err != nil {
		return err
	}

	// Anonymize activity descriptions
	if _, err := tx.Exec(ctx,
		`UPDATE activities SET description = NULL, updated_at = NOW() WHERE contact_id = $1`,
		contactID,
	); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *PostgresRepository) ContactExists(ctx context.Context, contactID uuid.UUID) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM contacts WHERE id = $1)`, contactID,
	).Scan(&exists)
	return exists, err
}

func scanConsentRecord(rows pgx.Rows) (*ConsentRecord, error) {
	var rec ConsentRecord
	var ipStr *string
	err := rows.Scan(
		&rec.ID, &rec.ContactID, &rec.ConsentType, &rec.Granted, &rec.LegalBasis,
		&rec.Source, &ipStr, &rec.Notes,
		&rec.GrantedAt, &rec.RevokedAt, &rec.CreatedBy, &rec.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	rec.IPAddress = ipStr
	return &rec, nil
}

// scanConsentRecordRow scans a single pgx.Row
func scanConsentRecordRow(row pgx.Row) (*ConsentRecord, error) {
	var rec ConsentRecord
	var ipStr *string
	err := row.Scan(
		&rec.ID, &rec.ContactID, &rec.ConsentType, &rec.Granted, &rec.LegalBasis,
		&rec.Source, &ipStr, &rec.Notes,
		&rec.GrantedAt, &rec.RevokedAt, &rec.CreatedBy, &rec.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrConsentNotFound
	}
	if err != nil {
		return nil, err
	}
	rec.IPAddress = ipStr
	return &rec, nil
}

// GetLastConsentByType returns the latest consent record for a contact/type pair, scoped to tenant.
func (r *PostgresRepository) GetLastConsentByType(ctx context.Context, tenantID, contactID uuid.UUID, consentType string) (*ConsentRecord, error) {
	return scanConsentRecordRow(r.pool.QueryRow(ctx,
		`SELECT id, contact_id, consent_type, granted, legal_basis, source, ip_address::text, notes,
		        granted_at, revoked_at, created_by, created_at
		 FROM consent_records
		 WHERE tenant_id = $1 AND contact_id = $2 AND consent_type = $3
		 ORDER BY created_at DESC LIMIT 1`,
		tenantID, contactID, consentType,
	))
}

// ValidConsentTypes lists all valid consent_type enum values.
var ValidConsentTypes = map[string]bool{
	"marketing_email":  true,
	"marketing_phone":  true,
	"profiling":        true,
	"newsletter":       true,
	"data_processing":  true,
	"data_sharing":     true,
}

// ValidLegalBases lists all valid consent_legal_basis enum values.
var ValidLegalBases = map[string]bool{
	"consent":              true,
	"legitimate_interest":  true,
	"contract":             true,
	"legal_obligation":     true,
}

// unused: suppress "time imported and not used"
var _ = time.Now
