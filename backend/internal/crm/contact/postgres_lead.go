package contact

// Lead persistence. Leads are contacts rows filtered by lifecycle_stage --
// see migration 000259 and the note at the top of lead.go.

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/kmuhub/kmuhub/internal/models"
)

// leadColumns is shared by the list and the post-update read so both return
// the same shape. company_name resolves the linked company first and falls
// back to the free-text employer a CSV or dialer intake supplied.
const leadColumns = `
	ct.id, ct.first_name, ct.last_name, ct.email, ct.phone,
	ct.company_id, ct.position, ct.notes,
	ct.visibility, ct.owner_id, ct.merged_into_id,
	ct.tenant_id, ct.created_by, ct.created_at, ct.updated_at,
	ct.lifecycle_stage, ct.lead_source, ct.lead_score, ct.lead_temperature,
	ct.lead_status, ct.lead_company,
	COALESCE(co.name, ct.lead_company)`

const leadFrom = `FROM contacts ct LEFT JOIN companies co ON co.id = ct.company_id`

func scanLeadFromRows(rows pgx.Rows) (*models.ContactWithRelations, error) {
	var c models.ContactWithRelations
	err := rows.Scan(
		&c.ID, &c.FirstName, &c.LastName, &c.Email, &c.Phone,
		&c.CompanyID, &c.Position, &c.Notes,
		&c.Visibility, &c.OwnerID, &c.MergedIntoID,
		&c.TenantID, &c.CreatedBy, &c.CreatedAt, &c.UpdatedAt,
		&c.LifecycleStage, &c.LeadSource, &c.LeadScore, &c.LeadTemperature,
		&c.LeadStatus, &c.LeadCompany,
		&c.CompanyName,
	)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// ListLeads returns the tenant's leads, newest first.
func (r *PostgresRepository) ListLeads(ctx context.Context, filter LeadFilter, offset, limit int) ([]*models.ContactWithRelations, int, error) {
	conditions := []string{"ct.tenant_id = $1"}
	args := []any{filter.TenantID}
	argNum := 2

	if filter.Stage != "" {
		conditions = append(conditions, fmt.Sprintf("ct.lifecycle_stage = $%d", argNum))
		args = append(args, filter.Stage)
		argNum++
	} else {
		// The inbox is everything that is not yet an ordinary contact.
		conditions = append(conditions, "ct.lifecycle_stage <> '"+LifecycleCustomer+"'")
	}

	if filter.Status != "" {
		conditions = append(conditions, fmt.Sprintf("ct.lead_status = $%d", argNum))
		args = append(args, filter.Status)
		argNum++
	}

	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf(
			"(LOWER(ct.first_name) LIKE $%d OR LOWER(ct.last_name) LIKE $%d OR LOWER(ct.email) LIKE $%d OR LOWER(ct.lead_company) LIKE $%d)",
			argNum, argNum, argNum, argNum,
		))
		args = append(args, "%"+strings.ToLower(filter.Search)+"%")
		argNum++
	}

	whereClause := "WHERE " + strings.Join(conditions, " AND ")

	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) %s %s", leadFrom, whereClause)
	if err := r.pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := fmt.Sprintf(`SELECT %s %s %s ORDER BY ct.created_at DESC LIMIT $%d OFFSET $%d`,
		leadColumns, leadFrom, whereClause, argNum, argNum+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	leads := make([]*models.ContactWithRelations, 0, limit)
	for rows.Next() {
		lead, scanErr := scanLeadFromRows(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		leads = append(leads, lead)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return leads, total, nil
}

// UpdateLead applies a partial update and returns the row as it now stands.
// The WHERE clause excludes customers on purpose: this endpoint must not be a
// side door for editing ordinary contacts.
func (r *PostgresRepository) UpdateLead(ctx context.Context, id uuid.UUID, tenantID uuid.UUID, patch LeadPatch) (*models.ContactWithRelations, error) {
	setClauses := []string{"updated_at = NOW()"}
	args := []any{}
	argNum := 1

	if patch.Stage != nil {
		setClauses = append(setClauses, fmt.Sprintf("lifecycle_stage = $%d", argNum))
		args = append(args, *patch.Stage)
		argNum++
	}
	if patch.Status != nil {
		setClauses = append(setClauses, fmt.Sprintf("lead_status = $%d", argNum))
		args = append(args, *patch.Status)
		argNum++
	}
	if patch.Source != nil {
		setClauses = append(setClauses, fmt.Sprintf("lead_source = $%d", argNum))
		args = append(args, *patch.Source)
		argNum++
	}
	if patch.Score != nil {
		setClauses = append(setClauses, fmt.Sprintf("lead_score = $%d", argNum))
		args = append(args, *patch.Score)
		argNum++
	}
	switch {
	case patch.ClearTemperature:
		setClauses = append(setClauses, "lead_temperature = NULL")
	case patch.Temperature != nil:
		setClauses = append(setClauses, fmt.Sprintf("lead_temperature = $%d", argNum))
		args = append(args, *patch.Temperature)
		argNum++
	}

	query := fmt.Sprintf(
		`UPDATE contacts SET %s WHERE id = $%d AND tenant_id = $%d AND lifecycle_stage <> '%s'`,
		strings.Join(setClauses, ", "), argNum, argNum+1, LifecycleCustomer,
	)
	args = append(args, id, tenantID)

	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrLeadNotFound
	}

	return r.getLead(ctx, id, tenantID)
}

func (r *PostgresRepository) getLead(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) (*models.ContactWithRelations, error) {
	query := fmt.Sprintf(`SELECT %s %s WHERE ct.id = $1 AND ct.tenant_id = $2`, leadColumns, leadFrom)

	rows, err := r.pool.Query(ctx, query, id, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return nil, ErrLeadNotFound
	}
	lead, err := scanLeadFromRows(rows)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrLeadNotFound
		}
		return nil, err
	}
	return lead, nil
}
