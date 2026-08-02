package vendoraccess_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/security/vendoraccess"
)

// fakeRepo is an in-memory Repository implementation for guardrail tests
// that don't need a real database (no RLS or SQL to exercise here -- the
// service layer's own state-machine and sensitive-scope logic).
type fakeRepo struct {
	byTenant map[uuid.UUID]map[uuid.UUID]*models.VendorAccessRequest
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{byTenant: make(map[uuid.UUID]map[uuid.UUID]*models.VendorAccessRequest)}
}

func (f *fakeRepo) CreateRequest(_ context.Context, req *models.VendorAccessRequest) error {
	if f.byTenant[req.TenantID] == nil {
		f.byTenant[req.TenantID] = make(map[uuid.UUID]*models.VendorAccessRequest)
	}
	clone := *req
	f.byTenant[req.TenantID][req.ID] = &clone
	return nil
}

func (f *fakeRepo) GetRequest(_ context.Context, tenantID, id uuid.UUID) (*models.VendorAccessRequest, error) {
	tenantRows, ok := f.byTenant[tenantID]
	if !ok {
		return nil, vendoraccess.ErrRequestNotFound
	}
	req, ok := tenantRows[id]
	if !ok {
		return nil, vendoraccess.ErrRequestNotFound
	}
	clone := *req
	return &clone, nil
}

func (f *fakeRepo) ListRequests(_ context.Context, tenantID uuid.UUID) ([]*models.VendorAccessRequest, error) {
	var out []*models.VendorAccessRequest
	for _, req := range f.byTenant[tenantID] {
		clone := *req
		out = append(out, &clone)
	}
	return out, nil
}

func (f *fakeRepo) UpdateStatus(_ context.Context, tenantID uuid.UUID, req *models.VendorAccessRequest) error {
	tenantRows, ok := f.byTenant[tenantID]
	if !ok {
		return vendoraccess.ErrRequestNotFound
	}
	if _, ok := tenantRows[req.ID]; !ok {
		return vendoraccess.ErrRequestNotFound
	}
	clone := *req
	tenantRows[req.ID] = &clone
	return nil
}

func testCtx(tenantID uuid.UUID) context.Context {
	return context.WithValue(context.Background(), middleware.TenantIDKey, tenantID.String())
}

func seedRequest(t *testing.T, repo *fakeRepo, tenantID uuid.UUID, status string, scope []string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	req := &models.VendorAccessRequest{
		ID:             id,
		TenantID:       tenantID,
		Reason:         "test",
		Description:    "test description",
		Agents:         []models.VendorAccessAgent{{Name: "Agent A"}},
		Scope:          scope,
		RequestedStart: time.Now().UTC(),
		DurationDays:   7,
		ExpiresAt:      time.Now().UTC().Add(7 * 24 * time.Hour),
		Status:         status,
		CreatedAt:      time.Now().UTC(),
	}
	if err := repo.CreateRequest(context.Background(), req); err != nil {
		t.Fatalf("seed request: %v", err)
	}
	return id
}

func TestApprove_NonSensitiveScope_Succeeds(t *testing.T) {
	repo := newFakeRepo()
	svc := vendoraccess.NewService(repo)
	tenantID := uuid.New()
	id := seedRequest(t, repo, tenantID, models.VendorAccessStatusPending, []string{"crm"})

	actorID := uuid.New()
	updated, err := svc.Approve(testCtx(tenantID), id, actorID, false)
	if err != nil {
		t.Fatalf("Approve: %v", err)
	}
	if updated.Status != models.VendorAccessStatusActive {
		t.Fatalf("status = %q, want active", updated.Status)
	}
	if updated.ApprovedBy == nil || *updated.ApprovedBy != actorID {
		t.Fatalf("approved_by = %v, want %v", updated.ApprovedBy, actorID)
	}
}

func TestApprove_SensitiveScopeWithoutAck_Fails422(t *testing.T) {
	repo := newFakeRepo()
	svc := vendoraccess.NewService(repo)
	tenantID := uuid.New()
	id := seedRequest(t, repo, tenantID, models.VendorAccessStatusPending, []string{"crm", "salary"})

	_, err := svc.Approve(testCtx(tenantID), id, uuid.New(), false)
	if !errors.Is(err, vendoraccess.ErrSensitiveAckRequired) {
		t.Fatalf("err = %v, want ErrSensitiveAckRequired", err)
	}
}

func TestApprove_SensitiveScopeWithAck_Succeeds(t *testing.T) {
	repo := newFakeRepo()
	svc := vendoraccess.NewService(repo)
	tenantID := uuid.New()
	id := seedRequest(t, repo, tenantID, models.VendorAccessStatusPending, []string{"hr_data"})

	updated, err := svc.Approve(testCtx(tenantID), id, uuid.New(), true)
	if err != nil {
		t.Fatalf("Approve: %v", err)
	}
	if updated.SensitiveAck == nil || !*updated.SensitiveAck {
		t.Fatalf("sensitive_ack not persisted as true")
	}
}

func TestApprove_FromCounterProposed_Succeeds(t *testing.T) {
	repo := newFakeRepo()
	svc := vendoraccess.NewService(repo)
	tenantID := uuid.New()
	id := seedRequest(t, repo, tenantID, models.VendorAccessStatusCounterProposed, []string{"crm"})

	updated, err := svc.Approve(testCtx(tenantID), id, uuid.New(), false)
	if err != nil {
		t.Fatalf("Approve: %v", err)
	}
	if updated.Status != models.VendorAccessStatusActive {
		t.Fatalf("status = %q, want active", updated.Status)
	}
}

func TestApprove_AlreadyActive_FailsInvalidStatus(t *testing.T) {
	repo := newFakeRepo()
	svc := vendoraccess.NewService(repo)
	tenantID := uuid.New()
	id := seedRequest(t, repo, tenantID, models.VendorAccessStatusActive, []string{"crm"})

	_, err := svc.Approve(testCtx(tenantID), id, uuid.New(), false)
	if !errors.Is(err, vendoraccess.ErrInvalidStatus) {
		t.Fatalf("err = %v, want ErrInvalidStatus", err)
	}
}

func TestDecline_Pending_Succeeds(t *testing.T) {
	repo := newFakeRepo()
	svc := vendoraccess.NewService(repo)
	tenantID := uuid.New()
	id := seedRequest(t, repo, tenantID, models.VendorAccessStatusPending, []string{"crm"})

	updated, err := svc.Decline(testCtx(tenantID), id)
	if err != nil {
		t.Fatalf("Decline: %v", err)
	}
	if updated.Status != models.VendorAccessStatusDeclined {
		t.Fatalf("status = %q, want declined", updated.Status)
	}
}

func TestDecline_Active_FailsInvalidStatus(t *testing.T) {
	repo := newFakeRepo()
	svc := vendoraccess.NewService(repo)
	tenantID := uuid.New()
	id := seedRequest(t, repo, tenantID, models.VendorAccessStatusActive, []string{"crm"})

	_, err := svc.Decline(testCtx(tenantID), id)
	if !errors.Is(err, vendoraccess.ErrInvalidStatus) {
		t.Fatalf("err = %v, want ErrInvalidStatus", err)
	}
}

func TestCounterPropose_Pending_Succeeds(t *testing.T) {
	repo := newFakeRepo()
	svc := vendoraccess.NewService(repo)
	tenantID := uuid.New()
	id := seedRequest(t, repo, tenantID, models.VendorAccessStatusPending, []string{"crm"})

	proposed := time.Now().UTC().Add(48 * time.Hour)
	updated, err := svc.CounterPropose(testCtx(tenantID), id, proposed)
	if err != nil {
		t.Fatalf("CounterPropose: %v", err)
	}
	if updated.Status != models.VendorAccessStatusCounterProposed {
		t.Fatalf("status = %q, want counter_proposed", updated.Status)
	}
	if updated.CounterProposedStart == nil || !updated.CounterProposedStart.Equal(proposed) {
		t.Fatalf("counter_proposed_start = %v, want %v", updated.CounterProposedStart, proposed)
	}
}

// CounterPropose is only valid from pending, not from an already
// counter-proposed request -- the frontend allows exactly one round trip.
func TestCounterPropose_AlreadyCounterProposed_FailsInvalidStatus(t *testing.T) {
	repo := newFakeRepo()
	svc := vendoraccess.NewService(repo)
	tenantID := uuid.New()
	id := seedRequest(t, repo, tenantID, models.VendorAccessStatusCounterProposed, []string{"crm"})

	_, err := svc.CounterPropose(testCtx(tenantID), id, time.Now().UTC())
	if !errors.Is(err, vendoraccess.ErrInvalidStatus) {
		t.Fatalf("err = %v, want ErrInvalidStatus", err)
	}
}

func TestRevoke_Active_Succeeds(t *testing.T) {
	repo := newFakeRepo()
	svc := vendoraccess.NewService(repo)
	tenantID := uuid.New()
	id := seedRequest(t, repo, tenantID, models.VendorAccessStatusActive, []string{"crm"})

	actorID := uuid.New()
	updated, err := svc.Revoke(testCtx(tenantID), id, actorID)
	if err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	if updated.Status != models.VendorAccessStatusRevoked {
		t.Fatalf("status = %q, want revoked", updated.Status)
	}
	if updated.RevokedBy == nil || *updated.RevokedBy != actorID {
		t.Fatalf("revoked_by = %v, want %v", updated.RevokedBy, actorID)
	}
}

func TestRevoke_Pending_FailsInvalidStatus(t *testing.T) {
	repo := newFakeRepo()
	svc := vendoraccess.NewService(repo)
	tenantID := uuid.New()
	id := seedRequest(t, repo, tenantID, models.VendorAccessStatusPending, []string{"crm"})

	_, err := svc.Revoke(testCtx(tenantID), id, uuid.New())
	if !errors.Is(err, vendoraccess.ErrInvalidStatus) {
		t.Fatalf("err = %v, want ErrInvalidStatus", err)
	}
}

// A request that lives in a different tenant must not resolve at all --
// GetRequest is always called with the caller's own tenantID, so a
// cross-tenant id looks identical to a nonexistent one.
func TestApprove_CrossTenantRequest_NotFound(t *testing.T) {
	repo := newFakeRepo()
	svc := vendoraccess.NewService(repo)
	ownerTenant := uuid.New()
	otherTenant := uuid.New()
	id := seedRequest(t, repo, ownerTenant, models.VendorAccessStatusPending, []string{"crm"})

	_, err := svc.Approve(testCtx(otherTenant), id, uuid.New(), false)
	if !errors.Is(err, vendoraccess.ErrRequestNotFound) {
		t.Fatalf("err = %v, want ErrRequestNotFound", err)
	}
}
