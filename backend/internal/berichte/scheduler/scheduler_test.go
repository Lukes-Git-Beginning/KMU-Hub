package scheduler

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/berichte"
)

var testTenant = uuid.MustParse("33333333-3333-3333-3333-333333333333")

// ============================================================================
// Mocks
// ============================================================================

type fixedClock struct{ t time.Time }

func (f *fixedClock) Now() time.Time { return f.t }

type mockRepo struct {
	mu            sync.Mutex
	schedules     []*berichte.Schedule
	claimedIDs    []uuid.UUID
	lastRunStates map[uuid.UUID]lastRunCall
	listErr       error
	claimErr      error
	// forceClaimResult allows a test to simulate a lost claim race without
	// inspecting previousLastRunAt; "" leaves the default behaviour.
	forceClaimResult string // "" | "deny"
}

type lastRunCall struct {
	status string
	err    *string
	at     time.Time
}

func newMockRepo(initial ...*berichte.Schedule) *mockRepo {
	return &mockRepo{
		schedules:     append([]*berichte.Schedule(nil), initial...),
		lastRunStates: map[uuid.UUID]lastRunCall{},
	}
}

func (m *mockRepo) ListDueSchedules(_ context.Context, _ time.Time) ([]*berichte.Schedule, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.listErr != nil {
		return nil, m.listErr
	}
	out := make([]*berichte.Schedule, 0, len(m.schedules))
	for _, s := range m.schedules {
		if !s.Active {
			continue
		}
		copy := *s
		out = append(out, &copy)
	}
	return out, nil
}

func (m *mockRepo) ClaimSchedule(_ context.Context, scheduleID uuid.UUID, previous *time.Time, now time.Time) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.claimErr != nil {
		return false, m.claimErr
	}
	if m.forceClaimResult == "deny" {
		return false, nil
	}
	for _, s := range m.schedules {
		if s.ID != scheduleID {
			continue
		}
		match := false
		if previous == nil {
			match = s.LastRunAt == nil
		} else if s.LastRunAt != nil && s.LastRunAt.Equal(*previous) {
			match = true
		}
		if !match {
			return false, nil
		}
		s.LastRunAt = &now
		m.claimedIDs = append(m.claimedIDs, s.ID)
		return true, nil
	}
	return false, nil
}

func (m *mockRepo) UpdateScheduleLastRun(_ context.Context, scheduleID uuid.UUID, status string, runErr *string, at time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastRunStates[scheduleID] = lastRunCall{status: status, err: runErr, at: at}
	for _, s := range m.schedules {
		if s.ID == scheduleID {
			s.LastRunStatus = &status
			s.LastRunError = runErr
			s.LastRunAt = &at
		}
	}
	return nil
}

type mockRunner struct {
	result *berichte.ReportResult
	run    *berichte.Run
	err    error
	calls  int
}

func (m *mockRunner) RunReport(_ context.Context, in berichte.RunReportInput) (*berichte.ReportResult, *berichte.Run, error) {
	m.calls++
	if m.err != nil {
		return nil, nil, m.err
	}
	if m.result == nil {
		m.result = &berichte.ReportResult{Meta: berichte.ReportMeta{RowCount: 1, DefinitionID: in.DefinitionID}}
	}
	return m.result, m.run, nil
}

type mockExporter struct {
	exported *ExportedReport
	err      error
	calls    int
}

func (m *mockExporter) Export(_ context.Context, _ *berichte.ReportResult, _ *berichte.Definition, format string) (*ExportedReport, error) {
	m.calls++
	if m.err != nil {
		return nil, m.err
	}
	if m.exported == nil {
		m.exported = &ExportedReport{
			Payload:     []byte("payload"),
			ContentType: "application/octet-stream",
			Filename:    "report." + format,
		}
	}
	return m.exported, nil
}

type mockMailer struct {
	lastInput *MailInput
	err       error
	calls     int
}

func (m *mockMailer) SendReport(_ context.Context, in MailInput) error {
	m.calls++
	copy := in
	m.lastInput = &copy
	return m.err
}

// ============================================================================
// Helpers
// ============================================================================

func buildSchedule(t *testing.T, opts func(s *berichte.Schedule)) *berichte.Schedule {
	t.Helper()
	sch := &berichte.Schedule{
		ID:             uuid.New(),
		TenantID:       testTenant,
		DefinitionID:   uuid.New(),
		Name:           "Daily Revenue",
		CronExpression: "* * * * *",
		Recipients:     []string{"team@example.com"},
		Format:         "pdf",
		Params:         json.RawMessage(`{}`),
		Active:         true,
		CreatedAt:      time.Date(2026, 4, 18, 10, 0, 0, 0, time.UTC),
		UpdatedAt:      time.Date(2026, 4, 18, 10, 0, 0, 0, time.UTC),
	}
	if opts != nil {
		opts(sch)
	}
	return sch
}

// ============================================================================
// Tests
// ============================================================================

func TestScheduler_DueDispatchesHappyPath(t *testing.T) {
	sch := buildSchedule(t, nil)
	repo := newMockRepo(sch)
	runner := &mockRunner{}
	exporter := &mockExporter{}
	mailer := &mockMailer{}
	now := sch.CreatedAt.Add(5 * time.Minute)
	s := New(repo, runner, exporter, mailer, Config{Clock: &fixedClock{t: now}})

	s.ProcessTick(context.Background(), now)

	if runner.calls != 1 {
		t.Errorf("expected runner called once, got %d", runner.calls)
	}
	if exporter.calls != 1 {
		t.Errorf("expected exporter called once, got %d", exporter.calls)
	}
	if mailer.calls != 1 {
		t.Errorf("expected mailer called once, got %d", mailer.calls)
	}
	if got := repo.lastRunStates[sch.ID].status; got != "success" {
		t.Errorf("expected status success, got %q", got)
	}
	if mailer.lastInput == nil || len(mailer.lastInput.Recipients) != 1 {
		t.Error("expected mailer input with one recipient")
	}
}

func TestScheduler_NotDue_NoAction(t *testing.T) {
	lastRun := time.Date(2026, 4, 18, 11, 0, 0, 0, time.UTC)
	sch := buildSchedule(t, func(s *berichte.Schedule) {
		s.CronExpression = "0 0 * * *" // once a day
		s.LastRunAt = &lastRun
	})
	repo := newMockRepo(sch)
	runner := &mockRunner{}
	exporter := &mockExporter{}
	mailer := &mockMailer{}
	now := lastRun.Add(5 * time.Minute) // still same day, not due yet
	s := New(repo, runner, exporter, mailer, Config{Clock: &fixedClock{t: now}})

	s.ProcessTick(context.Background(), now)

	if runner.calls != 0 {
		t.Errorf("expected no runner calls, got %d", runner.calls)
	}
}

func TestScheduler_InactiveSchedulesSkipped(t *testing.T) {
	sch := buildSchedule(t, func(s *berichte.Schedule) { s.Active = false })
	repo := newMockRepo(sch)
	runner := &mockRunner{}
	exporter := &mockExporter{}
	mailer := &mockMailer{}
	now := sch.CreatedAt.Add(5 * time.Minute)
	s := New(repo, runner, exporter, mailer, Config{Clock: &fixedClock{t: now}})

	s.ProcessTick(context.Background(), now)

	if runner.calls != 0 {
		t.Errorf("expected no runner calls for inactive schedule, got %d", runner.calls)
	}
}

func TestScheduler_ClaimLostToConcurrentTick(t *testing.T) {
	sch := buildSchedule(t, nil)
	repo := newMockRepo(sch)
	repo.forceClaimResult = "deny"
	runner := &mockRunner{}
	exporter := &mockExporter{}
	mailer := &mockMailer{}
	now := sch.CreatedAt.Add(5 * time.Minute)
	s := New(repo, runner, exporter, mailer, Config{Clock: &fixedClock{t: now}})

	s.ProcessTick(context.Background(), now)

	if runner.calls != 0 {
		t.Errorf("expected no runner call when claim denied, got %d", runner.calls)
	}
	if _, ok := repo.lastRunStates[sch.ID]; ok {
		t.Error("expected no UpdateScheduleLastRun when claim lost")
	}
}

func TestScheduler_RunReportError_MarksFailed(t *testing.T) {
	sch := buildSchedule(t, nil)
	repo := newMockRepo(sch)
	runner := &mockRunner{err: errors.New("boom")}
	exporter := &mockExporter{}
	mailer := &mockMailer{}
	now := sch.CreatedAt.Add(5 * time.Minute)
	s := New(repo, runner, exporter, mailer, Config{Clock: &fixedClock{t: now}})

	s.ProcessTick(context.Background(), now)

	state := repo.lastRunStates[sch.ID]
	if state.status != "failed" {
		t.Errorf("expected failed status, got %q", state.status)
	}
	if state.err == nil || *state.err == "" {
		t.Error("expected error string stored")
	}
	if exporter.calls != 0 || mailer.calls != 0 {
		t.Error("expected no export/mail when runner fails")
	}
}

func TestScheduler_ExportError_MarksFailed(t *testing.T) {
	sch := buildSchedule(t, nil)
	repo := newMockRepo(sch)
	runner := &mockRunner{}
	exporter := &mockExporter{err: errors.New("render failed")}
	mailer := &mockMailer{}
	now := sch.CreatedAt.Add(5 * time.Minute)
	s := New(repo, runner, exporter, mailer, Config{Clock: &fixedClock{t: now}})

	s.ProcessTick(context.Background(), now)

	if state := repo.lastRunStates[sch.ID]; state.status != "failed" {
		t.Errorf("expected failed on export error, got %q", state.status)
	}
	if mailer.calls != 0 {
		t.Error("expected no mail when export fails")
	}
}

func TestScheduler_MailerError_MarksFailed(t *testing.T) {
	sch := buildSchedule(t, nil)
	repo := newMockRepo(sch)
	runner := &mockRunner{}
	exporter := &mockExporter{}
	mailer := &mockMailer{err: errors.New("smtp refused")}
	now := sch.CreatedAt.Add(5 * time.Minute)
	s := New(repo, runner, exporter, mailer, Config{Clock: &fixedClock{t: now}})

	s.ProcessTick(context.Background(), now)

	if state := repo.lastRunStates[sch.ID]; state.status != "failed" {
		t.Errorf("expected failed on mail error, got %q", state.status)
	}
}

func TestScheduler_ExporterNil_MarksSkipped(t *testing.T) {
	sch := buildSchedule(t, nil)
	repo := newMockRepo(sch)
	runner := &mockRunner{}
	mailer := &mockMailer{}
	now := sch.CreatedAt.Add(5 * time.Minute)
	s := New(repo, runner, nil, mailer, Config{Clock: &fixedClock{t: now}})

	s.ProcessTick(context.Background(), now)

	if state := repo.lastRunStates[sch.ID]; state.status != "skipped" {
		t.Errorf("expected skipped without exporter, got %q", state.status)
	}
	if runner.calls != 0 || mailer.calls != 0 {
		t.Error("expected no run/mail when exporter nil")
	}
}

func TestScheduler_MailerNil_MarksSkipped(t *testing.T) {
	sch := buildSchedule(t, nil)
	repo := newMockRepo(sch)
	runner := &mockRunner{}
	exporter := &mockExporter{}
	now := sch.CreatedAt.Add(5 * time.Minute)
	s := New(repo, runner, exporter, nil, Config{Clock: &fixedClock{t: now}})

	s.ProcessTick(context.Background(), now)

	if state := repo.lastRunStates[sch.ID]; state.status != "skipped" {
		t.Errorf("expected skipped without mailer, got %q", state.status)
	}
}

func TestScheduler_NoRecipients_MarksSkipped(t *testing.T) {
	sch := buildSchedule(t, func(s *berichte.Schedule) { s.Recipients = nil })
	repo := newMockRepo(sch)
	runner := &mockRunner{}
	exporter := &mockExporter{}
	mailer := &mockMailer{}
	now := sch.CreatedAt.Add(5 * time.Minute)
	s := New(repo, runner, exporter, mailer, Config{Clock: &fixedClock{t: now}})

	s.ProcessTick(context.Background(), now)

	if state := repo.lastRunStates[sch.ID]; state.status != "skipped" {
		t.Errorf("expected skipped without recipients, got %q", state.status)
	}
	if mailer.calls != 0 {
		t.Error("expected no mail without recipients")
	}
}

func TestScheduler_InvalidCron_Skipped(t *testing.T) {
	sch := buildSchedule(t, func(s *berichte.Schedule) { s.CronExpression = "bogus" })
	repo := newMockRepo(sch)
	runner := &mockRunner{}
	exporter := &mockExporter{}
	mailer := &mockMailer{}
	now := sch.CreatedAt.Add(5 * time.Minute)
	s := New(repo, runner, exporter, mailer, Config{Clock: &fixedClock{t: now}})

	s.ProcessTick(context.Background(), now)

	if runner.calls != 0 {
		t.Error("expected no runner call for invalid cron")
	}
}

func TestScheduler_ListError_DoesNotCrash(t *testing.T) {
	repo := newMockRepo()
	repo.listErr = errors.New("db down")
	s := New(repo, &mockRunner{}, &mockExporter{}, &mockMailer{}, Config{Clock: &fixedClock{t: time.Now()}})
	s.ProcessTick(context.Background(), time.Now())
	// no panic = success; state unchanged
}

func TestScheduler_RunStops_OnContextCancel(t *testing.T) {
	repo := newMockRepo()
	s := New(repo, &mockRunner{}, &mockExporter{}, &mockMailer{},
		Config{Clock: &fixedClock{t: time.Now()}, Tick: 10 * time.Millisecond})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- s.Run(ctx) }()

	time.Sleep(30 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("unexpected error on shutdown: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("scheduler did not stop within 1s of cancel")
	}
}

func TestBuildBodies(t *testing.T) {
	sch := buildSchedule(t, nil)
	result := &berichte.ReportResult{Meta: berichte.ReportMeta{RowCount: 42, GeneratedAt: time.Date(2026, 4, 18, 10, 0, 0, 0, time.UTC), Warning: "stale"}}
	text := buildTextBody(sch, result)
	html := buildHTMLBody(sch, result)
	if text == "" || html == "" {
		t.Error("expected non-empty bodies")
	}
}
