package einkauf

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

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

// CreateContractCall records a call-off, enforces the contract's status and
// remaining value, and recomputes used_value — all inside one transaction.
//
// The contract row is locked FOR UPDATE before the budget is computed, so two
// concurrent call-offs cannot both pass a check that only one of them fits.
// The remaining value is derived from a fresh SUM(amount) over the existing
// calls, never from the stored used_value column: that column is a cache, and
// trusting it would let the cap drift open whenever a recompute was lost. For
// the same reason the recompute happens here rather than in a follow-up
// statement the caller could skip or fail.
func (r *PostgresRepository) CreateContractCall(ctx context.Context, call *FrameworkContractCall) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin contract call tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	remaining, fits, err := bookContractCall(ctx, tx, call, false)
	if err != nil {
		return err
	}
	if !fits {
		return fmt.Errorf("call-off of %s, remaining %s: %w", call.Amount, remaining, ErrContractBudgetExceeded)
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit contract call: %w", err)
	}
	return nil
}

// bookContractCall does the work of CreateContractCall inside the caller's
// transaction and reports the remaining value before the call-off and whether
// the call-off fit into it. With overBudget false a call-off that does not fit
// is not written; with overBudget true (goods receipts) it is written anyway.
func bookContractCall(ctx context.Context, tx pgx.Tx, call *FrameworkContractCall, overBudget bool) (string, bool, error) {
	var contractStatus, totalValue, contractCurrency string
	err := tx.QueryRow(ctx,
		`SELECT status, total_value::text, currency
		 FROM framework_contracts
		 WHERE id = $1 AND tenant_id = $2
		 FOR UPDATE`,
		call.ContractID, call.TenantID,
	).Scan(&contractStatus, &totalValue, &contractCurrency)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, ErrContractNotFound
	}
	if err != nil {
		return "", false, fmt.Errorf("lock framework contract: %w", err)
	}
	if contractStatus != string(ContractStatusActive) {
		return "", false, fmt.Errorf("contract is in status %q: %w", contractStatus, ErrContractNotActive)
	}

	// The contract is the source of the currency: an empty call-off currency
	// takes it, a different one is refused. The remaining-value check below
	// sums raw amounts, which is only meaningful within one currency.
	if call.Currency == "" {
		call.Currency = contractCurrency
	} else if !strings.EqualFold(call.Currency, contractCurrency) {
		return "", false, fmt.Errorf("call-off in %s, contract in %s: %w", call.Currency, contractCurrency, ErrContractCurrencyMismatch)
	}

	// Comparison and remaining value are computed by Postgres in numeric, not
	// in Go float64 — money must not round through binary floating point.
	var remaining string
	var fits bool
	err = tx.QueryRow(ctx,
		`SELECT ($1::numeric - COALESCE(SUM(amount), 0))::text,
		        $2::numeric <= $1::numeric - COALESCE(SUM(amount), 0)
		 FROM framework_contract_calls
		 WHERE contract_id = $3 AND tenant_id = $4`,
		totalValue, call.Amount, call.ContractID, call.TenantID,
	).Scan(&remaining, &fits)
	if err != nil {
		return "", false, fmt.Errorf("compute remaining contract value: %w", err)
	}
	if !fits && !overBudget {
		return remaining, false, nil
	}

	if _, err = tx.Exec(ctx,
		`INSERT INTO framework_contract_calls
		    (id, tenant_id, contract_id, po_id, amount, currency, called_at, notes, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		call.ID, call.TenantID, call.ContractID, call.POID,
		call.Amount, call.Currency, call.CalledAt, call.Notes,
		call.CreatedAt, call.UpdatedAt,
	); err != nil {
		return "", false, fmt.Errorf("insert contract call: %w", err)
	}

	if _, err = tx.Exec(ctx,
		`UPDATE framework_contracts
		 SET used_value = (
		     SELECT COALESCE(SUM(amount), 0)
		     FROM framework_contract_calls
		     WHERE contract_id = $1 AND tenant_id = $2
		 ), updated_at = NOW()
		 WHERE id = $1 AND tenant_id = $2`,
		call.ContractID, call.TenantID,
	); err != nil {
		return "", false, fmt.Errorf("update contract used_value: %w", err)
	}
	return remaining, fits, nil
}

// ApplyGoodsReceipt implements Repository.ApplyGoodsReceipt.
//
// The PO row is locked first, so two receipts of the same PO run one after the
// other and the second sees the quantities the first committed: the call-off
// value is derived from the received_quantity the UPDATE replaces, never from
// a read the service made before the transaction.
func (r *PostgresRepository) ApplyGoodsReceipt(ctx context.Context, receipt *GoodsReceipt) (*ReceiptCallOff, map[uuid.UUID]float64, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("begin goods receipt tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var status POStatus
	var contractID *uuid.UUID
	var currency, poNumber string
	err = tx.QueryRow(ctx,
		`SELECT status, framework_contract_id, currency, po_number
		 FROM purchase_orders
		 WHERE id = $1 AND tenant_id = $2
		 FOR UPDATE`,
		receipt.POID, receipt.TenantID,
	).Scan(&status, &contractID, &currency, &poNumber)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, ErrPONotFound
	}
	if err != nil {
		return nil, nil, fmt.Errorf("lock purchase order: %w", err)
	}
	if !isReceivable(status) {
		return nil, nil, ErrPONotReceivable
	}

	ids := make([]uuid.UUID, len(receipt.Lines))
	qtys := make([]string, len(receipt.Lines))
	for i, l := range receipt.Lines {
		ids[i], qtys[i] = l.LineID, l.ReceivedQuantity
	}
	// prev reads the quantities this statement replaces (same snapshot, taken
	// after the lock), so each line contributes unit_price * (new - old) to the
	// call-off and (new - old) to the stock, and nothing for a correction
	// downwards.
	// lean: no reversing call-off for a downward correction; add one when
	// received quantities get corrected in practice.
	rows, err := tx.Query(ctx,
		`WITH input AS (
		     SELECT * FROM unnest($3::uuid[], $4::numeric[]) AS t(id, qty)
		 ), prev AS (
		     SELECT id, received_quantity FROM po_lines WHERE po_id = $1 AND tenant_id = $2
		 ), upd AS (
		     UPDATE po_lines l
		     SET received_quantity = i.qty, updated_at = NOW()
		     FROM input i JOIN prev p ON p.id = i.id
		     WHERE l.id = i.id AND l.tenant_id = $2
		     RETURNING l.id, GREATEST(i.qty - p.received_quantity, 0) AS delta,
		               l.unit_price * GREATEST(i.qty - p.received_quantity, 0) AS value
		 )
		 SELECT id, delta::float8, ROUND(SUM(value) OVER (), 4)::text, SUM(value) OVER () > 0 FROM upd`,
		receipt.POID, receipt.TenantID, ids, qtys,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("update received quantities: %w", err)
	}
	defer rows.Close()
	updated := 0
	value, anything := "0", false
	deltas := make(map[uuid.UUID]float64, len(receipt.Lines))
	for rows.Next() {
		var id uuid.UUID
		var delta float64
		if scanErr := rows.Scan(&id, &delta, &value, &anything); scanErr != nil {
			return nil, nil, fmt.Errorf("scan received quantities: %w", scanErr)
		}
		updated++
		if delta > 0 {
			deltas[id] = delta
		}
	}
	if err = rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("update received quantities: %w", err)
	}
	rows.Close()
	if updated != len(receipt.Lines) {
		return nil, nil, ErrPOLineNotFound
	}

	if _, err = tx.Exec(ctx,
		`UPDATE purchase_orders SET status = $1, updated_at = NOW() WHERE id = $2 AND tenant_id = $3`,
		receipt.Status, receipt.POID, receipt.TenantID,
	); err != nil {
		return nil, nil, fmt.Errorf("update po status: %w", err)
	}

	var result *ReceiptCallOff
	if contractID != nil && anything {
		result = &ReceiptCallOff{ContractID: *contractID, Amount: value}
		now := time.Now()
		call := &FrameworkContractCall{
			ID:         uuid.New(),
			TenantID:   receipt.TenantID,
			ContractID: *contractID,
			POID:       &receipt.POID,
			Amount:     value,
			Currency:   currency,
			CalledAt:   now,
			Notes:      fmt.Sprintf("goods receipt PO %s", poNumber),
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		remaining, fits, bookErr := bookContractCall(ctx, tx, call, true)
		switch {
		case errors.Is(bookErr, ErrContractNotFound), errors.Is(bookErr, ErrContractNotActive),
			errors.Is(bookErr, ErrContractCurrencyMismatch):
			result.Warning = "call-off skipped: " + bookErr.Error()
		case bookErr != nil:
			return nil, nil, bookErr
		default:
			result.CallID = &call.ID
			result.Remaining = remaining
			if !fits {
				result.BudgetExceeded = true
				result.Warning = fmt.Sprintf("call-off of %s exceeds the contract's remaining value %s", value, remaining)
			}
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, nil, fmt.Errorf("commit goods receipt: %w", err)
	}
	return result, deltas, nil
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

