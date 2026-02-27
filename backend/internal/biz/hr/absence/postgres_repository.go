package absence

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresAbsenceRepo implements AbsenceRepository using PostgreSQL.
type PostgresAbsenceRepo struct {
	pool *pgxpool.Pool
}

// NewPostgresAbsenceRepo creates a new PostgreSQL absence repository.
func NewPostgresAbsenceRepo(pool *pgxpool.Pool) *PostgresAbsenceRepo {
	return &PostgresAbsenceRepo{pool: pool}
}

func (r *PostgresAbsenceRepo) GetAbsenceCalendar(ctx context.Context, filter AbsenceFilter) ([]*AbsenceEntry, error) {
	var conditions []string
	var args []any
	argNum := 1

	// Tenant filter
	conditions = append(conditions, fmt.Sprintf("lr.tenant_id = $%d", argNum))
	args = append(args, filter.TenantID)
	argNum++

	// Only approved absences
	conditions = append(conditions, "lr.status = 'approved'")

	// Date range overlap: absences that overlap with the requested period
	conditions = append(conditions, fmt.Sprintf(
		"daterange(lr.start_date, lr.end_date, '[]') && daterange($%d::date, $%d::date, '[]')",
		argNum, argNum+1,
	))
	args = append(args, filter.StartDate, filter.EndDate)
	argNum += 2

	// Optional department filter
	if filter.Department != "" {
		conditions = append(conditions, fmt.Sprintf("ep.department = $%d", argNum))
		args = append(args, filter.Department)
	}

	whereClause := "WHERE " + strings.Join(conditions, " AND ")

	query := fmt.Sprintf(`
		SELECT lr.employee_id,
			COALESCE(u.display_name, u.email, '') AS employee_name,
			COALESCE(ep.department, '') AS department,
			COALESCE(lt.name, '') AS leave_type_name,
			COALESCE(lt.key, '') AS leave_type_key,
			COALESCE(lt.color, '#9ca3af') AS color,
			lr.start_date, lr.end_date,
			lr.is_half_day_start, lr.half_day_period_start,
			lr.is_half_day_end, lr.half_day_period_end,
			lr.status
		FROM hr_leave_requests lr
		LEFT JOIN users u ON lr.employee_id = u.id
		LEFT JOIN hr_employee_profiles ep ON lr.employee_id = ep.user_id
		LEFT JOIN hr_leave_types lt ON lr.leave_type_id = lt.id
		%s
		ORDER BY employee_name, lr.start_date
	`, whereClause)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*AbsenceEntry
	for rows.Next() {
		entry := &AbsenceEntry{}
		if scanErr := rows.Scan(
			&entry.EmployeeID,
			&entry.EmployeeName,
			&entry.Department,
			&entry.LeaveTypeName,
			&entry.LeaveTypeKey,
			&entry.Color,
			&entry.StartDate,
			&entry.EndDate,
			&entry.IsHalfDayStart,
			&entry.HalfDayPeriodStart,
			&entry.IsHalfDayEnd,
			&entry.HalfDayPeriodEnd,
			&entry.Status,
		); scanErr != nil {
			return nil, scanErr
		}
		results = append(results, entry)
	}

	return results, rows.Err()
}
