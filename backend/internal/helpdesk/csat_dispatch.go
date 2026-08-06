package helpdesk

import (
	"context"
	"fmt"
	"html"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/database"
	"github.com/kmuhub/kmuhub/internal/middleware"
)

// ---------------------------------------------------------------------------
// Collaborators
// ---------------------------------------------------------------------------

// DueCsatSurvey is one pending survey whose send time has arrived, joined with
// everything the invitation mail needs: which ticket it is about and who gets
// asked.
type DueCsatSurvey struct {
	ResponseID     uuid.UUID
	TenantID       uuid.UUID
	TicketID       uuid.UUID
	TicketNumber   int
	Subject        string
	Token          string
	RecipientEmail string
	RecipientName  string
}

// CsatDispatchRepository is the narrow persistence surface the dispatcher
// needs. Kept out of the fat Repository interface on purpose: nothing but this
// background job touches these four statements, and a narrow interface keeps
// the dispatcher testable without a full ticket repository (same split as
// berichte/scheduler.ScheduleRepository).
type CsatDispatchRepository interface {
	ListDueCsatSurveys(ctx context.Context, now time.Time, maxAttempts int16, limit int) ([]*DueCsatSurvey, error)
	ClaimCsatSurveyDispatch(ctx context.Context, responseID uuid.UUID, now time.Time) (bool, error)
	ReleaseCsatSurveyDispatch(ctx context.Context, responseID uuid.UUID, claimedAt time.Time) error
	CancelCsatSurvey(ctx context.Context, responseID uuid.UUID, now time.Time) error
}

// CsatSurveyMail is one rendered invitation, ready to hand to a transport.
type CsatSurveyMail struct {
	TenantID  uuid.UUID
	TicketID  uuid.UUID
	Recipient string
	Name      string
	Subject   string
	BodyText  string
	BodyHTML  string
}

// CsatSurveyMailer delivers a rendered invitation. Kept narrow so the
// dispatcher can be tested without SMTP; the concrete implementation over the
// system SMTP account is SystemMailCsatMailer.
type CsatSurveyMailer interface {
	SendCsatSurvey(ctx context.Context, m CsatSurveyMail) error
}

// CsatConfigLoader resolves a tenant's CSAT configuration. The helpdesk gRPC
// server already implements exactly this (over the settings service), so
// cmd/helpdesk wires it in rather than growing a second settings client.
type CsatConfigLoader interface {
	GetCsatConfig(ctx context.Context, tenantID uuid.UUID) (CsatConfig, error)
}

// ---------------------------------------------------------------------------
// Dispatcher
// ---------------------------------------------------------------------------

const (
	// csatSurveyMaxDispatchAttempts bounds delivery retries. Without a cap a
	// permanently undeliverable address would be retried on every tick until
	// the token expires 30 days later.
	csatSurveyMaxDispatchAttempts int16 = 3

	// csatSurveyDefaultTick is how often due surveys are looked for. The delay
	// is configured in minutes, so minute-ish granularity is plenty.
	csatSurveyDefaultTick = 5 * time.Minute

	// csatSurveyDefaultBatch caps how many surveys one tick handles, so a
	// backlog is worked off over several ticks instead of blocking one.
	csatSurveyDefaultBatch = 100

	// defaultCsatSurveyBaseURL matches config.Config.CsatSurveyBaseURL's
	// default and only applies when the dispatcher is built without one.
	defaultCsatSurveyBaseURL = "https://app.zentria.tech/csat"
)

// CsatDispatcherConfig bundles the dispatcher's optional settings. Zero values
// default sensibly.
type CsatDispatcherConfig struct {
	// Tick is the poll interval. Defaults to csatSurveyDefaultTick.
	Tick time.Duration
	// BaseURL prefixes the survey link; the token is appended as the last path
	// segment. Defaults to defaultCsatSurveyBaseURL.
	BaseURL string
	// Batch caps surveys per tick. Defaults to csatSurveyDefaultBatch.
	Batch int
	// Now is mockable wall clock. Defaults to time.Now().UTC.
	Now func() time.Time
	// Logger defaults to slog.Default().
	Logger *slog.Logger
}

// CsatSurveyDispatcher mails out the survey invitations that CloseTicket
// parked on ticket_csat_responses.
//
// It follows berichte/scheduler: poll, claim atomically, then act. The claim is
// what keeps two overlapping ticks (or two helpdesk instances) from mailing the
// same customer twice -- a check-then-act on survey_sent_at would not.
type CsatSurveyDispatcher struct {
	repo    CsatDispatchRepository
	mailer  CsatSurveyMailer
	configs CsatConfigLoader
	baseURL string
	tick    time.Duration
	batch   int
	now     func() time.Time
	log     *slog.Logger
}

// NewCsatSurveyDispatcher wires a dispatcher. mailer must not be nil: a
// dispatcher without a transport would claim surveys it can never deliver, so
// cmd/helpdesk simply does not start one when SMTP is unconfigured.
func NewCsatSurveyDispatcher(
	repo CsatDispatchRepository,
	mailer CsatSurveyMailer,
	configs CsatConfigLoader,
	cfg CsatDispatcherConfig,
) *CsatSurveyDispatcher {
	d := &CsatSurveyDispatcher{
		repo:    repo,
		mailer:  mailer,
		configs: configs,
		baseURL: strings.TrimRight(cfg.BaseURL, "/"),
		tick:    cfg.Tick,
		batch:   cfg.Batch,
		now:     cfg.Now,
		log:     cfg.Logger,
	}
	if d.baseURL == "" {
		d.baseURL = defaultCsatSurveyBaseURL
	}
	if d.tick <= 0 {
		d.tick = csatSurveyDefaultTick
	}
	if d.batch <= 0 {
		d.batch = csatSurveyDefaultBatch
	}
	if d.now == nil {
		d.now = func() time.Time { return time.Now().UTC() }
	}
	if d.log == nil {
		d.log = slog.Default()
	}
	return d
}

// Run polls until ctx is cancelled. The first tick runs immediately so a
// restart does not add a full interval to an already overdue survey.
//
// The whole loop holds a system context: there is no logged-in user behind a
// poller, and the due query is cross-tenant by nature. Every row carries its
// own tenant from there on -- same shape as the berichte scheduler.
func (d *CsatSurveyDispatcher) Run(ctx context.Context) {
	ctx = database.WithSystemContext(ctx)
	d.log.Info("helpdesk csat survey dispatcher started", "tick", d.tick.String())
	defer d.log.Info("helpdesk csat survey dispatcher stopped")

	d.ProcessTick(ctx, d.now())

	ticker := time.NewTicker(d.tick)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			d.ProcessTick(ctx, d.now())
		}
	}
}

// ProcessTick runs a single dispatch pass. Exported for tests and for a future
// manual trigger.
func (d *CsatSurveyDispatcher) ProcessTick(ctx context.Context, now time.Time) {
	due, err := d.repo.ListDueCsatSurveys(ctx, now, csatSurveyMaxDispatchAttempts, d.batch)
	if err != nil {
		d.log.ErrorContext(ctx, "helpdesk csat dispatcher: list due surveys failed", "error", err)
		return
	}
	if len(due) == 0 {
		return
	}

	// One config lookup per tenant per tick, not per survey: a busy tenant
	// would otherwise fire one settings RPC per closed ticket.
	configs := make(map[uuid.UUID]CsatConfig, 4)
	for _, s := range due {
		cfg, ok := configs[s.TenantID]
		if !ok {
			// The settings service reads tenant_settings under RLS keyed on the
			// tenant in the *call context*, not on the tenant_id in the request
			// body. Called from the poller's tenantless system context it would
			// find no rows and quietly hand back defaults -- so the row's own
			// tenant goes on the context for this one call.
			cfg, err = d.configs.GetCsatConfig(withTenant(ctx, s.TenantID), s.TenantID)
			if err != nil {
				// Leave the row untouched and unclaimed; the next tick retries.
				d.log.WarnContext(ctx, "helpdesk csat dispatcher: config unavailable, survey deferred",
					"tenant_id", s.TenantID, "ticket_id", s.TicketID, "error", err)
				continue
			}
			configs[s.TenantID] = cfg
		}
		d.dispatchOne(ctx, s, cfg, now)
	}
}

func (d *CsatSurveyDispatcher) dispatchOne(ctx context.Context, s *DueCsatSurvey, cfg CsatConfig, now time.Time) {
	// The tenant switched surveys off between the close and the delayed send.
	// Revoking beats skipping: a skipped row would come back every tick for as
	// long as the token lives, and the link would stay redeemable.
	if !cfg.Enabled {
		if err := d.repo.CancelCsatSurvey(ctx, s.ResponseID, now); err != nil {
			d.log.WarnContext(ctx, "helpdesk csat dispatcher: cancel failed",
				"ticket_id", s.TicketID, "error", err)
			return
		}
		d.log.InfoContext(ctx, "helpdesk csat dispatcher: survey cancelled, tenant disabled csat",
			"tenant_id", s.TenantID, "ticket_id", s.TicketID)
		return
	}

	claimed, err := d.repo.ClaimCsatSurveyDispatch(ctx, s.ResponseID, now)
	if err != nil {
		d.log.ErrorContext(ctx, "helpdesk csat dispatcher: claim failed",
			"ticket_id", s.TicketID, "error", err)
		return
	}
	if !claimed {
		// Another tick got there first, or the customer rated in the meantime.
		d.log.DebugContext(ctx, "helpdesk csat dispatcher: claim lost", "ticket_id", s.TicketID)
		return
	}

	if err := d.mailer.SendCsatSurvey(ctx, d.render(s, cfg)); err != nil {
		d.log.WarnContext(ctx, "helpdesk csat dispatcher: send failed, claim released",
			"ticket_id", s.TicketID, "error", err)
		if relErr := d.repo.ReleaseCsatSurveyDispatch(ctx, s.ResponseID, now); relErr != nil {
			d.log.ErrorContext(ctx, "helpdesk csat dispatcher: release failed, survey stays claimed",
				"ticket_id", s.TicketID, "error", relErr)
		}
		return
	}

	d.log.InfoContext(ctx, "helpdesk csat dispatcher: survey sent",
		"tenant_id", s.TenantID, "ticket_id", s.TicketID, "ticket_number", s.TicketNumber)
}

// ---------------------------------------------------------------------------
// Rendering
// ---------------------------------------------------------------------------

// render builds the invitation. Bodies are assembled from a fixed shape with
// the tenant's own question, deliberately not through email/template.Service:
// that renders a tenant-owned template row looked up by ID with visibility
// rules, and there is no CSAT template row to look up.
func (d *CsatSurveyDispatcher) render(s *DueCsatSurvey, cfg CsatConfig) CsatSurveyMail {
	question := strings.TrimSpace(cfg.SurveyQuestion)
	if question == "" {
		question = DefaultCsatConfig().SurveyQuestion
	}
	link := d.surveyLink(s.Token)

	greeting := "Hallo"
	if name := strings.TrimSpace(s.RecipientName); name != "" {
		greeting = "Hallo " + name
	}

	subject := fmt.Sprintf("[Cosmi] Ihre Rückmeldung zu Ticket #%d", s.TicketNumber)

	bodyText := fmt.Sprintf(
		"%s,\n\nIhr Ticket #%d (%s) wurde abgeschlossen.\n\n%s\n\nBewerten Sie hier mit einem Klick:\n%s\n\nVielen Dank!\n",
		greeting, s.TicketNumber, s.Subject, question, link,
	)

	bodyHTML := fmt.Sprintf(
		`<p>%s,</p><p>Ihr Ticket <strong>#%d</strong> (%s) wurde abgeschlossen.</p><p>%s</p><p><a href="%s">Jetzt bewerten</a></p><p>Vielen Dank!</p>`,
		html.EscapeString(greeting), s.TicketNumber, html.EscapeString(s.Subject),
		html.EscapeString(question), html.EscapeString(link),
	)

	return CsatSurveyMail{
		TenantID:  s.TenantID,
		TicketID:  s.TicketID,
		Recipient: s.RecipientEmail,
		Name:      s.RecipientName,
		Subject:   subject,
		BodyText:  bodyText,
		BodyHTML:  bodyHTML,
	}
}

// withTenant puts a tenant resolved from a survey row on the context, so the
// outbound gRPC interceptor propagates it and the receiving service's RLS
// session sees it. Same single-purpose helper as berichte.WithTenant on the
// public share path: it is only ever called with the tenant the row itself
// resolved to, and never widens what the caller may reach.
func withTenant(ctx context.Context, tenantID uuid.UUID) context.Context {
	return context.WithValue(ctx, middleware.TenantIDKey, tenantID.String())
}

// surveyLink appends the token as the last path segment. The token is
// base64url, so it needs no escaping.
func (d *CsatSurveyDispatcher) surveyLink(token string) string {
	return d.baseURL + "/" + token
}
