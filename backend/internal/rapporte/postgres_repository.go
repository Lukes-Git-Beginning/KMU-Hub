package rapporte

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRepository implements Repository using PostgreSQL.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgreSQL-backed repository.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// ============================================================================
// Reports
// ============================================================================

func (r *PostgresRepository) CreateReport(ctx context.Context, report *WorkReport) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO work_reports
		    (id, tenant_id, title, description, status, author_id, reviewer_id,
		     reviewed_at, review_note, lat, lon, report_date, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		report.ID, report.TenantID, report.Title, report.Description, report.Status,
		report.AuthorID, report.ReviewerID, report.ReviewedAt, report.ReviewNote,
		report.Lat, report.Lon, report.ReportDate, report.CreatedAt, report.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) UpdateReport(ctx context.Context, report *WorkReport) error {
	ct, err := r.pool.Exec(ctx,
		`UPDATE work_reports
		 SET title=$1, description=$2, status=$3, reviewer_id=$4, reviewed_at=$5,
		     review_note=$6, lat=$7, lon=$8, report_date=$9, updated_at=$10
		 WHERE id=$11 AND tenant_id=$12 AND deleted_at IS NULL`,
		report.Title, report.Description, report.Status, report.ReviewerID,
		report.ReviewedAt, report.ReviewNote, report.Lat, report.Lon,
		report.ReportDate, report.UpdatedAt, report.ID, report.TenantID,
	)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrReportNotFound
	}
	return nil
}

func (r *PostgresRepository) SoftDeleteReport(ctx context.Context, tenantID, reportID uuid.UUID) error {
	ct, err := r.pool.Exec(ctx,
		`UPDATE work_reports SET deleted_at = NOW() WHERE id=$1 AND tenant_id=$2 AND deleted_at IS NULL`,
		reportID, tenantID,
	)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrReportNotFound
	}
	return nil
}

func (r *PostgresRepository) GetReport(ctx context.Context, tenantID, reportID uuid.UUID) (*WorkReport, error) {
	var rep WorkReport
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, title, description, status, author_id, reviewer_id,
		        reviewed_at, review_note, lat, lon, report_date, created_at, updated_at, deleted_at
		 FROM work_reports WHERE id=$1 AND tenant_id=$2 AND deleted_at IS NULL`,
		reportID, tenantID,
	).Scan(
		&rep.ID, &rep.TenantID, &rep.Title, &rep.Description, &rep.Status,
		&rep.AuthorID, &rep.ReviewerID, &rep.ReviewedAt, &rep.ReviewNote,
		&rep.Lat, &rep.Lon, &rep.ReportDate, &rep.CreatedAt, &rep.UpdatedAt, &rep.DeletedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrReportNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get work report: %w", err)
	}
	return &rep, nil
}

func (r *PostgresRepository) ListReports(ctx context.Context, tenantID uuid.UUID, filter ListReportsFilter, offset, limit int) ([]*WorkReport, int, error) {
	var conditions []string
	var args []any
	argNum := 1

	conditions = append(conditions, fmt.Sprintf("tenant_id = $%d", argNum))
	args = append(args, tenantID)
	argNum++

	conditions = append(conditions, "deleted_at IS NULL")

	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argNum))
		args = append(args, *filter.Status)
		argNum++
	}

	if filter.AuthorID != nil {
		conditions = append(conditions, fmt.Sprintf("author_id = $%d", argNum))
		args = append(args, *filter.AuthorID)
		argNum++
	}

	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("LOWER(title) LIKE $%d", argNum))
		args = append(args, "%"+strings.ToLower(filter.Search)+"%")
		argNum++
	}

	whereClause := "WHERE " + strings.Join(conditions, " AND ")

	var total int
	if err := r.pool.QueryRow(ctx,
		fmt.Sprintf("SELECT COUNT(*) FROM work_reports %s", whereClause), args...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count work reports: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT id, tenant_id, title, description, status, author_id, reviewer_id,
		       reviewed_at, review_note, lat, lon, report_date, created_at, updated_at, deleted_at
		FROM work_reports %s
		ORDER BY report_date DESC, created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argNum, argNum+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list work reports: %w", err)
	}
	defer rows.Close()

	var reports []*WorkReport
	for rows.Next() {
		rep, scanErr := r.scanReportFromRows(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		reports = append(reports, rep)
	}

	return reports, total, rows.Err()
}

// ============================================================================
// Lines
// ============================================================================

func (r *PostgresRepository) CreateLine(ctx context.Context, line *ReportLine) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO report_lines
		    (id, tenant_id, report_id, position, description, quantity, unit, notes, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		line.ID, line.TenantID, line.ReportID, line.Position,
		line.Description, line.Quantity, line.Unit, line.Notes,
		line.CreatedAt, line.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) UpdateLine(ctx context.Context, line *ReportLine) error {
	ct, err := r.pool.Exec(ctx,
		`UPDATE report_lines
		 SET position=$1, description=$2, quantity=$3, unit=$4, notes=$5, updated_at=$6
		 WHERE id=$7 AND tenant_id=$8`,
		line.Position, line.Description, line.Quantity, line.Unit,
		line.Notes, line.UpdatedAt, line.ID, line.TenantID,
	)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrLineNotFound
	}
	return nil
}

func (r *PostgresRepository) DeleteLine(ctx context.Context, tenantID, lineID uuid.UUID) error {
	ct, err := r.pool.Exec(ctx,
		`DELETE FROM report_lines WHERE id=$1 AND tenant_id=$2`,
		lineID, tenantID,
	)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrLineNotFound
	}
	return nil
}

func (r *PostgresRepository) GetLine(ctx context.Context, tenantID, lineID uuid.UUID) (*ReportLine, error) {
	var line ReportLine
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, report_id, position, description, quantity, unit, notes, created_at, updated_at
		 FROM report_lines WHERE id=$1 AND tenant_id=$2`,
		lineID, tenantID,
	).Scan(
		&line.ID, &line.TenantID, &line.ReportID, &line.Position,
		&line.Description, &line.Quantity, &line.Unit, &line.Notes,
		&line.CreatedAt, &line.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrLineNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get report line: %w", err)
	}
	return &line, nil
}

func (r *PostgresRepository) ListLines(ctx context.Context, tenantID, reportID uuid.UUID) ([]*ReportLine, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, report_id, position, description, quantity, unit, notes, created_at, updated_at
		 FROM report_lines WHERE tenant_id=$1 AND report_id=$2 ORDER BY position ASC, created_at ASC`,
		tenantID, reportID,
	)
	if err != nil {
		return nil, fmt.Errorf("list report lines: %w", err)
	}
	defer rows.Close()

	var lines []*ReportLine
	for rows.Next() {
		var line ReportLine
		if scanErr := rows.Scan(
			&line.ID, &line.TenantID, &line.ReportID, &line.Position,
			&line.Description, &line.Quantity, &line.Unit, &line.Notes,
			&line.CreatedAt, &line.UpdatedAt,
		); scanErr != nil {
			return nil, fmt.Errorf("scan report line: %w", scanErr)
		}
		lines = append(lines, &line)
	}

	return lines, rows.Err()
}

// ============================================================================
// Attachments
// ============================================================================

func (r *PostgresRepository) CreateAttachment(ctx context.Context, att *ReportAttachment) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO report_attachments
		    (id, tenant_id, report_id, line_id, filename, content_type, size_bytes, object_key, uploaded_by, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		att.ID, att.TenantID, att.ReportID, att.LineID, att.Filename,
		att.ContentType, att.SizeBytes, att.ObjectKey, att.UploadedBy, att.CreatedAt,
	)
	return err
}

func (r *PostgresRepository) DeleteAttachment(ctx context.Context, tenantID, attachmentID uuid.UUID) error {
	ct, err := r.pool.Exec(ctx,
		`DELETE FROM report_attachments WHERE id=$1 AND tenant_id=$2`,
		attachmentID, tenantID,
	)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrAttachmentNotFound
	}
	return nil
}

func (r *PostgresRepository) GetAttachment(ctx context.Context, tenantID, attachmentID uuid.UUID) (*ReportAttachment, error) {
	var att ReportAttachment
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, report_id, line_id, filename, content_type, size_bytes, object_key, uploaded_by, created_at
		 FROM report_attachments WHERE id=$1 AND tenant_id=$2`,
		attachmentID, tenantID,
	).Scan(
		&att.ID, &att.TenantID, &att.ReportID, &att.LineID, &att.Filename,
		&att.ContentType, &att.SizeBytes, &att.ObjectKey, &att.UploadedBy, &att.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrAttachmentNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get report attachment: %w", err)
	}
	return &att, nil
}

func (r *PostgresRepository) ListAttachments(ctx context.Context, tenantID, reportID uuid.UUID, lineID *uuid.UUID) ([]*ReportAttachment, error) {
	var conditions []string
	var args []any
	argNum := 1

	conditions = append(conditions, fmt.Sprintf("tenant_id = $%d", argNum))
	args = append(args, tenantID)
	argNum++

	conditions = append(conditions, fmt.Sprintf("report_id = $%d", argNum))
	args = append(args, reportID)
	argNum++

	if lineID != nil {
		conditions = append(conditions, fmt.Sprintf("line_id = $%d", argNum))
		args = append(args, *lineID)
	}

	whereClause := "WHERE " + strings.Join(conditions, " AND ")
	query := fmt.Sprintf(
		`SELECT id, tenant_id, report_id, line_id, filename, content_type, size_bytes, object_key, uploaded_by, created_at
		 FROM report_attachments %s ORDER BY created_at ASC`, whereClause,
	)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list report attachments: %w", err)
	}
	defer rows.Close()

	var atts []*ReportAttachment
	for rows.Next() {
		var att ReportAttachment
		if scanErr := rows.Scan(
			&att.ID, &att.TenantID, &att.ReportID, &att.LineID, &att.Filename,
			&att.ContentType, &att.SizeBytes, &att.ObjectKey, &att.UploadedBy, &att.CreatedAt,
		); scanErr != nil {
			return nil, fmt.Errorf("scan report attachment: %w", scanErr)
		}
		atts = append(atts, &att)
	}

	return atts, rows.Err()
}

// ============================================================================
// Atomic state transitions
// ============================================================================

// AtomicApproveReport performs a single UPDATE that both checks the current status
// and sets approved in one round-trip, closing the TOCTOU race window.
func (r *PostgresRepository) AtomicApproveReport(ctx context.Context, tenantID, reportID, reviewerID uuid.UUID, reviewNote string) (bool, error) {
	ct, err := r.pool.Exec(ctx, `
		UPDATE work_reports
		SET status='approved', reviewer_id=$1, reviewed_at=NOW(), review_note=$2, updated_at=NOW()
		WHERE id=$3 AND tenant_id=$4 AND status='submitted' AND deleted_at IS NULL`,
		reviewerID, reviewNote, reportID, tenantID,
	)
	if err != nil {
		return false, err
	}
	return ct.RowsAffected() > 0, nil
}

// AtomicRejectReport performs a single UPDATE that both checks the current status
// and sets rejected in one round-trip, closing the TOCTOU race window.
func (r *PostgresRepository) AtomicRejectReport(ctx context.Context, tenantID, reportID, reviewerID uuid.UUID, reviewNote string) (bool, error) {
	ct, err := r.pool.Exec(ctx, `
		UPDATE work_reports
		SET status='rejected', reviewer_id=$1, reviewed_at=NOW(), review_note=$2, updated_at=NOW()
		WHERE id=$3 AND tenant_id=$4 AND status='submitted' AND deleted_at IS NULL`,
		reviewerID, reviewNote, reportID, tenantID,
	)
	if err != nil {
		return false, err
	}
	return ct.RowsAffected() > 0, nil
}

// GetReportStatsCounts returns a map of status → count via GROUP BY.
func (r *PostgresRepository) GetReportStatsCounts(ctx context.Context, tenantID uuid.UUID) (map[string]int, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT status, COUNT(*) FROM work_reports WHERE tenant_id=$1 AND deleted_at IS NULL GROUP BY status`,
		tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("get report stats counts: %w", err)
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var status string
		var count int
		if scanErr := rows.Scan(&status, &count); scanErr != nil {
			return nil, fmt.Errorf("scan report stats: %w", scanErr)
		}
		counts[status] = count
	}
	return counts, rows.Err()
}

// ============================================================================
// Scan helpers
// ============================================================================

func (r *PostgresRepository) scanReportFromRows(rows pgx.Rows) (*WorkReport, error) {
	var rep WorkReport
	err := rows.Scan(
		&rep.ID, &rep.TenantID, &rep.Title, &rep.Description, &rep.Status,
		&rep.AuthorID, &rep.ReviewerID, &rep.ReviewedAt, &rep.ReviewNote,
		&rep.Lat, &rep.Lon, &rep.ReportDate, &rep.CreatedAt, &rep.UpdatedAt, &rep.DeletedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan work report row: %w", err)
	}
	return &rep, nil
}

// compile-time interface check
var _ Repository = (*PostgresRepository)(nil)
