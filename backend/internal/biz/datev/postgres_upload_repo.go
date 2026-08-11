package datev

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/models"
)

// ErrTenantMissing is returned when UpsertUploadConfig or CreateUploadLog is
// called on a context that carries no tenant — datev_upload_configs and
// datev_upload_log have had tenant_id NOT NULL with no default since
// migration 000115, so a write without one would fail at the constraint
// anyway; this turns that into a typed refusal before the pool is touched.
var ErrTenantMissing = errors.New("datev: upload write without tenant context")

type PostgresUploadRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresUploadRepository(pool *pgxpool.Pool) *PostgresUploadRepository {
	return &PostgresUploadRepository{pool: pool}
}

// tenantForWrite resolves the calling tenant from ctx for an INSERT.
// datev_upload_configs/datev_upload_log carry FORCE ROW LEVEL SECURITY
// (migration 000122); reads rely on that policy alone, but a write must
// supply tenant_id itself to satisfy the NOT NULL constraint and the
// policy's WITH CHECK.
func tenantForWrite(ctx context.Context) (uuid.UUID, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return uuid.Nil, ErrTenantMissing
	}
	return tenantID, nil
}

func (r *PostgresUploadRepository) GetUploadConfig(ctx context.Context, configID uuid.UUID) (*models.DatevUploadConfig, error) {
	var cfg models.DatevUploadConfig
	err := r.pool.QueryRow(ctx,
		`SELECT id, config_id, client_number, auto_upload_enabled, upload_after_export, created_at, updated_at
		FROM datev_upload_configs
		WHERE config_id = $1`,
		configID,
	).Scan(&cfg.ID, &cfg.ConfigID, &cfg.ClientNumber, &cfg.AutoUploadEnabled, &cfg.UploadAfterExport, &cfg.CreatedAt, &cfg.UpdatedAt)

	if err == pgx.ErrNoRows {
		return nil, fmt.Errorf("datev: upload config not found")
	}
	if err != nil {
		return nil, fmt.Errorf("datev: get upload config: %w", err)
	}
	return &cfg, nil
}

func (r *PostgresUploadRepository) UpsertUploadConfig(ctx context.Context, config *models.DatevUploadConfig) error {
	tenantID, err := tenantForWrite(ctx)
	if err != nil {
		return err
	}

	config.UpdatedAt = time.Now().UTC()
	if config.ID == uuid.Nil {
		config.ID = uuid.New()
		config.CreatedAt = config.UpdatedAt
	}

	_, err = r.pool.Exec(ctx,
		`INSERT INTO datev_upload_configs (id, tenant_id, config_id, client_number, auto_upload_enabled, upload_after_export, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (config_id) DO UPDATE SET
			client_number = EXCLUDED.client_number,
			auto_upload_enabled = EXCLUDED.auto_upload_enabled,
			upload_after_export = EXCLUDED.upload_after_export,
			updated_at = EXCLUDED.updated_at`,
		config.ID, tenantID, config.ConfigID, config.ClientNumber, config.AutoUploadEnabled, config.UploadAfterExport,
		config.CreatedAt, config.UpdatedAt,
	)
	return err
}

func (r *PostgresUploadRepository) CreateUploadLog(ctx context.Context, log *models.DatevUploadLog) error {
	tenantID, err := tenantForWrite(ctx)
	if err != nil {
		return err
	}

	if log.ID == uuid.Nil {
		log.ID = uuid.New()
	}

	metadataJSON, err := json.Marshal(log.Metadata)
	if err != nil {
		return fmt.Errorf("datev: marshal upload log metadata: %w", err)
	}

	_, err = r.pool.Exec(ctx,
		`INSERT INTO datev_upload_log (id, tenant_id, config_id, upload_type, status, file_size, document_count, error_message, started_at, completed_at, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		log.ID, tenantID, log.ConfigID, log.UploadType, log.Status,
		log.FileSize, log.DocumentCount, log.ErrorMessage,
		log.StartedAt, log.CompletedAt, metadataJSON,
	)
	return err
}

func (r *PostgresUploadRepository) UpdateUploadLog(ctx context.Context, log *models.DatevUploadLog) error {
	metadataJSON, err := json.Marshal(log.Metadata)
	if err != nil {
		return fmt.Errorf("datev: marshal upload log metadata: %w", err)
	}

	_, err = r.pool.Exec(ctx,
		`UPDATE datev_upload_log SET
			status = $1, file_size = $2, document_count = $3,
			error_message = $4, completed_at = $5, metadata = $6
		WHERE id = $7`,
		log.Status, log.FileSize, log.DocumentCount,
		log.ErrorMessage, log.CompletedAt, metadataJSON,
		log.ID,
	)
	return err
}

func (r *PostgresUploadRepository) ListUploadLogs(ctx context.Context, configID uuid.UUID, limit int) ([]models.DatevUploadLog, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := r.pool.Query(ctx,
		`SELECT id, config_id, upload_type, status, file_size, document_count, error_message, started_at, completed_at, metadata
		FROM datev_upload_log
		WHERE config_id = $1
		ORDER BY started_at DESC
		LIMIT $2`,
		configID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("datev: list upload logs: %w", err)
	}
	defer rows.Close()

	var logs []models.DatevUploadLog
	for rows.Next() {
		var log models.DatevUploadLog
		var metadataJSON []byte
		if err := rows.Scan(
			&log.ID, &log.ConfigID, &log.UploadType, &log.Status,
			&log.FileSize, &log.DocumentCount, &log.ErrorMessage,
			&log.StartedAt, &log.CompletedAt, &metadataJSON,
		); err != nil {
			return nil, fmt.Errorf("datev: scan upload log row: %w", err)
		}
		if err := json.Unmarshal(metadataJSON, &log.Metadata); err != nil {
			return nil, fmt.Errorf("datev: unmarshal upload log metadata: %w", err)
		}
		logs = append(logs, log)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("datev: upload log rows: %w", err)
	}
	return logs, nil
}
