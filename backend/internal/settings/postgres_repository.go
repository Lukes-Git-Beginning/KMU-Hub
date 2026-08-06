package settings

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRepository implements Repository using pgx.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new settings repository backed by PostgreSQL.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// PostgresRoleChecker implements RoleChecker by querying the user_roles table.
// The auth service is co-located, so we can safely share the same pool.
type PostgresRoleChecker struct {
	pool *pgxpool.Pool
}

// NewPostgresRoleChecker creates a role-checker backed by PostgreSQL.
func NewPostgresRoleChecker(pool *pgxpool.Pool) *PostgresRoleChecker {
	return &PostgresRoleChecker{pool: pool}
}

// IsAdmin returns true when the user has the 'admin' role within the tenant.
// tenant_id scoping is enforced via user_roles join: a user can only have
// roles assigned within their tenant (tenant_id is implicit via users.tenant_id).
func (c *PostgresRoleChecker) IsAdmin(ctx context.Context, tenantID, userID uuid.UUID) (bool, error) {
	const q = `
		SELECT EXISTS(
			SELECT 1
			FROM user_roles ur
			JOIN roles r ON r.id = ur.role_id
			JOIN users u ON u.id = ur.user_id
			WHERE ur.user_id = $1
			  AND u.tenant_id = $2
			  AND r.name = 'admin'
		)
	`
	var exists bool
	err := c.pool.QueryRow(ctx, q, userID, tenantID).Scan(&exists)
	return exists, err
}

// ============================================================================
// Module-lead queries
// ============================================================================

// ListModuleLeads lists module-lead records for a tenant, optionally filtered
// by user_id and/or module_id. All parameters except tenantID are optional.
func (r *PostgresRepository) ListModuleLeads(ctx context.Context, tenantID uuid.UUID, userID *uuid.UUID, moduleID *string) ([]*ModuleLead, error) {
	// Build query dynamically: only add WHERE clauses for non-nil filters.
	// This avoids allocating multiple query strings while keeping prepared-statement safety.
	query := `
		SELECT tenant_id, user_id, module_id, granted_by, granted_at
		FROM tenant_module_leads
		WHERE tenant_id = $1
	`
	args := []any{tenantID}

	if userID != nil {
		args = append(args, *userID)
		query += " AND user_id = $2"
	}
	if moduleID != nil {
		args = append(args, *moduleID)
		if userID != nil {
			query += " AND module_id = $3"
		} else {
			query += " AND module_id = $2"
		}
	}
	query += " ORDER BY module_id, user_id"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var leads []*ModuleLead
	for rows.Next() {
		ml := &ModuleLead{}
		if err := rows.Scan(&ml.TenantID, &ml.UserID, &ml.ModuleID, &ml.GrantedBy, &ml.GrantedAt); err != nil {
			return nil, err
		}
		leads = append(leads, ml)
	}
	return leads, rows.Err()
}

// GetModuleLead retrieves a single module-lead record.
func (r *PostgresRepository) GetModuleLead(ctx context.Context, tenantID, userID uuid.UUID, moduleID string) (*ModuleLead, error) {
	const q = `
		SELECT tenant_id, user_id, module_id, granted_by, granted_at
		FROM tenant_module_leads
		WHERE tenant_id = $1 AND user_id = $2 AND module_id = $3
	`
	ml := &ModuleLead{}
	err := r.pool.QueryRow(ctx, q, tenantID, userID, moduleID).
		Scan(&ml.TenantID, &ml.UserID, &ml.ModuleID, &ml.GrantedBy, &ml.GrantedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return ml, err
}

// GrantModuleLead inserts or updates a module-lead assignment (UPSERT).
func (r *PostgresRepository) GrantModuleLead(ctx context.Context, tenantID, userID uuid.UUID, moduleID string, grantedBy *uuid.UUID) (*ModuleLead, error) {
	const q = `
		INSERT INTO tenant_module_leads (tenant_id, user_id, module_id, granted_by, granted_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (tenant_id, user_id, module_id)
		DO UPDATE SET granted_by = EXCLUDED.granted_by, granted_at = NOW()
		RETURNING tenant_id, user_id, module_id, granted_by, granted_at
	`
	ml := &ModuleLead{}
	err := r.pool.QueryRow(ctx, q, tenantID, userID, moduleID, grantedBy).
		Scan(&ml.TenantID, &ml.UserID, &ml.ModuleID, &ml.GrantedBy, &ml.GrantedAt)
	return ml, err
}

// RevokeModuleLead removes a module-lead assignment. Idempotent — no error if absent.
func (r *PostgresRepository) RevokeModuleLead(ctx context.Context, tenantID, userID uuid.UUID, moduleID string) error {
	const q = `
		DELETE FROM tenant_module_leads
		WHERE tenant_id = $1 AND user_id = $2 AND module_id = $3
	`
	_, err := r.pool.Exec(ctx, q, tenantID, userID, moduleID)
	return err
}

// IsModuleLead checks whether a user is a Modul-Leiter for a specific module.
func (r *PostgresRepository) IsModuleLead(ctx context.Context, tenantID, userID uuid.UUID, moduleID string) (bool, error) {
	const q = `
		SELECT EXISTS(
			SELECT 1 FROM tenant_module_leads
			WHERE tenant_id = $1 AND user_id = $2 AND module_id = $3
		)
	`
	var exists bool
	err := r.pool.QueryRow(ctx, q, tenantID, userID, moduleID).Scan(&exists)
	return exists, err
}

// ListLeadModulesForUser returns all module IDs the user leads in the tenant.
func (r *PostgresRepository) ListLeadModulesForUser(ctx context.Context, tenantID, userID uuid.UUID) ([]string, error) {
	const q = `
		SELECT module_id
		FROM tenant_module_leads
		WHERE tenant_id = $1 AND user_id = $2
		ORDER BY module_id
	`
	rows, err := r.pool.Query(ctx, q, tenantID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var moduleIDs []string
	for rows.Next() {
		var m string
		if err := rows.Scan(&m); err != nil {
			return nil, err
		}
		moduleIDs = append(moduleIDs, m)
	}
	return moduleIDs, rows.Err()
}

// ============================================================================
// Module-access grant queries
// ============================================================================

// ListModuleGrants lists access grants for a tenant, optionally filtered by user
// and/or module. user_name is joined from users for display.
func (r *PostgresRepository) ListModuleGrants(ctx context.Context, tenantID uuid.UUID, userID *uuid.UUID, moduleID *string) ([]*UserModuleGrant, error) {
	query := `
		SELECT g.tenant_id, g.user_id,
		       COALESCE(NULLIF(TRIM(u.first_name || ' ' || u.last_name), ''), u.email, ''),
		       g.module_id, g.granted_by, g.granted_at, g.last_active_at
		FROM user_module_grants g
		JOIN users u ON u.id = g.user_id
		WHERE g.tenant_id = $1
	`
	args := []any{tenantID}
	if userID != nil {
		args = append(args, *userID)
		query += " AND g.user_id = $2"
	}
	if moduleID != nil {
		args = append(args, *moduleID)
		if userID != nil {
			query += " AND g.module_id = $3"
		} else {
			query += " AND g.module_id = $2"
		}
	}
	query += " ORDER BY u.last_name, u.first_name, g.module_id"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var grants []*UserModuleGrant
	for rows.Next() {
		g := &UserModuleGrant{}
		if err := rows.Scan(&g.TenantID, &g.UserID, &g.UserName, &g.ModuleID, &g.GrantedBy, &g.GrantedAt, &g.LastActiveAt); err != nil {
			return nil, err
		}
		grants = append(grants, g)
	}
	return grants, rows.Err()
}

// GrantModuleAccess inserts or updates an access grant (UPSERT) and returns it
// with the joined user_name.
func (r *PostgresRepository) GrantModuleAccess(ctx context.Context, tenantID, userID uuid.UUID, moduleID string, grantedBy *uuid.UUID) (*UserModuleGrant, error) {
	const q = `
		WITH upserted AS (
			INSERT INTO user_module_grants (tenant_id, user_id, module_id, granted_by, granted_at)
			VALUES ($1, $2, $3, $4, NOW())
			ON CONFLICT (tenant_id, user_id, module_id)
			DO UPDATE SET granted_by = EXCLUDED.granted_by, granted_at = NOW()
			RETURNING tenant_id, user_id, module_id, granted_by, granted_at, last_active_at
		)
		SELECT up.tenant_id, up.user_id,
		       COALESCE(NULLIF(TRIM(u.first_name || ' ' || u.last_name), ''), u.email, ''),
		       up.module_id, up.granted_by, up.granted_at, up.last_active_at
		FROM upserted up
		JOIN users u ON u.id = up.user_id
	`
	g := &UserModuleGrant{}
	err := r.pool.QueryRow(ctx, q, tenantID, userID, moduleID, grantedBy).
		Scan(&g.TenantID, &g.UserID, &g.UserName, &g.ModuleID, &g.GrantedBy, &g.GrantedAt, &g.LastActiveAt)
	return g, err
}

// RevokeModuleAccess removes an access grant. Idempotent — no error if absent.
func (r *PostgresRepository) RevokeModuleAccess(ctx context.Context, tenantID, userID uuid.UUID, moduleID string) error {
	const q = `
		DELETE FROM user_module_grants
		WHERE tenant_id = $1 AND user_id = $2 AND module_id = $3
	`
	_, err := r.pool.Exec(ctx, q, tenantID, userID, moduleID)
	return err
}

// BulkRevokeModuleAccess removes multiple (user, module) grants in one statement
// and returns the number of rows actually deleted.
func (r *PostgresRepository) BulkRevokeModuleAccess(ctx context.Context, tenantID uuid.UUID, refs []ModuleGrantRef) (int, error) {
	if len(refs) == 0 {
		return 0, nil
	}
	userIDs := make([]uuid.UUID, len(refs))
	moduleIDs := make([]string, len(refs))
	for i, ref := range refs {
		userIDs[i] = ref.UserID
		moduleIDs[i] = ref.ModuleID
	}
	const q = `
		DELETE FROM user_module_grants
		WHERE tenant_id = $1
		  AND (user_id, module_id) IN (
		      SELECT * FROM unnest($2::uuid[], $3::text[])
		  )
	`
	tag, err := r.pool.Exec(ctx, q, tenantID, userIDs, moduleIDs)
	if err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
}

// ============================================================================
// Tenant settings queries
// ============================================================================

// GetTenantSettings retrieves all setting entries for a module within a tenant.
func (r *PostgresRepository) GetTenantSettings(ctx context.Context, tenantID uuid.UUID, moduleID string) ([]*SettingEntry, error) {
	const q = `
		SELECT key, value
		FROM tenant_settings
		WHERE tenant_id = $1 AND module_id = $2
		ORDER BY key
	`
	rows, err := r.pool.Query(ctx, q, tenantID, moduleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []*SettingEntry
	for rows.Next() {
		e := &SettingEntry{}
		if err := rows.Scan(&e.Key, &e.Value); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// PutTenantSettings upserts setting entries for a module within a tenant.
// Patch semantics: only supplied keys are written; others remain untouched.
func (r *PostgresRepository) PutTenantSettings(ctx context.Context, tenantID uuid.UUID, moduleID string, updatedBy *uuid.UUID, entries []*SettingEntry) ([]*SettingEntry, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	now := time.Now().UTC()
	for _, e := range entries {
		_, err = tx.Exec(ctx, `
			INSERT INTO tenant_settings (tenant_id, module_id, key, value, updated_by, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (tenant_id, module_id, key)
			DO UPDATE SET value = EXCLUDED.value, updated_by = EXCLUDED.updated_by, updated_at = EXCLUDED.updated_at
		`, tenantID, moduleID, e.Key, e.Value, updatedBy, now)
		if err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	// Return the full updated set so the caller can return it to the client.
	return r.GetTenantSettings(ctx, tenantID, moduleID)
}

// ============================================================================
// User settings queries
// ============================================================================

// GetUserSettings retrieves all setting entries for a user/module pair.
func (r *PostgresRepository) GetUserSettings(ctx context.Context, tenantID, userID uuid.UUID, moduleID string) ([]*SettingEntry, error) {
	const q = `
		SELECT key, value
		FROM user_settings
		WHERE tenant_id = $1 AND user_id = $2 AND module_id = $3
		ORDER BY key
	`
	rows, err := r.pool.Query(ctx, q, tenantID, userID, moduleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []*SettingEntry
	for rows.Next() {
		e := &SettingEntry{}
		if err := rows.Scan(&e.Key, &e.Value); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// PutUserSettings upserts setting entries for a user/module pair.
// Patch semantics: only supplied keys are written; others remain untouched.
func (r *PostgresRepository) PutUserSettings(ctx context.Context, tenantID, userID uuid.UUID, moduleID string, entries []*SettingEntry) ([]*SettingEntry, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	now := time.Now().UTC()
	for _, e := range entries {
		_, err = tx.Exec(ctx, `
			INSERT INTO user_settings (tenant_id, user_id, module_id, key, value, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (tenant_id, user_id, module_id, key)
			DO UPDATE SET value = EXCLUDED.value, updated_at = EXCLUDED.updated_at
		`, tenantID, userID, moduleID, e.Key, e.Value, now)
		if err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return r.GetUserSettings(ctx, tenantID, userID, moduleID)
}

// ============================================================================
// Licensing
// ============================================================================

// ListModuleActivations returns module_id → active for the tenant. A module
// without a row is absent from the map and reads as not activated.
func (r *PostgresRepository) ListModuleActivations(ctx context.Context, tenantID uuid.UUID) (map[string]bool, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT module_id, active FROM tenant_module_activations WHERE tenant_id = $1`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[string]bool)
	for rows.Next() {
		var moduleID string
		var active bool
		if err := rows.Scan(&moduleID, &active); err != nil {
			return nil, err
		}
		out[moduleID] = active
	}
	return out, rows.Err()
}

// SetModuleActivation upserts the activation row and returns nothing but the
// error: the caller re-reads seats and catalogue data to build the response.
func (r *PostgresRepository) SetModuleActivation(ctx context.Context, tenantID uuid.UUID, moduleID string, active bool, updatedBy *uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO tenant_module_activations (tenant_id, module_id, active, updated_by, updated_at)
		 VALUES ($1, $2, $3, $4, NOW())
		 ON CONFLICT (tenant_id, module_id)
		 DO UPDATE SET active = EXCLUDED.active, updated_by = EXCLUDED.updated_by, updated_at = NOW()`,
		tenantID, moduleID, active, updatedBy)
	return err
}

// CountGrantsByModule returns module_id → number of users holding a grant,
// scoped to the tenant. Modules nobody holds are absent from the map.
func (r *PostgresRepository) CountGrantsByModule(ctx context.Context, tenantID uuid.UUID) (map[string]int32, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT module_id, COUNT(*)::int FROM user_module_grants
		 WHERE tenant_id = $1 GROUP BY module_id`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[string]int32)
	for rows.Next() {
		var moduleID string
		var count int32
		if err := rows.Scan(&moduleID, &count); err != nil {
			return nil, err
		}
		out[moduleID] = count
	}
	return out, rows.Err()
}

// GetTenantSubscription reads the booked contract off the tenant row. RLS on
// tenants only exposes the caller's own tenant, so a missing row is either a
// bad id or a cross-tenant read attempt — both are ErrNotFound.
func (r *PostgresRepository) GetTenantSubscription(ctx context.Context, tenantID uuid.UUID) (*TenantSubscription, error) {
	var sub TenantSubscription
	var periodEnd *time.Time
	var seats *int32
	err := r.pool.QueryRow(ctx,
		`SELECT plan_type, support_tier, subscription_status, billing_period_end, seat_limit
		 FROM tenants WHERE id = $1`, tenantID,
	).Scan(&sub.PlanType, &sub.SupportTier, &sub.Status, &periodEnd, &seats)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	sub.BillingPeriodEnd = periodEnd
	sub.TotalSeats = seats
	return &sub, nil
}

// ============================================================================
// Value-set overrides (migration 000295)
//
// These rows are ONLY what a tenant changed. The shipped baseline is the Go
// registry in valueset.go, so a tenant that never opened the editor has no
// rows here and still reads a complete list. Every query is tenant-scoped on
// both tables — RLS enforces it too, but a forgotten predicate here would
// surface as a phantom-404 rather than as a leak, which is harder to spot.
// ============================================================================

// scanValueSetOptions loads all options of the given sets in one query and
// returns them keyed by set id, already in display order.
func (r *PostgresRepository) scanValueSetOptions(ctx context.Context, tenantID uuid.UUID, setIDs []uuid.UUID) (map[uuid.UUID][]ValueSetOption, error) {
	out := make(map[uuid.UUID][]ValueSetOption, len(setIDs))
	if len(setIDs) == 0 {
		return out, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT value_set_id, option_key, label, COALESCE(color, ''), sort_order, is_active
		FROM customization_value_set_options
		WHERE tenant_id = $1 AND value_set_id = ANY($2)
		ORDER BY value_set_id, sort_order, option_key
	`, tenantID, setIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var setID uuid.UUID
		var opt ValueSetOption
		if err := rows.Scan(&setID, &opt.Key, &opt.Label, &opt.Color, &opt.Order, &opt.Active); err != nil {
			return nil, err
		}
		out[setID] = append(out[setID], opt)
	}
	return out, rows.Err()
}

// ListValueSetOverrides returns every override the tenant has stored.
func (r *PostgresRepository) ListValueSetOverrides(ctx context.Context, tenantID uuid.UUID) ([]*ValueSet, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, set_key, name
		FROM customization_value_sets
		WHERE tenant_id = $1
		ORDER BY set_key
	`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sets := make([]*ValueSet, 0)
	ids := make([]uuid.UUID, 0)
	byID := make(map[uuid.UUID]*ValueSet)
	for rows.Next() {
		var id uuid.UUID
		set := &ValueSet{}
		if err := rows.Scan(&id, &set.Key, &set.Name); err != nil {
			return nil, err
		}
		sets = append(sets, set)
		ids = append(ids, id)
		byID[id] = set
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	options, err := r.scanValueSetOptions(ctx, tenantID, ids)
	if err != nil {
		return nil, err
	}
	for id, opts := range options {
		byID[id].Options = opts
	}
	return sets, nil
}

// GetValueSetOverride returns one override, or ErrNotFound if the tenant has
// not overridden that key.
func (r *PostgresRepository) GetValueSetOverride(ctx context.Context, tenantID uuid.UUID, setKey string) (*ValueSet, error) {
	var id uuid.UUID
	set := &ValueSet{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, set_key, name
		FROM customization_value_sets
		WHERE tenant_id = $1 AND set_key = $2
	`, tenantID, setKey).Scan(&id, &set.Key, &set.Name)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	options, err := r.scanValueSetOptions(ctx, tenantID, []uuid.UUID{id})
	if err != nil {
		return nil, err
	}
	set.Options = options[id]
	return set, nil
}

// UpsertValueSetOverride writes a tenant's override wholesale: the option list
// sent replaces the stored one. That matches the PUT route and the FE mock's
// upsertValueSet — the editor always submits the full list it is showing.
//
// Options are deleted and re-inserted rather than diffed. Their surrogate ids
// are never referenced (records point at option_key), so churning them costs
// nothing and keeps removal of an option a single obvious statement.
func (r *PostgresRepository) UpsertValueSetOverride(ctx context.Context, tenantID uuid.UUID, createdBy *uuid.UUID, set *ValueSet) (*ValueSet, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	now := time.Now().UTC()
	var setID uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO customization_value_sets (tenant_id, set_key, name, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $5)
		ON CONFLICT (tenant_id, set_key)
		DO UPDATE SET name = EXCLUDED.name, updated_at = EXCLUDED.updated_at
		RETURNING id
	`, tenantID, set.Key, set.Name, createdBy, now).Scan(&setID)
	if err != nil {
		return nil, err
	}

	if _, err = tx.Exec(ctx, `
		DELETE FROM customization_value_set_options
		WHERE tenant_id = $1 AND value_set_id = $2
	`, tenantID, setID); err != nil {
		return nil, err
	}

	for _, opt := range set.Options {
		var color *string
		if opt.Color != "" {
			c := opt.Color
			color = &c
		}
		if _, err = tx.Exec(ctx, `
			INSERT INTO customization_value_set_options
				(tenant_id, value_set_id, option_key, label, color, sort_order, is_active, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8)
		`, tenantID, setID, opt.Key, opt.Label, color, opt.Order, opt.Active, now); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return r.GetValueSetOverride(ctx, tenantID, set.Key)
}

// DeleteValueSetOverride drops the tenant's override. Options go with it via
// the composite FK's ON DELETE CASCADE. For a system key this is a reset to
// the shipped baseline, not a deletion of the list — see Service.DeleteValueSet.
func (r *PostgresRepository) DeleteValueSetOverride(ctx context.Context, tenantID uuid.UUID, setKey string) error {
	tag, err := r.pool.Exec(ctx, `
		DELETE FROM customization_value_sets
		WHERE tenant_id = $1 AND set_key = $2
	`, tenantID, setKey)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
