package employee

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kmuhub/kmuhub/internal/auth"
	"github.com/kmuhub/kmuhub/internal/models"
)

// ============================================================================
// Mock repositories
// ============================================================================

type mockEmployeeRepo struct {
	profiles map[uuid.UUID]*models.EmployeeProfile // key: profile ID
	byUser   map[uuid.UUID]*models.EmployeeProfile // key: user ID
}

func newMockEmployeeRepo() *mockEmployeeRepo {
	return &mockEmployeeRepo{
		profiles: make(map[uuid.UUID]*models.EmployeeProfile),
		byUser:   make(map[uuid.UUID]*models.EmployeeProfile),
	}
}

func (m *mockEmployeeRepo) Create(_ context.Context, profile *models.EmployeeProfile) error {
	m.profiles[profile.ID] = profile
	m.byUser[profile.UserID] = profile
	return nil
}

func (m *mockEmployeeRepo) GetByID(_ context.Context, id uuid.UUID) (*models.EmployeeProfile, error) {
	p, ok := m.profiles[id]
	if !ok {
		return nil, ErrEmployeeNotFound
	}
	return p, nil
}

func (m *mockEmployeeRepo) GetByUserID(_ context.Context, userID uuid.UUID) (*models.EmployeeProfile, error) {
	p, ok := m.byUser[userID]
	if !ok {
		return nil, ErrEmployeeNotFound
	}
	return p, nil
}

func (m *mockEmployeeRepo) List(_ context.Context, _ EmployeeFilter) ([]*models.EmployeeProfile, int, error) {
	var results []*models.EmployeeProfile
	for _, p := range m.profiles {
		results = append(results, p)
	}
	return results, len(results), nil
}

func (m *mockEmployeeRepo) Update(_ context.Context, profile *models.EmployeeProfile) error {
	m.profiles[profile.ID] = profile
	m.byUser[profile.UserID] = profile
	return nil
}

type mockDocCategoryRepo struct {
	categories map[uuid.UUID]*models.HRDocumentCategory
}

func newMockDocCategoryRepo() *mockDocCategoryRepo {
	return &mockDocCategoryRepo{categories: make(map[uuid.UUID]*models.HRDocumentCategory)}
}

// visibleToTenant mirrors the Postgres repo: a tenant sees its own categories
// plus the system seeds carrying the zero-UUID tenant.
func visibleToTenant(c *models.HRDocumentCategory, tenantID uuid.UUID) bool {
	return c.TenantID == tenantID || c.TenantID == systemSeedTenantID
}

func (m *mockDocCategoryRepo) ListByTenant(_ context.Context, tenantID uuid.UUID) ([]*models.HRDocumentCategory, error) {
	var results []*models.HRDocumentCategory
	for _, c := range m.categories {
		if visibleToTenant(c, tenantID) {
			results = append(results, c)
		}
	}
	return results, nil
}

func (m *mockDocCategoryRepo) GetByID(_ context.Context, tenantID, id uuid.UUID) (*models.HRDocumentCategory, error) {
	c, ok := m.categories[id]
	if !ok || !visibleToTenant(c, tenantID) {
		return nil, ErrDocumentCategoryNotFound
	}
	return c, nil
}

type mockDocRepo struct {
	documents []*models.EmployeeDocument
}

func newMockDocRepo() *mockDocRepo {
	return &mockDocRepo{}
}

func (m *mockDocRepo) Create(_ context.Context, doc *models.EmployeeDocument) error {
	m.documents = append(m.documents, doc)
	return nil
}

func (m *mockDocRepo) ListByEmployee(_ context.Context, employeeID uuid.UUID, callerRole string) ([]*models.EmployeeDocument, error) {
	var results []*models.EmployeeDocument
	for _, d := range m.documents {
		if d.EmployeeID == employeeID {
			results = append(results, d)
		}
	}
	// Simulate visibility filtering
	if callerRole == "employee" {
		// Return only employee-visible docs
		var filtered []*models.EmployeeDocument
		for _, d := range results {
			if d.CategoryName == "Sonstiges" { // employee-visible category
				filtered = append(filtered, d)
			}
		}
		return filtered, nil
	}
	if callerRole == "manager" {
		// Return manager and employee visible docs
		var filtered []*models.EmployeeDocument
		for _, d := range results {
			if d.CategoryName == "Sonstiges" || d.CategoryName == "Zeugnisse" {
				filtered = append(filtered, d)
			}
		}
		return filtered, nil
	}
	// admin/hr: return all
	return results, nil
}

func (m *mockDocRepo) Delete(_ context.Context, _, id uuid.UUID) error {
	for i, d := range m.documents {
		if d.ID == id {
			m.documents = append(m.documents[:i], m.documents[i+1:]...)
			return nil
		}
	}
	return ErrDocumentNotFound
}

// ============================================================================
// Test fixtures
// ============================================================================

var (
	testTenantID      = uuid.MustParse("10000000-0000-0000-0000-000000000001")
	testEmployeeID    = uuid.MustParse("60000000-0000-0000-0000-000000000001")
	testUserID        = uuid.MustParse("20000000-0000-0000-0000-000000000001")
	testManagerUserID = uuid.MustParse("30000000-0000-0000-0000-000000000001")
	testCatID         = uuid.MustParse("70000000-0000-0000-0000-000000000001")
)

func setupTestService() (*Service, *mockEmployeeRepo, *mockDocCategoryRepo, *mockDocRepo) {
	empRepo := newMockEmployeeRepo()
	catRepo := newMockDocCategoryRepo()
	docRepo := newMockDocRepo()

	// Add test employee
	empRepo.profiles[testEmployeeID] = &models.EmployeeProfile{
		ID:              testEmployeeID,
		UserID:          testUserID,
		Department:      "Engineering",
		PositionTitle:   "Developer",
		ContractType:    models.HRContractFullTime,
		WorkDaysPerWeek: 5,
		AnnualLeaveDays: 30,
		ManagerUserID:   &testManagerUserID,
		StartDate:       time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	empRepo.byUser[testUserID] = empRepo.profiles[testEmployeeID]

	// Add document categories
	catRepo.categories[testCatID] = &models.HRDocumentCategory{
		ID:         testCatID,
		TenantID:   testTenantID,
		Name:       "Arbeitsvertrag",
		Key:        "arbeitsvertrag",
		Visibility: models.HRDocVisibilityHROnly,
		IsSystem:   true,
		SortOrder:  1,
	}

	svc := NewService(empRepo, catRepo, docRepo)
	return svc, empRepo, catRepo, docRepo
}

// ============================================================================
// Tests: Profile Update
// ============================================================================

func TestUpdateEmployee_EmployeeOnlySelfServiceFields(t *testing.T) {
	svc, _, _, _ := setupTestService()
	ctx := context.Background()

	// Employee updates only allowed fields
	name := "Jane Doe"
	phone := "+49 123 456789"
	result, err := svc.UpdateEmployee(ctx, testEmployeeID, UpdateEmployeeInput{
		EmergencyContactName:  &name,
		EmergencyContactPhone: &phone,
	}, "employee")
	require.NoError(t, err)
	assert.Equal(t, "Jane Doe", result.EmergencyContactName)
	assert.Equal(t, "+49 123 456789", result.EmergencyContactPhone)
}

func TestUpdateEmployee_EmployeeRestrictedFieldsDenied(t *testing.T) {
	svc, _, _, _ := setupTestService()
	ctx := context.Background()

	// Employee tries to change department
	dept := "Marketing"
	_, err := svc.UpdateEmployee(ctx, testEmployeeID, UpdateEmployeeInput{
		Department: &dept,
	}, "employee")
	assert.ErrorIs(t, err, ErrUnauthorizedFieldUpdate)
}

func TestUpdateEmployee_EmployeePositionDenied(t *testing.T) {
	svc, _, _, _ := setupTestService()
	ctx := context.Background()

	pos := "CTO"
	_, err := svc.UpdateEmployee(ctx, testEmployeeID, UpdateEmployeeInput{
		PositionTitle: &pos,
	}, "employee")
	assert.ErrorIs(t, err, ErrUnauthorizedFieldUpdate)
}

func TestUpdateEmployee_EmployeeContractTypeDenied(t *testing.T) {
	svc, _, _, _ := setupTestService()
	ctx := context.Background()

	ct := models.HRContractPartTime
	_, err := svc.UpdateEmployee(ctx, testEmployeeID, UpdateEmployeeInput{
		ContractType: &ct,
	}, "employee")
	assert.ErrorIs(t, err, ErrUnauthorizedFieldUpdate)
}

func TestUpdateEmployee_AdminAllFields(t *testing.T) {
	svc, _, _, _ := setupTestService()
	ctx := context.Background()

	dept := "Sales"
	pos := "Lead"
	ct := models.HRContractPartTime
	wdpw := 3
	ald := 25
	name := "Emergency Contact"

	result, err := svc.UpdateEmployee(ctx, testEmployeeID, UpdateEmployeeInput{
		Department:           &dept,
		PositionTitle:        &pos,
		ContractType:         &ct,
		WorkDaysPerWeek:      &wdpw,
		AnnualLeaveDays:      &ald,
		EmergencyContactName: &name,
	}, "admin")
	require.NoError(t, err)
	assert.Equal(t, "Sales", result.Department)
	assert.Equal(t, "Lead", result.PositionTitle)
	assert.Equal(t, models.HRContractPartTime, result.ContractType)
	assert.Equal(t, 3, result.WorkDaysPerWeek)
	assert.Equal(t, 25, result.AnnualLeaveDays)
	assert.Equal(t, "Emergency Contact", result.EmergencyContactName)
}

func TestUpdateEmployee_HRAllFields(t *testing.T) {
	svc, _, _, _ := setupTestService()
	ctx := context.Background()

	dept := "Operations"
	result, err := svc.UpdateEmployee(ctx, testEmployeeID, UpdateEmployeeInput{
		Department: &dept,
	}, "hr")
	require.NoError(t, err)
	assert.Equal(t, "Operations", result.Department)
}

func TestUpdateEmployee_ManagerAllFields(t *testing.T) {
	svc, _, _, _ := setupTestService()
	ctx := context.Background()

	pos := "Senior Developer"
	result, err := svc.UpdateEmployee(ctx, testEmployeeID, UpdateEmployeeInput{
		PositionTitle: &pos,
	}, "manager")
	require.NoError(t, err)
	assert.Equal(t, "Senior Developer", result.PositionTitle)
}

func TestUpdateSelfProfile(t *testing.T) {
	svc, _, _, _ := setupTestService()
	ctx := context.Background()

	street := "Musterstrasse 42"
	city := "Berlin"
	result, err := svc.UpdateSelfProfile(ctx, testUserID, SelfProfileInput{
		AddressStreet: &street,
		AddressCity:   &city,
	})
	require.NoError(t, err)
	assert.Equal(t, "Musterstrasse 42", result.AddressStreet)
	assert.Equal(t, "Berlin", result.AddressCity)
}

// ============================================================================
// Tests: Document Access
// ============================================================================

func TestListEmployeeDocuments_EmployeeVisibility(t *testing.T) {
	svc, _, _, docRepo := setupTestService()
	ctx := context.Background()

	// Add documents with different categories
	docRepo.documents = []*models.EmployeeDocument{
		{
			ID:           uuid.New(),
			EmployeeID:   testUserID,
			CategoryName: "Arbeitsvertrag", // hr_only
		},
		{
			ID:           uuid.New(),
			EmployeeID:   testUserID,
			CategoryName: "Zeugnisse", // manager
		},
		{
			ID:           uuid.New(),
			EmployeeID:   testUserID,
			CategoryName: "Sonstiges", // employee
		},
	}

	// Employee sees only employee-visible docs
	docs, err := svc.ListEmployeeDocuments(ctx, testUserID, testUserID, "employee")
	require.NoError(t, err)
	assert.Len(t, docs, 1)
	assert.Equal(t, "Sonstiges", docs[0].CategoryName)
}

func TestListEmployeeDocuments_ManagerVisibility(t *testing.T) {
	svc, _, _, docRepo := setupTestService()
	ctx := context.Background()

	docRepo.documents = []*models.EmployeeDocument{
		{
			ID:           uuid.New(),
			EmployeeID:   testUserID,
			CategoryName: "Arbeitsvertrag", // hr_only
		},
		{
			ID:           uuid.New(),
			EmployeeID:   testUserID,
			CategoryName: "Zeugnisse", // manager
		},
		{
			ID:           uuid.New(),
			EmployeeID:   testUserID,
			CategoryName: "Sonstiges", // employee
		},
	}

	// Manager sees manager + employee visible docs
	docs, err := svc.ListEmployeeDocuments(ctx, testUserID, testManagerUserID, "manager")
	require.NoError(t, err)
	assert.Len(t, docs, 2)
}

func TestListEmployeeDocuments_HRVisibility(t *testing.T) {
	svc, _, _, docRepo := setupTestService()
	ctx := context.Background()

	docRepo.documents = []*models.EmployeeDocument{
		{
			ID:           uuid.New(),
			EmployeeID:   testUserID,
			CategoryName: "Arbeitsvertrag", // hr_only
		},
		{
			ID:           uuid.New(),
			EmployeeID:   testUserID,
			CategoryName: "Zeugnisse", // manager
		},
		{
			ID:           uuid.New(),
			EmployeeID:   testUserID,
			CategoryName: "Sonstiges", // employee
		},
	}

	// HR sees all
	docs, err := svc.ListEmployeeDocuments(ctx, testUserID, uuid.New(), "hr")
	require.NoError(t, err)
	assert.Len(t, docs, 3)
}

func TestListEmployeeDocuments_AdminVisibility(t *testing.T) {
	svc, _, _, docRepo := setupTestService()
	ctx := context.Background()

	docRepo.documents = []*models.EmployeeDocument{
		{
			ID:           uuid.New(),
			EmployeeID:   testUserID,
			CategoryName: "Arbeitsvertrag",
		},
		{
			ID:           uuid.New(),
			EmployeeID:   testUserID,
			CategoryName: "Abmahnungen",
		},
		{
			ID:           uuid.New(),
			EmployeeID:   testUserID,
			CategoryName: "Sonstiges",
		},
	}

	// Admin sees all
	docs, err := svc.ListEmployeeDocuments(ctx, testUserID, uuid.New(), "admin")
	require.NoError(t, err)
	assert.Len(t, docs, 3)
}

// ============================================================================
// Tests: Document Upload
// ============================================================================

func TestUploadEmployeeDocument_Success(t *testing.T) {
	svc, _, _, _ := setupTestService()
	ctx := context.Background()

	doc, err := svc.UploadEmployeeDocument(ctx, testTenantID, UploadDocumentInput{
		EmployeeID: testUserID,
		CategoryID: testCatID,
		FileID:     uuid.New(),
		UploadedBy: testManagerUserID,
		Notes:      "Neuer Vertrag",
	})
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, doc.ID)
	assert.Equal(t, testUserID, doc.EmployeeID)
	assert.Equal(t, testCatID, doc.CategoryID)
	assert.Equal(t, "Neuer Vertrag", doc.Notes)
}

func TestUploadEmployeeDocument_InvalidCategory(t *testing.T) {
	svc, _, _, _ := setupTestService()
	ctx := context.Background()

	_, err := svc.UploadEmployeeDocument(ctx, testTenantID, UploadDocumentInput{
		EmployeeID: testUserID,
		CategoryID: uuid.New(), // Non-existent
		FileID:     uuid.New(),
		UploadedBy: testManagerUserID,
	})
	assert.ErrorIs(t, err, ErrDocumentCategoryNotFound)
}

// ============================================================================
// Tests: Document Category Visibility (fix-hr-document-paths)
// ============================================================================

// seedThreeVisibilityCategories adds one category per visibility tier to
// catRepo, all belonging to testTenantID, so the scope filter has something
// of each kind to include or exclude.
func seedThreeVisibilityCategories(catRepo *mockDocCategoryRepo) {
	catRepo.categories[uuid.New()] = &models.HRDocumentCategory{
		ID: uuid.New(), TenantID: testTenantID, Key: "hr_only_cat", Name: "Abmahnungen",
		Visibility: models.HRDocVisibilityHROnly,
	}
	catRepo.categories[uuid.New()] = &models.HRDocumentCategory{
		ID: uuid.New(), TenantID: testTenantID, Key: "manager_cat", Name: "Zeugnisse",
		Visibility: models.HRDocVisibilityManager,
	}
	catRepo.categories[uuid.New()] = &models.HRDocumentCategory{
		ID: uuid.New(), TenantID: testTenantID, Key: "employee_cat", Name: "Sonstiges",
		Visibility: models.HRDocVisibilityEmployee,
	}
}

func categoryKeys(cats []*models.HRDocumentCategory) []string {
	keys := make([]string, 0, len(cats))
	for _, c := range cats {
		keys = append(keys, c.Key)
	}
	return keys
}

func TestListDocumentCategories_ScopeAllSeesEverything(t *testing.T) {
	svc, _, catRepo, _ := setupTestService()
	seedThreeVisibilityCategories(catRepo)
	ctx := context.Background()

	cats, err := svc.ListDocumentCategories(ctx, testTenantID, auth.ScopeAll)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"arbeitsvertrag", "hr_only_cat", "manager_cat", "employee_cat"}, categoryKeys(cats))
}

func TestListDocumentCategories_ScopeTeamExcludesHROnly(t *testing.T) {
	svc, _, catRepo, _ := setupTestService()
	seedThreeVisibilityCategories(catRepo)
	ctx := context.Background()

	cats, err := svc.ListDocumentCategories(ctx, testTenantID, auth.ScopeTeam)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"manager_cat", "employee_cat"}, categoryKeys(cats))
}

// TestListDocumentCategories_ScopeOwnSeesOnlyEmployeeVisibility pins the
// "Fremdakte liefert keine hr_only-Kategorien" requirement: a caller whose
// team:documents:view grant is scope='own' (the member preset, migration
// 000256) never sees hr_only or manager-tier categories, regardless of whose
// employee id the request names.
func TestListDocumentCategories_ScopeOwnSeesOnlyEmployeeVisibility(t *testing.T) {
	svc, _, catRepo, _ := setupTestService()
	seedThreeVisibilityCategories(catRepo)
	ctx := context.Background()

	cats, err := svc.ListDocumentCategories(ctx, testTenantID, auth.ScopeOwn)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"employee_cat"}, categoryKeys(cats))
}

// TestListDocumentCategories_UnknownScopeDefaultsRestrictive pins fail-closed
// behavior: an empty or unrecognized scope string must not fall through to
// "all" — that would be the security-relevant default to get wrong.
func TestListDocumentCategories_UnknownScopeDefaultsRestrictive(t *testing.T) {
	svc, _, catRepo, _ := setupTestService()
	seedThreeVisibilityCategories(catRepo)
	ctx := context.Background()

	cats, err := svc.ListDocumentCategories(ctx, testTenantID, "")
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"employee_cat"}, categoryKeys(cats))
}
