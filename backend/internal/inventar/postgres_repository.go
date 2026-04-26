package inventar

import (
	"context"
	"errors"
	"fmt"
	"strings"

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

// ============================================================================
// Items
// ============================================================================

func (r *PostgresRepository) CreateItem(ctx context.Context, item *Item) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO inventory_items
		    (id, tenant_id, name, sku, barcode, quantity, min_quantity, unit, location, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		item.ID, item.TenantID, item.Name, item.SKU, item.Barcode,
		item.Quantity, item.MinQuantity, item.Unit, item.Location,
		item.CreatedAt, item.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) UpdateItem(ctx context.Context, item *Item) error {
	ct, err := r.pool.Exec(ctx,
		`UPDATE inventory_items
		 SET name = $1, sku = $2, barcode = $3, quantity = $4, min_quantity = $5,
		     unit = $6, location = $7, updated_at = $8
		 WHERE id = $9 AND tenant_id = $10 AND deleted_at IS NULL`,
		item.Name, item.SKU, item.Barcode, item.Quantity, item.MinQuantity,
		item.Unit, item.Location, item.UpdatedAt, item.ID, item.TenantID,
	)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrItemNotFound
	}
	return nil
}

func (r *PostgresRepository) SoftDeleteItem(ctx context.Context, tenantID, itemID uuid.UUID) error {
	ct, err := r.pool.Exec(ctx,
		`UPDATE inventory_items SET deleted_at = NOW() WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL`,
		itemID, tenantID,
	)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrItemNotFound
	}
	return nil
}

func (r *PostgresRepository) GetItem(ctx context.Context, tenantID, itemID uuid.UUID) (*Item, error) {
	var item Item
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, name, sku, barcode, quantity, min_quantity, unit, location,
		        created_at, updated_at, deleted_at
		 FROM inventory_items WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL`,
		itemID, tenantID,
	).Scan(
		&item.ID, &item.TenantID, &item.Name, &item.SKU, &item.Barcode,
		&item.Quantity, &item.MinQuantity, &item.Unit, &item.Location,
		&item.CreatedAt, &item.UpdatedAt, &item.DeletedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrItemNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get inventory item: %w", err)
	}
	return &item, nil
}

func (r *PostgresRepository) ListItems(ctx context.Context, tenantID uuid.UUID, filter ListItemsFilter, offset, limit int) ([]*Item, int, error) {
	var conditions []string
	var args []any
	argNum := 1

	conditions = append(conditions, fmt.Sprintf("tenant_id = $%d", argNum))
	args = append(args, tenantID)
	argNum++

	conditions = append(conditions, "deleted_at IS NULL")

	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(LOWER(name) LIKE $%d OR LOWER(sku) LIKE $%d)", argNum, argNum))
		args = append(args, "%"+strings.ToLower(filter.Search)+"%")
		argNum++
	}

	if filter.Location != nil {
		conditions = append(conditions, fmt.Sprintf("location = $%d", argNum))
		args = append(args, *filter.Location)
		argNum++
	}

	if filter.LowStock {
		conditions = append(conditions, "quantity <= min_quantity")
	}

	whereClause := "WHERE " + strings.Join(conditions, " AND ")

	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM inventory_items %s", whereClause)
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count inventory items: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT id, tenant_id, name, sku, barcode, quantity, min_quantity, unit, location,
		       created_at, updated_at, deleted_at
		FROM inventory_items %s
		ORDER BY name ASC
		LIMIT $%d OFFSET $%d
	`, whereClause, argNum, argNum+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list inventory items: %w", err)
	}
	defer rows.Close()

	var items []*Item
	for rows.Next() {
		item, scanErr := r.scanItemFromRows(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		items = append(items, item)
	}

	return items, total, rows.Err()
}

func (r *PostgresRepository) SKUExists(ctx context.Context, tenantID uuid.UUID, sku string, excludeID *uuid.UUID) (bool, error) {
	var exists bool
	if excludeID != nil {
		err := r.pool.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM inventory_items WHERE tenant_id = $1 AND sku = $2 AND id <> $3 AND deleted_at IS NULL)`,
			tenantID, sku, *excludeID,
		).Scan(&exists)
		return exists, err
	}
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM inventory_items WHERE tenant_id = $1 AND sku = $2 AND deleted_at IS NULL)`,
		tenantID, sku,
	).Scan(&exists)
	return exists, err
}

// ============================================================================
// Movements
// ============================================================================

func (r *PostgresRepository) CreateMovement(ctx context.Context, movement *Movement) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO inventory_movements
		    (id, tenant_id, item_id, movement_type, quantity, performed_by, reason, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		movement.ID, movement.TenantID, movement.ItemID, movement.MovementType,
		movement.Quantity, movement.PerformedBy, movement.Reason, movement.CreatedAt,
	)
	return err
}

func (r *PostgresRepository) GetMovement(ctx context.Context, tenantID, movementID uuid.UUID) (*Movement, error) {
	var m Movement
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, item_id, movement_type, quantity, performed_by, reason, created_at
		 FROM inventory_movements WHERE id = $1 AND tenant_id = $2`,
		movementID, tenantID,
	).Scan(&m.ID, &m.TenantID, &m.ItemID, &m.MovementType, &m.Quantity, &m.PerformedBy, &m.Reason, &m.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrMovementNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get inventory movement: %w", err)
	}
	return &m, nil
}

func (r *PostgresRepository) ListMovements(ctx context.Context, tenantID, itemID uuid.UUID, offset, limit int) ([]*Movement, int, error) {
	var total int
	if err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM inventory_movements WHERE tenant_id = $1 AND item_id = $2`,
		tenantID, itemID,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count movements: %w", err)
	}

	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, item_id, movement_type, quantity, performed_by, reason, created_at
		 FROM inventory_movements
		 WHERE tenant_id = $1 AND item_id = $2
		 ORDER BY created_at DESC
		 LIMIT $3 OFFSET $4`,
		tenantID, itemID, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list movements: %w", err)
	}
	defer rows.Close()

	var movements []*Movement
	for rows.Next() {
		m, scanErr := r.scanMovementFromRows(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		movements = append(movements, m)
	}

	return movements, total, rows.Err()
}

// ============================================================================
// Warnings
// ============================================================================

func (r *PostgresRepository) CreateWarning(ctx context.Context, warning *Warning) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO stock_warnings
		    (id, tenant_id, item_id, threshold, current_quantity, status, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		warning.ID, warning.TenantID, warning.ItemID, warning.Threshold,
		warning.CurrentQuantity, warning.Status, warning.CreatedAt,
	)
	return err
}

func (r *PostgresRepository) UpdateWarning(ctx context.Context, warning *Warning) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE stock_warnings
		 SET status = $1, acknowledged_at = $2, acknowledged_by = $3
		 WHERE id = $4 AND tenant_id = $5`,
		warning.Status, warning.AcknowledgedAt, warning.AcknowledgedBy,
		warning.ID, warning.TenantID,
	)
	return err
}

func (r *PostgresRepository) GetWarning(ctx context.Context, tenantID, warningID uuid.UUID) (*Warning, error) {
	var w Warning
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, item_id, threshold, current_quantity, status,
		        created_at, acknowledged_at, acknowledged_by
		 FROM stock_warnings WHERE id = $1 AND tenant_id = $2`,
		warningID, tenantID,
	).Scan(
		&w.ID, &w.TenantID, &w.ItemID, &w.Threshold, &w.CurrentQuantity,
		&w.Status, &w.CreatedAt, &w.AcknowledgedAt, &w.AcknowledgedBy,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrWarningNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get stock warning: %w", err)
	}
	return &w, nil
}

func (r *PostgresRepository) GetActiveWarningForItem(ctx context.Context, tenantID, itemID uuid.UUID) (*Warning, error) {
	var w Warning
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, item_id, threshold, current_quantity, status,
		        created_at, acknowledged_at, acknowledged_by
		 FROM stock_warnings WHERE tenant_id = $1 AND item_id = $2 AND status = 'active'
		 LIMIT 1`,
		tenantID, itemID,
	).Scan(
		&w.ID, &w.TenantID, &w.ItemID, &w.Threshold, &w.CurrentQuantity,
		&w.Status, &w.CreatedAt, &w.AcknowledgedAt, &w.AcknowledgedBy,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrWarningNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get active stock warning: %w", err)
	}
	return &w, nil
}

func (r *PostgresRepository) ListWarnings(ctx context.Context, tenantID uuid.UUID, status *WarningStatus, offset, limit int) ([]*Warning, int, error) {
	var conditions []string
	var args []any
	argNum := 1

	conditions = append(conditions, fmt.Sprintf("tenant_id = $%d", argNum))
	args = append(args, tenantID)
	argNum++

	if status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argNum))
		args = append(args, *status)
		argNum++
	}

	whereClause := "WHERE " + strings.Join(conditions, " AND ")

	var total int
	if err := r.pool.QueryRow(ctx, fmt.Sprintf("SELECT COUNT(*) FROM stock_warnings %s", whereClause), args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count stock warnings: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT id, tenant_id, item_id, threshold, current_quantity, status,
		       created_at, acknowledged_at, acknowledged_by
		FROM stock_warnings %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argNum, argNum+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list stock warnings: %w", err)
	}
	defer rows.Close()

	var warnings []*Warning
	for rows.Next() {
		w, scanErr := r.scanWarningFromRows(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		warnings = append(warnings, w)
	}

	return warnings, total, rows.Err()
}

// ============================================================================
// Scan helpers
// ============================================================================

func (r *PostgresRepository) scanItemFromRows(rows pgx.Rows) (*Item, error) {
	var item Item
	err := rows.Scan(
		&item.ID, &item.TenantID, &item.Name, &item.SKU, &item.Barcode,
		&item.Quantity, &item.MinQuantity, &item.Unit, &item.Location,
		&item.CreatedAt, &item.UpdatedAt, &item.DeletedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan inventory item row: %w", err)
	}
	return &item, nil
}

func (r *PostgresRepository) scanMovementFromRows(rows pgx.Rows) (*Movement, error) {
	var m Movement
	err := rows.Scan(&m.ID, &m.TenantID, &m.ItemID, &m.MovementType, &m.Quantity, &m.PerformedBy, &m.Reason, &m.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("scan inventory movement row: %w", err)
	}
	return &m, nil
}

func (r *PostgresRepository) scanWarningFromRows(rows pgx.Rows) (*Warning, error) {
	var w Warning
	err := rows.Scan(
		&w.ID, &w.TenantID, &w.ItemID, &w.Threshold, &w.CurrentQuantity,
		&w.Status, &w.CreatedAt, &w.AcknowledgedAt, &w.AcknowledgedBy,
	)
	if err != nil {
		return nil, fmt.Errorf("scan stock warning row: %w", err)
	}
	return &w, nil
}

// compile-time interface check
var _ Repository = (*PostgresRepository)(nil)
