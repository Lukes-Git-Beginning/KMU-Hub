package server

import (
	"context"
	"log/slog"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	"github.com/kmuhub/kmuhub/internal/models"
	bizv1 "github.com/kmuhub/kmuhub/proto/biz/v1"
)

// bizActorCtx is a tenant context with some authenticated caller, for tests that
// only need the RPC to get past callerID.
func bizActorCtx(tenantID uuid.UUID) context.Context {
	return ctxWithTenantAndUser(tenantID, uuid.New())
}

// attrCapture records the attributes of every slog record it sees.
type attrCapture struct {
	mu      sync.Mutex
	records []map[string]any
}

func (c *attrCapture) Enabled(context.Context, slog.Level) bool { return true }
func (c *attrCapture) WithAttrs([]slog.Attr) slog.Handler       { return c }
func (c *attrCapture) WithGroup(string) slog.Handler            { return c }
func (c *attrCapture) Handle(_ context.Context, r slog.Record) error {
	attrs := map[string]any{"msg": r.Message}
	r.Attrs(func(a slog.Attr) bool { attrs[a.Key] = a.Value.Any(); return true })
	c.mu.Lock()
	c.records = append(c.records, attrs)
	c.mu.Unlock()
	return nil
}

func (c *attrCapture) find(msg string, dunningID uuid.UUID) map[string]any {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, r := range c.records {
		if r["msg"] == msg && r["dunning_id"] == dunningID {
			return r
		}
	}
	return nil
}

// The finance send/convert RPCs take the actor from the caller context, never
// from a hardcoded uuid.Nil: a request without a caller is refused up front.
func TestBizGRPCServer_FinanceRPCsRefuseMissingCaller(t *testing.T) {
	tenantID := uuid.New()
	tenantOnly := ctxWithTenant(tenantID)
	id := uuid.New().String()
	tid := tenantID.String()
	srv := newFinanceTestServer(newStubInvoiceRepo(), newStubCreditNoteRepo(), newStubPaymentRepo(), &stubCompanySettingsRepo{}, nil, nil)
	srv.quoteService = newQuoteTestServer(newStubQuoteRepo(), &stubCompanySettingsRepo{}, nil).quoteService
	srv.dunningService = newDunningTestServer(newStubDunningRecordRepo(), &stubDunningConfigRepo{}, &stubDunningInvoiceReader{}).dunningService

	calls := map[string]func() error{
		"SendQuote":      func() error { _, err := srv.SendQuote(tenantOnly, &bizv1.SendQuoteRequest{TenantId: tid, Id: id}); return err },
		"SendInvoice":    func() error { _, err := srv.SendInvoice(tenantOnly, &bizv1.SendInvoiceRequest{TenantId: tid, Id: id}); return err },
		"CancelInvoice":  func() error { _, err := srv.CancelInvoice(tenantOnly, &bizv1.CancelInvoiceRequest{TenantId: tid, Id: id}); return err },
		"SendCreditNote": func() error { _, err := srv.SendCreditNote(tenantOnly, &bizv1.SendCreditNoteRequest{TenantId: tid, Id: id}); return err },
		"SendDunning":    func() error { _, err := srv.SendDunning(tenantOnly, &bizv1.SendDunningRequest{TenantId: tid, Id: id}); return err },
		"SendDunningNotice": func() error {
			_, err := srv.SendDunningNotice(tenantOnly, &bizv1.SendDunningNoticeRequest{TenantId: tid, Id: id})
			return err
		},
		"ConvertQuoteToInvoice": func() error {
			_, err := srv.ConvertQuoteToInvoice(tenantOnly, &bizv1.ConvertQuoteToInvoiceRequest{TenantId: tid, Id: id})
			return err
		},
	}
	for name, call := range calls {
		t.Run(name, func(t *testing.T) {
			requireGRPCCode(t, call(), codes.Unauthenticated)
		})
	}
}

// Sending a dunning notice logs who sent it; that must be the caller, not the
// zero UUID.
func TestBizGRPCServer_SendDunning_LogsTheCallerAsSender(t *testing.T) {
	for name, send := range map[string]func(*BizGRPCServer, context.Context, uuid.UUID, uuid.UUID) error{
		"SendDunning": func(s *BizGRPCServer, ctx context.Context, tenantID, id uuid.UUID) error {
			_, err := s.SendDunning(ctx, &bizv1.SendDunningRequest{TenantId: tenantID.String(), Id: id.String()})
			return err
		},
		"SendDunningNotice": func(s *BizGRPCServer, ctx context.Context, tenantID, id uuid.UUID) error {
			_, err := s.SendDunningNotice(ctx, &bizv1.SendDunningNoticeRequest{TenantId: tenantID.String(), Id: id.String()})
			return err
		},
	} {
		t.Run(name, func(t *testing.T) {
			tenantID, dunningID, actor := uuid.New(), uuid.New(), uuid.New()
			repo := newStubDunningRecordRepo()
			repo.records[dunningID] = &models.DunningRecord{ID: dunningID, TenantID: tenantID, Status: models.DunningStatusDraft}
			srv := newDunningTestServer(repo, &stubDunningConfigRepo{}, &stubDunningInvoiceReader{})

			capture := &attrCapture{}
			prev := slog.Default()
			slog.SetDefault(slog.New(capture))
			defer slog.SetDefault(prev)

			require.NoError(t, send(srv, ctxWithTenantAndUser(tenantID, actor), tenantID, dunningID))

			rec := capture.find("dunning notice sent", dunningID)
			require.NotNil(t, rec, "the send must log who sent the notice")
			assert.Equal(t, actor, rec["sent_by"])
		})
	}
}
