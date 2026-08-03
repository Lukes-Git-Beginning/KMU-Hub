package repository

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
)

// ManifestRepository handles plugin manifest persistence
type ManifestRepository struct {
	pool *pgxpool.Pool
}

// NewManifestRepository creates a new manifest repository
func NewManifestRepository(pool *pgxpool.Pool) *ManifestRepository {
	return &ManifestRepository{pool: pool}
}

// Create inserts a new plugin manifest
func (r *ManifestRepository) Create(ctx context.Context, m *models.PluginManifest) error {
	hookRegsJSON, err := json.Marshal(m.HookRegistrations)
	if err != nil {
		return err
	}

	_, err = r.pool.Exec(ctx, `
		INSERT INTO plugin_manifests (id, tenant_id, slug, name, description, version, author, settings_schema, permissions, hook_registrations, wasm_binary, wasm_binary_hash, plugin_type, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`,
		m.ID, m.TenantID, m.Slug, m.Name, m.Description, m.Version, m.Author,
		m.SettingsSchema, m.Permissions, hookRegsJSON,
		m.WASMBinary, m.WASMBinaryHash, m.PluginType, m.CreatedAt, m.UpdatedAt,
	)
	return err
}

// GetByID retrieves a manifest by ID.
//
// wasm_binary_hash is nullable with no default, and PluginManifest holds it as
// a plain string: any row inserted without it — a catalogue manifest seeded by
// migration, say — would otherwise fail the scan and take the read path down
// for every tenant. COALESCE on all three read queries, not a schema change,
// because the column is legitimately absent for config plugins.
func (r *ManifestRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.PluginManifest, error) {
	m := &models.PluginManifest{}
	var hookRegsJSON []byte
	err := r.pool.QueryRow(ctx, `
		SELECT id, tenant_id, slug, name, description, version, author, settings_schema, permissions, hook_registrations, COALESCE(wasm_binary_hash, '') AS wasm_binary_hash, plugin_type, created_at, updated_at
		FROM plugin_manifests WHERE id = $1`, id,
	).Scan(&m.ID, &m.TenantID, &m.Slug, &m.Name, &m.Description, &m.Version, &m.Author,
		&m.SettingsSchema, &m.Permissions, &hookRegsJSON,
		&m.WASMBinaryHash, &m.PluginType, &m.CreatedAt, &m.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if hookRegsJSON != nil {
		_ = json.Unmarshal(hookRegsJSON, &m.HookRegistrations)
	}
	return m, nil
}

// GetBySlug retrieves a manifest by slug.
//
// Slugs are unique per tenant since 000276, and RLS lets a caller see its own
// manifests plus the catalogue, so a slug can in principle match two rows.
// The tenant's own row wins — a manifest it defined itself is the one it means
// when it names the slug. Ordering by id keeps the choice deterministic.
func (r *ManifestRepository) GetBySlug(ctx context.Context, slug string) (*models.PluginManifest, error) {
	m := &models.PluginManifest{}
	var hookRegsJSON []byte
	err := r.pool.QueryRow(ctx, `
		SELECT id, tenant_id, slug, name, description, version, author, settings_schema, permissions, hook_registrations, COALESCE(wasm_binary_hash, '') AS wasm_binary_hash, plugin_type, created_at, updated_at
		FROM plugin_manifests WHERE slug = $1
		ORDER BY (tenant_id IS NULL), id LIMIT 1`, slug,
	).Scan(&m.ID, &m.TenantID, &m.Slug, &m.Name, &m.Description, &m.Version, &m.Author,
		&m.SettingsSchema, &m.Permissions, &hookRegsJSON,
		&m.WASMBinaryHash, &m.PluginType, &m.CreatedAt, &m.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if hookRegsJSON != nil {
		_ = json.Unmarshal(hookRegsJSON, &m.HookRegistrations)
	}
	return m, nil
}

// List retrieves all manifests with optional type filter
func (r *ManifestRepository) List(ctx context.Context, pluginType string) ([]*models.PluginManifest, error) {
	query := `SELECT id, tenant_id, slug, name, description, version, author, settings_schema, permissions, hook_registrations, COALESCE(wasm_binary_hash, '') AS wasm_binary_hash, plugin_type, created_at, updated_at FROM plugin_manifests`
	args := []any{}

	if pluginType != "" {
		query += ` WHERE plugin_type = $1`
		args = append(args, pluginType)
	}
	query += ` ORDER BY name ASC`

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var manifests []*models.PluginManifest
	for rows.Next() {
		m := &models.PluginManifest{}
		var hookRegsJSON []byte
		if scanErr := rows.Scan(&m.ID, &m.TenantID, &m.Slug, &m.Name, &m.Description, &m.Version, &m.Author,
			&m.SettingsSchema, &m.Permissions, &hookRegsJSON,
			&m.WASMBinaryHash, &m.PluginType, &m.CreatedAt, &m.UpdatedAt,
		); scanErr != nil {
			return nil, scanErr
		}
		if hookRegsJSON != nil {
			_ = json.Unmarshal(hookRegsJSON, &m.HookRegistrations)
		}
		manifests = append(manifests, m)
	}
	return manifests, nil
}

// Delete removes a manifest
func (r *ManifestRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM plugin_manifests WHERE id = $1`, id)
	return err
}

// HasActiveInstallations checks if any active installations exist for a manifest
func (r *ManifestRepository) HasActiveInstallations(ctx context.Context, manifestID uuid.UUID) (bool, error) {
	var count int
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM plugin_installations
		WHERE manifest_id = $1 AND status NOT IN ('uninstalled')`, manifestID,
	).Scan(&count)
	return count > 0, err
}

// GetWASMBinary retrieves the WASM binary for a manifest
func (r *ManifestRepository) GetWASMBinary(ctx context.Context, id uuid.UUID) ([]byte, error) {
	var binary []byte
	err := r.pool.QueryRow(ctx, `SELECT wasm_binary FROM plugin_manifests WHERE id = $1`, id).Scan(&binary)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return binary, err
}
