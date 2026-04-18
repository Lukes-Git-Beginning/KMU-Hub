// Package scheduler drives berichte report schedules. It polls active
// schedules once per tick, decides which ones are due based on their cron
// expression and last_run_at, atomically claims a due schedule to prevent
// double-execution on tick overlap, then dispatches the report through the
// service, exporter, and mailer.
//
// The scheduler depends on narrow interfaces (ScheduleRepository, Runner,
// Exporter, Mailer) so cmd/berichte can wire the concrete implementations
// while unit tests substitute minimal fakes. Exporter or Mailer may be nil;
// the scheduler records last_run_status=skipped with a descriptive error
// instead of crashing. This lets scheduler startup succeed while WP-3
// (exporters) is still under development and before SMTP is configured in
// pilot deployments.
package scheduler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"

	"github.com/kmuhub/kmuhub/internal/berichte"
)

// ============================================================================
// Collaborators
// ============================================================================

// ScheduleRepository is the minimal persistence surface the scheduler needs.
type ScheduleRepository interface {
	ListDueSchedules(ctx context.Context, now time.Time) ([]*berichte.Schedule, error)
	ClaimSchedule(ctx context.Context, scheduleID uuid.UUID, previousLastRunAt *time.Time, now time.Time) (bool, error)
	UpdateScheduleLastRun(ctx context.Context, scheduleID uuid.UUID, status string, runErr *string, at time.Time) error
}

// Runner executes a report on behalf of the scheduler. Narrower than
// berichte.Service so tests need not build the full service graph.
type Runner interface {
	RunReport(ctx context.Context, in berichte.RunReportInput) (*berichte.ReportResult, *berichte.Run, error)
}

// ExportedReport carries the rendered bytes alongside MIME metadata.
type ExportedReport struct {
	Payload     []byte
	ContentType string
	Filename    string
}

// Exporter produces a rendered report in a given format.
type Exporter interface {
	Export(ctx context.Context, result *berichte.ReportResult, def *berichte.Definition, format string) (*ExportedReport, error)
}

// MailInput is the payload sent to a Mailer.
type MailInput struct {
	TenantID   uuid.UUID
	ScheduleID uuid.UUID
	Recipients []string
	Subject    string
	BodyText   string
	BodyHTML   string
	Attachment ExportedReport
}

// Mailer sends a report to the configured recipients.
type Mailer interface {
	SendReport(ctx context.Context, in MailInput) error
}

// Clock abstracts time.Now for deterministic tests.
type Clock interface {
	Now() time.Time
}

type realClock struct{}

func (realClock) Now() time.Time { return time.Now() }

// ============================================================================
// Scheduler
// ============================================================================

// Config bundles optional configuration. Zero values default sensibly.
type Config struct {
	// Tick controls how often the scheduler checks for due schedules. Default
	// 60s, matches cron's minute granularity.
	Tick time.Duration
	// Clock is mockable. Defaults to wall clock.
	Clock Clock
	// Logger is optional; defaults to slog.Default().
	Logger *slog.Logger
}

// Scheduler polls schedules and dispatches due reports.
type Scheduler struct {
	repo    ScheduleRepository
	runner  Runner
	export  Exporter
	mailer  Mailer
	clock   Clock
	log     *slog.Logger
	tick    time.Duration
	parser  cron.Parser
}

// New creates a scheduler. export and mailer may be nil; in that case
// every due schedule is recorded as last_run_status=skipped with a
// descriptive error and never delivered.
func New(repo ScheduleRepository, runner Runner, export Exporter, mailer Mailer, cfg Config) *Scheduler {
	clk := cfg.Clock
	if clk == nil {
		clk = realClock{}
	}
	log := cfg.Logger
	if log == nil {
		log = slog.Default()
	}
	tick := cfg.Tick
	if tick <= 0 {
		tick = 60 * time.Second
	}
	return &Scheduler{
		repo:   repo,
		runner: runner,
		export: export,
		mailer: mailer,
		clock:  clk,
		log:    log,
		tick:   tick,
		parser: cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow),
	}
}

// Run blocks until ctx is cancelled. Processes one tick immediately so
// startup-time overdue schedules do not wait.
func (s *Scheduler) Run(ctx context.Context) error {
	s.log.Info("berichte scheduler started", "tick", s.tick.String())
	defer s.log.Info("berichte scheduler stopped")

	s.processTick(ctx, s.clock.Now())

	ticker := time.NewTicker(s.tick)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			s.processTick(ctx, s.clock.Now())
		}
	}
}

// ProcessTick runs a single scheduler iteration. Exposed for tests.
func (s *Scheduler) ProcessTick(ctx context.Context, now time.Time) {
	s.processTick(ctx, now)
}

func (s *Scheduler) processTick(ctx context.Context, now time.Time) {
	schedules, err := s.repo.ListDueSchedules(ctx, now)
	if err != nil {
		s.log.Error("berichte scheduler: list due schedules failed", "error", err)
		return
	}
	for _, sch := range schedules {
		if !s.isDue(sch, now) {
			continue
		}
		previousLastRunAt := sch.LastRunAt
		claimed, claimErr := s.repo.ClaimSchedule(ctx, sch.ID, previousLastRunAt, now)
		if claimErr != nil {
			s.log.Error("berichte scheduler: claim failed", "schedule_id", sch.ID, "error", claimErr)
			continue
		}
		if !claimed {
			s.log.Debug("berichte scheduler: claim lost to another tick", "schedule_id", sch.ID)
			continue
		}
		s.dispatch(ctx, sch, now)
	}
}

func (s *Scheduler) isDue(sch *berichte.Schedule, now time.Time) bool {
	parsed, err := s.parser.Parse(sch.CronExpression)
	if err != nil {
		s.log.Warn("berichte scheduler: invalid cron on stored schedule",
			"schedule_id", sch.ID, "cron", sch.CronExpression, "error", err)
		return false
	}
	reference := sch.CreatedAt
	if sch.LastRunAt != nil {
		reference = *sch.LastRunAt
	}
	next := parsed.Next(reference)
	return !next.After(now)
}

func (s *Scheduler) dispatch(ctx context.Context, sch *berichte.Schedule, now time.Time) {
	if s.export == nil {
		s.markSkip(ctx, sch, "exporter not configured", now)
		return
	}
	if s.mailer == nil {
		s.markSkip(ctx, sch, "mailer not configured", now)
		return
	}

	params := sch.Params
	if len(params) == 0 {
		params = []byte("{}")
	}

	result, _, err := s.runner.RunReport(ctx, berichte.RunReportInput{
		TenantID:     sch.TenantID,
		DefinitionID: sch.DefinitionID,
		Params:       json.RawMessage(params),
		Trigger:      "scheduled",
	})
	if err != nil {
		s.markFailed(ctx, sch, fmt.Sprintf("run report: %v", err), now)
		return
	}

	exported, err := s.export.Export(ctx, result, &berichte.Definition{
		ID:       sch.DefinitionID,
		TenantID: sch.TenantID,
		Name:     sch.Name,
	}, sch.Format)
	if err != nil {
		s.markFailed(ctx, sch, fmt.Sprintf("export: %v", err), now)
		return
	}

	if len(sch.Recipients) == 0 {
		s.markSkip(ctx, sch, "no recipients configured", now)
		return
	}

	subject := fmt.Sprintf("[Cosmi] %s", sch.Name)
	bodyText := buildTextBody(sch, result)
	bodyHTML := buildHTMLBody(sch, result)

	mailErr := s.mailer.SendReport(ctx, MailInput{
		TenantID:   sch.TenantID,
		ScheduleID: sch.ID,
		Recipients: sch.Recipients,
		Subject:    subject,
		BodyText:   bodyText,
		BodyHTML:   bodyHTML,
		Attachment: *exported,
	})
	if mailErr != nil {
		s.markFailed(ctx, sch, fmt.Sprintf("mail: %v", mailErr), now)
		return
	}

	if err := s.repo.UpdateScheduleLastRun(ctx, sch.ID, "success", nil, now); err != nil {
		s.log.Warn("berichte scheduler: update last run failed",
			"schedule_id", sch.ID, "error", err)
	}
	s.log.Info("berichte scheduler: dispatched",
		"schedule_id", sch.ID,
		"recipients", len(sch.Recipients),
		"format", sch.Format,
	)
}

func (s *Scheduler) markFailed(ctx context.Context, sch *berichte.Schedule, reason string, now time.Time) {
	s.log.Warn("berichte scheduler: run failed",
		"schedule_id", sch.ID, "reason", reason)
	if err := s.repo.UpdateScheduleLastRun(ctx, sch.ID, "failed", &reason, now); err != nil {
		s.log.Warn("berichte scheduler: update last run failed",
			"schedule_id", sch.ID, "error", err)
	}
}

func (s *Scheduler) markSkip(ctx context.Context, sch *berichte.Schedule, reason string, now time.Time) {
	s.log.Info("berichte scheduler: skipped",
		"schedule_id", sch.ID, "reason", reason)
	if err := s.repo.UpdateScheduleLastRun(ctx, sch.ID, "skipped", &reason, now); err != nil {
		s.log.Warn("berichte scheduler: update last run failed",
			"schedule_id", sch.ID, "error", err)
	}
}

// ============================================================================
// Body builders
// ============================================================================

func buildTextBody(sch *berichte.Schedule, result *berichte.ReportResult) string {
	var b bytes.Buffer
	fmt.Fprintf(&b, "Hallo,\n\nhier ist der Bericht %q (geplant).\n\n", sch.Name)
	fmt.Fprintf(&b, "Erzeugt: %s\n", result.Meta.GeneratedAt.Format(time.RFC3339))
	fmt.Fprintf(&b, "Zeilen: %d\n", result.Meta.RowCount)
	if result.Meta.Warning != "" {
		fmt.Fprintf(&b, "Hinweis: %s\n", result.Meta.Warning)
	}
	b.WriteString("\nDer Bericht liegt dieser E-Mail als Anhang bei.\n")
	return b.String()
}

func buildHTMLBody(sch *berichte.Schedule, result *berichte.ReportResult) string {
	warning := ""
	if result.Meta.Warning != "" {
		warning = fmt.Sprintf("<p><em>Hinweis: %s</em></p>", result.Meta.Warning)
	}
	return fmt.Sprintf(`<p>Hallo,</p>
<p>hier ist der Bericht <strong>%s</strong> (geplant).</p>
<ul><li>Erzeugt: %s</li><li>Zeilen: %d</li></ul>
%s
<p>Der Bericht liegt dieser E-Mail als Anhang bei.</p>`,
		sch.Name, result.Meta.GeneratedAt.Format(time.RFC3339), result.Meta.RowCount, warning)
}
