package account

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
)

// PostgresRepository implements Repository using PostgreSQL.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgreSQL account repository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) Create(ctx context.Context, account *models.EmailAccount) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO email_accounts (id, tenant_id, user_id, email_address, display_name,
			imap_host, imap_port, smtp_host, smtp_port, username, password_encrypted,
			use_ssl, sync_enabled, is_default, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)`,
		account.ID, account.TenantID, account.UserID, account.EmailAddress, account.DisplayName,
		account.IMAPHost, account.IMAPPort, account.SMTPHost, account.SMTPPort,
		account.Username, account.PasswordEncrypted,
		account.UseSSL, account.SyncEnabled, account.IsDefault, account.CreatedAt, account.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) (*models.EmailAccount, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, user_id, email_address, display_name,
			imap_host, imap_port, smtp_host, smtp_port, username, password_encrypted,
			use_ssl, last_sync_at, sync_enabled, is_default, created_at, updated_at
		 FROM email_accounts WHERE id = $1 AND tenant_id = $2`, id, tenantID,
	)
	return scanAccount(row)
}

// GetByUserIDAndTenant returns the user's default account. Callers that need
// every account the user holds must use ListByUserAndTenant instead.
func (r *PostgresRepository) GetByUserIDAndTenant(ctx context.Context, userID uuid.UUID, tenantID uuid.UUID) (*models.EmailAccount, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, user_id, email_address, display_name,
			imap_host, imap_port, smtp_host, smtp_port, username, password_encrypted,
			use_ssl, last_sync_at, sync_enabled, is_default, created_at, updated_at
		 FROM email_accounts WHERE user_id = $1 AND tenant_id = $2 AND is_default = true`, userID, tenantID,
	)
	return scanAccount(row)
}

func (r *PostgresRepository) ListByUserAndTenant(ctx context.Context, userID uuid.UUID, tenantID uuid.UUID) ([]*models.EmailAccount, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, user_id, email_address, display_name,
			imap_host, imap_port, smtp_host, smtp_port, username, password_encrypted,
			use_ssl, last_sync_at, sync_enabled, is_default, created_at, updated_at
		 FROM email_accounts WHERE user_id = $1 AND tenant_id = $2 ORDER BY created_at ASC`, userID, tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []*models.EmailAccount
	for rows.Next() {
		a, scanErr := scanAccount(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		accounts = append(accounts, a)
	}
	return accounts, rows.Err()
}

// SetDefault atomically makes id the sole default account for userID in one
// statement -- avoids the clear-then-set race a two-step update would have.
func (r *PostgresRepository) SetDefault(ctx context.Context, id uuid.UUID, userID uuid.UUID, tenantID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE email_accounts SET is_default = (id = $1), updated_at = $4
		 WHERE tenant_id = $2 AND user_id = $3`,
		id, tenantID, userID, time.Now().UTC(),
	)
	return err
}

func (r *PostgresRepository) Update(ctx context.Context, account *models.EmailAccount) error {
	account.UpdatedAt = time.Now().UTC()
	_, err := r.pool.Exec(ctx,
		`UPDATE email_accounts SET
			email_address = $2, display_name = $3,
			imap_host = $4, imap_port = $5, smtp_host = $6, smtp_port = $7,
			username = $8, password_encrypted = $9, use_ssl = $10,
			last_sync_at = $11, sync_enabled = $12, updated_at = $13
		 WHERE id = $1 AND tenant_id = $14`,
		account.ID, account.EmailAddress, account.DisplayName,
		account.IMAPHost, account.IMAPPort, account.SMTPHost, account.SMTPPort,
		account.Username, account.PasswordEncrypted, account.UseSSL,
		account.LastSyncAt, account.SyncEnabled, account.UpdatedAt,
		account.TenantID,
	)
	return err
}

func (r *PostgresRepository) Delete(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM email_accounts WHERE id = $1 AND tenant_id = $2`, id, tenantID)
	return err
}

func (r *PostgresRepository) ListActive(ctx context.Context, tenantID uuid.UUID) ([]*models.EmailAccount, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, user_id, email_address, display_name,
			imap_host, imap_port, smtp_host, smtp_port, username, password_encrypted,
			use_ssl, last_sync_at, sync_enabled, is_default, created_at, updated_at
		 FROM email_accounts WHERE sync_enabled = true AND tenant_id = $1`, tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []*models.EmailAccount
	for rows.Next() {
		a, scanErr := scanAccount(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		accounts = append(accounts, a)
	}
	return accounts, rows.Err()
}

// ListAllActive returns all sync-enabled accounts across ALL tenants.
//
// INTERNAL — cross-tenant system query. This function intentionally omits the
// tenant_id filter because it is called exclusively by the background sync
// engine, which must bootstrap IMAP/SMTP connections for every active account
// regardless of tenant boundary.
//
// Authorised callers (audit list — add new callers here before merging):
//   - internal/email/sync/engine.go → Engine.startSyncLoops (service startup)
//   - internal/email/sync/engine.go → Engine.triggerImmediateSync (manual re-sync)
//
// MUST NOT be called from HTTP handlers or gRPC service methods. Any new caller
// that is not a background system worker must use ListActive(ctx, tenantID)
// instead, which enforces tenant isolation.
func (r *PostgresRepository) ListAllActive(ctx context.Context) ([]*models.EmailAccount, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, user_id, email_address, display_name,
			imap_host, imap_port, smtp_host, smtp_port, username, password_encrypted,
			use_ssl, last_sync_at, sync_enabled, is_default, created_at, updated_at
		 FROM email_accounts WHERE sync_enabled = true`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []*models.EmailAccount
	for rows.Next() {
		a, scanErr := scanAccount(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		accounts = append(accounts, a)
	}
	return accounts, rows.Err()
}

// scanAccount scans a single email account row into a model.
func scanAccount(row pgx.Row) (*models.EmailAccount, error) {
	var a models.EmailAccount
	err := row.Scan(
		&a.ID, &a.TenantID, &a.UserID, &a.EmailAddress, &a.DisplayName,
		&a.IMAPHost, &a.IMAPPort, &a.SMTPHost, &a.SMTPPort,
		&a.Username, &a.PasswordEncrypted,
		&a.UseSSL, &a.LastSyncAt, &a.SyncEnabled, &a.IsDefault,
		&a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAccountNotFound
		}
		return nil, err
	}
	return &a, nil
}
