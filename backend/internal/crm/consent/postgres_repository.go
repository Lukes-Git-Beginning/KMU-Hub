package consent

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRepository implements Repository using PostgreSQL.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// AnonymizedFirstName and AnonymizedLastName are the placeholder names
// AnonymizeContact writes. Exported so callers (e.g. the contact retention
// handler) can recognise an already-anonymized row without a second copy of
// the literal.
const (
	AnonymizedFirstName = "Gelöschte"
	AnonymizedLastName  = "Person"
)

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
		`INSERT INTO gdpr_deletion_requests (id, tenant_id, contact_id, original_contact_id, requested_by, reason, status, created_at)
		 VALUES ($1, $2, $3, $3, $4, $5, $6, $7)`,
		req.ID, req.TenantID, req.ContactID, req.RequestedBy, req.Reason, req.Status, req.CreatedAt,
	)
	return err
}

func (r *PostgresRepository) GetDeletionRequest(ctx context.Context, id, tenantID uuid.UUID) (*GDPRDeletionRequest, error) {
	var req GDPRDeletionRequest
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, contact_id, original_contact_id, requested_by, reason, status, completed_at, created_at
		 FROM gdpr_deletion_requests WHERE id = $1 AND tenant_id = $2`,
		id, tenantID,
	).Scan(&req.ID, &req.TenantID, &req.ContactID, &req.OriginalContactID, &req.RequestedBy, &req.Reason, &req.Status, &req.CompletedAt, &req.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrDeletionRequestNotFound
	}
	return &req, err
}

func (r *PostgresRepository) UpdateDeletionRequest(ctx context.Context, req *GDPRDeletionRequest) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE gdpr_deletion_requests SET status = $1, completed_at = $2 WHERE id = $3 AND tenant_id = $4`,
		req.Status, req.CompletedAt, req.ID, req.TenantID,
	)
	return err
}

// AnonymizeContact executes the GDPR Art. 17 erasure for a contact by
// overwriting its personal data with a sentinel and stripping personal data
// from records that hang off it. The contact row itself is kept (not
// hard-deleted): dialer_campaign_contacts.contact_id and
// advisory_protocols.contact_id are ON DELETE RESTRICT (migrations
// 000130/000137) because they are audit/compliance material, and this is the
// erasure path that respects that -- see contact.PostgresRepository.IsInUse.
//
// What stays untouched, deliberately:
//   - company_id, owner_id, visibility, category, status, lifecycle_stage,
//     lead_* -- business/relationship metadata, not personal data. Same
//     rationale erasure.go's CRMErasureHandler uses to retain deal/pipeline
//     history on user erasure.
//   - merged_into_id, referred_by_contact_id -- relationship pointers
//     (UUIDs), not personal data themselves.
//   - dialer_campaign_contacts.notes, advisory_protocols content -- audit /
//     legal-retention material (advisory_protocols: 10y under FinVermV §18a).
//     Exactly why those FKs are RESTRICT instead of CASCADE: the anonymized
//     contact row stays as their anchor.
//   - consent_records rows -- Art. 7(1) requires the controller to be able to
//     demonstrate that consent was given/withdrawn, so the grant/revoke
//     history (type, granted, legal_basis, dates) is retained; only the two
//     directly identifying fields on each record (ip_address, notes) are
//     cleared.
func (r *PostgresRepository) AnonymizeContact(ctx context.Context, contactID, tenantID uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("gdpr contact anonymize: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	affected := 0

	res, err := tx.Exec(ctx,
		`UPDATE contacts SET
			first_name = $3,
			last_name = $4,
			email = NULL,
			phone = NULL,
			mobile = NULL,
			position = NULL,
			salutation = NULL,
			title = NULL,
			department = NULL,
			notes = NULL,
			address_street = NULL,
			address_zip = NULL,
			address_city = NULL,
			address_country = NULL,
			website = NULL,
			linkedin = NULL,
			xing = NULL,
			updated_at = NOW()
		 WHERE id = $1 AND tenant_id = $2`,
		contactID, tenantID, AnonymizedFirstName, AnonymizedLastName,
	)
	if err != nil {
		return fmt.Errorf("gdpr contact anonymize: update contact: %w", err)
	}
	affected += int(res.RowsAffected())

	// Free-text activity notes may name or describe the contact; the activity
	// skeleton (type, subject, dates) stays for business continuity -- same
	// pattern erasure.go's CRMErasureHandler uses for user erasure.
	res, err = tx.Exec(ctx,
		`UPDATE activities SET description = NULL, updated_at = NOW() WHERE contact_id = $1 AND tenant_id = $2`,
		contactID, tenantID,
	)
	if err != nil {
		return fmt.Errorf("gdpr contact anonymize: clear activity notes: %w", err)
	}
	affected += int(res.RowsAffected())

	// Tags are free-form labels ("VIP", "schwierig") and are themselves a form
	// of personal characterization.
	res, err = tx.Exec(ctx, `DELETE FROM contact_tags WHERE contact_id = $1`, contactID)
	if err != nil {
		return fmt.Errorf("gdpr contact anonymize: delete tags: %w", err)
	}
	affected += int(res.RowsAffected())

	// Custom fields are tenant-defined with no fixed schema, so there is no
	// way to know generically which hold PII -- clear all of them. Same call
	// erasure.go's WorkErasureHandler makes for time_entries: no business
	// retention need.
	res, err = tx.Exec(ctx, `DELETE FROM contact_custom_field_values WHERE contact_id = $1`, contactID)
	if err != nil {
		return fmt.Errorf("gdpr contact anonymize: delete custom fields: %w", err)
	}
	affected += int(res.RowsAffected())

	res, err = tx.Exec(ctx,
		`UPDATE consent_records SET ip_address = NULL, notes = NULL
		 WHERE contact_id = $1 AND tenant_id = $2 AND (ip_address IS NOT NULL OR COALESCE(notes, '') <> '')`,
		contactID, tenantID,
	)
	if err != nil {
		return fmt.Errorf("gdpr contact anonymize: scrub consent records: %w", err)
	}
	affected += int(res.RowsAffected())

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("gdpr contact anonymize: commit: %w", err)
	}

	slog.Info("gdpr contact anonymize: complete",
		"contact_id", contactID,
		"tenant_id", tenantID,
		"records_affected", affected,
	)
	return nil
}

func (r *PostgresRepository) ContactExists(ctx context.Context, contactID, tenantID uuid.UUID) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM contacts WHERE id = $1 AND tenant_id = $2)`, contactID, tenantID,
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
	"marketing_email": true,
	"marketing_phone": true,
	"profiling":       true,
	"newsletter":      true,
	"data_processing": true,
	"data_sharing":    true,
}

// ValidLegalBases lists all valid consent_legal_basis enum values.
var ValidLegalBases = map[string]bool{
	"consent":             true,
	"legitimate_interest": true,
	"contract":            true,
	"legal_obligation":    true,
}

// unused: suppress "time imported and not used"
var _ = time.Now
