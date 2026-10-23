package gdpr

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ---------------------------------------------------------------------------
// Work reports, shifts and week approvals of a matched user — Art. 15 GDPR.
//
// Read-only. A work report carries GPS coordinates and a signature: the
// signature image (work_reports.signature_data) is never selected, only a
// note saying who signed and when — same line as rentalsModule. Soft-deleted
// reports are still disclosed (the row is still held), flagged by "Gelöscht am".
// Workers listed on a report (report_workers) are free-text names without a
// user link and cannot be attributed to a user here.
// Shifts sit behind a feature flag, but the tables exist regardless.
// ---------------------------------------------------------------------------

func workReportsModule(ctx context.Context, pool *pgxpool.Pool, tenantID, userID uuid.UUID) (*DSARModule, error) {
	columns := []string{"Titel", "Rolle", "Status", "Rapportdatum", "Projekt", "Arbeitsbeginn", "Arbeitsende",
		"Pause (min)", "Breitengrad", "Längengrad", "Beschreibung", "Prüfnotiz", "Unterschrift", "Gelöscht am"}
	return userRowsModule(ctx, pool, "work reports", "Rapporte", columns,
		`SELECT title,
		        CASE WHEN author_id = $2 AND reviewer_id = $2 THEN 'Autor und Prüfer'
		             WHEN author_id = $2 THEN 'Autor' ELSE 'Prüfer' END,
		        status, report_date, COALESCE(project_name, ''),
		        COALESCE(to_char(work_start, 'HH24:MI'), ''), COALESCE(to_char(work_end, 'HH24:MI'), ''),
		        COALESCE(break_minutes::text, ''), COALESCE(lat::text, ''), COALESCE(lon::text, ''),
		        description, review_note,
		        CASE WHEN signature_data IS NOT NULL OR signed_at IS NOT NULL THEN COALESCE(signed_by, '') END,
		        signed_at, deleted_at
		 FROM work_reports
		 WHERE tenant_id = $1 AND (author_id = $2 OR reviewer_id = $2)
		 ORDER BY report_date DESC, created_at DESC
		 LIMIT $3`, []any{tenantID, userID, dsarMaxRows},
		func(rows pgx.Rows) (DSARRecord, error) {
			var title, role, status, project, start, end, pause, lat, lon, description, note string
			var reportDate time.Time
			var signedBy *string
			var signedAt, deletedAt *time.Time
			if err := rows.Scan(&title, &role, &status, &reportDate, &project, &start, &end, &pause,
				&lat, &lon, &description, &note, &signedBy, &signedAt, &deletedAt); err != nil {
				return DSARRecord{}, err
			}
			signature := ""
			if signedBy != nil {
				signature = "Unterschrift vorhanden"
				if *signedBy != "" {
					signature += " von " + *signedBy
				}
				if signedAt != nil {
					signature += " am " + signedAt.Format(dsarTimeLayout)
				}
			}
			return DSARRecord{Fields: []DSARField{
				{Key: "Titel", Value: title},
				{Key: "Rolle", Value: role},
				{Key: "Status", Value: status},
				{Key: "Rapportdatum", Value: reportDate.Format("02.01.2006")},
				{Key: "Projekt", Value: project},
				{Key: "Arbeitsbeginn", Value: start},
				{Key: "Arbeitsende", Value: end},
				{Key: "Pause (min)", Value: pause},
				{Key: "Breitengrad", Value: lat},
				{Key: "Längengrad", Value: lon},
				{Key: "Beschreibung", Value: description},
				{Key: "Prüfnotiz", Value: note},
				{Key: "Unterschrift", Value: signature},
				{Key: "Gelöscht am", Value: derefDateTimeOrEmpty(deletedAt)},
			}}, nil
		})
}

func shiftAssignmentsModule(ctx context.Context, pool *pgxpool.Pool, tenantID, userID uuid.UUID) (*DSARModule, error) {
	columns := []string{"Schicht", "Beginn", "Ende", "Ort", "Zugeteilt am"}
	return userRowsModule(ctx, pool, "shift assignments", "Schichtzuteilungen", columns,
		`SELECT s.title, s.start_time, s.end_time, COALESCE(s.location, ''), a.assigned_at
		 FROM shift_assignments a
		 JOIN shifts s ON s.id = a.shift_id AND s.tenant_id = a.tenant_id
		 WHERE a.tenant_id = $1 AND a.employee_id = $2
		 ORDER BY s.start_time DESC
		 LIMIT $3`, []any{tenantID, userID, dsarMaxRows},
		func(rows pgx.Rows) (DSARRecord, error) {
			var title, location string
			var start, end, assignedAt time.Time
			if err := rows.Scan(&title, &start, &end, &location, &assignedAt); err != nil {
				return DSARRecord{}, err
			}
			return DSARRecord{Fields: []DSARField{
				{Key: "Schicht", Value: title},
				{Key: "Beginn", Value: start.Format(dsarTimeLayout)},
				{Key: "Ende", Value: end.Format(dsarTimeLayout)},
				{Key: "Ort", Value: location},
				{Key: "Zugeteilt am", Value: assignedAt.Format(dsarTimeLayout)},
			}}, nil
		})
}

// shiftSwapRequestsModule lists swap requests the user filed or was named in.
func shiftSwapRequestsModule(ctx context.Context, pool *pgxpool.Pool, tenantID, userID uuid.UUID) (*DSARModule, error) {
	columns := []string{"Schicht", "Rolle", "Status", "Grund", "Angelegt"}
	return userRowsModule(ctx, pool, "shift swap requests", "Schicht-Tauschanfragen", columns,
		`SELECT s.title,
		        CASE WHEN r.requested_by_employee_id = $2 THEN 'Antragsteller' ELSE 'Tauschpartner' END,
		        r.status, r.reason, r.created_at
		 FROM shift_swap_requests r
		 JOIN shifts s ON s.id = r.shift_id AND s.tenant_id = r.tenant_id
		 WHERE r.tenant_id = $1 AND (r.requested_by_employee_id = $2 OR r.swap_with_employee_id = $2)
		 ORDER BY r.created_at DESC
		 LIMIT $3`, []any{tenantID, userID, dsarMaxRows},
		func(rows pgx.Rows) (DSARRecord, error) {
			var title, role, status, reason string
			var createdAt time.Time
			if err := rows.Scan(&title, &role, &status, &reason, &createdAt); err != nil {
				return DSARRecord{}, err
			}
			return DSARRecord{Fields: []DSARField{
				{Key: "Schicht", Value: title},
				{Key: "Rolle", Value: role},
				{Key: "Status", Value: status},
				{Key: "Grund", Value: reason},
				{Key: "Angelegt", Value: createdAt.Format(dsarTimeLayout)},
			}}, nil
		})
}

// weekApprovalsModule lists the weekly time-sheet approvals of the user as
// employee and those the user decided as approver.
func weekApprovalsModule(ctx context.Context, pool *pgxpool.Pool, tenantID, userID uuid.UUID) (*DSARModule, error) {
	columns := []string{"Woche ab", "Rolle", "Status", "Eingereicht", "Entschieden", "Ablehnungsgrund"}
	return userRowsModule(ctx, pool, "week approvals", "Wochenfreigaben", columns,
		`SELECT week_start,
		        CASE WHEN employee_id = $2 THEN 'Mitarbeiter' ELSE 'Freigebender' END,
		        status, submitted_at, approved_at, COALESCE(rejection_reason, '')
		 FROM hr_week_approvals
		 WHERE tenant_id = $1 AND (employee_id = $2 OR approved_by = $2)
		 ORDER BY week_start DESC
		 LIMIT $3`, []any{tenantID, userID, dsarMaxRows},
		func(rows pgx.Rows) (DSARRecord, error) {
			var weekStart time.Time
			var role, status, reason string
			var submittedAt, decidedAt *time.Time
			if err := rows.Scan(&weekStart, &role, &status, &submittedAt, &decidedAt, &reason); err != nil {
				return DSARRecord{}, err
			}
			return DSARRecord{Fields: []DSARField{
				{Key: "Woche ab", Value: weekStart.Format("02.01.2006")},
				{Key: "Rolle", Value: role},
				{Key: "Status", Value: status},
				{Key: "Eingereicht", Value: derefDateTimeOrEmpty(submittedAt)},
				{Key: "Entschieden", Value: derefDateTimeOrEmpty(decidedAt)},
				{Key: "Ablehnungsgrund", Value: reason},
			}}, nil
		})
}

// reportsShiftsModules collects the four modules for one user; empty ones are
// skipped.
func reportsShiftsModules(ctx context.Context, pool *pgxpool.Pool, tenantID, userID uuid.UUID) ([]DSARModule, error) {
	var out []DSARModule
	for _, load := range []func(context.Context, *pgxpool.Pool, uuid.UUID, uuid.UUID) (*DSARModule, error){
		workReportsModule, shiftAssignmentsModule, shiftSwapRequestsModule, weekApprovalsModule,
	} {
		mod, err := load(ctx, pool, tenantID, userID)
		if err != nil {
			return nil, fmt.Errorf("dsar: reports and shifts: %w", err)
		}
		if mod != nil {
			out = append(out, *mod)
		}
	}
	return out, nil
}
