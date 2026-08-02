package delivery_test

// End-to-end proof for the scheduled-report chain: a due schedule is rendered
// by the real exporter, handed to the real system mailer, and lands on an SMTP
// wire with the report as an attachment. The scheduler's own tests use fake
// collaborators and therefore cannot show that the rendered bytes ever reach a
// recipient -- exactly the gap that let "scheduled reports" look wired while
// cmd/berichte still passed nil for both exporter and mailer.

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/berichte"
	"github.com/kmuhub/kmuhub/internal/berichte/delivery"
	"github.com/kmuhub/kmuhub/internal/berichte/scheduler"
	"github.com/kmuhub/kmuhub/internal/email/systemmail"
	"github.com/kmuhub/kmuhub/internal/testutil/fakesmtp"
)

// ---------------------------------------------------------------------------
// Fakes for the two collaborators that are not under test here
// ---------------------------------------------------------------------------

type lastRun struct {
	status string
	err    *string
}

type fakeRepo struct {
	schedule *berichte.Schedule
	claimed  bool
	runs     []lastRun
}

func (r *fakeRepo) ListDueSchedules(context.Context, time.Time) ([]*berichte.Schedule, error) {
	return []*berichte.Schedule{r.schedule}, nil
}

func (r *fakeRepo) ClaimSchedule(_ context.Context, _ uuid.UUID, _ *time.Time, _ time.Time) (bool, error) {
	if r.claimed {
		return false, nil
	}
	r.claimed = true
	return true, nil
}

func (r *fakeRepo) UpdateScheduleLastRun(_ context.Context, _ uuid.UUID, status string, runErr *string, _ time.Time) error {
	r.runs = append(r.runs, lastRun{status: status, err: runErr})
	return nil
}

type fakeRunner struct{ generatedAt time.Time }

func (f *fakeRunner) RunReport(context.Context, berichte.RunReportInput) (*berichte.ReportResult, *berichte.Run, error) {
	return &berichte.ReportResult{
		Columns: []berichte.Column{
			{Name: "name", Label: "Name", Kind: "string"},
			{Name: "revenue", Label: "Umsatz (EUR)", Kind: "currency"},
		},
		Rows: []map[string]any{
			{"name": "Kunde A", "revenue": 12345.67},
		},
		Meta: berichte.ReportMeta{GeneratedAt: f.generatedAt, RowCount: 1},
	}, &berichte.Run{}, nil
}

type fixedClock struct{ t time.Time }

func (c fixedClock) Now() time.Time { return c.t }

// ---------------------------------------------------------------------------

func TestScheduledReport_RenderedAndDelivered(t *testing.T) {
	srv := fakesmtp.Start(t)
	sender := systemmail.New(systemmail.Config{
		Host: srv.Host,
		Port: srv.Port,
		From: "noreply@zentria.test",
	})

	created := time.Date(2026, 8, 2, 6, 0, 0, 0, time.UTC)
	now := created.Add(5 * time.Minute)

	repo := &fakeRepo{schedule: &berichte.Schedule{
		ID:             uuid.New(),
		TenantID:       uuid.New(),
		DefinitionID:   uuid.New(),
		Name:           "Umsatzübersicht Q3",
		CronExpression: "* * * * *",
		Recipients:     []string{"buchhaltung@example.test", "gf@example.test"},
		Format:         "csv",
		Params:         json.RawMessage(`{}`),
		Active:         true,
		CreatedAt:      created,
		UpdatedAt:      created,
	}}

	s := scheduler.New(repo, &fakeRunner{generatedAt: now},
		delivery.NewExporter(), delivery.NewMailer(sender),
		scheduler.Config{Clock: fixedClock{t: now}})

	s.ProcessTick(context.Background(), now)

	if len(repo.runs) != 1 {
		t.Fatalf("last-run updates = %d, want exactly 1", len(repo.runs))
	}
	if repo.runs[0].status != "success" {
		errText := "<nil>"
		if repo.runs[0].err != nil {
			errText = *repo.runs[0].err
		}
		t.Fatalf("last_run_status = %q (error: %s), want success", repo.runs[0].status, errText)
	}

	sess := srv.WaitSession(t, 5*time.Second)
	if len(sess.RcptTo) != 2 {
		t.Errorf("RCPT TO = %v, want both schedule recipients", sess.RcptTo)
	}
	wantFile := "umsatzuebersicht-q3_2026-08-02.csv"
	if !strings.Contains(sess.Data, wantFile) {
		t.Errorf("delivered message does not carry the attachment %q", wantFile)
	}

	// The attachment must contain the actual report, not an empty buffer.
	body := decodeAttachment(t, sess.Data, wantFile)
	if !strings.Contains(body, "Kunde A") {
		t.Errorf("attachment does not contain the report rows, got:\n%s", body)
	}
}

// A schedule whose mailer refuses delivery must be recorded as failed -- never
// as a success the operator would read as "report went out".
func TestScheduledReport_UnconfiguredMailerMarksFailed(t *testing.T) {
	created := time.Date(2026, 8, 2, 6, 0, 0, 0, time.UTC)
	now := created.Add(5 * time.Minute)

	repo := &fakeRepo{schedule: &berichte.Schedule{
		ID:             uuid.New(),
		TenantID:       uuid.New(),
		DefinitionID:   uuid.New(),
		Name:           "Umsatzübersicht Q3",
		CronExpression: "* * * * *",
		Recipients:     []string{"buchhaltung@example.test"},
		Format:         "csv",
		Params:         json.RawMessage(`{}`),
		Active:         true,
		CreatedAt:      created,
		UpdatedAt:      created,
	}}

	s := scheduler.New(repo, &fakeRunner{generatedAt: now},
		delivery.NewExporter(), delivery.NewMailer(systemmail.New(systemmail.Config{})),
		scheduler.Config{Clock: fixedClock{t: now}})

	s.ProcessTick(context.Background(), now)

	if len(repo.runs) != 1 || repo.runs[0].status != "failed" {
		t.Fatalf("last runs = %+v, want a single failed run", repo.runs)
	}
}

// decodeAttachment pulls the base64 body of the MIME part whose
// Content-Disposition names filename and decodes it.
func decodeAttachment(t *testing.T, message, filename string) string {
	t.Helper()

	idx := strings.Index(message, filename)
	if idx < 0 {
		t.Fatalf("attachment %q not present in message", filename)
	}
	rest := message[idx:]

	// Skip the remaining part headers: the body starts after the blank line.
	sep := strings.Index(rest, "\r\n\r\n")
	if sep < 0 {
		t.Fatal("attachment part has no body separator")
	}
	body := rest[sep+4:]

	var encoded strings.Builder
	for _, line := range strings.Split(body, "\r\n") {
		if strings.HasPrefix(line, "--") || line == "" {
			break
		}
		encoded.WriteString(line)
	}

	raw, err := base64.StdEncoding.DecodeString(encoded.String())
	if err != nil {
		t.Fatalf("decode attachment: %v", err)
	}
	return string(raw)
}
