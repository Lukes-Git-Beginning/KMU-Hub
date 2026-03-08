package contact

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

func (r *PostgresRepository) Create(ctx context.Context, contact *models.Contact) error {
	vis := contact.Visibility
	if vis == "" {
		vis = "shared"
	}
	_, err := r.pool.Exec(ctx,
		`INSERT INTO contacts (id, first_name, last_name, email, phone, company_id, position, notes, visibility, owner_id, created_by, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
		contact.ID, contact.FirstName, contact.LastName, contact.Email, contact.Phone,
		contact.CompanyID, contact.Position, contact.Notes,
		vis, contact.OwnerID,
		contact.CreatedBy, contact.CreatedAt, contact.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Contact, error) {
	return r.scanContact(r.pool.QueryRow(ctx,
		`SELECT id, first_name, last_name, email, phone, company_id, position, notes, visibility, owner_id, merged_into_id, created_by, created_at, updated_at
		 FROM contacts WHERE id = $1`, id,
	))
}

func (r *PostgresRepository) GetByEmail(ctx context.Context, email string) (*models.Contact, error) {
	return r.scanContact(r.pool.QueryRow(ctx,
		`SELECT id, first_name, last_name, email, phone, company_id, position, notes, visibility, owner_id, merged_into_id, created_by, created_at, updated_at
		 FROM contacts WHERE LOWER(email) = LOWER($1)`, email,
	))
}

func (r *PostgresRepository) List(ctx context.Context, filter ListFilter, offset, limit int) ([]*models.Contact, int, error) {
	// Build WHERE clause
	var conditions []string
	var args []any
	argNum := 1

	if filter.CompanyID != nil {
		conditions = append(conditions, fmt.Sprintf("company_id = $%d", argNum))
		args = append(args, *filter.CompanyID)
		argNum++
	}

	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf(
			"(LOWER(first_name) LIKE $%d OR LOWER(last_name) LIKE $%d OR LOWER(email) LIKE $%d)",
			argNum, argNum, argNum,
		))
		args = append(args, "%"+strings.ToLower(filter.Search)+"%")
		argNum++
	}

	if len(filter.TagIDs) > 0 {
		// Subquery to find contacts with all specified tags
		conditions = append(conditions, fmt.Sprintf(`id IN (
			SELECT contact_id FROM contact_tags WHERE tag_id = ANY($%d)
			GROUP BY contact_id HAVING COUNT(DISTINCT tag_id) = $%d
		)`, argNum, argNum+1))
		args = append(args, filter.TagIDs, len(filter.TagIDs))
		argNum += 2
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count total
	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM contacts %s", whereClause)
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Build ORDER BY
	orderBy := "created_at"
	switch filter.SortBy {
	case "first_name":
		orderBy = "LOWER(first_name)"
	case "last_name":
		orderBy = "LOWER(last_name)"
	case "email":
		orderBy = "LOWER(email)"
	}
	direction := "DESC"
	if !filter.SortDesc {
		direction = "ASC"
	}

	// Query with pagination
	query := fmt.Sprintf(`
		SELECT id, first_name, last_name, email, phone, company_id, position, notes, visibility, owner_id, merged_into_id, created_by, created_at, updated_at
		FROM contacts %s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, whereClause, orderBy, direction, argNum, argNum+1)

	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var contacts []*models.Contact
	for rows.Next() {
		contact, scanErr := r.scanContactFromRows(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		contacts = append(contacts, contact)
	}

	return contacts, total, rows.Err()
}

func (r *PostgresRepository) Update(ctx context.Context, contact *models.Contact) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE contacts SET first_name = $1, last_name = $2, email = $3, phone = $4,
		 company_id = $5, position = $6, notes = $7, visibility = $8, owner_id = $9, updated_at = $10
		 WHERE id = $11`,
		contact.FirstName, contact.LastName, contact.Email, contact.Phone,
		contact.CompanyID, contact.Position, contact.Notes,
		contact.Visibility, contact.OwnerID,
		contact.UpdatedAt, contact.ID,
	)
	return err
}

func (r *PostgresRepository) UpdateVisibility(ctx context.Context, contactID uuid.UUID, visibility string, ownerID *uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE contacts SET visibility = $1, owner_id = $2, updated_at = NOW() WHERE id = $3`,
		visibility, ownerID, contactID,
	)
	return err
}

func (r *PostgresRepository) ListWithVisibility(ctx context.Context, userID uuid.UUID, isAdmin bool, filter ListFilter, offset, limit int) ([]*models.Contact, int, error) {
	// Build WHERE clause with visibility filtering
	var conditions []string
	var args []any
	argNum := 1

	// Visibility filter: shared OR owner_id = userID OR isAdmin
	conditions = append(conditions, fmt.Sprintf("(visibility = 'shared' OR owner_id = $%d OR $%d = true)", argNum, argNum+1))
	args = append(args, userID, isAdmin)
	argNum += 2

	if filter.CompanyID != nil {
		conditions = append(conditions, fmt.Sprintf("company_id = $%d", argNum))
		args = append(args, *filter.CompanyID)
		argNum++
	}

	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf(
			"(LOWER(first_name) LIKE $%d OR LOWER(last_name) LIKE $%d OR LOWER(email) LIKE $%d)",
			argNum, argNum, argNum,
		))
		args = append(args, "%"+strings.ToLower(filter.Search)+"%")
		argNum++
	}

	if len(filter.TagIDs) > 0 {
		conditions = append(conditions, fmt.Sprintf(`id IN (
			SELECT contact_id FROM contact_tags WHERE tag_id = ANY($%d)
			GROUP BY contact_id HAVING COUNT(DISTINCT tag_id) = $%d
		)`, argNum, argNum+1))
		args = append(args, filter.TagIDs, len(filter.TagIDs))
		argNum += 2
	}

	if filter.VisibilityFilter != "" {
		conditions = append(conditions, fmt.Sprintf("visibility = $%d", argNum))
		args = append(args, filter.VisibilityFilter)
		argNum++
	}

	whereClause := "WHERE " + strings.Join(conditions, " AND ")

	// Count total
	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM contacts %s", whereClause)
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Build ORDER BY
	orderBy := "created_at"
	switch filter.SortBy {
	case "first_name":
		orderBy = "LOWER(first_name)"
	case "last_name":
		orderBy = "LOWER(last_name)"
	case "email":
		orderBy = "LOWER(email)"
	}
	direction := "DESC"
	if !filter.SortDesc {
		direction = "ASC"
	}

	query := fmt.Sprintf(`
		SELECT id, first_name, last_name, email, phone, company_id, position, notes, visibility, owner_id, merged_into_id, created_by, created_at, updated_at
		FROM contacts %s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, whereClause, orderBy, direction, argNum, argNum+1)

	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var contacts []*models.Contact
	for rows.Next() {
		contact, scanErr := r.scanContactFromRows(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		contacts = append(contacts, contact)
	}

	return contacts, total, rows.Err()
}

func (r *PostgresRepository) ListByIDs(ctx context.Context, ids []uuid.UUID) ([]*models.Contact, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, first_name, last_name, email, phone, company_id, position, notes, visibility, owner_id, merged_into_id, created_by, created_at, updated_at
		 FROM contacts WHERE id = ANY($1)`, ids,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contacts []*models.Contact
	for rows.Next() {
		contact, scanErr := r.scanContactFromRows(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		contacts = append(contacts, contact)
	}
	return contacts, rows.Err()
}

func (r *PostgresRepository) ListAll(ctx context.Context, userID uuid.UUID, isAdmin bool) ([]*models.Contact, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, first_name, last_name, email, phone, company_id, position, notes, visibility, owner_id, merged_into_id, created_by, created_at, updated_at
		 FROM contacts
		 WHERE (visibility = 'shared' OR owner_id = $1 OR $2 = true)
		 ORDER BY created_at DESC`, userID, isAdmin,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contacts []*models.Contact
	for rows.Next() {
		contact, scanErr := r.scanContactFromRows(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		contacts = append(contacts, contact)
	}
	return contacts, rows.Err()
}

func (r *PostgresRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM contacts WHERE id = $1`, id)
	return err
}

func (r *PostgresRepository) GetCompanyName(ctx context.Context, companyID uuid.UUID) (string, error) {
	var name string
	err := r.pool.QueryRow(ctx,
		`SELECT name FROM companies WHERE id = $1`, companyID,
	).Scan(&name)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return name, err
}

func (r *PostgresRepository) GetTags(ctx context.Context, contactID uuid.UUID) ([]*models.Tag, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT t.id, t.name, t.color, t.entity_type, t.created_at
		 FROM tags t
		 JOIN contact_tags ct ON t.id = ct.tag_id
		 WHERE ct.contact_id = $1
		 ORDER BY t.name`, contactID,
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

func (r *PostgresRepository) AddTags(ctx context.Context, contactID uuid.UUID, tagIDs []uuid.UUID) error {
	for _, tagID := range tagIDs {
		_, err := r.pool.Exec(ctx,
			`INSERT INTO contact_tags (contact_id, tag_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
			contactID, tagID,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *PostgresRepository) RemoveTags(ctx context.Context, contactID uuid.UUID, tagIDs []uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM contact_tags WHERE contact_id = $1 AND tag_id = ANY($2)`,
		contactID, tagIDs,
	)
	return err
}

func (r *PostgresRepository) GetCustomFieldValues(ctx context.Context, contactID uuid.UUID) ([]*models.CustomFieldValueRow, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT cfv.field_id, cfd.field_name, cfv.value
		 FROM contact_custom_field_values cfv
		 JOIN custom_field_definitions cfd ON cfv.field_id = cfd.id
		 WHERE cfv.contact_id = $1`, contactID,
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

func (r *PostgresRepository) SetCustomFieldValues(ctx context.Context, contactID uuid.UUID, values map[uuid.UUID]any) error {
	for fieldID, value := range values {
		valueJSON, err := json.Marshal(value)
		if err != nil {
			return err
		}
		_, execErr := r.pool.Exec(ctx,
			`INSERT INTO contact_custom_field_values (contact_id, field_id, value, created_at, updated_at)
			 VALUES ($1, $2, $3, NOW(), NOW())
			 ON CONFLICT (contact_id, field_id) DO UPDATE SET value = $3, updated_at = NOW()`,
			contactID, fieldID, valueJSON,
		)
		if execErr != nil {
			return execErr
		}
	}
	return nil
}

func (r *PostgresRepository) IsInUse(ctx context.Context, id uuid.UUID) (bool, error) {
	// For now, contacts are not in use by anything
	// In the future, check deal_contacts and activities
	return false, nil
}

func (r *PostgresRepository) CompanyExists(ctx context.Context, companyID uuid.UUID) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM companies WHERE id = $1)`, companyID,
	).Scan(&exists)
	return exists, err
}

func (r *PostgresRepository) TagExists(ctx context.Context, tagID uuid.UUID, entityType models.EntityType) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM tags WHERE id = $1 AND entity_type = $2)`,
		tagID, entityType,
	).Scan(&exists)
	return exists, err
}

// FindDuplicateCandidates finds potential duplicate contacts using email, phone, and trigram name matching.
func (r *PostgresRepository) FindDuplicateCandidates(ctx context.Context, contactID uuid.UUID) ([]*DuplicateCandidate, error) {
	// Get source contact
	src, err := r.GetByID(ctx, contactID)
	if err != nil {
		return nil, err
	}

	var candidates []*DuplicateCandidate

	// 1. Email exact match
	if src.Email != nil && *src.Email != "" {
		rows, queryErr := r.pool.Query(ctx,
			`SELECT id, first_name, last_name, email, phone, company_id, position, notes, visibility, owner_id, merged_into_id, created_by, created_at, updated_at
			 FROM contacts
			 WHERE LOWER(email) = LOWER($1) AND id != $2 AND merged_into_id IS NULL
			 LIMIT 10`,
			*src.Email, contactID,
		)
		if queryErr == nil {
			defer rows.Close()
			for rows.Next() {
				c, scanErr := r.scanContactFromRows(rows)
				if scanErr == nil {
					candidates = append(candidates, &DuplicateCandidate{
						Contact:    &models.ContactWithRelations{Contact: *c},
						Similarity: 1.0,
						MatchType:  "email_exact",
					})
				}
			}
		}
	}

	// 2. Trigram name match (>= 0.7 similarity)
	fullName := strings.ToLower(src.FirstName + " " + src.LastName)
	rows2, queryErr2 := r.pool.Query(ctx,
		`SELECT id, first_name, last_name, email, phone, company_id, position, notes, visibility, owner_id, merged_into_id, created_by, created_at, updated_at,
		        similarity(lower(first_name) || ' ' || lower(last_name), $1) AS sim
		 FROM contacts
		 WHERE similarity(lower(first_name) || ' ' || lower(last_name), $1) >= 0.7
		   AND id != $2
		   AND merged_into_id IS NULL
		 ORDER BY sim DESC
		 LIMIT 10`,
		fullName, contactID,
	)
	if queryErr2 == nil {
		defer rows2.Close()
		for rows2.Next() {
			var c models.Contact
			var sim float64
			scanErr := rows2.Scan(
				&c.ID, &c.FirstName, &c.LastName, &c.Email, &c.Phone,
				&c.CompanyID, &c.Position, &c.Notes,
				&c.Visibility, &c.OwnerID, &c.MergedIntoID,
				&c.CreatedBy, &c.CreatedAt, &c.UpdatedAt,
				&sim,
			)
			if scanErr == nil {
				// Skip if already found via email match
				alreadyAdded := false
				for _, existing := range candidates {
					if existing.Contact.ID == c.ID {
						alreadyAdded = true
						break
					}
				}
				if !alreadyAdded {
					candidates = append(candidates, &DuplicateCandidate{
						Contact:    &models.ContactWithRelations{Contact: c},
						Similarity: sim,
						MatchType:  "name_fuzzy",
					})
				}
			}
		}
	}

	// 3. Phone exact match
	if src.Phone != nil && *src.Phone != "" {
		rows3, queryErr3 := r.pool.Query(ctx,
			`SELECT id, first_name, last_name, email, phone, company_id, position, notes, visibility, owner_id, merged_into_id, created_by, created_at, updated_at
			 FROM contacts
			 WHERE phone = $1 AND id != $2 AND merged_into_id IS NULL
			 LIMIT 10`,
			*src.Phone, contactID,
		)
		if queryErr3 == nil {
			defer rows3.Close()
			for rows3.Next() {
				c, scanErr := r.scanContactFromRows(rows3)
				if scanErr == nil {
					alreadyAdded := false
					for _, existing := range candidates {
						if existing.Contact.ID == c.ID {
							alreadyAdded = true
							break
						}
					}
					if !alreadyAdded {
						candidates = append(candidates, &DuplicateCandidate{
							Contact:    &models.ContactWithRelations{Contact: *c},
							Similarity: 0.9,
							MatchType:  "phone_exact",
						})
					}
				}
			}
		}
	}

	// Sort by similarity descending (simple insertion sort for small slices)
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
func (r *PostgresRepository) MergeInto(ctx context.Context, primaryID, duplicateID uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Reassign activities
	if _, err := tx.Exec(ctx,
		`UPDATE activities SET contact_id = $1 WHERE contact_id = $2`,
		primaryID, duplicateID,
	); err != nil {
		return fmt.Errorf("reassign activities: %w", err)
	}

	// Reassign deals
	if _, err := tx.Exec(ctx,
		`UPDATE deals SET contact_id = $1 WHERE contact_id = $2`,
		primaryID, duplicateID,
	); err != nil {
		return fmt.Errorf("reassign deals: %w", err)
	}

	// Merge tags (insert missing, ignore duplicates)
	if _, err := tx.Exec(ctx,
		`INSERT INTO contact_tags (contact_id, tag_id)
		 SELECT $1, tag_id FROM contact_tags WHERE contact_id = $2
		 ON CONFLICT DO NOTHING`,
		primaryID, duplicateID,
	); err != nil {
		return fmt.Errorf("merge tags: %w", err)
	}

	// Merge custom fields (insert missing, ignore duplicates)
	if _, err := tx.Exec(ctx,
		`INSERT INTO contact_custom_field_values (contact_id, field_id, value, created_at, updated_at)
		 SELECT $1, field_id, value, NOW(), NOW() FROM contact_custom_field_values
		 WHERE contact_id = $2
		 ON CONFLICT (contact_id, field_id) DO NOTHING`,
		primaryID, duplicateID,
	); err != nil {
		return fmt.Errorf("merge custom fields: %w", err)
	}

	// Soft-delete duplicate
	if _, err := tx.Exec(ctx,
		`UPDATE contacts SET merged_into_id = $1, updated_at = NOW() WHERE id = $2`,
		primaryID, duplicateID,
	); err != nil {
		return fmt.Errorf("soft-delete duplicate: %w", err)
	}

	return tx.Commit(ctx)
}

// Helper to scan a single row into Contact
func (r *PostgresRepository) scanContact(row pgx.Row) (*models.Contact, error) {
	var c models.Contact
	err := row.Scan(
		&c.ID, &c.FirstName, &c.LastName, &c.Email, &c.Phone,
		&c.CompanyID, &c.Position, &c.Notes,
		&c.Visibility, &c.OwnerID, &c.MergedIntoID,
		&c.CreatedBy, &c.CreatedAt, &c.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrContactNotFound
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// Helper to scan rows iterator
func (r *PostgresRepository) scanContactFromRows(rows pgx.Rows) (*models.Contact, error) {
	var c models.Contact
	err := rows.Scan(
		&c.ID, &c.FirstName, &c.LastName, &c.Email, &c.Phone,
		&c.CompanyID, &c.Position, &c.Notes,
		&c.Visibility, &c.OwnerID, &c.MergedIntoID,
		&c.CreatedBy, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &c, nil
}
