package consent

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockRepository implements Repository for testing.
type MockRepository struct {
	contacts         map[uuid.UUID]bool
	consentRecords   []*ConsentRecord
	latestConsents   []*ConsentRecord
	deletionRequests map[uuid.UUID]*GDPRDeletionRequest

	// Error injection
	createConsentErr      error
	getConsentHistoryErr  error
	getLatestConsentsErr  error
	createDeletionErr     error
	getDeletionErr        error
	updateDeletionErr     error
	anonymizeErr          error
	contactExistsErr      error
}

func NewMockRepository() *MockRepository {
	return &MockRepository{
		contacts:         make(map[uuid.UUID]bool),
		consentRecords:   make([]*ConsentRecord, 0),
		deletionRequests: make(map[uuid.UUID]*GDPRDeletionRequest),
	}
}

func (m *MockRepository) CreateConsentRecord(_ context.Context, record *ConsentRecord) error {
	if m.createConsentErr != nil {
		return m.createConsentErr
	}
	m.consentRecords = append(m.consentRecords, record)
	return nil
}

func (m *MockRepository) GetConsentHistory(_ context.Context, tenantID, contactID uuid.UUID, consentType string) ([]*ConsentRecord, error) {
	if m.getConsentHistoryErr != nil {
		return nil, m.getConsentHistoryErr
	}
	var result []*ConsentRecord
	for _, r := range m.consentRecords {
		if r.ContactID == contactID && r.ConsentType == consentType {
			result = append(result, r)
		}
	}
	return result, nil
}

func (m *MockRepository) GetLatestConsents(_ context.Context, _, _ uuid.UUID) ([]*ConsentRecord, error) {
	if m.getLatestConsentsErr != nil {
		return nil, m.getLatestConsentsErr
	}
	return m.latestConsents, nil
}

func (m *MockRepository) CreateDeletionRequest(_ context.Context, req *GDPRDeletionRequest) error {
	if m.createDeletionErr != nil {
		return m.createDeletionErr
	}
	m.deletionRequests[req.ID] = req
	return nil
}

func (m *MockRepository) GetDeletionRequest(_ context.Context, id uuid.UUID) (*GDPRDeletionRequest, error) {
	if m.getDeletionErr != nil {
		return nil, m.getDeletionErr
	}
	req, ok := m.deletionRequests[id]
	if !ok {
		return nil, ErrDeletionRequestNotFound
	}
	return req, nil
}

func (m *MockRepository) UpdateDeletionRequest(_ context.Context, req *GDPRDeletionRequest) error {
	if m.updateDeletionErr != nil {
		return m.updateDeletionErr
	}
	m.deletionRequests[req.ID] = req
	return nil
}

func (m *MockRepository) AnonymizeContact(_ context.Context, _ uuid.UUID) error {
	if m.anonymizeErr != nil {
		return m.anonymizeErr
	}
	return nil
}

func (m *MockRepository) ContactExists(_ context.Context, contactID uuid.UUID) (bool, error) {
	if m.contactExistsErr != nil {
		return false, m.contactExistsErr
	}
	return m.contacts[contactID], nil
}

// AddContact registers a contact ID as existing.
func (m *MockRepository) AddContact(id uuid.UUID) {
	m.contacts[id] = true
}

// AddDeletionRequest pre-populates a deletion request.
func (m *MockRepository) AddDeletionRequest(req *GDPRDeletionRequest) {
	m.deletionRequests[req.ID] = req
}

// ============================================================================
// GrantConsent Tests
// ============================================================================

func TestService_GrantConsent_Success(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	contactID := uuid.New()
	repo.AddContact(contactID)

	ip := "192.168.1.1"
	createdBy := uuid.New()

	tenantID := uuid.New()
	record, err := svc.GrantConsent(
		context.Background(),
		tenantID,
		contactID,
		"marketing_email",
		"web_form",
		"consent",
		&ip,
		&createdBy,
	)

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, record.ID)
	assert.Equal(t, contactID, record.ContactID)
	assert.Equal(t, "marketing_email", record.ConsentType)
	assert.True(t, record.Granted)
	assert.Equal(t, "consent", record.LegalBasis)
	assert.Equal(t, "web_form", record.Source)
	assert.Equal(t, &ip, record.IPAddress)
	assert.NotNil(t, record.GrantedAt)
	assert.Nil(t, record.RevokedAt)
	assert.Equal(t, &createdBy, record.CreatedBy)
	assert.NotZero(t, record.CreatedAt)

	// Verify record was persisted
	assert.Len(t, repo.consentRecords, 1)
}

func TestService_GrantConsent_InvalidConsentType(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	contactID := uuid.New()
	repo.AddContact(contactID)

	_, err := svc.GrantConsent(
		context.Background(),
		uuid.New(),
		contactID,
		"invalid_type",
		"web_form",
		"consent",
		nil,
		nil,
	)

	assert.ErrorIs(t, err, ErrInvalidConsentType)
	assert.Empty(t, repo.consentRecords)
}

func TestService_GrantConsent_InvalidLegalBasis(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	contactID := uuid.New()
	repo.AddContact(contactID)

	_, err := svc.GrantConsent(
		context.Background(),
		uuid.New(),
		contactID,
		"marketing_email",
		"web_form",
		"invalid_basis",
		nil,
		nil,
	)

	assert.ErrorIs(t, err, ErrInvalidLegalBasis)
	assert.Empty(t, repo.consentRecords)
}

func TestService_GrantConsent_ContactNotFound(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	_, err := svc.GrantConsent(
		context.Background(),
		uuid.New(),
		uuid.New(), // non-existent contact
		"marketing_email",
		"web_form",
		"consent",
		nil,
		nil,
	)

	assert.ErrorIs(t, err, ErrContactNotFound)
	assert.Empty(t, repo.consentRecords)
}

func TestService_GrantConsent_RepoError(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	contactID := uuid.New()
	repo.AddContact(contactID)
	repo.createConsentErr = errors.New("database connection lost")

	_, err := svc.GrantConsent(
		context.Background(),
		uuid.New(),
		contactID,
		"newsletter",
		"web_form",
		"consent",
		nil,
		nil,
	)

	assert.Error(t, err)
	assert.Equal(t, "database connection lost", err.Error())
}

// ============================================================================
// RevokeConsent Tests
// ============================================================================

func TestService_RevokeConsent_Success(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	contactID := uuid.New()
	repo.AddContact(contactID)

	revokedBy := uuid.New()

	record, err := svc.RevokeConsent(
		context.Background(),
		uuid.New(),
		contactID,
		"marketing_email",
		"User requested opt-out",
		&revokedBy,
	)

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, record.ID)
	assert.Equal(t, contactID, record.ContactID)
	assert.Equal(t, "marketing_email", record.ConsentType)
	assert.False(t, record.Granted)
	assert.Equal(t, "consent", record.LegalBasis)
	assert.Equal(t, "User requested opt-out", record.Notes)
	assert.Nil(t, record.GrantedAt)
	assert.NotNil(t, record.RevokedAt)
	assert.Equal(t, &revokedBy, record.CreatedBy)
	assert.NotZero(t, record.CreatedAt)

	assert.Len(t, repo.consentRecords, 1)
}

func TestService_RevokeConsent_InvalidConsentType(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	contactID := uuid.New()
	repo.AddContact(contactID)

	_, err := svc.RevokeConsent(
		context.Background(),
		uuid.New(),
		contactID,
		"bogus_type",
		"notes",
		nil,
	)

	assert.ErrorIs(t, err, ErrInvalidConsentType)
}

func TestService_RevokeConsent_ContactNotFound(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	_, err := svc.RevokeConsent(
		context.Background(),
		uuid.New(),
		uuid.New(),
		"marketing_email",
		"notes",
		nil,
	)

	assert.ErrorIs(t, err, ErrContactNotFound)
}

// ============================================================================
// GetContactConsents Tests
// ============================================================================

func TestService_GetContactConsents_Success(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	contactID := uuid.New()
	now := time.Now()

	repo.latestConsents = []*ConsentRecord{
		{
			ID:          uuid.New(),
			ContactID:   contactID,
			ConsentType: "marketing_email",
			Granted:     true,
			LegalBasis:  "consent",
			GrantedAt:   &now,
			CreatedAt:   now,
		},
		{
			ID:          uuid.New(),
			ContactID:   contactID,
			ConsentType: "newsletter",
			Granted:     false,
			LegalBasis:  "consent",
			RevokedAt:   &now,
			CreatedAt:   now,
		},
	}

	summary, err := svc.GetContactConsents(context.Background(), uuid.New(), contactID)

	require.NoError(t, err)
	assert.Equal(t, contactID, summary.ContactID)
	assert.Len(t, summary.Consents, 2)

	emailConsent := summary.Consents["marketing_email"]
	require.NotNil(t, emailConsent)
	assert.True(t, emailConsent.Granted)

	newsletterConsent := summary.Consents["newsletter"]
	require.NotNil(t, newsletterConsent)
	assert.False(t, newsletterConsent.Granted)
}

func TestService_GetContactConsents_Empty(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	contactID := uuid.New()
	repo.latestConsents = []*ConsentRecord{}

	summary, err := svc.GetContactConsents(context.Background(), uuid.New(), contactID)

	require.NoError(t, err)
	assert.Equal(t, contactID, summary.ContactID)
	assert.Empty(t, summary.Consents)
}

// ============================================================================
// GetConsentHistory Tests
// ============================================================================

func TestService_GetConsentHistory_Success(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	contactID := uuid.New()
	now := time.Now()

	// Pre-populate consent records
	repo.consentRecords = []*ConsentRecord{
		{
			ID:          uuid.New(),
			ContactID:   contactID,
			ConsentType: "marketing_email",
			Granted:     true,
			LegalBasis:  "consent",
			GrantedAt:   &now,
			CreatedAt:   now,
		},
		{
			ID:          uuid.New(),
			ContactID:   contactID,
			ConsentType: "marketing_email",
			Granted:     false,
			LegalBasis:  "consent",
			RevokedAt:   &now,
			CreatedAt:   now.Add(time.Hour),
		},
		{
			ID:          uuid.New(),
			ContactID:   contactID,
			ConsentType: "newsletter", // different type, should not appear
			Granted:     true,
			LegalBasis:  "consent",
			CreatedAt:   now,
		},
	}

	records, err := svc.GetConsentHistory(context.Background(), uuid.New(), contactID, "marketing_email")

	require.NoError(t, err)
	assert.Len(t, records, 2)
	for _, r := range records {
		assert.Equal(t, "marketing_email", r.ConsentType)
	}
}

func TestService_GetConsentHistory_InvalidType(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	_, err := svc.GetConsentHistory(context.Background(), uuid.New(), uuid.New(), "invalid_type")

	assert.ErrorIs(t, err, ErrInvalidConsentType)
}

// ============================================================================
// RequestDeletion Tests
// ============================================================================

func TestService_RequestDeletion_Success(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	contactID := uuid.New()
	repo.AddContact(contactID)

	requestedBy := uuid.New()

	req, err := svc.RequestDeletion(
		context.Background(),
		contactID,
		"GDPR Art. 17 request from data subject",
		&requestedBy,
	)

	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, req.ID)
	assert.Equal(t, contactID, req.ContactID)
	assert.Equal(t, &requestedBy, req.RequestedBy)
	assert.Equal(t, "GDPR Art. 17 request from data subject", req.Reason)
	assert.Equal(t, "pending", req.Status)
	assert.Nil(t, req.CompletedAt)
	assert.NotZero(t, req.CreatedAt)

	// Verify persisted
	assert.Len(t, repo.deletionRequests, 1)
}

func TestService_RequestDeletion_ContactNotFound(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	_, err := svc.RequestDeletion(
		context.Background(),
		uuid.New(),
		"reason",
		nil,
	)

	assert.ErrorIs(t, err, ErrContactNotFound)
	assert.Empty(t, repo.deletionRequests)
}

// ============================================================================
// ProcessDeletion Tests
// ============================================================================

func TestService_ProcessDeletion_Success(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	contactID := uuid.New()
	requestID := uuid.New()

	repo.AddDeletionRequest(&GDPRDeletionRequest{
		ID:        requestID,
		ContactID: contactID,
		Status:    "pending",
		CreatedAt: time.Now(),
	})

	err := svc.ProcessDeletion(context.Background(), requestID)

	require.NoError(t, err)

	// Verify request was updated
	updated := repo.deletionRequests[requestID]
	assert.Equal(t, "completed", updated.Status)
	assert.NotNil(t, updated.CompletedAt)
}

func TestService_ProcessDeletion_AlreadyComplete(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	requestID := uuid.New()
	completedAt := time.Now()

	repo.AddDeletionRequest(&GDPRDeletionRequest{
		ID:          requestID,
		ContactID:   uuid.New(),
		Status:      "completed",
		CompletedAt: &completedAt,
		CreatedAt:   time.Now(),
	})

	err := svc.ProcessDeletion(context.Background(), requestID)

	assert.ErrorIs(t, err, ErrDeletionAlreadyComplete)
}

func TestService_ProcessDeletion_NotFound(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	err := svc.ProcessDeletion(context.Background(), uuid.New())

	assert.ErrorIs(t, err, ErrDeletionRequestNotFound)
}

func TestService_ProcessDeletion_AnonymizeError(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	contactID := uuid.New()
	requestID := uuid.New()

	repo.AddDeletionRequest(&GDPRDeletionRequest{
		ID:        requestID,
		ContactID: contactID,
		Status:    "pending",
		CreatedAt: time.Now(),
	})
	repo.anonymizeErr = errors.New("anonymization failed")

	err := svc.ProcessDeletion(context.Background(), requestID)

	assert.Error(t, err)
	assert.Equal(t, "anonymization failed", err.Error())

	// Request should NOT be marked completed
	req := repo.deletionRequests[requestID]
	assert.Equal(t, "pending", req.Status)
	assert.Nil(t, req.CompletedAt)
}
