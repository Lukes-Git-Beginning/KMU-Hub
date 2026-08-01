package timeentry

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/models"
)

// Service handles time entry business logic
type Service struct {
	repo Repository
}

// NewService creates a new time entry service
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// StartTimer starts a new timer for a user on a task.
// If the user already has a running timer on another task, it stops that timer first.
func (s *Service) StartTimer(ctx context.Context, taskID, userID, tenantID uuid.UUID) (*models.TimeEntry, *models.TimeEntry, error) {
	// Check for and stop any existing active timer
	var stoppedEntry *models.TimeEntry
	existing, err := s.repo.GetActiveTimer(ctx, userID, tenantID)
	if err != nil {
		return nil, nil, err
	}
	if existing != nil {
		stopped, stopErr := s.repo.StopActiveTimer(ctx, userID, tenantID)
		if stopErr != nil {
			return nil, nil, stopErr
		}
		stoppedEntry = stopped
		slog.Info("auto-stopped previous timer",
			"previous_task_id", existing.TaskID,
			"user_id", userID,
		)
	}

	now := time.Now()
	entry := &models.TimeEntry{
		ID:        uuid.New(),
		TenantID:  tenantID,
		TaskID:    taskID,
		UserID:    userID,
		StartedAt: now,
		IsManual:  false,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if createErr := s.repo.Create(ctx, entry); createErr != nil {
		return nil, nil, createErr
	}

	slog.Info("timer started",
		"time_entry_id", entry.ID,
		"task_id", taskID,
		"user_id", userID,
	)

	return entry, stoppedEntry, nil
}

// StopTimer stops the active timer for a user
func (s *Service) StopTimer(ctx context.Context, userID, tenantID uuid.UUID) (*models.TimeEntry, error) {
	stopped, err := s.repo.StopActiveTimer(ctx, userID, tenantID)
	if err != nil {
		return nil, err
	}
	if stopped == nil {
		return nil, ErrNoActiveTimer
	}

	slog.Info("timer stopped",
		"time_entry_id", stopped.ID,
		"task_id", stopped.TaskID,
		"user_id", userID,
		"duration_seconds", stopped.DurationSeconds,
	)

	return stopped, nil
}

// GetActiveTimer returns the currently running timer for a user, or nil
func (s *Service) GetActiveTimer(ctx context.Context, userID, tenantID uuid.UUID) (*models.ActiveTimer, error) {
	return s.repo.GetActiveTimer(ctx, userID, tenantID)
}

// ManualEntryInput contains the data for creating a manual time entry
type ManualEntryInput struct {
	TenantID        uuid.UUID
	TaskID          uuid.UUID
	UserID          uuid.UUID
	StartedAt       time.Time
	DurationSeconds int
	Description     *string
}

// AddManualEntry creates a manual time entry
func (s *Service) AddManualEntry(ctx context.Context, input ManualEntryInput) (*models.TimeEntry, error) {
	if input.DurationSeconds <= 0 {
		return nil, ErrInvalidDuration
	}

	if input.Description != nil {
		desc := strings.TrimSpace(*input.Description)
		if len(desc) > 2000 {
			return nil, ErrDescriptionTooLong
		}
		input.Description = &desc
	}

	now := time.Now()
	endedAt := input.StartedAt.Add(time.Duration(input.DurationSeconds) * time.Second)
	durationSec := input.DurationSeconds

	entry := &models.TimeEntry{
		ID:              uuid.New(),
		TenantID:        input.TenantID,
		TaskID:          input.TaskID,
		UserID:          input.UserID,
		StartedAt:       input.StartedAt,
		EndedAt:         &endedAt,
		DurationSeconds: &durationSec,
		Description:     input.Description,
		IsManual:        true,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := s.repo.Create(ctx, entry); err != nil {
		return nil, err
	}

	slog.Info("manual time entry added",
		"time_entry_id", entry.ID,
		"task_id", input.TaskID,
		"user_id", input.UserID,
		"duration_seconds", input.DurationSeconds,
	)

	return entry, nil
}

// UpdateEntryInput contains the data for updating a time entry
type UpdateEntryInput struct {
	StartedAt       *time.Time
	DurationSeconds *int
	Description     *string
}

// UpdateEntry updates an existing time entry
func (s *Service) UpdateEntry(ctx context.Context, entryID, actorID, tenantID uuid.UUID, input UpdateEntryInput) (*models.TimeEntryWithUser, error) {
	existing, err := s.repo.GetByID(ctx, entryID, tenantID)
	if err != nil {
		return nil, err
	}

	// Only the user who created the entry can edit it
	if existing.UserID != actorID {
		return nil, ErrCannotEditOthers
	}

	entry := &existing.TimeEntry

	if input.StartedAt != nil {
		entry.StartedAt = *input.StartedAt
	}

	if input.DurationSeconds != nil {
		if *input.DurationSeconds <= 0 {
			return nil, ErrInvalidDuration
		}
		dur := *input.DurationSeconds
		entry.DurationSeconds = &dur
		endedAt := entry.StartedAt.Add(time.Duration(dur) * time.Second)
		entry.EndedAt = &endedAt
	}

	if input.Description != nil {
		desc := strings.TrimSpace(*input.Description)
		if len(desc) > 2000 {
			return nil, ErrDescriptionTooLong
		}
		entry.Description = &desc
	}

	entry.UpdatedAt = time.Now()

	if updateErr := s.repo.Update(ctx, entry); updateErr != nil {
		return nil, updateErr
	}

	return s.repo.GetByID(ctx, entryID, tenantID)
}

// DeleteEntry deletes a time entry
func (s *Service) DeleteEntry(ctx context.Context, entryID, actorID, tenantID uuid.UUID, isAdmin bool) error {
	existing, err := s.repo.GetByID(ctx, entryID, tenantID)
	if err != nil {
		return err
	}

	// Only the owner or admin can delete
	if existing.UserID != actorID && !isAdmin {
		return ErrCannotDeleteOthers
	}

	return s.repo.Delete(ctx, entryID, tenantID)
}

// ListByTask returns time entries for a task
func (s *Service) ListByTask(ctx context.Context, taskID, tenantID uuid.UUID, page, pageSize int) ([]models.TimeEntryWithUser, int, error) {
	return s.repo.ListByTask(ctx, taskID, tenantID, page, pageSize)
}

// GetTaskTimeSummary returns the aggregate time summary for a task
func (s *Service) GetTaskTimeSummary(ctx context.Context, taskID, tenantID uuid.UUID) (*models.TimeEntrySummary, error) {
	return s.repo.GetTaskTimeSummary(ctx, taskID, tenantID)
}

// ListBillable returns completed time entries across the tenant for invoicing
// (Stunden -> Rechnung).
func (s *Service) ListBillable(ctx context.Context, tenantID uuid.UUID) ([]models.BillableTimeEntry, error) {
	return s.repo.ListBillable(ctx, tenantID)
}

// ListByProject returns completed time entries for a single project's tasks
// (project-level "Stunden abrechnen" roll-up).
func (s *Service) ListByProject(ctx context.Context, projectID, tenantID uuid.UUID) ([]models.ProjectTimeEntry, error) {
	return s.repo.ListByProject(ctx, projectID, tenantID)
}

// defaultWeeklyTargetHours is a fixed full-time-week stand-in used for every
// member's utilization target.
//
// lean: the real per-member value (EmployeeProfile.WorkDaysPerWeek x
// HRCompanySettings.WorkHoursPerDay) lives behind team:data_job:view, a
// permission this endpoint does not check -- it is gated on project-read
// only. Wiring it in would leak HR-gated data to anyone with project access.
// Upgrade once a capability-checked path exists for the work service to read
// that field, or once utilization moves behind its own HR-aware guard.
const defaultWeeklyTargetHours = 40

const (
	utilizationWeeklyBuckets  = 6
	utilizationMonthlyBuckets = 3
)

var germanMonthAbbr = [...]string{
	"Jan", "Feb", "Mär", "Apr", "Mai", "Jun",
	"Jul", "Aug", "Sep", "Okt", "Nov", "Dez",
}

// ProjectUtilization builds the team-utilization roll-up ("Auslastung") for
// a project: every member's tracked hours per week (last 6 ISO weeks) and
// per month (last 3 months), including members with zero tracked hours so
// managers can see who isn't logging time, not just who is. Carries no cost
// or salary figure -- see the note on MemberUtilization.
func (s *Service) ProjectUtilization(ctx context.Context, projectID, tenantID uuid.UUID, members []models.ProjectMember, now time.Time) ([]models.MemberUtilization, error) {
	weeklySince := startOfISOWeek(now).AddDate(0, 0, -7*(utilizationWeeklyBuckets-1))
	weekly, err := s.repo.AggregateProjectHours(ctx, projectID, tenantID, "week", weeklySince)
	if err != nil {
		return nil, err
	}

	monthlySince := startOfMonth(now).AddDate(0, -(utilizationMonthlyBuckets - 1), 0)
	monthly, err := s.repo.AggregateProjectHours(ctx, projectID, tenantID, "month", monthlySince)
	if err != nil {
		return nil, err
	}

	return buildMemberUtilization(members, weekly, monthly, now), nil
}

// startOfISOWeek returns midnight UTC of the Monday starting t's ISO week.
func startOfISOWeek(t time.Time) time.Time {
	t = t.UTC()
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7 // Sunday -> 7, so Monday is always day 1
	}
	monday := t.AddDate(0, 0, -(weekday - 1))
	return time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, time.UTC)
}

// startOfMonth returns midnight UTC of the first day of t's month.
func startOfMonth(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
}

func secondsToHours(seconds int) float64 {
	return math.Round(float64(seconds)/3600*100) / 100
}

// buildMemberUtilization zero-fills the last utilizationWeeklyBuckets weeks
// and utilizationMonthlyBuckets months for every given member and labels
// each bucket, so a member with no entries in a period still appears with
// 0 hours rather than a missing point. Pure function, no DB access -- keeps
// the bucket/label assembly unit-testable without a database.
func buildMemberUtilization(members []models.ProjectMember, weekly, monthly []models.UtilizationBucket, now time.Time) []models.MemberUtilization {
	weeklyByMember := bucketsByMemberAndPeriod(weekly)
	monthlyByMember := bucketsByMemberAndPeriod(monthly)

	weekStarts := make([]time.Time, utilizationWeeklyBuckets)
	base := startOfISOWeek(now)
	for i := range weekStarts {
		weekStarts[i] = base.AddDate(0, 0, -7*(utilizationWeeklyBuckets-1-i))
	}

	monthStarts := make([]time.Time, utilizationMonthlyBuckets)
	baseMonth := startOfMonth(now)
	for i := range monthStarts {
		monthStarts[i] = baseMonth.AddDate(0, -(utilizationMonthlyBuckets - 1 - i), 0)
	}

	result := make([]models.MemberUtilization, 0, len(members))
	for _, m := range members {
		result = append(result, models.MemberUtilization{
			UserID:       m.UserID,
			Name:         strings.TrimSpace(m.FirstName + " " + m.LastName),
			Role:         m.Role,
			WeeklyTarget: defaultWeeklyTargetHours,
			WeeklyData:   utilizationPoints(weeklyByMember[m.UserID], weekStarts, weekLabel),
			MonthlyData:  utilizationPoints(monthlyByMember[m.UserID], monthStarts, monthLabel),
		})
	}
	return result
}

func bucketsByMemberAndPeriod(buckets []models.UtilizationBucket) map[uuid.UUID]map[string]int {
	out := make(map[uuid.UUID]map[string]int, len(buckets))
	for _, b := range buckets {
		byPeriod, ok := out[b.UserID]
		if !ok {
			byPeriod = make(map[string]int)
			out[b.UserID] = byPeriod
		}
		byPeriod[periodKey(b.PeriodStart)] += b.TotalSeconds
	}
	return out
}

func periodKey(t time.Time) string {
	return t.Format("2006-01-02")
}

func weekLabel(t time.Time) string {
	_, week := t.ISOWeek()
	return fmt.Sprintf("KW %d", week)
}

func monthLabel(t time.Time) string {
	return fmt.Sprintf("%s %d", germanMonthAbbr[t.Month()-1], t.Year())
}

func utilizationPoints(byPeriod map[string]int, periodStarts []time.Time, label func(time.Time) string) []models.UtilizationPoint {
	points := make([]models.UtilizationPoint, len(periodStarts))
	for i, start := range periodStarts {
		points[i] = models.UtilizationPoint{
			Label: label(start),
			Hours: secondsToHours(byPeriod[periodKey(start)]),
		}
	}
	return points
}
