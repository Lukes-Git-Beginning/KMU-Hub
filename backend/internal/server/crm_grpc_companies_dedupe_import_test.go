package server

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"
	"google.golang.org/grpc/codes"

	"github.com/kmuhub/kmuhub/internal/crm/company"
	"github.com/kmuhub/kmuhub/internal/crm/contact"
	emailcontact "github.com/kmuhub/kmuhub/internal/email/contact"
	"github.com/kmuhub/kmuhub/internal/models"
	crmv1 "github.com/kmuhub/kmuhub/proto/crm/v1"
)

// ============================================================================
// Stub repository (server package copy — company.MockRepository lives in the
// internal/crm/company package's own _test.go file and isn't importable from
// here; same pattern as stubContactRepo in crm_grpc_fields_tags_contacts_test.go).
// ============================================================================

type stubCompanyRepo struct {
	companies   map[uuid.UUID]*models.Company
	hasContacts map[uuid.UUID]bool
	validTags   map[uuid.UUID]models.EntityType
	duplicates  map[uuid.UUID][]*company.DuplicateCandidate
}

func newStubCompanyRepo() *stubCompanyRepo {
	return &stubCompanyRepo{
		companies:   make(map[uuid.UUID]*models.Company),
		hasContacts: make(map[uuid.UUID]bool),
		validTags:   make(map[uuid.UUID]models.EntityType),
		duplicates:  make(map[uuid.UUID][]*company.DuplicateCandidate),
	}
}

func (r *stubCompanyRepo) Create(_ context.Context, c *models.Company) error {
	r.companies[c.ID] = c
	return nil
}

func (r *stubCompanyRepo) GetByID(_ context.Context, id, tenantID uuid.UUID) (*models.Company, error) {
	c, ok := r.companies[id]
	if !ok || c.TenantID != tenantID {
		return nil, company.ErrCompanyNotFound
	}
	return c, nil
}

func (r *stubCompanyRepo) GetByName(_ context.Context, tenantID uuid.UUID, name string) (*models.Company, error) {
	for _, c := range r.companies {
		if c.TenantID == tenantID && c.Name == name {
			return c, nil
		}
	}
	return nil, company.ErrCompanyNotFound
}

func (r *stubCompanyRepo) GetNamesByIDs(_ context.Context, ids []uuid.UUID, tenantID uuid.UUID) (map[uuid.UUID]string, error) {
	result := make(map[uuid.UUID]string)
	for _, id := range ids {
		if c, ok := r.companies[id]; ok && c.TenantID == tenantID {
			result[id] = c.Name
		}
	}
	return result, nil
}

func (r *stubCompanyRepo) List(_ context.Context, filter company.ListFilter, offset, limit int) ([]*models.Company, int, error) {
	var result []*models.Company
	for _, c := range r.companies {
		if c.TenantID != filter.TenantID {
			continue
		}
		if len(filter.TagIDs) > 0 {
			continue // stub does not join tags; unused by any test in this file
		}
		result = append(result, c)
	}
	total := len(result)
	if offset >= total {
		return []*models.Company{}, total, nil
	}
	end := min(offset+limit, total)
	return result[offset:end], total, nil
}

func (r *stubCompanyRepo) Update(_ context.Context, c *models.Company, tenantID uuid.UUID) error {
	existing, ok := r.companies[c.ID]
	if !ok || existing.TenantID != tenantID {
		return company.ErrCompanyNotFound
	}
	r.companies[c.ID] = c
	return nil
}

func (r *stubCompanyRepo) Delete(_ context.Context, id, tenantID uuid.UUID) error {
	c, ok := r.companies[id]
	if !ok || c.TenantID != tenantID {
		return company.ErrCompanyNotFound
	}
	delete(r.companies, id)
	return nil
}

func (r *stubCompanyRepo) GetContactCount(_ context.Context, _, _ uuid.UUID) (int, error) {
	return 0, nil
}

func (r *stubCompanyRepo) GetTags(_ context.Context, _ uuid.UUID) ([]*models.Tag, error) {
	return nil, nil
}

func (r *stubCompanyRepo) AddTags(_ context.Context, _ uuid.UUID, _ []uuid.UUID) error { return nil }

func (r *stubCompanyRepo) RemoveTags(_ context.Context, _ uuid.UUID, _ []uuid.UUID) error {
	return nil
}

func (r *stubCompanyRepo) GetCustomFieldValues(_ context.Context, _ uuid.UUID) ([]*models.CustomFieldValueRow, error) {
	return nil, nil
}

func (r *stubCompanyRepo) SetCustomFieldValues(_ context.Context, _ uuid.UUID, _ map[uuid.UUID]any) error {
	return nil
}

func (r *stubCompanyRepo) HasContacts(_ context.Context, id, _ uuid.UUID) (bool, error) {
	return r.hasContacts[id], nil
}

func (r *stubCompanyRepo) TagExists(_ context.Context, tagID uuid.UUID, entityType models.EntityType) (bool, error) {
	et, ok := r.validTags[tagID]
	return ok && et == entityType, nil
}

func (r *stubCompanyRepo) FindDuplicateCandidates(_ context.Context, id, _ uuid.UUID) ([]*company.DuplicateCandidate, error) {
	return r.duplicates[id], nil
}

func (r *stubCompanyRepo) MergeInto(_ context.Context, primaryID, duplicateID, _ uuid.UUID) error {
	if dup, ok := r.companies[duplicateID]; ok {
		mergedInto := primaryID
		dup.MergedIntoID = &mergedInto
	}
	return nil
}

// ============================================================================
// Test server constructors
// ============================================================================

func newCRMServerWithCompanyRepo(repo company.Repository) *CRMGRPCServer {
	return &CRMGRPCServer{companyService: company.NewService(repo)}
}

func newCRMServerWithContactAndCompanyRepo(contactRepo contact.Repository, companyRepo company.Repository) *CRMGRPCServer {
	return &CRMGRPCServer{
		contactService: contact.NewService(contactRepo),
		companyService: company.NewService(companyRepo),
	}
}

// newCRMServerWithImportExport wires real ImportService/ExportService instances
// so the handler's "not configured" nil-check passes. The actual import/export
// work happens through a locally-built service inside the handler
// (emailcontact.NewTenantScopedAdapter over contactService/companyService,
// see ImportContactsCSV et al. in crm_grpc.go) - the injected services here are
// only consulted for PreviewImportCSV and the Unimplemented gate, so a nil
// ContactProvider is safe.
func newCRMServerWithImportExport(contactRepo contact.Repository, companyRepo company.Repository) *CRMGRPCServer {
	srv := newCRMServerWithContactAndCompanyRepo(contactRepo, companyRepo)
	srv.SetImportExportServices(emailcontact.NewImportService(nil, nil), emailcontact.NewExportService(nil, nil), nil)
	return srv
}

func seedCompany(repo *stubCompanyRepo, tenantID uuid.UUID) *models.Company {
	c := &models.Company{
		ID:       uuid.New(),
		Name:     "Acme GmbH",
		TenantID: tenantID,
	}
	repo.companies[c.ID] = c
	return c
}

// ============================================================================
// Companies: CreateCompany, GetCompany, ListCompanies, UpdateCompany,
// DeleteCompany, GetCompanyContacts
// ============================================================================

func TestCreateCompany_MissingTenant(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.CreateCompany(context.Background(), &crmv1.CreateCompanyRequest{
		Name:      "Acme GmbH",
		CreatedBy: uuid.New().String(),
	})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestCreateCompany_InvalidTagID(t *testing.T) {
	repo := newStubCompanyRepo()
	srv := newCRMServerWithCompanyRepo(repo)
	_, err := srv.CreateCompany(ctxWithTenant(uuid.New()), &crmv1.CreateCompanyRequest{
		Name:      "Acme GmbH",
		CreatedBy: uuid.New().String(),
		TagIds:    []string{"not-a-uuid"},
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestCreateCompany_HappyPath(t *testing.T) {
	repo := newStubCompanyRepo()
	srv := newCRMServerWithCompanyRepo(repo)
	resp, err := srv.CreateCompany(ctxWithTenant(uuid.New()), &crmv1.CreateCompanyRequest{
		Name:      "Acme GmbH",
		CreatedBy: uuid.New().String(),
	})
	requireGRPCOK(t, err)
	if resp.Company == nil {
		t.Fatal("expected company in response")
	}
	if resp.Company.Name != "Acme GmbH" {
		t.Errorf("name mismatch: got %s", resp.Company.Name)
	}
}

func TestGetCompany_MissingTenant(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.GetCompany(context.Background(), &crmv1.GetCompanyRequest{Id: uuid.New().String()})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestGetCompany_InvalidID(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.GetCompany(ctxWithTenant(uuid.New()), &crmv1.GetCompanyRequest{Id: "bad-id"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestGetCompany_NotFound(t *testing.T) {
	repo := newStubCompanyRepo()
	srv := newCRMServerWithCompanyRepo(repo)
	_, err := srv.GetCompany(ctxWithTenant(uuid.New()), &crmv1.GetCompanyRequest{Id: uuid.New().String()})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestGetCompany_HappyPath(t *testing.T) {
	repo := newStubCompanyRepo()
	srv := newCRMServerWithCompanyRepo(repo)
	tenantID := uuid.New()
	c := seedCompany(repo, tenantID)

	resp, err := srv.GetCompany(ctxWithTenant(tenantID), &crmv1.GetCompanyRequest{Id: c.ID.String()})
	requireGRPCOK(t, err)
	if resp.Company.Id != c.ID.String() {
		t.Errorf("id mismatch: got %s", resp.Company.Id)
	}
}

func TestListCompanies_MissingTenant(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.ListCompanies(context.Background(), &crmv1.ListCompaniesRequest{})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestListCompanies_InvalidTagID(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.ListCompanies(ctxWithTenant(uuid.New()), &crmv1.ListCompaniesRequest{TagIds: []string{"not-a-uuid"}})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestListCompanies_HappyPath(t *testing.T) {
	repo := newStubCompanyRepo()
	srv := newCRMServerWithCompanyRepo(repo)
	tenantID := uuid.New()
	seedCompany(repo, tenantID)
	seedCompany(repo, uuid.New()) // other tenant, must not leak into the result

	resp, err := srv.ListCompanies(ctxWithTenant(tenantID), &crmv1.ListCompaniesRequest{PageSize: 20})
	requireGRPCOK(t, err)
	if resp.Total != 1 {
		t.Errorf("expected total 1, got %d", resp.Total)
	}
	if len(resp.Companies) != 1 {
		t.Errorf("expected 1 company, got %d", len(resp.Companies))
	}
}

// TestListCompanies_EmptyIsNilNotEmptySlice documents the wire-shape fix
// applied alongside fix-crm-list-nil-slice-wire-shape (Block A): the handler
// (crm_grpc.go) now pre-allocates `infos` with make(..., 0, ...), so an empty
// result serializes to `[]` rather than `null`.
func TestListCompanies_EmptyIsNilNotEmptySlice(t *testing.T) {
	repo := newStubCompanyRepo()
	srv := newCRMServerWithCompanyRepo(repo)

	resp, err := srv.ListCompanies(ctxWithTenant(uuid.New()), &crmv1.ListCompaniesRequest{PageSize: 20})
	requireGRPCOK(t, err)
	if resp.Companies == nil {
		t.Error("Companies should be an empty slice, not nil, so it serializes to [] rather than null")
	}
	if len(resp.Companies) != 0 {
		t.Errorf("expected 0 companies, got %d", len(resp.Companies))
	}
}

func TestUpdateCompany_MissingTenant(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.UpdateCompany(context.Background(), &crmv1.UpdateCompanyRequest{Id: uuid.New().String()})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestUpdateCompany_InvalidID(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.UpdateCompany(ctxWithTenant(uuid.New()), &crmv1.UpdateCompanyRequest{Id: "bad-id"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestUpdateCompany_NotFound(t *testing.T) {
	repo := newStubCompanyRepo()
	srv := newCRMServerWithCompanyRepo(repo)
	_, err := srv.UpdateCompany(ctxWithTenant(uuid.New()), &crmv1.UpdateCompanyRequest{Id: uuid.New().String()})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestUpdateCompany_BlankNameRejected(t *testing.T) {
	repo := newStubCompanyRepo()
	srv := newCRMServerWithCompanyRepo(repo)
	tenantID := uuid.New()
	c := seedCompany(repo, tenantID)
	blank := "   "

	_, err := srv.UpdateCompany(ctxWithTenant(tenantID), &crmv1.UpdateCompanyRequest{Id: c.ID.String(), Name: &blank})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestUpdateCompany_HappyPath(t *testing.T) {
	repo := newStubCompanyRepo()
	srv := newCRMServerWithCompanyRepo(repo)
	tenantID := uuid.New()
	c := seedCompany(repo, tenantID)
	newName := "Acme AG"

	resp, err := srv.UpdateCompany(ctxWithTenant(tenantID), &crmv1.UpdateCompanyRequest{Id: c.ID.String(), Name: &newName})
	requireGRPCOK(t, err)
	if resp.Company.Name != "Acme AG" {
		t.Errorf("name mismatch: got %s", resp.Company.Name)
	}
}

func TestDeleteCompany_MissingTenant(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.DeleteCompany(context.Background(), &crmv1.DeleteCompanyRequest{Id: uuid.New().String()})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestDeleteCompany_InvalidID(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.DeleteCompany(ctxWithTenant(uuid.New()), &crmv1.DeleteCompanyRequest{Id: "bad-id"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestDeleteCompany_InUse(t *testing.T) {
	repo := newStubCompanyRepo()
	srv := newCRMServerWithCompanyRepo(repo)
	tenantID := uuid.New()
	c := seedCompany(repo, tenantID)
	repo.hasContacts[c.ID] = true

	_, err := srv.DeleteCompany(ctxWithTenant(tenantID), &crmv1.DeleteCompanyRequest{Id: c.ID.String()})
	requireGRPCCode(t, err, codes.FailedPrecondition)
}

func TestDeleteCompany_HappyPath(t *testing.T) {
	repo := newStubCompanyRepo()
	srv := newCRMServerWithCompanyRepo(repo)
	tenantID := uuid.New()
	c := seedCompany(repo, tenantID)

	_, err := srv.DeleteCompany(ctxWithTenant(tenantID), &crmv1.DeleteCompanyRequest{Id: c.ID.String()})
	requireGRPCOK(t, err)
	if _, ok := repo.companies[c.ID]; ok {
		t.Error("expected company to be deleted from the repo")
	}
}

func TestGetCompanyContacts_MissingTenant(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.GetCompanyContacts(context.Background(), &crmv1.GetCompanyContactsRequest{CompanyId: uuid.New().String()})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestGetCompanyContacts_InvalidCompanyID(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.GetCompanyContacts(ctxWithTenant(uuid.New()), &crmv1.GetCompanyContactsRequest{CompanyId: "bad-id"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestGetCompanyContacts_HappyPath(t *testing.T) {
	contactRepo := newStubContactRepo()
	tenantID := uuid.New()
	companyID := uuid.New()
	c := &models.Contact{ID: uuid.New(), FirstName: "Jane", LastName: "Doe", CompanyID: &companyID, TenantID: tenantID}
	contactRepo.contacts[c.ID] = c
	srv := newCRMServerWithContactRepo(contactRepo)

	resp, err := srv.GetCompanyContacts(ctxWithTenant(tenantID), &crmv1.GetCompanyContactsRequest{CompanyId: companyID.String(), PageSize: 20})
	requireGRPCOK(t, err)
	if resp.Total != 1 {
		t.Errorf("expected total 1, got %d", resp.Total)
	}
}

// TestGetCompanyContacts_EmptyIsNilNotEmptySlice documents the wire-shape fix
// applied alongside fix-crm-list-nil-slice-wire-shape (Block A): the handler
// (crm_grpc.go) now pre-allocates `infos` with make(..., 0, ...), so a
// company with zero contacts serializes to `[]` rather than `null`.
func TestGetCompanyContacts_EmptyIsNilNotEmptySlice(t *testing.T) {
	repo := newStubContactRepo()
	srv := newCRMServerWithContactRepo(repo)

	resp, err := srv.GetCompanyContacts(ctxWithTenant(uuid.New()), &crmv1.GetCompanyContactsRequest{CompanyId: uuid.New().String(), PageSize: 20})
	requireGRPCOK(t, err)
	if resp.Contacts == nil {
		t.Error("Contacts should be an empty slice, not nil, so it serializes to [] rather than null")
	}
	if len(resp.Contacts) != 0 {
		t.Errorf("expected 0 contacts, got %d", len(resp.Contacts))
	}
}

// ============================================================================
// UpdateContactVisibility
// ============================================================================

func TestUpdateContactVisibility_MissingTenant(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.UpdateContactVisibility(context.Background(), &crmv1.UpdateContactVisibilityRequest{
		ContactId: uuid.New().String(),
		UserId:    uuid.New().String(),
	})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestUpdateContactVisibility_InvalidContactID(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.UpdateContactVisibility(ctxWithTenant(uuid.New()), &crmv1.UpdateContactVisibilityRequest{
		ContactId: "bad-id",
		UserId:    uuid.New().String(),
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestUpdateContactVisibility_InvalidUserID(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.UpdateContactVisibility(ctxWithTenant(uuid.New()), &crmv1.UpdateContactVisibilityRequest{
		ContactId: uuid.New().String(),
		UserId:    "bad-id",
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

// TestUpdateContactVisibility_FallbackHappyPath exercises the visibilityService==nil
// branch (direct contactService.UpdateVisibility call) - the handler-constructed test
// server never wires visibilityService, same as production before Welle 1b enabled it
// for a given tenant.
func TestUpdateContactVisibility_FallbackHappyPath(t *testing.T) {
	repo := newStubContactRepo()
	tenantID := uuid.New()
	c := &models.Contact{ID: uuid.New(), FirstName: "Jane", LastName: "Doe", Visibility: "shared", TenantID: tenantID}
	repo.contacts[c.ID] = c
	srv := newCRMServerWithContactRepo(repo)
	userID := uuid.New()

	resp, err := srv.UpdateContactVisibility(ctxWithTenant(tenantID), &crmv1.UpdateContactVisibilityRequest{
		ContactId:  c.ID.String(),
		UserId:     userID.String(),
		Visibility: "personal",
	})
	requireGRPCOK(t, err)
	if resp.Contact.Id != c.ID.String() {
		t.Errorf("id mismatch: got %s", resp.Contact.Id)
	}
	if repo.contacts[c.ID].Visibility != "personal" {
		t.Errorf("expected visibility to be updated to personal, got %s", repo.contacts[c.ID].Visibility)
	}
}

// ============================================================================
// Duplicate detection and merge: FindContactDuplicates, MergeContacts,
// FindCompanyDuplicates, MergeCompanies
// ============================================================================

func TestFindContactDuplicates_MissingTenant(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.FindContactDuplicates(context.Background(), &crmv1.FindContactDuplicatesRequest{ContactId: uuid.New().String()})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestFindContactDuplicates_InvalidContactID(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.FindContactDuplicates(ctxWithTenant(uuid.New()), &crmv1.FindContactDuplicatesRequest{ContactId: "bad-id"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestFindContactDuplicates_HappyPath(t *testing.T) {
	repo := newStubContactRepo()
	tenantID := uuid.New()
	c := &models.Contact{ID: uuid.New(), FirstName: "Jane", LastName: "Doe", TenantID: tenantID}
	dup := &models.Contact{ID: uuid.New(), FirstName: "Jane", LastName: "Doe", TenantID: tenantID}
	repo.contacts[c.ID] = c
	repo.contacts[dup.ID] = dup
	srv := newCRMServerWithContactRepo(repo)

	resp, err := srv.FindContactDuplicates(ctxWithTenant(tenantID), &crmv1.FindContactDuplicatesRequest{ContactId: c.ID.String()})
	requireGRPCOK(t, err)
	if resp.Total != 0 {
		t.Errorf("expected 0 (stub repo returns no candidates), got %d", resp.Total)
	}
}

func TestMergeContacts_MissingTenant(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.MergeContacts(context.Background(), &crmv1.MergeContactsRequest{
		PrimaryId:   uuid.New().String(),
		DuplicateId: uuid.New().String(),
	})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestMergeContacts_InvalidPrimaryID(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.MergeContacts(ctxWithTenant(uuid.New()), &crmv1.MergeContactsRequest{
		PrimaryId:   "bad-id",
		DuplicateId: uuid.New().String(),
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestMergeContacts_InvalidDuplicateID(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.MergeContacts(ctxWithTenant(uuid.New()), &crmv1.MergeContactsRequest{
		PrimaryId:   uuid.New().String(),
		DuplicateId: "bad-id",
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestMergeContacts_SelfMergeRejected(t *testing.T) {
	srv := newTestCRMServer()
	id := uuid.New().String()
	_, err := srv.MergeContacts(ctxWithTenant(uuid.New()), &crmv1.MergeContactsRequest{
		PrimaryId:   id,
		DuplicateId: id,
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestMergeContacts_AlreadyMerged(t *testing.T) {
	repo := newStubContactRepo()
	tenantID := uuid.New()
	primary := &models.Contact{ID: uuid.New(), FirstName: "A", LastName: "A", TenantID: tenantID}
	mergedInto := uuid.New()
	dup := &models.Contact{ID: uuid.New(), FirstName: "B", LastName: "B", TenantID: tenantID, MergedIntoID: &mergedInto}
	repo.contacts[primary.ID] = primary
	repo.contacts[dup.ID] = dup
	srv := newCRMServerWithContactRepo(repo)

	_, err := srv.MergeContacts(ctxWithTenant(tenantID), &crmv1.MergeContactsRequest{
		PrimaryId:   primary.ID.String(),
		DuplicateId: dup.ID.String(),
	})
	requireGRPCCode(t, err, codes.FailedPrecondition)
}

func TestMergeContacts_HappyPath(t *testing.T) {
	repo := newStubContactRepo()
	tenantID := uuid.New()
	primary := &models.Contact{ID: uuid.New(), FirstName: "A", LastName: "A", TenantID: tenantID}
	dup := &models.Contact{ID: uuid.New(), FirstName: "B", LastName: "B", TenantID: tenantID}
	repo.contacts[primary.ID] = primary
	repo.contacts[dup.ID] = dup
	srv := newCRMServerWithContactRepo(repo)

	resp, err := srv.MergeContacts(ctxWithTenant(tenantID), &crmv1.MergeContactsRequest{
		PrimaryId:   primary.ID.String(),
		DuplicateId: dup.ID.String(),
	})
	requireGRPCOK(t, err)
	if resp.Contact.Id != primary.ID.String() {
		t.Errorf("expected primary contact in response, got %s", resp.Contact.Id)
	}
	if repo.contacts[dup.ID].MergedIntoID == nil || *repo.contacts[dup.ID].MergedIntoID != primary.ID {
		t.Error("expected duplicate to be marked merged into primary")
	}
}

func TestFindCompanyDuplicates_MissingTenant(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.FindCompanyDuplicates(context.Background(), &crmv1.FindCompanyDuplicatesRequest{CompanyId: uuid.New().String()})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestFindCompanyDuplicates_InvalidCompanyID(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.FindCompanyDuplicates(ctxWithTenant(uuid.New()), &crmv1.FindCompanyDuplicatesRequest{CompanyId: "bad-id"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestFindCompanyDuplicates_HappyPath(t *testing.T) {
	repo := newStubCompanyRepo()
	tenantID := uuid.New()
	c := seedCompany(repo, tenantID)
	srv := newCRMServerWithCompanyRepo(repo)

	resp, err := srv.FindCompanyDuplicates(ctxWithTenant(tenantID), &crmv1.FindCompanyDuplicatesRequest{CompanyId: c.ID.String()})
	requireGRPCOK(t, err)
	if resp.Total != 0 {
		t.Errorf("expected 0 (stub repo returns no candidates), got %d", resp.Total)
	}
}

func TestMergeCompanies_MissingTenant(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.MergeCompanies(context.Background(), &crmv1.MergeCompaniesRequest{
		PrimaryId:   uuid.New().String(),
		DuplicateId: uuid.New().String(),
	})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestMergeCompanies_InvalidPrimaryID(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.MergeCompanies(ctxWithTenant(uuid.New()), &crmv1.MergeCompaniesRequest{
		PrimaryId:   "bad-id",
		DuplicateId: uuid.New().String(),
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestMergeCompanies_InvalidDuplicateID(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.MergeCompanies(ctxWithTenant(uuid.New()), &crmv1.MergeCompaniesRequest{
		PrimaryId:   uuid.New().String(),
		DuplicateId: "bad-id",
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestMergeCompanies_SelfMergeRejected(t *testing.T) {
	srv := newTestCRMServer()
	id := uuid.New().String()
	_, err := srv.MergeCompanies(ctxWithTenant(uuid.New()), &crmv1.MergeCompaniesRequest{
		PrimaryId:   id,
		DuplicateId: id,
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestMergeCompanies_AlreadyMerged(t *testing.T) {
	repo := newStubCompanyRepo()
	tenantID := uuid.New()
	primary := seedCompany(repo, tenantID)
	dup := seedCompany(repo, tenantID)
	mergedInto := uuid.New()
	dup.MergedIntoID = &mergedInto
	srv := newCRMServerWithCompanyRepo(repo)

	_, err := srv.MergeCompanies(ctxWithTenant(tenantID), &crmv1.MergeCompaniesRequest{
		PrimaryId:   primary.ID.String(),
		DuplicateId: dup.ID.String(),
	})
	requireGRPCCode(t, err, codes.FailedPrecondition)
}

func TestMergeCompanies_HappyPath(t *testing.T) {
	repo := newStubCompanyRepo()
	tenantID := uuid.New()
	primary := seedCompany(repo, tenantID)
	dup := seedCompany(repo, tenantID)
	srv := newCRMServerWithCompanyRepo(repo)

	resp, err := srv.MergeCompanies(ctxWithTenant(tenantID), &crmv1.MergeCompaniesRequest{
		PrimaryId:   primary.ID.String(),
		DuplicateId: dup.ID.String(),
	})
	requireGRPCOK(t, err)
	if resp.Company.Id != primary.ID.String() {
		t.Errorf("expected primary company in response, got %s", resp.Company.Id)
	}
	if repo.companies[dup.ID].MergedIntoID == nil || *repo.companies[dup.ID].MergedIntoID != primary.ID {
		t.Error("expected duplicate to be marked merged into primary")
	}
}

// ============================================================================
// Import/Export: ImportContactsCSV, ImportContactsVCard, ImportContactsXLSX,
// ExportContactsCSV, ExportContactsVCard, PreviewImportCSV
// ============================================================================

func TestImportContactsCSV_NotConfigured(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.ImportContactsCSV(ctxWithTenant(uuid.New()), &crmv1.ImportContactsCSVRequest{UserId: uuid.New().String()})
	requireGRPCCode(t, err, codes.Unimplemented)
}

func TestImportContactsCSV_InvalidUserID(t *testing.T) {
	srv := newCRMServerWithImportExport(newStubContactRepo(), newStubCompanyRepo())
	_, err := srv.ImportContactsCSV(ctxWithTenant(uuid.New()), &crmv1.ImportContactsCSVRequest{UserId: "bad-id"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestImportContactsCSV_HappyPath(t *testing.T) {
	srv := newCRMServerWithImportExport(newStubContactRepo(), newStubCompanyRepo())
	csvContent := "email,vorname,nachname\ntest@example.com,John,Doe\n"

	resp, err := srv.ImportContactsCSV(ctxWithTenant(uuid.New()), &crmv1.ImportContactsCSVRequest{
		FileContent:  []byte(csvContent),
		FieldMapping: map[string]string{"email": "email", "vorname": "first_name", "nachname": "last_name"},
		Visibility:   "shared",
		UserId:       uuid.New().String(),
	})
	requireGRPCOK(t, err)
	if resp.ImportedCount != 1 {
		t.Errorf("expected 1 imported contact, got %d", resp.ImportedCount)
	}
}

func TestImportContactsVCard_NotConfigured(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.ImportContactsVCard(ctxWithTenant(uuid.New()), &crmv1.ImportContactsVCardRequest{UserId: uuid.New().String()})
	requireGRPCCode(t, err, codes.Unimplemented)
}

func TestImportContactsVCard_HappyPath(t *testing.T) {
	srv := newCRMServerWithImportExport(newStubContactRepo(), newStubCompanyRepo())
	vcardContent := "BEGIN:VCARD\r\nVERSION:3.0\r\nN:Doe;John;;;\r\nFN:John Doe\r\nEMAIL:john@example.com\r\nEND:VCARD\r\n"

	resp, err := srv.ImportContactsVCard(ctxWithTenant(uuid.New()), &crmv1.ImportContactsVCardRequest{
		FileContent: []byte(vcardContent),
		Visibility:  "shared",
		UserId:      uuid.New().String(),
	})
	requireGRPCOK(t, err)
	if resp.ImportedCount != 1 {
		t.Errorf("expected 1 imported contact, got %d", resp.ImportedCount)
	}
}

func TestImportContactsXLSX_NotConfigured(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.ImportContactsXLSX(ctxWithTenant(uuid.New()), &crmv1.ImportContactsXLSXRequest{UserId: uuid.New().String()})
	requireGRPCCode(t, err, codes.Unimplemented)
}

func TestImportContactsXLSX_InvalidWorkbook(t *testing.T) {
	srv := newCRMServerWithImportExport(newStubContactRepo(), newStubCompanyRepo())
	_, err := srv.ImportContactsXLSX(ctxWithTenant(uuid.New()), &crmv1.ImportContactsXLSXRequest{
		FileContent: []byte("this is not a valid xlsx workbook"),
		UserId:      uuid.New().String(),
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestImportContactsXLSX_HappyPath(t *testing.T) {
	srv := newCRMServerWithImportExport(newStubContactRepo(), newStubCompanyRepo())

	f := excelize.NewFile()
	defer f.Close() //nolint:errcheck // test-only, read after close not needed
	requireNoErr(t, f.SetCellValue("Sheet1", "A1", "email"))
	requireNoErr(t, f.SetCellValue("Sheet1", "B1", "vorname"))
	requireNoErr(t, f.SetCellValue("Sheet1", "C1", "nachname"))
	requireNoErr(t, f.SetCellValue("Sheet1", "A2", "test@example.com"))
	requireNoErr(t, f.SetCellValue("Sheet1", "B2", "John"))
	requireNoErr(t, f.SetCellValue("Sheet1", "C2", "Doe"))
	buf, err := f.WriteToBuffer()
	requireNoErr(t, err)

	resp, err := srv.ImportContactsXLSX(ctxWithTenant(uuid.New()), &crmv1.ImportContactsXLSXRequest{
		FileContent:  buf.Bytes(),
		FieldMapping: map[string]string{"email": "email", "vorname": "first_name", "nachname": "last_name"},
		Visibility:   "shared",
		UserId:       uuid.New().String(),
	})
	requireGRPCOK(t, err)
	if resp.ImportedCount != 1 {
		t.Errorf("expected 1 imported contact, got %d", resp.ImportedCount)
	}
}

func TestExportContactsCSV_NotConfigured(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.ExportContactsCSV(ctxWithTenant(uuid.New()), &crmv1.ExportContactsCSVRequest{})
	requireGRPCCode(t, err, codes.Unimplemented)
}

func TestExportContactsCSV_InvalidContactID(t *testing.T) {
	srv := newCRMServerWithImportExport(newStubContactRepo(), newStubCompanyRepo())
	_, err := srv.ExportContactsCSV(ctxWithTenant(uuid.New()), &crmv1.ExportContactsCSVRequest{ContactIds: []string{"bad-id"}})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestExportContactsCSV_HappyPathByIDs(t *testing.T) {
	contactRepo := newStubContactRepo()
	tenantID := uuid.New()
	email := "jane@example.com"
	c := &models.Contact{ID: uuid.New(), FirstName: "Jane", LastName: "Doe", Email: &email, TenantID: tenantID}
	contactRepo.contacts[c.ID] = c
	srv := newCRMServerWithImportExport(contactRepo, newStubCompanyRepo())

	resp, err := srv.ExportContactsCSV(ctxWithTenant(tenantID), &crmv1.ExportContactsCSVRequest{ContactIds: []string{c.ID.String()}})
	requireGRPCOK(t, err)
	if resp.ContentType != "text/csv" {
		t.Errorf("expected text/csv content type, got %s", resp.ContentType)
	}
	if len(resp.FileContent) == 0 {
		t.Error("expected non-empty CSV content")
	}
}

func TestExportContactsVCard_NotConfigured(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.ExportContactsVCard(ctxWithTenant(uuid.New()), &crmv1.ExportContactsVCardRequest{})
	requireGRPCCode(t, err, codes.Unimplemented)
}

func TestExportContactsVCard_InvalidContactID(t *testing.T) {
	srv := newCRMServerWithImportExport(newStubContactRepo(), newStubCompanyRepo())
	_, err := srv.ExportContactsVCard(ctxWithTenant(uuid.New()), &crmv1.ExportContactsVCardRequest{ContactIds: []string{"bad-id"}})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestExportContactsVCard_HappyPath(t *testing.T) {
	contactRepo := newStubContactRepo()
	tenantID := uuid.New()
	c := &models.Contact{ID: uuid.New(), FirstName: "Jane", LastName: "Doe", TenantID: tenantID}
	contactRepo.contacts[c.ID] = c
	srv := newCRMServerWithImportExport(contactRepo, newStubCompanyRepo())

	resp, err := srv.ExportContactsVCard(ctxWithTenant(tenantID), &crmv1.ExportContactsVCardRequest{ContactIds: []string{c.ID.String()}})
	requireGRPCOK(t, err)
	if resp.ContentType != "text/vcard" {
		t.Errorf("expected text/vcard content type, got %s", resp.ContentType)
	}
	if len(resp.FileContent) == 0 {
		t.Error("expected non-empty vCard content")
	}
}

func TestPreviewImportCSV_NotConfigured(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.PreviewImportCSV(ctxWithTenant(uuid.New()), &crmv1.PreviewImportCSVRequest{})
	requireGRPCCode(t, err, codes.Unimplemented)
}

func TestPreviewImportCSV_HappyPath(t *testing.T) {
	srv := newCRMServerWithImportExport(newStubContactRepo(), newStubCompanyRepo())
	csvContent := "E-Mail,Vorname,Nachname\ntest@example.com,John,Doe\n"

	resp, err := srv.PreviewImportCSV(ctxWithTenant(uuid.New()), &crmv1.PreviewImportCSVRequest{FileContent: []byte(csvContent)})
	requireGRPCOK(t, err)
	if len(resp.Columns) != 3 {
		t.Errorf("expected 3 columns, got %d", len(resp.Columns))
	}
	if resp.DetectedMapping["E-Mail"] != "email" {
		t.Errorf("expected E-Mail to auto-map to email, got %q", resp.DetectedMapping["E-Mail"])
	}
}

func requireNoErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
