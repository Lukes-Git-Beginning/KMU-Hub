package einkauf

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRepository implements Repository using PostgreSQL.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgreSQL-backed einkauf repository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// ============================================================================
// Suppliers
// ============================================================================

func (r *PostgresRepository) CreateSupplier(ctx context.Context, s *Supplier) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO suppliers
		    (id, tenant_id, name, contact_id, email, phone, address, payment_terms, notes, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		s.ID, s.TenantID, s.Name, s.ContactID, s.Email, s.Phone,
		s.Address, s.PaymentTerms, s.Notes, s.CreatedAt, s.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) UpdateSupplier(ctx context.Context, s *Supplier) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE suppliers
		 SET name = $1, contact_id = $2, email = $3, phone = $4,
		     address = $5, payment_terms = $6, notes = $7, updated_at = $8
		 WHERE id = $9 AND tenant_id = $10 AND deleted_at IS NULL`,
		s.Name, s.ContactID, s.Email, s.Phone,
		s.Address, s.PaymentTerms, s.Notes, s.UpdatedAt, s.ID, s.TenantID,
	)
	return err
}

func (r *PostgresRepository) DeleteSupplier(ctx context.Context, tenantID, supplierID uuid.UUID) error {
	now := time.Now()
	ct, err := r.pool.Exec(ctx,
		`UPDATE suppliers SET deleted_at = $1, updated_at = $1 WHERE id = $2 AND tenant_id = $3 AND deleted_at IS NULL`,
		now, supplierID, tenantID,
	)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrSupplierNotFound
	}
	return nil
}

func (r *PostgresRepository) GetSupplier(ctx context.Context, tenantID, supplierID uuid.UUID) (*Supplier, error) {
	return r.scanSupplier(r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, name, contact_id, email, phone, address, payment_terms, notes, created_at, updated_at, deleted_at
		 FROM suppliers WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL`,
		supplierID, tenantID,
	))
}

func (r *PostgresRepository) ListSuppliers(ctx context.Context, tenantID uuid.UUID, filter ListSuppliersFilter, offset, limit int) ([]*Supplier, int, error) {
	var conditions []string
	var args []any
	argNum := 1

	conditions = append(conditions, fmt.Sprintf("tenant_id = $%d", argNum))
	args = append(args, tenantID)
	argNum++

	conditions = append(conditions, "deleted_at IS NULL")

	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("LOWER(name) LIKE $%d", argNum))
		args = append(args, "%"+strings.ToLower(filter.Search)+"%")
		argNum++
	}

	whereClause := "WHERE " + strings.Join(conditions, " AND ")

	var total int
	if err := r.pool.QueryRow(ctx, fmt.Sprintf("SELECT COUNT(*) FROM suppliers %s", whereClause), args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count suppliers: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT id, tenant_id, name, contact_id, email, phone, address, payment_terms, notes, created_at, updated_at, deleted_at
		FROM suppliers %s
		ORDER BY name ASC
		LIMIT $%d OFFSET $%d
	`, whereClause, argNum, argNum+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list suppliers: %w", err)
	}
	defer rows.Close()

	var suppliers []*Supplier
	for rows.Next() {
		s, scanErr := r.scanSupplierFromRows(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		suppliers = append(suppliers, s)
	}

	return suppliers, total, rows.Err()
}

// ============================================================================
// Purchase Orders
// ============================================================================

func (r *PostgresRepository) CreatePO(ctx context.Context, po *PurchaseOrder) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO purchase_orders
		    (id, tenant_id, supplier_id, po_number, status, order_date, expected_delivery_date,
		     total_amount, currency, notes, created_by, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
		po.ID, po.TenantID, po.SupplierID, po.PONumber, po.Status, po.OrderDate,
		po.ExpectedDeliveryDate, po.TotalAmount, po.Currency, po.Notes,
		po.CreatedBy, po.CreatedAt, po.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) UpdatePO(ctx context.Context, po *PurchaseOrder) error {
	ct, err := r.pool.Exec(ctx,
		`UPDATE purchase_orders
		 SET supplier_id = $1, po_number = $2, status = $3, order_date = $4,
		     expected_delivery_date = $5, total_amount = $6, currency = $7,
		     notes = $8, updated_at = $9
		 WHERE id = $10 AND tenant_id = $11`,
		po.SupplierID, po.PONumber, po.Status, po.OrderDate,
		po.ExpectedDeliveryDate, po.TotalAmount, po.Currency,
		po.Notes, po.UpdatedAt, po.ID, po.TenantID,
	)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrPONotFound
	}
	return nil
}

func (r *PostgresRepository) DeletePO(ctx context.Context, tenantID, poID uuid.UUID) error {
	// Defense-in-depth: only draft POs may be deleted. The service layer also
	// checks this, but a direct repo call (admin tool, future caller) must not
	// be able to wipe submitted/received purchase orders.
	ct, err := r.pool.Exec(ctx,
		`DELETE FROM purchase_orders WHERE id = $1 AND tenant_id = $2 AND status = 'draft'`,
		poID, tenantID,
	)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrPONotDraft
	}
	return nil
}

func (r *PostgresRepository) GetPO(ctx context.Context, tenantID, poID uuid.UUID) (*PurchaseOrder, error) {
	return r.scanPO(r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, supplier_id, po_number, status, order_date, expected_delivery_date,
		        total_amount, currency, notes, created_by, framework_contract_id, created_at, updated_at
		 FROM purchase_orders WHERE id = $1 AND tenant_id = $2`,
		poID, tenantID,
	))
}

func (r *PostgresRepository) GetPOWithLines(ctx context.Context, tenantID, poID uuid.UUID) (*PurchaseOrder, error) {
	po, err := r.GetPO(ctx, tenantID, poID)
	if err != nil {
		return nil, err
	}

	lines, err := r.ListPOLines(ctx, tenantID, poID)
	if err != nil {
		return nil, fmt.Errorf("load po lines: %w", err)
	}
	po.Lines = lines

	return po, nil
}

func (r *PostgresRepository) ListPOs(ctx context.Context, tenantID uuid.UUID, filter ListPOsFilter, offset, limit int) ([]*PurchaseOrder, int, error) {
	var conditions []string
	var args []any
	argNum := 1

	conditions = append(conditions, fmt.Sprintf("tenant_id = $%d", argNum))
	args = append(args, tenantID)
	argNum++

	if filter.SupplierID != nil {
		conditions = append(conditions, fmt.Sprintf("supplier_id = $%d", argNum))
		args = append(args, *filter.SupplierID)
		argNum++
	}

	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argNum))
		args = append(args, *filter.Status)
		argNum++
	}

	if filter.DateFrom != nil {
		conditions = append(conditions, fmt.Sprintf("order_date >= $%d", argNum))
		args = append(args, *filter.DateFrom)
		argNum++
	}

	if filter.DateTo != nil {
		conditions = append(conditions, fmt.Sprintf("order_date <= $%d", argNum))
		args = append(args, *filter.DateTo)
		argNum++
	}

	whereClause := "WHERE " + strings.Join(conditions, " AND ")

	var total int
	if err := r.pool.QueryRow(ctx, fmt.Sprintf("SELECT COUNT(*) FROM purchase_orders %s", whereClause), args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count purchase_orders: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT id, tenant_id, supplier_id, po_number, status, order_date, expected_delivery_date,
		       total_amount, currency, notes, created_by, framework_contract_id, created_at, updated_at
		FROM purchase_orders %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argNum, argNum+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list purchase_orders: %w", err)
	}
	defer rows.Close()

	var pos []*PurchaseOrder
	for rows.Next() {
		po, scanErr := r.scanPOFromRows(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		pos = append(pos, po)
	}

	return pos, total, rows.Err()
}

func (r *PostgresRepository) UpdatePOStatus(ctx context.Context, tenantID, poID uuid.UUID, status POStatus) error {
	now := time.Now()
	ct, err := r.pool.Exec(ctx,
		`UPDATE purchase_orders SET status = $1, updated_at = $2 WHERE id = $3 AND tenant_id = $4`,
		status, now, poID, tenantID,
	)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrPONotFound
	}
	return nil
}

func (r *PostgresRepository) PONumberExists(ctx context.Context, tenantID uuid.UUID, poNumber string, excludeID *uuid.UUID) (bool, error) {
	var exists bool
	if excludeID != nil {
		err := r.pool.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM purchase_orders WHERE tenant_id = $1 AND po_number = $2 AND id <> $3)`,
			tenantID, poNumber, *excludeID,
		).Scan(&exists)
		return exists, err
	}
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM purchase_orders WHERE tenant_id = $1 AND po_number = $2)`,
		tenantID, poNumber,
	).Scan(&exists)
	return exists, err
}

// ============================================================================
// PO Lines
// ============================================================================

func (r *PostgresRepository) CreatePOLine(ctx context.Context, line *POLine) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO po_lines
		    (id, tenant_id, po_id, product_name, sku, quantity, unit_price, tax_rate, received_quantity, line_position, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		line.ID, line.TenantID, line.POID, line.ProductName, line.SKU,
		line.Quantity, line.UnitPrice, line.TaxRate, line.ReceivedQuantity,
		line.LinePosition, line.CreatedAt, line.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) UpdatePOLine(ctx context.Context, line *POLine) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE po_lines
		 SET product_name = $1, sku = $2, quantity = $3, unit_price = $4,
		     tax_rate = $5, line_position = $6, updated_at = $7
		 WHERE id = $8 AND tenant_id = $9`,
		line.ProductName, line.SKU, line.Quantity, line.UnitPrice,
		line.TaxRate, line.LinePosition, line.UpdatedAt, line.ID, line.TenantID,
	)
	return err
}

func (r *PostgresRepository) DeletePOLine(ctx context.Context, tenantID, lineID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM po_lines WHERE id = $1 AND tenant_id = $2`,
		lineID, tenantID,
	)
	return err
}

func (r *PostgresRepository) GetPOLine(ctx context.Context, tenantID, lineID uuid.UUID) (*POLine, error) {
	var l POLine
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, po_id, product_name, sku, quantity, unit_price, tax_rate, received_quantity, line_position, created_at, updated_at
		 FROM po_lines WHERE id = $1 AND tenant_id = $2`,
		lineID, tenantID,
	).Scan(
		&l.ID, &l.TenantID, &l.POID, &l.ProductName, &l.SKU,
		&l.Quantity, &l.UnitPrice, &l.TaxRate, &l.ReceivedQuantity,
		&l.LinePosition, &l.CreatedAt, &l.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPOLineNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get po line: %w", err)
	}
	return &l, nil
}

func (r *PostgresRepository) ListPOLines(ctx context.Context, tenantID, poID uuid.UUID) ([]*POLine, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, po_id, product_name, sku, quantity, unit_price, tax_rate, received_quantity, line_position, created_at, updated_at
		 FROM po_lines WHERE po_id = $1 AND tenant_id = $2
		 ORDER BY line_position ASC`,
		poID, tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("list po lines: %w", err)
	}
	defer rows.Close()

	var lines []*POLine
	for rows.Next() {
		l, scanErr := r.scanPOLineFromRows(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		lines = append(lines, l)
	}

	return lines, rows.Err()
}

func (r *PostgresRepository) UpdatePOLineReceivedQuantity(ctx context.Context, tenantID, lineID uuid.UUID, receivedQty string) error {
	now := time.Now()
	_, err := r.pool.Exec(ctx,
		`UPDATE po_lines SET received_quantity = $1, updated_at = $2 WHERE id = $3 AND tenant_id = $4`,
		receivedQty, now, lineID, tenantID,
	)
	return err
}

func (r *PostgresRepository) CountPOLines(ctx context.Context, tenantID, poID uuid.UUID) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM po_lines WHERE po_id = $1 AND tenant_id = $2`,
		poID, tenantID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count po lines: %w", err)
	}
	return count, nil
}

// ============================================================================
// Scan helpers
// ============================================================================

func (r *PostgresRepository) scanSupplier(row pgx.Row) (*Supplier, error) {
	var s Supplier
	err := row.Scan(
		&s.ID, &s.TenantID, &s.Name, &s.ContactID, &s.Email, &s.Phone,
		&s.Address, &s.PaymentTerms, &s.Notes, &s.CreatedAt, &s.UpdatedAt, &s.DeletedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrSupplierNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan supplier: %w", err)
	}
	return &s, nil
}

func (r *PostgresRepository) scanSupplierFromRows(rows pgx.Rows) (*Supplier, error) {
	var s Supplier
	err := rows.Scan(
		&s.ID, &s.TenantID, &s.Name, &s.ContactID, &s.Email, &s.Phone,
		&s.Address, &s.PaymentTerms, &s.Notes, &s.CreatedAt, &s.UpdatedAt, &s.DeletedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan supplier row: %w", err)
	}
	return &s, nil
}

func (r *PostgresRepository) scanPO(row pgx.Row) (*PurchaseOrder, error) {
	var po PurchaseOrder
	err := row.Scan(
		&po.ID, &po.TenantID, &po.SupplierID, &po.PONumber, &po.Status,
		&po.OrderDate, &po.ExpectedDeliveryDate, &po.TotalAmount, &po.Currency,
		&po.Notes, &po.CreatedBy, &po.FrameworkContractID, &po.CreatedAt, &po.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrPONotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan purchase order: %w", err)
	}
	return &po, nil
}

func (r *PostgresRepository) scanPOFromRows(rows pgx.Rows) (*PurchaseOrder, error) {
	var po PurchaseOrder
	err := rows.Scan(
		&po.ID, &po.TenantID, &po.SupplierID, &po.PONumber, &po.Status,
		&po.OrderDate, &po.ExpectedDeliveryDate, &po.TotalAmount, &po.Currency,
		&po.Notes, &po.CreatedBy, &po.FrameworkContractID, &po.CreatedAt, &po.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan purchase order row: %w", err)
	}
	return &po, nil
}

func (r *PostgresRepository) scanPOLineFromRows(rows pgx.Rows) (*POLine, error) {
	var l POLine
	err := rows.Scan(
		&l.ID, &l.TenantID, &l.POID, &l.ProductName, &l.SKU,
		&l.Quantity, &l.UnitPrice, &l.TaxRate, &l.ReceivedQuantity,
		&l.LinePosition, &l.CreatedAt, &l.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan po line row: %w", err)
	}
	return &l, nil
}

// compile-time interface check
var _ Repository = (*PostgresRepository)(nil)
