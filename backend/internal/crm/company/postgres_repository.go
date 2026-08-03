package company

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kmuhub/kmuhub/internal/models"
)

// PostgresRepository implements Repository using PostgreSQL
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgreSQL repository
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) Create(ctx context.Context, company *models.Company) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO companies (id, name, domain, industry, employee_count, address, city, country, notes, tenant_id, created_by, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
		company.ID, company.Name, company.Domain, company.Industry, company.EmployeeCount,
		company.Address, company.City, company.Country, company.Notes,
		company.TenantID,
		company.CreatedBy, company.CreatedAt, company.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) (*models.Company, error) {
	return r.scanCompany(r.pool.QueryRow(ctx,
		`SELECT id, name, domain, industry, employee_count, address, city, country, notes, merged_into_id, tenant_id, created_by, created_at, updated_at
		 FROM companies WHERE id = $1 AND tenant_id = $2`, id, tenantID,
	))
}

// GetByName finds a non-merged company by case-insensitive exact name match within a tenant.
// Used by contact import, where a bare company name string is the only signal available.
func (r *PostgresRepository) GetByName(ctx context.Context, tenantID uuid.UUID, name string) (*models.Company, error) {
	return r.scanCompany(r.pool.QueryRow(ctx,
		`SELECT id, name, domain, industry, employee_count, address, city, country, notes, merged_into_id, tenant_id, created_by, created_at, updated_at
		 FROM companies
		 WHERE tenant_id = $1 AND LOWER(name) = LOWER($2) AND merged_into_id IS NULL
		 ORDER BY created_at ASC
		 LIMIT 1`, tenantID, name,
	))
}

// GetNamesByIDs returns company names keyed by ID for the given IDs, scoped to a tenant.
// Used by contact export to resolve company_id -> name in a single query instead of one per contact.
func (r *PostgresRepository) GetNamesByIDs(ctx context.Context, ids []uuid.UUID, tenantID uuid.UUID) (map[uuid.UUID]string, error) {
	names := make(map[uuid.UUID]string, len(ids))
	if len(ids) == 0 {
		return names, nil
	}

	rows, err := r.pool.Query(ctx,
		`SELECT id, name FROM companies WHERE id = ANY($1) AND tenant_id = $2`, ids, tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var id uuid.UUID
		var name string
		if scanErr := rows.Scan(&id, &name); scanErr != nil {
			return nil, scanErr
		}
		names[id] = name
	}
	return names, rows.Err()
}

func (r *PostgresRepository) List(ctx context.Context, filter ListFilter, offset, limit int) ([]*models.Company, int, error) {
	// Build WHERE clause
	var conditions []string
	var args []any
	argNum := 1

	// Always filter by tenant
	conditions = append(conditions, fmt.Sprintf("tenant_id = $%d", argNum))
	args = append(args, filter.TenantID)
	argNum++

	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(LOWER(name) LIKE $%d OR LOWER(domain) LIKE $%d)", argNum, argNum))
		args = append(args, "%"+strings.ToLower(filter.Search)+"%")
		argNum++
	}

	if filter.Industry != nil {
		conditions = append(conditions, fmt.Sprintf("industry = $%d", argNum))
		args = append(args, *filter.Industry)
		argNum++
	}

	if len(filter.TagIDs) > 0 {
		// Subquery to find companies with all specified tags
		conditions = append(conditions, fmt.Sprintf(`id IN (
			SELECT company_id FROM company_tags WHERE tag_id = ANY($%d)
			GROUP BY company_id HAVING COUNT(DISTINCT tag_id) = $%d
		)`, argNum, argNum+1))
		args = append(args, filter.TagIDs, len(filter.TagIDs))
		argNum += 2
	}

	whereClause := "WHERE " + strings.Join(conditions, " AND ")

	// Count total
	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM companies %s", whereClause)
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Build ORDER BY
	orderBy := "created_at"
	if filter.SortBy == "name" {
		orderBy = "LOWER(name)"
	}
	direction := "DESC"
	if !filter.SortDesc {
		direction = "ASC"
	}

	// Query with pagination
	query := fmt.Sprintf(`
		SELECT id, name, domain, industry, employee_count, address, city, country, notes, merged_into_id, tenant_id, created_by, created_at, updated_at
		FROM companies %s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, whereClause, orderBy, direction, argNum, argNum+1)

	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var companies []*models.Company
	for rows.Next() {
		company, scanErr := r.scanCompanyFromRows(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		companies = append(companies, company)
	}

	return companies, total, rows.Err()
}

func (r *PostgresRepository) Update(ctx context.Context, company *models.Company, tenantID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE companies SET name = $1, domain = $2, industry = $3, employee_count = $4,
		 address = $5, city = $6, country = $7, notes = $8, updated_at = $9
		 WHERE id = $10 AND tenant_id = $11`,
		company.Name, company.Domain, company.Industry, company.EmployeeCount,
		company.Address, company.City, company.Country, company.Notes,
		company.UpdatedAt, company.ID, tenantID,
	)
	return err
}

func (r *PostgresRepository) Delete(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM companies WHERE id = $1 AND tenant_id = $2`, id, tenantID)
	return err
}

func (r *PostgresRepository) GetContactCount(ctx context.Context, companyID uuid.UUID, tenantID uuid.UUID) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM contacts WHERE company_id = $1 AND tenant_id = $2`, companyID, tenantID,
	).Scan(&count)
	return count, err
}

func (r *PostgresRepository) GetTags(ctx context.Context, companyID uuid.UUID) ([]*models.Tag, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT t.id, t.name, t.color, t.entity_type, t.created_at
		 FROM tags t
		 JOIN company_tags ct ON t.id = ct.tag_id
		 WHERE ct.company_id = $1
		 ORDER BY t.name`, companyID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []*models.Tag
	for rows.Next() {
		var tag models.Tag
		if scanErr := rows.Scan(&tag.ID, &tag.Name, &tag.Color, &tag.EntityType, &tag.CreatedAt); scanErr != nil {
			return nil, scanErr
		}
		tags = append(tags, &tag)
	}
	return tags, rows.Err()
}

func (r *PostgresRepository) AddTags(ctx context.Context, companyID uuid.UUID, tagIDs []uuid.UUID) error {
	if len(tagIDs) == 0 {
		return nil
	}
	_, err := r.pool.Exec(ctx,
		`INSERT INTO company_tags (company_id, tag_id, tenant_id)
		 SELECT $1, unnest($2::uuid[]),
		        (SELECT tenant_id FROM companies WHERE id = $1)
		 ON CONFLICT DO NOTHING`,
		companyID, tagIDs,
	)
	return err
}

func (r *PostgresRepository) RemoveTags(ctx context.Context, companyID uuid.UUID, tagIDs []uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM company_tags WHERE company_id = $1 AND tag_id = ANY($2)`,
		companyID, tagIDs,
	)
	return err
}

func (r *PostgresRepository) GetCustomFieldValues(ctx context.Context, companyID uuid.UUID) ([]*models.CustomFieldValueRow, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT cfv.field_id, cfd.field_name, cfv.value
		 FROM company_custom_field_values cfv
		 JOIN custom_field_definitions cfd ON cfv.field_id = cfd.id
		 WHERE cfv.company_id = $1`, companyID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var values []*models.CustomFieldValueRow
	for rows.Next() {
		var v models.CustomFieldValueRow
		var valueJSON []byte
		if scanErr := rows.Scan(&v.FieldID, &v.FieldName, &valueJSON); scanErr != nil {
			return nil, scanErr
		}
		if unmarshalErr := json.Unmarshal(valueJSON, &v.Value); unmarshalErr != nil {
			return nil, unmarshalErr
		}
		values = append(values, &v)
	}
	return values, rows.Err()
}

func (r *PostgresRepository) SetCustomFieldValues(ctx context.Context, companyID uuid.UUID, values map[uuid.UUID]any) error {
	for fieldID, value := range values {
		valueJSON, err := json.Marshal(value)
		if err != nil {
			return err
		}
		_, execErr := r.pool.Exec(ctx,
			`INSERT INTO company_custom_field_values (company_id, field_id, value, created_at, updated_at)
			 VALUES ($1, $2, $3, NOW(), NOW())
			 ON CONFLICT (company_id, field_id) DO UPDATE SET value = $3, updated_at = NOW()`,
			companyID, fieldID, valueJSON,
		)
		if execErr != nil {
			return execErr
		}
	}
	return nil
}

func (r *PostgresRepository) HasContacts(ctx context.Context, companyID uuid.UUID, tenantID uuid.UUID) (bool, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM contacts WHERE company_id = $1 AND tenant_id = $2 LIMIT 1`, companyID, tenantID,
	).Scan(&count)
	return count > 0, err
}

func (r *PostgresRepository) TagExists(ctx context.Context, tagID uuid.UUID, entityType models.EntityType) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM tags WHERE id = $1 AND entity_type = $2)`,
		tagID, entityType,
	).Scan(&exists)
	return exists, err
}

// FindDuplicateCandidates finds potential duplicate companies using domain and trigram name matching.
func (r *PostgresRepository) FindDuplicateCandidates(ctx context.Context, companyID uuid.UUID, tenantID uuid.UUID) ([]*DuplicateCandidate, error) {
	src, err := r.GetByID(ctx, companyID, tenantID)
	if err != nil {
		return nil, err
	}

	var candidates []*DuplicateCandidate

	// 1. Domain exact match
	if src.Domain != nil && *src.Domain != "" {
		rows, queryErr := r.pool.Query(ctx,
			`SELECT id, name, domain, industry, employee_count, address, city, country, notes, merged_into_id, tenant_id, created_by, created_at, updated_at
			 FROM companies
			 WHERE LOWER(domain) = LOWER($1) AND id != $2 AND merged_into_id IS NULL AND tenant_id = $3
			 LIMIT 10`,
			*src.Domain, companyID, tenantID,
		)
		if queryErr == nil {
			defer rows.Close()
			for rows.Next() {
				c, scanErr := r.scanCompanyFromRows(rows)
				if scanErr == nil {
					candidates = append(candidates, &DuplicateCandidate{
						Company:    &models.CompanyWithRelations{Company: *c},
						Similarity: 1.0,
						MatchType:  "domain_exact",
					})
				}
			}
		}
	}

	// 2. Trigram name match (>= 0.7 similarity)
	rows2, queryErr2 := r.pool.Query(ctx,
		`SELECT id, name, domain, industry, employee_count, address, city, country, notes, merged_into_id, tenant_id, created_by, created_at, updated_at,
		        similarity(lower(name), lower($1)) AS sim
		 FROM companies
		 WHERE similarity(lower(name), lower($1)) >= 0.7
		   AND id != $2
		   AND merged_into_id IS NULL
		   AND tenant_id = $3
		 ORDER BY sim DESC
		 LIMIT 10`,
		src.Name, companyID, tenantID,
	)
	if queryErr2 == nil {
		defer rows2.Close()
		for rows2.Next() {
			var c models.Company
			var sim float64
			scanErr := rows2.Scan(
				&c.ID, &c.Name, &c.Domain, &c.Industry, &c.EmployeeCount,
				&c.Address, &c.City, &c.Country, &c.Notes, &c.MergedIntoID,
				&c.TenantID,
				&c.CreatedBy, &c.CreatedAt, &c.UpdatedAt,
				&sim,
			)
			if scanErr == nil {
				alreadyAdded := false
				for _, existing := range candidates {
					if existing.Company.ID == c.ID {
						alreadyAdded = true
						break
					}
				}
				if !alreadyAdded {
					candidates = append(candidates, &DuplicateCandidate{
						Company:    &models.CompanyWithRelations{Company: c},
						Similarity: sim,
						MatchType:  "name_fuzzy",
					})
				}
			}
		}
	}

	for i := 1; i < len(candidates); i++ {
		for j := i; j > 0 && candidates[j].Similarity > candidates[j-1].Similarity; j-- {
			candidates[j], candidates[j-1] = candidates[j-1], candidates[j]
		}
	}
	if len(candidates) > 10 {
		candidates = candidates[:10]
	}

	return candidates, nil
}

// MergeInto reassigns all related records from duplicateID to primaryID, then soft-deletes duplicate.
func (r *PostgresRepository) MergeInto(ctx context.Context, primaryID, duplicateID uuid.UUID, tenantID uuid.UUID) error {
	// Verify both companies belong to the same tenant
	var primaryTenant, dupTenant uuid.UUID
	if err := r.pool.QueryRow(ctx, `SELECT tenant_id FROM companies WHERE id = $1`, primaryID).Scan(&primaryTenant); err != nil {
		return fmt.Errorf("verify primary tenant: %w", err)
	}
	if err := r.pool.QueryRow(ctx, `SELECT tenant_id FROM companies WHERE id = $1`, duplicateID).Scan(&dupTenant); err != nil {
		return fmt.Errorf("verify duplicate tenant: %w", err)
	}
	if primaryTenant != tenantID || dupTenant != tenantID {
		return fmt.Errorf("companies do not belong to the specified tenant")
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Reassign contacts
	if _, err := tx.Exec(ctx,
		`UPDATE contacts SET company_id = $1 WHERE company_id = $2`,
		primaryID, duplicateID,
	); err != nil {
		return fmt.Errorf("reassign contacts: %w", err)
	}

	// Reassign activities
	if _, err := tx.Exec(ctx,
		`UPDATE activities SET company_id = $1 WHERE company_id = $2`,
		primaryID, duplicateID,
	); err != nil {
		return fmt.Errorf("reassign activities: %w", err)
	}

	// Reassign deals
	if _, err := tx.Exec(ctx,
		`UPDATE deals SET company_id = $1 WHERE company_id = $2`,
		primaryID, duplicateID,
	); err != nil {
		return fmt.Errorf("reassign deals: %w", err)
	}

	// Merge tags
	if _, err := tx.Exec(ctx,
		`INSERT INTO company_tags (company_id, tag_id)
		 SELECT $1, tag_id FROM company_tags WHERE company_id = $2
		 ON CONFLICT DO NOTHING`,
		primaryID, duplicateID,
	); err != nil {
		return fmt.Errorf("merge tags: %w", err)
	}

	// Merge custom fields
	if _, err := tx.Exec(ctx,
		`INSERT INTO company_custom_field_values (company_id, field_id, value, created_at, updated_at)
		 SELECT $1, field_id, value, NOW(), NOW() FROM company_custom_field_values
		 WHERE company_id = $2
		 ON CONFLICT (company_id, field_id) DO NOTHING`,
		primaryID, duplicateID,
	); err != nil {
		return fmt.Errorf("merge custom fields: %w", err)
	}

	// Soft-delete duplicate
	if _, err := tx.Exec(ctx,
		`UPDATE companies SET merged_into_id = $1, updated_at = NOW() WHERE id = $2`,
		primaryID, duplicateID,
	); err != nil {
		return fmt.Errorf("soft-delete duplicate: %w", err)
	}

	return tx.Commit(ctx)
}

// Helper to scan a single row into Company
func (r *PostgresRepository) scanCompany(row pgx.Row) (*models.Company, error) {
	var c models.Company
	err := row.Scan(
		&c.ID, &c.Name, &c.Domain, &c.Industry, &c.EmployeeCount,
		&c.Address, &c.City, &c.Country, &c.Notes, &c.MergedIntoID,
		&c.TenantID,
		&c.CreatedBy, &c.CreatedAt, &c.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrCompanyNotFound
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// Helper to scan rows iterator
func (r *PostgresRepository) scanCompanyFromRows(rows pgx.Rows) (*models.Company, error) {
	var c models.Company
	err := rows.Scan(
		&c.ID, &c.Name, &c.Domain, &c.Industry, &c.EmployeeCount,
		&c.Address, &c.City, &c.Country, &c.Notes, &c.MergedIntoID,
		&c.TenantID,
		&c.CreatedBy, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &c, nil
}
