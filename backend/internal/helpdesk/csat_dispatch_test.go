package helpdesk

// Dispatcher behaviour without a database: one mail per due survey, no second
// mail on a second tick, a failed send retried, and a tenant that switched CSAT
// off never mailed at all.

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// Fakes
// ---------------------------------------------------------------------------

// fakeDispatchRepo models the SQL semantics of the four dispatch statements:
// a claimed row is no longer due, a released row is due again, and the attempt
// counter advances with the claim.
type fakeDispatchRepo struct {
	rows     map[uuid.UUID]*fakeSurveyRow
	listErr  error
	claimErr error
}

type fakeSurveyRow struct {
	survey    DueCsatSurvey
	sendAfter time.Time
	sentAt    *time.Time
	attempts  int16
	cancelled bool
}

func newFakeDispatchRepo() *fakeDispatchRepo {
	return &fakeDispatchRepo{rows: make(map[uuid.UUID]*fakeSurveyRow)}
}

func (r *fakeDispatchRepo) add(tenantID uuid.UUID, sendAfter time.Time) *fakeSurveyRow {
	row := &fakeSurveyRow{
		survey: DueCsatSurvey{
			ResponseID: uuid.New(), TenantID: tenantID, TicketID: uuid.New(),
			TicketNumber: 42, Subject: "Drucker <klemmt>", Token: "tok-abc",
			RecipientEmail: "cara@example.test", RecipientName: "Cara Customer",
		},
		sendAfter: sendAfter,
	}
	r.rows[row.survey.ResponseID] = row
	return row
}

func (r *fakeDispatchRepo) ListDueCsatSurveys(_ context.Context, now time.Time, maxAttempts int16, _ int) ([]*DueCsatSurvey, error) {
	if r.listErr != nil {
		return nil, r.listErr
	}
	due := make([]*DueCsatSurvey, 0)
	for _, row := range r.rows {
		if row.cancelled || row.sentAt != nil || row.attempts >= maxAttempts {
			continue
		}
		if row.sendAfter.After(now) {
			continue
		}
		s := row.survey
		due = append(due, &s)
	}
	return due, nil
}

func (r *fakeDispatchRepo) ClaimCsatSurveyDispatch(_ context.Context, responseID uuid.UUID, now time.Time) (bool, error) {
	if r.claimErr != nil {
		return false, r.claimErr
	}
	row, ok := r.rows[responseID]
	if !ok || row.cancelled || row.sentAt != nil {
		return false, nil
	}
	stamp := now
	row.sentAt = &stamp
	row.attempts++
	return true, nil
}

func (r *fakeDispatchRepo) ReleaseCsatSurveyDispatch(_ context.Context, responseID uuid.UUID, claimedAt time.Time) error {
	row, ok := r.rows[responseID]
	if !ok || row.sentAt == nil || !row.sentAt.Equal(claimedAt) {
		return nil
	}
	row.sentAt = nil
	return nil
}

func (r *fakeDispatchRepo) CancelCsatSurvey(_ context.Context, responseID uuid.UUID, _ time.Time) error {
	if row, ok := r.rows[responseID]; ok {
		row.cancelled = true
	}
	return nil
}

type fakeMailer struct {
	sent []CsatSurveyMail
	err  error
}

func (m *fakeMailer) SendCsatSurvey(_ context.Context, mail CsatSurveyMail) error {
	if m.err != nil {
		return m.err
	}
	m.sent = append(m.sent, mail)
	return nil
}

type fakeConfigLoader struct {
	cfg   CsatConfig
	err   error
	calls int
}

func (l *fakeConfigLoader) GetCsatConfig(_ context.Context, _ uuid.UUID) (CsatConfig, error) {
	l.calls++
	if l.err != nil {
		return CsatConfig{}, l.err
	}
	return l.cfg, nil
}

func quietDispatcher(repo CsatDispatchRepository, mailer CsatSurveyMailer, loader CsatConfigLoader) *CsatSurveyDispatcher {
	return NewCsatSurveyDispatcher(repo, mailer, loader, CsatDispatcherConfig{
		BaseURL: "https://app.example.test/csat/",
		Logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestDispatcher_SendsOnceAndNotAgainOnTheNextTick(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	repo := newFakeDispatchRepo()
	tenantID := uuid.New()
	repo.add(tenantID, now.Add(-time.Hour))

	mailer := &fakeMailer{}
	loader := &fakeConfigLoader{cfg: CsatConfig{
		Enabled: true, SurveyDelayMinutes: 60, SurveyQuestion: "Wie war's?",
	}}
	d := quietDispatcher(repo, mailer, loader)

	d.ProcessTick(context.Background(), now)
	d.ProcessTick(context.Background(), now.Add(time.Minute))

	if len(mailer.sent) != 1 {
		t.Fatalf("sent %d mails across two ticks, expected exactly 1", len(mailer.sent))
	}

	mail := mailer.sent[0]
	if mail.Recipient != "cara@example.test" {
		t.Fatalf("recipient = %q", mail.Recipient)
	}
	if !strings.Contains(mail.BodyText, "https://app.example.test/csat/tok-abc") {
		t.Fatalf("survey link missing from text body:\n%s", mail.BodyText)
	}
	if !strings.Contains(mail.BodyText, "Wie war's?") {
		t.Fatalf("tenant question missing from text body:\n%s", mail.BodyText)
	}
	if !strings.Contains(mail.BodyText, "Drucker <klemmt>") {
		t.Fatalf("ticket subject missing from text body:\n%s", mail.BodyText)
	}
	// The subject carries angle brackets; the HTML body must not.
	if strings.Contains(mail.BodyHTML, "<klemmt>") {
		t.Fatalf("ticket subject reached the HTML body unescaped:\n%s", mail.BodyHTML)
	}
	if !strings.Contains(mail.BodyHTML, "&lt;klemmt&gt;") {
		t.Fatalf("escaped subject missing from HTML body:\n%s", mail.BodyHTML)
	}
	if !strings.Contains(mail.Subject, "#42") {
		t.Fatalf("subject = %q, expected the ticket number", mail.Subject)
	}
}

func TestDispatcher_SkipsSurveysThatAreNotDueYet(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	repo := newFakeDispatchRepo()
	repo.add(uuid.New(), now.Add(24*time.Hour))

	mailer := &fakeMailer{}
	d := quietDispatcher(repo, mailer, &fakeConfigLoader{cfg: DefaultCsatConfig()})

	d.ProcessTick(context.Background(), now)

	if len(mailer.sent) != 0 {
		t.Fatalf("sent %d mails before the delay elapsed", len(mailer.sent))
	}
}

func TestDispatcher_FailedSendReleasesTheClaimAndRetries(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	repo := newFakeDispatchRepo()
	row := repo.add(uuid.New(), now.Add(-time.Hour))

	mailer := &fakeMailer{err: errors.New("smtp down")}
	d := quietDispatcher(repo, mailer, &fakeConfigLoader{cfg: DefaultCsatConfig()})

	d.ProcessTick(context.Background(), now)
	if row.sentAt != nil {
		t.Fatal("a failed send left the survey marked as sent")
	}
	if row.attempts != 1 {
		t.Fatalf("attempts = %d after one failed send, expected 1", row.attempts)
	}

	// SMTP recovers; the next tick delivers.
	mailer.err = nil
	d.ProcessTick(context.Background(), now.Add(time.Minute))
	if len(mailer.sent) != 1 {
		t.Fatalf("sent %d mails after recovery, expected 1", len(mailer.sent))
	}
	if row.sentAt == nil {
		t.Fatal("a successful send did not stick")
	}
}

func TestDispatcher_DisabledTenantCancelsInsteadOfSending(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	repo := newFakeDispatchRepo()
	row := repo.add(uuid.New(), now.Add(-time.Hour))

	mailer := &fakeMailer{}
	d := quietDispatcher(repo, mailer, &fakeConfigLoader{cfg: CsatConfig{Enabled: false}})

	d.ProcessTick(context.Background(), now)

	if len(mailer.sent) != 0 {
		t.Fatalf("sent %d mails for a tenant with csat switched off", len(mailer.sent))
	}
	if !row.cancelled {
		t.Fatal("the pending survey link was not revoked")
	}
	if row.sentAt != nil {
		t.Fatal("a cancelled survey was marked as sent")
	}
}

func TestDispatcher_UnreadableConfigDefersInsteadOfBurningTheSurvey(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	repo := newFakeDispatchRepo()
	row := repo.add(uuid.New(), now.Add(-time.Hour))

	mailer := &fakeMailer{}
	loader := &fakeConfigLoader{err: errors.New("settings unreachable")}
	d := quietDispatcher(repo, mailer, loader)

	d.ProcessTick(context.Background(), now)

	if len(mailer.sent) != 0 {
		t.Fatalf("sent %d mails without a readable config", len(mailer.sent))
	}
	if row.attempts != 0 {
		t.Fatalf("attempts = %d, a config outage must not spend the survey's retries", row.attempts)
	}
	if row.cancelled || row.sentAt != nil {
		t.Fatal("a config outage changed the survey's state")
	}
}

func TestDispatcher_LooksTenantConfigUpOncePerTick(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()
	tenantID := uuid.New()
	repo := newFakeDispatchRepo()
	repo.add(tenantID, now.Add(-time.Hour))
	repo.add(tenantID, now.Add(-time.Hour))
	repo.add(tenantID, now.Add(-time.Hour))

	loader := &fakeConfigLoader{cfg: DefaultCsatConfig()}
	d := quietDispatcher(repo, &fakeMailer{}, loader)

	d.ProcessTick(context.Background(), now)

	if loader.calls != 1 {
		t.Fatalf("config was loaded %d times for one tenant in one tick, expected 1", loader.calls)
	}
}
