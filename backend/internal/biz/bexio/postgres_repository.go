package bexio

import (
	"context"
	"encoding/json"
	"fmt"
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

// NewPostgresRepository creates a new PostgreSQL Bexio repository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// --- Sync Config ---

func (r *PostgresRepository) GetSyncConfig(ctx context.Context, configID uuid.UUID) (*models.BexioSyncConfig, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, config_id, contact_sync_enabled, contact_sync_interval_minutes,
			invoice_push_enabled, quote_push_enabled,
			payment_poll_enabled, payment_poll_interval_minutes,
			last_contact_sync_at, last_payment_poll_at,
			created_at, updated_at
		FROM bexio_sync_configs
		WHERE config_id = $1`,
		configID,
	)
	return r.scanSyncConfig(row)
}

func (r *PostgresRepository) UpsertSyncConfig(ctx context.Context, config *models.BexioSyncConfig) error {
	config.UpdatedAt = time.Now().UTC()
	if config.ID == uuid.Nil {
		config.ID = uuid.New()
		config.CreatedAt = config.UpdatedAt
	}

	_, err := r.pool.Exec(ctx,
		`INSERT INTO bexio_sync_configs (
			id, config_id, contact_sync_enabled, contact_sync_interval_minutes,
			invoice_push_enabled, quote_push_enabled,
			payment_poll_enabled, payment_poll_interval_minutes,
			last_contact_sync_at, last_payment_poll_at,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (config_id) DO UPDATE SET
			contact_sync_enabled = EXCLUDED.contact_sync_enabled,
			contact_sync_interval_minutes = EXCLUDED.contact_sync_interval_minutes,
			invoice_push_enabled = EXCLUDED.invoice_push_enabled,
			quote_push_enabled = EXCLUDED.quote_push_enabled,
			payment_poll_enabled = EXCLUDED.payment_poll_enabled,
			payment_poll_interval_minutes = EXCLUDED.payment_poll_interval_minutes,
			updated_at = EXCLUDED.updated_at`,
		config.ID, config.ConfigID, config.ContactSyncEnabled, config.ContactSyncIntervalMin,
		config.InvoicePushEnabled, config.QuotePushEnabled,
		config.PaymentPollEnabled, config.PaymentPollIntervalMin,
		config.LastContactSyncAt, config.LastPaymentPollAt,
		config.CreatedAt, config.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) UpdateLastSyncTime(ctx context.Context, configID uuid.UUID, syncType string, syncedAt time.Time) error {
	var column string
	switch syncType {
	case "contact_full", "contact_delta":
		column = "last_contact_sync_at"
	case "payment_poll":
		column = "last_payment_poll_at"
	default:
		return fmt.Errorf("bexio: unknown sync type %q for last sync time update", syncType)
	}

	_, err := r.pool.Exec(ctx,
		fmt.Sprintf(`UPDATE bexio_sync_configs SET %s = $1, updated_at = $2 WHERE config_id = $3`, column),
		syncedAt, time.Now().UTC(), configID,
	)
	return err
}

func (r *PostgresRepository) scanSyncConfig(row pgx.Row) (*models.BexioSyncConfig, error) {
	var sc models.BexioSyncConfig
	err := row.Scan(
		&sc.ID, &sc.ConfigID, &sc.ContactSyncEnabled, &sc.ContactSyncIntervalMin,
		&sc.InvoicePushEnabled, &sc.QuotePushEnabled,
		&sc.PaymentPollEnabled, &sc.PaymentPollIntervalMin,
		&sc.LastContactSyncAt, &sc.LastPaymentPollAt,
		&sc.CreatedAt, &sc.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, ErrConfigNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("bexio: scan sync config: %w", err)
	}
	return &sc, nil
}

// --- Entity Mappings ---

func (r *PostgresRepository) GetEntityMapping(ctx context.Context, configID uuid.UUID, entityType string, kmuhubID uuid.UUID) (*models.BexioEntityMapping, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, config_id, entity_type, kmuhub_id, bexio_id,
			last_synced_at, bexio_updated_at, kmuhub_updated_at, sync_direction,
			created_at, updated_at
		FROM bexio_entity_mappings
		WHERE config_id = $1 AND entity_type = $2 AND kmuhub_id = $3`,
		configID, entityType, kmuhubID,
	)
	return r.scanEntityMapping(row)
}

func (r *PostgresRepository) GetEntityMappingByBexioID(ctx context.Context, configID uuid.UUID, entityType string, bexioID int) (*models.BexioEntityMapping, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, config_id, entity_type, kmuhub_id, bexio_id,
			last_synced_at, bexio_updated_at, kmuhub_updated_at, sync_direction,
			created_at, updated_at
		FROM bexio_entity_mappings
		WHERE config_id = $1 AND entity_type = $2 AND bexio_id = $3`,
		configID, entityType, bexioID,
	)
	return r.scanEntityMapping(row)
}

func (r *PostgresRepository) UpsertEntityMapping(ctx context.Context, mapping *models.BexioEntityMapping) error {
	mapping.UpdatedAt = time.Now().UTC()
	if mapping.ID == uuid.Nil {
		mapping.ID = uuid.New()
		mapping.CreatedAt = mapping.UpdatedAt
	}

	_, err := r.pool.Exec(ctx,
		`INSERT INTO bexio_entity_mappings (
			id, config_id, entity_type, kmuhub_id, bexio_id,
			last_synced_at, bexio_updated_at, kmuhub_updated_at, sync_direction,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (config_id, entity_type, kmuhub_id) DO UPDATE SET
			bexio_id = EXCLUDED.bexio_id,
			last_synced_at = EXCLUDED.last_synced_at,
			bexio_updated_at = EXCLUDED.bexio_updated_at,
			kmuhub_updated_at = EXCLUDED.kmuhub_updated_at,
			sync_direction = EXCLUDED.sync_direction,
			updated_at = EXCLUDED.updated_at`,
		mapping.ID, mapping.ConfigID, mapping.EntityType, mapping.KmuhubID, mapping.BexioID,
		mapping.LastSyncedAt, mapping.BexioUpdatedAt, mapping.KmuhubUpdatedAt, mapping.SyncDirection,
		mapping.CreatedAt, mapping.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) ListEntityMappings(ctx context.Context, configID uuid.UUID, entityType string) ([]models.BexioEntityMapping, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, config_id, entity_type, kmuhub_id, bexio_id,
			last_synced_at, bexio_updated_at, kmuhub_updated_at, sync_direction,
			created_at, updated_at
		FROM bexio_entity_mappings
		WHERE config_id = $1 AND entity_type = $2
		ORDER BY created_at`,
		configID, entityType,
	)
	if err != nil {
		return nil, fmt.Errorf("bexio: list entity mappings: %w", err)
	}
	defer rows.Close()

	var mappings []models.BexioEntityMapping
	for rows.Next() {
		var m models.BexioEntityMapping
		if err := rows.Scan(
			&m.ID, &m.ConfigID, &m.EntityType, &m.KmuhubID, &m.BexioID,
			&m.LastSyncedAt, &m.BexioUpdatedAt, &m.KmuhubUpdatedAt, &m.SyncDirection,
			&m.CreatedAt, &m.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("bexio: scan entity mapping row: %w", err)
		}
		mappings = append(mappings, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("bexio: entity mapping rows: %w", err)
	}
	return mappings, nil
}

func (r *PostgresRepository) DeleteEntityMapping(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM bexio_entity_mappings WHERE id = $1`, id)
	return err
}

func (r *PostgresRepository) scanEntityMapping(row pgx.Row) (*models.BexioEntityMapping, error) {
	var m models.BexioEntityMapping
	err := row.Scan(
		&m.ID, &m.ConfigID, &m.EntityType, &m.KmuhubID, &m.BexioID,
		&m.LastSyncedAt, &m.BexioUpdatedAt, &m.KmuhubUpdatedAt, &m.SyncDirection,
		&m.CreatedAt, &m.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, ErrMappingNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("bexio: scan entity mapping: %w", err)
	}
	return &m, nil
}

// --- Field Mappings ---

func (r *PostgresRepository) GetFieldMappings(ctx context.Context, configID uuid.UUID, entityType string) (*models.BexioFieldMapping, error) {
	var fm models.BexioFieldMapping
	var mappingsJSON []byte
	err := r.pool.QueryRow(ctx,
		`SELECT id, config_id, entity_type, mappings, created_at, updated_at
		FROM bexio_field_mappings
		WHERE config_id = $1 AND entity_type = $2`,
		configID, entityType,
	).Scan(&fm.ID, &fm.ConfigID, &fm.EntityType, &mappingsJSON, &fm.CreatedAt, &fm.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("bexio: get field mappings: %w", err)
	}

	if err := json.Unmarshal(mappingsJSON, &fm.Mappings); err != nil {
		return nil, fmt.Errorf("bexio: unmarshal field mappings: %w", err)
	}
	return &fm, nil
}

func (r *PostgresRepository) UpsertFieldMappings(ctx context.Context, mapping *models.BexioFieldMapping) error {
	mapping.UpdatedAt = time.Now().UTC()
	if mapping.ID == uuid.Nil {
		mapping.ID = uuid.New()
		mapping.CreatedAt = mapping.UpdatedAt
	}

	mappingsJSON, err := json.Marshal(mapping.Mappings)
	if err != nil {
		return fmt.Errorf("bexio: marshal field mappings: %w", err)
	}

	_, err = r.pool.Exec(ctx,
		`INSERT INTO bexio_field_mappings (id, config_id, entity_type, mappings, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (config_id, entity_type) DO UPDATE SET
			mappings = EXCLUDED.mappings,
			updated_at = EXCLUDED.updated_at`,
		mapping.ID, mapping.ConfigID, mapping.EntityType, mappingsJSON,
		mapping.CreatedAt, mapping.UpdatedAt,
	)
	return err
}

// --- Sync Log ---

func (r *PostgresRepository) CreateSyncLog(ctx context.Context, log *models.BexioSyncLog) error {
	if log.ID == uuid.Nil {
		log.ID = uuid.New()
	}

	metadataJSON, err := json.Marshal(log.Metadata)
	if err != nil {
		return fmt.Errorf("bexio: marshal sync log metadata: %w", err)
	}

	_, err = r.pool.Exec(ctx,
		`INSERT INTO bexio_sync_log (
			id, config_id, sync_type, status,
			items_processed, items_created, items_updated, items_failed,
			error_message, started_at, completed_at, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		log.ID, log.ConfigID, log.SyncType, log.Status,
		log.ItemsProcessed, log.ItemsCreated, log.ItemsUpdated, log.ItemsFailed,
		log.ErrorMessage, log.StartedAt, log.CompletedAt, metadataJSON,
	)
	return err
}

func (r *PostgresRepository) UpdateSyncLog(ctx context.Context, log *models.BexioSyncLog) error {
	metadataJSON, err := json.Marshal(log.Metadata)
	if err != nil {
		return fmt.Errorf("bexio: marshal sync log metadata: %w", err)
	}

	_, err = r.pool.Exec(ctx,
		`UPDATE bexio_sync_log SET
			status = $1,
			items_processed = $2,
			items_created = $3,
			items_updated = $4,
			items_failed = $5,
			error_message = $6,
			completed_at = $7,
			metadata = $8
		WHERE id = $9`,
		log.Status,
		log.ItemsProcessed, log.ItemsCreated, log.ItemsUpdated, log.ItemsFailed,
		log.ErrorMessage, log.CompletedAt, metadataJSON,
		log.ID,
	)
	return err
}

func (r *PostgresRepository) ListSyncLogs(ctx context.Context, configID uuid.UUID, limit int) ([]models.BexioSyncLog, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := r.pool.Query(ctx,
		`SELECT id, config_id, sync_type, status,
			items_processed, items_created, items_updated, items_failed,
			error_message, started_at, completed_at, metadata
		FROM bexio_sync_log
		WHERE config_id = $1
		ORDER BY started_at DESC
		LIMIT $2`,
		configID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("bexio: list sync logs: %w", err)
	}
	defer rows.Close()

	var logs []models.BexioSyncLog
	for rows.Next() {
		log, err := r.scanSyncLogFromRows(rows)
		if err != nil {
			return nil, err
		}
		logs = append(logs, *log)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("bexio: sync log rows: %w", err)
	}
	return logs, nil
}

func (r *PostgresRepository) GetLatestSyncLog(ctx context.Context, configID uuid.UUID, syncType string) (*models.BexioSyncLog, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, config_id, sync_type, status,
			items_processed, items_created, items_updated, items_failed,
			error_message, started_at, completed_at, metadata
		FROM bexio_sync_log
		WHERE config_id = $1 AND sync_type = $2
		ORDER BY started_at DESC
		LIMIT 1`,
		configID, syncType,
	)

	var log models.BexioSyncLog
	var metadataJSON []byte
	err := row.Scan(
		&log.ID, &log.ConfigID, &log.SyncType, &log.Status,
		&log.ItemsProcessed, &log.ItemsCreated, &log.ItemsUpdated, &log.ItemsFailed,
		&log.ErrorMessage, &log.StartedAt, &log.CompletedAt, &metadataJSON,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("bexio: get latest sync log: %w", err)
	}

	if err := json.Unmarshal(metadataJSON, &log.Metadata); err != nil {
		return nil, fmt.Errorf("bexio: unmarshal sync log metadata: %w", err)
	}
	return &log, nil
}

func (r *PostgresRepository) scanSyncLogFromRows(rows pgx.Rows) (*models.BexioSyncLog, error) {
	var log models.BexioSyncLog
	var metadataJSON []byte
	if err := rows.Scan(
		&log.ID, &log.ConfigID, &log.SyncType, &log.Status,
		&log.ItemsProcessed, &log.ItemsCreated, &log.ItemsUpdated, &log.ItemsFailed,
		&log.ErrorMessage, &log.StartedAt, &log.CompletedAt, &metadataJSON,
	); err != nil {
		return nil, fmt.Errorf("bexio: scan sync log row: %w", err)
	}
	if err := json.Unmarshal(metadataJSON, &log.Metadata); err != nil {
		return nil, fmt.Errorf("bexio: unmarshal sync log metadata: %w", err)
	}
	return &log, nil
}
