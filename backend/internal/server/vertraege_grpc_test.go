package server

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/kmuhub/kmuhub/internal/vertraege"
	vertraegev1 "github.com/kmuhub/kmuhub/proto/vertraege/v1"
)

// ============================================================================
// Stub repository (implements vertraege.Repository)
// ============================================================================

type stubVertraegeRepo struct {
	mu sync.Mutex

	contracts map[uuid.UUID]*vertraege.Contract
	parties   map[uuid.UUID]*vertraege.ContractParty
	reminders map[uuid.UUID]*vertraege.ContractReminder
	events    []*vertraege.ContractEvent

	saveSignatureErr error
}

func newStubVertraegeRepo() *stubVertraegeRepo {
	return &stubVertraegeRepo{
		contracts: make(map[uuid.UUID]*vertraege.Contract),
		parties:   make(map[uuid.UUID]*vertraege.ContractParty),
		reminders: make(map[uuid.UUID]*vertraege.ContractReminder),
	}
}

// --- Contracts ---

func (r *stubVertraegeRepo) CreateContract(_ context.Context, c *vertraege.Contract) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *c
	r.contracts[c.ID] = &cp
	return nil
}

func (r *stubVertraegeRepo) UpdateContract(_ context.Context, c *vertraege.Contract) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	existing, ok := r.contracts[c.ID]
	if !ok || existing.TenantID != c.TenantID {
		return vertraege.ErrContractNotFound
	}
	cp := *c
	r.contracts[c.ID] = &cp
	return nil
}

func (r *stubVertraegeRepo) GetContract(_ context.Context, tenantID, contractID uuid.UUID) (*vertraege.Contract, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.contracts[contractID]
	if !ok || c.TenantID != tenantID {
		return nil, vertraege.ErrContractNotFound
	}
	cp := *c
	for _, p := range r.parties {
		if p.ContractID == contractID {
			cp.Parties = append(cp.Parties, p)
		}
	}
	for _, rem := range r.reminders {
		if rem.ContractID == contractID {
			cp.Reminders = append(cp.Reminders, rem)
		}
	}
	return &cp, nil
}

func (r *stubVertraegeRepo) ListContracts(_ context.Context, tenantID uuid.UUID, filter vertraege.ListContractsFilter, offset, limit int) ([]*vertraege.Contract, int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var matched []*vertraege.Contract
	for _, c := range r.contracts {
		if c.TenantID != tenantID {
			continue
		}
		if filter.Status != nil && c.Status != *filter.Status {
			continue
		}
		if filter.Type != nil && c.ContractType != *filter.Type {
			continue
		}
		if filter.StartsAfter != nil && !c.StartsOn.After(*filter.StartsAfter) {
			continue
		}
		if filter.StartsBefore != nil && !c.StartsOn.Before(*filter.StartsBefore) {
			continue
		}
		if filter.ContactID != nil {
			hasContact := false
			for _, p := range r.parties {
				if p.ContractID == c.ID && p.ContactID != nil && *p.ContactID == *filter.ContactID {
					hasContact = true
					break
				}
			}
			if !hasContact {
				continue
			}
		}
		matched = append(matched, c)
	}
	total := len(matched)
	if offset >= total {
		return []*vertraege.Contract{}, total, nil
	}
	end := min(offset+limit, total)
	return matched[offset:end], total, nil
}

func (r *stubVertraegeRepo) DeleteContract(_ context.Context, tenantID, contractID uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.contracts[contractID]
	if !ok || c.TenantID != tenantID {
		return vertraege.ErrContractNotFound
	}
	delete(r.contracts, contractID)
	return nil
}

func (r *stubVertraegeRepo) ContractNumberExists(_ context.Context, tenantID uuid.UUID, number string, excludeID *uuid.UUID) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, c := range r.contracts {
		if c.TenantID != tenantID || c.ContractNumber != number {
			continue
		}
		if excludeID != nil && c.ID == *excludeID {
			continue
		}
		return true, nil
	}
	return false, nil
}

func (r *stubVertraegeRepo) SaveSignature(_ context.Context, tenantID, contractID, signatureData, signedBy string) (*vertraege.Contract, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.saveSignatureErr != nil {
		return nil, r.saveSignatureErr
	}
	tid, err := uuid.Parse(tenantID)
	if err != nil {
		return nil, err
	}
	cid, err := uuid.Parse(contractID)
	if err != nil {
		return nil, err
	}
	c, ok := r.contracts[cid]
	if !ok || c.TenantID != tid {
		return nil, vertraege.ErrContractNotFound
	}
	now := time.Now()
	c.SignatureData = &signatureData
	c.SignedAt = &now
	c.SignedBy = &signedBy
	cp := *c
	return &cp, nil
}

func (r *stubVertraegeRepo) ExpireContracts(_ context.Context) (int64, error) {
	return 0, nil
}

// --- Parties ---

func (r *stubVertraegeRepo) AddParty(_ context.Context, p *vertraege.ContractParty) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *p
	r.parties[p.ID] = &cp
	return nil
}

func (r *stubVertraegeRepo) RemoveParty(_ context.Context, tenantID, partyID uuid.UUID) (uuid.UUID, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.parties[partyID]
	if !ok || p.TenantID != tenantID {
		return uuid.Nil, nil
	}
	delete(r.parties, partyID)
	return p.ContractID, nil
}

func (r *stubVertraegeRepo) ListParties(_ context.Context, tenantID, contractID uuid.UUID) ([]*vertraege.ContractParty, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*vertraege.ContractParty
	for _, p := range r.parties {
		if p.TenantID == tenantID && p.ContractID == contractID {
			out = append(out, p)
		}
	}
	return out, nil
}

// --- Events ---

func (r *stubVertraegeRepo) CreateContractEvent(_ context.Context, e *vertraege.ContractEvent) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, e)
	return nil
}

func (r *stubVertraegeRepo) ListContractEvents(_ context.Context, tenantID, contractID uuid.UUID, offset, limit int) ([]*vertraege.ContractEvent, int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var matched []*vertraege.ContractEvent
	for _, e := range r.events {
		if e.TenantID == tenantID && e.ContractID == contractID {
			matched = append(matched, e)
		}
	}
	total := len(matched)
	if offset >= total {
		return []*vertraege.ContractEvent{}, total, nil
	}
	end := min(offset+limit, total)
	return matched[offset:end], total, nil
}

// --- Reminders ---

func (r *stubVertraegeRepo) CreateReminder(_ context.Context, rem *vertraege.ContractReminder) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *rem
	r.reminders[rem.ID] = &cp
	return nil
}

func (r *stubVertraegeRepo) UpdateReminder(_ context.Context, rem *vertraege.ContractReminder) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	existing, ok := r.reminders[rem.ID]
	if !ok || existing.TenantID != rem.TenantID {
		return vertraege.ErrReminderNotFound
	}
	cp := *rem
	r.reminders[rem.ID] = &cp
	return nil
}

func (r *stubVertraegeRepo) GetReminder(_ context.Context, tenantID, reminderID uuid.UUID) (*vertraege.ContractReminder, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	rem, ok := r.reminders[reminderID]
	if !ok || rem.TenantID != tenantID {
		return nil, vertraege.ErrReminderNotFound
	}
	cp := *rem
	return &cp, nil
}

func (r *stubVertraegeRepo) DeleteReminder(_ context.Context, tenantID, reminderID uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	rem, ok := r.reminders[reminderID]
	if !ok || rem.TenantID != tenantID {
		return vertraege.ErrReminderNotFound
	}
	delete(r.reminders, reminderID)
	return nil
}

func (r *stubVertraegeRepo) ListReminders(_ context.Context, tenantID, contractID uuid.UUID, onlyPending bool) ([]*vertraege.ContractReminder, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []*vertraege.ContractReminder
	for _, rem := range r.reminders {
		if rem.TenantID != tenantID || rem.ContractID != contractID {
			continue
		}
		if onlyPending && rem.Status != vertraege.ReminderStatusPending {
			continue
		}
		out = append(out, rem)
	}
	return out, nil
}

// --- Worker (unused by the gRPC-layer tests, present only to satisfy the interface) ---

func (r *stubVertraegeRepo) ClaimDueReminders(_ context.Context) ([]*vertraege.ContractReminder, error) {
	return nil, nil
}

func (r *stubVertraegeRepo) MarkReminderSent(_ context.Context, _ uuid.UUID, _ time.Time) error {
	return nil
}

// ============================================================================
// Test server helpers
// ============================================================================

func newTestVertraegeServer() *VertraegeGRPCServer {
	return NewVertraegeGRPCServer(nil)
}

func newVertraegeServerWithRepo(repo vertraege.Repository) (*VertraegeGRPCServer, *vertraege.Service) {
	svc := vertraege.NewService(repo)
	return NewVertraegeGRPCServer(svc), svc
}

func seedVertraegeContract(repo *stubVertraegeRepo, tenantID uuid.UUID, mutate func(*vertraege.Contract)) *vertraege.Contract {
	c := &vertraege.Contract{
		ID:             uuid.New(),
		TenantID:       tenantID,
		ContractNumber: "V-" + uuid.NewString()[:8],
		Title:          "Test Contract",
		ContractType:   vertraege.ContractTypeService,
		Status:         vertraege.ContractStatusDraft,
		StartsOn:       time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	if mutate != nil {
		mutate(c)
	}
	repo.contracts[c.ID] = c
	return c
}

// ============================================================================
// Validation cluster — every RPC rejects a malformed UUID before touching the
// (nil) service.
// ============================================================================

func TestVertraege_ValidationErrors(t *testing.T) {
	srv := newTestVertraegeServer()
	ctx := context.Background()
	validTenant := uuid.New().String()

	t.Run("CreateContract invalid tenant_id", func(t *testing.T) {
		_, err := srv.CreateContract(ctx, &vertraegev1.CreateContractRequest{TenantId: "not-a-uuid"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("CreateContract invalid created_by", func(t *testing.T) {
		badID := "not-a-uuid"
		_, err := srv.CreateContract(ctx, &vertraegev1.CreateContractRequest{
			TenantId:  validTenant,
			CreatedBy: &badID,
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("UpdateContract invalid tenant_id", func(t *testing.T) {
		_, err := srv.UpdateContract(ctx, &vertraegev1.UpdateContractRequest{TenantId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("UpdateContract invalid contract_id", func(t *testing.T) {
		_, err := srv.UpdateContract(ctx, &vertraegev1.UpdateContractRequest{TenantId: validTenant, ContractId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("DeleteContract invalid tenant_id", func(t *testing.T) {
		_, err := srv.DeleteContract(ctx, &vertraegev1.DeleteContractRequest{TenantId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("DeleteContract invalid contract_id", func(t *testing.T) {
		_, err := srv.DeleteContract(ctx, &vertraegev1.DeleteContractRequest{TenantId: validTenant, ContractId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("GetContract invalid tenant_id", func(t *testing.T) {
		_, err := srv.GetContract(ctx, &vertraegev1.GetContractRequest{TenantId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("GetContract invalid contract_id", func(t *testing.T) {
		_, err := srv.GetContract(ctx, &vertraegev1.GetContractRequest{TenantId: validTenant, ContractId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("ListContracts invalid tenant_id", func(t *testing.T) {
		_, err := srv.ListContracts(ctx, &vertraegev1.ListContractsRequest{TenantId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("ListContracts invalid contact_id", func(t *testing.T) {
		badID := "bad"
		_, err := srv.ListContracts(ctx, &vertraegev1.ListContractsRequest{TenantId: validTenant, ContactId: &badID})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("ExportContract invalid tenant_id", func(t *testing.T) {
		_, err := srv.ExportContract(ctx, &vertraegev1.ExportContractRequest{TenantId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("ExportContract invalid contract_id", func(t *testing.T) {
		_, err := srv.ExportContract(ctx, &vertraegev1.ExportContractRequest{TenantId: validTenant, ContractId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("SaveSignature invalid tenant_id", func(t *testing.T) {
		_, err := srv.SaveSignature(ctx, &vertraegev1.SaveContractSignatureRequest{TenantId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("SaveSignature invalid contract_id", func(t *testing.T) {
		_, err := srv.SaveSignature(ctx, &vertraegev1.SaveContractSignatureRequest{TenantId: validTenant, ContractId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("AddParty invalid tenant_id", func(t *testing.T) {
		_, err := srv.AddParty(ctx, &vertraegev1.AddPartyRequest{TenantId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("AddParty invalid contract_id", func(t *testing.T) {
		_, err := srv.AddParty(ctx, &vertraegev1.AddPartyRequest{TenantId: validTenant, ContractId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("AddParty invalid contact_id", func(t *testing.T) {
		badID := "bad"
		_, err := srv.AddParty(ctx, &vertraegev1.AddPartyRequest{
			TenantId:   validTenant,
			ContractId: uuid.New().String(),
			ContactId:  &badID,
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("AddParty invalid company_id", func(t *testing.T) {
		badID := "bad"
		_, err := srv.AddParty(ctx, &vertraegev1.AddPartyRequest{
			TenantId:   validTenant,
			ContractId: uuid.New().String(),
			CompanyId:  &badID,
		})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("RemoveParty invalid tenant_id", func(t *testing.T) {
		_, err := srv.RemoveParty(ctx, &vertraegev1.RemovePartyRequest{TenantId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("RemoveParty invalid party_id", func(t *testing.T) {
		_, err := srv.RemoveParty(ctx, &vertraegev1.RemovePartyRequest{TenantId: validTenant, PartyId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("ListParties invalid tenant_id", func(t *testing.T) {
		_, err := srv.ListParties(ctx, &vertraegev1.ListPartiesRequest{TenantId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("ListParties invalid contract_id", func(t *testing.T) {
		_, err := srv.ListParties(ctx, &vertraegev1.ListPartiesRequest{TenantId: validTenant, ContractId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("CreateReminder invalid tenant_id", func(t *testing.T) {
		_, err := srv.CreateReminder(ctx, &vertraegev1.CreateReminderRequest{TenantId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("CreateReminder invalid contract_id", func(t *testing.T) {
		_, err := srv.CreateReminder(ctx, &vertraegev1.CreateReminderRequest{TenantId: validTenant, ContractId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("UpdateReminder invalid tenant_id", func(t *testing.T) {
		_, err := srv.UpdateReminder(ctx, &vertraegev1.UpdateReminderRequest{TenantId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("UpdateReminder invalid reminder_id", func(t *testing.T) {
		_, err := srv.UpdateReminder(ctx, &vertraegev1.UpdateReminderRequest{TenantId: validTenant, ReminderId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("DeleteReminder invalid tenant_id", func(t *testing.T) {
		_, err := srv.DeleteReminder(ctx, &vertraegev1.DeleteReminderRequest{TenantId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("DeleteReminder invalid reminder_id", func(t *testing.T) {
		_, err := srv.DeleteReminder(ctx, &vertraegev1.DeleteReminderRequest{TenantId: validTenant, ReminderId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("ListReminders invalid tenant_id", func(t *testing.T) {
		_, err := srv.ListReminders(ctx, &vertraegev1.ListRemindersRequest{TenantId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("ListReminders invalid contract_id", func(t *testing.T) {
		_, err := srv.ListReminders(ctx, &vertraegev1.ListRemindersRequest{TenantId: validTenant, ContractId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("ListContractEvents invalid tenant_id", func(t *testing.T) {
		_, err := srv.ListContractEvents(ctx, &vertraegev1.ListContractEventsRequest{TenantId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("ListContractEvents invalid contract_id", func(t *testing.T) {
		_, err := srv.ListContractEvents(ctx, &vertraegev1.ListContractEventsRequest{TenantId: validTenant, ContractId: "bad"})
		requireGRPCCode(t, err, codes.InvalidArgument)
	})
	t.Run("UploadDocument is unimplemented", func(t *testing.T) {
		//nolint:staticcheck // SA1019: intentionally exercising the deprecated stub
		_, err := srv.UploadDocument(ctx, &vertraegev1.UploadDocumentRequest{})
		requireGRPCCode(t, err, codes.Unimplemented)
	})
}

// ============================================================================
// Contract lifecycle — happy path + error mapping
// ============================================================================

func TestVertraege_CreateContract(t *testing.T) {
	repo := newStubVertraegeRepo()
	srv, _ := newVertraegeServerWithRepo(repo)
	tenantID := uuid.New()
	ctx := context.Background()

	req := &vertraegev1.CreateContractRequest{
		TenantId:       tenantID.String(),
		ContractNumber: "V-2026-001",
		Title:          "Mietvertrag Buero",
		ContractType:   string(vertraege.ContractTypeRental),
		StartsOn:       timestamppb.New(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)),
	}
	resp, err := srv.CreateContract(ctx, req)
	require.NoError(t, err)
	require.NotNil(t, resp.Contract)
	assert.Equal(t, "V-2026-001", resp.Contract.ContractNumber)
	assert.Equal(t, "draft", resp.Contract.Status)

	// Duplicate contract number in the same tenant -> AlreadyExists.
	_, err = srv.CreateContract(ctx, req)
	requireGRPCCode(t, err, codes.AlreadyExists)

	// Empty title -> InvalidArgument (service-level validation, not a UUID parse).
	_, err = srv.CreateContract(ctx, &vertraegev1.CreateContractRequest{
		TenantId:       tenantID.String(),
		ContractNumber: "V-2026-002",
		ContractType:   string(vertraege.ContractTypeRental),
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestVertraege_UpdateContract_NotFound(t *testing.T) {
	repo := newStubVertraegeRepo()
	srv, _ := newVertraegeServerWithRepo(repo)
	_, err := srv.UpdateContract(context.Background(), &vertraegev1.UpdateContractRequest{
		TenantId:   uuid.New().String(),
		ContractId: uuid.New().String(),
	})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestVertraege_UpdateContract_ClearEndsOn(t *testing.T) {
	repo := newStubVertraegeRepo()
	srv, _ := newVertraegeServerWithRepo(repo)
	tenantID := uuid.New()
	endsOn := time.Date(2026, 12, 31, 23, 59, 59, 0, time.UTC)
	c := seedVertraegeContract(repo, tenantID, func(c *vertraege.Contract) {
		c.EndsOn = &endsOn
	})

	resp, err := srv.UpdateContract(context.Background(), &vertraegev1.UpdateContractRequest{
		TenantId:    tenantID.String(),
		ContractId:  c.ID.String(),
		ClearEndsOn: true,
		EndsOn:      timestamppb.New(time.Date(2027, 6, 1, 0, 0, 0, 0, time.UTC)), // must be ignored when ClearEndsOn is set
	})
	require.NoError(t, err)
	assert.Nil(t, resp.Contract.EndsOn)
}

func TestVertraege_DeleteContract(t *testing.T) {
	repo := newStubVertraegeRepo()
	srv, _ := newVertraegeServerWithRepo(repo)
	tenantID := uuid.New()

	draft := seedVertraegeContract(repo, tenantID, nil)
	_, err := srv.DeleteContract(context.Background(), &vertraegev1.DeleteContractRequest{
		TenantId:   tenantID.String(),
		ContractId: draft.ID.String(),
	})
	require.NoError(t, err)

	active := seedVertraegeContract(repo, tenantID, func(c *vertraege.Contract) {
		c.Status = vertraege.ContractStatusActive
	})
	_, err = srv.DeleteContract(context.Background(), &vertraegev1.DeleteContractRequest{
		TenantId:   tenantID.String(),
		ContractId: active.ID.String(),
	})
	requireGRPCCode(t, err, codes.FailedPrecondition)

	_, err = srv.DeleteContract(context.Background(), &vertraegev1.DeleteContractRequest{
		TenantId:   tenantID.String(),
		ContractId: uuid.New().String(),
	})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestVertraege_GetContract_CrossTenant(t *testing.T) {
	repo := newStubVertraegeRepo()
	srv, _ := newVertraegeServerWithRepo(repo)
	tenantID := uuid.New()
	c := seedVertraegeContract(repo, tenantID, nil)

	_, err := srv.GetContract(context.Background(), &vertraegev1.GetContractRequest{
		TenantId:   uuid.New().String(), // different tenant
		ContractId: c.ID.String(),
	})
	requireGRPCCode(t, err, codes.NotFound)
}

// ============================================================================
// Date boundary cases — month-end and year-boundary values must round-trip
// through timestamppb without truncation or off-by-one drift.
// ============================================================================

func TestVertraege_ListContracts_DateBoundaries(t *testing.T) {
	repo := newStubVertraegeRepo()
	srv, _ := newVertraegeServerWithRepo(repo)
	tenantID := uuid.New()

	// One contract starting exactly at the last instant of a year, one at the
	// first instant of the next — the classic month/year-boundary pair.
	oldYearEnd := seedVertraegeContract(repo, tenantID, func(c *vertraege.Contract) {
		c.StartsOn = time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC)
	})
	newYearStart := seedVertraegeContract(repo, tenantID, func(c *vertraege.Contract) {
		c.StartsOn = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	})

	resp, err := srv.ListContracts(context.Background(), &vertraegev1.ListContractsRequest{
		TenantId:     tenantID.String(),
		StartsAfter:  timestamppb.New(time.Date(2025, 12, 31, 23, 59, 58, 0, time.UTC)),
		StartsBefore: timestamppb.New(time.Date(2026, 1, 1, 0, 0, 1, 0, time.UTC)),
		PageSize:     100,
	})
	require.NoError(t, err)
	gotIDs := map[string]bool{}
	for _, c := range resp.Contracts {
		gotIDs[c.Id] = true
	}
	assert.True(t, gotIDs[oldYearEnd.ID.String()], "contract starting at the last second of the year must be included")
	assert.True(t, gotIDs[newYearStart.ID.String()], "contract starting at the first instant of the next year must be included")

	// StartsAfter set to the exact new-year instant must exclude it (After is strict).
	resp, err = srv.ListContracts(context.Background(), &vertraegev1.ListContractsRequest{
		TenantId:    tenantID.String(),
		StartsAfter: timestamppb.New(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)),
		PageSize:    100,
	})
	require.NoError(t, err)
	for _, c := range resp.Contracts {
		assert.NotEqual(t, newYearStart.ID.String(), c.Id, "StartsAfter must be a strict comparison, not inclusive")
	}
}

func TestVertraege_CreateReminder_LeapDayAndYearEnd(t *testing.T) {
	repo := newStubVertraegeRepo()
	srv, _ := newVertraegeServerWithRepo(repo)
	tenantID := uuid.New()
	c := seedVertraegeContract(repo, tenantID, nil)

	// Leap-day reminder (2028-02-29) — must round-trip unchanged.
	leapDay := time.Date(2028, 2, 29, 9, 0, 0, 0, time.UTC)
	resp, err := srv.CreateReminder(context.Background(), &vertraegev1.CreateReminderRequest{
		TenantId:     tenantID.String(),
		ContractId:   c.ID.String(),
		RemindAt:     timestamppb.New(leapDay),
		ReminderType: string(vertraege.ReminderTypeExpiry),
		Subject:      "Vertrag laeuft aus",
	})
	require.NoError(t, err)
	assert.True(t, leapDay.Equal(resp.Reminder.RemindAt.AsTime()))

	// Reminder scheduled at the last instant of the year.
	yearEnd := time.Date(2026, 12, 31, 23, 59, 59, 0, time.UTC)
	resp, err = srv.CreateReminder(context.Background(), &vertraegev1.CreateReminderRequest{
		TenantId:     tenantID.String(),
		ContractId:   c.ID.String(),
		RemindAt:     timestamppb.New(yearEnd),
		ReminderType: string(vertraege.ReminderTypeRenewal),
		Subject:      "Verlaengerung pruefen",
	})
	require.NoError(t, err)
	assert.True(t, yearEnd.Equal(resp.Reminder.RemindAt.AsTime()))

	// Omitted remind_at -> zero time -> InvalidArgument from the service.
	_, err = srv.CreateReminder(context.Background(), &vertraegev1.CreateReminderRequest{
		TenantId:     tenantID.String(),
		ContractId:   c.ID.String(),
		ReminderType: string(vertraege.ReminderTypeCustom),
		Subject:      "Ohne Datum",
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

// ============================================================================
// Signature — must not leak into ListContracts.
// ============================================================================

func TestVertraege_SaveSignature_NotInListResponse(t *testing.T) {
	repo := newStubVertraegeRepo()
	srv, _ := newVertraegeServerWithRepo(repo)
	tenantID := uuid.New()
	c := seedVertraegeContract(repo, tenantID, nil)

	saveResp, err := srv.SaveSignature(context.Background(), &vertraegev1.SaveContractSignatureRequest{
		TenantId:      tenantID.String(),
		ContractId:    c.ID.String(),
		SignatureData: "data:image/png;base64,aGVsbG8=",
		SignedBy:      "Max Mustermann",
	})
	require.NoError(t, err)
	require.NotNil(t, saveResp.Contract)

	// The domain record now carries the signature (verified against the repo
	// directly, since vertraegeContractToProto never surfaces it — see the
	// journal entry for this unit).
	stored, err := repo.GetContract(context.Background(), tenantID, c.ID)
	require.NoError(t, err)
	require.NotNil(t, stored.SignatureData)
	assert.Equal(t, "data:image/png;base64,aGVsbG8=", *stored.SignatureData)

	listResp, err := srv.ListContracts(context.Background(), &vertraegev1.ListContractsRequest{
		TenantId: tenantID.String(),
		PageSize: 100,
	})
	require.NoError(t, err)
	require.Len(t, listResp.Contracts, 1)
	assert.Empty(t, listResp.Contracts[0].SignatureData, "signature data must never appear in a list response")
	assert.Empty(t, listResp.Contracts[0].SignedBy, "signed_by must never appear in a list response")
	assert.Nil(t, listResp.Contracts[0].SignedAt, "signed_at must never appear in a list response")

	// Also true of GetContract today — vertraegeContractToProto omits the
	// signature fields unconditionally, not just for the list path. That is a
	// real gap (the proto documents signature_data as "Populated only by
	// GetContract"), tracked separately rather than fixed in this coverage-only
	// unit; this assertion documents the current behaviour.
	getResp, err := srv.GetContract(context.Background(), &vertraegev1.GetContractRequest{
		TenantId:   tenantID.String(),
		ContractId: c.ID.String(),
	})
	require.NoError(t, err)
	assert.Empty(t, getResp.Contract.SignatureData, "documents current gap: GetContract also drops signature data")
}

func TestVertraege_SaveSignature_InvalidInput(t *testing.T) {
	repo := newStubVertraegeRepo()
	srv, _ := newVertraegeServerWithRepo(repo)
	tenantID := uuid.New()
	c := seedVertraegeContract(repo, tenantID, nil)

	_, err := srv.SaveSignature(context.Background(), &vertraegev1.SaveContractSignatureRequest{
		TenantId:      tenantID.String(),
		ContractId:    c.ID.String(),
		SignatureData: "not-a-data-uri",
		SignedBy:      "Someone",
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

// ============================================================================
// Parties, reminders, events — happy path clusters
// ============================================================================

func TestVertraege_PartyLifecycle(t *testing.T) {
	repo := newStubVertraegeRepo()
	srv, _ := newVertraegeServerWithRepo(repo)
	tenantID := uuid.New()
	c := seedVertraegeContract(repo, tenantID, nil)

	_, err := srv.AddParty(context.Background(), &vertraegev1.AddPartyRequest{
		TenantId:       tenantID.String(),
		ContractId:     c.ID.String(),
		PartyType:      string(vertraege.PartyTypeExternal),
		RoleInContract: "Mieter",
	})
	requireGRPCCode(t, err, codes.InvalidArgument) // external party without external_name

	extName := "Externe GmbH"
	addResp, err := srv.AddParty(context.Background(), &vertraegev1.AddPartyRequest{
		TenantId:       tenantID.String(),
		ContractId:     c.ID.String(),
		PartyType:      string(vertraege.PartyTypeExternal),
		RoleInContract: "Mieter",
		ExternalName:   &extName,
	})
	require.NoError(t, err)
	require.NotNil(t, addResp.Party)

	listResp, err := srv.ListParties(context.Background(), &vertraegev1.ListPartiesRequest{
		TenantId:   tenantID.String(),
		ContractId: c.ID.String(),
	})
	require.NoError(t, err)
	require.Len(t, listResp.Parties, 1)

	_, err = srv.RemoveParty(context.Background(), &vertraegev1.RemovePartyRequest{
		TenantId: tenantID.String(),
		PartyId:  addResp.Party.Id,
	})
	require.NoError(t, err)

	listResp, err = srv.ListParties(context.Background(), &vertraegev1.ListPartiesRequest{
		TenantId:   tenantID.String(),
		ContractId: c.ID.String(),
	})
	require.NoError(t, err)
	assert.Empty(t, listResp.Parties)
}

func TestVertraege_ReminderLifecycle(t *testing.T) {
	repo := newStubVertraegeRepo()
	srv, _ := newVertraegeServerWithRepo(repo)
	tenantID := uuid.New()
	c := seedVertraegeContract(repo, tenantID, nil)

	createResp, err := srv.CreateReminder(context.Background(), &vertraegev1.CreateReminderRequest{
		TenantId:     tenantID.String(),
		ContractId:   c.ID.String(),
		RemindAt:     timestamppb.New(time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC)),
		ReminderType: string(vertraege.ReminderTypeRenewal),
		Subject:      "Verlaengerung",
	})
	require.NoError(t, err)

	sentStatus := string(vertraege.ReminderStatusSent)
	updateResp, err := srv.UpdateReminder(context.Background(), &vertraegev1.UpdateReminderRequest{
		TenantId:   tenantID.String(),
		ReminderId: createResp.Reminder.Id,
		Status:     &sentStatus,
	})
	require.NoError(t, err)
	assert.Equal(t, "sent", updateResp.Reminder.Status)

	_, err = srv.UpdateReminder(context.Background(), &vertraegev1.UpdateReminderRequest{
		TenantId:   tenantID.String(),
		ReminderId: uuid.New().String(),
	})
	requireGRPCCode(t, err, codes.NotFound)

	listResp, err := srv.ListReminders(context.Background(), &vertraegev1.ListRemindersRequest{
		TenantId:   tenantID.String(),
		ContractId: c.ID.String(),
	})
	require.NoError(t, err)
	require.Len(t, listResp.Reminders, 1)

	_, err = srv.DeleteReminder(context.Background(), &vertraegev1.DeleteReminderRequest{
		TenantId:   tenantID.String(),
		ReminderId: createResp.Reminder.Id,
	})
	require.NoError(t, err)
}

func TestVertraege_ListContractEvents(t *testing.T) {
	repo := newStubVertraegeRepo()
	srv, _ := newVertraegeServerWithRepo(repo)
	tenantID := uuid.New()

	_, err := srv.ListContractEvents(context.Background(), &vertraegev1.ListContractEventsRequest{
		TenantId:   tenantID.String(),
		ContractId: uuid.New().String(),
	})
	requireGRPCCode(t, err, codes.NotFound) // no such contract -> 404, not an empty list

	c := seedVertraegeContract(repo, tenantID, nil)
	_, err = srv.CreateReminder(context.Background(), &vertraegev1.CreateReminderRequest{
		TenantId:     tenantID.String(),
		ContractId:   c.ID.String(),
		RemindAt:     timestamppb.New(time.Now().Add(time.Hour)),
		ReminderType: string(vertraege.ReminderTypeCustom),
		Subject:      "x",
	})
	require.NoError(t, err)

	resp, err := srv.ListContractEvents(context.Background(), &vertraegev1.ListContractEventsRequest{
		TenantId:   tenantID.String(),
		ContractId: c.ID.String(),
	})
	require.NoError(t, err)
	// The contract was seeded directly into the repo (bypassing CreateContract),
	// and CreateReminder does not emit an audit event — so the trail is empty,
	// distinct from the 404 case above for a contract that does not exist at all.
	assert.Equal(t, int32(0), resp.Total)
	assert.Empty(t, resp.Items)
}

func TestVertraege_ExportContract(t *testing.T) {
	repo := newStubVertraegeRepo()
	srv, _ := newVertraegeServerWithRepo(repo)
	tenantID := uuid.New()
	c := seedVertraegeContract(repo, tenantID, nil)

	resp, err := srv.ExportContract(context.Background(), &vertraegev1.ExportContractRequest{
		TenantId:   tenantID.String(),
		ContractId: c.ID.String(),
	})
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Payload)
	assert.Equal(t, "application/pdf", resp.ContentType)

	_, err = srv.ExportContract(context.Background(), &vertraegev1.ExportContractRequest{
		TenantId:   tenantID.String(),
		ContractId: uuid.New().String(),
	})
	requireGRPCCode(t, err, codes.NotFound)
}

// ============================================================================
// toProto conversion helpers — nil safety and wire-shape.
// ============================================================================

func TestVertraege_ToProtoNilSafety(t *testing.T) {
	assert.Nil(t, vertraegeContractToProto(nil))
	assert.Nil(t, vertraegePartyToProto(nil))
	assert.Nil(t, vertraegeReminderToProto(nil))
	assert.Nil(t, vertraegeEventToProto(nil))
}

func TestVertraege_ContractToProto_OptionalFields(t *testing.T) {
	c := &vertraege.Contract{
		ID:             uuid.New(),
		TenantID:       uuid.New(),
		ContractNumber: "V-1",
		Title:          "T",
		ContractType:   vertraege.ContractTypeOther,
		Status:         vertraege.ContractStatusDraft,
		StartsOn:       time.Now(),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	// No optional fields set.
	proto := vertraegeContractToProto(c)
	assert.Nil(t, proto.EndsOn)
	assert.Nil(t, proto.DocumentUrl)
	assert.Nil(t, proto.CreatedBy)
	assert.Nil(t, proto.SignatureProvider)
	assert.Empty(t, proto.Parties)
	assert.Empty(t, proto.Reminders)

	endsOn := time.Now().Add(24 * time.Hour)
	docURL := "https://example.com/doc.pdf"
	createdBy := uuid.New()
	sigProvider := "skribble"
	c.EndsOn = &endsOn
	c.DocumentURL = &docURL
	c.CreatedBy = &createdBy
	c.SignatureProvider = &sigProvider
	c.Parties = []*vertraege.ContractParty{{ID: uuid.New(), TenantID: c.TenantID, ContractID: c.ID}}
	c.Reminders = []*vertraege.ContractReminder{{ID: uuid.New(), TenantID: c.TenantID, ContractID: c.ID}}

	proto = vertraegeContractToProto(c)
	require.NotNil(t, proto.EndsOn)
	require.NotNil(t, proto.DocumentUrl)
	assert.Equal(t, docURL, *proto.DocumentUrl)
	require.NotNil(t, proto.CreatedBy)
	assert.Equal(t, createdBy.String(), *proto.CreatedBy)
	require.NotNil(t, proto.SignatureProvider)
	assert.Equal(t, sigProvider, *proto.SignatureProvider)
	assert.Len(t, proto.Parties, 1)
	assert.Len(t, proto.Reminders, 1)
}

func TestVertraege_PartyToProto_OptionalFields(t *testing.T) {
	p := &vertraege.ContractParty{
		ID:             uuid.New(),
		TenantID:       uuid.New(),
		ContractID:     uuid.New(),
		PartyType:      vertraege.PartyTypeContact,
		RoleInContract: "Vermieter",
		CreatedAt:      time.Now(),
	}
	proto := vertraegePartyToProto(p)
	assert.Nil(t, proto.ContactId)
	assert.Nil(t, proto.CompanyId)
	assert.Nil(t, proto.ExternalName)
	assert.Nil(t, proto.SignedOn)

	contactID := uuid.New()
	companyID := uuid.New()
	extName := "x"
	signedOn := time.Now()
	p.ContactID = &contactID
	p.CompanyID = &companyID
	p.ExternalName = &extName
	p.SignedOn = &signedOn

	proto = vertraegePartyToProto(p)
	require.NotNil(t, proto.ContactId)
	assert.Equal(t, contactID.String(), *proto.ContactId)
	require.NotNil(t, proto.CompanyId)
	assert.Equal(t, companyID.String(), *proto.CompanyId)
	require.NotNil(t, proto.SignedOn)
}

func TestVertraege_ReminderToProto_SentAt(t *testing.T) {
	r := &vertraege.ContractReminder{
		ID:           uuid.New(),
		TenantID:     uuid.New(),
		ContractID:   uuid.New(),
		RemindAt:     time.Now(),
		ReminderType: vertraege.ReminderTypePayment,
		Status:       vertraege.ReminderStatusPending,
		CreatedAt:    time.Now(),
	}
	proto := vertraegeReminderToProto(r)
	assert.Nil(t, proto.SentAt)

	sentAt := time.Now()
	r.SentAt = &sentAt
	proto = vertraegeReminderToProto(r)
	require.NotNil(t, proto.SentAt)
}

func TestVertraege_EventToProto_UserIDAndPayload(t *testing.T) {
	e := &vertraege.ContractEvent{
		ID:         uuid.New(),
		TenantID:   uuid.New(),
		ContractID: uuid.New(),
		Action:     vertraege.ContractEventCreated,
		Payload:    map[string]any{"title": "x"},
		CreatedAt:  time.Now(),
	}
	proto := vertraegeEventToProto(e)
	assert.Nil(t, proto.UserId)
	require.NotNil(t, proto.Payload)

	userID := uuid.New()
	e.UserID = &userID
	proto = vertraegeEventToProto(e)
	require.NotNil(t, proto.UserId)
	assert.Equal(t, userID.String(), *proto.UserId)

	// A payload structpb.NewStruct cannot encode (e.g. a channel value) must
	// not fail the whole conversion — the entry survives, only the payload is
	// dropped.
	unencodable := &vertraege.ContractEvent{
		ID:         uuid.New(),
		TenantID:   uuid.New(),
		ContractID: uuid.New(),
		Action:     vertraege.ContractEventUpdated,
		Payload:    map[string]any{"bad": make(chan int)},
		CreatedAt:  time.Now(),
	}
	proto = vertraegeEventToProto(unencodable)
	require.NotNil(t, proto)
	assert.Nil(t, proto.Payload)
	assert.Equal(t, "updated", proto.Action)
}

// ============================================================================
// Error mapping — table test over every sentinel plus the default fallback.
// ============================================================================

var errStubVertraegeFailure = errors.New("stub failure")

func TestMapVertraegeError_Table(t *testing.T) {
	cases := []struct {
		name string
		err  error
		code codes.Code
	}{
		{"contract_not_found", vertraege.ErrContractNotFound, codes.NotFound},
		{"party_not_found", vertraege.ErrPartyNotFound, codes.NotFound},
		{"reminder_not_found", vertraege.ErrReminderNotFound, codes.NotFound},
		{"contract_number_taken", vertraege.ErrContractNumberTaken, codes.AlreadyExists},
		{"delete_non_draft", vertraege.ErrDeleteNonDraft, codes.FailedPrecondition},
		{"invalid_input", vertraege.ErrInvalidInput, codes.InvalidArgument},
		{"generic_fallback", errStubVertraegeFailure, codes.Internal},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			requireGRPCCode(t, mapVertraegeError(tc.err), tc.code)
		})
	}
	assert.NoError(t, mapVertraegeError(nil))
}
