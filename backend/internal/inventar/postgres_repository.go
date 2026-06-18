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
		    (id, tenant_id, name, sku, barcode, quantity, min_quantity, unit, location, location_id,
		     category, price, currency, batch_number, serial_numbers, linked_purchase_order,
		     created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)`,
		item.ID, item.TenantID, item.Name, item.SKU, item.Barcode,
		item.Quantity, item.MinQuantity, item.Unit, item.Location, item.LocationID,
		item.Category, item.Price, item.Currency, item.BatchNumber, item.SerialNumbers, item.LinkedPurchaseOrder,
		item.CreatedAt, item.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) UpdateItem(ctx context.Context, item *Item) error {
	ct, err := r.pool.Exec(ctx,
		`UPDATE inventory_items
		 SET name=$1, sku=$2, barcode=$3, quantity=$4, min_quantity=$5,
		     unit=$6, location=$7, location_id=$8,
		     category=$9, price=$10, currency=$11, batch_number=$12, serial_numbers=$13,
		     linked_purchase_order=$14, updated_at=$15
		 WHERE id=$16 AND tenant_id=$17 AND deleted_at IS NULL`,
		item.Name, item.SKU, item.Barcode, item.Quantity, item.MinQuantity,
		item.Unit, item.Location, item.LocationID,
		item.Category, item.Price, item.Currency, item.BatchNumber, item.SerialNumbers,
		item.LinkedPurchaseOrder, item.UpdatedAt, item.ID, item.TenantID,
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
		`SELECT id, tenant_id, name, sku, barcode, quantity, min_quantity, unit, location, location_id,
		        category, price, currency, batch_number, serial_numbers, linked_purchase_order,
		        created_at, updated_at, deleted_at
		 FROM inventory_items WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL`,
		itemID, tenantID,
	).Scan(
		&item.ID, &item.TenantID, &item.Name, &item.SKU, &item.Barcode,
		&item.Quantity, &item.MinQuantity, &item.Unit, &item.Location, &item.LocationID,
		&item.Category, &item.Price, &item.Currency, &item.BatchNumber, &item.SerialNumbers, &item.LinkedPurchaseOrder,
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
		SELECT id, tenant_id, name, sku, barcode, quantity, min_quantity, unit, location, location_id,
		       category, price, currency, batch_number, serial_numbers, linked_purchase_order,
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
		    (id, tenant_id, item_id, movement_type, quantity, performed_by, reason,
		     reference, location_from, location_to, notes, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		movement.ID, movement.TenantID, movement.ItemID, movement.MovementType,
		movement.Quantity, movement.PerformedBy, movement.Reason,
		movement.Reference, movement.LocationFrom, movement.LocationTo, movement.Notes,
		movement.CreatedAt,
	)
	return err
}

func (r *PostgresRepository) GetMovement(ctx context.Context, tenantID, movementID uuid.UUID) (*Movement, error) {
	var m Movement
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, item_id, movement_type, quantity, performed_by, reason,
		        reference, location_from, location_to, notes, created_at
		 FROM inventory_movements WHERE id = $1 AND tenant_id = $2`,
		movementID, tenantID,
	).Scan(
		&m.ID, &m.TenantID, &m.ItemID, &m.MovementType, &m.Quantity, &m.PerformedBy, &m.Reason,
		&m.Reference, &m.LocationFrom, &m.LocationTo, &m.Notes, &m.CreatedAt,
	)
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
		`SELECT id, tenant_id, item_id, movement_type, quantity, performed_by, reason,
		        reference, location_from, location_to, notes, created_at
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
// Locations
// ============================================================================

func (r *PostgresRepository) CreateLocation(ctx context.Context, loc *InventoryLocation) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO inventory_locations (id, tenant_id, name, address, type, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		loc.ID, loc.TenantID, loc.Name, loc.Address, loc.Type, loc.CreatedAt, loc.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) UpdateLocation(ctx context.Context, loc *InventoryLocation) error {
	ct, err := r.pool.Exec(ctx,
		`UPDATE inventory_locations SET name=$1, address=$2, type=$3, updated_at=$4
		 WHERE id=$5 AND tenant_id=$6 AND deleted_at IS NULL`,
		loc.Name, loc.Address, loc.Type, loc.UpdatedAt, loc.ID, loc.TenantID,
	)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrLocationNotFound
	}
	return nil
}

func (r *PostgresRepository) SoftDeleteLocation(ctx context.Context, tenantID, locID uuid.UUID) error {
	ct, err := r.pool.Exec(ctx,
		`UPDATE inventory_locations SET deleted_at=NOW() WHERE id=$1 AND tenant_id=$2 AND deleted_at IS NULL`,
		locID, tenantID,
	)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrLocationNotFound
	}
	return nil
}

func (r *PostgresRepository) GetLocation(ctx context.Context, tenantID, locID uuid.UUID) (*InventoryLocation, error) {
	var loc InventoryLocation
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, name, address, type, created_at, updated_at, deleted_at
		 FROM inventory_locations WHERE id=$1 AND tenant_id=$2 AND deleted_at IS NULL`,
		locID, tenantID,
	).Scan(&loc.ID, &loc.TenantID, &loc.Name, &loc.Address, &loc.Type, &loc.CreatedAt, &loc.UpdatedAt, &loc.DeletedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrLocationNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get inventory location: %w", err)
	}
	return &loc, nil
}

func (r *PostgresRepository) ListLocations(ctx context.Context, tenantID uuid.UUID, offset, limit int) ([]*InventoryLocation, int, error) {
	var total int
	if err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM inventory_locations WHERE tenant_id=$1 AND deleted_at IS NULL`,
		tenantID,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count locations: %w", err)
	}

	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, name, address, type, created_at, updated_at, deleted_at
		 FROM inventory_locations WHERE tenant_id=$1 AND deleted_at IS NULL
		 ORDER BY name ASC LIMIT $2 OFFSET $3`,
		tenantID, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list locations: %w", err)
	}
	defer rows.Close()

	var locs []*InventoryLocation
	for rows.Next() {
		var loc InventoryLocation
		if scanErr := rows.Scan(&loc.ID, &loc.TenantID, &loc.Name, &loc.Address, &loc.Type, &loc.CreatedAt, &loc.UpdatedAt, &loc.DeletedAt); scanErr != nil {
			return nil, 0, fmt.Errorf("scan location row: %w", scanErr)
		}
		locs = append(locs, &loc)
	}
	return locs, total, rows.Err()
}

// ============================================================================
// Inventur Sessions
// ============================================================================

func (r *PostgresRepository) CreateInventurSession(ctx context.Context, session *InventurSession) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO inventur_sessions (id, tenant_id, name, date, status, location_id, created_by, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		session.ID, session.TenantID, session.Name, session.Date, session.Status,
		session.LocationID, session.CreatedBy, session.CreatedAt, session.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) UpdateInventurSession(ctx context.Context, session *InventurSession) error {
	ct, err := r.pool.Exec(ctx,
		`UPDATE inventur_sessions SET name=$1, status=$2, updated_at=$3
		 WHERE id=$4 AND tenant_id=$5`,
		session.Name, session.Status, session.UpdatedAt, session.ID, session.TenantID,
	)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrInventurSessionNotFound
	}
	return nil
}

func (r *PostgresRepository) DeleteInventurSession(ctx context.Context, tenantID, sessionID uuid.UUID) error {
	ct, err := r.pool.Exec(ctx,
		`DELETE FROM inventur_sessions WHERE id=$1 AND tenant_id=$2`,
		sessionID, tenantID,
	)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrInventurSessionNotFound
	}
	return nil
}

func (r *PostgresRepository) GetInventurSession(ctx context.Context, tenantID, sessionID uuid.UUID) (*InventurSession, error) {
	var s InventurSession
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, name, date, status, location_id, created_by, created_at, updated_at
		 FROM inventur_sessions WHERE id=$1 AND tenant_id=$2`,
		sessionID, tenantID,
	).Scan(&s.ID, &s.TenantID, &s.Name, &s.Date, &s.Status, &s.LocationID, &s.CreatedBy, &s.CreatedAt, &s.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrInventurSessionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get inventur session: %w", err)
	}

	counts, cErr := r.ListInventurCounts(ctx, s.ID)
	if cErr != nil {
		return nil, cErr
	}
	for _, c := range counts {
		s.Counts = append(s.Counts, *c)
	}
	return &s, nil
}

func (r *PostgresRepository) ListInventurSessions(ctx context.Context, tenantID uuid.UUID, offset, limit int) ([]*InventurSession, int, error) {
	var total int
	if err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM inventur_sessions WHERE tenant_id=$1`,
		tenantID,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count inventur sessions: %w", err)
	}

	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, name, date, status, location_id, created_by, created_at, updated_at
		 FROM inventur_sessions WHERE tenant_id=$1
		 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		tenantID, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list inventur sessions: %w", err)
	}
	defer rows.Close()

	var sessions []*InventurSession
	for rows.Next() {
		var s InventurSession
		if scanErr := rows.Scan(&s.ID, &s.TenantID, &s.Name, &s.Date, &s.Status, &s.LocationID, &s.CreatedBy, &s.CreatedAt, &s.UpdatedAt); scanErr != nil {
			return nil, 0, fmt.Errorf("scan inventur session row: %w", scanErr)
		}
		sessions = append(sessions, &s)
	}
	return sessions, total, rows.Err()
}

func (r *PostgresRepository) UpsertInventurCount(ctx context.Context, count *InventurCount) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO inventur_counts (id, tenant_id, session_id, item_id, expected, counted, counted_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 ON CONFLICT (session_id, item_id) DO UPDATE
		   SET counted=$6, counted_at=$7`,
		count.ID, count.TenantID, count.SessionID, count.ItemID, count.Expected, count.Counted, count.CountedAt,
	)
	return err
}

func (r *PostgresRepository) ListInventurCounts(ctx context.Context, sessionID uuid.UUID) ([]*InventurCount, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, session_id, item_id, expected, counted, counted_at
		 FROM inventur_counts WHERE session_id=$1 ORDER BY item_id`,
		sessionID,
	)
	if err != nil {
		return nil, fmt.Errorf("list inventur counts: %w", err)
	}
	defer rows.Close()

	var counts []*InventurCount
	for rows.Next() {
		var c InventurCount
		if scanErr := rows.Scan(&c.ID, &c.TenantID, &c.SessionID, &c.ItemID, &c.Expected, &c.Counted, &c.CountedAt); scanErr != nil {
			return nil, fmt.Errorf("scan inventur count row: %w", scanErr)
		}
		counts = append(counts, &c)
	}
	return counts, rows.Err()
}

// ============================================================================
// Scan helpers
// ============================================================================

func (r *PostgresRepository) scanItemFromRows(rows pgx.Rows) (*Item, error) {
	var item Item
	err := rows.Scan(
		&item.ID, &item.TenantID, &item.Name, &item.SKU, &item.Barcode,
		&item.Quantity, &item.MinQuantity, &item.Unit, &item.Location, &item.LocationID,
		&item.Category, &item.Price, &item.Currency, &item.BatchNumber, &item.SerialNumbers, &item.LinkedPurchaseOrder,
		&item.CreatedAt, &item.UpdatedAt, &item.DeletedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan inventory item row: %w", err)
	}
	return &item, nil
}

func (r *PostgresRepository) scanMovementFromRows(rows pgx.Rows) (*Movement, error) {
	var m Movement
	err := rows.Scan(
		&m.ID, &m.TenantID, &m.ItemID, &m.MovementType, &m.Quantity, &m.PerformedBy, &m.Reason,
		&m.Reference, &m.LocationFrom, &m.LocationTo, &m.Notes, &m.CreatedAt,
	)
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
