package gdpr

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
)

var (
	ErrExportNotFound = errors.New("gdpr: export request not found")
	ErrTokenNotFound  = errors.New("gdpr: download token not found")
)

// PostgresRepository implements Repository using PostgreSQL.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgreSQL-backed GDPR repository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) CreateExportRequest(ctx context.Context, req *models.GDPRExportRequest) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO gdpr_export_requests (id, user_id, tenant_id, status, requested_at)
		 VALUES ($1, $2, (SELECT tenant_id FROM users WHERE id = $2), $3, $4)`,
		req.ID, req.UserID, req.Status, req.RequestedAt,
	)
	return err
}

func (r *PostgresRepository) GetExportRequest(ctx context.Context, tenantID, id uuid.UUID) (*models.GDPRExportRequest, error) {
	var req models.GDPRExportRequest
	var reviewNote, downloadToken sql.NullString
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, status, requested_at, reviewed_by, reviewed_at,
		        review_note, export_data, download_token, download_expires_at, downloaded_at
		 FROM gdpr_export_requests WHERE id = $1 AND tenant_id = $2`, id, tenantID,
	).Scan(
		&req.ID, &req.UserID, &req.Status, &req.RequestedAt,
		&req.ReviewedBy, &req.ReviewedAt, &reviewNote,
		&req.ExportData, &downloadToken, &req.DownloadExpiresAt, &req.DownloadedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrExportNotFound
	}
	if err != nil {
		return nil, err
	}
	req.ReviewNote = reviewNote.String
	req.DownloadToken = downloadToken.String
	return &req, nil
}

func (r *PostgresRepository) ListExportRequests(ctx context.Context, tenantID, userID uuid.UUID, status string) ([]*models.GDPRExportRequest, error) {
	query := `SELECT id, user_id, status, requested_at, reviewed_by, reviewed_at,
	                 review_note, download_token, download_expires_at, downloaded_at
	          FROM gdpr_export_requests WHERE tenant_id = $1`
	args := []any{tenantID}
	argIdx := 2

	if userID != uuid.Nil {
		query += fmt.Sprintf(" AND user_id = $%d", argIdx)
		args = append(args, userID)
		argIdx++
	}
	if status != "" {
		query += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, status)
	}

	query += " ORDER BY requested_at DESC"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var requests []*models.GDPRExportRequest
	for rows.Next() {
		var req models.GDPRExportRequest
		var reviewNote, downloadToken sql.NullString
		if scanErr := rows.Scan(
			&req.ID, &req.UserID, &req.Status, &req.RequestedAt,
			&req.ReviewedBy, &req.ReviewedAt, &reviewNote,
			&downloadToken, &req.DownloadExpiresAt, &req.DownloadedAt,
		); scanErr != nil {
			return nil, scanErr
		}
		req.ReviewNote = reviewNote.String
		req.DownloadToken = downloadToken.String
		req.TenantID = tenantID
		// ExportData intentionally omitted from list query for performance
		requests = append(requests, &req)
	}
	return requests, rows.Err()
}

func (r *PostgresRepository) UpdateExportStatus(ctx context.Context, tenantID uuid.UUID, req *models.GDPRExportRequest) error {
	result, err := r.pool.Exec(ctx,
		`UPDATE gdpr_export_requests
		 SET status = $1, reviewed_by = $2, reviewed_at = $3, review_note = $4
		 WHERE id = $5 AND tenant_id = $6`,
		req.Status, req.ReviewedBy, req.ReviewedAt, req.ReviewNote, req.ID, tenantID,
	)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrExportNotFound
	}
	return nil
}

func (r *PostgresRepository) StoreExportResult(ctx context.Context, tenantID, id uuid.UUID, data []byte, token string, expiresAt interface{}) error {
	var expiresTime *time.Time
	switch v := expiresAt.(type) {
	case time.Time:
		expiresTime = &v
	case *time.Time:
		expiresTime = v
	}

	result, err := r.pool.Exec(ctx,
		`UPDATE gdpr_export_requests
		 SET status = $1, export_data = $2, download_token = $3, download_expires_at = $4
		 WHERE id = $5 AND tenant_id = $6`,
		models.ExportStatusReady, data, token, expiresTime, id, tenantID,
	)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrExportNotFound
	}
	return nil
}

func (r *PostgresRepository) GetExportByToken(ctx context.Context, tenantID uuid.UUID, token string) (*models.GDPRExportRequest, error) {
	var req models.GDPRExportRequest
	var reviewNote, downloadToken sql.NullString
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, status, requested_at, reviewed_by, reviewed_at,
		        review_note, export_data, download_token, download_expires_at, downloaded_at
		 FROM gdpr_export_requests WHERE download_token = $1 AND tenant_id = $2`, token, tenantID,
	).Scan(
		&req.ID, &req.UserID, &req.Status, &req.RequestedAt,
		&req.ReviewedBy, &req.ReviewedAt, &reviewNote,
		&req.ExportData, &downloadToken, &req.DownloadExpiresAt, &req.DownloadedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrTokenNotFound
	}
	if err != nil {
		return nil, err
	}
	req.ReviewNote = reviewNote.String
	req.DownloadToken = downloadToken.String
	return &req, nil
}

func (r *PostgresRepository) MarkDownloaded(ctx context.Context, tenantID, id uuid.UUID) error {
	now := time.Now()
	result, err := r.pool.Exec(ctx,
		`UPDATE gdpr_export_requests SET status = $1, downloaded_at = $2 WHERE id = $3 AND tenant_id = $4`,
		models.ExportStatusDownloaded, now, id, tenantID,
	)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrExportNotFound
	}
	return nil
}

func (r *PostgresRepository) CreateErasureLog(ctx context.Context, entry *models.GDPRErasureLog) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO gdpr_erasure_log (id, tenant_id, original_user_id, anonymized_label, executed_by, executed_at, modules_affected, confirmation_hash)
		 VALUES ($1, (SELECT tenant_id FROM users WHERE id = $2), $2, $3, $4, $5, $6, $7)`,
		entry.ID, entry.OriginalUserID, entry.AnonymizedLabel,
		entry.ExecutedBy, entry.ExecutedAt, entry.ModulesAffected, entry.ConfirmationHash,
	)
	return err
}

func (r *PostgresRepository) GetNextAnonymizedLabel(ctx context.Context) (string, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM gdpr_erasure_log`,
	).Scan(&count)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("Geloeschter Benutzer #%d", count+1), nil
}

// Ensure PostgresRepository implements Repository at compile time.
var _ Repository = (*PostgresRepository)(nil)
