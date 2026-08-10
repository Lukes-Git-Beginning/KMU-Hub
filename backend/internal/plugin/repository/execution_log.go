package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
)

// ExecutionLogRepository handles plugin execution log persistence
type ExecutionLogRepository struct {
	pool *pgxpool.Pool
}

// NewExecutionLogRepository creates a new execution log repository
func NewExecutionLogRepository(pool *pgxpool.Pool) *ExecutionLogRepository {
	return &ExecutionLogRepository{pool: pool}
}

// Create inserts a new execution log entry. tenant_id is derived from the
// owning plugin installation via subquery so RLS (mig 000126) is satisfied
// without changing the callsite signature. If installationID does not resolve
// under the subquery, the INSERT affects zero rows and Create returns
// ErrInstallationNotFound instead of silently no-op'ing (same shape as
// KVStoreRepository.Set).
func (r *ExecutionLogRepository) Create(ctx context.Context, log *models.PluginExecutionLog) error {
	tag, err := r.pool.Exec(ctx, `
		INSERT INTO plugin_execution_log (id, installation_id, tenant_id, hook_type, module, entity_type, entity_id, duration_ms, status, error_message, request_data, response_data, created_at)
		SELECT $1, $2, pi.tenant_id, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
		FROM plugin_installations pi
		WHERE pi.id = $2`,
		log.ID, log.InstallationID, log.HookType, log.Module, log.EntityType, log.EntityID,
		log.DurationMs, log.Status, log.ErrorMessage, log.RequestData, log.ResponseData, log.CreatedAt,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrInstallationNotFound
	}
	return nil
}

// List retrieves execution logs with optional filters
func (r *ExecutionLogRepository) List(ctx context.Context, tenantID uuid.UUID, installationID *uuid.UUID, limit int) ([]*models.PluginExecutionLog, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}

	query := `
		SELECT el.id, el.installation_id, el.hook_type, el.module, el.entity_type, el.entity_id,
		       el.duration_ms, el.status, el.error_message, el.created_at
		FROM plugin_execution_log el
		JOIN plugin_installations i ON i.id = el.installation_id
		WHERE i.tenant_id = $1`
	args := []any{tenantID}

	if installationID != nil {
		query += ` AND el.installation_id = $2`
		args = append(args, *installationID)
	}
	query += ` ORDER BY el.created_at DESC LIMIT ` + intToStr(limit)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*models.PluginExecutionLog
	for rows.Next() {
		l := &models.PluginExecutionLog{}
		if scanErr := rows.Scan(&l.ID, &l.InstallationID, &l.HookType, &l.Module, &l.EntityType, &l.EntityID,
			&l.DurationMs, &l.Status, &l.ErrorMessage, &l.CreatedAt,
		); scanErr != nil {
			return nil, scanErr
		}
		logs = append(logs, l)
	}
	return logs, nil
}

func intToStr(n int) string {
	if n <= 0 {
		return "100"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}
