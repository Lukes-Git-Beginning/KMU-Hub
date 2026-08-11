package server

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"google.golang.org/grpc/codes"

	"github.com/kmuhub/kmuhub/internal/crm/deal"
	"github.com/kmuhub/kmuhub/internal/crm/pipelinestage"
	"github.com/kmuhub/kmuhub/internal/models"
	crmv1 "github.com/kmuhub/kmuhub/proto/crm/v1"
)

// ============================================================================
// Stub repositories (server package copies — pipelinestage.MockRepository and
// deal.MockRepository, if they exist, live in package-private _test.go files
// and can't be imported here; same pattern as stubCompanyRepo in
// crm_grpc_companies_dedupe_import_test.go).
// ============================================================================

type stubPipelineStageRepo struct {
	stages      map[uuid.UUID]*models.PipelineStage
	hasDealsMap map[uuid.UUID]bool
}

func newStubPipelineStageRepo() *stubPipelineStageRepo {
	return &stubPipelineStageRepo{
		stages:      make(map[uuid.UUID]*models.PipelineStage),
		hasDealsMap: make(map[uuid.UUID]bool),
	}
}

func (r *stubPipelineStageRepo) Create(_ context.Context, s *models.PipelineStage) error {
	r.stages[s.ID] = s
	return nil
}

func (r *stubPipelineStageRepo) GetByID(_ context.Context, id, tenantID uuid.UUID) (*models.PipelineStage, error) {
	s, ok := r.stages[id]
	if !ok || s.TenantID != tenantID {
		return nil, pipelinestage.ErrStageNotFound
	}
	return s, nil
}

func (r *stubPipelineStageRepo) List(_ context.Context, tenantID uuid.UUID) ([]*models.PipelineStage, error) {
	var result []*models.PipelineStage
	for _, s := range r.stages {
		if s.TenantID == tenantID {
			result = append(result, s)
		}
	}
	return result, nil
}

func (r *stubPipelineStageRepo) ListWithStats(_ context.Context, tenantID uuid.UUID) ([]*models.PipelineStageWithStats, error) {
	var result []*models.PipelineStageWithStats
	for _, s := range r.stages {
		if s.TenantID == tenantID {
			result = append(result, &models.PipelineStageWithStats{PipelineStage: *s})
		}
	}
	return result, nil
}

func (r *stubPipelineStageRepo) Update(_ context.Context, s *models.PipelineStage) error {
	existing, ok := r.stages[s.ID]
	if !ok || existing.TenantID != s.TenantID {
		return pipelinestage.ErrStageNotFound
	}
	r.stages[s.ID] = s
	return nil
}

func (r *stubPipelineStageRepo) Delete(_ context.Context, id, tenantID uuid.UUID) error {
	s, ok := r.stages[id]
	if !ok || s.TenantID != tenantID {
		return pipelinestage.ErrStageNotFound
	}
	delete(r.stages, id)
	return nil
}

func (r *stubPipelineStageRepo) Reorder(_ context.Context, _ uuid.UUID, stageIDs []uuid.UUID) error {
	for i, id := range stageIDs {
		if s, ok := r.stages[id]; ok {
			s.SortOrder = i
		}
	}
	return nil
}

func (r *stubPipelineStageRepo) HasDeals(_ context.Context, id, _ uuid.UUID) (bool, error) {
	return r.hasDealsMap[id], nil
}

func (r *stubPipelineStageRepo) HasWonStage(_ context.Context, tenantID uuid.UUID, excludeID *uuid.UUID) (bool, error) {
	for _, s := range r.stages {
		if s.TenantID == tenantID && s.IsWon && (excludeID == nil || s.ID != *excludeID) {
			return true, nil
		}
	}
	return false, nil
}

func (r *stubPipelineStageRepo) HasLostStage(_ context.Context, tenantID uuid.UUID, excludeID *uuid.UUID) (bool, error) {
	for _, s := range r.stages {
		if s.TenantID == tenantID && s.IsLost && (excludeID == nil || s.ID != *excludeID) {
			return true, nil
		}
	}
	return false, nil
}

func (r *stubPipelineStageRepo) GetNextSortOrder(_ context.Context, tenantID uuid.UUID) (int, error) {
	next := 0
	for _, s := range r.stages {
		if s.TenantID == tenantID && s.SortOrder >= next {
			next = s.SortOrder + 1
		}
	}
	return next, nil
}

func (r *stubPipelineStageRepo) CountStages(_ context.Context, tenantID uuid.UUID) (int, error) {
	count := 0
	for _, s := range r.stages {
		if s.TenantID == tenantID {
			count++
		}
	}
	return count, nil
}

type stubDealRepo struct {
	deals     map[uuid.UUID]*models.Deal
	stages    map[uuid.UUID]*models.PipelineStage
	contacts  map[uuid.UUID]bool
	companies map[uuid.UUID]bool
	owners    map[uuid.UUID]bool
	validTags map[uuid.UUID]models.EntityType
	tags      map[uuid.UUID][]*models.Tag
}

func newStubDealRepo() *stubDealRepo {
	return &stubDealRepo{
		deals:     make(map[uuid.UUID]*models.Deal),
		stages:    make(map[uuid.UUID]*models.PipelineStage),
		contacts:  make(map[uuid.UUID]bool),
		companies: make(map[uuid.UUID]bool),
		owners:    make(map[uuid.UUID]bool),
		validTags: make(map[uuid.UUID]models.EntityType),
		tags:      make(map[uuid.UUID][]*models.Tag),
	}
}

func (r *stubDealRepo) Create(_ context.Context, d *models.Deal) error {
	r.deals[d.ID] = d
	return nil
}

func (r *stubDealRepo) GetByID(_ context.Context, id, tenantID uuid.UUID) (*models.Deal, error) {
	d, ok := r.deals[id]
	if !ok || d.TenantID != tenantID {
		return nil, deal.ErrDealNotFound
	}
	return d, nil
}

func (r *stubDealRepo) List(_ context.Context, tenantID uuid.UUID, filter deal.ListFilter, offset, limit int) ([]*models.Deal, int, error) {
	var result []*models.Deal
	for _, d := range r.deals {
		if d.TenantID != tenantID {
			continue
		}
		if filter.StageID != nil && d.StageID != *filter.StageID {
			continue
		}
		if filter.ContactID != nil && (d.ContactID == nil || *d.ContactID != *filter.ContactID) {
			continue
		}
		if filter.CompanyID != nil && (d.CompanyID == nil || *d.CompanyID != *filter.CompanyID) {
			continue
		}
		if filter.OwnerID != nil && (d.OwnerID == nil || *d.OwnerID != *filter.OwnerID) {
			continue
		}
		result = append(result, d)
	}
	total := len(result)
	if offset >= total {
		return []*models.Deal{}, total, nil
	}
	end := min(offset+limit, total)
	return result[offset:end], total, nil
}

func (r *stubDealRepo) Update(_ context.Context, d *models.Deal) error {
	existing, ok := r.deals[d.ID]
	if !ok || existing.TenantID != d.TenantID {
		return deal.ErrDealNotFound
	}
	r.deals[d.ID] = d
	return nil
}

func (r *stubDealRepo) Delete(_ context.Context, id, tenantID uuid.UUID) error {
	d, ok := r.deals[id]
	if !ok || d.TenantID != tenantID {
		return deal.ErrDealNotFound
	}
	delete(r.deals, id)
	return nil
}

func (r *stubDealRepo) GetStageName(_ context.Context, stageID uuid.UUID) (string, error) {
	if s, ok := r.stages[stageID]; ok {
		return s.Name, nil
	}
	return "", nil
}

func (r *stubDealRepo) GetContactName(_ context.Context, _ uuid.UUID) (string, error) { return "", nil }
func (r *stubDealRepo) GetCompanyName(_ context.Context, _ uuid.UUID) (string, error) { return "", nil }
func (r *stubDealRepo) GetOwnerName(_ context.Context, _ uuid.UUID) (string, error)   { return "", nil }

func (r *stubDealRepo) GetTags(_ context.Context, dealID uuid.UUID) ([]*models.Tag, error) {
	return r.tags[dealID], nil
}

func (r *stubDealRepo) AddTags(_ context.Context, dealID uuid.UUID, tagIDs []uuid.UUID) error {
	for _, id := range tagIDs {
		r.tags[dealID] = append(r.tags[dealID], &models.Tag{ID: id})
	}
	return nil
}

func (r *stubDealRepo) RemoveTags(_ context.Context, dealID uuid.UUID, tagIDs []uuid.UUID) error {
	remaining := r.tags[dealID][:0]
	for _, t := range r.tags[dealID] {
		if !slices.Contains(tagIDs, t.ID) {
			remaining = append(remaining, t)
		}
	}
	r.tags[dealID] = remaining
	return nil
}

func (r *stubDealRepo) GetCustomFieldValues(_ context.Context, _ uuid.UUID) ([]*models.CustomFieldValueRow, error) {
	return nil, nil
}

func (r *stubDealRepo) SetCustomFieldValues(_ context.Context, _ uuid.UUID, _ map[uuid.UUID]any) error {
	return nil
}

func (r *stubDealRepo) GetStageNames(_ context.Context, stageIDs []uuid.UUID) (map[uuid.UUID]string, error) {
	result := make(map[uuid.UUID]string)
	for _, id := range stageIDs {
		if s, ok := r.stages[id]; ok {
			result[id] = s.Name
		}
	}
	return result, nil
}

func (r *stubDealRepo) GetContactNames(_ context.Context, _ []uuid.UUID) (map[uuid.UUID]string, error) {
	return map[uuid.UUID]string{}, nil
}

func (r *stubDealRepo) GetCompanyNames(_ context.Context, _ []uuid.UUID) (map[uuid.UUID]string, error) {
	return map[uuid.UUID]string{}, nil
}

func (r *stubDealRepo) GetOwnerNames(_ context.Context, _ []uuid.UUID) (map[uuid.UUID]string, error) {
	return map[uuid.UUID]string{}, nil
}

func (r *stubDealRepo) GetTagsBatch(_ context.Context, dealIDs []uuid.UUID) (map[uuid.UUID][]*models.Tag, error) {
	result := make(map[uuid.UUID][]*models.Tag)
	for _, id := range dealIDs {
		result[id] = r.tags[id]
	}
	return result, nil
}

func (r *stubDealRepo) GetCustomFieldValuesBatch(_ context.Context, _ []uuid.UUID) (map[uuid.UUID][]*models.CustomFieldValueRow, error) {
	return map[uuid.UUID][]*models.CustomFieldValueRow{}, nil
}

func (r *stubDealRepo) StageExists(_ context.Context, stageID, _ uuid.UUID) (bool, error) {
	_, ok := r.stages[stageID]
	return ok, nil
}

func (r *stubDealRepo) GetStage(_ context.Context, stageID uuid.UUID) (*models.PipelineStage, error) {
	s, ok := r.stages[stageID]
	if !ok {
		return nil, pipelinestage.ErrStageNotFound
	}
	return s, nil
}

func (r *stubDealRepo) ContactExists(_ context.Context, contactID uuid.UUID) (bool, error) {
	return r.contacts[contactID], nil
}

func (r *stubDealRepo) CompanyExists(_ context.Context, companyID uuid.UUID) (bool, error) {
	return r.companies[companyID], nil
}

func (r *stubDealRepo) OwnerExists(_ context.Context, ownerID uuid.UUID) (bool, error) {
	return r.owners[ownerID], nil
}

func (r *stubDealRepo) TagExists(_ context.Context, tagID uuid.UUID, entityType models.EntityType) (bool, error) {
	et, ok := r.validTags[tagID]
	return ok && et == entityType, nil
}

func (r *stubDealRepo) SetClosedAt(_ context.Context, dealID, tenantID uuid.UUID, closedAt *time.Time) error {
	d, ok := r.deals[dealID]
	if !ok || d.TenantID != tenantID {
		return deal.ErrDealNotFound
	}
	d.ClosedAt = closedAt
	return nil
}

// ============================================================================
// Test server constructors
// ============================================================================

func newCRMServerWithPipelineStageRepo(repo pipelinestage.Repository) *CRMGRPCServer {
	return &CRMGRPCServer{pipelineStageService: pipelinestage.NewService(repo)}
}

func newCRMServerWithDealRepo(repo deal.Repository) *CRMGRPCServer {
	return &CRMGRPCServer{dealService: deal.NewService(repo)}
}

func seedStage(repo *stubPipelineStageRepo, tenantID uuid.UUID, opts func(*models.PipelineStage)) *models.PipelineStage {
	s := &models.PipelineStage{
		ID:          uuid.New(),
		TenantID:    tenantID,
		Name:        "Lead",
		Color:       "#3b82f6",
		Probability: decimal.NewFromInt(10),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if opts != nil {
		opts(s)
	}
	repo.stages[s.ID] = s
	return s
}

func seedDealStage(repo *stubDealRepo, tenantID uuid.UUID, opts func(*models.PipelineStage)) *models.PipelineStage {
	s := &models.PipelineStage{
		ID:        uuid.New(),
		TenantID:  tenantID,
		Name:      "Lead",
		Color:     "#3b82f6",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if opts != nil {
		opts(s)
	}
	repo.stages[s.ID] = s
	return s
}

func seedDeal(repo *stubDealRepo, tenantID, stageID uuid.UUID, opts func(*models.Deal)) *models.Deal {
	d := &models.Deal{
		ID:        uuid.New(),
		TenantID:  tenantID,
		Name:      "Big Deal",
		Value:     decimal.NewFromInt(1000),
		Currency:  "EUR",
		StageID:   stageID,
		CreatedBy: uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if opts != nil {
		opts(d)
	}
	repo.deals[d.ID] = d
	return d
}

// ============================================================================
// CreatePipelineStage
// ============================================================================

func TestCreatePipelineStage_MissingTenant(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.CreatePipelineStage(context.Background(), &crmv1.CreatePipelineStageRequest{Name: "Lead"})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestCreatePipelineStage_NameRequired(t *testing.T) {
	repo := newStubPipelineStageRepo()
	srv := newCRMServerWithPipelineStageRepo(repo)
	_, err := srv.CreatePipelineStage(ctxWithTenant(uuid.New()), &crmv1.CreatePipelineStageRequest{Name: "   "})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestCreatePipelineStage_HappyPath(t *testing.T) {
	repo := newStubPipelineStageRepo()
	srv := newCRMServerWithPipelineStageRepo(repo)
	resp, err := srv.CreatePipelineStage(ctxWithTenant(uuid.New()), &crmv1.CreatePipelineStageRequest{
		Name:        "Lead",
		Color:       "#3b82f6",
		Probability: 20,
	})
	requireGRPCOK(t, err)
	if resp.Stage.Name != "Lead" {
		t.Fatalf("expected stage name Lead, got %q", resp.Stage.Name)
	}
}

// ============================================================================
// GetPipelineStage
// ============================================================================

func TestGetPipelineStage_MissingTenant(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.GetPipelineStage(context.Background(), &crmv1.GetPipelineStageRequest{Id: uuid.New().String()})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestGetPipelineStage_InvalidID(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.GetPipelineStage(ctxWithTenant(uuid.New()), &crmv1.GetPipelineStageRequest{Id: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestGetPipelineStage_NotFound(t *testing.T) {
	repo := newStubPipelineStageRepo()
	srv := newCRMServerWithPipelineStageRepo(repo)
	_, err := srv.GetPipelineStage(ctxWithTenant(uuid.New()), &crmv1.GetPipelineStageRequest{Id: uuid.New().String()})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestGetPipelineStage_HappyPath(t *testing.T) {
	repo := newStubPipelineStageRepo()
	srv := newCRMServerWithPipelineStageRepo(repo)
	tenantID := uuid.New()
	stage := seedStage(repo, tenantID, nil)

	resp, err := srv.GetPipelineStage(ctxWithTenant(tenantID), &crmv1.GetPipelineStageRequest{Id: stage.ID.String()})
	requireGRPCOK(t, err)
	if resp.Stage.Id != stage.ID.String() {
		t.Fatalf("expected stage id %s, got %s", stage.ID, resp.Stage.Id)
	}
}

// ============================================================================
// ListPipelineStages
// ============================================================================

func TestListPipelineStages_MissingTenant(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.ListPipelineStages(context.Background(), &crmv1.ListPipelineStagesRequest{})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestListPipelineStages_HappyPath(t *testing.T) {
	repo := newStubPipelineStageRepo()
	srv := newCRMServerWithPipelineStageRepo(repo)
	tenantID := uuid.New()
	seedStage(repo, tenantID, nil)
	seedStage(repo, tenantID, func(s *models.PipelineStage) { s.Name = "Won"; s.IsWon = true })
	seedStage(repo, uuid.New(), nil) // other tenant, must not leak

	resp, err := srv.ListPipelineStages(ctxWithTenant(tenantID), &crmv1.ListPipelineStagesRequest{})
	requireGRPCOK(t, err)
	if len(resp.Stages) != 2 {
		t.Fatalf("expected 2 stages for tenant, got %d", len(resp.Stages))
	}
}

// ============================================================================
// UpdatePipelineStage
// ============================================================================

func TestUpdatePipelineStage_MissingTenant(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.UpdatePipelineStage(context.Background(), &crmv1.UpdatePipelineStageRequest{Id: uuid.New().String()})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestUpdatePipelineStage_InvalidID(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.UpdatePipelineStage(ctxWithTenant(uuid.New()), &crmv1.UpdatePipelineStageRequest{Id: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestUpdatePipelineStage_NotFound(t *testing.T) {
	repo := newStubPipelineStageRepo()
	srv := newCRMServerWithPipelineStageRepo(repo)
	_, err := srv.UpdatePipelineStage(ctxWithTenant(uuid.New()), &crmv1.UpdatePipelineStageRequest{Id: uuid.New().String()})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestUpdatePipelineStage_InvalidColor(t *testing.T) {
	repo := newStubPipelineStageRepo()
	srv := newCRMServerWithPipelineStageRepo(repo)
	tenantID := uuid.New()
	stage := seedStage(repo, tenantID, nil)
	badColor := "not-a-color"

	_, err := srv.UpdatePipelineStage(ctxWithTenant(tenantID), &crmv1.UpdatePipelineStageRequest{
		Id:    stage.ID.String(),
		Color: &badColor,
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestUpdatePipelineStage_HappyPath(t *testing.T) {
	repo := newStubPipelineStageRepo()
	srv := newCRMServerWithPipelineStageRepo(repo)
	tenantID := uuid.New()
	stage := seedStage(repo, tenantID, nil)
	newName := "Qualified"

	resp, err := srv.UpdatePipelineStage(ctxWithTenant(tenantID), &crmv1.UpdatePipelineStageRequest{
		Id:   stage.ID.String(),
		Name: &newName,
	})
	requireGRPCOK(t, err)
	if resp.Stage.Name != "Qualified" {
		t.Fatalf("expected updated name Qualified, got %q", resp.Stage.Name)
	}
}

// ============================================================================
// DeletePipelineStage
// ============================================================================

func TestDeletePipelineStage_MissingTenant(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.DeletePipelineStage(context.Background(), &crmv1.DeletePipelineStageRequest{Id: uuid.New().String()})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestDeletePipelineStage_InvalidID(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.DeletePipelineStage(ctxWithTenant(uuid.New()), &crmv1.DeletePipelineStageRequest{Id: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestDeletePipelineStage_HasDeals(t *testing.T) {
	repo := newStubPipelineStageRepo()
	srv := newCRMServerWithPipelineStageRepo(repo)
	tenantID := uuid.New()
	stage := seedStage(repo, tenantID, nil)
	repo.hasDealsMap[stage.ID] = true

	_, err := srv.DeletePipelineStage(ctxWithTenant(tenantID), &crmv1.DeletePipelineStageRequest{Id: stage.ID.String()})
	requireGRPCCode(t, err, codes.FailedPrecondition)
}

func TestDeletePipelineStage_HappyPath(t *testing.T) {
	repo := newStubPipelineStageRepo()
	srv := newCRMServerWithPipelineStageRepo(repo)
	tenantID := uuid.New()
	stage := seedStage(repo, tenantID, nil)

	_, err := srv.DeletePipelineStage(ctxWithTenant(tenantID), &crmv1.DeletePipelineStageRequest{Id: stage.ID.String()})
	requireGRPCOK(t, err)
	if _, ok := repo.stages[stage.ID]; ok {
		t.Fatal("expected stage to be deleted from repo")
	}
}

// ============================================================================
// ReorderPipelineStages
// ============================================================================

func TestReorderPipelineStages_MissingTenant(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.ReorderPipelineStages(context.Background(), &crmv1.ReorderPipelineStagesRequest{})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestReorderPipelineStages_InvalidStageIDInList(t *testing.T) {
	repo := newStubPipelineStageRepo()
	srv := newCRMServerWithPipelineStageRepo(repo)
	_, err := srv.ReorderPipelineStages(ctxWithTenant(uuid.New()), &crmv1.ReorderPipelineStagesRequest{
		StageIds: []string{"not-a-uuid"},
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestReorderPipelineStages_CountMismatch(t *testing.T) {
	repo := newStubPipelineStageRepo()
	srv := newCRMServerWithPipelineStageRepo(repo)
	tenantID := uuid.New()
	seedStage(repo, tenantID, nil)
	seedStage(repo, tenantID, nil)

	// Only one of the two tenant stages is listed - reorder must reject.
	_, err := srv.ReorderPipelineStages(ctxWithTenant(tenantID), &crmv1.ReorderPipelineStagesRequest{
		StageIds: []string{uuid.New().String()},
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestReorderPipelineStages_HappyPath(t *testing.T) {
	repo := newStubPipelineStageRepo()
	srv := newCRMServerWithPipelineStageRepo(repo)
	tenantID := uuid.New()
	s1 := seedStage(repo, tenantID, func(s *models.PipelineStage) { s.SortOrder = 0 })
	s2 := seedStage(repo, tenantID, func(s *models.PipelineStage) { s.SortOrder = 1 })

	resp, err := srv.ReorderPipelineStages(ctxWithTenant(tenantID), &crmv1.ReorderPipelineStagesRequest{
		StageIds: []string{s2.ID.String(), s1.ID.String()},
	})
	requireGRPCOK(t, err)
	if len(resp.Stages) != 2 {
		t.Fatalf("expected 2 stages in reorder response, got %d", len(resp.Stages))
	}
	if repo.stages[s2.ID].SortOrder != 0 || repo.stages[s1.ID].SortOrder != 1 {
		t.Fatal("expected reorder to have swapped sort_order")
	}
}

// ============================================================================
// CreateDeal
// ============================================================================

func TestCreateDeal_MissingTenant(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.CreateDeal(context.Background(), &crmv1.CreateDealRequest{
		Name:      "Big Deal",
		CreatedBy: uuid.New().String(),
		StageId:   uuid.New().String(),
	})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestCreateDeal_InvalidCreatedBy(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.CreateDeal(ctxWithTenant(uuid.New()), &crmv1.CreateDealRequest{
		Name:      "Big Deal",
		CreatedBy: "not-a-uuid",
		StageId:   uuid.New().String(),
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestCreateDeal_InvalidStageID(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.CreateDeal(ctxWithTenant(uuid.New()), &crmv1.CreateDealRequest{
		Name:      "Big Deal",
		CreatedBy: uuid.New().String(),
		StageId:   "not-a-uuid",
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestCreateDeal_StageNotFound(t *testing.T) {
	repo := newStubDealRepo()
	srv := newCRMServerWithDealRepo(repo)
	_, err := srv.CreateDeal(ctxWithTenant(uuid.New()), &crmv1.CreateDealRequest{
		Name:      "Big Deal",
		CreatedBy: uuid.New().String(),
		StageId:   uuid.New().String(),
	})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestCreateDeal_HappyPath(t *testing.T) {
	repo := newStubDealRepo()
	srv := newCRMServerWithDealRepo(repo)
	tenantID := uuid.New()
	stage := seedDealStage(repo, tenantID, nil)

	resp, err := srv.CreateDeal(ctxWithTenant(tenantID), &crmv1.CreateDealRequest{
		Name:      "Big Deal",
		Value:     500,
		Currency:  "EUR",
		CreatedBy: uuid.New().String(),
		StageId:   stage.ID.String(),
	})
	requireGRPCOK(t, err)
	if resp.Deal.Name != "Big Deal" {
		t.Fatalf("expected deal name Big Deal, got %q", resp.Deal.Name)
	}
}

// ============================================================================
// GetDeal
// ============================================================================

func TestGetDeal_MissingTenant(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.GetDeal(context.Background(), &crmv1.GetDealRequest{Id: uuid.New().String()})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestGetDeal_InvalidID(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.GetDeal(ctxWithTenant(uuid.New()), &crmv1.GetDealRequest{Id: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestGetDeal_NotFound(t *testing.T) {
	repo := newStubDealRepo()
	srv := newCRMServerWithDealRepo(repo)
	_, err := srv.GetDeal(ctxWithTenant(uuid.New()), &crmv1.GetDealRequest{Id: uuid.New().String()})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestGetDeal_HappyPath(t *testing.T) {
	repo := newStubDealRepo()
	srv := newCRMServerWithDealRepo(repo)
	tenantID := uuid.New()
	stage := seedDealStage(repo, tenantID, nil)
	d := seedDeal(repo, tenantID, stage.ID, nil)

	resp, err := srv.GetDeal(ctxWithTenant(tenantID), &crmv1.GetDealRequest{Id: d.ID.String()})
	requireGRPCOK(t, err)
	if resp.Deal.Id != d.ID.String() {
		t.Fatalf("expected deal id %s, got %s", d.ID, resp.Deal.Id)
	}
}

// ============================================================================
// ListDeals
// ============================================================================

func TestListDeals_MissingTenant(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.ListDeals(context.Background(), &crmv1.ListDealsRequest{})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestListDeals_InvalidStageIDFilter(t *testing.T) {
	srv := newTestCRMServer()
	badStageID := "not-a-uuid"
	_, err := srv.ListDeals(ctxWithTenant(uuid.New()), &crmv1.ListDealsRequest{StageId: &badStageID})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestListDeals_HappyPath(t *testing.T) {
	repo := newStubDealRepo()
	srv := newCRMServerWithDealRepo(repo)
	tenantID := uuid.New()
	stage := seedDealStage(repo, tenantID, nil)
	seedDeal(repo, tenantID, stage.ID, nil)
	seedDeal(repo, tenantID, stage.ID, nil)
	seedDeal(repo, uuid.New(), stage.ID, nil) // other tenant, must not leak

	resp, err := srv.ListDeals(ctxWithTenant(tenantID), &crmv1.ListDealsRequest{})
	requireGRPCOK(t, err)
	if resp.Total != 2 || len(resp.Deals) != 2 {
		t.Fatalf("expected 2 deals for tenant, got total=%d len=%d", resp.Total, len(resp.Deals))
	}
}

// ============================================================================
// UpdateDeal
// ============================================================================

func TestUpdateDeal_MissingTenant(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.UpdateDeal(context.Background(), &crmv1.UpdateDealRequest{Id: uuid.New().String()})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestUpdateDeal_InvalidID(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.UpdateDeal(ctxWithTenant(uuid.New()), &crmv1.UpdateDealRequest{Id: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestUpdateDeal_InvalidCurrency(t *testing.T) {
	repo := newStubDealRepo()
	srv := newCRMServerWithDealRepo(repo)
	tenantID := uuid.New()
	stage := seedDealStage(repo, tenantID, nil)
	d := seedDeal(repo, tenantID, stage.ID, nil)
	badCurrency := "XXX"

	_, err := srv.UpdateDeal(ctxWithTenant(tenantID), &crmv1.UpdateDealRequest{
		Id:       d.ID.String(),
		Currency: &badCurrency,
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestUpdateDeal_ClearContactID(t *testing.T) {
	repo := newStubDealRepo()
	srv := newCRMServerWithDealRepo(repo)
	tenantID := uuid.New()
	stage := seedDealStage(repo, tenantID, nil)
	contactID := uuid.New()
	d := seedDeal(repo, tenantID, stage.ID, func(d *models.Deal) { d.ContactID = &contactID })
	empty := ""

	resp, err := srv.UpdateDeal(ctxWithTenant(tenantID), &crmv1.UpdateDealRequest{
		Id:        d.ID.String(),
		ContactId: &empty,
	})
	requireGRPCOK(t, err)
	if resp.Deal.ContactId != nil {
		t.Fatalf("expected contact_id cleared, got %v", *resp.Deal.ContactId)
	}
}

func TestUpdateDeal_HappyPath(t *testing.T) {
	repo := newStubDealRepo()
	srv := newCRMServerWithDealRepo(repo)
	tenantID := uuid.New()
	stage := seedDealStage(repo, tenantID, nil)
	d := seedDeal(repo, tenantID, stage.ID, nil)
	newName := "Renamed Deal"

	resp, err := srv.UpdateDeal(ctxWithTenant(tenantID), &crmv1.UpdateDealRequest{
		Id:   d.ID.String(),
		Name: &newName,
	})
	requireGRPCOK(t, err)
	if resp.Deal.Name != "Renamed Deal" {
		t.Fatalf("expected updated name Renamed Deal, got %q", resp.Deal.Name)
	}
}

// ============================================================================
// DeleteDeal
// ============================================================================

func TestDeleteDeal_MissingTenant(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.DeleteDeal(context.Background(), &crmv1.DeleteDealRequest{Id: uuid.New().String()})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestDeleteDeal_InvalidID(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.DeleteDeal(ctxWithTenant(uuid.New()), &crmv1.DeleteDealRequest{Id: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestDeleteDeal_NotFound(t *testing.T) {
	repo := newStubDealRepo()
	srv := newCRMServerWithDealRepo(repo)
	_, err := srv.DeleteDeal(ctxWithTenant(uuid.New()), &crmv1.DeleteDealRequest{Id: uuid.New().String()})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestDeleteDeal_HappyPath(t *testing.T) {
	repo := newStubDealRepo()
	srv := newCRMServerWithDealRepo(repo)
	tenantID := uuid.New()
	stage := seedDealStage(repo, tenantID, nil)
	d := seedDeal(repo, tenantID, stage.ID, nil)

	_, err := srv.DeleteDeal(ctxWithTenant(tenantID), &crmv1.DeleteDealRequest{Id: d.ID.String()})
	requireGRPCOK(t, err)
	if _, ok := repo.deals[d.ID]; ok {
		t.Fatal("expected deal to be deleted from repo")
	}
}

// ============================================================================
// MoveDealToStage
// ============================================================================

func TestMoveDealToStage_MissingTenant(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.MoveDealToStage(context.Background(), &crmv1.MoveDealToStageRequest{
		DealId:  uuid.New().String(),
		StageId: uuid.New().String(),
	})
	requireGRPCCode(t, err, codes.Unauthenticated)
}

func TestMoveDealToStage_InvalidDealID(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.MoveDealToStage(ctxWithTenant(uuid.New()), &crmv1.MoveDealToStageRequest{
		DealId:  "not-a-uuid",
		StageId: uuid.New().String(),
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestMoveDealToStage_InvalidStageID(t *testing.T) {
	srv := newTestCRMServer()
	_, err := srv.MoveDealToStage(ctxWithTenant(uuid.New()), &crmv1.MoveDealToStageRequest{
		DealId:  uuid.New().String(),
		StageId: "not-a-uuid",
	})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestMoveDealToStage_StageNotFound(t *testing.T) {
	repo := newStubDealRepo()
	srv := newCRMServerWithDealRepo(repo)
	tenantID := uuid.New()
	stage := seedDealStage(repo, tenantID, nil)
	d := seedDeal(repo, tenantID, stage.ID, nil)

	_, err := srv.MoveDealToStage(ctxWithTenant(tenantID), &crmv1.MoveDealToStageRequest{
		DealId:  d.ID.String(),
		StageId: uuid.New().String(),
	})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestMoveDealToStage_HappyPath(t *testing.T) {
	repo := newStubDealRepo()
	srv := newCRMServerWithDealRepo(repo)
	tenantID := uuid.New()
	fromStage := seedDealStage(repo, tenantID, nil)
	toStage := seedDealStage(repo, tenantID, func(s *models.PipelineStage) { s.Name = "Won"; s.IsWon = true })
	d := seedDeal(repo, tenantID, fromStage.ID, nil)

	resp, err := srv.MoveDealToStage(ctxWithTenant(tenantID), &crmv1.MoveDealToStageRequest{
		DealId:  d.ID.String(),
		StageId: toStage.ID.String(),
	})
	requireGRPCOK(t, err)
	if resp.Deal.StageId != toStage.ID.String() {
		t.Fatalf("expected stage_id %s, got %s", toStage.ID, resp.Deal.StageId)
	}
	if repo.deals[d.ID].ClosedAt == nil {
		t.Fatal("expected closed_at to be set when moving into a won stage")
	}
}
