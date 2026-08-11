package server

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	"github.com/kmuhub/kmuhub/internal/crm/advisoryprotocol"
	"github.com/kmuhub/kmuhub/internal/middleware"
	crmv1 "github.com/kmuhub/kmuhub/proto/crm/v1"
)

// ---------------------------------------------------------------------------
// Fake advisoryprotocol.Repository
// ---------------------------------------------------------------------------

type fakeAdvisoryRepo struct {
	protocols map[uuid.UUID]*advisoryprotocol.Protocol
	contactOK map[uuid.UUID]bool
	referral  []*advisoryprotocol.ReferralEntry
}

func newFakeAdvisoryRepo() *fakeAdvisoryRepo {
	return &fakeAdvisoryRepo{
		protocols: make(map[uuid.UUID]*advisoryprotocol.Protocol),
		contactOK: make(map[uuid.UUID]bool),
	}
}

func (r *fakeAdvisoryRepo) Create(_ context.Context, p *advisoryprotocol.Protocol) error {
	r.protocols[p.ID] = p
	return nil
}

func (r *fakeAdvisoryRepo) GetByID(_ context.Context, id, tenantID uuid.UUID) (*advisoryprotocol.Protocol, error) {
	p, ok := r.protocols[id]
	if !ok || p.TenantID != tenantID {
		return nil, advisoryprotocol.ErrProtocolNotFound
	}
	return p, nil
}

func (r *fakeAdvisoryRepo) ListByContact(_ context.Context, contactID, tenantID uuid.UUID) ([]*advisoryprotocol.Protocol, error) {
	var out []*advisoryprotocol.Protocol
	for _, p := range r.protocols {
		if p.ContactID == contactID && p.TenantID == tenantID {
			out = append(out, p)
		}
	}
	return out, nil
}

func (r *fakeAdvisoryRepo) Update(_ context.Context, p *advisoryprotocol.Protocol) error {
	r.protocols[p.ID] = p
	return nil
}

func (r *fakeAdvisoryRepo) Delete(_ context.Context, id, tenantID uuid.UUID) error {
	p, ok := r.protocols[id]
	if !ok || p.TenantID != tenantID {
		return advisoryprotocol.ErrProtocolNotFound
	}
	delete(r.protocols, id)
	return nil
}

func (r *fakeAdvisoryRepo) HandOver(_ context.Context, id, tenantID uuid.UUID, at time.Time) error {
	p, ok := r.protocols[id]
	if !ok || p.TenantID != tenantID {
		return advisoryprotocol.ErrProtocolNotFound
	}
	p.Status = "finalized"
	p.HandedOverAt = &at
	return nil
}

func (r *fakeAdvisoryRepo) ContactExists(_ context.Context, contactID, _ uuid.UUID) (bool, error) {
	return r.contactOK[contactID], nil
}

func (r *fakeAdvisoryRepo) GetReferralReport(_ context.Context, _ uuid.UUID) ([]*advisoryprotocol.ReferralEntry, error) {
	return r.referral, nil
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

func newTestAdvisoryCRMServer() (*CRMGRPCServer, *fakeAdvisoryRepo) {
	repo := newFakeAdvisoryRepo()
	s := &CRMGRPCServer{}
	s.SetAdvisoryProtocolService(advisoryprotocol.NewService(repo))
	return s, repo
}

func advisoryCtx(tenantID, userID uuid.UUID) context.Context {
	ctx := context.WithValue(context.Background(), middleware.TenantIDKey, tenantID.String())
	ctx = context.WithValue(ctx, middleware.UserIDKey, userID.String())
	return ctx
}

func seedAdvisoryProtocol(repo *fakeAdvisoryRepo, tenantID, contactID uuid.UUID, status string) *advisoryprotocol.Protocol {
	now := time.Now()
	p := &advisoryprotocol.Protocol{
		ID:                uuid.New(),
		TenantID:          tenantID,
		ContactID:         contactID,
		CreatedBy:         uuid.New(),
		Status:            status,
		KnownAssetClasses: []string{},
		InvestmentPurpose: []string{},
		WarningsGiven:     []string{},
		Products:          []advisoryprotocol.Product{},
		RiskClass:         3,
		DeliveryForm:      "email",
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	if status == "finalized" {
		at := now
		p.HandedOverAt = &at
	}
	repo.protocols[p.ID] = p
	return p
}

// ---------------------------------------------------------------------------
// mapCRMError — advisory protocol sentinels
// ---------------------------------------------------------------------------

func TestMapCRMError_AdvisoryProtocol(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode codes.Code
	}{
		{"protocol not found", advisoryprotocol.ErrProtocolNotFound, codes.NotFound},
		{"protocol finalized", advisoryprotocol.ErrProtocolFinalized, codes.FailedPrecondition},
		{"contact not found (advisory)", advisoryprotocol.ErrContactNotFound, codes.NotFound},
		{"invalid risk class", advisoryprotocol.ErrInvalidRiskClass, codes.InvalidArgument},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requireGRPCCode(t, mapCRMError(tt.err), tt.wantCode)
		})
	}
}

// ---------------------------------------------------------------------------
// Service-not-configured guard (checked once, shared by all handlers)
// ---------------------------------------------------------------------------

func TestAdvisoryProtocol_ServiceNotConfigured(t *testing.T) {
	s := &CRMGRPCServer{}
	ctx := advisoryCtx(uuid.New(), uuid.New())

	_, err := s.CreateAdvisoryProtocol(ctx, &crmv1.CreateAdvisoryProtocolRequest{ContactId: uuid.New().String()})
	requireGRPCCode(t, err, codes.Unavailable)
}

// ---------------------------------------------------------------------------
// CreateAdvisoryProtocol
// ---------------------------------------------------------------------------

func TestCreateAdvisoryProtocol(t *testing.T) {
	t.Run("missing tenant", func(t *testing.T) {
		s, _ := newTestAdvisoryCRMServer()
		ctx := context.WithValue(context.Background(), middleware.UserIDKey, uuid.New().String())
		_, err := s.CreateAdvisoryProtocol(ctx, &crmv1.CreateAdvisoryProtocolRequest{ContactId: uuid.New().String()})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})

	t.Run("missing user", func(t *testing.T) {
		s, _ := newTestAdvisoryCRMServer()
		ctx := context.WithValue(context.Background(), middleware.TenantIDKey, uuid.New().String())
		_, err := s.CreateAdvisoryProtocol(ctx, &crmv1.CreateAdvisoryProtocolRequest{ContactId: uuid.New().String()})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})

	t.Run("invalid contact id", func(t *testing.T) {
		s, _ := newTestAdvisoryCRMServer()
		ctx := advisoryCtx(uuid.New(), uuid.New())
		_, err := s.CreateAdvisoryProtocol(ctx, &crmv1.CreateAdvisoryProtocolRequest{ContactId: "not-a-uuid"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("contact not found in tenant", func(t *testing.T) {
		s, _ := newTestAdvisoryCRMServer()
		ctx := advisoryCtx(uuid.New(), uuid.New())
		_, err := s.CreateAdvisoryProtocol(ctx, &crmv1.CreateAdvisoryProtocolRequest{ContactId: uuid.New().String()})
		requireGRPCCode(t, err, codes.NotFound)
	})

	t.Run("happy path creates a draft", func(t *testing.T) {
		s, repo := newTestAdvisoryCRMServer()
		tenantID := uuid.New()
		contactID := uuid.New()
		repo.contactOK[contactID] = true
		ctx := advisoryCtx(tenantID, uuid.New())

		resp, err := s.CreateAdvisoryProtocol(ctx, &crmv1.CreateAdvisoryProtocolRequest{ContactId: contactID.String()})
		requireGRPCOK(t, err)
		require.NotNil(t, resp.Protocol)
		require.Equal(t, tenantID.String(), resp.Protocol.TenantId)
		require.Equal(t, contactID.String(), resp.Protocol.ContactId)
		require.Equal(t, "draft", resp.Protocol.Status)
		require.Len(t, repo.protocols, 1)
	})
}

// ---------------------------------------------------------------------------
// GetAdvisoryProtocol
// ---------------------------------------------------------------------------

func TestGetAdvisoryProtocol(t *testing.T) {
	t.Run("missing tenant", func(t *testing.T) {
		s, _ := newTestAdvisoryCRMServer()
		_, err := s.GetAdvisoryProtocol(context.Background(), &crmv1.GetAdvisoryProtocolRequest{Id: uuid.New().String()})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})

	t.Run("invalid id", func(t *testing.T) {
		s, _ := newTestAdvisoryCRMServer()
		ctx := advisoryCtx(uuid.New(), uuid.New())
		_, err := s.GetAdvisoryProtocol(ctx, &crmv1.GetAdvisoryProtocolRequest{Id: "not-a-uuid"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("not found", func(t *testing.T) {
		s, _ := newTestAdvisoryCRMServer()
		ctx := advisoryCtx(uuid.New(), uuid.New())
		_, err := s.GetAdvisoryProtocol(ctx, &crmv1.GetAdvisoryProtocolRequest{Id: uuid.New().String()})
		requireGRPCCode(t, err, codes.NotFound)
	})

	t.Run("wrong tenant is treated as not found", func(t *testing.T) {
		s, repo := newTestAdvisoryCRMServer()
		p := seedAdvisoryProtocol(repo, uuid.New(), uuid.New(), "draft")
		ctx := advisoryCtx(uuid.New(), uuid.New())
		_, err := s.GetAdvisoryProtocol(ctx, &crmv1.GetAdvisoryProtocolRequest{Id: p.ID.String()})
		requireGRPCCode(t, err, codes.NotFound)
	})

	t.Run("happy path", func(t *testing.T) {
		s, repo := newTestAdvisoryCRMServer()
		tenantID := uuid.New()
		contactID := uuid.New()
		p := seedAdvisoryProtocol(repo, tenantID, contactID, "draft")
		ctx := advisoryCtx(tenantID, uuid.New())

		resp, err := s.GetAdvisoryProtocol(ctx, &crmv1.GetAdvisoryProtocolRequest{Id: p.ID.String()})
		requireGRPCOK(t, err)
		require.Equal(t, p.ID.String(), resp.Protocol.Id)
		require.Equal(t, contactID.String(), resp.Protocol.ContactId)
	})
}

// ---------------------------------------------------------------------------
// ListAdvisoryProtocols
// ---------------------------------------------------------------------------

func TestListAdvisoryProtocols(t *testing.T) {
	t.Run("missing tenant", func(t *testing.T) {
		s, _ := newTestAdvisoryCRMServer()
		_, err := s.ListAdvisoryProtocols(context.Background(), &crmv1.ListAdvisoryProtocolsRequest{ContactId: uuid.New().String()})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})

	t.Run("invalid contact id", func(t *testing.T) {
		s, _ := newTestAdvisoryCRMServer()
		ctx := advisoryCtx(uuid.New(), uuid.New())
		_, err := s.ListAdvisoryProtocols(ctx, &crmv1.ListAdvisoryProtocolsRequest{ContactId: "not-a-uuid"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("scopes to contact and tenant", func(t *testing.T) {
		s, repo := newTestAdvisoryCRMServer()
		tenantID := uuid.New()
		contactID := uuid.New()
		seedAdvisoryProtocol(repo, tenantID, contactID, "draft")
		seedAdvisoryProtocol(repo, tenantID, contactID, "finalized")
		seedAdvisoryProtocol(repo, tenantID, uuid.New(), "draft")   // other contact
		seedAdvisoryProtocol(repo, uuid.New(), contactID, "draft") // other tenant
		ctx := advisoryCtx(tenantID, uuid.New())

		resp, err := s.ListAdvisoryProtocols(ctx, &crmv1.ListAdvisoryProtocolsRequest{ContactId: contactID.String()})
		requireGRPCOK(t, err)
		require.Len(t, resp.Protocols, 2)
	})

	t.Run("no protocols returns empty list", func(t *testing.T) {
		s, _ := newTestAdvisoryCRMServer()
		ctx := advisoryCtx(uuid.New(), uuid.New())
		resp, err := s.ListAdvisoryProtocols(ctx, &crmv1.ListAdvisoryProtocolsRequest{ContactId: uuid.New().String()})
		requireGRPCOK(t, err)
		require.Empty(t, resp.Protocols)
	})
}

// ---------------------------------------------------------------------------
// UpdateAdvisoryProtocol
// ---------------------------------------------------------------------------

func TestUpdateAdvisoryProtocol(t *testing.T) {
	t.Run("missing tenant", func(t *testing.T) {
		s, _ := newTestAdvisoryCRMServer()
		_, err := s.UpdateAdvisoryProtocol(context.Background(), &crmv1.UpdateAdvisoryProtocolRequest{Id: uuid.New().String()})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})

	t.Run("invalid id", func(t *testing.T) {
		s, _ := newTestAdvisoryCRMServer()
		ctx := advisoryCtx(uuid.New(), uuid.New())
		_, err := s.UpdateAdvisoryProtocol(ctx, &crmv1.UpdateAdvisoryProtocolRequest{Id: "not-a-uuid"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("missing protocol payload", func(t *testing.T) {
		s, repo := newTestAdvisoryCRMServer()
		tenantID := uuid.New()
		p := seedAdvisoryProtocol(repo, tenantID, uuid.New(), "draft")
		ctx := advisoryCtx(tenantID, uuid.New())
		_, err := s.UpdateAdvisoryProtocol(ctx, &crmv1.UpdateAdvisoryProtocolRequest{Id: p.ID.String()})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("not found", func(t *testing.T) {
		s, _ := newTestAdvisoryCRMServer()
		ctx := advisoryCtx(uuid.New(), uuid.New())
		_, err := s.UpdateAdvisoryProtocol(ctx, &crmv1.UpdateAdvisoryProtocolRequest{
			Id:       uuid.New().String(),
			Protocol: &crmv1.AdvisoryProtocolProto{RiskClass: 3},
		})
		requireGRPCCode(t, err, codes.NotFound)
	})

	t.Run("finalized protocol is immutable", func(t *testing.T) {
		s, repo := newTestAdvisoryCRMServer()
		tenantID := uuid.New()
		p := seedAdvisoryProtocol(repo, tenantID, uuid.New(), "finalized")
		ctx := advisoryCtx(tenantID, uuid.New())

		_, err := s.UpdateAdvisoryProtocol(ctx, &crmv1.UpdateAdvisoryProtocolRequest{
			Id:       p.ID.String(),
			Protocol: &crmv1.AdvisoryProtocolProto{RiskClass: 3},
		})
		requireGRPCCode(t, err, codes.FailedPrecondition)
	})

	t.Run("invalid risk class", func(t *testing.T) {
		s, repo := newTestAdvisoryCRMServer()
		tenantID := uuid.New()
		p := seedAdvisoryProtocol(repo, tenantID, uuid.New(), "draft")
		ctx := advisoryCtx(tenantID, uuid.New())

		_, err := s.UpdateAdvisoryProtocol(ctx, &crmv1.UpdateAdvisoryProtocolRequest{
			Id:       p.ID.String(),
			Protocol: &crmv1.AdvisoryProtocolProto{RiskClass: 9},
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("happy path applies fields", func(t *testing.T) {
		s, repo := newTestAdvisoryCRMServer()
		tenantID := uuid.New()
		p := seedAdvisoryProtocol(repo, tenantID, uuid.New(), "draft")
		ctx := advisoryCtx(tenantID, uuid.New())

		resp, err := s.UpdateAdvisoryProtocol(ctx, &crmv1.UpdateAdvisoryProtocolRequest{
			Id: p.ID.String(),
			Protocol: &crmv1.AdvisoryProtocolProto{
				RiskClass: 5,
				Advisor:   "Jane Advisor",
				Location:  "video",
			},
		})
		requireGRPCOK(t, err)
		require.Equal(t, int32(5), resp.Protocol.RiskClass)
		require.Equal(t, "Jane Advisor", resp.Protocol.Advisor)
		require.Equal(t, "video", resp.Protocol.Location)
	})
}

// ---------------------------------------------------------------------------
// DeleteAdvisoryProtocol
// ---------------------------------------------------------------------------

func TestDeleteAdvisoryProtocol(t *testing.T) {
	t.Run("missing tenant", func(t *testing.T) {
		s, _ := newTestAdvisoryCRMServer()
		_, err := s.DeleteAdvisoryProtocol(context.Background(), &crmv1.DeleteAdvisoryProtocolRequest{Id: uuid.New().String()})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})

	t.Run("invalid id", func(t *testing.T) {
		s, _ := newTestAdvisoryCRMServer()
		ctx := advisoryCtx(uuid.New(), uuid.New())
		_, err := s.DeleteAdvisoryProtocol(ctx, &crmv1.DeleteAdvisoryProtocolRequest{Id: "not-a-uuid"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("not found", func(t *testing.T) {
		s, _ := newTestAdvisoryCRMServer()
		ctx := advisoryCtx(uuid.New(), uuid.New())
		_, err := s.DeleteAdvisoryProtocol(ctx, &crmv1.DeleteAdvisoryProtocolRequest{Id: uuid.New().String()})
		requireGRPCCode(t, err, codes.NotFound)
	})

	t.Run("finalized protocol cannot be deleted", func(t *testing.T) {
		s, repo := newTestAdvisoryCRMServer()
		tenantID := uuid.New()
		p := seedAdvisoryProtocol(repo, tenantID, uuid.New(), "finalized")
		ctx := advisoryCtx(tenantID, uuid.New())

		_, err := s.DeleteAdvisoryProtocol(ctx, &crmv1.DeleteAdvisoryProtocolRequest{Id: p.ID.String()})
		requireGRPCCode(t, err, codes.FailedPrecondition)
		require.Contains(t, repo.protocols, p.ID)
	})

	t.Run("happy path removes draft", func(t *testing.T) {
		s, repo := newTestAdvisoryCRMServer()
		tenantID := uuid.New()
		p := seedAdvisoryProtocol(repo, tenantID, uuid.New(), "draft")
		ctx := advisoryCtx(tenantID, uuid.New())

		resp, err := s.DeleteAdvisoryProtocol(ctx, &crmv1.DeleteAdvisoryProtocolRequest{Id: p.ID.String()})
		requireGRPCOK(t, err)
		require.True(t, resp.Ok)
		require.NotContains(t, repo.protocols, p.ID)
	})
}

// ---------------------------------------------------------------------------
// HandOverAdvisoryProtocol
// ---------------------------------------------------------------------------

func TestHandOverAdvisoryProtocol(t *testing.T) {
	t.Run("missing tenant", func(t *testing.T) {
		s, _ := newTestAdvisoryCRMServer()
		_, err := s.HandOverAdvisoryProtocol(context.Background(), &crmv1.HandOverAdvisoryProtocolRequest{Id: uuid.New().String()})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})

	t.Run("invalid id", func(t *testing.T) {
		s, _ := newTestAdvisoryCRMServer()
		ctx := advisoryCtx(uuid.New(), uuid.New())
		_, err := s.HandOverAdvisoryProtocol(ctx, &crmv1.HandOverAdvisoryProtocolRequest{Id: "not-a-uuid"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("not found", func(t *testing.T) {
		s, _ := newTestAdvisoryCRMServer()
		ctx := advisoryCtx(uuid.New(), uuid.New())
		_, err := s.HandOverAdvisoryProtocol(ctx, &crmv1.HandOverAdvisoryProtocolRequest{Id: uuid.New().String()})
		requireGRPCCode(t, err, codes.NotFound)
	})

	t.Run("already finalized is idempotent, not an error", func(t *testing.T) {
		s, repo := newTestAdvisoryCRMServer()
		tenantID := uuid.New()
		p := seedAdvisoryProtocol(repo, tenantID, uuid.New(), "finalized")
		ctx := advisoryCtx(tenantID, uuid.New())

		resp, err := s.HandOverAdvisoryProtocol(ctx, &crmv1.HandOverAdvisoryProtocolRequest{Id: p.ID.String()})
		requireGRPCOK(t, err)
		require.Equal(t, "finalized", resp.Protocol.Status)
	})

	t.Run("happy path finalizes a draft", func(t *testing.T) {
		s, repo := newTestAdvisoryCRMServer()
		tenantID := uuid.New()
		p := seedAdvisoryProtocol(repo, tenantID, uuid.New(), "draft")
		ctx := advisoryCtx(tenantID, uuid.New())

		resp, err := s.HandOverAdvisoryProtocol(ctx, &crmv1.HandOverAdvisoryProtocolRequest{Id: p.ID.String()})
		requireGRPCOK(t, err)
		require.Equal(t, "finalized", resp.Protocol.Status)
		require.NotNil(t, resp.Protocol.HandedOverAt)
		require.Equal(t, "finalized", repo.protocols[p.ID].Status)
	})
}

// ---------------------------------------------------------------------------
// GenerateAdvisoryProtocolPDF
// ---------------------------------------------------------------------------

func TestGenerateAdvisoryProtocolPDF(t *testing.T) {
	t.Run("missing tenant", func(t *testing.T) {
		s, _ := newTestAdvisoryCRMServer()
		_, err := s.GenerateAdvisoryProtocolPDF(context.Background(), &crmv1.GenerateAdvisoryProtocolPDFRequest{Id: uuid.New().String()})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})

	t.Run("invalid id", func(t *testing.T) {
		s, _ := newTestAdvisoryCRMServer()
		ctx := advisoryCtx(uuid.New(), uuid.New())
		_, err := s.GenerateAdvisoryProtocolPDF(ctx, &crmv1.GenerateAdvisoryProtocolPDFRequest{Id: "not-a-uuid"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})

	t.Run("not found", func(t *testing.T) {
		s, _ := newTestAdvisoryCRMServer()
		ctx := advisoryCtx(uuid.New(), uuid.New())
		_, err := s.GenerateAdvisoryProtocolPDF(ctx, &crmv1.GenerateAdvisoryProtocolPDFRequest{Id: uuid.New().String()})
		requireGRPCCode(t, err, codes.NotFound)
	})

	t.Run("happy path renders bytes without a wired contact service", func(t *testing.T) {
		s, repo := newTestAdvisoryCRMServer()
		tenantID := uuid.New()
		p := seedAdvisoryProtocol(repo, tenantID, uuid.New(), "finalized")
		ctx := advisoryCtx(tenantID, uuid.New())

		resp, err := s.GenerateAdvisoryProtocolPDF(ctx, &crmv1.GenerateAdvisoryProtocolPDFRequest{Id: p.ID.String()})
		requireGRPCOK(t, err)
		require.NotEmpty(t, resp.PdfData)
		require.True(t, strings.HasPrefix(resp.Filename, "Beratungsprotokoll_"))
	})
}

// ---------------------------------------------------------------------------
// GetReferralReport
// ---------------------------------------------------------------------------

func TestGetReferralReport(t *testing.T) {
	t.Run("missing tenant", func(t *testing.T) {
		s, _ := newTestAdvisoryCRMServer()
		_, err := s.GetReferralReport(context.Background(), &crmv1.GetReferralReportRequest{})
		requireGRPCCode(t, err, codes.Unauthenticated)
	})

	t.Run("happy path maps entries", func(t *testing.T) {
		s, repo := newTestAdvisoryCRMServer()
		referrerID := uuid.New()
		repo.referral = []*advisoryprotocol.ReferralEntry{
			{ReferrerID: referrerID, ReferrerFirstName: "Anna", ReferrerLastName: "Berater", ReferredCount: 4},
		}
		ctx := advisoryCtx(uuid.New(), uuid.New())

		resp, err := s.GetReferralReport(ctx, &crmv1.GetReferralReportRequest{})
		requireGRPCOK(t, err)
		require.Len(t, resp.Entries, 1)
		require.Equal(t, referrerID.String(), resp.Entries[0].ReferrerId)
		require.Equal(t, "Anna", resp.Entries[0].ReferrerFirstName)
		require.Equal(t, "Berater", resp.Entries[0].ReferrerLastName)
		require.Equal(t, int32(4), resp.Entries[0].ReferredCount)
	})

	t.Run("no entries returns empty list", func(t *testing.T) {
		s, _ := newTestAdvisoryCRMServer()
		ctx := advisoryCtx(uuid.New(), uuid.New())
		resp, err := s.GetReferralReport(ctx, &crmv1.GetReferralReportRequest{})
		requireGRPCOK(t, err)
		require.Empty(t, resp.Entries)
	})
}
