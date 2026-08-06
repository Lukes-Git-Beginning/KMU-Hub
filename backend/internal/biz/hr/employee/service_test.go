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

	// Offboard fixtures. otherRoleAdmins is what the last-admin guard reads,
	// directReports what the successor guard reads.
	otherRoleAdmins int
	directReports   int
	offboardCalls   []OffboardWrite
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

func (m *mockEmployeeRepo) CountOtherActiveRoleAdmins(_ context.Context, _ uuid.UUID) (int, error) {
	return m.otherRoleAdmins, nil
}

func (m *mockEmployeeRepo) CountDirectReports(_ context.Context, _, _ uuid.UUID) (int, error) {
	return m.directReports, nil
}

func (m *mockEmployeeRepo) Offboard(_ context.Context, in OffboardWrite) (*models.EmployeeProfile, error) {
	m.offboardCalls = append(m.offboardCalls, in)
	p, ok := m.profiles[in.EmployeeID]
	if !ok {
		return nil, ErrEmployeeNotFound
	}
	p.Status = models.EmployeeStatusInactive
	p.ExitDate = &in.ExitDate
	p.ExitType = in.ExitType
	return p, nil
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

func (m *mockDocCategoryRepo) GetByKey(_ context.Context, tenantID uuid.UUID, key string) (*models.HRDocumentCategory, error) {
	// Mirrors the Postgres repo's ORDER BY (tenant_id = $2) DESC: the tenant's
	// own row wins over the system seed carrying the same key.
	var seed *models.HRDocumentCategory
	for _, c := range m.categories {
		if c.Key != key {
			continue
		}
		if c.TenantID == tenantID {
			return c, nil
		}
		if c.TenantID == systemSeedTenantID {
			seed = c
		}
	}
	if seed != nil {
		return seed, nil
	}
	return nil, ErrDocumentCategoryNotFound
}

type mockDocRepo struct {
	documents []*models.EmployeeDocument
}

func (m *mockDocRepo) ListByTenant(_ context.Context, tenantID uuid.UUID) ([]*models.EmployeeDocument, error) {
	results := make([]*models.EmployeeDocument, 0)
	for _, d := range m.documents {
		if d.TenantID == tenantID {
			results = append(results, d)
		}
	}
	return results, nil
}

func (m *mockDocRepo) GetByID(_ context.Context, tenantID, id uuid.UUID) (*models.EmployeeDocument, error) {
	for _, d := range m.documents {
		if d.ID == id && d.TenantID == tenantID {
			return d, nil
		}
	}
	return nil, ErrDocumentNotFound
}

func newMockDocRepo() *mockDocRepo {
	return &mockDocRepo{}
}

func (m *mockDocRepo) Create(_ context.Context, doc *models.EmployeeDocument) error {
	m.documents = append(m.documents, doc)
	return nil
}

// ListByEmployee does no visibility filtering, mirroring the real repository:
// the tiers live in RLS policy hr_document_access, which a mock cannot stand
// in for. TestListEmployeeDocuments_LeavesVisibilityToThePolicy pins that
// down; the tiers themselves are covered against the real policy in
// internal/biz/hr/hr_role_based_test.go.
func (m *mockDocRepo) ListByEmployee(_ context.Context, employeeID uuid.UUID) ([]*models.EmployeeDocument, error) {
	var results []*models.EmployeeDocument
	for _, d := range m.documents {
		if d.EmployeeID == employeeID {
			results = append(results, d)
		}
	}
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

// The four tests that used to live here asserted the visibility tiers against
// a mock that simulated them by category NAME -- they proved that the fake
// filtered, not that anything real does. The tiers are enforced by RLS policy
// hr_document_access and are covered against that policy, with actual roles on
// the session, in internal/biz/hr/hr_role_based_test.go. What is worth pinning
// down here is the boundary itself: this layer passes documents through and
// takes no role from its caller.
func TestListEmployeeDocuments_LeavesVisibilityToThePolicy(t *testing.T) {
	svc, _, _, docRepo := setupTestService()
	ctx := context.Background()

	other := uuid.New()
	docRepo.documents = []*models.EmployeeDocument{
		{ID: uuid.New(), EmployeeID: testUserID, CategoryName: "Arbeitsvertrag"}, // hr_only
		{ID: uuid.New(), EmployeeID: testUserID, CategoryName: "Zeugnisse"},      // manager
		{ID: uuid.New(), EmployeeID: testUserID, CategoryName: "Sonstiges"},      // employee
		{ID: uuid.New(), EmployeeID: other, CategoryName: "Sonstiges"},
	}

	docs, err := svc.ListEmployeeDocuments(ctx, testUserID)
	require.NoError(t, err)

	// Every tier comes through: the service applies no visibility filter of
	// its own, so a second one can never drift away from the policy.
	assert.Len(t, docs, 3)
	// The employee scoping is this layer's own job and still holds.
	for _, d := range docs {
		assert.Equal(t, testUserID, d.EmployeeID)
	}
}

// ============================================================================
// Tests: Document Upload
// ============================================================================

func TestUploadEmployeeDocument_Success(t *testing.T) {
	svc, _, _, _ := setupTestService()
	ctx := context.Background()

	fileID := uuid.New()
	doc, err := svc.UploadEmployeeDocument(ctx, testTenantID, UploadDocumentInput{
		EmployeeID: testUserID,
		CategoryID: testCatID,
		FileID:     &fileID,
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

	fileID := uuid.New()
	_, err := svc.UploadEmployeeDocument(ctx, testTenantID, UploadDocumentInput{
		EmployeeID: testUserID,
		CategoryID: uuid.New(), // Non-existent
		FileID:     &fileID,
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

// ============================================================================
// Tests: Personnel documents (hr-personnel-documents)
// ============================================================================

func TestUploadEmployeeDocument_ByCategoryKey(t *testing.T) {
	svc, _, _, docRepo := setupTestService()
	ctx := context.Background()

	expires := time.Date(2027, 3, 1, 0, 0, 0, 0, time.UTC)
	doc, err := svc.UploadEmployeeDocument(ctx, testTenantID, UploadDocumentInput{
		EmployeeID:  testUserID,
		CategoryKey: "arbeitsvertrag",
		UploadedBy:  testManagerUserID,
		Title:       "Arbeitsvertrag 2027",
		FileName:    "vertrag.pdf",
		FileSize:    "120 KB",
		ExpiresAt:   &expires,
	})
	require.NoError(t, err)

	// The slug must resolve to the category id, and the tier the category
	// carries must come back with the document — the UI gates on it.
	assert.Equal(t, testCatID, doc.CategoryID)
	assert.Equal(t, "arbeitsvertrag", doc.CategoryKey)
	assert.Equal(t, string(models.HRDocVisibilityHROnly), doc.Visibility)
	assert.Equal(t, "Arbeitsvertrag 2027", doc.Title)
	assert.Equal(t, "vertrag.pdf", doc.FileName)
	require.NotNil(t, doc.ExpiresAt)
	assert.Equal(t, expires, *doc.ExpiresAt)
	// Metadata-only: no file linked yet, which must not be a nil-UUID either.
	assert.Nil(t, doc.FileID)

	require.Len(t, docRepo.documents, 1)
	assert.Equal(t, doc.ID, docRepo.documents[0].ID)
}

func TestUploadEmployeeDocument_UnknownCategoryKey(t *testing.T) {
	svc, _, _, docRepo := setupTestService()

	_, err := svc.UploadEmployeeDocument(context.Background(), testTenantID, UploadDocumentInput{
		EmployeeID:  testUserID,
		CategoryKey: "gibt-es-nicht",
		UploadedBy:  testManagerUserID,
		Title:       "Irgendwas",
	})
	assert.ErrorIs(t, err, ErrDocumentCategoryNotFound)
	assert.Empty(t, docRepo.documents, "a rejected upload must not leave a row behind")
}

func TestUploadEmployeeDocument_NoCategoryAtAll(t *testing.T) {
	svc, _, _, _ := setupTestService()

	_, err := svc.UploadEmployeeDocument(context.Background(), testTenantID, UploadDocumentInput{
		EmployeeID: testUserID,
		UploadedBy: testManagerUserID,
		Title:      "Ohne Kategorie",
	})
	assert.ErrorIs(t, err, ErrDocumentCategoryNotFound)
}

// A document must never land in an unnamed personnel record: without an
// employee it would silently attach to the zero UUID.
func TestUploadEmployeeDocument_EmployeeRequired(t *testing.T) {
	svc, _, _, docRepo := setupTestService()

	_, err := svc.UploadEmployeeDocument(context.Background(), testTenantID, UploadDocumentInput{
		CategoryKey: "arbeitsvertrag",
		UploadedBy:  testManagerUserID,
		Title:       "Herrenlos",
	})
	assert.ErrorIs(t, err, ErrEmployeeRequired)
	assert.Empty(t, docRepo.documents)
}

// A foreign tenant's category id must not resolve, even though the caller may
// upload into their own tenant.
func TestUploadEmployeeDocument_ForeignTenantCategory(t *testing.T) {
	svc, _, catRepo, _ := setupTestService()

	foreignTenant := uuid.MustParse("10000000-0000-0000-0000-000000000009")
	foreignCat := uuid.MustParse("70000000-0000-0000-0000-000000000009")
	catRepo.categories[foreignCat] = &models.HRDocumentCategory{
		ID:         foreignCat,
		TenantID:   foreignTenant,
		Name:       "Fremd",
		Key:        "fremd",
		Visibility: models.HRDocVisibilityEmployee,
	}

	_, err := svc.UploadEmployeeDocument(context.Background(), testTenantID, UploadDocumentInput{
		EmployeeID: testUserID,
		CategoryID: foreignCat,
		UploadedBy: testManagerUserID,
		Title:      "Fremdes Dokument",
	})
	assert.ErrorIs(t, err, ErrDocumentCategoryNotFound)

	// …and neither does its slug.
	_, keyErr := svc.UploadEmployeeDocument(context.Background(), testTenantID, UploadDocumentInput{
		EmployeeID:  testUserID,
		CategoryKey: "fremd",
		UploadedBy:  testManagerUserID,
		Title:       "Fremdes Dokument",
	})
	assert.ErrorIs(t, keyErr, ErrDocumentCategoryNotFound)
}

func TestListPersonnelDocuments_TenantScoped(t *testing.T) {
	svc, _, _, docRepo := setupTestService()
	ctx := context.Background()

	foreignTenant := uuid.MustParse("10000000-0000-0000-0000-000000000009")
	docRepo.documents = append(docRepo.documents,
		&models.EmployeeDocument{ID: uuid.New(), TenantID: testTenantID, EmployeeID: testUserID, Title: "Eigen"},
		&models.EmployeeDocument{ID: uuid.New(), TenantID: foreignTenant, EmployeeID: uuid.New(), Title: "Fremd"},
	)

	docs, err := svc.ListPersonnelDocuments(ctx, testTenantID)
	require.NoError(t, err)
	require.Len(t, docs, 1)
	assert.Equal(t, "Eigen", docs[0].Title)
}

// An empty Akte must serialize as [], not null — the UI maps over the value.
func TestListPersonnelDocuments_EmptyIsSlice(t *testing.T) {
	svc, _, _, _ := setupTestService()

	docs, err := svc.ListPersonnelDocuments(context.Background(), testTenantID)
	require.NoError(t, err)
	assert.NotNil(t, docs)
	assert.Empty(t, docs)
}

func TestGetEmployeeDocument_ForeignTenantIsNotFound(t *testing.T) {
	svc, _, _, docRepo := setupTestService()

	foreignTenant := uuid.MustParse("10000000-0000-0000-0000-000000000009")
	foreignDoc := &models.EmployeeDocument{ID: uuid.New(), TenantID: foreignTenant, Title: "Fremd"}
	docRepo.documents = append(docRepo.documents, foreignDoc)

	// A guessed id from another tenant must be indistinguishable from an id
	// that does not exist at all.
	_, err := svc.GetEmployeeDocument(context.Background(), testTenantID, foreignDoc.ID)
	assert.ErrorIs(t, err, ErrDocumentNotFound)

	_, unknownErr := svc.GetEmployeeDocument(context.Background(), testTenantID, uuid.New())
	assert.ErrorIs(t, unknownErr, ErrDocumentNotFound)
}
