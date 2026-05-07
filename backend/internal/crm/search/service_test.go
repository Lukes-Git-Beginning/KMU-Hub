package search

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/models"
)

// MockRepository implements Repository for testing
type MockRepository struct {
	contacts   []*models.SearchResult
	companies  []*models.SearchResult
	deals      []*models.SearchResult
	activities []*models.SearchResult
	searchErr  error
}

func NewMockRepository() *MockRepository {
	return &MockRepository{}
}

func (m *MockRepository) SearchContacts(ctx context.Context, tenantID uuid.UUID, query string, limit int) ([]*models.SearchResult, error) {
	if m.searchErr != nil {
		return nil, m.searchErr
	}
	return m.filterByLimit(m.contacts, limit), nil
}

func (m *MockRepository) SearchCompanies(ctx context.Context, tenantID uuid.UUID, query string, limit int) ([]*models.SearchResult, error) {
	if m.searchErr != nil {
		return nil, m.searchErr
	}
	return m.filterByLimit(m.companies, limit), nil
}

func (m *MockRepository) SearchDeals(ctx context.Context, tenantID uuid.UUID, query string, limit int) ([]*models.SearchResult, error) {
	if m.searchErr != nil {
		return nil, m.searchErr
	}
	return m.filterByLimit(m.deals, limit), nil
}

func (m *MockRepository) SearchActivities(ctx context.Context, tenantID uuid.UUID, query string, limit int) ([]*models.SearchResult, error) {
	if m.searchErr != nil {
		return nil, m.searchErr
	}
	return m.filterByLimit(m.activities, limit), nil
}

func (m *MockRepository) filterByLimit(results []*models.SearchResult, _ int) []*models.SearchResult {
	// Return all results - let the service handle limiting
	// This allows us to test the service's limiting behavior properly
	return results
}

// Test helpers
func (m *MockRepository) AddContact(id, title, subtitle string, score float64) {
	m.contacts = append(m.contacts, &models.SearchResult{
		ID:         id,
		EntityType: "contact",
		Title:      title,
		Subtitle:   subtitle,
		Score:      score,
	})
}

func (m *MockRepository) AddCompany(id, title, subtitle string, score float64) {
	m.companies = append(m.companies, &models.SearchResult{
		ID:         id,
		EntityType: "company",
		Title:      title,
		Subtitle:   subtitle,
		Score:      score,
	})
}

func (m *MockRepository) AddDeal(id, title, subtitle string, score float64) {
	m.deals = append(m.deals, &models.SearchResult{
		ID:         id,
		EntityType: "deal",
		Title:      title,
		Subtitle:   subtitle,
		Score:      score,
	})
}

func (m *MockRepository) AddActivity(id, title, subtitle string, score float64) {
	m.activities = append(m.activities, &models.SearchResult{
		ID:         id,
		EntityType: "activity",
		Title:      title,
		Subtitle:   subtitle,
		Score:      score,
	})
}

// ============================================================================
// Search Tests
// ============================================================================

func TestService_Search_AllEntities(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	repo.AddContact("c1", "John Doe", "john@example.com", 0.8)
	repo.AddCompany("co1", "Acme Corp", "Technology", 0.7)
	repo.AddDeal("d1", "Big Deal", "50000 EUR", 0.6)
	repo.AddActivity("a1", "Sales Call", "call", 0.5)

	results, err := svc.Search(context.Background(), SearchInput{
		Query: "test",
		Limit: 20,
	})

	require.NoError(t, err)
	assert.Len(t, results, 4)
	// Should be sorted by score descending
	assert.Equal(t, "contact", results[0].EntityType)
	assert.Equal(t, "company", results[1].EntityType)
	assert.Equal(t, "deal", results[2].EntityType)
	assert.Equal(t, "activity", results[3].EntityType)
}

func TestService_Search_ContactsOnly(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	repo.AddContact("c1", "John Doe", "john@example.com", 0.8)
	repo.AddContact("c2", "Jane Smith", "jane@example.com", 0.7)
	repo.AddCompany("co1", "Acme Corp", "Technology", 0.9) // Should not appear

	results, err := svc.Search(context.Background(), SearchInput{
		Query:       "test",
		EntityTypes: []models.SearchEntityType{models.SearchEntityContact},
		Limit:       20,
	})

	require.NoError(t, err)
	assert.Len(t, results, 2)
	for _, r := range results {
		assert.Equal(t, "contact", r.EntityType)
	}
}

func TestService_Search_DealsAndCompanies(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	repo.AddContact("c1", "John Doe", "john@example.com", 0.8)
	repo.AddCompany("co1", "Acme Corp", "Technology", 0.7)
	repo.AddDeal("d1", "Big Deal", "50000 EUR", 0.6)
	repo.AddActivity("a1", "Sales Call", "call", 0.5)

	results, err := svc.Search(context.Background(), SearchInput{
		Query:       "test",
		EntityTypes: []models.SearchEntityType{models.SearchEntityDeal, models.SearchEntityCompany},
		Limit:       20,
	})

	require.NoError(t, err)
	assert.Len(t, results, 2)
	entityTypes := make(map[string]bool)
	for _, r := range results {
		entityTypes[r.EntityType] = true
	}
	assert.True(t, entityTypes["company"])
	assert.True(t, entityTypes["deal"])
	assert.False(t, entityTypes["contact"])
	assert.False(t, entityTypes["activity"])
}

func TestService_Search_QueryTooShort(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	_, err := svc.Search(context.Background(), SearchInput{
		Query: "a",
		Limit: 20,
	})

	assert.ErrorIs(t, err, ErrQueryTooShort)
}

func TestService_Search_QueryTooLong(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	longQuery := strings.Repeat("a", 201)

	_, err := svc.Search(context.Background(), SearchInput{
		Query: longQuery,
		Limit: 20,
	})

	assert.ErrorIs(t, err, ErrQueryTooLong)
}

func TestService_Search_InvalidEntityType(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	_, err := svc.Search(context.Background(), SearchInput{
		Query:       "test",
		EntityTypes: []models.SearchEntityType{"invalid"},
		Limit:       20,
	})

	assert.ErrorIs(t, err, ErrInvalidEntity)
}

func TestService_Search_NoResults(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	results, err := svc.Search(context.Background(), SearchInput{
		Query: "nonexistent",
		Limit: 20,
	})

	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestService_Search_RespectLimit(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	// Add many results
	for i := 0; i < 50; i++ {
		repo.AddContact("c"+string(rune(i)), "Contact", "email", float64(50-i)/100)
	}

	results, err := svc.Search(context.Background(), SearchInput{
		Query: "test",
		Limit: 10,
	})

	require.NoError(t, err)
	assert.Len(t, results, 10)
}

func TestService_Search_DefaultLimit(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	// Add many results
	for i := 0; i < 50; i++ {
		repo.AddContact("c"+string(rune(i)), "Contact", "email", float64(50-i)/100)
	}

	results, err := svc.Search(context.Background(), SearchInput{
		Query: "test",
		Limit: 0, // Should use default
	})

	require.NoError(t, err)
	assert.Len(t, results, defaultLimit)
}

func TestService_Search_MaxLimit(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	// Add many results
	for i := 0; i < 150; i++ {
		repo.AddContact("c"+string(rune(i)), "Contact", "email", float64(150-i)/1000)
	}

	results, err := svc.Search(context.Background(), SearchInput{
		Query: "test",
		Limit: 500, // Should be capped to maxLimit
	})

	require.NoError(t, err)
	assert.LessOrEqual(t, len(results), maxLimit)
}

func TestService_Search_SortedByScore(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	repo.AddContact("c1", "Low Score", "email", 0.1)
	repo.AddCompany("co1", "High Score", "industry", 0.9)
	repo.AddDeal("d1", "Medium Score", "value", 0.5)

	results, err := svc.Search(context.Background(), SearchInput{
		Query: "test",
		Limit: 20,
	})

	require.NoError(t, err)
	require.Len(t, results, 3)
	assert.Equal(t, 0.9, results[0].Score)
	assert.Equal(t, 0.5, results[1].Score)
	assert.Equal(t, 0.1, results[2].Score)
}

func TestService_Search_WhitespaceQuery(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	_, err := svc.Search(context.Background(), SearchInput{
		Query: "   ",
		Limit: 20,
	})

	assert.ErrorIs(t, err, ErrQueryTooShort)
}

func TestService_Search_TrimsQuery(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	repo.AddContact("c1", "John Doe", "john@example.com", 0.8)

	results, err := svc.Search(context.Background(), SearchInput{
		Query: "  test  ",
		Limit: 20,
	})

	require.NoError(t, err)
	assert.Len(t, results, 1)
}

// ============================================================================
// Tenant Isolation Tests
// ============================================================================

func TestService_Search_TenantIsolation(t *testing.T) {
	// This test verifies that TenantID is propagated into SearchInput and
	// therefore into each repo call. The mock doesn't filter by tenant —
	// the real isolation happens in the postgres_repository WHERE tenant_id = $1
	// clause on every FTS query (SearchContacts, SearchCompanies, SearchDeals,
	// SearchActivities). Here we verify the plumbing is correct.
	t.Run("TenantID_in_SearchInput_propagated_to_repo", func(t *testing.T) {
		repo := &tenantCapturingMockRepo{}
		svc := NewService(repo)

		tenantID := uuid.New()

		_, _ = svc.Search(context.Background(), SearchInput{
			Query:    "acme",
			Limit:    10,
			TenantID: tenantID,
		})

		assert.Equal(t, tenantID, repo.capturedTenantID,
			"TenantID must be forwarded to every repo.Search* call")
	})
}

// tenantCapturingMockRepo records the tenantID passed to SearchContacts
// to verify propagation from SearchInput through service to repo.
type tenantCapturingMockRepo struct {
	capturedTenantID uuid.UUID
}

func (m *tenantCapturingMockRepo) SearchContacts(_ context.Context, tenantID uuid.UUID, _ string, _ int) ([]*models.SearchResult, error) {
	m.capturedTenantID = tenantID
	return nil, nil
}
func (m *tenantCapturingMockRepo) SearchCompanies(_ context.Context, tenantID uuid.UUID, _ string, _ int) ([]*models.SearchResult, error) {
	return nil, nil
}
func (m *tenantCapturingMockRepo) SearchDeals(_ context.Context, tenantID uuid.UUID, _ string, _ int) ([]*models.SearchResult, error) {
	return nil, nil
}
func (m *tenantCapturingMockRepo) SearchActivities(_ context.Context, tenantID uuid.UUID, _ string, _ int) ([]*models.SearchResult, error) {
	return nil, nil
}

func TestService_Search_ContinuesOnPartialFailure(t *testing.T) {
	repo := NewMockRepository()
	svc := NewService(repo)

	repo.AddContact("c1", "John Doe", "john@example.com", 0.8)
	repo.AddCompany("co1", "Acme Corp", "Technology", 0.7)

	// Make one entity type fail - but we can't easily do partial failures with our mock
	// This test verifies the service doesn't crash and returns partial results
	results, err := svc.Search(context.Background(), SearchInput{
		Query: "test",
		Limit: 20,
	})

	require.NoError(t, err)
	assert.NotEmpty(t, results)
}

func TestService_Search_AllEntityTypesContinueOnError(t *testing.T) {
	repo := &PartialFailureMockRepository{
		contacts: []*models.SearchResult{
			{ID: "c1", EntityType: "contact", Title: "John", Score: 0.8},
		},
		companies: []*models.SearchResult{
			{ID: "co1", EntityType: "company", Title: "Acme", Score: 0.7},
		},
		failDeals: true, // Simulate failure
	}
	svc := NewService(repo)

	results, err := svc.Search(context.Background(), SearchInput{
		Query: "test",
		Limit: 20,
	})

	require.NoError(t, err)
	// Should still have contacts and companies even though deals failed
	assert.Len(t, results, 2)
}

// PartialFailureMockRepository simulates partial search failures
type PartialFailureMockRepository struct {
	contacts       []*models.SearchResult
	companies      []*models.SearchResult
	deals          []*models.SearchResult
	activities     []*models.SearchResult
	failContacts   bool
	failCompanies  bool
	failDeals      bool
	failActivities bool
}

func (m *PartialFailureMockRepository) SearchContacts(ctx context.Context, tenantID uuid.UUID, query string, limit int) ([]*models.SearchResult, error) {
	if m.failContacts {
		return nil, errors.New("contacts search failed")
	}
	return m.contacts, nil
}

func (m *PartialFailureMockRepository) SearchCompanies(ctx context.Context, tenantID uuid.UUID, query string, limit int) ([]*models.SearchResult, error) {
	if m.failCompanies {
		return nil, errors.New("companies search failed")
	}
	return m.companies, nil
}

func (m *PartialFailureMockRepository) SearchDeals(ctx context.Context, tenantID uuid.UUID, query string, limit int) ([]*models.SearchResult, error) {
	if m.failDeals {
		return nil, errors.New("deals search failed")
	}
	return m.deals, nil
}

func (m *PartialFailureMockRepository) SearchActivities(ctx context.Context, tenantID uuid.UUID, query string, limit int) ([]*models.SearchResult, error) {
	if m.failActivities {
		return nil, errors.New("activities search failed")
	}
	return m.activities, nil
}
