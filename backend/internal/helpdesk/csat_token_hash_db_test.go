package helpdesk

// Decision 2026-10-26 against a real database as kmuhub_app: the survey link
// is minted by the dispatcher at send time, the table only ever holds its
// SHA-256, a retry after a failed send replaces the hash, and links minted
// before 000365 (backfilled with the migration's SQL expression) still redeem.

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/linksecret"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

// oneTicketDispatchRepo narrows the due query to one ticket. ProcessTick runs
// cross-tenant on purpose; without the filter this test would claim and mail
// the surveys of every parallel test in the package.
type oneTicketDispatchRepo struct {
	*PostgresRepository
	ticketID uuid.UUID
}

func (r oneTicketDispatchRepo) ListDueCsatSurveys(ctx context.Context, now time.Time, maxAttempts int16, limit int) ([]*DueCsatSurvey, error) {
	due, err := r.PostgresRepository.ListDueCsatSurveys(ctx, now, maxAttempts, limit)
	if err != nil {
		return nil, err
	}
	if s := surveyFor(due, r.ticketID); s != nil {
		return []*DueCsatSurvey{s}, nil
	}
	return nil, nil
}

// recordingMailer keeps every mail it is handed, including the ones it then
// reports as failed -- a failed SMTP call may still have delivered.
type recordingMailer struct {
	mails []CsatSurveyMail
	err   error
}

func (m *recordingMailer) SendCsatSurvey(_ context.Context, mail CsatSurveyMail) error {
	m.mails = append(m.mails, mail)
	return m.err
}

func TestCsatDispatcher_DB_MintsLinkAtSendAndStoresOnlyTheHash(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Helpdesk CSAT Token Hash Tenant")

	now := time.Now().UTC()
	sysCtx := testutil.WithSystemCtx(context.Background())
	repo, ticketID := seedDueSurvey(t, pool, tenantID, "hash", now, -time.Hour)

	mailer := &recordingMailer{err: errors.New("smtp down")}
	d := NewCsatSurveyDispatcher(
		oneTicketDispatchRepo{PostgresRepository: repo, ticketID: ticketID},
		mailer,
		&fakeConfigLoader{cfg: enabledCsatConfig()},
		CsatDispatcherConfig{
			BaseURL: "https://app.example.test/csat",
			Logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		},
	)
	svc := NewService(repo, slog.New(slog.NewTextHandler(io.Discard, nil)))

	// First tick: SMTP fails after the claim. The claim already stored the
	// hash of the link that was handed to the mailer.
	d.ProcessTick(sysCtx, now)
	if len(mailer.mails) != 1 {
		t.Fatalf("first tick handed %d mails to the transport, expected 1", len(mailer.mails))
	}
	failedLink := mailedToken(t, mailer.mails[0])
	if got := csatTokenHashOf(t, pool, sysCtx, ticketID); got != linksecret.Hash(failedLink) {
		t.Fatalf("after the failed send the row holds %q, expected the hash of the link handed out", got)
	}

	// Second tick: SMTP is back, the retry draws a new link.
	mailer.err = nil
	d.ProcessTick(sysCtx, now.Add(time.Minute))
	if len(mailer.mails) != 2 {
		t.Fatalf("retry handed %d mails in total to the transport, expected 2", len(mailer.mails))
	}
	sentLink := mailedToken(t, mailer.mails[1])
	if sentLink == failedLink {
		t.Fatal("the retry mailed the link of the failed attempt again")
	}
	if got := csatTokenHashOf(t, pool, sysCtx, ticketID); got != linksecret.Hash(sentLink) {
		t.Fatalf("after the retry the row holds %q, expected the hash of the mailed link", got)
	}

	// Nothing in the row spells out either link: only the hash is persisted.
	var row string
	if err := pool.QueryRow(sysCtx,
		`SELECT row_to_json(r)::text FROM ticket_csat_responses r WHERE ticket_id = $1`, ticketID,
	).Scan(&row); err != nil {
		t.Fatalf("read csat row: %v", err)
	}
	for _, link := range []string{failedLink, sentLink} {
		if strings.Contains(row, link) {
			t.Fatalf("the clear link %q is stored in the row: %s", link, row)
		}
	}

	// The link of the failed attempt is dead, the mailed one redeems -- through
	// the public path, without a tenant.
	if _, err := svc.SubmitCsatByToken(context.Background(), failedLink, 2, nil); !errors.Is(err, ErrCsatSurveyNotFound) {
		t.Fatalf("the replaced link still redeems: err=%v", err)
	}
	res, err := svc.SubmitCsatByToken(context.Background(), sentLink, 4, nil)
	if err != nil {
		t.Fatalf("the mailed link does not redeem: %v", err)
	}
	if res.Rating != 4 {
		t.Fatalf("redemption reported rating %d, expected 4", res.Rating)
	}
}

// A link mailed before 000365 lives on only as the hash the migration's
// backfill computed in SQL. This pins that the SQL expression and
// linksecret.Hash agree, so those links keep redeeming.
func TestSubmitCsatByToken_DB_LinkBackfilledByMigrationRedeems(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Helpdesk CSAT Legacy Link Tenant")

	now := time.Now().UTC()
	sysCtx := testutil.WithSystemCtx(context.Background())
	repo, ticketID := seedDueSurvey(t, pool, tenantID, "legacy", now, -time.Hour)

	// Same statement shape as the 000365 backfill, run on a legacy token that
	// carries a non-ASCII byte to make the UTF-8 conversion count.
	legacy := "legacy-ü-" + uuid.NewString()
	if _, err := pool.Exec(sysCtx,
		`UPDATE ticket_csat_responses
		    SET token_hash = encode(sha256(convert_to($2::text, 'UTF8')), 'hex'),
		        survey_sent_at = $3
		  WHERE ticket_id = $1`,
		ticketID, legacy, now,
	); err != nil {
		t.Fatalf("backfill legacy hash: %v", err)
	}

	svc := NewService(repo, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if _, err := svc.SubmitCsatByToken(context.Background(), legacy, 5, nil); err != nil {
		t.Fatalf("a link backfilled by the migration does not redeem: %v", err)
	}
}
