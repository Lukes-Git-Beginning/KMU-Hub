package audit

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
)

// MockRepository implements Repository for testing.
type MockRepository struct {
	entries     []*models.AuditEntry
	lastHash   string
	chainValid bool
	brokenSeq  int64

	createErr      error
	listErr        error
	getLastHashErr error
	verifyChainErr error
}

func NewMockRepository() *MockRepository {
	return &MockRepository{
		chainValid: true,
	}
}

func (m *MockRepository) Create(ctx context.Context, entry *models.AuditEntry) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.entries = append(m.entries, entry)
	return nil
}

func (m *MockRepository) List(ctx context.Context, filter *models.AuditFilter) ([]*models.AuditEntry, int, error) {
	if m.listErr != nil {
		return nil, 0, m.listErr
	}
	// If filter specifies a non-nil TenantID, scope results accordingly.
	if filter != nil && filter.TenantID != uuid.Nil {
		var result []*models.AuditEntry
		for _, e := range m.entries {
			if e.TenantID == filter.TenantID {
				result = append(result, e)
			}
		}
		return result, len(result), nil
	}
	return m.entries, len(m.entries), nil
}

func (m *MockRepository) GetLastHash(ctx context.Context) (string, error) {
	if m.getLastHashErr != nil {
		return "", m.getLastHashErr
	}
	return m.lastHash, nil
}

func (m *MockRepository) VerifyChain(ctx context.Context, fromSequence, toSequence int64) (bool, int64, error) {
	if m.verifyChainErr != nil {
		return false, 0, m.verifyChainErr
	}
	return m.chainValid, m.brokenSeq, nil
}

// ============================================================================
// LogEvent Tests
// ============================================================================

func TestService_LogEvent_Success(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	userID := uuid.New()
	svc.LogEvent(
		context.Background(),
		tenantID,
		&userID,
		"contact.create",
		"contact-123",
		"contact",
		nil,
		"192.168.1.1",
		"Mozilla/5.0",
		"success",
	)

	require.Len(t, repo.entries, 1)
	entry := repo.entries[0]
	assert.NotEqual(t, uuid.Nil, entry.ID)
	assert.False(t, entry.Timestamp.IsZero())
	assert.Equal(t, &userID, entry.UserID)
	assert.Equal(t, "contact.create", entry.Action)
	assert.Equal(t, "contact-123", entry.Target)
	assert.Equal(t, "contact", entry.TargetType)
	assert.Equal(t, "192.168.1.1", entry.IPAddress)
	assert.Equal(t, "Mozilla/5.0", entry.UserAgent)
	assert.Equal(t, "success", entry.Result)
	assert.Empty(t, entry.Details)
}

func TestService_LogEvent_WithDetails(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	userID := uuid.New()
	details := map[string]interface{}{
		"old_email": "old@example.com",
		"new_email": "new@example.com",
	}

	svc.LogEvent(
		context.Background(),
		tenantID,
		&userID,
		"contact.update",
		"contact-123",
		"contact",
		details,
		"10.0.0.1",
		"TestAgent",
		"success",
	)

	require.Len(t, repo.entries, 1)
	entry := repo.entries[0]
	assert.NotEmpty(t, entry.Details)

	// Verify details is valid JSON with expected content
	var parsed map[string]interface{}
	err := json.Unmarshal([]byte(entry.Details), &parsed)
	require.NoError(t, err)
	assert.Equal(t, "old@example.com", parsed["old_email"])
	assert.Equal(t, "new@example.com", parsed["new_email"])
}

func TestService_LogEvent_NilDetails(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	tenantID := uuid.New()
	userID := uuid.New()
	svc.LogEvent(
		context.Background(),
		tenantID,
		&userID,
		"user.login",
		"",
		"user",
		nil,
		"10.0.0.1",
		"TestAgent",
		"success",
	)

	require.Len(t, repo.entries, 1)
	assert.Empty(t, repo.entries[0].Details)
}

func TestService_LogEvent_RepoError(t *testing.T) {
	repo := NewMockRepository()
	repo.createErr = errors.New("database connection lost")
	svc := NewService(repo)

	tenantID := uuid.New()
	userID := uuid.New()

	// LogEvent is fire-and-forget: it does NOT return an error.
	// It should not panic either.
	svc.LogEvent(
		context.Background(),
		tenantID,
		&userID,
		"contact.delete",
		"contact-456",
		"contact",
		nil,
		"10.0.0.1",
		"TestAgent",
		"success",
	)

	// Entry was not persisted due to error
	assert.Empty(t, repo.entries)
}

func TestService_LogEvent_NilUserID(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	svc.LogEvent(
		context.Background(),
		uuid.Nil,
		nil,
		"system.startup",
		"",
		"system",
		nil,
		"",
		"",
		"success",
	)

	require.Len(t, repo.entries, 1)
	assert.Nil(t, repo.entries[0].UserID)
}

// ============================================================================
// ListEntries Tests
// ============================================================================

func TestService_ListEntries_Success(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	id1, id2 := uuid.New(), uuid.New()
	repo.entries = []*models.AuditEntry{
		{ID: id1, Action: "contact.create", Result: "success", Timestamp: time.Now()},
		{ID: id2, Action: "user.login", Result: "success", Timestamp: time.Now()},
	}

	entries, total, err := svc.ListEntries(context.Background(), &models.AuditFilter{
		Limit: 50,
	})

	require.NoError(t, err)
	assert.Len(t, entries, 2)
	assert.Equal(t, 2, total)
}

func TestService_ListEntries_DefaultLimit(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	filter := &models.AuditFilter{Limit: 0} // Should default to 50
	_, _, err := svc.ListEntries(context.Background(), filter)

	require.NoError(t, err)
	assert.Equal(t, 50, filter.Limit)
}

func TestService_ListEntries_MaxLimit(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	filter := &models.AuditFilter{Limit: 5000} // Should be clamped to 1000
	_, _, err := svc.ListEntries(context.Background(), filter)

	require.NoError(t, err)
	assert.Equal(t, 1000, filter.Limit)
}

func TestService_ListEntries_NegativeOffset(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	filter := &models.AuditFilter{Offset: -10} // Should be clamped to 0
	_, _, err := svc.ListEntries(context.Background(), filter)

	require.NoError(t, err)
	assert.Equal(t, 0, filter.Offset)
}

func TestService_ListEntries_RepoError(t *testing.T) {
	repo := NewMockRepository()
	repo.listErr = errors.New("query timeout")
	svc := NewService(repo)

	entries, total, err := svc.ListEntries(context.Background(), &models.AuditFilter{Limit: 50})

	assert.Error(t, err)
	assert.Nil(t, entries)
	assert.Equal(t, 0, total)
}

// ============================================================================
// VerifyChainIntegrity Tests
// ============================================================================

func TestService_VerifyChainIntegrity_Valid(t *testing.T) {
	repo := NewMockRepository()
	repo.chainValid = true
	repo.brokenSeq = 0
	svc := NewService(repo)

	valid, brokenSeq, err := svc.VerifyChainIntegrity(context.Background(), 1, 100)

	require.NoError(t, err)
	assert.True(t, valid)
	assert.Equal(t, int64(0), brokenSeq)
}

func TestService_VerifyChainIntegrity_Invalid(t *testing.T) {
	repo := NewMockRepository()
	repo.chainValid = false
	repo.brokenSeq = 42
	svc := NewService(repo)

	valid, brokenSeq, err := svc.VerifyChainIntegrity(context.Background(), 1, 100)

	require.NoError(t, err)
	assert.False(t, valid)
	assert.Equal(t, int64(42), brokenSeq)
}

func TestService_VerifyChainIntegrity_MinFromSequence(t *testing.T) {
	repo := NewMockRepository()
	repo.chainValid = true
	svc := NewService(repo)

	// fromSequence < 1 should be clamped to 1
	valid, _, err := svc.VerifyChainIntegrity(context.Background(), -5, 10)

	require.NoError(t, err)
	assert.True(t, valid)
}

func TestService_VerifyChainIntegrity_ToLessThanFrom(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	valid, _, err := svc.VerifyChainIntegrity(context.Background(), 50, 10)

	assert.Error(t, err)
	assert.False(t, valid)
	assert.Contains(t, err.Error(), "toSequence")
}

// ============================================================================
// ExportEntries Tests
// ============================================================================

func TestService_ExportEntries_CSV(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	userID := uuid.New()
	repo.entries = []*models.AuditEntry{
		{
			ID:          uuid.New(),
			SequenceNum: 1,
			Timestamp:   time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
			UserID:      &userID,
			Action:      "contact.create",
			Target:      "contact-abc",
			TargetType:  "contact",
			Details:     `{"name":"John"}`,
			IPAddress:   "10.0.0.1",
			UserAgent:   "TestAgent",
			Result:      "success",
			EntryHash:   "abc123",
		},
	}

	data, err := svc.ExportEntries(context.Background(), &models.AuditFilter{Limit: 50, Offset: 10}, "csv")

	require.NoError(t, err)
	assert.NotEmpty(t, data)

	csvStr := string(data)
	// Should contain CSV header
	assert.Contains(t, csvStr, "ID")
	assert.Contains(t, csvStr, "Action")
	// Should contain entry data
	assert.Contains(t, csvStr, "contact.create")
	assert.Contains(t, csvStr, "success")
}

func TestService_ExportEntries_JSON(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	userID := uuid.New()
	repo.entries = []*models.AuditEntry{
		{
			ID:          uuid.New(),
			SequenceNum: 1,
			Timestamp:   time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
			UserID:      &userID,
			Action:      "user.login",
			Target:      "",
			TargetType:  "user",
			Result:      "success",
			EntryHash:   "def456",
		},
	}

	data, err := svc.ExportEntries(context.Background(), &models.AuditFilter{Limit: 50}, "json")

	require.NoError(t, err)
	assert.NotEmpty(t, data)

	// Should be valid JSON
	var parsed []map[string]interface{}
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)
	assert.Len(t, parsed, 1)
	assert.Equal(t, "user.login", parsed[0]["action"])
	assert.Equal(t, "success", parsed[0]["result"])
}

func TestService_ExportEntries_UnsupportedFormat(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	repo.entries = []*models.AuditEntry{}

	data, err := svc.ExportEntries(context.Background(), &models.AuditFilter{}, "xml")

	assert.Error(t, err)
	assert.Nil(t, data)
	assert.True(t, strings.Contains(err.Error(), "unsupported export format"))
}

// ============================================================================
// Cross-tenant isolation tests (audit_log)
// ============================================================================

// TestLogEvent_TenantID verifies that LogEvent correctly tags the audit entry
// with the supplied tenantID, ensuring entries are scoped per tenant.
func TestLogEvent_TenantID(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	tenantA := uuid.New()
	tenantB := uuid.New()
	userID := uuid.New()

	svc.LogEvent(context.Background(), tenantA, &userID, "contact.create", "c1", "contact", nil, "", "", "success")
	svc.LogEvent(context.Background(), tenantB, &userID, "contact.create", "c2", "contact", nil, "", "", "success")

	require.Len(t, repo.entries, 2)
	assert.Equal(t, tenantA, repo.entries[0].TenantID, "first entry must carry tenant A's ID")
	assert.Equal(t, tenantB, repo.entries[1].TenantID, "second entry must carry tenant B's ID")
}

// TestListEntries_TenantScoped verifies that ListEntries with a TenantID filter
// only returns entries belonging to that tenant.
func TestListEntries_TenantScoped(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	tenantA := uuid.New()
	tenantB := uuid.New()

	// Seed 2 entries for tenant A and 1 for tenant B.
	repo.entries = []*models.AuditEntry{
		{ID: uuid.New(), TenantID: tenantA, Action: "a1", Result: "success"},
		{ID: uuid.New(), TenantID: tenantA, Action: "a2", Result: "success"},
		{ID: uuid.New(), TenantID: tenantB, Action: "b1", Result: "success"},
	}

	entries, total, err := svc.ListEntries(context.Background(), &models.AuditFilter{
		TenantID: tenantA,
		Limit:    50,
	})

	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, entries, 2)
	for _, e := range entries {
		assert.Equal(t, tenantA, e.TenantID, "ListEntries must only return entries for the requested tenant")
	}
}
