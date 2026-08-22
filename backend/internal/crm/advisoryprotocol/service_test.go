package advisoryprotocol

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Mock repository
// ---------------------------------------------------------------------------

type mockRepository struct {
	protocols map[string]*Protocol
	contacts  map[string]bool // contactKey "contactID:tenantID" -> exists

	// Error injection
	createErr      error
	getErr         error
	updateErr      error
	deleteErr      error
	handOverErr    error
	referralErr    error
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		protocols: make(map[string]*Protocol),
		contacts:  make(map[string]bool),
	}
}

func (m *mockRepository) addContact(contactID, tenantID uuid.UUID) {
	m.contacts[contactID.String()+":"+tenantID.String()] = true
}

func (m *mockRepository) Create(_ context.Context, p *Protocol) error {
	if m.createErr != nil {
		return m.createErr
	}
	clone := *p
	m.protocols[p.ID.String()] = &clone
	return nil
}

func (m *mockRepository) GetByID(_ context.Context, id, tenantID uuid.UUID) (*Protocol, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	p, ok := m.protocols[id.String()]
	if !ok || p.TenantID != tenantID {
		return nil, ErrProtocolNotFound
	}
	clone := *p
	return &clone, nil
}

func (m *mockRepository) ListByContact(_ context.Context, contactID, tenantID uuid.UUID) ([]*Protocol, error) {
	var result []*Protocol
	for _, p := range m.protocols {
		if p.ContactID == contactID && p.TenantID == tenantID {
			clone := *p
			result = append(result, &clone)
		}
	}
	return result, nil
}

func (m *mockRepository) Update(_ context.Context, p *Protocol) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	existing, ok := m.protocols[p.ID.String()]
	if !ok {
		return ErrProtocolNotFound
	}
	if existing.Status == "finalized" {
		return errors.New("cannot update finalized protocol")
	}
	clone := *p
	m.protocols[p.ID.String()] = &clone
	return nil
}

func (m *mockRepository) Delete(_ context.Context, id, tenantID uuid.UUID) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.protocols, id.String())
	return nil
}

func (m *mockRepository) HandOver(_ context.Context, id, tenantID uuid.UUID, at time.Time) error {
	if m.handOverErr != nil {
		return m.handOverErr
	}
	p, ok := m.protocols[id.String()]
	if !ok || p.TenantID != tenantID {
		return ErrProtocolNotFound
	}
	p.Status = "finalized"
	p.HandedOverAt = &at
	return nil
}

func (m *mockRepository) ContactExists(_ context.Context, contactID, tenantID uuid.UUID) (bool, error) {
	return m.contacts[contactID.String()+":"+tenantID.String()], nil
}

func (m *mockRepository) GetReferralReport(_ context.Context, tenantID uuid.UUID) ([]*ReferralEntry, error) {
	if m.referralErr != nil {
		return nil, m.referralErr
	}
	return []*ReferralEntry{}, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func setupService(t *testing.T) (*Service, *mockRepository, uuid.UUID, uuid.UUID) {
	t.Helper()
	repo := newMockRepository()
	svc := NewService(repo)
	tenantID := uuid.New()
	contactID := uuid.New()
	repo.addContact(contactID, tenantID)
	return svc, repo, tenantID, contactID
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestCreate_Success(t *testing.T) {
	svc, _, tenantID, contactID := setupService(t)
	createdBy := uuid.New()

	p, err := svc.Create(context.Background(), CreateInput{
		TenantID:  tenantID,
		ContactID: contactID,
		CreatedBy: createdBy,
	})

	require.NoError(t, err)
	assert.Equal(t, "draft", p.Status)
	assert.Equal(t, tenantID, p.TenantID)
	assert.Equal(t, contactID, p.ContactID)
	assert.Equal(t, 3, p.RiskClass) // default
	assert.NotEqual(t, uuid.Nil, p.ID)
}

func TestCreate_ContactNotFound(t *testing.T) {
	svc, _, tenantID, _ := setupService(t)

	_, err := svc.Create(context.Background(), CreateInput{
		TenantID:  tenantID,
		ContactID: uuid.New(), // unknown contact
		CreatedBy: uuid.New(),
	})

	assert.ErrorIs(t, err, ErrContactNotFound)
}

func TestCreate_InvalidTenant(t *testing.T) {
	svc, _, _, contactID := setupService(t)

	_, err := svc.Create(context.Background(), CreateInput{
		TenantID:  uuid.Nil,
		ContactID: contactID,
		CreatedBy: uuid.New(),
	})

	assert.ErrorIs(t, err, ErrInvalidTenant)
}

func TestUpdate_DraftSucceeds(t *testing.T) {
	svc, _, tenantID, contactID := setupService(t)

	p, err := svc.Create(context.Background(), CreateInput{
		TenantID:  tenantID,
		ContactID: contactID,
		CreatedBy: uuid.New(),
	})
	require.NoError(t, err)

	in := UpdateInput{
		Advisor:   "Test Berater",
		RiskClass: 5,
		Products:  []Product{},
		KnownAssetClasses: []string{"stocks", "etf"},
		InvestmentPurpose: []string{"retirement"},
		WarningsGiven:     []string{"risk"},
	}

	updated, err := svc.Update(context.Background(), p.ID, tenantID, in)
	require.NoError(t, err)
	assert.Equal(t, "Test Berater", updated.Advisor)
	assert.Equal(t, 5, updated.RiskClass)
	assert.Equal(t, []string{"stocks", "etf"}, updated.KnownAssetClasses)
}

func TestUpdate_FinalizedProtocolIsImmutable(t *testing.T) {
	svc, repo, tenantID, contactID := setupService(t)

	p, err := svc.Create(context.Background(), CreateInput{
		TenantID:  tenantID,
		ContactID: contactID,
		CreatedBy: uuid.New(),
	})
	require.NoError(t, err)

	// Finalize directly in the mock
	repo.protocols[p.ID.String()].Status = "finalized"

	in := UpdateInput{Advisor: "should not work", RiskClass: 3, Products: []Product{}}
	_, err = svc.Update(context.Background(), p.ID, tenantID, in)
	assert.ErrorIs(t, err, ErrProtocolFinalized)
}

func TestDelete_DraftSucceeds(t *testing.T) {
	svc, _, tenantID, contactID := setupService(t)

	p, err := svc.Create(context.Background(), CreateInput{
		TenantID:  tenantID,
		ContactID: contactID,
		CreatedBy: uuid.New(),
	})
	require.NoError(t, err)

	err = svc.Delete(context.Background(), p.ID, tenantID)
	require.NoError(t, err)
}

func TestDelete_FinalizedProtocolIsImmutable(t *testing.T) {
	svc, repo, tenantID, contactID := setupService(t)

	p, err := svc.Create(context.Background(), CreateInput{
		TenantID:  tenantID,
		ContactID: contactID,
		CreatedBy: uuid.New(),
	})
	require.NoError(t, err)
	repo.protocols[p.ID.String()].Status = "finalized"

	err = svc.Delete(context.Background(), p.ID, tenantID)
	assert.ErrorIs(t, err, ErrProtocolFinalized)
}

func TestHandOver_SetsImmutable(t *testing.T) {
	svc, _, tenantID, contactID := setupService(t)

	p, err := svc.Create(context.Background(), CreateInput{
		TenantID:  tenantID,
		ContactID: contactID,
		CreatedBy: uuid.New(),
	})
	require.NoError(t, err)

	finalized, err := svc.HandOver(context.Background(), p.ID, tenantID)
	require.NoError(t, err)
	assert.Equal(t, "finalized", finalized.Status)
	assert.NotNil(t, finalized.HandedOverAt)

	// Subsequent update must be rejected
	in := UpdateInput{Advisor: "attempt", RiskClass: 3, Products: []Product{}}
	_, err = svc.Update(context.Background(), p.ID, tenantID, in)
	assert.ErrorIs(t, err, ErrProtocolFinalized)
}

func TestHandOver_IsIdempotent(t *testing.T) {
	svc, _, tenantID, contactID := setupService(t)

	p, err := svc.Create(context.Background(), CreateInput{
		TenantID:  tenantID,
		ContactID: contactID,
		CreatedBy: uuid.New(),
	})
	require.NoError(t, err)

	_, err = svc.HandOver(context.Background(), p.ID, tenantID)
	require.NoError(t, err)

	// Second HandOver must not return error
	_, err = svc.HandOver(context.Background(), p.ID, tenantID)
	assert.NoError(t, err)
}

func TestHandOver_NotFoundProtocol(t *testing.T) {
	svc, _, tenantID, _ := setupService(t)

	_, err := svc.HandOver(context.Background(), uuid.New(), tenantID)
	assert.ErrorIs(t, err, ErrProtocolNotFound)
}

// ---------------------------------------------------------------------------
// Cross-tenant isolation
// ---------------------------------------------------------------------------

func TestGetByID_CrossTenantIsolation(t *testing.T) {
	svc, _, tenantA, contactID := setupService(t)
	tenantB := uuid.New()

	p, err := svc.Create(context.Background(), CreateInput{
		TenantID:  tenantA,
		ContactID: contactID,
		CreatedBy: uuid.New(),
	})
	require.NoError(t, err)

	// TenantB must not see TenantA's protocol
	_, err = svc.GetByID(context.Background(), p.ID, tenantB)
	assert.ErrorIs(t, err, ErrProtocolNotFound)
}

func TestUpdate_CrossTenantIsolation(t *testing.T) {
	svc, _, tenantA, contactID := setupService(t)
	tenantB := uuid.New()

	p, err := svc.Create(context.Background(), CreateInput{
		TenantID:  tenantA,
		ContactID: contactID,
		CreatedBy: uuid.New(),
	})
	require.NoError(t, err)

	in := UpdateInput{Advisor: "attacker", RiskClass: 3, Products: []Product{}}
	_, err = svc.Update(context.Background(), p.ID, tenantB, in)
	assert.ErrorIs(t, err, ErrProtocolNotFound)
}

func TestDelete_CrossTenantIsolation(t *testing.T) {
	svc, _, tenantA, contactID := setupService(t)
	tenantB := uuid.New()

	p, err := svc.Create(context.Background(), CreateInput{
		TenantID:  tenantA,
		ContactID: contactID,
		CreatedBy: uuid.New(),
	})
	require.NoError(t, err)

	// TenantB must not delete TenantA's protocol
	err = svc.Delete(context.Background(), p.ID, tenantB)
	assert.ErrorIs(t, err, ErrProtocolNotFound)
}

// ---------------------------------------------------------------------------
// Risk class validation
// ---------------------------------------------------------------------------

func TestUpdate_InvalidRiskClass(t *testing.T) {
	svc, _, tenantID, contactID := setupService(t)

	p, err := svc.Create(context.Background(), CreateInput{
		TenantID:  tenantID,
		ContactID: contactID,
		CreatedBy: uuid.New(),
	})
	require.NoError(t, err)

	in := UpdateInput{Advisor: "ok", RiskClass: 0, Products: []Product{}}
	_, err = svc.Update(context.Background(), p.ID, tenantID, in)
	assert.ErrorIs(t, err, ErrInvalidRiskClass)

	in2 := UpdateInput{Advisor: "ok", RiskClass: 8, Products: []Product{}}
	_, err = svc.Update(context.Background(), p.ID, tenantID, in2)
	assert.ErrorIs(t, err, ErrInvalidRiskClass)
}

// TestUpdate_InvalidProductRiskClass proves fix-gateway-advisory-product-riskclass-not-validated
// at the service layer: a Product.RiskClass outside 1-7 is rejected even though the
// protocol-level RiskClass is valid. This is the root-cause check, independent of
// whatever validate tags the gateway's HTTP struct carries.
func TestUpdate_InvalidProductRiskClass(t *testing.T) {
	svc, _, tenantID, contactID := setupService(t)

	p, err := svc.Create(context.Background(), CreateInput{
		TenantID:  tenantID,
		ContactID: contactID,
		CreatedBy: uuid.New(),
	})
	require.NoError(t, err)

	in := UpdateInput{Advisor: "ok", RiskClass: 3, Products: []Product{{ID: "p1", Name: "Fund A", RiskClass: 0}}}
	_, err = svc.Update(context.Background(), p.ID, tenantID, in)
	assert.ErrorIs(t, err, ErrInvalidRiskClass)

	in2 := UpdateInput{Advisor: "ok", RiskClass: 3, Products: []Product{{ID: "p1", Name: "Fund A", RiskClass: 99}}}
	_, err = svc.Update(context.Background(), p.ID, tenantID, in2)
	assert.ErrorIs(t, err, ErrInvalidRiskClass)

	in3 := UpdateInput{Advisor: "ok", RiskClass: 3, Products: []Product{{ID: "p1", Name: "Fund A", RiskClass: 4}}}
	_, err = svc.Update(context.Background(), p.ID, tenantID, in3)
	assert.NoError(t, err)
}

// ---------------------------------------------------------------------------
// Segment rule / referral report
// ---------------------------------------------------------------------------

func TestGetReferralReport_Empty(t *testing.T) {
	svc, _, tenantID, _ := setupService(t)

	entries, err := svc.GetReferralReport(context.Background(), tenantID)
	require.NoError(t, err)
	assert.Empty(t, entries)
}
