package vertraege

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// mockRepository
// ============================================================================

type mockRepository struct {
	contracts map[uuid.UUID]*Contract
	parties   map[uuid.UUID]*ContractParty
	reminders map[uuid.UUID]*ContractReminder

	// number -> contractID for uniqueness checks (tenant-scoped)
	numbers map[string]uuid.UUID

	createContractErr error
	expiredCount      int64
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		contracts: make(map[uuid.UUID]*Contract),
		parties:   make(map[uuid.UUID]*ContractParty),
		reminders: make(map[uuid.UUID]*ContractReminder),
		numbers:   make(map[string]uuid.UUID),
	}
}

func (m *mockRepository) CreateContract(_ context.Context, c *Contract) error {
	if m.createContractErr != nil {
		return m.createContractErr
	}
	m.contracts[c.ID] = c
	m.numbers[c.TenantID.String()+":"+c.ContractNumber] = c.ID
	return nil
}

func (m *mockRepository) UpdateContract(_ context.Context, c *Contract) error {
	m.contracts[c.ID] = c
	return nil
}

func (m *mockRepository) GetContract(_ context.Context, tenantID, contractID uuid.UUID) (*Contract, error) {
	c, ok := m.contracts[contractID]
	if !ok || c.TenantID != tenantID {
		return nil, ErrContractNotFound
	}
	// populate parties & reminders
	var parties []*ContractParty
	for _, p := range m.parties {
		if p.ContractID == contractID {
			parties = append(parties, p)
		}
	}
	var reminders []*ContractReminder
	for _, r := range m.reminders {
		if r.ContractID == contractID {
			reminders = append(reminders, r)
		}
	}
	clone := *c
	clone.Parties = parties
	clone.Reminders = reminders
	return &clone, nil
}

func (m *mockRepository) ListContracts(_ context.Context, tenantID uuid.UUID, filter ListContractsFilter, offset, limit int) ([]*Contract, int, error) {
	var result []*Contract
	for _, c := range m.contracts {
		if c.TenantID != tenantID {
			continue
		}
		if filter.Status != nil && c.Status != *filter.Status {
			continue
		}
		if filter.Type != nil && c.ContractType != *filter.Type {
			continue
		}
		if filter.ContactID != nil {
			// Check whether any party for this contract has the requested contact_id.
			hasContact := false
			for _, p := range m.parties {
				if p.ContractID == c.ID && p.ContactID != nil && *p.ContactID == *filter.ContactID {
					hasContact = true
					break
				}
			}
			if !hasContact {
				continue
			}
		}
		result = append(result, c)
	}
	total := len(result)
	if offset >= total {
		return []*Contract{}, total, nil
	}
	end := min(offset+limit, total)
	return result[offset:end], total, nil
}

func (m *mockRepository) DeleteContract(_ context.Context, tenantID, contractID uuid.UUID) error {
	delete(m.contracts, contractID)
	return nil
}

func (m *mockRepository) ContractNumberExists(_ context.Context, tenantID uuid.UUID, number string, excludeID *uuid.UUID) (bool, error) {
	key := tenantID.String() + ":" + number
	existingID, ok := m.numbers[key]
	if !ok {
		return false, nil
	}
	if excludeID != nil && existingID == *excludeID {
		return false, nil
	}
	return true, nil
}

func (m *mockRepository) ExpireContracts(_ context.Context) (int64, error) {
	return m.expiredCount, nil
}

func (m *mockRepository) AddParty(_ context.Context, p *ContractParty) error {
	m.parties[p.ID] = p
	return nil
}

func (m *mockRepository) RemoveParty(_ context.Context, tenantID, partyID uuid.UUID) error {
	delete(m.parties, partyID)
	return nil
}

func (m *mockRepository) ListParties(_ context.Context, tenantID, contractID uuid.UUID) ([]*ContractParty, error) {
	var result []*ContractParty
	for _, p := range m.parties {
		if p.ContractID == contractID && p.TenantID == tenantID {
			result = append(result, p)
		}
	}
	return result, nil
}

func (m *mockRepository) CreateReminder(_ context.Context, r *ContractReminder) error {
	m.reminders[r.ID] = r
	return nil
}

func (m *mockRepository) UpdateReminder(_ context.Context, r *ContractReminder) error {
	m.reminders[r.ID] = r
	return nil
}

func (m *mockRepository) GetReminder(_ context.Context, tenantID, reminderID uuid.UUID) (*ContractReminder, error) {
	r, ok := m.reminders[reminderID]
	if !ok || r.TenantID != tenantID {
		return nil, ErrReminderNotFound
	}
	return r, nil
}

func (m *mockRepository) DeleteReminder(_ context.Context, tenantID, reminderID uuid.UUID) error {
	delete(m.reminders, reminderID)
	return nil
}

func (m *mockRepository) ListReminders(_ context.Context, tenantID, contractID uuid.UUID, onlyPending bool) ([]*ContractReminder, error) {
	var result []*ContractReminder
	for _, r := range m.reminders {
		if r.ContractID != contractID || r.TenantID != tenantID {
			continue
		}
		if onlyPending && r.Status != ReminderStatusPending {
			continue
		}
		result = append(result, r)
	}
	return result, nil
}

func (m *mockRepository) ClaimDueReminders(_ context.Context) ([]*ContractReminder, error) {
	var due []*ContractReminder
	now := time.Now()
	for _, r := range m.reminders {
		if r.Status == ReminderStatusPending && !r.RemindAt.After(now) {
			r.Status = ReminderStatusSent
			t := now
			r.SentAt = &t
			due = append(due, r)
		}
	}
	return due, nil
}

func (m *mockRepository) MarkReminderSent(_ context.Context, reminderID uuid.UUID, sentAt time.Time) error {
	r, ok := m.reminders[reminderID]
	if !ok {
		return ErrReminderNotFound
	}
	r.Status = ReminderStatusSent
	r.SentAt = &sentAt
	return nil
}

// compile-time interface check
var _ Repository = (*mockRepository)(nil)

// ============================================================================
// Helper
// ============================================================================

func addContract(repo *mockRepository, tenantID uuid.UUID, number, title string, status ContractStatus) *Contract {
	c := &Contract{
		ID:             uuid.New(),
		TenantID:       tenantID,
		ContractNumber: number,
		Title:          title,
		ContractType:   ContractTypeService,
		Status:         status,
		StartsOn:       time.Now(),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	repo.contracts[c.ID] = c
	repo.numbers[tenantID.String()+":"+number] = c.ID
	return c
}

// ============================================================================
// CreateContract Tests
// ============================================================================

func TestService_CreateContract_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	c, err := svc.CreateContract(context.Background(), CreateContractInput{
		TenantID:       tenantID,
		ContractNumber: "CT-001",
		Title:          "Dienstleistungsvertrag",
		ContractType:   ContractTypeService,
		StartsOn:       time.Now(),
	})

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, c.ID)
	assert.Equal(t, ContractStatusDraft, c.Status)
	assert.Equal(t, "CT-001", c.ContractNumber)
}

func TestService_CreateContract_EmptyTitle(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	_, err := svc.CreateContract(context.Background(), CreateContractInput{
		TenantID:       uuid.New(),
		ContractNumber: "CT-001",
		Title:          "   ",
		ContractType:   ContractTypeService,
	})

	assert.ErrorIs(t, err, ErrInvalidInput)
}

func TestService_CreateContract_EmptyNumber(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	_, err := svc.CreateContract(context.Background(), CreateContractInput{
		TenantID:     uuid.New(),
		Title:        "Vertrag",
		ContractType: ContractTypeService,
	})

	assert.ErrorIs(t, err, ErrInvalidInput)
}

func TestService_CreateContract_NumberTaken(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	addContract(repo, tenantID, "CT-001", "Existing", ContractStatusDraft)

	_, err := svc.CreateContract(context.Background(), CreateContractInput{
		TenantID:       tenantID,
		ContractNumber: "CT-001",
		Title:          "Duplicate",
		ContractType:   ContractTypeService,
	})

	assert.ErrorIs(t, err, ErrContractNumberTaken)
}

func TestService_CreateContract_DifferentTenants_SameNumber(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantA := uuid.New()
	tenantB := uuid.New()
	addContract(repo, tenantA, "CT-001", "Tenant A", ContractStatusDraft)

	c, err := svc.CreateContract(context.Background(), CreateContractInput{
		TenantID:       tenantB,
		ContractNumber: "CT-001",
		Title:          "Tenant B contract",
		ContractType:   ContractTypeService,
	})

	require.NoError(t, err)
	assert.Equal(t, tenantB, c.TenantID)
}

// ============================================================================
// UpdateContract Tests
// ============================================================================

func TestService_UpdateContract_UpdatesTitle(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	c := addContract(repo, tenantID, "CT-001", "Old Title", ContractStatusDraft)

	newTitle := "New Title"
	updated, err := svc.UpdateContract(context.Background(), UpdateContractInput{
		TenantID:   tenantID,
		ContractID: c.ID,
		Title:      &newTitle,
	})

	require.NoError(t, err)
	assert.Equal(t, "New Title", updated.Title)
}

func TestService_UpdateContract_NotFound(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	_, err := svc.UpdateContract(context.Background(), UpdateContractInput{
		TenantID:   uuid.New(),
		ContractID: uuid.New(),
	})

	assert.ErrorIs(t, err, ErrContractNotFound)
}

func TestService_UpdateContract_ClearEndsOn(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	ends := time.Now().Add(365 * 24 * time.Hour)
	c := addContract(repo, tenantID, "CT-002", "Fixed Term", ContractStatusActive)
	c.EndsOn = &ends
	repo.contracts[c.ID] = c

	updated, err := svc.UpdateContract(context.Background(), UpdateContractInput{
		TenantID:    tenantID,
		ContractID:  c.ID,
		ClearEndsOn: true,
	})

	require.NoError(t, err)
	assert.Nil(t, updated.EndsOn)
}

// ============================================================================
// DeleteContract Tests
// ============================================================================

func TestService_DeleteContract_Draft(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	c := addContract(repo, tenantID, "CT-001", "Draft", ContractStatusDraft)

	err := svc.DeleteContract(context.Background(), tenantID, c.ID)

	require.NoError(t, err)
	assert.NotContains(t, repo.contracts, c.ID)
}

func TestService_DeleteContract_NonDraft_Rejected(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	c := addContract(repo, tenantID, "CT-001", "Active", ContractStatusActive)

	err := svc.DeleteContract(context.Background(), tenantID, c.ID)

	assert.ErrorIs(t, err, ErrDeleteNonDraft)
}

func TestService_DeleteContract_NotFound(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	err := svc.DeleteContract(context.Background(), uuid.New(), uuid.New())

	assert.ErrorIs(t, err, ErrContractNotFound)
}

// ============================================================================
// ListContracts Tests
// ============================================================================

func TestService_ListContracts_Pagination(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	for i := range 25 {
		addContract(repo, tenantID, fmt.Sprintf("CT-%03d", i), "Contract", ContractStatusDraft)
	}

	contracts, total, err := svc.ListContracts(context.Background(), ListContractsInput{
		TenantID: tenantID,
	})

	require.NoError(t, err)
	assert.Equal(t, 25, total)
	assert.Len(t, contracts, 20) // default page size
}

func TestService_ListContracts_FilterByStatus(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	addContract(repo, tenantID, "CT-001", "Draft One", ContractStatusDraft)
	addContract(repo, tenantID, "CT-002", "Active One", ContractStatusActive)
	addContract(repo, tenantID, "CT-003", "Active Two", ContractStatusActive)

	active := ContractStatusActive
	contracts, total, err := svc.ListContracts(context.Background(), ListContractsInput{
		TenantID: tenantID,
		Status:   &active,
	})

	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, contracts, 2)
}

func TestService_ListContracts_FilterByContactID(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	contactID := uuid.New()
	otherContactID := uuid.New()

	// Contract A: party with contactID
	cA := addContract(repo, tenantID, "CT-A", "Contract A", ContractStatusDraft)
	repo.parties[uuid.New()] = &ContractParty{
		ID:         uuid.New(),
		TenantID:   tenantID,
		ContractID: cA.ID,
		PartyType:  PartyTypeContact,
		ContactID:  &contactID,
	}

	// Contract B: party with a different contact
	cB := addContract(repo, tenantID, "CT-B", "Contract B", ContractStatusActive)
	repo.parties[uuid.New()] = &ContractParty{
		ID:         uuid.New(),
		TenantID:   tenantID,
		ContractID: cB.ID,
		PartyType:  PartyTypeContact,
		ContactID:  &otherContactID,
	}

	// Contract C: no parties at all
	addContract(repo, tenantID, "CT-C", "Contract C", ContractStatusDraft)

	contracts, total, err := svc.ListContracts(context.Background(), ListContractsInput{
		TenantID:  tenantID,
		ContactID: &contactID,
		PageSize:  20,
	})

	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, contracts, 1)
	assert.Equal(t, cA.ID, contracts[0].ID)
}

// ============================================================================
// Party Tests
// ============================================================================

func TestService_AddParty_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	c := addContract(repo, tenantID, "CT-001", "Service Agreement", ContractStatusDraft)

	p, err := svc.AddParty(context.Background(), AddPartyInput{
		TenantID:       tenantID,
		ContractID:     c.ID,
		PartyType:      PartyTypeContact,
		RoleInContract: "lessee",
	})

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, p.ID)
	assert.Equal(t, "lessee", p.RoleInContract)
}

func TestService_AddParty_External_NoName_Rejected(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()

	_, err := svc.AddParty(context.Background(), AddPartyInput{
		TenantID:       tenantID,
		ContractID:     uuid.New(),
		PartyType:      PartyTypeExternal,
		RoleInContract: "lessor",
		// ExternalName missing
	})

	assert.ErrorIs(t, err, ErrInvalidInput)
}

func TestService_AddParty_EmptyRole_Rejected(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	_, err := svc.AddParty(context.Background(), AddPartyInput{
		TenantID:   uuid.New(),
		ContractID: uuid.New(),
		PartyType:  PartyTypeContact,
		// RoleInContract missing
	})

	assert.ErrorIs(t, err, ErrInvalidInput)
}

func TestService_ListParties_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	contractID := uuid.New()

	for range 3 {
		repo.parties[uuid.New()] = &ContractParty{
			ID:             uuid.New(),
			TenantID:       tenantID,
			ContractID:     contractID,
			PartyType:      PartyTypeContact,
			RoleInContract: "party",
			CreatedAt:      time.Now(),
		}
	}

	parties, err := svc.ListParties(context.Background(), tenantID, contractID)

	require.NoError(t, err)
	assert.Len(t, parties, 3)
}

// ============================================================================
// Reminder Tests
// ============================================================================

func TestService_CreateReminder_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	contractID := uuid.New()

	rem, err := svc.CreateReminder(context.Background(), CreateReminderInput{
		TenantID:     tenantID,
		ContractID:   contractID,
		RemindAt:     time.Now().Add(30 * 24 * time.Hour),
		ReminderType: ReminderTypeRenewal,
		Subject:      "Renewal due",
		Message:      "Contract is due for renewal.",
	})

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, rem.ID)
	assert.Equal(t, ReminderStatusPending, rem.Status)
}

func TestService_CreateReminder_EmptySubject_Rejected(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	_, err := svc.CreateReminder(context.Background(), CreateReminderInput{
		TenantID:     uuid.New(),
		ContractID:   uuid.New(),
		RemindAt:     time.Now().Add(time.Hour),
		ReminderType: ReminderTypeExpiry,
		Subject:      "  ",
	})

	assert.ErrorIs(t, err, ErrInvalidInput)
}

func TestService_CreateReminder_ZeroTime_Rejected(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	_, err := svc.CreateReminder(context.Background(), CreateReminderInput{
		TenantID:     uuid.New(),
		ContractID:   uuid.New(),
		ReminderType: ReminderTypeExpiry,
		Subject:      "Test",
	})

	assert.ErrorIs(t, err, ErrInvalidInput)
}

func TestService_UpdateReminder_NotFound(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	_, err := svc.UpdateReminder(context.Background(), UpdateReminderInput{
		TenantID:   uuid.New(),
		ReminderID: uuid.New(),
	})

	assert.ErrorIs(t, err, ErrReminderNotFound)
}

func TestService_DeleteReminder_Success(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	rem := &ContractReminder{
		ID:         uuid.New(),
		TenantID:   tenantID,
		ContractID: uuid.New(),
		Status:     ReminderStatusPending,
		CreatedAt:  time.Now(),
	}
	repo.reminders[rem.ID] = rem

	err := svc.DeleteReminder(context.Background(), tenantID, rem.ID)

	require.NoError(t, err)
	assert.NotContains(t, repo.reminders, rem.ID)
}

func TestService_ListReminders_OnlyPending(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	contractID := uuid.New()

	repo.reminders[uuid.New()] = &ContractReminder{
		ID: uuid.New(), TenantID: tenantID, ContractID: contractID,
		Status: ReminderStatusPending, CreatedAt: time.Now(),
	}
	repo.reminders[uuid.New()] = &ContractReminder{
		ID: uuid.New(), TenantID: tenantID, ContractID: contractID,
		Status: ReminderStatusSent, CreatedAt: time.Now(),
	}

	pending, err := svc.ListReminders(context.Background(), tenantID, contractID, true)

	require.NoError(t, err)
	assert.Len(t, pending, 1)
	assert.Equal(t, ReminderStatusPending, pending[0].Status)
}

// ============================================================================
// ExportContract
// ============================================================================

func TestService_ExportContract_TextDump(t *testing.T) {
	repo := newMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	c := addContract(repo, tenantID, "CT-EXP-001", "Export Test Contract", ContractStatusActive)

	data, err := svc.ExportContract(context.Background(), tenantID, c.ID)

	require.NoError(t, err)
	assert.Contains(t, string(data), "Export Test Contract")
	assert.Contains(t, string(data), "CT-EXP-001")
}
