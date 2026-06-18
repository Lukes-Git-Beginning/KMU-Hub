package einkauf

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ============================================================================
// Catalog Items
// ============================================================================

func (r *PostgresRepository) CreateCatalogItem(ctx context.Context, item *CatalogItem) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO supplier_catalog_items
		    (id, tenant_id, supplier_id, name, sku, category, price, currency, unit, available, min_order_qty, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
		item.ID, item.TenantID, item.SupplierID, item.Name, item.SKU, item.Category,
		item.Price, item.Currency, item.Unit, item.Available, item.MinOrderQty,
		item.CreatedAt, item.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) UpdateCatalogItem(ctx context.Context, item *CatalogItem) error {
	ct, err := r.pool.Exec(ctx,
		`UPDATE supplier_catalog_items
		 SET name = $1, sku = $2, category = $3, price = $4, currency = $5,
		     unit = $6, available = $7, min_order_qty = $8, updated_at = $9
		 WHERE id = $10 AND tenant_id = $11`,
		item.Name, item.SKU, item.Category, item.Price, item.Currency,
		item.Unit, item.Available, item.MinOrderQty, item.UpdatedAt, item.ID, item.TenantID,
	)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrCatalogItemNotFound
	}
	return nil
}

func (r *PostgresRepository) DeleteCatalogItem(ctx context.Context, tenantID, itemID uuid.UUID) error {
	ct, err := r.pool.Exec(ctx,
		`DELETE FROM supplier_catalog_items WHERE id = $1 AND tenant_id = $2`,
		itemID, tenantID,
	)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrCatalogItemNotFound
	}
	return nil
}

func (r *PostgresRepository) GetCatalogItem(ctx context.Context, tenantID, itemID uuid.UUID) (*CatalogItem, error) {
	var item CatalogItem
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, supplier_id, name, sku, category, price::text, currency, unit, available, min_order_qty::text, created_at, updated_at
		 FROM supplier_catalog_items WHERE id = $1 AND tenant_id = $2`,
		itemID, tenantID,
	).Scan(
		&item.ID, &item.TenantID, &item.SupplierID, &item.Name, &item.SKU, &item.Category,
		&item.Price, &item.Currency, &item.Unit, &item.Available, &item.MinOrderQty,
		&item.CreatedAt, &item.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrCatalogItemNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get catalog item: %w", err)
	}
	return &item, nil
}

func (r *PostgresRepository) ListCatalogItems(ctx context.Context, tenantID uuid.UUID, filter ListCatalogItemsFilter, offset, limit int) ([]*CatalogItem, int, error) {
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

	if filter.Category != "" {
		conditions = append(conditions, fmt.Sprintf("category = $%d", argNum))
		args = append(args, filter.Category)
		argNum++
	}

	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(LOWER(name) LIKE $%d OR LOWER(sku) LIKE $%d)", argNum, argNum))
		args = append(args, "%"+strings.ToLower(filter.Search)+"%")
		argNum++
	}

	if filter.Available != nil {
		conditions = append(conditions, fmt.Sprintf("available = $%d", argNum))
		args = append(args, *filter.Available)
		argNum++
	}

	whereClause := "WHERE " + strings.Join(conditions, " AND ")

	var total int
	if err := r.pool.QueryRow(ctx,
		fmt.Sprintf("SELECT COUNT(*) FROM supplier_catalog_items %s", whereClause),
		args...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count catalog items: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT id, tenant_id, supplier_id, name, sku, category, price::text, currency, unit, available, min_order_qty::text, created_at, updated_at
		FROM supplier_catalog_items %s
		ORDER BY name ASC
		LIMIT $%d OFFSET $%d
	`, whereClause, argNum, argNum+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list catalog items: %w", err)
	}
	defer rows.Close()

	var items []*CatalogItem
	for rows.Next() {
		var item CatalogItem
		if scanErr := rows.Scan(
			&item.ID, &item.TenantID, &item.SupplierID, &item.Name, &item.SKU, &item.Category,
			&item.Price, &item.Currency, &item.Unit, &item.Available, &item.MinOrderQty,
			&item.CreatedAt, &item.UpdatedAt,
		); scanErr != nil {
			return nil, 0, fmt.Errorf("scan catalog item row: %w", scanErr)
		}
		items = append(items, &item)
	}

	return items, total, rows.Err()
}

// ============================================================================
// Supplier Ratings
// ============================================================================

func (r *PostgresRepository) CreateSupplierRating(ctx context.Context, rating *SupplierRating) error {
	// Duplicate check: same tenant + supplier + category
	var exists bool
	if err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM supplier_ratings WHERE tenant_id = $1 AND supplier_id = $2 AND category = $3)`,
		rating.TenantID, rating.SupplierID, rating.Category,
	).Scan(&exists); err != nil {
		return fmt.Errorf("check duplicate rating: %w", err)
	}
	if exists {
		return ErrDuplicateRating
	}

	_, err := r.pool.Exec(ctx,
		`INSERT INTO supplier_ratings
		    (id, tenant_id, supplier_id, category, rating, comment, rated_by, rated_at, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		rating.ID, rating.TenantID, rating.SupplierID, rating.Category, rating.Rating,
		rating.Comment, rating.RatedBy, rating.RatedAt, rating.CreatedAt, rating.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) DeleteSupplierRating(ctx context.Context, tenantID, ratingID uuid.UUID) error {
	ct, err := r.pool.Exec(ctx,
		`DELETE FROM supplier_ratings WHERE id = $1 AND tenant_id = $2`,
		ratingID, tenantID,
	)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrSupplierRatingNotFound
	}
	return nil
}

func (r *PostgresRepository) GetSupplierRating(ctx context.Context, tenantID, ratingID uuid.UUID) (*SupplierRating, error) {
	var rt SupplierRating
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, supplier_id, category, rating, comment, rated_by, rated_at, created_at, updated_at
		 FROM supplier_ratings WHERE id = $1 AND tenant_id = $2`,
		ratingID, tenantID,
	).Scan(
		&rt.ID, &rt.TenantID, &rt.SupplierID, &rt.Category, &rt.Rating,
		&rt.Comment, &rt.RatedBy, &rt.RatedAt, &rt.CreatedAt, &rt.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrSupplierRatingNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get supplier rating: %w", err)
	}
	return &rt, nil
}

func (r *PostgresRepository) ListSupplierRatings(ctx context.Context, tenantID, supplierID uuid.UUID) ([]*SupplierRating, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, supplier_id, category, rating, comment, rated_by, rated_at, created_at, updated_at
		 FROM supplier_ratings WHERE tenant_id = $1 AND supplier_id = $2
		 ORDER BY rated_at DESC`,
		tenantID, supplierID,
	)
	if err != nil {
		return nil, fmt.Errorf("list supplier ratings: %w", err)
	}
	defer rows.Close()

	var ratings []*SupplierRating
	for rows.Next() {
		var rt SupplierRating
		if scanErr := rows.Scan(
			&rt.ID, &rt.TenantID, &rt.SupplierID, &rt.Category, &rt.Rating,
			&rt.Comment, &rt.RatedBy, &rt.RatedAt, &rt.CreatedAt, &rt.UpdatedAt,
		); scanErr != nil {
			return nil, fmt.Errorf("scan supplier rating row: %w", scanErr)
		}
		ratings = append(ratings, &rt)
	}

	return ratings, rows.Err()
}

// ============================================================================
// Framework Contracts
// ============================================================================

func (r *PostgresRepository) CreateFrameworkContract(ctx context.Context, fc *FrameworkContract) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO framework_contracts
		    (id, tenant_id, supplier_id, title, contract_nr, start_date, end_date, total_value, used_value, currency, status, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
		fc.ID, fc.TenantID, fc.SupplierID, fc.Title, fc.ContractNr,
		fc.StartDate, fc.EndDate, fc.TotalValue, fc.UsedValue, fc.Currency, fc.Status,
		fc.CreatedAt, fc.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) UpdateFrameworkContract(ctx context.Context, fc *FrameworkContract) error {
	ct, err := r.pool.Exec(ctx,
		`UPDATE framework_contracts
		 SET supplier_id = $1, title = $2, contract_nr = $3, start_date = $4, end_date = $5,
		     total_value = $6, currency = $7, status = $8, updated_at = $9
		 WHERE id = $10 AND tenant_id = $11`,
		fc.SupplierID, fc.Title, fc.ContractNr, fc.StartDate, fc.EndDate,
		fc.TotalValue, fc.Currency, fc.Status, fc.UpdatedAt, fc.ID, fc.TenantID,
	)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrContractNotFound
	}
	return nil
}

func (r *PostgresRepository) DeleteFrameworkContract(ctx context.Context, tenantID, contractID uuid.UUID) error {
	ct, err := r.pool.Exec(ctx,
		`DELETE FROM framework_contracts WHERE id = $1 AND tenant_id = $2`,
		contractID, tenantID,
	)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrContractNotFound
	}
	return nil
}

func (r *PostgresRepository) GetFrameworkContract(ctx context.Context, tenantID, contractID uuid.UUID) (*FrameworkContract, error) {
	var fc FrameworkContract
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, supplier_id, title, contract_nr, start_date, end_date,
		        total_value::text, used_value::text, currency, status, created_at, updated_at
		 FROM framework_contracts WHERE id = $1 AND tenant_id = $2`,
		contractID, tenantID,
	).Scan(
		&fc.ID, &fc.TenantID, &fc.SupplierID, &fc.Title, &fc.ContractNr,
		&fc.StartDate, &fc.EndDate, &fc.TotalValue, &fc.UsedValue, &fc.Currency,
		&fc.Status, &fc.CreatedAt, &fc.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrContractNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get framework contract: %w", err)
	}
	return &fc, nil
}

func (r *PostgresRepository) GetFrameworkContractWithItems(ctx context.Context, tenantID, contractID uuid.UUID) (*FrameworkContract, error) {
	fc, err := r.GetFrameworkContract(ctx, tenantID, contractID)
	if err != nil {
		return nil, err
	}

	items, err := r.ListContractItems(ctx, tenantID, contractID)
	if err != nil {
		return nil, fmt.Errorf("load contract items: %w", err)
	}
	fc.Items = items

	return fc, nil
}

func (r *PostgresRepository) ListFrameworkContracts(ctx context.Context, tenantID uuid.UUID, filter ListContractsFilter, offset, limit int) ([]*FrameworkContract, int, error) {
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

	whereClause := "WHERE " + strings.Join(conditions, " AND ")

	var total int
	if err := r.pool.QueryRow(ctx,
		fmt.Sprintf("SELECT COUNT(*) FROM framework_contracts %s", whereClause),
		args...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count framework contracts: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT id, tenant_id, supplier_id, title, contract_nr, start_date, end_date,
		       total_value::text, used_value::text, currency, status, created_at, updated_at
		FROM framework_contracts %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argNum, argNum+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list framework contracts: %w", err)
	}
	defer rows.Close()

	var contracts []*FrameworkContract
	for rows.Next() {
		var fc FrameworkContract
		if scanErr := rows.Scan(
			&fc.ID, &fc.TenantID, &fc.SupplierID, &fc.Title, &fc.ContractNr,
			&fc.StartDate, &fc.EndDate, &fc.TotalValue, &fc.UsedValue, &fc.Currency,
			&fc.Status, &fc.CreatedAt, &fc.UpdatedAt,
		); scanErr != nil {
			return nil, 0, fmt.Errorf("scan framework contract row: %w", scanErr)
		}
		contracts = append(contracts, &fc)
	}

	return contracts, total, rows.Err()
}

func (r *PostgresRepository) ContractNrExists(ctx context.Context, tenantID uuid.UUID, contractNr string, excludeID *uuid.UUID) (bool, error) {
	var exists bool
	if excludeID != nil {
		err := r.pool.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM framework_contracts WHERE tenant_id = $1 AND contract_nr = $2 AND id <> $3)`,
			tenantID, contractNr, *excludeID,
		).Scan(&exists)
		return exists, err
	}
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM framework_contracts WHERE tenant_id = $1 AND contract_nr = $2)`,
		tenantID, contractNr,
	).Scan(&exists)
	return exists, err
}

func (r *PostgresRepository) UpdateContractUsedValue(ctx context.Context, tenantID, contractID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE framework_contracts
		 SET used_value = (
		     SELECT COALESCE(SUM(amount), 0)
		     FROM framework_contract_calls
		     WHERE contract_id = $1 AND tenant_id = $2
		 ), updated_at = NOW()
		 WHERE id = $1 AND tenant_id = $2`,
		contractID, tenantID,
	)
	return err
}

// ============================================================================
// Framework Contract Items
// ============================================================================

func (r *PostgresRepository) CreateContractItem(ctx context.Context, item *FrameworkContractItem) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO framework_contract_items
		    (id, tenant_id, contract_id, name, unit_price, unit, agreed_qty, called_qty, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		item.ID, item.TenantID, item.ContractID, item.Name,
		item.UnitPrice, item.Unit, item.AgreedQty, item.CalledQty,
		item.CreatedAt, item.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) UpdateContractItem(ctx context.Context, item *FrameworkContractItem) error {
	ct, err := r.pool.Exec(ctx,
		`UPDATE framework_contract_items
		 SET name = $1, unit_price = $2, unit = $3, agreed_qty = $4, updated_at = $5
		 WHERE id = $6 AND tenant_id = $7`,
		item.Name, item.UnitPrice, item.Unit, item.AgreedQty, item.UpdatedAt, item.ID, item.TenantID,
	)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrContractItemNotFound
	}
	return nil
}

// QueryRowContractItem fetches a single framework contract item by item ID and tenant ID.
// This is used internally by the service layer for update operations.
func (r *PostgresRepository) QueryRowContractItem(ctx context.Context, tenantID, itemID uuid.UUID) (*FrameworkContractItem, error) {
	var item FrameworkContractItem
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, contract_id, name, unit_price::text, unit, agreed_qty::text, called_qty::text, created_at, updated_at
		 FROM framework_contract_items WHERE id = $1 AND tenant_id = $2`,
		itemID, tenantID,
	).Scan(
		&item.ID, &item.TenantID, &item.ContractID, &item.Name,
		&item.UnitPrice, &item.Unit, &item.AgreedQty, &item.CalledQty,
		&item.CreatedAt, &item.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrContractItemNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get contract item: %w", err)
	}
	return &item, nil
}

func (r *PostgresRepository) DeleteContractItem(ctx context.Context, tenantID, itemID uuid.UUID) error {
	ct, err := r.pool.Exec(ctx,
		`DELETE FROM framework_contract_items WHERE id = $1 AND tenant_id = $2`,
		itemID, tenantID,
	)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrContractItemNotFound
	}
	return nil
}

func (r *PostgresRepository) ListContractItems(ctx context.Context, tenantID, contractID uuid.UUID) ([]*FrameworkContractItem, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, contract_id, name, unit_price::text, unit, agreed_qty::text, called_qty::text, created_at, updated_at
		 FROM framework_contract_items WHERE contract_id = $1 AND tenant_id = $2
		 ORDER BY created_at ASC`,
		contractID, tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("list contract items: %w", err)
	}
	defer rows.Close()

	var items []*FrameworkContractItem
	for rows.Next() {
		var item FrameworkContractItem
		if scanErr := rows.Scan(
			&item.ID, &item.TenantID, &item.ContractID, &item.Name,
			&item.UnitPrice, &item.Unit, &item.AgreedQty, &item.CalledQty,
			&item.CreatedAt, &item.UpdatedAt,
		); scanErr != nil {
			return nil, fmt.Errorf("scan contract item row: %w", scanErr)
		}
		items = append(items, &item)
	}

	return items, rows.Err()
}

// ============================================================================
// Framework Contract Calls
// ============================================================================

func (r *PostgresRepository) CreateContractCall(ctx context.Context, call *FrameworkContractCall) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO framework_contract_calls
		    (id, tenant_id, contract_id, po_id, amount, currency, called_at, notes, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		call.ID, call.TenantID, call.ContractID, call.POID,
		call.Amount, call.Currency, call.CalledAt, call.Notes,
		call.CreatedAt, call.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) ListContractCalls(ctx context.Context, tenantID, contractID uuid.UUID) ([]*FrameworkContractCall, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, contract_id, po_id, amount::text, currency, called_at, notes, created_at, updated_at
		 FROM framework_contract_calls WHERE contract_id = $1 AND tenant_id = $2
		 ORDER BY called_at DESC`,
		contractID, tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("list contract calls: %w", err)
	}
	defer rows.Close()

	var calls []*FrameworkContractCall
	for rows.Next() {
		var call FrameworkContractCall
		if scanErr := rows.Scan(
			&call.ID, &call.TenantID, &call.ContractID, &call.POID,
			&call.Amount, &call.Currency, &call.CalledAt, &call.Notes,
			&call.CreatedAt, &call.UpdatedAt,
		); scanErr != nil {
			return nil, fmt.Errorf("scan contract call row: %w", scanErr)
		}
		calls = append(calls, &call)
	}

	return calls, rows.Err()
}

// ============================================================================
// Compile-time interface check
// ============================================================================

var _ RepositoryExtended = (*PostgresRepository)(nil)

