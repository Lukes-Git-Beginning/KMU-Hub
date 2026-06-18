package rapporte

import (
	"context"
	"database/sql"
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
		     reviewed_at, review_note, lat, lon, report_date,
		     weather, temperature, work_start, work_end, break_minutes, project_name,
		     created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20)`,
		report.ID, report.TenantID, report.Title, report.Description, report.Status,
		report.AuthorID, report.ReviewerID, report.ReviewedAt, report.ReviewNote,
		report.Lat, report.Lon, report.ReportDate,
		report.Weather, report.Temperature, report.WorkStart, report.WorkEnd,
		report.BreakMinutes, report.ProjectName,
		report.CreatedAt, report.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) UpdateReport(ctx context.Context, report *WorkReport) error {
	ct, err := r.pool.Exec(ctx,
		`UPDATE work_reports
		 SET title=$1, description=$2, status=$3, reviewer_id=$4, reviewed_at=$5,
		     review_note=$6, lat=$7, lon=$8, report_date=$9,
		     weather=$10, temperature=$11, work_start=$12, work_end=$13,
		     break_minutes=$14, project_name=$15, updated_at=$16
		 WHERE id=$17 AND tenant_id=$18 AND deleted_at IS NULL`,
		report.Title, report.Description, report.Status, report.ReviewerID,
		report.ReviewedAt, report.ReviewNote, report.Lat, report.Lon,
		report.ReportDate,
		report.Weather, report.Temperature, report.WorkStart, report.WorkEnd,
		report.BreakMinutes, report.ProjectName,
		report.UpdatedAt, report.ID, report.TenantID,
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
		        reviewed_at, review_note, lat, lon, report_date,
		        signature_data, signed_at, signed_by,
		        weather, temperature, work_start, work_end, break_minutes, project_name,
		        created_at, updated_at, deleted_at
		 FROM work_reports WHERE id=$1 AND tenant_id=$2 AND deleted_at IS NULL`,
		reportID, tenantID,
	).Scan(
		&rep.ID, &rep.TenantID, &rep.Title, &rep.Description, &rep.Status,
		&rep.AuthorID, &rep.ReviewerID, &rep.ReviewedAt, &rep.ReviewNote,
		&rep.Lat, &rep.Lon, &rep.ReportDate,
		&rep.SignatureData, &rep.SignedAt, &rep.SignedBy,
		&rep.Weather, &rep.Temperature, &rep.WorkStart, &rep.WorkEnd,
		&rep.BreakMinutes, &rep.ProjectName,
		&rep.CreatedAt, &rep.UpdatedAt, &rep.DeletedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrReportNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get work report: %w", err)
	}

	// Load workers
	workers, wErr := r.ListWorkers(ctx, tenantID, rep.ID)
	if wErr != nil {
		return nil, fmt.Errorf("load workers for report %s: %w", rep.ID, wErr)
	}
	rep.Workers = workers

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
		       reviewed_at, review_note, lat, lon, report_date,
		       signature_data, signed_at, signed_by,
		       weather, temperature, work_start, work_end, break_minutes, project_name,
		       created_at, updated_at, deleted_at
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
		    (id, tenant_id, report_id, position, description, quantity, unit, notes,
		     category, article, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		line.ID, line.TenantID, line.ReportID, line.Position,
		line.Description, line.Quantity, line.Unit, line.Notes,
		line.Category, line.Article,
		line.CreatedAt, line.UpdatedAt,
	)
	return err
}

func (r *PostgresRepository) UpdateLine(ctx context.Context, line *ReportLine) error {
	ct, err := r.pool.Exec(ctx,
		`UPDATE report_lines
		 SET position=$1, description=$2, quantity=$3, unit=$4, notes=$5,
		     category=$6, article=$7, updated_at=$8
		 WHERE id=$9 AND tenant_id=$10`,
		line.Position, line.Description, line.Quantity, line.Unit, line.Notes,
		line.Category, line.Article,
		line.UpdatedAt, line.ID, line.TenantID,
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
		`SELECT id, tenant_id, report_id, position, description, quantity, unit, notes,
		        category, article, created_at, updated_at
		 FROM report_lines WHERE id=$1 AND tenant_id=$2`,
		lineID, tenantID,
	).Scan(
		&line.ID, &line.TenantID, &line.ReportID, &line.Position,
		&line.Description, &line.Quantity, &line.Unit, &line.Notes,
		&line.Category, &line.Article,
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
		`SELECT id, tenant_id, report_id, position, description, quantity, unit, notes,
		        category, article, created_at, updated_at
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
			&line.Category, &line.Article,
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
// Workers
// ============================================================================

func (r *PostgresRepository) AddWorker(ctx context.Context, tenantID, reportID uuid.UUID, name, role string, hours float64) (*Worker, error) {
	w := Worker{
		ID:        uuid.New(),
		TenantID:  tenantID,
		ReportID:  reportID,
		Name:      name,
		CreatedAt: time.Now(),
	}
	if role != "" {
		w.Role = sql.NullString{String: role, Valid: true}
	}
	if hours > 0 {
		w.Hours = sql.NullFloat64{Float64: hours, Valid: true}
	}

	_, err := r.pool.Exec(ctx,
		`INSERT INTO report_workers (id, tenant_id, report_id, name, role, hours, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		w.ID, w.TenantID, w.ReportID, w.Name, w.Role, w.Hours, w.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("add worker: %w", err)
	}
	return &w, nil
}

func (r *PostgresRepository) RemoveWorker(ctx context.Context, tenantID, workerID uuid.UUID) error {
	ct, err := r.pool.Exec(ctx,
		`DELETE FROM report_workers WHERE id=$1 AND tenant_id=$2`,
		workerID, tenantID,
	)
	if err != nil {
		return fmt.Errorf("remove worker: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrWorkerNotFound
	}
	return nil
}

func (r *PostgresRepository) ListWorkers(ctx context.Context, tenantID, reportID uuid.UUID) ([]Worker, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, report_id, name, role, hours, created_at
		 FROM report_workers WHERE tenant_id=$1 AND report_id=$2 ORDER BY created_at ASC`,
		tenantID, reportID,
	)
	if err != nil {
		return nil, fmt.Errorf("list workers: %w", err)
	}
	defer rows.Close()

	var workers []Worker
	for rows.Next() {
		var w Worker
		if scanErr := rows.Scan(&w.ID, &w.TenantID, &w.ReportID, &w.Name, &w.Role, &w.Hours, &w.CreatedAt); scanErr != nil {
			return nil, fmt.Errorf("scan worker: %w", scanErr)
		}
		workers = append(workers, w)
	}
	return workers, rows.Err()
}

// ============================================================================
// Measurements
// ============================================================================

func (r *PostgresRepository) CreateMeasurement(ctx context.Context, tenantID uuid.UUID, reportID *uuid.UUID, title, location, measuredBy, measuredAt, notes string) (*Measurement, error) {
	m := Measurement{
		ID:        uuid.New(),
		TenantID:  tenantID,
		ReportID:  reportID,
		Title:     title,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if location != "" {
		m.Location = sql.NullString{String: location, Valid: true}
	}
	if measuredBy != "" {
		m.MeasuredBy = sql.NullString{String: measuredBy, Valid: true}
	}
	if measuredAt != "" {
		m.MeasuredAt = sql.NullString{String: measuredAt, Valid: true}
	}
	if notes != "" {
		m.Notes = sql.NullString{String: notes, Valid: true}
	}

	_, err := r.pool.Exec(ctx,
		`INSERT INTO measurements (id, tenant_id, report_id, title, location, measured_by, measured_at, notes, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		m.ID, m.TenantID, m.ReportID, m.Title, m.Location, m.MeasuredBy, m.MeasuredAt, m.Notes, m.CreatedAt, m.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create measurement: %w", err)
	}
	return &m, nil
}

func (r *PostgresRepository) GetMeasurement(ctx context.Context, tenantID, measurementID uuid.UUID) (*Measurement, error) {
	var m Measurement
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, report_id, title, location, measured_by, measured_at, notes, created_at, updated_at
		 FROM measurements WHERE id=$1 AND tenant_id=$2`,
		measurementID, tenantID,
	).Scan(&m.ID, &m.TenantID, &m.ReportID, &m.Title, &m.Location, &m.MeasuredBy, &m.MeasuredAt, &m.Notes, &m.CreatedAt, &m.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrMeasurementNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get measurement: %w", err)
	}

	// Load positions
	positions, pErr := r.listMeasurementPositions(ctx, tenantID, m.ID)
	if pErr != nil {
		return nil, fmt.Errorf("load positions for measurement %s: %w", m.ID, pErr)
	}
	m.Positions = positions

	return &m, nil
}

func (r *PostgresRepository) ListMeasurements(ctx context.Context, tenantID uuid.UUID, reportID *uuid.UUID, page, pageSize int) ([]Measurement, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 50
	}
	offset := (page - 1) * pageSize

	var conditions []string
	var args []any
	argNum := 1

	conditions = append(conditions, fmt.Sprintf("tenant_id = $%d", argNum))
	args = append(args, tenantID)
	argNum++

	if reportID != nil {
		conditions = append(conditions, fmt.Sprintf("report_id = $%d", argNum))
		args = append(args, *reportID)
		argNum++
	}

	whereClause := "WHERE " + strings.Join(conditions, " AND ")

	var total int
	if err := r.pool.QueryRow(ctx,
		fmt.Sprintf("SELECT COUNT(*) FROM measurements %s", whereClause), args...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count measurements: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT id, tenant_id, report_id, title, location, measured_by, measured_at, notes, created_at, updated_at
		FROM measurements %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argNum, argNum+1)
	args = append(args, pageSize, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list measurements: %w", err)
	}
	defer rows.Close()

	var measurements []Measurement
	for rows.Next() {
		var m Measurement
		if scanErr := rows.Scan(&m.ID, &m.TenantID, &m.ReportID, &m.Title, &m.Location, &m.MeasuredBy, &m.MeasuredAt, &m.Notes, &m.CreatedAt, &m.UpdatedAt); scanErr != nil {
			return nil, 0, fmt.Errorf("scan measurement: %w", scanErr)
		}
		measurements = append(measurements, m)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return measurements, total, nil
}

func (r *PostgresRepository) UpdateMeasurement(ctx context.Context, tenantID, measurementID uuid.UUID, title, location, measuredBy, measuredAt, notes string) (*Measurement, error) {
	var m Measurement
	err := r.pool.QueryRow(ctx,
		`UPDATE measurements
		 SET title=$3, location=$4, measured_by=$5, measured_at=$6, notes=$7, updated_at=NOW()
		 WHERE id=$1 AND tenant_id=$2
		 RETURNING id, tenant_id, report_id, title, location, measured_by, measured_at, notes, created_at, updated_at`,
		measurementID, tenantID,
		title,
		sql.NullString{String: location, Valid: location != ""},
		sql.NullString{String: measuredBy, Valid: measuredBy != ""},
		sql.NullString{String: measuredAt, Valid: measuredAt != ""},
		sql.NullString{String: notes, Valid: notes != ""},
	).Scan(&m.ID, &m.TenantID, &m.ReportID, &m.Title, &m.Location, &m.MeasuredBy, &m.MeasuredAt, &m.Notes, &m.CreatedAt, &m.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrMeasurementNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("update measurement: %w", err)
	}

	positions, pErr := r.listMeasurementPositions(ctx, tenantID, m.ID)
	if pErr != nil {
		return nil, fmt.Errorf("load positions: %w", pErr)
	}
	m.Positions = positions
	return &m, nil
}

func (r *PostgresRepository) DeleteMeasurement(ctx context.Context, tenantID, measurementID uuid.UUID) error {
	ct, err := r.pool.Exec(ctx,
		`DELETE FROM measurements WHERE id=$1 AND tenant_id=$2`,
		measurementID, tenantID,
	)
	if err != nil {
		return fmt.Errorf("delete measurement: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrMeasurementNotFound
	}
	return nil
}

func (r *PostgresRepository) AddMeasurementPosition(ctx context.Context, tenantID, measurementID uuid.UUID, positionNumber int, description, unit string, quantity, unitPrice float64) (*MeasurementPosition, error) {
	p := MeasurementPosition{
		ID:             uuid.New(),
		TenantID:       tenantID,
		MeasurementID:  measurementID,
		PositionNumber: positionNumber,
		Description:    description,
		CreatedAt:      time.Now(),
	}
	if unit != "" {
		p.Unit = sql.NullString{String: unit, Valid: true}
	}
	if quantity != 0 {
		p.Quantity = sql.NullFloat64{Float64: quantity, Valid: true}
	}
	if unitPrice != 0 {
		p.UnitPrice = sql.NullFloat64{Float64: unitPrice, Valid: true}
	}

	_, err := r.pool.Exec(ctx,
		`INSERT INTO measurement_positions (id, tenant_id, measurement_id, position_number, description, unit, quantity, unit_price, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		p.ID, p.TenantID, p.MeasurementID, p.PositionNumber, p.Description, p.Unit, p.Quantity, p.UnitPrice, p.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("add measurement position: %w", err)
	}
	return &p, nil
}

func (r *PostgresRepository) DeleteMeasurementPosition(ctx context.Context, tenantID, positionID uuid.UUID) error {
	ct, err := r.pool.Exec(ctx,
		`DELETE FROM measurement_positions WHERE id=$1 AND tenant_id=$2`,
		positionID, tenantID,
	)
	if err != nil {
		return fmt.Errorf("delete measurement position: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrPositionNotFound
	}
	return nil
}

func (r *PostgresRepository) listMeasurementPositions(ctx context.Context, tenantID, measurementID uuid.UUID) ([]MeasurementPosition, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, tenant_id, measurement_id, position_number, description, unit, quantity, unit_price, created_at
		 FROM measurement_positions WHERE tenant_id=$1 AND measurement_id=$2 ORDER BY position_number ASC`,
		tenantID, measurementID,
	)
	if err != nil {
		return nil, fmt.Errorf("list measurement positions: %w", err)
	}
	defer rows.Close()

	var positions []MeasurementPosition
	for rows.Next() {
		var p MeasurementPosition
		if scanErr := rows.Scan(&p.ID, &p.TenantID, &p.MeasurementID, &p.PositionNumber, &p.Description, &p.Unit, &p.Quantity, &p.UnitPrice, &p.CreatedAt); scanErr != nil {
			return nil, fmt.Errorf("scan measurement position: %w", scanErr)
		}
		positions = append(positions, p)
	}
	return positions, rows.Err()
}

// ============================================================================
// Templates
// ============================================================================

func (r *PostgresRepository) CreateTemplate(ctx context.Context, tenantID uuid.UUID, name, description, category, defaultLinesJSON string) (*ReportTemplate, error) {
	t := ReportTemplate{
		ID:        uuid.New(),
		TenantID:  tenantID,
		Name:      name,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if description != "" {
		t.Description = sql.NullString{String: description, Valid: true}
	}
	if category != "" {
		t.Category = sql.NullString{String: category, Valid: true}
	}
	if defaultLinesJSON != "" {
		t.DefaultLinesJSON = sql.NullString{String: defaultLinesJSON, Valid: true}
	}

	_, err := r.pool.Exec(ctx,
		`INSERT INTO report_templates (id, tenant_id, name, description, category, default_lines, is_active, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		t.ID, t.TenantID, t.Name, t.Description, t.Category,
		t.DefaultLinesJSON, // stored in default_lines JSONB — pgx handles NullString → null JSON
		t.IsActive, t.CreatedAt, t.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create template: %w", err)
	}
	return &t, nil
}

func (r *PostgresRepository) GetTemplate(ctx context.Context, tenantID, templateID uuid.UUID) (*ReportTemplate, error) {
	var t ReportTemplate
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, name, description, category, default_lines::text, is_active, created_at, updated_at
		 FROM report_templates WHERE id=$1 AND tenant_id=$2`,
		templateID, tenantID,
	).Scan(&t.ID, &t.TenantID, &t.Name, &t.Description, &t.Category, &t.DefaultLinesJSON, &t.IsActive, &t.CreatedAt, &t.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrTemplateNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get template: %w", err)
	}
	return &t, nil
}

func (r *PostgresRepository) ListTemplates(ctx context.Context, tenantID uuid.UUID, activeOnly bool, page, pageSize int) ([]ReportTemplate, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 50
	}
	offset := (page - 1) * pageSize

	var conditions []string
	var args []any
	argNum := 1

	conditions = append(conditions, fmt.Sprintf("tenant_id = $%d", argNum))
	args = append(args, tenantID)
	argNum++

	if activeOnly {
		conditions = append(conditions, "is_active = TRUE")
	}

	whereClause := "WHERE " + strings.Join(conditions, " AND ")

	var total int
	if err := r.pool.QueryRow(ctx,
		fmt.Sprintf("SELECT COUNT(*) FROM report_templates %s", whereClause), args...,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count templates: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT id, tenant_id, name, description, category, default_lines::text, is_active, created_at, updated_at
		FROM report_templates %s
		ORDER BY name ASC
		LIMIT $%d OFFSET $%d
	`, whereClause, argNum, argNum+1)
	args = append(args, pageSize, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list templates: %w", err)
	}
	defer rows.Close()

	var templates []ReportTemplate
	for rows.Next() {
		var t ReportTemplate
		if scanErr := rows.Scan(&t.ID, &t.TenantID, &t.Name, &t.Description, &t.Category, &t.DefaultLinesJSON, &t.IsActive, &t.CreatedAt, &t.UpdatedAt); scanErr != nil {
			return nil, 0, fmt.Errorf("scan template: %w", scanErr)
		}
		templates = append(templates, t)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return templates, total, nil
}

func (r *PostgresRepository) UpdateTemplate(ctx context.Context, tenantID, templateID uuid.UUID, name, description, category, defaultLinesJSON string, isActive bool) (*ReportTemplate, error) {
	var t ReportTemplate
	err := r.pool.QueryRow(ctx,
		`UPDATE report_templates
		 SET name=$3, description=$4, category=$5, default_lines=$6, is_active=$7, updated_at=NOW()
		 WHERE id=$1 AND tenant_id=$2
		 RETURNING id, tenant_id, name, description, category, default_lines::text, is_active, created_at, updated_at`,
		templateID, tenantID, name,
		sql.NullString{String: description, Valid: description != ""},
		sql.NullString{String: category, Valid: category != ""},
		sql.NullString{String: defaultLinesJSON, Valid: defaultLinesJSON != ""},
		isActive,
	).Scan(&t.ID, &t.TenantID, &t.Name, &t.Description, &t.Category, &t.DefaultLinesJSON, &t.IsActive, &t.CreatedAt, &t.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrTemplateNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("update template: %w", err)
	}
	return &t, nil
}

func (r *PostgresRepository) DeleteTemplate(ctx context.Context, tenantID, templateID uuid.UUID) error {
	ct, err := r.pool.Exec(ctx,
		`DELETE FROM report_templates WHERE id=$1 AND tenant_id=$2`,
		templateID, tenantID,
	)
	if err != nil {
		return fmt.Errorf("delete template: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrTemplateNotFound
	}
	return nil
}

// ============================================================================
// Scan helpers
// ============================================================================

func (r *PostgresRepository) scanReportFromRows(rows pgx.Rows) (*WorkReport, error) {
	var rep WorkReport
	err := rows.Scan(
		&rep.ID, &rep.TenantID, &rep.Title, &rep.Description, &rep.Status,
		&rep.AuthorID, &rep.ReviewerID, &rep.ReviewedAt, &rep.ReviewNote,
		&rep.Lat, &rep.Lon, &rep.ReportDate,
		&rep.SignatureData, &rep.SignedAt, &rep.SignedBy,
		&rep.Weather, &rep.Temperature, &rep.WorkStart, &rep.WorkEnd,
		&rep.BreakMinutes, &rep.ProjectName,
		&rep.CreatedAt, &rep.UpdatedAt, &rep.DeletedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan work report row: %w", err)
	}
	return &rep, nil
}

// SaveSignature writes signature_data, signed_at and signed_by atomically.
// Returns ErrReportNotFound when no live report with (id, tenant_id) exists.
func (r *PostgresRepository) SaveSignature(ctx context.Context, tenantID, reportID uuid.UUID, signatureData, signedBy string) (*WorkReport, error) {
	var rep WorkReport
	err := r.pool.QueryRow(ctx, `
		UPDATE work_reports
		SET signature_data=$3, signed_at=NOW(), signed_by=$4, updated_at=NOW()
		WHERE id=$1 AND tenant_id=$2 AND deleted_at IS NULL
		RETURNING id, tenant_id, title, description, status, author_id, reviewer_id,
		          reviewed_at, review_note, lat, lon, report_date,
		          signature_data, signed_at, signed_by,
		          weather, temperature, work_start, work_end, break_minutes, project_name,
		          created_at, updated_at, deleted_at`,
		reportID, tenantID, signatureData, signedBy,
	).Scan(
		&rep.ID, &rep.TenantID, &rep.Title, &rep.Description, &rep.Status,
		&rep.AuthorID, &rep.ReviewerID, &rep.ReviewedAt, &rep.ReviewNote,
		&rep.Lat, &rep.Lon, &rep.ReportDate,
		&rep.SignatureData, &rep.SignedAt, &rep.SignedBy,
		&rep.Weather, &rep.Temperature, &rep.WorkStart, &rep.WorkEnd,
		&rep.BreakMinutes, &rep.ProjectName,
		&rep.CreatedAt, &rep.UpdatedAt, &rep.DeletedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrReportNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("save signature: %w", err)
	}
	return &rep, nil
}

// compile-time interface check
var _ Repository = (*PostgresRepository)(nil)
