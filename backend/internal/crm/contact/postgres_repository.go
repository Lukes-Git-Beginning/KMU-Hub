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

// contactColumns is the column list behind every contact SELECT in this file.
// It was spelled out nine times until the 000314 profile fields had to be added
// to all of them; scanContactRow reads exactly this order, so the two belong
// together and change together.
//
// Note the lifecycle_stage / lead_* columns are deliberately absent: no read
// path in this repository loads them today.
const contactColumns = `id, first_name, last_name, email, phone, company_id, position, notes,
	       visibility, owner_id, merged_into_id, tenant_id, created_by, created_at, updated_at,
	       salutation, title, mobile, department, address_street, address_zip, address_city,
	       address_country, website, linkedin, xing, category, status`

func (r *PostgresRepository) Create(ctx context.Context, contact *models.Contact) error {
	vis := contact.Visibility
	if vis == "" {
		vis = "shared"
	}
	stage := contact.LifecycleStage
	if stage == "" {
		stage = LifecycleCustomer
	}
	_, err := r.pool.Exec(ctx,
		`INSERT INTO contacts (id, first_name, last_name, email, phone, company_id, position, notes, visibility, owner_id, tenant_id, created_by, created_at, updated_at,
		                       lifecycle_stage, lead_source, lead_score, lead_temperature, lead_status, lead_company,
		                       salutation, title, mobile, department, address_street, address_zip, address_city,
		                       address_country, website, linkedin, xing, category, status)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20,
		         $21, $22, $23, $24, $25, $26, $27, $28, $29, $30, $31, $32, $33)`,
		contact.ID, contact.FirstName, contact.LastName, contact.Email, contact.Phone,
		contact.CompanyID, contact.Position, contact.Notes,
		vis, contact.OwnerID, contact.TenantID,
		contact.CreatedBy, contact.CreatedAt, contact.UpdatedAt,
		stage, contact.LeadSource, contact.LeadScore, contact.LeadTemperature,
		contact.LeadStatus, contact.LeadCompany,
		contact.Salutation, contact.Title, contact.Mobile, contact.Department,
		contact.AddressStreet, contact.AddressZip, contact.AddressCity, contact.AddressCountry,
		contact.Website, contact.LinkedIn, contact.Xing, contact.Category, contact.Status,
	)
	return err
}

func (r *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) (*models.Contact, error) {
	return r.scanContact(r.pool.QueryRow(ctx,
		`SELECT `+contactColumns+`
		 FROM contacts WHERE id = $1 AND tenant_id = $2`, id, tenantID,
	))
}

func (r *PostgresRepository) GetByEmail(ctx context.Context, email string, tenantID uuid.UUID) (*models.Contact, error) {
	return r.scanContact(r.pool.QueryRow(ctx,
		`SELECT `+contactColumns+`
		 FROM contacts WHERE LOWER(email) = LOWER($1) AND tenant_id = $2`, email, tenantID,
	))
}

func (r *PostgresRepository) List(ctx context.Context, filter ListFilter, offset, limit int) ([]*models.Contact, int, error) {
	// Build WHERE clause
	var conditions []string
	var args []any
	argNum := 1

	// Always filter by tenant
	conditions = append(conditions, fmt.Sprintf("tenant_id = $%d", argNum))
	args = append(args, filter.TenantID)
	argNum++

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

	// Query with pagination
	query := fmt.Sprintf(`
		SELECT `+contactColumns+`
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

func (r *PostgresRepository) Update(ctx context.Context, contact *models.Contact, tenantID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE contacts SET first_name = $1, last_name = $2, email = $3, phone = $4,
		 company_id = $5, position = $6, notes = $7, visibility = $8, owner_id = $9, updated_at = $10,
		 salutation = $13, title = $14, mobile = $15, department = $16,
		 address_street = $17, address_zip = $18, address_city = $19, address_country = $20,
		 website = $21, linkedin = $22, xing = $23, category = $24, status = $25
		 WHERE id = $11 AND tenant_id = $12`,
		contact.FirstName, contact.LastName, contact.Email, contact.Phone,
		contact.CompanyID, contact.Position, contact.Notes,
		contact.Visibility, contact.OwnerID,
		contact.UpdatedAt, contact.ID, tenantID,
		contact.Salutation, contact.Title, contact.Mobile, contact.Department,
		contact.AddressStreet, contact.AddressZip, contact.AddressCity, contact.AddressCountry,
		contact.Website, contact.LinkedIn, contact.Xing, contact.Category, contact.Status,
	)
	return err
}

func (r *PostgresRepository) UpdateVisibility(ctx context.Context, contactID uuid.UUID, visibility string, ownerID *uuid.UUID, tenantID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE contacts SET visibility = $1, owner_id = $2, updated_at = NOW() WHERE id = $3 AND tenant_id = $4`,
		visibility, ownerID, contactID, tenantID,
	)
	return err
}

func (r *PostgresRepository) ListWithVisibility(ctx context.Context, userID uuid.UUID, isAdmin bool, filter ListFilter, offset, limit int) ([]*models.Contact, int, error) {
	// Build WHERE clause with visibility filtering
	var conditions []string
	var args []any
	argNum := 1

	// Always filter by tenant
	conditions = append(conditions, fmt.Sprintf("tenant_id = $%d", argNum))
	args = append(args, filter.TenantID)
	argNum++

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
		SELECT `+contactColumns+`
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

func (r *PostgresRepository) ListByIDs(ctx context.Context, ids []uuid.UUID, tenantID uuid.UUID) ([]*models.Contact, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := r.pool.Query(ctx,
		`SELECT `+contactColumns+`
		 FROM contacts WHERE id = ANY($1) AND tenant_id = $2`, ids, tenantID,
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

func (r *PostgresRepository) ListAll(ctx context.Context, userID uuid.UUID, isAdmin bool, tenantID uuid.UUID) ([]*models.Contact, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+contactColumns+`
		 FROM contacts
		 WHERE tenant_id = $1 AND (visibility = 'shared' OR owner_id = $2 OR $3 = true)
		 ORDER BY created_at DESC`, tenantID, userID, isAdmin,
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

func (r *PostgresRepository) Delete(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM contacts WHERE id = $1 AND tenant_id = $2`, id, tenantID)
	return err
}

func (r *PostgresRepository) GetCompanyName(ctx context.Context, companyID uuid.UUID, tenantID uuid.UUID) (string, error) {
	var name string
	err := r.pool.QueryRow(ctx,
		`SELECT name FROM companies WHERE id = $1 AND tenant_id = $2`, companyID, tenantID,
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
	if len(tagIDs) == 0 {
		return nil
	}
	_, err := r.pool.Exec(ctx,
		`INSERT INTO contact_tags (contact_id, tag_id, tenant_id)
		 SELECT $1, unnest($2::uuid[]),
		        (SELECT tenant_id FROM contacts WHERE id = $1)
		 ON CONFLICT DO NOTHING`,
		contactID, tagIDs,
	)
	return err
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
	if len(values) == 0 {
		return nil
	}
	batch := &pgx.Batch{}
	for fieldID, value := range values {
		valueJSON, err := json.Marshal(value)
		if err != nil {
			return err
		}
		batch.Queue(
			`INSERT INTO contact_custom_field_values (contact_id, field_id, value, created_at, updated_at)
			 VALUES ($1, $2, $3, NOW(), NOW())
			 ON CONFLICT (contact_id, field_id) DO UPDATE SET value = $3, updated_at = NOW()`,
			contactID, fieldID, valueJSON,
		)
	}
	br := r.pool.SendBatch(ctx, batch)
	defer br.Close()
	for range values {
		if _, err := br.Exec(); err != nil {
			return err
		}
	}
	return nil
}

// GetCompanyNames fetches company names for multiple company IDs in one query.
func (r *PostgresRepository) GetCompanyNames(ctx context.Context, companyIDs []uuid.UUID, tenantID uuid.UUID) (map[uuid.UUID]string, error) {
	if len(companyIDs) == 0 {
		return nil, nil
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, name FROM companies WHERE id = ANY($1) AND tenant_id = $2`, companyIDs, tenantID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[uuid.UUID]string)
	for rows.Next() {
		var id uuid.UUID
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, err
		}
		result[id] = name
	}
	return result, rows.Err()
}

// GetTagsBatch fetches tags for multiple contact IDs in one query.
func (r *PostgresRepository) GetTagsBatch(ctx context.Context, contactIDs []uuid.UUID) (map[uuid.UUID][]*models.Tag, error) {
	if len(contactIDs) == 0 {
		return nil, nil
	}
	rows, err := r.pool.Query(ctx,
		`SELECT ct.contact_id, t.id, t.name, t.color, t.entity_type, t.created_at
		 FROM tags t
		 JOIN contact_tags ct ON t.id = ct.tag_id
		 WHERE ct.contact_id = ANY($1)
		 ORDER BY t.name`, contactIDs,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[uuid.UUID][]*models.Tag)
	for rows.Next() {
		var contactID uuid.UUID
		var tag models.Tag
		if err := rows.Scan(&contactID, &tag.ID, &tag.Name, &tag.Color, &tag.EntityType, &tag.CreatedAt); err != nil {
			return nil, err
		}
		result[contactID] = append(result[contactID], &tag)
	}
	return result, rows.Err()
}

// GetCustomFieldValuesBatch fetches custom field values for multiple contact IDs in one query.
func (r *PostgresRepository) GetCustomFieldValuesBatch(ctx context.Context, contactIDs []uuid.UUID) (map[uuid.UUID][]*models.CustomFieldValueRow, error) {
	if len(contactIDs) == 0 {
		return nil, nil
	}
	rows, err := r.pool.Query(ctx,
		`SELECT cfv.contact_id, cfv.field_id, cfd.field_name, cfv.value
		 FROM contact_custom_field_values cfv
		 JOIN custom_field_definitions cfd ON cfv.field_id = cfd.id
		 WHERE cfv.contact_id = ANY($1)`, contactIDs,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[uuid.UUID][]*models.CustomFieldValueRow)
	for rows.Next() {
		var contactID uuid.UUID
		var v models.CustomFieldValueRow
		var valueJSON []byte
		if err := rows.Scan(&contactID, &v.FieldID, &v.FieldName, &valueJSON); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(valueJSON, &v.Value); err != nil {
			return nil, err
		}
		result[contactID] = append(result[contactID], &v)
	}
	return result, rows.Err()
}

func (r *PostgresRepository) IsInUse(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) (bool, error) {
	// For now, contacts are not in use by anything
	// In the future, check deal_contacts and activities
	return false, nil
}

func (r *PostgresRepository) CompanyExists(ctx context.Context, companyID uuid.UUID, tenantID uuid.UUID) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM companies WHERE id = $1 AND tenant_id = $2)`, companyID, tenantID,
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
func (r *PostgresRepository) FindDuplicateCandidates(ctx context.Context, contactID uuid.UUID, tenantID uuid.UUID) ([]*DuplicateCandidate, error) {
	// Get source contact
	src, err := r.GetByID(ctx, contactID, tenantID)
	if err != nil {
		return nil, err
	}

	var candidates []*DuplicateCandidate

	// 1. Email exact match
	if src.Email != nil && *src.Email != "" {
		rows, queryErr := r.pool.Query(ctx,
			`SELECT `+contactColumns+`
			 FROM contacts
			 WHERE LOWER(email) = LOWER($1) AND id != $2 AND merged_into_id IS NULL AND tenant_id = $3
			 LIMIT 10`,
			*src.Email, contactID, tenantID,
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
		`SELECT `+contactColumns+`,
		        similarity(lower(first_name) || ' ' || lower(last_name), $1) AS sim
		 FROM contacts
		 WHERE similarity(lower(first_name) || ' ' || lower(last_name), $1) >= 0.7
		   AND id != $2
		   AND merged_into_id IS NULL
		   AND tenant_id = $3
		 ORDER BY sim DESC
		 LIMIT 10`,
		fullName, contactID, tenantID,
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
				&c.TenantID,
				&c.CreatedBy, &c.CreatedAt, &c.UpdatedAt,
				&c.Salutation, &c.Title, &c.Mobile, &c.Department,
				&c.AddressStreet, &c.AddressZip, &c.AddressCity, &c.AddressCountry,
				&c.Website, &c.LinkedIn, &c.Xing, &c.Category, &c.Status,
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
			`SELECT `+contactColumns+`
			 FROM contacts
			 WHERE phone = $1 AND id != $2 AND merged_into_id IS NULL AND tenant_id = $3
			 LIMIT 10`,
			*src.Phone, contactID, tenantID,
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
func (r *PostgresRepository) MergeInto(ctx context.Context, primaryID, duplicateID uuid.UUID, tenantID uuid.UUID) error {
	// Verify both contacts belong to the same tenant
	var primaryTenant, dupTenant uuid.UUID
	if err := r.pool.QueryRow(ctx, `SELECT tenant_id FROM contacts WHERE id = $1`, primaryID).Scan(&primaryTenant); err != nil {
		return fmt.Errorf("verify primary tenant: %w", err)
	}
	if err := r.pool.QueryRow(ctx, `SELECT tenant_id FROM contacts WHERE id = $1`, duplicateID).Scan(&dupTenant); err != nil {
		return fmt.Errorf("verify duplicate tenant: %w", err)
	}
	if primaryTenant != tenantID || dupTenant != tenantID {
		return fmt.Errorf("contacts do not belong to the specified tenant")
	}

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

	// Merge tags (insert missing, ignore duplicates). tenant_id is NOT NULL +
	// RLS-checked on contact_tags, so it must be carried explicitly — the
	// same pattern AddTags uses.
	if _, err := tx.Exec(ctx,
		`INSERT INTO contact_tags (contact_id, tag_id, tenant_id)
		 SELECT $1, tag_id, tenant_id FROM contact_tags WHERE contact_id = $2
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

// scanTarget is what pgx.Row and pgx.Rows have in common. One scan body for
// both keeps the field order in step with contactColumns; two copies had to be
// edited in lockstep and there is no reason for that.
type scanTarget interface {
	Scan(dest ...any) error
}

// scanContactRow reads exactly the columns of contactColumns, in that order.
func scanContactRow(row scanTarget) (*models.Contact, error) {
	var c models.Contact
	err := row.Scan(
		&c.ID, &c.FirstName, &c.LastName, &c.Email, &c.Phone,
		&c.CompanyID, &c.Position, &c.Notes,
		&c.Visibility, &c.OwnerID, &c.MergedIntoID,
		&c.TenantID,
		&c.CreatedBy, &c.CreatedAt, &c.UpdatedAt,
		&c.Salutation, &c.Title, &c.Mobile, &c.Department,
		&c.AddressStreet, &c.AddressZip, &c.AddressCity, &c.AddressCountry,
		&c.Website, &c.LinkedIn, &c.Xing, &c.Category, &c.Status,
	)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// Helper to scan a single row into Contact
func (r *PostgresRepository) scanContact(row pgx.Row) (*models.Contact, error) {
	c, err := scanContactRow(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrContactNotFound
	}
	if err != nil {
		return nil, err
	}
	return c, nil
}

// Helper to scan rows iterator
func (r *PostgresRepository) scanContactFromRows(rows pgx.Rows) (*models.Contact, error) {
	return scanContactRow(rows)
}
