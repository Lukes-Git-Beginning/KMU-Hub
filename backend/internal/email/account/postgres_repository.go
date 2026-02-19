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
		`INSERT INTO email_accounts (id, user_id, email_address, display_name,
			imap_host, imap_port, smtp_host, smtp_port, username, password_encrypted,
			use_ssl, sync_enabled, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`,
		account.ID, account.UserID, account.EmailAddress, account.DisplayName,
		account.IMAPHost, account.IMAPPort, account.SMTPHost, account.SMTPPort,
		account.Username, account.PasswordEncrypted,
		account.UseSSL, account.SyncEnabled, account.CreatedAt, account.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.EmailAccount, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, user_id, email_address, display_name,
			imap_host, imap_port, smtp_host, smtp_port, username, password_encrypted,
			use_ssl, last_sync_at, sync_enabled, created_at, updated_at
		 FROM email_accounts WHERE id = $1`, id,
	)
	return scanAccount(row)
}

func (r *PostgresRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*models.EmailAccount, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, user_id, email_address, display_name,
			imap_host, imap_port, smtp_host, smtp_port, username, password_encrypted,
			use_ssl, last_sync_at, sync_enabled, created_at, updated_at
		 FROM email_accounts WHERE user_id = $1`, userID,
	)
	return scanAccount(row)
}

func (r *PostgresRepository) Update(ctx context.Context, account *models.EmailAccount) error {
	account.UpdatedAt = time.Now().UTC()
	_, err := r.pool.Exec(ctx,
		`UPDATE email_accounts SET
			email_address = $2, display_name = $3,
			imap_host = $4, imap_port = $5, smtp_host = $6, smtp_port = $7,
			username = $8, password_encrypted = $9, use_ssl = $10,
			last_sync_at = $11, sync_enabled = $12, updated_at = $13
		 WHERE id = $1`,
		account.ID, account.EmailAddress, account.DisplayName,
		account.IMAPHost, account.IMAPPort, account.SMTPHost, account.SMTPPort,
		account.Username, account.PasswordEncrypted, account.UseSSL,
		account.LastSyncAt, account.SyncEnabled, account.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM email_accounts WHERE id = $1`, id)
	return err
}

func (r *PostgresRepository) ListActive(ctx context.Context) ([]*models.EmailAccount, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, email_address, display_name,
			imap_host, imap_port, smtp_host, smtp_port, username, password_encrypted,
			use_ssl, last_sync_at, sync_enabled, created_at, updated_at
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
		&a.ID, &a.UserID, &a.EmailAddress, &a.DisplayName,
		&a.IMAPHost, &a.IMAPPort, &a.SMTPHost, &a.SMTPPort,
		&a.Username, &a.PasswordEncrypted,
		&a.UseSSL, &a.LastSyncAt, &a.SyncEnabled,
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
