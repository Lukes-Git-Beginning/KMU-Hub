package produktion

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ============================================================================
// BOMs
// ============================================================================

func (r *PostgresRepository) CreateBOM(ctx context.Context, bom *BOM) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	_, err = tx.Exec(ctx,
		`INSERT INTO production_boms
		    (id, tenant_id, product_name, sku, version, active, notes, created_by, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		bom.ID, bom.TenantID, bom.ProductName, bom.SKU, bom.Version,
		bom.Active, bom.Notes, bom.CreatedBy, bom.CreatedAt, bom.UpdatedAt,
	)
	if err != nil {
		if strings.Contains(err.Error(), "idx_production_boms_tenant_sku") {
			return ErrBOMSKUTaken
		}
		return fmt.Errorf("insert bom: %w", err)
	}

	for _, item := range bom.Items {
		_, err = tx.Exec(ctx,
			`INSERT INTO production_bom_items
			    (id, tenant_id, bom_id, material_name, quantity, unit, sort_order, created_at, updated_at)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
			item.ID, item.TenantID, item.BomID, item.MaterialName,
			item.Quantity, item.Unit, item.SortOrder, bom.CreatedAt, bom.UpdatedAt,
		)
		if err != nil {
			return fmt.Errorf("insert bom item: %w", err)
		}
	}

	return tx.Commit(ctx)
}

func (r *PostgresRepository) UpdateBOM(ctx context.Context, bom *BOM) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE production_boms
		 SET product_name=$1, sku=$2, version=$3, active=$4, notes=$5, updated_at=$6
		 WHERE id=$7 AND tenant_id=$8`,
		bom.ProductName, bom.SKU, bom.Version, bom.Active, bom.Notes, bom.UpdatedAt,
		bom.ID, bom.TenantID,
	)
	if err != nil {
		if strings.Contains(err.Error(), "idx_production_boms_tenant_sku") {
			return ErrBOMSKUTaken
		}
		return fmt.Errorf("update bom: %w", err)
	}
	return nil
}

func (r *PostgresRepository) GetBOM(ctx context.Context, tenantID, bomID uuid.UUID) (*BOM, error) {
	var bom BOM
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, product_name, sku, version, active, notes, created_by, created_at, updated_at
		 FROM production_boms WHERE id=$1 AND tenant_id=$2`,
		bomID, tenantID,
	).Scan(&bom.ID, &bom.TenantID, &bom.ProductName, &bom.SKU, &bom.Version,
		&bom.Active, &bom.Notes, &bom.CreatedBy, &bom.CreatedAt, &bom.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrBOMNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get bom: %w", err)
	}

	// Load items
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, bom_id, material_name, quantity, unit, sort_order
		 FROM production_bom_items WHERE bom_id=$1 ORDER BY sort_order ASC, id ASC`,
		bomID,
	)
	if err != nil {
		return nil, fmt.Errorf("list bom items: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var item BomItem
		if scanErr := rows.Scan(&item.ID, &item.TenantID, &item.BomID,
			&item.MaterialName, &item.Quantity, &item.Unit, &item.SortOrder); scanErr != nil {
			return nil, fmt.Errorf("scan bom item: %w", scanErr)
		}
		bom.Items = append(bom.Items, &item)
	}

	return &bom, rows.Err()
}

func (r *PostgresRepository) ListBOMs(ctx context.Context, tenantID uuid.UUID, activeOnly *bool, offset, limit int) ([]*BOM, int, error) {
	var conditions []string
	var args []any
	argNum := 1

	conditions = append(conditions, fmt.Sprintf("tenant_id = $%d", argNum))
	args = append(args, tenantID)
	argNum++

	if activeOnly != nil {
		conditions = append(conditions, fmt.Sprintf("active = $%d", argNum))
		args = append(args, *activeOnly)
		argNum++
	}

	whereClause := "WHERE " + strings.Join(conditions, " AND ")

	var total int
	if err := r.pool.QueryRow(ctx,
		fmt.Sprintf("SELECT COUNT(*) FROM production_boms %s", whereClause),
		args...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count boms: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT id, tenant_id, product_name, sku, version, active, notes, created_by, created_at, updated_at
		FROM production_boms %s
		ORDER BY product_name ASC
		LIMIT $%d OFFSET $%d
	`, whereClause, argNum, argNum+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list boms: %w", err)
	}
	defer rows.Close()

	var boms []*BOM
	for rows.Next() {
		var b BOM
		if scanErr := rows.Scan(&b.ID, &b.TenantID, &b.ProductName, &b.SKU, &b.Version,
			&b.Active, &b.Notes, &b.CreatedBy, &b.CreatedAt, &b.UpdatedAt); scanErr != nil {
			return nil, 0, fmt.Errorf("scan bom: %w", scanErr)
		}
		boms = append(boms, &b)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	// Load items for all BOMs in one query
	if len(boms) > 0 {
		bomIDs := make([]uuid.UUID, len(boms))
		bomIndex := make(map[uuid.UUID]*BOM)
		for i, b := range boms {
			bomIDs[i] = b.ID
			bomIndex[b.ID] = b
		}

		itemRows, itemErr := r.pool.Query(ctx,
			`SELECT id, tenant_id, bom_id, material_name, quantity, unit, sort_order
			 FROM production_bom_items WHERE bom_id = ANY($1) ORDER BY bom_id, sort_order ASC`,
			bomIDs,
		)
		if itemErr != nil {
			return nil, 0, fmt.Errorf("list bom items bulk: %w", itemErr)
		}
		defer itemRows.Close()

		for itemRows.Next() {
			var item BomItem
			if scanErr := itemRows.Scan(&item.ID, &item.TenantID, &item.BomID,
				&item.MaterialName, &item.Quantity, &item.Unit, &item.SortOrder); scanErr != nil {
				return nil, 0, fmt.Errorf("scan bom item bulk: %w", scanErr)
			}
			if b, ok := bomIndex[item.BomID]; ok {
				b.Items = append(b.Items, &item)
			}
		}
		if err := itemRows.Err(); err != nil {
			return nil, 0, err
		}
	}

	return boms, total, nil
}

func (r *PostgresRepository) DeleteBOM(ctx context.Context, tenantID, bomID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM production_boms WHERE id=$1 AND tenant_id=$2`,
		bomID, tenantID,
	)
	return err
}

// ============================================================================
// Work Steps
// ============================================================================

func (r *PostgresRepository) CreateWorkStep(ctx context.Context, step *WorkStep) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO production_work_steps
		    (id, tenant_id, order_id, step_nr, name, description, duration_minutes, status, assignee, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		step.ID, step.TenantID, step.OrderID, step.StepNr, step.Name,
		step.Description, step.DurationMinutes, step.Status, step.Assignee,
		step.CreatedAt, step.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) UpdateWorkStep(ctx context.Context, step *WorkStep) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE production_work_steps
		 SET name=$1, description=$2, duration_minutes=$3, status=$4, assignee=$5, updated_at=$6
		 WHERE id=$7 AND tenant_id=$8`,
		step.Name, step.Description, step.DurationMinutes, step.Status,
		step.Assignee, step.UpdatedAt, step.ID, step.TenantID,
	)
	return err
}

func (r *PostgresRepository) GetWorkStep(ctx context.Context, tenantID, stepID uuid.UUID) (*WorkStep, error) {
	var s WorkStep
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, order_id, step_nr, name, description, duration_minutes, status, assignee, created_at, updated_at
		 FROM production_work_steps WHERE id=$1 AND tenant_id=$2`,
		stepID, tenantID,
	).Scan(&s.ID, &s.TenantID, &s.OrderID, &s.StepNr, &s.Name, &s.Description,
		&s.DurationMinutes, &s.Status, &s.Assignee, &s.CreatedAt, &s.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrWorkStepNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get work step: %w", err)
	}
	return &s, nil
}

func (r *PostgresRepository) ListWorkSteps(ctx context.Context, tenantID, orderID uuid.UUID) ([]*WorkStep, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, order_id, step_nr, name, description, duration_minutes, status, assignee, created_at, updated_at
		 FROM production_work_steps WHERE tenant_id=$1 AND order_id=$2 ORDER BY step_nr ASC`,
		tenantID, orderID,
	)
	if err != nil {
		return nil, fmt.Errorf("list work steps: %w", err)
	}
	defer rows.Close()

	var steps []*WorkStep
	for rows.Next() {
		var s WorkStep
		if scanErr := rows.Scan(&s.ID, &s.TenantID, &s.OrderID, &s.StepNr, &s.Name,
			&s.Description, &s.DurationMinutes, &s.Status, &s.Assignee, &s.CreatedAt, &s.UpdatedAt); scanErr != nil {
			return nil, fmt.Errorf("scan work step: %w", scanErr)
		}
		steps = append(steps, &s)
	}
	return steps, rows.Err()
}

func (r *PostgresRepository) DeleteWorkStep(ctx context.Context, tenantID, stepID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM production_work_steps WHERE id=$1 AND tenant_id=$2`,
		stepID, tenantID,
	)
	return err
}

// ============================================================================
// Machines
// ============================================================================

func (r *PostgresRepository) CreateMachine(ctx context.Context, machine *Machine) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO production_machines
		    (id, tenant_id, name, type, status, notes, created_by, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		machine.ID, machine.TenantID, machine.Name, machine.Type,
		machine.Status, machine.Notes, machine.CreatedBy, machine.CreatedAt, machine.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) UpdateMachine(ctx context.Context, machine *Machine) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE production_machines
		 SET name=$1, type=$2, status=$3, notes=$4, updated_at=$5
		 WHERE id=$6 AND tenant_id=$7`,
		machine.Name, machine.Type, machine.Status, machine.Notes,
		machine.UpdatedAt, machine.ID, machine.TenantID,
	)
	return err
}

func (r *PostgresRepository) GetMachine(ctx context.Context, tenantID, machineID uuid.UUID) (*Machine, error) {
	var m Machine
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, name, type, status, notes, created_by, created_at, updated_at
		 FROM production_machines WHERE id=$1 AND tenant_id=$2`,
		machineID, tenantID,
	).Scan(&m.ID, &m.TenantID, &m.Name, &m.Type, &m.Status,
		&m.Notes, &m.CreatedBy, &m.CreatedAt, &m.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrMachineNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get machine: %w", err)
	}
	return &m, nil
}

func (r *PostgresRepository) ListMachines(ctx context.Context, tenantID uuid.UUID, status *MachineStatus, offset, limit int) ([]*Machine, int, error) {
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
	if err := r.pool.QueryRow(ctx,
		fmt.Sprintf("SELECT COUNT(*) FROM production_machines %s", whereClause),
		args...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count machines: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT id, tenant_id, name, type, status, notes, created_by, created_at, updated_at
		FROM production_machines %s
		ORDER BY name ASC
		LIMIT $%d OFFSET $%d
	`, whereClause, argNum, argNum+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list machines: %w", err)
	}
	defer rows.Close()

	var machines []*Machine
	for rows.Next() {
		var m Machine
		if scanErr := rows.Scan(&m.ID, &m.TenantID, &m.Name, &m.Type, &m.Status,
			&m.Notes, &m.CreatedBy, &m.CreatedAt, &m.UpdatedAt); scanErr != nil {
			return nil, 0, fmt.Errorf("scan machine: %w", scanErr)
		}
		machines = append(machines, &m)
	}
	return machines, total, rows.Err()
}

func (r *PostgresRepository) DeleteMachine(ctx context.Context, tenantID, machineID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM production_machines WHERE id=$1 AND tenant_id=$2`,
		machineID, tenantID,
	)
	return err
}

// ============================================================================
// Quality Checks
// ============================================================================

func (r *PostgresRepository) CreateQualityCheck(ctx context.Context, check *QualityCheck) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO production_quality_checks
		    (id, tenant_id, order_id, inspector, checked_at, passed, defects_found, notes, created_by, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		check.ID, check.TenantID, check.OrderID, check.Inspector, check.CheckedAt,
		check.Passed, check.DefectsFound, check.Notes, check.CreatedBy,
		check.CreatedAt, check.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) GetQualityCheck(ctx context.Context, tenantID, checkID uuid.UUID) (*QualityCheck, error) {
	var c QualityCheck
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, order_id, inspector, checked_at, passed, defects_found, notes, created_by, created_at, updated_at
		 FROM production_quality_checks WHERE id=$1 AND tenant_id=$2`,
		checkID, tenantID,
	).Scan(&c.ID, &c.TenantID, &c.OrderID, &c.Inspector, &c.CheckedAt,
		&c.Passed, &c.DefectsFound, &c.Notes, &c.CreatedBy, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrQualityCheckNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get quality check: %w", err)
	}
	return &c, nil
}

func (r *PostgresRepository) ListQualityChecks(ctx context.Context, tenantID uuid.UUID, orderID *uuid.UUID, offset, limit int) ([]*QualityCheck, int, error) {
	var conditions []string
	var args []any
	argNum := 1

	conditions = append(conditions, fmt.Sprintf("tenant_id = $%d", argNum))
	args = append(args, tenantID)
	argNum++

	if orderID != nil {
		conditions = append(conditions, fmt.Sprintf("order_id = $%d", argNum))
		args = append(args, *orderID)
		argNum++
	}

	whereClause := "WHERE " + strings.Join(conditions, " AND ")

	var total int
	if err := r.pool.QueryRow(ctx,
		fmt.Sprintf("SELECT COUNT(*) FROM production_quality_checks %s", whereClause),
		args...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count quality checks: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT id, tenant_id, order_id, inspector, checked_at, passed, defects_found, notes, created_by, created_at, updated_at
		FROM production_quality_checks %s
		ORDER BY checked_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argNum, argNum+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list quality checks: %w", err)
	}
	defer rows.Close()

	var checks []*QualityCheck
	for rows.Next() {
		var c QualityCheck
		if scanErr := rows.Scan(&c.ID, &c.TenantID, &c.OrderID, &c.Inspector, &c.CheckedAt,
			&c.Passed, &c.DefectsFound, &c.Notes, &c.CreatedBy, &c.CreatedAt, &c.UpdatedAt); scanErr != nil {
			return nil, 0, fmt.Errorf("scan quality check: %w", scanErr)
		}
		checks = append(checks, &c)
	}
	return checks, total, rows.Err()
}
