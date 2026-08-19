package server

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"

	"github.com/kmuhub/kmuhub/internal/crm/contact"
	"github.com/kmuhub/kmuhub/internal/crm/customfield"
	"github.com/kmuhub/kmuhub/internal/crm/tag"
	"github.com/kmuhub/kmuhub/internal/models"
	crmv1 "github.com/kmuhub/kmuhub/proto/crm/v1"
)

// ============================================================================
// Stub repositories (server package copies — the internal MockRepository
// types in customfield/tag/contact live in package-private _test.go files and
// can't be imported here; same pattern as stubFormulareRepo in
// formulare_grpc_test.go).
// ============================================================================

type stubCustomFieldRepo struct {
	fields      map[uuid.UUID]*models.CustomFieldDefinition
	fieldsByKey map[string]*models.CustomFieldDefinition
}

func newStubCustomFieldRepo() *stubCustomFieldRepo {
	return &stubCustomFieldRepo{
		fields:      make(map[uuid.UUID]*models.CustomFieldDefinition),
		fieldsByKey: make(map[string]*models.CustomFieldDefinition),
	}
}

func customFieldKey(tenantID uuid.UUID, entityType models.EntityType, fieldName string) string {
	return tenantID.String() + ":" + string(entityType) + ":" + fieldName
}

func (r *stubCustomFieldRepo) Create(_ context.Context, field *models.CustomFieldDefinition) error {
	r.fields[field.ID] = field
	r.fieldsByKey[customFieldKey(field.TenantID, field.EntityType, field.FieldName)] = field
	return nil
}

func (r *stubCustomFieldRepo) GetByID(_ context.Context, id, tenantID uuid.UUID) (*models.CustomFieldDefinition, error) {
	f, ok := r.fields[id]
	if !ok || f.TenantID != tenantID {
		return nil, customfield.ErrFieldNotFound
	}
	return f, nil
}

func (r *stubCustomFieldRepo) GetByEntityAndName(_ context.Context, tenantID uuid.UUID, entityType models.EntityType, fieldName string) (*models.CustomFieldDefinition, error) {
	f, ok := r.fieldsByKey[customFieldKey(tenantID, entityType, fieldName)]
	if !ok {
		return nil, customfield.ErrFieldNotFound
	}
	return f, nil
}

func (r *stubCustomFieldRepo) List(_ context.Context, tenantID uuid.UUID, entityType *models.EntityType, offset, limit int) ([]*models.CustomFieldDefinition, int, error) {
	var result []*models.CustomFieldDefinition
	for _, f := range r.fields {
		if f.TenantID != tenantID {
			continue
		}
		if entityType == nil || f.EntityType == *entityType {
			result = append(result, f)
		}
	}
	total := len(result)
	if offset >= total {
		return nil, total, nil
	}
	end := min(offset+limit, total)
	return result[offset:end], total, nil
}

func (r *stubCustomFieldRepo) Update(_ context.Context, field *models.CustomFieldDefinition) error {
	existing, ok := r.fields[field.ID]
	if !ok || existing.TenantID != field.TenantID {
		return customfield.ErrFieldNotFound
	}
	r.fields[field.ID] = field
	r.fieldsByKey[customFieldKey(field.TenantID, field.EntityType, field.FieldName)] = field
	return nil
}

func (r *stubCustomFieldRepo) Delete(_ context.Context, id, tenantID uuid.UUID) error {
	f, ok := r.fields[id]
	if !ok || f.TenantID != tenantID {
		return customfield.ErrFieldNotFound
	}
	delete(r.fields, id)
	return nil
}

type stubTagRepo struct {
	tags     map[uuid.UUID]*models.Tag
	inUseIDs map[uuid.UUID]bool
}

func newStubTagRepo() *stubTagRepo {
	return &stubTagRepo{tags: make(map[uuid.UUID]*models.Tag), inUseIDs: make(map[uuid.UUID]bool)}
}

func (r *stubTagRepo) Create(_ context.Context, t *models.Tag) error {
	r.tags[t.ID] = t
	return nil
}

func (r *stubTagRepo) GetByID(_ context.Context, id, tenantID uuid.UUID) (*models.Tag, error) {
	t, ok := r.tags[id]
	if !ok || t.TenantID != tenantID {
		return nil, tag.ErrTagNotFound
	}
	return t, nil
}

func (r *stubTagRepo) GetByEntityAndName(_ context.Context, tenantID uuid.UUID, entityType models.EntityType, name string) (*models.Tag, error) {
	for _, t := range r.tags {
		if t.TenantID == tenantID && t.EntityType == entityType && t.Name == name {
			return t, nil
		}
	}
	return nil, tag.ErrTagNotFound
}

func (r *stubTagRepo) List(_ context.Context, tenantID uuid.UUID, entityType *models.EntityType, offset, limit int) ([]*models.Tag, int, error) {
	var result []*models.Tag
	for _, t := range r.tags {
		if t.TenantID != tenantID {
			continue
		}
		if entityType == nil || t.EntityType == *entityType {
			result = append(result, t)
		}
	}
	total := len(result)
	if offset >= total {
		return nil, total, nil
	}
	end := min(offset+limit, total)
	return result[offset:end], total, nil
}

func (r *stubTagRepo) Update(_ context.Context, t *models.Tag) error {
	existing, ok := r.tags[t.ID]
	if !ok || existing.TenantID != t.TenantID {
		return tag.ErrTagNotFound
	}
	r.tags[t.ID] = t
	return nil
}

func (r *stubTagRepo) Delete(_ context.Context, id, tenantID uuid.UUID) error {
	t, ok := r.tags[id]
	if !ok || t.TenantID != tenantID {
		return tag.ErrTagNotFound
	}
	delete(r.tags, id)
	return nil
}

func (r *stubTagRepo) IsInUse(_ context.Context, id, _ uuid.UUID) (bool, error) {
	return r.inUseIDs[id], nil
}

type stubContactRepo struct {
	contacts      map[uuid.UUID]*models.Contact
	contactTags   map[uuid.UUID][]*models.Tag
	companies     map[uuid.UUID]string
	validTags     map[uuid.UUID]models.EntityType
	inUseContacts map[uuid.UUID]bool
}

func newStubContactRepo() *stubContactRepo {
	return &stubContactRepo{
		contacts:      make(map[uuid.UUID]*models.Contact),
		contactTags:   make(map[uuid.UUID][]*models.Tag),
		companies:     make(map[uuid.UUID]string),
		validTags:     make(map[uuid.UUID]models.EntityType),
		inUseContacts: make(map[uuid.UUID]bool),
	}
}

func (r *stubContactRepo) Create(_ context.Context, c *models.Contact) error {
	r.contacts[c.ID] = c
	return nil
}

func (r *stubContactRepo) GetByID(_ context.Context, id, _ uuid.UUID) (*models.Contact, error) {
	c, ok := r.contacts[id]
	if !ok {
		return nil, contact.ErrContactNotFound
	}
	return c, nil
}

func (r *stubContactRepo) GetByEmail(_ context.Context, email string, _ uuid.UUID) (*models.Contact, error) {
	for _, c := range r.contacts {
		if c.Email != nil && *c.Email == email {
			return c, nil
		}
	}
	return nil, contact.ErrContactNotFound
}

func (r *stubContactRepo) List(_ context.Context, _ contact.ListFilter, offset, limit int) ([]*models.Contact, int, error) {
	var result []*models.Contact
	for _, c := range r.contacts {
		result = append(result, c)
	}
	total := len(result)
	if offset >= total {
		return []*models.Contact{}, total, nil
	}
	end := min(offset+limit, total)
	return result[offset:end], total, nil
}

func (r *stubContactRepo) ListWithVisibility(ctx context.Context, _ uuid.UUID, _ bool, filter contact.ListFilter, offset, limit int) ([]*models.Contact, int, error) {
	return r.List(ctx, filter, offset, limit)
}

func (r *stubContactRepo) ListByIDs(_ context.Context, ids []uuid.UUID, _ uuid.UUID) ([]*models.Contact, error) {
	var result []*models.Contact
	for _, id := range ids {
		if c, ok := r.contacts[id]; ok {
			result = append(result, c)
		}
	}
	return result, nil
}

func (r *stubContactRepo) ListAll(_ context.Context, _ uuid.UUID, _ bool, _ uuid.UUID) ([]*models.Contact, error) {
	var result []*models.Contact
	for _, c := range r.contacts {
		result = append(result, c)
	}
	return result, nil
}

func (r *stubContactRepo) Update(_ context.Context, c *models.Contact, _ uuid.UUID) error {
	r.contacts[c.ID] = c
	return nil
}

func (r *stubContactRepo) UpdateVisibility(_ context.Context, contactID uuid.UUID, visibility string, ownerID *uuid.UUID, _ uuid.UUID) error {
	if c, ok := r.contacts[contactID]; ok {
		c.Visibility = visibility
		c.OwnerID = ownerID
	}
	return nil
}

func (r *stubContactRepo) Delete(_ context.Context, id, _ uuid.UUID) error {
	delete(r.contacts, id)
	return nil
}

func (r *stubContactRepo) GetCompanyName(_ context.Context, companyID, _ uuid.UUID) (string, error) {
	return r.companies[companyID], nil
}

func (r *stubContactRepo) GetTags(_ context.Context, contactID uuid.UUID) ([]*models.Tag, error) {
	return r.contactTags[contactID], nil
}

func (r *stubContactRepo) AddTags(_ context.Context, _ uuid.UUID, _ []uuid.UUID) error {
	return nil
}

func (r *stubContactRepo) RemoveTags(_ context.Context, _ uuid.UUID, _ []uuid.UUID) error {
	return nil
}

func (r *stubContactRepo) GetCompanyNames(_ context.Context, companyIDs []uuid.UUID, _ uuid.UUID) (map[uuid.UUID]string, error) {
	result := make(map[uuid.UUID]string)
	for _, id := range companyIDs {
		if name, ok := r.companies[id]; ok {
			result[id] = name
		}
	}
	return result, nil
}

func (r *stubContactRepo) GetTagsBatch(_ context.Context, contactIDs []uuid.UUID) (map[uuid.UUID][]*models.Tag, error) {
	result := make(map[uuid.UUID][]*models.Tag)
	for _, id := range contactIDs {
		if tags, ok := r.contactTags[id]; ok {
			result[id] = tags
		}
	}
	return result, nil
}

func (r *stubContactRepo) GetCustomFieldValuesBatch(_ context.Context, _ []uuid.UUID) (map[uuid.UUID][]*models.CustomFieldValueRow, error) {
	return make(map[uuid.UUID][]*models.CustomFieldValueRow), nil
}

func (r *stubContactRepo) GetCustomFieldValues(_ context.Context, _ uuid.UUID) ([]*models.CustomFieldValueRow, error) {
	return nil, nil
}

func (r *stubContactRepo) SetCustomFieldValues(_ context.Context, _ uuid.UUID, _ map[uuid.UUID]any) error {
	return nil
}

func (r *stubContactRepo) IsInUse(_ context.Context, id, _ uuid.UUID) (bool, string, error) {
	if r.inUseContacts[id] {
		return true, "call campaign history", nil
	}
	return false, "", nil
}

func (r *stubContactRepo) CompanyExists(_ context.Context, companyID, _ uuid.UUID) (bool, error) {
	_, ok := r.companies[companyID]
	return ok, nil
}

func (r *stubContactRepo) TagExists(_ context.Context, tagID uuid.UUID, entityType models.EntityType) (bool, error) {
	et, ok := r.validTags[tagID]
	return ok && et == entityType, nil
}

func (r *stubContactRepo) FindDuplicateCandidates(_ context.Context, _, _ uuid.UUID) ([]*contact.DuplicateCandidate, error) {
	return nil, nil
}

func (r *stubContactRepo) MergeInto(_ context.Context, primaryID, duplicateID, _ uuid.UUID) error {
	if dup, ok := r.contacts[duplicateID]; ok {
		mergedInto := primaryID
		dup.MergedIntoID = &mergedInto
	}
	return nil
}

func (r *stubContactRepo) ListLeads(_ context.Context, _ contact.LeadFilter, _, _ int) ([]*models.ContactWithRelations, int, error) {
	return nil, 0, nil
}

func (r *stubContactRepo) UpdateLead(_ context.Context, _, _ uuid.UUID, _ contact.LeadPatch) (*models.ContactWithRelations, error) {
	return nil, contact.ErrContactNotFound
}

// ============================================================================
// Test server constructors
// ============================================================================

// newTestCRMServer builds a server with every sub-service left nil. Usable
// only for request paths that fail (uuid parse, missing tenant) before the
// handler reaches into a service, and for domain validation that a Service
// method checks before touching its repo (nil pointer receiver call is legal
// in Go as long as the method body never dereferences it) — same pattern as
// newTestFormulareServer in formulare_grpc_test.go.
func newTestCRMServer() *CRMGRPCServer {
	return &CRMGRPCServer{}
}

func newCRMServerWithCustomFieldRepo(repo customfield.Repository) *CRMGRPCServer {
	return &CRMGRPCServer{customFieldService: customfield.NewService(repo)}
}

func newCRMServerWithTagRepo(repo tag.Repository) *CRMGRPCServer {
	return &CRMGRPCServer{tagService: tag.NewService(repo)}
}

func newCRMServerWithContactRepo(repo contact.Repository) *CRMGRPCServer {
	return &CRMGRPCServer{contactService: contact.NewService(repo)}
}

// ============================================================================
// Custom Fields
// ============================================================================

func TestCreateCustomField_MissingTenant(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.CreateCustomField(context.Background(), &crmv1.CreateCustomFieldRequest{
		CreatedBy: uuid.New().String(),
	})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

// TestCreateCustomField_InvalidCreatedBy uses a repo-backed server, not
// newTestCRMServer(), and fills every other field with valid values. A
// nil-service server or an empty EntityType would make ErrInvalidEntityType
// fire first and mask whether the created_by check ran at all - a mutation
// probe against this exact test caught that mistake in an earlier draft.
func TestCreateCustomField_InvalidCreatedBy(t *testing.T) {
	repo := newStubCustomFieldRepo()
	srv := newCRMServerWithCustomFieldRepo(repo)
	_, err := srv.CreateCustomField(ctxWithTenant(uuid.New()), &crmv1.CreateCustomFieldRequest{
		CreatedBy:  "not-a-uuid",
		EntityType: string(models.EntityTypeContact),
		FieldName:  "x",
		FieldLabel: "X",
		FieldType:  string(models.FieldTypeText),
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestCreateCustomField_InvalidEntityType(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.CreateCustomField(ctxWithTenant(uuid.New()), &crmv1.CreateCustomFieldRequest{
		CreatedBy:  uuid.New().String(),
		EntityType: "not-a-real-entity",
		FieldName:  "x",
		FieldLabel: "X",
		FieldType:  string(models.FieldTypeText),
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestCreateCustomField_HappyPath(t *testing.T) {
	repo := newStubCustomFieldRepo()
	srv := newCRMServerWithCustomFieldRepo(repo)
	tenantID := uuid.New()

	resp, err := srv.CreateCustomField(ctxWithTenant(tenantID), &crmv1.CreateCustomFieldRequest{
		CreatedBy:  uuid.New().String(),
		EntityType: string(models.EntityTypeContact),
		FieldName:  "loyalty_tier",
		FieldLabel: "Loyalty Tier",
		FieldType:  string(models.FieldTypeText),
	})
	requireGRPCOK(t, err)
	if resp.CustomField == nil {
		t.Fatal("expected custom field in response")
	}
	if resp.CustomField.FieldName != "loyalty_tier" {
		t.Errorf("field_name mismatch: got %s", resp.CustomField.FieldName)
	}
}

func TestGetCustomField_InvalidID(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.GetCustomField(ctxWithTenant(uuid.New()), &crmv1.GetCustomFieldRequest{Id: "bad-id"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestGetCustomField_NotFound(t *testing.T) {
	repo := newStubCustomFieldRepo()
	srv := newCRMServerWithCustomFieldRepo(repo)
	_, err := srv.GetCustomField(ctxWithTenant(uuid.New()), &crmv1.GetCustomFieldRequest{Id: uuid.New().String()})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestGetCustomField_HappyPath(t *testing.T) {
	repo := newStubCustomFieldRepo()
	srv := newCRMServerWithCustomFieldRepo(repo)
	tenantID := uuid.New()

	createResp, err := srv.CreateCustomField(ctxWithTenant(tenantID), &crmv1.CreateCustomFieldRequest{
		CreatedBy:  uuid.New().String(),
		EntityType: string(models.EntityTypeContact),
		FieldName:  "vip",
		FieldLabel: "VIP",
		FieldType:  string(models.FieldTypeBoolean),
	})
	requireGRPCOK(t, err)

	getResp, err := srv.GetCustomField(ctxWithTenant(tenantID), &crmv1.GetCustomFieldRequest{Id: createResp.CustomField.Id})
	requireGRPCOK(t, err)
	if getResp.CustomField.Id != createResp.CustomField.Id {
		t.Error("id mismatch")
	}
}

func TestListCustomFields_MissingTenant(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.ListCustomFields(context.Background(), &crmv1.ListCustomFieldsRequest{})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestListCustomFields_HappyPath(t *testing.T) {
	repo := newStubCustomFieldRepo()
	srv := newCRMServerWithCustomFieldRepo(repo)
	tenantID := uuid.New()

	for i := range 2 {
		_, err := srv.CreateCustomField(ctxWithTenant(tenantID), &crmv1.CreateCustomFieldRequest{
			CreatedBy:  uuid.New().String(),
			EntityType: string(models.EntityTypeContact),
			FieldName:  "f" + string(rune('a'+i)),
			FieldLabel: "F",
			FieldType:  string(models.FieldTypeText),
		})
		requireGRPCOK(t, err)
	}

	resp, err := srv.ListCustomFields(ctxWithTenant(tenantID), &crmv1.ListCustomFieldsRequest{})
	requireGRPCOK(t, err)
	if resp.Total != 2 {
		t.Errorf("expected total=2, got %d", resp.Total)
	}
	if len(resp.CustomFields) != 2 {
		t.Errorf("expected 2 custom fields, got %d", len(resp.CustomFields))
	}
}

// TestListCustomFields_EmptyIsNilNotEmptySlice documents the wire-shape of
// ListCustomFields for a tenant with zero fields: the handler (crm_grpc.go)
// pre-allocates `infos` with make(..., 0, ...), so an empty result serializes
// to `[]` rather than `null` — the same class of fix that was made for
// document_grpc.go's toProtoFile (see that file's
// TestToProtoFile_EmptyTagsIsNotNil).
func TestListCustomFields_EmptyIsNilNotEmptySlice(t *testing.T) {
	repo := newStubCustomFieldRepo()
	srv := newCRMServerWithCustomFieldRepo(repo)

	resp, err := srv.ListCustomFields(ctxWithTenant(uuid.New()), &crmv1.ListCustomFieldsRequest{})
	requireGRPCOK(t, err)
	if resp.Total != 0 {
		t.Errorf("expected total=0, got %d", resp.Total)
	}
	if resp.CustomFields == nil {
		t.Error("CustomFields should be an empty slice, not nil, so it serializes to [] rather than null")
	}
	if len(resp.CustomFields) != 0 {
		t.Errorf("expected 0 custom fields, got %d", len(resp.CustomFields))
	}
}

func TestUpdateCustomField_InvalidID(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.UpdateCustomField(ctxWithTenant(uuid.New()), &crmv1.UpdateCustomFieldRequest{Id: "bad-id"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestUpdateCustomField_NotFound(t *testing.T) {
	repo := newStubCustomFieldRepo()
	srv := newCRMServerWithCustomFieldRepo(repo)
	label := "New Label"
	_, err := srv.UpdateCustomField(ctxWithTenant(uuid.New()), &crmv1.UpdateCustomFieldRequest{
		Id:         uuid.New().String(),
		FieldLabel: &label,
	})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestUpdateCustomField_HappyPath(t *testing.T) {
	repo := newStubCustomFieldRepo()
	srv := newCRMServerWithCustomFieldRepo(repo)
	tenantID := uuid.New()

	createResp, err := srv.CreateCustomField(ctxWithTenant(tenantID), &crmv1.CreateCustomFieldRequest{
		CreatedBy:  uuid.New().String(),
		EntityType: string(models.EntityTypeContact),
		FieldName:  "tier",
		FieldLabel: "Tier",
		FieldType:  string(models.FieldTypeText),
	})
	requireGRPCOK(t, err)

	newLabel := "Loyalty Tier"
	updateResp, err := srv.UpdateCustomField(ctxWithTenant(tenantID), &crmv1.UpdateCustomFieldRequest{
		Id:         createResp.CustomField.Id,
		FieldLabel: &newLabel,
	})
	requireGRPCOK(t, err)
	if updateResp.CustomField.FieldLabel != newLabel {
		t.Errorf("label not updated: got %s", updateResp.CustomField.FieldLabel)
	}
}

func TestDeleteCustomField_InvalidID(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.DeleteCustomField(ctxWithTenant(uuid.New()), &crmv1.DeleteCustomFieldRequest{Id: "bad-id"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestDeleteCustomField_HappyPath(t *testing.T) {
	repo := newStubCustomFieldRepo()
	srv := newCRMServerWithCustomFieldRepo(repo)
	tenantID := uuid.New()

	createResp, err := srv.CreateCustomField(ctxWithTenant(tenantID), &crmv1.CreateCustomFieldRequest{
		CreatedBy:  uuid.New().String(),
		EntityType: string(models.EntityTypeContact),
		FieldName:  "throwaway",
		FieldLabel: "Throwaway",
		FieldType:  string(models.FieldTypeText),
	})
	requireGRPCOK(t, err)

	resp, err := srv.DeleteCustomField(ctxWithTenant(tenantID), &crmv1.DeleteCustomFieldRequest{Id: createResp.CustomField.Id})
	requireGRPCOK(t, err)
	if resp == nil {
		t.Fatal("expected non-nil empty response")
	}

	_, err = srv.GetCustomField(ctxWithTenant(tenantID), &crmv1.GetCustomFieldRequest{Id: createResp.CustomField.Id})
	requireGRPCCode(t, err, codes.NotFound)
}

// ============================================================================
// Tags
// ============================================================================

func TestCreateTag_MissingTenant(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.CreateTag(context.Background(), &crmv1.CreateTagRequest{Name: "VIP", EntityType: string(models.EntityTypeContact)})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestCreateTag_NameRequired(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.CreateTag(ctxWithTenant(uuid.New()), &crmv1.CreateTagRequest{
		Name:       "   ",
		EntityType: string(models.EntityTypeContact),
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestCreateTag_InvalidColor(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.CreateTag(ctxWithTenant(uuid.New()), &crmv1.CreateTagRequest{
		Name:       "VIP",
		Color:      "not-a-hex-color",
		EntityType: string(models.EntityTypeContact),
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestCreateTag_HappyPath(t *testing.T) {
	repo := newStubTagRepo()
	srv := newCRMServerWithTagRepo(repo)
	resp, err := srv.CreateTag(ctxWithTenant(uuid.New()), &crmv1.CreateTagRequest{
		Name:       "VIP",
		Color:      "#ef4444",
		EntityType: string(models.EntityTypeContact),
	})
	requireGRPCOK(t, err)
	if resp.Tag == nil || resp.Tag.Name != "VIP" {
		t.Fatal("expected tag VIP in response")
	}
}

func TestGetTag_InvalidID(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.GetTag(ctxWithTenant(uuid.New()), &crmv1.GetTagRequest{Id: "bad-id"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestGetTag_NotFound(t *testing.T) {
	repo := newStubTagRepo()
	srv := newCRMServerWithTagRepo(repo)
	_, err := srv.GetTag(ctxWithTenant(uuid.New()), &crmv1.GetTagRequest{Id: uuid.New().String()})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestGetTag_HappyPath(t *testing.T) {
	repo := newStubTagRepo()
	srv := newCRMServerWithTagRepo(repo)
	tenantID := uuid.New()

	createResp, err := srv.CreateTag(ctxWithTenant(tenantID), &crmv1.CreateTagRequest{
		Name:       "Newsletter",
		EntityType: string(models.EntityTypeContact),
	})
	requireGRPCOK(t, err)

	getResp, err := srv.GetTag(ctxWithTenant(tenantID), &crmv1.GetTagRequest{Id: createResp.Tag.Id})
	requireGRPCOK(t, err)
	if getResp.Tag.Id != createResp.Tag.Id {
		t.Error("id mismatch")
	}
}

func TestListTags_MissingTenant(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.ListTags(context.Background(), &crmv1.ListTagsRequest{})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestListTags_HappyPath(t *testing.T) {
	repo := newStubTagRepo()
	srv := newCRMServerWithTagRepo(repo)
	tenantID := uuid.New()

	for _, name := range []string{"VIP", "Newsletter"} {
		_, err := srv.CreateTag(ctxWithTenant(tenantID), &crmv1.CreateTagRequest{Name: name, EntityType: string(models.EntityTypeContact)})
		requireGRPCOK(t, err)
	}

	resp, err := srv.ListTags(ctxWithTenant(tenantID), &crmv1.ListTagsRequest{})
	requireGRPCOK(t, err)
	if resp.Total != 2 {
		t.Errorf("expected total=2, got %d", resp.Total)
	}
}

// TestListTags_EmptyIsNilNotEmptySlice — same wire-shape fix as
// TestListCustomFields_EmptyIsNilNotEmptySlice, for the Tags field. See that
// test's comment.
func TestListTags_EmptyIsNilNotEmptySlice(t *testing.T) {
	repo := newStubTagRepo()
	srv := newCRMServerWithTagRepo(repo)

	resp, err := srv.ListTags(ctxWithTenant(uuid.New()), &crmv1.ListTagsRequest{})
	requireGRPCOK(t, err)
	if resp.Tags == nil {
		t.Error("Tags should be an empty slice, not nil, so it serializes to [] rather than null")
	}
	if len(resp.Tags) != 0 {
		t.Errorf("expected 0 tags, got %d", len(resp.Tags))
	}
}

func TestUpdateTag_InvalidID(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.UpdateTag(ctxWithTenant(uuid.New()), &crmv1.UpdateTagRequest{Id: "bad-id"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestUpdateTag_NotFound(t *testing.T) {
	repo := newStubTagRepo()
	srv := newCRMServerWithTagRepo(repo)
	name := "Renamed"
	_, err := srv.UpdateTag(ctxWithTenant(uuid.New()), &crmv1.UpdateTagRequest{Id: uuid.New().String(), Name: &name})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestUpdateTag_HappyPath(t *testing.T) {
	repo := newStubTagRepo()
	srv := newCRMServerWithTagRepo(repo)
	tenantID := uuid.New()

	createResp, err := srv.CreateTag(ctxWithTenant(tenantID), &crmv1.CreateTagRequest{Name: "Old", EntityType: string(models.EntityTypeContact)})
	requireGRPCOK(t, err)

	newName := "New"
	updateResp, err := srv.UpdateTag(ctxWithTenant(tenantID), &crmv1.UpdateTagRequest{Id: createResp.Tag.Id, Name: &newName})
	requireGRPCOK(t, err)
	if updateResp.Tag.Name != "New" {
		t.Errorf("name not updated: got %s", updateResp.Tag.Name)
	}
}

func TestDeleteTag_InvalidID(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.DeleteTag(ctxWithTenant(uuid.New()), &crmv1.DeleteTagRequest{Id: "bad-id"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestDeleteTag_HappyPath(t *testing.T) {
	repo := newStubTagRepo()
	srv := newCRMServerWithTagRepo(repo)
	tenantID := uuid.New()

	createResp, err := srv.CreateTag(ctxWithTenant(tenantID), &crmv1.CreateTagRequest{Name: "Throwaway", EntityType: string(models.EntityTypeContact)})
	requireGRPCOK(t, err)

	resp, err := srv.DeleteTag(ctxWithTenant(tenantID), &crmv1.DeleteTagRequest{Id: createResp.Tag.Id})
	requireGRPCOK(t, err)
	if resp == nil {
		t.Fatal("expected non-nil empty response")
	}
}

func TestDeleteTag_InUse(t *testing.T) {
	repo := newStubTagRepo()
	srv := newCRMServerWithTagRepo(repo)
	tenantID := uuid.New()

	createResp, err := srv.CreateTag(ctxWithTenant(tenantID), &crmv1.CreateTagRequest{Name: "Pinned", EntityType: string(models.EntityTypeContact)})
	requireGRPCOK(t, err)

	tagID, err := uuid.Parse(createResp.Tag.Id)
	requireGRPCOK(t, err)
	repo.inUseIDs[tagID] = true

	_, err = srv.DeleteTag(ctxWithTenant(tenantID), &crmv1.DeleteTagRequest{Id: createResp.Tag.Id})
	requireGRPCCode(t, err, codes.FailedPrecondition)
}

// ============================================================================
// Contacts
// ============================================================================

func TestCreateContact_MissingTenant(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.CreateContact(context.Background(), &crmv1.CreateContactRequest{
		CreatedBy: uuid.New().String(),
		FirstName: "Max",
		LastName:  "Mustermann",
	})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestCreateContact_InvalidCreatedBy(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.CreateContact(ctxWithTenant(uuid.New()), &crmv1.CreateContactRequest{
		CreatedBy: "not-a-uuid",
		FirstName: "Max",
		LastName:  "Mustermann",
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestCreateContact_InvalidCompanyID(t *testing.T) {
	srv := newTestCRMServer()
	badCompany := "not-a-uuid"
	_, err := srv.CreateContact(ctxWithTenant(uuid.New()), &crmv1.CreateContactRequest{
		CreatedBy: uuid.New().String(),
		FirstName: "Max",
		LastName:  "Mustermann",
		CompanyId: &badCompany,
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestCreateContact_FirstNameRequired(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.CreateContact(ctxWithTenant(uuid.New()), &crmv1.CreateContactRequest{
		CreatedBy: uuid.New().String(),
		FirstName: "  ",
		LastName:  "Mustermann",
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestCreateContact_HappyPath(t *testing.T) {
	repo := newStubContactRepo()
	srv := newCRMServerWithContactRepo(repo)
	resp, err := srv.CreateContact(ctxWithTenant(uuid.New()), &crmv1.CreateContactRequest{
		CreatedBy: uuid.New().String(),
		FirstName: "Max",
		LastName:  "Mustermann",
	})
	requireGRPCOK(t, err)
	if resp.Contact == nil || resp.Contact.FirstName != "Max" {
		t.Fatal("expected contact Max in response")
	}
}

func TestGetContact_InvalidID(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.GetContact(ctxWithTenant(uuid.New()), &crmv1.GetContactRequest{Id: "bad-id"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestGetContact_NotFound(t *testing.T) {
	repo := newStubContactRepo()
	srv := newCRMServerWithContactRepo(repo)
	_, err := srv.GetContact(ctxWithTenant(uuid.New()), &crmv1.GetContactRequest{Id: uuid.New().String()})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestGetContact_HappyPath(t *testing.T) {
	repo := newStubContactRepo()
	srv := newCRMServerWithContactRepo(repo)
	tenantID := uuid.New()

	createResp, err := srv.CreateContact(ctxWithTenant(tenantID), &crmv1.CreateContactRequest{
		CreatedBy: uuid.New().String(),
		FirstName: "Erika",
		LastName:  "Musterfrau",
	})
	requireGRPCOK(t, err)

	getResp, err := srv.GetContact(ctxWithTenant(tenantID), &crmv1.GetContactRequest{Id: createResp.Contact.Id})
	requireGRPCOK(t, err)
	if getResp.Contact.Id != createResp.Contact.Id {
		t.Error("id mismatch")
	}
}

func TestListContacts_MissingTenant(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.ListContacts(context.Background(), &crmv1.ListContactsRequest{})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestListContacts_HappyPath(t *testing.T) {
	repo := newStubContactRepo()
	srv := newCRMServerWithContactRepo(repo)
	tenantID := uuid.New()

	for _, name := range []string{"Max", "Erika"} {
		_, err := srv.CreateContact(ctxWithTenant(tenantID), &crmv1.CreateContactRequest{
			CreatedBy: uuid.New().String(),
			FirstName: name,
			LastName:  "Mustermann",
		})
		requireGRPCOK(t, err)
	}

	resp, err := srv.ListContacts(ctxWithTenant(tenantID), &crmv1.ListContactsRequest{})
	requireGRPCOK(t, err)
	if resp.Total != 2 {
		t.Errorf("expected total=2, got %d", resp.Total)
	}
	if len(resp.Contacts) != 2 {
		t.Errorf("expected 2 contacts, got %d", len(resp.Contacts))
	}
}

// TestListContacts_EmptyIsNilNotEmptySlice — same wire-shape fix as
// TestListCustomFields_EmptyIsNilNotEmptySlice, for the Contacts field. Here
// the nil traced one layer further back than the other two: contact.Service.
// List calls enrichWithRelationsBatch, which used to return `nil, nil` for
// zero input contacts (internal/crm/contact/service.go:401-403) before
// crm_grpc.go ever got a slice to range over.
func TestListContacts_EmptyIsNilNotEmptySlice(t *testing.T) {
	repo := newStubContactRepo()
	srv := newCRMServerWithContactRepo(repo)

	resp, err := srv.ListContacts(ctxWithTenant(uuid.New()), &crmv1.ListContactsRequest{})
	requireGRPCOK(t, err)
	if resp.Contacts == nil {
		t.Error("Contacts should be an empty slice, not nil, so it serializes to [] rather than null")
	}
	if len(resp.Contacts) != 0 {
		t.Errorf("expected 0 contacts, got %d", len(resp.Contacts))
	}
}

// TestListContacts_WithVisibility_EmptyIsNilNotEmptySlice covers the same
// fix on the visibility-aware branch (UserId set), which goes through
// contact.Service.ListWithVisibility instead of .List but shares the same
// enrichWithRelationsBatch root cause.
func TestListContacts_WithVisibility_EmptyIsNilNotEmptySlice(t *testing.T) {
	repo := newStubContactRepo()
	srv := newCRMServerWithContactRepo(repo)

	resp, err := srv.ListContacts(ctxWithTenant(uuid.New()), &crmv1.ListContactsRequest{
		UserId: uuid.New().String(),
	})
	requireGRPCOK(t, err)
	if resp.Contacts == nil {
		t.Error("Contacts should be an empty slice, not nil, so it serializes to [] rather than null")
	}
	if len(resp.Contacts) != 0 {
		t.Errorf("expected 0 contacts, got %d", len(resp.Contacts))
	}
}

func TestUpdateContact_InvalidID(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.UpdateContact(ctxWithTenant(uuid.New()), &crmv1.UpdateContactRequest{Id: "bad-id"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestUpdateContact_InvalidCompanyID(t *testing.T) {
	repo := newStubContactRepo()
	srv := newCRMServerWithContactRepo(repo)
	tenantID := uuid.New()

	createResp, err := srv.CreateContact(ctxWithTenant(tenantID), &crmv1.CreateContactRequest{
		CreatedBy: uuid.New().String(),
		FirstName: "Max",
		LastName:  "Mustermann",
	})
	requireGRPCOK(t, err)

	badCompany := "not-a-uuid"
	_, err = srv.UpdateContact(ctxWithTenant(tenantID), &crmv1.UpdateContactRequest{
		Id:        createResp.Contact.Id,
		CompanyId: &badCompany,
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestUpdateContact_HappyPath(t *testing.T) {
	repo := newStubContactRepo()
	srv := newCRMServerWithContactRepo(repo)
	tenantID := uuid.New()

	createResp, err := srv.CreateContact(ctxWithTenant(tenantID), &crmv1.CreateContactRequest{
		CreatedBy: uuid.New().String(),
		FirstName: "Max",
		LastName:  "Mustermann",
	})
	requireGRPCOK(t, err)

	newFirstName := "Maximilian"
	updateResp, err := srv.UpdateContact(ctxWithTenant(tenantID), &crmv1.UpdateContactRequest{
		Id:        createResp.Contact.Id,
		FirstName: &newFirstName,
	})
	requireGRPCOK(t, err)
	if updateResp.Contact.FirstName != "Maximilian" {
		t.Errorf("first_name not updated: got %s", updateResp.Contact.FirstName)
	}
}

func TestDeleteContact_InvalidID(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.DeleteContact(ctxWithTenant(uuid.New()), &crmv1.DeleteContactRequest{Id: "bad-id"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestDeleteContact_HappyPath(t *testing.T) {
	repo := newStubContactRepo()
	srv := newCRMServerWithContactRepo(repo)
	tenantID := uuid.New()

	createResp, err := srv.CreateContact(ctxWithTenant(tenantID), &crmv1.CreateContactRequest{
		CreatedBy: uuid.New().String(),
		FirstName: "Throwaway",
		LastName:  "Contact",
	})
	requireGRPCOK(t, err)

	resp, err := srv.DeleteContact(ctxWithTenant(tenantID), &crmv1.DeleteContactRequest{Id: createResp.Contact.Id})
	requireGRPCOK(t, err)
	if resp == nil {
		t.Fatal("expected non-nil empty response")
	}
}

func TestDeleteContact_InUse(t *testing.T) {
	repo := newStubContactRepo()
	srv := newCRMServerWithContactRepo(repo)
	tenantID := uuid.New()

	createResp, err := srv.CreateContact(ctxWithTenant(tenantID), &crmv1.CreateContactRequest{
		CreatedBy: uuid.New().String(),
		FirstName: "Linked",
		LastName:  "Contact",
	})
	requireGRPCOK(t, err)

	contactID, err := uuid.Parse(createResp.Contact.Id)
	requireGRPCOK(t, err)
	repo.inUseContacts[contactID] = true

	_, err = srv.DeleteContact(ctxWithTenant(tenantID), &crmv1.DeleteContactRequest{Id: createResp.Contact.Id})
	requireGRPCCode(t, err, codes.FailedPrecondition)
}

func TestAddContactTags_InvalidContactID(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.AddContactTags(ctxWithTenant(uuid.New()), &crmv1.AddContactTagsRequest{ContactId: "bad-id"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestAddContactTags_InvalidTagID(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.AddContactTags(ctxWithTenant(uuid.New()), &crmv1.AddContactTagsRequest{
		ContactId: uuid.New().String(),
		TagIds:    []string{"not-a-uuid"},
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestAddContactTags_HappyPath(t *testing.T) {
	repo := newStubContactRepo()
	srv := newCRMServerWithContactRepo(repo)
	tenantID := uuid.New()

	createResp, err := srv.CreateContact(ctxWithTenant(tenantID), &crmv1.CreateContactRequest{
		CreatedBy: uuid.New().String(),
		FirstName: "Max",
		LastName:  "Mustermann",
	})
	requireGRPCOK(t, err)

	tagID := uuid.New()
	repo.validTags[tagID] = models.EntityTypeContact

	resp, err := srv.AddContactTags(ctxWithTenant(tenantID), &crmv1.AddContactTagsRequest{
		ContactId: createResp.Contact.Id,
		TagIds:    []string{tagID.String()},
	})
	requireGRPCOK(t, err)
	if resp.Contact == nil {
		t.Fatal("expected contact in response")
	}
}

func TestAddContactTags_UnknownTag(t *testing.T) {
	repo := newStubContactRepo()
	srv := newCRMServerWithContactRepo(repo)
	tenantID := uuid.New()

	createResp, err := srv.CreateContact(ctxWithTenant(tenantID), &crmv1.CreateContactRequest{
		CreatedBy: uuid.New().String(),
		FirstName: "Max",
		LastName:  "Mustermann",
	})
	requireGRPCOK(t, err)

	_, err = srv.AddContactTags(ctxWithTenant(tenantID), &crmv1.AddContactTagsRequest{
		ContactId: createResp.Contact.Id,
		TagIds:    []string{uuid.New().String()},
	})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestRemoveContactTags_InvalidContactID(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.RemoveContactTags(ctxWithTenant(uuid.New()), &crmv1.RemoveContactTagsRequest{ContactId: "bad-id"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestRemoveContactTags_InvalidTagID(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.RemoveContactTags(ctxWithTenant(uuid.New()), &crmv1.RemoveContactTagsRequest{
		ContactId: uuid.New().String(),
		TagIds:    []string{"not-a-uuid"},
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestRemoveContactTags_HappyPath(t *testing.T) {
	repo := newStubContactRepo()
	srv := newCRMServerWithContactRepo(repo)
	tenantID := uuid.New()

	createResp, err := srv.CreateContact(ctxWithTenant(tenantID), &crmv1.CreateContactRequest{
		CreatedBy: uuid.New().String(),
		FirstName: "Max",
		LastName:  "Mustermann",
	})
	requireGRPCOK(t, err)

	tagID := uuid.New()
	resp, err := srv.RemoveContactTags(ctxWithTenant(tenantID), &crmv1.RemoveContactTagsRequest{
		ContactId: createResp.Contact.Id,
		TagIds:    []string{tagID.String()},
	})
	requireGRPCOK(t, err)
	if resp.Contact == nil {
		t.Fatal("expected contact in response")
	}
}

func TestRemoveContactTags_NotFound(t *testing.T) {
	repo := newStubContactRepo()
	srv := newCRMServerWithContactRepo(repo)
	_, err := srv.RemoveContactTags(ctxWithTenant(uuid.New()), &crmv1.RemoveContactTagsRequest{
		ContactId: uuid.New().String(),
		TagIds:    []string{uuid.New().String()},
	})
	requireGRPCCode(t, err, codes.NotFound)
}
