package server

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"google.golang.org/grpc/codes"

	"github.com/kmuhub/kmuhub/internal/crm/activity"
	"github.com/kmuhub/kmuhub/internal/crm/consent"
	"github.com/kmuhub/kmuhub/internal/crm/report"
	"github.com/kmuhub/kmuhub/internal/crm/savedfilter"
	"github.com/kmuhub/kmuhub/internal/crm/search"
	"github.com/kmuhub/kmuhub/internal/models"
	crmv1 "github.com/kmuhub/kmuhub/proto/crm/v1"
)

// ============================================================================
// Stub repositories (server package copies — same pattern as stubDealRepo in
// crm_grpc_pipelines_deals_test.go).
// ============================================================================

type stubActivityRepo struct {
	activities map[uuid.UUID]*models.Activity
	contacts   map[uuid.UUID]bool
	companies  map[uuid.UUID]bool
	deals      map[uuid.UUID]bool
	users      map[uuid.UUID]bool
	validTags  map[uuid.UUID]models.EntityType
	tags       map[uuid.UUID][]*models.Tag
	timelines  map[uuid.UUID][]*activity.TimelineEvent
}

func newStubActivityRepo() *stubActivityRepo {
	return &stubActivityRepo{
		activities: make(map[uuid.UUID]*models.Activity),
		contacts:   make(map[uuid.UUID]bool),
		companies:  make(map[uuid.UUID]bool),
		deals:      make(map[uuid.UUID]bool),
		users:      make(map[uuid.UUID]bool),
		validTags:  make(map[uuid.UUID]models.EntityType),
		tags:       make(map[uuid.UUID][]*models.Tag),
		timelines:  make(map[uuid.UUID][]*activity.TimelineEvent),
	}
}

func (r *stubActivityRepo) Create(_ context.Context, a *models.Activity) error {
	r.activities[a.ID] = a
	return nil
}

func (r *stubActivityRepo) GetByID(_ context.Context, id, tenantID uuid.UUID) (*models.Activity, error) {
	a, ok := r.activities[id]
	if !ok || a.TenantID != tenantID {
		return nil, activity.ErrActivityNotFound
	}
	return a, nil
}

func (r *stubActivityRepo) List(_ context.Context, tenantID uuid.UUID, filter activity.ListFilter, offset, limit int) ([]*models.Activity, int, error) {
	var result []*models.Activity
	for _, a := range r.activities {
		if a.TenantID != tenantID {
			continue
		}
		if filter.ActivityType != nil && a.ActivityType != *filter.ActivityType {
			continue
		}
		if filter.ContactID != nil && (a.ContactID == nil || *a.ContactID != *filter.ContactID) {
			continue
		}
		if filter.IsCompleted != nil && a.IsCompleted != *filter.IsCompleted {
			continue
		}
		result = append(result, a)
	}
	total := len(result)
	if offset >= total {
		return []*models.Activity{}, total, nil
	}
	end := min(offset+limit, total)
	return result[offset:end], total, nil
}

func (r *stubActivityRepo) Update(_ context.Context, a *models.Activity) error {
	existing, ok := r.activities[a.ID]
	if !ok || existing.TenantID != a.TenantID {
		return activity.ErrActivityNotFound
	}
	r.activities[a.ID] = a
	return nil
}

func (r *stubActivityRepo) Delete(_ context.Context, id, tenantID uuid.UUID) error {
	a, ok := r.activities[id]
	if !ok || a.TenantID != tenantID {
		return activity.ErrActivityNotFound
	}
	delete(r.activities, id)
	return nil
}

func (r *stubActivityRepo) GetContactName(_ context.Context, _ uuid.UUID) (string, error) {
	return "", nil
}
func (r *stubActivityRepo) GetCompanyName(_ context.Context, _ uuid.UUID) (string, error) {
	return "", nil
}
func (r *stubActivityRepo) GetDealName(_ context.Context, _ uuid.UUID) (string, error) {
	return "", nil
}
func (r *stubActivityRepo) GetUserName(_ context.Context, _ uuid.UUID) (string, error) {
	return "", nil
}

func (r *stubActivityRepo) GetTags(_ context.Context, activityID uuid.UUID) ([]*models.Tag, error) {
	return r.tags[activityID], nil
}

func (r *stubActivityRepo) AddTags(_ context.Context, activityID uuid.UUID, tagIDs []uuid.UUID) error {
	for _, id := range tagIDs {
		r.tags[activityID] = append(r.tags[activityID], &models.Tag{ID: id})
	}
	return nil
}

func (r *stubActivityRepo) RemoveTags(_ context.Context, _ uuid.UUID, _ []uuid.UUID) error {
	return nil
}

func (r *stubActivityRepo) GetCustomFieldValues(_ context.Context, _ uuid.UUID) ([]*models.CustomFieldValueRow, error) {
	return nil, nil
}

func (r *stubActivityRepo) SetCustomFieldValues(_ context.Context, _ uuid.UUID, _ map[uuid.UUID]any) error {
	return nil
}

func (r *stubActivityRepo) ContactExists(_ context.Context, id uuid.UUID) (bool, error) {
	return r.contacts[id], nil
}
func (r *stubActivityRepo) CompanyExists(_ context.Context, id uuid.UUID) (bool, error) {
	return r.companies[id], nil
}
func (r *stubActivityRepo) DealExists(_ context.Context, id uuid.UUID) (bool, error) {
	return r.deals[id], nil
}
func (r *stubActivityRepo) UserExists(_ context.Context, id uuid.UUID) (bool, error) {
	return r.users[id], nil
}

func (r *stubActivityRepo) TagExists(_ context.Context, tagID uuid.UUID, entityType models.EntityType) (bool, error) {
	et, ok := r.validTags[tagID]
	return ok && et == entityType, nil
}

func (r *stubActivityRepo) GetContactTimeline(_ context.Context, contactID, _ uuid.UUID, offset, limit int) ([]*activity.TimelineEvent, int, error) {
	events := r.timelines[contactID]
	total := len(events)
	if offset >= total {
		return []*activity.TimelineEvent{}, total, nil
	}
	end := min(offset+limit, total)
	return events[offset:end], total, nil
}

func seedActivity(repo *stubActivityRepo, tenantID uuid.UUID, opts func(*models.Activity)) *models.Activity {
	a := &models.Activity{
		ID:           uuid.New(),
		TenantID:     tenantID,
		ActivityType: models.ActivityTypeCall,
		Subject:      "Follow up call",
		CreatedBy:    uuid.New(),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	if opts != nil {
		opts(a)
	}
	repo.activities[a.ID] = a
	return a
}

type stubSavedFilterRepo struct {
	filters map[uuid.UUID]*models.SavedFilter
}

func newStubSavedFilterRepo() *stubSavedFilterRepo {
	return &stubSavedFilterRepo{filters: make(map[uuid.UUID]*models.SavedFilter)}
}

func (r *stubSavedFilterRepo) Create(_ context.Context, f *models.SavedFilter) error {
	r.filters[f.ID] = f
	return nil
}

func (r *stubSavedFilterRepo) GetByID(_ context.Context, id uuid.UUID, tenantID uuid.UUID) (*models.SavedFilter, error) {
	f, ok := r.filters[id]
	if !ok || f.TenantID != tenantID {
		return nil, savedfilter.ErrFilterNotFound
	}
	return f, nil
}

func (r *stubSavedFilterRepo) List(_ context.Context, tenantID uuid.UUID, filter savedfilter.ListFilter) ([]*models.SavedFilter, error) {
	var result []*models.SavedFilter
	for _, f := range r.filters {
		if f.TenantID != tenantID {
			continue
		}
		if filter.EntityType != nil && f.EntityType != *filter.EntityType {
			continue
		}
		if filter.UserID != nil && f.CreatedBy != *filter.UserID {
			continue
		}
		result = append(result, f)
	}
	return result, nil
}

func (r *stubSavedFilterRepo) Update(_ context.Context, f *models.SavedFilter) error {
	existing, ok := r.filters[f.ID]
	if !ok || existing.TenantID != f.TenantID {
		return savedfilter.ErrFilterNotFound
	}
	r.filters[f.ID] = f
	return nil
}

func (r *stubSavedFilterRepo) Delete(_ context.Context, id uuid.UUID, tenantID uuid.UUID) error {
	f, ok := r.filters[id]
	if !ok || f.TenantID != tenantID {
		return savedfilter.ErrFilterNotFound
	}
	delete(r.filters, id)
	return nil
}

func (r *stubSavedFilterRepo) ClearDefault(_ context.Context, tenantID uuid.UUID, userID uuid.UUID, entityType models.EntityType) error {
	for _, f := range r.filters {
		if f.TenantID == tenantID && f.CreatedBy == userID && f.EntityType == entityType {
			f.IsDefault = false
		}
	}
	return nil
}

func seedSavedFilter(repo *stubSavedFilterRepo, tenantID uuid.UUID, opts func(*models.SavedFilter)) *models.SavedFilter {
	f := &models.SavedFilter{
		ID:         uuid.New(),
		TenantID:   tenantID,
		Name:       "My contacts",
		EntityType: models.EntityTypeContact,
		FilterJSON: `{"status":"active"}`,
		CreatedBy:  uuid.New(),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	if opts != nil {
		opts(f)
	}
	repo.filters[f.ID] = f
	return f
}

type stubSearchRepo struct {
	contacts   []*models.SearchResult
	companies  []*models.SearchResult
	deals      []*models.SearchResult
	activities []*models.SearchResult
}

func (r *stubSearchRepo) SearchContacts(_ context.Context, _ uuid.UUID, _ string, _ int) ([]*models.SearchResult, error) {
	return r.contacts, nil
}

func (r *stubSearchRepo) SearchCompanies(_ context.Context, _ uuid.UUID, _ string, _ int) ([]*models.SearchResult, error) {
	return r.companies, nil
}

func (r *stubSearchRepo) SearchDeals(_ context.Context, _ uuid.UUID, _ string, _ int) ([]*models.SearchResult, error) {
	return r.deals, nil
}

func (r *stubSearchRepo) SearchActivities(_ context.Context, _ uuid.UUID, _ string, _ int) ([]*models.SearchResult, error) {
	return r.activities, nil
}

type stubReportRepo struct {
	pipeline   *models.PipelineReport
	conversion *models.ConversionReport
	activityR  *models.ActivityReport
}

func (r *stubReportRepo) GetPipelineReport(_ context.Context, _ report.PipelineFilter) (*models.PipelineReport, error) {
	return r.pipeline, nil
}

func (r *stubReportRepo) GetConversionReport(_ context.Context, _ uuid.UUID, _, _ time.Time) (*models.ConversionReport, error) {
	return r.conversion, nil
}

func (r *stubReportRepo) GetActivityReport(_ context.Context, _ report.ActivityFilter) (*models.ActivityReport, error) {
	return r.activityR, nil
}

type stubConsentRepo struct {
	contacts         map[uuid.UUID]bool
	consentHistory   map[uuid.UUID][]*consent.ConsentRecord
	latestConsents   map[uuid.UUID][]*consent.ConsentRecord
	deletionRequests map[uuid.UUID]*consent.GDPRDeletionRequest
	anonymized       map[uuid.UUID]bool
}

func newStubConsentRepo() *stubConsentRepo {
	return &stubConsentRepo{
		contacts:         make(map[uuid.UUID]bool),
		consentHistory:   make(map[uuid.UUID][]*consent.ConsentRecord),
		latestConsents:   make(map[uuid.UUID][]*consent.ConsentRecord),
		deletionRequests: make(map[uuid.UUID]*consent.GDPRDeletionRequest),
		anonymized:       make(map[uuid.UUID]bool),
	}
}

func (r *stubConsentRepo) CreateConsentRecord(_ context.Context, rec *consent.ConsentRecord) error {
	r.consentHistory[rec.ContactID] = append(r.consentHistory[rec.ContactID], rec)
	r.latestConsents[rec.ContactID] = append(r.latestConsents[rec.ContactID], rec)
	return nil
}

func (r *stubConsentRepo) GetConsentHistory(_ context.Context, _, contactID uuid.UUID, consentType string) ([]*consent.ConsentRecord, error) {
	var result []*consent.ConsentRecord
	for _, rec := range r.consentHistory[contactID] {
		if rec.ConsentType == consentType {
			result = append(result, rec)
		}
	}
	return result, nil
}

func (r *stubConsentRepo) GetLatestConsents(_ context.Context, _, contactID uuid.UUID) ([]*consent.ConsentRecord, error) {
	return r.latestConsents[contactID], nil
}

func (r *stubConsentRepo) CreateDeletionRequest(_ context.Context, req *consent.GDPRDeletionRequest) error {
	r.deletionRequests[req.ID] = req
	return nil
}

func (r *stubConsentRepo) GetDeletionRequest(_ context.Context, id, tenantID uuid.UUID) (*consent.GDPRDeletionRequest, error) {
	req, ok := r.deletionRequests[id]
	if !ok || req.TenantID != tenantID {
		return nil, consent.ErrDeletionRequestNotFound
	}
	return req, nil
}

func (r *stubConsentRepo) UpdateDeletionRequest(_ context.Context, req *consent.GDPRDeletionRequest) error {
	r.deletionRequests[req.ID] = req
	return nil
}

func (r *stubConsentRepo) AnonymizeContact(_ context.Context, contactID, _ uuid.UUID) error {
	r.anonymized[contactID] = true
	return nil
}

func (r *stubConsentRepo) ContactExists(_ context.Context, contactID, _ uuid.UUID) (bool, error) {
	return r.contacts[contactID], nil
}

// ============================================================================
// Test server constructors
// ============================================================================

func newCRMServerWithActivityRepo(repo activity.Repository) *CRMGRPCServer {
	return &CRMGRPCServer{activityService: activity.NewService(repo)}
}

func newCRMServerWithSavedFilterRepo(repo savedfilter.Repository) *CRMGRPCServer {
	return &CRMGRPCServer{savedFilterService: savedfilter.NewService(repo)}
}

func newCRMServerWithSearchRepo(repo search.Repository) *CRMGRPCServer {
	return &CRMGRPCServer{searchService: search.NewService(repo)}
}

func newCRMServerWithReportRepo(repo report.Repository) *CRMGRPCServer {
	return &CRMGRPCServer{reportService: report.NewService(repo)}
}

func newCRMServerWithConsentRepo(repo consent.Repository) *CRMGRPCServer {
	return &CRMGRPCServer{consentService: consent.NewService(repo)}
}

// ============================================================================
// CreateActivity
// ============================================================================

func TestCreateActivity_MissingTenant(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.CreateActivity(context.Background(), &crmv1.CreateActivityRequest{})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestCreateActivity_InvalidCreatedBy(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.CreateActivity(ctxWithTenant(uuid.New()), &crmv1.CreateActivityRequest{CreatedBy: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestCreateActivity_InvalidActivityType(t *testing.T) {
	repo := newStubActivityRepo()
	srv := newCRMServerWithActivityRepo(repo)
	_, err := srv.CreateActivity(ctxWithTenant(uuid.New()), &crmv1.CreateActivityRequest{
		ActivityType: "bogus",
		Subject:      "x",
		CreatedBy:    uuid.New().String(),
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestCreateActivity_SubjectRequired(t *testing.T) {
	repo := newStubActivityRepo()
	srv := newCRMServerWithActivityRepo(repo)
	_, err := srv.CreateActivity(ctxWithTenant(uuid.New()), &crmv1.CreateActivityRequest{
		ActivityType: "call",
		Subject:      "   ",
		CreatedBy:    uuid.New().String(),
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestCreateActivity_ContactNotFound(t *testing.T) {
	repo := newStubActivityRepo()
	srv := newCRMServerWithActivityRepo(repo)
	contactID := uuid.New().String()
	_, err := srv.CreateActivity(ctxWithTenant(uuid.New()), &crmv1.CreateActivityRequest{
		ActivityType: "call",
		Subject:      "x",
		CreatedBy:    uuid.New().String(),
		ContactId:    &contactID,
	})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestCreateActivity_HappyPath(t *testing.T) {
	repo := newStubActivityRepo()
	srv := newCRMServerWithActivityRepo(repo)
	contactID := uuid.New()
	repo.contacts[contactID] = true
	contactIDStr := contactID.String()

	resp, err := srv.CreateActivity(ctxWithTenant(uuid.New()), &crmv1.CreateActivityRequest{
		ActivityType: "call",
		Subject:      "Follow up",
		CreatedBy:    uuid.New().String(),
		ContactId:    &contactIDStr,
		DueDate:      time.Now().Format(time.RFC3339),
	})
	requireGRPCOK(t, err)
	if resp.Activity.Subject != "Follow up" {
		t.Fatalf("expected subject Follow up, got %q", resp.Activity.Subject)
	}
	if resp.Activity.ContactId == nil || *resp.Activity.ContactId != contactIDStr {
		t.Fatalf("expected contact_id %s, got %v", contactIDStr, resp.Activity.ContactId)
	}
}

// ============================================================================
// GetActivity
// ============================================================================

func TestGetActivity_MissingTenant(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.GetActivity(context.Background(), &crmv1.GetActivityRequest{Id: uuid.New().String()})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestGetActivity_InvalidID(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.GetActivity(ctxWithTenant(uuid.New()), &crmv1.GetActivityRequest{Id: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestGetActivity_NotFound(t *testing.T) {
	repo := newStubActivityRepo()
	srv := newCRMServerWithActivityRepo(repo)
	_, err := srv.GetActivity(ctxWithTenant(uuid.New()), &crmv1.GetActivityRequest{Id: uuid.New().String()})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestGetActivity_HappyPath(t *testing.T) {
	repo := newStubActivityRepo()
	srv := newCRMServerWithActivityRepo(repo)
	tenantID := uuid.New()
	a := seedActivity(repo, tenantID, nil)

	resp, err := srv.GetActivity(ctxWithTenant(tenantID), &crmv1.GetActivityRequest{Id: a.ID.String()})
	requireGRPCOK(t, err)
	if resp.Activity.Id != a.ID.String() {
		t.Fatalf("expected activity id %s, got %s", a.ID, resp.Activity.Id)
	}
}

// ============================================================================
// ListActivities
// ============================================================================

func TestListActivities_MissingTenant(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.ListActivities(context.Background(), &crmv1.ListActivitiesRequest{})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestListActivities_InvalidContactFilter(t *testing.T) {
	repo := newStubActivityRepo()
	srv := newCRMServerWithActivityRepo(repo)
	badID := "not-a-uuid"
	_, err := srv.ListActivities(ctxWithTenant(uuid.New()), &crmv1.ListActivitiesRequest{ContactId: &badID})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestListActivities_HappyPath(t *testing.T) {
	repo := newStubActivityRepo()
	srv := newCRMServerWithActivityRepo(repo)
	tenantID := uuid.New()
	seedActivity(repo, tenantID, nil)
	seedActivity(repo, tenantID, nil)
	seedActivity(repo, uuid.New(), nil) // other tenant, must not be counted

	resp, err := srv.ListActivities(ctxWithTenant(tenantID), &crmv1.ListActivitiesRequest{Page: 1, PageSize: 10})
	requireGRPCOK(t, err)
	if resp.Total != 2 {
		t.Fatalf("expected total 2, got %d", resp.Total)
	}
	if len(resp.Activities) != 2 {
		t.Fatalf("expected 2 activities, got %d", len(resp.Activities))
	}
}

// TestListActivities_EmptyIsNilNotEmptySlice documents the wire-shape fix
// applied alongside fix-crm-list-nil-slice-wire-shape (Block A): the handler
// (crm_grpc.go) now pre-allocates `infos` with make(..., 0, ...), so an empty
// result serializes to `[]` rather than `null`.
func TestListActivities_EmptyIsNilNotEmptySlice(t *testing.T) {
	repo := newStubActivityRepo()
	srv := newCRMServerWithActivityRepo(repo)

	resp, err := srv.ListActivities(ctxWithTenant(uuid.New()), &crmv1.ListActivitiesRequest{Page: 1, PageSize: 10})
	requireGRPCOK(t, err)
	if resp.Activities == nil {
		t.Error("Activities should be an empty slice, not nil, so it serializes to [] rather than null")
	}
	if len(resp.Activities) != 0 {
		t.Errorf("expected 0 activities, got %d", len(resp.Activities))
	}
}

// ============================================================================
// UpdateActivity
// ============================================================================

func TestUpdateActivity_MissingTenant(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.UpdateActivity(context.Background(), &crmv1.UpdateActivityRequest{Id: uuid.New().String()})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestUpdateActivity_InvalidID(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.UpdateActivity(ctxWithTenant(uuid.New()), &crmv1.UpdateActivityRequest{Id: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestUpdateActivity_SubjectRequired(t *testing.T) {
	repo := newStubActivityRepo()
	srv := newCRMServerWithActivityRepo(repo)
	tenantID := uuid.New()
	a := seedActivity(repo, tenantID, nil)
	blank := "   "

	_, err := srv.UpdateActivity(ctxWithTenant(tenantID), &crmv1.UpdateActivityRequest{Id: a.ID.String(), Subject: &blank})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestUpdateActivity_ClearContactID(t *testing.T) {
	repo := newStubActivityRepo()
	srv := newCRMServerWithActivityRepo(repo)
	tenantID := uuid.New()
	contactID := uuid.New()
	a := seedActivity(repo, tenantID, func(a *models.Activity) { a.ContactID = &contactID })
	empty := ""

	resp, err := srv.UpdateActivity(ctxWithTenant(tenantID), &crmv1.UpdateActivityRequest{Id: a.ID.String(), ContactId: &empty})
	requireGRPCOK(t, err)
	if resp.Activity.ContactId != nil {
		t.Fatalf("expected contact_id cleared, got %v", resp.Activity.ContactId)
	}
}

func TestUpdateActivity_HappyPath(t *testing.T) {
	repo := newStubActivityRepo()
	srv := newCRMServerWithActivityRepo(repo)
	tenantID := uuid.New()
	a := seedActivity(repo, tenantID, nil)
	newSubject := "Updated subject"

	resp, err := srv.UpdateActivity(ctxWithTenant(tenantID), &crmv1.UpdateActivityRequest{Id: a.ID.String(), Subject: &newSubject})
	requireGRPCOK(t, err)
	if resp.Activity.Subject != newSubject {
		t.Fatalf("expected subject %q, got %q", newSubject, resp.Activity.Subject)
	}
}

// ============================================================================
// DeleteActivity
// ============================================================================

func TestDeleteActivity_MissingTenant(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.DeleteActivity(context.Background(), &crmv1.DeleteActivityRequest{Id: uuid.New().String()})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestDeleteActivity_InvalidID(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.DeleteActivity(ctxWithTenant(uuid.New()), &crmv1.DeleteActivityRequest{Id: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestDeleteActivity_NotFound(t *testing.T) {
	repo := newStubActivityRepo()
	srv := newCRMServerWithActivityRepo(repo)
	_, err := srv.DeleteActivity(ctxWithTenant(uuid.New()), &crmv1.DeleteActivityRequest{Id: uuid.New().String()})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestDeleteActivity_HappyPath(t *testing.T) {
	repo := newStubActivityRepo()
	srv := newCRMServerWithActivityRepo(repo)
	tenantID := uuid.New()
	a := seedActivity(repo, tenantID, nil)

	_, err := srv.DeleteActivity(ctxWithTenant(tenantID), &crmv1.DeleteActivityRequest{Id: a.ID.String()})
	requireGRPCOK(t, err)
	if _, ok := repo.activities[a.ID]; ok {
		t.Fatalf("expected activity to be deleted")
	}
}

// ============================================================================
// CompleteActivity
// ============================================================================

func TestCompleteActivity_MissingTenant(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.CompleteActivity(context.Background(), &crmv1.CompleteActivityRequest{Id: uuid.New().String()})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestCompleteActivity_InvalidID(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.CompleteActivity(ctxWithTenant(uuid.New()), &crmv1.CompleteActivityRequest{Id: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestCompleteActivity_NotFound(t *testing.T) {
	repo := newStubActivityRepo()
	srv := newCRMServerWithActivityRepo(repo)
	_, err := srv.CompleteActivity(ctxWithTenant(uuid.New()), &crmv1.CompleteActivityRequest{Id: uuid.New().String()})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestCompleteActivity_HappyPath(t *testing.T) {
	repo := newStubActivityRepo()
	srv := newCRMServerWithActivityRepo(repo)
	tenantID := uuid.New()
	a := seedActivity(repo, tenantID, nil)

	resp, err := srv.CompleteActivity(ctxWithTenant(tenantID), &crmv1.CompleteActivityRequest{Id: a.ID.String()})
	requireGRPCOK(t, err)
	if !resp.Activity.IsCompleted {
		t.Fatalf("expected activity to be completed")
	}
}

// ============================================================================
// Search
// ============================================================================

func TestSearch_MissingTenant(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.Search(context.Background(), &crmv1.SearchRequest{Query: "acme"})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestSearch_QueryTooShort(t *testing.T) {
	repo := &stubSearchRepo{}
	srv := newCRMServerWithSearchRepo(repo)
	_, err := srv.Search(ctxWithTenant(uuid.New()), &crmv1.SearchRequest{Query: "a"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestSearch_InvalidEntityType(t *testing.T) {
	repo := &stubSearchRepo{}
	srv := newCRMServerWithSearchRepo(repo)
	_, err := srv.Search(ctxWithTenant(uuid.New()), &crmv1.SearchRequest{Query: "acme", EntityTypes: []string{"bogus"}})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestSearch_HappyPath(t *testing.T) {
	repo := &stubSearchRepo{
		contacts: []*models.SearchResult{{ID: "1", EntityType: "contact", Title: "Acme Contact", Score: 1.0}},
	}
	srv := newCRMServerWithSearchRepo(repo)
	resp, err := srv.Search(ctxWithTenant(uuid.New()), &crmv1.SearchRequest{Query: "acme", EntityTypes: []string{"contact"}})
	requireGRPCOK(t, err)
	if len(resp.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(resp.Results))
	}
	if resp.Results[0].Title != "Acme Contact" {
		t.Fatalf("expected title Acme Contact, got %q", resp.Results[0].Title)
	}
}

// TestSearch_EmptyIsNilNotEmptySlice documents the wire-shape fix applied
// alongside fix-crm-list-nil-slice-wire-shape (Block A): the handler
// (crm_grpc.go) now pre-allocates `infos` with make(..., 0, ...), so a query
// with zero matches serializes to `[]` rather than `null`.
func TestSearch_EmptyIsNilNotEmptySlice(t *testing.T) {
	repo := &stubSearchRepo{}
	srv := newCRMServerWithSearchRepo(repo)

	resp, err := srv.Search(ctxWithTenant(uuid.New()), &crmv1.SearchRequest{Query: "acme"})
	requireGRPCOK(t, err)
	if resp.Results == nil {
		t.Error("Results should be an empty slice, not nil, so it serializes to [] rather than null")
	}
	if len(resp.Results) != 0 {
		t.Errorf("expected 0 results, got %d", len(resp.Results))
	}
}

// ============================================================================
// CreateSavedFilter
// ============================================================================

func TestCreateSavedFilter_MissingTenant(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.CreateSavedFilter(context.Background(), &crmv1.CreateSavedFilterRequest{})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestCreateSavedFilter_InvalidCreatedBy(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.CreateSavedFilter(ctxWithTenant(uuid.New()), &crmv1.CreateSavedFilterRequest{CreatedBy: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestCreateSavedFilter_NameRequired(t *testing.T) {
	repo := newStubSavedFilterRepo()
	srv := newCRMServerWithSavedFilterRepo(repo)
	_, err := srv.CreateSavedFilter(ctxWithTenant(uuid.New()), &crmv1.CreateSavedFilterRequest{
		Name:       "   ",
		EntityType: "contact",
		FilterJson: "{}",
		CreatedBy:  uuid.New().String(),
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestCreateSavedFilter_InvalidEntityType(t *testing.T) {
	repo := newStubSavedFilterRepo()
	srv := newCRMServerWithSavedFilterRepo(repo)
	_, err := srv.CreateSavedFilter(ctxWithTenant(uuid.New()), &crmv1.CreateSavedFilterRequest{
		Name:       "My filter",
		EntityType: "bogus",
		FilterJson: "{}",
		CreatedBy:  uuid.New().String(),
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestCreateSavedFilter_InvalidFilterJSON(t *testing.T) {
	repo := newStubSavedFilterRepo()
	srv := newCRMServerWithSavedFilterRepo(repo)
	_, err := srv.CreateSavedFilter(ctxWithTenant(uuid.New()), &crmv1.CreateSavedFilterRequest{
		Name:       "My filter",
		EntityType: "contact",
		FilterJson: "not-json",
		CreatedBy:  uuid.New().String(),
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestCreateSavedFilter_HappyPath(t *testing.T) {
	repo := newStubSavedFilterRepo()
	srv := newCRMServerWithSavedFilterRepo(repo)
	resp, err := srv.CreateSavedFilter(ctxWithTenant(uuid.New()), &crmv1.CreateSavedFilterRequest{
		Name:       "My filter",
		EntityType: "contact",
		FilterJson: `{"status":"active"}`,
		CreatedBy:  uuid.New().String(),
	})
	requireGRPCOK(t, err)
	if resp.Filter.Name != "My filter" {
		t.Fatalf("expected name My filter, got %q", resp.Filter.Name)
	}
}

// ============================================================================
// GetSavedFilter
// ============================================================================

func TestGetSavedFilter_MissingTenant(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.GetSavedFilter(context.Background(), &crmv1.GetSavedFilterRequest{Id: uuid.New().String()})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestGetSavedFilter_InvalidID(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.GetSavedFilter(ctxWithTenant(uuid.New()), &crmv1.GetSavedFilterRequest{Id: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestGetSavedFilter_NotFound(t *testing.T) {
	repo := newStubSavedFilterRepo()
	srv := newCRMServerWithSavedFilterRepo(repo)
	_, err := srv.GetSavedFilter(ctxWithTenant(uuid.New()), &crmv1.GetSavedFilterRequest{Id: uuid.New().String()})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestGetSavedFilter_HappyPath(t *testing.T) {
	repo := newStubSavedFilterRepo()
	srv := newCRMServerWithSavedFilterRepo(repo)
	tenantID := uuid.New()
	f := seedSavedFilter(repo, tenantID, nil)

	resp, err := srv.GetSavedFilter(ctxWithTenant(tenantID), &crmv1.GetSavedFilterRequest{Id: f.ID.String()})
	requireGRPCOK(t, err)
	if resp.Filter.Id != f.ID.String() {
		t.Fatalf("expected filter id %s, got %s", f.ID, resp.Filter.Id)
	}
}

// ============================================================================
// ListSavedFilters
// ============================================================================

func TestListSavedFilters_MissingTenant(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.ListSavedFilters(context.Background(), &crmv1.ListSavedFiltersRequest{})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestListSavedFilters_InvalidUserID(t *testing.T) {
	repo := newStubSavedFilterRepo()
	srv := newCRMServerWithSavedFilterRepo(repo)
	_, err := srv.ListSavedFilters(ctxWithTenant(uuid.New()), &crmv1.ListSavedFiltersRequest{UserId: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestListSavedFilters_HappyPath(t *testing.T) {
	repo := newStubSavedFilterRepo()
	srv := newCRMServerWithSavedFilterRepo(repo)
	tenantID := uuid.New()
	seedSavedFilter(repo, tenantID, nil)
	seedSavedFilter(repo, tenantID, nil)
	seedSavedFilter(repo, uuid.New(), nil) // other tenant

	resp, err := srv.ListSavedFilters(ctxWithTenant(tenantID), &crmv1.ListSavedFiltersRequest{})
	requireGRPCOK(t, err)
	if len(resp.Filters) != 2 {
		t.Fatalf("expected 2 filters, got %d", len(resp.Filters))
	}
}

// TestListSavedFilters_EmptyIsNilNotEmptySlice documents the wire-shape fix
// applied alongside fix-crm-list-nil-slice-wire-shape (Block A): the handler
// (crm_grpc.go) now pre-allocates `infos` with make(..., 0, ...), so an empty
// result serializes to `[]` rather than `null`.
func TestListSavedFilters_EmptyIsNilNotEmptySlice(t *testing.T) {
	repo := newStubSavedFilterRepo()
	srv := newCRMServerWithSavedFilterRepo(repo)

	resp, err := srv.ListSavedFilters(ctxWithTenant(uuid.New()), &crmv1.ListSavedFiltersRequest{})
	requireGRPCOK(t, err)
	if resp.Filters == nil {
		t.Error("Filters should be an empty slice, not nil, so it serializes to [] rather than null")
	}
	if len(resp.Filters) != 0 {
		t.Errorf("expected 0 filters, got %d", len(resp.Filters))
	}
}

// ============================================================================
// UpdateSavedFilter
// ============================================================================

func TestUpdateSavedFilter_MissingTenant(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.UpdateSavedFilter(context.Background(), &crmv1.UpdateSavedFilterRequest{Id: uuid.New().String()})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestUpdateSavedFilter_InvalidID(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.UpdateSavedFilter(ctxWithTenant(uuid.New()), &crmv1.UpdateSavedFilterRequest{Id: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestUpdateSavedFilter_NotFound(t *testing.T) {
	repo := newStubSavedFilterRepo()
	srv := newCRMServerWithSavedFilterRepo(repo)
	_, err := srv.UpdateSavedFilter(ctxWithTenant(uuid.New()), &crmv1.UpdateSavedFilterRequest{Id: uuid.New().String()})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestUpdateSavedFilter_HappyPath(t *testing.T) {
	repo := newStubSavedFilterRepo()
	srv := newCRMServerWithSavedFilterRepo(repo)
	tenantID := uuid.New()
	f := seedSavedFilter(repo, tenantID, nil)
	newName := "Renamed filter"

	resp, err := srv.UpdateSavedFilter(ctxWithTenant(tenantID), &crmv1.UpdateSavedFilterRequest{Id: f.ID.String(), Name: &newName})
	requireGRPCOK(t, err)
	if resp.Filter.Name != newName {
		t.Fatalf("expected name %q, got %q", newName, resp.Filter.Name)
	}
}

// ============================================================================
// DeleteSavedFilter
// ============================================================================

func TestDeleteSavedFilter_MissingTenant(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.DeleteSavedFilter(context.Background(), &crmv1.DeleteSavedFilterRequest{Id: uuid.New().String()})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestDeleteSavedFilter_InvalidID(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.DeleteSavedFilter(ctxWithTenant(uuid.New()), &crmv1.DeleteSavedFilterRequest{Id: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestDeleteSavedFilter_NotFound(t *testing.T) {
	repo := newStubSavedFilterRepo()
	srv := newCRMServerWithSavedFilterRepo(repo)
	_, err := srv.DeleteSavedFilter(ctxWithTenant(uuid.New()), &crmv1.DeleteSavedFilterRequest{Id: uuid.New().String()})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestDeleteSavedFilter_HappyPath(t *testing.T) {
	repo := newStubSavedFilterRepo()
	srv := newCRMServerWithSavedFilterRepo(repo)
	tenantID := uuid.New()
	f := seedSavedFilter(repo, tenantID, nil)

	_, err := srv.DeleteSavedFilter(ctxWithTenant(tenantID), &crmv1.DeleteSavedFilterRequest{Id: f.ID.String()})
	requireGRPCOK(t, err)
	if _, ok := repo.filters[f.ID]; ok {
		t.Fatalf("expected filter to be deleted")
	}
}

// ============================================================================
// GetPipelineReport
// ============================================================================

func TestGetPipelineReport_MissingTenant(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.GetPipelineReport(context.Background(), &crmv1.GetPipelineReportRequest{})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestGetPipelineReport_InvalidStartDate(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.GetPipelineReport(ctxWithTenant(uuid.New()), &crmv1.GetPipelineReportRequest{StartDate: "not-a-date", EndDate: "2026-01-31"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestGetPipelineReport_InvalidOwnerID(t *testing.T) {
	srv := newTestCRMServer()
	ownerID := "not-a-uuid"
	_, err := srv.GetPipelineReport(ctxWithTenant(uuid.New()), &crmv1.GetPipelineReportRequest{
		StartDate: "2026-01-01",
		EndDate:   "2026-01-31",
		OwnerId:   &ownerID,
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestGetPipelineReport_HappyPath(t *testing.T) {
	stageID := uuid.New()
	repo := &stubReportRepo{
		pipeline: &models.PipelineReport{
			Stages: []*models.PipelineStageValue{
				{StageID: stageID, StageName: "Lead", DealCount: 3, TotalValue: decimal.NewFromInt(3000), WeightedValue: decimal.NewFromInt(600)},
			},
			TotalDeals:         3,
			TotalValue:         decimal.NewFromInt(3000),
			TotalWeightedValue: decimal.NewFromInt(600),
		},
	}
	srv := newCRMServerWithReportRepo(repo)

	resp, err := srv.GetPipelineReport(ctxWithTenant(uuid.New()), &crmv1.GetPipelineReportRequest{StartDate: "2026-01-01", EndDate: "2026-01-31"})
	requireGRPCOK(t, err)
	if resp.TotalDeals != 3 {
		t.Fatalf("expected total_deals 3, got %d", resp.TotalDeals)
	}
	if len(resp.Stages) != 1 || resp.Stages[0].StageId != stageID.String() {
		t.Fatalf("expected 1 stage with id %s, got %+v", stageID, resp.Stages)
	}
}

// ============================================================================
// GetConversionReport
// ============================================================================

func TestGetConversionReport_MissingTenant(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.GetConversionReport(context.Background(), &crmv1.GetConversionReportRequest{})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestGetConversionReport_InvalidEndDate(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.GetConversionReport(ctxWithTenant(uuid.New()), &crmv1.GetConversionReportRequest{StartDate: "2026-01-01", EndDate: "not-a-date"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestGetConversionReport_HappyPath(t *testing.T) {
	repo := &stubReportRepo{
		conversion: &models.ConversionReport{
			Metrics:          []*models.ConversionMetric{{FromStage: "Lead", ToStage: "Won", ConvertedCount: 2, ConversionRate: 0.5, AverageDays: 10}},
			OverallWinRate:   0.5,
			AverageDealCycle: 12,
		},
	}
	srv := newCRMServerWithReportRepo(repo)

	resp, err := srv.GetConversionReport(ctxWithTenant(uuid.New()), &crmv1.GetConversionReportRequest{StartDate: "2026-01-01", EndDate: "2026-01-31"})
	requireGRPCOK(t, err)
	if resp.OverallWinRate != 0.5 {
		t.Fatalf("expected overall_win_rate 0.5, got %v", resp.OverallWinRate)
	}
	if len(resp.Metrics) != 1 || resp.Metrics[0].FromStage != "Lead" {
		t.Fatalf("expected 1 metric from Lead, got %+v", resp.Metrics)
	}
}

// ============================================================================
// GetActivityReport
// ============================================================================

func TestGetActivityReport_MissingTenant(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.GetActivityReport(context.Background(), &crmv1.GetActivityReportRequest{})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestGetActivityReport_InvalidUserID(t *testing.T) {
	srv := newTestCRMServer()
	userID := "not-a-uuid"
	_, err := srv.GetActivityReport(ctxWithTenant(uuid.New()), &crmv1.GetActivityReportRequest{
		StartDate: "2026-01-01",
		EndDate:   "2026-01-31",
		UserId:    &userID,
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestGetActivityReport_HappyPath(t *testing.T) {
	repo := &stubReportRepo{
		activityR: &models.ActivityReport{
			Metrics:         []*models.ActivityMetric{{ActivityType: "call", TotalCount: 5, CompletedCount: 3, OverdueCount: 1}},
			TotalActivities: 5,
			CompletionRate:  0.6,
		},
	}
	srv := newCRMServerWithReportRepo(repo)

	resp, err := srv.GetActivityReport(ctxWithTenant(uuid.New()), &crmv1.GetActivityReportRequest{StartDate: "2026-01-01", EndDate: "2026-01-31"})
	requireGRPCOK(t, err)
	if resp.TotalActivities != 5 {
		t.Fatalf("expected total_activities 5, got %d", resp.TotalActivities)
	}
	if len(resp.Metrics) != 1 || resp.Metrics[0].ActivityType != "call" {
		t.Fatalf("expected 1 call metric, got %+v", resp.Metrics)
	}
}

// ============================================================================
// GetContactTimeline
// ============================================================================

func TestGetContactTimeline_InvalidContactID(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.GetContactTimeline(ctxWithTenant(uuid.New()), &crmv1.GetContactTimelineRequest{ContactId: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestGetContactTimeline_MissingTenant(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.GetContactTimeline(context.Background(), &crmv1.GetContactTimelineRequest{ContactId: uuid.New().String()})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestGetContactTimeline_HappyPath(t *testing.T) {
	repo := newStubActivityRepo()
	srv := newCRMServerWithActivityRepo(repo)
	tenantID := uuid.New()
	contactID := uuid.New()
	repo.timelines[contactID] = []*activity.TimelineEvent{
		{ID: uuid.New(), EventType: "activity", OccurredAt: time.Now(), Title: "Called customer"},
	}

	resp, err := srv.GetContactTimeline(ctxWithTenant(tenantID), &crmv1.GetContactTimelineRequest{ContactId: contactID.String(), Page: 1, PageSize: 20})
	requireGRPCOK(t, err)
	if resp.Total != 1 {
		t.Fatalf("expected total 1, got %d", resp.Total)
	}
	if len(resp.Events) != 1 || resp.Events[0].Title != "Called customer" {
		t.Fatalf("expected 1 event 'Called customer', got %+v", resp.Events)
	}
}

// ============================================================================
// GDPR Consent Management
// ============================================================================

func TestGetContactConsents_MissingTenant(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.GetContactConsents(context.Background(), &crmv1.GetContactConsentsRequest{ContactId: uuid.New().String()})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestGetContactConsents_InvalidContactID(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.GetContactConsents(ctxWithTenant(uuid.New()), &crmv1.GetContactConsentsRequest{ContactId: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestGetContactConsents_HappyPath(t *testing.T) {
	repo := newStubConsentRepo()
	srv := newCRMServerWithConsentRepo(repo)
	tenantID := uuid.New()
	contactID := uuid.New()
	repo.contacts[contactID] = true

	_, err := srv.GrantConsent(ctxWithTenant(tenantID), &crmv1.GrantConsentRequest{
		ContactId:   contactID.String(),
		ConsentType: "marketing_email",
		Source:      "signup_form",
		LegalBasis:  "consent",
	})
	requireGRPCOK(t, err)

	resp, err := srv.GetContactConsents(ctxWithTenant(tenantID), &crmv1.GetContactConsentsRequest{ContactId: contactID.String()})
	requireGRPCOK(t, err)
	rec, ok := resp.Summary.Consents["marketing_email"]
	if !ok || !rec.Granted {
		t.Fatalf("expected granted marketing_email consent, got %+v", resp.Summary.Consents)
	}
}

// ============================================================================
// GrantConsent
// ============================================================================

func TestGrantConsent_MissingTenant(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.GrantConsent(context.Background(), &crmv1.GrantConsentRequest{ContactId: uuid.New().String()})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestGrantConsent_InvalidContactID(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.GrantConsent(ctxWithTenant(uuid.New()), &crmv1.GrantConsentRequest{ContactId: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestGrantConsent_InvalidConsentType(t *testing.T) {
	repo := newStubConsentRepo()
	srv := newCRMServerWithConsentRepo(repo)
	_, err := srv.GrantConsent(ctxWithTenant(uuid.New()), &crmv1.GrantConsentRequest{
		ContactId:   uuid.New().String(),
		ConsentType: "bogus",
		LegalBasis:  "consent",
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestGrantConsent_InvalidLegalBasis(t *testing.T) {
	repo := newStubConsentRepo()
	srv := newCRMServerWithConsentRepo(repo)
	_, err := srv.GrantConsent(ctxWithTenant(uuid.New()), &crmv1.GrantConsentRequest{
		ContactId:   uuid.New().String(),
		ConsentType: "marketing_email",
		LegalBasis:  "bogus",
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestGrantConsent_ContactNotFound(t *testing.T) {
	repo := newStubConsentRepo()
	srv := newCRMServerWithConsentRepo(repo)
	_, err := srv.GrantConsent(ctxWithTenant(uuid.New()), &crmv1.GrantConsentRequest{
		ContactId:   uuid.New().String(),
		ConsentType: "marketing_email",
		LegalBasis:  "consent",
	})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestGrantConsent_HappyPath(t *testing.T) {
	repo := newStubConsentRepo()
	srv := newCRMServerWithConsentRepo(repo)
	contactID := uuid.New()
	repo.contacts[contactID] = true

	resp, err := srv.GrantConsent(ctxWithTenant(uuid.New()), &crmv1.GrantConsentRequest{
		ContactId:   contactID.String(),
		ConsentType: "marketing_email",
		Source:      "signup_form",
		LegalBasis:  "consent",
	})
	requireGRPCOK(t, err)
	if !resp.Record.Granted {
		t.Fatalf("expected granted=true, got %+v", resp.Record)
	}
}

// ============================================================================
// RevokeConsent
// ============================================================================

func TestRevokeConsent_MissingTenant(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.RevokeConsent(context.Background(), &crmv1.RevokeConsentRequest{ContactId: uuid.New().String()})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestRevokeConsent_InvalidConsentType(t *testing.T) {
	repo := newStubConsentRepo()
	srv := newCRMServerWithConsentRepo(repo)
	_, err := srv.RevokeConsent(ctxWithTenant(uuid.New()), &crmv1.RevokeConsentRequest{
		ContactId:   uuid.New().String(),
		ConsentType: "bogus",
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestRevokeConsent_ContactNotFound(t *testing.T) {
	repo := newStubConsentRepo()
	srv := newCRMServerWithConsentRepo(repo)
	_, err := srv.RevokeConsent(ctxWithTenant(uuid.New()), &crmv1.RevokeConsentRequest{
		ContactId:   uuid.New().String(),
		ConsentType: "marketing_email",
	})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestRevokeConsent_HappyPath(t *testing.T) {
	repo := newStubConsentRepo()
	srv := newCRMServerWithConsentRepo(repo)
	contactID := uuid.New()
	repo.contacts[contactID] = true

	resp, err := srv.RevokeConsent(ctxWithTenant(uuid.New()), &crmv1.RevokeConsentRequest{
		ContactId:   contactID.String(),
		ConsentType: "marketing_email",
		Notes:       "user asked to opt out",
	})
	requireGRPCOK(t, err)
	if resp.Record.Granted {
		t.Fatalf("expected granted=false, got %+v", resp.Record)
	}
}

// ============================================================================
// GetConsentHistory
// ============================================================================

func TestGetConsentHistory_MissingTenant(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.GetConsentHistory(context.Background(), &crmv1.GetConsentHistoryRequest{ContactId: uuid.New().String()})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestGetConsentHistory_InvalidContactID(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.GetConsentHistory(ctxWithTenant(uuid.New()), &crmv1.GetConsentHistoryRequest{ContactId: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestGetConsentHistory_InvalidConsentType(t *testing.T) {
	repo := newStubConsentRepo()
	srv := newCRMServerWithConsentRepo(repo)
	_, err := srv.GetConsentHistory(ctxWithTenant(uuid.New()), &crmv1.GetConsentHistoryRequest{
		ContactId:   uuid.New().String(),
		ConsentType: "bogus",
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestGetConsentHistory_HappyPath(t *testing.T) {
	repo := newStubConsentRepo()
	srv := newCRMServerWithConsentRepo(repo)
	tenantID := uuid.New()
	contactID := uuid.New()
	repo.contacts[contactID] = true

	_, err := srv.GrantConsent(ctxWithTenant(tenantID), &crmv1.GrantConsentRequest{
		ContactId:   contactID.String(),
		ConsentType: "marketing_email",
		LegalBasis:  "consent",
	})
	requireGRPCOK(t, err)

	resp, err := srv.GetConsentHistory(ctxWithTenant(tenantID), &crmv1.GetConsentHistoryRequest{
		ContactId:   contactID.String(),
		ConsentType: "marketing_email",
	})
	requireGRPCOK(t, err)
	if resp.Total != 1 || len(resp.History) != 1 {
		t.Fatalf("expected 1 history entry, got %+v", resp.History)
	}
}

// ============================================================================
// RequestDeletion
// ============================================================================

func TestRequestDeletion_MissingTenant(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.RequestDeletion(context.Background(), &crmv1.RequestDeletionRequest{ContactId: uuid.New().String()})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestRequestDeletion_InvalidContactID(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.RequestDeletion(ctxWithTenant(uuid.New()), &crmv1.RequestDeletionRequest{ContactId: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestRequestDeletion_ContactNotFound(t *testing.T) {
	repo := newStubConsentRepo()
	srv := newCRMServerWithConsentRepo(repo)
	_, err := srv.RequestDeletion(ctxWithTenant(uuid.New()), &crmv1.RequestDeletionRequest{ContactId: uuid.New().String()})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestRequestDeletion_HappyPath(t *testing.T) {
	repo := newStubConsentRepo()
	srv := newCRMServerWithConsentRepo(repo)
	contactID := uuid.New()
	repo.contacts[contactID] = true

	resp, err := srv.RequestDeletion(ctxWithTenant(uuid.New()), &crmv1.RequestDeletionRequest{
		ContactId: contactID.String(),
		Reason:    "user request",
	})
	requireGRPCOK(t, err)
	if resp.DeletionRequest.Status != "pending" {
		t.Fatalf("expected status pending, got %q", resp.DeletionRequest.Status)
	}
}

// ============================================================================
// ProcessDeletion
// ============================================================================

func TestProcessDeletion_MissingTenant(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.ProcessDeletion(context.Background(), &crmv1.ProcessDeletionRequest{RequestId: uuid.New().String()})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestProcessDeletion_InvalidRequestID(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.ProcessDeletion(ctxWithTenant(uuid.New()), &crmv1.ProcessDeletionRequest{RequestId: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestProcessDeletion_NotFound(t *testing.T) {
	repo := newStubConsentRepo()
	srv := newCRMServerWithConsentRepo(repo)
	_, err := srv.ProcessDeletion(ctxWithTenant(uuid.New()), &crmv1.ProcessDeletionRequest{RequestId: uuid.New().String()})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestProcessDeletion_AlreadyComplete(t *testing.T) {
	repo := newStubConsentRepo()
	srv := newCRMServerWithConsentRepo(repo)
	tenantID := uuid.New()
	now := time.Now()
	contactID := uuid.New()
	req := &consent.GDPRDeletionRequest{
		ID:                uuid.New(),
		TenantID:          tenantID,
		ContactID:         &contactID,
		OriginalContactID: contactID,
		Status:            "completed",
		CompletedAt:       &now,
		CreatedAt:         now,
	}
	repo.deletionRequests[req.ID] = req

	_, err := srv.ProcessDeletion(ctxWithTenant(tenantID), &crmv1.ProcessDeletionRequest{RequestId: req.ID.String()})
	requireGRPCCode(t, err, codes.AlreadyExists)
}

func TestProcessDeletion_HappyPath(t *testing.T) {
	repo := newStubConsentRepo()
	srv := newCRMServerWithConsentRepo(repo)
	tenantID := uuid.New()
	contactID := uuid.New()
	req := &consent.GDPRDeletionRequest{
		ID:                uuid.New(),
		TenantID:          tenantID,
		ContactID:         &contactID,
		OriginalContactID: contactID,
		Status:            "pending",
		CreatedAt:         time.Now(),
	}
	repo.deletionRequests[req.ID] = req

	resp, err := srv.ProcessDeletion(ctxWithTenant(tenantID), &crmv1.ProcessDeletionRequest{RequestId: req.ID.String()})
	requireGRPCOK(t, err)
	if resp.Status != "completed" {
		t.Fatalf("expected status completed, got %q", resp.Status)
	}
	if !repo.anonymized[contactID] {
		t.Fatalf("expected contact %s to be anonymized", contactID)
	}
	if repo.deletionRequests[req.ID].Status != "completed" {
		t.Fatalf("expected deletion request status updated to completed")
	}
}
